import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { HealthStats } from '../api/types'
import styles from './Page.module.css'

function StatusBadge({ ok }: { ok: boolean }) {
  return (
    <span
      style={{
        display: 'inline-block',
        padding: '2px 10px',
        borderRadius: 12,
        fontSize: 12,
        fontWeight: 600,
        background: ok ? '#d1fae5' : '#fee2e2',
        color: ok ? '#065f46' : '#991b1b',
      }}
    >
      {ok ? 'OK' : 'WARN'}
    </span>
  )
}

export default function HealthPage() {
  const [stats, setStats] = useState<HealthStats | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [refreshedAt, setRefreshedAt] = useState<Date>(new Date())

  useEffect(() => {
    setLoading(true)
    api
      .getHealth()
      .then((s) => {
        setStats(s)
        setError(null)
      })
      .catch((e: unknown) => setError(e instanceof Error ? e.message : 'Unknown error'))
      .finally(() => setLoading(false))
  }, [refreshedAt])

  return (
    <div className={styles.page}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 24 }}>
        <h2 style={{ margin: 0 }}>System Health</h2>
        <button
          onClick={() => setRefreshedAt(new Date())}
          style={{ marginLeft: 'auto', cursor: 'pointer' }}
        >
          Refresh
        </button>
      </div>

      {loading && <p>Loading…</p>}
      {error && <p style={{ color: '#b91c1c' }}>Error: {error}</p>}

      {stats && (
        <table style={{ borderCollapse: 'collapse', width: '100%', maxWidth: 560 }}>
          <tbody>
            <tr style={{ borderBottom: '1px solid #e5e7eb' }}>
              <td style={{ padding: '12px 8px', fontWeight: 600, width: '50%' }}>
                Queue size (pending)
              </td>
              <td style={{ padding: '12px 8px' }}>
                {stats.queue_size}
                &nbsp;
                <StatusBadge ok={stats.queue_size < 1000} />
              </td>
            </tr>
            <tr style={{ borderBottom: '1px solid #e5e7eb' }}>
              <td style={{ padding: '12px 8px', fontWeight: 600 }}>Last crawl time</td>
              <td style={{ padding: '12px 8px' }}>
                {stats.last_crawl_time
                  ? new Date(stats.last_crawl_time).toLocaleString()
                  : '—'}
                &nbsp;
                <StatusBadge ok={stats.last_crawl_time !== null} />
              </td>
            </tr>
            <tr>
              <td style={{ padding: '12px 8px', fontWeight: 600 }}>
                Sources with failures
              </td>
              <td style={{ padding: '12px 8px' }}>
                {stats.source_fail_count}
                &nbsp;
                <StatusBadge ok={stats.source_fail_count === 0} />
              </td>
            </tr>
          </tbody>
        </table>
      )}
    </div>
  )
}
