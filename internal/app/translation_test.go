package app

import "testing"

func TestSmartTarget(t *testing.T) {
	for _, tc := range []struct{ source, target, text, want string }{
		{"auto", "smart", "port is in use", "zh"},
		{"auto", "smart", "端口 8080 被占用", "en"},
		{"auto", "smart", "日本語を翻訳する", "zh"},
		{"auto", "de", "端口", "de"},
		{"zh", "smart", "API", "en"},
	} {
		if got := targetLanguage(tc.source, tc.target, tc.text); got != tc.want {
			t.Errorf("%s → %s for %q: got %s, want %s", tc.source, tc.target, tc.text, got, tc.want)
		}
	}
}
