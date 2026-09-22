import type { Job, Project } from '../types'

export function formatNumber(value: number | null | undefined): string {
  return value == null ? '—' : new Intl.NumberFormat('ru-RU', { maximumFractionDigits: 1 }).format(value)
}

export function formatDate(value: string | null | undefined): string {
  if (!value) return 'Ещё не анализировался'
  return new Intl.DateTimeFormat('ru-RU', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }).format(new Date(value))
}

export type ScoreTone = 'green' | 'amber' | 'red' | null

export function scoreTone(score: number | null | undefined): ScoreTone {
  if (score == null) return null
  if (score >= 80) return 'green'
  if (score >= 50) return 'amber'
  return 'red'
}

export function scoreLabel(score: number | null | undefined): string {
  if (score == null) return 'Недостаточно данных'
  if (score >= 80) return 'Хорошее состояние'
  if (score >= 50) return 'Есть что улучшить'
  return 'Требует внимания'
}

export function isPending(job: Job | null | undefined): boolean {
  return job != null && (job.status === 'queued' || job.status === 'running')
}

export function isPendingProject(project: Project): boolean {
  return isPending(project.job)
}
