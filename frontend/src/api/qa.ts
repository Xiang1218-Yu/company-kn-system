import api from './client'
import type { QALog, SourceRef } from '../types'

export interface AskRequest {
  kb_id: string
  question: string
  history?: { question: string; answer: string }[]
}

export interface AskResult {
  log_id: string
  answer: string
  sources: SourceRef[]
}

export interface SSEEvent {
  event: 'delta' | 'done' | 'error'
  data: string
}

// QA API. `ask` is non-streaming; `askStream` consumes the SSE endpoint and
// invokes callbacks per token and on completion. We parse SSE manually rather
// than relying on EventSource because EventSource cannot set the Authorization
// header and only supports GET — our ask endpoint is POST with a JSON body.
export const qaApi = {
  ask: (body: AskRequest) => api.post<unknown, AskResult>('/qa/ask', body),
  history: () => api.get<unknown, QALog[]>('/qa/history'),
  feedback: (logId: string, feedback: 'up' | 'down') =>
    api.post<unknown, void>('/qa/feedback', { log_id: logId, feedback }),
}

// askStream posts the question and consumes the SSE response. The fetch API is
// used (not axios) so we can read the response body as a stream chunk by chunk.
export async function askStream(
  body: AskRequest,
  onDelta: (text: string) => void,
  onDone: (result: AskResult) => void,
  onError: (message: string) => void,
): Promise<void> {
  const token = localStorage.getItem('kn_token') ?? ''
  try {
    const resp = await fetch('/api/v1/qa/ask', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Accept: 'text/event-stream',
        Authorization: `Bearer ${token}`,
      },
      body: JSON.stringify(body),
    })
    if (!resp.ok || !resp.body) {
      onError(`请求失败 (${resp.status})`)
      return
    }
    const reader = resp.body.getReader()
    const decoder = new TextDecoder()
    let buffer = ''
    // SSE framing: events separated by blank lines, each line "field: value".
    while (true) {
      const { done, value } = await reader.read()
      if (done) break
      buffer += decoder.decode(value, { stream: true })
      // Split on the event terminator \n\n; keep any trailing partial frame.
      let idx: number
      while ((idx = buffer.indexOf('\n\n')) !== -1) {
        const frame = buffer.slice(0, idx)
        buffer = buffer.slice(idx + 2)
        handleFrame(frame, onDelta, onDone, onError)
      }
    }
  } catch (e) {
    onError((e as Error).message)
  }
}

// handleFrame parses one SSE frame into an event + data and dispatches it.
function handleFrame(
  frame: string,
  onDelta: (text: string) => void,
  onDone: (result: AskResult) => void,
  onError: (message: string) => void,
) {
  let event = 'message'
  const dataLines: string[] = []
  for (const line of frame.split('\n')) {
    if (line.startsWith('event:')) {
      event = line.slice(6).trim()
    } else if (line.startsWith('data:')) {
      dataLines.push(line.slice(5).trim())
    }
  }
  const data = dataLines.join('')
  if (event === 'delta') {
    // delta data is a JSON-encoded string token
    try {
      onDelta(JSON.parse(data))
    } catch {
      onDelta(data)
    }
  } else if (event === 'done') {
    onDone(JSON.parse(data) as AskResult)
  } else if (event === 'error') {
    try {
      onError((JSON.parse(data) as { message: string }).message)
    } catch {
      onError(data)
    }
  }
}
