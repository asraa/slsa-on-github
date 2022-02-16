package pkg

import (
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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
				t.Errorf(cmp.Diff(err, tt.expected.err))
			}

			if err != nil {
				return
			}

			if fn != tt.expected.fn {
				t.Errorf(cmp.Diff(fn, tt.expected.fn))
			}
		})
	}
}

func TestArgEnvVariables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		argEnv   string
		expected struct {
			err error
			env map[string]string
		}
	}{
		{
			name:   "valid arg envs",
			argEnv: "VAR1:value1, VAR2:value2",
			expected: struct {
				err error
				env map[string]string
			}{
				err: nil,
				env: map[string]string{"VAR1": "value1", "VAR2": "value2"},
			},
		},
		{
			name:   "empty arg envs",
			argEnv: "",
			expected: struct {
				err error
				env map[string]string
			}{
				err: nil,
				env: map[string]string{},
			},
		},
		{
			name:   "valid arg envs not space",
			argEnv: "VAR1:value1,VAR2:value2",
			expected: struct {
				err error
				env map[string]string
			}{
				err: nil,
				env: map[string]string{"VAR1": "value1", "VAR2": "value2"},
			},
		},
		{
			name:   "invalid arg empty 2 values",
			argEnv: "VAR1:value1,",
			expected: struct {
				err error
				env map[string]string
			}{
				err: errorInvalidEnvArgument,
			},
		},
		{
			name:   "invalid arg empty 3 values",
			argEnv: "VAR1:value1,, VAR3:value3",
			expected: struct {
				err error
				env map[string]string
			}{
				err: errorInvalidEnvArgument,
			},
		},
		{
			name:   "invalid arg uses equal",
			argEnv: "VAR1=value1",
			expected: struct {
				err error
				env map[string]string
			}{
				err: errorInvalidEnvArgument,
			},
		},
		{
			name:   "valid single arg",
			argEnv: "VAR1:value1",
			expected: struct {
				err error
				env map[string]string
			}{
				err: nil,
				env: map[string]string{"VAR1": "value1"},
			},
		},
		{
			name:   "invalid valid single arg with empty",
			argEnv: "VAR1:value1:",
			expected: struct {
				err error
				env map[string]string
			}{
				err: errorInvalidEnvArgument,
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := goReleaserConfigFile{
				Version: 1,
			}
			c, err := fromConfig(&cfg)
			if err != nil {
				t.Errorf("fromConfig: %v", err)
			}
			b := GoBuildNew("go compiler", c)

			err = b.SetArgEnvVariables(tt.argEnv)
			if !errCmp(err, tt.expected.err) {
				t.Errorf(cmp.Diff(err, tt.expected.err))
			}

			if err != nil {
				return
			}

			sorted := cmpopts.SortSlices(func(a, b string) bool { return a < b })
			if !cmp.Equal(b.argEnv, tt.expected.env, sorted) {
				t.Errorf(cmp.Diff(b.argEnv, tt.expected.env))
			}
		})
	}
}

