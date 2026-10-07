package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"golang.org/x/net/html"
)

// The 公众号 editor at /editor: formatting is front-end only; this service is the server side —
// WeChat (access token, image / cover upload, draft), importing an article from its link, and AI
// rewriting / images on the learner's own key. AppID / AppSecret come with each request and are
// never stored or logged; access tokens are cached in memory under a hash of the pair.

const (
	editorWechatAPI        = "https://api.weixin.qq.com"
	editorContentImageMax  = 1 << 20  // media/uploadimg: jpg / png under 1MB
	editorCoverImageMax    = 10 << 20 // material/add_material: up to 10MB
	editorArticlePageMax   = 8 << 20
	editorProxyImageMax    = 10 << 20
	editorDraftContentMax  = 2 << 20
	editorAITextMax        = 20000
	editorAIInstructionMax = 500
	editorAIPromptMax      = 2000
	editorAIMaxTokens      = 16000
	editorImageModel       = "gpt-image-2"
	editorUserAgent        = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0 Safari/537.36"
)

var (
	editorAppIDRe     = regexp.MustCompile(`^wx[0-9a-zA-Z]{16}$`)
	editorSecretRe    = regexp.MustCompile(`^[0-9a-zA-Z]{32}$`)
	editorInvalidIPRe = regexp.MustCompile(`invalid ip ([0-9a-fA-F:.]+)`)
	editorImageHosts  = map[string]bool{"mmbiz.qpic.cn": true, "mmbiz.qlogo.cn": true}
	editorImageSizes  = map[string]bool{"1536x1024": true, "1024x1536": true, "1024x1024": true}

	ErrEditorCredentials = infraerrors.BadRequest("EDITOR_WECHAT_CREDENTIALS", "请先在设置里填写正确的公众号 AppID 和 AppSecret（Invalid AppID / AppSecret）")
	ErrEditorArticleURL  = infraerrors.BadRequest("EDITOR_ARTICLE_URL", "只支持 mp.weixin.qq.com 的公众号文章链接（Only mp.weixin.qq.com article links）")
	ErrEditorImageURL    = infraerrors.BadRequest("EDITOR_IMAGE_URL", "只能代理公众号图片地址（Only WeChat image hosts）")
	ErrEditorAIText      = infraerrors.BadRequest("EDITOR_AI_TEXT", "请选中或输入要处理的内容，最多 20000 字（Text 1–20000 characters）")
	ErrEditorAIAction    = infraerrors.BadRequest("EDITOR_AI_ACTION", "不支持的 AI 操作（Unknown action）")
	ErrEditorAIPrompt    = infraerrors.BadRequest("EDITOR_AI_PROMPT", "画面描述需要 1–2000 字，尺寸只能是 1536x1024、1024x1536 或 1024x1024（Invalid prompt or size）")
)

func editorErr(reason, msg string) error { return infraerrors.BadRequest(reason, msg) }

type editorToken struct {
	token string
	exp   time.Time
}

type EditorService struct {
	learn     *LearnService
	client    *http.Client // WeChat and article pages
	aiClient  *http.Client // long AI calls on the gateway
	wechatAPI string
	now       func() time.Time

	mu     sync.Mutex
	tokens map[string]editorToken
}

func NewEditorService(learn *LearnService) *EditorService {
	return &EditorService{learn: learn, client: &http.Client{Timeout: 30 * time.Second}, aiClient: &http.Client{Timeout: 5 * time.Minute},
		wechatAPI: editorWechatAPI, now: time.Now, tokens: map[string]editorToken{}}
}

// ---------------------------------------------------------------------------------------------
// WeChat

type EditorCredentials struct {
	AppID  string `json:"appid" form:"appid"`
	Secret string `json:"secret" form:"secret"`
}

func (c EditorCredentials) valid() bool {
	return editorAppIDRe.MatchString(strings.TrimSpace(c.AppID)) && editorSecretRe.MatchString(strings.TrimSpace(c.Secret))
}

func (c EditorCredentials) cacheKey() string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(c.AppID) + ":" + strings.TrimSpace(c.Secret)))
	return hex.EncodeToString(sum[:])
}

type wechatReply struct {
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	MediaID     string `json:"media_id"`
	URL         string `json:"url"`
}

// editorDevKeyPath is where a 公众号's AppID, AppSecret and API IP whitelist live since WeChat moved 「开发接口管理」 out
// of the 公众平台 (the old 「设置与开发 → 基本配置」 entry is gone for many accounts).
const editorDevKeyPath = "微信开发者平台（developers.weixin.qq.com/platform）「我的业务 → 公众号 → 基础信息 → 开发密钥」"

