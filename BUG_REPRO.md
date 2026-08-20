# 多轮问答历史角色和顺序错误

## Bug 是什么
问答接口携带多轮历史时，历史轮次进入模型上下文的顺序不符合对话发生顺序，导致模型先看到较新的问题，后看到较早的问题，连续追问的上下文因此失真。

## 如何触发
提交带有两轮历史问答和当前问题的追问请求，检查送入问答服务的消息是否按最早历史轮次到最近历史轮次，再到当前问题的顺序排列，并保持每轮的 user、assistant 角色。

## 运行指令
```bash
go test ./internal/service -run '^TestBuildMessagesPreservesConversationHistoryOrder$' -count=1
```

## 错误信息
定向测试失败，说明当前实现未满足题面要求。

## 错误堆栈
```text
--- FAIL: TestBuildMessagesPreservesConversationHistoryOrder (0.00s)
    qa_history_test.go:28: message 0 = llm.Message{Role:"user", Content:"第二个问题"}, want llm.Message{Role:"user", Content:"第一个问题"}
FAIL
FAIL	kn-system/internal/service	0.005s
FAIL
```
