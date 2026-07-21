package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx         context.Context
	history     *HistoryStore
	activeTasks atomic.Int32
	jobSeq      atomic.Int32
	jobsMu      sync.Mutex
	jobs        map[string]context.CancelFunc // jobID(=历史记录 ID) → 取消函数
}

func NewApp() *App {
	return &App{
		history: NewHistoryStore(),
		jobs:    make(map[string]context.CancelFunc),
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.history.Load()
}

// --- Task history ---

// beginTask 记录任务开始并计入活跃任务数（关窗二次确认依据）
func (a *App) beginTask(rec HistoryRecord) string {
	a.activeTasks.Add(1)
	rec.Status = "running"
	id := a.history.Add(rec)
	runtime.EventsEmit(a.ctx, "history:changed")
	return id
}

func (a *App) endTask(id string, ctx context.Context, success, fail int) {
	status, detail := taskOutcome(ctx, success, fail)
	a.history.Update(id, status, detail)
	a.activeTasks.Add(-1)
	runtime.EventsEmit(a.ctx, "history:changed")
}

func taskOutcome(ctx context.Context, success, fail int) (string, string) {
	if ctx != nil && ctx.Err() != nil {
		return "stopped", fmt.Sprintf("已手动停止（%d 个已完成），可继续", success)
	}
	switch {
	case fail == 0:
		return "done", fmt.Sprintf("%d 个全部完成", success)
	case success > 0:
		return "partial", fmt.Sprintf("%d 成功 · %d 失败", success, fail)
	default:
		return "failed", fmt.Sprintf("%d 个全部失败", fail)
	}
}

func (a *App) HasActiveTasks() bool {
	return a.activeTasks.Load() > 0
}

func (a *App) GetHistory() []HistoryRecord {
	return a.history.List()
}

func (a *App) DeleteHistory(id string) {
	a.history.Delete(id)
}

func (a *App) ClearHistory() int {
	return a.history.Clear()
}

// --- Dialogs ---

func (a *App) OpenDirectoryDialog(title string) string {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
	if err != nil {
		return ""
	}
	return dir
}

func (a *App) OpenFilesDialog(title string, filterName string, pattern string) []string {
	files, err := runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: filterName, Pattern: pattern},
		},
	})
	if err != nil {
		return nil
	}
	return files
}

func (a *App) OpenFileDialog(title string, filterName string, pattern string) string {
	file, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: title,
		Filters: []runtime.FileFilter{
			{DisplayName: filterName, Pattern: pattern},
		},
	})
	if err != nil {
		return ""
	}
	return file
}

// --- System check ---

func (a *App) CheckFFmpeg() bool {
	_, err := getFFmpegBin()
	return err == nil
}

// --- File system ---

func (a *App) OpenInFileManager(path string) error {
	if path == "" {
		return fmt.Errorf("路径不能为空")
	}
	var cmd *exec.Cmd
	switch goruntime.GOOS {
	case "darwin":
		cmd = exec.Command("open", path)
	case "windows":
		cmd = exec.Command("explorer", path)
	default:
		cmd = exec.Command("xdg-open", path)
	}
	return cmd.Start()
}

// --- NCM Converter ---

type NCMConvertRequest struct {
	InputDir   string   `json:"inputDir"`
	OutputDir  string   `json:"outputDir"`
	Format     string   `json:"format"`
	Bitrate    string   `json:"bitrate"`
	Overwrite  bool     `json:"overwrite"`
	Workers    int      `json:"workers"`
	Extensions []string `json:"extensions"`
}

type NCMConvertResponse struct {
	Total   int             `json:"total"`
	Success int             `json:"success"`
	Failed  int             `json:"failed"`
	Results []ConvertResult `json:"results"`
	Error   string          `json:"error,omitempty"`
}

