import { useEffect, useState } from "react";
import { api } from "../api/client";
import { Source,Topic } from "../api/types";
import styles from "./Page.module.css";

type SourceForm ={
    id: string 
    name: string
    url: string
    enabled: boolean
    scheduleFreq: string
    topicIDs: string[]
}

const initialForm: SourceForm = {
    id: '',
    name: '',
    url: '',
    enabled: true,
    scheduleFreq: '',
    topicIDs: []
}

export default function SourcesPage() {
    const [sources, setSources] = useState<Source[]>([])
    const [topics, setTopics] = useState<Topic[]>([])
    const [loading, setLoading] = useState(true)
    const [error, setError] = useState<string | null>(null)
    const [form, setForm] = useState<SourceForm>(initialForm)
    const [busy, setBusy] = useState(false)
    const [editing, setEditing] = useState(false)
    const [message, setMessage] = useState<string | null>(null)

    useEffect(() => {
        Promise.all([api.listSources(), api.listTopics()])
            .then(([sources, topics]) => {
                setSources(sources)
                setTopics(topics)
                setLoading(false)
            })
            .catch((err) => {
                setError(err.message)
                setLoading(false)
            })
    }, [])

    const refresh = async () => {
        setBusy(true)
        setError(null)
        try {
            const [sources, topics] = await Promise.all([api.listSources(), api.listTopics()])
            setSources(sources)
            setTopics(topics)
            setMessage('updated successfully')
        } catch (err: unknown) {
            setError((err as Error).message)
        } finally {
            setBusy(false)
        }
    }

    const setField = (field: keyof SourceForm, value: string | boolean | string[]) => {
        setForm((f) => ({ ...f, [field]: value }))
    }

    const handleEdit = (source: Source) => {
        setForm({
            id: source.id,
            name: source.name,
            url: source.url,
            enabled: source.enabled,
            scheduleFreq: source.schedule_freq,
            topicIDs: source.topic_ids || []
        })
        setEditing(true)
        setMessage(null)
        setError(null)
    }

    const handleReset = () => {
        setEditing(false)
        setForm(initialForm)
        setError(null)
        setMessage(null)
    }

    const handleSubmit = async (e: React.FormEvent) => {
        e.preventDefault()
        setBusy(true)
        setError(null)
        setMessage(null)
        try {
            const payload = {
                id: form.id,
                name: form.name,
                url: form.url,
                enabled: form.enabled,
                schedule_freq: form.scheduleFreq,
                topic_ids: form.topicIDs
            }

            if (editing) {
                await api.updateSource(payload)
                setMessage('Source updated successfully')
            } else {
                await api.createSource(payload)
                setMessage('Source created successfully')
            }
            await refresh()
            handleReset()
        } catch (err: unknown) {
            setError((err as Error).message)
        } finally {
            setBusy(false)
        }

    }

    const handleTopicSelection = (event: React.ChangeEvent<HTMLSelectElement>) => {
        const selected = Array.from(event.target.selectedOptions, (option) => option.value);
        setField('topicIDs', selected)
    };

    return (
        <div>
            <div className={styles.pageHeader}>
                <h2 className={styles.pageTitle}>Sources</h2>
                <div>
                    <button className="btn" type="button" onClick={handleReset} >
                        New Source
                    </button>
                    <button className="btnSecondary" type="button" onClick={refresh} disabled={busy}>
                        Refresh
                    </button>
                </div>
            </div>

            {message && <p className={styles.info}>{message}</p>}
            {error && <p className={styles.error}>{error}</p>}
            {loading && <p className={styles.muted}>Loading sources…</p>}

            <form onSubmit={handleSubmit} className={styles.filters}>
                <div className={styles.filterField}>
                    <span>Source ID</span>
                    <input
                        value={form.id}
                        onChange={(e) => setField('id', e.target.value)}
                        required
                        disabled={editing}
                    />
                </div>
                <div className={styles.filterField}>
                    <span>Name</span>
                    <input
                        type="text"
                        value={form.name}
                        onChange={(e) => setField('name', e.target.value)}
                        required
                    />
                </div>
                <div className={styles.filterField}>
                    <span>URL</span>
                    <input
                        value={form.url}
                        onChange={(e) => setField('url', e.target.value)}
                        required
                    />
                </div>
                <div className={styles.filterField}>
                    <span>Schedule Frequency (cron syntax)</span>
                    <input
                       
                        value={form.scheduleFreq}
                        onChange={(e) => setField('scheduleFreq', e.target.value)}
                        placeholder = "daily, hourly, weekly"
                        
                    />
                </div>
                <div className={styles.filterField}>
                  <span> Topics </span>
                  <select multiple value={form.topicIDs} onChange={handleTopicSelection} size ={Math.min(5, topics.length)}>
                    {topics.map((topic) => (
                      <option key={topic.id} value={topic.id}> 
                    {topic.name} </option> ))}
                  </select>
                </div>
                <div className={styles.filterActions}>
                    <span> Enabled</span>
                    <label>
                        <input
                            type="checkbox"
                            checked={form.enabled}
                            onChange={(e) => setField('enabled', e.target.checked)}
                        /> {' '} Active
                    </label>
                </div>

                <div className={styles.filterActions}>
                    <button className="btnPrimary" type="submit" disabled={busy}>
                        {editing ? 'Update Source' : 'Create Source'}
                    </button>
                    {editing && (
                        <button className="btnSecondary" type="button" onClick={handleReset} disabled={busy}>
                            Cancel
                        </button>
                    )}
                </div>
            </form>

            <table className={styles.table}>
        <thead>
          <tr>
            <th>ID</th>
            <th>URL</th>
            <th>Enabled</th>
            <th>Schedule</th>
            <th>Topics</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {sources.map((source) => (
            <tr key={source.id}>
              <td><code>{source.id}</code></td>
              <td>{source.url}</td>
              <td>{source.enabled ? 'Yes' : 'No'}</td>
              <td>{source.schedule_freq || '—'}</td>
              <td>{(source.topic_ids || []).join(', ')}</td>
              <td>
                <button
                  className="btnSecondary"
                  type="button"
                  onClick={() => handleEdit(source)}
                >
                  Edit
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
        </div>
    )
}