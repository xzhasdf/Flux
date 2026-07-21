import { useEffect, useState } from 'react'
import { ConvertFolder, OpenDirectoryDialog, OpenInFileManager } from '../../wailsjs/go/main/App'
import { Card, Field, DirInput, SegmentedControl, CheckBox, Btn, Badge, Row, Grid } from '../components/ui'
import { HistoryPanel, type HistoryRecord } from '../components/HistoryPanel'
import { loadSetting, saveSetting } from '../theme'

type Format = 'flac' | 'mp3' | 'wav' | 'm4a' | 'ogg'
type Bitrate = '128k' | '192k' | '256k' | '320k' | 'lossless'

const FORMATS: Format[] = ['flac', 'mp3', 'wav', 'm4a', 'ogg']
const BITRATES: { label: string; value: Bitrate }[] = [
  { label: '128k', value: '128k' },
  { label: '192k', value: '192k' },
  { label: '256k', value: '256k' },
  { label: '320k', value: '320k' },
  { label: '无损', value: 'lossless' },
]
const ALL_EXTS = [
  '.ncm', '.qmc0', '.qmc3', '.qmcflac', '.qmcogg',
  '.bkcmp3', '.bkcflac', '.tkm',
  '.mp3', '.flac', '.wav', '.m4a', '.aac', '.ogg', '.wma',
]

interface ConvertResult {
  src: string
  dst: string
  success: boolean
  message: string
}

