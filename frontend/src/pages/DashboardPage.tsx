import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { DashboardOverview, NearDuplicateGroup } from '../api/types'
import styles from './Page.module.css'

export default function DashboardPage() {
  const [overview, setOverview] = useState<DashboardOverview | null>(null)
  const [nearDupGroups, setNearDupGroups] = useState<NearDuplicateGroup[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [refreshKey, setRefreshKey] = useState(0)

  useEffect(() => {
    setLoading(true)
    setError(null)

    Promise.all([api.getDashboard(), api.listNearDuplicates(200, 5)])
      .then(([dashboard, duplicates]) => {
        setOverview(dashboard)
        setNearDupGroups(duplicates)
      })
      .catch((err) => {
        setError(err instanceof Error ? err.message : 'Failed to load dashboard')
      })
      .finally(() => setLoading(false))
  }, [refreshKey])

  const refresh = () => setRefreshKey((key) => key + 1)

  return (
    <div>
      <div className={styles.pageHeader}>
        <h2 className={styles.pageTitle}>Dashboard</h2>
        <button className="btnSecondary" type="button" onClick={refresh}>
          Refresh
        </button>
      </div>

      {error && <p className={styles.error}>{error}</p>}
      {loading && <p className={styles.muted}>Loading dashboard…</p>}

      {overview && (
        <>
          <div style={{ display: 'grid', gap: 16, marginBottom: 24, gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))' }}>
            <div style={{ padding: 18, background: '#fff', borderRadius: 12, border: '1px solid #e2e8f0' }}>
              <div style={{ color: '#64748b', fontSize: 12, textTransform: 'uppercase', letterSpacing: 0.04 }}>
                Top sources
              </div>
              <div style={{ marginTop: 12, fontSize: 28, fontWeight: 700 }}>
                {(overview.top_sources ?? []).length}
              </div>
              <div style={{ marginTop: 8, color: '#475569' }}>
                Source article counts and top performers.
              </div>
            </div>
            <div style={{ padding: 18, background: '#fff', borderRadius: 12, border: '1px solid #e2e8f0' }}>
              <div style={{ color: '#64748b', fontSize: 12, textTransform: 'uppercase', letterSpacing: 0.04 }}>
                Fail rate
              </div>
              <div style={{ marginTop: 12, fontSize: 28, fontWeight: 700 }}>
                {(overview.fail_rate * 100).toFixed(1)}%
              </div>
              <div style={{ marginTop: 8, color: '#475569' }}>
                Percent of processed crawl tasks that failed.
              </div>
            </div>
            <div style={{ padding: 18, background: '#fff', borderRadius: 12, border: '1px solid #e2e8f0' }}>
              <div style={{ color: '#64748b', fontSize: 12, textTransform: 'uppercase', letterSpacing: 0.04 }}>
                Stale sources
              </div>
              <div style={{ marginTop: 12, fontSize: 28, fontWeight: 700 }}>
                {(overview.sla_sources ?? []).filter((item) => item.stale).length}
              </div>
              <div style={{ marginTop: 8, color: '#475569' }}>
                Sources with no new article in the last 5 days.
              </div>
            </div>
          </div>

          <section style={{ marginBottom: 32 }}>
            <h3>Top sources</h3>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Source</th>
                  <th>Articles</th>
                </tr>
              </thead>
              <tbody>
                {(overview.top_sources ?? []).map((source) => (
                  <tr key={source.source_id}>
                    <td>{source.name}</td>
                    <td>{source.count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>

          <section style={{ marginBottom: 32 }}>
            <h3>Documents by topic</h3>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Topic</th>
                  <th>Count</th>
                </tr>
              </thead>
              <tbody>
                {(overview.counts_by_topic ?? []).map((item) => (
                  <tr key={item.topic_id}>
                    <td>{item.topic_name || item.topic_id}</td>
                    <td>{item.count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>

          <section style={{ marginBottom: 32 }}>
            <h3>Daily document volume</h3>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Topic</th>
                  <th>Count</th>
                </tr>
              </thead>
              <tbody>
                {(overview.counts_by_day ?? []).map((item) => (
                  <tr key={`${item.date}-${item.topic_id}`}>
                    <td>{item.date}</td>
                    <td>{item.topic_name || item.topic_id}</td>
                    <td>{item.count}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>

          <section style={{ marginBottom: 32 }}>
            <h3>Source SLA monitor</h3>
            <table className={styles.table}>
              <thead>
                <tr>
                  <th>Source</th>
                  <th>Last post</th>
                  <th>Fail count</th>
                  <th>Stale</th>
                </tr>
              </thead>
              <tbody>
                {(overview.sla_sources ?? []).map((item) => (
                  <tr key={item.source_id} style={{ background: item.stale ? '#fef2f2' : undefined }}>
                    <td>{item.name}</td>
                    <td>{item.last_post_at ? new Date(item.last_post_at).toLocaleDateString() : 'No posts'}</td>
                    <td>{item.fail_count}</td>
                    <td>{item.stale ? 'Yes' : 'No'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </section>

          <section>
            <h3>Near-duplicate candidates</h3>
            {(nearDupGroups ?? []).length === 0 ? (
              <p className={styles.muted}>No near-duplicate groups detected.</p>
            ) : (
              <div style={{ display: 'grid', gap: 16 }}>
                {(nearDupGroups ?? []).map((group, index) => (
                  <div key={index} style={{ padding: 18, background: '#fff', borderRadius: 12, border: '1px solid #e2e8f0' }}>
                    <div style={{ marginBottom: 10, fontWeight: 600 }}>
                      Group {index + 1} — {group.docs.length} similar articles (distance ≤ {group.max_distance})
                    </div>
                    <ul style={{ margin: 0, paddingLeft: 20 }}>
                      {group.docs.map((doc) => (
                        <li key={doc.ID} style={{ marginBottom: 6 }}>
                          <a href={doc.URL} target="_blank" rel="noreferrer" style={{ color: '#1e40af' }}>
                            {doc.Title || doc.URL}
                          </a>{' '}
                          <span style={{ color: '#64748b' }}>[{doc.SourceID}]</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                ))}
              </div>
            )}
          </section>
        </>
      )}
    </div>
  )
}
