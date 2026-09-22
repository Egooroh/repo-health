import { useCallback, useEffect, useRef, useState } from 'react'
import { ApiError, setPersonalUnauthorizedHandler } from '../api/client'
import { connectSession, disconnectSession, fetchSession, type SessionStatus } from '../api/session'
import { useToast } from '../components/Toast/ToastProvider'

export type SessionScope = 'public' | 'personal'

const CHANNEL_NAME = 'repo-health-access'

export interface UseSessionResult {
  connected: boolean
  expiresAt: string | undefined
  scope: SessionScope
  connectPending: boolean
  connectError: string | null
  disconnectPending: boolean
  connect: (token: string) => Promise<boolean>
  resetConnectError: () => void
  disconnect: () => void
  setScope: (scope: SessionScope) => void
}

export function useSession(): UseSessionResult {
  const [connected, setConnected] = useState(false)
  const [expiresAt, setExpiresAt] = useState<string | undefined>(undefined)
  const [scope, setScopeState] = useState<SessionScope>('public')
  const [connectPending, setConnectPending] = useState(false)
  const [connectError, setConnectError] = useState<string | null>(null)
  const [disconnectPending, setDisconnectPending] = useState(false)

  const expiryTimerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  const channelRef = useRef<BroadcastChannel | null>(null)
  const { showToast } = useToast()

  const scheduleExpiry = useCallback(
    (value: string | undefined) => {
      clearTimeout(expiryTimerRef.current)
      if (!value) return
      const delay = Math.max(0, new Date(value).getTime() - Date.now())
      expiryTimerRef.current = setTimeout(() => {
        setConnected(false)
        setExpiresAt(undefined)
        setScopeState('public')
        showToast('Личная сессия завершена. Подключите токен снова.')
      }, delay)
    },
    [showToast],
  )

  const applyStatus = useCallback(
    (status: SessionStatus) => {
      setConnected(status.connected)
      setExpiresAt(status.expires_at)
      // A resync (tab focus, another tab's broadcast) never invents a scope —
      // it only drops back to 'public' once the session is actually gone.
      setScopeState((current) => (status.connected ? current : 'public'))
      scheduleExpiry(status.expires_at)
    },
    [scheduleExpiry],
  )

  const refresh = useCallback(() => {
    fetchSession()
      .then(applyStatus)
      .catch(() => {
        setConnected(false)
        setExpiresAt(undefined)
        setScopeState('public')
      })
  }, [applyStatus])

  useEffect(() => {
    refresh()
    return () => clearTimeout(expiryTimerRef.current)
    // Intentionally mount-only: later resyncs are driven by the listeners below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // Other browser tabs share the same session cookie — resync whenever one of
  // them connects or disconnects (ported from access.js's BroadcastChannel).
  useEffect(() => {
    if (typeof BroadcastChannel !== 'function') return
    const channel = new BroadcastChannel(CHANNEL_NAME)
    channel.onmessage = () => refresh()
    channelRef.current = channel
    return () => {
      channel.close()
      channelRef.current = null
    }
  }, [refresh])

  // Recheck when returning to a tab, in case another tab changed the cookie
  // while this one was hidden or bfcache-restored.
  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState === 'visible') refresh()
    }
    const onPageShow = (event: PageTransitionEvent) => {
      if (event.persisted) refresh()
    }
    document.addEventListener('visibilitychange', onVisible)
    window.addEventListener('pageshow', onPageShow)
    return () => {
      document.removeEventListener('visibilitychange', onVisible)
      window.removeEventListener('pageshow', onPageShow)
    }
  }, [refresh])

  // A 401 on a personal-scoped request means the cookie already expired
  // server-side — drop the connected flag without waiting for a resync.
  useEffect(() => {
    setPersonalUnauthorizedHandler(() => {
      setConnected(false)
      setExpiresAt(undefined)
      setScopeState('public')
    })
    return () => setPersonalUnauthorizedHandler(null)
  }, [])

  const resetConnectError = useCallback(() => setConnectError(null), [])

  const connect = useCallback(
    async (token: string) => {
      setConnectError(null)
      setConnectPending(true)
      try {
        const result = await connectSession(token)
        setConnected(true)
        setExpiresAt(result.expires_at)
        setScopeState('personal')
        scheduleExpiry(result.expires_at)
        channelRef.current?.postMessage('changed')
        showToast('SourceCraft подключён. Теперь можно запустить личный анализ.')
        return true
      } catch (err) {
        setConnectError(err instanceof ApiError ? err.message : 'Не удалось подключить токен.')
        return false
      } finally {
        setConnectPending(false)
      }
    },
    [scheduleExpiry, showToast],
  )

  const disconnect = useCallback(() => {
    setDisconnectPending(true)
    disconnectSession()
      .then(() => {
        setConnected(false)
        setExpiresAt(undefined)
        setScopeState('public')
        scheduleExpiry(undefined)
        channelRef.current?.postMessage('changed')
        showToast('Токен отключён. Личные данные сессии удалены.')
      })
      .catch((err: unknown) => {
        showToast(err instanceof ApiError ? err.message : 'Не удалось отключить токен.')
      })
      .finally(() => setDisconnectPending(false))
  }, [scheduleExpiry, showToast])

  const setScope = useCallback(
    (next: SessionScope) => {
      if (next === 'personal' && !connected) return
      setScopeState(next)
    },
    [connected],
  )

  return { connected, expiresAt, scope, connectPending, connectError, disconnectPending, connect, resetConnectError, disconnect, setScope }
}
