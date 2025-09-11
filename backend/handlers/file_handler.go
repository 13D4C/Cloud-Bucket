package handlers

import (
	"archive/zip"
	"database/sql"
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
	"github.com/google/uuid"
)

// --- Structs ---

type FileHandler struct {
	db *sql.DB
}

type ItemInfo struct {
	ID           string    `json:"id"`
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
	ID   string `json:"ID"`
	Size int64  `json:"Size"`
}

type BulkDownloadPayload struct {
	Paths []string `json:"paths" binding:"required"`
}

// --- Constructor & Helper ---

func NewFileHandler(db *sql.DB) *FileHandler {
	return &FileHandler{db: db}
}

func getUsername(c *gin.Context) (string, bool) {
	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return "", false
	}
	return username.(string), true
}

func (h *FileHandler) getUserId(username string) (int, error) {
	var id int
	err := h.db.QueryRow("SELECT USER_ID FROM USERS WHERE USERNAME = ?", username).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// --- Core File Operations ---

func (h *FileHandler) ListFiles(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	relativePath := c.Query("path")
	if relativePath == "" || relativePath == "/" {
		relativePath = "/"
	}

	var items []ItemInfo

	// Get Folders
	folderRows, err := h.db.Query("SELECT FOLDER_ID, FOLDER_NAME, modified_at, PATH FROM FOLDER_LIST WHERE OWNER_ID = ? AND PATH = ? AND STATUS = 'active'", userID, relativePath)
	if err != nil {
		log.Printf("Error fetching folders: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch folders"})
		return
	}
	defer folderRows.Close()

	for folderRows.Next() {
		var item ItemInfo
		var folderID, folderName, path string
		var modified time.Time
		if err := folderRows.Scan(&folderID, &folderName, &modified, &path); err != nil {
			continue
		}
		item.ID = folderID // Add the ID
		item.Name = folderName
		item.Modified = modified
		item.IsDir = true
		item.Path = filepath.ToSlash(filepath.Join(path, folderName))
		items = append(items, item)
	}

	// Get Files
	fileRows, err := h.db.Query("SELECT FILE_ID, FILE_NAME, FILE_SIZE, modified_at, FILE_PATH FROM FILE_LIST WHERE OWNER_ID = ? AND FILE_PATH = ? AND STATUS = 'active'", userID, relativePath)
	if err != nil {
		log.Printf("Error fetching files: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch files"})
		return
	}
	defer fileRows.Close()

	for fileRows.Next() {
		var item ItemInfo
		var fileID int64
		var size int64
		var name, path string
		var modified time.Time
		if err := fileRows.Scan(&fileID, &name, &size, &modified, &path); err != nil {
			continue
		}
		item.ID = fmt.Sprintf("%d", fileID) // Add the ID
		item.Name = name
		item.Size = size
		item.Modified = modified
		item.IsDir = false
		item.Path = filepath.ToSlash(filepath.Join(path, name))
		items = append(items, item)
	}

	c.JSON(http.StatusOK, items)
}