func TestEnvVariables(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		goos     string
		goarch   string
		env      []string
		expected struct {
			err   error
			flags []string
		}
	}{
		{
			name:   "empty flags",
			goos:   "linux",
			goarch: "x86",
			expected: struct {
				err   error
				flags []string
			}{
				flags: []string{"GOOS=linux", "GOARCH=x86"},
				err:   nil,
			},
		},
		{
			name:   "empty goos",
			goarch: "x86",
			expected: struct {
				err   error
				flags []string
			}{
				flags: []string{},
				err:   errorEnvVariableNameEmpty,
			},
		},
		{
			name: "empty goarch",
			goos: "windows",
			expected: struct {
				err   error
				flags []string
			}{
				flags: []string{},
				err:   errorEnvVariableNameEmpty,
			},
		},
		{
			name:   "invalid flags",
			goos:   "windows",
			goarch: "amd64",
			env:    []string{"VAR1=value1", "VAR2=value2"},
			expected: struct {
				err   error
				flags []string
			}{
				err: errorEnvVariableNameNotAllowed,
			},
		},
		{
			name:   "invalid flags",
			goos:   "windows",
			goarch: "amd64",
			env:    []string{"GOVAR1=value1", "GOVAR2=value2", "CGO_VAR1=val1", "CGO_VAR2=val2"},
			expected: struct {
				err   error
				flags []string
			}{
				flags: []string{
					"GOOS=windows", "GOARCH=amd64",
					"GOVAR1=value1", "GOVAR2=value2", "CGO_VAR1=val1", "CGO_VAR2=val2",
				},
				err: nil,
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := goReleaserConfigFile{
				Version: 1,
				Goos:    tt.goos,
				Goarch:  tt.goarch,
				Env:     tt.env,
			}
			c, err := fromConfig(&cfg)
			if err != nil {
				t.Errorf("fromConfig: %v", err)
			}
			b := GoBuildNew("go compiler", c)

			flags, err := b.generateEnvVariables()

			if !errCmp(err, tt.expected.err) {
				t.Errorf(cmp.Diff(err, tt.expected.err))
			}
			if err != nil {
				return
			}
			// Note: generated env variables contain the process's env variables too.
			expectedFlags := append(os.Environ(), tt.expected.flags...)
			sorted := cmpopts.SortSlices(func(a, b string) bool { return a < b })
			if !cmp.Equal(flags, expectedFlags, sorted) {
				t.Errorf(cmp.Diff(flags, expectedFlags))
			}
		})
	}
}

func TestGenerateLdflags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		argEnv   string
		ldflags  []string
		expected struct {
			err     error
			ldflags string
		}
	}{
		{
			name:    "version ldflags",
			argEnv:  "VERSION_LDFLAGS:value1",
			ldflags: []string{"{{ .Env.VERSION_LDFLAGS }}"},
			expected: struct {
				err     error
				ldflags string
			}{
				ldflags: "value1",
				err:     nil,
			},
		},
		{
			name:    "one value with text",
			argEnv:  "VAR1:value1, VAR2:value2",
			ldflags: []string{"name-{{ .Env.VAR1 }}"},
			expected: struct {
				err     error
				ldflags string
			}{
				ldflags: "name-value1",
				err:     nil,
			},
		},
		{
			name:    "two values with text",
			argEnv:  "VAR1:value1, VAR2:value2",
			ldflags: []string{"name-{{ .Env.VAR1 }}-{{ .Env.VAR2 }}"},
			expected: struct {
				err     error
				ldflags string
			}{
				ldflags: "name-value1-value2",
				err:     nil,
			},
		},
		{
			name:    "two values with text and not space between env",
			argEnv:  "VAR1:value1,VAR2:value2",
			ldflags: []string{"name-{{ .Env.VAR1 }}-{{ .Env.VAR2 }}"},
			expected: struct {
				err     error
				ldflags string
			}{
				ldflags: "name-value1-value2",
				err:     nil,
			},
		},
		{
			name:    "same two values with text",
			argEnv:  "VAR1:value1, VAR2:value2",
			ldflags: []string{"name-{{ .Env.VAR1 }}-{{ .Env.VAR1 }}"},
			expected: struct {
				err     error
				ldflags string
			}{
				ldflags: "name-value1-value1",
				err:     nil,
			},
		},
		{
			name:    "same value extremeties",
			argEnv:  "VAR1:value1, VAR2:value2",
			ldflags: []string{"{{ .Env.VAR1 }}-name-{{ .Env.VAR1 }}"},
			expected: struct {
				err     error
				ldflags string
			}{
				ldflags: "value1-name-value1",
				err:     nil,
			},
		},
		{
			name:    "two different value extremeties",
			argEnv:  "VAR1:value1, VAR2:value2",
			ldflags: []string{"{{ .Env.VAR1 }}-name-{{ .Env.VAR2 }}"},
			expected: struct {
				err     error
				ldflags string
			}{
				ldflags: "value1-name-value2",
				err:     nil,
			},
		},
		{
			name:    "undefined env variable",
			argEnv:  "VAR2:value2",
			ldflags: []string{"{{ .Env.VAR1 }}-name-{{ .Env.VAR1 }}"},
			expected: struct {
				err     error
				ldflags string
			}{
				err: errorEnvVariableNameEmpty,
			},
		},
		{
			name:    "undefined env variable 1",
			argEnv:  "VAR2:value2",
			ldflags: []string{"{{ .Env.VAR2 }}-name-{{ .Env.VAR1 }}"},
			expected: struct {
				err     error
				ldflags string
			}{
				err: errorEnvVariableNameEmpty,
			},
		},
		{
			name:    "empty env variable",
			argEnv:  "",
			ldflags: []string{"{{ .Env.VAR1 }}-name-{{ .Env.VAR1 }}"},
			expected: struct {
				err     error
				ldflags string
			}{
				err: errorEnvVariableNameEmpty,
			},
		},
	}

	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := goReleaserConfigFile{
				Version: 1,
				Ldflags: tt.ldflags,
			}
			c, err := fromConfig(&cfg)
			if err != nil {
				t.Errorf("fromConfig: %v", err)
			}
			b := GoBuildNew("go compiler", c)

			err = b.SetArgEnvVariables(tt.argEnv)
			if err != nil {
				t.Errorf("SetArgEnvVariables: %v", err)
			}
			ldflags, err := b.generateLdflags()

			if !errCmp(err, tt.expected.err) {
				t.Errorf(cmp.Diff(err, tt.expected.err))
			}
			if err != nil {
				return
			}
			// Note: generated env variables contain the process's env variables too.
			if !cmp.Equal(ldflags, tt.expected.ldflags) {
				t.Errorf(cmp.Diff(ldflags, tt.expected.ldflags))
			}
		})
	}
}

