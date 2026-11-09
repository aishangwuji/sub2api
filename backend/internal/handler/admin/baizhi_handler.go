package admin

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type BaizhiHandler struct {
	baizhiService *service.BaizhiService
}

func NewBaizhiHandler(baizhiService *service.BaizhiService) *BaizhiHandler {
	return &BaizhiHandler{
		baizhiService: baizhiService,
	}
}

// maskBaizhiAPIKey masks all but the first 4 and last 4 characters.
func maskBaizhiAPIKey(key string) string {
	k := strings.TrimSpace(key)
	if len(k) <= 8 {
		return "****"
	}
	prefix := k[:4]
	suffix := k[len(k)-4:]
	return prefix + "****" + suffix
}

func isBaizhiMasked(key string) bool {
	return strings.Contains(key, "****")
}

// GetConfig returns the full Baizhi configuration with masked API keys.
// GET /api/v1/admin/baizhi/config
func (h *BaizhiHandler) GetConfig(c *gin.Context) {
	cfg, err := h.baizhiService.GetConfig(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to load baizhi configuration: "+err.Error())
		return
	}

	// Create a safe copy with masked keys
	masked := cfg

	maskKeys := func(keys []service.BaizhiKeyEntry) []service.BaizhiKeyEntry {
		out := make([]service.BaizhiKeyEntry, len(keys))
		for i, k := range keys {
			out[i] = k
			out[i].APIKey = maskBaizhiAPIKey(k.APIKey)
			if strings.TrimSpace(k.Cookie) != "" {
				out[i].Cookie = maskBaizhiAPIKey(k.Cookie)
			}
		}
		return out
	}

	masked.WebScrape.Keys = maskKeys(cfg.WebScrape.Keys)
	masked.WebSearch.Keys = maskKeys(cfg.WebSearch.Keys)
	masked.Voice.Keys = maskKeys(cfg.Voice.Keys)
	masked.ImageGenerate.Keys = maskKeys(cfg.ImageGenerate.Keys)

	response.Success(c, masked)
}

// UpdateConfig updates the Baizhi configuration.
// PUT /api/v1/admin/baizhi/config
func (h *BaizhiHandler) UpdateConfig(c *gin.Context) {
	var req service.BaizhiConfig
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid baizhi configuration payload: "+err.Error())
		return
	}

	stored, err := h.baizhiService.GetConfig(c.Request.Context())
	if err != nil {
		stored = service.DefaultBaizhiConfig()
	}

	// Helper maps of existing keys and cookies by ID
	existingKeys := make(map[string]string)
	existingCookies := make(map[string]string)
	addExisting := func(keys []service.BaizhiKeyEntry) {
		for _, k := range keys {
			if k.ID != "" {
				existingKeys[k.ID] = k.APIKey
				existingCookies[k.ID] = k.Cookie
			}
		}
	}
	addExisting(stored.WebScrape.Keys)
	addExisting(stored.WebSearch.Keys)
	addExisting(stored.Voice.Keys)
	addExisting(stored.ImageGenerate.Keys)

	// Unmask any key or cookie that was submitted unchanged with asterisks
	resolveKeys := func(keys []service.BaizhiKeyEntry) []service.BaizhiKeyEntry {
		out := make([]service.BaizhiKeyEntry, len(keys))
		for i, k := range keys {
			out[i] = k
			if isBaizhiMasked(k.APIKey) && k.ID != "" {
				if oldKey, ok := existingKeys[k.ID]; ok && oldKey != "" {
					out[i].APIKey = oldKey
				}
			}
			if isBaizhiMasked(k.Cookie) && k.ID != "" {
				if oldCookie, ok := existingCookies[k.ID]; ok && oldCookie != "" {
					out[i].Cookie = oldCookie
				}
			}
		}
		return out
	}

	req.WebScrape.Keys = resolveKeys(req.WebScrape.Keys)
	req.WebSearch.Keys = resolveKeys(req.WebSearch.Keys)
	req.Voice.Keys = resolveKeys(req.Voice.Keys)
	req.ImageGenerate.Keys = resolveKeys(req.ImageGenerate.Keys)

	if err := h.baizhiService.UpdateConfig(c.Request.Context(), req); err != nil {
		response.InternalError(c, "Failed to update baizhi configuration: "+err.Error())
		return
	}

	response.Success(c, gin.H{"updated": true})
}

