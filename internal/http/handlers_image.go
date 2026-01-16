package http

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// ImageHandlers は画像アップロードを処理するハンドラーです
type ImageHandlers struct {
	uploadDir     string
	maxUploadSize int64
}

// NewImageHandlers は新しいImageHandlersを作成します
func NewImageHandlers(uploadDir string, maxUploadSize int64) *ImageHandlers {
	return &ImageHandlers{
		uploadDir:     uploadDir,
		maxUploadSize: maxUploadSize,
	}
}

// ImageUploadResponse は画像アップロード成功時のレスポンスです
type ImageUploadResponse struct {
	URL      string `json:"url"`      // 画像のURL（例: /uploads/abc123.jpg）
	Filename string `json:"filename"` // 元のファイル名
}

// 許可されるMIMEタイプとその拡張子
var allowedMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// 許可されるファイルのマジックバイト（ファイルシグネチャ）
var allowedMagicBytes = map[string][]byte{
	"image/jpeg": {0xFF, 0xD8, 0xFF},
	"image/png":  {0x89, 0x50, 0x4E, 0x47},
	"image/gif":  {0x47, 0x49, 0x46, 0x38},
	"image/webp": {0x52, 0x49, 0x46, 0x46}, // RIFF header
}

// HandleUploadImage は画像アップロードを処理します
// POST /api/v1/images
func (h *ImageHandlers) HandleUploadImage(w http.ResponseWriter, r *http.Request) {
	// リクエストボディのサイズ制限
	r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadSize)

	// multipart/form-dataのパース
	if err := r.ParseMultipartForm(h.maxUploadSize); err != nil {
		if err.Error() == "http: request body too large" {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("File too large. Maximum size is %d bytes", h.maxUploadSize))
			return
		}
		writeError(w, http.StatusBadRequest, "Invalid form data")
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			r.MultipartForm.RemoveAll()
		}
	}()

	// ファイルを取得
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "No image file provided")
		return
	}
	defer file.Close()

	// ファイルサイズの検証
	if header.Size > h.maxUploadSize {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("File too large. Maximum size is %d bytes", h.maxUploadSize))
		return
	}

	// Content-Typeの取得と検証
	contentType := header.Header.Get("Content-Type")
	ext, ok := allowedMimeTypes[contentType]
	if !ok {
		writeError(w, http.StatusBadRequest, "Invalid file type. Allowed types: JPEG, PNG, GIF, WebP")
		return
	}

	// ファイル内容を読み込み（マジックバイト検証用）
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		log.Printf("failed to read file: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to process image")
		return
	}

	// マジックバイトの検証
	if !validateMagicBytes(fileBytes, contentType) {
		writeError(w, http.StatusBadRequest, "Invalid file content. File type does not match content")
		return
	}

	// 一意なファイル名を生成（UUID v4 + 拡張子）
	newFilename := uuid.New().String() + ext

	// ファイルパスの構築（ディレクトリトラバーサル防止）
	safePath := filepath.Join(h.uploadDir, filepath.Base(newFilename))

	// ファイルを保存
	if err := os.WriteFile(safePath, fileBytes, 0644); err != nil {
		log.Printf("failed to save file: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to save image")
		return
	}

	// レスポンスを返す
	response := ImageUploadResponse{
		URL:      "/uploads/" + newFilename,
		Filename: sanitizeFilename(header.Filename),
	}

	writeJSON(w, http.StatusCreated, response)
}

// validateMagicBytes はファイルのマジックバイトを検証します
func validateMagicBytes(data []byte, contentType string) bool {
	expectedMagic, ok := allowedMagicBytes[contentType]
	if !ok {
		return false
	}

	if len(data) < len(expectedMagic) {
		return false
	}

	// WebPの場合は特殊な検証（RIFFヘッダー + WEBPフォーマット）
	if contentType == "image/webp" {
		if len(data) < 12 {
			return false
		}
		// RIFF....WEBP のパターンをチェック
		return bytes.HasPrefix(data, expectedMagic) && string(data[8:12]) == "WEBP"
	}

	return bytes.HasPrefix(data, expectedMagic)
}

// sanitizeFilename はファイル名から危険な文字を除去します
func sanitizeFilename(filename string) string {
	// パス区切り文字を除去
	filename = filepath.Base(filename)
	// 制御文字や特殊文字を除去
	filename = strings.Map(func(r rune) rune {
		if r < 32 || r == '<' || r == '>' || r == ':' || r == '"' || r == '/' || r == '\\' || r == '|' || r == '?' || r == '*' {
			return -1
		}
		return r
	}, filename)
	return filename
}
