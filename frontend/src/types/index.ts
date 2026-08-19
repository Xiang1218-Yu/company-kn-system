// Shared API domain types. These mirror the Go model structs so the frontend
// and backend agree on shapes without a codegen step. Keep field names in sync
// with the JSON tags in internal/model.

export type Role = 'admin' | 'manager' | 'member'

export interface User {
  id: string
  email: string
  name: string
  role: Role
  created_at: string
  updated_at: string
}

export interface KnowledgeBase {
  id: string
  name: string
  description: string
  team_id?: string
  created_by: string
  created_at: string
  updated_at: string
}

export type DocStatus = 'pending' | 'indexing' | 'indexed' | 'failed'

export interface Document {
  id: string
  kb_id: string
  name: string
  file_path: string
  file_size: number
  file_type: string
  status: DocStatus
  chunk_count: number
  uploaded_by: string
  created_at: string
  updated_at: string
}

export interface SourceRef {
  document_id: string
  document_name: string
  snippet: string
  chunk_index: number
}

export interface QALog {
  id: string
  user_id: string
  kb_id: string
  question: string
  answer: string
  sources: SourceRef[]
  feedback: 'up' | 'down' | 'none'
  response_time_ms: number
  created_at: string
}

export interface Dashboard {
  document_count: number
  qa_count: number
  satisfaction: number
}

// The envelope returned by every endpoint. `data` is present on success,
// `error` carries a machine code + message on failure.
export interface APIEnvelope<T> {
  success: boolean
  data?: T
  error?: { code: string; message: string }
}

export interface AuthResponse {
  user: User
  token: string
}