type BaizhiTestConnectionRequest struct {
	ServiceType   string `json:"service_type" binding:"required"`
	APIKey        string `json:"api_key"`
	KeyID         string `json:"key_id"`
	CustomBaseURL string `json:"custom_base_url"`
}

// TestConnection verifies an API key against upstream /pricing.
// POST /api/v1/admin/baizhi/test
func (h *BaizhiHandler) TestConnection(c *gin.Context) {
	var req BaizhiTestConnectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid test request: "+err.Error())
		return
	}

	apiKey := strings.TrimSpace(req.APIKey)
	if (apiKey == "" || isBaizhiMasked(apiKey)) && req.KeyID != "" {
		stored, err := h.baizhiService.GetConfig(c.Request.Context())
		if err == nil {
			findKey := func(keys []service.BaizhiKeyEntry) string {
				for _, k := range keys {
					if k.ID == req.KeyID {
						return k.APIKey
					}
				}
				return ""
			}

			switch req.ServiceType {
			case service.BaizhiServiceScrape:
				apiKey = findKey(stored.WebScrape.Keys)
			case service.BaizhiServiceSearch:
				apiKey = findKey(stored.WebSearch.Keys)
			case service.BaizhiServiceVoice:
				apiKey = findKey(stored.Voice.Keys)
			case service.BaizhiServiceImage:
				apiKey = findKey(stored.ImageGenerate.Keys)
			}
		}
	}

	if apiKey == "" || isBaizhiMasked(apiKey) {
		response.BadRequest(c, "API Key is required or cannot be resolved")
		return
	}

	result, err := h.baizhiService.TestKey(c.Request.Context(), req.ServiceType, apiKey, req.CustomBaseURL)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, result)
}

type BaizhiInspectAccountRequest struct {
	Cookie string `json:"cookie" binding:"required"`
}

// InspectAccount fetches live profile, wallet balance, and discovered API keys from baizhi.cloud.
// POST /api/v1/admin/baizhi/account/inspect
func (h *BaizhiHandler) InspectAccount(c *gin.Context) {
	var req BaizhiInspectAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Cookie or session is required: "+err.Error())
		return
	}

	result, err := h.baizhiService.InspectAccount(c.Request.Context(), req.Cookie)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, result)
}

// SyncBalances triggers a live refresh of wallet balances for all configured accounts.
// POST /api/v1/admin/baizhi/sync-balances
func (h *BaizhiHandler) SyncBalances(c *gin.Context) {
	updatedCfg, err := h.baizhiService.SyncAllBalances(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Failed to sync balances: "+err.Error())
		return
	}

	// Create a safe copy with masked keys & cookies
	masked := *updatedCfg
	maskKeys := func(keys []service.BaizhiKeyEntry) []service.BaizhiKeyEntry {
		out := make([]service.BaizhiKeyEntry, len(keys))
		for i, k := range keys {
			out[i] = k
			out[i].APIKey = maskBaizhiAPIKey(k.APIKey)
			if strings.TrimSpace(k.Cookie) != "" {
				out[i].Cookie = maskBaizhiAPIKey(k.Cookie)
			}
		}
		return out
	}

	masked.WebScrape.Keys = maskKeys(updatedCfg.WebScrape.Keys)
	masked.WebSearch.Keys = maskKeys(updatedCfg.WebSearch.Keys)
	masked.Voice.Keys = maskKeys(updatedCfg.Voice.Keys)
	masked.ImageGenerate.Keys = maskKeys(updatedCfg.ImageGenerate.Keys)

	response.Success(c, masked)
}
