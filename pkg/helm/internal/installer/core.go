package installer

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/evanlouie/go/pkg/helm"
	"github.com/google/go-github/v33/github"
)

func decompressedGZippedBin(body []byte) ([]byte, error) {
	byteReader := bytes.NewReader(body)
	gzr, err := gzip.NewReader(byteReader)
	if err != nil {
		return nil, fmt.Errorf(`creating gzip reader: %w`, err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil, fmt.Errorf(`no file with name "helm" found in tar.gz file`)
		case err != nil:
			return nil, fmt.Errorf(`parsing file in tar.gz file: %w`, err)
		case header == nil:
			continue
		default:
			filename := filepath.Base(header.Name)
			if filename == "helm" && header.Typeflag == tar.TypeReg {
				helmBytes, err := io.ReadAll(tr)
				if err != nil {
					return nil, fmt.Errorf(`reading bytes from %s in tar.gz file`, header.Name)
				}
				return helmBytes, nil
			}
		}
	}
}

func decompressZippedBin(body []byte) ([]byte, error) {
	r := bytes.NewReader(body)
	rdr, err := zip.NewReader(r, int64(len(body)))
	if err != nil {
		return nil, fmt.Errorf(`creating zip reader: %s`, err)
	}

	for _, zipFile := range rdr.File {
		filename := filepath.Base(zipFile.Name)
		if filename == "helm.exe" {
			f, err := zipFile.Open()
			if err != nil {
				return nil, err
			}
			helmBytes, err := io.ReadAll(f)
			if err != nil {
				return nil, err
			}
			return helmBytes, err
		}
	}

	return nil, fmt.Errorf(`no file named "helm.exe" found in zip file`)
}

// downloadLatest downloads the helm latest binary from the latest release from
// github for the OS corresponding to runtime.GOOS and return it as a byte
// slice.
func downloadLatest() ([]byte, error) {
	// get the latest github release
	client := github.NewClient(nil)
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), "helm", "Helm")
	if err != nil {
		return nil, fmt.Errorf(`getting latest release from github for helm/helm: %w`, err)
	}
	if len(strings.TrimSpace(*release.Body)) == 0 {
		return nil, fmt.Errorf(`getting latest release from github for helm/helm: empty release body was found for release %s`, *release.Name)
	}

	// get the correct compressed extension
	var compressExt string
	switch runtime.GOOS {
	case "darwin":
		fallthrough
	case "linux":
		compressExt = "tar.gz"
	case "windows":
		compressExt = "zip"
	default:
		return nil, fmt.Errorf(`downloading helm binary: unsupported host %s`, runtime.GOOS)
	}

	// download the os specific release
	downloadURL := fmt.Sprintf(`https://get.helm.sh/helm-%s-%s-amd64.%s`, *release.TagName, runtime.GOOS, compressExt)
	resp, err := http.Get(downloadURL)
	if err != nil {
		return nil, fmt.Errorf(`downloading helm from %s: %w`, downloadURL, err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(`reading body of helm download response from %s: %w`, downloadURL, err)
	}

	// decompress the file and get the helm bin bytes
	var helmBinBytes []byte
	switch runtime.GOOS {
	case "darwin":
		fallthrough
	case "linux":
		helmBinBytes, err = decompressedGZippedBin(bodyBytes)
	case "windows":
		helmBinBytes, err = decompressZippedBin(bodyBytes)
	default:
		return nil, fmt.Errorf(`unsupported os for decompressing downloaded helm binary`)
	}

	// ensure final data is valid-ish
	switch {
	case err != nil:
		return nil, fmt.Errorf(`decompressing downloaded helm binary: %w`, err)
	case helmBinBytes == nil:
		return nil, fmt.Errorf(`empty byte slice found for "helm" file in compressed download file`)
	}

	return helmBinBytes, nil
}

// Install the latest Helm release to a temporary file on the host. Returns a
// the path to the installed binary.
// It is the users callers responsibility to ensure that the file is cleaned up.
func Install() (string, error) {
	// create a temp file and write out helm to it
	f, tmpErr := os.CreateTemp("", "fabrikate")
	if tmpErr != nil {
		return "", fmt.Errorf(`creating temporary file to hold downloaded helm binary: %w`, tmpErr)
	}
	downloadedBytes, downloadErr := downloadLatest()
	if downloadErr != nil {
		return "", fmt.Errorf(`downloaded latest helm release: %w`, downloadErr)
	}

	// write the bytes out to the temp file and return the temp file path
	if _, err := f.Write(downloadedBytes); err != nil {
		return "", fmt.Errorf(`writing downloaded helm binary to temporary file %s: %w`, f.Name(), err)
	}

	// make the type file is executable
	if err := os.Chmod(f.Name(), os.ModePerm); err != nil {
		return "", fmt.Errorf(`setting permission %s to downloaded Helm binary %s`, os.ModePerm, f.Name())
	}

	return f.Name(), nil
}

// GetHelm gets the path to a Helm 3 binary first searching for it on the user
// $PATH or installing it to a temporary file if it is not found.
func GetHelm() (string, error) {
	helmPath, err := exec.LookPath("helm")
	switch {
	case err == exec.ErrNotFound:
		return Install()
	case err != nil:
		return "", fmt.Errorf(`finding "helm" in $PATH: %w`, err)
	default:
		if v, err := helm.Version(); err == nil && v.IsHelm3() {
			return helmPath, nil
		}
		return Install()
	}
}
