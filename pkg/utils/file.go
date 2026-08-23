package utils

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const MaxUploadSize int64 = 8 << 20

func SaveUpload(file multipart.File, header *multipart.FileHeader, directory string) (string, error) {
	defer file.Close()
	if header == nil {
		return "", fmt.Errorf("上传文件信息缺失")
	}
	if header.Size > MaxUploadSize {
		return "", fmt.Errorf("图片不能超过 8MB")
	}
	extension := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtension(extension) {
		return "", fmt.Errorf("仅支持 jpg、png、webp 图片")
	}

	content, err := io.ReadAll(io.LimitReader(file, MaxUploadSize+1))
	if err != nil {
		return "", fmt.Errorf("读取上传文件: %w", err)
	}
	if int64(len(content)) > MaxUploadSize {
		return "", fmt.Errorf("图片不能超过 8MB")
	}
	contentType := detectImageType(content)
	if !matchesExtension(extension, contentType) {
		return "", fmt.Errorf("文件内容不是有效的 %s 图片", strings.TrimPrefix(extension, "."))
	}

	filename := uuid.NewString() + extension
	path := filepath.Join(directory, filename)
	if err := os.WriteFile(path, content, 0644); err != nil {
		return "", fmt.Errorf("create upload: %w", err)
	}
	return "/uploads/" + filename, nil
}

func DeleteUpload(uploadURL, directory string) error {
	if uploadURL == "" {
		return nil
	}
	filename := filepath.Base(filepath.Clean(uploadURL))
	if filename == "." || filename == string(filepath.Separator) || filename == "" {
		return nil
	}
	path := filepath.Join(directory, filename)
	if filepath.Dir(path) != filepath.Clean(directory) {
		return fmt.Errorf("拒绝删除上传目录之外的文件")
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete upload: %w", err)
	}
	return nil
}

func DeleteUploadAfterUse(uploadURL, directory string) error {
	if strings.TrimSpace(uploadURL) == "" {
		return nil
	}
	if !strings.HasPrefix(uploadURL, "/uploads/") {
		return fmt.Errorf("上传地址格式不正确")
	}
	return DeleteUpload(uploadURL, directory)
}

func allowedExtension(extension string) bool {
	return extension == ".jpg" || extension == ".jpeg" || extension == ".png" || extension == ".webp"
}

func detectImageType(content []byte) string {
	if len(content) >= 12 && bytes.Equal(content[:4], []byte("RIFF")) && bytes.Equal(content[8:12], []byte("WEBP")) {
		return "image/webp"
	}
	return http.DetectContentType(content)
}

func matchesExtension(extension, contentType string) bool {
	switch extension {
	case ".jpg", ".jpeg":
		return contentType == "image/jpeg"
	case ".png":
		return contentType == "image/png"
	case ".webp":
		return contentType == "image/webp"
	default:
		return false
	}
}
