package controllers

import (
	"archive/zip"
	"crane-system/config"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
)

const (
	defaultMaxUploadSize      int64 = 100 * 1024 * 1024
	maxBatchZipFileEntries          = 2000
	maxBatchZipEntryNameBytes       = 512
)

var errUploadTooLarge = errors.New("upload too large")

func maxUploadSize() int64 {
	if config.AppConfig != nil && config.AppConfig.Storage.MaxUploadSize > 0 {
		return config.AppConfig.Storage.MaxUploadSize
	}
	return defaultMaxUploadSize
}

func validateMultipartUploadSize(files []*multipart.FileHeader) error {
	limit := maxUploadSize()
	var total int64
	for _, file := range files {
		if file.Size <= 0 {
			continue
		}
		if file.Size > limit {
			return errUploadTooLarge
		}
		total += file.Size
		if total > limit {
			return errUploadTooLarge
		}
	}
	return nil
}

func copyWithLimit(dst io.Writer, src io.Reader, limit int64) error {
	limited := &io.LimitedReader{R: src, N: limit + 1}
	written, err := io.Copy(dst, limited)
	if err != nil {
		return err
	}
	if written > limit || limited.N == 0 {
		return errUploadTooLarge
	}
	return nil
}

func validateZipArchiveLimits(files []*zip.File) error {
	limit := uint64(maxUploadSize())
	var total uint64
	var fileCount int
	for _, file := range files {
		if file.FileInfo().IsDir() {
			continue
		}
		if len(file.Name) > maxBatchZipEntryNameBytes {
			return fmt.Errorf("zip entry name too long")
		}
		fileCount++
		if fileCount > maxBatchZipFileEntries {
			return fmt.Errorf("zip contains too many files")
		}
		if file.UncompressedSize64 > limit {
			return errUploadTooLarge
		}
		total += file.UncompressedSize64
		if total > limit {
			return errUploadTooLarge
		}
	}
	return nil
}