func (h *FileHandler) CreateFolder(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var payload struct {
		FolderName  string `json:"folderName"`
		CurrentPath string `json:"path"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil || payload.FolderName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid folder name or path"})
		return
	}

	parentPath := payload.CurrentPath
	if parentPath == "" {
		parentPath = "/"
	}

	// Physically create the directory
	fullPath, err := utils.GetSafePathForUser(username, filepath.Join(parentPath, payload.FolderName))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create physical directory"})
		return
	}

	_, err = h.db.Exec("INSERT INTO FOLDER_LIST (FOLDER_ID, OWNER_ID, FOLDER_NAME, PATH, STATUS) VALUES (?, ?, ?, ?, 'active')", uuid.New().String(), userID, payload.FolderName, parentPath)
	if err != nil {
		log.Printf("Error creating folder in DB: %v", err)
		os.RemoveAll(fullPath) // Rollback physical creation
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create folder metadata"})
		return
	}

	c.Status(http.StatusCreated)
}

func (h *FileHandler) CreateFolderPath(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var payload struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil || payload.Path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid path provided"})
		return
	}

	// Create physical directories
	fullPhysicalPath, err := utils.GetSafePathForUser(username, payload.Path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := os.MkdirAll(fullPhysicalPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create folder structure"})
		return
	}

	// Create database entries
	cleanedPath := filepath.Clean(payload.Path)
	parts := strings.Split(cleanedPath, string(filepath.Separator))
	currentPath := "/"

	for _, part := range parts {
		if part == "" {
			continue
		}
		var exists int
		err := h.db.QueryRow("SELECT COUNT(*) FROM FOLDER_LIST WHERE OWNER_ID = ? AND FOLDER_NAME = ? AND PATH = ?", userID, part, currentPath).Scan(&exists)
		if err == nil && exists == 0 {
			_, err = h.db.Exec("INSERT INTO FOLDER_LIST (FOLDER_ID, OWNER_ID, FOLDER_NAME, PATH, STATUS) VALUES (?, ?, ?, ?, 'active')", uuid.New().String(), userID, part, currentPath)
			if err != nil {
				log.Printf("Failed to insert path component %s: %v", part, err)
				continue
			}
		}
		currentPath = filepath.ToSlash(filepath.Join(currentPath, part))
	}

	c.Status(http.StatusCreated)
}

func (h *FileHandler) FinalizeUpload(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var payload struct {
		UploadID        string `json:"uploadId"`
		DestinationPath string `json:"destinationPath"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil || payload.UploadID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	baseUploadPath, _ := utils.GetBaseUploadPath()
	sourceFile := filepath.Join(baseUploadPath, payload.UploadID)
	sourceInfo := sourceFile + ".info"
	infoData, err := os.ReadFile(sourceInfo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not read upload metadata"})
		return
	}

	var tusInfo TusInfo
	if err := json.Unmarshal(infoData, &tusInfo); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not parse upload metadata"})
		return
	}

	fileInfo, err := os.Stat(sourceFile)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not get file info"})
		return
	}

	res, err := h.db.Exec("INSERT INTO FILE_LIST (OWNER_ID, FILE_NAME, FILE_TYPE, FILE_SIZE, FILE_PATH, STATUS) VALUES (?, ?, ?, ?, ?, 'active')", userID, tusInfo.MetaData.Filename, tusInfo.MetaData.Filetype, fileInfo.Size(), payload.DestinationPath)
	if err != nil {
		log.Printf("DB Error on finalize: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file metadata"})
		return
	}

	newFileID, _ := res.LastInsertId()
	destinationFolder, err := utils.GetSafePathForUser(username, payload.DestinationPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid destination path"})
		return
	}

	if err := os.MkdirAll(destinationFolder, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create destination directory"})
		return
	}

	newFileLocation := filepath.Join(destinationFolder, fmt.Sprintf("%d", newFileID))
	if err := os.Rename(sourceFile, newFileLocation); err != nil {
		h.db.Exec("DELETE FROM FILE_LIST WHERE FILE_ID = ?", newFileID) // Rollback
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move file"})
		return
	}

	os.Remove(sourceInfo)
	c.JSON(http.StatusOK, gin.H{"message": "File finalized successfully"})
}

