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

// --- Structs ---

type FileHandler struct{}

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

type BulkDownloadPayload struct {
	Paths []string `json:"paths" binding:"required"`
}

type itemToZip struct {
	FullPath  string
	PathInZip string
}

// --- Constructor & Helper ---

func NewFileHandler() *FileHandler { return &FileHandler{} }

func getUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return "", false
	}
	return username.(string), true
}

// --- Core File Operations ---

func (h *FileHandler) ListFiles(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }
	relativePath := c.Query("path")
	currentPath, err := utils.GetSafePathForUser(username, relativePath)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	entries, err := os.ReadDir(currentPath)
	if err != nil { c.JSON(http.StatusNotFound, gin.H{"error": "Directory not found"}); return }
	var displayItems []DisplayFileInfo
	for _, entry := range entries {
		if entry.Name() == ".trash" { continue }
		itemRelativePath := filepath.ToSlash(filepath.Join(relativePath, entry.Name()))
		fullPath := filepath.Join(currentPath, entry.Name())
		info, err := os.Stat(fullPath); if err != nil { continue }
		if info.IsDir() {
			displayItems = append(displayItems, DisplayFileInfo{ Name: info.Name(), Modified: info.ModTime(), IsDir: true, Path: itemRelativePath })
		} else if filepath.Ext(entry.Name()) == ".info" {
			dataPath := strings.TrimSuffix(fullPath, ".info"); dataRelativePath := strings.TrimSuffix(itemRelativePath, ".info")
			fileInfo, err := os.Stat(dataPath); if os.IsNotExist(err) { continue }
			infoData, err := os.ReadFile(fullPath); if err != nil { continue }
			var tusInfo TusInfo; if json.Unmarshal(infoData, &tusInfo) != nil { continue }
			displayItems = append(displayItems, DisplayFileInfo{ OriginalName: tusInfo.MetaData.Filename, Name: filepath.Base(dataPath), Size: fileInfo.Size(), Modified: fileInfo.ModTime(), IsDir: false, Path: dataRelativePath })
		}
	}
	c.JSON(http.StatusOK, displayItems)
}

func (h *FileHandler) CreateFolder(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }
	relativePath := c.Query("path")
	var payload struct { FolderName string `json:"folderName"` }
	if err := c.ShouldBindJSON(&payload); err != nil || payload.FolderName == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder name"}); return }
	newFolderPath := filepath.Join(relativePath, payload.FolderName)
	fullPath, err := utils.GetSafePathForUser(username, newFolderPath)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if err := os.MkdirAll(fullPath, 0755); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create directory"}); return }
	c.Status(http.StatusCreated)
}

func (h *FileHandler) CreateFolderPath(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }
	var payload struct { Path string `json:"path"` }
	if err := c.ShouldBindJSON(&payload); err != nil || payload.Path == "" { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path provided"}); return }
	fullPath, err := utils.GetSafePathForUser(username, payload.Path)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}); return }
	if err := os.MkdirAll(fullPath, 0755); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create folder structure"}); return }
	c.Status(http.StatusCreated)
}

func (h *FileHandler) FinalizeUpload(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }
	var payload struct { UploadID string `json:"uploadId"`; DestinationPath string `json:"destinationPath"` }
	if err := c.ShouldBindJSON(&payload); err != nil || payload.UploadID == "" { log.Printf("FinalizeUpload binding error: %v", err); c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"}); return }
	if strings.Contains(payload.UploadID, "/") || strings.Contains(payload.UploadID, "..") { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid upload ID"}); return }
	baseUploadPath, _ := utils.GetBaseUploadPath(); sourceFile := filepath.Join(baseUploadPath, payload.UploadID); sourceInfo := sourceFile + ".info"
	destinationFolder, err := utils.GetSafePathForUser(username, payload.DestinationPath)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid destination path"}); return }
	if _, err := os.Stat(destinationFolder); os.IsNotExist(err) { c.JSON(http.StatusBadRequest, gin.H{"error": "Destination folder does not exist"}); return }
	newFileLocation := filepath.Join(destinationFolder, payload.UploadID)
	if err := os.Rename(sourceFile, newFileLocation); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move file"}); return }
	os.Rename(sourceInfo, newFileLocation+".info")
	c.JSON(http.StatusOK, gin.H{"message": "File finalized successfully"})
}

func (h *FileHandler) MoveItem(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }
	var payload struct { SourcePath string `json:"sourcePath"`; DestinationFolder string `json:"destinationFolder"` }
	if err := c.ShouldBindJSON(&payload); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"}); return }
	sourceFullPath, err := utils.GetSafePathForUser(username, payload.SourcePath); if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source path"}); return }
	destinationFullPath, err := utils.GetSafePathForUser(username, payload.DestinationFolder); if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid destination path"}); return }
	destInfo, err := os.Stat(destinationFullPath); if os.IsNotExist(err) || !destInfo.IsDir() { c.JSON(http.StatusBadRequest, gin.H{"error": "Destination is not a valid folder"}); return }
	newFullPath := filepath.Join(destinationFullPath, filepath.Base(payload.SourcePath))
	if err := os.Rename(sourceFullPath, newFullPath); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move item"}); return }
	if _, err := os.Stat(sourceFullPath + ".info"); err == nil { os.Rename(sourceFullPath+".info", newFullPath+".info") }
	c.Status(http.StatusOK)
}

