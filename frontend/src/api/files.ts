import type { DirectoryListing, MetaInfo, SearchResult, SessionInfo } from '../types/files'

export class APIError extends Error {
  constructor(message: string, public readonly code: string, public readonly status: number) {
    super(message)
  }
}

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, init)
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { error?: { code?: string; message?: string } } | null
    throw new APIError(body?.error?.message ?? `Request failed (${response.status})`, body?.error?.code ?? 'request_failed', response.status)
  }
  if (response.status === 204) return undefined as T
  return response.json() as Promise<T>
}

function json(method: string, body?: unknown): RequestInit {
  return { method, headers: { 'Content-Type': 'application/json' }, body: body === undefined ? undefined : JSON.stringify(body) }
}

export const contentURL = (path: string, disposition: 'inline' | 'attachment' = 'inline') =>
  `/api/v1/content?path=${encodeURIComponent(path)}&disposition=${disposition}`
export const archiveURL = (path: string) => `/api/v1/archive?path=${encodeURIComponent(path)}`
export const getMeta = () => request<MetaInfo>('/api/v1/meta')
export const getSession = (path = '/') => request<SessionInfo>(`/api/v1/session?path=${encodeURIComponent(path)}`)
export const listFiles = (path: string) => request<DirectoryListing>(`/api/v1/files?path=${encodeURIComponent(path)}`)
export const searchFiles = (path: string, query: string) => request<SearchResult>(`/api/v1/search?path=${encodeURIComponent(path)}&query=${encodeURIComponent(query)}`)
export const readText = async (path: string) => {
  const response = await fetch(contentURL(path, 'inline'))
  if (!response.ok) throw new APIError(`Request failed (${response.status})`, 'request_failed', response.status)
  return response.text()
}
export const login = (username: string, password: string) => request<SessionInfo>('/api/v1/session', json('POST', { username, password }))
export const logout = () => request<void>('/api/v1/session', { method: 'DELETE' })
export const createDirectory = (path: string) => request<{ path: string }>('/api/v1/directories', json('POST', { path }))
export const createText = (path: string, content: string) => request<{ path: string }>('/api/v1/text-files', json('POST', { path, content }))
export const updateText = (path: string, content: string) => request<{ path: string }>('/api/v1/text-files', json('PUT', { path, content }))
export const renameFile = (path: string, newName: string) => request<{ path: string }>('/api/v1/files', json('PATCH', { path, newName }))
export const deleteFile = (path: string) => request<void>(`/api/v1/files?path=${encodeURIComponent(path)}`, { method: 'DELETE' })

export interface UploadTask { promise: Promise<string[]>; abort: () => void }

export function uploadFiles(path: string, files: File[], onProgress: (percent: number) => void): UploadTask {
  const xhr = new XMLHttpRequest()
  const form = new FormData()
  files.forEach((file) => {
    form.append('file', file, file.name)
    form.append('relativePath', file.webkitRelativePath || file.name)
  })
  const promise = new Promise<string[]>((resolve, reject) => {
    xhr.upload.addEventListener('progress', (event) => {
      if (event.lengthComputable) onProgress(Math.round((event.loaded / event.total) * 100))
    })
    xhr.addEventListener('load', () => {
      const body = JSON.parse(xhr.responseText || '{}') as { paths?: string[]; error?: { code?: string; message?: string } }
      if (xhr.status >= 200 && xhr.status < 300) resolve(body.paths ?? [])
      else reject(new APIError(body.error?.message ?? `Request failed (${xhr.status})`, body.error?.code ?? 'request_failed', xhr.status))
    })
    xhr.addEventListener('error', () => reject(new APIError('Network error', 'network_error', 0)))
    xhr.addEventListener('abort', () => reject(new APIError('Upload canceled', 'upload_canceled', 0)))
    xhr.open('POST', `/api/v1/uploads?path=${encodeURIComponent(path)}`)
    xhr.send(form)
  })
  return { promise, abort: () => xhr.abort() }
}
