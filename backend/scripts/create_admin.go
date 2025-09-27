// scripts/create_admin.go
// Run this script to create an admin user: go run scripts/create_admin.go

package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"syscall"

	_ "github.com/go-sql-driver/mysql" // or your database driver
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/ssh/terminal"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	// Connect to database
	dbConnection := os.Getenv("DB_CONNECTION")
	if dbConnection == "" {
		// Default connection string - adjust as needed
		dbConnection = "root:password@tcp(localhost:3306)/clouddb"
	}

	db, err := sql.Open("mysql", dbConnection)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=== Create Admin User ===")
	fmt.Println()

	// Get username
	fmt.Print("Enter username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	// Check if user exists
	var existingUser string
	err = db.QueryRow("SELECT USERNAME FROM USERS WHERE USERNAME = ?", username).Scan(&existingUser)
	if err != sql.ErrNoRows {
		fmt.Printf("User '%s' already exists. Do you want to promote them to admin? (y/n): ", username)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "y" || response == "yes" {
			_, err = db.Exec("UPDATE USERS SET ROLE = 'Admin' WHERE USERNAME = ?", username)
			if err != nil {
				log.Fatal("Failed to update user role:", err)
			}
			fmt.Println("✅ User promoted to admin successfully!")
			return
		}
		fmt.Println("Operation cancelled.")
		return
	}

	// Get email
	fmt.Print("Enter email: ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSpace(email)

	// Get phone
	fmt.Print("Enter phone: ")
	phone, _ := reader.ReadString('\n')
	phone = strings.TrimSpace(phone)

	// Get password (hidden input)
	fmt.Print("Enter password: ")
	passwordBytes, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal("Failed to read password:", err)
	}
	password := string(passwordBytes)
	fmt.Println()

	// Confirm password
	fmt.Print("Confirm password: ")
	confirmBytes, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatal("Failed to read password confirmation:", err)
	}
	confirmPassword := string(confirmBytes)
	fmt.Println()

	if password != confirmPassword {
		log.Fatal("Passwords do not match!")
	}

	if len(password) < 6 {
		log.Fatal("Password must be at least 6 characters long!")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal("Failed to hash password:", err)
	}

	// Create admin user
	_, err = db.Exec(`
		INSERT INTO USERS (
			USERNAME, 
			PASSWORD,
			EMAIL, 
			PHONE, 
			ROLE, 
			USER_QUOTA, 
			USED_QUOTA, 
		) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		username,
		string(hashedPassword),
		email,
		phone,
		"Admin",
		10737418240, // 10GB for admin
		0,
	)

	if err != nil {
		log.Fatal("Failed to create admin user:", err)
	}

	// Create user directory
	userDir := fmt.Sprintf("./uploads/users/%s", username)
	if err := os.MkdirAll(userDir, 0755); err != nil {
		log.Printf("Warning: Could not create user directory: %v", err)
	}

	fmt.Println()
	fmt.Println("✅ Admin user created successfully!")
	fmt.Printf("   Username: %s\n", username)
	fmt.Printf("   Email: %s\n", email)
	fmt.Printf("   Role: Admin\n")
	fmt.Printf("   Storage Quota: 10 GB\n")
	fmt.Println()
	fmt.Println("You can now login at http://localhost:5173 with your credentials.")
	fmt.Println("Access the admin panel at http://localhost:5173/admin")
}
