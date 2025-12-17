package gitignore

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/vbhat161/go-path-ignore/match"
	regexp "github.com/wasilibs/go-re2"
)

var (
	gitEscapedFirstChar       = regexp.MustCompile(`^([#!])`)
	gitDirFileEscape          = regexp.MustCompile(`([^/+])/.*\*\.`)
	gitEscapeDot              = regexp.MustCompile(`\.`)
	gitEscapeAsterisk         = regexp.MustCompile(`\\\*`)
	gitEscapeAsterisk2        = regexp.MustCompile(`\*`)
	gitDoubleAsterisk         = regexp.MustCompile(`/\*\*/`)
	gitDoubleAsteriskParent   = regexp.MustCompile(`\*\*/`)
	gitDoubleAsteriskChildren = regexp.MustCompile(`/\*\*`)
	gitQuestionMark           = regexp.MustCompile(`(^|[^\\])\?`)
)

var _ match.PathMatcher = (*Matcher)(nil) // enfore interface

// rule encapsulates a pattern and if it is a negated pattern.
type rule struct {
	re         *regexp.Regexp
	src, rePat string
}

// Matcher wraps a list of ignore pattern.
type Matcher struct {
	src []string

	posRules []*rule
	negRules []*rule

	posSet, negSet *match.RE2Set
}

type Options struct {
	Patterns []string
	FilePath string
}

// NewMatcher returns a new matcher for given patterns or from a file path. At least one
// of patterns or filePath has to be present.
func NewMatcher(opts Options) (*Matcher, error) {
	return newMatcher(opts, false /*parallel*/)
}

func NewParallelMatcher(opts Options) (*Matcher, error) {
	return newMatcher(opts, true /*parallel*/)
}

func newMatcher(opts Options, parallel bool) (*Matcher, error) {
	if len(opts.Patterns) == 0 && opts.FilePath == "" {
		return nil, fmt.Errorf("atleast one gitignore source required: file or lines")
	}

	if opts.FilePath != "" {
		patterns, err := readPath(opts.FilePath)
		if err != nil {
			return nil, fmt.Errorf("read gitignore file: %w", err)
		}
		opts.Patterns = append(opts.Patterns, patterns...)
	}

	matcher := &Matcher{
		src: opts.Patterns,
	}
	for _, pattern := range opts.Patterns {
		res, err := matcher.parse(pattern)
		if err != nil {
			return nil, fmt.Errorf("parse gitignore line(%s): %w", pattern, err)
		}
		if res == nil { // skip
			continue
		}

		r := res.rule
		if !parallel {
			if re, err := regexp.Compile(r.rePat); err != nil {
				return nil, fmt.Errorf("compile pattern %s - %w", pattern, err)
			} else {
				r.re = re
			}
		}

		if res.negate {
			matcher.negRules = append(matcher.negRules, res.rule)
		} else {
			matcher.posRules = append(matcher.posRules, res.rule)
		}
	}

	if parallel {
		patterns := make([]string, 0, len(matcher.posRules))
		for _, p := range matcher.posRules {
			patterns = append(patterns, p.rePat)
		}

		if set, err := match.NewRE2Set(patterns); err != nil {
			return nil, fmt.Errorf("parallel: re2 set - %w", err)
		} else {
			matcher.posSet = set
		}

		if len(matcher.negRules) > 0 {
			negPatterns := make([]string, 0, len(matcher.negRules))
			for _, p := range matcher.negRules {
				negPatterns = append(negPatterns, p.rePat)
			}
			if set, err := match.NewRE2Set(negPatterns); err != nil {
				return nil, fmt.Errorf("parallel: negation re2 set - %w", err)
			} else {
				matcher.negSet = set
			}
		}

	}

	return matcher, nil
}

func (gi *Matcher) Type() match.Type {
	return match.GitIgnore
}

// Matches takes a path and returns whether it is ignored according to the list of
// ignore patterns. It returns true if the path should be ignored, and false otherwise.
func (gi *Matcher) Match(ctx context.Context, path string) (bool, error) {
	res, err := gi.Match2(ctx, path)
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
	return match.GitIgnore
}

func (r result) String() string {
	return fmt.Sprintf("%s:%s", r.Type(), r.src)
}

