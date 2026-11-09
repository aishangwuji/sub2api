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

	StrategyRoundRobin      = "round_robin"
	StrategyFailover        = "failover"
	StrategyTempCreditFirst = "temp_credit_first"
)

var (
	ErrBaizhiServiceDisabled = errors.New("baizhi service is currently disabled")
	ErrBaizhiNoActiveKeys    = errors.New("no active upstream API key available for this service")
	ErrBaizhiUnknownService  = errors.New("unknown baizhi service type")
)

// BaizhiKeyEntry represents an API key configuration for a specific account.
type BaizhiKeyEntry struct {
	ID               string     `json:"id"`
	AccountName      string     `json:"account_name"`
	APIKey           string     `json:"api_key"`
	Cookie           string     `json:"cookie,omitempty"`
	Enabled          bool       `json:"enabled"`
	Weight           int        `json:"weight"`
	TemporaryCredits float64    `json:"temporary_credits"`
	PermanentCredits float64    `json:"permanent_credits"`
	AvailableCredits float64    `json:"available_credits"`
	DailyClaimed     bool       `json:"daily_claimed"`
	UserID           int64      `json:"user_id,omitempty"`
	UserName         string     `json:"user_name,omitempty"`
	Avatar           string     `json:"avatar,omitempty"`
	LastSyncAt       *time.Time `json:"last_sync_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// BaizhiDiscoveredKey represents an API Key auto-discovered from Baizhi Cloud console.
type BaizhiDiscoveredKey struct {
	Name      string  `json:"name"`
	Key       string  `json:"key"`
	TotalUsed float64 `json:"total_used"`
	TodayUsed float64 `json:"today_used"`
}

// BaizhiAccountInspectResult represents diagnostic information fetched from baizhi.cloud for an account.
type BaizhiAccountInspectResult struct {
	UserID           int64                 `json:"user_id"`
	UserName         string                `json:"user_name"`
	Avatar           string                `json:"avatar"`
	TemporaryCredits float64               `json:"temporary_credits"`
	PermanentCredits float64               `json:"permanent_credits"`
	AvailableCredits float64               `json:"available_credits"`
	DailyClaimed     bool                  `json:"daily_claimed"`
	DiscoveredKeys   []BaizhiDiscoveredKey `json:"discovered_keys"`
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

	var selected BaizhiKeyEntry
	switch svc.Strategy {
	case StrategyFailover:
		// Always pick first healthy key
		selected = activeKeys[0]
	case StrategyTempCreditFirst:
		// Prefer accounts with remaining temporary credits (daily free 100 quota)
		tempCreditKeys := make([]BaizhiKeyEntry, 0, len(activeKeys))
		for _, k := range activeKeys {
			if k.TemporaryCredits > 0 {
				tempCreditKeys = append(tempCreditKeys, k)
			}
		}
		if len(tempCreditKeys) > 0 {
			idx := atomic.AddUint64(counter, 1) % uint64(len(tempCreditKeys))
			selected = tempCreditKeys[idx]
		} else {
			// Fallback to round-robin among all active keys once all temporary credits are exhausted
			idx := atomic.AddUint64(counter, 1) % uint64(len(activeKeys))
			selected = activeKeys[idx]
		}
	default: // StrategyRoundRobin
		idx := atomic.AddUint64(counter, 1) % uint64(len(activeKeys))
		selected = activeKeys[idx]
	}

	return &selected, svc.BaseURL, svc.WssURL, nil
}

// DeductTempCredit decreases in-memory temporary credits for a key following a paid usage event.
func (s *BaizhiService) DeductTempCredit(keyID string, credits float64) {
	if credits <= 0 || keyID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cachedCfg == nil {
		return
	}

	deductFromSlice := func(keys []BaizhiKeyEntry) {
		for i := range keys {
			if keys[i].ID == keyID {
				keys[i].TemporaryCredits -= credits
				if keys[i].TemporaryCredits < 0 {
					keys[i].TemporaryCredits = 0
				}
				if keys[i].AvailableCredits >= credits {
					keys[i].AvailableCredits -= credits
				}
			}
		}
	}

	deductFromSlice(s.cachedCfg.WebScrape.Keys)
	deductFromSlice(s.cachedCfg.WebSearch.Keys)
	deductFromSlice(s.cachedCfg.Voice.Keys)
	deductFromSlice(s.cachedCfg.ImageGenerate.Keys)
}

// NormalizeBaizhiSession extracts clean baizhi_session token from raw cookie or session string.
func NormalizeBaizhiSession(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	// Case 1: Cookie string containing "baizhi_session=..."
	if idx := strings.Index(raw, "baizhi_session="); idx != -1 {
		sub := raw[idx+len("baizhi_session="):]
		if semi := strings.Index(sub, ";"); semi != -1 {
			return strings.TrimSpace(sub[:semi])
		}
		return strings.TrimSpace(sub)
	}
	// Case 2: Raw session UUID string
	return raw
}

// InspectAccount queries baizhi.cloud for user profile, wallet balance, and any existing API keys.
func (s *BaizhiService) InspectAccount(ctx context.Context, rawCookie string) (*BaizhiAccountInspectResult, error) {
	session := NormalizeBaizhiSession(rawCookie)
	if session == "" {
		return nil, errors.New("cookie or baizhi_session is empty")
	}

	cookieHeader := "baizhi_session=" + session

	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// 1. Fetch Profile: GET https://baizhi.cloud/api/v1/user/profile
	profileReq, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "https://baizhi.cloud/api/v1/user/profile", nil)
	if err != nil {
		return nil, fmt.Errorf("create profile request: %w", err)
	}
	profileReq.Header.Set("Cookie", cookieHeader)
	profileReq.Header.Set("Accept", "application/json")

	profileResp, err := s.httpClient.Do(profileReq)
	if err != nil {
		return nil, fmt.Errorf("fetch profile failed: %w", err)
	}
	defer profileResp.Body.Close()

	if profileResp.StatusCode == http.StatusUnauthorized || profileResp.StatusCode == http.StatusForbidden {
		return nil, errors.New("baizhi session is invalid or expired (Unauthorized)")
	}
	if profileResp.StatusCode < 200 || profileResp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch profile upstream returned HTTP %d", profileResp.StatusCode)
	}

	var profileEnvelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			ID     int64  `json:"id"`
			Name   string `json:"name"`
			Avatar string `json:"avatar"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.NewDecoder(profileResp.Body).Decode(&profileEnvelope); err != nil {
		return nil, fmt.Errorf("decode profile response: %w", err)
	}
	if profileEnvelope.Code != 0 {
		return nil, fmt.Errorf("baizhi profile error: %s", profileEnvelope.Message)
	}

	// 2. Fetch Balance: GET https://baizhi.cloud/api/v1/user/wallet/balance
	balanceReq, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "https://baizhi.cloud/api/v1/user/wallet/balance", nil)
	if err != nil {
		return nil, fmt.Errorf("create balance request: %w", err)
	}
	balanceReq.Header.Set("Cookie", cookieHeader)
	balanceReq.Header.Set("Accept", "application/json")

	balanceResp, err := s.httpClient.Do(balanceReq)
	if err != nil {
		return nil, fmt.Errorf("fetch balance failed: %w", err)
	}
	defer balanceResp.Body.Close()

	var balanceEnvelope struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    struct {
			UserID                      int64 `json:"user_id"`
			TemporaryCreditCents        int64 `json:"temporary_credit_cents"`
			PermanentCreditCents        int64 `json:"permanent_credit_cents"`
			FrozenCreditCents           int64 `json:"frozen_credit_cents"`
			AvailableCreditCents        int64 `json:"available_credit_cents"`
			DailyTemporaryCreditClaimed bool  `json:"daily_temporary_credit_claimed"`
		} `json:"data"`
	}
	if err := json.NewDecoder(balanceResp.Body).Decode(&balanceEnvelope); err != nil {
		return nil, fmt.Errorf("decode balance response: %w", err)
	}
	if balanceEnvelope.Code != 0 {
		return nil, fmt.Errorf("baizhi balance error: %s", balanceEnvelope.Message)
	}

	// 3. Discover WebSearch API Keys (if available): GET https://websearch.app.baizhi.cloud/api/web_search/api_keys
	discovered := make([]BaizhiDiscoveredKey, 0)
	keysReq, err := http.NewRequestWithContext(reqCtx, http.MethodGet, "https://websearch.app.baizhi.cloud/api/web_search/api_keys?page=1&page_size=20", nil)
	if err == nil {
		keysReq.Header.Set("Cookie", cookieHeader)
		keysReq.Header.Set("Accept", "application/json")
		if keysResp, err := s.httpClient.Do(keysReq); err == nil {
			defer keysResp.Body.Close()
			if keysResp.StatusCode >= 200 && keysResp.StatusCode < 300 {
				var keysEnvelope struct {
					Code int `json:"code"`
					Data struct {
						Items []struct {
							Name             string  `json:"name"`
							PlainTextKey     string  `json:"plain_text_key"`
							TotalUsedCredits float64 `json:"total_used_credits"`
							TodayUsedCredits float64 `json:"today_used_credits"`
						} `json:"items"`
					} `json:"data"`
				}
				if err := json.NewDecoder(keysResp.Body).Decode(&keysEnvelope); err == nil && keysEnvelope.Code == 0 {
					for _, it := range keysEnvelope.Data.Items {
						if strings.TrimSpace(it.PlainTextKey) != "" {
							discovered = append(discovered, BaizhiDiscoveredKey{
								Name:      it.Name,
								Key:       it.PlainTextKey,
								TotalUsed: it.TotalUsedCredits,
								TodayUsed: it.TodayUsedCredits,
							})
						}
					}
				}
			}
		}
	}

	return &BaizhiAccountInspectResult{
		UserID:           profileEnvelope.Data.ID,
		UserName:         profileEnvelope.Data.Name,
		Avatar:           profileEnvelope.Data.Avatar,
		TemporaryCredits: float64(balanceEnvelope.Data.TemporaryCreditCents) / 100.0,
		PermanentCredits: float64(balanceEnvelope.Data.PermanentCreditCents) / 100.0,
		AvailableCredits: float64(balanceEnvelope.Data.AvailableCreditCents) / 100.0,
		DailyClaimed:     balanceEnvelope.Data.DailyTemporaryCreditClaimed,
		DiscoveredKeys:   discovered,
	}, nil
}

