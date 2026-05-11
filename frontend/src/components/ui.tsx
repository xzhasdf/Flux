import { CSSProperties, ReactNode, forwardRef, useEffect, useRef, useState } from 'react'

const s = {
  card: {
    background: 'var(--surface)',
    border: '1px solid var(--border)',
    borderRadius: 8,
    padding: '16px 20px',
  } as CSSProperties,

  label: {
    display: 'block',
    fontSize: 12,
    color: 'var(--muted)',
    marginBottom: 6,
    fontWeight: 500,
    textTransform: 'uppercase' as const,
    letterSpacing: '0.05em',
  } as CSSProperties,

  input: {
    width: '100%',
    background: 'var(--surface2)',
    border: '1px solid var(--border)',
    borderRadius: 6,
    padding: '7px 10px',
    color: 'var(--text)',
    fontSize: 13,
    outline: 'none',
  } as CSSProperties,

  btn: {
    background: 'var(--accent)',
    color: '#fff',
    border: 'none',
    borderRadius: 6,
    padding: '8px 18px',
    fontSize: 13,
    fontWeight: 600,
    cursor: 'pointer',
    transition: 'background 0.15s',
    whiteSpace: 'nowrap' as const,
  } as CSSProperties,

  btnOutline: {
    background: 'transparent',
    color: 'var(--text)',
    border: '1px solid var(--border)',
    borderRadius: 6,
    padding: '7px 14px',
    fontSize: 13,
    cursor: 'pointer',
    transition: 'border-color 0.15s',
    whiteSpace: 'nowrap' as const,
  } as CSSProperties,

  btnDanger: {
    background: 'var(--danger)',
    color: '#fff',
    border: 'none',
    borderRadius: 6,
    padding: '8px 18px',
    fontSize: 13,
    fontWeight: 600,
    cursor: 'pointer',
    whiteSpace: 'nowrap' as const,
  } as CSSProperties,
}

export function Card({ children, style }: { children: ReactNode; style?: CSSProperties }) {
  return <div style={{ ...s.card, ...style }}>{children}</div>
}

export function Label({ children }: { children: ReactNode }) {
  return <label style={s.label}>{children}</label>
}

export function Input({ style, ...props }: React.InputHTMLAttributes<HTMLInputElement>) {
  return <input style={{ ...s.input, ...style }} {...props} />
}

export function Textarea({ style, ...props }: React.TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      style={{ ...s.input, resize: 'none', minHeight: 72, ...style }}
      {...props}
    />
  )
}

export function Btn({ style, variant = 'primary', ...props }: React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: 'primary' | 'outline' | 'danger' }) {
  const base = variant === 'outline' ? s.btnOutline : variant === 'danger' ? s.btnDanger : s.btn
  return <button style={{ ...base, ...style }} {...props} />
}

export function SegmentedControl<T extends string>({
  options, value, onChange,
}: {
  options: { label: string; value: T }[]
  value: T
  onChange: (v: T) => void
}) {
  return (
    <div style={{ display: 'flex', gap: 0, background: 'var(--surface2)', borderRadius: 6, padding: 3, border: '1px solid var(--border)' }}>
      {options.map(o => (
        <button
          key={o.value}
          onClick={() => onChange(o.value)}
          style={{
            flex: 1, background: value === o.value ? 'var(--accent)' : 'transparent',
            color: value === o.value ? '#fff' : 'var(--muted)',
            border: 'none', borderRadius: 4, padding: '5px 12px',
            fontSize: 12, fontWeight: 500, cursor: 'pointer',
            transition: 'all 0.15s',
          }}
        >
          {o.label}
        </button>
      ))}
    </div>
  )
}

export function CheckBox({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
  return (
    <label style={{ display: 'flex', alignItems: 'center', gap: 7, cursor: 'pointer', fontSize: 13, color: 'var(--text)' }}>
      <input
        type="checkbox"
        checked={checked}
        onChange={e => onChange(e.target.checked)}
        style={{ width: 14, height: 14, accentColor: 'var(--accent)', cursor: 'pointer' }}
      />
      {label}
    </label>
  )
}

export function Badge({ label, color }: { label: string; color: 'success' | 'danger' | 'muted' }) {
  const colors = { success: 'var(--success)', danger: 'var(--danger)', muted: 'var(--muted)' }
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: 4,
      fontSize: 11, fontWeight: 600, padding: '2px 8px',
      borderRadius: 999, border: `1px solid ${colors[color]}`,
      color: colors[color], background: `${colors[color]}18`,
    }}>{label}</span>
  )
}

