package psiphon

import (
	"encoding/json"
	"runtime"
	"strings"
)

// ShiroKhorshid public signature keys (matching MehrON and ShiroKhorshid official core)
const (
	ShiroRemoteServerListSignaturePublicKey = "MIICIDANBgkqhkiG9w0BAQEFAAOCAg0AMIICCAKCAgEAt7Ls+/39r+T6zNW7GiVpJfzq/xvL9SBH5rIFnk0RXYEYavax3WS6HOD35eTAqn8AniOwiH+DOkvgSKF2caqk/y1dfq47Pdymtwzp9ikpB1C5OfAysXzBiwVJlCdajBKvBZDerV1cMvRzCKvKwRmvDmHgphQQ7WfXIGbRbmmk6opMBh3roE42KcotLFtqp0RRwLtcBRNtCdsrVsjiI1Lqz/lH+T61sGjSjQ3CHMuZYSQJZo/KrvzgQXpkaCTdbObxHqb6/+i1qaVOfEsvjoiyzTxJADvSytVtcTjijhPEV6XskJVHE1Zgl+7rATr/pDQkw6DPCNBS1+Y6fy7GstZALQXwEDN/qhQI9kWkHijT8ns+i1vGg00Mk/6J75arLhqcodWsdeG/M/moWgqQAnlZAGVtJI1OgeF5fsPpXu4kctOfuZlGjVZXQNW34aOzm8r8S0eVZitPlbhcPiR4gT/aSMz/wd8lZlzZYsje/Jr8u/YtlwjjreZrGRmG8KMOzukV3lLmMppXFMvl4bxv6YFEmIuTsOhbLTwFgh7KYNjodLj/LsqRVfwz31PgWQFTEPICV7GCvgVlPRxnofqKSjgTWI4mxDhBpVcATvaoBl1L/6WLbFvBsoAUBItWwctO2xalKxF5szhGm8lccoc5MZr8kfE0uxMgsxz4er68iCID+rsCAQM="
	ShiroServerEntrySignaturePublicKey      = "sHuUVTWaRyh5pZwy4UguSgkwmBe0EHtJJkoF5WrxmvA="
	ShiroPropagationChannelID               = "2C595532AC17D0E6"
	ShiroSponsorID                          = "1BC527D3D09985CF"
	ShiroClientVersion                      = "158"
)

// ShiroConfigOptions configures the ShiroKhorshid Psiphon core
type ShiroConfigOptions struct {
	SocksPort        int      `json:"socks_port"`
	HTTPPort         int      `json:"http_port"`
	DataDir          string   `json:"data_dir"`
	EgressRegion     string   `json:"egress_region"` // e.g. "DE", "US", "GB" or "" for auto
	TunnelPoolSize   int      `json:"tunnel_pool_size"`
	CdnFronting      bool     `json:"cdn_fronting"`
	CdnEdges         []string `json:"cdn_edges"` // Clean edge IPs, CIDRs, or hostnames
	UpstreamProxyURL string   `json:"upstream_proxy_url,omitempty"`
	SNI              string   `json:"sni,omitempty"`
}

