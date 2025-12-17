package glob

import (
	"context"
	"fmt"
	"sync"

	"github.com/gobwas/glob"
	"github.com/vbhat161/go-path-ignore/match"
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
	Patterns    []string
	RawPatterns []string
}

func NewMatcher(opts Options) (*Matcher, []error) {
	globs := make([]glob.Glob, 0, len(opts.Patterns))
	var errs []error
	for _, p := range opts.Patterns {
		g, err := glob.Compile(p)
		if err != nil {
			errs = append(errs, newCompileError(p, err))
			continue
		}
		globs = append(globs, g)
	}

	for _, p := range opts.RawPatterns {
		escaped := glob.QuoteMeta(p)
		g, err := glob.Compile(escaped)
		if err != nil {
			errs = append(errs, newCompileError(p, err))
			continue
		}
		globs = append(globs, g)
	}

	return &Matcher{globs: globs, parallel: false}, errs
}

func NewStrictMatcher(opts Options) (*Matcher, error) {
	return newStrictMatcher(opts, false /*parallel*/)
}

func NewStrictParallelMatcher(opts Options) (*Matcher, error) {
	return newStrictMatcher(opts, true /*parallel*/)
}

func newStrictMatcher(opts Options, llel bool) (*Matcher, error) {
	globs := make([]glob.Glob, 0, len(opts.Patterns))
	for _, p := range opts.Patterns {
		g, err := glob.Compile(p)
		if err != nil {
			return nil, newCompileError(p, err)
		}
		globs = append(globs, g)
	}
	for _, p := range opts.RawPatterns {
		escaped := glob.QuoteMeta(p)
		g, err := glob.Compile(escaped)
		if err != nil {
			return nil, newCompileError(p, err)
		}
		globs = append(globs, g)
	}
	return &Matcher{globs: globs, parallel: llel}, nil
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

func (r result) String() string {
	return fmt.Sprintf("%s:%s", r.Type(), r.src)
}

func (m *Matcher) Match2(ctx context.Context, path string) (match.MatchInfo, error) {
	if ctx.Err() != nil {
		return match.NoMatch, ctx.Err()
	}

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
			select {
			case <-ctx.Done():
				return res, ctx.Err()
			default:
				if g.Match(path) {
					res.src = path
					return res, nil
				}
			}
		}
		return res, nil
	}
}

func (m *Matcher) concurrentMatch(ctx context.Context, path string) (string, error) {
	foundSrc := make(chan string, 1)

	matchCtx, stopMatch := context.WithCancel(ctx)
	defer stopMatch()

	var wg sync.WaitGroup
	for _, g := range m.globs {
		wg.Go(func() {
			if matchCtx.Err() != nil {
				return
			}
			if g.Match(path) {
				select {
				case <-matchCtx.Done():
				case foundSrc <- path:
					stopMatch()
				}
			}
		})
	}

	go func() {
		wg.Wait()
		close(foundSrc)
	}()

	select {
	case p := <-foundSrc:
		return p, nil
	case <-matchCtx.Done():
		return "", nil
	}
}
