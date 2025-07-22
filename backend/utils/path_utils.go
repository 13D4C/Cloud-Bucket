package utils

import (
	"fmt"
	"path/filepath"
	"strings"
)

const baseUploadPath = "./uploads"

func GetSafePathForUser(username string, targetPath string) (string, error) {
	userRoot := filepath.Join(baseUploadPath, username)
	fullPath := filepath.Join(userRoot, targetPath)
	cleanedPath := filepath.Clean(fullPath)
	if !strings.HasPrefix(cleanedPath, filepath.Clean(userRoot)) {
		return "", fmt.Errorf("invalid path: access denied")
	}
	return cleanedPath, nil
}

func GetUserRootPath(username string) string {
	return filepath.Join(baseUploadPath, username)
}

func GetBaseUploadPath() (string, error) {
	return filepath.Abs(baseUploadPath)
}