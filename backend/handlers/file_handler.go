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

type SharePayload struct {
	ItemID            string `json:"itemId"`
	ItemType          string `json:"itemType"` // "file" or "folder"
	ShareWithUsername string `json:"shareWithUsername"`
	Permission        string `json:"permission"`
}

type UnsharePayload struct {
	ItemID          string `json:"itemId"`
	ItemType        string `json:"itemType"` // "file" or "folder"
	ShareWithUserID int    `json:"shareWithUserId"`
}

type SharedItemInfo struct {
	ItemInfo
	OwnerName  string `json:"ownerName"`
	Permission string `json:"permission"`
}

type SharedByMeInfo struct {
	ItemInfo
	SharedWith []struct {
		UserID     int    `json:"userId"`
		Username   string `json:"username"`
		Permission string `json:"permission"`
	} `json:"sharedWith"`
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

// getUserQuotaInfo returns user's quota limit and used quota
func (h *FileHandler) getUserQuotaInfo(userID int) (quotaLimit int64, quotaUsed int64, err error) {
	err = h.db.QueryRow("SELECT USER_QUOTA, USED_QUOTA FROM USERS WHERE USER_ID = ?", userID).Scan(&quotaLimit, &quotaUsed)
	return quotaLimit, quotaUsed, err
}

// updateUserQuota updates the user's used quota by adding the specified amount
func (h *FileHandler) updateUserQuota(tx *sql.Tx, userID int, sizeChange int64) error {
	_, err := tx.Exec("UPDATE USERS SET USED_QUOTA = USED_QUOTA + ? WHERE USER_ID = ?", sizeChange, userID)
	return err
}

// checkQuotaLimit verifies if the user has enough quota for the new file
func (h *FileHandler) checkQuotaLimit(userID int, fileSize int64) error {
	quotaLimit, quotaUsed, err := h.getUserQuotaInfo(userID)
	if err != nil {
		return fmt.Errorf("failed to get user quota info: %w", err)
	}

	if quotaUsed+fileSize > quotaLimit {
		return fmt.Errorf("quota exceeded: would use %d bytes but limit is %d bytes", quotaUsed+fileSize, quotaLimit)
	}

	return nil
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

	// Use transaction for database operations
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction could not be started"})
		return
	}
	defer tx.Rollback()

	newFolderID := uuid.New().String()
	_, err = tx.Exec("INSERT INTO FOLDER_LIST (FOLDER_ID, OWNER_ID, FOLDER_NAME, PATH, STATUS) VALUES (?, ?, ?, ?, 'active')", newFolderID, userID, payload.FolderName, parentPath)
	if err != nil {
		log.Printf("Error creating folder in DB: %v", err)
		os.RemoveAll(fullPath) // Rollback physical creation
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create folder metadata"})
		return
	}

	// Auto-share the new folder if it's created inside shared folders
	newFolderPath := filepath.ToSlash(filepath.Join(parentPath, payload.FolderName))
	if err := h.autoShareNewItem(tx, userID, newFolderID, "folder", newFolderPath); err != nil {
		log.Printf("Warning: Failed to auto-share new folder: %v", err)
		// Don't fail the entire operation, just log the warning
	}

	if err := tx.Commit(); err != nil {
		os.RemoveAll(fullPath) // Rollback physical creation
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit folder creation"})
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

	// Check quota limit before processing upload
	if err := h.checkQuotaLimit(userID, fileInfo.Size()); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	// Use transaction for database operations
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction could not be started"})
		return
	}
	defer tx.Rollback()

	res, err := tx.Exec("INSERT INTO FILE_LIST (OWNER_ID, FILE_NAME, FILE_TYPE, FILE_SIZE, FILE_PATH, STATUS) VALUES (?, ?, ?, ?, ?, 'active')", userID, tusInfo.MetaData.Filename, tusInfo.MetaData.Filetype, fileInfo.Size(), payload.DestinationPath)
	if err != nil {
		log.Printf("DB Error on finalize: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file metadata"})
		return
	}

	newFileID, _ := res.LastInsertId()

	// Update user's quota usage
	if err := h.updateUserQuota(tx, userID, fileInfo.Size()); err != nil {
		log.Printf("Failed to update user quota: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quota usage"})
		return
	}

	// Auto-share the new file if it's uploaded to shared folders
	newFilePath := filepath.ToSlash(filepath.Join(payload.DestinationPath, tusInfo.MetaData.Filename))
	if err := h.autoShareNewItem(tx, userID, fmt.Sprintf("%d", newFileID), "file", newFilePath); err != nil {
		log.Printf("Warning: Failed to auto-share new file: %v", err)
		// Don't fail the entire operation, just log the warning
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit file upload"})
		return
	}

	// Move physical file after successful database commit
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
		// Rollback database entry and quota if physical move fails
		h.db.Exec("DELETE FROM FILE_LIST WHERE FILE_ID = ?", newFileID)
		h.db.Exec("UPDATE USERS SET USED_QUOTA = USED_QUOTA - ? WHERE USER_ID = ?", fileInfo.Size(), userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move file"})
		return
	}

	os.Remove(sourceInfo)
	c.JSON(http.StatusOK, gin.H{"message": "File finalized successfully"})
}

