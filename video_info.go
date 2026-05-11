package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os/exec"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/proxy"
)

type cachedVideoInfo struct {
	info      *VideoInfo
	timestamp time.Time
}

var (
	videoInfoCache   = make(map[string]cachedVideoInfo)
	videoInfoCacheMu sync.RWMutex
	videoInfoTTL     = 1 * time.Hour
)

const maxThumbnailBytes = 5 * 1024 * 1024 // 5MB

func makeHTTPClientWithProxy(proxyURL string) *http.Client {
	transport := &http.Transport{}
	if proxyURL != "" {
		if u, err := neturl.Parse(proxyURL); err == nil {
			switch u.Scheme {
			case "http", "https":
				transport.Proxy = http.ProxyURL(u)
			case "socks5", "socks5h":
				if dialer, err := proxy.FromURL(u, proxy.Direct); err == nil {
					transport.Dial = dialer.Dial
				}
			}
		}
	}
	return &http.Client{Timeout: 10 * time.Second, Transport: transport}
}

func fetchImageAsDataURL(url, proxyURL string) (string, error) {
	client := makeHTTPClientWithProxy(proxyURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxThumbnailBytes+1))
	if err != nil {
		return "", err
	}
	if len(body) > maxThumbnailBytes {
		return "", fmt.Errorf("image too large")
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" || !strings.HasPrefix(contentType, "image/") {
		contentType = "image/jpeg"
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(body), nil
}

type EpisodeInfo struct {
	Title    string `json:"title"`
	Duration int    `json:"duration"`
}

