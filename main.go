package lint

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"flag"
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

var ver = flag.String("v", "1.33.0", "version of the golangci-lint to use")

var Logger = log.New(os.Stdout, "", 0)

func Lint() {
	bin, err := getLinter()
	if err != nil {
		Logger.Fatalln(err)
	}

	cmd := exec.Command(bin, getLintArgs()...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		Logger.Fatalln(err)
	}
}

func getLinter() (string, error) {
	bin := "golangci-lint"
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	found, err := exec.LookPath(bin)
	if err == nil {
		return found, nil
	}

	ext := "tar.gz"
	bin = filepath.Join(build.Default.GOPATH, "bin", "golangci-lint")

	_ = os.MkdirAll(filepath.Dir(bin), 0755)

	if runtime.GOOS == "windows" {
		ext = "zip"
		bin += ".exe"
	}

	u := fmt.Sprintf(
		"https://github.com/golangci/golangci-lint/releases/download/v%s/golangci-lint-%s-%s-%s.%s",
		*ver,
		*ver,
		runtime.GOOS,
		runtime.GOARCH,
		ext,
	)

	return bin, download(u)
}

func getLintArgs() []string {
	args := []string{}
	lintArgs := []string{}
	sep := false
	for _, v := range os.Args {
		if v == "--" {
			sep = true
			continue
		}
		if sep {
			lintArgs = append(lintArgs, v)
		} else {
			args = append(args, v)
		}
	}

	os.Args = args

	flag.Parse()

	return lintArgs
}

func download(u string) error {
	Logger.Println("Download golangci-lint:", u)

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

	progress := &progresser{size: int(size)}

	_, err = io.Copy(io.MultiWriter(progress, zipFile), res.Body)
	if err != nil {
		return err
	}

	Logger.Println("Downloaded:", zipPath)

	err = zipFile.Close()
	if err != nil {
		return err
	}

	if path.Ext(zipPath) == ".zip" {
		return unZip(zipPath)
	}
	return unTar(zipPath)
}

func unZip(from string) error {
	zr, err := zip.OpenReader(from)
	if err != nil {
		return err
	}

	var p string
	for _, f := range zr.File {
		name := filepath.Base(f.Name)

		if !strings.HasPrefix(name, "golangci-lint") {
			continue
		}

		p = filepath.Join(build.Default.GOPATH, "bin", name)

		r, err := f.Open()
		if err != nil {
			return err
		}

		dst, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_TRUNC, f.Mode())
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

func unTar(from string) error {
	f, err := os.Open(from)
	if err != nil {
		return err
	}

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}

	tr := tar.NewReader(gr)

	var p string

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

		p = filepath.Join(build.Default.GOPATH, "bin", name)

		dst, err := os.OpenFile(p, os.O_RDWR|os.O_CREATE|os.O_TRUNC, hdr.FileInfo().Mode())
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
	size  int
	count int
	last  time.Time
}

func (p *progresser) Write(b []byte) (n int, err error) {
	n = len(b)

	if p.count == 0 {
		_, _ = fmt.Fprint(Logger.Writer(), "Progress:")
	}

	p.count += n

	if p.count == p.size {
		_, _ = fmt.Fprintln(Logger.Writer(), " 100%")
		return
	}

	if time.Since(p.last) < time.Second {
		return
	}

	p.last = time.Now()
	_, _ = fmt.Fprintf(Logger.Writer(), " %02d%%", p.count*100/p.size)

	return
}
