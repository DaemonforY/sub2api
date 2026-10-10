package service

import (
	"encoding/json"
	"net/url"
	"regexp"
	"strings"
)

// What GEO 监测 looks for in an answer: our brand, the links the assistant cited (and which of them
// are ours), and competitors.

const geoMaxCitedURLs = 50

// GeoDetection is what an answer says about us.
type GeoDetection struct {
	Mentioned   bool     `json:"mentioned"`
	CitedURLs   []string `json:"cited_urls"`
	OurURLs     []string `json:"our_urls"`
	Competitors []string `json:"competitors"`
}

// geoURLPattern finds http(s) links in answer text. Stops at whitespace, quotes, brackets and
// full-width punctuation, so Markdown links and Chinese sentences don't swallow what follows.
var geoURLPattern = regexp.MustCompile(`https?://[^\s<>"'()\[\]{}，。、；：！？（）【】《》「」“”‘’]+`)

// DetectGeo checks an answer (and the links cited alongside it) for brand and competitor keywords.
// cited are the links the engine returned in structured form; links in the text are added.
func DetectGeo(answer string, cited []string, brand, competitors []string) GeoDetection {
	urls := mergeGeoURLs(cited, geoURLsInText(answer))
	lowerAnswer := strings.ToLower(answer)
	lowerURLs := strings.ToLower(strings.Join(urls, "\n"))
	contains := func(kw string) bool {
		kw = strings.ToLower(strings.TrimSpace(kw))
		return kw != "" && (strings.Contains(lowerAnswer, kw) || strings.Contains(lowerURLs, kw))
	}
	out := GeoDetection{CitedURLs: urls, OurURLs: []string{}, Competitors: []string{}}
	for _, kw := range brand {
		if contains(kw) {
			out.Mentioned = true
			break
		}
	}
	domains := geoBrandDomains(brand)
	for _, u := range urls {
		if geoURLIsOurs(u, domains) {
			out.OurURLs = append(out.OurURLs, u)
		}
	}
	for _, kw := range competitors {
		if contains(kw) {
			out.Competitors = append(out.Competitors, strings.TrimSpace(kw))
		}
	}
	return out
}

// geoBrandDomains: the brand keywords that look like domains ("hivegpt.cn", "https://hivegpt.cn/").
func geoBrandDomains(brand []string) []string {
	var out []string
	for _, kw := range brand {
		d := strings.ToLower(strings.TrimSpace(kw))
		if !strings.Contains(d, ".") {
			continue
		}
		if strings.Contains(d, "://") {
			if u, err := url.Parse(d); err == nil {
				d = u.Hostname()
			}
		}
		d = strings.Trim(strings.SplitN(d, "/", 2)[0], ".")
		d = strings.TrimPrefix(d, "www.")
		if d != "" && strings.Contains(d, ".") {
			out = append(out, d)
		}
	}
	return out
}

func geoURLIsOurs(raw string, domains []string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "" {
		return false
	}
	for _, d := range domains {
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

func geoURLsInText(text string) []string {
	found := geoURLPattern.FindAllString(text, -1)
	out := make([]string, 0, len(found))
	for _, u := range found {
		u = strings.TrimRight(u, ".,;:!?*_~`")
		if len(u) > len("https://") {
			out = append(out, u)
		}
	}
	return out
}

// mergeGeoURLs keeps the http(s) links in order, without duplicates, at most geoMaxCitedURLs.
func mergeGeoURLs(lists ...[]string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, list := range lists {
		for _, u := range list {
			u = strings.TrimSpace(u)
			lu := strings.ToLower(u)
			if !strings.HasPrefix(lu, "http://") && !strings.HasPrefix(lu, "https://") || len(u) > 2000 || seen[u] {
				continue
			}
			seen[u] = true
			out = append(out, u)
			if len(out) >= geoMaxCitedURLs {
				return out
			}
		}
	}
	return out
}

// geoChatResponse is the part of a chat completion GEO 监测 reads. Search-capable engines put their
// sources in different places: Perplexity's top-level citations / search_results, OpenAI-style
// annotations on the message.
type geoChatResponse struct {
	Choices []struct {
		Message struct {
			Content     json.RawMessage `json:"content"`
			Annotations []struct {
				URL         string `json:"url"`
				URLCitation struct {
					URL string `json:"url"`
				} `json:"url_citation"`
			} `json:"annotations"`
		} `json:"message"`
	} `json:"choices"`
	Citations     []json.RawMessage `json:"citations"`
	SearchResults []struct {
		URL string `json:"url"`
	} `json:"search_results"`
	WebSearch []struct {
		Link string `json:"link"`
	} `json:"web_search"` // 智谱 GLM web_search tool results
}

// parseGeoChatResponse returns the answer text and the structured citations of a chat completion.
func parseGeoChatResponse(body []byte) (answer string, citations []string, err error) {
	var r geoChatResponse
	if err := json.Unmarshal(body, &r); err != nil {
		return "", nil, err
	}
	if len(r.Choices) > 0 {
		m := r.Choices[0].Message
		answer = geoContentText(m.Content)
		for _, a := range m.Annotations {
			if a.URLCitation.URL != "" {
				citations = append(citations, a.URLCitation.URL)
			} else if a.URL != "" {
				citations = append(citations, a.URL)
			}
		}
	}
	for _, c := range r.Citations {
		var s string
		if json.Unmarshal(c, &s) == nil {
			citations = append(citations, s)
			continue
		}
		var obj struct {
			URL string `json:"url"`
		}
		if json.Unmarshal(c, &obj) == nil && obj.URL != "" {
			citations = append(citations, obj.URL)
		}
	}
	for _, sr := range r.SearchResults {
		if sr.URL != "" {
			citations = append(citations, sr.URL)
		}
	}
	for _, w := range r.WebSearch {
		if w.Link != "" {
			citations = append(citations, w.Link)
		}
	}
	return answer, citations, nil
}

// geoContentText: message content is a string, or (some engines) an array of {type, text} parts.
func geoContentText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var parts []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &parts) == nil {
		var b strings.Builder
		for _, p := range parts {
			_, _ = b.WriteString(p.Text)
		}
		return b.String()
	}
	return ""
}

// buildGeoRequestBody is the chat completion request: the engine's extra_body with model, the single
// user message and stream=false on top — extra_body can add search switches but never replace those.
func buildGeoRequestBody(extra json.RawMessage, model, question string) ([]byte, error) {
	body := map[string]any{}
	if len(extra) > 0 && string(extra) != "null" {
		if err := json.Unmarshal(extra, &body); err != nil || body == nil {
			return nil, errGeoBadExtraBody
		}
	}
	body["model"] = model
	body["messages"] = []map[string]string{{"role": "user", "content": question}}
	body["stream"] = false
	return json.Marshal(body)
}
