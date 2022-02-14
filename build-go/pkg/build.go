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
	errEmptyFilename               = errors.New("filename is not set")
)

// See `go build help`.
// `-asmflags`, `-n`, `-mod`, `-installsuffix`, `-modfile`,
// `-workfile`, `-overlay`, `-pkgdir`, `-toolexec`, `-o`,
// `-modcacherw` not in list for now.

var allowedBuildArgs = map[string]bool{
	"-a": true, "-race": true, "-msan": true, "-asan": true,
	"-v": true, "-work": true, "-x": true, "-buildinfo": true,
	"-buildmode": true, "-buildvcs": true, "-compiler": true,
	"-gccgoflags": true, "-gcflags": true,
	"-ldflags": true, "-linkshared": true, "-mod": true,
	"-tags": true, "-trimpath": true,
}

var allowedEnvVariablePrefix = map[string]bool{
	"GO": true, "CGO_": true,
}

type GoBuild struct {
	cfg *GoReleaserConfig
	goc string
	// flags []string
	// Env variable passed via workflow, which are dynamically computed.
	// Static env variables are contained in cfg.Env.
	argEnv   map[string]string
	ldflags  string
	filename string
}

func GoBuildNew(goc string, cfg *GoReleaserConfig) *GoBuild {
	c := GoBuild{
		cfg:    cfg,
		goc:    goc,
		argEnv: make(map[string]string),
	}

	return &c
}

func (b *GoBuild) Run() error {
	// Set flags.
	flags, err := b.generateFlags()
	if err != nil {
		return err
	}

	// Generate env variables.
	envs, err := b.generateEnvVariables()
	if err != nil {
		return err
	}

	// Generate ldflags.
	ldflags, err := b.generateLdflags()
	if err != nil {
		return err
	}

	// Add ldflags.
	if len(ldflags) > 0 {
		flags = append(flags, fmt.Sprintf("-ldflags=%s", ldflags))
	}

	// Set filename as last argument.
	if b.filename == "" {
		return errEmptyFilename
	}
	flags = append(flags, []string{"-o", b.filename}...)
	fmt.Println(flags)
	fmt.Println(envs)
	return syscall.Exec(b.goc, flags, envs)
}

func (b *GoBuild) generateEnvVariables() ([]string, error) {
	env := os.Environ()

	if b.cfg.Goos == "" {
		return nil, fmt.Errorf("%w: %s", errorEnvVariableNameEmpty, "GOOS")
	}
	env = append(env, fmt.Sprintf("GOOS=%s", b.cfg.Goos))

	if b.cfg.Goarch == "" {
		return nil, fmt.Errorf("%w: %s", errorEnvVariableNameEmpty, "GOARCH")
	}
	env = append(env, fmt.Sprintf("GOARCH=%s", b.cfg.Goarch))

	// Set env variables from config file.
	for k, v := range b.cfg.Env {
		if !isAllowedEnvVariable(k) {
			return env, fmt.Errorf("%w: %s", errorEnvVariableNameNotAllowed, v)
		}

		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return env, nil
}

func (b *GoBuild) SetArgEnvVariables(envs string) error {
	// Notes:
	// - I've tried running the re-usable workflow in a step
	// and set the env variable in a previous step, but found that a re-usable workflow is not
	// allowed to run in a step; they have to run as `job.uses`. Using `job.env` with `job.uses`
	// is not allowed.
	// - We don't want to allow env variables set in the workflow because of injections
	// e.g. LD_PRELOAD, etc.
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

		fmt.Printf("arg env: %s:%s\n", name, value)
		b.argEnv[name] = value

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

func (b *GoBuild) generateFlags() ([]string, error) {
	// -x
	flags := []string{b.goc, "build", "-mod=vendor"}

	for _, v := range b.cfg.Flags {
		if !isAllowedArg(v) {
			return nil, fmt.Errorf("%w: %s", errorUnsupportedArguments, v)
		}
		flags = append(flags, v)

	}
	return flags, nil
}

func isAllowedArg(arg string) bool {
	for k, _ := range allowedBuildArgs {
		if strings.HasPrefix(arg, k) {
			return true
		}
	}
	return false
}

// Check if the env variable is allowed. We want to avoid
// variable injection, e.g. LD_PRELOAD, etc.
// See an overview in https://www.hale-legacy.com/class/security/s20/handout/slides-env-vars.pdf.
func isAllowedEnvVariable(name string) bool {
	for k, _ := range allowedEnvVariablePrefix {
		if strings.HasPrefix(name, k) {
			return true
		}
	}
	return false
}

// TODO: maybe not needed if handled directly by go compiler.
func (b *GoBuild) generateLdflags() (string, error) {
	var a []string
	regex := regexp.MustCompile(`{{\s*\.Env\.(.*)\s*}}`)

	for _, v := range b.cfg.Ldflags {
		var res string
		m := regex.FindStringSubmatch(v)
		// fmt.Println("match", m[1])
		if len(m) > 2 {
			return "", fmt.Errorf("%w: %s", errorEnvVariableNameEmpty, v)
		}
		if len(m) == 2 {
			name := strings.Trim(m[1], " ")

			val, exists := b.argEnv[name]
			if !exists {
				return "", fmt.Errorf("%w: %s", errorEnvVariableNameEmpty, name)
			}
			res = val
		} else {
			res = v
		}
		a = append(a, res)
	}
	if len(a) > 0 {
		return fmt.Sprintf("%s", strings.Join(a, " ")), nil
	}

	return "", nil
}
