import { Form, Input, Button, Card, Typography, App as AntdApp, Select } from 'antd'
import { useNavigate, Link } from 'react-router-dom'
import { useAuthStore } from '../store/auth'

// RegisterPage collects email, password, name, and role. New users default to
// member; the role select is included so an admin seeding accounts can pick a
// role, though in production elevation would go through an admin-only path.
export default function RegisterPage() {
  const register = useAuthStore((s) => s.register)
  const loading = useAuthStore((s) => s.loading)
  const navigate = useNavigate()
  const { message } = AntdApp.useApp()

  const onFinish = async (values: { email: string; password: string; name: string; role?: string }) => {
    try {
      await register(values.email, values.password, values.name, values.role)
      navigate('/kb')
    } catch (e) {
      message.error((e as Error).message)
    }
  }

  return (
    <div style={{ minHeight: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Card style={{ width: 420 }}>
        <Typography.Title level={3} style={{ textAlign: 'center' }}>注册</Typography.Title>
        <Form layout="vertical" onFinish={onFinish} initialValues={{ role: 'member' }}>
          <Form.Item label="姓名" name="name" rules={[{ required: true, message: '请输入姓名' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="邮箱" name="email" rules={[{ required: true, type: 'email', message: '请输入有效邮箱' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="密码" name="password" rules={[{ required: true, min: 6, message: '至少 6 位' }]}>
            <Input.Password />
          </Form.Item>
          <Form.Item label="角色" name="role">
            <Select options={[
              { value: 'member', label: '普通成员' },
              { value: 'manager', label: '知识管理员' },
              { value: 'admin', label: '系统管理员' },
            ]} />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" block loading={loading}>注册</Button>
          </Form.Item>
          <div style={{ textAlign: 'center' }}>
            已有账号？<Link to="/login">登录</Link>
          </div>
        </Form>
      </Card>
    </div>
  )
}