export function DirInput({ value, placeholder, onPick, onOpen }: {
  value: string; placeholder?: string; onPick: () => void; onOpen?: () => void
}) {
  return (
    <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
      <div style={{
        flex: 1, background: 'var(--surface2)', border: '1px solid var(--border)',
        borderRadius: 6, padding: '7px 10px', fontSize: 13,
        color: value ? 'var(--text)' : 'var(--muted)',
        overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
        cursor: 'default',
      }} title={value}>
        {value || placeholder || '未选择'}
      </div>
      <Btn variant="outline" onClick={onPick} style={{ padding: '7px 14px' }}>选择</Btn>
      {value && onOpen && (
        <Btn variant="outline" onClick={onOpen} style={{ padding: '7px 14px' }}>打开</Btn>
      )}
    </div>
  )
}

export const LogPanel = forwardRef<HTMLDivElement, { lines: string[] }>(({ lines }, externalRef) => {
  const innerRef = useRef<HTMLDivElement | null>(null)
  // Auto-scroll to bottom whenever lines change
  useEffect(() => {
    const el = innerRef.current
    if (el) el.scrollTop = el.scrollHeight
  }, [lines])

  return (
    <div
      ref={(node) => {
        innerRef.current = node
        if (typeof externalRef === 'function') externalRef(node)
        else if (externalRef) (externalRef as React.MutableRefObject<HTMLDivElement | null>).current = node
      }}
      style={{
        background: 'var(--log-bg)', border: '1px solid var(--border)', borderRadius: 8,
        padding: '10px 12px', fontFamily: 'monospace', fontSize: 12,
        color: 'var(--log-text)', height: '100%', overflow: 'auto',
        display: 'flex', flexDirection: 'column', gap: 1,
      }}>
      {lines.length === 0
        ? <span style={{ color: 'var(--muted)' }}>等待操作…</span>
        : lines.map((l, i) => (
          <span key={i} style={{
            color: l.includes('✅') ? 'var(--success)'
              : l.includes('❌') ? 'var(--danger)'
              : l.includes('⏹') ? 'var(--muted)'
              : 'var(--log-text)',
          }}>{l}</span>
        ))
      }
    </div>
  )
})

export function Row({ children, style }: { children: ReactNode; style?: CSSProperties }) {
  return <div style={{ display: 'flex', gap: 12, alignItems: 'center', ...style }}>{children}</div>
}

export function Grid({ children, cols = 2, style }: { children: ReactNode; cols?: number; style?: CSSProperties }) {
  return (
    <div style={{ display: 'grid', gridTemplateColumns: `repeat(${cols}, 1fr)`, gap: 16, ...style }}>
      {children}
    </div>
  )
}

export function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div>
      <Label>{label}</Label>
      {children}
    </div>
  )
}

