package admin

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const defaultCollectionID = "default"
const defaultEmbeddingModel = "nomic-embed-text"

type KnowledgeHandler struct {
	db *sql.DB
}

type knowledgeConfig struct {
	VectorBackend  string `json:"vector_backend"`
	EmbeddingModel string `json:"embedding_model"`
	Chunking       struct {
		Type         string `json:"type"`
		ChunkSize    int    `json:"chunk_size"`
		ChunkOverlap int    `json:"chunk_overlap"`
	} `json:"chunking_strategy"`
	Retrieval struct {
		TopK                int     `json:"top_k"`
		SimilarityThreshold float64 `json:"similarity_threshold"`
	} `json:"retrieval"`
}

func NewKnowledgeHandler(db *sql.DB) *KnowledgeHandler {
	h := &KnowledgeHandler{db: db}
	h.ensureSchema()
	h.ensureDefaultConfig()
	return h
}

func (h *KnowledgeHandler) ensureSchema() {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS kb_documents (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			type TEXT NOT NULL,
			size INTEGER NOT NULL,
			chunk_count INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL,
			collection_id TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS kb_chunks (
			id TEXT PRIMARY KEY,
			document_id TEXT NOT NULL,
			content TEXT NOT NULL,
			score REAL NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			FOREIGN KEY(document_id) REFERENCES kb_documents(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS kb_config (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			payload TEXT NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
	}

	for _, stmt := range stmts {
		if _, err := h.db.Exec(stmt); err != nil {
			panic(fmt.Sprintf("initialize knowledge schema failed: %v", err))
		}
	}
}

func defaultKnowledgeConfig() knowledgeConfig {
	var cfg knowledgeConfig
	cfg.VectorBackend = "sqlite"
	cfg.EmbeddingModel = defaultEmbeddingModel
	cfg.Chunking.Type = "fixed_size"
	cfg.Chunking.ChunkSize = 500
	cfg.Chunking.ChunkOverlap = 50
	cfg.Retrieval.TopK = 5
	cfg.Retrieval.SimilarityThreshold = 0.7
	return cfg
}

func (h *KnowledgeHandler) ensureDefaultConfig() {
	var count int
	if err := h.db.QueryRow(`SELECT COUNT(1) FROM kb_config WHERE id = 1`).Scan(&count); err != nil {
		panic(fmt.Sprintf("query knowledge config failed: %v", err))
	}
	if count > 0 {
		return
	}

	b, err := json.Marshal(defaultKnowledgeConfig())
	if err != nil {
		panic(fmt.Sprintf("marshal default knowledge config failed: %v", err))
	}
	if _, err := h.db.Exec(`INSERT INTO kb_config(id, payload, updated_at) VALUES (1, ?, ?)`, string(b), time.Now().UTC()); err != nil {
		panic(fmt.Sprintf("insert default knowledge config failed: %v", err))
	}
}

func (h *KnowledgeHandler) ListDocuments(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}
	status := strings.TrimSpace(c.Query("status"))
	search := strings.TrimSpace(c.Query("search"))

	total, rows, err := h.queryDocumentList(status, search, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	defer rows.Close()

	items := make([]gin.H, 0)
	for rows.Next() {
		var (
			id           string
			name         string
			typ          string
			size         int64
			chunkCount   int
			docStatus    string
			collectionID string
			createdAt    time.Time
			updatedAt    time.Time
		)
		if err := rows.Scan(&id, &name, &typ, &size, &chunkCount, &docStatus, &collectionID, &createdAt, &updatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
		items = append(items, gin.H{
			"id":            id,
			"name":          name,
			"type":          typ,
			"size":          size,
			"chunk_count":   chunkCount,
			"status":        docStatus,
			"collection_id": collectionID,
			"created_at":    createdAt,
			"updated_at":    updatedAt,
		})
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"total": total,
			"items": items,
		},
	})
}

func (h *KnowledgeHandler) queryDocumentList(status, search string, page, pageSize int) (int, *sql.Rows, error) {
	var total int
	var err error
	searchLike := "%" + search + "%"
	switch {
	case status != "" && search != "":
		err = h.db.QueryRow(`SELECT COUNT(1) FROM kb_documents WHERE status = ? AND name LIKE ?`, status, searchLike).Scan(&total)
	case status != "":
		err = h.db.QueryRow(`SELECT COUNT(1) FROM kb_documents WHERE status = ?`, status).Scan(&total)
	case search != "":
		err = h.db.QueryRow(`SELECT COUNT(1) FROM kb_documents WHERE name LIKE ?`, searchLike).Scan(&total)
	default:
		err = h.db.QueryRow(`SELECT COUNT(1) FROM kb_documents`).Scan(&total)
	}
	if err != nil {
		return 0, nil, err
	}

	offset := (page - 1) * pageSize
	var rows *sql.Rows
	switch {
	case status != "" && search != "":
		rows, err = h.db.Query(`SELECT id, name, type, size, chunk_count, status, collection_id, created_at, updated_at
			FROM kb_documents WHERE status = ? AND name LIKE ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, status, searchLike, pageSize, offset)
	case status != "":
		rows, err = h.db.Query(`SELECT id, name, type, size, chunk_count, status, collection_id, created_at, updated_at
			FROM kb_documents WHERE status = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, status, pageSize, offset)
	case search != "":
		rows, err = h.db.Query(`SELECT id, name, type, size, chunk_count, status, collection_id, created_at, updated_at
			FROM kb_documents WHERE name LIKE ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, searchLike, pageSize, offset)
	default:
		rows, err = h.db.Query(`SELECT id, name, type, size, chunk_count, status, collection_id, created_at, updated_at
			FROM kb_documents ORDER BY created_at DESC LIMIT ? OFFSET ?`, pageSize, offset)
	}
	if err != nil {
		return 0, nil, err
	}
	return total, rows, nil
}

func (h *KnowledgeHandler) UploadDocument(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "file is required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	defer file.Close()

	body, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	collectionID := strings.TrimSpace(c.PostForm("collection"))
	if collectionID == "" {
		collectionID = defaultCollectionID
	}

	docID, err := h.storeUploadedDocument(fileHeader, body, collectionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"document_id": docID,
			"status":      "completed",
			"message":     "文档上传并处理成功",
		},
	})
}

func (h *KnowledgeHandler) storeUploadedDocument(fileHeader *multipart.FileHeader, body []byte, collectionID string) (string, error) {
	docID := "doc_" + uuid.NewString()
	now := time.Now().UTC()
	docType := strings.TrimPrefix(strings.ToLower(filepath.Ext(fileHeader.Filename)), ".")
	if docType == "" {
		docType = "txt"
	}

	tx, err := h.db.Begin()
	if err != nil {
		return "", err
	}
	defer func() {
		rbErr := tx.Rollback()
		if rbErr != nil && rbErr != sql.ErrTxDone {
			return
		}
	}()

	_, err = tx.Exec(`INSERT INTO kb_documents(id, name, type, size, chunk_count, status, collection_id, created_at, updated_at)
		VALUES(?, ?, ?, ?, 0, ?, ?, ?, ?)`, docID, fileHeader.Filename, docType, fileHeader.Size, "processing", collectionID, now, now)
	if err != nil {
		return "", err
	}

	chunks := chunkText(string(body), 500, 50)
	if len(chunks) == 0 {
		chunks = []string{strings.TrimSpace(string(body))}
	}

	insertChunkStmt, err := tx.Prepare(`INSERT INTO kb_chunks(id, document_id, content, score, created_at) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		return "", err
	}
	defer insertChunkStmt.Close()

	for _, chunk := range chunks {
		trimmed := strings.TrimSpace(chunk)
		if trimmed == "" {
			continue
		}
		if _, execErr := insertChunkStmt.Exec("chunk_"+uuid.NewString(), docID, trimmed, 0, now); execErr != nil {
			return "", execErr
		}
	}

	_, err = tx.Exec(`UPDATE kb_documents SET chunk_count = (SELECT COUNT(1) FROM kb_chunks WHERE document_id = ?), status = ?, updated_at = ? WHERE id = ?`, docID, "completed", now, docID)
	if err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}

	return docID, nil
}

func (h *KnowledgeHandler) GetDocument(c *gin.Context) {
	id := c.Param("id")
	row := h.db.QueryRow(`SELECT id, name, type, size, chunk_count, status, collection_id, created_at, updated_at FROM kb_documents WHERE id = ?`, id)
	var (
		docID        string
		name         string
		typ          string
		size         int64
		chunkCount   int
		status       string
		collectionID string
		createdAt    time.Time
		updatedAt    time.Time
	)
	if err := row.Scan(&docID, &name, &typ, &size, &chunkCount, &status, &collectionID, &createdAt, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "document not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	chunks, err := h.listChunksByDocument(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":            docID,
			"name":          name,
			"type":          typ,
			"size":          size,
			"chunk_count":   chunkCount,
			"status":        status,
			"collection_id": collectionID,
			"chunks":        chunks,
			"created_at":    createdAt,
			"updated_at":    updatedAt,
		},
	})
}