export default function NCMConverter() {
  const [inputDir, setInputDir] = useState(() => loadSetting('ncm.inputDir'))
  const [outputDir, setOutputDir] = useState(() => loadSetting('ncm.outputDir'))
  useEffect(() => saveSetting('ncm.inputDir', inputDir), [inputDir])
  useEffect(() => saveSetting('ncm.outputDir', outputDir), [outputDir])
  const [format, setFormat] = useState<Format>('flac')
  const [bitrate, setBitrate] = useState<Bitrate>('lossless')
  const [overwrite, setOverwrite] = useState(false)
  const [workers, setWorkers] = useState(3)
  const [exts, setExts] = useState<string[]>([...ALL_EXTS])
  const [running, setRunning] = useState(false)
  const [results, setResults] = useState<ConvertResult[]>([])
  const [summary, setSummary] = useState<{ total: number; success: number; failed: number } | null>(null)
  const [error, setError] = useState('')

  const losslessFormats = new Set(['flac', 'wav', 'm4a'])
  const canLossless = losslessFormats.has(format)

  function onFormatChange(f: Format) {
    setFormat(f)
    if (!losslessFormats.has(f) && bitrate === 'lossless') {
      setBitrate('320k')
    }
  }

  function toggleExt(ext: string) {
    setExts(prev => prev.includes(ext) ? prev.filter(e => e !== ext) : [...prev, ext])
  }

  async function pickInputDir() {
    const dir = await OpenDirectoryDialog('选择输入目录')
    if (dir) {
      setInputDir(dir)
      if (!outputDir) setOutputDir(dir)
    }
  }

  async function pickOutputDir() {
    const dir = await OpenDirectoryDialog('选择输出目录')
    if (dir) setOutputDir(dir)
  }

  async function runWith(req: any) {
    setError('')
    setRunning(true)
    setResults([])
    setSummary(null)
    try {
      const resp = await ConvertFolder(req)
      if (resp.error) {
        setError(resp.error)
      } else {
        setResults(resp.results || [])
        setSummary({ total: resp.total, success: resp.success, failed: resp.failed })
      }
    } catch (e: any) {
      setError(String(e))
    } finally {
      setRunning(false)
    }
  }

  async function run() {
    if (!inputDir) { setError('请选择输入目录'); return }
    if (exts.length === 0) { setError('请至少选择一个扩展名'); return }
    await runWith({
      inputDir,
      outputDir: outputDir || inputDir,
      format,
      bitrate,
      overwrite,
      workers,
      extensions: exts,
    })
  }

  // 从历史记录重跑：不覆盖模式下已转完的文件会自动跳过，相当于断点续转
  async function resumeRecord(r: HistoryRecord) {
    if (running) return
    try {
      await runWith(JSON.parse(r.payload))
    } catch {
      setError('历史记录已损坏，无法恢复')
    }
  }

  return (
    <div style={{ height: '100%', overflow: 'auto', padding: 20, display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Card>
        <Grid cols={2}>
          <Field label="输入目录">
            <DirInput value={inputDir} placeholder="选择包含音频文件的目录" onPick={pickInputDir} onOpen={() => OpenInFileManager(inputDir)} />
          </Field>
          <Field label="输出目录">
            <DirInput value={outputDir} placeholder="默认与输入目录相同" onPick={pickOutputDir} onOpen={() => OpenInFileManager(outputDir)} />
          </Field>
        </Grid>
      </Card>

      <Card>
        <Grid cols={2}>
          <Field label="目标格式">
            <SegmentedControl
              options={FORMATS.map(f => ({ label: f.toUpperCase(), value: f }))}
              value={format}
              onChange={onFormatChange}
            />
          </Field>
          <Field label="比特率">
            <SegmentedControl
              options={BITRATES.map(b => ({ label: b.label, value: b.value }))}
              value={bitrate}
              onChange={v => canLossless || v !== 'lossless' ? setBitrate(v) : undefined}
            />
          </Field>
        </Grid>

        <div style={{ marginTop: 16 }}>
          <Field label="扫描扩展名">
            <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap', marginTop: 2 }}>
              {ALL_EXTS.map(ext => (
                <label key={ext} style={{
                  display: 'flex', alignItems: 'center', gap: 5, cursor: 'pointer',
                  background: exts.includes(ext) ? 'var(--accent)18' : 'var(--surface2)',
                  border: `1px solid ${exts.includes(ext) ? 'var(--accent)' : 'var(--border)'}`,
                  borderRadius: 5, padding: '4px 10px', fontSize: 12,
                  color: exts.includes(ext) ? 'var(--accent)' : 'var(--muted)',
                  transition: 'all 0.15s',
                }}>
                  <input type="checkbox" checked={exts.includes(ext)} onChange={() => toggleExt(ext)}
                    style={{ display: 'none' }} />
                  {ext}
                </label>
              ))}
            </div>
            <div style={{
              marginTop: 10, padding: '8px 12px',
              background: 'var(--surface2)', border: '1px solid var(--border)',
              borderRadius: 6, fontSize: 11.5, color: 'var(--muted)',
              lineHeight: 1.6,
            }}>
              <div style={{ color: 'var(--text)', fontWeight: 500, marginBottom: 4 }}>支持说明</div>
              <div>
                <span style={{ color: 'var(--success)' }}>✓ 已支持</span>
                ：网易云 <code style={{ color: 'var(--text)' }}>.ncm</code> ／QQ音乐 <code style={{ color: 'var(--text)' }}>.qmc0/.qmc3/.qmcflac/.qmcogg</code>
                ／其他 <code style={{ color: 'var(--text)' }}>.bkcmp3/.bkcflac/.tkm</code>
                ／普通音频 <code style={{ color: 'var(--text)' }}>.mp3/.flac/.wav/.m4a/.aac/.ogg/.wma</code>
              </div>
              <div style={{ marginTop: 3 }}>
                <span style={{ color: 'var(--danger)' }}>✗ 暂不支持</span>
                ：QQ音乐新版加密格式 <code style={{ color: 'var(--text)' }}>.mflac/.mgg/.mflac0/.mgg1</code>
              </div>
            </div>
          </Field>
        </div>

        <Row style={{ marginTop: 16, gap: 20 }}>
          <CheckBox label="覆盖已存在文件" checked={overwrite} onChange={setOverwrite} />
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <span style={{ fontSize: 12, color: 'var(--muted)', textTransform: 'uppercase', fontWeight: 500 }}>并发数</span>
            <input type="range" min={1} max={8} value={workers} onChange={e => setWorkers(+e.target.value)}
              style={{ width: 80, accentColor: 'var(--accent)' }} />
            <span style={{ fontSize: 13, minWidth: 16, textAlign: 'center' }}>{workers}</span>
          </div>
        </Row>
      </Card>

      <Row>
        <Btn onClick={run} style={{ minWidth: 120 }} disabled={running}>
          {running ? '转码中…' : '开始转码'}
        </Btn>
        {error && <span style={{ color: 'var(--danger)', fontSize: 13 }}>{error}</span>}
        {summary && (
          <Row>
            <Badge label={`共 ${summary.total} 个`} color="muted" />
            <Badge label={`成功 ${summary.success}`} color="success" />
            {summary.failed > 0 && <Badge label={`失败 ${summary.failed}`} color="danger" />}
          </Row>
        )}
      </Row>

      <Card>
        <Field label="转码历史">
          <HistoryPanel types={['convert']} onResume={resumeRecord} busy={running} />
        </Field>
      </Card>

      {results.length > 0 && (
        <Card style={{ padding: 0, overflow: 'hidden' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
            <thead>
              <tr style={{ background: 'var(--surface2)', borderBottom: '1px solid var(--border)' }}>
                {['状态', '源文件', '目标文件', '信息'].map(h => (
                  <th key={h} style={{ padding: '8px 12px', textAlign: 'left', color: 'var(--muted)', fontWeight: 500 }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {results.map((r, i) => (
                <tr key={i} style={{ borderBottom: '1px solid var(--border)' }}>
                  <td style={{ padding: '7px 12px' }}>
                    <Badge label={r.success ? 'OK' : 'FAIL'} color={r.success ? 'success' : 'danger'} />
                  </td>
                  <td style={{ padding: '7px 12px', color: 'var(--muted)', maxWidth: 260, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={r.src}>
                    {r.src.split(/[/\\]/).pop()}
                  </td>
                  <td style={{ padding: '7px 12px', color: 'var(--muted)', maxWidth: 260, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={r.dst}>
                    {r.dst.split(/[/\\]/).pop()}
                  </td>
                  <td style={{ padding: '7px 12px', color: r.success ? 'var(--muted)' : 'var(--danger)' }}>{r.message}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Card>
      )}
    </div>
  )
}
