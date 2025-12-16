package glob

import (
	"context"
	"sync"

	"github.com/VishwaBhat/go-path-ignore/match"
	"github.com/gobwas/glob"
)

type CompileError struct {
	Err  error
	Path string
}

func newCompileError(path string, err error) *CompileError {
	return &CompileError{Err: err, Path: path}
}

func (c CompileError) Error() string {
	return c.Err.Error()
}

/*
* This is a convenient wrapper around github.com/gobwas/glob
* that allows for both sequential and parallel glob matching.
* The glob patterns are compiled only once and reused.
 */
type Matcher struct {
	globs    []glob.Glob
	parallel bool
}

type Options struct {
	Paths    []string
	RawPaths []string
	Parallel bool
}

func NewMatcher(opts Options) (*Matcher, []error) {
	globs := make([]glob.Glob, 0, len(opts.Paths))
	var errs []error
	for _, p := range opts.Paths {
		g, err := glob.Compile(p)
		if err != nil {
			errs = append(errs, newCompileError(p, err))
			continue
		}
		globs = append(globs, g)
	}

	for _, p := range opts.RawPaths {
		escaped := glob.QuoteMeta(p)
		g, err := glob.Compile(escaped)
		if err != nil {
			errs = append(errs, newCompileError(p, err))
			continue
		}
		globs = append(globs, g)
	}

	return &Matcher{globs: globs, parallel: opts.Parallel}, errs
}

func NewStrictMatcher(opts Options) (*Matcher, error) {
	globs := make([]glob.Glob, 0, len(opts.Paths))
	for _, p := range opts.Paths {
		g, err := glob.Compile(p)
		if err != nil {
			return nil, newCompileError(p, err)
		}
		globs = append(globs, g)
	}
	for _, p := range opts.RawPaths {
		escaped := glob.QuoteMeta(p)
		g, err := glob.Compile(escaped)
		if err != nil {
			return nil, newCompileError(p, err)
		}
		globs = append(globs, g)
	}
	return &Matcher{globs: globs, parallel: opts.Parallel}, nil
}

func (m *Matcher) Type() match.Type {
	return match.Glob
}

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
	return match.Glob
}

func (m *Matcher) Match2(ctx context.Context, path string) (match.MatchInfo, error) {
	res := result{}
	if m.parallel {
		if path, err := m.concurrentMatch(ctx, path); err != nil {
			return res, err
		} else {
			res.src = path
			return res, nil
		}
	} else {
		for _, g := range m.globs {
			if ctx.Err() != nil {
				return res, ctx.Err()
			}
			if g.Match(path) {
				res.src = path
				return res, nil
			}
		}
		return res, nil
	}
}

func (m *Matcher) concurrentMatch(ctx context.Context, path string) (string, error) {
	foundSrc := make(chan string, 1)
	defer close(foundSrc)

	matchCtx, stopMatch := context.WithCancel(ctx)
	defer stopMatch()

	var wg sync.WaitGroup
	for _, g := range m.globs {
		wg.Go(func() {
			if matchCtx.Err() != nil {
				return
			}
			if g.Match(path) {
				foundSrc <- path
				stopMatch()
			}
		})
	}

	wg.Wait()

	select {
	case p := <-foundSrc:
		return p, nil
	case <-matchCtx.Done():
		return "", matchCtx.Err()
	default:
		return "", nil
	}
}