func (a *App) ConvertFolder(req NCMConvertRequest) NCMConvertResponse {
	if req.InputDir == "" {
		return NCMConvertResponse{Error: "请选择输入目录"}
	}
	if !supportedFormats[req.Format] {
		return NCMConvertResponse{Error: fmt.Sprintf("不支持的格式: %s", req.Format)}
	}

	extMap := make(map[string]bool)
	for _, e := range req.Extensions {
		ext := strings.ToLower(e)
		if !strings.HasPrefix(ext, ".") {
			ext = "." + ext
		}
		extMap[ext] = true
	}

	outputDir := req.OutputDir
	if outputDir == "" {
		outputDir = req.InputDir
	}

	files, err := findAudioFiles(req.InputDir, extMap, false)
	if err != nil {
		return NCMConvertResponse{Error: fmt.Sprintf("扫描目录失败: %v", err)}
	}
	if len(files) == 0 {
		return NCMConvertResponse{Error: "未找到符合扩展名的文件"}
	}

	var tasks []ConvertTask
	for _, src := range files {
		rel, err := filepath.Rel(req.InputDir, src)
		if err != nil {
			rel = filepath.Base(src)
		}
		dst := filepath.Join(outputDir, rel)
		dst = strings.TrimSuffix(dst, filepath.Ext(dst)) + "." + req.Format
		tasks = append(tasks, ConvertTask{Src: src, Dst: dst})
	}

	workers := req.Workers
	if workers < 1 {
		workers = 3
	}

	recID := a.beginTask(HistoryRecord{
		Type: "convert", Title: req.InputDir, Count: len(tasks),
		SaveDir: outputDir, Payload: marshalPayload(req),
	})

	results := convertBatch(tasks, req.Format, req.Bitrate, req.Overwrite, workers)

	resp := NCMConvertResponse{Total: len(results), Results: results}
	for _, r := range results {
		if r.Success {
			resp.Success++
		} else {
			resp.Failed++
		}
	}
	a.endTask(recID, nil, resp.Success, resp.Failed)
	return resp
}

// --- Video / M3U8 / Magnet Downloader ---

func marshalPayload(req interface{}) string {
	data, err := json.Marshal(req)
	if err != nil {
		return ""
	}
	return string(data)
}

func firstOf(items []string) string {
	if len(items) == 0 {
		return ""
	}
	return items[0]
}

// JobInfo Start* 的返回值，前端立即拿到任务标识
type JobInfo struct {
	ID    string `json:"id"`
	Seq   int    `json:"seq"`
	Title string `json:"title"`
}

// startJob 注册一个可并行的下载任务：独立 ctx/cancel + 历史记录 + 带任务标识的事件
func (a *App) startJob(recType, title string, count int, saveDir, payload string,
	run func(ctx context.Context, emit func(string), emitProgress func(ProgressEvent)) (int, int)) JobInfo {

	recID := a.beginTask(HistoryRecord{
		Type: recType, Title: title, Count: count,
		SaveDir: saveDir, Payload: payload,
	})
	seq := int(a.jobSeq.Add(1))

	ctx, cancel := context.WithCancel(a.ctx)
	a.jobsMu.Lock()
	a.jobs[recID] = cancel
	a.jobsMu.Unlock()

	info := JobInfo{ID: recID, Seq: seq, Title: title}
	runtime.EventsEmit(a.ctx, "download:started", info)

	emit := func(msg string) {
		runtime.EventsEmit(a.ctx, "download:log", LogEvent{JobID: recID, Seq: seq, Msg: msg})
	}
	emitProgress := func(p ProgressEvent) {
		p.JobID = recID
		p.Seq = seq
		runtime.EventsEmit(a.ctx, "download:progress", p)
	}

	go func() {
		defer func() {
			a.jobsMu.Lock()
			delete(a.jobs, recID)
			a.jobsMu.Unlock()
			cancel()
			runtime.EventsEmit(a.ctx, "download:done", info)
		}()
		success, fail := run(ctx, emit, emitProgress)
		a.endTask(recID, ctx, success, fail)
	}()
	return info
}

func (a *App) StartDownload(req DownloadRequest) JobInfo {
	return a.startJob("general", firstOf(req.URLs), len(req.URLs), req.SaveDir, marshalPayload(req),
		func(ctx context.Context, emit func(string), emitProgress func(ProgressEvent)) (int, int) {
			return downloadGeneral(ctx, req, emit, emitProgress)
		})
}

func (a *App) StartM3U8(req M3U8Request) JobInfo {
	return a.startJob("m3u8", firstOf(req.URLs), len(req.URLs), req.SaveDir, marshalPayload(req),
		func(ctx context.Context, emit func(string), _ func(ProgressEvent)) (int, int) {
			return downloadM3U8(ctx, req, emit)
		})
}

func (a *App) StartMagnet(req MagnetRequest) JobInfo {
	return a.startJob("magnet", firstOf(req.Links), len(req.Links), req.SaveDir, marshalPayload(req),
		func(ctx context.Context, emit func(string), emitProgress func(ProgressEvent)) (int, int) {
			return downloadMagnet(ctx, req, emit, emitProgress)
		})
}

// StopDownload 停止指定任务；jobID 为空时停止全部（兼容旧行为）
func (a *App) StopDownload(jobID string) {
	a.jobsMu.Lock()
	defer a.jobsMu.Unlock()
	if jobID == "" {
		for _, cancel := range a.jobs {
			cancel()
		}
		return
	}
	if cancel, ok := a.jobs[jobID]; ok {
		cancel()
	}
}
