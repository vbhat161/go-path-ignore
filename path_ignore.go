package gopathignore

import (
	"context"
	"fmt"
	"time"

	"github.com/vbhat161/go-path-ignore/match"
	"github.com/vbhat161/go-path-ignore/match/gitignore"
	"github.com/vbhat161/go-path-ignore/match/glob"
	"github.com/vbhat161/go-path-ignore/match/regex"
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
	Parallel  bool
}

func New(opts Options) (*PathIgnore, error) {
	matchers := make([]match.PathMatcher, 0, 3)
	atleastOneMatcher := opts.Regex != nil || opts.Glob != nil || opts.GitIgnore != nil

	if !atleastOneMatcher {
		return nil, fmt.Errorf("atleast one matching strategy required")
	}

	if opts.Regex != nil {
		var matcher *regex.Matcher
		var err error
		if opts.Parallel {
			matcher, err = regex.NewParallelMatcher(*opts.Regex)
		} else {
			matcher, err = regex.NewMatcher(*opts.Regex)
		}
		if err != nil {
			return nil, fmt.Errorf("regex - %w", err)
		}
		matchers = append(matchers, matcher)
	}

	if opts.GitIgnore != nil {
		var matcher *gitignore.Matcher
		var err error
		if opts.Parallel {
			matcher, err = gitignore.NewParallelMatcher(*opts.GitIgnore)
		} else {
			matcher, err = gitignore.NewMatcher(*opts.GitIgnore)
		}
		if err != nil {
			return nil, fmt.Errorf("gitignore - %w", err)
		}
		matchers = append(matchers, matcher)
	}

	if opts.Glob != nil {
		var matcher *glob.Matcher
		var err error
		if opts.Parallel {
			matcher, err = glob.NewStrictParallelMatcher(*opts.Glob)
		} else {
			matcher, err = glob.NewStrictMatcher(*opts.Glob)
		}
		if err != nil {
			return nil, fmt.Errorf("glob - %w", err)
		}
		matchers = append(matchers, matcher)
	}

	return &PathIgnore{matchers: matchers, timeout: opts.Timeout}, nil
}

func (pi *PathIgnore) Match(ctx context.Context, path string) (bool, error) {
	res, err := pi.Match2(ctx, path)
	return res.Ok(), err
}

func (pi *PathIgnore) Match2(ctx context.Context, path string) (match.MatchInfo, error) {
	timeout := pi.timeout
	if timeout == 0 {
		timeout = time.Hour // max
	}

	matchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for _, matcher := range pi.matchers {
		if m, err := matcher.Match2(matchCtx, path); err != nil {
			return nil, err
		} else if m.Ok() {
			cancel()
			return m, nil
		}
	}

	return match.NoMatch, nil
}
