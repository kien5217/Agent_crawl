// Shared TypeScript types matching Go models.

export interface Topic {
  id: string
  name: string
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
}

export interface WorkflowExecution {
  ID: string
  Name: string
  Status: string
  StartedAt: string
  FinishedAt: string | null
  Error: string | null
}

export interface StepExecution {
  ID: string
  WorkflowID: string
  Name: string
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
