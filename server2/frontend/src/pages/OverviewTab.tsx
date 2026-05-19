import { useRef, useState, useEffect, useMemo } from 'react'
import { useVirtualizer } from '@tanstack/react-virtual'
import type { Host, Hit } from '../lib/types'
import { fetchApi } from '../lib/types'

// ── Types ──────────────────────────────────────────────────────────────────

interface Props {
  domain: string
  hosts: Host[]
  hits: Hit[]
  initialHostId?: number | null
  onTriageChange: (id: number, status: string) => void
  onNotesChange: (id: number, notes: string) => void
}

type SsPhase = 'idle' | 'polling' | 'done' | 'failed'

interface SsState {
  phase: SsPhase
  imgPath?: string
  error?: string
}

// ── Screenshot hook ────────────────────────────────────────────────────────

function useScreenshot(domain: string, hostID: string) {
  const [ss, setSs] = useState<SsState>({ phase: 'idle' })
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const tokenRef = useRef<string | null>(null)

  const enc = encodeURIComponent

  function stopPolling() {
    if (pollRef.current) { clearInterval(pollRef.current); pollRef.current = null }
  }

  useEffect(() => {
    stopPolling()
    tokenRef.current = null
    setSs({ phase: 'idle' })

    const img = new Image()
    img.onload = () => setSs({ phase: 'done', imgPath: `/api/${enc(domain)}/host/${hostID}/screenshot` })
    img.src = `/api/${enc(domain)}/host/${hostID}/screenshot?t=${Date.now()}`

    return stopPolling
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [domain, hostID])

  async function take() {
    if (pollRef.current) return
    setSs({ phase: 'polling' })
    try {
      const r = await fetchApi(`/api/${enc(domain)}/host/${hostID}/screenshot`, { method: 'POST' })
      if (!r.ok) { setSs({ phase: 'failed', error: 'Failed to start' }); return }
      const data = await r.json()
      tokenRef.current = data.token

      pollRef.current = setInterval(async () => {
        try {
          const r2 = await fetchApi(
            `/api/${enc(domain)}/host/${hostID}/screenshot/status?token=${tokenRef.current}`
          )
          const d = await r2.json()
          if (d.status === 'done') {
            stopPolling()
            setSs({ phase: 'done', imgPath: d.img_path })
          } else if (d.status === 'failed') {
            stopPolling()
            setSs({ phase: 'failed', error: d.error || 'Screenshot failed' })
          }
        } catch {
          stopPolling()
          setSs({ phase: 'failed', error: 'Network error while polling' })
        }
      }, 1500)
    } catch {
      setSs({ phase: 'failed', error: 'Failed to start screenshot' })
    }
  }

  return { ss, take }
}

// ── Host detail panel ──────────────────────────────────────────────────────

function HostDetail({ domain, host, hits, onTriageChange, onNotesChange }: { domain: string; host: Host; hits: Hit[]; onTriageChange: (id: number, status: string) => void; onNotesChange: (id: number, notes: string) => void }) {
  const [triage, setTriage] = useState(host.triage_status || 'none')
  const [notes, setNotes] = useState(host.notes || '')
  const [notesToast, setNotesToast] = useState<string | null>(null)
  const { ss, take } = useScreenshot(domain, host.host_id)

  const enc = encodeURIComponent

  const scColor: Record<string, string> = {
    s200: 'var(--green)', s201: 'var(--green)', s301: 'var(--orange)',
    s302: 'var(--orange)', s403: 'var(--red)', s400: 'var(--red)',
  }
  const statusColor = scColor[host.sc] || 'var(--text)'

  const sevBadge: Record<string, string> = { high: 'badge-red', medium: 'badge-orange', low: 'badge-yellow' }

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
    const data = await r.json()
    setNotesToast(data.status === 'Note added!' ? 'Saved' : 'Failed')
    setTimeout(() => setNotesToast(null), 2000)
    if (data.status === 'Note added!') onNotesChange(host.id, notes)
  }

  function infoCell(label: string, value: React.ReactNode, mono = false, full = false) {
    return (
      <div className={`ov-info-cell${full ? ' full' : ''}`} key={label}>
        <div className="ov-info-label">{label}</div>
        <div className={`ov-info-value${mono ? ' mono' : ''}`}>
          {value || <span style={{ color: 'var(--text-muted)' }}>—</span>}
        </div>
      </div>
    )
  }

  const origin = (() => { try { return new URL(host.url).origin } catch { return host.url } })()
  const hostHits = hits.filter(h => h.url.startsWith(origin + '/') || h.url === origin)

  return (
    <div className="ov-main-content">
      {/* Header */}
      <div className="ov-header">
        <div className="ov-host-url">
          <a href={host.url} target="_blank" rel="noreferrer">{host.url}</a>
          {host.badges?.map(b => (
            <span key={b} className="badge badge-yellow" style={{ marginLeft: 6 }}>{b}</span>
          ))}
        </div>
        <div className="ov-meta-row">
          <span className="status-val" style={{ color: statusColor, fontSize: 16 }}>{host.status}</span>
          <span style={{ color: 'var(--text-muted)', fontSize: 12, fontFamily: "'Fira Code', monospace" }}>
            {host.server}
          </span>
        </div>
      </div>

      <div className="ov-body">

        {/* Screenshot */}
        <div className="ov-section">
          <div className="ov-section-title action-section">
            Screenshot
            <button
              className="ov-action-btn"
              disabled={ss.phase === 'polling'}
              onClick={take}
            >
              {ss.phase === 'polling' ? 'Running...' : 'Take Screenshot'}
            </button>
          </div>
          <div className="ov-screenshot">
            {ss.phase === 'idle' && (
              <span style={{ color: 'var(--text-muted)', fontSize: 12 }}>No screenshot yet</span>
            )}
            {ss.phase === 'polling' && (
              <div style={{ display: 'flex', alignItems: 'center', gap: 10, color: 'var(--text-muted)', fontSize: 12 }}>
                <div className="spinner" />
                Taking screenshot...
              </div>
            )}
            {ss.phase === 'done' && ss.imgPath && (
              <img src={ss.imgPath} style={{ maxWidth: '100%', maxHeight: 360, display: 'block' }} alt="screenshot" />
            )}
            {ss.phase === 'failed' && (
              <span style={{ color: 'var(--red)', fontSize: 12 }}>{ss.error}</span>
            )}
          </div>
        </div>

        {/* Port Scan */}
        <div className="ov-section">
          <div className="ov-section-title action-section">
            Port Scan
            <button className="ov-action-btn" disabled>Scan Ports</button>
          </div>
          <div style={{ fontSize: 12, color: 'var(--text-muted)', marginTop: 8 }}>Not yet implemented.</div>
        </div>

        {/* Host Info */}
        <div className="ov-section">
          <div className="ov-section-title">Host Info</div>
          <div className="ov-info-grid">
            {infoCell('Status Code',
              <span style={{ color: statusColor, fontWeight: 600, fontFamily: "'Fira Code', monospace" }}>{host.status}</span>
            )}
            {infoCell('Content-Type', host.ctype, true)}
            {infoCell('Title', host.title)}
            {infoCell('Server', host.server, true)}
            {infoCell('IP Address(es)', host.ips?.join(', '), true)}
            {infoCell('CNAME', host.cname?.join(', '), true)}
            {infoCell('Tech Stack',
              host.tech?.length
                ? <>{host.tech.map(t => <span key={t} className="tech-pill">{t}</span>)}</>
                : null,
              false, true
            )}
            {infoCell('Open Ports',
              host.ports?.length
                ? <>{host.ports.map(p => (
                    <span key={p.port} className="tech-pill">
                      <span style={{ color: 'var(--accent)' }}>{p.port}</span>
                      <span style={{ color: 'var(--text-muted)' }}> — {p.service}</span>
                    </span>
                  ))}</>
                : null,
              false, true
            )}
          </div>
        </div>

        {/* Path Hits */}
        <div className="ov-section">
          <div className="ov-section-title">Path Hits ({hostHits.length})</div>
          {hostHits.length === 0 ? (
            <div className="ov-hits-empty">No path hits for this host.</div>
          ) : (
            <table className="ov-hits-table">
              <thead>
                <tr>
                  <th>Path</th><th>Status</th><th>Size</th><th>Severity</th>
                </tr>
              </thead>
              <tbody>
                {hostHits.map((h, i) => (
                  <tr key={i}>
                    <td><a href={h.url} target="_blank" rel="noreferrer">{h.url}</a></td>
                    <td className={h.sc}>{h.status}</td>
                    <td>{h.size}</td>
                    <td><span className={`badge ${sevBadge[h.severity] ?? 'badge-yellow'}`}>{h.severity}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        {/* Triage */}
        <div className="ov-section">
          <div className="ov-section-title">Triage</div>
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

        {/* Notes */}
        <div className="ov-section">
          <div className="ov-section-title">Notes</div>
          <textarea
            className="notes-area"
            placeholder="Notes about this host..."
            value={notes}
            onChange={e => setNotes(e.target.value)}
          />
          <div style={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: 10, marginTop: 6 }}>
            {notesToast && (
              <span style={{ fontSize: 12, color: notesToast === 'Saved' ? 'var(--green)' : 'var(--red)' }}>
                {notesToast}
              </span>
            )}
            <button className="notes-save" onClick={saveNotes}>Save</button>
          </div>
        </div>

      </div>
    </div>
  )
}

// ── Overview tab ───────────────────────────────────────────────────────────

export default function OverviewTab({ domain, hosts, hits, initialHostId, onTriageChange, onNotesChange }: Props) {
  const [activeId, setActiveId] = useState<number | null>(initialHostId ?? null)

  useEffect(() => {
    if (initialHostId != null) setActiveId(initialHostId)
  }, [initialHostId])
  const [filter, setFilter] = useState('')
  const [collapsed, setCollapsed] = useState(false)

  const scrollRef = useRef<HTMLDivElement>(null)

  const filtered = useMemo(
    () => filter ? hosts.filter(h => h.url.toLowerCase().includes(filter.toLowerCase())) : hosts,
    [hosts, filter]
  )

  const activeHost = hosts.find(h => h.id === activeId) ?? null

  const rowVirtualizer = useVirtualizer({
    count: filtered.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => 58,
    overscan: 8,
  })

  const scColor: Record<string, string> = {
    s200: 'var(--green)', s201: 'var(--green)', s301: 'var(--orange)',
    s302: 'var(--orange)', s403: 'var(--red)', s400: 'var(--red)',
  }

  return (
    <div className="overview-layout">
      {/* Sidebar */}
      <div className={`overview-sidebar${collapsed ? ' collapsed' : ''}`}>
        <div className="sidebar-header">
          {!collapsed && (
            <div className="sidebar-search-wrap">
              <input
                type="text"
                placeholder="Filter hosts..."
                value={filter}
                onChange={e => setFilter(e.target.value)}
              />
            </div>
          )}
          <button
            className="sidebar-collapse-btn"
            onClick={() => setCollapsed(c => !c)}
            title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
          >
            {collapsed ? '▶' : '◀'}
          </button>
        </div>

        {!collapsed && (
          <div ref={scrollRef} className="sidebar-list">
            <div style={{ height: rowVirtualizer.getTotalSize(), position: 'relative' }}>
              {rowVirtualizer.getVirtualItems().map(vr => {
                const h = filtered[vr.index]
                return (
                  <div
                    key={h.id}
                    className={`sidebar-item${activeId === h.id ? ' si-active' : ''}`}
                    style={{ position: 'absolute', top: vr.start, width: '100%', height: vr.size }}
                    onClick={() => setActiveId(h.id)}
                  >
                    <div className="sidebar-item-url">{h.url}</div>
                    <div className="sidebar-item-meta">
                      <span
                        className="sidebar-item-status"
                        style={{ color: scColor[h.sc] || 'var(--text-dim)' }}
                      >
                        {h.status}
                      </span>
                      {h.badges?.map(b => (
                        <span key={b} className="badge badge-yellow">{b}</span>
                      ))}
                    </div>
                  </div>
                )
              })}
            </div>
          </div>
        )}
      </div>

      {/* Main content */}
      <div className="overview-main">
        {activeHost ? (
          <HostDetail key={activeHost.id} domain={domain} host={activeHost} hits={hits} onTriageChange={onTriageChange} onNotesChange={onNotesChange} />
        ) : (
          <div className="overview-empty-msg">← Select a host from the sidebar</div>
        )}
      </div>
    </div>
  )
}