func TestGenerateFlags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flags    []string
		expected error
	}{
		{
			name:     "valid flags",
			flags:    []string{"-race", "-x"},
			expected: nil,
		},
		{
			name:     "invalid -mod flags",
			flags:    []string{"-mod=whatever", "-x"},
			expected: errorUnsupportedArguments,
		},
		{
			name: "invalid random flags",
			flags: []string{
				"-a", "-race", "-msan", "-asan",
				"-v", "-x", "-buildinfo", "-buildmode",
				"-buildvcs", "-compiler", "-gccgoflags",
				"-gcflags", "-ldflags", "-linkshared",
				"-tags", "-trimpath", "bla",
			},
			expected: errorUnsupportedArguments,
		},
		{
			name: "valid all flags",
			flags: []string{
				"-a", "-race", "-msan", "-asan",
				"-v", "-x", "-buildinfo", "-buildmode",
				"-buildvcs", "-compiler", "-gccgoflags",
				"-gcflags", "-ldflags", "-linkshared",
				"-tags", "-trimpath",
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		tt := tt // Re-initializing variable so it is not changed while executing the closure below
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cfg := goReleaserConfigFile{
				Version: 1,
				Flags:   tt.flags,
			}
			c, err := fromConfig(&cfg)
			if err != nil {
				t.Errorf("fromConfig: %v", err)
			}
			b := GoBuildNew("gocompiler", c)

			flags, err := b.generateFlags()
			expectedFlags := append([]string{"gocompiler", "build", "-mod=vendor"}, tt.flags...)
			fmt.Println(err)
			if !errCmp(err, tt.expected) {
				t.Errorf(cmp.Diff(err, tt.expected))
			}
			if err != nil {
				return
			}
			// Note: generated env variables contain the process's env variables too.
			sorted := cmpopts.SortSlices(func(a, b string) bool { return a < b })
			if !cmp.Equal(flags, expectedFlags, sorted) {
				t.Errorf(cmp.Diff(flags, expectedFlags))
			}
		})
	}
}
