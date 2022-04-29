// Package sdk provides MyDynDNS API integrations.
package sdk

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

// maxIPStrLen defines the maximum amount of characters in a valid IP (v6) address.
const maxIPStrLen = 48

// Client is an SDK for the MyDynDNS API.
type Client struct {
	BaseURL    string
	apiKey     string
	HTTPClient *http.Client
}

// NewClient returns a pointer to a new Client configured to make requests
// authenticated with apiKey to a MyDynDNS web service hosted at BaseURL.
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL:    baseURL,
		apiKey:     apiKey,
		HTTPClient: &http.Client{Timeout: time.Second * 30},
	}
}

// MyIP wraps MyIPWithContext using context.Background.
func (c *Client) MyIP() (net.IP, error) {
	return c.MyIPWithContext(context.Background())
}

// MyIPWithContext retrieves the apparent IP address of the host from which the request originated.
// Calling this function should not result in modification to the DNS alias maintained by the mydyndns web service.
// It returns the retrieved net.IP address or an error that caused the operation to fail.
func (c *Client) MyIPWithContext(ctx context.Context) (net.IP, error) {
	return c.fetchIP(ctx, "GET", "my-ip")
}

// UpdateAlias wraps UpdateAliasWithContext using context.Background.
func (c *Client) UpdateAlias() (net.IP, error) {
	return c.UpdateAliasWithContext(context.Background())
}

// UpdateAliasWithContext retrieves the apparent IP address of the host from which the request originated
// and requests that the DNS alias maintained by the mydyndns web service be updated to that IP address.
// It returns the apparent net.IP address or an error that caused the operation to fail.
func (c *Client) UpdateAliasWithContext(ctx context.Context) (net.IP, error) {
	return c.fetchIP(ctx, "POST", "dns-value")
}

func (c *Client) fetchIP(ctx context.Context, method, path string) (ip net.IP, err error) {
	req, err := c.newRequest(ctx, method, path)
	if err != nil {
		return
	}

	resp, err := c.doRequest(req)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return
	}

	return c.parseIP(resp.Body)
}

func (c *Client) newRequest(ctx context.Context, method, path string) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, fmt.Sprintf("%s/%s", c.BaseURL, path), http.NoBody)
	if err == nil {
		req.Header.Set("accept", "text/plain")
		req.Header.Set("x-api-key", c.apiKey)
	}

	return req, err
}

func (c *Client) doRequest(req *http.Request) (resp *http.Response, err error) {
	resp, err = c.HTTPClient.Do(req)
	if err == nil && resp.StatusCode != 200 {
		err = NewUnexpectedStatusCode(req, resp)
	}

	return
}

// parseIP reads up to maxIPStrLen bytes from (a response body) io.Reader and parses as an IP address.
// When the returned error is not nil, the IP address is considered invalid.
func (c *Client) parseIP(r io.Reader) (ip net.IP, err error) {
	lr := io.LimitReader(r, maxIPStrLen)
	buf := make([]byte, maxIPStrLen)
	size := 0
	for {
		n, err := lr.Read(buf[size:])
		size += n
		if err == io.EOF {
			break
		}
	}
	err = ip.UnmarshalText(buf[:size])
	return
}
