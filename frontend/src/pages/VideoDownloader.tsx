import { useState, useEffect } from 'react'
import { StartDownload, StartM3U8, StartMagnet, StopDownload, OpenDirectoryDialog, OpenFileDialog, OpenInFileManager, ParseVideoInfo } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { Card, Field, DirInput, SegmentedControl, CheckBox, Btn, Row, Textarea, Input, LogPanel, ProgressBar, VideoInfoCard, HelpTooltip, type VideoInfoData } from '../components/ui'
import { HistoryPanel, type HistoryRecord } from '../components/HistoryPanel'
import { loadSetting, saveSetting } from '../theme'

type SubTab = 'general' | 'm3u8' | 'magnet' | 'history'
type Quality = 'best' | '4k' | '2k' | '1080p' | '720p' | '480p' | '360p'
type VideoFormat = 'mp4' | 'mkv' | 'webm' | 'mp3'
type CookieMode = 'none' | 'browser' | 'file'

interface ProgressState {
  jobId: string
  seq: number
  taskIdx: number
  total: number
  url: string
  percent: number
  speed: string
  eta: string
  track: string
  status: 'downloading' | 'done' | 'failed' | 'stopped'
}

// 一次 Start* 调用 = 一个可并行的任务
interface JobState {
  id: string
  seq: number
  title: string
  progress: ProgressState | null
}

