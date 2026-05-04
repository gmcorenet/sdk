package gmcore_httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	client   *http.Client
	baseURL  string
	headers  map[string]string
	timeout  time.Duration
}

func NewClient() *Client {
	return &Client{
		client:  &http.Client{Timeout: 30 * time.Second},
		headers: make(map[string]string),
		timeout: 30 * time.Second,
	}
}

func (c *Client) SetBaseURL(url string) *Client    { c.baseURL = url; return c }
func (c *Client) SetTimeout(t time.Duration) *Client { c.timeout = t; c.client.Timeout = t; return c }
func (c *Client) SetHeader(k, v string) *Client      { c.headers[k] = v; return c }

func (c *Client) buildURL(rawURL string) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", errors.New("url cannot be empty")
	}

	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}
		if parsed.Host == "" {
			return "", errors.New("URL must have a host")
		}
		return rawURL, nil
	}

	if c.baseURL == "" {
		return "", errors.New("base URL not set and relative URL provided")
	}

	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	relative, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid relative URL: %w", err)
	}

	if relative.Host != "" {
		return "", errors.New("relative URL cannot have a host")
	}

	resolved := base.ResolveReference(relative)
	return resolved.String(), nil
}

func (c *Client) Get(ctx context.Context, url string) (*Response, error) {
	return c.Do(ctx, "GET", url, nil)
}

func (c *Client) Post(ctx context.Context, url string, body interface{}) (*Response, error) {
	return c.Do(ctx, "POST", url, body)
}

func (c *Client) Put(ctx context.Context, url string, body interface{}) (*Response, error) {
	return c.Do(ctx, "PUT", url, body)
}

func (c *Client) Delete(ctx context.Context, url string) (*Response, error) {
	return c.Do(ctx, "DELETE", url, nil)
}

func (c *Client) Patch(ctx context.Context, url string, body interface{}) (*Response, error) {
	return c.Do(ctx, "PATCH", url, body)
}

func (c *Client) Do(ctx context.Context, method, url string, body interface{}) (*Response, error) {
	fullURL, err := c.buildURL(url)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	var bodyReader io.Reader

	if body != nil {
		switch v := body.(type) {
		case string:
			bodyReader = bytes.NewBufferString(v)
		case []byte:
			bodyReader = bytes.NewBuffer(v)
		default:
			jsonBytes, err := json.Marshal(body)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal request body: %w", err)
			}
			bodyReader = bytes.NewBuffer(jsonBytes)
			c.headers["Content-Type"] = "application/json"
		}
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return nil, err
	}

	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	headers := make(map[string]string)
	for k, v := range resp.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	return &Response{statusCode: resp.StatusCode, headers: headers, body: respBody}, nil
}

type Response struct {
	statusCode int
	headers    map[string]string
	body       []byte
}

func (r *Response) StatusCode() int   { return r.statusCode }
func (r *Response) Headers() map[string]string {
	result := make(map[string]string, len(r.headers))
	for k, v := range r.headers {
		result[k] = v
	}
	return result
}
func (r *Response) Body() []byte       { return r.body }
func (r *Response) BodyString() string { return string(r.body) }
func (r *Response) Ok() bool           { return r.statusCode >= 200 && r.statusCode < 300 }
func (r *Response) Json(v interface{}) error { return json.Unmarshal(r.body, v) }
