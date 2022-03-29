package lint

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"go/build"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// DefaultVer of golangci-lint to use
const DefaultVer = "1.45.2"

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

// New default linter
func New() *Linter {
	return &Linter{
		Version: DefaultVer,
		Logger:  log.New(os.Stdout, "", 0),
		Stdin:   os.Stdin,
		Stdout:  os.Stdout,
		Stderr:  os.Stderr,
	}
}

// Lint downloads and runs the golangci-lint
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

// GetLinter downloads the the golangci-lint if not exists
func (ltr *Linter) GetLinter() error {
	bin := ltr.binPath()

	_, err := exec.LookPath(bin)
	if err == nil {
		return nil
	}

	ext := "tar.gz"

	_ = os.MkdirAll(filepath.Dir(bin), 0755)

	if runtime.GOOS == "windows" {
		ext = "zip"
	}

	u := fmt.Sprintf(
		"https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.%s",
		ltr.Version,
		ltr.Version,
		runtime.GOOS,
		runtime.GOARCH,
		ext,
	)

	return ltr.download(u)
}

func (ltr *Linter) binPath() string {
	p := filepath.Join(build.Default.GOPATH, "bin", fmt.Sprintf("golangci-lint%s", ltr.Version))
	if runtime.GOOS == "windows" {
		p += ".exe"
	}
	return p
}

func (ltr *Linter) download(u string) error {
	ltr.Logger.Println("Download golangci-lint:", u)

	zipPath := filepath.Join(os.TempDir(), path.Base(u))

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return err
	}

	q, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	res, err := (&http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: true,
			IdleConnTimeout:   30 * time.Second,
		},
	}).Do(q)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()

	size, err := strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return err
	}

	progress := &progresser{size: int(size), logger: ltr.Logger}

	_, err = io.Copy(io.MultiWriter(progress, zipFile), res.Body)
	if err != nil {
		return err
	}

	ltr.Logger.Println("Downloaded:", zipPath)

	err = zipFile.Close()
	if err != nil {
		return err
	}

	if path.Ext(zipPath) == ".zip" {
		return ltr.unZip(zipPath)
	}
	return ltr.unTar(zipPath)
}

func (ltr *Linter) unZip(from string) error {
	zr, err := zip.OpenReader(from)
	if err != nil {
		return err
	}

	for _, f := range zr.File {
		name := filepath.Base(f.Name)

		if !strings.HasPrefix(name, "golangci-lint") {
			continue
		}

		r, err := f.Open()
		if err != nil {
			return err
		}

		dst, err := os.OpenFile(ltr.binPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		_, err = io.Copy(dst, r)
		if err != nil {
			return err
		}

		err = dst.Close()
		if err != nil {
			return err
		}
	}

	return zr.Close()
}

func (ltr *Linter) unTar(from string) error {
	f, err := os.Open(from)
	if err != nil {
		return err
	}

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}

		name := filepath.Base(hdr.Name)

		if !strings.HasPrefix(name, "golangci-lint") {
			continue
		}

		dst, err := os.OpenFile(ltr.binPath(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode())
		if err != nil {
			return err
		}

		_, err = io.Copy(dst, tr)
		if err != nil {
			return err
		}

		err = dst.Close()
		if err != nil {
			return err
		}
	}

	return f.Close()
}

type progresser struct {
	size   int
	count  int
	last   time.Time
	logger *log.Logger
}

func (p *progresser) Write(b []byte) (n int, err error) {
	n = len(b)

	if p.count == 0 {
		_, _ = fmt.Fprint(p.logger.Writer(), "Progress:")
	}

	p.count += n

	if p.count == p.size {
		_, _ = fmt.Fprintln(p.logger.Writer(), " 100%")
		return
	}

	if time.Since(p.last) < time.Second {
		return
	}

	p.last = time.Now()
	_, _ = fmt.Fprintf(p.logger.Writer(), " %02d%%", p.count*100/p.size)

	return
}
