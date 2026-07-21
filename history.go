package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HistoryRecord 一条任务历史。Payload 存原始请求 JSON，
// 前端点「继续」时原样重新发起（yt-dlp/aria2c 靠 .part/.aria2 控制文件断点续传）。
type HistoryRecord struct {
	ID        string `json:"id"`
	Type      string `json:"type"`  // general | m3u8 | magnet | convert
	Title     string `json:"title"` // 首个 URL / 输入目录
	Count     int    `json:"count"` // 任务条数
	SaveDir   string `json:"saveDir"`
	Status    string `json:"status"` // running | done | partial | failed | stopped | interrupted
	Detail    string `json:"detail"`
	Payload   string `json:"payload"`
	CreatedAt int64  `json:"createdAt"` // unix 秒
	UpdatedAt int64  `json:"updatedAt"`
}

const historyLimit = 200

type HistoryStore struct {
	mu      sync.Mutex
	path    string
	records []HistoryRecord // 新的在前
}

func NewHistoryStore() *HistoryStore {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return &HistoryStore{path: filepath.Join(dir, "flux", "history.json")}
}

// Load 读取历史；上次退出时仍是 running 的任务标记为 interrupted
func (h *HistoryStore) Load() {
	h.mu.Lock()
	defer h.mu.Unlock()
	data, err := os.ReadFile(h.path)
	if err != nil {
		return
	}
	if err := json.Unmarshal(data, &h.records); err != nil {
		h.records = nil
		return
	}
	changed := false
	for i := range h.records {
		if h.records[i].Status == "running" {
			h.records[i].Status = "interrupted"
			h.records[i].Detail = "上次退出时未完成，可继续"
			h.records[i].UpdatedAt = time.Now().Unix()
			changed = true
		}
	}
	if changed {
		h.saveLocked()
	}
}

func (h *HistoryStore) saveLocked() {
	if err := os.MkdirAll(filepath.Dir(h.path), 0755); err != nil {
		return
	}
	data, err := json.MarshalIndent(h.records, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(h.path, data, 0644)
}

// Add 新增记录并返回 ID；同类型同参数的旧记录会被去重（重试/继续不产生重复条目）
func (h *HistoryStore) Add(rec HistoryRecord) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now()
	rec.ID = fmt.Sprintf("%d", now.UnixNano())
	rec.CreatedAt = now.Unix()
	rec.UpdatedAt = rec.CreatedAt

	kept := h.records[:0]
	for _, r := range h.records {
		if r.Type == rec.Type && r.Payload == rec.Payload && r.Status != "running" {
			continue
		}
		kept = append(kept, r)
	}
	h.records = append([]HistoryRecord{rec}, kept...)
	if len(h.records) > historyLimit {
		h.records = h.records[:historyLimit]
	}
	h.saveLocked()
	return rec.ID
}

func (h *HistoryStore) Update(id, status, detail string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := range h.records {
		if h.records[i].ID == id {
			h.records[i].Status = status
			h.records[i].Detail = detail
			h.records[i].UpdatedAt = time.Now().Unix()
			h.saveLocked()
			return
		}
	}
}

func (h *HistoryStore) List() []HistoryRecord {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]HistoryRecord, len(h.records))
	copy(out, h.records)
	return out
}

func (h *HistoryStore) Delete(id string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i := range h.records {
		if h.records[i].ID == id {
			h.records = append(h.records[:i], h.records[i+1:]...)
			h.saveLocked()
			return
		}
	}
}

func (h *HistoryStore) Clear() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	n := len(h.records)
	h.records = nil
	h.saveLocked()
	return n
}
