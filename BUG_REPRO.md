# Bug 复现说明

## Bug 是什么

txt、docx 和 PDF 解析器在 context 已取消时仍会读取输入流，而不是立即返回取消错误，因此调用方无法得到一致的取消语义，并且已经取消的请求仍消耗了输入。

## 如何触发

分别构造一个已取消的 context 和会记录读取行为的输入流，将它们交给 txt、docx、PDF 三种解析器。解析器开始读取后，测试会检测到输入已被消费且返回的错误不是 `context.Canceled`。

## 运行指令

```bash
go test ./backend/internal/parser -run '^TestParsersStopBeforeReadingAfterCancellation$' -count=1
```

## 错误信息

三种解析器都会报告 `reader was consumed after cancellation`，而目标行为要求返回 `context.Canceled`。

## 错误堆栈

以下为上述指令实际产生的原始输出：

```text
--- FAIL: TestParsersStopBeforeReadingAfterCancellation (0.00s)
    --- FAIL: TestParsersStopBeforeReadingAfterCancellation/plain (0.00s)
        parser_cancellation_test.go:36: Parse() error = reader was consumed after cancellation, want context.Canceled
    --- FAIL: TestParsersStopBeforeReadingAfterCancellation/docx (0.00s)
        parser_cancellation_test.go:36: Parse() error = read docx: reader was consumed after cancellation, want context.Canceled
    --- FAIL: TestParsersStopBeforeReadingAfterCancellation/pdf (0.00s)
        parser_cancellation_test.go:36: Parse() error = read pdf: reader was consumed after cancellation, want context.Canceled
FAIL
FAIL	kn-system/backend/internal/parser	0.509s
FAIL
```
