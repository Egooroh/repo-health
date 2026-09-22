import { apiGet, apiPost } from './client'

export interface SessionStatus {
  connected: boolean
  expires_at?: string
  appsec_connected: boolean
}

export function fetchSession(): Promise<SessionStatus> {
  return apiGet<SessionStatus>('/session')
}

export function connectSession(token: string): Promise<SessionStatus> {
  return apiPost<SessionStatus>('/session', { token })
}

export function disconnectSession(): Promise<{ connected: boolean }> {
  return apiPost<{ connected: boolean }>('/session/disconnect', {})
}
