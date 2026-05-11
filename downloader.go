package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DownloadRequest struct {
	URLs          []string `json:"urls"`
	SaveDir       string   `json:"saveDir"`
	Quality       string   `json:"quality"`
	Format        string   `json:"format"`
	Subtitle      bool     `json:"subtitle"`
	EmbedSub      bool     `json:"embedSub"`
	Thumbnail     bool     `json:"thumbnail"`
	Danmaku       bool     `json:"danmaku"` // B站/A站弹幕（XML→ASS 自动转换）
	CookieMode    string   `json:"cookieMode"`
	CookieFile    string   `json:"cookieFile"`
	Proxy         string   `json:"proxy"`
	RateLimit     string   `json:"rateLimit"`
	PlaylistItems string   `json:"playlistItems"`
	Workers       int      `json:"workers"`
	FragThreads   int      `json:"fragThreads"`
}

type M3U8Request struct {
	URLs    []string `json:"urls"`
	SaveDir string   `json:"saveDir"`
}

type MagnetRequest struct {
	Links       []string `json:"links"`
	SaveDir     string   `json:"saveDir"`
	DLLimit     string   `json:"dlLimit"`
	ULLimit     string   `json:"ulLimit"`
	MaxConn     int      `json:"maxConn"`
	SeedTime    string   `json:"seedTime"`
	ExtraTracker bool    `json:"extraTracker"`
}

type ProgressEvent struct {
	TaskIdx int     `json:"taskIdx"`
	Total   int     `json:"total"`
	URL     string  `json:"url"`
	Percent float64 `json:"percent"`
	Speed   string  `json:"speed"`
	ETA     string  `json:"eta"`
	Track   string  `json:"track"`
	Status  string  `json:"status"` // downloading | done | failed | stopped
}

var (
	progressRe       = regexp.MustCompile(`\[download\]\s+(\d+(?:\.\d+)?)%\s+of`)
	speedRe          = regexp.MustCompile(`at\s+(\S+)`)
	etaRe            = regexp.MustCompile(`ETA\s+(\S+)`)
	ffmpegDurationRe = regexp.MustCompile(`Duration:\s+(\d+):(\d+):([\d.]+)`)
	ffmpegBitrateRe  = regexp.MustCompile(`bitrate:\s*(\d+(?:\.\d+)?)\s*kb/s`)
	ffmpegTimeRe     = regexp.MustCompile(`time=(\d+):(\d+):([\d.]+)`)
	ffmpegSpeedRe    = regexp.MustCompile(`speed=\s*([\d.]+)x`)
)

// 把 ffmpeg 的 speed 倍率 + 源码率换算成可读速率
func formatFFmpegSpeed(mult, bitrateKbps float64) string {
	if mult <= 0 {
		return ""
	}
	if bitrateKbps <= 0 {
		return fmt.Sprintf("%.2fx", mult)
	}
	// kbps × 1000 / 8 = bytes/s；再 / 1024 / 1024 = MB/s
	mbps := mult * bitrateKbps * 1000 / 8 / 1024 / 1024
	if mbps >= 1 {
		return fmt.Sprintf("%.2f MB/s", mbps)
	}
	return fmt.Sprintf("%.0f KB/s", mbps*1024)
}

func parseYtdlpProgress(line string) (percent float64, speed, eta string, ok bool) {
	m := progressRe.FindStringSubmatch(line)
	if len(m) < 2 {
		return
	}
	if v, err := strconv.ParseFloat(m[1], 64); err == nil {
		percent = v
	}
	if s := speedRe.FindStringSubmatch(line); len(s) >= 2 {
		speed = s[1]
	}
	if e := etaRe.FindStringSubmatch(line); len(e) >= 2 {
		eta = e[1]
	}
	ok = true
	return
}

func parseHMS(h, m, s string) float64 {
	hh, _ := strconv.ParseFloat(h, 64)
	mm, _ := strconv.ParseFloat(m, 64)
	ss, _ := strconv.ParseFloat(s, 64)
	return hh*3600 + mm*60 + ss
}

// 这些站点的 HLS 流签名/CDN 不稳定，yt-dlp 原生 HLS 容易失败。
// 检测到后改用 ffmpeg 作为外部下载器（支持自动重连）+ 单线程分片。
var unstableHLSSites = []string{
	"pornhub.com", "xvideos.com", "xhamster.com",
	"spankbang.com", "eporner.com",
}