// BuildShiroKhorshidConfig generates the complete Psiphon JSON configuration
// compatible with the ShiroKhorshid Psiphon fork and MehrON client.
func BuildShiroKhorshidConfig(opts ShiroConfigOptions) ([]byte, error) {
	socksPort := opts.SocksPort
	if socksPort <= 0 {
		socksPort = 10808
	}
	httpPort := opts.HTTPPort
	if httpPort <= 0 {
		httpPort = socksPort + 1
	}

	poolSize := opts.TunnelPoolSize
	if poolSize <= 0 {
		poolSize = 1
	}

	clientPlatform := "Linux"
	if runtime.GOOS == "windows" {
		clientPlatform = "Windows_10.0.26200_11"
	}

	reg := strings.ToUpper(strings.TrimSpace(opts.EgressRegion))
	if reg == "AUTO" || reg == "ANY" || reg == "NONE" || reg == "DEFAULT" {
		reg = ""
	}

	sni := strings.TrimSpace(opts.SNI)
	if sni == "" {
		sni = "a248.e.akamai.net"
	}

	remoteURLs := []map[string]any{
		{"OnlyAfterAttempts": 0, "SkipVerify": false, "URL": "aHR0cHM6Ly9zMy5hbWF6b25hd3MuY29tL3BzaXBob24vd2ViL2p2bmstMmN5ZC1tcDRoL3NlcnZlcl9saXN0X2NvbXByZXNzZWQ="},
		{"OnlyAfterAttempts": 2, "SkipVerify": true, "URL": "aHR0cHM6Ly93d3cud2hlZWxyc3NrbWluc2lkZS5jb20vd2ViL2p2bmstMmN5ZC1tcDRoL3NlcnZlcl9saXN0X2NvbXByZXNzZWQ="},
		{"OnlyAfterAttempts": 2, "SkipVerify": true, "URL": "aHR0cHM6Ly93d3cuYnJhbmRpbmd1c2FnYW1lcmVwLmNvbS93ZWIvanZuay0yY3lkLW1wNGgvc2VydmVyX2xpc3RfY29tcHJlc3NlZA=="},
		{"OnlyAfterAttempts": 2, "SkipVerify": true, "URL": "aHR0cHM6Ly93d3cuZ3BhbGx0aGluZ3NudW1iZXJ3ZWF0aGVyLmNvbS93ZWIvanZuay0yY3lkLW1wNGgvc2VydmVyX2xpc3RfY29tcHJlc3NlZA=="},
	}

	obfuscatedURLs := []map[string]any{
		{"OnlyAfterAttempts": 0, "SkipVerify": false, "URL": "aHR0cHM6Ly9zMy5hbWF6b25hd3MuY29tL3BzaXBob24vd2ViL2p2bmstMmN5ZC1tcDRoL29zbA=="},
		{"OnlyAfterAttempts": 2, "SkipVerify": true, "URL": "aHR0cHM6Ly93d3cud2hlZWxyc3NrbWluc2lkZS5jb20vd2ViL2p2bmstMmN5ZC1tcDRoL29zbA=="},
		{"OnlyAfterAttempts": 2, "SkipVerify": true, "URL": "aHR0cHM6Ly93d3cuYnJhbmRpbmd1c2FnYW1lcmVwLmNvbS93ZWIvanZuay0yY3lkLW1wNGgvb3Ns"},
		{"OnlyAfterAttempts": 2, "SkipVerify": true, "URL": "aHR0cHM6Ly93d3cuZ3BhbGx0aGluZ3NudW1iZXJ3ZWF0aGVyLmNvbS93ZWIvanZuay0yY3lkLW1wNGgvb3Ns"},
	}

	cfg := map[string]any{
		"LocalSocksProxyPort":                 socksPort,
		"LocalHttpProxyPort":                  httpPort,
		"DataRootDirectory":                   opts.DataDir,
		"EgressRegion":                        reg,
		"TunnelPoolSize":                      poolSize,
		"EmitDiagnosticNotices":               true,
		"EmitServerAlerts":                    true,
		"PropagationChannelId":                ShiroPropagationChannelID,
		"SponsorId":                           ShiroSponsorID,
		"ClientPlatform":                      clientPlatform,
		"ClientVersion":                       ShiroClientVersion,
		"UseIndistinguishableTLS":             true,
		"DisableLocalSocksProxy":              false,
		"DisableLocalHTTPProxy":               false,
		"RemoteServerListSignaturePublicKey": ShiroRemoteServerListSignaturePublicKey,
		"ServerEntrySignaturePublicKey":      ShiroServerEntrySignaturePublicKey,
		"RemoteServerListURLs":                remoteURLs,
		"ObfuscatedServerListRootURLs":        obfuscatedURLs,
	}

	if opts.UpstreamProxyURL != "" {
		cfg["UpstreamProxyURL"] = opts.UpstreamProxyURL
		cfg["UpstreamProxyUrl"] = opts.UpstreamProxyURL
	}

	// Clean CDN Edge addresses injection (ShiroKhorshid fork feature)
	if opts.CdnFronting && len(opts.CdnEdges) > 0 {
		var specs []map[string]string
		for _, edge := range opts.CdnEdges {
			trimmed := strings.TrimSpace(edge)
			if trimmed != "" {
				specs = append(specs, map[string]string{"Address": trimmed})
			}
		}
		if len(specs) > 0 {
			cfg["CdnFrontingSpecs"] = specs
		}

		// Prune inproxy WebRTC protocols because WebRTC STUN/ICE is filtered in Iran,
		// and prioritize Fronted Meek protocols for fast CDN connection
		cfg["PruneInproxyTunnelProtocols"] = true
		cfg["LimitTunnelProtocols"] = []string{
			"FRONTED-MEEK-OSSH",
			"FRONTED-MEEK-HTTP-OSSH",
			"FRONTED-MEEK-QUIC-OSSH",
		}
	} else {
		// Even in direct mode, prune inproxy WebRTC protocols so Psiphon directly uses working protocols
		cfg["PruneInproxyTunnelProtocols"] = true
	}

	return json.MarshalIndent(cfg, "", "  ")
}
