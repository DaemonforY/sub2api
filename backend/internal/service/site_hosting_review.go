package service

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Review of every published version. A quick content check runs on every upload: pages with
// password / card fields, gambling or adult wording, or a well-known brand next to login wording
// wait for an admin. When the admin configured a model, it reads the page text and decides
// instead (its verdict wins over the keyword check; on errors the keyword check applies).

const (
	SiteReviewApproved   = "approved"
	SiteReviewPending    = "pending"
	SiteReviewRejected   = "rejected"
	SiteReviewSuperseded = "superseded"

	siteReviewExcerptRunes = 4000
)

// SiteReview is the outcome of reviewing one version.
type SiteReview struct {
	Status  string
	Reason  string
	Flags   []string
	Excerpt string
	By      string // auto | model | admin
}

var (
	siteTagRe       = regexp.MustCompile(`(?is)<(script|style|noscript|template)\b.*?</(script|style|noscript|template)\s*>`)
	siteAnyTagRe    = regexp.MustCompile(`(?s)<[^>]*>`)
	siteSpaceRe     = regexp.MustCompile(`\s+`)
	sitePasswordRe  = regexp.MustCompile(`(?i)<input\b[^>]*type\s*=\s*["']?password`)
	siteCardFieldRe = regexp.MustCompile(`(?i)<input\b[^>]*(name|id|placeholder)\s*=\s*["'][^"']*(card.?num|cvv|cvc|银行卡|卡号|身份证|idcard|支付密码)`)
	siteTitleRe     = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
)

var siteRiskWords = map[string][]string{
	"gambling": {"博彩", "赌场", "百家乐", "六合彩", "时时彩", "老虎机", "真人荷官", "下注", "彩票平台", "棋牌游戏", "casino", "baccarat", "sportsbook"},
	"adult":    {"约炮", "裸聊", "成人视频", "色情", "黄色网站", "援交", "porn", "xxx video"},
	"fraud":    {"刷单", "日结返利", "高额返利", "保本高收益", "稳赚不赔", "内部渠道", "资金盘", "杀猪盘", "代办贷款", "无抵押贷款秒批"},
}

var siteBrandWords = []string{"支付宝", "微信支付", "淘宝", "京东", "拼多多", "工商银行", "建设银行", "农业银行", "中国银行", "招商银行", "交通银行", "邮储银行",
	"中国移动", "中国联通", "中国电信", "苹果", "apple id", "icloud", "paypal", "腾讯", "qq安全", "公安", "法院", "检察院", "税务局", "社保", "医保", "steam", "microsoft", "outlook", "gmail"}

var sitePhishingWords = []string{"登录", "登陆", "密码", "验证码", "账号冻结", "账户异常", "安全中心", "身份验证", "解冻", "退款", "sign in", "log in", "verify your account", "password"}

// siteContentText is the readable text of the site's HTML pages (index first), for the check and
// for the admin's review queue.
func siteContentText(files []siteFile) (text string, pages []string) {
	sorted := make([]siteFile, 0, len(files))
	for _, f := range files {
		if strings.HasPrefix(SiteContentType(f.Path), "text/html") {
			sorted = append(sorted, f)
		}
	}
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Path == "index.html" && sorted[j].Path != "index.html" })
	var b strings.Builder
	for _, f := range sorted {
		page := string(f.Data)
		pages = append(pages, page)
		if b.Len() > siteReviewExcerptRunes*3 {
			continue
		}
		title := ""
		if m := siteTitleRe.FindStringSubmatch(page); m != nil {
			title = strings.TrimSpace(html.UnescapeString(siteAnyTagRe.ReplaceAllString(m[1], " ")))
		}
		body := siteTagRe.ReplaceAllString(page, " ")
		body = html.UnescapeString(siteAnyTagRe.ReplaceAllString(body, " "))
		body = strings.TrimSpace(siteSpaceRe.ReplaceAllString(body, " "))
		fmt.Fprintf(&b, "[%s] %s\n%s\n", f.Path, title, body)
	}
	return truncateRunes(strings.ToValidUTF8(b.String(), ""), siteReviewExcerptRunes), pages
}

// checkSiteContent is the keyword check: what it found and whether an admin should look first.
func checkSiteContent(files []siteFile) (flags []string, excerpt string, risky bool) {
	excerpt, pages := siteContentText(files)
	lower := strings.ToLower(excerpt)
	add := func(flag string) {
		for _, f := range flags {
			if f == flag {
				return
			}
		}
		flags = append(flags, flag)
	}
	for _, page := range pages {
		if sitePasswordRe.MatchString(page) {
			add("password_field")
		}
		if siteCardFieldRe.MatchString(page) {
			add("card_or_id_field")
		}
	}
	for kind, words := range siteRiskWords {
		for _, w := range words {
			if strings.Contains(lower, strings.ToLower(w)) {
				add(kind + ":" + w)
				break
			}
		}
	}
	brand := ""
	for _, w := range siteBrandWords {
		if strings.Contains(lower, w) {
			brand = w
			break
		}
	}
	if brand != "" {
		for _, w := range sitePhishingWords {
			if strings.Contains(lower, w) {
				add("brand_login:" + brand)
				break
			}
		}
	}
	sort.Strings(flags)
	return flags, excerpt, len(flags) > 0
}