func isUnstableHLS(url string) bool {
	low := strings.ToLower(url)
	for _, s := range unstableHLSSites {
		if strings.Contains(low, s) {
			return true
		}
	}
	return false
}

var publicTrackers = []string{
	"udp://tracker.opentrackr.org:1337/announce",
	"udp://open.tracker.cl:1337/announce",
	"udp://tracker.openbittorrent.com:6969/announce",
	"udp://open.stealth.si:80/announce",
	"udp://exodus.desync.com:6969/announce",
}

func getBinPath(name string) string {
	binName := name
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}
	exe, err := os.Executable()
	if err == nil {
		// macOS .app: <exe>/../Resources/bin/  (生产环境的 bundle 内置)
		if p := filepath.Join(filepath.Dir(exe), "..", "Resources", "bin", binName); fileExists(p) {
			abs, _ := filepath.Abs(p)
			return abs
		}
		// 从 exe 目录逐级向上查找 bin/<name>，最多 8 层
		// 覆盖：生产 (exe 旁 bin/)、Wails dev 裸 exe (build/bin/)、Wails dev .app (build/bin/Flux.app/Contents/MacOS/)
		dir := filepath.Dir(exe)
		for i := 0; i < 8; i++ {
			candidate := filepath.Join(dir, "bin", binName)
			if fileExists(candidate) {
				abs, _ := filepath.Abs(candidate)
				return abs
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}
	// 兜底：走系统 PATH
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	return name
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func streamCommand(ctx context.Context, emit func(string), cmd *exec.Cmd) error {
	setupProcessGroup(cmd)
	cmd.Cancel = func() error { return killProcessTree(cmd) }
	cmd.WaitDelay = 3 * time.Second

	cmd.Stderr = cmd.Stdout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			cmd.Process.Kill()
			return ctx.Err()
		default:
			line := scanner.Text()
			if strings.TrimSpace(line) != "" {
				emit(line)
			}
		}
	}
	return cmd.Wait()
}

