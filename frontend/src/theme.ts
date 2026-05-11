export type Mode = 'light' | 'dark'

export interface Accent {
  id: string
  name: string
  color: string
  hover: string
}

export interface ModePalette {
  bg: string
  surface: string
  surface2: string
  border: string
  text: string
  muted: string
  logBg: string
  logText: string
}

export const ACCENTS: Accent[] = [
  { id: 'indigo',  name: '靛蓝',  color: '#6366f1', hover: '#818cf8' },
  { id: 'cyan',    name: '青绿',  color: '#06b6d4', hover: '#22d3ee' },
  { id: 'amber',   name: '琥珀',  color: '#f59e0b', hover: '#fbbf24' },
  { id: 'pink',    name: '玫红',  color: '#ec4899', hover: '#f472b6' },
  { id: 'emerald', name: '翡翠',  color: '#10b981', hover: '#34d399' },
  { id: 'crimson', name: '朱红',  color: '#dc2626', hover: '#ef4444' },
  { id: 'violet',  name: '紫罗兰', color: '#8b5cf6', hover: '#a78bfa' },
  { id: 'rose',    name: '蔷薇',  color: '#f43f5e', hover: '#fb7185' },
]

// 模式只控制中性色，不带 accent 倾向
export const PALETTES: Record<Mode, ModePalette> = {
  dark: {
    bg:       '#0e1014',
    surface:  '#181a20',
    surface2: '#22252c',
    border:   '#2d3038',
    text:     '#e5e7eb',
    muted:    '#7c8593',
    logBg:    '#0a0c12',
    logText:  '#a0aec0',
  },
  light: {
    bg:       '#f8fafc',
    surface:  '#ffffff',
    surface2: '#f1f5f9',
    border:   '#e2e8f0',
    text:     '#0f172a',
    muted:    '#64748b',
    logBg:    '#f8fafc',
    logText:  '#475569',
  },
}

const KEY_MODE = 'flux:theme:mode'
const KEY_ACCENT = 'flux:theme:accent'

export function applyTheme(mode: Mode, accent: Accent) {
  const p = PALETTES[mode]
  const root = document.documentElement
  root.style.setProperty('--bg', p.bg)
  root.style.setProperty('--surface', p.surface)
  root.style.setProperty('--surface2', p.surface2)
  root.style.setProperty('--border', p.border)
  root.style.setProperty('--text', p.text)
  root.style.setProperty('--muted', p.muted)
  root.style.setProperty('--log-bg', p.logBg)
  root.style.setProperty('--log-text', p.logText)
  root.style.setProperty('--accent', accent.color)
  root.style.setProperty('--accent-hover', accent.hover)
}

export function loadMode(): Mode {
  const v = localStorage.getItem(KEY_MODE)
  return v === 'light' ? 'light' : 'dark'
}
export function saveMode(m: Mode) { localStorage.setItem(KEY_MODE, m) }

export function loadAccentId(): string {
  return localStorage.getItem(KEY_ACCENT) || 'indigo'
}
export function saveAccentId(id: string) { localStorage.setItem(KEY_ACCENT, id) }

export function getAccent(id: string): Accent {
  return ACCENTS.find(a => a.id === id) || ACCENTS[0]
}

// 通用设置持久化
export function loadSetting(key: string, defaultValue = ''): string {
  return localStorage.getItem(`flux:${key}`) || defaultValue
}

export function saveSetting(key: string, value: string) {
  if (value) localStorage.setItem(`flux:${key}`, value)
  else localStorage.removeItem(`flux:${key}`)
}