type VideoInfo struct {
	Title         string        `json:"title"`
	Uploader      string        `json:"uploader"`
	Duration      int           `json:"duration"`
	Thumbnail     string        `json:"thumbnail"`
	Description   string        `json:"description"`
	UploadDate    string        `json:"uploadDate"`
	ViewCount     int64         `json:"viewCount"`
	Resolutions   []int         `json:"resolutions"`
	Filesize      int64         `json:"filesize"`
	IsPlaylist    bool          `json:"isPlaylist"`
	PlaylistTitle string        `json:"playlistTitle,omitempty"`
	Episodes      []EpisodeInfo `json:"episodes,omitempty"`
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getInt(m map[string]interface{}, key string) int {
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

func getInt64(m map[string]interface{}, key string) int64 {
	if v, ok := m[key].(float64); ok {
		return int64(v)
	}
	return 0
}

func firstNonEmpty(args ...string) string {
	for _, s := range args {
		if s != "" {
			return s
		}
	}
	return ""
}

// ClearVideoInfoCache 清空所有缓存的视频信息（含封面）。
// 返回清掉的条目数。
func (a *App) ClearVideoInfoCache() int {
	videoInfoCacheMu.Lock()
	n := len(videoInfoCache)
	videoInfoCache = make(map[string]cachedVideoInfo)
	videoInfoCacheMu.Unlock()
	return n
}

type ParseInfoRequest struct {
	URL        string `json:"url"`
	CookieMode string `json:"cookieMode"`
	CookieFile string `json:"cookieFile"`
	Proxy      string `json:"proxy"`
}

func (a *App) ParseVideoInfo(req ParseInfoRequest) (*VideoInfo, error) {
	if strings.TrimSpace(req.URL) == "" {
		return nil, fmt.Errorf("URL 不能为空")
	}

	// 缓存命中（同 URL + cookie 模式 + 代理）
	cacheKey := req.URL + "|" + req.CookieMode + "|" + req.CookieFile + "|" + req.Proxy
	videoInfoCacheMu.RLock()
	entry, hit := videoInfoCache[cacheKey]
	videoInfoCacheMu.RUnlock()
	if hit && time.Since(entry.timestamp) < videoInfoTTL {
		return entry.info, nil
	}

	ytdlp := getBinPath("yt-dlp")
	args := []string{
		"--dump-single-json",
		"--skip-download",
		"--no-warnings",
		"--no-playlist", // get single video by default; playlist users 解析 first entry
		"--socket-timeout", "30",
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
	args = append(args, "--", req.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, ytdlp, args...)
	out, err := cmd.Output()
	if err != nil {
		stderr := ""
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = strings.TrimSpace(string(exitErr.Stderr))
		}
		if stderr == "" {
			stderr = err.Error()
		}
		return nil, fmt.Errorf("解析失败: %s", stderr)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("解析 JSON 失败: %v", err)
	}

	info := &VideoInfo{
		Title:       getString(raw, "title"),
		Uploader:    firstNonEmpty(getString(raw, "uploader"), getString(raw, "channel"), getString(raw, "creator")),
		Duration:    getInt(raw, "duration"),
		Thumbnail:   getString(raw, "thumbnail"),
		Description: getString(raw, "description"),
		UploadDate:  getString(raw, "upload_date"),
		ViewCount:   getInt64(raw, "view_count"),
	}

	if info.Filesize == 0 {
		info.Filesize = getInt64(raw, "filesize_approx")
	}

	// Extract resolutions and approx total filesize from formats
	if formats, ok := raw["formats"].([]interface{}); ok {
		resSet := make(map[int]struct{})
		var maxFilesize int64
		var bestBitrate float64 // Kbps，作为 filesize 的 fallback 估算依据
		for _, f := range formats {
			fm, ok := f.(map[string]interface{})
			if !ok {
				continue
			}
			if h := getInt(fm, "height"); h > 0 {
				resSet[h] = struct{}{}
			}
			fs := getInt64(fm, "filesize")
			if fs == 0 {
				fs = getInt64(fm, "filesize_approx")
			}
			if fs > maxFilesize {
				maxFilesize = fs
			}
			// 收集最高码率：tbr 优先（总码率），否则 vbr+abr
			var br float64
			if v, ok := fm["tbr"].(float64); ok && v > 0 {
				br = v
			} else {
				vbr, _ := fm["vbr"].(float64)
				abr, _ := fm["abr"].(float64)
				br = vbr + abr
			}
			if br > bestBitrate {
				bestBitrate = br
			}
		}
		for r := range resSet {
			info.Resolutions = append(info.Resolutions, r)
		}
		sort.Sort(sort.Reverse(sort.IntSlice(info.Resolutions)))
		if info.Filesize == 0 {
			info.Filesize = maxFilesize
		}
		// HLS 流通常没 filesize 字段，用 bitrate(Kbps) * duration(s) / 8 估算（Bytes）
		if info.Filesize == 0 && info.Duration > 0 && bestBitrate > 0 {
			info.Filesize = int64(bestBitrate * 1000 / 8 * float64(info.Duration))
		}
	}

	// Playlist
	if entries, ok := raw["entries"].([]interface{}); ok && len(entries) > 0 {
		info.IsPlaylist = true
		info.PlaylistTitle = info.Title
		// Use first entry as the displayed video
		if first, ok := entries[0].(map[string]interface{}); ok {
			if t := getString(first, "title"); t != "" {
				info.Title = t
			}
			if t := getString(first, "thumbnail"); t != "" {
				info.Thumbnail = t
			}
		}
		for _, e := range entries {
			em, ok := e.(map[string]interface{})
			if !ok {
				continue
			}
			info.Episodes = append(info.Episodes, EpisodeInfo{
				Title:    getString(em, "title"),
				Duration: getInt(em, "duration"),
			})
		}
	}

	// 后端代理拉取封面（绕过浏览器 CORS/防盗链/海外延迟）
	if info.Thumbnail != "" {
		if dataURL, err := fetchImageAsDataURL(info.Thumbnail, req.Proxy); err == nil {
			info.Thumbnail = dataURL
		}
		// 失败时保留原 URL，前端浏览器再尝试直链
	}

	// 写入缓存
	videoInfoCacheMu.Lock()
	videoInfoCache[cacheKey] = cachedVideoInfo{info: info, timestamp: time.Now()}
	videoInfoCacheMu.Unlock()

	return info, nil
}
