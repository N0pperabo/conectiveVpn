package mhr

import (
	"crypto/tls"
	"fmt"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

// CandidateGoogleIPs is the standard list of candidate Google edge IPs for fronting
var CandidateGoogleIPs = []string{
	"216.239.38.120",
	"216.239.32.120",
	"216.239.34.120",
	"216.239.36.120",
	"142.250.80.142",
	"142.250.80.138",
	"142.250.179.110",
	"142.250.185.110",
	"142.250.184.206",
	"142.250.190.238",
	"142.250.191.78",
	"172.217.1.206",
	"172.217.14.206",
	"172.217.16.142",
	"172.217.22.174",
	"172.217.164.110",
	"172.217.168.206",
	"172.217.169.206",
	"34.107.221.82",
	"142.251.32.110",
	"142.251.33.110",
	"142.251.46.206",
	"142.251.46.238",
	"142.250.80.170",
	"142.250.72.206",
	"142.250.64.206",
	"142.250.72.110",
}

// ScanGoogleIPs probes candidate Google IPs using TLS SNI fronting
func (m *Manager) ScanGoogleIPs(frontDomain string, timeout time.Duration) ([]GoogleIPResult, error) {
	if frontDomain == "" {
		frontDomain = DefaultFrontDomain
	}
	if timeout <= 0 {
		timeout = 2500 * time.Millisecond
	}

	results := make([]GoogleIPResult, len(CandidateGoogleIPs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 10) // 10 concurrent probes

	for i, ip := range CandidateGoogleIPs {
		wg.Add(1)
		go func(idx int, targetIP string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			res := probeGoogleIP(targetIP, frontDomain, timeout)
			results[idx] = res
		}(i, ip)
	}

	wg.Wait()

	// Sort results: reachable first, then ascending latency
	sort.Slice(results, func(i, j int) bool {
		if results[i].Reachable && !results[j].Reachable {
			return true
		}
		if !results[i].Reachable && results[j].Reachable {
			return false
		}
		if results[i].Reachable && results[j].Reachable {
			return results[i].LatencyMs < results[j].LatencyMs
		}
		return results[i].IP < results[j].IP
	})

	return results, nil
}

func probeGoogleIP(ip, frontDomain string, timeout time.Duration) GoogleIPResult {
	start := time.Now()
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	tlsConfig := &tls.Config{
		ServerName:         frontDomain,
		InsecureSkipVerify: true,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", fmt.Sprintf("%s:443", ip), tlsConfig)
	if err != nil {
		return GoogleIPResult{
			IP:        ip,
			Reachable: false,
			Error:     err.Error(),
		}
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	req := fmt.Sprintf("HEAD / HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", frontDomain)
	if _, err := conn.Write([]byte(req)); err != nil {
		return GoogleIPResult{
			IP:        ip,
			Reachable: false,
			Error:     err.Error(),
		}
	}

	buf := make([]byte, 128)
	n, err := conn.Read(buf)
	if err != nil && n == 0 {
		return GoogleIPResult{
			IP:        ip,
			Reachable: false,
			Error:     err.Error(),
		}
	}

	respStr := string(buf[:n])
	if !strings.HasPrefix(respStr, "HTTP/") {
		return GoogleIPResult{
			IP:        ip,
			Reachable: false,
			Error:     "non-HTTP response",
		}
	}

	latency := int(time.Since(start).Milliseconds())
	return GoogleIPResult{
		IP:        ip,
		LatencyMs: latency,
		Reachable: true,
	}
}
