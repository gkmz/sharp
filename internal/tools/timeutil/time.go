package timeutil

import (
	"context"
	"fmt"
	"github.com/hank/sharp/pkg/tool"
	"strconv"
	"strings"
	"time"
)

func Register(r *tool.Registry) {
	r.MustRegister(tool.SimpleTool{
		IDValue:          "time.now",
		NameValue:        "Current Time",
		CategoryValue:    tool.CategoryTime,
		DescriptionValue: "Show current local and UTC time with Unix timestamps.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			now := time.Now()
			return tool.Output{Text: formatTime(now)}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "time.from",
		NameValue:        "Timestamp to Time",
		CategoryValue:    tool.CategoryTime,
		DescriptionValue: "Convert Unix seconds or milliseconds to datetime.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			s := strings.TrimSpace(input.Text)
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return tool.Output{}, fmt.Errorf("invalid timestamp: %w", err)
			}
			var t time.Time
			switch {
			case len(s) >= 19:
				t = time.Unix(0, n)
			case len(s) >= 13:
				t = time.UnixMilli(n)
			default:
				t = time.Unix(n, 0)
			}
			return tool.Output{Text: formatTime(t)}, nil
		},
	})
	r.MustRegister(tool.SimpleTool{
		IDValue:          "time.parse",
		NameValue:        "Time to Timestamp",
		CategoryValue:    tool.CategoryTime,
		DescriptionValue: "Parse datetime and show Unix timestamps.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			t, err := parseTime(strings.TrimSpace(input.Text))
			if err != nil {
				return tool.Output{}, err
			}
			return tool.Output{Text: formatTime(t)}, nil
		},
	})
}

func formatTime(t time.Time) string {
	return fmt.Sprintf("Local: %s\nUTC:   %s\nUnix seconds: %d\nUnix milliseconds: %d\nUnix nanoseconds: %d",
		t.Local().Format(time.RFC3339Nano),
		t.UTC().Format(time.RFC3339Nano),
		t.Unix(),
		t.UnixMilli(),
		t.UnixNano(),
	)
}

func parseTime(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time format: %s", s)
}
