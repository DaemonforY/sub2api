package logredact

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Patterns for RedactContent. They target the shapes secrets usually take inside prompts, tool
// arguments and model replies; everything matched is replaced with ***.
var (
	reContentPrivateKey = regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----.*?(?:-----END [A-Z0-9 ]*PRIVATE KEY-----|$)`)
	reContentJWT        = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{5,}\.eyJ[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]*`)
	reContentAnthropic  = regexp.MustCompile(`\bsk-ant-[A-Za-z0-9_-]{8,}`)
	reContentOpenAI     = regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{16,}`)
	reContentGitHub     = regexp.MustCompile(`\b(?:gh[pousr]_[A-Za-z0-9]{16,}|github_pat_[A-Za-z0-9_]{16,})`)
	reContentAWS        = regexp.MustCompile(`\b(?:AKIA|ASIA)[0-9A-Z]{16}\b`)
	reContentSlack      = regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{8,}`)
	reContentGoogle     = regexp.MustCompile(`\bAIza[0-9A-Za-z_-]{35}`)

	// scheme://user:password@host — only the password goes (DATABASE_URL, *_DSN, git remotes …).
	reContentURLUserinfo = regexp.MustCompile(`(?i)\b([a-z][a-z0-9+.-]*://[^\s:/@'"]+:)([^\s@/'"]+)(@)`)
	// Authorization: Basic|Digest|Bearer|Token <value> (header or curl -H), value to the end of the line.
	reContentAuthHeader = regexp.MustCompile(`(?i)\b(authorization\s*[:=]\s*["']?(?:basic|digest|bearer|token)\s+)([^\r\n"']+)`)
	// Cookie / Set-Cookie header values, to the end of the line.
	reContentCookieHeader = regexp.MustCompile(`(?i)\b((?:set-)?cookie\s*:\s*)([^\r\n"']+)`)
	// mysql -uroot -pSecret (password attached to -p in a MySQL client command; -P is the port).
	reContentMySQLPassword = regexp.MustCompile(`\b((?i:mysql|mysqldump|mysqladmin|mariadb)\b[^\n]*?\s-p)(['"]?[^\s'"]+['"]?)`)
	// 密码是 xxx / 口令：xxx / 密钥为 xxx (the value is the ASCII run that follows).
	reContentCNSecret = regexp.MustCompile(`(密码|口令|密钥|秘钥)(\s*(?:是|为|:|：)?\s*)([\x21-\x7E]+)`)

	// "db_password": "value" (also inside escaped JSON: \"password\": \"value\").
	reContentSecretJSONPair = regexp.MustCompile(`(?i)((?:\\?")[A-Za-z0-9_.-]*(?:password|passwd|secret|token|api[_-]?key|apikey|access[_-]?key|private[_-]?key)[A-Za-z0-9_.-]*\\?"\s*:\s*\\?")((?:[^"\\]|\\[^"])*)(\\?")`)
	// password=value, API_KEY: value, export SECRET_TOKEN='value'
	reContentSecretAssign = regexp.MustCompile(`(?i)\b([A-Za-z0-9_.-]*(?:password|passwd|secret|token|api[_-]?key|apikey|access[_-]?key|private[_-]?key)[A-Za-z0-9_.-]*)(\s*[:=]\s*)(["']?)([^\s"',;&]+)`)

	// From securityaudit.RedactPreview (kept in sync by hand: securityaudit imports service, so it
	// can't be imported from here).
	reContentBearer = regexp.MustCompile(`(?i)\bBearer\s+[A-Za-z0-9._~+\-/]+=*`)
	reContentAPIKey = regexp.MustCompile(`(?i)\b(sk|rk|pk|api[_-]?key|token|secret|password)([-_:=\s]+)([A-Za-z0-9._~+\-/]{8,})`)
	reContentEmail  = regexp.MustCompile(`(?i)\b[A-Z0-9._%+\-]+@[A-Z0-9.\-]+\.[A-Z]{2,}\b`)
	// Phones: mainland mobile numbers and +country numbers. Narrower than RedactPreview's pattern,
	// which also swallows dates, line numbers and other digit runs that are common in code.
	reContentPhoneCN   = regexp.MustCompile(`(?:^|[^0-9])(1[3-9][0-9]{9})(?:[^0-9]|$)`)
	reContentPhoneIntl = regexp.MustCompile(`\+[0-9]{1,3}[\s-]?\(?[0-9]{1,4}\)?(?:[\s-]?[0-9]{2,4}){2,4}`)
)

