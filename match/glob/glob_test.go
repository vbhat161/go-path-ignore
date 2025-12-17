package glob

import (
	"context"
	"testing"

	"github.com/gobwas/glob"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestNewMatcher(t *testing.T) {
	m, errors := NewMatcher(Options{
		Patterns: []string{
			"node_modules/**/*.js",
			"*.css",
		},
		RawPatterns: []string{
			"*.log",
		},
	})
	require.Empty(t, errors)
	require.NotNil(t, m)
	require.Len(t, m.globs, 3)
	require.False(t, m.parallel)
}

func TestNewMatcher_Invalid(t *testing.T) {
	invalidPath := "*[a-"
	m, errors := NewMatcher(Options{
		Patterns: []string{
			"node_modules/**/*.js",
			"*.css",
			invalidPath, // invalid
		},
		RawPatterns: []string{
			"*.log",
		},
	})
	require.Len(t, errors, 1)
	err := errors[0].(*CompileError)
	require.Equal(t, invalidPath, err.Path)
	require.NotNil(t, m)
	require.Len(t, m.globs, 3)
	require.False(t, m.parallel)
}

func TestNewStrictMatcher(t *testing.T) {
	m, errors := NewStrictMatcher(Options{
		Patterns: []string{
			"node_modules/**/*.js",
			"*.css",
		},
		RawPatterns: []string{
			"*.log",
		},
	})
	require.Empty(t, errors)
	require.NotNil(t, m)
	require.Len(t, m.globs, 3)
	require.False(t, m.parallel)
}

func TestStrictNewMatcher_Invalid(t *testing.T) {
	invalidPath := "*[a-"
	m, e := NewStrictMatcher(Options{
		Patterns: []string{
			"node_modules/**/*.js",
			"*.css",
			invalidPath, // invalid
		},
		RawPatterns: []string{
			"*.log",
		},
	})
	require.Error(t, e)
	err := e.(*CompileError)
	require.Equal(t, invalidPath, err.Path)
	require.Nil(t, m)
}

func TestGlob(t *testing.T) {
	gg := glob.MustCompile("*test*")
	val := gg.Match("atest.go")
	require.True(t, val)
}

func TestMatch(t *testing.T) {
	defer goleak.VerifyNone(t) // Ensure no goroutines are leaked

	testCases := []struct {
		name        string
		options     Options
		input       string
		parallel    bool
		expected    bool
		expectedErr error
	}{
		{
			name: "single glob match",
			options: Options{
				Patterns: []string{"*.go"},
			},
			input:    "main.go",
			expected: true,
		},
		{
			name: "single glob no match",
			options: Options{
				Patterns: []string{"*.go"},
			},
			input:    "main.txt",
			expected: false,
		},
		{
			name: "multiple globs match first",
			options: Options{
				Patterns: []string{"*.txt", "*.go"},
			},
			input:    "file.txt",
			expected: true,
		},
		{
			name: "multiple globs match second",
			options: Options{
				Patterns: []string{"*.txt", "*.go"},
			},
			input:    "file.go",
			expected: true,
		},
		{
			name: "multiple globs no match",
			options: Options{
				Patterns: []string{"*.txt", "*.go"},
			},
			input:    "file.md",
			expected: false,
		},
		{
			name: "raw path match",
			options: Options{
				RawPatterns: []string{"foo/bar.txt"},
			},
			input:    "foo/bar.txt",
			expected: true,
		},
		{
			name: "raw path no match",
			options: Options{
				RawPatterns: []string{"foo/bar.txt"},
			},
			input:    "foo/baz.txt",
			expected: false,
		},
		{
			name: "glob and raw path match glob",
			options: Options{
				Patterns:    []string{"*.log"},
				RawPatterns: []string{"foo/bar.txt"},
			},
			input:    "debug.log",
			expected: true,
		},
		{
			name: "glob and raw path match raw",
			options: Options{
				Patterns:    []string{"*.log"},
				RawPatterns: []string{"foo/bar.txt"},
			},
			input:    "foo/bar.txt",
			expected: true,
		},
		{
			name: "parallel matching match",
			options: Options{
				Patterns: []string{"*.txt", "*.go", "*.md"},
			},
			parallel: true,
			input:    "document.md",
			expected: true,
		},
		{
			name: "parallel matching no match",
			options: Options{
				Patterns: []string{"*.txt", "*.go", "*.md"},
			},
			parallel: true,
			input:    "image.png",
			expected: false,
		},
		{
			name: "context cancelled before match",
			options: Options{
				Patterns: []string{"*.txt", "*.go", "*.md"},
			},
			parallel:    true,
			input:       "document.md",
			expected:    false,
			expectedErr: context.Canceled,
		},
		{
			name: "context cancelled during sequential match",
			options: Options{
				Patterns: []string{"*.txt", "*.go", "*.md"},
			},
			input:       "document.md",
			expected:    false,
			expectedErr: context.Canceled,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var m *Matcher

			if tc.parallel {
				matcher, err := NewStrictParallelMatcher(tc.options)
				require.NoError(t, err)
				m = matcher
			} else {
				matcher, err := NewStrictMatcher(tc.options)
				require.NoError(t, err)
				m = matcher
			}

			require.NotNil(t, m)

			ctx, cancel := context.WithCancel(context.Background())
			if tc.expectedErr == context.Canceled {
				cancel() // Cancel immediately
			} else {
				defer cancel()
			}

			matched, err := m.Match(ctx, tc.input)

			if tc.expectedErr != nil {
				require.ErrorIs(t, err, tc.expectedErr)
				require.False(t, matched)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, matched)
			}
		})
	}
}