// --- Trash Bin Operations ---

func (h *FileHandler) DeleteItem(c *gin.Context) {
    username, ok := getUsername(c); if !ok { return }
    relativePath := strings.TrimPrefix(c.Param("path"), "/")
    sourcePath, err := utils.GetSafePathForUser(username, relativePath)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source path"}); return }
	trashPath := filepath.Join(utils.GetUserRootPath(username), ".trash")
	if err := os.MkdirAll(trashPath, 0755); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create trash directory"}); return }
	destinationPath := filepath.Join(trashPath, filepath.Base(sourcePath))
	if err := os.Rename(sourcePath, destinationPath); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move item to trash"}); return }
	if _, err := os.Stat(sourcePath + ".info"); err == nil {
		os.Rename(sourcePath + ".info", destinationPath + ".info")
	}
	c.Status(http.StatusOK)
}

func (h *FileHandler) BulkDeleteItems(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }
	var payload struct { Paths []string `json:"paths" binding:"required"` }
	if err := c.ShouldBindJSON(&payload); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"}); return }
	if len(payload.Paths) == 0 { c.JSON(http.StatusBadRequest, gin.H{"error": "No paths provided"}); return }
	trashPath := filepath.Join(utils.GetUserRootPath(username), ".trash")
	if err := os.MkdirAll(trashPath, 0755); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create trash directory"}); return }
	for _, relativePath := range payload.Paths {
		sourcePath, err := utils.GetSafePathForUser(username, relativePath)
		if err != nil { log.Printf("Bulk delete error: Invalid path %s for user %s", relativePath, username); continue }
		destinationPath := filepath.Join(trashPath, filepath.Base(sourcePath))
		if err := os.Rename(sourcePath, destinationPath); err != nil { log.Printf("Bulk delete error: Could not move %s for user %s", sourcePath, username); continue }
		if _, err := os.Stat(sourcePath + ".info"); err == nil {
			os.Rename(sourcePath + ".info", destinationPath + ".info")
		}
	}
	c.Status(http.StatusOK)
}

func (h *FileHandler) ListTrashItems(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }
	trashPath := filepath.Join(utils.GetUserRootPath(username), ".trash")
	entries, err := os.ReadDir(trashPath)
	if os.IsNotExist(err) { c.JSON(http.StatusOK, []DisplayFileInfo{}); return }
	if err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read trash directory"}); return }
	var displayItems []DisplayFileInfo
	for _, entry := range entries {
		itemRelativePath := entry.Name()
		fullPath := filepath.Join(trashPath, entry.Name())
		info, err := os.Stat(fullPath); if err != nil { continue }
		if info.IsDir() {
			displayItems = append(displayItems, DisplayFileInfo{ Name: info.Name(), Modified: info.ModTime(), IsDir: true, Path: itemRelativePath })
		} else if filepath.Ext(entry.Name()) == ".info" {
			dataPath := strings.TrimSuffix(fullPath, ".info"); dataRelativePath := strings.TrimSuffix(itemRelativePath, ".info")
			fileInfo, err := os.Stat(dataPath); if os.IsNotExist(err) { continue }
			infoData, err := os.ReadFile(fullPath); if err != nil { continue }
			var tusInfo TusInfo; if json.Unmarshal(infoData, &tusInfo) != nil { continue }
			displayItems = append(displayItems, DisplayFileInfo{ OriginalName: tusInfo.MetaData.Filename, Name: filepath.Base(dataPath), Size: fileInfo.Size(), Modified: fileInfo.ModTime(), IsDir: false, Path: dataRelativePath })
		}
	}
	c.JSON(http.StatusOK, displayItems)
}

func (h *FileHandler) RestoreItem(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }
	var payload struct { Path string `json:"path"` }
	if err := c.ShouldBindJSON(&payload); err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"}); return }
	trashPath := filepath.Join(utils.GetUserRootPath(username), ".trash"); userRootPath := utils.GetUserRootPath(username)
	sourcePath := filepath.Join(trashPath, payload.Path); destinationPath := filepath.Join(userRootPath, payload.Path)
	if err := os.Rename(sourcePath, destinationPath); err != nil { c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore item"}); return }
	if _, err := os.Stat(sourcePath + ".info"); err == nil {
		os.Rename(sourcePath + ".info", destinationPath + ".info")
	}
	c.Status(http.StatusOK)
}

func (h *FileHandler) PermanentDeleteItem(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }
    relativePath := strings.TrimPrefix(c.Param("path"), "/")
	trashPath := filepath.Join(utils.GetUserRootPath(username), ".trash")
	itemPath := filepath.Join(trashPath, relativePath)
	if !strings.HasPrefix(filepath.Clean(itemPath), filepath.Clean(trashPath)) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path"}); return
	}
	info, err := os.Stat(itemPath)
	if os.IsNotExist(err) { c.JSON(http.StatusNotFound, gin.H{"error": "Item not found in trash"}); return }
	if info.IsDir() { os.RemoveAll(itemPath) } else { os.Remove(itemPath); os.Remove(itemPath + ".info") }
	c.Status(http.StatusOK)
}

