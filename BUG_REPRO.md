## Bug 是什么
删除文档时，存储对象删除失败没有被正确传播。接口仍返回成功，而且持久化层已经删除 Document 与 Chunk 元数据，导致磁盘对象和数据库状态不一致。

## 如何触发
准备一个带有上传对象和一个切片记录的文档，并让对象存储删除操作返回错误。调用删除接口后，测试会同时检查 HTTP 状态、文档元数据和切片元数据。

## 运行指令
```bash
go test ./internal/handler -run '^TestDeleteDocumentKeepsMetadataWhenObjectRemovalFails$' -count=1
```

## 错误信息
当前实现把存储失败当作成功继续处理，既向调用方报告 200，也提前删除数据库元数据；修正必须约束服务层的调用顺序和仓储删除边界。

## 错误堆栈
以下为 2026-08-20 在缺陷环境执行上述指令得到的原始测试失败输出：
```text
2026/08/20 14:27:35 /Users/tog/Desktop/code/go标注/我的go/2026-08-20/company-kn-system__002/env/internal/repository/document.go:28 record not found
[0.027ms] [rows:0] SELECT * FROM `documents` WHERE id = "5e264957-0ebf-4740-a15f-a9d80c5f4666" AND `documents`.`deleted_at` IS NULL ORDER BY `documents`.`id` LIMIT 1
--- FAIL: TestDeleteDocumentKeepsMetadataWhenObjectRemovalFails (0.00s)
    document_delete_storage_failure_test.go:68: delete endpoint returned 200 after storage deletion failure, want 500
    document_delete_storage_failure_test.go:74: document metadata was removed after storage deletion failure: record not found
    document_delete_storage_failure_test.go:81: document chunks were removed after storage deletion failure: got 0, want 1
FAIL
FAIL	kn-system/internal/handler	0.725s
FAIL
```