// wechatError turns a WeChat error code into an actionable Chinese message.
func wechatError(code int, msg string) error {
	switch code {
	case 40164, 61004:
		ip := "本站服务器"
		if m := editorInvalidIPRe.FindStringSubmatch(msg); m != nil {
			ip = m[1]
		}
		return editorErr("EDITOR_WECHAT_IP", fmt.Sprintf("服务器 IP %s 不在公众号的 API IP 白名单里：到%s的「API IP 白名单」添加这个 IP，保存后等几分钟再试（IP %s not whitelisted）", ip, editorDevKeyPath, ip))
	case 40001, 40125, 41004:
		return editorErr("EDITOR_WECHAT_SECRET", "AppSecret 不正确，或已经重置过：到"+editorDevKeyPath+"重新获取后填写（Invalid AppSecret）")
	case 40243:
		return editorErr("EDITOR_WECHAT_SECRET", "AppSecret 已被冻结：到"+editorDevKeyPath+"解冻，约 10 分钟后生效（AppSecret frozen）")
	case 40013, 41002:
		return editorErr("EDITOR_WECHAT_APPID", "AppID 不正确：到微信开发者平台「我的业务 → 公众号 → 基础信息」复制 wx 开头的 18 位 AppID（Invalid AppID）")
	case 48001:
		return editorErr("EDITOR_WECHAT_UNAUTHORIZED", "这个公众号没有该接口权限：个人或未认证的公众号可能用不了草稿箱或素材接口，可在微信开发者平台「我的业务 → 公众号 → 接口管理 → 接口权限与额度」查看（API unauthorized）")
	case 45009, 45011, 45047:
		return editorErr("EDITOR_WECHAT_LIMIT", "调用微信接口太频繁或已到今日上限，请稍后再试（Rate limited）")
	case 40007:
		return editorErr("EDITOR_WECHAT_MEDIA", "封面素材无效，请重新上传封面（Invalid media_id）")
	case 40005, 40009, 40006:
		return editorErr("EDITOR_WECHAT_FILE", "图片格式或大小不符合微信要求（正文图片 jpg/png 小于 1MB，封面小于 10MB）（Invalid image）")
	case 45002, 45003, 45004:
		return editorErr("EDITOR_WECHAT_TOO_LONG", "标题、摘要或正文超出了微信的长度限制，请删减后再试（Too long）")
	case 53402, 53403, 53404:
		return editorErr("EDITOR_WECHAT_CONTENT", "内容没有通过微信检查，请修改后再试（Content rejected）")
	}
	return editorErr("EDITOR_WECHAT", fmt.Sprintf("微信接口返回错误 %d：%s（WeChat error）", code, msg))
}

func (s *EditorService) dropToken(c EditorCredentials) {
	s.mu.Lock()
	delete(s.tokens, c.cacheKey())
	s.mu.Unlock()
}

func (s *EditorService) accessToken(ctx context.Context, c EditorCredentials) (string, error) {
	if !c.valid() {
		return "", ErrEditorCredentials
	}
	key := c.cacheKey()
	s.mu.Lock()
	if t, ok := s.tokens[key]; ok && s.now().Before(t.exp) {
		s.mu.Unlock()
		return t.token, nil
	}
	s.mu.Unlock()
	body, _ := json.Marshal(map[string]any{"grant_type": "client_credential", "appid": strings.TrimSpace(c.AppID),
		"secret": strings.TrimSpace(c.Secret), "force_refresh": false})
	var r wechatReply
	if err := s.wechatJSON(ctx, "/cgi-bin/stable_token", "", body, &r); err != nil {
		return "", err
	}
	if r.ErrCode != 0 || r.AccessToken == "" {
		return "", wechatError(r.ErrCode, r.ErrMsg)
	}
	ttl := time.Duration(max(r.ExpiresIn-300, 60)) * time.Second
	s.mu.Lock()
	s.tokens[key] = editorToken{token: r.AccessToken, exp: s.now().Add(ttl)}
	s.mu.Unlock()
	return r.AccessToken, nil
}

