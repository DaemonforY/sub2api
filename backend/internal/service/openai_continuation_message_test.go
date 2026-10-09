//go:build unit

package service

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// The gateway's own 400 for previous_response_id is Chinese with the English kept in parentheses;
// a sub2api downstream of this one must still recognise it as "continuation unsupported".
func TestPreviousResponseUnsupportedMatchesLocalizedMessage(t *testing.T) {
	msg := "当前分组不支持 previous_response_id（服务端不保存上一轮对话）：请去掉这个参数，把之前的对话记录放进 input 一起发送；本次请求未扣费（previous_response_id requires an OpenAI API-key account for HTTP requests）"
	body := []byte(`{"error":{"message":"` + msg + `","type":"invalid_request_error"}}`)
	require.True(t, isOpenAICompatPreviousResponseUnsupported(http.StatusBadRequest, msg, body))
}
