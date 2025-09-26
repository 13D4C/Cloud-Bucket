package main

import (
	"log"
	"my-cloud-project/backend/database"
	"my-cloud-project/backend/handlers"
	"my-cloud-project/backend/middleware"
	"my-cloud-project/backend/utils"
	"net/http"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/tus/tusd/v2/pkg/filestore"
	tusd "github.com/tus/tusd/v2/pkg/handler"
)

func main() {
	log.Println("--- ✅✅✅ BACKEND SERVER IS STARTING ✅✅✅ ---")

	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found.")
	}

	baseUploadPath, _ := utils.GetBaseUploadPath()
	if err := os.MkdirAll(baseUploadPath, 0755); err != nil {
		log.Fatalf("Fatal: Unable to create base upload directory: %v", err)
	}

	db, err := database.Connect()
	if err != nil {
		log.Fatalf("Fatal: Failed to connect to database: %v", err)
	}
	defer db.Close()

	router := gin.Default()

	corsConfig := cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Tus-Resumable", "Upload-Length", "Upload-Metadata", "Upload-Offset"},
		ExposeHeaders:    []string{"Location", "Upload-Offset", "Upload-Length"},
		AllowCredentials: true,
	}
	router.Use(cors.New(corsConfig))

	store := filestore.FileStore{Path: baseUploadPath}
	composer := tusd.NewStoreComposer()
	store.UseIn(composer)

	tusdHandler, err := tusd.NewHandler(tusd.Config{
		BasePath:      "/uploads/",
		StoreComposer: composer,
	})
	if err != nil {
		log.Fatalf("Fatal: Unable to create tusd handler: %s", err)
	}

	authHandler := handlers.NewAuthHandler(db)
	fileHandler := handlers.NewFileHandler(db)
	adminHandler := handlers.NewAdminHandler(db)

	router.POST("/auth/register", authHandler.Register)
	router.POST("/login", authHandler.Login)
	router.Any("/uploads/*path", gin.WrapH(http.StripPrefix("/uploads/", tusdHandler)))

	router.Use(func(c *gin.Context) {
		c.Set("db", db)
		c.Next()
	})

	api := router.Group("/api")
	api.Use(middleware.AuthMiddleware())
	{
		// File & Folder Management
		api.GET("/files", fileHandler.ListFiles)
		api.POST("/folders", fileHandler.CreateFolder)
		api.POST("/folders/structure", fileHandler.CreateFolderPath)

		api.POST("/move", fileHandler.MoveItem)
		api.POST("/finalize-upload", fileHandler.FinalizeUpload)
		api.GET("/quota", fileHandler.GetQuotaInfo)
		api.DELETE("/items/*path", fileHandler.DeleteItem)
		api.POST("/items/bulk-delete", fileHandler.BulkDeleteItems)

		// Bulk Download Route - MUST BE PRESENT
		api.POST("/items/bulk-download", fileHandler.BulkDownloadItems)

		// Trash Management
		api.GET("/trash", fileHandler.ListTrashItems)
		api.POST("/trash/restore", fileHandler.RestoreItem)
		api.DELETE("/trash/*path", fileHandler.PermanentDeleteItem)

		// Download Operations
		api.GET("/download/*path", fileHandler.DownloadFile)
		api.GET("/download-folder/*path", fileHandler.DownloadFolder)

		// Admin routes (requires admin role)
		admin := api.Group("/admin")
		admin.Use(func(c *gin.Context) {
			c.Set("db", db)
			c.Next()
		})
		admin.Use(handlers.AdminMiddleware())
		{
			admin.GET("/users", adminHandler.GetAllUsers)
			admin.GET("/stats", adminHandler.GetSystemStats)
			admin.PUT("/users/:id", adminHandler.UpdateUser)
			admin.DELETE("/users/:id", adminHandler.DeleteUser)
			admin.GET("/settings", adminHandler.GetSettings)
			admin.PUT("/settings", adminHandler.UpdateSettings)
		}

		api.POST("/share", fileHandler.ShareItem)
		api.POST("/unshare", fileHandler.UnshareItem)
		api.GET("/share-info", fileHandler.ListAllSharedItems)

		// Shared item access routes
		api.GET("/shared-files/:fileId/download", fileHandler.DownloadSharedFile)
		api.GET("/shared-folders/:folderId/download", fileHandler.DownloadSharedFolder)
		api.GET("/shared-folders/:folderId/contents", fileHandler.ListSharedFolderContents)
		api.POST("/shared-folders/finalize-upload", fileHandler.FinalizeSharedFolderUpload)

	}

	log.Println("--- ROUTES ARE SET UP. SERVER IS LISTENING ON PORT 8080 ---")
	if err := router.Run(":8080"); err != nil {
		log.Fatalf("Fatal: Failed to run server: %v", err)
	}
}
