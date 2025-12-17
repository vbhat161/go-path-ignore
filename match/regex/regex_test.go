package regex

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewMatcher(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr bool
	}{
		{
			name:    "valid pattern",
			opts:    Options{Patterns: []string{"foo"}},
			wantErr: false,
		},
		{
			name:    "multiple valid patterns",
			opts:    Options{Patterns: []string{"foo", "bar.*"}},
			wantErr: false,
		},
		{
			name:    "empty patterns list",
			opts:    Options{Patterns: []string{}},
			wantErr: true,
		},
		{
			name:    "invalid regex pattern",
			opts:    Options{Patterns: []string{"["}},
			wantErr: true,
		},
		{
			name:    "literal patterns",
			opts:    Options{Patterns: []string{"foo.bar", "baz*"}, Literals: true},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewMatcher(tt.opts)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.opts.Literals {
					require.Len(t, m.regexps, 1)
				} else {
					require.Len(t, m.regexps, len(tt.opts.Patterns))
				}
			}
		})
	}
}

func TestNewParallelMatcher(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		wantErr bool
	}{
		{
			name:    "valid pattern",
			opts:    Options{Patterns: []string{"foo"}},
			wantErr: false,
		},
		{
			name:    "multiple valid patterns",
			opts:    Options{Patterns: []string{"foo", "bar.*"}},
			wantErr: false,
		},
		{
			name:    "empty patterns list",
			opts:    Options{Patterns: []string{}},
			wantErr: true,
		},
		{
			name:    "invalid regex pattern",
			opts:    Options{Patterns: []string{"["}},
			wantErr: true,
		},
		{
			name:    "literal patterns",
			opts:    Options{Patterns: []string{"foo.bar", "baz*"}, Literals: true},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, err := NewParallelMatcher(tt.opts)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, m.set)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		path    string
		want    bool
		wantErr bool
	}{
		{
			name:    "single pattern match",
			opts:    Options{Patterns: []string{"foo"}},
			path:    "foo",
			want:    true,
			wantErr: false,
		},
		{
			name:    "single pattern no match",
			opts:    Options{Patterns: []string{"foo"}},
			path:    "bar",
			want:    false,
			wantErr: false,
		},
		{
			name:    "multiple patterns match first",
			opts:    Options{Patterns: []string{"foo", "bar"}},
			path:    "foo",
			want:    true,
			wantErr: false,
		},
		{
			name:    "multiple patterns match second",
			opts:    Options{Patterns: []string{"foo", "bar"}},
			path:    "bar",
			want:    true,
			wantErr: false,
		},
		{
			name:    "multiple patterns no match",
			opts:    Options{Patterns: []string{"foo", "bar"}},
			path:    "baz",
			want:    false,
			wantErr: false,
		},
		{
			name:    "literal pattern match",
			opts:    Options{Patterns: []string{"foo.bar"}, Literals: true},
			path:    "foo.bar",
			want:    true,
			wantErr: false,
		},
		{
			name:    "literal pattern no match regex char",
			opts:    Options{Patterns: []string{"foo.bar"}, Literals: true},
			path:    "fooxbar",
			want:    false,
			wantErr: false,
		},
		{
			name:    "regex pattern match with dot",
			opts:    Options{Patterns: []string{"foo.bar"}},
			path:    "fooxbar",
			want:    true,
			wantErr: false,
		},
		{
			name:    "empty path no match",
			opts:    Options{Patterns: []string{"foo"}},
			path:    "",
			want:    false,
			wantErr: false,
		},
		{
			name:    "empty pattern list (should error on matcher creation)",
			opts:    Options{Patterns: []string{}},
			path:    "foo",
			want:    false,
			wantErr: true,
		},
		{
			name:    "invalid regex pattern (should error on matcher creation)",
			opts:    Options{Patterns: []string{"["}},
			path:    "foo",
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewMatcher(tt.opts)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			got, err := matcher.Match(context.Background(), tt.path)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParallelMatch(t *testing.T) {
	tests := []struct {
		name    string
		opts    Options
		path    string
		want    bool
		wantErr bool
	}{
		{
			name:    "single pattern match",
			opts:    Options{Patterns: []string{"foo"}},
			path:    "foo",
			want:    true,
			wantErr: false,
		},
		{
			name:    "single pattern no match",
			opts:    Options{Patterns: []string{"foo"}},
			path:    "bar",
			want:    false,
			wantErr: false,
		},
		{
			name:    "multiple patterns match first",
			opts:    Options{Patterns: []string{"foo", "bar"}},
			path:    "foo",
			want:    true,
			wantErr: false,
		},
		{
			name:    "multiple patterns match second",
			opts:    Options{Patterns: []string{"foo", "bar"}},
			path:    "bar",
			want:    true,
			wantErr: false,
		},
		{
			name:    "multiple patterns no match",
			opts:    Options{Patterns: []string{"foo", "bar"}},
			path:    "baz",
			want:    false,
			wantErr: false,
		},
		{
			name:    "literal pattern match",
			opts:    Options{Patterns: []string{"foo.bar"}, Literals: true},
			path:    "foo.bar",
			want:    true,
			wantErr: false,
		},
		{
			name:    "literal pattern no match regex char",
			opts:    Options{Patterns: []string{"foo.bar"}, Literals: true},
			path:    "fooxbar",
			want:    false,
			wantErr: false,
		},
		{
			name:    "regex pattern match with dot",
			opts:    Options{Patterns: []string{"foo.bar"}},
			path:    "fooxbar",
			want:    true,
			wantErr: false,
		},
		{
			name:    "empty path no match",
			opts:    Options{Patterns: []string{"foo"}},
			path:    "",
			want:    false,
			wantErr: false,
		},
		{
			name:    "empty pattern list (should error on matcher creation)",
			opts:    Options{Patterns: []string{}},
			path:    "foo",
			want:    false,
			wantErr: true,
		},
		{
			name:    "invalid regex pattern (should error on matcher creation)",
			opts:    Options{Patterns: []string{"["}},
			path:    "foo",
			want:    false,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher, err := NewParallelMatcher(tt.opts)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			got, err := matcher.Match(context.Background(), tt.path)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}
