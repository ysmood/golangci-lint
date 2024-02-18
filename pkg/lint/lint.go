// Package lint ...
package lint

import (
	"fmt"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ysmood/fetchup" //nolint: depguard
)

// DefaultVer of golangci-lint to use.
const DefaultVer = "1.56.2"

// Linter ...
type Linter struct {
	// Version of the golangci-lint to use
	Version string

	// Logger for logs not generated by golangci-lint
	Logger *log.Logger

	Stdin  *os.File
	Stdout *os.File
	Stderr *os.File
}

// New default linter.
func New() *Linter {
	return &Linter{
		Version: DefaultVer,
		Logger:  log.New(os.Stdout, "", 0),
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
}

// Lint downloads and runs the golangci-lint.
func (ltr *Linter) Lint(args ...string) error {
	err := ltr.GetLinter()
	if err != nil {
		return err
	}

	bin := ltr.binPath()

	ltr.Logger.Println(bin, strings.Join(args, " "))

	cmd := exec.Command(bin, args...)
	cmd.Stdin = ltr.Stdin
	cmd.Stdout = ltr.Stdout
	cmd.Stderr = ltr.Stderr

	return cmd.Run()
}

// GetLinter downloads the golangci-lint if not exists.
func (ltr *Linter) GetLinter() error {
	bin := ltr.binPath()

	_, err := exec.LookPath(bin)
	if err == nil {
		return nil
	}

	ext := "tar.gz"

	const defaultPerm = 0o750

	_ = os.MkdirAll(filepath.Dir(bin), defaultPerm)

	if runtime.GOOS == "windows" {
		ext = "zip"
	}

	binURL := fmt.Sprintf(
		"https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.%s",
		ltr.Version,
		ltr.Version,
		runtime.GOOS,
		runtime.GOARCH,
		ext,
	)

	ltr.Logger.Println("Download golangci-lint:", binURL)

	dir, err := os.MkdirTemp("", "*")
	if err != nil {
		return err
	}

	err = fetchup.New(dir, binURL).Fetch()
	if err != nil {
		return err
	}

	defer func() { _ = os.RemoveAll(dir) }()

	err = fetchup.StripFirstDir(dir)
	if err != nil {
		return err
	}

	return os.Rename(normalizeBin(filepath.Join(dir, "golangci-lint")), bin)
}

func (ltr *Linter) binPath() string {
	dir := filepath.Join(build.Default.GOPATH, "bin")
	p := filepath.Join(dir, "golangci-lint"+ltr.Version)

	return normalizeBin(p)
}

func normalizeBin(b string) string {
	if runtime.GOOS == "windows" {
		b += ".exe"
	}

	return b
}