// GetQuotaInfo returns the current quota information for the authenticated user
func (h *FileHandler) GetQuotaInfo(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	quotaLimit, quotaUsed, err := h.getUserQuotaInfo(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get quota information"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"quotaLimit":     quotaLimit,
		"quotaUsed":      quotaUsed,
		"quotaAvailable": quotaLimit - quotaUsed,
	})
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
	var fileSize int64
	err = tx.QueryRow("SELECT FILE_ID, FILE_PATH, FILE_SIZE FROM FILE_LIST WHERE OWNER_ID = ? AND FILE_NAME = ? AND FILE_PATH = ? AND STATUS = 'trashed'", userID, baseName, dirName).Scan(&fileID, &filePath, &fileSize)
	if err == nil { // It's a file
		// Update quota before deleting file record
		if err := h.updateUserQuota(tx, userID, -fileSize); err != nil {
			log.Printf("Failed to update quota on permanent delete: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quota"})
			return
		}

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

	// Get file info including sizes for quota update
	rows, err := tx.Query("SELECT FILE_ID, FILE_SIZE FROM FILE_LIST WHERE OWNER_ID = ? AND FILE_PATH = ?", userID, fullPath)
	if err != nil {
		return err
	}
	var fileInfo []struct {
		id   int
		size int64
	}
	for rows.Next() {
		var id int
		var size int64
		if err := rows.Scan(&id, &size); err == nil {
			fileInfo = append(fileInfo, struct {
				id   int
				size int64
			}{id, size})
		}
	}
	rows.Close()

	for _, info := range fileInfo {
		// Update quota before deleting file record
		if err := h.updateUserQuota(tx, userID, -info.size); err != nil {
			log.Printf("Failed to update quota on folder delete: %v", err)
			return err
		}

		_, err := tx.Exec("DELETE FROM FILE_LIST WHERE FILE_ID = ?", info.id)
		if err != nil {
			return err
		}
		physicalPath, _ := utils.GetSafePathForUser(username, filepath.Join(fullPath, fmt.Sprintf("%d", info.id)))
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

func (h *FileHandler) ShareItem(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	ownerID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Owner not found"})
		return
	}

	var payload SharePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	var targetUserID int
	err = h.db.QueryRow("SELECT USER_ID FROM USERS WHERE USERNAME = ?", payload.ShareWithUsername).Scan(&targetUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"error": "User to share with not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error looking up user"})
		}
		return
	}

	if ownerID == targetUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "You cannot share an item with yourself"})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction could not be started"})
		return
	}
	defer tx.Rollback()

	if payload.ItemType == "file" {
		// Handle file sharing (unchanged)
		var count int
		err := tx.QueryRow("SELECT COUNT(*) FROM FILE_LIST WHERE FILE_ID = ? AND OWNER_ID = ?", payload.ItemID, ownerID).Scan(&count)
		if err != nil || count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not own this file or it does not exist"})
			return
		}
		_, err = tx.Exec("INSERT INTO SHARED_FILE (USER_ID, FILE_ID, PERMISSION) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE PERMISSION = ?", targetUserID, payload.ItemID, payload.Permission, payload.Permission)
		if err != nil {
			log.Printf("Error sharing file: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to share file"})
			return
		}
	} else if payload.ItemType == "folder" {
		// Handle folder sharing with recursive sharing
		var count int
		err := tx.QueryRow("SELECT COUNT(*) FROM FOLDER_LIST WHERE FOLDER_ID = ? AND OWNER_ID = ?", payload.ItemID, ownerID).Scan(&count)
		if err != nil || count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "You do not own this folder or it does not exist"})
			return
		}

		// Share the main folder
		_, err = tx.Exec("INSERT INTO SHARED_FOLDER (USER_ID, FOLDER_ID, PERMISSION) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE PERMISSION = ?", targetUserID, payload.ItemID, payload.Permission, payload.Permission)
		if err != nil {
			log.Printf("Error sharing folder: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to share folder"})
			return
		}

		// Recursively share all contents
		if err := h.recursiveShareFolderContents(tx, ownerID, payload.ItemID, targetUserID, payload.Permission); err != nil {
			log.Printf("Error recursively sharing folder contents: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to share folder contents"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item type"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit share transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item shared successfully"})
}