func (s *EditorService) wechatJSON(ctx context.Context, path, token string, body []byte, out *wechatReply) error {
	u := s.wechatAPI + path
	if token != "" {
		u += "?access_token=" + url.QueryEscape(token)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	return s.wechatDo(req, out)
}

func (s *EditorService) wechatDo(req *http.Request, out *wechatReply) error {
	resp, err := s.client.Do(req)
	if err != nil {
		return editorErr("EDITOR_WECHAT_NETWORK", "连接微信服务器失败，请稍后再试（WeChat unreachable）")
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil || json.Unmarshal(raw, out) != nil {
		return editorErr("EDITOR_WECHAT_NETWORK", "微信服务器返回了无法识别的内容，请稍后再试（Bad WeChat response）")
	}
	return nil
}

// withToken runs call with an access token, refreshing once when WeChat says it has expired.
func (s *EditorService) withToken(ctx context.Context, c EditorCredentials, call func(token string) (*wechatReply, error)) (*wechatReply, error) {
	for attempt := 0; ; attempt++ {
		token, err := s.accessToken(ctx, c)
		if err != nil {
			return nil, err
		}
		r, err := call(token)
		if err != nil {
			return nil, err
		}
		if (r.ErrCode == 40001 || r.ErrCode == 42001 || r.ErrCode == 40014) && attempt == 0 {
			s.dropToken(c)
			continue
		}
		if r.ErrCode != 0 {
			return nil, wechatError(r.ErrCode, r.ErrMsg)
		}
		return r, nil
	}
}

// WechatCheck verifies the AppID / AppSecret (and that this server's IP is whitelisted).
func (s *EditorService) WechatCheck(ctx context.Context, c EditorCredentials) error {
	_, err := s.accessToken(ctx, c)
	return err
}

type EditorUpload struct {
	URL     string `json:"url,omitempty"`
	MediaID string `json:"media_id,omitempty"`
}

// WechatUpload uploads a body image (media/uploadimg → URL) or a cover (permanent material → media_id).
func (s *EditorService) WechatUpload(ctx context.Context, c EditorCredentials, kind, filename string, data []byte) (*EditorUpload, error) {
	mime := http.DetectContentType(data)
	switch kind {
	case "content":
		if len(data) == 0 || len(data) >= editorContentImageMax || (mime != "image/jpeg" && mime != "image/png") {
			return nil, editorErr("EDITOR_WECHAT_FILE", "正文图片要是 jpg 或 png，且小于 1MB（Body images: jpg/png under 1MB）")
		}
	case "cover":
		if len(data) == 0 || len(data) > editorCoverImageMax || (mime != "image/jpeg" && mime != "image/png" && mime != "image/gif" && mime != "image/bmp") {
			return nil, editorErr("EDITOR_WECHAT_FILE", "封面要是 jpg、png、gif 或 bmp，且不超过 10MB（Cover: image up to 10MB）")
		}
	default:
		return nil, editorErr("EDITOR_WECHAT_FILE", "kind 只能是 content 或 cover（Invalid kind）")
	}
	name := editorFilename(filename, mime)
	path := "/cgi-bin/media/uploadimg"
	query := ""
	if kind == "cover" {
		path, query = "/cgi-bin/material/add_material", "&type=image"
	}
	r, err := s.withToken(ctx, c, func(token string) (*wechatReply, error) {
		var buf bytes.Buffer
		w := multipart.NewWriter(&buf)
		part, err := w.CreateFormFile("media", name)
		if err != nil {
			return nil, err
		}
		if _, err := part.Write(data); err != nil {
			return nil, err
		}
		if err := w.Close(); err != nil {
			return nil, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.wechatAPI+path+"?access_token="+url.QueryEscape(token)+query, &buf)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", w.FormDataContentType())
		var out wechatReply
		return &out, s.wechatDo(req, &out)
	})
	if err != nil {
		return nil, err
	}
	return &EditorUpload{URL: r.URL, MediaID: r.MediaID}, nil
}

func editorFilename(name, mime string) string {
	ext := map[string]string{"image/jpeg": ".jpg", "image/png": ".png", "image/gif": ".gif", "image/bmp": ".bmp"}[mime]
	base := strings.TrimSuffix(name, path.Ext(name))
	if base == "" || strings.ContainsAny(base, `/\"`) || utf8.RuneCountInString(base) > 60 {
		base = "image"
	}
	return base + ext
}

type EditorArticle struct {
	Title              string `json:"title"`
	Author             string `json:"author"`
	Digest             string `json:"digest"`
	Content            string `json:"content"`
	ContentSourceURL   string `json:"content_source_url"`
	ThumbMediaID       string `json:"thumb_media_id"`
	NeedOpenComment    bool   `json:"need_open_comment"`
	OnlyFansCanComment bool   `json:"only_fans_can_comment"`
}

type EditorDraftInput struct {
	EditorCredentials
	Article EditorArticle `json:"article"`
}

// WechatDraft creates a draft in the 公众号's draft box and returns its media_id.
func (s *EditorService) WechatDraft(ctx context.Context, in EditorDraftInput) (string, error) {
	a := in.Article
	a.Title, a.Author, a.Digest = strings.TrimSpace(a.Title), strings.TrimSpace(a.Author), strings.TrimSpace(a.Digest)
	if a.Title == "" || strings.TrimSpace(a.Content) == "" {
		return "", editorErr("EDITOR_DRAFT", "标题和正文不能为空（Title and content required）")
	}
	if strings.TrimSpace(a.ThumbMediaID) == "" {
		return "", editorErr("EDITOR_DRAFT", "公众号草稿必须有封面，请先选择或生成一张封面（Cover required）")
	}
	if len(a.Content) > editorDraftContentMax {
		return "", editorErr("EDITOR_WECHAT_TOO_LONG", "正文太长了，请删减后再试（Content too long）")
	}
	if u := strings.TrimSpace(a.ContentSourceURL); u != "" && !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "http://") {
		return "", editorErr("EDITOR_DRAFT", "原文链接要以 http:// 或 https:// 开头（Invalid source URL）")
	}
	type wxArticle struct {
		Title              string `json:"title"`
		Author             string `json:"author,omitempty"`
		Digest             string `json:"digest,omitempty"`
		Content            string `json:"content"`
		ContentSourceURL   string `json:"content_source_url,omitempty"`
		ThumbMediaID       string `json:"thumb_media_id"`
		NeedOpenComment    int    `json:"need_open_comment"`
		OnlyFansCanComment int    `json:"only_fans_can_comment"`
	}
	b2i := func(b bool) int {
		if b {
			return 1
		}
		return 0
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false) // WeChat stores the body as sent; keep "<" etc. literal
	if err := enc.Encode(map[string]any{"articles": []wxArticle{{Title: a.Title, Author: a.Author, Digest: a.Digest, Content: a.Content,
		ContentSourceURL: strings.TrimSpace(a.ContentSourceURL), ThumbMediaID: strings.TrimSpace(a.ThumbMediaID),
		NeedOpenComment: b2i(a.NeedOpenComment), OnlyFansCanComment: b2i(a.OnlyFansCanComment)}}}); err != nil {
		return "", err
	}
	body := buf.Bytes()
	r, err := s.withToken(ctx, in.EditorCredentials, func(token string) (*wechatReply, error) {
		var out wechatReply
		return &out, s.wechatJSON(ctx, "/cgi-bin/draft/add", token, body, &out)
	})
	if err != nil {
		return "", err
	}
	return r.MediaID, nil
}

// ---------------------------------------------------------------------------------------------
// Importing an article

type EditorImported struct {
	Title  string   `json:"title"`
	Author string   `json:"author"`
	HTML   string   `json:"html"`
	Images []string `json:"images"`
}

func editorArticleURL(raw string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || u.Host != "mp.weixin.qq.com" || !strings.HasPrefix(u.Path, "/s") {
		return nil, ErrEditorArticleURL
	}
	u.Scheme = "https"
	u.Fragment = ""
	return u, nil
}

func (s *EditorService) get(ctx context.Context, u string, limit int64) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", editorUserAgent)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, "", editorErr("EDITOR_FETCH", "打开链接失败，请稍后再试（Fetch failed）")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, "", editorErr("EDITOR_FETCH", fmt.Sprintf("打开链接失败：HTTP %d（Fetch failed）", resp.StatusCode))
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, "", editorErr("EDITOR_FETCH", "读取内容中断，请稍后再试（Read failed）")
	}
	if int64(len(data)) > limit {
		return nil, "", editorErr("EDITOR_FETCH", "内容太大了（Too large）")
	}
	return data, resp.Header.Get("Content-Type"), nil
}

