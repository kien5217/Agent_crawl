import { JSX, useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { api } from '../api/client'
import type { Document } from '../api/types'
import styles from './Page.module.css'

function highlightText(text: string, keywords: string[]): JSX.Element {
  if (!keywords.length) return <>{text}</>

  // Create a regex that matches any of the keywords (case-insensitive)
  const regex = new RegExp(`(${keywords.map(kw => kw.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')).join('|')})`, 'gi')
  const parts = text.split(regex)

  return (
    <>
      {parts.map((part, i) =>
        keywords.some(kw => kw.toLowerCase() === part.toLowerCase()) ? (
          <mark key={i} className={styles.highlight}>{part}</mark>
        ) : (
          part
        )
      )}
    </>
  )
}

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
        {doc.Keywords.length > 0 && (
          <>
            <dt>Keywords</dt>
            <dd>{doc.Keywords.join(', ')}</dd>
          </>
        )}
      </dl>
      <section>
        <h3>Content</h3>
        <div className={styles.content}>
          {highlightText(doc.ContentText, doc.Keywords)}
        </div>
      </section>
    </div>
  )
}
