# Go Path Ignore

Go library for path matching and filtering using multiple strategies: GitIgnore, Glob, and Regex patterns.

## Features

- **Multiple matching strategies** - Combine GitIgnore, Glob, and Regex patterns in a single matcher
- **High performance** - Powered by [go-re2](https://github.com/wasilibs/go-re2) for optimized regex operations
- **Pattern source tracking** - Identify which pattern matched a given path

## Use Cases

- Build tools that need to filter files (e.g., ignore patterns for file watchers)
- CLI applications with complex path filtering requirements
- Code analysis tools that need to skip certain directories or files
- Backup utilities with customizable exclusion rules

## Installation

```sh
go get github.com/vbhat161/go-path-ignore
```

## Quick Start

### Basic Example

```go
package main

import (
 "context"
 "fmt"
 "time"

 pathignore "github.com/vbhat161/go-path-ignore"
 "github.com/vbhat161/go-path-ignore/match/gitignore"
 "github.com/vbhat161/go-path-ignore/match/glob"
)

func main() {
 // Create a new PathIgnore instance with GitIgnore and Glob options
 opts := pathignore.Options{
  GitIgnore: &gitignore.Options{
   Patterns: []string{"*.txt"}, // Ignore any .txt files
  },
  
  Glob: &glob.Options{
   Paths: []string{"logs/**/*.log"}, // Ignore all .log files inside logs dir
  },
  
  Timeout:  100 * time.Millisecond, // Optional: set a timeout for match operations (default=1h)
  Parallel: true,                   // Optional: enable concurrent matching for better performance
 }

 pi, err := pathignore.New(opts)
 if err != nil {
  panic(err) // Handle initialization errors (invalid patterns, etc.)
 }

 // Check if "temp.txt" should be ignored
 matches, err := pi.Match(context.Background(), "temp.txt")
 if err != nil {
  panic(err) // Handle match errors (timeout, context cancellation, etc.)
 }

 fmt.Println(matches) // true

 // Check the source of the match
 matchInfo, err := pi.Match2(context.Background(), "temp.txt")
 if err != nil {
  panic(err)
 }

 fmt.Println(matchInfo.Ok())   // true
 fmt.Println(matchInfo.Src())  // *.txt
 fmt.Println(matchInfo.Type()) // gitignore
}
```

### API Methods

- **`Match(ctx, path)`** - Returns `true` if the path matches any pattern, `false` otherwise
- **`Match2(ctx, path)`** - Returns detailed match information including the matched pattern and strategy type

## Matching Strategies

You can use one or more matching strategies. Matchers are evaluated in order: **Regex → GitIgnore → Glob**. The first matcher that returns a positive match determines the outcome.

### GitIgnore Matching

This strategy follows the `.gitignore` specification.

```go
// Example using GitIgnore patterns
pi, err := pathignore.New(pathignore.Options{
 GitIgnore: &gitignore.Options{
  Patterns: []string{
   "*.log",          // Ignore all .log files
   "build/",         // Ignore the build directory
   "!important.log", // Do not ignore important.log
  },
  // FilePath: "/path/to/.gitignore", // Alternatively, load patterns from a .gitignore file
 },
})
```

### Glob Matching

This strategy uses standard glob patterns. The library uses [github.com/gobwas/glob](https://github.com/gobwas/glob) internally to match glob patterns.

```go
// Example using Glob patterns
pi, err := pathignore.New(pathignore.Options{
 Glob: &glob.Options{
  Patterns: []string{
   "*.tmp",   // Ignore all .tmp files
   "data/**", // Ignore everything inside the data directory
  },
  // RawPatterns: []string{"foo/bar"}, // Use RawPatterns for literal strings that should be quoted
 },
})
```

### Regex Matching

This strategy uses regular expressions powered by RE2.

```go
import "github.com/vbhat161/go-path-ignore/match/regex"

// Example using Regex patterns
pi, err := pathignore.New(pathignore.Options{
 Regex: &regex.Options{
  Patterns: []string{
   `\.bak$`,   // Ignore files ending with .bak
   `^vendor/`, // Ignore the vendor directory
  },
  Literals: false, // Set to true if patterns are literal strings to be escaped
 },
})
```

### Combining Strategies

Combine multiple strategies for flexible matching:

```go
import (
 pathignore "github.com/vbhat161/go-path-ignore"
 "github.com/vbhat161/go-path-ignore/match/gitignore"
 "github.com/vbhat161/go-path-ignore/match/regex"
)

// Example combining Regex and GitIgnore
pi, err := pathignore.New(pathignore.Options{
 Regex: &regex.Options{
  Patterns: []string{`\.DS_Store$`},
 },
 GitIgnore: &gitignore.Options{
  Patterns: []string{"node_modules/", "*.log"},
 },
 Timeout: 500 * time.Millisecond,
})

if err != nil {
 panic(err) // Handle initialization errors
}

ctx := context.Background()

// Check paths
pi.Match(ctx, ".DS_Store")                 // true (matched by Regex)
pi.Match(ctx, "node_modules/package.json") // true (matched by GitIgnore)
pi.Match(ctx, "app.log")                   // true (matched by GitIgnore)
pi.Match(ctx, "main.go")                   // false
```

## Configuration

### Timeout

Set a global timeout for matching operations. Defaults to 1 hour if not specified.

```go
pi, err := pathignore.New(pathignore.Options{
 GitIgnore: &gitignore.Options{
  Patterns: []string{"*.log"},
 },
 Timeout: 10 * time.Millisecond, // Set a 10ms timeout
})

if err != nil {
 panic(err)
}

ctx := context.Background()
match, err := pi.Match(ctx, "some/path")
if err != nil {
 // Handle timeout or context cancellation errors
 panic(err)
}
```

### Parallel Matching

Enable parallel matching for improved performance when working with many patterns. Uses RE2 Set for concurrent pattern matching across strategies.

```go
opts := pathignore.Options{
 GitIgnore: &gitignore.Options{
  Patterns: []string{"*.log", "*.tmp", "build/"},
 },
 Glob: &glob.Options{
  Paths: []string{"data/**", "cache/**"},
 },
 Parallel: true, // Enable parallel matching
}

pi, err := pathignore.New(opts)
if err != nil {
 panic(err)
}
```

## Performance

Benchmark results on Apple M1 Max ran on 30 input values against 40 patterns across matchers:

```plaintext
goos: darwin
goarch: arm64
pkg: github.com/vbhat161/go-path-ignore
cpu: Apple M1 Max
Benchmark/sequential-10              3926     305806 ns/op   158992 B/op     4169 allocs/op
Benchmark/parallel-10                6447     162454 ns/op    59819 B/op     1179 allocs/op
PASS
```

Parallel mode shows ~2x throughput improvement and reduced memory allocations.

## Configuration Options Reference

| Option | Type | Description | Default |
|--------|------|-------------|---------|
| `Regex` | `*regex.Options` | Regular expression patterns using RE2 | `nil` |
| `GitIgnore` | `*gitignore.Options` | GitIgnore-style patterns | `nil` |
| `Glob` | `*glob.Options` | Glob patterns | `nil` |
| `Timeout` | `time.Duration` | Global timeout for match operations | 1 hour |
| `Parallel` | `bool` | Enable concurrent matching across strategies | `false` |

**Note:** At least one matching strategy (Regex, GitIgnore, or Glob) must be provided.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
