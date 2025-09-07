package handlers

import (
	"database/sql"
	"log"
	"my-cloud-project/backend/utils"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Struct สำหรับรับข้อมูลตอน Login
type LoginPayload struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Struct สำหรับรับข้อมูลตอน Register
type RegisterPayload struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Phone    string `json:"phone" binding:"required"`
}

// Struct สำหรับเก็บข้อมูลหลักของผู้ใช้หลังจากดึงจาก DB
type UserPrincipal struct {
	ID       int
	Username string
}

type AuthHandler struct {
	DB *sql.DB
}

func NewAuthHandler(db *sql.DB) *AuthHandler {
	return &AuthHandler{DB: db}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var payload RegisterPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input: " + err.Error()})
		return
	}

	var existingUsername string
	err := h.DB.QueryRow("SELECT USERNAME FROM USERS WHERE USERNAME = ?", payload.Username).Scan(&existingUsername)
	if err != sql.ErrNoRows {
		c.JSON(http.StatusConflict, gin.H{"error": "Username already taken"})
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	_, err = h.DB.Exec("INSERT INTO USERS (USERNAME, PASSWORD, EMAIL, PHONE, ROLE) VALUES (?, ?, ?, ?, ?)",
		payload.Username,
		string(hashedPassword),
		payload.Email,
		payload.Phone,
		"User",
	)
	if err != nil {
		log.Printf("Error creating user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
		return
	}

	userFolderPath := utils.GetUserRootPath(payload.Username)
	if err := os.MkdirAll(userFolderPath, 0755); err != nil {
		log.Printf("Warning: could not create directory for user %s: %v", payload.Username, err)
	}

	c.JSON(http.StatusOK, gin.H{"message": "Registration successful! You can now log in."})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var payload LoginPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	var storedPassword string
	err := h.DB.QueryRow("SELECT PASSWORD FROM USERS WHERE USERNAME = ?", payload.Username).Scan(&storedPassword)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}
	if err = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(payload.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	var user UserPrincipal
	var userRole string
	// แก้ไข SQL Query ให้ตรงกับชื่อคอลัมน์ในตารางของคุณ
	err = h.DB.QueryRow("SELECT USER_ID, USERNAME, ROLE FROM USERS WHERE USERNAME = ?", payload.Username).Scan(&user.ID, &user.Username, &userRole)
	if err != nil {
		// Log Error ที่แท้จริงออกมาใน Terminal ของ Backend
		log.Printf("Failed to retrieve user details for %s: %v", payload.Username, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not retrieve user details"})
		return
	}

	userFolderPath := utils.GetUserRootPath(user.Username)
	if _, err := os.Stat(userFolderPath); os.IsNotExist(err) {
		log.Printf("User folder for '%s' not found, creating it now.", user.Username)
		os.MkdirAll(userFolderPath, 0755)
	}

	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Username: user.Username,
		Role:     userRole,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.Username,
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	jwtKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}
