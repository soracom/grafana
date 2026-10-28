package httpstatic

import (
	"net/http"
	"net/http/httptest"
	"testing"

	_ "github.com/grafana/grafana/pkg/api/response" // Provides the web handler wrapper.
	"github.com/grafana/grafana/pkg/web"
)

func TestDirectoryRedirectCleansPath(t *testing.T) {
	tests := []struct {
		name     string
		request  string
		location string
	}{
		{
			name:     "directory without trailing slash",
			request:  "/subdir",
			location: "/subdir/",
		},
		{
			name:     "encoded path that could become an external URL",
			request:  "/subdir/..%2F%5C127.0.0.1:80%2F%3F%2F..%2F..",
			location: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := web.New()
			mux.Use(Static(t.TempDir(), StaticOptions{Prefix: "subdir"}))
			request := httptest.NewRequest(http.MethodGet, tt.request, nil)
			response := httptest.NewRecorder()

			mux.ServeHTTP(response, request)

			if response.Code != http.StatusFound {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusFound)
			}
			if location := response.Header().Get("Location"); location != tt.location {
				t.Errorf("Location = %q, want %q", location, tt.location)
			}
		})
	}
}
