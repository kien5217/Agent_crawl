import type { Document, ScheduleResult, StepExecution, Topic, WorkflowExecution } from './types'

const BASE = '/api'

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`)
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error((body as { error?: string }).error ?? `HTTP ${res.status}`)
  }
  return res.json() as Promise<T>
}

async function post<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`, { method: 'POST' })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error((body as { error?: string }).error ?? `HTTP ${res.status}`)
  }
  return res.json() as Promise<T>
}

export const api = {
  listTopics: () => get<Topic[]>('/topics'),

  listDocuments: (topic = 'all', limit = 50) =>
    get<Document[]>(`/documents?topic=${encodeURIComponent(topic)}&limit=${limit}`),

  getDocument: (id: number) => get<Document>(`/documents/${id}`),

  triggerSchedule: () => post<ScheduleResult>('/schedule'),

  listWorkflows: (limit = 20) => get<WorkflowExecution[]>(`/workflows?limit=${limit}`),

  listSteps: (workflowID: string) => get<StepExecution[]>(`/workflows/${workflowID}/steps`),
}
