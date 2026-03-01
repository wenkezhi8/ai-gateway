package admin

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

type UploadHandler struct{}

func NewUploadHandler() *UploadHandler {
	return &UploadHandler{}
}

// UploadLogo handles logo file upload.
func (h *UploadHandler) UploadLogo(c *gin.Context) {
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	// Get filename from form or use original
	filename := c.PostForm("filename")
	if filename == "" {
		filename = header.Filename
	}

	// Validate SVG extension
	if filepath.Ext(filename) != ".svg" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only SVG files are allowed"})
		return
	}

	// Determine logos directory
	logosDir := "./web/dist/logos"
	if _, statErr := os.Stat(logosDir); os.IsNotExist(statErr) {
		logosDir = "./dist/logos"
	}
	if _, statErr := os.Stat(logosDir); os.IsNotExist(statErr) {
		logosDir = "./logos"
	}

	// Create directory if not exists
	if mkdirErr := os.MkdirAll(logosDir, 0755); mkdirErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	// Create destination file
	dst := filepath.Join(logosDir, filename)
	dstFile, err := os.Create(dst)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create file"})
		return
	}
	defer dstFile.Close()

	// Copy file content
	if _, err := io.Copy(dstFile, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"path":    "/logos/" + filename,
	})
}
