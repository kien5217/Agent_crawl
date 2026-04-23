import { useEffect, useState } from 'react'
import { api } from '../api/client'
import type { Topic } from '../api/types'
import styles from './Page.module.css'

export default function TopicsPage() {
  const [topics, setTopics] = useState<Topic[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api.listTopics().then(setTopics).finally(() => setLoading(false))
  }, [])

  return (
    <div>
      <h2 className={styles.pageTitle}>Topics</h2>
      {loading && <p className={styles.muted}>Loading…</p>}
      {!loading && (
        <table className={styles.table}>
          <thead>
            <tr>
              <th>ID</th>
              <th>Name</th>
            </tr>
          </thead>
          <tbody>
            {topics.map((t) => (
              <tr key={t.id}>
                <td><code>{t.id}</code></td>
                <td>{t.name}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}
