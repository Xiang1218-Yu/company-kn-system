# Bug 复现说明

## Bug 是什么

问答请求的取消信号没有完整传递到外部模型 HTTP 客户端。客户端取消请求后，模型请求仍可能被发出，导致已经取消的问答继续占用下游资源。

## 如何触发

在含 Bug 的工作区直接执行下方定向测试即可触发：

## 运行指令

```bash
go test ./pkg/openai -run '^TestOpenAIClientPropagatesCanceledContext$' -count=1
```

## 错误信息

该测试应因已取消的 Context 仍然发出供应商请求而失败。

## 错误堆栈

```text
--- FAIL: TestOpenAIClientPropagatesCanceledContext (0.00s)
    client_context_test.go:39: expected canceled context to stop the provider request
FAIL
FAIL	kn-system/pkg/openai	0.258s
FAIL
```
