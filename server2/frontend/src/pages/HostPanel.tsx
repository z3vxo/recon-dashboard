import { useState } from 'react'
import type { Host } from '../lib/types'
import { fetchApi } from '../lib/types'

interface Props {
  host: Host
  domain: string
  onClose: () => void
  onOpenInOverview: (id: number) => void
  onTriageChange: (id: number, status: string) => void
  onNotesChange: (id: number, notes: string) => void
}

export default function HostPanel({ host, domain, onClose, onOpenInOverview, onTriageChange, onNotesChange }: Props) {
  const [triage, setTriage] = useState(host.triage_status || 'none')
  const [notes, setNotes]   = useState(host.notes || '')
  const [saved, setSaved]   = useState<'ok' | 'err' | null>(null)

  const enc = encodeURIComponent

  const scColor: Record<string, string> = {
    s200: 'var(--green)', s201: 'var(--green)',
    s301: 'var(--orange)', s302: 'var(--orange)',
    s403: 'var(--red)', s400: 'var(--red)',
  }
  const statusColor = scColor[host.sc] || 'var(--text)'

  async function saveTriage(status: string) {
    setTriage(status)
    await fetchApi(`/api/${enc(domain)}/host/${host.host_id}/triage`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ domain, status }),
    })
    onTriageChange(host.id, status)
  }

  async function saveNotes() {
    const r = await fetchApi(`/api/${enc(domain)}/host/${host.host_id}/notes`, {
      method: 'PATCH',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ domain, notes }),
    })
    const d = await r.json()
    setSaved(d.status === 'Note added!' ? 'ok' : 'err')
    setTimeout(() => setSaved(null), 2000)
    if (d.status === 'Note added!') onNotesChange(host.id, notes)
  }

  const val = (v: string | string[] | undefined) => {
    if (!v || (Array.isArray(v) && !v.length)) return <span style={{ color: 'var(--text-muted)' }}>—</span>
    return Array.isArray(v) ? v.join(', ') : v
  }

  return (
    <>
      {/* Overlay */}
      <div className="panel-overlay" onClick={onClose} />

      {/* Panel */}
      <div className="detail-panel">
        <div className="panel-header">
          <span className="panel-title">Host Details</span>
          <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
            <button className="panel-ov-btn" onClick={() => { onClose(); onOpenInOverview(host.id) }}>
              Overview ↗
            </button>
            <button className="panel-close" onClick={onClose}>
              <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2">
                <line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>
        </div>

        <div className="panel-url">
          <a href={host.url} target="_blank" rel="noreferrer">{host.url}</a>
        </div>

        <div className="panel-body">
          <Section label="Status">
            <span className="status-val" style={{ color: statusColor }}>{host.status}</span>
          </Section>
          <div className="panel-divider" />
          <Section label="Title">{val(host.title)}</Section>
          <div className="panel-divider" />
          <Section label="Server" mono>{val(host.server)}</Section>
          <div className="panel-divider" />
          <Section label="Tech Stack">
            {host.tech?.length
              ? host.tech.map(t => <span key={t} className="tech-pill">{t}</span>)
              : <span style={{ color: 'var(--text-muted)' }}>—</span>}
          </Section>
          <div className="panel-divider" />
          <Section label="Open Ports">
            {host.ports?.length
              ? host.ports.map(p => (
                  <span key={p.port} className="tech-pill">
                    <span style={{ color: 'var(--accent)', fontWeight: 600 }}>{p.port}</span>
                    <span style={{ color: 'var(--text-muted)' }}> — {p.service}</span>
                  </span>
                ))
              : <span style={{ color: 'var(--text-muted)' }}>—</span>}
          </Section>
          <div className="panel-divider" />
          <Section label="IP Address(es)" mono>{val(host.ips)}</Section>
          <div className="panel-divider" />
          <Section label="CNAME" mono>{val(host.cname)}</Section>
          <div className="panel-divider" />
          <Section label="Content-Type" mono>{val(host.ctype)}</Section>
          <div className="panel-divider" />

          {/* Triage */}
          <div className="panel-section">
            <div className="panel-section-label">Triage</div>
            <div className="triage-btns">
              {(['none', 'to-test', 'dead-end', 'tested'] as const).map(s => (
                <button
                  key={s}
                  className={`triage-btn${triage === s ? ' active' : ''}`}
                  data-status={s}
                  onClick={() => saveTriage(s)}
                >
                  {s}
                </button>
              ))}
            </div>
          </div>
          <div className="panel-divider" />

          {/* Notes */}
          <div className="panel-section">
            <div className="panel-section-label">Notes</div>
            <textarea
              className="notes-area"
              placeholder="e.g. takes URL param at /search, worth testing..."
              value={notes}
              onChange={e => setNotes(e.target.value)}
            />
            <div style={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: 10, marginTop: 6 }}>
              {saved && (
                <span style={{ fontSize: 12, color: saved === 'ok' ? 'var(--green)' : 'var(--red)' }}>
                  {saved === 'ok' ? 'Saved' : 'Failed'}
                </span>
              )}
              <button className="notes-save" onClick={saveNotes}>Save</button>
            </div>
          </div>
        </div>
      </div>
    </>
  )
}

function Section({ label, children, mono }: { label: string; children: React.ReactNode; mono?: boolean }) {
  return (
    <div className="panel-section">
      <div className="panel-section-label">{label}</div>
      <div className={`panel-section-value${mono ? ' mono' : ''}`}>{children}</div>
    </div>
  )
}
