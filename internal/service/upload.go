package service

import (
	"errors"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"

	"hm-dianping/internal/config"
)

type UploadService struct {
	cfg *config.Config
}

func NewUploadService(cfg *config.Config) *UploadService {
	return &UploadService{cfg: cfg}
}

func (s *UploadService) SaveBlogImage(file *multipart.FileHeader) (string, error) {
	if file == nil || file.Filename == "" {
		return "", errors.New("文件不能为空")
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext == "" {
		return "", errors.New("文件后缀不能为空")
	}
	name := uuid.NewString() + ext
	dir1 := name[0:1]
	dir2 := name[1:2]
	relative := filepath.Join("blogs", dir1, dir2, name)
	publicPath := strings.ReplaceAll(filepath.ToSlash(filepath.Join(s.cfg.Upload.PublicPrefix, relative)), "//", "/")
	fullPath := filepath.Join(s.cfg.Upload.ImageDir, relative)
	return publicPath, saveUploadedFile(file, fullPath)
}

func (s *UploadService) DeleteBlogImage(name string) error {
	relative := strings.TrimPrefix(name, s.cfg.Upload.PublicPrefix)
	relative = strings.TrimLeft(relative, `/\`)
	clean := filepath.Clean(relative)
	if clean == "." || filepath.IsAbs(clean) || strings.Contains(clean, "..") {
		return errors.New("非法文件路径")
	}
	return removeFile(filepath.Join(s.cfg.Upload.ImageDir, clean))
}
