package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	SettingKeyBaizhiCloudConfig = "baizhi_cloud_config"

	// Default Upstream Base URLs
	DefaultBaizhiScrapeBaseURL = "https://web-scrape.app.baizhi.cloud"
	DefaultBaizhiSearchBaseURL = "https://websearch.app.baizhi.cloud"
	DefaultBaizhiVoiceBaseURL  = "https://asrmodel.app.baizhi.cloud/openapi/v1"
	DefaultBaizhiVoiceWssURL   = "wss://asrmodel.app.baizhi.cloud/openapi/v1"
	DefaultBaizhiImageBaseURL  = "https://image-generate.app.baizhi.cloud/openapi/v1"

	// Supported Service Identifiers
	BaizhiServiceScrape = "web_scrape"
	BaizhiServiceSearch = "web_search"
	BaizhiServiceVoice  = "voice"
	BaizhiServiceImage  = "image_generate"

	StrategyRoundRobin = "round_robin"
	StrategyFailover   = "failover"
)

var (
	ErrBaizhiServiceDisabled = errors.New("baizhi service is currently disabled")
	ErrBaizhiNoActiveKeys    = errors.New("no active upstream API key available for this service")
	ErrBaizhiUnknownService  = errors.New("unknown baizhi service type")
)

// BaizhiKeyEntry represents an API key configuration for a specific account.
type BaizhiKeyEntry struct {
	ID          string    `json:"id"`
	AccountName string    `json:"account_name"`
	APIKey      string    `json:"api_key"`
	Enabled     bool      `json:"enabled"`
	Weight      int       `json:"weight"`
	CreatedAt   time.Time `json:"created_at"`
}

// BaizhiServiceConfig represents configuration for one of the four business modules.
type BaizhiServiceConfig struct {
	Enabled  bool             `json:"enabled"`
	Strategy string           `json:"strategy"`
	BaseURL  string           `json:"base_url"`
	WssURL   string           `json:"wss_url,omitempty"`
	Keys     []BaizhiKeyEntry `json:"keys"`
}

// BaizhiConfig holds the complete configuration across all 4 services.
type BaizhiConfig struct {
	AuthRequired  bool                `json:"auth_required"`
	WebScrape     BaizhiServiceConfig `json:"web_scrape"`
	WebSearch     BaizhiServiceConfig `json:"web_search"`
	Voice         BaizhiServiceConfig `json:"voice"`
	ImageGenerate BaizhiServiceConfig `json:"image_generate"`
}

// DefaultBaizhiConfig returns a clean, initialized default configuration.
func DefaultBaizhiConfig() BaizhiConfig {
	return BaizhiConfig{
		AuthRequired: true,
		WebScrape: BaizhiServiceConfig{
			Enabled:  true,
			Strategy: StrategyRoundRobin,
			BaseURL:  DefaultBaizhiScrapeBaseURL,
			Keys:     make([]BaizhiKeyEntry, 0),
		},
		WebSearch: BaizhiServiceConfig{
			Enabled:  true,
			Strategy: StrategyRoundRobin,
			BaseURL:  DefaultBaizhiSearchBaseURL,
			Keys:     make([]BaizhiKeyEntry, 0),
		},
		Voice: BaizhiServiceConfig{
			Enabled:  true,
			Strategy: StrategyRoundRobin,
			BaseURL:  DefaultBaizhiVoiceBaseURL,
			WssURL:   DefaultBaizhiVoiceWssURL,
			Keys:     make([]BaizhiKeyEntry, 0),
		},
		ImageGenerate: BaizhiServiceConfig{
			Enabled:  true,
			Strategy: StrategyRoundRobin,
			BaseURL:  DefaultBaizhiImageBaseURL,
			Keys:     make([]BaizhiKeyEntry, 0),
		},
	}
}

// BaizhiService manages Baizhi Cloud upstream gateway and account key pools.
type BaizhiService struct {
	settingRepo SettingRepository
	httpClient  *http.Client

	mu        sync.RWMutex
	cachedCfg *BaizhiConfig

	// Round-robin counters for each service
	scrapeCounter uint64
	searchCounter uint64
	voiceCounter  uint64
	imageCounter  uint64
}

