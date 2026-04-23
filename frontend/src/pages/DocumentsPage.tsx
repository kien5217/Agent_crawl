import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { Document, Topic } from '../api/types'
import styles from './Page.module.css'

export default function DocumentsPage() {
  const [topics, setTopics] = useState<Topic[]>([])
  const [docs, setDocs] = useState<Document[]>([])
  const [topic, setTopic] = useState('all')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [scheduling, setScheduling] = useState(false)
  const [scheduleMsg, setScheduleMsg] = useState<string | null>(null)

  useEffect(() => {
    api.listTopics().then(setTopics).catch(() => {})
  }, [])

  useEffect(() => {
    setLoading(true)
    setError(null)
    api
      .listDocuments(topic)
      .then(setDocs)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false))
  }, [topic])

  async function handleSchedule() {
    setScheduling(true)
    setScheduleMsg(null)
    try {
      const result = await api.triggerSchedule()
      const counts = Object.entries(result.Counts)
        .map(([k, v]) => `${k}: ${v}`)
        .join(', ')
      setScheduleMsg(`Schedule done — ${counts || 'no new items'}`)
    } catch (e) {
      setScheduleMsg(`Error: ${(e as Error).message}`)
    } finally {
      setScheduling(false)
    }
  }

  return (
    <div>
      <div className={styles.pageHeader}>
        <h2>Documents</h2>
        <button
          className={styles.btn}
          onClick={handleSchedule}
          disabled={scheduling}
        >
          {scheduling ? 'Running…' : 'Run Schedule'}
        </button>
      </div>

      {scheduleMsg && <p className={styles.info}>{scheduleMsg}</p>}

      <div className={styles.filters}>
        <label>
          Topic:&nbsp;
          <select value={topic} onChange={(e) => setTopic(e.target.value)}>
            <option value="all">All</option>
            {topics.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name}
              </option>
            ))}
          </select>
        </label>
      </div>

      {loading && <p className={styles.muted}>Loading…</p>}
      {error && <p className={styles.error}>{error}</p>}

      {!loading && !error && (
        <table className={styles.table}>
          <thead>
            <tr>
              <th>Title</th>
              <th>Topic</th>
              <th>Source</th>
              <th>Published</th>
            </tr>
          </thead>
          <tbody>
            {docs.map((d) => (
              <tr key={d.ID}>
                <td>
                  <Link to={`/documents/${d.ID}`}>{d.Title || '(no title)'}</Link>
                </td>
                <td>{d.TopicID}</td>
                <td>{d.SourceID}</td>
                <td>{d.PublishedAt ? new Date(d.PublishedAt).toLocaleDateString() : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {!loading && !error && docs.length === 0 && (
        <p className={styles.muted}>No documents found.</p>
      )}
    </div>
  )
}
