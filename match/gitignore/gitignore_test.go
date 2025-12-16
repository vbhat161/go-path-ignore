package gitignore

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGitIgnore(t *testing.T) {
	gi, err := NewMatcher(Options{Patterns: []string{`file-?.*.log`, `!important.txt`}})
	require.NoError(t, err)
	require.NotEmpty(t, gi)
	require.Len(t, gi.posRules, 1)
	require.Len(t, gi.negRules, 1)
	require.Equal(t, `^(?:|.*/)file-[^/]\.[^/]*\.log(?:|/.*)$`, gi.posRules[0].re.String())
	require.Equal(t, `^(?:|.*/)important\.txt(?:|/.*)$`, gi.negRules[0].re.String())
}

func TestGitIgnoreMatches(t *testing.T) {
	dir, err := os.MkdirTemp("", "test-gitignore-*")
	require.NoError(t, err)

	gitAvailable := isGitAvailable(t)

	if gitAvailable {
		cmd := exec.Command("git", "init")
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git init cmd run failure: %s", out)
		defer os.RemoveAll(dir)
	}

	t.Run("Individual Patterns", func(tt *testing.T) {
		testCases := []struct {
			pattern               string
			name                  string
			matching, nonMatching []string
		}{
			{
				pattern: "*.log",
				name:    "Wildcard - matches any .log file at any depth",
				matching: []string{
					"debug.log",
					"error.log",
					"src/debug.log",
					"src/test/error.log",
					"a/b/c/d/file.log",
					".log",
				},
				nonMatching: []string{
					"log",
					"debug.txt",
					"error.log.bak",
				},
			},
			{
				pattern: "*.[oa]",
				name:    "Character class - matches .o and .a files anywhere",
				matching: []string{
					"file.o",
					"lib.a",
					"src/internal.o",
					"build/output.a",
				},
				nonMatching: []string{
					"file.obj",
					"archive.tar",
					"test.c",
				},
			},
			{
				pattern: "build/",
				name:    "Trailing slash - matches directory only at any level",
				matching: []string{
					"build/",
					"src/build/",
					"project/src/build/",
				},
				nonMatching: []string{
					"build",
					"build.txt",
					"mybuild/",
				},
			},
			{
				pattern: "/TODO",
				name:    "Leading slash - matches only at root level",
				matching: []string{
					"TODO",
				},
				nonMatching: []string{
					"src/TODO",
					"doc/TODO",
					"a/b/TODO",
				},
			},
			{
				pattern: "doc/frotz",
				name:    "Middle slash - relative to gitignore location",
				matching: []string{
					"doc/frotz",
				},
				nonMatching: []string{
					"frotz",
					"src/doc/frotz",
					"a/doc/frotz",
					"doc/src/frotz",
				},
			},
			{
				pattern: "foo/*",
				name:    "Wildcard after slash - matches immediate children only",
				matching: []string{
					"foo/test.json",    // Direct match
					"foo/bar",          // Direct match (can be file or directory)
					"foo/index.html",   // Direct match
					"foo/bar/hello.c",  // Transitive: ignored because parent foo/bar matched
					"foo/bar/test.txt", // Transitive: ignored because parent foo/bar matched
				},
				nonMatching: []string{
					"src/foo/test.json",
					"abc/src/foo/",
				},
			},
			{
				pattern: "**/logs",
				name:    "Leading ** - matches directory at any depth",
				matching: []string{
					"logs",
					"build/logs",
					"a/b/c/logs",
					"src/test/logs",
					"logs/debug.log", // Transitive: ignored because parent logs matched
				},
				nonMatching: []string{
					"logs.txt",
					"mylogs",
				},
			},
			{
				pattern: "a/**/b",
				name:    "Middle ** - matches zero or more directories",
				matching: []string{
					"a/b",
					"a/x/b",
					"a/x/y/b",
					"a/x/y/z/b",
					"a/b/c",   // Transitive: ignored because parent a/b matched
					"a/b/x/y", // Transitive: ignored because parent a/b matched
				},
				nonMatching: []string{
					"b",
					"x/a/b",
				},
			},
			{
				pattern: "abc/**",
				name:    "Trailing /** - matches all files inside directory",
				matching: []string{
					"abc/file.txt",
					"abc/x/file.txt",
					"abc/x/y/z/deep.txt",
				},
				nonMatching: []string{
					"abc",
					"xabc/file.txt",
					"abcd/file.txt",
				},
			},
			{
				pattern: "?.txt",
				name:    "Question mark - matches single character",
				matching: []string{
					"a.txt",
					"b.txt",
					"1.txt",
				},
				nonMatching: []string{
					"ab.txt",
					"test.txt",
					".txt",
				},
			},
			{
				pattern: "[a-z].txt",
				name:    "Character range - matches lowercase letter",
				matching: []string{
					"a.txt",
					"m.txt",
					"z.txt",
				},
				nonMatching: []string{
					"1.txt",
					"ab.txt",
				},
			},
			{
				pattern: `\#file.txt`,
				name:    "Escaped hash - matches literal #file.txt",
				matching: []string{
					"#file.txt",
				},
				nonMatching: []string{
					"file.txt",
					"\\#file.txt",
				},
			},
			{
				pattern: `foo\*.txt`,
				name:    "Escaped asterisk - matches literal asterisk",
				matching: []string{
					"foo*.txt",
				},
				nonMatching: []string{
					"foobar.txt",
					"foo.txt",
				},
			},
			{
				pattern: ".gitignore",
				name:    "Hidden file pattern",
				matching: []string{
					".gitignore",
					"src/.gitignore",
					"docs/project/.gitignore",
				},
				nonMatching: []string{
					"gitignore",
					".gitignore.bak",
				},
			},
			// COMMON USE CASES
			{
				pattern: "node_modules/",
				name:    "Common - node_modules directory at any level",
				matching: []string{
					"node_modules/",
					"project/node_modules/",
					"src/frontend/node_modules/",
				},
				nonMatching: []string{
					"node_modules.bak/",
					"my_node_modules/",
				},
			},
			{
				pattern: ".DS_Store",
				name:    "macOS metadata file at any level",
				matching: []string{
					".DS_Store",
					"src/.DS_Store",
					"a/b/c/.DS_Store",
				},
				nonMatching: []string{
					"DS_Store",
					".DS_Store.backup",
				},
			},
			{
				pattern: "/bin/",
				name:    "Directory only at root",
				matching: []string{
					"bin/",
				},
				nonMatching: []string{
					"src/bin/",
					"tools/bin/",
				},
			},
			{
				pattern: "logs/*",
				name:    "All immediate children in logs directory",
				matching: []string{
					"logs/debug.log",      // Direct match
					"logs/error.log",      // Direct match
					"logs/app.log",        // Direct match
					"logs/2024/debug.log", // Transitive: ignored because parent logs/2024 matched
				},
				nonMatching: []string{
					"src/logs/error.log",
					"abc/src/logs",
				},
			},
			{
				pattern: "src/*.c",
				name:    "C files in src directory only",
				matching: []string{
					"src/main.c",
					"src/util.c",
				},
				nonMatching: []string{
					"src/subdir/main.c",
					"main.c",
					"src/main.h",
				},
			},
			{
				pattern: "**/cache/**",
				name:    "Cache directory and all contents at any level",
				matching: []string{
					"cache/file.tmp",
					"src/cache/data.bin",
					"a/b/cache/x/y/file.tmp",
				},
				nonMatching: []string{
					"cached/file.tmp",
					"mycache/file.tmp",
				},
			},
			{
				pattern: "*.swp",
				name:    "Vim swap files",
				matching: []string{
					".file.swp",
					"main.c.swp",
					"src/.index.swp",
				},
				nonMatching: []string{
					"swp",
					"file.swap",
				},
			},
			{
				pattern: "*.[Ll]og",
				name:    "Log files with case variations",
				matching: []string{
					"app.log",
					"debug.Log",
				},
				nonMatching: []string{
					"app.logg",
					"app.Labc",
				},
			},
			{
				pattern: "docs/**/*.md",
				name:    "All markdown files in docs directory",
				matching: []string{
					"docs/README.md",
					"docs/api/endpoints.md",
					"docs/guide/intro.md",
				},
				nonMatching: []string{
					"README.md",
					"src/docs/README.md",
				},
			},
			{
				pattern: ".env*",
				name:    "Environment configuration files",
				matching: []string{
					".env",
					".env.local",
					".env.production",
					"config/.env.test",
				},
				nonMatching: []string{
					"env",
					"environment.txt",
				},
			},
			{
				pattern: "**/*.test.js",
				name:    "Test files at any depth",
				matching: []string{
					"app.test.js",
					"src/app.test.js",
					"src/components/button.test.js",
				},
				nonMatching: []string{
					"app.spec.js",
					"test.js",
				},
			},
			{
				pattern: "/*",
				name:    "Everything in root (typically used with negation)",
				matching: []string{
					"file.txt",         // Direct match
					"dir/",             // Direct match (the directory itself)
					"script.sh",        // Direct match
					"src/dir/file.txt", // Transitive: ignored because parent src matched
				},
			},
			// COMMENTS AND BLANK LINES
			{
				pattern:  "# This is a comment",
				name:     "Comment line - should be ignored entirely",
				matching: []string{},
				nonMatching: []string{
					"# This is a comment",
					"file.txt",
				},
			},
			{
				pattern:  "",
				name:     "Blank line - should be ignored entirely",
				matching: []string{},
				nonMatching: []string{
					"anything",
					"file.txt",
				},
			},
			// TRAILING SPACES
			{
				pattern: "file.txt   ", // Has trailing spaces
				name:    "Trailing spaces - should be ignored",
				matching: []string{
					"file.txt",
				},
				nonMatching: []string{
					"file.txt   ", // The spaces are not part of the pattern
				},
			},
			// CHARACTER CLASS VARIATIONS
			{
				pattern: "[!a-z].txt",
				name:    "Negated character class - matches non-lowercase letters",
				matching: []string{
					"1.txt",
					"_.txt",
					"-.txt",
				},
				nonMatching: []string{
					"a.txt",
					"z.txt",
					"m.txt",
				},
			},
			{
				pattern: "[-a-z].txt",
				name:    "Range starting with dash",
				matching: []string{
					"-.txt",
					"a.txt",
					"m.txt",
					"z.txt",
				},
				nonMatching: []string{
					"1.txt",
				},
			},
			{
				pattern: "[a-z][0-9].txt",
				name:    "Multiple character classes in sequence",
				matching: []string{
					"a1.txt",
					"z9.txt",
					"m5.txt",
				},
				nonMatching: []string{
					"a.txt",
					"1.txt",
					"ab.txt",
					"12.txt",
				},
			},
			// ASTERISK PATTERNS
			{
				pattern: "*",
				name:    "Single asterisk - matches any filename at any level",
				matching: []string{
					"file.txt",
					"README",
					"src/file.txt",   // Transitive: ignored because parent src matched
					"a/b/c/file.txt", // Transitive: ignored because parent a matched
				},
				nonMatching: []string{},
			},
			{
				pattern: "/**",
				name:    "Root /** - matches everything below root at all depths",
				matching: []string{
					"file.txt",
					"dir/",
					"dir/file.txt",
					"a/b/c/file.txt",
				},
				nonMatching: []string{},
			},
			{
				pattern: "**logs",
				name:    "** without slash - treated as regular ** followed by pattern",
				matching: []string{
					"logs",
					"mylogs",
					"buildlogs",
					"src/logs",
					"src/mylogs",
					"a/b/c/buildlogs",
				},
				nonMatching: []string{
					"log",
					"src/log",
				},
			},
			{
				pattern: "file***.log",
				name:    "Multiple asterisks - treated as single *",
				matching: []string{
					"file.log",
					"fileabc.log",
					"file123.log",
				},
				nonMatching: []string{
					"file/debug.log", // * doesn't cross /
				},
			},
			{
				pattern: "*.min.*",
				name:    "Wildcards at both ends",
				matching: []string{
					"app.min.js",
					"style.min.css",
					"vendor.min.map",
					"src/bundle.min.js",
				},
				nonMatching: []string{
					"app.js",
					"min.js",
					"appmin.js",
				},
			},
			{
				pattern: "*/",
				name:    "Wildcard with trailing slash - immediate subdirectories only",
				matching: []string{
					"src/",
					"build/",
					"docs/",
				},
				nonMatching: []string{
					"file.txt",
					"src", // File named src
				},
			},
			// MULTIPLE ** PATTERNS
			{
				pattern: "**/logs/**/debug.log",
				name:    "Multiple ** - logs dir at any depth, debug.log with any intermediate paths",
				matching: []string{
					"logs/debug.log",
					"logs/2024/debug.log",
					"logs/2024/01/debug.log",
					"src/logs/debug.log",
					"src/logs/app/debug.log",
					"a/b/logs/x/y/debug.log",
				},
				nonMatching: []string{
					"logs/error.log",
					"debug.log",
					"src/debug.log",
				},
			},
			{
				pattern: "\\!important.txt",
				name:    "Escaped exclamation - matches literal ! at start",
				matching: []string{
					"!important.txt",
					"src/!important.txt",
				},
				nonMatching: []string{
					"important.txt",
					"\\!important.txt",
				},
			},
			{
				pattern: "\\?file.txt",
				name:    "Escaped question mark - matches literal ?",
				matching: []string{
					"?file.txt",
					"src/?file.txt",
				},
				nonMatching: []string{
					"afile.txt",
					"xfile.txt",
				},
			},
			{
				pattern: "\\[test\\].txt",
				name:    "Escaped brackets - matches literal brackets",
				matching: []string{
					"[test].txt",
					"src/[test].txt",
				},
				nonMatching: []string{
					"test.txt",
					"t.txt",
				},
			},
			{
				pattern: "my\\ file.txt",
				name:    "Escaped space - matches filename with space",
				matching: []string{
					"my file.txt",
					"src/my file.txt",
				},
				nonMatching: []string{
					"myfile.txt",
					"my_file.txt",
				},
			},
			{
				pattern: "/build/",
				name:    "Leading and trailing slash - root directory only",
				matching: []string{
					"build/",
				},
				nonMatching: []string{
					"src/build/",
					"tools/build/",
					"build", // File named build (no trailing slash)
				},
			},
			{
				pattern: ".*",
				name:    "All hidden files (starting with dot)",
				matching: []string{
					".gitignore",
					".env",
					".hidden",
					"src/.hidden",
					"a/b/.secret",
				},
				nonMatching: []string{
					"file.txt",
					"hidden",
					"src/file.txt",
				},
			},
			{
				pattern: ".git/",
				name:    "Hidden directory at any level",
				matching: []string{
					".git/",
					"project/.git/",
					"src/.git/",
				},
				nonMatching: []string{
					".gitignore",
					"git/",
				},
			},
			{
				pattern: "*.tar.gz",
				name:    "Multi-part extension",
				matching: []string{
					"archive.tar.gz",
					"backup.tar.gz",
					"src/data.tar.gz",
				},
				nonMatching: []string{
					"archive.tar",
					"archive.gz",
					"file.targz",
				},
			},
			{
				pattern: "file-?.*.log",
				name:    "Question mark and asterisk combined",
				matching: []string{
					"file-1.debug.log",
					"file-a.error.log",
					"file-x.app.log",
				},
				nonMatching: []string{
					"file-.debug.log",   // ? requires exactly one char
					"file-12.debug.log", // ? matches only one char
					"file-a.log",        // Missing middle part
				},
			},
			{
				pattern: "src/*/test.js",
				name:    "Wildcard in middle - matches one level only",
				matching: []string{
					"src/app/test.js",
					"src/lib/test.js",
					"src/utils/test.js",
				},
				nonMatching: []string{
					"src/test.js",
					"src/app/unit/test.js",
				},
			},
			{
				pattern: "src/test/fixtures",
				name:    "Multiple slashes in pattern - relative path",
				matching: []string{
					"src/test/fixtures",
					"src/test/fixtures/file.txt",
				},
				nonMatching: []string{
					"src/fixtures",
					"test/fixtures",
					"a/src/test/fixtures",
				},
			},
			{
				pattern: "src/**/test/",
				name:    "** in middle with trailing slash",
				matching: []string{
					"src/test/",
					"src/lib/test/",
					"src/a/b/c/test/",
				},
				nonMatching: []string{
					"src/test", // No trailing slash (file)
					"test/",
					"lib/src/test/",
				},
			},
			{
				pattern: "file!.txt",
				name:    "Exclamation in middle - literal character, not negation",
				matching: []string{
					"file!.txt",
					"src/file!.txt",
				},
				nonMatching: []string{
					"file.txt",
					"!file.txt",
				},
			},
			{
				pattern: "file#1.txt",
				name:    "Hash in middle - literal character, not comment",
				matching: []string{
					"file#1.txt",
					"src/file#1.txt",
				},
				nonMatching: []string{
					"file1.txt",
					"#file1.txt",
				},
			},
			{
				pattern: "doc/frotz/",
				name:    "Pattern from gitignore docs - middle and trailing slash",
				matching: []string{
					"doc/frotz/",
				},
				nonMatching: []string{
					"a/doc/frotz/",
					"doc/frotz", // File, not directory
				},
			},
			{
				pattern: "frotz/",
				name:    "Pattern from gitignore docs - matches directory at any level",
				matching: []string{
					"frotz/",
					"a/frotz/",
					"a/b/frotz/",
				},
				nonMatching: []string{
					"frotz", // File, not directory
					"frotz.txt",
				},
			},
			{
				pattern: "Debug[01]/",
				name:    "Character class in directory name",
				matching: []string{
					"Debug0/",
					"Debug1/",
				},
				nonMatching: []string{
					"Debug/",
					"Debug2/",
					"Debug01/",
				},
			},
			{
				pattern: "foo/**/*.txt",
				name:    "** with wildcard extension - matches txt files in foo at any depth",
				matching: []string{
					"foo/bar.txt",
					"foo/a/b/c.txt",
					"foo/x/y/z/file.txt",
				},
				nonMatching: []string{
					"foo/bar.md",
					"bar/foo/file.txt",
				},
			},
			{
				pattern: "**/foo",
				name:    "Leading ** - matches foo file or directory at any depth",
				matching: []string{
					"foo",
					"a/foo",
					"a/b/c/foo",
				},
				nonMatching: []string{
					"foo.txt",
					"foobar",
				},
			},
			{
				pattern: "foo",
				name:    "Simple name - matches file or directory at any level",
				matching: []string{
					"foo",
					"src/foo",
					"a/b/c/foo",
				},
				nonMatching: []string{
					"foo.txt",
					"foobar",
				},
			},
			{
				pattern: "/foo",
				name:    "Leading slash - matches only at root",
				matching: []string{
					"foo",
				},
				nonMatching: []string{
					"src/foo",
					"a/foo",
				},
			},
			{
				pattern: "foo/bar",
				name:    "Relative path - anchored to gitignore location",
				matching: []string{
					"foo/bar",
				},
				nonMatching: []string{
					"bar",
					"foo",
					"src/foo/bar",
				},
			},
			{
				pattern: "/foo/bar",
				name:    "Absolute path from root",
				matching: []string{
					"foo/bar",
				},
				nonMatching: []string{
					"bar",
					"src/foo/bar",
				},
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(tt *testing.T) {
				gitIgnorePath := dir + "/.gitignore"
				require.NoError(tt, os.WriteFile(gitIgnorePath, []byte(tc.pattern+"\n"), 0o600))
				defer os.Remove(gitIgnorePath)
				_, err = os.Stat(gitIgnorePath)
				require.NoError(tt, err, ".gitignore is not created")
				gi, err := NewMatcher(Options{FilePath: gitIgnorePath})
				if err != nil {
					tt.Fatalf("failed to create gitignore from pattern %q: %v", tc.pattern, err)
				}
				for _, path := range tc.matching {
					if gitAvailable {
						checkIgnoreOK, err := matchesGitCheckIgnore(tt, path, dir)
						require.NoError(tt, err)
						require.True(tt, checkIgnoreOK,
							"wrong match expectation - pattern: %q, path: %q", tc.pattern, path)
					}
					matches, err := gi.Match(context.Background(), path)
					require.NoError(tt, err)

					require.True(tt, matches,
						"should match - pattern: %q, path: %q", tc.pattern, path)
				}
				for _, path := range tc.nonMatching {
					if gitAvailable {
						checkIgnoreOK, err := matchesGitCheckIgnore(tt, path, dir)
						require.NoError(tt, err, "git-check-ignore failure - pattern: %q, path: %q", tc.pattern, path)
						require.False(tt, checkIgnoreOK, "wrong non-match expectation - pattern: %q, path: %q", tc.pattern, path)
					}
					matches, err := gi.Match(context.Background(), path)
					require.NoError(tt, err)
					require.False(tt, matches, "should not match - pattern: %q, path: %q", tc.pattern, path)
				}
			})
		}
	})

	t.Run("With Negation", func(tt *testing.T) {
		patterns := []string{"src/dir-*", "!src/dir-2"}
		matching := []string{"src/dir-1", "src/dir-v2", "src/dir-final/test.txt"}
		nonMatching := []string{"src/dir-2", "src/dir-2/test.txt"}

		if gitAvailable {
			gitIgnorePath := dir + "/.gitignore"
			require.NoError(tt, os.WriteFile(gitIgnorePath, []byte(strings.Join(patterns, "\n")+"\n"), 0o600))
			defer os.Remove(gitIgnorePath)
			_, err = os.Stat(gitIgnorePath)
			require.NoError(tt, err, ".gitignore is not created")
		}

		gi, err := NewMatcher(Options{Patterns: patterns})
		if err != nil {
			t.Fatalf("failed to create gitignore from patterns %v: %v", patterns, err)
		}
		for _, path := range matching {
			if gitAvailable {
				checkIgnoreOK, err := matchesGitCheckIgnore(tt, path, dir)
				require.NoError(tt, err)
				require.True(tt, checkIgnoreOK,
					"wrong match expectation - path: %q", path)
			}
			matches, err := gi.Match(context.Background(), path)
			require.NoError(tt, err)
			require.True(tt, matches,
				"should match - path: %q", path)
		}
		for _, path := range nonMatching {
			if gitAvailable {
				checkIgnoreOK, err := matchesGitCheckIgnore(tt, path, dir)
				require.NoError(tt, err, "git-check-ignore failure - path: %q", path)
				require.False(tt, checkIgnoreOK, "wrong non-match expectation - path: %q", path)
			}
			matches, err := gi.Match(context.Background(), path)
			require.NoError(tt, err)
			require.False(tt, matches, "should not match - path: %q", path)
		}
	})
}

