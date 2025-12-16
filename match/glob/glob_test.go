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
		Paths: []string{
			"node_modules/**/*.js",
			"*.css",
		},
		RawPaths: []string{
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
		Paths: []string{
			"node_modules/**/*.js",
			"*.css",
			invalidPath, // invalid
		},
		RawPaths: []string{
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
		Paths: []string{
			"node_modules/**/*.js",
			"*.css",
		},
		RawPaths: []string{
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
		Paths: []string{
			"node_modules/**/*.js",
			"*.css",
			invalidPath, // invalid
		},
		RawPaths: []string{
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
	testCases := []struct {
		name        string
		options     Options
		path        string
		expected    bool
		expectedErr error
	}{
		{
			name: "single glob match",
			options: Options{
				Paths: []string{"*.go"},
			},
			path:     "main.go",
			expected: true,
		},
		{
			name: "single glob no match",
			options: Options{
				Paths: []string{"*.go"},
			},
			path:     "main.txt",
			expected: false,
		},
		{
			name: "multiple globs match first",
			options: Options{
				Paths: []string{"*.txt", "*.go"},
			},
			path:     "file.txt",
			expected: true,
		},
		{
			name: "multiple globs match second",
			options: Options{
				Paths: []string{"*.txt", "*.go"},
			},
			path:     "file.go",
			expected: true,
		},
		{
			name: "multiple globs no match",
			options: Options{
				Paths: []string{"*.txt", "*.go"},
			},
			path:     "file.md",
			expected: false,
		},
		{
			name: "raw path match",
			options: Options{
				RawPaths: []string{"foo/bar.txt"},
			},
			path:     "foo/bar.txt",
			expected: true,
		},
		{
			name: "raw path no match",
			options: Options{
				RawPaths: []string{"foo/bar.txt"},
			},
			path:     "foo/baz.txt",
			expected: false,
		},
		{
			name: "glob and raw path match glob",
			options: Options{
				Paths:    []string{"*.log"},
				RawPaths: []string{"foo/bar.txt"},
			},
			path:     "debug.log",
			expected: true,
		},
		{
			name: "glob and raw path match raw",
			options: Options{
				Paths:    []string{"*.log"},
				RawPaths: []string{"foo/bar.txt"},
			},
			path:     "foo/bar.txt",
			expected: true,
		},
		{
			name: "parallel matching match",
			options: Options{
				Paths:    []string{"*.txt", "*.go", "*.md"},
				Parallel: true,
			},
			path:     "document.md",
			expected: true,
		},
		{
			name: "parallel matching no match",
			options: Options{
				Paths:    []string{"*.txt", "*.go", "*.md"},
				Parallel: true,
			},
			path:     "image.png",
			expected: false,
		},
		{
			name: "context cancelled before match",
			options: Options{
				Paths:    []string{"*.txt", "*.go", "*.md"},
				Parallel: true,
			},
			path:        "document.md",
			expected:    false,
			expectedErr: context.Canceled,
		},
		{
			name: "context cancelled during sequential match",
			options: Options{
				Paths: []string{"*.txt", "*.go", "*.md"},
			},
			path:        "document.md",
			expected:    false,
			expectedErr: context.Canceled,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.options.Parallel {
				defer goleak.VerifyNone(t) // Ensure no goroutines are leaked
			}
			m, errors := NewMatcher(tc.options)
			require.Empty(t, errors)
			require.NotNil(t, m)

			ctx, cancel := context.WithCancel(context.Background())
			if tc.expectedErr == context.Canceled {
				cancel() // Cancel immediately
			} else {
				defer cancel()
			}

			matched, err := m.Match(ctx, tc.path)

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
