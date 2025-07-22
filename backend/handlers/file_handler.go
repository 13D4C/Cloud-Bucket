package handlers

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"my-cloud-project/backend/utils"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// FileHandler doesn't need a DB connection for the file-based trash system
type FileHandler struct{}

// Constructor no longer needs DB
func NewFileHandler() *FileHandler {
	return &FileHandler{}
}

// Structs for data models
type DisplayFileInfo struct {
	OriginalName string    `json:"originalName,omitempty"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	Modified     time.Time `json:"modified"`
	IsDir        bool      `json:"isDir"`
	Path         string    `json:"path"`
}
type TusInfo struct {
	MetaData struct {
		Filename string `json:"filename"`
		Filetype string `json:"filetype"`
	} `json:"MetaData"`
}

// Helper to get username from JWT context
func getUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return "", false
	}
	return username.(string), true
}


// --- Trash Feature Functions ---

// DeleteItem (Soft Delete) - Moves a file/folder to the 'del' directory
func (h *FileHandler) DeleteItem(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }

	relativePath := strings.TrimPrefix(c.Param("path"), "/")
	sourcePath, err := utils.GetSafePathForUser(username, relativePath)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source path"}); return }

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) { c.JSON(http.StatusNotFound, gin.H{"error": "Item not found"}); return }

	delPath := filepath.Join(utils.GetUserRootPath(username), "del")
	os.MkdirAll(delPath, 0755)

	destinationPath := filepath.Join(delPath, filepath.Base(sourcePath))

	if err := os.Rename(sourcePath, destinationPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move item to trash"}); return
	}
	os.Rename(sourcePath + ".info", destinationPath + ".info")

	c.Status(http.StatusOK)
}

// ListTrashedItems - Reads files from the 'del' directory and performs auto-cleanup
func (h *FileHandler) ListTrashedItems(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }

	delPath, err := utils.GetSafePathForUser(username, "del")
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not access trash folder"}); return }
	os.MkdirAll(delPath, 0755)

	// Auto-Cleanup Logic
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)
	entries, err := os.ReadDir(delPath)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read trash folder"}); return }

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil { continue }
		if info.ModTime().Before(sevenDaysAgo) {
			itemToDelete := filepath.Join(delPath, entry.Name())
			os.RemoveAll(itemToDelete)
			os.RemoveAll(itemToDelete + ".info")
		}
	}

	// Re-read entries after cleanup
	entries, err = os.ReadDir(delPath)
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not re-read trash folder"}); return }

	var trashedItems []DisplayFileInfo
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".info") { continue }
		
		info, err := entry.Info()
		if err != nil { continue }
		
		trashedItems = append(trashedItems, DisplayFileInfo{
			Name: info.Name(),
			Modified: info.ModTime(),
			IsDir: info.IsDir(),
			Size: info.Size(),
			Path: filepath.ToSlash(filepath.Join("del", info.Name())),
		})
	}
	c.JSON(http.StatusOK, trashedItems)
}

// RestoreItem - Moves a file/folder from 'del' back to the user's root
func (h *FileHandler) RestoreItem(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }

	var payload struct{ Path string `json:"path"`}
	if err := c.ShouldBindJSON(&payload); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item path"}); return }

	sourcePath, err := utils.GetSafePathForUser(username, payload.Path)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source path in trash"}); return }

	destinationPath, err := utils.GetSafePathForUser(username, filepath.Base(payload.Path))
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid destination path"}); return }

	if err := os.Rename(sourcePath, destinationPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore item"}); return
	}
	os.Rename(sourcePath + ".info", destinationPath + ".info")
	
	c.Status(http.StatusOK)
}

// PermanentlyDeleteItem - Actually deletes a file/folder from the 'del' directory
func (h *FileHandler) PermanentlyDeleteItem(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }

	relativePath := strings.TrimPrefix(c.Param("path"), "/")
	itemToDelete, err := utils.GetSafePathForUser(username, relativePath)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"}); return }

	os.RemoveAll(itemToDelete)
	os.RemoveAll(itemToDelete + ".info")

	c.Status(http.StatusOK)
}


// --- Standard File Operations ---

// ListFiles - Hides the 'del' directory from the main view
func (h *FileHandler) ListFiles(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }
	relativePath := c.Query("path"); currentPath, err := utils.GetSafePathForUser(username, relativePath)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	entries, err := os.ReadDir(currentPath); if err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "Directory not found"}); return }
	
	var displayItems []DisplayFileInfo
	for _, entry := range entries {
		if entry.Name() == "del" || entry.Name() == ".trash" {
			continue
		}
		itemRelativePath := filepath.ToSlash(filepath.Join(relativePath, entry.Name())); fullPath := filepath.Join(currentPath, entry.Name())
		info, err := os.Stat(fullPath); if err != nil { continue }
		if info.IsDir() {
			displayItems = append(displayItems, DisplayFileInfo{Name: info.Name(), Modified: info.ModTime(), IsDir: true, Path: itemRelativePath})
		} else if filepath.Ext(entry.Name()) == ".info" {
			dataPath := strings.TrimSuffix(fullPath, ".info"); dataRelativePath := strings.TrimSuffix(itemRelativePath, ".info")
			fileInfo, err := os.Stat(dataPath); if os.IsNotExist(err) { continue }
			infoData, err := os.ReadFile(fullPath); if err != nil { continue }
			var tusInfo TusInfo; if json.Unmarshal(infoData, &tusInfo) != nil { continue }
			displayItems = append(displayItems, DisplayFileInfo{OriginalName: tusInfo.MetaData.Filename, Name: filepath.Base(dataPath), Size: fileInfo.Size(), Modified: fileInfo.ModTime(), IsDir: false, Path: dataRelativePath})
		}
	}
	c.JSON(http.StatusOK, displayItems)
}

func (h *FileHandler) CreateFolder(c *gin.Context) {
    username, ok := getUsername(c); if !ok { return }
    relativePath := c.Query("path"); var payload struct{ FolderName string `json:"folderName"`}
	if err := c.ShouldBindJSON(&payload); err != nil || payload.FolderName == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder name"}); return }
    newFolderPath := filepath.Join(relativePath, payload.FolderName); fullPath, err := utils.GetSafePathForUser(username, newFolderPath)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if err := os.MkdirAll(fullPath, 0755); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"}); return }
	c.Status(http.StatusCreated)
}

func (h *FileHandler) MoveItem(c *gin.Context) {
    username, ok := getUsername(c); if !ok { return }
    var payload struct { SourcePath string `json:"sourcePath"`; DestinationFolder string `json:"destinationFolder"`}
	if err := c.ShouldBindJSON(&payload); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"}); return }
    sourceFullPath, err := utils.GetSafePathForUser(username, payload.SourcePath); if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source path"}); return }
    destinationFullPath, err := utils.GetSafePathForUser(username, payload.DestinationFolder); if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid destination path"}); return }
	destInfo, err := os.Stat(destinationFullPath); if os.IsNotExist(err) || !destInfo.IsDir() { c.JSON(http.StatusBadRequest, gin.H{"error": "Destination is not a valid folder"}); return }
    newFullPath := filepath.Join(destinationFullPath, filepath.Base(payload.SourcePath))
	if err := os.Rename(sourceFullPath, newFullPath); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move item"}); return }
	if _, err := os.Stat(sourceFullPath + ".info"); err == nil { os.Rename(sourceFullPath+".info", newFullPath+".info") }
	c.Status(http.StatusOK)
}

func (h *FileHandler) FinalizeUpload(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }
	var payload struct { UploadID string `json:"uploadId"`; DestinationPath string `json:"destinationPath"` }
	if err := c.ShouldBindJSON(&payload); err != nil || payload.UploadID == "" {
		log.Printf("FinalizeUpload binding error: %v.", err); c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"}); return
	}
	baseUploadPath, _ := utils.GetBaseUploadPath(); sourceFile := filepath.Join(baseUploadPath, payload.UploadID); sourceInfo := sourceFile + ".info"
	destinationFolder, err := utils.GetSafePathForUser(username, payload.DestinationPath)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid destination path"}); return }
	if _, err := os.Stat(destinationFolder); os.IsNotExist(err) { c.JSON(http.StatusBadRequest, gin.H{"error": "Destination folder does not exist"}); return }
	newFileLocation := filepath.Join(destinationFolder, payload.UploadID)
	if err := os.Rename(sourceFile, newFileLocation); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move file"}); return }
	os.Rename(sourceInfo, newFileLocation+".info")
	c.JSON(http.StatusOK, gin.H{"message": "File finalized successfully"})
}

func (h *FileHandler) DownloadFile(c *gin.Context) {
    username, ok := getUsername(c); if !ok { return }
    relativePath := strings.TrimPrefix(c.Param("path"), "/"); itemPath, err := utils.GetSafePathForUser(username, relativePath)
    if err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "File not found"}); return }
	info, err := os.Stat(itemPath); if os.IsNotExist(err) { c.JSON(http.StatusNotFound, gin.H{"error": "File not found"}); return }
    infoData, err := os.ReadFile(itemPath + ".info"); if err != nil { c.FileAttachment(itemPath, info.Name()); return }
    var tusInfo TusInfo; json.Unmarshal(infoData, &tusInfo)
    c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", tusInfo.MetaData.Filename)); c.Header("Content-Type", tusInfo.MetaData.Filetype); c.File(itemPath)
}

func (h *FileHandler) DownloadFolder(c *gin.Context) {
    username, ok := getUsername(c); if !ok { return }
	relativePath := strings.TrimPrefix(c.Param("path"), "/"); folderPath, err := utils.GetSafePathForUser(username, relativePath)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"}); return }
	info, err := os.Stat(folderPath); if os.IsNotExist(err) || !info.IsDir() { c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found"}); return }
	zipFileName := info.Name() + ".zip"; c.Header("Content-Type", "application/zip"); c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))
	zipWriter := zip.NewWriter(c.Writer); defer zipWriter.Close()
	filepath.Walk(folderPath, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }; relPath, _ := filepath.Rel(folderPath, path); if relPath == "." { return nil }
		header, _ := zip.FileInfoHeader(info); header.Name = filepath.ToSlash(relPath)
		if info.IsDir() { header.Name += "/" } else { header.Method = zip.Deflate }
		writer, _ := zipWriter.CreateHeader(header)
		if !info.IsDir() { fileToZip, _ := os.Open(path); defer fileToZip.Close(); io.Copy(writer, fileToZip) }
		return nil
	})
}