func (h *KnowledgeHandler) DeleteDocument(c *gin.Context) {
	id := c.Param("id")
	res, err := h.db.Exec(`DELETE FROM kb_documents WHERE id = ?`, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	affected, raErr := res.RowsAffected()
	if raErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": raErr.Error()})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "document not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "文档删除成功"}})
}

func (h *KnowledgeHandler) VectorizeDocument(c *gin.Context) {
	id := c.Param("id")
	res, err := h.db.Exec(`UPDATE kb_documents SET status = ?, updated_at = ? WHERE id = ?`, "completed", time.Now().UTC(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	affected, raErr := res.RowsAffected()
	if raErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": raErr.Error()})
		return
	}
	if affected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "document not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "已重新向量化"}})
}

func (h *KnowledgeHandler) ListChunks(c *gin.Context) {
	documentID := strings.TrimSpace(c.Query("document_id"))
	if documentID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "document_id is required"})
		return
	}
	chunks, err := h.listChunksByDocument(documentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"total": len(chunks), "items": chunks}})
}

func (h *KnowledgeHandler) GetChunk(c *gin.Context) {
	chunkID := c.Param("id")
	row := h.db.QueryRow(`SELECT id, document_id, content, score, created_at FROM kb_chunks WHERE id = ?`, chunkID)
	var (
		id         string
		documentID string
		content    string
		score      float64
		createdAt  time.Time
	)
	if err := row.Scan(&id, &documentID, &content, &score, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "chunk not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"id": id, "document_id": documentID, "content": content, "score": score, "created_at": createdAt}})
}

func (h *KnowledgeHandler) ChatMessage(c *gin.Context) {
	var req struct {
		Query               string  `json:"query" binding:"required"`
		CollectionID        string  `json:"collection_id"`
		TopK                int     `json:"top_k"`
		SimilarityThreshold float64 `json:"similarity_threshold"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	if req.TopK <= 0 {
		req.TopK = 5
	}

	keywords := buildSearchKeywords(req.Query)
	if len(keywords) == 0 {
		keywords = []string{strings.TrimSpace(req.Query)}
	}
	collectionID := strings.TrimSpace(req.CollectionID)
	if collectionID == "" {
		collectionID = defaultCollectionID
	}

	rows, err := h.db.Query(`SELECT c.id, c.document_id, d.name, c.content
		FROM kb_chunks c
		JOIN kb_documents d ON d.id = c.document_id
		WHERE d.collection_id = ?
		ORDER BY c.created_at DESC
		LIMIT ?`, collectionID, 200)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	defer rows.Close()

	sources := make([]gin.H, 0)
	answerParts := make([]string, 0)
	for rows.Next() {
		var chunkID, documentID, documentName, content string
		if err := rows.Scan(&chunkID, &documentID, &documentName, &content); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
			return
		}
		if !matchesAnyKeyword(content, keywords) {
			continue
		}
		snippet := content
		if len([]rune(snippet)) > 120 {
			snippet = string([]rune(snippet)[:120]) + "..."
		}
		sources = append(sources, gin.H{
			"chunk_id":      chunkID,
			"document_id":   documentID,
			"document_name": documentName,
			"content":       content,
			"score":         1,
		})
		answerParts = append(answerParts, fmt.Sprintf("- %s", snippet))
		if len(sources) >= req.TopK {
			break
		}
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	answer := "未检索到相关内容，请先上传文档。"
	if len(answerParts) > 0 {
		answer = "根据知识库检索结果：\n" + strings.Join(answerParts, "\n")
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"answer":  answer,
			"sources": sources,
			"metadata": gin.H{
				"retrieval_time": "0s",
				"top_k":          req.TopK,
			},
		},
	})
}

func buildSearchKeywords(query string) []string {
	cleaned := strings.NewReplacer("？", " ", "?", " ", "，", " ", ",", " ", "。", " ", ".", " ", "！", " ", "!", " ", "：", " ", ":", " ", "；", " ", ";", " ").Replace(strings.TrimSpace(query))
	parts := strings.Fields(cleaned)
	if len(parts) > 1 {
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if len([]rune(part)) >= 2 {
				result = append(result, part)
			}
		}
		return result
	}
	runes := []rune(strings.TrimSpace(query))
	if len(runes) <= 2 {
		return []string{strings.TrimSpace(query)}
	}
	result := make([]string, 0)
	for i := 0; i < len(runes)-1; i++ {
		result = append(result, string(runes[i:i+2]))
	}
	if len(result) > 6 {
		return result[:6]
	}
	return result
}

func (h *KnowledgeHandler) GetConfig(c *gin.Context) {
	var payload string
	if err := h.db.QueryRow(`SELECT payload FROM kb_config WHERE id = 1`).Scan(&payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}

	cfgMap := map[string]any{}
	if err := json.Unmarshal([]byte(payload), &cfgMap); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	cfgMap["collections"] = h.listCollections()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": cfgMap})
}

func (h *KnowledgeHandler) UpdateConfig(c *gin.Context) {
	var update map[string]any
	if err := c.ShouldBindJSON(&update); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": err.Error()})
		return
	}

	var payload string
	if err := h.db.QueryRow(`SELECT payload FROM kb_config WHERE id = 1`).Scan(&payload); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	base := map[string]any{}
	if err := json.Unmarshal([]byte(payload), &base); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	mergeMap(base, update)
	merged, err := json.Marshal(base)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	if _, err := h.db.Exec(`UPDATE kb_config SET payload = ?, updated_at = ? WHERE id = 1`, string(merged), time.Now().UTC()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "配置更新成功"}})
}

func (h *KnowledgeHandler) listChunksByDocument(documentID string) ([]gin.H, error) {
	rows, err := h.db.Query(`SELECT id, content, score, created_at FROM kb_chunks WHERE document_id = ? ORDER BY created_at ASC`, documentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	chunks := make([]gin.H, 0)
	for rows.Next() {
		var id, content string
		var score float64
		var createdAt time.Time
		if err := rows.Scan(&id, &content, &score, &createdAt); err != nil {
			return nil, err
		}
		chunks = append(chunks, gin.H{"id": id, "document_id": documentID, "content": content, "score": score, "created_at": createdAt})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return chunks, nil
}

func (h *KnowledgeHandler) listCollections() []gin.H {
	rows, err := h.db.Query(`SELECT collection_id, COUNT(1), COALESCE(SUM(chunk_count), 0) FROM kb_documents GROUP BY collection_id ORDER BY collection_id`)
	if err != nil {
		return []gin.H{{"id": defaultCollectionID, "name": "默认知识库", "document_count": 0, "chunk_count": 0}}
	}
	defer rows.Close()

	collections := make([]gin.H, 0)
	for rows.Next() {
		var id string
		var docCount, chunkCount int
		if err := rows.Scan(&id, &docCount, &chunkCount); err != nil {
			continue
		}
		collections = append(collections, gin.H{"id": id, "name": id, "document_count": docCount, "chunk_count": chunkCount})
	}
	if err := rows.Err(); err != nil {
		return []gin.H{{"id": defaultCollectionID, "name": "默认知识库", "document_count": 0, "chunk_count": 0}}
	}
	if len(collections) == 0 {
		collections = append(collections, gin.H{"id": defaultCollectionID, "name": "默认知识库", "document_count": 0, "chunk_count": 0})
	}
	return collections
}

func parsePositiveInt(raw string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}

func chunkText(content string, size, overlap int) []string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return nil
	}
	runes := []rune(trimmed)
	if len(runes) <= size {
		return []string{trimmed}
	}
	if overlap < 0 {
		overlap = 0
	}
	if overlap >= size {
		overlap = size / 4
	}

	chunks := make([]string, 0)
	step := size - overlap
	for start := 0; start < len(runes); start += step {
		end := start + size
		if end > len(runes) {
			end = len(runes)
		}
		chunk := strings.TrimSpace(string(runes[start:end]))
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
		if end == len(runes) {
			break
		}
	}
	return chunks
}

func mergeMap(dst, src map[string]any) {
	for k, v := range src {
		srcMap, srcIsMap := v.(map[string]any)
		dstMap, dstIsMap := dst[k].(map[string]any)
		if srcIsMap && dstIsMap {
			mergeMap(dstMap, srcMap)
			dst[k] = dstMap
			continue
		}
		dst[k] = v
	}
}

func matchesAnyKeyword(content string, keywords []string) bool {
	text := strings.ToLower(content)
	for _, keyword := range keywords {
		if keyword == "" {
			continue
		}
		if strings.Contains(text, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}
