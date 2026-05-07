// Shared TypeScript types matching Go models.

export interface Topic {
  id: string
  name: string
}

export interface Source {
  id: string
  name: string
  url: string
  enabled: boolean
  schedule_freq: string
  topic_ids: string[]
}

export interface Document {
  ID: number
  Title: string
  URL: string
  CanonicalURL: string
  TopicID: string
  PublishedAt: string | null
  ContentText: string
  SourceID: string
  Keywords: string[]
}

export interface WorkflowExecution {
ID: string
Name?: string
WorkflowName?: string
Status: string
StartedAt: string
FinishedAt: string | null
Error: string | null
}

export interface StepExecution {
ID: string
WorkflowID: string
Name?: string
StepName?: string
Status: string
StartedAt: string
FinishedAt: string | null
Error: string | null
}

export interface ScheduleResult {
  Counts: Record<string, number>
}

export interface HealthStats {
  queue_size: number
  last_crawl_time: string | null
  source_fail_count: number
}

export interface LabelQueueEntry {
  ID: number
  DocumentID: number
  Status: string
  Reason: string
  Margin: number
  CreatedAt: string
  Title: string
  URL: string
  TopicID: string
  ContentText: string
}

export interface DashboardOverview {
  counts_by_day: Array<{
    date: string
    topic_id: string
    topic_name: string
    count: number
  }>
  counts_by_topic: Array<{
    topic_id: string
    topic_name: string
    count: number
  }>
  top_sources: Array<{
    source_id: string
    name: string
    count: number
  }>
  fail_rate: number
  sla_sources: Array<{
    source_id: string
    name: string
    enabled: boolean
    last_post_at: string | null
    fail_count: number
    days_since_last_post: number
    stale: boolean
  }>
}

export interface NearDuplicateGroup {
  docs: Array<{
    ID: number
    Title: string
    URL: string
    SourceID: string
    PublishedAt: string | null
  }>
  max_distance: number
}
