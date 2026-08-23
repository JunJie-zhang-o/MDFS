package access

import (
	"path"
	"strings"
)

type Permissions struct {
	Read   bool `json:"read"`
	Write  bool `json:"write"`
	Delete bool `json:"delete"`
}

func Parse(value string) Permissions {
	value = strings.ToUpper(value)
	return Permissions{
		Read:   strings.Contains(value, "R"),
		Write:  strings.Contains(value, "W"),
		Delete: strings.Contains(value, "D"),
	}
}

type Rule struct {
	Pattern     string
	Permissions Permissions
}

type Policy struct {
	Default Permissions
	Rules   []Rule
}

func (p Policy) CanDiscover(requestPath string) bool {
	if p.For(requestPath).Read {
		return true
	}
	prefix := strings.TrimSuffix(normalize(requestPath), "/") + "/"
	for _, rule := range p.Rules {
		if !rule.Permissions.Read {
			continue
		}
		literal := normalize(rule.Pattern)
		if wildcard := strings.IndexAny(literal, "*?"); wildcard >= 0 {
			literal = literal[:wildcard]
		}
		if strings.HasPrefix(literal, prefix) {
			return true
		}
	}
	return false
}

func (p Policy) For(requestPath string) Permissions {
	requestPath = normalize(requestPath)
	best := -1
	permissions := p.Default
	for _, rule := range p.Rules {
		pattern := normalize(rule.Pattern)
		if match(pattern, requestPath) && specificity(pattern) > best {
			best = specificity(pattern)
			permissions = rule.Permissions
		}
	}
	return permissions
}

func normalize(value string) string {
	return path.Clean("/" + strings.TrimPrefix(value, "/"))
}

func specificity(pattern string) int {
	return len(strings.ReplaceAll(strings.ReplaceAll(pattern, "*", ""), "?", ""))
}

func match(pattern, value string) bool {
	if pattern == "/" {
		return value == "/"
	}
	p := strings.Split(strings.Trim(pattern, "/"), "/")
	v := strings.Split(strings.Trim(value, "/"), "/")
	return matchSegments(p, v)
}

func matchSegments(pattern, value []string) bool {
	if len(pattern) == 0 {
		return len(value) == 0
	}
	if pattern[0] == "**" {
		if matchSegments(pattern[1:], value) {
			return true
		}
		return len(value) > 0 && matchSegments(pattern, value[1:])
	}
	if len(value) == 0 {
		return false
	}
	matched, err := path.Match(pattern[0], value[0])
	return err == nil && matched && matchSegments(pattern[1:], value[1:])
}
