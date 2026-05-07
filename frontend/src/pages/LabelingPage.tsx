import { useCallback, useEffect, useState } from 'react'
import { api } from '../api/client'
import type { LabelQueueEntry, Topic } from '../api/types'
import styles from './Page.module.css'

export default function LabelingPage() {
  const [items, setItems] = useState<LabelQueueEntry[]>([])
  const [topics, setTopics] = useState<Topic[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [current, setCurrent] = useState(0)
  const [selectedTopic, setSelectedTopic] = useState('')
  const [busy, setBusy] = useState(false)
  const [actionMsg, setActionMsg] = useState<string | null>(null)

  const loadQueue = useCallback(() => {
    setLoading(true)
    setError(null)
    Promise.all([api.listLabelQueue(50), api.listTopics()])
      .then(([q, t]) => {
        setItems(q ?? [])
        setTopics(t)
        setCurrent(0)
        setSelectedTopic(q?.[0]?.TopicID ?? '')
      })
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    loadQueue()
  }, [loadQueue])

  const item = items[current] ?? null

  function advanceOrReload() {
    const next = current + 1
    if (next < items.length) {
      setCurrent(next)
      setSelectedTopic(items[next].TopicID)
      setActionMsg(null)
    } else {
      setActionMsg('All items reviewed! Reloading queue…')
      loadQueue()
    }
  }

  async function handleLabel() {
    if (!item || !selectedTopic) return
    setBusy(true)
    try {
      await api.submitLabel(item.ID, selectedTopic)
      advanceOrReload()
    } catch (e: unknown) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  async function handleSkip() {
    if (!item) return
    setBusy(true)
    try {
      await api.skipLabelQueue(item.ID)
      advanceOrReload()
    } catch (e: unknown) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  if (loading) return <p className={styles.muted}>Loading label queue…</p>
  if (error) return <p className={styles.error}>{error}</p>

  if (!item) {
    return (
      <div className={styles.detail}>
        <h2 className={styles.pageTitle}>Labeling</h2>
        <p className={styles.muted}>{actionMsg ?? 'No pending items in the label queue.'}</p>
      </div>
    )
  }

  const progress = `${current + 1} / ${items.length}`

  return (
    <div className={styles.detail}>
      <div className={styles.pageHeader}>
        <h2 className={styles.pageTitle} style={{ margin: 0 }}>Labeling</h2>
        <span className={styles.badge}>{progress} pending</span>
      </div>

      {actionMsg && <p className={styles.muted}>{actionMsg}</p>}

      <div className={styles.labelCard}>
        <div className={styles.labelMeta}>
          <span className={styles.chip}>Margin {item.Margin.toFixed(3)}</span>
          {item.Reason && <span className={styles.chip}>{item.Reason}</span>}
          <span className={styles.chip}>Current topic: {item.TopicID || '—'}</span>
        </div>

        <h3 className={styles.labelTitle}>
          <a href={item.URL} target="_blank" rel="noreferrer">{item.Title || item.URL}</a>
        </h3>

        <pre className={styles.labelContent}>{item.ContentText.slice(0, 800)}{item.ContentText.length > 800 ? '…' : ''}</pre>

        <div className={styles.labelActions}>
          <div className={styles.filterField}>
            <span>Assign topic</span>
            <select
              value={selectedTopic}
              onChange={e => setSelectedTopic(e.target.value)}
              disabled={busy}
            >
              <option value="">— select —</option>
              {topics.map(t => (
                <option key={t.id} value={t.id}>{t.name}</option>
              ))}
            </select>
          </div>

          <button
            className={styles.btnPrimary}
            onClick={handleLabel}
            disabled={busy || !selectedTopic}
          >
            ✓ Approve
          </button>

          <button
            className={styles.btnSecondary}
            onClick={handleSkip}
            disabled={busy}
          >
            Skip
          </button>
        </div>
      </div>
    </div>
  )
}
