package handler

import "github.com/Wei-Shaw/sub2api/internal/service"

// Client-visible texts for gateway-side failures. End users read these in their client, so the
// Chinese part says what happened and what to do; the original English stays in parentheses because
// clients, tests and the ops classifiers match on it. Failed requests are not billed.
const (
	msgUpstreamAuthFailed      = "上游账号鉴权失败，已通知管理员处理，请稍后重试；本次请求未扣费（Upstream authentication failed, please contact administrator）"
	msgUpstreamForbidden       = "上游账号访问受限，已通知管理员处理，请稍后重试；本次请求未扣费（Upstream access forbidden, please contact administrator）"
	msgUpstreamRateLimited     = "上游模型服务限流中，请过 1–2 分钟再试；本次请求未扣费（Upstream rate limit exceeded, please retry later）"
	msgUpstreamOverloaded      = "上游模型服务过载，请稍后重试；本次请求未扣费（Upstream service overloaded, please retry later）"
	msgUpstreamUnavailable     = "上游模型服务暂时不可用（已自动切换账号重试仍失败），请稍后重试；本次请求未扣费（Upstream service temporarily unavailable）"
	msgUpstreamTooLarge        = "请求内容太大：上游单次请求上限约 20MB，请压缩参考图或减少参考图数量后再试；本次请求未扣费（Request body too large）"
	msgUpstreamTimeout         = "上游生成超时（超过 2 分钟没有返回），通常是上游繁忙或尺寸、质量设置较高，请稍后重试或调低尺寸/质量；本次请求未扣费（Upstream request timed out）"
	msgUpstreamFailed          = "上游请求失败，请稍后重试；如持续出现请联系客服；本次请求未扣费（Upstream request failed）"
	msgQueueFull               = "排队的请求太多，请等正在进行的请求完成后再试（Too many pending requests, please retry later）"
	msgServiceBusyRetry        = "服务繁忙，请稍后重试（Service temporarily unavailable, please retry later）"
	msgContinuationUnsupported = service.OpenAIContinuationUnsupportedClientMessage
	msgHostedWebSearchFailed   = "联网搜索工具（web_search）在当前分组暂时不可用：请去掉 tools 里的 web_search 后重试；需要最新信息时，可以先自己搜索，再把结果放进提示词；本次请求未扣费（Hosted web_search tool failed upstream）"
	msgBillingNoBalance        = "账户余额不足，请到 " + service.RechargeURL + " 充值后重试，或改用订阅分组的 Key；本次请求未扣费（insufficient balance）"
)

func concurrencyLimitMessage(slotType string) string {
	who := "当前账号"
	switch slotType {
	case "user":
		who = "你的账号"
	case "api_key", "key":
		who = "这个 API Key"
	case "account":
		who = "上游账号"
	}
	return "同时进行的请求数已达" + who + "的并发上限，请等其它请求完成后再试（Concurrency limit exceeded for " + slotType + ", please retry later）"
}
