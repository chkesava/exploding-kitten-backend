package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

var db *sql.DB

// User struct to hold leaderboard data
type User struct {
	Username string `json:"username"`
	Points   int    `json:"points"`
}

// Initialize the database connection and create schema if not exists
func initDB() {
	var err error
	dsn := os.Getenv("DATABASE_URL")
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	// Create leaderboard table if it doesn't exist
	createTableQuery := `
    CREATE TABLE IF NOT EXISTS leaderboard (
        username VARCHAR(50) PRIMARY KEY,
        points INT DEFAULT 0
    );`

	_, err = db.Exec(createTableQuery)
	if err != nil {
		log.Fatalf("Error creating table: %q", err)
	}
}

func main() {
	// Load environment variables
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	// Initialize DB
	initDB()

	// Set up Gin
	router := gin.Default()

	// Configure CORS middleware
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://your-react-app.com"}, // Allow your frontend origins
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Routes
	router.POST("/api/leaderboard", addScore)              // Route to add/update score
	router.GET("/api/leaderboard", getLeaderboard)         // Route to get the leaderboard
	router.GET("/api/leaderboard/:username", getUserScore) // Route to get the score of a specific user

	// Run the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default to port 8080 if not set
	}
	router.Run(fmt.Sprintf(":%s", port))
}

// Add a score to the leaderboard
func addScore(c *gin.Context) {
	var user User
	if err := c.BindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Insert or update the score for the user
	_, err := db.Exec("INSERT INTO leaderboard (username, points) VALUES ($1, $2) ON CONFLICT (username) DO UPDATE SET points = leaderboard.points + 1", user.Username, 1)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Score updated successfully"})
}

// Retrieve the full leaderboard
func getLeaderboard(c *gin.Context) {
	rows, err := db.Query("SELECT username, points FROM leaderboard ORDER BY points DESC")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var leaderboard []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.Username, &user.Points); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		leaderboard = append(leaderboard, user)
	}

	c.JSON(http.StatusOK, leaderboard)
}

// Retrieve the score of a specific user by username
func getUserScore(c *gin.Context) {
	username := c.Param("username")
	log.Printf("Fetching score for user: %s", username) // Log the request

	var user User
	err := db.QueryRow("SELECT username, points FROM leaderboard WHERE username = $1", username).Scan(&user.Username, &user.Points)
	if err != nil {
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"message": "User not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		}
		return
	}

	c.JSON(http.StatusOK, user)
}
