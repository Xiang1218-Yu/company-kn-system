import { useEffect, useState } from 'react'
import { Card, Table, Button, Space, Modal, Form, Input, Typography, Tag, App as AntdApp, Popconfirm } from 'antd'
import { useNavigate } from 'react-router-dom'
import type { ColumnsType } from 'antd/es/table'
import { kbApi, type KBInput } from '../api/kb'
import type { KnowledgeBase } from '../types'
import { useAuthStore } from '../store/auth'

// KBListPage lists knowledge bases and offers create/edit/delete. Delete is
// admin-only on the backend; we hide the button for non-admins to avoid an
// obvious 403. Clicking a row enters the knowledge base to manage documents.
export default function KBListPage() {
  const [kbs, setKbs] = useState<KnowledgeBase[]>([])
  const [loading, setLoading] = useState(false)
  const [open, setOpen] = useState(false)
  const [editing, setEditing] = useState<KnowledgeBase | null>(null)
  const [form] = Form.useForm<KBInput>()
  const navigate = useNavigate()
  const { message } = AntdApp.useApp()
  const role = useAuthStore((s) => s.user?.role)

  const load = async () => {
    setLoading(true)
    try {
      setKbs(await kbApi.list())
    } catch (e) {
      message.error((e as Error).message)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { void load() }, [])

  const openCreate = () => {
    setEditing(null)
    form.resetFields()
    setOpen(true)
  }

  const openEdit = (kb: KnowledgeBase) => {
    setEditing(kb)
    form.setFieldsValue({ name: kb.name, description: kb.description })
    setOpen(true)
  }

  const submit = async () => {
    const values = await form.validateFields()
    try {
      if (editing) {
        await kbApi.update(editing.id, values)
      } else {
        await kbApi.create(values)
      }
      setOpen(false)
      await load()
      message.success(editing ? '已更新' : '已创建')
    } catch (e) {
      message.error((e as Error).message)
    }
  }

  const remove = async (id: string) => {
    try {
      await kbApi.remove(id)
      await load()
      message.success('已删除')
    } catch (e) {
      message.error((e as Error).message)
    }
  }

  const columns: ColumnsType<KnowledgeBase> = [
    { title: '名称', dataIndex: 'name', key: 'name' },
    { title: '描述', dataIndex: 'description', key: 'description', ellipsis: true },
    {
      title: '创建时间', dataIndex: 'created_at', key: 'created_at',
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: '操作', key: 'actions',
      render: (_, record) => (
        <Space>
          <Button size="small" onClick={() => navigate(`/kb/${record.id}/documents`)}>进入</Button>
          <Button size="small" onClick={() => openEdit(record)}>编辑</Button>
          {role === 'admin' && (
            <Popconfirm title="确认删除该知识库？" onConfirm={() => remove(record.id)}>
              <Button size="small" danger>删除</Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ]

  return (
    <Card
      title={<Typography.Title level={4} style={{ margin: 0 }}>知识库</Typography.Title>}
      extra={<Button type="primary" onClick={openCreate}>新建知识库</Button>}
    >
      <Table
        rowKey="id"
        loading={loading}
        dataSource={kbs}
        columns={columns}
        onRow={(record) => ({ onClick: () => navigate(`/kb/${record.id}/documents`), style: { cursor: 'pointer' } })}
      />
      <Tag color="blue" style={{ marginTop: 8 }}>
        点击知识库行可进入文档管理；问答入口在文档页右上角。
      </Tag>

      <Modal
        title={editing ? '编辑知识库' : '新建知识库'}
        open={open}
        onCancel={() => setOpen(false)}
        onOk={submit}
        okText="保存"
        cancelText="取消"
      >
        <Form form={form} layout="vertical">
          <Form.Item label="名称" name="name" rules={[{ required: true, message: '请输入名称' }]}>
            <Input />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </Card>
  )
}
