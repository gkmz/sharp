package tools

import (
	"github.com/gkmz/sharp/internal/tools/convert"
	"github.com/gkmz/sharp/internal/tools/encode"
	"github.com/gkmz/sharp/internal/tools/generator"
	"github.com/gkmz/sharp/internal/tools/hash"
	jsontools "github.com/gkmz/sharp/internal/tools/json"
	"github.com/gkmz/sharp/internal/tools/jwt"
	"github.com/gkmz/sharp/internal/tools/network"
	"github.com/gkmz/sharp/internal/tools/text"
	"github.com/gkmz/sharp/internal/tools/timeutil"
	"github.com/gkmz/sharp/pkg/tool"
)

// NewRegistry returns the default registry with every built-in sharp tool.
func NewRegistry() *tool.Registry {
	r := tool.NewRegistry()
	jsontools.Register(r)
	encode.Register(r)
	hash.Register(r)
	timeutil.Register(r)
	generator.Register(r)
	text.Register(r)
	network.Register(r)
	convert.Register(r)
	jwt.Register(r)
	return r
}
