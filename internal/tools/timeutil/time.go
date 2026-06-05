package timeutil

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/gkmz/sharp/pkg/tool"
)

// Register adds time formatting and timestamp conversion tools to r.
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
		DescriptionValue: "Convert Unix seconds, milliseconds, or nanoseconds to datetime.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			s := strings.TrimSpace(input.Text)
			n, err := strconv.ParseInt(s, 10, 64)
			if err != nil {
				return tool.Output{}, fmt.Errorf("invalid timestamp: %w", err)
			}
			var t time.Time
			digits := timestampDigitCount(s)
			switch {
			case digits >= 19:
				t = time.Unix(0, n)
			case digits >= 13:
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

func timestampDigitCount(s string) int {
	// 时间戳单位只由有效数字位数决定，正负号和前导零不应让同量级数值走到不同单位分支。
	digits := strings.TrimLeft(strings.TrimLeft(s, "+-"), "0")
	if digits == "" {
		return 1
	}
	return len(digits)
}

func formatTime(t time.Time) string {
	return fmt.Sprintf("Local: %s\nUTC:   %s\nUnix seconds: %d\nUnix milliseconds: %s\nUnix nanoseconds: %s",
		t.Local().Format(time.RFC3339Nano),
		t.UTC().Format(time.RFC3339Nano),
		t.Unix(),
		unixMillisecondsString(t),
		unixNanosecondsString(t),
	)
}

func unixMillisecondsString(t time.Time) string {
	seconds := big.NewInt(t.Unix())
	milliseconds := new(big.Int).Mul(seconds, big.NewInt(1_000))
	// 使用秒内纳秒余量计算毫秒，避免 time.UnixMilli 在极端年份上出现未定义结果。
	return milliseconds.Add(milliseconds, big.NewInt(int64(t.Nanosecond()/1_000_000))).String()
}

func unixNanosecondsString(t time.Time) string {
	seconds := big.NewInt(t.Unix())
	nanoseconds := new(big.Int).Mul(seconds, big.NewInt(1_000_000_000))
	// 使用大整数保留完整 Unix 纳秒值，避免 time.UnixNano 的 int64 溢出边界。
	return nanoseconds.Add(nanoseconds, big.NewInt(int64(t.Nanosecond()))).String()
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
