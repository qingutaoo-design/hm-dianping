package service

import (
	"mime/multipart"
	"os"
	"path/filepath"
)

// FileStore 文件存储接口（替代 var saveMultipartFile hack）
type FileStore interface {
	Save(file *multipart.FileHeader, target string) error
	Remove(path string) error
}

// localFS 默认本地文件系统实现
type localFS struct{}

func (l *localFS) Save(file *multipart.FileHeader, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.Create(target)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = dst.ReadFrom(src)
	return err
}

func (l *localFS) Remove(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// IDGenerator 全局唯一 ID 生成器接口
type IDGenerator interface {
	NextID(prefix string) (uint64, error)
}