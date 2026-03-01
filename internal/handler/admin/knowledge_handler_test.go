package admin

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"ai-gateway/internal/storage"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnowledgeHandler_DocumentLifecycle(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "knowledge-handler.db")
	store, err := storage.NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	h := NewKnowledgeHandler(store.GetDB())
	r := gin.New()
	r.POST("/admin/knowledge/documents/upload", h.UploadDocument)
	r.GET("/admin/knowledge/documents", h.ListDocuments)
	r.GET("/admin/knowledge/documents/:id", h.GetDocument)
	r.DELETE("/admin/knowledge/documents/:id", h.DeleteDocument)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "gateway-guide.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("AI Gateway 支持向量检索与问答。"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	uploadReq := httptest.NewRequest(http.MethodPost, "/admin/knowledge/documents/upload", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadResp := httptest.NewRecorder()
	r.ServeHTTP(uploadResp, uploadReq)
	require.Equal(t, http.StatusOK, uploadResp.Code)

	var uploadPayload struct {
		Success bool `json:"success"`
		Data    struct {
			DocumentID string `json:"document_id"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(uploadResp.Body.Bytes(), &uploadPayload))
	require.True(t, uploadPayload.Success)
	require.NotEmpty(t, uploadPayload.Data.DocumentID)

	listReq := httptest.NewRequest(http.MethodGet, "/admin/knowledge/documents?page=1&page_size=20", http.NoBody)
	listResp := httptest.NewRecorder()
	r.ServeHTTP(listResp, listReq)
	require.Equal(t, http.StatusOK, listResp.Code)

	var listPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Total int                      `json:"total"`
			Items []map[string]interface{} `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listResp.Body.Bytes(), &listPayload))
	require.True(t, listPayload.Success)
	assert.Equal(t, 1, listPayload.Data.Total)
	require.Len(t, listPayload.Data.Items, 1)

	docID := uploadPayload.Data.DocumentID
	getReq := httptest.NewRequest(http.MethodGet, "/admin/knowledge/documents/"+docID, http.NoBody)
	getResp := httptest.NewRecorder()
	r.ServeHTTP(getResp, getReq)
	require.Equal(t, http.StatusOK, getResp.Code)

	var getPayload struct {
		Success bool                   `json:"success"`
		Data    map[string]interface{} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(getResp.Body.Bytes(), &getPayload))
	require.True(t, getPayload.Success)
	assert.Equal(t, "completed", getPayload.Data["status"])

	delReq := httptest.NewRequest(http.MethodDelete, "/admin/knowledge/documents/"+docID, http.NoBody)
	delResp := httptest.NewRecorder()
	r.ServeHTTP(delResp, delReq)
	require.Equal(t, http.StatusOK, delResp.Code)
}

func TestKnowledgeHandler_ChatAndConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dbPath := filepath.Join(t.TempDir(), "knowledge-handler-chat.db")
	store, err := storage.NewSQLiteStorage(dbPath)
	require.NoError(t, err)
	defer store.Close()

	h := NewKnowledgeHandler(store.GetDB())
	r := gin.New()
	r.POST("/admin/knowledge/documents/upload", h.UploadDocument)
	r.POST("/admin/knowledge/chat/message", h.ChatMessage)
	r.GET("/admin/knowledge/config", h.GetConfig)
	r.PUT("/admin/knowledge/config", h.UpdateConfig)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "faq.txt")
	require.NoError(t, err)
	_, err = part.Write([]byte("AI Gateway 默认向量后端是 SQLite，支持 Qdrant。"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	uploadReq := httptest.NewRequest(http.MethodPost, "/admin/knowledge/documents/upload", body)
	uploadReq.Header.Set("Content-Type", writer.FormDataContentType())
	uploadResp := httptest.NewRecorder()
	r.ServeHTTP(uploadResp, uploadReq)
	require.Equal(t, http.StatusOK, uploadResp.Code)

	chatBody := bytes.NewBufferString(`{"query":"默认向量后端是什么？","top_k":3}`)
	chatReq := httptest.NewRequest(http.MethodPost, "/admin/knowledge/chat/message", chatBody)
	chatReq.Header.Set("Content-Type", "application/json")
	chatResp := httptest.NewRecorder()
	r.ServeHTTP(chatResp, chatReq)
	require.Equal(t, http.StatusOK, chatResp.Code)

	var chatPayload struct {
		Success bool `json:"success"`
		Data    struct {
			Answer  string `json:"answer"`
			Sources []any  `json:"sources"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(chatResp.Body.Bytes(), &chatPayload))
	require.True(t, chatPayload.Success)
	assert.NotEmpty(t, chatPayload.Data.Answer)
	assert.NotEmpty(t, chatPayload.Data.Sources)

	updateReq := httptest.NewRequest(http.MethodPut, "/admin/knowledge/config", bytes.NewBufferString(`{"vector_backend":"qdrant","retrieval":{"top_k":7,"similarity_threshold":0.8}}`))
	updateReq.Header.Set("Content-Type", "application/json")
	updateResp := httptest.NewRecorder()
	r.ServeHTTP(updateResp, updateReq)
	require.Equal(t, http.StatusOK, updateResp.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/admin/knowledge/config", http.NoBody)
	getResp := httptest.NewRecorder()
	r.ServeHTTP(getResp, getReq)
	require.Equal(t, http.StatusOK, getResp.Code)

	var configPayload struct {
		Success bool `json:"success"`
		Data    struct {
			VectorBackend string `json:"vector_backend"`
			Retrieval     struct {
				TopK int `json:"top_k"`
			} `json:"retrieval"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(getResp.Body.Bytes(), &configPayload))
	require.True(t, configPayload.Success)
	assert.Equal(t, "qdrant", configPayload.Data.VectorBackend)
	assert.Equal(t, 7, configPayload.Data.Retrieval.TopK)
}
