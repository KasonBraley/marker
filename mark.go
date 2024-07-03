// Package marker provides a [slog.Handler] and an associated API for
// implementing explicit code coverage marks for *linking* source code and tests together.
//
// In production code, you use your logger as normal, and can use it to say "this should be covered by a test".
// In test code, you can then assert that a _specific_ test covers a specific log line.
//
// The purpose of this is to help with test maintenance over time in larger projects. Large projects
// often have a lot of tests. Finding the tests for a specific piece of code, and vice versa, can be
// a challenge. This package provides a simple solution to that problem by leveraging your existing
// logger, and simply enabling the use of `grep` to search for a corresponding test. For example, if
// you see `logger.Debug("request sent, waiting on response")` in the code, you can grep for that log
// message and immediately find the test that goes with that code path.
package marker

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

type state struct {
	markName *string
	markHit  bool
}

// Stores the currently active mark and its hit count.
// State is not synchronized and assumes single threaded execution.
var globalState = state{}

type handler struct {
	h slog.Handler
}

// NewHandler returns a [slog.Handler] implementation to help trace tests to source code.
// In a test environment, reported by [testing.Testing], the [slog.Handler] returned records
// that a log message was hit.
//
// In a test, [Check] is used to say that the code under test should log a specific message. It
// returns a [Mark] where [Mark.ExpectHit] is expected to be called after the code under test
// is ran.
//
// In non-tests(i.e. normal code operation), this recording of [Mark]'s is a no-op.
func NewHandler(h slog.Handler) *handler {
	return &handler{h: h}
}

func (m *handler) Handle(ctx context.Context, r slog.Record) error {
	if testing.Testing() {
		recordMark(r.Message)
	}

	return m.h.Handle(ctx, r)
}

func (m *handler) Enabled(ctx context.Context, lvl slog.Level) bool {
	return m.h.Enabled(ctx, lvl)
}

func (m *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return m.h.WithAttrs(attrs)
}

func (m *handler) WithGroup(name string) slog.Handler {
	return m.h.WithGroup(name)
}

func recordMark(msg string) {
	if globalState.markName != nil {
		activeMark := *globalState.markName
		if strings.Contains(msg, activeMark) {
			globalState.markHit = true
		}
	}
}

type Mark struct {
	name string
}

// Check stores the given mark name in global state to be subsequently asserted it was hit
// with [Mark.ExpectHit].
//
// Check will panic if not used in a testing environment, as reported by [testing.Testing].
func Check(name string) Mark {
	if !testing.Testing() {
		panic("mark: marker.Check can only be used in tests")
	}

	if globalState.markName != nil {
		// This is possible to happen, due to misuse of the API. For instance, this would occur
		// if two [Check] calls are called in a row without a corresponding [Mark.ExpectHit] call.
		//
		// Like:
		// mark := marker.Check("foo")
		// mark2 := marker.Check("foo2")
		//
		panic(fmt.Sprintf("mark: mark name %q should be nil, missing the corresponding ExpectHit call", name))
	}

	if globalState.markHit {
		// This should never happen.
		panic(fmt.Sprintf("mark: hit count should be false for mark %q", name))
	}

	globalState.markName = &name
	return Mark{name: name}
}

// ExpectHit returns an error if the stored name on Mark was not hit. ExpectHit requires [Check]
// to have been called first with the mark name that you expect to have been logged in the function
// under test.
//
// ExpectHit will panic if not used in a testing environment, as reported by [testing.Testing].
func (m Mark) ExpectHit() error {
	if !testing.Testing() {
		panic("mark: ExpectHit can only be used in tests")
	}

	defer func() {
		globalState = state{}
	}()

	if globalState.markName == nil {
		// This occuring means incorrect use of the API. The [Check] function was not called first.
		panic("mark: ExpectHit called without first calling Check")
	}

	if globalState.markName != nil && *globalState.markName != m.name {
		// This should never happen.
		panic("mark: global state does not match the given Mark")
	}

	if !globalState.markHit {
		// This is the expected behavior if something went wrong.
		// Can be one of:
		// - The mark name in the test is wrong
		// - The mark name(log message) in the code under test is wrong
		// - Or in the real scenario this package is made for, the code under test was actually
		// not executed like it was expected to be.
		return fmt.Errorf("mark %q not hit", m.name)
	}

	return nil
}
