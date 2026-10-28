package loki

import (
	"context"
	"net/http"
	"testing"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/grafana/pkg/infra/log"
)

func TestCallResourceRejectsPathTraversal(t *testing.T) {
	for _, resourceURL := range []string{
		"label/../status/buildinfo",
		"label/app/../status/buildinfo",
		"label/../../../admin",
		"label/%2e%2e/%2e%2e/%2e%2e/admin",
		"label/%2E%2e%2f..%2f..%2fadmin",
		"label/%252e%252e/%252e%252e/%252e%252e/admin",
		"label/%252f..%252f..%252fadmin",
		"label/%5c..%5c..%5cadmin",
		"label/%zz",
	} {
		t.Run(resourceURL, func(t *testing.T) {
			dsInfo := makeMockedDsInfoForOauth(nil, func(*http.Request) {
				t.Fatal("invalid resource path reached the Loki backend")
			})
			dsInfo.URL = "http://loki.local"
			sender := &mockedCallResourceResponseSenderForOauth{}
			err := callResource(context.Background(), &backend.CallResourceRequest{Method: http.MethodGet, URL: resourceURL}, sender, &dsInfo, log.New("testlog"))
			require.NoError(t, err)
			require.NotNil(t, sender.Response)
			assert.Equal(t, http.StatusBadRequest, sender.Response.Status)
		})
	}
}

func TestCallResourceAllowsLabelValues(t *testing.T) {
	called := false
	dsInfo := makeMockedDsInfoForOauth([]byte(`{}`), func(req *http.Request) {
		called = true
		assert.Equal(t, "/loki/api/v1/label/app/values", req.URL.Path)
	})
	dsInfo.URL = "http://loki.local"
	sender := &mockedCallResourceResponseSenderForOauth{}
	err := callResource(context.Background(), &backend.CallResourceRequest{Method: http.MethodGet, URL: "label/app/values"}, sender, &dsInfo, log.New("testlog"))
	require.NoError(t, err)
	require.True(t, called)
	require.NotNil(t, sender.Response)
	assert.Equal(t, http.StatusOK, sender.Response.Status)
}

func TestCallResourceAllowsEncodedQueryParameters(t *testing.T) {
	called := false
	dsInfo := makeMockedDsInfoForOauth([]byte(`{}`), func(req *http.Request) {
		called = true
		assert.Equal(t, "/loki/api/v1/labels", req.URL.Path)
		assert.Equal(t, "match%5B%5D=%7Bapp%3D%22lagoon%22%7D", req.URL.RawQuery)
	})
	dsInfo.URL = "http://loki.local"
	sender := &mockedCallResourceResponseSenderForOauth{}
	err := callResource(context.Background(), &backend.CallResourceRequest{Method: http.MethodGet, URL: "labels?match%5B%5D=%7Bapp%3D%22lagoon%22%7D"}, sender, &dsInfo, log.New("testlog"))
	require.NoError(t, err)
	require.True(t, called)
	require.NotNil(t, sender.Response)
	assert.Equal(t, http.StatusOK, sender.Response.Status)
}

func TestGetHeadersForCallResource(t *testing.T) {
	const idTokn1 = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	const idTokn2 = "eyJhbGciOiJIUzI1NiJ9.eyJuYW1lIjoiSm9obiBEb2UiLCJleHAiOjE2Njg2MjExODQsImlhdCI6MTY2ODYyMTE4NH0.bg0Y0S245DeANhNnnLBCfGYBseTld29O0xynhQwZZlU"
	const authTokn1 = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	const authTokn2 = "Bearer eyJhbGciOiJIUzI1NiJ9.eyJuYW1lIjoiSm9obiBEb2UiLCJleHAiOjE2Njg2MjExODQsImlhdCI6MTY2ODYyMTE4NH0.bg0Y0S245DeANhNnnLBCfGYBseTld29O0xynhQwZZlU"

	testCases := map[string]struct {
		headers         map[string][]string
		expectedHeaders map[string]string
	}{
		"Headers with empty value": {
			headers: map[string][]string{
				"X-Grafana-Org-Id": {"1"},
				"Cookie":           {""},
				"X-Id-Token":       {""},
				"Accept-Encoding":  {""},
				"Authorization":    {""},
			},
			expectedHeaders: map[string]string{},
		},
		"Headers with multiple values": {
			headers: map[string][]string{
				"Authorization":    {authTokn1, authTokn2},
				"Cookie":           {"a=1"},
				"X-Grafana-Org-Id": {"1"},
				"Accept-Encoding":  {"gzip", "compress"},
				"X-Id-Token":       {idTokn1, idTokn2},
			},
			expectedHeaders: map[string]string{
				"Authorization":   authTokn1,
				"Cookie":          "a=1",
				"Accept-Encoding": "gzip",
				"X-ID-Token":      idTokn1,
			},
		},
		"Headers with single value": {
			headers: map[string][]string{
				"Authorization":    {authTokn1},
				"X-Grafana-Org-Id": {"1"},
				"Cookie":           {"a=1"},
				"Accept-Encoding":  {"gzip"},
				"X-Id-Token":       {idTokn1},
			},
			expectedHeaders: map[string]string{
				"Authorization":   authTokn1,
				"Cookie":          "a=1",
				"Accept-Encoding": "gzip",
				"X-ID-Token":      idTokn1,
			},
		},
		"Non Canonical 'X-Id-Token' header key": {
			headers: map[string][]string{
				"X-ID-TOKEN": {idTokn1},
			},
			expectedHeaders: map[string]string{
				"X-ID-Token": idTokn1,
			},
		},
	}
	for name, test := range testCases {
		t.Run(name, func(t *testing.T) {
			headers := getHeadersForCallResource(test.headers)
			assert.Equal(t, test.expectedHeaders, headers)
		})
	}
}