// ImportArticle fetches a 公众号 article and returns its title, author and body HTML (images point
// at their original addresses; the editor pulls them through FetchImage).
func (s *EditorService) ImportArticle(ctx context.Context, raw string) (*EditorImported, error) {
	u, err := editorArticleURL(raw)
	if err != nil {
		return nil, err
	}
	page, _, err := s.get(ctx, u.String(), editorArticlePageMax)
	if err != nil {
		return nil, err
	}
	return parseWechatArticle(page)
}

func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func setAttr(n *html.Node, key, val string) {
	for i, a := range n.Attr {
		if a.Key == key {
			n.Attr[i].Val = val
			return
		}
	}
	n.Attr = append(n.Attr, html.Attribute{Key: key, Val: val})
}

func parseWechatArticle(page []byte) (*EditorImported, error) {
	doc, err := html.Parse(bytes.NewReader(page))
	if err != nil {
		return nil, editorErr("EDITOR_FETCH", "页面解析失败（Parse failed）")
	}
	out := &EditorImported{Images: []string{}}
	var content, titleNode *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch {
			case n.Data == "meta" && attr(n, "property") == "og:title" && out.Title == "":
				out.Title = strings.TrimSpace(attr(n, "content"))
			case n.Data == "meta" && attr(n, "name") == "author" && out.Author == "":
				out.Author = strings.TrimSpace(attr(n, "content"))
			case attr(n, "id") == "js_content" && content == nil:
				content = n
			case attr(n, "id") == "activity-name" && titleNode == nil:
				titleNode = n
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	if out.Title == "" && titleNode != nil {
		out.Title = strings.TrimSpace(textOf(titleNode))
	}
	if content == nil {
		return nil, editorErr("EDITOR_FETCH", "没有找到正文：链接可能已失效、被删除，或需要在微信里打开（No article body）")
	}
	seen := map[string]bool{}
	var fix func(*html.Node)
	fix = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "script" || n.Data == "style" {
				return
			}
			if n.Data == "img" {
				src := attr(n, "data-src")
				if src == "" {
					src = attr(n, "src")
				}
				if strings.HasPrefix(src, "//") {
					src = "https:" + src
				}
				if src != "" {
					setAttr(n, "src", src)
					if !seen[src] {
						seen[src] = true
						out.Images = append(out.Images, src)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			fix(c)
		}
	}
	fix(content)
	var buf bytes.Buffer
	for c := content.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && (c.Data == "script" || c.Data == "style") {
			continue
		}
		if err := html.Render(&buf, c); err != nil {
			return nil, editorErr("EDITOR_FETCH", "页面解析失败（Parse failed）")
		}
	}
	out.HTML = buf.String()
	return out, nil
}

func textOf(n *html.Node) string {
	if n.Type == html.TextNode {
		return n.Data
	}
	text := ""
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		text += textOf(c)
	}
	return text
}

