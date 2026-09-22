import { useCallback, useEffect, useRef, useState } from 'react'
import { fetchLeaderboard, fetchRepositories } from '../api/repositories'
import { ApiError } from '../api/client'
import type { Project } from '../types'
import { isPendingProject } from '../utils/format'

const POLL_INTERVAL_MS = 1200

export type ProjectListKind = 'repositories' | 'leaderboard'

interface UseProjectsResult {
  items: Project[]
  count: number
  loading: boolean
  error: string | null
  reload: () => void
}

export function useProjects(kind: ProjectListKind, demo: boolean, personal: boolean): UseProjectsResult {
  const [items, setItems] = useState<Project[]>([])
  const [count, setCount] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  // A generation counter, bumped on every reload/dependency change, lets an
  // in-flight fetch or a pending poll timer recognize it's stale and bail
  // out instead of clobbering newer state (ports app.js's `state.serial`).
  const serialRef = useRef(0)
  const timerRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined)
  // Holds the latest `load` so the poll timer below can call it without
  // closing over `load` from within its own not-yet-finished initializer.
  const loadRef = useRef<() => void>(() => {})

  const load = useCallback(() => {
    const serial = ++serialRef.current
    clearTimeout(timerRef.current)
    const fetchProjects = kind === 'leaderboard' ? fetchLeaderboard(demo) : fetchRepositories(demo, personal)
    fetchProjects
      .then((data) => {
        if (serial !== serialRef.current) return
        setItems(data.items)
        setCount(data.count)
        setError(null)
        setLoading(false)
        if (data.items.some(isPendingProject)) {
          timerRef.current = setTimeout(() => loadRef.current(), POLL_INTERVAL_MS)
        }
      })
      .catch((err: unknown) => {
        if (serial !== serialRef.current) return
        setLoading(false)
        setError(err instanceof ApiError ? err.message : 'Не удалось выполнить запрос.')
      })
  }, [kind, demo, personal])

  useEffect(() => {
    loadRef.current = load
  }, [load])

  useEffect(() => {
    setLoading(true)
    setItems([])
    setCount(0)
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

  return { items, count, loading, error, reload }
}
