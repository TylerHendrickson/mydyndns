package sdk

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClient(t *testing.T) {
	for _, tt := range []struct {
		name       string
		respStatus int
		respBody   []byte
		expectPath string
		expectIP   net.IP
		expectErr  func(s *httptest.Server) error
		do         func(c *Client) (net.IP, error)
	}{
		{
			"MyIP() 200 response",
			http.StatusOK,
			[]byte("1.2.3.4"),
			"/my-ip",
			net.ParseIP("1.2.3.4"),
			func(*httptest.Server) error { return nil },
			func(c *Client) (net.IP, error) { return c.MyIP() },
		},
		{
			"MyIP() 404 response",
			http.StatusNotFound,
			[]byte("not found"),
			"/my-ip",
			nil,
			func(s *httptest.Server) error {
				return UnexpectedStatusCode{url: s.URL + "/my-ip", receivedStatus: http.StatusNotFound}
			},
			func(c *Client) (net.IP, error) { return c.MyIP() },
		},
		{
			"UpdateAlias() 200 response",
			http.StatusOK,
			[]byte("9.8.7.6"),
			"/dns-alias",
			net.ParseIP("9.8.7.6"),
			func(*httptest.Server) error { return nil },
			func(c *Client) (net.IP, error) { return c.UpdateAlias() },
		},
		{
			"UpdateAlias() with unparseable IP",
			http.StatusOK,
			[]byte("badip"),
			"/dns-alias",
			nil,
			func(*httptest.Server) error { return &net.ParseError{Type: "IP address", Text: "badip"} },
			func(c *Client) (net.IP, error) { return c.UpdateAlias() },
		},
		{
			"UpdateAlias() with too long response body",
			http.StatusOK,
			[]byte(strings.Repeat("a", maxIPStrLen+1)),
			"/dns-alias",
			nil,
			func(*httptest.Server) error {
				return &net.ParseError{Type: "IP address", Text: strings.Repeat("a", maxIPStrLen)}
			},
			func(c *Client) (net.IP, error) { return c.UpdateAlias() },
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			apiKey := "asdfjkl"
			server := httptest.NewServer(http.HandlerFunc(func(resp http.ResponseWriter, req *http.Request) {
				assert.Equal(t, apiKey, req.Header.Get("x-api-key"))
				assert.Equal(t, "text/plain", req.Header.Get("accept"))
				assert.Equal(t, tt.expectPath, req.RequestURI)

				resp.WriteHeader(tt.respStatus)
				resp.Header().Set("content-type", "text/plain")
				resp.Write(tt.respBody)
			}))
			defer server.Close()
			c := NewClient(server.URL, apiKey)
			ip, err := tt.do(c)

			assert.Equal(t, tt.expectIP.String(), ip.String())
			if expectedErr := tt.expectErr(server); expectedErr != nil {
				assert.EqualError(t, err, expectedErr.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
