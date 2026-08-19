import { useState, useRef, useEffect } from 'react'
import { Card, Input, Button, Space, Typography, List, Tag, Breadcrumb, App as AntdApp, Empty, Spin, Tooltip } from 'antd'
import { LikeOutlined, DislikeOutlined, CopyOutlined } from '@ant-design/icons'
import { useNavigate, useParams } from 'react-router-dom'
import { askStream, qaApi, type AskResult } from '../api/qa'
import type { SourceRef, QALog } from '../types'

// A single exchange in the conversation. answer accumulates as tokens stream.
interface Turn {
  question: string
  answer: string
  sources?: SourceRef[]
  logId?: string
  feedback?: 'up' | 'down' | 'none'
  streaming?: boolean
}

// QAPage renders an ongoing conversation. Each question is sent to the SSE
// endpoint; answer tokens append live, sources appear on completion, and the
// user can copy the answer or vote. History is folded into the conversation so
// follow-up context carries over.
export default function QAPage() {
  const { kbId = '' } = useParams()
  const navigate = useNavigate()
  const { message } = AntdApp.useApp()
  const [input, setInput] = useState('')
  const [turns, setTurns] = useState<Turn[]>([])
  const [asking, setAsking] = useState(false)
  const [history, setHistory] = useState<QALog[]>([])
  const scrollRef = useRef<HTMLDivElement>(null)

  // Load recent history once so the user sees their prior questions on entry.
  useEffect(() => {
    qaApi.history().then(setHistory).catch(() => { /* non-fatal */ })
  }, [])

  // Auto-scroll to the latest turn when new content arrives.
  useEffect(() => {
    scrollRef.current?.scrollTo({ top: scrollRef.current.scrollHeight, behavior: 'smooth' })
  }, [turns])

  const ask = async () => {
    const question = input.trim()
    if (!question || asking) return
    setInput('')
    setAsking(true)
    // Seed the new turn with an empty answer we append to as tokens arrive.
    const turnIndex = turns.length
    setTurns((prev) => [...prev, { question, answer: '', streaming: true }])

    // Build follow-up history from prior turns so the model has context.
    const history = turns
      .filter((t) => t.answer && !t.streaming)
      .map((t) => ({ question: t.question, answer: t.answer }))

    const onDelta = (tok: string) => {
      setTurns((prev) => {
        const next = [...prev]
        next[turnIndex] = { ...next[turnIndex], answer: next[turnIndex].answer + tok }
        return next
      })
    }
    const onDone = (res: AskResult) => {
      setTurns((prev) => {
        const next = [...prev]
        next[turnIndex] = {
          ...next[turnIndex],
          answer: next[turnIndex].answer || res.answer,
          sources: res.sources,
          logId: res.log_id,
          feedback: 'none',
          streaming: false,
        }
        return next
      })
      setAsking(false)
    }
    const onError = (msg: string) => {
      setTurns((prev) => {
        const next = [...prev]
        next[turnIndex] = {
          ...next[turnIndex],
          answer: next[turnIndex].answer || `错误：${msg}`,
          streaming: false,
        }
        return next
      })
      setAsking(false)
    }

    await askStream({ kb_id: kbId, question, history }, onDelta, onDone, onError)
  }

  const vote = async (turn: Turn, idx: number, feedback: 'up' | 'down') => {
    if (!turn.logId) return
    const newFb = turn.feedback === feedback ? 'none' : feedback
    try {
      await qaApi.feedback(turn.logId, newFb === 'none' ? 'up' : feedback)
      setTurns((prev) => {
        const next = [...prev]
        next[idx] = { ...next[idx], feedback: newFb }
        return next
      })
    } catch (e) {
      message.error((e as Error).message)
    }
  }

  const copy = (text: string) => {
    void navigator.clipboard.writeText(text)
    message.success('已复制')
  }

  return (
    <Card
      style={{ height: 'calc(100vh - 112px)', display: 'flex', flexDirection: 'column' }}
      title={
        <Space direction="vertical" size={0}>
          <Breadcrumb items={[
            { title: <a onClick={() => navigate('/kb')}>知识库</a> },
            { title: '智能问答' },
          ]} />
          <Typography.Title level={4} style={{ margin: 0 }}>智能问答</Typography.Title>
        </Space>
      }
      bodyStyle={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
    >
      <div ref={scrollRef} style={{ flex: 1, overflowY: 'auto', paddingRight: 8 }}>
        {turns.length === 0 ? (
          <Empty description={history.length ? '从历史记录中选择，或在下方输入你的问题' : '在下方输入你的问题开始问答'} style={{ marginTop: 64 }}>
            {history.length > 0 && (
              <List
                size="small"
                bordered
                style={{ marginTop: 16, textAlign: 'left' }}
                dataSource={history.slice(0, 5)}
                renderItem={(h) => (
                  <List.Item>
                    <a onClick={() => setInput(h.question)}>{h.question}</a>
                  </List.Item>
                )}
              />
            )}
          </Empty>
        ) : (
          turns.map((t, idx) => (
            <div key={idx} style={{ marginBottom: 24 }}>
              <div style={{ textAlign: 'right' }}>
                <Tag color="blue" style={{ maxWidth: '80%', whiteSpace: 'pre-wrap', textAlign: 'left', padding: '8px 12px' }}>
                  {t.question}
                </Tag>
              </div>
              <div style={{ marginTop: 8 }}>
                <Space align="start" style={{ width: '100%' }}>
                  <Spin spinning={!!t.streaming && !t.answer} size="small" />
                  <div style={{ flex: 1 }}>
                    <div className="answer-stream">{t.answer}</div>
                    {t.sources && t.sources.length > 0 && (
                      <div style={{ marginTop: 8 }}>
                        <Typography.Text type="secondary">来源引用：</Typography.Text>
                        {t.sources.map((s, i) => (
                          <div className="source-card" key={i}>
                            <Typography.Text strong>[{i + 1}] {s.document_name}</Typography.Text>
                            <div style={{ color: '#666', fontSize: 13 }}>{s.snippet}</div>
                          </div>
                        ))}
                      </div>
                    )}
                    {!t.streaming && t.answer && (
                      <Space style={{ marginTop: 8 }}>
                        <Tooltip title="复制"><Button size="small" icon={<CopyOutlined />} onClick={() => copy(t.answer)} /></Tooltip>
                        <Tooltip title="赞">
                          <Button size="small" type={t.feedback === 'up' ? 'primary' : 'default'} icon={<LikeOutlined />} onClick={() => vote(t, idx, 'up')} />
                        </Tooltip>
                        <Tooltip title="踩">
                          <Button size="small" type={t.feedback === 'down' ? 'primary' : 'default'} danger icon={<DislikeOutlined />} onClick={() => vote(t, idx, 'down')} />
                        </Tooltip>
                      </Space>
                    )}
                  </div>
                </Space>
              </div>
            </div>
          ))
        )}
      </div>
      <Space.Compact style={{ marginTop: 12 }}>
        <Input
          placeholder="输入你的问题，回车发送…"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onPressEnter={ask}
          disabled={asking}
        />
        <Button type="primary" onClick={ask} loading={asking} style={{ width: 100 }}>发送</Button>
      </Space.Compact>
    </Card>
  )
}
