package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var wsUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for API gateway
	},
	ReadBufferSize:  64 * 1024,
	WriteBufferSize: 64 * 1024,
}

type BaizhiGatewayHandler struct {
	baizhiService *service.BaizhiService
	apiKeyService *service.APIKeyService
	httpClient    *http.Client
}

func NewBaizhiGatewayHandler(
	baizhiService *service.BaizhiService,
	apiKeyService *service.APIKeyService,
) *BaizhiGatewayHandler {
	return &BaizhiGatewayHandler{
		baizhiService: baizhiService,
		apiKeyService: apiKeyService,
		httpClient: &http.Client{
			Timeout: 10 * time.Minute, // Long timeout for streaming and large exports
			Transport: &http.Transport{
				MaxIdleConns:        500,
				MaxIdleConnsPerHost: 100,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// verifyAuth checks whether caller is authenticated if AuthRequired is enabled.
func (h *BaizhiGatewayHandler) verifyAuth(c *gin.Context) bool {
	cfg, err := h.baizhiService.GetConfig(c.Request.Context())
	if err != nil || !cfg.AuthRequired {
		return true
	}

	authHeader := c.GetHeader("Authorization")
	token := ""
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	} else if qKey := c.Query("api_key"); qKey != "" {
		token = qKey
	}

	if token == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"type":    "unauthorized",
				"message": "Missing required API key in Authorization header or api_key parameter",
			},
		})
		return false
	}

	// Validate against Sub2API api_keys if apiKeyService is available
	if h.apiKeyService != nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		apiKey, user, err := h.apiKeyService.ValidateKey(ctx, token)
		if err != nil || apiKey == nil || user == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{
					"type":    "unauthorized",
					"message": "Invalid or expired Sub2API API key",
				},
			})
			return false
		}
		_ = h.apiKeyService.TouchLastUsed(context.Background(), apiKey.ID)
	}

	return true
}

// normalizeScrapeSubpath resolves short aliases to official OpenAPI paths.
func normalizeScrapeSubpath(subpath string) string {
	s := "/" + strings.TrimLeft(subpath, "/")
	if strings.HasPrefix(s, "/openapi") {
		return s
	}
	switch s {
	case "/scrape":
		return "/openapi/scrape"
	case "/extract":
		return "/openapi/extract"
	case "/extract/suggest":
		return "/openapi/extract/suggest"
	case "/extract/batch":
		return "/openapi/extract/batch"
	case "/extract/organize":
		return "/openapi/extract/organize"
	case "/pricing", "/v1/pricing":
		return "/openapi/v1/pricing"
	default:
		return "/openapi" + s
	}
}

// normalizeSearchSubpath resolves short search aliases.
func normalizeSearchSubpath(subpath string) string {
	s := "/" + strings.TrimLeft(subpath, "/")
	if strings.HasPrefix(s, "/openapi") {
		return s
	}
	switch s {
	case "/search":
		return "/openapi/search"
	case "/image_search_by_text":
		return "/openapi/image_search_by_text"
	case "/image_search_by_image":
		return "/openapi/image_search_by_image"
	case "/news_search":
		return "/openapi/news_search"
	case "/ai_search":
		return "/openapi/ai_search"
	case "/pricing", "/v1/pricing":
		return "/openapi/v1/pricing"
	default:
		return "/openapi" + s
	}
}

// normalizeVoiceSubpath resolves voice endpoints.
func normalizeVoiceSubpath(subpath string, baseURL string) string {
	s := "/" + strings.TrimLeft(subpath, "/")
	// If upstream baseURL ends in /openapi/v1, strip prefix if client supplied it
	if strings.HasSuffix(strings.TrimRight(baseURL, "/"), "/openapi/v1") {
		s = strings.TrimPrefix(s, "/openapi/v1")
		if !strings.HasPrefix(s, "/") {
			s = "/" + s
		}
	}
	return s
}

// normalizeImageSubpath resolves image generation endpoints.
func normalizeImageSubpath(subpath string, baseURL string) string {
	s := "/" + strings.TrimLeft(subpath, "/")
	if strings.HasSuffix(strings.TrimRight(baseURL, "/"), "/openapi/v1") {
		s = strings.TrimPrefix(s, "/openapi/v1")
		if !strings.HasPrefix(s, "/") {
			s = "/" + s
		}
	}
	return s
}

// ForwardScrape forwards requests to Baizhi Web Scrape service.
func (h *BaizhiGatewayHandler) ForwardScrape(c *gin.Context) {
	if !h.verifyAuth(c) {
		return
	}

	key, baseURL, _, err := h.baizhiService.SelectKey(c.Request.Context(), service.BaizhiServiceScrape)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{"type": "service_unavailable", "message": err.Error()},
		})
		return
	}

	subpath := normalizeScrapeSubpath(c.Param("subpath"))
	targetURL := strings.TrimRight(baseURL, "/") + subpath
	h.forwardHTTP(c, targetURL, key.APIKey)
}

// ForwardSearch forwards requests to Baizhi Web Search service.
func (h *BaizhiGatewayHandler) ForwardSearch(c *gin.Context) {
	if !h.verifyAuth(c) {
		return
	}

	key, baseURL, _, err := h.baizhiService.SelectKey(c.Request.Context(), service.BaizhiServiceSearch)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{"type": "service_unavailable", "message": err.Error()},
		})
		return
	}

	subpath := normalizeSearchSubpath(c.Param("subpath"))
	targetURL := strings.TrimRight(baseURL, "/") + subpath
	h.forwardHTTP(c, targetURL, key.APIKey)
}

