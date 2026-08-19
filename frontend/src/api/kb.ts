import api from './client'
import type { KnowledgeBase } from '../types'

export interface KBInput {
  name: string
  description?: string
}

// Knowledge-base CRUD. The create/update bodies are typed so callers can't
// send the wrong shape.
export const kbApi = {
  list: () => api.get<unknown, KnowledgeBase[]>('/kb'),
  get: (id: string) => api.get<unknown, KnowledgeBase>(`/kb/${id}`),
  create: (body: KBInput) => api.post<unknown, KnowledgeBase>('/kb', body),
  update: (id: string, body: Partial<KBInput>) => api.put<unknown, KnowledgeBase>(`/kb/${id}`, body),
  remove: (id: string) => api.delete<unknown, void>(`/kb/${id}`),
  invite: (id: string, email: string) => api.post<unknown, unknown>(`/kb/${id}/invite`, { email }),
}
