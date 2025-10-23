package liboc

import (
	"os"
	"path/filepath"
)

var serviceErrorPath string

func init() {
	if sWorkingPath != "" {
		serviceErrorPath = filepath.Join(sWorkingPath, "service_error")
	}
}

func ClearServiceError() {
	if serviceErrorPath == "" {
		return
	}
	os.Remove(serviceErrorPath)
}

func ReadServiceError() (*StringBox, error) {
	if serviceErrorPath == "" {
		return nil, os.ErrNotExist
	}
	content, err := os.ReadFile(serviceErrorPath)
	if err != nil {
		return nil, err
	}
	os.Remove(serviceErrorPath)
	return &StringBox{Value: string(content)}, nil
}

func WriteServiceError(message string) error {
	if serviceErrorPath == "" {
		return os.ErrInvalid
	}
	return os.WriteFile(serviceErrorPath, []byte(message), 0644)
}