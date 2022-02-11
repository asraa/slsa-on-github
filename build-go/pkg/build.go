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
	errorInvalidEnvArgument        = errors.New("invalid env passed via argument")
	errorEnvVariableNameNotAllowed = errors.New("env variable not allowed")
	errorInvalidFilename           = errors.New("invalid filename")
)

// TODO: move to an allowedArgs list.
var disallowedArgs = map[string]bool{
	// Allows exucting another script/commmand on the machine.
	"-toolexec": true,
	// Allows overwriting existing files on the machine.
	"-o": true,
	// Allows turning off vendoring/hermeticity.
	// See https://golang.org/ref/mod#build-commands.
	"-mod=": true,
}

type GoBuild struct {
	cfg      *GoReleaserConfig
	goc      string
	flags    []string
	ldflags  string
	filename string
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

	return syscall.Exec(b.goc, b.flags, os.Environ())
}

func (b *GoBuild) SetEnvVariables(envs string) error {
	if err := os.Setenv("GOOS", b.cfg.Goos); err != nil {
		return fmt.Errorf("os.Setenv: %w", err)
	}

	if err := os.Setenv("GOARCH", b.cfg.Goarch); err != nil {
		return fmt.Errorf("os.Setenv: %w", err)
	}

	ees := os.Environ()
	for k, v := range b.cfg.Env {
		if !isAllowedEnvVariable(k, ees) {
			return fmt.Errorf("%w: %s", errorEnvVariableNameNotAllowed, v)
		}

		if err := os.Setenv(k, v); err != nil {
			return fmt.Errorf("os.Setenv: %w", err)
		}
	}

	// Add additional environment variables encoded as argument.
	// I've tried running the re-usable workflow in a step
	// and set the env variable in a previous step, but found that a re-usable workflow is not
	// allowed to run in a step; they have to run as `job.uses`. Using `job.env` with `job.uses`
	// is not allowed. So for now we need this additional variable.
	for _, e := range strings.Split(envs, ",") {
		s := strings.Trim(e, " ")
		if len(s) == 0 {
			continue
		}
		sp := strings.Split(s, ":")
		if len(sp) != 2 {
			return fmt.Errorf("%w: %s", errorInvalidEnvArgument, s)
		}
		name := strings.Trim(sp[0], " ")
		value := strings.Trim(sp[1], " ")
		if !isAllowedEnvVariable(name, ees) {
			return fmt.Errorf("%w: %s", errorEnvVariableNameNotAllowed, name)
		}

		fmt.Printf("arg env: %s:%s\n", name, value)
		if err := os.Setenv(name, value); err != nil {
			return fmt.Errorf("os.Setenv: %w", err)
		}
	}
	return nil
}

func (b *GoBuild) SetOutputFilename(name string) error {
	const alpha = "abcdefghijklmnopqrstuvwxyz1234567890-_"

	for _, char := range name {
		if !strings.Contains(alpha, strings.ToLower(string(char))) {
			return fmt.Errorf("%w: found character '%c'", errorInvalidFilename, char)
		}
	}

	b.filename = name

	return nil
}

// TODO: set -x flag to display the command used.
func (b *GoBuild) SetFlags(flags []string) error {
	b.flags = []string{b.goc, "build", "-mod=vendor", "-o", b.filename}

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
		// TODO: use strings.HasPrefix with allowedList?
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
func (b *GoBuild) SetLdflags(ldflags []string) error {
	var a []string
	regex := regexp.MustCompile(`{{\s*\.Env\.(.*)\s*}}`)

	for _, v := range ldflags {
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
