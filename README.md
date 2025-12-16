# Go Path Ignore Utility

This utility will help you ignore all files and directories in a Go project. The goal is to create a `.gitignore` file that includes all the common Go project files and directories.

## Installation

```sh
go get github.com/VishwaBhat/go-path-ignore
```

## Usage

The `go-path-ignore` utility provides a flexible way to define ignore rules using different matching strategies: GitIgnore, Glob, and Regex.

### Basic Example

Here's a simple example demonstrating how to use the utility to ignore a specific file:

```go
package main

import (
	"fmt"
	"time"

	pathignore "github.com/VishwaBhat/go-path-ignore"
	"github.com/VishwaBhat/go-path-ignore/match/gitignore"
	"github.com/VishwaBhat/go-path-ignore/match/glob"
)

func main() {
	// Create a new PathIgnore instance with GitIgnore options
	pi, err := pathignore.New(pathignore.Options{
		GitIgnore: &gitignore.Options{
			Patterns: []string{"*.txt"}, // Ignore any .txt files
		},
    Glob: &glob.Options{
			Patterns: []string{"logs/**/*.log"}, // Ignore all .log files inside logs dir
    }
		Timeout: 100 * time.Millisecond, // Optional: set a timeout for overall operation
	})
	if err != nil {
		fmt.Printf("Error creating PathIgnore: %v\n", err)
		return
	}

	// Check if "temp.txt" should be ignored
	shouldIgnore, err := pi.ShouldIgnore("temp.txt")
	if err != nil {
		fmt.Printf("Error checking path: %v\n", err)
		return
	}

  fmt.Println("ignore temp.txt", shouldIgnore) // true
}
```

### Matching Strategies

You can combine different matching strategies by providing options for `Regex`, `Glob`, and `GitIgnore` when creating a `PathIgnore` instance. The order in which matchers are added (Regex, then Glob, then GitIgnore) determines their precedence if a path matches multiple rules. The first matcher that returns a positive match will determine the outcome.

#### GitIgnore Matching

This strategy follows the `.gitignore` specification.

```go
// Example using GitIgnore patterns
pi, err := pathignore.NewPathIgnore(pathignore.IgnoreOptions{
	GitIgnore: &gitignore.Options{
		Patterns: []string{
			"*.log",        // Ignore all .log files
			"build/",       // Ignore the build directory
			"!important.log", // Do not ignore important.log
		},
		// FilePath: ".gitignore", // Alternatively, load patterns from a .gitignore file
		Parallel: true, // Optional: enable parallel matching for performance
	},
})
```

#### Glob Matching

This strategy uses standard glob patterns.

```go
// Example using Glob patterns
pi, err := pathignore.NewPathIgnore(pathignore.IgnoreOptions{
	Glob: &glob.Options{
		Paths: []string{
			"*.tmp",      // Ignore all .tmp files
			"data/**",    // Ignore everything inside the data directory
		},
		// RawPaths: []string{"foo/bar"}, // Use RawPaths for literal strings that should be quoted
		Parallel: true, // Optional: enable parallel matching for performance
	},
})
```

#### Regex Matching

This strategy uses regular expressions.

```go
// Example using Regex patterns
pi, err := pathignore.NewPathIgnore(pathignore.IgnoreOptions{
	Regex: &regex.Options{
		Patterns: []string{
			`\.bak$`,         // Ignore files ending with .bak
			`^vendor/`,       // Ignore the vendor directory
		},
		Literals: false, // Set to true if patterns are literal strings to be escaped
		Parallel: true,  // Optional: enable parallel matching for performance
	},
})
```

### Combining Strategies

You can combine multiple strategies. The `PathIgnore` utility will evaluate them in the order they are defined in the `IgnoreOptions` struct (Regex, Glob, GitIgnore).

```go
// Example combining GitIgnore and Regex
pi, err := pathignore.NewPathIgnore(pathignore.IgnoreOptions{
	Regex: &regex.Options{
		Patterns: []string{`\.DS_Store$`},
	},
	GitIgnore: &gitignore.Options{
		Patterns: []string{"node_modules/", "*.log"},
	},
	Timeout: 500 * time.Millisecond,
})

if err != nil {
	// Handle error
}

// Check paths
pi.ShouldIgnore(".DS_Store")    // true (matched by Regex)
pi.ShouldIgnore("node_modules/package.json") // true (matched by GitIgnore)
pi.ShouldIgnore("app.log")      // true (matched by GitIgnore)
pi.ShouldIgnore("main.go")      // false
```

### Timeout

You can specify a global timeout for all matching operations. If a match takes longer than the specified timeout, the operation will be cancelled, and an error will be returned.

```go
pi, err := pathignore.NewPathIgnore(pathignore.IgnoreOptions{
	GitIgnore: &gitignore.Options{
		Patterns: []string{"long_running_pattern_that_might_timeout"},
	},
	Timeout: 10 * time.Millisecond, // Set a 10ms timeout
})
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