// FetchImage proxies a WeChat image (they refuse other sites' Referer).
func (s *EditorService) FetchImage(ctx context.Context, raw string) ([]byte, string, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") || !editorImageHosts[u.Hostname()] {
		return nil, "", ErrEditorImageURL
	}
	u.Scheme = "https"
	data, _, err := s.get(ctx, u.String(), editorProxyImageMax)
	if err != nil {
		return nil, "", err
	}
	mime := http.DetectContentType(data)
	if !strings.HasPrefix(mime, "image/") {
		return nil, "", editorErr("EDITOR_FETCH", "这个地址不是图片（Not an image）")
	}
	return data, mime, nil
}

// ---------------------------------------------------------------------------------------------
// AI on the learner's own key

const editorAIBase = "你是资深的中文公众号编辑。输入是 Markdown。保持 Markdown 结构：标题、列表、引用、代码块原样保留，图片 ![..](..) 和链接的地址一个字也不要改。只输出结果本身，不要解释、不要加前言后语。"

var editorAIActions = map[string]string{
	"polish":       "润色这段内容：修正错别字和病句，让表达更通顺、更有节奏，保持原意、事实和作者的语气，不加入原文没有的事实。",
	"shorten":      "精简这段内容：保留核心观点和关键事实，删掉重复和冗余，篇幅压缩到原来的一半左右。",
	"expand":       "扩写这段内容：补充解释、例子或过渡，让内容更充实；不要编造具体数据、引语或事实，篇幅约为原来的 1.5–2 倍。",
	"title":        "根据文章内容拟 5 个公众号标题，每行一个，不要编号和引号；准确概括内容，不夸大、不做标题党。",
	"digest":       "为这篇文章写一段公众号摘要，不超过 120 字，纯文本，不要 Markdown。",
	"outline":      "根据内容列出文章提纲，用 Markdown 多级列表。",
	"image_prompt": "根据这段内容写一段适合 AI 画图的画面描述（中文，80–150 字）：主体、场景、构图、光线、风格；画面里不要出现文字和 Logo。只输出描述。",
	"custom":       "按用户的要求修改这段内容。用户的要求：",
}