func (h *FileHandler) MoveItem(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var payload struct {
		SourcePath        string `json:"sourcePath"`
		DestinationFolder string `json:"destinationFolder"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	sourceFullPath, err := utils.GetSafePathForUser(username, payload.SourcePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid source path"})
		return
	}

	destFullPath, err := utils.GetSafePathForUser(username, payload.DestinationFolder)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid destination path"})
		return
	}

	baseName := filepath.Base(payload.SourcePath)
	parentDir := filepath.ToSlash(filepath.Dir(payload.SourcePath))
	if parentDir == "." {
		parentDir = "/"
	}

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	// Check if it's a file
	var fileID int
	err = tx.QueryRow("SELECT FILE_ID FROM FILE_LIST WHERE OWNER_ID = ? AND FILE_NAME = ? AND FILE_PATH = ?", userID, baseName, parentDir).Scan(&fileID)
	if err == nil { // It's a file
		_, err = tx.Exec("UPDATE FILE_LIST SET FILE_PATH = ? WHERE FILE_ID = ?", payload.DestinationFolder, fileID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update file path in DB"})
			return
		}

		newPhysicalPath := filepath.Join(destFullPath, fmt.Sprintf("%d", fileID))
		if err := os.Rename(sourceFullPath, newPhysicalPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move physical file"})
			return
		}

	} else if err == sql.ErrNoRows { // Check if it's a folder
		var folderID string
		err = tx.QueryRow("SELECT FOLDER_ID FROM FOLDER_LIST WHERE OWNER_ID = ? AND FOLDER_NAME = ? AND PATH = ?", userID, baseName, parentDir).Scan(&folderID)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Source item not found"})
			return
		}

		_, err = tx.Exec("UPDATE FOLDER_LIST SET PATH = ? WHERE FOLDER_ID = ?", payload.DestinationFolder, folderID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update folder path in DB"})
			return
		}

		oldPathPrefix := filepath.ToSlash(filepath.Join(parentDir, baseName))
		newPathPrefix := filepath.ToSlash(filepath.Join(payload.DestinationFolder, baseName))

		if err := h.recursivePathUpdate(tx, userID, oldPathPrefix, newPathPrefix); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update children paths"})
			return
		}

		newPhysicalPath := filepath.Join(destFullPath, baseName)
		if err := os.Rename(sourceFullPath, newPhysicalPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move physical folder"})
			return
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error checking source item"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}
	c.Status(http.StatusOK)
}

func (h *FileHandler) recursivePathUpdate(tx *sql.Tx, userID int, oldPrefix, newPrefix string) error {
	_, err := tx.Exec(`UPDATE FOLDER_LIST SET PATH = CONCAT(?, SUBSTRING(PATH, ?)) WHERE OWNER_ID = ? AND PATH LIKE ?`, newPrefix, len(oldPrefix)+1, userID, oldPrefix+"%")
	if err != nil {
		return err
	}

	_, err = tx.Exec(`UPDATE FILE_LIST SET FILE_PATH = CONCAT(?, SUBSTRING(FILE_PATH, ?)) WHERE OWNER_ID = ? AND FILE_PATH LIKE ?`, newPrefix, len(oldPrefix)+1, userID, oldPrefix+"%")
	return err
}

// --- Trash Bin Operations ---

func (h *FileHandler) updateItemsStatus(userID int, paths []string, newStatus string) error {
	tx, err := h.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, path := range paths {
		baseName := filepath.Base(path)
		dirName := filepath.ToSlash(filepath.Dir(path))
		if dirName == "." {
			dirName = "/"
		}

		_, errFile := tx.Exec("UPDATE FILE_LIST SET STATUS = ? WHERE OWNER_ID = ? AND FILE_NAME = ? AND FILE_PATH = ?", newStatus, userID, baseName, dirName)
		if errFile != nil {
			return errFile
		}

		_, errFolder := tx.Exec("UPDATE FOLDER_LIST SET STATUS = ? WHERE OWNER_ID = ? AND FOLDER_NAME = ? AND PATH = ?", newStatus, userID, baseName, dirName)
		if errFolder != nil {
			return errFolder
		}
	}
	return tx.Commit()
}

func (h *FileHandler) DeleteItem(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	relativePath := strings.TrimPrefix(c.Param("path"), "/")
	if err := h.updateItemsStatus(userID, []string{relativePath}, "trashed"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move item to trash"})
		return
	}
	c.Status(http.StatusOK)
}

func buildInClause(columnName string, ids []string) (string, []interface{}) {
	if len(ids) == 0 {
		return "", nil
	}
	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	query := fmt.Sprintf("%s IN (%s)", columnName, placeholders)
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return query, args
}

// And a version for integer IDs
func buildInClauseInt(columnName string, ids []int) (string, []interface{}) {
	if len(ids) == 0 {
		return "", nil
	}
	placeholders := strings.Repeat("?,", len(ids)-1) + "?"
	query := fmt.Sprintf("%s IN (%s)", columnName, placeholders)
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return query, args
}

func (h *FileHandler) BulkDeleteItems(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Define a new payload structure to accept IDs
	var payload struct {
		FileIDs   []int    `json:"file_ids"`
		FolderIDs []string `json:"folder_ids"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	if len(payload.FileIDs) == 0 && len(payload.FolderIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No item IDs provided"})
		return
	}

	// Use a transaction for atomicity
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("Failed to begin transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback() // Rollback if not committed

	// Update files
	if len(payload.FileIDs) > 0 {
		fileInClause, fileArgs := buildInClauseInt("FILE_ID", payload.FileIDs)
		query := fmt.Sprintf("UPDATE FILE_LIST SET STATUS = 'trashed' WHERE OWNER_ID = ? AND %s", fileInClause)

		// Prepend userID to the arguments slice
		allFileArgs := append([]interface{}{userID}, fileArgs...)

		_, err := tx.Exec(query, allFileArgs...)
		if err != nil {
			log.Printf("Failed to bulk-trash files: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move files to trash"})
			return
		}
	}

	// Update folders
	if len(payload.FolderIDs) > 0 {
		folderInClause, folderArgs := buildInClause("FOLDER_ID", payload.FolderIDs)
		query := fmt.Sprintf("UPDATE FOLDER_LIST SET STATUS = 'trashed' WHERE OWNER_ID = ? AND %s", folderInClause)

		allFolderArgs := append([]interface{}{userID}, folderArgs...)

		_, err := tx.Exec(query, allFolderArgs...)
		if err != nil {
			log.Printf("Failed to bulk-trash folders: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move folders to trash"})
			return
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit transaction: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to finalize operation"})
		return
	}

	c.Status(http.StatusOK)
}

func (h *FileHandler) ListTrashItems(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var items []ItemInfo

	folderRows, err := h.db.Query("SELECT FOLDER_ID, FOLDER_NAME, modified_at, PATH FROM FOLDER_LIST WHERE OWNER_ID = ? AND STATUS = 'trashed'", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch trashed folders"})
		return
	}
	defer folderRows.Close()
	for folderRows.Next() {
		var item ItemInfo
		var folderID, folderName, path string
		var modified time.Time
		if err := folderRows.Scan(&folderID, &folderName, &modified, &path); err != nil {
			continue
		}
		item.ID = folderID // Add the ID
		item.Name = folderName
		item.Modified = modified
		item.IsDir = true
		item.Path = filepath.ToSlash(filepath.Join(path, folderName))
		items = append(items, item)
	}

	fileRows, err := h.db.Query("SELECT FILE_ID, FILE_NAME, FILE_SIZE, modified_at, FILE_PATH FROM FILE_LIST WHERE OWNER_ID = ? AND STATUS = 'trashed'", userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch trashed files"})
		return
	}
	defer fileRows.Close()
	for fileRows.Next() {
		var item ItemInfo
		var fileID, size int64
		var name, path string
		var modified time.Time
		if err := fileRows.Scan(&fileID, &name, &size, &modified, &path); err != nil {
			continue
		}
		item.ID = fmt.Sprintf("%d", fileID) // Add the ID
		item.Name = name
		item.Size = size
		item.Modified = modified
		item.IsDir = false
		item.Path = filepath.ToSlash(filepath.Join(path, name))
		items = append(items, item)
	}

	c.JSON(http.StatusOK, items)
}

func (h *FileHandler) RestoreItem(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var payload struct {
		Path string `json:"path"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.updateItemsStatus(userID, []string{payload.Path}, "active"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore item"})
		return
	}
	c.Status(http.StatusOK)
}

func (h *FileHandler) PermanentDeleteItem(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	relativePath := strings.TrimPrefix(c.Param("path"), "/")
	baseName := filepath.Base(relativePath)
	dirName := filepath.ToSlash(filepath.Dir(relativePath))
	if dirName == "." {
		dirName = "/"
	}

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}
	defer tx.Rollback()

	var fileID int
	var filePath string
	err = tx.QueryRow("SELECT FILE_ID, FILE_PATH FROM FILE_LIST WHERE OWNER_ID = ? AND FILE_NAME = ? AND FILE_PATH = ? AND STATUS = 'trashed'", userID, baseName, dirName).Scan(&fileID, &filePath)
	if err == nil { // It's a file
		_, execErr := tx.Exec("DELETE FROM FILE_LIST WHERE FILE_ID = ?", fileID)
		if execErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "DB delete failed"})
			return
		}
		physicalPath, _ := utils.GetSafePathForUser(username, filepath.Join(filePath, fmt.Sprintf("%d", fileID)))
		os.Remove(physicalPath)
	} else if err == sql.ErrNoRows { // It's a folder
		if err := h.deleteFolderRecursive(tx, userID, username, baseName, dirName); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete folder content"})
			return
		}
	} else {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error finding item to delete"})
		return
	}

	tx.Commit()
	c.Status(http.StatusOK)
}

func (h *FileHandler) deleteFolderRecursive(tx *sql.Tx, userID int, username, folderName, path string) error {
	fullPath := filepath.ToSlash(filepath.Join(path, folderName))

	rows, err := tx.Query("SELECT FILE_ID FROM FILE_LIST WHERE OWNER_ID = ? AND FILE_PATH = ?", userID, fullPath)
	if err != nil {
		return err
	}
	var fileIDs []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err == nil {
			fileIDs = append(fileIDs, id)
		}
	}
	rows.Close()

	for _, id := range fileIDs {
		_, err := tx.Exec("DELETE FROM FILE_LIST WHERE FILE_ID = ?", id)
		if err != nil {
			return err
		}
		physicalPath, _ := utils.GetSafePathForUser(username, filepath.Join(fullPath, fmt.Sprintf("%d", id)))
		os.Remove(physicalPath)
	}

	subRows, err := tx.Query("SELECT FOLDER_NAME FROM FOLDER_LIST WHERE OWNER_ID = ? AND PATH = ?", userID, fullPath)
	if err != nil {
		return err
	}
	var subFolders []string
	for subRows.Next() {
		var name string
		if err := subRows.Scan(&name); err == nil {
			subFolders = append(subFolders, name)
		}
	}
	subRows.Close()

	for _, name := range subFolders {
		if err := h.deleteFolderRecursive(tx, userID, username, name, fullPath); err != nil {
			return err
		}
	}

	_, err = tx.Exec("DELETE FROM FOLDER_LIST WHERE OWNER_ID = ? AND FOLDER_NAME = ? AND PATH = ?", userID, folderName, path)
	if err != nil {
		return err
	}
	physicalFolderPath, _ := utils.GetSafePathForUser(username, fullPath)
	os.RemoveAll(physicalFolderPath)
	return nil
}

// --- Download Operations ---

func (h *FileHandler) DownloadFile(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	relativePath := strings.TrimPrefix(c.Param("path"), "/")
	baseName := filepath.Base(relativePath)
	dirName := filepath.ToSlash(filepath.Dir(relativePath))
	if dirName == "." {
		dirName = "/"
	}

	var fileID int
	var fileName, fileType, filePath string
	err = h.db.QueryRow("SELECT FILE_ID, FILE_NAME, FILE_TYPE, FILE_PATH FROM FILE_LIST WHERE OWNER_ID = ? AND FILE_NAME = ? AND FILE_PATH = ? AND STATUS = 'active'", userID, baseName, dirName).Scan(&fileID, &fileName, &fileType, &filePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found in database"})
		return
	}

	physicalPath, err := utils.GetSafePathForUser(username, filepath.Join(filePath, fmt.Sprintf("%d", fileID)))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file path"})
		return
	}
	if _, err := os.Stat(physicalPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File does not exist on server"})
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	if fileType == "" {
		fileType = "application/octet-stream"
	}
	c.Header("Content-Type", fileType)
	c.File(physicalPath)
}

func (h *FileHandler) DownloadFolder(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	relativePath := strings.TrimPrefix(c.Param("path"), "/")

	zipFileName := fmt.Sprintf("%s.zip", filepath.Base(relativePath))
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))

	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	if err := h.addPathToZipDB(zipWriter, userID, username, relativePath, ""); err != nil {
		log.Printf("[ERROR] DownloadFolder: Error during zipping for %s: %v", relativePath, err)
	}
}

func (h *FileHandler) BulkDownloadItems(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	var payload BulkDownloadPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload: " + err.Error()})
		return
	}
	if len(payload.Paths) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No items selected for download."})
		return
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	zipFileName := fmt.Sprintf("IT-Cloud-Bulk-%s.zip", timestamp)
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))

	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	for _, relPath := range payload.Paths {
		if err := h.addPathToZipDB(zipWriter, userID, username, relPath, ""); err != nil {
			log.Printf("[ERROR] BulkDownload: Failed to add '%s' to zip. Error: %v", relPath, err)
		}
	}
}

func (h *FileHandler) addPathToZipDB(zipWriter *zip.Writer, userID int, username, relativePath, baseInZip string) error {
	baseName := filepath.Base(relativePath)
	dirName := filepath.ToSlash(filepath.Dir(relativePath))
	if dirName == "." {
		dirName = "/"
	}

	var fileID int
	var fileName string
	err := h.db.QueryRow("SELECT FILE_ID, FILE_NAME FROM FILE_LIST WHERE OWNER_ID = ? AND FILE_NAME = ? AND FILE_PATH = ? AND STATUS = 'active'", userID, baseName, dirName).Scan(&fileID, &fileName)
	if err == nil { // It's a file
		physicalPath, _ := utils.GetSafePathForUser(username, filepath.Join(dirName, fmt.Sprintf("%d", fileID)))
		fileToZip, err := os.Open(physicalPath)
		if err != nil {
			return err
		}
		defer fileToZip.Close()

		info, err := fileToZip.Stat()
		if err != nil {
			return err
		}
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Name = filepath.ToSlash(filepath.Join(baseInZip, fileName))
		header.Method = zip.Deflate
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, fileToZip)
		return err

	} else if err == sql.ErrNoRows { // It's a folder
		var modTime time.Time
		err := h.db.QueryRow("SELECT modified_at FROM FOLDER_LIST WHERE OWNER_ID = ? AND FOLDER_NAME = ? AND PATH = ? AND STATUS = 'active'", userID, baseName, dirName).Scan(&modTime)
		if err != nil {
			return fmt.Errorf("item not found: %s", relativePath)
		}

		currentBaseInZip := filepath.ToSlash(filepath.Join(baseInZip, baseName))
		// --- EDITED: Fixed field name from ModTime to Modified ---
		_, err = zipWriter.CreateHeader(&zip.FileHeader{Name: currentBaseInZip + "/", Modified: modTime})
		if err != nil {
			return err
		}

		fullFolderPath := filepath.ToSlash(filepath.Join(dirName, baseName))

		fileRows, err := h.db.Query("SELECT FILE_NAME FROM FILE_LIST WHERE OWNER_ID = ? AND FILE_PATH = ? AND STATUS = 'active'", userID, fullFolderPath)
		if err != nil {
			return err
		}
		defer fileRows.Close()
		for fileRows.Next() {
			var name string
			if err := fileRows.Scan(&name); err == nil {
				h.addPathToZipDB(zipWriter, userID, username, filepath.ToSlash(filepath.Join(fullFolderPath, name)), currentBaseInZip)
			}
		}

		folderRows, err := h.db.Query("SELECT FOLDER_NAME FROM FOLDER_LIST WHERE OWNER_ID = ? AND PATH = ? AND STATUS = 'active'", userID, fullFolderPath)
		if err != nil {
			return err
		}
		defer folderRows.Close()
		for folderRows.Next() {
			var name string
			if err := folderRows.Scan(&name); err == nil {
				h.addPathToZipDB(zipWriter, userID, username, filepath.ToSlash(filepath.Join(fullFolderPath, name)), currentBaseInZip)
			}
		}
		return nil
	}
	return err
}