// ForwardVoice forwards requests to Baizhi Voice service (supports HTTP and WebSocket).
func (h *BaizhiGatewayHandler) ForwardVoice(c *gin.Context) {
	if !h.verifyAuth(c) {
		return
	}

	key, baseURL, wssURL, err := h.baizhiService.SelectKey(c.Request.Context(), service.BaizhiServiceVoice)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{"type": "service_unavailable", "message": err.Error()},
		})
		return
	}

	rawSubpath := c.Param("subpath")
	if strings.Contains(rawSubpath, "realtime") || c.IsWebsocket() {
		wsTargetURL := strings.TrimRight(wssURL, "/") + "/realtime"
		h.forwardWebSocket(c, wsTargetURL, key.APIKey)
		return
	}

	subpath := normalizeVoiceSubpath(rawSubpath, baseURL)
	targetURL := strings.TrimRight(baseURL, "/") + subpath
	h.forwardHTTP(c, targetURL, key.APIKey)
}

// ForwardImage forwards requests to Baizhi Image Generation service.
func (h *BaizhiGatewayHandler) ForwardImage(c *gin.Context) {
	if !h.verifyAuth(c) {
		return
	}

	key, baseURL, _, err := h.baizhiService.SelectKey(c.Request.Context(), service.BaizhiServiceImage)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": gin.H{"type": "service_unavailable", "message": err.Error()},
		})
		return
	}

	subpath := normalizeImageSubpath(c.Param("subpath"), baseURL)
	targetURL := strings.TrimRight(baseURL, "/") + subpath
	h.forwardHTTP(c, targetURL, key.APIKey)
}

// forwardHTTP executes the standard HTTP reverse proxy with SSE streaming support.
func (h *BaizhiGatewayHandler) forwardHTTP(c *gin.Context, targetURL, apiKey string) {
	if c.Request.URL.RawQuery != "" {
		targetURL += "?" + c.Request.URL.RawQuery
	}

	req, err := http.NewRequestWithContext(c.Request.Context(), c.Request.Method, targetURL, c.Request.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"type": "proxy_error", "message": "Failed to create upstream request: " + err.Error()},
		})
		return
	}

	// Copy original headers
	for k, vv := range c.Request.Header {
		lowerK := strings.ToLower(k)
		if lowerK == "host" || lowerK == "authorization" || lowerK == "connection" {
			continue
		}
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}

	// Inject upstream API Key
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error": gin.H{"type": "upstream_error", "message": "Upstream service error: " + err.Error()},
		})
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for k, vv := range resp.Header {
		for _, v := range vv {
			c.Writer.Header().Add(k, v)
		}
	}

	c.Writer.WriteHeader(resp.StatusCode)

	// Detect if SSE stream or chunked transfer
	contentType := resp.Header.Get("Content-Type")
	isSSE := strings.Contains(contentType, "text/event-stream")

	if isSSE {
		c.Writer.Header().Set("Cache-Control", "no-cache")
		c.Writer.Header().Set("Connection", "keep-alive")
		c.Writer.Header().Set("X-Accel-Buffering", "no")
		c.Writer.Flush()

		buf := make([]byte, 4096)
		for {
			n, rErr := resp.Body.Read(buf)
			if n > 0 {
				_, wErr := c.Writer.Write(buf[:n])
				c.Writer.Flush()
				if wErr != nil {
					return
				}
			}
			if rErr != nil {
				return
			}
		}
	} else {
		_, _ = io.Copy(c.Writer, resp.Body)
	}
}

// forwardWebSocket proxies bi-directional WebSocket frames for Baizhi Realtime Voice.
func (h *BaizhiGatewayHandler) forwardWebSocket(c *gin.Context, upstreamURL, apiKey string) {
	clientWS, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer clientWS.Close()

	upstreamHeader := http.Header{}
	upstreamHeader.Set("Authorization", "Bearer "+apiKey)

	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = 15 * time.Second

	upstreamWS, resp, err := dialer.DialContext(c.Request.Context(), upstreamURL, upstreamHeader)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to connect upstream WebSocket: %v", err)
		if resp != nil {
			errMsg = fmt.Sprintf("Upstream returned HTTP %d", resp.StatusCode)
		}
		_ = clientWS.WriteJSON(gin.H{
			"type":    "error",
			"code":    "upstream_connection_error",
			"message": errMsg,
		})
		return
	}
	defer upstreamWS.Close()

	errChan := make(chan error, 2)

	// Goroutine 1: Client -> Upstream
	go func() {
		for {
			msgType, data, rErr := clientWS.ReadMessage()
			if rErr != nil {
				errChan <- rErr
				return
			}
			if wErr := upstreamWS.WriteMessage(msgType, data); wErr != nil {
				errChan <- wErr
				return
			}
		}
	}()

	// Goroutine 2: Upstream -> Client
	go func() {
		for {
			msgType, data, rErr := upstreamWS.ReadMessage()
			if rErr != nil {
				errChan <- rErr
				return
			}
			if wErr := clientWS.WriteMessage(msgType, data); wErr != nil {
				errChan <- wErr
				return
			}
		}
	}()

	// Wait for any error / disconnect
	<-errChan
}