// RedactContent masks secrets and personal data in free text that is about to be stored for admin
// review (prompts, tool-call arguments, model replies). It never truncates; callers trim afterwards.
// It errs on the side of masking: a few ordinary words next to "token" or "password" may be masked too.
func RedactContent(s string) string {
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, "\x00", "")
	s = reContentPrivateKey.ReplaceAllString(s, "***")
	s = reContentURLUserinfo.ReplaceAllString(s, "${1}***${3}")
	s = reContentAuthHeader.ReplaceAllString(s, "${1}***")
	s = reContentCookieHeader.ReplaceAllString(s, "${1}***")
	s = reContentMySQLPassword.ReplaceAllString(s, "${1}***")
	s = reContentCNSecret.ReplaceAllString(s, "${1}${2}***")
	s = reContentJWT.ReplaceAllString(s, "***")
	s = reContentAnthropic.ReplaceAllString(s, "***")
	s = reContentOpenAI.ReplaceAllString(s, "***")
	s = reContentGitHub.ReplaceAllString(s, "***")
	s = reContentAWS.ReplaceAllString(s, "***")
	s = reContentSlack.ReplaceAllString(s, "***")
	s = reContentGoogle.ReplaceAllString(s, "***")
	s = reGOCSPX.ReplaceAllString(s, "GOCSPX-***")
	s = reContentSecretJSONPair.ReplaceAllString(s, `${1}***${3}`)
	s = reContentSecretAssign.ReplaceAllStringFunc(s, func(match string) string {
		sub := reContentSecretAssign.FindStringSubmatch(match)
		if len(sub) != 5 || strings.HasPrefix(sub[4], "***") {
			return match
		}
		// Token counts (max_tokens: 4096, input_tokens=12) are not secrets.
		if key := strings.ToLower(sub[1]); strings.HasSuffix(key, "tokens") || strings.Contains(key, "tokenizer") {
			return match
		}
		return sub[1] + sub[2] + sub[3] + "***"
	})
	s = reContentBearer.ReplaceAllString(s, "Bearer ***")
	s = reContentAPIKey.ReplaceAllStringFunc(s, func(match string) string {
		sub := reContentAPIKey.FindStringSubmatch(match)
		if len(sub) != 4 {
			return "***"
		}
		// Ordinary prose ("the token limit") has no digits and is short; secrets rarely are.
		if !strings.ContainsAny(sub[3], "0123456789") && len(sub[3]) < 16 {
			return match
		}
		return sub[1] + sub[2] + "***"
	})
	s = reContentEmail.ReplaceAllString(s, "***@***")
	s = reContentPhoneCN.ReplaceAllStringFunc(s, func(match string) string {
		sub := reContentPhoneCN.FindStringSubmatch(match)
		if len(sub) != 2 {
			return match
		}
		return strings.Replace(match, sub[1], "***PHONE***", 1)
	})
	s = reContentPhoneIntl.ReplaceAllString(s, "***PHONE***")
	// RedactText re-encodes JSON documents (sorting keys, escaping <>); JSON text was already handled
	// by the key/value patterns above, so only free text goes through it.
	if !json.Valid([]byte(strings.TrimSpace(s))) {
		s = redactTextKeepSpace(s)
	}
	return s
}

// redactTextKeepSpace runs RedactText but keeps the text's leading/trailing whitespace.
func redactTextKeepSpace(s string) string {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return s
	}
	start := strings.Index(s, trimmed)
	return s[:start] + RedactText(trimmed) + s[start+len(trimmed):]
}
