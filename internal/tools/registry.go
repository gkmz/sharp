package tools

import (
	"github.com/hank/sharp/internal/tools/convert"
	"github.com/hank/sharp/internal/tools/encode"
	"github.com/hank/sharp/internal/tools/generator"
	"github.com/hank/sharp/internal/tools/hash"
	jsontools "github.com/hank/sharp/internal/tools/json"
	"github.com/hank/sharp/internal/tools/jwt"
	"github.com/hank/sharp/internal/tools/network"
	"github.com/hank/sharp/internal/tools/text"
	"github.com/hank/sharp/internal/tools/timeutil"
	"github.com/hank/sharp/pkg/tool"
)

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
