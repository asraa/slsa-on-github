package pkg

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestAllowedEnvVariable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		variable string
		expected bool
	}{
		{
			name:     "BLA variable",
			variable: "BLA",
			expected: false,
		},
		{
			name:     "random variable",
			variable: "random",
			expected: false,
		},
		{
			name:     "GOSOMETHING variable",
			variable: "GOSOMETHING",
			expected: true,
		},
		{
			name:     "CGO_SOMETHING variable",
			variable: "CGO_SOMETHING",
			expected: true,
		},
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := isAllowedEnvVariable(tt.variable)
			if !cmp.Equal(r, tt.expected) {
				t.Errorf(cmp.Diff(r, tt.expected))
			}
		})
	}
}

func TestAllowedArgument(t *testing.T) {
	t.Parallel()

	var tests []struct {
		name     string
		argument string
		expected bool
	}

	for k, _ := range allowedBuildArgs {
		tests = append(tests, struct {
			name     string
			argument string
			expected bool
		}{
			name:     fmt.Sprintf("%s argument", k),
			argument: k,
			expected: true,
		})

		tests = append(tests, struct {
			name     string
			argument string
			expected bool
		}{
			name:     fmt.Sprintf("%sbla argument", k),
			argument: fmt.Sprintf("%sbla", k),
			expected: true,
		})

		tests = append(tests, struct {
			name     string
			argument string
			expected bool
		}{
			name:     fmt.Sprintf("bla %s argument", k),
			argument: fmt.Sprintf("bla%s", k),
			expected: false,
		})

		tests = append(tests, struct {
			name     string
			argument string
			expected bool
		}{
			name:     fmt.Sprintf("space %s argument", k),
			argument: fmt.Sprintf(" %s", k),
			expected: false,
		})
	}
	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			r := isAllowedArg(tt.argument)
			if !cmp.Equal(r, tt.expected) {
				t.Errorf(cmp.Diff(r, tt.expected))
			}
		})
	}
}
