package gopathignore_test

import (
	"testing"

	gopathignore "github.com/VishwaBhat/go-path-ignore"
	"github.com/VishwaBhat/go-path-ignore/match/gitignore"
	"github.com/VishwaBhat/go-path-ignore/match/glob"
	"github.com/VishwaBhat/go-path-ignore/match/regex"
	"github.com/stretchr/testify/require"
)

func TestNewPathIgnore(t *testing.T) {

	tests := []struct {
		name    string
		opts    gopathignore.Options
		wantErr bool
	}{
		{
			name:    "empty options",
			opts:    gopathignore.Options{},
			wantErr: false,
		},
		{
			name: "regex patterns",
			opts: gopathignore.Options{
				Regex: &regex.Options{
					Patterns: []string{"^foo.*"},
				},
			},
			wantErr: false,
		},
		{
			name: "glob patterns",
			opts: gopathignore.Options{
				Glob: &glob.Options{
					Paths: []string{"foo/*"},
				},
			},
			wantErr: false,
		},
		{
			name: "gitignore patterns",
			opts: gopathignore.Options{
				GitIgnore: &gitignore.Options{
					Patterns: []string{"foo/"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid regex pattern",
			opts: gopathignore.Options{
				Regex: &regex.Options{
					Patterns: []string{"["}, // Invalid regex
				},
			},
			wantErr: true,
		},
		{
			name: "invalid glob pattern",
			opts: gopathignore.Options{
				Glob: &glob.Options{
					Paths: []string{"["}, // Invalid glob
				},
			},
			wantErr: true,
		},
		{
			name: "invalid gitignore options (no patterns or file)",
			opts: gopathignore.Options{
				GitIgnore: &gitignore.Options{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := gopathignore.New(tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPathIgnore() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestShouldIgnore(t *testing.T) {
	tests := []struct {
		name    string
		opts    gopathignore.Options
		path    string
		want    bool
		wantErr bool
	}{
		{
			name: "regex match",
			opts: gopathignore.Options{
				Regex: &regex.Options{
					Patterns: []string{"^foo.*"},
				},
			},
			path: "foobar",
			want: true,
		},
		{
			name: "regex no match",
			opts: gopathignore.Options{
				Regex: &regex.Options{
					Patterns: []string{"^foo.*"},
				},
			},
			path: "barfoo",
			want: false,
		},
		{
			name: "glob match",
			opts: gopathignore.Options{
				Glob: &glob.Options{
					Paths: []string{"foo/*"},
				},
			},
			path: "foo/bar",
			want: true,
		},
		{
			name: "glob no match",
			opts: gopathignore.Options{
				Glob: &glob.Options{
					Paths: []string{"foo/*"},
				},
			},
			path: "bar/foo",
			want: false,
		},
		{
			name: "gitignore match",
			opts: gopathignore.Options{
				GitIgnore: &gitignore.Options{
					Patterns: []string{"foo/"},
				},
			},
			path: "foo/bar",
			want: true,
		},
		{
			name: "gitignore no match",
			opts: gopathignore.Options{
				GitIgnore: &gitignore.Options{
					Patterns: []string{"foo/"},
				},
			},
			path: "bar/foo",
			want: false,
		},
		{
			name: "multiple matchers, regex matches",
			opts: gopathignore.Options{
				Regex: &regex.Options{
					Patterns: []string{"^foo.*"},
				},
				Glob: &glob.Options{
					Paths: []string{"bar/*"},
				},
			},
			path: "foobar",
			want: true,
		},
		{
			name: "multiple matchers, glob matches",
			opts: gopathignore.Options{
				Regex: &regex.Options{
					Patterns: []string{"^foo.*"},
				},
				Glob: &glob.Options{
					Paths: []string{"bar/*"},
				},
			},
			path: "bar/baz",
			want: true,
		},
		{
			name: "multiple matchers, no match",
			opts: gopathignore.Options{
				Regex: &regex.Options{
					Patterns: []string{"^foo.*"},
				},
				Glob: &glob.Options{
					Paths: []string{"bar/*"},
				},
			},
			path: "baz/qux",
			want: false,
		},
		{
			name: "invalid regex pattern during creation",
			opts: gopathignore.Options{
				Regex: &regex.Options{
					Patterns: []string{"["},
				},
			},
			path:    "foobar",
			wantErr: true,
		},
		{
			name: "invalid glob pattern during creation",
			opts: gopathignore.Options{
				Glob: &glob.Options{
					Paths: []string{"["},
				},
			},
			path:    "foobar",
			wantErr: true,
		},
		{
			name: "invalid gitignore options during creation",
			opts: gopathignore.Options{
				GitIgnore: &gitignore.Options{},
			},
			path:    "foobar",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pi, err := gopathignore.New(tt.opts)
			if tt.wantErr {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}
			got, err := pi.ShouldIgnore(tt.path)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
