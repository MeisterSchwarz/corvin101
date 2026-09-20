package rest

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const baseURL = "https://raw.githubusercontent.com/MeisterSchwarz/ravendex-data/master/"

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func New() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},

		baseURL: baseURL,
	}
}

func (c *Client) ReadFile(
	ctx context.Context,
	filePath string,
) ([]byte, error) {
	if ctx == nil {
		ctx =
			context.Background()
	}

	ctx, cancel :=
		context.WithTimeout(
			ctx,
			10*time.Second,
		)

	defer cancel()

	url :=
		c.baseURL +
			strings.TrimLeft(
				filePath,
				"/",
			)

	req, err :=
		http.NewRequestWithContext(
			ctx,
			http.MethodGet,
			url,
			nil,
		)

	if err != nil {
		return nil, fmt.Errorf(
			"create request: %w",
			err,
		)
	}

	resp, err :=
		c.httpClient.Do(req)

	if err != nil {
		return nil, fmt.Errorf(
			"download %s: %w",
			filePath,
			err,
		)
	}

	defer resp.Body.Close()

	if resp.StatusCode !=
		http.StatusOK {

		return nil, fmt.Errorf(
			"download %s: unexpected HTTP status %s",
			filePath,
			resp.Status,
		)
	}

	data, err :=
		io.ReadAll(
			io.LimitReader(
				resp.Body,
				10<<20,
			),
		)

	if err != nil {
		return nil, fmt.Errorf(
			"read %s: %w",
			filePath,
			err,
		)
	}

	return data, nil
}
