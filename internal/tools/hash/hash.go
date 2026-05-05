package hash

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"hash"
	"hash/crc32"

	"github.com/gkmz/sharp/pkg/tool"
)

// Register adds hash and HMAC tools to r.
func Register(r *tool.Registry) {
	digest(r, "hash.md5", "MD5", md5.New)
	digest(r, "hash.sha1", "SHA1", sha1.New)
	digest(r, "hash.sha224", "SHA224", sha256.New224)
	digest(r, "hash.sha256", "SHA256", sha256.New)
	digest(r, "hash.sha384", "SHA384", sha512.New384)
	digest(r, "hash.sha512", "SHA512", sha512.New)
	r.MustRegister(tool.SimpleTool{
		IDValue:          "hash.crc32",
		NameValue:        "CRC32",
		CategoryValue:    tool.CategoryCrypto,
		DescriptionValue: "Calculate CRC32 checksum.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			sum := crc32.ChecksumIEEE([]byte(input.Text))
			return tool.Output{Text: hex.EncodeToString([]byte{byte(sum >> 24), byte(sum >> 16), byte(sum >> 8), byte(sum)})}, nil
		},
	})
	hmacTool(r, "hmac.sha256", "HMAC SHA256", sha256.New)
	hmacTool(r, "hmac.sha512", "HMAC SHA512", sha512.New)
}

func digest(r *tool.Registry, id string, name string, newHash func() hash.Hash) {
	r.MustRegister(tool.SimpleTool{
		IDValue:          id,
		NameValue:        name,
		CategoryValue:    tool.CategoryCrypto,
		DescriptionValue: "Calculate " + name + " hash.",
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			h := newHash()
			_, _ = h.Write([]byte(input.Text))
			return tool.Output{Text: hex.EncodeToString(h.Sum(nil))}, nil
		},
	})
}

func hmacTool(r *tool.Registry, id string, name string, newHash func() hash.Hash) {
	r.MustRegister(tool.SimpleTool{
		IDValue:          id,
		NameValue:        name,
		CategoryValue:    tool.CategoryCrypto,
		DescriptionValue: "Calculate " + name + ".",
		OptionsValue: []tool.Option{{
			Name:        "key",
			Description: "HMAC secret key",
			Required:    true,
			Secret:      true,
		}},
		RunFunc: func(ctx context.Context, input tool.Input, options tool.Options) (tool.Output, error) {
			mac := hmac.New(newHash, []byte(options["key"]))
			_, _ = mac.Write([]byte(input.Text))
			return tool.Output{Text: hex.EncodeToString(mac.Sum(nil))}, nil
		},
	})
}
