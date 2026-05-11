package main

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewApp() *App {
	return &App{}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
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

	results := convertBatch(tasks, req.Format, req.Bitrate, req.Overwrite, workers)

	resp := NCMConvertResponse{Total: len(results), Results: results}
	for _, r := range results {
		if r.Success {
			resp.Success++
		} else {
			resp.Failed++
		}
	}
	return resp
}

// --- Video / M3U8 / Magnet Downloader ---

func (a *App) StartDownload(req DownloadRequest) {
	a.stopCurrent()
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelFunc = cancel

	emit := func(msg string) { runtime.EventsEmit(a.ctx, "download:log", msg) }
	emitProgress := func(p ProgressEvent) { runtime.EventsEmit(a.ctx, "download:progress", p) }
	go func() {
		defer func() { runtime.EventsEmit(a.ctx, "download:done", nil) }()
		downloadGeneral(ctx, req, emit, emitProgress)
	}()
}

func (a *App) StartM3U8(req M3U8Request) {
	a.stopCurrent()
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelFunc = cancel

	emit := func(msg string) { runtime.EventsEmit(a.ctx, "download:log", msg) }
	go func() {
		defer func() { runtime.EventsEmit(a.ctx, "download:done", nil) }()
		downloadM3U8(ctx, req, emit)
	}()
}

func (a *App) StartMagnet(req MagnetRequest) {
	a.stopCurrent()
	ctx, cancel := context.WithCancel(a.ctx)
	a.cancelFunc = cancel

	emit := func(msg string) { runtime.EventsEmit(a.ctx, "download:log", msg) }
	go func() {
		defer func() { runtime.EventsEmit(a.ctx, "download:done", nil) }()
		downloadMagnet(ctx, req, emit)
	}()
}

func (a *App) StopDownload() {
	a.stopCurrent()
}

func (a *App) stopCurrent() {
	if a.cancelFunc != nil {
		a.cancelFunc()
		a.cancelFunc = nil
	}
}