// SyncAllBalances loops through all configured keys that have a Cookie, fetches live balances, and saves the updated config.
func (s *BaizhiService) SyncAllBalances(ctx context.Context) (*BaizhiConfig, error) {
	cfg, err := s.GetConfig(ctx)
	if err != nil {
		return nil, err
	}

	sessionCache := make(map[string]*BaizhiAccountInspectResult)
	now := time.Now()

	syncKeys := func(keys []BaizhiKeyEntry) []BaizhiKeyEntry {
		out := make([]BaizhiKeyEntry, len(keys))
		copy(out, keys)
		for i := range out {
			session := NormalizeBaizhiSession(out[i].Cookie)
			if session == "" {
				continue
			}

			info, ok := sessionCache[session]
			if !ok {
				res, err := s.InspectAccount(ctx, session)
				if err == nil && res != nil {
					info = res
					sessionCache[session] = res
				}
			}

			if info != nil {
				out[i].TemporaryCredits = info.TemporaryCredits
				out[i].PermanentCredits = info.PermanentCredits
				out[i].AvailableCredits = info.AvailableCredits
				out[i].DailyClaimed = info.DailyClaimed
				out[i].UserID = info.UserID
				if out[i].AccountName == "" || strings.HasPrefix(out[i].AccountName, "账号 ") {
					out[i].AccountName = info.UserName
				}
				out[i].UserName = info.UserName
				out[i].Avatar = info.Avatar
				out[i].LastSyncAt = &now
			}
		}
		return out
	}

	cfg.WebScrape.Keys = syncKeys(cfg.WebScrape.Keys)
	cfg.WebSearch.Keys = syncKeys(cfg.WebSearch.Keys)
	cfg.Voice.Keys = syncKeys(cfg.Voice.Keys)
	cfg.ImageGenerate.Keys = syncKeys(cfg.ImageGenerate.Keys)

	if err := s.UpdateConfig(ctx, cfg); err != nil {
		return nil, fmt.Errorf("persist synced baizhi config: %w", err)
	}

	return &cfg, nil
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
