package network

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"sort"
	"strings"

	"github.com/gkmz/sharp/pkg/tool"
)

// Register adds URL, query string, DNS, and CIDR tools to r.
func Register(r *tool.Registry) {
	r.MustRegister(tool.SimpleTool{
		IDValue:          "url.parse",
		NameValue:        "Parse URL",
		CategoryValue:    tool.CategoryNetwork,
		DescriptionValue: "Parse a URL into structured parts.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			raw := strings.TrimSpace(input.Text)
			if raw == "" {
				return tool.Output{}, fmt.Errorf("URL is required")
			}
			u, err := url.Parse(normalizeURLInput(raw))
			if err != nil {
				return tool.Output{}, err
			}
			if u.Scheme == "" && u.Host == "" {
				return tool.Output{}, fmt.Errorf("URL must include a scheme or host")
			}
			if urlSchemeRequiresHost(u.Scheme) && u.Host == "" {
				return tool.Output{}, fmt.Errorf("URL scheme %q requires a host", u.Scheme)
			}
			data := map[string]any{
				"scheme":   u.Scheme,
				"opaque":   u.Opaque,
				"host":     u.Host,
				"path":     u.Path,
				"rawQuery": u.RawQuery,
				"fragment": u.Fragment,
				"query":    u.Query(),
			}
			b, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: string(b), MimeType: "application/json"}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "query.parse",
		NameValue:        "Parse Query String",
		CategoryValue:    tool.CategoryNetwork,
		DescriptionValue: "Parse URL query string as JSON.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			raw := strings.TrimPrefix(strings.TrimSpace(input.Text), "?")
			if raw == "" {
				return tool.Output{}, fmt.Errorf("query string is required")
			}
			values, err := url.ParseQuery(raw)
			if err != nil {
				return tool.Output{}, err
			}
			b, err := json.MarshalIndent(values, "", "  ")
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: string(b), MimeType: "application/json"}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "dns.lookup",
		NameValue:        "DNS Lookup",
		CategoryValue:    tool.CategoryNetwork,
		DescriptionValue: "Resolve host IP addresses.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			host := strings.TrimSpace(input.Text)
			if host == "" {
				return tool.Output{}, fmt.Errorf("host is required")
			}
			ips, err := net.LookupHost(host)
			if err != nil {
				return tool.Output{}, err
			}
			sort.Strings(ips)
			return tool.Output{Text: strings.Join(ips, "\n")}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "cidr.parse",
		NameValue:        "CIDR Parse",
		CategoryValue:    tool.CategoryNetwork,
		DescriptionValue: "Parse CIDR notation.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			raw := strings.TrimSpace(input.Text)
			if raw == "" {
				return tool.Output{}, fmt.Errorf("CIDR is required")
			}
			ip, network, err := net.ParseCIDR(raw)
			if err != nil {
				return tool.Output{}, err
			}
			ones, bits := network.Mask.Size()
			return tool.Output{Text: fmt.Sprintf("IP: %s\nNetwork: %s\nPrefix: %d\nBits: %d", ip, network, ones, bits)}, nil
		},
	})
}

func normalizeURLInput(raw string) string {
	if strings.Contains(raw, "://") || strings.HasPrefix(raw, "//") {
		return raw
	}
	first := raw
	for _, sep := range []string{"/", "?", "#"} {
		if before, _, ok := strings.Cut(first, sep); ok {
			first = before
		}
	}
	if strings.HasPrefix(first, "[") {
		// 无 scheme 的 IPv6 host 会以 bracket 形式出现；它不是 opaque scheme，应按 host 解析。
		return "//" + raw
	}
	if prefix, _, ok := strings.Cut(first, ":"); ok && prefix != "localhost" && !strings.Contains(prefix, ".") {
		return raw
	}
	if first == "localhost" || strings.Contains(first, ".") || strings.Contains(first, ":") {
		return "//" + raw
	}
	return raw
}

func urlSchemeRequiresHost(scheme string) bool {
	switch strings.ToLower(scheme) {
	case "http", "https", "ws", "wss", "ftp":
		return true
	default:
		return false
	}
}
