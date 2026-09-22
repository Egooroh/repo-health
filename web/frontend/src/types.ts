export type SourceStatus = 'available' | 'unavailable' | 'truncated' | 'not_connected'

export interface SourceInfo {
  status: SourceStatus
  detail: string
}

export interface RepositoryInfo {
  ref: string
  description: string
  url: string
  language: string
  default_branch: string
  revision?: string
  empty: boolean
  fork: boolean
  visibility?: string
}

export interface Metric {
  id: string
  title: string
  weight: number
  score: number | null
  state: string
  evidence: string
}

export interface Category {
  id: string
  title: string
  weight: number
  score: number | null
  metrics: Metric[]
}

export interface Recommendation {
  metric_id: string
  priority: 'high' | 'medium'
  title: string
  reason: string
  action: string
}

export interface Report {
  repository: RepositoryInfo
  demo: boolean
  version: string
  status?: string
  analyzed_at: string
  score: number | null
  coverage: number
  eligible: boolean
  eligibility_reason: string
  categories: Category[]
  recommendations: Recommendation[]
  sources: Record<string, SourceInfo>
}

export interface Job {
  id: string
  repository_id: string
  ref: string
  demo: boolean
  status: 'queued' | 'running' | 'completed' | 'failed'
  created_at: string
  started_at?: string
  finished_at?: string
  error?: string
}

export interface Project {
  id: string
  ref: string
  demo: boolean
  report: Report | null
  job: Job | null
}
