import type { Document, HealthStats, ScheduleResult, StepExecution, Topic, WorkflowExecution } from './types'

const BASE = '/api'
const API_KEY = import.meta.env.VITE_API_KEY as string | undefined
const API_BEARER = import.meta.env.VITE_API_BEARER_TOKEN as string | undefined

type ListDocumentsParams = {
  topic?: string
  source?: string
  fromDate?: string
  toDate?: string
  mlConfidenceMin?: number
  limit?: number
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(`${BASE}${path}`)
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error((body as { error?: string }).error ?? `HTTP ${res.status}`)
  }
  return res.json() as Promise<T>
}

async function post<T>(path: string): Promise<T> {
  const headers: Record<string, string> = {}
  if (API_KEY && API_KEY.trim() !== '') {
    headers['X-API-Key'] = API_KEY
  }
  if (API_BEARER && API_BEARER.trim() !== '') {
    headers.Authorization = `Bearer ${API_BEARER}`
  }

  const res = await fetch(`${BASE}${path}`, {
    method: 'POST',
    headers,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error((body as { error?: string }).error ?? `HTTP ${res.status}`)
  }
  return res.json() as Promise<T>
}

export const api = {
  listTopics: () => get<Topic[]>('/topics'),

  listDocuments: ({
    topic = 'all',
    source,
    fromDate,
    toDate,
    mlConfidenceMin,
    limit,
  }: ListDocumentsParams = {}) => {
    const qs = new URLSearchParams()
    qs.set('topic', topic)
    if (typeof limit === 'number' && limit >= 0) qs.set('limit', String(limit))
    if (source && source.trim() !== '') qs.set('source', source)
    if (fromDate && fromDate.trim() !== '') qs.set('from_date', fromDate)
    if (toDate && toDate.trim() !== '') qs.set('to_date', toDate)
    if (typeof mlConfidenceMin === 'number') qs.set('ml_confidence_min', String(mlConfidenceMin))
    return get<Document[]>(`/documents?${qs.toString()}`)
  },

  getDocument: (id: number) => get<Document>(`/documents/${id}`),

  triggerSchedule: () => post<ScheduleResult>('/schedule'),

  listWorkflows: (limit = 20) => get<WorkflowExecution[]>(`/workflows?limit=${limit}`),

  listSteps: (workflowID: string) => get<StepExecution[]>(`/workflows/${workflowID}/steps`),

  getHealth: () => get<HealthStats>('/health'),
}
