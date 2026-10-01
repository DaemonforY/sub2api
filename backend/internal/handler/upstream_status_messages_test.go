//go:build unit

package handler

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpstreamSizeAndTimeoutErrorsAreExplained(t *testing.T) {
	openai := &OpenAIGatewayHandler{}
	status, _, msg := openai.mapUpstreamError(http.StatusRequestEntityTooLarge)
	require.Equal(t, http.StatusRequestEntityTooLarge, status)
	require.Contains(t, msg, "20MB")
	for _, code := range []int{408, 504, 524} {
		status, _, msg = openai.mapUpstreamError(code)
		require.Equal(t, http.StatusGatewayTimeout, status, code)
		require.Equal(t, msgUpstreamTimeout, msg)
	}
	status, _, _ = openai.mapUpstreamError(503)
	require.Equal(t, http.StatusBadGateway, status)

	claude := &GatewayHandler{}
	status, _, msg = claude.mapUpstreamError(413)
	require.Equal(t, http.StatusRequestEntityTooLarge, status)
	require.Equal(t, msgUpstreamTooLarge, msg)

	status, msg = mapGeminiUpstreamError(524)
	require.Equal(t, http.StatusGatewayTimeout, status)
	require.Equal(t, msgUpstreamTimeout, msg)
}
