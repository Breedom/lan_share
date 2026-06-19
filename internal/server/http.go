package server

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Breedom/lan_share/internal/core"
	"github.com/gorilla/websocket"
)

type HTTPServer struct {
	config    *core.Config
	discovery *core.Discovery
	transfer  *core.TransferManager
	chat      *core.ChatManager
	share     *core.ShareManager
	upgrader  websocket.Upgrader
	wsClients map[*websocket.Conn]bool
	mu        sync.RWMutex
}

func NewHTTPServer(config *core.Config) *HTTPServer {
	return &HTTPServer{
		config:    config,
		discovery: core.NewDiscovery(config),
		transfer:  core.NewTransferManager(config),
		chat:      core.NewChatManager(),
		share:     core.NewShareManager(config),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		wsClients: make(map[*websocket.Conn]bool),
	}
}

func (s *HTTPServer) Start() {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/device", s.handleDeviceInfo)
	mux.HandleFunc("/api/peers", s.handlePeers)
	mux.HandleFunc("/api/shares", s.handleShares)
	mux.HandleFunc("/api/files/", s.handleFiles)
	mux.HandleFunc("/api/download/", s.handleDownload)
	mux.HandleFunc("/api/upload/", s.handleUpload)
	mux.HandleFunc("/api/chat/history", s.handleChatHistory)
	mux.HandleFunc("/ws", s.handleWebSocket)

	addr := fmt.Sprintf(":%d", s.config.Server.HTTPPort)
	log.Printf("HTTP server starting on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

func (s *HTTPServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	html := `<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
    <title>LanShare</title>
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
    <link rel="stylesheet" href="/static/style.css">
    <link rel="icon" href="data:image/svg+xml,<svg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'><text y='.9em' font-size='90'>📡</text></svg>">
</head>
<body>
    <div class="app">
        <header class="header">
            <div class="header-content">
                <div class="logo">
                    <div class="logo-icon">📡</div>
                    <div class="logo-text">
                        <h1>LanShare</h1>
                        <p>局域网文件共享</p>
                    </div>
                </div>
                <div class="header-status">
                    <div class="status-badge">
                        <span class="status-dot"></span>
                        <span id="device-count">在线</span>
                    </div>
                </div>
            </div>
        </header>

        <main class="main">
            <div class="content-grid">
                <div class="card" id="devices-card">
                    <div class="card-header">
                        <div class="card-title">
                            <span class="card-title-icon">📱</span>
                            设备列表
                        </div>
                        <span class="card-badge" id="peers-count">0</span>
                    </div>
                    <div class="card-body">
                        <div id="device-list" class="device-list"></div>
                    </div>
                </div>

                <div class="card" id="transfer-card">
                    <div class="card-header">
                        <div class="card-title">
                            <span class="card-title-icon">📁</span>
                            文件传输
                        </div>
                    </div>
                    <div class="card-body transfer-section">
                        <div class="upload-zone" id="upload-zone">
                            <div class="upload-icon">☁️</div>
                            <p class="upload-text">拖拽文件到此处上传</p>
                            <p class="upload-hint">或点击选择文件</p>
                            <button class="upload-btn" id="upload-btn">
                                <span>📎</span> 选择文件
                            </button>
                            <input type="file" id="file-input" multiple hidden>
                        </div>
                        <div id="transfer-list" class="transfer-list"></div>
                    </div>
                </div>

                <div class="card chat-section" id="chat-card">
                    <div class="card-header">
                        <div class="card-title">
                            <span class="card-title-icon">💬</span>
                            消息
                        </div>
                    </div>
                    <div class="chat-messages" id="chat-messages"></div>
                    <div class="chat-input-area">
                        <div class="chat-input-wrapper">
                            <input type="text" class="chat-input" id="message-input" placeholder="输入消息...">
                            <button class="chat-send-btn" id="send-btn" disabled>
                                <span>➤</span>
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </main>

        <div class="toast-container" id="toast-container"></div>

        <nav class="mobile-nav">
            <div class="nav-items">
                <button class="nav-item active" data-tab="devices">
                    <span class="nav-item-icon">📱</span>
                    设备
                </button>
                <button class="nav-item" data-tab="transfer">
                    <span class="nav-item-icon">📁</span>
                    传输
                </button>
                <button class="nav-item" data-tab="chat">
                    <span class="nav-item-icon">💬</span>
                    消息
                </button>
            </div>
        </nav>
    </div>
    <script src="/static/app.js"></script>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func (s *HTTPServer) handleDeviceInfo(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(s.discovery.GetDevice())
}

func (s *HTTPServer) handlePeers(w http.ResponseWriter, r *http.Request) {
	peers := s.discovery.GetPeers()
	json.NewEncoder(w).Encode(peers)
}

func (s *HTTPServer) handleShares(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		shares := s.share.ListShares()
		json.NewEncoder(w).Encode(shares)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *HTTPServer) handleFiles(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/files/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) < 1 || parts[0] == "" {
		http.Error(w, "Share name required", http.StatusBadRequest)
		return
	}

	shareName := parts[0]
	subPath := ""
	if len(parts) > 1 {
		subPath = parts[1]
	}

	files, err := s.share.ListFiles(shareName, subPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(files)
}

func (s *HTTPServer) handleDownload(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/download/")
	parts := strings.SplitN(path, "/", 2)

	if len(parts) < 2 {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	shareName := parts[0]
	fileName := parts[1]

	filePath, err := s.share.GetFilePath(shareName, fileName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	http.ServeFile(w, r, filePath)
}

func (s *HTTPServer) handleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	shareName := strings.TrimPrefix(r.URL.Path, "/api/upload/")
	if shareName == "" {
		shareName = s.config.Shares[0].Name
	}

	r.ParseMultipartForm(32 << 20)

	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()

	sharePath := ""
	for _, share := range s.config.Shares {
		if share.Name == shareName {
			sharePath = share.Path
			break
		}
	}

	if sharePath == "" {
		http.Error(w, "Share not found", http.StatusNotFound)
		return
	}

	dstPath := filepath.Join(sharePath, handler.Filename)
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	io.Copy(dst, file)

	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "File uploaded successfully",
	})
}

func (s *HTTPServer) handleChatHistory(w http.ResponseWriter, r *http.Request) {
	messages := s.chat.GetMessages(50)
	json.NewEncoder(w).Encode(messages)
}

func (s *HTTPServer) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	s.mu.Lock()
	s.wsClients[conn] = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.wsClients, conn)
		s.mu.Unlock()
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg struct {
			Type    string          `json:"type"`
			Payload json.RawMessage `json:"payload"`
		}

		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case "chat":
			var chatMsg struct {
				To      string `json:"to"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(msg.Payload, &chatMsg); err == nil {
				s.handleChatMessage(conn, chatMsg.To, chatMsg.Content)
			}
		}
	}
}

func (s *HTTPServer) handleChatMessage(from *websocket.Conn, to string, content string) {
	msg := core.ChatMessage{
		ID:        fmt.Sprintf("%d", len(s.chat.GetMessages(0))+1),
		From:      "local",
		FromName:  s.config.General.DeviceName,
		To:        to,
		Content:   content,
	}

	s.chat.AddMessage(msg)

	s.mu.RLock()
	for client := range s.wsClients {
		if client != from {
			data, _ := json.Marshal(msg)
			client.WriteMessage(websocket.TextMessage, data)
		}
	}
	s.mu.RUnlock()
}