// New helper function to recursively share folder contents
func (h *FileHandler) recursiveShareFolderContents(tx *sql.Tx, ownerID int, folderID string, targetUserID int, permission string) error {
	// Get the folder's path information
	var folderName, folderPath string
	err := tx.QueryRow("SELECT FOLDER_NAME, PATH FROM FOLDER_LIST WHERE FOLDER_ID = ? AND OWNER_ID = ?", folderID, ownerID).Scan(&folderName, &folderPath)
	if err != nil {
		return fmt.Errorf("failed to get folder info: %v", err)
	}

	// Construct the full path of this folder
	fullFolderPath := filepath.ToSlash(filepath.Join(folderPath, folderName))

	// Collect all file IDs first, then process them
	fileRows, err := tx.Query("SELECT FILE_ID FROM FILE_LIST WHERE OWNER_ID = ? AND FILE_PATH = ? AND STATUS = 'active'", ownerID, fullFolderPath)
	if err != nil {
		return fmt.Errorf("failed to query files in folder: %v", err)
	}

	var fileIDs []int
	for fileRows.Next() {
		var fileID int
		if err := fileRows.Scan(&fileID); err == nil {
			fileIDs = append(fileIDs, fileID)
		}
	}
	fileRows.Close() // Important: close before using the transaction again

	// Now share all collected files
	for _, fileID := range fileIDs {
		_, err = tx.Exec("INSERT INTO SHARED_FILE (USER_ID, FILE_ID, PERMISSION) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE PERMISSION = ?", targetUserID, fileID, permission, permission)
		if err != nil {
			log.Printf("Warning: Failed to share file %d: %v", fileID, err)
		}
	}

	// Collect all subfolder IDs first, then process them
	folderRows, err := tx.Query("SELECT FOLDER_ID FROM FOLDER_LIST WHERE OWNER_ID = ? AND PATH = ? AND STATUS = 'active'", ownerID, fullFolderPath)
	if err != nil {
		return fmt.Errorf("failed to query subfolders: %v", err)
	}

	var subFolderIDs []string
	for folderRows.Next() {
		var subFolderID string
		if err := folderRows.Scan(&subFolderID); err == nil {
			subFolderIDs = append(subFolderIDs, subFolderID)
		}
	}
	folderRows.Close() // Important: close before using the transaction again

	// Now share all collected subfolders and recursively share their contents
	for _, subFolderID := range subFolderIDs {
		// Share the subfolder
		_, err = tx.Exec("INSERT INTO SHARED_FOLDER (USER_ID, FOLDER_ID, PERMISSION) VALUES (?, ?, ?) ON DUPLICATE KEY UPDATE PERMISSION = ?", targetUserID, subFolderID, permission, permission)
		if err != nil {
			log.Printf("Warning: Failed to share subfolder %s: %v", subFolderID, err)
			continue
		}
		// Recursively share the subfolder's contents
		if err := h.recursiveShareFolderContents(tx, ownerID, subFolderID, targetUserID, permission); err != nil {
			log.Printf("Warning: Failed to recursively share subfolder %s: %v", subFolderID, err)
		}
	}

	return nil
}

