package gopathignore_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	gopathignore "github.com/vbhat161/go-path-ignore"
	"github.com/vbhat161/go-path-ignore/match/gitignore"
	"github.com/vbhat161/go-path-ignore/match/glob"
	"github.com/vbhat161/go-path-ignore/match/regex"
	"go.uber.org/goleak"
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
			wantErr: true,
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
					Patterns: []string{"foo/*"},
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
					Patterns: []string{"["}, // Invalid glob
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
			if tt.wantErr {
				require.Error(t, err)
				return
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	defer goleak.VerifyNone(t)

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
					Patterns: []string{"foo/*"},
				},
			},
			path: "foo/bar",
			want: true,
		},
		{
			name: "glob no match",
			opts: gopathignore.Options{
				Glob: &glob.Options{
					Patterns: []string{"foo/*"},
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
					Patterns: []string{"bar/*"},
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
					Patterns: []string{"bar/*"},
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
					Patterns: []string{"bar/*"},
				},
			},
			path: "baz/qux",
			want: false,
		},
		{
			name: "multiple matchers, glob matches - llel",
			opts: gopathignore.Options{
				Regex: &regex.Options{
					Patterns: []string{"^foo.*"},
				},
				Glob: &glob.Options{
					Patterns: []string{"bar/*"},
				},
				Parallel: true,
			},
			path: "bar/baz",
			want: true,
		},
		{
			name: "multiple matchers, no match - llel",
			opts: gopathignore.Options{
				Regex: &regex.Options{
					Patterns: []string{"^foo.*"},
				},
				Glob: &glob.Options{
					Patterns: []string{"bar/*"},
				},
				Parallel: true,
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
					Patterns: []string{"["},
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
			got, err := pi.Match(context.Background(), tt.path)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func Benchmark(b *testing.B) {
	bench := func(parallel bool) func(*testing.B) {
		return func(bench *testing.B) {
			opts := gopathignore.Options{
				GitIgnore: &gitignore.Options{
					Patterns: []string{
						"foo/",
						"/dir/test.*",
						"*.go",
						"!important.txt",
						"*.exe",
						"*.exe~",
						"*.dll",
						"*.so",
						"*.dylib",
						"*.test",
						"*.out",
						"coverage.*",
						"*.coverprofile",
						"profile.cov",
						"go.work",
						".env",
						".idea/",
						".vscode/",
					},
				},
				Regex: &regex.Options{
					Patterns: []string{
						`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`,
						`^(?:https?://)?(?:www\.)?[a-zA-Z0-9-]+\.[a-zA-Z]{2,}(?:/[^\s]*)?$`,
						`^(?:\+?1)?[-.\s]?\(?[0-9]{3}\)?[-.\s]?[0-9]{3}[-.\s]?[0-9]{4}$`,
						`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`,
						`^[A-Z]{2}[0-9]{6}[A-Z0-9]{3}$`,
						`^#(?:[0-9a-fA-F]{3}){1,2}$`,
						`^(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13})$`,
						`^[A-Z]{1,2}[0-9]{1,4}[A-Z]{2}$`,
						`^v?[0-9]+\.[0-9]+\.[0-9]+(?:-[a-zA-Z0-9]+)?$`,
						`^[A-Za-z0-9._%+-]+(?:\+[A-Za-z0-9.-]*)?@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}$`,
					},
				},
				Glob: &glob.Options{
					Patterns: []string{
						"*.go",
						"src/**/test_*.py",
						"**/*.json",
						"docs/**/*.md",
						"*.{txt,log,err}",
						"build/**/output_*",
						"config/**.yaml",
						".env*",
						"node_modules/**/package.json",
						"**/*_test.go",
						"src/*/main.py",
						"*.min.js",
					},
				},
				Parallel: parallel,
			}
			pi, err := gopathignore.New(opts)
			require.NoError(bench, err)
			paths := []string{
				"build/release/output_binary",
				"config/app.yaml",
				"node_modules/express/package.json",
				"test.exe",
				"/envs/.env",
				"profs/output.out",
				"CA123456ABC",
				"#FF5733",
				"4532123456789012",
				"M1 1AA",
				"1.2.3-beta",
				"user+filter@gmail.com",
				"invalid.email@",
				"not a url at all",
				"555-12345",
				"2024/12/25",
				"docs/api/reference.md",
				"docs/guides/setup.md",
				".env",
				".env.local",
				"build/dist/output_app",
				"build/release/output_binary",
				"config/app.yaml",
				"node_modules/express/package.json",
				"services_test.go",
				"utils_test.go",
				"src/auth/main.py",
				"app.min.js",
				"vendor.min.js",
				"helpers.go",
			}

			bench.ResetTimer()
			for bench.Loop() {
				for _, p := range paths {
					pi.Match2(context.Background(), p)
				}
			}
		}
	}

	b.Run("sequential", bench(false /*parallel*/))
	b.Run("parallel", bench(true /*parallel*/))
}
