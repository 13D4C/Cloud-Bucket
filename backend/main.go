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
	log.Println("--- ✅✅✅ BACKEND SERVER (FILE-BASED TRASH) IS STARTING ✅✅✅ ---")
	if err := godotenv.Load(); err != nil { log.Println("Warning: .env file not found.") }
	baseUploadPath, _ := utils.GetBaseUploadPath(); if err := os.MkdirAll(baseUploadPath, 0755); err != nil { log.Fatalf("Unable to create base upload directory: %v", err) }
	db, err := database.Connect(); if err != nil { log.Fatalf("Failed to connect to database: %v", err) }
	if db != nil { defer db.Close() }
	
	router := gin.Default()
	corsConfig := cors.DefaultConfig(); corsConfig.AllowOrigins = []string{"http://localhost:5173"}; corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "HEAD", "PATCH"}; corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "Tus-Resumable", "Upload-Length", "Upload-Metadata", "Upload-Offset"}; corsConfig.ExposeHeaders = []string{"Location", "Upload-Offset", "Upload-Length"}
	router.Use(cors.New(corsConfig))

	store := filestore.FileStore{Path: baseUploadPath}; composer := tusd.NewStoreComposer(); store.UseIn(composer); tusdHandler, _ := tusd.NewHandler(tusd.Config{ BasePath: "/uploads/", StoreComposer: composer })

	authHandler := handlers.NewAuthHandler(db)
	fileHandler := handlers.NewFileHandler() // ✅ No longer needs db

	router.POST("/register", authHandler.Register)
	router.POST("/login", authHandler.Login)
	router.Any("/uploads/*path", gin.WrapH(http.StripPrefix("/uploads/", tusdHandler)))

	api := router.Group("/api", middleware.AuthMiddleware())
	{
		api.GET("/files", fileHandler.ListFiles)
		api.POST("/folders", fileHandler.CreateFolder)
		api.POST("/move", fileHandler.MoveItem)
		api.POST("/finalize-upload", fileHandler.FinalizeUpload)
		api.DELETE("/items/*path", fileHandler.DeleteItem)
		api.GET("/download/*path", fileHandler.DownloadFile)
		api.GET("/download-folder/*path", fileHandler.DownloadFolder)

        // ✅ Updated Trash Routes
        api.GET("/trash", fileHandler.ListTrashedItems)
        api.POST("/trash/restore", fileHandler.RestoreItem)
        api.DELETE("/trash/*path", fileHandler.PermanentlyDeleteItem)
	}
	
	log.Println("--- ROUTES ARE SET UP. SERVER IS LISTENING ON PORT 8080 ---")
	router.Run(":8080")
}