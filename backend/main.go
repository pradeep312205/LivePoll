package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func main() {
	// Load local environment variables.
	if err := godotenv.Load(".env"); err != nil {
		log.Println("No .env file loaded; checking system environment variables")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mongoURI := os.Getenv("MONGODB_URI")
	redisAddr := os.Getenv("REDIS_ADDR")
	redisURL := os.Getenv("REDIS_URL")

	if mongoURI == "" || (redisAddr == "" && redisURL == "") {
		log.Fatal("MONGODB_URI and either REDIS_URL or REDIS_ADDR must be configured")
	}
	if len(os.Getenv("JWT_SECRET")) < 32 {
		log.Fatal("JWT_SECRET must contain at least 32 characters")
	}

	// Connect to MongoDB.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatalf("MongoDB connection failed: %v", err)
	}
	defer func() {
		if err := mongoClient.Disconnect(context.Background()); err != nil {
			log.Printf("MongoDB disconnect error: %v", err)
		}
	}()

	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("MongoDB ping failed: %v", err)
	}
	log.Println("MongoDB connected successfully")

	// Initialize the users collection.
	dbName := os.Getenv("MONGODB_DATABASE")
	if dbName == "" {
		dbName = "livepoll"
	}

	usersCollection := mongoClient.Database(dbName).Collection("users")
	pollsCollection := mongoClient.Database(dbName).Collection("polls")
	votesCollection := mongoClient.Database(dbName).Collection("votes")
	commentsCollection := mongoClient.Database(dbName).Collection("comments")
	followersCollection := mongoClient.Database(dbName).Collection("poll_followers")

	// Create a unique index to prevent duplicate email addresses.
	indexCtx, indexCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer indexCancel()

	if err := ensureUserIndexes(indexCtx, usersCollection); err != nil {
		log.Fatalf("Failed to create user email index: %v", err)
	}
	if err := ensurePollIndexes(indexCtx, pollsCollection, votesCollection); err != nil {
		log.Fatalf("Failed to create poll indexes: %v", err)
	}
	if err := ensureSocialIndexes(indexCtx, commentsCollection, followersCollection); err != nil {
		log.Fatalf("Failed to create social indexes: %v", err)
	}
	log.Println("User email index ready")

	// Connect to Redis.
	var redisClient *redis.Client
	if redisURL != "" {
		redisOptions, err := redis.ParseURL(redisURL)
		if err != nil {
			log.Fatalf("Invalid REDIS_URL: %v", err)
		}
		redisClient = redis.NewClient(redisOptions)
	} else {
		redisClient = redis.NewClient(&redis.Options{Addr: redisAddr})
	}
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis ping failed: %v", err)
	}
	log.Println("Redis connected successfully")

	// Initialize the Gin router.
	router := gin.Default()
	router.MaxMultipartMemory = 8 << 20
	router.Use(corsMiddleware())
	router.Use(func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1<<20)
		c.Next()
	})

	// Authentication endpoints.
	authHandler := &AuthHandler{
		Users: usersCollection,
	}

	router.POST("/api/auth/register", authHandler.Register)
	router.POST("/api/auth/login", authHandler.Login)
	pollHandler := &PollHandler{Polls: pollsCollection, Votes: votesCollection, Redis: redisClient}
	socialHandler := &SocialHandler{Polls: pollsCollection, Comments: commentsCollection, Followers: followersCollection}
	router.GET("/api/polls/:id", pollHandler.GetPublic)
	router.POST("/api/polls/:id/votes", pollHandler.Vote)
	router.GET("/api/polls/:id/events", pollHandler.Events)
	router.GET("/api/polls/:id/comments", socialHandler.CommentsList)
	router.POST("/api/polls/:id/comments", socialHandler.AddComment)
	router.POST("/api/polls/:id/followers", socialHandler.Follow)

	// Health check: verifies MongoDB and Redis connectivity.
	router.GET("/api/health", func(c *gin.Context) {
		checkCtx, checkCancel := context.WithTimeout(
			c.Request.Context(),
			3*time.Second,
		)
		defer checkCancel()

		if err := mongoClient.Ping(checkCtx, nil); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":   "unhealthy",
				"database": "MongoDB unavailable",
			})
			return
		}

		if err := redisClient.Ping(checkCtx).Err(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "unhealthy",
				"redis":  "Redis unavailable",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":   "ok",
			"database": "connected",
			"redis":    "connected",
		})
	})

	protected := router.Group("/api")
	protected.Use(AuthMiddleware())
	dashboardHandler := &DashboardHandler{Polls: pollsCollection, Votes: votesCollection, Redis: redisClient}
	protected.POST("/polls", pollHandler.Create)
	protected.GET("/polls", dashboardHandler.ListMine)
	protected.GET("/dashboard/summary", dashboardHandler.Summary)
	protected.GET("/analytics", dashboardHandler.Analytics)
	protected.POST("/assistant/chat", dashboardHandler.Chat)
	protected.PATCH("/polls/:id", pollHandler.Edit)
	protected.PATCH("/polls/:id/close", pollHandler.Close)
	protected.DELETE("/polls/:id", pollHandler.Delete)
	protected.GET("/auth/me", authHandler.Me)
	protected.PATCH("/auth/me", authHandler.UpdateProfile)
	protected.GET("/polls/:id/followers", socialHandler.ListFollowers)

	log.Printf("LivePoll API starting on http://localhost:%s", port)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("API server failed: %v", err)
	}
}