const siteReviewInstruction = `You review web pages that users of a Chinese website-hosting service publish. Decide whether a page may go online at once or must wait for a human.
Answer "review" for: phishing or imitation of a bank, payment app, government office, telecom or well-known brand (especially login / password / verification-code / card forms), scams (investment, rebates, fake jobs, loans, refunds), gambling, sexual content, malware or fake downloads, content illegal in China (politics, violence, drugs, extremism), or anything collecting other people's personal data under false pretences.
Answer "allow" for ordinary pages: personal homepages, resumes, portfolios, product or event pages, documentation, demos, games, tools, even when they contain a sign-up form of their own.
Input: JSON {"flags": [what a keyword check found], "text": "readable text of the pages"}.
Output ONLY a JSON object: {"verdict": "allow" | "review", "reason": "<short Chinese reason, at most 40 characters>"}.`

// reviewSite runs the review of a new version.
func (s *SiteHostingService) reviewSite(ctx context.Context, c SiteHostingConfig, files []siteFile) SiteReview {
	flags, excerpt, risky := checkSiteContent(files)
	review := SiteReview{Status: SiteReviewApproved, Flags: flags, Excerpt: excerpt, By: "auto"}
	if risky {
		review.Status, review.Reason = SiteReviewPending, "页面含有需要人工确认的内容："+siteFlagSummary(flags)
	}
	if model := s.reviewModel(ctx); model.ready() && excerpt != "" {
		if verdict, reason, err := s.askReviewModel(ctx, model, flags, excerpt); err == nil {
			review.By = "model"
			if verdict == "allow" {
				review.Status, review.Reason = SiteReviewApproved, ""
			} else {
				review.Status, review.Reason = SiteReviewPending, "自动审核认为需要人工确认："+reason
			}
		}
	}
	if c.ReviewAll && review.Status == SiteReviewApproved {
		review.Status, review.Reason = SiteReviewPending, "所有新内容都需要人工审核"
	}
	return review
}

var siteFlagNames = map[string]string{"password_field": "密码输入框", "card_or_id_field": "银行卡 / 身份证输入框", "gambling": "疑似赌博", "adult": "疑似色情", "fraud": "疑似诈骗", "brand_login": "知名品牌 + 登录字样"}

func siteFlagSummary(flags []string) string {
	seen := map[string]bool{}
	var parts []string
	for _, f := range flags {
		kind, _, _ := strings.Cut(f, ":")
		if name := siteFlagNames[kind]; name != "" && !seen[name] {
			seen[name] = true
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, "、")
}

type siteReviewModel struct {
	baseURL, model, apiKey string
}

func (m siteReviewModel) ready() bool { return m.baseURL != "" && m.model != "" && m.apiKey != "" }

func (s *SiteHostingService) reviewModel(ctx context.Context) siteReviewModel {
	if s.settings == nil {
		return siteReviewModel{}
	}
	values, err := s.settings.GetMultiple(ctx, []string{settingSitesReviewBaseURL, settingSitesReviewModel, settingSitesReviewAPIKey})
	if err != nil {
		return siteReviewModel{}
	}
	return siteReviewModel{baseURL: values[settingSitesReviewBaseURL], model: values[settingSitesReviewModel], apiKey: values[settingSitesReviewAPIKey]}
}

func (s *SiteHostingService) askReviewModel(ctx context.Context, m siteReviewModel, flags []string, text string) (verdict, reason string, err error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	payload := map[string]any{"flags": flags, "text": text}
	if flags == nil {
		payload["flags"] = []string{}
	}
	content, err := openAIChat(ctx, s.httpClient, m.baseURL, m.apiKey, m.model, siteReviewInstruction, payload)
	if err != nil {
		return "", "", err
	}
	start, end := strings.Index(content, "{"), strings.LastIndex(content, "}")
	if start < 0 || end <= start {
		return "", "", fmt.Errorf("review model answered without JSON")
	}
	var out struct {
		Verdict string `json:"verdict"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(content[start:end+1]), &out); err != nil {
		return "", "", err
	}
	verdict = strings.ToLower(strings.TrimSpace(out.Verdict))
	if verdict != "allow" && verdict != "review" {
		return "", "", fmt.Errorf("review model verdict %q", out.Verdict)
	}
	reason = strings.TrimSpace(out.Reason)
	if !utf8.ValidString(reason) || reason == "" {
		reason = "内容可能违规"
	}
	return verdict, truncateRunes(reason, 60), nil
}
