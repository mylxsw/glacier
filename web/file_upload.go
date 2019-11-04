package web

import (
	"mime/multipart"
	"os"
	"strings"
)

// UploadedFile 上传的文件
type UploadedFile struct {
	Header   *multipart.FileHeader
	SavePath string
}

// Extension get the file's extension.
func (file *UploadedFile) Extension() string {
	segs := strings.Split(file.Header.Filename, ".")
	return segs[len(segs)-1]
}

// Store store the uploaded file on a filesystem disk.
func (file *UploadedFile) Store(path string) error {
	if err := os.Rename(file.SavePath, path); err != nil {
		return err
	}

	file.SavePath = path
	return nil
}

// Delete 删除文件
func (file *UploadedFile) Delete() error {
	return os.Remove(file.SavePath)
}

// Name 获取上传的文件名
func (file *UploadedFile) Name() string {
	return file.Header.Filename
}

// Size 获取文件大小
func (file *UploadedFile) Size() int64 {
	return file.Header.Size
}

// GetTempFilename 获取文件临时保存的地址
func (file *UploadedFile) GetTempFilename() string {
	return file.SavePath
}