func BenchmarkGitIgnoreMatches(b *testing.B) {
	patterns := []string{
		"*.log",
		"*.[oa]",
		"build/",
		"/TODO",
		"doc/frotz",
		"foo/*",
		"**/logs",
		"a/**/b",
		"abc/**",
		"?.txt",
		"[a-z].txt",
		`\#file.txt`,
		`foo\*.txt`,
		".gitignore",
		"node_modules/",
		".DS_Store",
		"/bin/",
		"logs/*",
		"src/*.c",
		"**/cache/**",
		"*.swp",
		"*.[Ll]og",
		"docs/**/*.md",
		".env*",
		"**/*.test.js",
		"/*",
		"# This is a comment",
		"",
		"file.txt   ",
		"[!a-z].txt",
		"[-a-z].txt",
		"[a-z][0-9].txt",
		"*",
		"/**",
		"**logs",
		"file***.log",
		"*.min.*",
		"*/",
		"**/logs/**/debug.log",
		"\\!important.txt",
		"\\?file.txt",
		"\\[test\\].txt",
		"my\\ file.txt",
		"/build/",
		".*",
		".git/",
		"*.tar.gz",
		"file-?.*.log",
		"src/*/test.js",
		"src/test/fixtures",
		"src/**/test/",
		"file!.txt",
		"file#1.txt",
		"doc/frotz/",
		"frotz/",
		"Debug[01]/",
		"foo/**/*.txt",
		"**/foo",
		"foo",
		"/foo",
		"foo/bar",
		"/foo/bar",
	}

	matching := []string{
		"debug.log",
		"error.log",
		"src/debug.log",
		"src/test/error.log",
		"a/b/c/d/file.log",
		".log",
		"file.o",
		"lib.a",
		"src/internal.o",
		"build/output.a",
		"build/",
		"src/build/",
		"project/src/build/",
		"TODO",
		"doc/frotz",
		"foo/test.json",
		"foo/bar",
		"foo/index.html",
		"foo/bar/hello.c",
		"foo/bar/test.txt",
		"logs",
		"build/logs",
		"a/b/c/logs",
		"src/test/logs",
		"logs/debug.log",
		"a/b",
		"a/x/b",
		"a/x/y/b",
		"a/x/y/z/b",
		"a/b/c",
		"a/b/x/y",
		"abc/file.txt",
		"abc/x/file.txt",
		"abc/x/y/z/deep.txt",
		"a.txt",
		"b.txt",
		"1.txt",
		"m.txt",
		"z.txt",
		"#file.txt",
		"foo*.txt",
		".gitignore",
		"src/.gitignore",
		"docs/project/.gitignore",
		"node_modules/",
		"project/node_modules/",
		"src/frontend/node_modules/",
		".DS_Store",
		"src/.DS_Store",
		"a/b/c/.DS_Store",
		"bin/",
		"logs/error.log",
		"logs/app.log",
		"logs/2024/debug.log",
		"src/main.c",
		"src/util.c",
		"cache/file.tmp",
		"src/cache/data.bin",
		"a/b/cache/x/y/file.tmp",
		".file.swp",
		"main.c.swp",
		"src/.index.swp",
		"app.log",
		"debug.Log",
		"docs/README.md",
		"docs/api/endpoints.md",
		"docs/guide/intro.md",
		".env",
		".env.local",
		".env.production",
		"config/.env.test",
		"app.test.js",
		"src/app.test.js",
		"src/components/button.test.js",
		"file.txt",
		"dir/",
		"script.sh",
		"src/dir/file.txt",
		"_.txt",
		"-.txt",
		"a1.txt",
		"z9.txt",
		"m5.txt",
		"README",
		"src/file.txt",
		"a/b/c/file.txt",
		"dir/file.txt",
		"mylogs",
		"buildlogs",
		"src/logs",
		"src/mylogs",
		"a/b/c/buildlogs",
		"file.log",
		"fileabc.log",
		"file123.log",
		"app.min.js",
		"style.min.css",
		"vendor.min.map",
		"src/bundle.min.js",
		"src/",
		"docs/",
		"logs/2024/01/debug.log",
		"src/logs/debug.log",
		"src/logs/app/debug.log",
		"a/b/logs/x/y/debug.log",
		"!important.txt",
		"src/!important.txt",
		"?file.txt",
		"src/?file.txt",
		"[test].txt",
		"src/[test].txt",
		"my file.txt",
		"src/my file.txt",
		".env",
		".hidden",
		"src/.hidden",
		"a/b/.secret",
		".git/",
		"project/.git/",
		"src/.git/",
		"archive.tar.gz",
		"backup.tar.gz",
		"src/data.tar.gz",
		"file-1.debug.log",
		"file-a.error.log",
		"file-x.app.log",
		"src/app/test.js",
		"src/lib/test.js",
		"src/utils/test.js",
		"src/test/fixtures",
		"src/test/fixtures/file.txt",
		"src/test/",
		"src/lib/test/",
		"src/a/b/c/test/",
		"file!.txt",
		"src/file!.txt",
		"file#1.txt",
		"src/file#1.txt",
		"doc/frotz/",
		"frotz/",
		"a/frotz/",
		"a/b/frotz/",
		"Debug0/",
		"Debug1/",
		"foo/bar.txt",
		"foo/a/b/c.txt",
		"foo/x/y/z/file.txt",
		"foo",
		"a/foo",
		"a/b/c/foo",
		"src/foo",
		"foo/bar",
	}

	nonMatching := []string{
		"log",
		"debug.txt",
		"error.log.bak",
		"file.obj",
		"archive.tar",
		"test.c",
		"build",
		"build.txt",
		"mybuild/",
		"src/TODO",
		"doc/TODO",
		"a/b/TODO",
		"frotz",
		"src/doc/frotz",
		"a/doc/frotz",
		"doc/src/frotz",
		"src/foo/test.json",
		"abc/src/foo/",
		"logs.txt",
		"mylogs",
		"b",
		"x/a/b",
		"abc",
		"xabc/file.txt",
		"abcd/file.txt",
		"ab.txt",
		"test.txt",
		".txt",
		"1.txt",
		"file.txt",
		"\\#file.txt",
		"foobar.txt",
		"foo.txt",
		"gitignore",
		".gitignore.bak",
		"node_modules.bak/",
		"my_node_modules/",
		"DS_Store",
		".DS_Store.backup",
		"src/bin/",
		"tools/bin/",
		"src/logs/error.log",
		"abc/src/logs",
		"src/subdir/main.c",
		"main.c",
		"src/main.h",
		"cached/file.tmp",
		"mycache/file.tmp",
		"swp",
		"file.swap",
		"app.logg",
		"app.Labc",
		"README.md",
		"src/docs/README.md",
		"env",
		"environment.txt",
		"app.spec.js",
		"test.js",
		"# This is a comment",
		"anything",
		"file.txt   ",
		"a.txt",
		"z.txt",
		"m.txt",
		"12.txt",
		"src/log",
		"file/debug.log",
		"app.js",
		"min.js",
		"appmin.js",
		"src",
		"logs/error.log",
		"debug.log",
		"src/debug.log",
		"important.txt",
		"\\!important.txt",
		"afile.txt",
		"xfile.txt",
		"t.txt",
		"myfile.txt",
		"my_file.txt",
		"src/build/",
		"tools/build/",
		"hidden",
		"src/file.txt",
		".gitignore",
		"git/",
		"archive.gz",
		"file.targz",
		"file-.debug.log",
		"file-12.debug.log",
		"file-a.log",
		"src/test.js",
		"src/app/unit/test.js",
		"src/fixtures",
		"test/fixtures",
		"a/src/test/fixtures",
		"src/test",
		"test/",
		"lib/src/test/",
		"!file.txt",
		"file1.txt",
		"#file1.txt",
		"a/doc/frotz/",
		"doc/frotz",
		"frotz.txt",
		"Debug/",
		"Debug2/",
		"Debug01/",
		"foo/bar.md",
		"bar/foo/file.txt",
		"foobar",
		"src/foo",
		"a/foo",
		"bar",
		"foo",
		"src/foo/bar",
	}

	b.Run("stable:matches", func(bb *testing.B) {
		m, e := NewMatcher(Options{Patterns: patterns})
		require.NoError(bb, e)
		bb.ResetTimer()
		for bb.Loop() {
			for _, path := range matching {
				_, _ = m.Match(context.Background(), path)
			}
		}
	})

	b.Run("stable:non-matches", func(bb *testing.B) {
		m, e := NewMatcher(Options{Patterns: patterns})
		require.NoError(bb, e)
		bb.ResetTimer()
		for bb.Loop() {
			for _, path := range nonMatching {
				_, _ = m.Match(context.Background(), path)

			}
		}
	})

	b.Run("llel:matches", func(bb *testing.B) {
		m, e := NewMatcher(Options{Patterns: patterns, Parallel: true})
		require.NoError(bb, e)
		bb.ResetTimer()
		for bb.Loop() {
			for _, path := range matching {
				_, _ = m.Match(context.Background(), path)
			}
		}
	})

	b.Run("llel:non-matches", func(bb *testing.B) {
		m, e := NewMatcher(Options{Patterns: patterns, Parallel: true})
		require.NoError(bb, e)
		bb.ResetTimer()
		for bb.Loop() {
			for _, path := range nonMatching {
				_, _ = m.Match(context.Background(), path)
			}
		}
	})
}

func isGitAvailable(t *testing.T) bool {
	t.Helper()
	cmd := exec.Command("git", "--version")
	if err := cmd.Run(); err != nil {
		var execErr *exec.ExitError
		if errors.As(err, &execErr) {
			switch execErr.ExitCode() {
			case 0:
				return true
			default:
				return false
			}
		}
	}
	return true
}

func matchesGitCheckIgnore(t *testing.T, path string, dir string) (bool, error) {
	t.Helper()

	cmd := exec.Command("git", "check-ignore", "--", path)
	cmd.Dir = dir
	out, cmdErr := cmd.CombinedOutput()
	var execErr *exec.ExitError
	if errors.As(cmdErr, &execErr) {
		switch execErr.ExitCode() {
		case 0:
			return true, nil
		case 1:
			return false, nil
		default:
			t.Logf("git check-ignore cmd failure: %s", out)
			return false, execErr
		}
	}
	cmdOut := strings.Trim(string(out), "\n")
	return cmdOut == path, cmdErr
}
