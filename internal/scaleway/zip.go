package scaleway

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
)

const avoidZipBombMaxSize = 1024 * 1024 * 100 // 100MB

var (
	ErrUploadingCodeArchive   = errors.New("uploading code archive")
	ErrDownloadingCodeArchive = errors.New("downloading code archive")
)

type CodeArchive struct {
	Path string
	Size uint64
}

func NewCodeArchive(from string) (*CodeArchive, error) {
	zipFile, err := os.CreateTemp("", "function-archive-*.zip")
	if err != nil {
		return nil, fmt.Errorf("creating temp zip file: %w", err)
	}

	defer func() {
		_ = zipFile.Close()
	}()

	err = zipDirectory(zipFile, from)
	if err != nil {
		return nil, fmt.Errorf("zipping directory: %w", err)
	}

	_, err = zipFile.Seek(0, 0)
	if err != nil {
		return nil, fmt.Errorf("seeking zip file: %w", err)
	}

	stat, err := zipFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("getting zip file stat: %w", err)
	}

	return &CodeArchive{
		Path: zipFile.Name(),
		Size: safeConvertInt64ToUint64(stat.Size()),
	}, nil
}

func (f *CodeArchive) Upload(ctx context.Context, preSignedURL string) error {
	zipFile, err := os.Open(f.Path)
	if err != nil {
		return fmt.Errorf("opening zip file: %w", err)
	}

	// You might think we're missing a Close() here, but it's subtly handled by
	// go's http.NewRequestWithContext().
	// Even though it takes an io.Reader (and not an io.ReadCloser), it checks
	// if the reader is a Closer and closes it when the request is done.
	// Reference: https://pkg.go.dev/net/http#NewRequestWithContext
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, preSignedURL, zipFile)
	if err != nil {
		return fmt.Errorf("creating upload request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")

	// Required in newer versions of Go to avoid chunked encoding
	req.ContentLength = safeConvertUint64ToInt64(f.Size)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("uploading code archive: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status code %d", ErrUploadingCodeArchive, resp.StatusCode)
	}

	return nil
}

func DownloadAndExtractCodeArchive(ctx context.Context, url, toDir string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating download request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("downloading code archive: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: status code %d", ErrDownloadingCodeArchive, resp.StatusCode)
	}

	tmpFile, err := os.CreateTemp("", "function-download-*.zip")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	defer func() {
		_ = tmpFile.Close()
	}()

	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return fmt.Errorf("copying response body to temp file: %w", err)
	}

	err = unzipDirectory(tmpFile.Name(), toDir)
	if err != nil {
		return fmt.Errorf("unzipping directory: %w", err)
	}

	return nil
}

func zipDirectory(zipFile *os.File, pathToDir string) error {
	zipWriter := zip.NewWriter(zipFile)

	defer func() {
		_ = zipWriter.Close()
	}()

	walker := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("walking directory: %w", err)
		}

		if info.IsDir() {
			return nil
		}

		// Very important to use filepath.Rel() here to avoid zipping the full path.
		// Otherwise, we end up with a zip file containing the full path to the file like: `workspaces/e2e/assets/...`.
		relativePath, err := filepath.Rel(pathToDir, path)
		if err != nil {
			return fmt.Errorf("getting relative path: %w", err)
		}

		// We use os.OpenInRoot to avoid local inclusion vulnerabilities.
		file, err := os.OpenInRoot(pathToDir, relativePath)
		if err != nil {
			return fmt.Errorf("opening file: %w", err)
		}

		defer func() {
			_ = file.Close()
		}()

		f, err := zipWriter.Create(relativePath)
		if err != nil {
			return fmt.Errorf("creating file in zip: %w", err)
		}

		_, err = io.Copy(f, file)
		if err != nil {
			return fmt.Errorf("copying file to zip: %w", err)
		}

		return nil
	}

	err := filepath.Walk(pathToDir, walker)
	if err != nil {
		return fmt.Errorf("walking directory %q: %w", pathToDir, err)
	}

	return nil
}

//nolint:revive,funlen
func unzipDirectory(zipPath, toDir string) error {
	root, err := os.OpenRoot(toDir)
	if err != nil {
		return fmt.Errorf("opening root directory: %w", err)
	}

	defer func() {
		_ = root.Close()
	}()

	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("opening zip file: %w", err)
	}

	defer func() {
		_ = zipReader.Close()
	}()

	for _, file := range zipReader.File {
		if file.FileInfo().IsDir() {
			err := root.MkdirAll(file.Name, 0o750)
			if err != nil {
				return fmt.Errorf("creating directory: %w", err)
			}

			continue
		}

		if err := root.MkdirAll(filepath.Dir(file.Name), 0o750); err != nil {
			return fmt.Errorf("creating directory for file: %w", err)
		}

		// We use os.OpenInRoot to avoid local inclusion vulnerabilities.
		outFile, err := root.Open(file.Name)

		if os.IsNotExist(err) {
			outFile, err = root.Create(file.Name)
			if err != nil {
				return fmt.Errorf("creating file: %w", err)
			}
		}

		if err != nil {
			return fmt.Errorf("opening file: %w", err)
		}

		rc, err := file.Open()
		if err != nil {
			return fmt.Errorf("opening zipped file: %w", err)
		}

		_, err = io.CopyN(outFile, rc, avoidZipBombMaxSize+1)
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("copying zipped file to disk: %w", err)
		}

		if err := outFile.Close(); err != nil {
			return fmt.Errorf("closing output file: %w", err)
		}

		if err := rc.Close(); err != nil {
			return fmt.Errorf("closing zipped file: %w", err)
		}
	}

	return nil
}

func safeConvertInt64ToUint64(i int64) uint64 {
	if i < 0 {
		return 0
	}

	return uint64(i)
}

func safeConvertUint64ToInt64(u uint64) int64 {
	if u > math.MaxInt64 {
		return math.MaxInt64
	}

	return int64(u)
}
