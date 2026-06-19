package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ClipboardManager struct {
	config      *Config
	lastContent string
	lastType    string
	mu          sync.RWMutex
	onChange    func(content, contentType string)
}

func NewClipboardManager(config *Config) *ClipboardManager {
	return &ClipboardManager{
		config: config,
	}
}

func (cm *ClipboardManager) SetOnChangeCallback(callback func(string, string)) {
	cm.onChange = callback
}

func (cm *ClipboardManager) Start() {
	if cm.config.Clipboard.SyncEnabled {
		go cm.monitorLoop()
	}
}

func (cm *ClipboardManager) monitorLoop() {
	interval := time.Duration(cm.config.Clipboard.SyncInterval) * time.Millisecond
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		content, contentType := cm.readClipboard()

		cm.mu.Lock()
		if content != cm.lastContent || contentType != cm.lastType {
			cm.lastContent = content
			cm.lastType = contentType

			if cm.onChange != nil {
				go cm.onChange(content, contentType)
			}
		}
		cm.mu.Unlock()
	}
}

func (cm *ClipboardManager) readClipboard() (string, string) {
	return "", "text"
}

func (cm *ClipboardManager) WriteClipboard(content string) error {
	cm.mu.Lock()
	cm.lastContent = content
	cm.lastType = "text"
	cm.mu.Unlock()

	return nil
}

func (cm *ClipboardManager) GetLastContent() (string, string) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.lastContent, cm.lastType
}

type ChatManager struct {
	messages []ChatMessage
	mu       sync.RWMutex
}

type ChatMessage struct {
	ID        string    `json:"id"`
	From      string    `json:"from"`
	FromName  string    `json:"from_name"`
	To        string    `json:"to"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

func NewChatManager() *ChatManager {
	return &ChatManager{
		messages: make([]ChatMessage, 0),
	}
}

func (cm *ChatManager) AddMessage(msg ChatMessage) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.messages = append(cm.messages, msg)

	if len(cm.messages) > 1000 {
		cm.messages = cm.messages[len(cm.messages)-1000:]
	}
}

func (cm *ChatManager) GetMessages(limit int) []ChatMessage {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if limit <= 0 || limit > len(cm.messages) {
		limit = len(cm.messages)
	}

	start := len(cm.messages) - limit
	return cm.messages[start:]
}

func (cm *ChatManager) GetMessagesSince(since time.Time) []ChatMessage {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var result []ChatMessage
	for _, msg := range cm.messages {
		if msg.Timestamp.After(since) {
			result = append(result, msg)
		}
	}
	return result
}

type ShareManager struct {
	config *Config
}

func NewShareManager(config *Config) *ShareManager {
	return &ShareManager{config: config}
}

func (sm *ShareManager) ListShares() []ShareInfo {
	var shares []ShareInfo
	for _, share := range sm.config.Shares {
		info := ShareInfo{
			Name:     share.Name,
			Path:     share.Path,
			Readonly: share.Readonly,
		}

		if stat, err := os.Stat(share.Path); err == nil {
			info.Exists = true
			info.IsDir = stat.IsDir()
		}

		shares = append(shares, info)
	}
	return shares
}

type ShareInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Readonly bool   `json:"readonly"`
	Exists   bool   `json:"exists"`
	IsDir    bool   `json:"is_dir"`
}

func (sm *ShareManager) ListFiles(shareName string, subPath string) ([]FileInfo, error) {
	var sharePath string
	for _, share := range sm.config.Shares {
		if share.Name == shareName {
			sharePath = share.Path
			break
		}
	}

	if sharePath == "" {
		return nil, fmt.Errorf("share not found: %s", shareName)
	}

	fullPath := filepath.Join(sharePath, filepath.Clean("/"+subPath))

	if !strings.HasPrefix(fullPath, sharePath) {
		return nil, fmt.Errorf("path traversal detected")
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, err
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		files = append(files, FileInfo{
			Name:    entry.Name(),
			Size:    info.Size(),
			IsDir:   entry.IsDir(),
			ModTime: info.ModTime(),
		})
	}

	return files, nil
}

type FileInfo struct {
	Name    string      `json:"name"`
	Size    int64       `json:"size"`
	IsDir   bool        `json:"is_dir"`
	ModTime time.Time   `json:"mod_time"`
}

func (sm *ShareManager) GetFilePath(shareName string, fileName string) (string, error) {
	for _, share := range sm.config.Shares {
		if share.Name == shareName {
			fullPath := filepath.Join(share.Path, fileName)
			if _, err := os.Stat(fullPath); err == nil {
				return fullPath, nil
			}
		}
	}
	return "", fmt.Errorf("file not found")
}