// --- Download Operations ---

func (h *FileHandler) DownloadFile(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok { return }
	relativePath := strings.TrimPrefix(c.Param("path"), "/")
	itemPath, err := utils.GetSafePathForUser(username, relativePath)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path format"}); return }
	if _, err := os.Stat(itemPath); os.IsNotExist(err) { c.JSON(http.StatusNotFound, gin.H{"error": "File does not exist on server"}); return }
	
	infoData, err := os.ReadFile(itemPath + ".info")
	var tusInfo TusInfo
	if err == nil && json.Unmarshal(infoData, &tusInfo) == nil && tusInfo.MetaData.Filename != "" {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", tusInfo.MetaData.Filename))
		fileType := tusInfo.MetaData.Filetype
		if fileType == "" { fileType = "application/octet-stream" }
		c.Header("Content-Type", fileType)
		c.File(itemPath)
	} else {
		c.FileAttachment(itemPath, filepath.Base(itemPath))
	}
}

func (h *FileHandler) DownloadFolder(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }
	relativePath := strings.TrimPrefix(c.Param("path"), "/")
	folderPath, err := utils.GetSafePathForUser(username, relativePath)
	if err != nil { c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder path format"}); return }
	
	_, err = os.Stat(folderPath)
	if os.IsNotExist(err) { c.JSON(http.StatusNotFound, gin.H{"error": "Folder not found"}); return }

	// --- EDITED: Changed zip filename format ---
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	zipFileName := fmt.Sprintf("IT-Cloud-%s.zip", timestamp)
	
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))
	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()
	
	if err := addPathToZipSmart(zipWriter, folderPath, ""); err != nil {
		log.Printf("[ERROR] DownloadFolder: Error during zipping process for %s: %v", folderPath, err)
	}
}

func (h *FileHandler) BulkDownloadItems(c *gin.Context) {
	username, ok := getUsername(c); if !ok { return }

	var payload BulkDownloadPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()}); return
	}

	var validItemsToZip []itemToZip
	for _, relPath := range payload.Paths {
		fullPath, err := utils.GetSafePathForUser(username, relPath)
		if err != nil || os.IsNotExist(err) { continue }
		if _, err := os.Stat(fullPath); os.IsNotExist(err) { continue }
		validItemsToZip = append(validItemsToZip, itemToZip{
			FullPath:  fullPath,
			PathInZip: filepath.Base(fullPath),
		})
	}

	if len(validItemsToZip) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "None of the selected items could be found or accessed."}); return
	}

	// --- EDITED: Changed zip filename format ---
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	zipFileName := fmt.Sprintf("IT-Cloud-%s.zip", timestamp)

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))

	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	for _, item := range validItemsToZip {
		if err := addPathToZipSmart(zipWriter, item.FullPath, item.PathInZip); err != nil {
			log.Printf("[ERROR] BulkDownload: Failed to add '%s' to zip. Error: %v", item.FullPath, err)
		}
	}
}


// --- REVISED ZIP HELPER FUNCTIONS ---

func addPathToZipSmart(zipWriter *zip.Writer, fullPath, baseInZip string) error {
	info, err := os.Stat(fullPath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		if baseInZip != "" {
			header, _ := zip.FileInfoHeader(info)
			header.Name = filepath.ToSlash(baseInZip) + "/"
			_, err := zipWriter.CreateHeader(header)
			if err != nil { return err }
		}

		entries, err := os.ReadDir(fullPath)
		if err != nil { return err }

		for _, entry := range entries {
			childFullPath := filepath.Join(fullPath, entry.Name())
			childBaseInZip := filepath.Join(baseInZip, entry.Name())
			if err := addPathToZipSmart(zipWriter, childFullPath, childBaseInZip); err != nil {
				log.Printf("Warning: could not add %s to zip: %v", childFullPath, err)
			}
		}
		return nil
	}

	if filepath.Ext(fullPath) == ".info" {
		return nil
	}
    
	infoFilePath := fullPath + ".info"
	infoData, err := os.ReadFile(infoFilePath)
	if err == nil {
		var tusInfo TusInfo
		if json.Unmarshal(infoData, &tusInfo) == nil && tusInfo.MetaData.Filename != "" {
			dirPath := filepath.Dir(baseInZip)
			baseInZip = filepath.Join(dirPath, tusInfo.MetaData.Filename)
		}
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil { return err }
	
	header.Name = filepath.ToSlash(baseInZip)
	header.Method = zip.Deflate

	writer, err := zipWriter.CreateHeader(header)
	if err != nil { return err }

	fileToZip, err := os.Open(fullPath)
	if err != nil { return err }
	defer fileToZip.Close()

	_, err = io.Copy(writer, fileToZip)
	return err
}