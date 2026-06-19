package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type DeviceType string

const (
	DeviceWindows DeviceType = "windows"
	DeviceAndroid DeviceType = "android"
	DeviceLinux   DeviceType = "linux"
	DeviceMacOS   DeviceType = "macos"
)

type Device struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Type         DeviceType   `json:"type"`
	IP           string       `json:"ip"`
	Port         int          `json:"port"`
	Capabilities []string     `json:"capabilities"`
	LastSeen     time.Time    `json:"last_seen"`
	PublicKey    []byte       `json:"public_key,omitempty"`
	Paired       bool         `json:"paired"`
}

func NewDevice(name string, deviceType DeviceType, ip string, port int) *Device {
	return &Device{
		ID:           generateID(),
		Name:         name,
		Type:         deviceType,
		IP:           ip,
		Port:         port,
		Capabilities: []string{"file", "clipboard", "chat"},
		LastSeen:     time.Now(),
	}
}

func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (d *Device) IsOnline() bool {
	return time.Since(d.LastSeen) < 30*time.Second
}
