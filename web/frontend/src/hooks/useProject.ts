import { useCallback, useEffect, useRef, useState } from 'react'
import { fetchRepository } from '../api/repositories'
import { ApiError } from '../api/client'
import type { Project } from '../types'
import { isPendingProject } from '../utils/format'

const POLL_INTERVAL_MS = 1200

interface UseProjectResult {
  project: Project | null
  loading: boolean
  error: string | null
  notFound: boolean
  reload: () => void
}

export function useProject(id: string | undefined, personal: boolean): UseProjectResult {
  const [project, setProject] = useState<Project | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [notFound, setNotFound] = useState(false)

  const serialRef = useRef(0)
  const timerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  // Holds the latest `load` so the poll timer below can call it without
  // closing over `load` from within its own not-yet-finished initializer.
  const loadRef = useRef<() => void>(() => {})

  const load = useCallback(() => {
    if (!id) return
    const serial = ++serialRef.current
    clearTimeout(timerRef.current)
    fetchRepository(id, personal)
      .then((data) => {
        if (serial !== serialRef.current) return
        setProject(data)
        setNotFound(false)
        setError(null)
        setLoading(false)
        if (isPendingProject(data)) {
          timerRef.current = setTimeout(() => loadRef.current(), POLL_INTERVAL_MS)
        }
      })
      .catch((err: unknown) => {
        if (serial !== serialRef.current) return
        setLoading(false)
        if (err instanceof ApiError && err.status === 404) {
          setProject(null)
          setNotFound(true)
          return
        }
        setError(err instanceof ApiError ? err.message : 'Не удалось выполнить запрос.')
      })
  }, [id, personal])

  useEffect(() => {
    loadRef.current = load
  }, [load])

  useEffect(() => {
    setLoading(true)
    setProject(null)
    setNotFound(false)
    setError(null)
    load()
    return () => {
      ++serialRef.current
      clearTimeout(timerRef.current)
    }
  }, [load])

  const reload = useCallback(() => {
    setLoading(true)
    load()
  }, [load])

  return { project, loading, error, notFound, reload }
}
