package core

import (
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/Breedom/lan_share/internal/models"
)

const (
	BroadcastPort = 53317
	BroadcastAddr = "255.255.255.255"
	HeartbeatInterval = 3 * time.Second
	DeviceTimeout = 30 * time.Second
)

type Discovery struct {
	config     *Config
	device     *models.Device
	peers      map[string]*models.Device
	mu         sync.RWMutex
	conn       *net.UDPConn
	running    bool
	onDeviceFound func(*models.Device)
	onDeviceLost  func(*models.Device)
}

type HeartbeatPacket struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Type         models.DeviceType `json:"type"`
	IP           string            `json:"ip"`
	Port         int               `json:"port"`
	Capabilities []string          `json:"capabilities"`
	Timestamp    int64             `json:"timestamp"`
}

func NewDiscovery(config *Config) *Discovery {
	return &Discovery{
		config: config,
		peers:  make(map[string]*models.Device),
	}
}

func (d *Discovery) SetCallbacks(onFound func(*models.Device), onLost func(*models.Device)) {
	d.onDeviceFound = onFound
	d.onDeviceLost = onLost
}

func (d *Discovery) Start() error {
	addr := &net.UDPAddr{
		Port: BroadcastPort,
	}

	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return fmt.Errorf("failed to listen UDP: %w", err)
	}

	d.conn = conn
	d.running = true

	ip, err := d.getLocalIP()
	if err != nil {
		ip = "127.0.0.1"
	}

	d.device = models.NewDevice(
		d.config.General.DeviceName,
		models.DeviceWindows,
		ip,
		d.config.Server.TCPPort,
	)

	go d.broadcastLoop()
	go d.listenLoop()
	go d.cleanupLoop()

	return nil
}

func (d *Discovery) Stop() {
	d.running = false
	if d.conn != nil {
		d.conn.Close()
	}
}

func (d *Discovery) GetPeers() []*models.Device {
	d.mu.RLock()
	defer d.mu.RUnlock()

	peers := make([]*models.Device, 0, len(d.peers))
	for _, peer := range d.peers {
		if peer.IsOnline() {
			peers = append(peers, peer)
		}
	}
	return peers
}

func (d *Discovery) GetPeer(id string) (*models.Device, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	peer, ok := d.peers[id]
	if ok && peer.IsOnline() {
		return peer, true
	}
	return nil, false
}

func (d *Discovery) broadcastLoop() {
	for d.running {
		packet := HeartbeatPacket{
			ID:           d.device.ID,
			Name:         d.device.Name,
			Type:         d.device.Type,
			IP:           d.device.IP,
			Port:         d.device.Port,
			Capabilities: d.device.Capabilities,
			Timestamp:    time.Now().Unix(),
		}

		data, err := json.Marshal(packet)
		if err != nil {
			time.Sleep(HeartbeatInterval)
			continue
		}

		addr := &net.UDPAddr{
			IP:   net.ParseIP(BroadcastAddr),
			Port: BroadcastPort,
		}

		d.conn.WriteToUDP(data, addr)
		time.Sleep(HeartbeatInterval)
	}
}

func (d *Discovery) listenLoop() {
	buffer := make([]byte, 4096)

	for d.running {
		n, addr, err := d.conn.ReadFromUDP(buffer)
		if err != nil {
			continue
		}

		var packet HeartbeatPacket
		if err := json.Unmarshal(buffer[:n], &packet); err != nil {
			continue
		}

		if packet.ID == d.device.ID {
			continue
		}

		peer := &models.Device{
			ID:           packet.ID,
			Name:         packet.Name,
			Type:         packet.Type,
			IP:           packet.IP,
			Port:         packet.Port,
			Capabilities: packet.Capabilities,
			LastSeen:     time.Now(),
		}

		d.mu.Lock()
		existing, exists := d.peers[peer.ID]
		d.peers[peer.ID] = peer
		d.mu.Unlock()

		if !exists && d.onDeviceFound != nil {
			go d.onDeviceFound(peer)
		}

		_ = addr
	}
}

func (d *Discovery) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for d.running {
		<-ticker.C

		d.mu.Lock()
		for id, peer := range d.peers {
			if !peer.IsOnline() {
				delete(d.peers, id)
				if d.onDeviceLost != nil {
					go d.onDeviceLost(peer)
				}
			}
		}
		d.mu.Unlock()
	}
}

func (d *Discovery) getLocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no local IP found")
}
