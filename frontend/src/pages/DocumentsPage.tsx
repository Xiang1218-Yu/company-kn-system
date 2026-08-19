import { useEffect, useState, useRef, useCallback } from 'react'
import {
  Card, Table, Button, Space, Upload, Tag, Input, Typography, App as AntdApp, Popconfirm, Tooltip, Breadcrumb,
} from 'antd'
import { UploadOutlined, DownloadOutlined, ReloadOutlined } from '@ant-design/icons'
import { useNavigate, useParams } from 'react-router-dom'
import type { ColumnsType } from 'antd/es/table'
import type { UploadProps } from 'antd'
import { documentApi } from '../api/document'
import type { Document, DocStatus } from '../types'

const statusColor: Record<DocStatus, string> = {
  pending: 'default',
  indexing: 'processing',
  indexed: 'success',
  failed: 'error',
}
const statusLabel: Record<DocStatus, string> = {
  pending: '待处理', indexing: '索引中', indexed: '已索引', failed: '索引失败',
}

// DocumentsPage manages the documents of one knowledge base: upload, search,
// status polling, download, delete, and re-index. Polling is paused on unmount
// so navigating away stops the timers.
export default function DocumentsPage() {
  const { kbId = '' } = useParams()
  const navigate = useNavigate()
  const { message } = AntdApp.useApp()
  const [docs, setDocs] = useState<Document[]>([])
  const [loading, setLoading] = useState(false)
  const [query, setQuery] = useState('')
  // pendingIds tracks documents still being indexed so the poller only re-fetches
  // status until those finish, avoiding a perpetual refresh.
  const [pendingIds, setPendingIds] = useState<Set<string>>(new Set())
  const pollRef = useRef<number | null>(null)

  const load = useCallback(async (q?: string) => {
    setLoading(true)
    try {
      const list = q ? await documentApi.search(kbId, q) : await documentApi.list(kbId)
      setDocs(list)
      setPendingIds(new Set(list.filter((d) => d.status === 'pending' || d.status === 'indexing').map((d) => d.id)))
    } catch (e) {
      message.error((e as Error).message)
    } finally {
      setLoading(false)
    }
  }, [kbId, message])

  useEffect(() => { void load() }, [load])

  // Poll status of pending documents every 1.5s. Stops once none remain pending.
  useEffect(() => {
    if (pendingIds.size === 0) return
    const tick = async () => {
      const next = new Set<string>()
      for (const id of pendingIds) {
        try {
          const s = await documentApi.status(id)
          if (s.status === 'pending' || s.status === 'indexing') next.add(id)
        } catch { /* keep id on transient error */ }
      }
      setPendingIds(next)
      if (next.size === 0) await load(query || undefined)
    }
    pollRef.current = window.setInterval(tick, 1500)
    return () => { if (pollRef.current) window.clearInterval(pollRef.current) }
  }, [pendingIds, load, query])

  const uploadProps: UploadProps = {
    multiple: true,
    showUploadList: false,
    accept: '.pdf,.docx,.md,.txt',
    customRequest: async (options) => {
      const { file, onSuccess, onError } = options
      try {
        await documentApi.upload(kbId, file as File)
        onSuccess?.({}, new XMLHttpRequest())
        await load(query || undefined)
        message.success(`${(file as File).name} 上传成功`)
      } catch (e) {
        onError?.(e as Error)
        message.error((e as Error).message)
      }
    },
  }

  const remove = async (id: string) => {
    try {
      await documentApi.remove(id)
      await load(query || undefined)
      message.success('已删除')
    } catch (e) {
      message.error((e as Error).message)
    }
  }

  const reindex = async (id: string) => {
    try {
      await documentApi.reindex(id)
      await load(query || undefined)
      message.success('已重新加入索引队列')
    } catch (e) {
      message.error((e as Error).message)
    }
  }

  const download = async (id: string, name: string) => {
    try {
      const blob = await documentApi.download(id)
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = name
      a.click()
      URL.revokeObjectURL(url)
    } catch (e) {
      message.error((e as Error).message)
    }
  }

  const columns: ColumnsType<Document> = [
    { title: '文档名', dataIndex: 'name', key: 'name', ellipsis: true },
    { title: '类型', dataIndex: 'file_type', key: 'file_type', width: 80 },
    {
      title: '大小', dataIndex: 'file_size', key: 'file_size', width: 100,
      render: (v: number) => `${(v / 1024).toFixed(1)} KB`,
    },
    {
      title: '状态', dataIndex: 'status', key: 'status', width: 110,
      render: (s: DocStatus) => <Tag color={statusColor[s]}>{statusLabel[s]}</Tag>,
    },
    { title: '切片', dataIndex: 'chunk_count', key: 'chunk_count', width: 70 },
    {
      title: '上传时间', dataIndex: 'created_at', key: 'created_at', width: 170,
      render: (v: string) => new Date(v).toLocaleString(),
    },
    {
      title: '操作', key: 'actions', width: 220,
      render: (_, r) => (
        <Space>
          <Tooltip title="下载">
            <Button size="small" icon={<DownloadOutlined />} onClick={() => download(r.id, r.name)} />
          </Tooltip>
          {r.status === 'failed' && (
            <Tooltip title="重新索引">
              <Button size="small" icon={<ReloadOutlined />} onClick={() => reindex(r.id)} />
            </Tooltip>
          )}
          <Popconfirm title="确认删除该文档？" onConfirm={() => remove(r.id)}>
            <Button size="small" danger>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <Card
      title={
        <Space direction="vertical" size={0}>
          <Breadcrumb items={[
            { title: <a onClick={() => navigate('/kb')}>知识库</a> },
            { title: '文档管理' },
          ]} />
          <Typography.Title level={4} style={{ margin: 0 }}>文档管理</Typography.Title>
        </Space>
      }
      extra={
        <Space>
          <Button onClick={() => navigate(`/kb/${kbId}/qa`)} type="primary">进入问答</Button>
          <Upload {...uploadProps}>
            <Button type="primary" icon={<UploadOutlined />}>上传文档</Button>
          </Upload>
        </Space>
      }
    >
      <Space style={{ marginBottom: 16, width: '100%', justifyContent: 'space-between' }}>
        <Input.Search
          placeholder="搜索文档名"
          allowClear
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onSearch={(v) => void load(v)}
          style={{ width: 320 }}
        />
        <Button onClick={() => void load(query || undefined)}>刷新</Button>
      </Space>
      <Table
        rowKey="id"
        loading={loading}
        dataSource={docs}
        columns={columns}
        pagination={{ pageSize: 10 }}
      />
      <Typography.Text type="secondary">
        支持 PDF / Word / Markdown / 纯文本。上传后系统会异步解析并建索引，状态实时刷新。
      </Typography.Text>
    </Card>
  )
}
