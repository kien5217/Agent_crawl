import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { WorkflowExecution } from '../api/types'
import styles from './Page.module.css'

const statusColor: Record<string, string> = {
  running: '#f59e0b',
  done: '#22c55e',
  failed: '#ef4444',
  halted: '#6b7280',
}

export default function WorkflowsPage() {
  const [workflows, setWorkflows] = useState<WorkflowExecution[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    api
      .listWorkflows()
      .then((data) => setWorkflows(Array.isArray(data) ? data : []))
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  return (
    <div>
      <h2 className={styles.pageTitle}>Workflows</h2>
      {loading && <p className={styles.muted}>Loading…</p>}
      {error && <p className={styles.error}>{error}</p>}
      {!loading && !error && (
        <table className={styles.table}>
          <thead>
            <tr>
              <th>ID</th>
              <th>Name</th>
              <th>Status</th>
              <th>Started</th>
              <th>Finished</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {workflows.map((wf) => (
              <tr key={wf.ID}>
                <td><code>{wf.ID.slice(0, 8)}…</code></td>
                <td>{wf.Name}</td>
                <td>
                  <span style={{ color: statusColor[wf.Status] ?? '#222', fontWeight: 600 }}>
                    {wf.Status}
                  </span>
                </td>
                <td>{new Date(wf.StartedAt).toLocaleString()}</td>
                <td>{wf.FinishedAt ? new Date(wf.FinishedAt).toLocaleString() : '—'}</td>
                <td>
                  <Link to={`/workflows/${wf.ID}/steps`}>Steps</Link>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {!loading && !error && workflows.length === 0 && (
        <p className={styles.muted}>No workflows found.</p>
      )}
    </div>
  )
}
