package sdk

import (
	"fmt"
	"net/http"
)

// UnexpectedStatusCode indicates that a request to the mydyndns API resulted in a response with an HTTP status code
// that was unexpected, indicating that the requested operation failed.
type UnexpectedStatusCode struct {
	url            string
	receivedStatus int
}

func NewUnexpectedStatusCode(req *http.Request, resp *http.Response) UnexpectedStatusCode {
	return UnexpectedStatusCode{url: req.URL.String(), receivedStatus: resp.StatusCode}
}

// Error represents an UnexpectedStatusCode as formatted string error message that contains the request URL and the
// unexpected status code from the response.
func (err UnexpectedStatusCode) Error() string {
	return fmt.Sprintf("request to %s responded with unexpected status code %d (%s)",
		err.url, err.receivedStatus, err.StatusText())
}

// URL returns the requested URL which responded with an unexpected status code.
func (err *UnexpectedStatusCode) URL() string {
	return err.url
}

// StatusCode returns the HTTP status code value that was unexpected for the API response.
func (err *UnexpectedStatusCode) StatusCode() int {
	return err.receivedStatus
}

// StatusText returns a text for the HTTP status code that was unexpected for the API response.
func (err *UnexpectedStatusCode) StatusText() string {
	return http.StatusText(err.receivedStatus)
}
