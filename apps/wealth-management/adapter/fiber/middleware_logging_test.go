package fiber

import "testing"

func TestGivenAnOAuthCallbackQueryWhenLoggingThenTheCodeAndStateAreRedacted(t *testing.T) {
	cases := map[string]string{
		"state=abc&code=4%2Fsecret&scope=x": "code=REDACTED&scope=x&state=REDACTED",
		"error=access_denied&state=abc":     "error=access_denied&state=REDACTED",
		"month=2026-10&limit=5":             "month=2026-10&limit=5",
		"code=a;b":                          "[unparsable query]",
	}
	for raw, want := range cases {
		if got := redactQuery(raw); got != want {
			t.Errorf("redactQuery(%q) = %q, want %q", raw, got, want)
		}
	}
}
