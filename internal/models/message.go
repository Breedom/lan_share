package models

import (
	"time"
)

type MessageType string

const (
	MessageClipboard MessageType = "clipboard"
	MessageChat      MessageType = "chat"
	MessageFile      MessageType = "file"
	MessageTransfer  MessageType = "transfer"
)

type Message struct {
	Type      MessageType `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

type ClipboardData struct {
	ContentType string `json:"content_type"`
	Content     string `json:"content"`
	SourceDevice string `json:"source_device"`
}

type ChatData struct {
	MessageID    string `json:"message_id"`
	From         string `json:"from"`
	FromName     string `json:"from_name"`
	To           string `json:"to"`
	Content      string `json:"content"`
}

type TransferData struct {
	SessionID  string  `json:"session_id"`
	FileName   string  `json:"file_name"`
	FileSize   int64   `json:"file_size"`
	Progress   float64 `json:"progress"`
	Status     string  `json:"status"`
	Error      string  `json:"error,omitempty"`
}

type DeviceInfo struct {
	Device *Device `json:"device"`
	Online bool    `json:"online"`
}
