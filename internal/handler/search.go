package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

type SearchHandler struct {
	httpClient *http.Client
}

type SearchRequest struct {
	Query string `json:"query" binding:"required"`
	Limit int    `json:"limit,omitempty"`
}

type SearchResult struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Snippet string `json:"snippet"`
}

type SearchResponse struct {
	Success bool           `json:"success"`
	Data    []SearchResult `json:"data,omitempty"`
	Error   string         `json:"error,omitempty"`
}

func NewSearchHandler() *SearchHandler {
	return &SearchHandler{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (h *SearchHandler) Search(c *gin.Context) {
	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, "invalid_request", "Invalid request body")
		return
	}

	if req.Query == "" {
		Error(c, http.StatusBadRequest, "invalid_request", "Query is required")
		return
	}

	if req.Limit == 0 {
		req.Limit = 5
	}
	if req.Limit > 10 {
		req.Limit = 10
	}

	apiKey := os.Getenv("SERPER_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("BING_API_KEY")
		if apiKey == "" {
			Error(c, http.StatusServiceUnavailable, "search_unavailable", "Search API not configured")
			return
		}
		results, err := h.bingSearch(req.Query, apiKey, req.Limit)
		if err != nil {
			Error(c, http.StatusInternalServerError, "search_failed", err.Error())
			return
		}
		Success(c, SearchResponse{Success: true, Data: results})
		return
	}

	results, err := h.serperSearch(req.Query, apiKey, req.Limit)
	if err != nil {
		Error(c, http.StatusInternalServerError, "search_failed", err.Error())
		return
	}

	Success(c, SearchResponse{Success: true, Data: results})
}

func (h *SearchHandler) serperSearch(query, apiKey string, limit int) ([]SearchResult, error) {
	reqURL := "https://google.serper.dev/search"

	formData := url.Values{}
	formData.Set("q", query)
	formData.Set("gl", "cn")
	formData.Set("hl", "zh-cn")

	req, err := http.NewRequest("POST", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.URL.RawQuery = formData.Encode()
	req.Header.Set("X-API-KEY", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var serperResp struct {
		Organic []struct {
			Title   string `json:"title"`
			Link    string `json:"link"`
			Snippet string `json:"snippet"`
		} `json:"organic"`
	}

	if err := json.Unmarshal(body, &serperResp); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, limit)
	for i, item := range serperResp.Organic {
		if i >= limit {
			break
		}
		results = append(results, SearchResult{
			Title:   item.Title,
			Link:    item.Link,
			Snippet: item.Snippet,
		})
	}

	return results, nil
}

func (h *SearchHandler) bingSearch(query, apiKey string, limit int) ([]SearchResult, error) {
	reqURL := fmt.Sprintf(
		"https://api.bing.microsoft.com/v7.0/search?q=%s&count=%d&responseFilter=Webpages",
		url.QueryEscape(query),
		limit,
	)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Ocp-Apim-Subscription-Key", apiKey)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var bingResp struct {
		WebPages struct {
			Value []struct {
				Name    string `json:"name"`
				URL     string `json:"url"`
				Snippet string `json:"snippet"`
			} `json:"value"`
		} `json:"webPages"`
	}

	if err := json.Unmarshal(body, &bingResp); err != nil {
		return nil, err
	}

	results := make([]SearchResult, 0, limit)
	for _, item := range bingResp.WebPages.Value {
		results = append(results, SearchResult{
			Title:   item.Name,
			Link:    item.URL,
			Snippet: item.Snippet,
		})
	}

	return results, nil
}
