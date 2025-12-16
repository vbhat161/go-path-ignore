package regex

import (
	"context"
	"fmt"
	"strings"

	"github.com/VishwaBhat/go-path-ignore/match"
	regexp "github.com/wasilibs/go-re2"
)

type Matcher struct {
	regexps []*regexp.Regexp
	set     *match.RE2Set
}

type Options struct {
	Patterns []string
	Parallel bool
	Literals bool
}

func NewMatcher(opts Options) (*Matcher, error) {
	if len(opts.Patterns) == 0 {
		return nil, fmt.Errorf("atleast one pattern required for regex matcher")
	}

	var literalRegex string
	if opts.Literals {
		quoted := quotePatterns(opts.Patterns)
		literalRegex = strings.Join(quoted, "|")
		opts.Patterns = []string{literalRegex}
	}

	if opts.Parallel {
		set, e := match.NewRE2Set(opts.Patterns)
		if e != nil {
			return nil, fmt.Errorf("patterns compilation - %w", e)
		}
		return &Matcher{set: set}, nil
	}

	regexps := make([]*regexp.Regexp, 0, len(opts.Patterns))
	for _, p := range opts.Patterns {
		if re, e := regexp.Compile(p); e != nil {
			return nil, fmt.Errorf("pattern(%s) compilation - %w", p, e)
		} else {
			regexps = append(regexps, re)
		}
	}
	return &Matcher{regexps: regexps}, nil
}

func (m *Matcher) Type() match.Type {
	return match.Regex
}

// Matches takes a path and returns whether it is ignored according to the list of
// ignore patterns. It returns true if the path should be ignored, and false otherwise.
func (m *Matcher) Match(ctx context.Context, path string) (bool, error) {
	res, err := m.Match2(ctx, path)
	return res.Ok(), err
}

type result struct {
	src string
}

func (r result) Ok() bool {
	return r.src != ""
}

func (r result) Src() string {
	return r.src
}

func (r result) Type() match.Type {
	return match.Regex
}

// Matches takes a path and returns whether it is ignored according to the list of
// ignore patterns. It returns true if the path should be ignored, and false otherwise.
func (m *Matcher) Match2(ctx context.Context, path string) (match.MatchInfo, error) {
	res := result{}
	if ctx.Err() != nil {
		return res, ctx.Err()
	}

	if m.set != nil {
		ok, path := m.set.Matches(path)
		if ok {
			res.src = path
		}
		return res, nil
	}

	for _, re := range m.regexps {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		if re.MatchString(path) {
			res.src = path
			return res, nil
		}
	}
	return res, nil
}

func quotePatterns(patterns []string) []string {
	quoted := make([]string, 0, len(patterns))
	for _, p := range patterns {
		quoted = append(quoted, regexp.QuoteMeta(p))
	}

	return quoted
}
