import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '../api/client'
import type { Document, Topic } from '../api/types'
import styles from './Page.module.css'

export default function DocumentsPage() {
  const [topics, setTopics] = useState<Topic[]>([])
  const [docs, setDocs] = useState<Document[]>([])
  const [topic, setTopic] = useState('all')
  const [source, setSource] = useState('')
  const [fromDate, setFromDate] = useState('')
  const [toDate, setToDate] = useState('')
  const [limitChoice, setLimitChoice] = useState('auto')
  const [appliedTopic, setAppliedTopic] = useState('all')
  const [appliedSource, setAppliedSource] = useState('')
  const [appliedFromDate, setAppliedFromDate] = useState('')
  const [appliedToDate, setAppliedToDate] = useState('')
  const [appliedLimitChoice, setAppliedLimitChoice] = useState('auto')
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
      .listDocuments({
        topic: appliedTopic,
        source: appliedSource,
        fromDate: appliedFromDate,
        toDate: appliedToDate,
        limit: appliedLimitChoice === 'auto' ? undefined : Number(appliedLimitChoice),
      })
      .then(setDocs)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false))
  }, [appliedTopic, appliedSource, appliedFromDate, appliedToDate, appliedLimitChoice])

  function applyFilters() {
    setAppliedTopic(topic)
    setAppliedSource(source.trim())
    setAppliedFromDate(fromDate)
    setAppliedToDate(toDate)
    setAppliedLimitChoice(limitChoice)
  }

  function clearFilters() {
    setTopic('all')
    setSource('')
    setFromDate('')
    setToDate('')
    setLimitChoice('auto')
    setAppliedTopic('all')
    setAppliedSource('')
    setAppliedFromDate('')
    setAppliedToDate('')
    setAppliedLimitChoice('auto')
  }

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
        <label className={styles.filterField}>
          <span>Topic</span>
          <select value={topic} onChange={(e) => setTopic(e.target.value)}>
            <option value="all">All</option>
            {topics.map((t) => (
              <option key={t.id} value={t.id}>
                {t.name}
              </option>
            ))}
          </select>
        </label>

        <label className={styles.filterField}>
          <span>Source</span>
          <input
            type="text"
            placeholder="securityweek"
            value={source}
            onChange={(e) => setSource(e.target.value)}
          />
        </label>

        <label className={styles.filterField}>
          <span>From date</span>
          <input type="date" value={fromDate} onChange={(e) => setFromDate(e.target.value)} />
        </label>

        <label className={styles.filterField}>
          <span>To date</span>
          <input type="date" value={toDate} onChange={(e) => setToDate(e.target.value)} />
        </label>

        <label className={styles.filterField}>
          <span>Limit</span>
          <select value={limitChoice} onChange={(e) => setLimitChoice(e.target.value)}>
            <option value="auto">Auto</option>
            <option value="0">All</option>
            <option value="50">50</option>
            <option value="100">100</option>
            <option value="200">200</option>
          </select>
        </label>

        <div className={styles.filterActions}>
          <button className={styles.btn} onClick={applyFilters}>Apply</button>
          <button className={styles.btnSecondary} onClick={clearFilters}>Clear</button>
        </div>
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