// NewBaizhiService initializes a new BaizhiService.
func NewBaizhiService(settingRepo SettingRepository) *BaizhiService {
	return &BaizhiService{
		settingRepo: settingRepo,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetConfig reads configuration from storage (or returns in-memory cache).
func (s *BaizhiService) GetConfig(ctx context.Context) (BaizhiConfig, error) {
	s.mu.RLock()
	if s.cachedCfg != nil {
		cfg := *s.cachedCfg
		s.mu.RUnlock()
		return cfg, nil
	}
	s.mu.RUnlock()

	return s.reloadConfig(ctx)
}

func (s *BaizhiService) reloadConfig(ctx context.Context) (BaizhiConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	def := DefaultBaizhiConfig()
	if s.settingRepo == nil {
		s.cachedCfg = &def
		return def, nil
	}

	val, err := s.settingRepo.GetValue(ctx, SettingKeyBaizhiCloudConfig)
	if err != nil || strings.TrimSpace(val) == "" {
		s.cachedCfg = &def
		return def, nil
	}

	var parsed BaizhiConfig
	if err := json.Unmarshal([]byte(val), &parsed); err != nil {
		s.cachedCfg = &def
		return def, nil
	}

	// Ensure default URLs if empty
	if strings.TrimSpace(parsed.WebScrape.BaseURL) == "" {
		parsed.WebScrape.BaseURL = DefaultBaizhiScrapeBaseURL
	}
	if strings.TrimSpace(parsed.WebSearch.BaseURL) == "" {
		parsed.WebSearch.BaseURL = DefaultBaizhiSearchBaseURL
	}
	if strings.TrimSpace(parsed.Voice.BaseURL) == "" {
		parsed.Voice.BaseURL = DefaultBaizhiVoiceBaseURL
	}
	if strings.TrimSpace(parsed.Voice.WssURL) == "" {
		parsed.Voice.WssURL = DefaultBaizhiVoiceWssURL
	}
	if strings.TrimSpace(parsed.ImageGenerate.BaseURL) == "" {
		parsed.ImageGenerate.BaseURL = DefaultBaizhiImageBaseURL
	}

	s.cachedCfg = &parsed
	return parsed, nil
}

// UpdateConfig updates and persists the BaizhiConfig to PostgreSQL settings table.
func (s *BaizhiService) UpdateConfig(ctx context.Context, updated BaizhiConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(updated)
	if err != nil {
		return fmt.Errorf("marshal baizhi config: %w", err)
	}

	if s.settingRepo != nil {
		if err := s.settingRepo.Set(ctx, SettingKeyBaizhiCloudConfig, string(data)); err != nil {
			return fmt.Errorf("persist baizhi config: %w", err)
		}
	}

	s.cachedCfg = &updated
	return nil
}

// SelectKey picks an active API Key and target BaseURL for the specified service type using Round-Robin.
func (s *BaizhiService) SelectKey(ctx context.Context, serviceType string) (*BaizhiKeyEntry, string, string, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, "", "", err
	}

	var svc BaizhiServiceConfig
	var counter *uint64

	switch serviceType {
	case BaizhiServiceScrape:
		svc = cfg.WebScrape
		counter = &s.scrapeCounter
	case BaizhiServiceSearch:
		svc = cfg.WebSearch
		counter = &s.searchCounter
	case BaizhiServiceVoice:
		svc = cfg.Voice
		counter = &s.voiceCounter
	case BaizhiServiceImage:
		svc = cfg.ImageGenerate
		counter = &s.imageCounter
	default:
		return nil, "", "", ErrBaizhiUnknownService
	}

	if !svc.Enabled {
		return nil, "", "", ErrBaizhiServiceDisabled
	}

	// Filter active keys
	activeKeys := make([]BaizhiKeyEntry, 0, len(svc.Keys))
	for _, k := range svc.Keys {
		if k.Enabled && strings.TrimSpace(k.APIKey) != "" {
			activeKeys = append(activeKeys, k)
		}
	}

	if len(activeKeys) == 0 {
		return nil, "", "", ErrBaizhiNoActiveKeys
	}

	idx := atomic.AddUint64(counter, 1) % uint64(len(activeKeys))
	selected := activeKeys[idx]
	return &selected, svc.BaseURL, svc.WssURL, nil
}

// BaizhiTestResult returns diagnostic results when verifying an upstream key.
type BaizhiTestResult struct {
	Success      bool   `json:"success"`
	StatusCode   int    `json:"status_code"`
	LatencyMs    int64  `json:"latency_ms"`
	Message      string `json:"message"`
	ResponseBody string `json:"response_body,omitempty"`
}

// TestKey directly tests an API key against the corresponding upstream pricing/auth endpoint.
func (s *BaizhiService) TestKey(ctx context.Context, serviceType, apiKey, customBaseURL string) (*BaizhiTestResult, error) {
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return &BaizhiTestResult{
			Success: false,
			Message: "API Key cannot be empty",
		}, nil
	}

	var targetURL string
	switch serviceType {
	case BaizhiServiceScrape:
		baseURL := DefaultBaizhiScrapeBaseURL
		if strings.TrimSpace(customBaseURL) != "" {
			baseURL = strings.TrimRight(customBaseURL, "/")
		}
		targetURL = baseURL + "/openapi/v1/pricing"
	case BaizhiServiceSearch:
		baseURL := DefaultBaizhiSearchBaseURL
		if strings.TrimSpace(customBaseURL) != "" {
			baseURL = strings.TrimRight(customBaseURL, "/")
		}
		targetURL = baseURL + "/openapi/v1/pricing"
	case BaizhiServiceVoice:
		baseURL := DefaultBaizhiVoiceBaseURL
		if strings.TrimSpace(customBaseURL) != "" {
			baseURL = strings.TrimRight(customBaseURL, "/")
		}
		// Notice voice pricing is /pricing under /openapi/v1
		if strings.HasSuffix(baseURL, "/openapi/v1") {
			targetURL = baseURL + "/pricing"
		} else {
			targetURL = baseURL + "/openapi/v1/pricing"
		}
	case BaizhiServiceImage:
		baseURL := DefaultBaizhiImageBaseURL
		if strings.TrimSpace(customBaseURL) != "" {
			baseURL = strings.TrimRight(customBaseURL, "/")
		}
		if strings.HasSuffix(baseURL, "/openapi/v1") {
			targetURL = baseURL + "/pricing"
		} else {
			targetURL = baseURL + "/openapi/v1/pricing"
		}
	default:
		return nil, ErrBaizhiUnknownService
	}

	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, targetURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create test request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := s.httpClient.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &BaizhiTestResult{
			Success:   false,
			LatencyMs: latency,
			Message:   fmt.Sprintf("Connection failed: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024*32))
	bodyStr := string(bodyBytes)

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	msg := "OK"
	if !success {
		msg = fmt.Sprintf("Upstream returned HTTP %d", resp.StatusCode)
	}

	return &BaizhiTestResult{
		Success:      success,
		StatusCode:   resp.StatusCode,
		LatencyMs:    latency,
		Message:      msg,
		ResponseBody: bodyStr,
	}, nil
}
