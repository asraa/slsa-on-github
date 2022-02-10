package pkg

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"syscall"
)

var (
	errorEnvVariableNameEmpty      = errors.New("env variable empty or not set")
	errorUnsupportedArguments      = errors.New("argument not supported")
	errorEnvVariableNameNotAllowed = errors.New("env variable not allowed")
)

var disallowedArgs = map[string]bool{
	// Allows exucting another script/commmand on the machine.
	"-toolexec": true,
	// Allows overwriting existing files on the machine.
	"-o": true,
}

type GoBuild struct {
	cfg     *GoReleaserConfig
	goc     string
	flags   []string
	ldflags string
}

func GoBuildNew(goc string, cfg *GoReleaserConfig) *GoBuild {
	c := GoBuild{
		cfg: cfg,
		goc: goc,
	}
	return &c
}

func (b *GoBuild) Run() error {
	if len(b.ldflags) > 0 {
		b.flags = append(b.flags, "-ldflags", b.ldflags)
	}
	fmt.Println("ldflags:", b.ldflags)
	fmt.Println("flags:", b.flags)
	fmt.Println("env:", os.Environ())
	return syscall.Exec(b.goc, b.flags, os.Environ())
}

func (b *GoBuild) SetEnvVariables() error {
	if err := os.Setenv("GOOS", b.cfg.Goos); err != nil {
		return fmt.Errorf("os.Setenv: %w", err)
	}

	if err := os.Setenv("GOARCH", b.cfg.Goarch); err != nil {
		return fmt.Errorf("os.Setenv: %w", err)
	}

	envs := os.Environ()
	for k, v := range b.cfg.Env {
		if !isAllowedEnvVariable(k, envs) {
			return fmt.Errorf("%w: %s", errorEnvVariableNameNotAllowed, v)
		}

		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("os.Setenv: %w", err)
		}
	}
	return nil
}

func (b *GoBuild) SetFlags(flags []string) error {
	b.flags = []string{b.goc, "build"}

	for _, v := range flags {
		if !isAllowedArg(v) {
			return fmt.Errorf("%w: %s", errorUnsupportedArguments, v)
		}
		b.flags = append(b.flags, v)

	}
	return nil
}

func isAllowedArg(arg string) bool {
	for k, _ := range disallowedArgs {
		if strings.Contains(arg, k) {
			return false
		}
	}
	return true
}

// Check if the env variable the use wants to set already exists
// Note: Probably we would relax this in practice, and maybe specifically
// look for some names like PATH.
func isAllowedEnvVariable(name string, disallowedEnvs []string) bool {
	for _, e := range disallowedEnvs {
		v := strings.Trim(e, " ")
		if strings.HasPrefix(v, fmt.Sprintf("%s=", name)) {
			return false
		}
	}
	return true
}

// TODO: maybe not needed if handled directly by go compiler.
func (b *GoBuild) SetLdflags(flags []string) error {
	var a []string
	regex := regexp.MustCompile(`{{\s*\.Env\.(.*)\s*}}`)

	for _, v := range flags {
		var res string
		m := regex.FindStringSubmatch(v)
		// fmt.Println("match", m[1])
		if len(m) > 2 {
			return fmt.Errorf("%w: %s", errorEnvVariableNameEmpty, v)
		}
		if len(m) == 2 {
			name := strings.Trim(m[1], " ")
			val, exists := os.LookupEnv(name)
			if !exists {
				return fmt.Errorf("%w: %s", errorEnvVariableNameEmpty, name)
			}
			res = val
		} else {
			res = v
		}
		a = append(a, res)
	}
	if len(a) > 0 {
		b.ldflags = fmt.Sprintf("'%s'", strings.Join(a, " "))
	}
	return nil
}