type EditorAITextInput struct {
	KeyID       int64  `json:"key_id"`
	Action      string `json:"action"`
	Text        string `json:"text"`
	Instruction string `json:"instruction"`
	Title       string `json:"title"`
}

// AIText streams an AI rewrite of the text on one of the user's keys.
func (s *EditorService) AIText(ctx context.Context, userID int64, in EditorAITextInput, onDelta func(string) error) error {
	task, ok := editorAIActions[in.Action]
	if !ok {
		return ErrEditorAIAction
	}
	text := strings.TrimSpace(in.Text)
	if n := utf8.RuneCountInString(text); n == 0 || n > editorAITextMax {
		return ErrEditorAIText
	}
	if in.Action == "custom" {
		ins := strings.TrimSpace(in.Instruction)
		if ins == "" || utf8.RuneCountInString(ins) > editorAIInstructionMax {
			return editorErr("EDITOR_AI_INSTRUCTION", "请写下修改要求，最多 500 字（Instruction 1–500 characters）")
		}
		task += ins
	}
	key, model, err := s.aiKey(ctx, userID, in.KeyID)
	if err != nil {
		return err
	}
	user := text
	if t := strings.TrimSpace(in.Title); t != "" && utf8.RuneCountInString(t) <= 200 {
		user = "文章标题：" + t + "\n\n" + text
	}
	msgs := []LearnMessage{{Role: "system", Content: editorAIBase + "\n任务：" + task}, {Role: "user", Content: user}}
	_, err = streamChatCompletion(ctx, s.aiClient, s.learn.gatewayURL, "hivegpt-editor/1", key, model, msgs, editorAIMaxTokens, onDelta)
	return err
}

func (s *EditorService) aiKey(ctx context.Context, userID, keyID int64) (string, string, error) {
	if keyID <= 0 {
		return "", "", ErrLearnOwnKeyRequired
	}
	k, err := s.learn.ownKey(ctx, userID, keyID)
	if err != nil {
		return "", "", err
	}
	st, _ := s.learn.loadSettings(ctx)
	model := st.Model
	if model == "" {
		model = "gpt-5.5"
	}
	return k.Key, model, nil
}

type EditorAIImageInput struct {
	KeyID  int64  `json:"key_id"`
	Prompt string `json:"prompt"`
	Size   string `json:"size"`
}

type EditorAIImage struct {
	B64JSON string `json:"b64_json"`
	Mime    string `json:"mime"`
}

// AIImage draws an image with gpt-image-2 on one of the user's keys.
func (s *EditorService) AIImage(ctx context.Context, userID int64, in EditorAIImageInput) (*EditorAIImage, error) {
	prompt := strings.TrimSpace(in.Prompt)
	if n := utf8.RuneCountInString(prompt); n == 0 || n > editorAIPromptMax || !editorImageSizes[in.Size] {
		return nil, ErrEditorAIPrompt
	}
	key, _, err := s.aiKey(ctx, userID, in.KeyID)
	if err != nil {
		return nil, err
	}
	body, _ := json.Marshal(map[string]any{"model": editorImageModel, "prompt": prompt + "\n画面里不要出现任何文字、字母、Logo 和水印。",
		"size": in.Size, "quality": "medium", "n": 1})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.learn.gatewayURL+"/v1/images/generations", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "hivegpt-editor/1")
	resp, err := s.aiClient.Do(req)
	if err != nil {
		return nil, errLearnRunFailed("连接画图服务超时")
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 40<<20))
	if err != nil {
		return nil, errLearnRunFailed("读取图片中断")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errLearnRunFailed(upstreamErrorText(raw, resp.StatusCode))
	}
	var r struct {
		Data []struct {
			B64JSON string `json:"b64_json"`
			URL     string `json:"url"`
		} `json:"data"`
	}
	if json.Unmarshal(raw, &r) != nil || len(r.Data) == 0 {
		return nil, errLearnRunFailed("画图服务没有返回图片")
	}
	b64 := r.Data[0].B64JSON
	if b64 == "" && r.Data[0].URL != "" {
		img, _, err := s.get(ctx, r.Data[0].URL, 40<<20)
		if err != nil {
			return nil, err
		}
		b64 = base64.StdEncoding.EncodeToString(img)
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil || len(data) == 0 {
		return nil, errLearnRunFailed("画图服务返回的图片无法识别")
	}
	return &EditorAIImage{B64JSON: b64, Mime: http.DetectContentType(data)}, nil
}
