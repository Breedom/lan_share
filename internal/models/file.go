package models

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"
	"time"
)

type FileMeta struct {
	Name      string    `json:"name"`
	Path      string    `json:"path"`
	Size      int64     `json:"size"`
	IsDir     bool      `json:"is_dir"`
	MD5       string    `json:"md5,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	ModTime   time.Time `json:"mod_time"`
}

func NewFileMeta(path string) (*FileMeta, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	meta := &FileMeta{
		Name:      info.Name(),
		Path:      path,
		Size:      info.Size(),
		IsDir:     info.IsDir(),
		CreatedAt: info.ModTime(),
		ModTime:   info.ModTime(),
	}

	if !info.IsDir() {
		md5Hash, err := calculateMD5(path)
		if err == nil {
			meta.MD5 = md5Hash
		}
	}

	return meta, nil
}

func calculateMD5(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