func buildYtdlpArgs(req DownloadRequest, url string) []string {
	ytdlp := getBinPath("yt-dlp")
	ffmpegPath := getBinPath("ffmpeg")
	ffmpegDir := filepath.Dir(ffmpegPath)
	unstableHLS := isUnstableHLS(url)

	qualityMap := map[string]string{
		"best":  "bestvideo+bestaudio/best",
		"4k":    "bestvideo[height<=2160]+bestaudio/best[height<=2160]",
		"2k":    "bestvideo[height<=1440]+bestaudio/best[height<=1440]",
		"1080p": "bestvideo[height<=1080]+bestaudio/best[height<=1080]",
		"720p":  "bestvideo[height<=720]+bestaudio/best[height<=720]",
		"480p":  "bestvideo[height<=480]+bestaudio/best[height<=480]",
		"360p":  "bestvideo[height<=360]+bestaudio/best[height<=360]",
	}

	isAudio := req.Format == "mp3"
	var fmtStr string
	if isAudio {
		fmtStr = "bestaudio/best"
	} else {
		if f, ok := qualityMap[req.Quality]; ok {
			fmtStr = f
		} else {
			fmtStr = "bestvideo+bestaudio/best"
		}
	}

	fragRetries := "5"
	concurrentFrags := fmt.Sprintf("%d", max(1, req.FragThreads))
	if unstableHLS {
		// 不稳定站点：单线程分片 + 多次重试
		fragRetries = "10"
		concurrentFrags = "1"
	}

	args := []string{
		ytdlp,
		"--format", fmtStr,
		"--output", filepath.Join(req.SaveDir, "%(title)s.%(ext)s"),
		"--ffmpeg-location", ffmpegDir,
		"--socket-timeout", "60",
		"--retries", "10",
		"--fragment-retries", fragRetries,
		"--concurrent-fragments", concurrentFrags,
		"--newline",
		"--progress",
	}

	// 自动降级：探测候选画质实际能否下载，过滤掉 Premium/失效的，选实际可下的最高画质
	// 触发条件：不稳定站点 / 选 Best/4K/2K 这类高画质（容易踩 Premium 墙）
	needCheckFormats := unstableHLS ||
		req.Quality == "best" || req.Quality == "4k" || req.Quality == "2k"
	if needCheckFormats {
		args = append(args, "--check-formats")
	}

	if unstableHLS {
		// 用 ffmpeg 作为外部下载器，比 yt-dlp 原生 HLS 更稳：
		//   -reconnect 1: 连接失败重连
		//   -reconnect_streamed 1: 流中断重连
		//   -reconnect_delay_max 5: 最长重连等待 5 秒
		args = append(args,
			"--downloader", ffmpegPath,
			"--downloader-args", "ffmpeg:-reconnect 1 -reconnect_streamed 1 -reconnect_delay_max 5",
		)
	}

	if !isAudio {
		args = append(args, "--merge-output-format", req.Format)
	} else {
		args = append(args, "--extract-audio", "--audio-format", "mp3", "--audio-quality", "192K")
	}

	// 弹幕：B站/A站 URL 才有效，把 danmaku 加到字幕语言列表
	wantDanmaku := req.Danmaku && isBiliOrAcFun(url)

	if req.Subtitle || wantDanmaku {
		langs := "zh-Hans,zh,en,ja,ko"
		if wantDanmaku {
			langs += ",danmaku"
		}
		args = append(args, "--write-subs")
		if req.Subtitle {
			args = append(args, "--write-auto-subs")
		}
		args = append(args, "--sub-langs", langs)
	}
	if req.EmbedSub {
		args = append(args, "--embed-subs")
	}
	if req.Thumbnail {
		args = append(args, "--write-thumbnail", "--embed-thumbnail")
	}

	switch req.CookieMode {
	case "browser":
		args = append(args, "--cookies-from-browser", "chrome")
	case "file":
		if req.CookieFile != "" {
			args = append(args, "--cookies", req.CookieFile)
		}
	}

	if req.Proxy != "" {
		args = append(args, "--proxy", req.Proxy)
	}
	if req.RateLimit != "" {
		args = append(args, "--limit-rate", req.RateLimit)
	}
	if req.PlaylistItems != "" {
		args = append(args, "--playlist-items", req.PlaylistItems)
	}

	args = append(args, "--", url)
	return args
}

