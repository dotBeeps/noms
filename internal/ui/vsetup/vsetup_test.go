package vsetup_test

import (
	"testing"

	"github.com/dotBeeps/noms/internal/ui/vsetup"
)

func TestNormalizeCookie(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "empty input",
			raw:  "",
			want: "",
		},
		{
			name: "whitespace only",
			raw:  "   ",
			want: "",
		},
		{
			name: "bare value",
			raw:  "abc123token",
			want: "__Host-voresky_session=abc123token",
		},
		{
			name: "bare value with whitespace",
			raw:  "  abc123token  ",
			want: "__Host-voresky_session=abc123token",
		},
		{
			name: "full cookie name and value",
			raw:  "__Host-voresky_session=abc123token",
			want: "__Host-voresky_session=abc123token",
		},
		{
			name: "bare session name without host prefix",
			raw:  "voresky_session=abc123token",
			want: "voresky_session=abc123token",
		},
		{
			name: "cookie name with empty value",
			raw:  "__Host-voresky_session=",
			want: "",
		},
		{
			name: "cookie name with whitespace value",
			raw:  "__Host-voresky_session=   ",
			want: "",
		},
		{
			name: "Cookie: header prefix stripped",
			raw:  "Cookie: __Host-voresky_session=abc123token",
			want: "__Host-voresky_session=abc123token",
		},
		{
			name: "cookie: lowercase header prefix stripped",
			raw:  "cookie: __Host-voresky_session=abc123token",
			want: "__Host-voresky_session=abc123token",
		},
		{
			name: "Cookie: header prefix with bare value",
			raw:  "Cookie: abc123token",
			want: "__Host-voresky_session=abc123token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := vsetup.NormalizeCookie(tt.raw)
			if got != tt.want {
				t.Errorf("NormalizeCookie(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
