import { useCallback, useState } from 'react'
import { ApiError } from '../api/client'
import { seedDemo } from '../api/repositories'
import { useToast } from '../components/Toast/ToastProvider'

const STORAGE_KEY = 'repo-health-demo'

function readInitial(): boolean {
  try {
    return sessionStorage.getItem(STORAGE_KEY) === 'true'
  } catch {
    return false
  }
}

function persist(value: boolean) {
  try {
    sessionStorage.setItem(STORAGE_KEY, String(value))
  } catch {
    // Private browsing / storage disabled — demo mode just won't survive a reload.
  }
}

interface UseDemoModeResult {
  demoMode: boolean
  pending: boolean
  toggleDemo: () => void
  // Syncs local demo state from a fetched project without re-seeding the
  // server catalog — used when a demo repository is opened via a direct link.
  syncDemoMode: (demo: boolean) => void
}

export function useDemoMode(): UseDemoModeResult {
  const [demoMode, setDemoMode] = useState(readInitial)
  const [pending, setPending] = useState(false)
  const { showToast } = useToast()

  const syncDemoMode = useCallback((demo: boolean) => {
    setDemoMode(demo)
    persist(demo)
  }, [])

  const toggleDemo = useCallback(() => {
    if (demoMode) {
      syncDemoMode(false)
      return
    }
    setPending(true)
    seedDemo()
      .then(() => syncDemoMode(true))
      .catch((err: unknown) => {
        showToast(err instanceof ApiError ? err.message : 'Не удалось включить демо-режим.')
      })
      .finally(() => setPending(false))
  }, [demoMode, syncDemoMode, showToast])

  return { demoMode, pending, toggleDemo, syncDemoMode }
}
