import { useEffect, useState } from 'react'
import { Card, Row, Col, Statistic, Spin, Typography, Progress, App as AntdApp } from 'antd'
import { FileTextOutlined, MessageOutlined, SmileOutlined } from '@ant-design/icons'
import { dashboardApi } from '../api/dashboard'
import type { Dashboard } from '../types'

// DashboardPage shows the three operational metrics from the spec: document
// count, Q&A volume, and satisfaction ratio. Satisfaction is rendered as a
// progress percentage since it's a 0-1 ratio.
export default function DashboardPage() {
  const [data, setData] = useState<Dashboard | null>(null)
  const [loading, setLoading] = useState(false)
  const { message } = AntdApp.useApp()

  const load = async () => {
    setLoading(true)
    try {
      setData(await dashboardApi.load())
    } catch (e) {
      message.error((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { void load() }, [])

  return (
    <Card title={<Typography.Title level={4} style={{ margin: 0 }}>运营看板</Typography.Title>}>
      <Spin spinning={loading}>
        <Row gutter={24}>
          <Col xs={24} sm={8}>
            <Statistic title="文档总数" value={data?.document_count ?? 0} prefix={<FileTextOutlined />} />
          </Col>
          <Col xs={24} sm={8}>
            <Statistic title="问答次数" value={data?.qa_count ?? 0} prefix={<MessageOutlined />} />
          </Col>
          <Col xs={24} sm={8}>
            <Statistic
              title="满意率"
              value={((data?.satisfaction ?? 0) * 100).toFixed(1)}
              suffix="%"
              prefix={<SmileOutlined />}
            />
          </Col>
        </Row>
        <div style={{ marginTop: 32 }}>
          <Typography.Text type="secondary">满意率（点赞 / 已投票）</Typography.Text>
          <Progress
            percent={Math.round((data?.satisfaction ?? 0) * 100)}
            status={((data?.satisfaction ?? 0) * 100) >= 80 ? 'success' : 'active'}
          />
        </div>
      </Spin>
    </Card>
  )
}
