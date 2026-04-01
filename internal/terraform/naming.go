package terraform

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

var nonAlphanumRegexp = regexp.MustCompile(`[^a-z0-9]+`)
var multiUnderscoreRegexp = regexp.MustCompile(`_+`)

func sanitizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "unnamed"
	}

	// Normalize unicode to decomposed form, then strip non-ASCII
	name = norm.NFD.String(name)
	var b strings.Builder
	for _, r := range name {
		if r < unicode.MaxASCII {
			b.WriteRune(r)
		}
	}
	name = b.String()

	name = strings.ToLower(name)
	name = nonAlphanumRegexp.ReplaceAllString(name, "_")
	name = multiUnderscoreRegexp.ReplaceAllString(name, "_")
	name = strings.Trim(name, "_")

	if name == "" {
		return "unnamed"
	}

	if name[0] >= '0' && name[0] <= '9' {
		name = "resource_" + name
	}

	return name
}

type NameRegistry struct {
	used map[string]map[string]int
}

func NewNameRegistry() *NameRegistry {
	return &NameRegistry{used: make(map[string]map[string]int)}
}

func (r *NameRegistry) Name(resourceType, rawName string) string {
	name := sanitizeName(rawName)

	if _, ok := r.used[resourceType]; !ok {
		r.used[resourceType] = make(map[string]int)
	}

	count := r.used[resourceType][name]
	r.used[resourceType][name] = count + 1

	if count == 0 {
		return name
	}
	return fmt.Sprintf("%s_%d", name, count+1)
}
