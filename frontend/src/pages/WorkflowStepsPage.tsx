import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { StepExecution } from '../api/types'
import styles from './Page.module.css'

export default function WorkflowStepsPage() {
  const { id } = useParams<{ id: string }>()
  const [steps, setSteps] = useState<StepExecution[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    api
      .listSteps(id)
      .then(setSteps)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false))
  }, [id])

  return (
    <div>
      <Link to="/workflows" className={styles.back}>← Workflows</Link>
      <h2 className={styles.pageTitle}>Steps for <code>{id?.slice(0, 8)}…</code></h2>
      {loading && <p className={styles.muted}>Loading…</p>}
      {error && <p className={styles.error}>{error}</p>}
      {!loading && !error && (
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Name</th>
              <th>Status</th>
              <th>Started</th>
              <th>Finished</th>
              <th>Error</th>
            </tr>
          </thead>
          <tbody>
            {steps.map((s) => (
              <tr key={s.ID}>
                <td>{s.Name}</td>
                <td>{s.Status}</td>
                <td>{new Date(s.StartedAt).toLocaleString()}</td>
                <td>{s.FinishedAt ? new Date(s.FinishedAt).toLocaleString() : '—'}</td>
                <td>{s.Error ?? '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {!loading && !error && steps.length === 0 && (
        <p className={styles.muted}>No steps found.</p>
      )}
    </div>
  )
}