func (h *FileHandler) UnshareItem(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	ownerID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Owner not found"})
		return
	}

	var payload UnsharePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction could not be started"})
		return
	}
	defer tx.Rollback()

	if payload.ItemType == "file" {
		// Handle file unsharing (unchanged)
		var count int
		err := tx.QueryRow("SELECT COUNT(*) FROM FILE_LIST WHERE FILE_ID = ? AND OWNER_ID = ?", payload.ItemID, ownerID).Scan(&count)
		if err != nil || count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}
		_, err = tx.Exec("DELETE FROM SHARED_FILE WHERE FILE_ID = ? AND USER_ID = ?", payload.ItemID, payload.ShareWithUserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove file share"})
			return
		}
	} else if payload.ItemType == "folder" {
		// Handle folder unsharing with recursive unsharing
		var count int
		err := tx.QueryRow("SELECT COUNT(*) FROM FOLDER_LIST WHERE FOLDER_ID = ? AND OWNER_ID = ?", payload.ItemID, ownerID).Scan(&count)
		if err != nil || count == 0 {
			c.JSON(http.StatusForbidden, gin.H{"error": "Permission denied"})
			return
		}

		// Unshare the main folder
		_, err = tx.Exec("DELETE FROM SHARED_FOLDER WHERE FOLDER_ID = ? AND USER_ID = ?", payload.ItemID, payload.ShareWithUserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove folder share"})
			return
		}

		// Recursively unshare all contents
		if err := h.recursiveUnshareFolderContents(tx, ownerID, payload.ItemID, payload.ShareWithUserID); err != nil {
			log.Printf("Error recursively unsharing folder contents: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unshare folder contents"})
			return
		}
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item type"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit unshare transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Item unshared successfully"})
}

// New helper function to recursively unshare folder contents
func (h *FileHandler) recursiveUnshareFolderContents(tx *sql.Tx, ownerID int, folderID string, targetUserID int) error {
	// Get the folder's path information
	var folderName, folderPath string
	err := tx.QueryRow("SELECT FOLDER_NAME, PATH FROM FOLDER_LIST WHERE FOLDER_ID = ? AND OWNER_ID = ?", folderID, ownerID).Scan(&folderName, &folderPath)
	if err != nil {
		return fmt.Errorf("failed to get folder info: %v", err)
	}

	// Construct the full path of this folder
	fullFolderPath := filepath.ToSlash(filepath.Join(folderPath, folderName))

	// Collect all file IDs first, then process them
	fileRows, err := tx.Query("SELECT FILE_ID FROM FILE_LIST WHERE OWNER_ID = ? AND FILE_PATH = ? AND STATUS = 'active'", ownerID, fullFolderPath)
	if err != nil {
		return fmt.Errorf("failed to query files in folder: %v", err)
	}

	var fileIDs []int
	for fileRows.Next() {
		var fileID int
		if err := fileRows.Scan(&fileID); err == nil {
			fileIDs = append(fileIDs, fileID)
		}
	}
	fileRows.Close() // Important: close before using the transaction again

	// Now unshare all collected files
	for _, fileID := range fileIDs {
		_, err = tx.Exec("DELETE FROM SHARED_FILE WHERE FILE_ID = ? AND USER_ID = ?", fileID, targetUserID)
		if err != nil {
			log.Printf("Warning: Failed to unshare file %d: %v", fileID, err)
		}
	}

	// Collect all subfolder IDs first, then process them
	folderRows, err := tx.Query("SELECT FOLDER_ID FROM FOLDER_LIST WHERE OWNER_ID = ? AND PATH = ? AND STATUS = 'active'", ownerID, fullFolderPath)
	if err != nil {
		return fmt.Errorf("failed to query subfolders: %v", err)
	}

	var subFolderIDs []string
	for folderRows.Next() {
		var subFolderID string
		if err := folderRows.Scan(&subFolderID); err == nil {
			subFolderIDs = append(subFolderIDs, subFolderID)
		}
	}
	folderRows.Close() // Important: close before using the transaction again

	// Now unshare all collected subfolders and recursively unshare their contents
	for _, subFolderID := range subFolderIDs {
		// Unshare the subfolder
		_, err = tx.Exec("DELETE FROM SHARED_FOLDER WHERE FOLDER_ID = ? AND USER_ID = ?", subFolderID, targetUserID)
		if err != nil {
			log.Printf("Warning: Failed to unshare subfolder %s: %v", subFolderID, err)
			continue
		}
		// Recursively unshare the subfolder's contents
		if err := h.recursiveUnshareFolderContents(tx, ownerID, subFolderID, targetUserID); err != nil {
			log.Printf("Warning: Failed to recursively unshare subfolder %s: %v", subFolderID, err)
		}
	}

	return nil
}

