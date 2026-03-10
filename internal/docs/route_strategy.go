package docs

import "strings"

const swaggerIndexPath = "/swagger/index.html"

func SwaggerRedirectTarget() string {
	return swaggerIndexPath
}

func IsSPAFallbackExcludedPath(path string) bool {
	return strings.HasPrefix(path, "/api/") ||
		strings.HasPrefix(path, "/v1/") ||
		strings.HasPrefix(path, "/swagger") ||
		strings.HasPrefix(path, "/debug/") ||
		path == "/debug" ||
		path == "/metrics" ||
		strings.HasPrefix(path, "/metrics/")
}
