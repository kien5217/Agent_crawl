import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Document } from '../api/types'
import styles from './Page.module.css'

export default function DocumentDetailPage() {
  const { id } = useParams<{ id: string }>()
  const [doc, setDoc] = useState<Document | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (!id) return
    api
      .getDocument(Number(id))
      .then(setDoc)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false))
  }, [id])

  if (loading) return <p className={styles.muted}>Loading…</p>
  if (error) return <p className={styles.error}>{error}</p>
  if (!doc) return null

  return (
    <div className={styles.detail}>
      <Link to="/documents" className={styles.back}>← Back</Link>
      <h2>{doc.Title}</h2>
      <dl className={styles.meta}>
        <dt>URL</dt>
        <dd><a href={doc.URL} target="_blank" rel="noreferrer">{doc.URL}</a></dd>
        <dt>Topic</dt>
        <dd>{doc.TopicID}</dd>
        <dt>Source</dt>
        <dd>{doc.SourceID}</dd>
        <dt>Published</dt>
        <dd>{doc.PublishedAt ? new Date(doc.PublishedAt).toLocaleString() : '—'}</dd>
      </dl>
      <section>
        <h3>Content</h3>
        <pre className={styles.content}>{doc.ContentText}</pre>
      </section>
    </div>
  )
}
