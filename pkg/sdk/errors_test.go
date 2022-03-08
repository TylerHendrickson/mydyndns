package sdk

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnexpectedStatusCode(t *testing.T) {
	t.Run("", func(t *testing.T) {
		req, newRequestErr := http.NewRequest("GET", "https://example.com", http.NoBody)
		require.NoError(t, newRequestErr)

		err := NewUnexpectedStatusCode(req, &http.Response{StatusCode: http.StatusBadRequest})

		assert.Equal(t, http.StatusBadRequest, err.StatusCode())
		assert.Equal(t, "https://example.com", err.URL())
		assert.EqualError(t, err,
			"request to https://example.com responded with unexpected status code 400 (Bad Request)")
	})
}