export default function VideoDownloader() {
  const [subTab, setSubTab] = useState<SubTab>('general')
  const [saveDir, setSaveDir] = useState(() => loadSetting('video.saveDir'))
  useEffect(() => saveSetting('video.saveDir', saveDir), [saveDir])
  const [jobs, setJobs] = useState<Record<string, JobState>>({})
  const [logs, setLogs] = useState<string[]>([])
  const jobList = Object.values(jobs).sort((a, b) => a.seq - b.seq)
  const running = jobList.length > 0

  // General
  const [urls, setUrls] = useState('')
  const [quality, setQuality] = useState<Quality>('best')
  const [videoFormat, setVideoFormat] = useState<VideoFormat>('mp4')
  const [subtitle, setSubtitle] = useState(false)
  const [embedSub, setEmbedSub] = useState(false)
  const [thumbnail, setThumbnail] = useState(false)
  const [danmaku, setDanmaku] = useState(false)
  const [cookieMode, setCookieMode] = useState<CookieMode>('browser')
  const [cookieFile, setCookieFile] = useState('')
  const [proxy, setProxy] = useState('')
  const [rateLimit, setRateLimit] = useState('')
  const [playlistItems, setPlaylistItems] = useState('')
  const [workers, setWorkers] = useState(2)
  const [fragThreads, setFragThreads] = useState(4)

  // Parse
  const [videoInfo, setVideoInfo] = useState<VideoInfoData | null>(null)
  const [parsing, setParsing] = useState(false)
  const [parseError, setParseError] = useState('')

  // M3U8
  const [m3u8Urls, setM3u8Urls] = useState('')

  // Magnet
  const [magnetLinks, setMagnetLinks] = useState('')
  const [dlLimit, setDlLimit] = useState('')
  const [ulLimit, setUlLimit] = useState('100')
  const [maxConn, setMaxConn] = useState(50)
  const [seedTime, setSeedTime] = useState('0')
  const [extraTracker, setExtraTracker] = useState(true)

  useEffect(() => {
    const offStarted = EventsOn('download:started', (j: { id: string; seq: number; title: string }) => {
      setJobs(prev => ({ ...prev, [j.id]: { id: j.id, seq: j.seq, title: j.title, progress: prev[j.id]?.progress ?? null } }))
    })
    const offLog = EventsOn('download:log', (e: { jobId: string; seq: number; msg: string }) => {
      setLogs(prev => [...prev, `#${e.seq} ${e.msg.replace(/^\n+/, '')}`])
    })
    const offProgress = EventsOn('download:progress', (p: ProgressState) => {
      setJobs(prev => ({
        ...prev,
        [p.jobId]: prev[p.jobId]
          ? { ...prev[p.jobId], progress: p }
          : { id: p.jobId, seq: p.seq, title: p.url, progress: p },
      }))
    })
    const offDone = EventsOn('download:done', (j: { id: string }) => {
      setJobs(prev => {
        const next = { ...prev }
        delete next[j.id]
        return next
      })
    })
    return () => { offStarted(); offLog(); offProgress(); offDone() }
  }, [])

  async function pickSaveDir() {
    const dir = await OpenDirectoryDialog('选择保存目录')
    if (dir) setSaveDir(dir)
  }

  async function pickCookieFile() {
    const file = await OpenFileDialog('选择 Cookie 文件', 'Cookie 文件', '*.txt')
    if (file) setCookieFile(file)
  }

  function appendLog(msg: string) { setLogs(prev => [...prev, msg]) }

  async function startGeneral() {
    const urlList = urls.split('\n').map(u => u.trim()).filter(Boolean)
    if (!urlList.length) { appendLog('❌ 请输入视频 URL'); return }
    if (!saveDir) { appendLog('❌ 请选择保存目录'); return }

    if (!running) setLogs([])
    setVideoInfo(null)
    setParseError('')

    // 并行：解析首条视频信息（失败不影响下载）
    setParsing(true)
    ParseVideoInfo({ url: urlList[0], cookieMode, cookieFile, proxy })
      .then(info => setVideoInfo(info as unknown as VideoInfoData))
      .catch(e => setParseError(String(e?.message || e)))
      .finally(() => setParsing(false))

    await StartDownload({
      urls: urlList, saveDir, quality, format: videoFormat,
      subtitle, embedSub, thumbnail, danmaku,
      cookieMode, cookieFile, proxy, rateLimit, playlistItems,
      workers, fragThreads,
    })
  }

  async function startM3U8() {
    const urlList = m3u8Urls.split('\n').map(u => u.trim()).filter(Boolean)
    if (!urlList.length) { appendLog('❌ 请输入 M3U8 地址'); return }
    if (!saveDir) { appendLog('❌ 请选择保存目录'); return }
    if (!running) setLogs([])
    await StartM3U8({ urls: urlList, saveDir })
  }

  async function startMagnet() {
    const linkList = magnetLinks.split('\n').map(l => l.trim()).filter(Boolean)
    if (!linkList.length) { appendLog('❌ 请输入磁力链接'); return }
    if (!saveDir) { appendLog('❌ 请选择保存目录'); return }
    if (!running) setLogs([])
    await StartMagnet({ links: linkList, saveDir, dlLimit, ulLimit, maxConn, seedTime, extraTracker })
  }

  async function stopJob(id: string) {
    await StopDownload(id)
  }

  async function stopAll() {
    await StopDownload('')
  }

  // 从历史记录继续/重试：用原参数重新发起，可与进行中的任务并行。
  // yt-dlp 靠 .part 文件、aria2c 靠 .aria2 控制文件断点续传；M3U8（ffmpeg）会重新下载。
  async function resumeRecord(r: HistoryRecord) {
    let payload: any
    try { payload = JSON.parse(r.payload) } catch { appendLog('❌ 历史记录已损坏，无法恢复'); return }

    if (!running) setLogs([])
    appendLog(`▶ 从历史记录继续：${r.title}`)

    if (r.type === 'general') await StartDownload(payload)
    else if (r.type === 'm3u8') await StartM3U8(payload)
    else if (r.type === 'magnet') await StartMagnet(payload)
  }

  const SUB_TABS: { id: SubTab; label: string }[] = [
    { id: 'general', label: '通用下载' },
    { id: 'm3u8', label: 'M3U8' },
    { id: 'magnet', label: '磁力/种子' },
    { id: 'history', label: '历史记录' },
  ]

  function jobLabel(j: JobState) {
    const p = j.progress
    return `#${j.seq} `
      + (p && p.total > 1 ? `[${p.taskIdx + 1}/${p.total}] ` : '')
      + (p?.track ? `${p.track} · ` : '')
      + (p?.url || j.title)
  }

  function jobSublabel(j: JobState) {
    const p = j.progress
    if (!p) return '准备中…'
    if (p.status === 'downloading') return `速度: ${p.speed || '—'}    剩余: ${p.eta || '—'}`
    if (p.status === 'done') return '✅ 已完成'
    if (p.status === 'failed') return '❌ 失败'
    if (p.status === 'stopped') return '⏹ 已停止'
    return ''
  }

  return (
    <div style={{ display: 'flex', flexDirection: 'column', height: '100%', padding: 20, gap: 12 }}>
      <Card style={{ padding: '12px 16px' }}>
        <Field label="保存目录">
          <DirInput
            value={saveDir}
            placeholder="选择文件保存位置"
            onPick={pickSaveDir}
            onOpen={() => OpenInFileManager(saveDir)}
          />
        </Field>
      </Card>

      <div style={{ display: 'flex', flex: 1, gap: 12, overflow: 'hidden' }}>
        {/* Left: controls */}
        <div style={{ width: 560, flexShrink: 0, display: 'flex', flexDirection: 'column', gap: 12, overflow: 'auto' }}>
          <div style={{ display: 'flex', gap: 0, borderBottom: '1px solid var(--border)' }}>
            {SUB_TABS.map(t => (
              <button key={t.id} onClick={() => setSubTab(t.id)} style={{
                background: 'none', border: 'none', cursor: 'pointer',
                padding: '8px 14px', fontSize: 13,
                color: subTab === t.id ? 'var(--text)' : 'var(--muted)',
                borderBottom: subTab === t.id ? '2px solid var(--accent)' : '2px solid transparent',
                fontWeight: subTab === t.id ? 600 : 400,
                marginBottom: -1,
              }}>{t.label}</button>
            ))}
          </div>

          {subTab === 'general' && (
            <Card style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              <Field label="视频 URL（每行一个）">
                <Textarea value={urls} onChange={e => setUrls(e.target.value)} placeholder="https://..." rows={3} style={{ userSelect: 'text' }} />
              </Field>
              <div style={{ fontSize: 11, color: 'var(--muted)' }}>支持：B站 / YouTube / Twitter(X) / TikTok / 抖音 / Instagram / AcFun 等</div>

              <Field label="画质">
                <SegmentedControl
                  options={(['best','4k','2k','1080p','720p','480p','360p'] as Quality[]).map(q => ({
                    label: q === 'best' ? 'Best' : q.toUpperCase(),
                    value: q,
                  }))}
                  value={quality} onChange={setQuality}
                />
              </Field>
              <Field label="输出格式">
                <SegmentedControl
                  options={(['mp4','mkv','webm','mp3'] as VideoFormat[]).map(f => ({ label: f.toUpperCase(), value: f }))}
                  value={videoFormat} onChange={setVideoFormat}
                />
              </Field>

              <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', alignItems: 'center' }}>
                <CheckBox label="下载字幕" checked={subtitle} onChange={setSubtitle} />
                <CheckBox label="下载弹幕" checked={danmaku} onChange={setDanmaku} />
                <CheckBox label="内嵌字幕" checked={embedSub} onChange={setEmbedSub} />
                <CheckBox label="嵌入封面" checked={thumbnail} onChange={setThumbnail} />
                <HelpTooltip
                  rows={[
                    { label: '下载字幕', desc: '拉视频自带的字幕和自动字幕（中/英/日/韩），生成同名 .vtt 或 .srt 外挂文件' },
                    { label: '下载弹幕', desc: '仅 B 站 / A 站 有效。下载弹幕，自动转成 .ass 外挂字幕（PotPlayer / IINA / mpv 自动加载）' },
                    { label: '内嵌字幕', desc: '把上面两项下到的字幕"焊死"在视频文件里。MP4 容器仅支持简单字幕，建议配合 MKV 输出' },
                    { label: '嵌入封面', desc: '把视频的缩略图作为封面图嵌入视频文件，部分播放器和资源管理器会显示' },
                  ]}
                />
              </div>

              <Field label="Cookie 来源">
                <SegmentedControl
                  options={[{ label: '不使用', value: 'none' }, { label: 'Chrome', value: 'browser' }, { label: '文件', value: 'file' }] as { label: string; value: CookieMode }[]}
                  value={cookieMode} onChange={setCookieMode}
                />
                {cookieMode === 'file' && (
                  <DirInput value={cookieFile} placeholder="选择 Cookie .txt 文件" onPick={pickCookieFile} />
                )}
              </Field>

              <Grid2>
                <Field label="代理">
                  <Input value={proxy} onChange={e => setProxy(e.target.value)} placeholder="socks5://127.0.0.1:1080" style={{ userSelect: 'text' }} />
                </Field>
                <Field label="限速">
                  <Input value={rateLimit} onChange={e => setRateLimit(e.target.value)} placeholder="如 5M 或 500K" style={{ userSelect: 'text' }} />
                </Field>
                <Field label="播放列表选集">
                  <Input value={playlistItems} onChange={e => setPlaylistItems(e.target.value)} placeholder="如 1-5 或 3,5,7" style={{ userSelect: 'text' }} />
                </Field>
                <Field label={`并发任务 ${workers} | 分片 ${fragThreads}`}>
                  <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
                    <input type="range" min={1} max={8} value={workers} onChange={e => setWorkers(+e.target.value)} style={{ flex: 1, accentColor: 'var(--accent)' }} />
                    <input type="range" min={1} max={16} value={fragThreads} onChange={e => setFragThreads(+e.target.value)} style={{ flex: 1, accentColor: 'var(--accent)' }} />
                  </div>
                </Field>
              </Grid2>
            </Card>
          )}

          {subTab === 'm3u8' && (
            <Card style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              <Field label="M3U8 地址（每行一个）">
                <Textarea value={m3u8Urls} onChange={e => setM3u8Urls(e.target.value)} placeholder="https://.../index.m3u8" rows={5} style={{ userSelect: 'text' }} />
              </Field>
            </Card>
          )}

          {subTab === 'history' && (
            <HistoryPanel
              types={['general', 'm3u8', 'magnet']}
              onResume={resumeRecord}
              busy={false}
            />
          )}

          {subTab === 'magnet' && (
            <Card style={{ display: 'flex', flexDirection: 'column', gap: 14 }}>
              <Field label="磁力链接 / 种子路径（每行一个）">
                <Textarea value={magnetLinks} onChange={e => setMagnetLinks(e.target.value)} placeholder="magnet:?xt=..." rows={4} style={{ userSelect: 'text' }} />
              </Field>
              <Grid2>
                <Field label="下载限速 (KB/s, 0不限)">
                  <Input value={dlLimit} onChange={e => setDlLimit(e.target.value)} placeholder="0" style={{ userSelect: 'text' }} />
                </Field>
                <Field label="上传限速 (KB/s)">
                  <Input value={ulLimit} onChange={e => setUlLimit(e.target.value)} placeholder="100" style={{ userSelect: 'text' }} />
                </Field>
                <Field label="做种时间 (分钟)">
                  <Input value={seedTime} onChange={e => setSeedTime(e.target.value)} placeholder="0" style={{ userSelect: 'text' }} />
                </Field>
                <Field label={`最大连接数 ${maxConn}`}>
                  <input type="range" min={10} max={200} value={maxConn} onChange={e => setMaxConn(+e.target.value)} style={{ width: '100%', accentColor: 'var(--accent)' }} />
                </Field>
              </Grid2>
              <CheckBox label="自动添加公共 Tracker（提升连接速度）" checked={extraTracker} onChange={setExtraTracker} />
            </Card>
          )}

          <Row>
            {subTab !== 'history' && (
              <Btn onClick={subTab === 'general' ? startGeneral : subTab === 'm3u8' ? startM3U8 : startMagnet} style={{ minWidth: 120 }}>
                开始下载
              </Btn>
            )}
            {running && (
              <Btn variant="danger" onClick={stopAll} style={{ minWidth: 100 }}>
                全部停止 ({jobList.length})
              </Btn>
            )}
          </Row>
        </div>

        {/* Right: video info + progress + log */}
        <div style={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column', gap: 10 }}>
          {parsing && (
            <div style={{
              background: 'var(--surface)', border: '1px solid var(--border)',
              borderRadius: 8, padding: 12, fontSize: 12, color: 'var(--muted)',
            }}>
              🔍 正在解析视频信息…
            </div>
          )}
          {videoInfo && <VideoInfoCard info={videoInfo} />}
          {parseError && !parsing && (
            <div style={{
              background: 'var(--surface)', border: '1px solid var(--border)',
              borderRadius: 8, padding: '10px 12px', fontSize: 12, color: 'var(--danger)',
            }}>
              ❌ 解析失败：{parseError}
            </div>
          )}
          {jobList.map(j => (
            <div key={j.id} style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
              <div style={{ flex: 1, minWidth: 0 }}>
                <ProgressBar
                  percent={j.progress?.percent ?? 0}
                  label={jobLabel(j)}
                  sublabel={jobSublabel(j)}
                  status={j.progress?.status || 'downloading'}
                />
              </div>
              <Btn variant="danger" onClick={() => stopJob(j.id)}
                style={{ padding: '6px 12px', fontSize: 12, flexShrink: 0 }}>
                停止
              </Btn>
            </div>
          ))}
          <div style={{ flex: 1, overflow: 'hidden' }}>
            <LogPanel lines={logs} />
          </div>
        </div>
      </div>
    </div>
  )
}

function Grid2({ children }: { children: React.ReactNode }) {
  return <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 12 }}>{children}</div>
}
