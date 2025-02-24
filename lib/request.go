package lib

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

func RetrieveFile(url string) (io.Reader, error) {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	file := bytes.NewBuffer(make([]byte, 0, resp.ContentLength))
	if _, err := io.Copy(file, resp.Body); err != nil {
		return nil, err
	}

	return file, nil
}
