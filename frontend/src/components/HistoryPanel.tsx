import { useCallback, useEffect, useState } from 'react'
import { GetHistory, DeleteHistory, ClearHistory, OpenInFileManager } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { main } from '../../wailsjs/go/models'
import { Badge, Btn } from './ui'

export type HistoryRecord = main.HistoryRecord

const STATUS_META: Record<string, { label: string; color: 'success' | 'danger' | 'muted' }> = {
  running:     { label: '进行中',   color: 'muted' },
  done:        { label: '已完成',   color: 'success' },
  partial:     { label: '部分完成', color: 'danger' },
  failed:      { label: '失败',     color: 'danger' },
  stopped:     { label: '已停止',   color: 'muted' },
  interrupted: { label: '已中断',   color: 'danger' },
}

const TYPE_LABEL: Record<string, string> = {
  general: '视频', m3u8: 'M3U8', magnet: '磁力', convert: '转码',
}

// 这些状态的任务可以「继续」（yt-dlp .part / aria2 .aria2 控制文件断点续传）
const RESUMABLE = new Set(['interrupted', 'stopped', 'partial', 'failed'])

function formatTime(unixSec: number) {
  const d = new Date(unixSec * 1000)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getMonth() + 1}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function HistoryPanel({ types, onResume, busy }: {
  types: string[]
  onResume: (r: HistoryRecord) => void
  busy: boolean
}) {
  const [records, setRecords] = useState<HistoryRecord[]>([])

  const refresh = useCallback(async () => {
    const all = await GetHistory()
    setRecords((all || []).filter(r => types.includes(r.type)))
  }, [types.join(',')])

  useEffect(() => {
    refresh()
    const off = EventsOn('history:changed', refresh)
    return off
  }, [refresh])

  async function remove(id: string) {
    await DeleteHistory(id)
    refresh()
  }

  async function clearAll() {
    await ClearHistory()
    refresh()
  }

  if (records.length === 0) {
    return (
      <div style={{ padding: '24px 12px', textAlign: 'center', fontSize: 12.5, color: 'var(--muted)' }}>
        暂无历史记录。开始一个任务后会自动记录，中断的任务可以在这里继续。
      </div>
    )
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
        <button onClick={clearAll} style={{
          background: 'none', border: 'none', cursor: 'pointer',
          fontSize: 11.5, color: 'var(--muted)', padding: '2px 4px',
        }}>清空历史</button>
      </div>
      {records.map(r => {
        const meta = STATUS_META[r.status] || { label: r.status, color: 'muted' as const }
        const resumable = RESUMABLE.has(r.status)
        return (
          <div key={r.id} style={{
            display: 'flex', alignItems: 'center', gap: 10,
            background: 'var(--surface)', border: '1px solid var(--border)',
            borderRadius: 8, padding: '10px 12px',
          }}>
            <div style={{ flex: 1, minWidth: 0 }}>
              <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                <Badge label={TYPE_LABEL[r.type] || r.type} color="muted" />
                <Badge label={meta.label} color={meta.color} />
                <span style={{ fontSize: 11, color: 'var(--muted)' }}>{formatTime(r.createdAt)}</span>
                {r.count > 1 && <span style={{ fontSize: 11, color: 'var(--muted)' }}>共 {r.count} 条</span>}
              </div>
              <div title={r.title} style={{
                fontSize: 12.5, color: 'var(--text)',
                overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
              }}>{r.title}</div>
              {r.detail && (
                <div style={{ fontSize: 11, color: 'var(--muted)', marginTop: 2 }}>{r.detail}</div>
              )}
            </div>
            <div style={{ display: 'flex', gap: 6, flexShrink: 0 }}>
              {r.status !== 'running' && (
                <Btn variant={resumable ? 'primary' : 'outline'} disabled={busy}
                  onClick={() => onResume(r)}
                  style={{ padding: '5px 12px', fontSize: 12 }}>
                  {resumable ? '继续' : '重新执行'}
                </Btn>
              )}
              {r.saveDir && (
                <Btn variant="outline" onClick={() => OpenInFileManager(r.saveDir)}
                  style={{ padding: '5px 12px', fontSize: 12 }}>
                  打开目录
                </Btn>
              )}
              <Btn variant="outline" onClick={() => remove(r.id)}
                style={{ padding: '5px 10px', fontSize: 12, color: 'var(--muted)' }}>
                ✕
              </Btn>
            </div>
          </div>
        )
      })}
    </div>
  )
}