export function ProgressBar({ percent, label, status, sublabel }: {
  percent: number
  label?: string
  sublabel?: string
  status?: 'idle' | 'downloading' | 'done' | 'failed' | 'stopped'
}) {
  const pct = Math.max(0, Math.min(100, percent))
  const barColor =
    status === 'failed' ? 'var(--danger)' :
    status === 'done' ? 'var(--success)' :
    status === 'stopped' ? 'var(--muted)' :
    'var(--accent)'
  return (
    <div style={{
      background: 'var(--surface)', border: '1px solid var(--border)',
      borderRadius: 8, padding: '10px 14px',
    }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 6 }}>
        <span style={{ fontSize: 12, color: 'var(--text)', fontWeight: 500, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {label || '等待中…'}
        </span>
        <span style={{ fontSize: 12, color: barColor, fontWeight: 600, fontFamily: 'monospace' }}>
          {pct.toFixed(1)}%
        </span>
      </div>
      <div style={{
        height: 6, background: 'var(--surface2)', borderRadius: 3, overflow: 'hidden',
      }}>
        <div style={{
          width: `${pct}%`, height: '100%', background: barColor,
          transition: 'width 0.2s ease, background 0.2s',
        }} />
      </div>
      {sublabel && (
        <div style={{ marginTop: 6, fontSize: 11, color: 'var(--muted)', fontFamily: 'monospace' }}>
          {sublabel}
        </div>
      )}
    </div>
  )
}

export interface VideoInfoData {
  title: string
  uploader: string
  duration: number
  thumbnail: string
  description: string
  uploadDate: string
  viewCount: number
  resolutions: number[]
  filesize: number
  isPlaylist: boolean
  playlistTitle?: string
  episodes?: { title: string; duration: number }[]
}

function fmtDuration(sec: number): string {
  if (!sec) return '未知'
  const h = Math.floor(sec / 3600)
  const m = Math.floor((sec % 3600) / 60)
  const s = sec % 60
  if (h > 0) return `${h}:${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
  return `${m}:${String(s).padStart(2, '0')}`
}

function fmtSize(bytes: number): string {
  if (!bytes) return '未知'
  const units = ['B', 'KB', 'MB', 'GB']
  let n = bytes
  for (const u of units) {
    if (n < 1024) return `${n.toFixed(1)} ${u}`
    n /= 1024
  }
  return `${n.toFixed(1)} TB`
}

function fmtDate(d: string): string {
  if (!d || d.length !== 8) return d || ''
  return `${d.slice(0, 4)}-${d.slice(4, 6)}-${d.slice(6, 8)}`
}

export function HelpTooltip({ rows }: { rows: { label: string; desc: string }[] }) {
  const [open, setOpen] = useState(false)
  return (
    <span
      onMouseEnter={() => setOpen(true)}
      onMouseLeave={() => setOpen(false)}
      style={{
        position: 'relative',
        display: 'inline-flex', alignItems: 'center', justifyContent: 'center',
        width: 16, height: 16, borderRadius: '50%',
        border: '1px solid var(--border)',
        color: 'var(--muted)', fontSize: 11, fontWeight: 600,
        cursor: 'help', userSelect: 'none',
      }}
      title=""
    >
      ?
      {open && (
        <div style={{
          position: 'absolute', top: 'calc(100% + 8px)', left: '50%',
          transform: 'translateX(-50%)',
          background: 'var(--surface)', border: '1px solid var(--border)',
          borderRadius: 8, padding: '10px 12px',
          boxShadow: '0 8px 24px rgba(0,0,0,0.35)',
          zIndex: 50, minWidth: 320, maxWidth: 380,
          fontSize: 11.5, color: 'var(--text)',
          fontWeight: 400, lineHeight: 1.55, textAlign: 'left',
          cursor: 'default', whiteSpace: 'normal',
        }}>
          {rows.map((r, i) => (
            <div key={r.label} style={{ marginTop: i === 0 ? 0 : 8 }}>
              <span style={{ color: 'var(--accent)', fontWeight: 600 }}>{r.label}</span>
              <span style={{ color: 'var(--muted)', marginLeft: 8 }}>{r.desc}</span>
            </div>
          ))}
        </div>
      )}
    </span>
  )
}

function Meta({ label, value }: { label: string; value: string }) {
  return (
    <span style={{ display: 'inline-flex', alignItems: 'center', gap: 4 }}>
      <span style={{ color: 'var(--muted)', opacity: 0.7 }}>{label}</span>
      <span style={{ color: 'var(--text)' }}>{value}</span>
    </span>
  )
}

export function VideoInfoCard({ info }: { info: VideoInfoData }) {
  return (
    <div style={{
      background: 'var(--surface)', border: '1px solid var(--border)',
      borderRadius: 8, padding: 12, display: 'flex', gap: 12,
    }}>
      <div style={{
        flexShrink: 0, width: 140, height: 80,
        background: 'var(--surface2)', borderRadius: 6,
        overflow: 'hidden', display: 'flex', alignItems: 'center', justifyContent: 'center',
      }}>
        {info.thumbnail
          ? <img src={info.thumbnail} alt="" style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
          : <span style={{ fontSize: 11, color: 'var(--muted)' }}>暂无封面</span>
        }
      </div>
      <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 4 }}>
        <div style={{
          fontSize: 13, color: 'var(--text)', fontWeight: 600,
          overflow: 'hidden', textOverflow: 'ellipsis',
          display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical' as const,
        }}>
          {info.isPlaylist && <span style={{ color: 'var(--accent)', marginRight: 4 }}>[合集]</span>}
          {info.title || '未知'}
        </div>
        <div style={{ fontSize: 11.5, color: 'var(--muted)', display: 'flex', gap: 12, flexWrap: 'wrap' }}>
          <Meta label="作者" value={info.uploader || '未知'} />
          <Meta label="时长" value={fmtDuration(info.duration)} />
          {info.uploadDate && <Meta label="日期" value={fmtDate(info.uploadDate)} />}
          {info.viewCount > 0 && <Meta label="播放" value={info.viewCount.toLocaleString()} />}
        </div>
        {info.resolutions && info.resolutions.length > 0 && (
          <div style={{ fontSize: 11.5, color: 'var(--muted)', display: 'flex', gap: 12, flexWrap: 'wrap' }}>
            <Meta label="画质" value={info.resolutions.slice(0, 6).map(r => `${r}P`).join(' / ')} />
            {info.filesize > 0 && <Meta label="大小" value={`≈ ${fmtSize(info.filesize)}`} />}
          </div>
        )}
        {info.isPlaylist && info.episodes && info.episodes.length > 1 && (
          <div style={{ fontSize: 11, color: 'var(--muted)' }}>
            合集共 {info.episodes.length} 集
          </div>
        )}
      </div>
    </div>
  )
}
