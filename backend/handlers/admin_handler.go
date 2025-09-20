package handlers

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// AdminHandler handles admin-related operations
type AdminHandler struct {
	DB *sql.DB
}

// NewAdminHandler creates a new admin handler instance
func NewAdminHandler(db *sql.DB) *AdminHandler {
	return &AdminHandler{DB: db}
}

// UserResponse represents user data in API responses
type UserResponse struct {
	ID         int    `json:"id"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	Phone      string `json:"phone"`
	Role       string `json:"role"`
	QuotaLimit int64  `json:"quotaLimit"`
	QuotaUsed  int64  `json:"quotaUsed"`
}

// SystemStats represents system-wide statistics
type SystemStats struct {
	TotalUsers   int   `json:"totalUsers"`
	ActiveUsers  int   `json:"activeUsers"`
	TotalStorage int64 `json:"totalStorage"`
	UsedStorage  int64 `json:"usedStorage"`
}

// UpdateUserRequest represents the request body for updating a user
type UpdateUserRequest struct {
	Email      string  `json:"email,omitempty"`
	Phone      string  `json:"phone,omitempty"`
	Role       string  `json:"role,omitempty"`
	QuotaLimit *int64  `json:"quotaLimit,omitempty"`
	Password   *string `json:"password,omitempty"`
}

// AdminMiddleware checks if the user has admin privileges
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the username from the auth middleware
		username, exists := c.Get("username")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Here you would check the user's role from the database
		// For now, we'll implement a simple check
		db, exists := c.Get("db")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database connection not found"})
			c.Abort()
			return
		}

		var role string
		err := db.(*sql.DB).QueryRow(
			"SELECT ROLE FROM USERS WHERE USERNAME = ?",
			username,
		).Scan(&role)

		if err != nil || role != "Admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetAllUsers retrieves all users from the database
func (h *AdminHandler) GetAllUsers(c *gin.Context) {
	query := `
		SELECT 
			USER_ID, 
			USERNAME, 
			EMAIL, 
			PHONE, 
			ROLE, 
			USER_QUOTA,
			USED_QUOTA
		FROM USERS;
	`

	rows, err := h.DB.Query(query)
	if err != nil {
		log.Printf("Error fetching users: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	defer rows.Close()

	var users []UserResponse
	for rows.Next() {
		var user UserResponse
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Phone,
			&user.Role,
			&user.QuotaLimit,
			&user.QuotaUsed,
		)
		if err != nil {
			log.Printf("Error scanning user row: %v", err)
			continue
		}

		users = append(users, user)
	}

	c.JSON(http.StatusOK, users)
}

// GetSystemStats retrieves system-wide statistics
func (h *AdminHandler) GetSystemStats(c *gin.Context) {
	var stats SystemStats

	// Get total users
	err := h.DB.QueryRow("SELECT COUNT(*) FROM USERS").Scan(&stats.TotalUsers)
	if err != nil {
		log.Printf("Error getting total users: %v", err)
	}

	// Get storage statistics
	err = h.DB.QueryRow(`
		SELECT 
			COALESCE(SUM(USER_QUOTA), 0) as total_storage,
			COALESCE(SUM(USED_QUOTA), 0) as used_storage
		FROM USERS
	`).Scan(&stats.TotalStorage, &stats.UsedStorage)
	if err != nil {
		log.Printf("Error getting storage stats: %v", err)
	}

	c.JSON(http.StatusOK, stats)
}

// UpdateUser updates user information
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Build dynamic update query
	updates := []string{}
	values := []interface{}{}

	if req.Email != "" {
		updates = append(updates, "EMAIL = ?")
		values = append(values, req.Email)
	}

	if req.Phone != "" {
		updates = append(updates, "PHONE = ?")
		values = append(values, req.Phone)
	}

	if req.Role != "" {
		if req.Role != "Admin" && req.Role != "User" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
			return
		}
		updates = append(updates, "ROLE = ?")
		values = append(values, req.Role)
	}

	if req.QuotaLimit != nil {
		updates = append(updates, "USER_QUOTA = ?")
		values = append(values, *req.QuotaLimit)
	}

	if req.Password != nil && *req.Password != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
			return
		}
		updates = append(updates, "PASSWORD = ?")
		values = append(values, string(hashedPassword))
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No updates provided"})
		return
	}

	// Add user ID to values
	values = append(values, userID)

	// Execute update
	query := "UPDATE USERS SET " + joinStrings(updates, ", ") + " WHERE USER_ID = ?"
	result, err := h.DB.Exec(query, values...)
	if err != nil {
		log.Printf("Error updating user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated successfully"})
}

// DeleteUser deletes a user from the system
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	// Check if user exists and get username for directory deletion
	var username string
	err = h.DB.QueryRow("SELECT USERNAME FROM USERS WHERE USER_ID = ?", userID).Scan(&username)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check user"})
		return
	}

	// Delete user from database
	result, err := h.DB.Exec("DELETE FROM USERS WHERE USER_ID = ?", userID)
	if err != nil {
		log.Printf("Error deleting user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// TODO: Delete user's directory and files
	// userPath := utils.GetUserRootPath(username)
	// os.RemoveAll(userPath)

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// Settings structures
type SystemSettings struct {
	SiteName                 string   `json:"siteName"`
	SiteDescription          string   `json:"siteDescription"`
	MaintenanceMode          bool     `json:"maintenanceMode"`
	AllowRegistration        bool     `json:"allowRegistration"`
	MaxFileSize              int64    `json:"maxFileSize"`
	AllowedFileTypes         []string `json:"allowedFileTypes"`
	RequireEmailVerification bool     `json:"requireEmailVerification"`
	SupportEmail             string   `json:"supportEmail"`
}

type StorageSettings struct {
	DefaultUserQuota        int64 `json:"defaultUserQuota"`
	MaxUserQuota            int64 `json:"maxUserQuota"`
	AutoCleanupEnabled      bool  `json:"autoCleanupEnabled"`
	CleanupDays             int   `json:"cleanupDays"`
	StorageWarningThreshold int   `json:"storageWarningThreshold"`
	CompressionEnabled      bool  `json:"compressionEnabled"`
}

type SecuritySettings struct {
	SessionTimeout        int  `json:"sessionTimeout"`
	PasswordMinLength     int  `json:"passwordMinLength"`
	RequireStrongPassword bool `json:"requireStrongPassword"`
	MaxLoginAttempts      int  `json:"maxLoginAttempts"`
	LockoutDuration       int  `json:"lockoutDuration"`
	TwoFactorEnabled      bool `json:"twoFactorEnabled"`
	AutoBackupEnabled     bool `json:"autoBackupEnabled"`
	BackupRetentionDays   int  `json:"backupRetentionDays"`
}

type AllSettings struct {
	System   SystemSettings   `json:"system"`
	Storage  StorageSettings  `json:"storage"`
	Security SecuritySettings `json:"security"`
}

// GetSettings retrieves all system settings
func (h *AdminHandler) GetSettings(c *gin.Context) {
	// For now, return default settings
	// In a real implementation, you would load these from a database or config file
	settings := AllSettings{
		System: SystemSettings{
			SiteName:                 "IT Cloud Storage",
			SiteDescription:          "Secure file storage and sharing platform",
			MaintenanceMode:          false,
			AllowRegistration:        true,
			MaxFileSize:              100, // MB
			AllowedFileTypes:         []string{"pdf", "doc", "docx", "txt", "jpg", "png", "gif", "zip"},
			RequireEmailVerification: false,
			SupportEmail:             "admin@itcloud.com",
		},
		Storage: StorageSettings{
			DefaultUserQuota:        5000,  // MB
			MaxUserQuota:            50000, // MB
			AutoCleanupEnabled:      true,
			CleanupDays:             30,
			StorageWarningThreshold: 80, // percentage
			CompressionEnabled:      true,
		},
		Security: SecuritySettings{
			SessionTimeout:        24, // hours
			PasswordMinLength:     8,
			RequireStrongPassword: true,
			MaxLoginAttempts:      5,
			LockoutDuration:       15, // minutes
			TwoFactorEnabled:      false,
			AutoBackupEnabled:     true,
			BackupRetentionDays:   30,
		},
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateSettings updates system settings
func (h *AdminHandler) UpdateSettings(c *gin.Context) {
	var settings AllSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Validate settings
	if settings.System.MaxFileSize < 1 || settings.System.MaxFileSize > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Max file size must be between 1 and 1000 MB"})
		return
	}

	if settings.Storage.DefaultUserQuota < 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Default user quota must be at least 100 MB"})
		return
	}

	if settings.Storage.MaxUserQuota < settings.Storage.DefaultUserQuota {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Max user quota must be greater than default quota"})
		return
	}

	if settings.Security.PasswordMinLength < 6 || settings.Security.PasswordMinLength > 32 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Password minimum length must be between 6 and 32 characters"})
		return
	}

	// In a real implementation, you would save these settings to a database or config file
	// For now, we'll just return success
	log.Printf("Settings updated: %+v", settings)

	c.JSON(http.StatusOK, gin.H{"message": "Settings updated successfully"})
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}
