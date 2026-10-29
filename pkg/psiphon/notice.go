package psiphon

import (
	"encoding/json"
	"strings"
)

// Notice represents a parsed Psiphon core notice
type Notice struct {
	Timestamp  string         `json:"timestamp,omitempty"`
	NoticeType string         `json:"noticeType"`
	Data       map[string]any `json:"data,omitempty"`
}

// ParseNotice parses either a formatted notice ("<ISO-timestamp> <NoticeType> <JSON>")
// or a raw JSON notice string ("{"noticeType":"...", "data":{...}}").
func ParseNotice(raw string) *Notice {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil
	}

	// 1. Raw JSON notice: {"noticeType": "...", "data": {...}}
	if strings.HasPrefix(trimmed, "{") && strings.HasSuffix(trimmed, "}") {
		var rawMap map[string]any
		if err := json.Unmarshal([]byte(trimmed), &rawMap); err == nil {
			if nt, ok := rawMap["noticeType"].(string); ok && nt != "" {
				n := &Notice{
					NoticeType: nt,
				}
				if d, ok := rawMap["data"].(map[string]any); ok {
					n.Data = d
				}
				if ts, ok := rawMap["timestamp"].(string); ok {
					n.Timestamp = ts
				}
				return n
			}
		}
	}

	// 2. Formatted notice: "2026-09-24T12:50:15.277Z Tunnels {"count":1}"
	firstSpace := strings.IndexByte(trimmed, ' ')
	if firstSpace > 0 {
		secondSpace := strings.IndexByte(trimmed[firstSpace+1:], ' ')
		if secondSpace > 0 {
			actualSecondSpace := firstSpace + 1 + secondSpace
			ts := trimmed[:firstSpace]
			noticeType := trimmed[firstSpace+1 : actualSecondSpace]
			jsonPart := strings.TrimSpace(trimmed[actualSecondSpace+1:])

			n := &Notice{
				Timestamp:  ts,
				NoticeType: noticeType,
			}

			if strings.HasPrefix(jsonPart, "{") && strings.HasSuffix(jsonPart, "}") {
				var data map[string]any
				if err := json.Unmarshal([]byte(jsonPart), &data); err == nil {
					n.Data = data
				}
			}
			return n
		}
	}

	return nil
}

// FormatNoticeLog formats a notice into a human-readable log string matching MehrON
func FormatNoticeLog(n *Notice) string {
	if n == nil || n.NoticeType == "" {
		return ""
	}

	switch n.NoticeType {
	case "ListeningSocksProxyPort":
		port := "1080"
		if n.Data != nil {
			if p, ok := n.Data["port"]; ok {
				port = strings.TrimSpace(string(p.(json.Number)))
				if port == "" {
					port = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(string(mustMarshal(p)), "\"", ""), " ", ""))
				}
			}
		}
		return "[Psiphon Shirokhorshid] SOCKS proxy listening on 127.0.0.1:" + port

	case "ListeningHttpProxyPort":
		port := "1081"
		if n.Data != nil {
			if p, ok := n.Data["port"]; ok {
				port = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(string(mustMarshal(p)), "\"", ""), " ", ""))
			}
		}
		return "[Psiphon Shirokhorshid] HTTP proxy listening on 127.0.0.1:" + port

	case "CandidateServers":
		count := "0"
		if n.Data != nil {
			if c, ok := n.Data["count"]; ok {
				count = strings.TrimSpace(strings.ReplaceAll(string(mustMarshal(c)), "\"", ""))
			}
		}
		return "[Psiphon Shirokhorshid] Found " + count + " candidate servers. Connecting..."

	case "ConnectingServer":
		detail := ""
		if n.Data != nil {
			reg, _ := n.Data["region"].(string)
			if reg == "" {
				reg, _ = n.Data["egressRegion"].(string)
			}
			proto, _ := n.Data["protocol"].(string)
			if reg != "" || proto != "" {
				detail = " (" + reg + " / " + proto + ")"
			}
		}
		return "[Psiphon Shirokhorshid] Connecting to server" + detail + "..."

	case "ConnectedServer", "ConnectedServerRegion":
		region := ""
		if n.Data != nil {
			if r, ok := n.Data["serverRegion"].(string); ok && r != "" {
				region = " in " + r
			} else if r, ok := n.Data["egressRegion"].(string); ok && r != "" {
				region = " in " + r
			} else if r, ok := n.Data["region"].(string); ok && r != "" {
				region = " in " + r
			}
		}
		return "[Psiphon Shirokhorshid] Connected to server" + region + ". Establishing tunnel..."

	case "ActiveTunnel":
		proto := ""
		if n.Data != nil {
			if p, ok := n.Data["protocol"].(string); ok && p != "" {
				proto = " (" + p + ")"
			}
		}
		return "[Psiphon Shirokhorshid] Active tunnel established" + proto + "."

	case "Tunnels":
		count := 0
		if n.Data != nil {
			if c, ok := n.Data["count"].(float64); ok {
				count = int(c)
			}
		}
		if count > 0 {
			return "[Psiphon Shirokhorshid] Connected! Active tunnels: " + string(mustMarshal(count))
		}
		return "[Psiphon Shirokhorshid] Disconnected (0 active tunnels). Reconnecting..."

	case "Alert":
		if n.Data != nil {
			if msg, ok := n.Data["message"].(string); ok && msg != "" {
				return "[Psiphon Shirokhorshid Alert] " + msg
			}
		}
	}

	return ""
}

func mustMarshal(v any) []byte {
	b, _ := json.Marshal(v)
	return b
}
