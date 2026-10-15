//go:build unit

package logredact

import (
	"strings"
	"testing"
)

func TestRedactContent_Secrets(t *testing.T) {
	cases := []struct {
		name   string
		in     string
		leak   string
		expect string
	}{
		{"openai key", "use sk-" + "proj-AbCdEfGhIjKlMnOpQrStUv123 please", "AbCdEfGhIjKlMnOp", "***"},
		{"anthropic key", "key sk-" + "ant-api03-abcdefghijklmnop_QRST", "api03-abcdefghij", "***"},
		{"github pat", "gh" + "p_abcdefghijklmnopqrstuvwxyz0123456789", "abcdefghijklmnop", "***"},
		{"github fine grained", "github" + "_pat_11ABCDEFG0123456789_abcdefghijkl", "11ABCDEFG", "***"},
		{"aws", "AKIAIOSFODNN7EXAMPLE is my id", "IOSFODNN7", "***"},
		{"slack", "xo" + "xb-1234567890-abcdefghij", "1234567890-abc", "***"},
		{"jwt", "token eyJhbGciOiJIUzI1NiJ9." + "eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U", "dozjgNryP4J3", "***"},
		{"private key", "-----BEGIN RSA " + "PRIVATE KEY-----\nMIIEowIBAAKCAQEA\nabc\n-----END RSA PRIVATE KEY-----\nafter", "MIIEowIBAAKCAQEA", "after"},
		{"json pair", `{"db_password": "hunter2!", "user": "bob"}`, "hunter2", `"db_password": "***"`},
		{"escaped json pair", `{"cmd":"{\"api_key\": \"abc123secret\"}"}`, "abc123secret", `\"api_key\": \"***\"`},
		{"assign", "export OPENAI_API_KEY=abcdef0123456789 && run", "abcdef0123456789", "OPENAI_API_KEY=***"},
		{"assign quoted", "password: 'p@ssw0rd'", "p@ssw0rd", "password: "},
		{"bearer", "Authorization: Bearer abc.def.ghi123", "abc.def.ghi123", "Bearer ***"},
		{"email", "mail me at alice@example.com", "alice@example.com", "***@***"},
		{"cn phone", "电话13812345678谢谢", "13812345678", "***PHONE***"},
		{"intl phone", "call +1 415 555 2671 now", "555 2671", "***PHONE***"},
		{"url userinfo", "connect to postgres://admin:Sup3rS3cret@localhost:5432/app ok", "Sup3rS3cret", "postgres://admin:***@localhost:5432/app"},
		{"database url env", "DATABASE_URL=postgres://admin:Sup3rS3cret@db:5432/app", "Sup3rS3cret", "DATABASE_URL=postgres://admin:***@db:5432/app"},
		{"basic auth", "curl -H 'Authorization: Basic YWRtaW46cGFzc3dvcmQ=' https://x", "YWRtaW46cGFzc3dvcmQ", "Authorization: Basic ***"},
		{"cookie", "Cookie: session=abcdef1234567890abcdef", "abcdef1234567890abcdef", "Cookie: ***"},
		{"set-cookie", "Set-Cookie: sid=zz99887766; Path=/", "zz99887766", "Set-Cookie: ***"},
		{"mysql -p", "mysql -uroot -pS3cretPass123 -P3306 app", "S3cretPass123", "-p***"},
		{"cn password", "我的密码是 Hunter2024!", "Hunter2024", "密码是 ***"},
		{"cn key", "密钥：AbCd1234xyz，别外传", "AbCd1234xyz", "密钥：***"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := RedactContent(tc.in)
			if strings.Contains(out, tc.leak) {
				t.Fatalf("secret leaked: %q -> %q", tc.in, out)
			}
			if !strings.Contains(out, tc.expect) {
				t.Fatalf("expected %q in %q", tc.expect, out)
			}
		})
	}
}

func TestRedactContent_KeepsOrdinaryText(t *testing.T) {
	in := "请帮我写一个函数，max_tokens: 4096，max_tokens: 100, the token limit is fine, 2026-10-16 12:00, line 1234567。" +
		"密码学很有意思。see https://example.com/docs?a=1 and ssh://github.com/org/repo.git; mysql -P3306 -h db app"
	out := RedactContent(in)
	if out != in {
		t.Fatalf("ordinary text changed: %q -> %q", in, out)
	}
	if RedactContent("") != "" {
		t.Fatal("empty must stay empty")
	}
	// JSON tool arguments keep their layout (no re-encoding).
	args := `{"command":["bash","-lc","ls <dir>"]}`
	if got := RedactContent(args); got != args {
		t.Fatalf("json args changed: %q", got)
	}
}
