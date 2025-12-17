package match

import "context"

var NoMatch = noMatch{}

type Type int

func (t Type) String() string {
	switch t {
	case GitIgnore:
		return "gitignore"
	case Glob:
		return "glob"
	case Regex:
		return "regex"
	default:
		return "unknown"
	}
}

const (
	Unknown Type = iota
	GitIgnore
	Glob
	Regex
)

type MatchInfo interface {
	Ok() bool
	Src() string
	Type() Type
	String() string
}

type PathMatcher interface {
	Type() Type
	Match(ctx context.Context, path string) (bool, error)
	Match2(ctx context.Context, path string) (MatchInfo, error)
}

type noMatch struct{}

func (noMatch) Ok() bool {
	return false
}

func (noMatch) Src() string {
	return ""
}

func (noMatch) Type() Type {
	return Unknown
}

func (noMatch) String() string {
	return ""
}