func runYtdlpForURL(ctx context.Context, args []string, idx, total int, url string,
	emitLog func(string), emitProgress func(ProgressEvent)) error {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	setupProcessGroup(cmd)
	// ctx 取消时杀掉整个进程组（含 yt-dlp 派生的 ffmpeg）
	cmd.Cancel = func() error { return killProcessTree(cmd) }
	cmd.WaitDelay = 3 * time.Second

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		return err
	}

	const maxFragFails = 30
	fragFails := 0
	currentTrack := ""
	var ffmpegTotalSec float64    // 输入总时长
	var ffmpegBitrateKbps float64 // 输入码率，用于把 speed 倍率换算成 MB/s
	var lastEmit time.Time
	const emitMinInterval = 250 * time.Millisecond

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	// 自定义切分：同时按 \n 和 \r 拆行（ffmpeg 用 \r 刷新进度行）
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		for i, b := range data {
			if b == '\n' || b == '\r' {
				return i + 1, data[:i], nil
			}
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Track type detection
		if strings.Contains(line, "Downloading") {
			if strings.Contains(line, "video") {
				currentTrack = "视频"
			} else if strings.Contains(line, "audio") {
				currentTrack = "音频"
			}
		}

		// 抓 ffmpeg 输入头部的总时长和源码率（只抓第一次）
		if ffmpegTotalSec == 0 {
			if m := ffmpegDurationRe.FindStringSubmatch(line); len(m) >= 4 {
				ffmpegTotalSec = parseHMS(m[1], m[2], m[3])
			}
		}
		if ffmpegBitrateKbps == 0 {
			if m := ffmpegBitrateRe.FindStringSubmatch(line); len(m) >= 2 {
				if v, err := strconv.ParseFloat(m[1], 64); err == nil {
					ffmpegBitrateKbps = v
				}
			}
		}

		// Fragment failure threshold
		if strings.Contains(line, "fragment not found") ||
			strings.Contains(line, "Got error: HTTP Error 404") {
			fragFails++
			if fragFails >= maxFragFails {
				emitLog(fmt.Sprintf("❌ 连续 %d 个分片失败，中止下载（建议降画质或换源）", maxFragFails))
				cmd.Process.Kill()
				cmd.Wait()
				return fmt.Errorf("分片失败次数超阈值")
			}
		}

		// 1. yt-dlp 原生进度
		if pct, speed, eta, ok := parseYtdlpProgress(line); ok {
			if time.Since(lastEmit) >= emitMinInterval || pct >= 100 {
				emitProgress(ProgressEvent{
					TaskIdx: idx, Total: total, URL: url,
					Percent: pct, Speed: speed, ETA: eta,
					Track: currentTrack, Status: "downloading",
				})
				lastEmit = time.Now()
			}
			continue
		}

		// 2. ffmpeg 进度（time=HH:MM:SS.MS speed=Nx）
		if m := ffmpegTimeRe.FindStringSubmatch(line); len(m) >= 4 && ffmpegTotalSec > 0 {
			currentSec := parseHMS(m[1], m[2], m[3])
			pct := currentSec / ffmpegTotalSec * 100
			if pct > 100 {
				pct = 100
			}
			var speedMult float64
			if s := ffmpegSpeedRe.FindStringSubmatch(line); len(s) >= 2 {
				if v, err := strconv.ParseFloat(s[1], 64); err == nil {
					speedMult = v
				}
			}
			speed := formatFFmpegSpeed(speedMult, ffmpegBitrateKbps)
			eta := ""
			if speedMult > 0 && currentSec < ffmpegTotalSec {
				remainSec := (ffmpegTotalSec - currentSec) / speedMult
				eta = fmt.Sprintf("%d:%02d", int(remainSec)/60, int(remainSec)%60)
			}
			if time.Since(lastEmit) >= emitMinInterval {
				emitProgress(ProgressEvent{
					TaskIdx: idx, Total: total, URL: url,
					Percent: pct, Speed: speed, ETA: eta,
					Track: currentTrack, Status: "downloading",
				})
				lastEmit = time.Now()
			}
			continue
		}

		emitLog(line)
	}
	return cmd.Wait()
}

func downloadGeneral(ctx context.Context, req DownloadRequest,
	emit func(string), emitProgress func(ProgressEvent)) {
	total := len(req.URLs)
	workers := max(1, req.Workers)
	emit(fmt.Sprintf("📥 共 %d 个任务，并发 %d 个", total, workers))

	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)
	successCount := 0
	failCount := 0
	var mu sync.Mutex

	for i, url := range req.URLs {
		wg.Add(1)
		go func(idx int, u string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			tag := ""
			if total > 1 {
				tag = fmt.Sprintf("[%d/%d] ", idx+1, total)
			}
			emit(fmt.Sprintf("\n%s开始下载：%s", tag, u))
			if isUnstableHLS(u) {
				emit(tag + "💡 检测到 HLS 不稳定站点，已切换 ffmpeg 下载器 + 单线程分片")
			}
			if isUnstableHLS(u) || req.Quality == "best" || req.Quality == "4k" || req.Quality == "2k" {
				emit(tag + "🔍 启用画质可用性探测（自动跳过 Premium/失效画质，可能延迟几秒）")
			}

			args := buildYtdlpArgs(req, u)
			tagEmit := func(line string) { emit(tag + line) }
			err := runYtdlpForURL(ctx, args, idx, total, u, tagEmit, emitProgress)

			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				failCount++
				if ctx.Err() != nil {
					emit(tag + "⏹ 已停止")
					emitProgress(ProgressEvent{TaskIdx: idx, Total: total, URL: u, Status: "stopped"})
				} else {
					emit(fmt.Sprintf("%s❌ 失败: %v", tag, err))
					emitProgress(ProgressEvent{TaskIdx: idx, Total: total, URL: u, Status: "failed"})
				}
				return
			}
			successCount++
			emit(tag + "✅ 下载完成")
			emitProgress(ProgressEvent{TaskIdx: idx, Total: total, URL: u, Percent: 100, Status: "done"})
		}(i, url)
	}
	wg.Wait()

	// 弹幕后处理：把 yt-dlp 下到的 xml 弹幕转成 ass，方便播放器加载
	if req.Danmaku && successCount > 0 {
		converted, errs := ConvertDanmakuFilesInDir(req.SaveDir)
		if len(converted) > 0 {
			emit(fmt.Sprintf("🎬 已转换 %d 个弹幕文件为 ASS 格式（同名加载即可）", len(converted)))
		}
		for _, e := range errs {
			emit(fmt.Sprintf("⚠️ 弹幕转换失败: %v", e))
		}
	}

	if failCount > 0 {
		emit(fmt.Sprintf("\n完成：%d 成功，%d 失败", successCount, failCount))
	} else {
		emit(fmt.Sprintf("\n🎉 全部 %d 个任务下载完成", successCount))
	}
}

