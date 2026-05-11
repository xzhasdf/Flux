import { useEffect, useState, useRef } from 'react'
import './style.css'
import NCMConverter from './pages/NCMConverter'
import VideoDownloader from './pages/VideoDownloader'
import { ClearVideoInfoCache } from '../wailsjs/go/main/App'
import {
  ACCENTS, applyTheme, getAccent,
  loadMode, saveMode, loadAccentId, saveAccentId,
  type Mode,
} from './theme'

const TABS = [
  { id: 'ncm', label: '音乐转码' },
  { id: 'video', label: '视频下载' },
]

export default function App() {
  const [tab, setTab] = useState('ncm')
  const [mode, setMode] = useState<Mode>(loadMode)
  const [accentId, setAccentId] = useState(loadAccentId)
  const [menuOpen, setMenuOpen] = useState(false)
  const [toast, setToast] = useState('')
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    applyTheme(mode, getAccent(accentId))
    saveMode(mode)
    saveAccentId(accentId)
  }, [mode, accentId])

  useEffect(() => {
    function onClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    if (menuOpen) document.addEventListener('mousedown', onClickOutside)
    return () => document.removeEventListener('mousedown', onClickOutside)
  }, [menuOpen])

  useEffect(() => {
    if (!toast) return
    const id = setTimeout(() => setToast(''), 2200)
    return () => clearTimeout(id)
  }, [toast])

  async function clearCache() {
    const n = await ClearVideoInfoCache()
    setToast(n > 0 ? `已清除 ${n} 条缓存` : '缓存已经是空的')
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100vh', background: 'var(--bg)' }}>
      <div style={{
        display: 'flex', alignItems: 'center',
        borderBottom: '1px solid var(--border)',
        background: 'var(--surface)',
        padding: '0 16px',
        flexShrink: 0,
      }}>
        <span style={{ fontWeight: 700, fontSize: 15, color: 'var(--accent)', marginRight: 24, letterSpacing: '0.08em' }}>
          FLUX
        </span>
        {TABS.map(t => (
          <button key={t.id} onClick={() => setTab(t.id)} style={{
            background: 'none', border: 'none', cursor: 'pointer',
            padding: '11px 14px', fontSize: 13,
            color: tab === t.id ? 'var(--text)' : 'var(--muted)',
            borderBottom: tab === t.id ? '2px solid var(--accent)' : '2px solid transparent',
            fontWeight: tab === t.id ? 600 : 400,
            transition: 'color 0.15s',
            marginBottom: -1,
          }}>
            {t.label}
          </button>
        ))}

        {/* Settings menu */}
        <div ref={menuRef} style={{ marginLeft: 'auto', position: 'relative' }}>
          <button
            onClick={() => setMenuOpen(v => !v)}
            title="设置"
            style={{
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              width: 32, height: 32,
              background: menuOpen ? 'var(--surface2)' : 'transparent',
              border: '1px solid var(--border)',
              borderRadius: 6, cursor: 'pointer',
              color: 'var(--muted)',
              transition: 'all 0.15s',
            }}
          >
            <GearIcon />
          </button>

          {menuOpen && (
            <div style={{
              position: 'absolute', top: 'calc(100% + 6px)', right: 0,
              background: 'var(--surface)', border: '1px solid var(--border)',
              borderRadius: 8, padding: 12, zIndex: 100,
              boxShadow: '0 8px 24px rgba(0,0,0,0.3)',
              minWidth: 260,
            }}>
              {/* Theme section */}
              <SectionTitle>外观</SectionTitle>
              <div style={{ display: 'flex', gap: 6, marginBottom: 14 }}>
                {([
                  { id: 'dark' as Mode, label: '深色' },
                  { id: 'light' as Mode, label: '浅色' },
                ]).map(m => (
                  <button
                    key={m.id}
                    onClick={() => setMode(m.id)}
                    style={{
                      flex: 1, padding: '7px 0', fontSize: 12,
                      background: mode === m.id ? 'var(--accent)' : 'var(--surface2)',
                      color: mode === m.id ? '#fff' : 'var(--text)',
                      border: '1px solid', borderColor: mode === m.id ? 'var(--accent)' : 'var(--border)',
                      borderRadius: 6, cursor: 'pointer',
                      fontWeight: mode === m.id ? 600 : 400,
                      transition: 'all 0.15s',
                    }}
                  >
                    {m.label}
                  </button>
                ))}
              </div>

              <SectionTitle>主题色</SectionTitle>
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 6, marginBottom: 14 }}>
                {ACCENTS.map(a => (
                  <button
                    key={a.id}
                    onClick={() => setAccentId(a.id)}
                    title={a.name}
                    style={{
                      display: 'flex', flexDirection: 'column', alignItems: 'center',
                      gap: 4, padding: '8px 4px',
                      background: accentId === a.id ? 'var(--surface2)' : 'transparent',
                      border: `1px solid ${accentId === a.id ? a.color : 'transparent'}`,
                      borderRadius: 6, cursor: 'pointer',
                      transition: 'all 0.15s',
                    }}
                  >
                    <span style={{
                      width: 22, height: 22, borderRadius: '50%',
                      background: a.color,
                      boxShadow: accentId === a.id ? `0 0 0 2px ${a.color}40` : 'none',
                    }} />
                    <span style={{ fontSize: 10.5, color: 'var(--text)' }}>{a.name}</span>
                  </button>
                ))}
              </div>

              <div style={{ height: 1, background: 'var(--border)', margin: '4px 0 12px' }} />

              <SectionTitle>数据</SectionTitle>
              <button
                onClick={() => { clearCache(); setMenuOpen(false) }}
                style={{
                  width: '100%', padding: '8px 12px', fontSize: 12,
                  background: 'var(--surface2)', color: 'var(--text)',
                  border: '1px solid var(--border)', borderRadius: 6,
                  cursor: 'pointer', textAlign: 'left',
                  transition: 'border-color 0.15s',
                }}
              >
                清除视频解析缓存
              </button>
            </div>
          )}
        </div>
      </div>

      <div style={{ flex: 1, overflow: 'hidden', position: 'relative' }}>
        <div style={{ display: tab === 'ncm' ? 'block' : 'none', height: '100%' }}>
          <NCMConverter />
        </div>
        <div style={{ display: tab === 'video' ? 'block' : 'none', height: '100%' }}>
          <VideoDownloader />
        </div>
      </div>

      {toast && (
        <div style={{
          position: 'fixed', bottom: 24, left: '50%', transform: 'translateX(-50%)',
          background: 'var(--surface)', color: 'var(--text)',
          border: '1px solid var(--accent)', borderRadius: 8,
          padding: '8px 16px', fontSize: 13, zIndex: 200,
          boxShadow: '0 4px 16px rgba(0,0,0,0.3)',
        }}>
          {toast}
        </div>
      )}
    </div>
  )
}

function SectionTitle({ children }: { children: React.ReactNode }) {
  return (
    <div style={{
      fontSize: 11, color: 'var(--muted)',
      textTransform: 'uppercase', fontWeight: 600,
      letterSpacing: '0.05em', marginBottom: 8,
    }}>{children}</div>
  )
}

function GearIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="12" cy="12" r="3" />
      <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z" />
    </svg>
  )
}
