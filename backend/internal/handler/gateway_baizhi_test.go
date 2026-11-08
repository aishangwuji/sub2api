package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockSettingRepoForGateway struct {
	data map[string]string
}

func (m *mockSettingRepoForGateway) Get(ctx context.Context, key string) (*service.Setting, error) {
	val, ok := m.data[key]
	if !ok {
		return nil, service.ErrSettingNotFound
	}
	return &service.Setting{Key: key, Value: val}, nil
}
func (m *mockSettingRepoForGateway) GetValue(ctx context.Context, key string) (string, error) {
	val, ok := m.data[key]
	if !ok {
		return "", service.ErrSettingNotFound
	}
	return val, nil
}
func (m *mockSettingRepoForGateway) Set(ctx context.Context, key, value string) error {
	m.data[key] = value
	return nil
}
func (m *mockSettingRepoForGateway) GetMultiple(ctx context.Context, keys []string) (map[string]string, error) {
	return nil, nil
}
func (m *mockSettingRepoForGateway) SetMultiple(ctx context.Context, settings map[string]string) error {
	return nil
}
func (m *mockSettingRepoForGateway) GetAll(ctx context.Context) (map[string]string, error) {
	return nil, nil
}
func (m *mockSettingRepoForGateway) Delete(ctx context.Context, key string) error {
	return nil
}

func TestBaizhiGateway_ForwardScrape(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Mock upstream server
	receivedAuth := ""
	receivedPath := ""
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"status":"ok"}}`))
	}))
	defer upstream.Close()

	repo := &mockSettingRepoForGateway{data: make(map[string]string)}
	svc := service.NewBaizhiService(repo)

	// Configure mock upstream
	cfg := service.DefaultBaizhiConfig()
	cfg.AuthRequired = false // Test without token requirement
	cfg.WebScrape.BaseURL = upstream.URL
	cfg.WebScrape.Keys = []service.BaizhiKeyEntry{
		{
			ID:          "key-1",
			AccountName: "Scrape-Account",
			APIKey:      "baizhi-secret-key-123",
			Enabled:     true,
			Weight:      1,
		},
	}
	err := svc.UpdateConfig(context.Background(), cfg)
	require.NoError(t, err)

	gw := NewBaizhiGatewayHandler(svc, nil)

	r := gin.New()
	r.Any("/baizhi/scrape/*subpath", gw.ForwardScrape)

	req := httptest.NewRequest(http.MethodPost, "/baizhi/scrape/openapi/scrape", strings.NewReader(`{"url":"https://example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Bearer baizhi-secret-key-123", receivedAuth)
	assert.Equal(t, "/openapi/scrape", receivedPath)
}

func TestBaizhiGateway_ForwardSearch_ShortPath(t *testing.T) {
	gin.SetMode(gin.TestMode)

	receivedAuth := ""
	receivedPath := ""
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		receivedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"code":0,"data":{"status":"ok"}}`))
	}))
	defer upstream.Close()

	repo := &mockSettingRepoForGateway{data: make(map[string]string)}
	svc := service.NewBaizhiService(repo)

	cfg := service.DefaultBaizhiConfig()
	cfg.AuthRequired = false
	cfg.WebSearch.BaseURL = upstream.URL
	cfg.WebSearch.Keys = []service.BaizhiKeyEntry{
		{
			ID:          "key-search-1",
			AccountName: "Search-Account",
			APIKey:      "search-secret-key-456",
			Enabled:     true,
			Weight:      1,
		},
	}
	err := svc.UpdateConfig(context.Background(), cfg)
	require.NoError(t, err)

	gw := NewBaizhiGatewayHandler(svc, nil)

	r := gin.New()
	r.Any("/baizhi/search/*subpath", gw.ForwardSearch)

	// Test short path alias: /search -> /openapi/search
	req := httptest.NewRequest(http.MethodPost, "/baizhi/search/search", strings.NewReader(`{"query":"test"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "Bearer search-secret-key-456", receivedAuth)
	assert.Equal(t, "/openapi/search", receivedPath)
}