func downloadM3U8(ctx context.Context, req M3U8Request, emit func(string)) {
	ffmpeg := getBinPath("ffmpeg")
	total := len(req.URLs)

	for i, url := range req.URLs {
		select {
		case <-ctx.Done():
			emit("⏹ 已停止")
			return
		default:
		}
		filename := fmt.Sprintf("video_%d.mp4", i+1)
		output := filepath.Join(req.SaveDir, filename)
		tag := fmt.Sprintf("[%d/%d] ", i+1, total)
		emit(fmt.Sprintf("\n%s开始下载 %s", tag, filename))

		cmd := exec.CommandContext(ctx, ffmpeg,
			"-y", "-loglevel", "info",
			"-i", url,
			"-map", "0:v:0", "-map", "0:a?",
			"-c", "copy", "-bsf:a", "aac_adtstoasc",
			output)
		lineEmit := func(line string) {
			if strings.Contains(line, "time=") {
				emit(tag + line)
			}
		}
		if err := streamCommand(ctx, lineEmit, cmd); err != nil {
			emit(fmt.Sprintf("%s❌ 失败: %v", tag, err))
		} else {
			emit(tag + "✅ 完成")
		}
	}
	emit("\n🎉 所有任务完成")
}

func downloadMagnet(ctx context.Context, req MagnetRequest, emit func(string)) {
	aria2c := getBinPath("aria2c")
	total := len(req.Links)
	emit(fmt.Sprintf("\n🧲 共 %d 个磁力任务", total))

	for i, link := range req.Links {
		select {
		case <-ctx.Done():
			emit("⏹ 已停止")
			return
		default:
		}
		tag := fmt.Sprintf("[%d/%d] ", i+1, total)
		emit(fmt.Sprintf("\n%s开始: %s", tag, truncate(link, 80)))

		dlLimit := req.DLLimit
		if dlLimit == "" || dlLimit == "0" {
			dlLimit = "0"
		} else {
			dlLimit += "K"
		}
		ulLimit := req.ULLimit
		if ulLimit == "" || ulLimit == "0" {
			ulLimit = "0"
		} else {
			ulLimit += "K"
		}
		seedTime := req.SeedTime
		if seedTime == "" {
			seedTime = "0"
		}
		maxConn := fmt.Sprintf("%d", max(1, req.MaxConn))

		args := []string{
			aria2c,
			"--dir", req.SaveDir,
			"--max-connection-per-server=16",
			"--split=16",
			"--min-split-size=1M",
			"--max-overall-download-limit", dlLimit,
			"--max-overall-upload-limit", ulLimit,
			"--bt-max-peers", maxConn,
			"--seed-time", seedTime,
			"--summary-interval=3",
			"--console-log-level=notice",
			"--file-allocation=none",
			"--enable-dht=true",
			"--enable-peer-exchange=true",
		}

		if req.ExtraTracker {
			args = append(args, "--bt-tracker="+strings.Join(publicTrackers, ","))
		}

		if _, err := os.Stat(link); err == nil {
			args = append(args, "--torrent-file", link)
		} else {
			args = append(args, link)
		}

		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		lineEmit := func(line string) { emit(tag + line) }
		if err := streamCommand(ctx, lineEmit, cmd); err != nil {
			if ctx.Err() != nil {
				emit(tag + "⏹ 已终止")
			} else {
				emit(fmt.Sprintf("%s❌ 失败: %v", tag, err))
			}
		} else {
			emit(tag + "✅ 下载完成")
		}
	}
	emit("\n🎉 所有磁力任务完成")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
