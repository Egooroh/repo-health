import type { Job, Project } from '../types'
import { apiGet, apiPost } from './client'

export interface ProjectListResponse {
  items: Project[]
  count: number
}

export function fetchRepositories(demo: boolean, personal = false): Promise<ProjectListResponse> {
  return apiGet<ProjectListResponse>(`/repositories?demo=${demo}`, personal)
}

// The leaderboard is always public — old app.js forced state.personal=false here too.
export function fetchLeaderboard(demo: boolean): Promise<ProjectListResponse> {
  return apiGet<ProjectListResponse>(`/leaderboard?demo=${demo}`)
}

export function fetchRepository(id: string, personal = false): Promise<Project> {
  return apiGet<Project>(`/repositories/${encodeURIComponent(id)}`, personal)
}

export function createAnalysis(ref: string, demo: boolean, personal = false): Promise<Job> {
  return apiPost<Job>('/analyses', { ref, demo }, personal)
}

export function seedDemo(): Promise<{ status: string }> {
  return apiPost<{ status: string }>('/demo', {})
}
