package service

import (
	"mime/multipart"
	"os"
	"path/filepath"
)

func saveUploadedFile(file *multipart.FileHeader, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}
	return saveMultipartFile(file, target)
}

var saveMultipartFile = func(file *multipart.FileHeader, target string) error {
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

func removeFile(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
