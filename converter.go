package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var supportedFormats = map[string]bool{
	"mp3": true, "wav": true, "flac": true, "m4a": true, "ogg": true,
}

type ConvertTask struct {
	Src string
	Dst string
}

type ConvertResult struct {
	Src     string `json:"src"`
	Dst     string `json:"dst"`
	Success bool   `json:"success"`
	Message string `json:"message"`
}

func getFFmpegBin() (string, error) {
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		return path, nil
	}
	var candidates []string
	if runtime.GOOS == "windows" {
		candidates = []string{
			`C:\ffmpeg\bin\ffmpeg.exe`,
			`C:\Program Files\ffmpeg\bin\ffmpeg.exe`,
			`C:\ProgramData\chocolatey\bin\ffmpeg.exe`,
		}
	} else if runtime.GOOS == "darwin" {
		candidates = []string{
			"/opt/homebrew/bin/ffmpeg",
			"/usr/local/bin/ffmpeg",
		}
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("未找到 ffmpeg，请安装并确保命令可用")
}

func buildFFmpegCmd(ffmpeg, input, output, format, bitrate string) ([]string, error) {
	lossless := strings.ToLower(bitrate) == "lossless"
	base := []string{ffmpeg, "-y", "-hide_banner", "-loglevel", "error", "-i", input, "-map_metadata", "0"}

	switch format {
	case "mp3":
		if lossless {
			return nil, fmt.Errorf("mp3 不支持无损，请选择 flac/wav/m4a")
		}
		return append(base, "-codec:a", "libmp3lame", "-b:a", bitrate, output), nil
	case "m4a":
		if lossless {
			return append(base, "-codec:a", "alac", output), nil
		}
		return append(base, "-codec:a", "aac", "-b:a", bitrate, output), nil
	case "ogg":
		if lossless {
			return nil, fmt.Errorf("ogg 不支持无损，请选择 flac/wav/m4a")
		}
		return append(base, "-codec:a", "libvorbis", "-b:a", bitrate, output), nil
	case "flac":
		return append(base, "-codec:a", "flac", output), nil
	case "wav":
		return append(base, "-codec:a", "pcm_s16le", output), nil
	}
	return nil, fmt.Errorf("不支持的格式: %s", format)
}

func runFFmpeg(ffmpeg, input, output, format, bitrate string, overwrite bool) (bool, string) {
	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return false, fmt.Sprintf("创建目录失败: %v", err)
	}
	if !overwrite {
		if _, err := os.Stat(output); err == nil {
			return true, "skip: 目标文件已存在"
		}
	}
	cmd, err := buildFFmpegCmd(ffmpeg, input, output, format, bitrate)
	if err != nil {
		return false, err.Error()
	}
	if !overwrite {
		cmd[1] = "-n"
	}
	out, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return false, fmt.Sprintf("ffmpeg 失败: %s", msg)
	}
	return true, "ok"
}

func convertOne(task ConvertTask, format, bitrate string, overwrite bool) ConvertResult {
	if !overwrite {
		if _, err := os.Stat(task.Dst); err == nil {
			return ConvertResult{Src: task.Src, Dst: task.Dst, Success: true, Message: "skip: 目标文件已存在"}
		}
	}

	ffmpeg, err := getFFmpegBin()
	if err != nil {
		return ConvertResult{Src: task.Src, Dst: task.Dst, Success: false, Message: err.Error()}
	}

	input := task.Src
	var tmpPath string

	srcExt := strings.ToLower(filepath.Ext(task.Src))
	if srcExt == ".ncm" {
		decoded, err := decodeNcmToTemp(task.Src)
		if err != nil {
			return ConvertResult{Src: task.Src, Dst: task.Dst, Success: false, Message: fmt.Sprintf(".ncm 解码失败: %v", err)}
		}
		input = decoded.Path
		tmpPath = decoded.Path
		defer os.Remove(tmpPath)
	} else if isQMCFile(task.Src) {
		decoded, err := decodeQMCToTemp(task.Src)
		if err != nil {
			return ConvertResult{Src: task.Src, Dst: task.Dst, Success: false, Message: fmt.Sprintf("QMC 解码失败: %v", err)}
		}
		input = decoded.Path
		tmpPath = decoded.Path
		defer os.Remove(tmpPath)
	}

	if err := os.MkdirAll(filepath.Dir(task.Dst), 0755); err != nil {
		return ConvertResult{Src: task.Src, Dst: task.Dst, Success: false, Message: fmt.Sprintf("创建目录失败: %v", err)}
	}

	ok, msg := runFFmpeg(ffmpeg, input, task.Dst, format, bitrate, overwrite)
	return ConvertResult{Src: task.Src, Dst: task.Dst, Success: ok, Message: msg}
}

func convertBatch(tasks []ConvertTask, format, bitrate string, overwrite bool, workers int) []ConvertResult {
	results := make([]ConvertResult, len(tasks))
	var wg sync.WaitGroup
	sem := make(chan struct{}, workers)

	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t ConvertTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[idx] = convertOne(t, format, bitrate, overwrite)
		}(i, task)
	}
	wg.Wait()
	return results
}

func findAudioFiles(dir string, exts map[string]bool, recursive bool) ([]string, error) {
	var files []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			if recursive {
				sub, err := findAudioFiles(filepath.Join(dir, e.Name()), exts, true)
				if err == nil {
					files = append(files, sub...)
				}
			}
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if exts[ext] {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}
