## Bug 是什么
公开注册接口把客户端提交的 `role=admin` 当作新账号角色写入并回显。未登录请求因此能够获得管理员身份，随后注册令牌也会携带该高权限角色进入受保护请求。

## 如何触发
在一个干净的测试数据库中，向公开注册入口提交普通用户资料，同时把角色字段指定为 `admin`。测试会检查注册响应中的角色；当前缺陷环境返回了管理员角色。

## 运行指令
```bash
go test ./internal/handler -run '^TestRegisterPublicRoleCannotCreatePrivilegedSession$' -count=1
```

## 错误信息
公开注册请求返回了 `admin`，说明服务层没有为外部注册请求建立角色边界，认证令牌可继承该越权角色。

## 错误堆栈
以下为 2026-08-20 在缺陷环境执行上述指令得到的原始测试失败输出：
```text
2026/08/20 14:27:28 /Users/tog/Desktop/code/go标注/我的go/2026-08-20/company-kn-system__001/env/internal/repository/user.go:26 record not found
[0.036ms] [rows:0] SELECT * FROM `users` WHERE email = "new@example.test" AND `users`.`deleted_at` IS NULL ORDER BY `users`.`id` LIMIT 1
--- FAIL: TestRegisterPublicRoleCannotCreatePrivilegedSession (0.08s)
    public_registration_role_test.go:56: register returned elevated role "admin" for a public request
FAIL
FAIL	kn-system/internal/handler	1.173s
FAIL
```
