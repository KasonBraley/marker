# Marker

[![Go Reference](https://pkg.go.dev/badge/github.com/KasonBraley/marker.svg)](https://pkg.go.dev/github.com/KasonBraley/marker)

This package provides a [slog.Handler](https://pkg.go.dev/log/slog#Handler) and an associated API for
implementing explicit code coverage marks for _linking_ source code and tests together.

In production code, you use your logger as normal, and can use it to say "this should be covered by a test".
In test code, you can then assert that a _specific_ test covers a specific log line.

The purpose of this is to help with test maintenance over time in larger projects. Large projects
often have a lot of tests. Finding the tests for a specific piece of code, and vice versa, can be
a challenge. This package provides a simple solution to that problem by leveraging your existing
logger, and simply enabling the use of `grep` to search for a corresponding test. For example, if
you see `logger.Debug("request sent, waiting on response")` in the code, you can grep for that log
message and immediately find the test that goes with that code path.

The blog post that inspired this package goes over this testing technique and why it's useful in much
more detail. https://ferrous-systems.com/blog/coverage-marks/

This is not for "coverage". Coverage is the answer to the question "Is this tested?".
Marks answer "Why does this code need to exist?".

Inspired by:

- https://ferrous-systems.com/blog/coverage-marks/
- https://en.wikipedia.org/wiki/Requirements_traceability

Implementations of this concept in other languages:

- [Rust](https://crates.io/crates/cov-mark)

##### Example:

```go
package main

import (
    "fmt"
    "io"
    "log/slog"

    "github.com/KasonBraley/marker"
)

func main() {
    run()
}

func run() {
    logger := slog.New(marker.NewHandler(slog.NewTextHandler(io.Discard, nil)))
    svc := newService(logger)
    svc.isEven(2)
}

type service struct {
    logger *slog.Logger
}

func newService(logger *slog.Logger) *service {
    return &service{logger: logger}
}

func (s *service) isEven(x int) {
    if x%2 == 0 {
        s.logger.Info(fmt.Sprintf("x is even (x=%v)", x))
    }
    s.logger.Info(fmt.Sprintf("x is odd (x=%v)", x))
}
```

Corresponding test:

```go
func TestIsEven(t *testing.T) {
    svc := newService()

    t.Run("even", func(t *testing.T) {
        mark := marker.Check("x is even")
        svc.isEven(2)
        if err := mark.ExpectHit(); err != nil {
            t.Error(err)
        }
    })

    t.Run("odd", func(t *testing.T) {
        mark := marker.Check("x is even") // If we change this to "x is odd", it will pass.
        svc.isEven(3) // Odd number passed to show that we don't hit the expected mark.
        if err := mark.ExpectHit(); err != nil {
            t.Error(err) // The error `mark "x is even" not hit` is returned.
        }
    })
}
```
