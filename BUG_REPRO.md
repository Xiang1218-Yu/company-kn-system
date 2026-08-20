## Bug 是什么
同一文档被重复建立索引时，旧的 Chunk 记录没有在写入新索引前替换或清理。异步任务重试后会保留重复片段，影响检索结果和文档切片计数。

## 如何触发
先为同一个文档写入既有切片，再触发一次重建索引。测试会统计该文档关联的 Chunk 数量；当前缺陷环境把新切片追加到旧切片后面。

## 运行指令
```bash
go test ./internal/service -run '^TestReindexReplacesExistingDocumentChunks$' -count=1
```

## 错误信息
当前重建流程缺少旧切片与新切片之间的一致性边界，重复任务执行后同一内容出现两份记录。

## 错误堆栈
以下为 2026-08-20 在缺陷环境执行上述指令得到的原始测试失败输出：
```text
--- FAIL: TestReindexReplacesExistingDocumentChunks (0.00s)
    reindex_replace_chunks_test.go:77: reindex retained duplicate chunks: got 2, want 1
FAIL
FAIL	kn-system/internal/service	0.980s
FAIL
```
