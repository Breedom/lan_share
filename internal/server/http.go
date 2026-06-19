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

	exePath, _ := os.Executable()
	webDir := filepath.Join(filepath.Dir(exePath), "web")
	if _, err := os.Stat(filepath.Join(webDir, "static")); os.IsNotExist(err) {
		webDir = filepath.Join("web")
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(filepath.Join(webDir, "static")))))

	addr := fmt.Sprintf(":%d", s.config.Server.HTTPPort)
	log.Printf("HTTP server starting on %s", addr)
	log.Printf("Web UI: http://localhost%s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}

func (s *HTTPServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	exePath, _ := os.Executable()
	webDir := filepath.Join(filepath.Dir(exePath), "web")
	if _, err := os.Stat(filepath.Join(webDir, "index.html")); os.IsNotExist(err) {
		webDir = filepath.Join("web")
	}

	http.ServeFile(w, r, filepath.Join(webDir, "index.html"))
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
