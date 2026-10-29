package snispoof

import (
	"strings"
	"time"
)

// Engine constants
const (
	EngineAuto   = "auto"
	EngineBinary = "binary"
	EngineRust   = "rust"
	EnginePython = "python"
)

// Default settings
const (
	DefaultListenHost  = "127.0.0.1"
	DefaultListenPort  = 40443
	DefaultConnectPort = 443
	DefaultFakeSNI     = "auth.vercel.com"
)

// Config represents the user-configurable settings for SNI Spoofing
type Config struct {
	Enabled     bool   `json:"enabled"`
	Engine      string `json:"engine"`       // "auto", "binary", "rust", "python"
	ListenHost  string `json:"listen_host"`  // Default: "127.0.0.1"
	ListenPort  int    `json:"listen_port"`  // Default: 40443
	ConnectIP   string `json:"connect_ip"`   // Optional static override; empty = resolve from active node
	ConnectPort int    `json:"connect_port"` // Default: 443 (or node port)
	FakeSNI     string `json:"fake_sni"`     // Default: "auth.vercel.com"
}

// Normalize validates and fills in defaults
func (c *Config) Normalize() {
	if c.ListenHost == "" {
		c.ListenHost = DefaultListenHost
	}
	if c.ListenPort <= 0 || c.ListenPort > 65535 {
		c.ListenPort = DefaultListenPort
	}
	if c.ConnectPort <= 0 || c.ConnectPort > 65535 {
		c.ConnectPort = DefaultConnectPort
	}
	c.FakeSNI = strings.TrimSpace(c.FakeSNI)
	if c.FakeSNI == "" {
		c.FakeSNI = DefaultFakeSNI
	}
	c.Engine = strings.ToLower(strings.TrimSpace(c.Engine))
	if c.Engine == "" {
		c.Engine = EngineAuto
	}
}

// Status represents the runtime state of the SNI Spoofing subsystem
type Status struct {
	Enabled        bool       `json:"enabled"`
	Running        bool       `json:"running"`
	ActiveEngine   string     `json:"active_engine"` // "Rust", "Binary", "Python", "None"
	ListenHost     string     `json:"listen_host"`
	ListenPort     int        `json:"listen_port"`
	TargetIP       string     `json:"target_ip"`
	TargetPort     int        `json:"target_port"`
	FakeSNI        string     `json:"fake_sni"`
	ActiveNodeID   int64      `json:"active_node_id,omitempty"`
	ActiveNodeName string     `json:"active_node_name,omitempty"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	Error          string     `json:"error,omitempty"`
}
