package gopathignore

import (
	"context"
	"fmt"
	"time"

	"github.com/VishwaBhat/go-path-ignore/match"
	"github.com/VishwaBhat/go-path-ignore/match/gitignore"
	"github.com/VishwaBhat/go-path-ignore/match/glob"
	"github.com/VishwaBhat/go-path-ignore/match/regex"
)

type PathIgnore struct {
	matchers []match.PathMatcher
	timeout  time.Duration
}

type Options struct {
	Regex     *regex.Options
	Glob      *glob.Options
	GitIgnore *gitignore.Options
	Timeout   time.Duration
}

func New(opts Options) (*PathIgnore, error) {
	matchers := make([]match.PathMatcher, 0, 3)

	if opts.Regex != nil {
		if matcher, err := regex.NewMatcher(*opts.Regex); err != nil {
			return nil, fmt.Errorf("regex - %w", err)
		} else {
			matchers = append(matchers, matcher)
		}
	}

	if opts.Glob != nil {
		if matcher, err := glob.NewStrictMatcher(*opts.Glob); err != nil {
			return nil, fmt.Errorf("glob - %w", err)
		} else {
			matchers = append(matchers, matcher)
		}
	}

	if opts.GitIgnore != nil {
		if matcher, err := gitignore.NewMatcher(*opts.GitIgnore); err != nil {
			return nil, fmt.Errorf("gitignore - %w", err)
		} else {
			matchers = append(matchers, matcher)
		}
	}

	return &PathIgnore{matchers: matchers, timeout: opts.Timeout}, nil
}

func (pi *PathIgnore) ShouldIgnore(path string) (bool, error) {
	res, err := pi.ShouldIgnore2(path)
	return res.Ok(), err
}

func (pi *PathIgnore) ShouldIgnore2(path string) (match.MatchInfo, error) {
	ctx := context.TODO()
	if pi.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), pi.timeout)
		defer cancel()
	}

	for _, matcher := range pi.matchers {
		if m, err := matcher.Match2(ctx, path); err != nil {
			return nil, err
		} else if m.Ok() {
			return m, nil
		}
	}

	return match.NoMatch, nil
}
