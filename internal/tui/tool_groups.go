package tui

import "github.com/gkmz/sharp/pkg/tool"

type toolActionSpec struct {
	id    string
	label string
}

type toolGroupSpec struct {
	category tool.Category
	key      string
	actions  []toolActionSpec
}

// UI groups describe a workflow, independently of CLI command prefixes. Keep
// the action IDs stable so search, recent history and scripts stay compatible.
var consolidatedToolGroups = []toolGroupSpec{
	{tool.CategoryConvert, "convert", []toolActionSpec{
		{"json.to_yaml", "JSON→YAML"},
		{"yaml.to_json", "YAML→JSON"},
		{"csv.to_json", "CSV→JSON"},
		{"headers.to_json", "Headers→JSON"},
	}},
	{tool.CategoryEncode, "b64", []toolActionSpec{
		{"b64.encode", "Standard Encode"},
		{"b64.decode", "Standard Decode"},
		{"b64url.encode", "URL Encode"},
		{"b64url.decode", "URL Decode"},
		{"b64rawurl.encode", "Raw URL Encode"},
		{"b64rawurl.decode", "Raw URL Decode"},
	}},
	{tool.CategoryGenerator, "generator", []toolActionSpec{
		{"password", "Password"},
		{"token", "Token"},
		{"uuid.v4", "UUID v4"},
	}},
}

// The category check keeps equally named commands in unrelated categories from
// sharing a workspace. Unlisted tools retain their existing prefix groups.
func consolidatedGroup(t tool.Tool) (*toolGroupSpec, int) {
	for i := range consolidatedToolGroups {
		group := &consolidatedToolGroups[i]
		if group.category != t.Category() {
			continue
		}
		for index, action := range group.actions {
			if action.id == t.ID() {
				return group, index
			}
		}
	}
	return nil, -1
}
