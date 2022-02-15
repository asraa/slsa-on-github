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

func TestFilenameAllowed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		filename string
		goos     string
		goarch   string
		argEnvs  string
		envs     string
		expected struct {
			err error
			fn  string
		}
	}{
		{
			name:     "valid filename",
			filename: "../filename",
			expected: struct {
				err error
				fn  string
			}{
				err: errorInvalidFilename,
			},
		},
		{
			name:     "valid filename",
			filename: "",
			expected: struct {
				err error
				fn  string
			}{
				err: errorEmptyFilename,
			},
		},
		{
			name:     "filename arch",
			filename: "name-{{ .Arch }}",
			expected: struct {
				err error
				fn  string
			}{
				err: errorEnvVariableNameEmpty,
			},
		},
		{
			name:     "filename os",
			filename: "name-{{ .OS }}",
			expected: struct {
				err error
				fn  string
			}{
				err: errorEnvVariableNameEmpty,
			},
		},
		{
			name:     "filename os",
			filename: "$bla",
			expected: struct {
				err error
				fn  string
			}{
				err: errorInvalidFilename,
			},
		},
		{
			name:     "filename os",
			filename: "name-{{ .OS }}",
			expected: struct {
				err error
				fn  string
			}{
				err: errorEnvVariableNameEmpty,
			},
		},
		{
			name:     "filename linux os",
			filename: "name-{{ .OS }}",
			goos:     "linux",
			expected: struct {
				err error
				fn  string
			}{
				err: nil,
				fn:  "name-linux",
			},
		},
		{
			name:     "filename amd64 arch",
			filename: "name-{{ .Arch }}",
			goarch:   "amd64",
			expected: struct {
				err error
				fn  string
			}{
				err: nil,
				fn:  "name-amd64",
			},
		},
		{
			name:     "filename amd64/linux arch",
			filename: "name-{{ .OS }}-{{ .Arch }}",
			goarch:   "amd64",
			goos:     "linux",
			expected: struct {
				err error
				fn  string
			}{
				err: nil,
				fn:  "name-linux-amd64",
			},
		},
		{
			name:     "filename invalid arch",
			filename: "name-{{ .Arch }}",
			goarch:   "something/../../",
			expected: struct {
				err error
				fn  string
			}{
				err: errorInvalidFilename,
			},
		},
		{
			name:     "filename invalid not supported",
			filename: "name-{{ .Bla }}",
			goarch:   "something/../../",
			expected: struct {
				err error
				fn  string
			}{
				err: errorInvalidFilename,
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := goReleaserConfigFile{
				Binary:  tt.filename,
				Version: 1,
				Goos:    tt.goos,
				Goarch:  tt.goarch,
			}
			c, err := fromConfig(&cfg)
			if err != nil {
				t.Errorf("fromConfig: %v", err)
			}
			b := GoBuildNew("go compiler", c)

			fn, err := b.generateOutputFilename()
			if !errCmp(err, tt.expected.err) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}

			if fn != tt.expected.fn {
				t.Errorf(cmp.Diff(fn, tt.expected.fn))
			}
		})
	}
}
