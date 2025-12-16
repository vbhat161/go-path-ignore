package match

import (
	"fmt"

	re2exp "github.com/wasilibs/go-re2/experimental"
)

type RE2Set struct {
	src []string
	set *re2exp.Set
}

func NewRE2Set(patterns []string) (*RE2Set, error) {
	if len(patterns) == 0 {
		return nil, fmt.Errorf("empty input patterns")
	}

	set, err := re2exp.CompileSet(patterns)
	if err != nil {
		return nil, err
	}

	return &RE2Set{src: patterns, set: set}, nil
}

func (s *RE2Set) Matches(path string) (bool, string) {
	res := s.set.FindAllString(path, 1)
	if len(res) == 0 {
		return false, ""
	}
	return true, s.src[res[0]]
}
