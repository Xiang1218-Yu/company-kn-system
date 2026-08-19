import api from './client'
import type { Document, DocStatus } from '../types'

export interface DocStatusResponse {
  status: DocStatus
  chunk_count: number
}

// Documents API. Upload posts multipart form data; the others are JSON.
export const documentApi = {
  upload: (kbId: string, file: File) => {
    const form = new FormData()
    form.append('file', file)
    return api.post<unknown, Document>(`/kb/${kbId}/documents`, form, {
      headers: { 'Content-Type': 'multipart/form-data' },
    })
  },
  list: (kbId: string) => api.get<unknown, Document[]>(`/kb/${kbId}/documents`),
  get: (id: string) => api.get<unknown, Document>(`/documents/${id}`),
  status: (id: string) => api.get<unknown, DocStatusResponse>(`/documents/${id}/status`),
  remove: (id: string) => api.delete<unknown, void>(`/documents/${id}`),
  download: (id: string) =>
    api.get<unknown, Blob>(`/documents/${id}/download`, { responseType: 'blob' }),
  search: (kbId: string, q: string) =>
    api.get<unknown, Document[]>('/search', { params: { kbId, q } }),
  reindex: (id: string) => api.post<unknown, { status: string }>(`/documents/${id}/reindex`),
}

// downloadUrl builds a URL the browser can fetch directly (e.g. in an <a>), with
// the token appended. Used for file downloads that should open in a new tab
// rather than go through axios.
export function downloadUrl(id: string): string {
  const token = localStorage.getItem('kn_token') ?? ''
  return `/api/v1/documents/${id}/download?token=${encodeURIComponent(token)}`
}
