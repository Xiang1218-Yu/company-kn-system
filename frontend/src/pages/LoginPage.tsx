import { Form, Input, Button, Card, Typography, App as AntdApp } from 'antd'
import { useNavigate, Link } from 'react-router-dom'
import { useAuthStore } from '../store/auth'

// LoginPage is the email+password entry. On success it navigates to the main
// app; on failure the store's promise rejects and the message surfaces here.
export default function LoginPage() {
  const login = useAuthStore((s) => s.login)
  const loading = useAuthStore((s) => s.loading)
  const navigate = useNavigate()
  const { message } = AntdApp.useApp()

  const onFinish = async (values: { email: string; password: string }) => {
    try {
      await login(values.email, values.password)
      navigate('/kb')
    } catch (e) {
      message.error((e as Error).message)
    }
  }

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Card style={{ width: 380 }}>
        <Typography.Title level={3} style={{ textAlign: 'center' }}>登录</Typography.Title>
        <Form layout="vertical" onFinish={onFinish} initialValues={{ email: '', password: '' }}>
          <Form.Item label="邮箱" name="email" rules={[{ required: true, type: 'email', message: '请输入有效邮箱' }]}>
            <Input placeholder="you@company.com" />
          </Form.Item>
          <Form.Item label="密码" name="password" rules={[{ required: true, message: '请输入密码' }]}>
            <Input.Password placeholder="至少 6 位" />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block loading={loading}>登录</Button>
          </Form.Item>
          <div style={{ textAlign: 'center' }}>
            还没有账号？<Link to="/register">注册</Link>
          </div>
        </Form>
      </Card>
    </div>
  )
}
