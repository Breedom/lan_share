package core

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Breedom/lan_share/internal/models"
)

const (
	ChunkSize       = 64 * 1024
	ProtocolMagic   = 0x4C414E54
	ProtocolVersion = 1
)

type PacketType byte

const (
	PacketMeta PacketType = iota + 1
	PacketData
	PacketAck
	PacketDone
	PacketError
	PacketResume
)

type PacketHeader struct {
	Magic      uint32
	Version    uint8
	Type       PacketType
	SessionID  uint64
	Sequence   uint32
	PayloadLen uint32
	Checksum   uint32
}

type TransferSession struct {
	ID        string
	FilePath  string
	FileSize  int64
	Offset    int64
	StartTime time.Time
}

type TransferManager struct {
	config   *Config
	crypto   *Crypto
	sessions map[string]*TransferSession
	mu       sync.RWMutex
}

func NewTransferManager(config *Config) *TransferManager {
	return &TransferManager{
		config:   config,
		crypto:   NewCrypto("lanshare-default-key"),
		sessions: make(map[string]*TransferSession),
	}
}

func (tm *TransferManager) Start() {
	go tm.cleanupOldSessions()
}

func (tm *TransferManager) SendFile(targetIP string, targetPort int, filePath string, onProgress func(float64)) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("file not found: %w", err)
	}

	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", targetIP, targetPort), 5*time.Second)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	sessionID := generateSessionID()

	meta := &models.FileMeta{
		Name: info.Name(),
		Size: info.Size(),
	}

	if err := tm.sendMeta(conn, sessionID, meta); err != nil {
		return err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var offset int64
	buf := make([]byte, ChunkSize)

	for {
		n, err := file.Read(buf)
		if n > 0 {
			encrypted, err := tm.crypto.Encrypt(buf[:n])
			if err != nil {
				return err
			}

			if err := tm.sendData(conn, sessionID, offset, encrypted); err != nil {
				return err
			}

			offset += int64(n)
			progress := float64(offset) / float64(info.Size()) * 100
			if onProgress != nil {
				onProgress(progress)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
	}

	return tm.sendDone(conn, sessionID)
}

func (tm *TransferManager) StartServer() error {
	addr := &net.TCPAddr{
		Port: tm.config.Server.TCPPort,
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to start TCP server: %w", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			go tm.handleIncoming(conn)
		}
	}()

	return nil
}

func (tm *TransferManager) handleIncoming(conn net.Conn) {
	defer conn.Close()

	header := make([]byte, 32)
	if _, err := io.ReadFull(conn, header); err != nil {
		return
	}

	pkt := tm.parseHeader(header)
	if pkt.Magic != ProtocolMagic {
		return
	}

	switch pkt.Type {
	case PacketMeta:
		tm.handleMeta(conn, pkt)
	case PacketData:
		tm.handleData(conn, pkt)
	case PacketResume:
		tm.handleResume(conn, pkt)
	}
}

func (tm *TransferManager) handleMeta(conn net.Conn, pkt *PacketHeader) {
	payload := make([]byte, pkt.PayloadLen)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return
	}

	var meta models.FileMeta
	if err := json.Unmarshal(payload, &meta); err != nil {
		return
	}

	sharePath := tm.getSharePath(meta.Path)
	if sharePath == "" {
		tm.sendError(conn, pkt.SessionID, "no share path")
		return
	}

	fullPath := filepath.Join(sharePath, meta.Name)

	tm.mu.Lock()
	tm.sessions[fmt.Sprintf("%d", pkt.SessionID)] = &TransferSession{
		ID:       fmt.Sprintf("%d", pkt.SessionID),
		FilePath: fullPath,
		FileSize: meta.Size,
	}
	tm.mu.Unlock()

	existingSize := int64(0)
	if info, err := os.Stat(fullPath); err == nil {
		existingSize = info.Size()
	}

	tm.sendAck(conn, pkt.SessionID, existingSize)
}

