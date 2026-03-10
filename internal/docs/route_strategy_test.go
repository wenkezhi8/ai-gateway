package docs

import "testing"

func TestRouteStrategy_IsSPAFallbackExcludedPath_ExpectedReleaseGuards(t *testing.T) {
	cases := []struct {
		name string
		path string
		want bool
	}{
		{name: "api path", path: "/api/models", want: true},
		{name: "v1 path", path: "/v1/chat/completions", want: true},
		{name: "swagger root", path: "/swagger", want: true},
		{name: "swagger slash", path: "/swagger/", want: true},
		{name: "swagger asset", path: "/swagger/index.html", want: true},
		{name: "debug root", path: "/debug", want: true},
		{name: "debug pprof", path: "/debug/pprof/", want: true},
		{name: "metrics root", path: "/metrics", want: true},
		{name: "metrics child", path: "/metrics/internal", want: true},
		{name: "docs center", path: "/docs", want: false},
		{name: "docs center slash", path: "/docs/", want: false},
		{name: "frontend route", path: "/trace", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsSPAFallbackExcludedPath(tc.path); got != tc.want {
				t.Fatalf("IsSPAFallbackExcludedPath(%q)=%v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestRouteStrategy_SwaggerRedirectTarget_IsStable(t *testing.T) {
	if got := SwaggerRedirectTarget(); got != "/swagger/index.html" {
		t.Fatalf("SwaggerRedirectTarget()=%q, want %q", got, "/swagger/index.html")
	}
}
