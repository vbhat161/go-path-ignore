package match

import "context"

type Type int

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
}

type PathMatcher interface {
	Type() Type
	Match(ctx context.Context, path string) (bool, error)
	Match2(ctx context.Context, path string) (MatchInfo, error)
}

type dummyMatch struct{}

func (dummyMatch) Ok() bool {
	return false
}
func (dummyMatch) Src() string {
	return ""
}
func (dummyMatch) Type() Type {
	return Unknown
}

var NoMatch = dummyMatch{}