func (tm *TransferManager) handleData(conn net.Conn, pkt *PacketHeader) {
	payload := make([]byte, pkt.PayloadLen)
	if _, err := io.ReadFull(conn, payload); err != nil {
		return
	}

	tm.mu.RLock()
	session, ok := tm.sessions[fmt.Sprintf("%d", pkt.SessionID)]
	tm.mu.RUnlock()

	if !ok {
		return
	}

	decrypted, err := tm.crypto.Decrypt(payload)
	if err != nil {
		return
	}

	file, err := os.OpenFile(session.FilePath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer file.Close()

	offset := int64(pkt.Sequence) * ChunkSize
	file.WriteAt(decrypted, offset)

	tm.sendAck(conn, pkt.SessionID, offset+int64(len(decrypted)))
}

func (tm *TransferManager) handleResume(conn net.Conn, pkt *PacketHeader) {
	tm.sendAck(conn, pkt.SessionID, tm.getFileSize(pkt.SessionID))
}

func (tm *TransferManager) sendMeta(conn net.Conn, sessionID string, meta *models.FileMeta) error {
	payload, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	sid := parseSessionID(sessionID)
	header := tm.buildHeader(PacketMeta, sid, 0, uint32(len(payload)))

	conn.Write(header)
	conn.Write(payload)
	return nil
}

func (tm *TransferManager) sendData(conn net.Conn, sessionID string, offset int64, data []byte) error {
	sid := parseSessionID(sessionID)
	seq := uint32(offset / ChunkSize)
	header := tm.buildHeader(PacketData, sid, seq, uint32(len(data)))

	conn.Write(header)
	conn.Write(data)
	return nil
}

func (tm *TransferManager) sendDone(conn net.Conn, sessionID string) error {
	sid := parseSessionID(sessionID)
	header := tm.buildHeader(PacketDone, sid, 0, 0)

	conn.Write(header)
	return nil
}

func (tm *TransferManager) sendAck(conn net.Conn, sessionID string, offset int64) error {
	sid := parseSessionID(sessionID)
	header := tm.buildHeader(PacketAck, sid, 0, 8)

	conn.Write(header)
	binary.Write(conn, binary.LittleEndian, offset)
	return nil
}

func (tm *TransferManager) sendError(conn net.Conn, sessionID string, errMsg string) error {
	sid := parseSessionID(sessionID)
	data := []byte(errMsg)
	header := tm.buildHeader(PacketError, sid, 0, uint32(len(data)))

	conn.Write(header)
	conn.Write(data)
	return nil
}

func (tm *TransferManager) buildHeader(pktType PacketType, sessionID uint64, seq uint32, payloadLen uint32) []byte {
	header := make([]byte, 32)
	binary.LittleEndian.PutUint32(header[0:4], ProtocolMagic)
	header[4] = ProtocolVersion
	header[5] = byte(pktType)
	binary.LittleEndian.PutUint64(header[6:14], sessionID)
	binary.LittleEndian.PutUint32(header[14:18], seq)
	binary.LittleEndian.PutUint32(header[18:22], payloadLen)
	return header
}

func (tm *TransferManager) parseHeader(data []byte) *PacketHeader {
	return &PacketHeader{
		Magic:      binary.LittleEndian.Uint32(data[0:4]),
		Version:    data[4],
		Type:       PacketType(data[5]),
		SessionID:  binary.LittleEndian.Uint64(data[6:14]),
		Sequence:   binary.LittleEndian.Uint32(data[14:18]),
		PayloadLen: binary.LittleEndian.Uint32(data[18:22]),
		Checksum:   binary.LittleEndian.Uint32(data[22:26]),
	}
}

func (tm *TransferManager) getSharePath(virtualPath string) string {
	for _, share := range tm.config.Shares {
		if virtualPath == share.Name || virtualPath == "" {
			return share.Path
		}
	}
	if len(tm.config.Shares) > 0 {
		return tm.config.Shares[0].Path
	}
	return ""
}

func (tm *TransferManager) getFileSize(sessionID string) int64 {
	tm.mu.RLock()
	session, ok := tm.sessions[sessionID]
	tm.mu.RUnlock()

	if !ok {
		return 0
	}

	info, err := os.Stat(session.FilePath)
	if err != nil {
		return 0
	}
	return info.Size()
}

func (tm *TransferManager) cleanupOldSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		tm.mu.Lock()
		for id, session := range tm.sessions {
			if time.Since(session.StartTime) > 30*time.Minute {
				delete(tm.sessions, id)
			}
		}
		tm.mu.Unlock()
	}
}

func generateSessionID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func parseSessionID(s string) uint64 {
	var id uint64
	fmt.Sscanf(s, "%d", &id)
	return id
}
