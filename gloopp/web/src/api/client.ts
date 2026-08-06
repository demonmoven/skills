import type { ActivityResponse } from './types'
import i18n from '../i18n'

const TOKEN_KEY = 'gloop_token'

export class ApiError extends Error {
  status: number
  data: unknown

  constructor(message: string, status: number, data: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.data = data
  }
}

function readToken(): string {
  const url = new URL(window.location.href)
  const fromUrl = url.searchParams.get('t')
  if (fromUrl) {
    sessionStorage.setItem(TOKEN_KEY, fromUrl)
    return fromUrl
  }
  return sessionStorage.getItem(TOKEN_KEY) || ''
}

export function currentToken(): string {
  return readToken()
}

export function withToken(path: string): string {
  const t = readToken()
  if (!t) return path
  return path + (path.includes('?') ? '&' : '?') + 't=' + encodeURIComponent(t)
}

export type DownloadResult = {
  url: string
  filename?: string
}

function filenameFromDisposition(disposition: string | null): string | undefined {
  if (!disposition) return undefined
  const utf8 = disposition.match(/filename\*=UTF-8''([^;]+)/i)
  if (utf8?.[1]) {
    try {
      return decodeURIComponent(utf8[1].trim())
    } catch {
      return utf8[1].trim()
    }
  }
  const plain = disposition.match(/filename="?([^";]+)"?/i)
  return plain?.[1]?.trim()
}

async function request<T>(path: string, opts: RequestInit = {}): Promise<T> {
  const token = readToken()
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...(opts.headers as Record<string, string> | undefined),
  }
  if (token) headers.Authorization = 'Bearer ' + token

  const res = await fetch(path, { ...opts, headers })
  let data: unknown = null
  try {
    data = await res.json()
  } catch {
    throw new Error(i18n.t('apiError.responseParse', { status: res.status }))
  }
  const shaped = data as { ok?: boolean; error?: string }
  if (!res.ok || shaped.ok === false) {
    throw new ApiError(shaped.error || res.statusText || i18n.t('apiError.requestFailed'), res.status, data)
  }
  return data as T
}

async function requestForm<T>(path: string, body: FormData): Promise<T> {
  const token = readToken()
  const headers: Record<string, string> = {}
  if (token) headers.Authorization = 'Bearer ' + token
  const res = await fetch(path, { method: 'POST', body, headers })
  let data: unknown = null
  try {
    data = await res.json()
  } catch {
    throw new Error(i18n.t('apiError.responseParse', { status: res.status }))
  }
  const shaped = data as { ok?: boolean; error?: string }
  if (!res.ok || shaped.ok === false) {
    throw new ApiError(shaped.error || res.statusText || i18n.t('apiError.requestFailed'), res.status, data)
  }
  return data as T
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'POST', body: JSON.stringify(body ?? {}) }),
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'PUT', body: JSON.stringify(body ?? {}) }),
  patch: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'PATCH', body: JSON.stringify(body ?? {}) }),
  delete: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'DELETE', body: JSON.stringify(body ?? {}) }),
  postForm: <T>(path: string, body: FormData) => requestForm<T>(path, body),

  // Download helper: returns a blob URL plus server-provided filename when present.
  download: async (path: string): Promise<DownloadResult> => {
    const token = readToken()
    const headers: Record<string, string> = {}
    if (token) headers.Authorization = 'Bearer ' + token
    const res = await fetch(path, { headers })
    if (!res.ok) {
      throw new Error(res.statusText || i18n.t('apiError.downloadFailed'))
    }
    const contentType = res.headers.get('Content-Type') || ''
    if (contentType.includes('text/html')) {
      throw new Error(i18n.t('apiError.downloadReturnedHtml'))
    }
    const blob = await res.blob()
    return {
      url: URL.createObjectURL(blob),
      filename: filenameFromDisposition(res.headers.get('Content-Disposition')),
    }
  },
}

// fetchActivity 拉首页 Feed 平铺流（post=agent发言，X/Twitter 模型）。
// project 空=全部；limit 默认后端 50，上限 200。
export async function fetchActivity(project?: string, limit?: number): Promise<ActivityResponse> {
  const params = new URLSearchParams()
  if (project) params.set('project', project)
  if (limit) params.set('limit', String(limit))
  const qs = params.toString()
  return api.get<ActivityResponse>('/api/activity' + (qs ? '?' + qs : ''))
}
