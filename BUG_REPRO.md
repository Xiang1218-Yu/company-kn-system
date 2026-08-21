# 文档切片编号错位

## Bug 是什么
长文档包含多个段落时，切片编号没有保持整份文档范围内的唯一连续关系，索引结果中的引用序号可能重复或与原文位置错位。

## 如何触发
使用包含至少两个段落、且第一段能够产生多个切片的文档执行切片和索引流程，检查后续切片的编号是否仍按整份文档的全局顺序递增。

## 运行指令
```bash
go test ./internal/parser -run '^TestChunkerKeepsGlobalChunkIndexesForMultiParagraphDocument$' -count=1
```

## 错误信息
定向测试失败，说明当前实现未满足题面要求。

## 错误堆栈
```text
--- FAIL: TestChunkerKeepsGlobalChunkIndexesForMultiParagraphDocument (0.00s)
    chunker_index_test.go:16: chunk 2 has index 0, want global index 2
FAIL
FAIL	kn-system/internal/parser	0.807s
FAIL
```
