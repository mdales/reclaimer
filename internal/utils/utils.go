package utils

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"syscall"
)

func HTTPGet(url string, headers map[string]string) (*http.Response, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if nil != err {
		return nil, err
	}

	req.Header.Set("User-Agent", "Reclaimer/0.1")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return client.Do(req)
}

func HTTPPost(url string, headers map[string]string, body string) (*http.Response, error) {
	client := &http.Client{}

	req, err := http.NewRequest("POST", url, strings.NewReader(body))
	if nil != err {
		return nil, err
	}

	req.Header.Set("User-Agent", "Reclaimer/0.1")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	return client.Do(req)
}

func MoveFileByPath(sourcePath string, destinationPath string) error {

	err := os.Rename(sourcePath, destinationPath)
	if nil == err {
		return nil
	}

	if (nil != err) && (!errors.Is(err, syscall.EXDEV)) {
		return err
	}

	src, err := os.Open(sourcePath)
	if nil != err {
		return fmt.Errorf("failed to open source when copying to destination: %w", err)
	}

	dest, err := os.Create(destinationPath)
	if nil != err {
		src.Close()
		return fmt.Errorf("failed to open destination for final copy: %w", err)
	}
	defer dest.Close()

	_, err = io.Copy(dest, src)
	if nil != err {
		src.Close()
		return fmt.Errorf("error copying result to final place: %w", err)
	}
	src.Close()
	err = os.Remove(sourcePath)

	return err
}

func MakeOutputPath(sourceName string, outputName string) (string, error) {

	if "" == sourceName {
		return "", fmt.Errorf("expected source name, got empty name")
	}

	cwd, err := os.Getwd()
	if nil != err {
		return "", fmt.Errorf("failed to find current dir: %w", err)
	}

	if "" == outputName {
		outputName = cwd
	} else {
		if !path.IsAbs(outputName) {
			outputName = path.Join(cwd, outputName)
		}
	}

	sourceBaseName := path.Base(sourceName)

	info, err := os.Stat(outputName)
	if nil == err && info.IsDir() {
		return path.Join(outputName, sourceBaseName), nil
	}

	dirPath := path.Dir(outputName)
	err = os.MkdirAll(dirPath, os.ModePerm)
	if nil != err {
		return "", fmt.Errorf("failed to make output dir: %w", err)
	}

	return outputName, nil
}