func (gi *Matcher) Match2(ctx context.Context, path string) (match.MatchInfo, error) {
	// Replace OS-specific path separator.
	path = strings.ReplaceAll(path, string(os.PathSeparator), "/")

	res := result{}

	var matchPath string
	if gi.posSet != nil {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		_, matchPath = gi.posSet.Matches(path)
	} else {
		for _, r := range gi.posRules {
			if ctx.Err() != nil {
				return res, ctx.Err()
			}
			if r.re.MatchString(path) {
				matchPath = r.src
				break
			}
		}
	}

	if matchPath == "" {
		return res, nil
	}

	res.src = matchPath

	if gi.negSet != nil {
		if ctx.Err() != nil {
			return res, ctx.Err()
		}
		if ok, path := gi.negSet.Matches(path); ok {
			res.src = path
		}
		return res, nil
	} else {
		for _, r := range gi.negRules {
			if ctx.Err() != nil {
				return res, ctx.Err()
			}
			if r.re.MatchString(path) {
				res.src = ""
				return res, nil
			}
		}
		return res, nil
	}
}

// readPath uses an ignore file as the input, parses the lines out of
// the file and invokes the NewGitIgnore method.
func readPath(gitignorePath string) ([]string, error) {
	data, err := os.ReadFile(gitignorePath)
	if err != nil {
		return nil, err
	}

	return strings.Split(string(data), "\n"), nil
}

type parseOut struct {
	rule   *rule
	negate bool
}

// This code is an improvised version of github.com/sabhiram/go-gitignore
// with additional bug fixes
func (gi *Matcher) parse(l string) (*parseOut, error) {
	input := l
	// Trim OS-specific carriage returns.
	l = strings.TrimRight(l, "\r")

	// Strip comments [Rule 2]
	if strings.HasPrefix(l, `#`) {
		return nil, nil
	}

	// Trim string [Rule 3]
	l = strings.Trim(l, " ")

	// Exit for no-ops and return nil which will prevent us from
	// appending a pattern against this line
	if l == "" {
		return nil, nil
	}

	hasfwSlashSuffix := strings.HasSuffix(l, "/")
	hasFwSlash := strings.Contains(l[:len(l)-1], "/") // except for terminal fw slash

	negate := false
	if l[0] == '!' {
		negate = true
		l = l[1:]
	}

	// replace range negations with regex negation
	l = strings.ReplaceAll(l, "[!", "[^")

	// Handle [Rule 2, 4], when # or ! is escaped with a \
	// Handle [Rule 4] once we tag negatePattern, strip the leading ! char
	if gitEscapedFirstChar.MatchString(l) {
		l = l[1:]
	}

	// If we encounter a foo/*.blah in a folder, prepend the / char
	if gitDirFileEscape.MatchString(l) && l[0] != '/' {
		l = "/" + l
	}

	// Handle escaping the "." char
	l = gitEscapeDot.ReplaceAllString(l, `\.`)

	placeholder := "#$~"

	// Handle "/**/" usage
	if strings.HasPrefix(l, "/**/") {
		l = l[1:]
	}
	// Handle escaping the "?" char
	l = gitQuestionMark.ReplaceAllString(l, `$1[^/]`)

	l = gitDoubleAsterisk.ReplaceAllString(l, `(?:/|/.+/)`)
	l = gitDoubleAsteriskParent.ReplaceAllString(l, `(?:|.`+placeholder+`/)`)
	l = gitDoubleAsteriskChildren.ReplaceAllString(l, `/.`+placeholder)

	// Handle escaping the "*" char
	l = gitEscapeAsterisk.ReplaceAllString(l, `\`+placeholder)
	l = gitEscapeAsterisk2.ReplaceAllString(l, `[^/]*`)

	l = strings.ReplaceAll(l, placeholder, "*")

	expr := ""
	if hasfwSlashSuffix {
		expr = l + "(?:|.*)$"
	} else {
		expr = l + "(?:|/.*)$"
	}

	if hasFwSlash {
		if strings.HasPrefix(l, "/") {
			expr = expr[1:]
		}
		expr = "^(?:|/)" + expr
	} else {
		expr = "^(?:|.*/)" + expr
	}

	rule := &rule{src: input, rePat: expr}
	return &parseOut{rule: rule, negate: negate}, nil
}
