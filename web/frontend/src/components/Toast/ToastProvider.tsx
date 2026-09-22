import { createContext, useCallback, useContext, useRef, useState, type ReactNode } from 'react'
import styles from './Toast.module.scss'

const DISMISS_MS = 4500

interface ToastContextValue {
  showToast: (message: string) => void
}

const ToastContext = createContext<ToastContextValue | null>(null)

export function ToastProvider({ children }: { children: ReactNode }) {
  const [message, setMessage] = useState<string | null>(null)
  const timerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)

  const showToast = useCallback((next: string) => {
    clearTimeout(timerRef.current)
    setMessage(next)
    timerRef.current = setTimeout(() => setMessage(null), DISMISS_MS)
  }, [])

  return (
    <ToastContext.Provider value={{ showToast }}>
      {children}
      {message && (
        <div className={styles.toast} role="status">
          {message}
        </div>
      )}
    </ToastContext.Provider>
  )
}

export function useToast(): ToastContextValue {
  const ctx = useContext(ToastContext)
  if (!ctx) throw new Error('useToast must be used within a ToastProvider')
  return ctx
}
