package marker_test

import (
	"fmt"
	"io"
	"log/slog"
	"testing"

	"github.com/KasonBraley/marker"
)

type productionCode struct {
	logger *slog.Logger
}

func newProduction() *productionCode {
	p := productionCode{
		logger: slog.New(marker.NewHandler(slog.NewTextHandler(io.Discard, nil))),
	}

	return &p
}

func (p *productionCode) functionUnderTest(x int) {
	if x%2 == 0 {
		p.logger.Info(fmt.Sprintf("x is even (x=%v)", x))
	}
	p.logger.Info(fmt.Sprintf("x is odd (x=%v)", x))
}

func TestMarkLogger(t *testing.T) {
	realCode := newProduction()

	mark := marker.Check("x is even")
	realCode.functionUnderTest(2)
	if err := mark.ExpectHit(); err != nil {
		t.Error(err)
	}

	tests := map[string]struct {
		markValue string
		value     int
	}{
		"even": {markValue: "x is even", value: 2},
		"odd":  {markValue: "x is odd", value: 3},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			mark := marker.Check(tt.markValue)
			realCode.functionUnderTest(tt.value)
			if err := mark.ExpectHit(); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestMarkLogger_ExpectError(t *testing.T) {
	t.Run("ExpectHit without Check should panic", func(t *testing.T) {
		defer func() {
			want := `mark: ExpectHit called without first calling Check`
			got := recover()
			if got == nil {
				t.Errorf("expected panic %q, but got nil", want)
			}
			if got != want {
				t.Errorf("expected panic message %q, got %v", want, got)
			}
		}()

		logger := slog.New(marker.NewHandler(slog.NewTextHandler(io.Discard, nil)))
		logger.Info("foo")
		_ = (marker.Mark{}).ExpectHit()
	})

	t.Run("No corresponding log message returns error", func(t *testing.T) {
		mark := marker.Check("foo")
		err := mark.ExpectHit()
		if err == nil {
			t.Error("expected error, got nil")
		}
		want := `mark "foo" not hit`
		if err.Error() != want {
			t.Errorf("want error message %q, got %v", want, err.Error())
		}
	})

	t.Run("Check without ExpectHit", func(t *testing.T) {
		defer func() {
			got := recover()
			if got == nil {
				t.Error("expected a panic")
			}
			want := `mark: mark name "foo2" should be nil, missing the corresponding ExpectHit call`
			if got != want {
				t.Errorf("expected %q, got %v", want, got)
			}
		}()

		logger := slog.New(marker.NewHandler(slog.NewTextHandler(io.Discard, nil)))
		_ = marker.Check("foo")
		_ = marker.Check("foo2")
		logger.Info("foo")
	})
}