func (h *FileHandler) autoShareNewItem(tx *sql.Tx, ownerID int, itemID string, itemType string, itemPath string) error {
	// Find all parent folders that are shared and inherit their sharing
	// Collect all sharing information first, then process
	folderRows, err := tx.Query(`
		SELECT DISTINCT sf.USER_ID, sf.PERMISSION, fl.FOLDER_ID 
		FROM SHARED_FOLDER sf
		JOIN FOLDER_LIST fl ON sf.FOLDER_ID = fl.FOLDER_ID
		WHERE fl.OWNER_ID = ? AND fl.STATUS = 'active' 
		AND ? LIKE CONCAT(fl.PATH, '/', fl.FOLDER_NAME, '/%')
	`, ownerID, itemPath)

	if err != nil {
		return fmt.Errorf("failed to query parent shared folders: %v", err)
	}

	// Collect all sharing info first
	type shareInfo struct {
		UserID     int
		Permission string
		FolderID   string
	}
	var shareInfos []shareInfo

	for folderRows.Next() {
		var si shareInfo
		if err := folderRows.Scan(&si.UserID, &si.Permission, &si.FolderID); err == nil {
			shareInfos = append(shareInfos, si)
		}
	}
	folderRows.Close() // Important: close before using the transaction again

	// Now process all the sharing
	for _, si := range shareInfos {
		// Share the new item based on its type
		if itemType == "file" {
			_, err = tx.Exec(`
				INSERT INTO SHARED_FILE (USER_ID, FILE_ID, PERMISSION) 
				VALUES (?, ?, ?) 
				ON DUPLICATE KEY UPDATE PERMISSION = ?
			`, si.UserID, itemID, si.Permission, si.Permission)
		} else if itemType == "folder" {
			_, err = tx.Exec(`
				INSERT INTO SHARED_FOLDER (USER_ID, FOLDER_ID, PERMISSION) 
				VALUES (?, ?, ?) 
				ON DUPLICATE KEY UPDATE PERMISSION = ?
			`, si.UserID, itemID, si.Permission, si.Permission)
		}

		if err != nil {
			log.Printf("Warning: Failed to auto-share %s %s with user %d: %v", itemType, itemID, si.UserID, err)
		}
	}

	return nil
}

