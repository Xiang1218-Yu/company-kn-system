import { Layout, Menu, Button, Space, Typography } from 'antd'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '../store/auth'

const { Header, Content } = Layout

// MainLayout is the authenticated app shell: a top nav with the user's name and
// logout, plus the routed page content. Keeping it in one component means every
// authenticated page shares the same chrome without re-declaring it.
export default function MainLayout() {
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)
  const navigate = useNavigate()
  const location = useLocation()

  // Derive the selected nav key from the current path so the nav highlights the
  // active section.
  const selectedKey = location.pathname.startsWith('/kb')
    ? '/kb'
    : location.pathname.startsWith('/dashboard')
      ? '/dashboard'
      : location.pathname

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Space size="large">
          <Typography.Title level={4} style={{ color: '#fff', margin: 0 }}>
            知识库问答系统
          </Typography.Title>
          <Menu
            theme="dark"
            mode="horizontal"
            selectedKeys={[selectedKey]}
            onClick={({ key }) => navigate(key)}
            items={[
              { key: '/kb', label: '知识库' },
              { key: '/dashboard', label: '运营看板' },
            ]}
            style={{ minWidth: 240, borderBottom: 'none' }}
          />
        </Space>
        <Space>
          <Typography.Text style={{ color: '#fff' }}>
            {user?.name}（{roleLabel(user?.role)}）
          </Typography.Text>
          <Button type="text" style={{ color: '#fff' }} onClick={() => { logout(); navigate('/login') }}>
            退出
          </Button>
        </Space>
      </Header>
      <Content style={{ padding: '24px', maxWidth: 1200, width: '100%', margin: '0 auto' }}>
        <Outlet />
      </Content>
    </Layout>
  )
}

function roleLabel(role?: string) {
  switch (role) {
    case 'admin': return '系统管理员'
    case 'manager': return '知识管理员'
    default: return '成员'
  }
}
