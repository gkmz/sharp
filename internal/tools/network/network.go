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

func Register(r *tool.Registry) {
	r.MustRegister(tool.SimpleTool{
		IDValue:          "url.parse",
		NameValue:        "Parse URL",
		CategoryValue:    tool.CategoryNetwork,
		DescriptionValue: "Parse a URL into structured parts.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			u, err := url.Parse(strings.TrimSpace(input.Text))
			if err != nil {
				return tool.Output{}, err
			}
			data := map[string]any{
				"scheme":   u.Scheme,
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
			values, err := url.ParseQuery(strings.TrimPrefix(strings.TrimSpace(input.Text), "?"))
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
			ip, network, err := net.ParseCIDR(strings.TrimSpace(input.Text))
			if err != nil {
				return tool.Output{}, err
			}
			ones, bits := network.Mask.Size()
			return tool.Output{Text: fmt.Sprintf("IP: %s\nNetwork: %s\nPrefix: %d\nBits: %d", ip, network, ones, bits)}, nil
		},
	})
}