func (h *FileHandler) ListAllSharedItems(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// --- Items Shared WITH ME ---
	var sharedWithMe []SharedItemInfo
	folderRows, err := h.db.Query(`
		SELECT sf.PERMISSION, fl.FOLDER_ID, fl.FOLDER_NAME, fl.modified_at, fl.PATH, u.USERNAME 
		FROM SHARED_FOLDER sf
		JOIN FOLDER_LIST fl ON sf.FOLDER_ID = fl.FOLDER_ID
		JOIN USERS u ON fl.OWNER_ID = u.USER_ID
		WHERE sf.USER_ID = ? AND fl.STATUS = 'active'
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shared folders"})
		return
	}
	defer folderRows.Close()
	for folderRows.Next() {
		var item SharedItemInfo
		var path string
		folderRows.Scan(&item.Permission, &item.ID, &item.Name, &item.Modified, &path, &item.OwnerName)
		item.IsDir = true
		item.Path = filepath.ToSlash(filepath.Join(path, item.Name))
		sharedWithMe = append(sharedWithMe, item)
	}

	fileRows, err := h.db.Query(`
		SELECT sf.PERMISSION, fl.FILE_ID, fl.FILE_NAME, fl.FILE_SIZE, fl.modified_at, fl.FILE_PATH, u.USERNAME
		FROM SHARED_FILE sf
		JOIN FILE_LIST fl ON sf.FILE_ID = fl.FILE_ID
		JOIN USERS u ON fl.OWNER_ID = u.USER_ID
		WHERE sf.USER_ID = ? AND fl.STATUS = 'active'
	`, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch shared files"})
		return
	}
	defer fileRows.Close()
	for fileRows.Next() {
		var item SharedItemInfo
		var path string
		fileRows.Scan(&item.Permission, &item.ID, &item.Name, &item.Size, &item.Modified, &path, &item.OwnerName)
		item.IsDir = false
		item.Path = filepath.ToSlash(filepath.Join(path, item.Name))
		sharedWithMe = append(sharedWithMe, item)
	}

	// --- Items Shared BY ME ---
	sharedFoldersMap := make(map[string]*SharedByMeInfo)
	sharedFilesMap := make(map[string]*SharedByMeInfo)

	mySharedFolders, _ := h.db.Query(`
		SELECT fl.FOLDER_ID, fl.FOLDER_NAME, fl.modified_at, fl.PATH, u.USER_ID, u.USERNAME, sf.PERMISSION
		FROM FOLDER_LIST fl JOIN SHARED_FOLDER sf ON fl.FOLDER_ID = sf.FOLDER_ID JOIN USERS u ON sf.USER_ID = u.USER_ID
		WHERE fl.OWNER_ID = ? AND fl.STATUS = 'active' ORDER BY fl.FOLDER_NAME, u.USERNAME
	`, userID)
	defer mySharedFolders.Close()
	for mySharedFolders.Next() {
		var fID, fName, path, sUsername, perm string
		var mod time.Time
		var sUserID int
		mySharedFolders.Scan(&fID, &fName, &mod, &path, &sUserID, &sUsername, &perm)
		if _, exists := sharedFoldersMap[fID]; !exists {
			sharedFoldersMap[fID] = &SharedByMeInfo{
				ItemInfo: ItemInfo{ID: fID, Name: fName, Modified: mod, IsDir: true, Path: filepath.ToSlash(filepath.Join(path, fName))},
				SharedWith: []struct {
					UserID     int    `json:"userId"`
					Username   string `json:"username"`
					Permission string `json:"permission"`
				}{},
			}
		}
		sharedFoldersMap[fID].SharedWith = append(sharedFoldersMap[fID].SharedWith, struct {
			UserID     int    `json:"userId"`
			Username   string `json:"username"`
			Permission string `json:"permission"`
		}{sUserID, sUsername, perm})
	}

	mySharedFiles, _ := h.db.Query(`
		SELECT fl.FILE_ID, fl.FILE_NAME, fl.FILE_SIZE, fl.modified_at, fl.FILE_PATH, u.USER_ID, u.USERNAME, sf.PERMISSION
		FROM FILE_LIST fl JOIN SHARED_FILE sf ON fl.FILE_ID = sf.FILE_ID JOIN USERS u ON sf.USER_ID = u.USER_ID
		WHERE fl.OWNER_ID = ? AND fl.STATUS = 'active' ORDER BY fl.FILE_NAME, u.USERNAME
	`, userID)
	defer mySharedFiles.Close()
	for mySharedFiles.Next() {
		var fileID, fileSize int64
		var fName, path, sUsername, perm string
		var mod time.Time
		var sUserID int
		mySharedFiles.Scan(&fileID, &fName, &fileSize, &mod, &path, &sUserID, &sUsername, &perm)
		sFileID := fmt.Sprintf("%d", fileID)
		if _, exists := sharedFilesMap[sFileID]; !exists {
			sharedFilesMap[sFileID] = &SharedByMeInfo{
				ItemInfo: ItemInfo{ID: sFileID, Name: fName, Size: fileSize, Modified: mod, IsDir: false, Path: filepath.ToSlash(filepath.Join(path, fName))},
				SharedWith: []struct {
					UserID     int    `json:"userId"`
					Username   string `json:"username"`
					Permission string `json:"permission"`
				}{},
			}
		}
		sharedFilesMap[sFileID].SharedWith = append(sharedFilesMap[sFileID].SharedWith, struct {
			UserID     int    `json:"userId"`
			Username   string `json:"username"`
			Permission string `json:"permission"`
		}{sUserID, sUsername, perm})
	}

	sharedByMe := []SharedByMeInfo{}
	for _, v := range sharedFoldersMap {
		sharedByMe = append(sharedByMe, *v)
	}
	for _, v := range sharedFilesMap {
		sharedByMe = append(sharedByMe, *v)
	}

	c.JSON(http.StatusOK, gin.H{"sharedWithMe": sharedWithMe, "sharedByMe": sharedByMe})
}

func (h *FileHandler) DownloadSharedFile(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	fileID := c.Param("fileId")

	// Check if user has access to this shared file
	var fileName, fileType, filePath string
	var ownerID int
	err = h.db.QueryRow(`
		SELECT fl.FILE_NAME, fl.FILE_TYPE, fl.FILE_PATH, fl.OWNER_ID 
		FROM FILE_LIST fl 
		JOIN SHARED_FILE sf ON fl.FILE_ID = sf.FILE_ID 
		WHERE sf.USER_ID = ? AND fl.FILE_ID = ? AND fl.STATUS = 'active'
	`, userID, fileID).Scan(&fileName, &fileType, &filePath, &ownerID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shared file not found or access denied"})
		return
	}

	// Get owner's username for file path construction
	var ownerUsername string
	err = h.db.QueryRow("SELECT USERNAME FROM USERS WHERE USER_ID = ?", ownerID).Scan(&ownerUsername)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not determine file owner"})
		return
	}

	physicalPath, err := utils.GetSafePathForUser(ownerUsername, filepath.Join(filePath, fileID))
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

func (h *FileHandler) DownloadSharedFolder(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	folderID := c.Param("folderId")

	// Check if user has access to this shared folder
	var folderName, folderPath string
	var ownerID int
	err = h.db.QueryRow(`
		SELECT fl.FOLDER_NAME, fl.PATH, fl.OWNER_ID 
		FROM FOLDER_LIST fl 
		JOIN SHARED_FOLDER sf ON fl.FOLDER_ID = sf.FOLDER_ID 
		WHERE sf.USER_ID = ? AND fl.FOLDER_ID = ? AND fl.STATUS = 'active'
	`, userID, folderID).Scan(&folderName, &folderPath, &ownerID)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shared folder not found or access denied"})
		return
	}

	// Get owner's username for file path construction
	var ownerUsername string
	err = h.db.QueryRow("SELECT USERNAME FROM USERS WHERE USER_ID = ?", ownerID).Scan(&ownerUsername)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not determine folder owner"})
		return
	}

	zipFileName := fmt.Sprintf("%s.zip", folderName)
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", zipFileName))

	zipWriter := zip.NewWriter(c.Writer)
	defer zipWriter.Close()

	relativePath := filepath.ToSlash(filepath.Join(folderPath, folderName))
	if err := h.addPathToZipDB(zipWriter, ownerID, ownerUsername, relativePath, ""); err != nil {
		log.Printf("[ERROR] DownloadSharedFolder: Error during zipping for %s: %v", relativePath, err)
	}
}

func (h *FileHandler) ListSharedFolderContents(c *gin.Context) {
	username, ok := getUsername(c)
	if !ok {
		return
	}
	userID, err := h.getUserId(username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	folderID := c.Param("folderId")
	relativePath := c.Query("path")
	if relativePath == "" || relativePath == "/" {
		relativePath = "/"
	}

	// Check if user has access to this shared folder and get permission
	var folderName, folderPath, permission string
	var ownerID int
	err = h.db.QueryRow(`
		SELECT fl.FOLDER_NAME, fl.PATH, fl.OWNER_ID, sf.PERMISSION
		FROM FOLDER_LIST fl 
		JOIN SHARED_FOLDER sf ON fl.FOLDER_ID = sf.FOLDER_ID 
		WHERE sf.USER_ID = ? AND fl.FOLDER_ID = ? AND fl.STATUS = 'active'
	`, userID, folderID).Scan(&folderName, &folderPath, &ownerID, &permission)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shared folder not found or access denied"})
		return
	}

	baseFolderPath := filepath.ToSlash(filepath.Join(folderPath, folderName))
	requestedPath := filepath.ToSlash(filepath.Join(baseFolderPath, strings.TrimPrefix(relativePath, "/")))

	var items []ItemInfo

	// Get folders within the shared folder
	folderRows, err := h.db.Query("SELECT FOLDER_ID, FOLDER_NAME, modified_at, PATH FROM FOLDER_LIST WHERE OWNER_ID = ? AND PATH = ? AND STATUS = 'active'", ownerID, requestedPath)
	if err != nil {
		log.Printf("Error fetching shared folder contents: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch folder contents"})
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
		item.ID = folderID
		item.Name = folderName
		item.Modified = modified
		item.IsDir = true
		item.Path = filepath.ToSlash(filepath.Join(path, folderName))
		items = append(items, item)
	}

	// Get files within the shared folder
	fileRows, err := h.db.Query("SELECT FILE_ID, FILE_NAME, FILE_SIZE, modified_at, FILE_PATH FROM FILE_LIST WHERE OWNER_ID = ? AND FILE_PATH = ? AND STATUS = 'active'", ownerID, requestedPath)
	if err != nil {
		log.Printf("Error fetching shared files: %v", err)
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
		item.ID = fmt.Sprintf("%d", fileID)
		item.Name = name
		item.Size = size
		item.Modified = modified
		item.IsDir = false
		item.Path = filepath.ToSlash(filepath.Join(path, name))
		items = append(items, item)
	}

	response := gin.H{
		"items":          items,
		"permission":     permission,
		"folderName":     folderName,
		"sharedFolderId": folderID,
	}

	c.JSON(http.StatusOK, response)
}

func (h *FileHandler) FinalizeSharedFolderUpload(c *gin.Context) {
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
		UploadID       string `json:"uploadId"`
		SharedFolderID string `json:"sharedFolderId"`
		RelativePath   string `json:"relativePath"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil || payload.UploadID == "" || payload.SharedFolderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// Check if user has write permission to this shared folder
	var folderName, folderPath string
	var ownerID int
	var permission string
	err = h.db.QueryRow(`
		SELECT fl.FOLDER_NAME, fl.PATH, fl.OWNER_ID, sf.PERMISSION
		FROM FOLDER_LIST fl 
		JOIN SHARED_FOLDER sf ON fl.FOLDER_ID = sf.FOLDER_ID 
		WHERE sf.USER_ID = ? AND fl.FOLDER_ID = ? AND fl.STATUS = 'active'
	`, userID, payload.SharedFolderID).Scan(&folderName, &folderPath, &ownerID, &permission)

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Shared folder not found or access denied"})
		return
	}

	if permission != "write" {
		c.JSON(http.StatusForbidden, gin.H{"error": "Write permission required"})
		return
	}

	// Get owner's username for file operations
	var ownerUsername string
	err = h.db.QueryRow("SELECT USERNAME FROM USERS WHERE USER_ID = ?", ownerID).Scan(&ownerUsername)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not determine folder owner"})
		return
	}

	// Process upload similar to regular finalize
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

	// Check quota limit for folder owner before processing upload
	if err := h.checkQuotaLimit(ownerID, fileInfo.Size()); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": fmt.Sprintf("Folder owner's %s", err.Error())})
		return
	}

	// Construct destination path within shared folder
	baseFolderPath := filepath.ToSlash(filepath.Join(folderPath, folderName))
	destinationPath := filepath.ToSlash(filepath.Join(baseFolderPath, payload.RelativePath))

	// Use transaction for database operations
	tx, err := h.db.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database transaction could not be started"})
		return
	}
	defer tx.Rollback()

	// Insert file record under owner's account
	res, err := tx.Exec("INSERT INTO FILE_LIST (OWNER_ID, FILE_NAME, FILE_TYPE, FILE_SIZE, FILE_PATH, STATUS) VALUES (?, ?, ?, ?, ?, 'active')",
		ownerID, tusInfo.MetaData.Filename, tusInfo.MetaData.Filetype, fileInfo.Size(), destinationPath)
	if err != nil {
		log.Printf("DB Error on shared folder finalize: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file metadata"})
		return
	}

	newFileID, _ := res.LastInsertId()

	// Update owner's quota usage
	if err := h.updateUserQuota(tx, ownerID, fileInfo.Size()); err != nil {
		log.Printf("Failed to update owner quota: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update quota usage"})
		return
	}

	if err := tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit file upload"})
		return
	}

	// Move physical file after successful database commit
	destinationFolder, err := utils.GetSafePathForUser(ownerUsername, destinationPath)
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
		// Rollback database entry and quota if physical move fails
		h.db.Exec("DELETE FROM FILE_LIST WHERE FILE_ID = ?", newFileID)
		h.db.Exec("UPDATE USERS SET USED_QUOTA = USED_QUOTA - ? WHERE USER_ID = ?", fileInfo.Size(), ownerID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to move file"})
		return
	}

	os.Remove(sourceInfo)
	c.JSON(http.StatusOK, gin.H{"message": "File uploaded to shared folder successfully"})
}
