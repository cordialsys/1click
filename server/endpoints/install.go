package endpoints

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cordialsys/panel/server/panel"
	"github.com/cordialsys/panel/server/servererrors"
	"github.com/gofiber/fiber/v2"
)

func formatDownloadUrl(version, binaryName string) string {
	arch := strings.Replace(runtime.GOARCH, "aarch64", "arm64", 1)
	arch = strings.Replace(arch, "x86_64", "amd64", 1)
	os := runtime.GOOS
	os = strings.Replace(os, "darwin", "macos", 1)
	// Example: https://dl.cordial.systems/bin/24.5.5/treasury-cli-24.5.5-linux-amd64.tar.gz
	remote := fmt.Sprintf(
		"https://dl.cordial.systems/bin/%s/%s-%s-%s-%s.tar.gz",
		version, binaryName, version, os, arch,
	)
	if version == "latest" || version == "preview" || version == "pre" {
		// download the latest version uses different path & different identifiers :/
		build := ""
		if os == "macos" || os == "darwin" {
			// this means macos arm64
			build = "mac"
		} else {
			if arch == "amd64" {
				// this means linux amd64/x86_64
				build = "x86"
			} else if arch == "arm64" {
				// this means linux arm64
				build = "arm"
			} else {
				// ?? don't know what to do in this case, an unsure
				// how to offload to the dl server.
				slog.Warn("unknown arch", "arch", arch)
				build = arch
			}
		}

		remote = fmt.Sprintf(
			"https://dl.cordial.systems/%s/%s/%s",
			version, build, binaryName,
		)
	}
	return remote
}

func (endpoints *Endpoints) Install(c *fiber.Ctx) error {
	binaryName := c.Params("binary")
	version := c.Params("version")
	remote := formatDownloadUrl(version, binaryName)

	fmt.Println("downloading", remote)

	err := DownloadAndUntar(endpoints.panel, remote, endpoints.panel.BinaryDir, true)
	if err != nil {
		// handle error
		return err
	}

	return endpoints.GetBinaryVersion(c)
}

// Base64 encode if needed
func encodeApiKey(userPwMaybe string) string {
	if strings.Contains(userPwMaybe, ":") {
		return base64.StdEncoding.EncodeToString([]byte(userPwMaybe))
	}
	return userPwMaybe
}

func DownloadSigFor(apiKey, url string) (string, error) {
	req, err := http.NewRequest("GET", url+"/sig", nil)
	if err != nil {
		return "", servererrors.InternalErrorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", encodeApiKey(apiKey)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", servererrors.InternalErrorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", servererrors.InternalErrorf("failed to read response body: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		bz, _ := io.ReadAll(resp.Body)
		return "", servererrors.InternalErrorf("download server error: %s: %s", resp.Status, string(bz))
	}

	return string(body), nil
}

func DownloadAndUntar(panel *panel.Panel, url, destPath string, verify bool) error {
	// Create HTTP request
	log := slog.With("url", url)
	log.Info("downloading")
	t1 := time.Now()
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return servererrors.InternalErrorf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", encodeApiKey(panel.ApiKey)))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return servererrors.InternalErrorf("failed to download file: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bz, _ := io.ReadAll(resp.Body)
		return servererrors.InternalErrorf("download server error: %s: %s", resp.Status, string(bz))
	}

	if verify {
		sig, err := DownloadSigFor(panel.ApiKey, url)
		if err != nil {
			return servererrors.InternalErrorf("failed to download signature: %v", err)
		}

		tempDir, err := os.MkdirTemp("", "binary-install")
		if err != nil {
			return servererrors.InternalErrorf("failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)
		tmpPath := filepath.Join(tempDir, "dl.tar.gz")

		tmpFile, err := os.Create(tmpPath)
		if err != nil {
			return servererrors.InternalErrorf("failed to create temp file: %v", err)
		}
		defer tmpFile.Close()

		// Stream to a temp file
		_, err = io.Copy(tmpFile, resp.Body)
		if err != nil {
			return servererrors.InternalErrorf("failed to copy to temp file: %v", err)
		}
		tmpFile.Close()

		// Now create two readers:
		// - one for the signature verification
		// - one for the payload
		payloadReader, err := os.Open(tmpPath)
		if err != nil {
			return servererrors.InternalErrorf("failed to open temp file: %v", err)
		}
		defer payloadReader.Close()

		payloadReaderForResp, err := os.Open(tmpPath)
		if err != nil {
			return servererrors.InternalErrorf("failed to open temp file: %v", err)
		}
		defer payloadReaderForResp.Close()

		resp.Body = payloadReaderForResp
		decodedSig, err := base64.StdEncoding.DecodeString(sig)
		if err != nil {
			return servererrors.InternalErrorf("failed to decode signature: %v", err)
		}

		verifier := panel.GetBinaryVerifierOrDefault()
		err = verifier.VerifySignature(
			bytes.NewBuffer(decodedSig),
			payloadReader,
		)
		if err != nil {
			return servererrors.InternalErrorf("failed to verify signature: %v", err)
		}
	}

	// Create gzip reader
	gzipReader, err := gzip.NewReader(resp.Body)
	if err != nil {
		return servererrors.InternalErrorf("failed to create gzip reader: %v", err)
	}
	defer gzipReader.Close()
	// Create tar reader
	tarReader := tar.NewReader(gzipReader)

	// Ensure destination directory exists
	if err := os.MkdirAll(destPath, 0755); err != nil {
		return servererrors.InternalErrorf("failed to create destination directory: %v", err)
	}

	// Extract tar contents
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar reading error: %w", err)
		}

		// Get the target path
		target := filepath.Join(destPath, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			// Create the file
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}

			// Copy the file contents
			if _, err := io.Copy(f, tarReader); err != nil {
				f.Close()
				return fmt.Errorf("failed to write file %s: %w", target, err)
			}
			f.Close()
		}
	}
	log.Info("done", "duration", time.Since(t1).String())

	return nil
}
