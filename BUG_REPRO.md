## Bug 是什么
认证上下文中的角色和用户身份被直接类型断言或解引用。角色格式错误时权限中间件会 panic；缺少用户身份时仪表盘入口返回了禁止而不是稳定的未认证响应。

## 如何触发
构造 Gin 上下文，其中角色被污染为字符串，再调用权限检查；同时构造没有认证用户的仪表盘请求。测试会捕获 panic 并检查未认证状态码。

## 运行指令
```bash
go test ./internal/middleware -run '^TestContextGuardsRejectMalformedOrMissingIdentity$' -count=1
```

## 错误信息
当前身份读取没有 fail-closed：异常类型触发运行时崩溃，缺失用户也没有被统一映射为 401。修正需要同时收紧中间件解码和 handler 的身份前置条件。

## 错误堆栈
以下为 2026-08-20 在缺陷环境执行上述指令得到的原始测试失败输出：
```text
--- FAIL: TestContextGuardsRejectMalformedOrMissingIdentity (0.00s)
    --- FAIL: TestContextGuardsRejectMalformedOrMissingIdentity/malformed_role (0.00s)
        context_guard_test.go:32: RequireRole panicked for malformed role: interface conversion: interface {} is string, not model.Role
    --- FAIL: TestContextGuardsRejectMalformedOrMissingIdentity/missing_user (0.00s)
        context_guard_test.go:50: dashboard returned HTTP 403 without authenticated user, want 401
FAIL
FAIL	kn-system/internal/middleware	0.550s
FAIL
```
