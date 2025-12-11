package main

import (
    "crypto/sha512"
    "encoding/base64"
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"

    "github.com/gin-gonic/gin"
    "github.com/gorilla/websocket"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"
)

const DB_NAME = "../ascii.db"

var db *gorm.DB

// -------------------- Models --------------------

type User struct {
    UserID   uint   `gorm:"primaryKey;autoIncrement" json:"user_id"`
    Name     string `gorm:"not null;uniqueIndex" json:"name"`
    Password string `gorm:"not null" json:"password"`
    Arts     []Art  `gorm:"foreignKey:Author" json:"-"`
}

type Art struct {
    ArtID    uint   `gorm:"primaryKey;autoIncrement" json:"art_id"`
    ArtName  string `gorm:"not null" json:"artName"`
    Author   uint   `gorm:"not null" json:"author"`
    JSONData string `gorm:"not null" json:"json_data"`
}

type ASCIIart struct {
    Author  string   `json:"author"`
    ArtName string   `json:"artName"`
    Width   int      `json:"width"`
    Height  int      `json:"height"`
    Letters []string `json:"letters"`
    Colors  []int32  `json:"colors"`
}

type ASCIIartRequest struct {
    ID            int     `json:"id"`
    Count         int     `json:"count"`
    AuthorFilter  *string `json:"author,omitempty"`
    ArtNameFilter *string `json:"artname,omitempty"`
}

type ASCIIartResponse struct {
    ID  int        `json:"id"`
    Msg []ASCIIart `json:"msg"`
}

type ImageCountResponse struct {
    Count int64 `json:"count"`
}

// -------------------- Database Setup --------------------

func initDB() {
    var err error
    db, err = gorm.Open(sqlite.Open(DB_NAME), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })
    if err != nil {
        log.Fatalf("[FATAL] Failed to connect to database: %v", err)
    }

    if err := db.AutoMigrate(&User{}, &Art{}); err != nil {
        log.Fatalf("[FATAL] Failed to migrate database: %v", err)
    }

    log.Println("[DB] Database initialized successfully")
}

// -------------------- User Functions --------------------

func addUser(u User) error {
    log.Printf("[DB] Adding user '%s'", u.Name)
    if err := db.Create(&u).Error; err != nil {
        log.Printf("[ERROR] Failed to insert user '%s': %v", u.Name, err)
        return err
    }
    log.Printf("[DB] User '%s' added successfully", u.Name)
    return nil
}

func getUserID(u User) (uint, error) {
    log.Printf("[USER] Finding user: '%s'", u.Name)
    var user User
    if err := db.Model(&User{}).Where("name = ?", u.Name).First(&user).Error; err != nil {
        log.Printf("[USER] User '%s' not found: %v", u.Name, err)
        return 0, err
    }
    log.Printf("[USER] Found user '%s' with ID %d", u.Name, user.UserID)
    return user.UserID, nil
}

func userExists(u User) bool {
    log.Printf("[DB] Checking if user '%s' exists", u.Name)
    var count int64
    db.Model(&User{}).Where("name = ?", u.Name).Count(&count)
    exists := count > 0
    log.Printf("[DB] User '%s' exists: %v", u.Name, exists)
    return exists
}

func correctPassword(u User) bool {
    log.Printf("[AUTH] Verifying password for user '%s'", u.Name)
    var user User
    if err := db.Model(&User{}).Where("name = ?", u.Name).First(&user).Error; err != nil {
        log.Printf("[AUTH] User '%s' not found", u.Name)
        return false
    }
    match := user.Password == u.Password
    if match {
        log.Printf("[AUTH] Password match for user '%s'", u.Name)
    } else {
        log.Printf("[AUTH] Password mismatch for user '%s'", u.Name)
    }
    return match
}

// -------------------- ASCII Art Functions --------------------

func getTotalImageCount() int64 {
    var count int64
    db.Model(&Art{}).Count(&count)
    return count
}

func addImage(a ASCIIart, filename string) error {
    log.Printf("[IMAGE] Adding image '%s' for author '%s'", a.ArtName, a.Author)
    var user User
    if err := db.Where("name = ?", a.Author).First(&user).Error; err != nil {
        log.Printf("[ERROR] Failed to find user '%s': %v", a.Author, err)
        return err
    }

    art := Art{
        ArtName:  a.ArtName,
        Author:   user.UserID,
        JSONData: filename,
    }

    if err := db.Create(&art).Error; err != nil {
        log.Printf("[ERROR] Failed to insert into Arts: %v", err)
        return err
    }

    log.Printf("[DB] Image '%s' added successfully", a.ArtName)
    return nil
}

func getAsciiArtFilePaths(req ASCIIartRequest, sentPaths *map[string]bool) []string {
    query := db.Table("arts").
        Select("arts.json_data").
        Joins("INNER JOIN users ON arts.author = users.user_id")

    if req.AuthorFilter != nil && *req.AuthorFilter != "" {
        query = query.Where("users.name LIKE ?", "%"+*req.AuthorFilter+"%")
    }

    if req.ArtNameFilter != nil && *req.ArtNameFilter != "" {
        if req.AuthorFilter != nil && *req.AuthorFilter != "" {
            query = query.Or("arts.art_name LIKE ?", "%"+*req.ArtNameFilter+"%")
        } else {
            query = query.Where("arts.art_name LIKE ?", "%"+*req.ArtNameFilter+"%")
        }
    }

    query = query.Order("RANDOM()")

    var allPaths []string
    if err := query.Pluck("json_data", &allPaths).Error; err != nil {
        log.Printf("[DB] Query error: %v", err)
        return []string{}
    }

    var availablePaths []string
    for _, path := range allPaths {
        if !(*sentPaths)[path] {
            availablePaths = append(availablePaths, path)
        }
    }

    count := req.Count
    if count > len(availablePaths) {
        count = len(availablePaths)
    }

    result := availablePaths[:count]
    for _, path := range result {
        (*sentPaths)[path] = true
    }

    log.Printf("[DB] Found %d available paths, sending %d", len(availablePaths), len(result))
    return result
}

// -------------------- WebSocket Handler --------------------

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool { return true },
}

func handleArtStream(c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Printf("[WS] Upgrade error: %v", err)
        return
    }
    defer conn.Close()
    log.Println("[WS] Client connected to /art/stream")

    sentPaths := make(map[string]bool)
    var prevArtFilter *string
    var prevAuthorFilter *string

    for {
        _, msg, err := conn.ReadMessage()
        if err != nil {
            log.Printf("[WS] Read error: %v", err)
            break
        }
        log.Printf("[WS] Parsing request: %v", string(msg))

        var req ASCIIartRequest
        if err := json.Unmarshal(msg, &req); err != nil {
            log.Printf("[WS] JSON unmarshal error: %v", err)
            break
        }

        if req.ArtNameFilter != nil && *req.ArtNameFilter == "" {
            req.ArtNameFilter = nil
        }
        if req.AuthorFilter != nil && *req.AuthorFilter == "" {
            req.AuthorFilter = nil
        }

        filtersChanged := !sameStringPtr(prevArtFilter, req.ArtNameFilter) ||
            !sameStringPtr(prevAuthorFilter, req.AuthorFilter)

        if filtersChanged {
            log.Println("[WS] Filters changed. Resetting sentPaths.")
            sentPaths = make(map[string]bool)
        }

        prevArtFilter = req.ArtNameFilter
        prevAuthorFilter = req.AuthorFilter

        paths := getAsciiArtFilePaths(req, &sentPaths)

        arts := make([]ASCIIart, 0)
        for _, path := range paths {
            data, err := readFile(path)
            if err != nil {
                log.Printf("[WS] Read file error for %s: %v", path, err)
                continue
            }

            var tmp ASCIIart
            if err := json.Unmarshal(data, &tmp); err != nil {
                log.Printf("[WS] JSON unmarshal error for file %s: %v", path, err)
                continue
            }
            arts = append(arts, tmp)
        }

        res := ASCIIartResponse{
            ID:  req.ID,
            Msg: arts,
        }

        log.Printf("[WS] Sending response with %v items (%v total sent)", len(res.Msg), len(sentPaths))
        if err := conn.WriteJSON(res); err != nil {
            log.Printf("[WS] Write error: %v", err)
            break
        }
    }
    log.Println("[WS] Client disconnected from /art/stream")
}

func sameStringPtr(a, b *string) bool {
    if a == nil && b == nil {
        return true
    }
    if a == nil || b == nil {
        return false
    }
    return *a == *b
}

func readFile(path string) ([]byte, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        log.Printf("[FILE] Error reading file %s: %v", path, err)
        return nil, err
    }
    return data, nil
}

// -------------------- HTTP Handlers --------------------

func healthCheck(c *gin.Context) {
    c.String(http.StatusOK, "Working")
}

func debugLog(c *gin.Context) {
    bytes, err := c.GetRawData()
    if err != nil {
        log.Printf("[ERROR] Reading body in /debug: %v", err)
        c.String(http.StatusBadRequest, "Failed to read body")
        return
    }
    log.Println("[DEBUG]", string(bytes))
    c.String(http.StatusOK, "Debug logged")
}

func authenticateUser(c *gin.Context) {
    var u User
    if err := c.ShouldBindJSON(&u); err != nil {
        log.Printf("[ERROR] Bind JSON in /auth/login: %v", err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Bad JSON"})
        return
    }

    if correctPassword(u) {
        log.Printf("[LOGIN] User '%s' authenticated successfully", u.Name)
        c.Status(http.StatusOK)
    } else {
        log.Printf("[LOGIN] Bad password attempt for user '%s'", u.Name)
        c.JSON(http.StatusUnauthorized, gin.H{"error": "Bad name or password"})
    }
}

func registerUser(c *gin.Context) {
    var u User
    if err := c.ShouldBindJSON(&u); err != nil {
        log.Printf("[ERROR] Bind JSON in /auth/register: %v", err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Bad JSON"})
        return
    }

    if userExists(u) {
        log.Printf("[SIGNUP] User '%s' already exists", u.Name)
        c.JSON(http.StatusConflict, gin.H{"error": "User exists"})
    } else {
        log.Printf("[SIGNUP] Creating user '%s'", u.Name)
        if err := addUser(u); err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user"})
            return
        }
        c.Status(http.StatusOK)
    }
}

func uploadArt(c *gin.Context) {
    var req ASCIIart
    bytes, err := c.GetRawData()
    if err != nil {
        log.Printf("[ERROR] Reading body in /art/upload: %v", err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Bad request"})
        return
    }

    if err := json.Unmarshal(bytes, &req); err != nil {
        log.Printf("[ERROR] Unmarshal in /art/upload: %v", err)
        c.JSON(http.StatusBadRequest, gin.H{"error": "Bad JSON"})
        return
    }

    log.Printf("[IMAGE] Adding image '%s' by '%s'", req.ArtName, req.Author)
    filename := generateASCIIFilePath(bytes)
    createASCIIFile(filename, bytes)  // unchanged

    if err := addImage(req, filename); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add image"})
        return
    }

    c.Status(http.StatusCreated)
}

func getArtCount(c *gin.Context) {
    count := getTotalImageCount()
    log.Printf("[API] Returning image count: %d", count)
    c.JSON(http.StatusOK, ImageCountResponse{Count: count})
}

// -------------------- File Helpers --------------------

func generateASCIIFilePath(bytes []byte) string {
    log.Println("[FILE] Generating filename from SHA512 hash")
    hash := hash512(bytes)
    filename := fmt.Sprintf("../images/%s.json", base64.URLEncoding.EncodeToString(hash))
    return filename
}

func createASCIIFile(filename string, bytes []byte) {
    log.Printf("[FILE] Creating file: %s", filename)
    file, err := os.Create(filename)
    if err != nil {
        log.Printf("[ERROR] Creating file: %v", err)
        return
    }
    defer file.Close()

    if _, err := file.Write(bytes); err != nil {
        log.Printf("[ERROR] Writing to file: %v", err)
    } else {
        log.Printf("[FILE] Write successful")
    }
}

func hash512(bytes []byte) []byte {
    sha := sha512.New()
    if _, err := sha.Write(bytes); err != nil {
        log.Printf("[ERROR] SHA512 write error: %v", err)
    }
    return sha.Sum(nil)
}

// -------------------- Main --------------------

func main() {
    initDB()

    gin.SetMode(gin.ReleaseMode)
    r := gin.Default()

    r.GET("/", healthCheck)
    r.HEAD("/", healthCheck)

    r.POST("/debug", debugLog)

    auth := r.Group("/auth")
    {
        auth.POST("/login", authenticateUser)
        auth.POST("/signup", registerUser)
    }

    // Art endpoints
    art := r.Group("/art")
    {
        art.POST("/upload", uploadArt)
        art.GET("/count", getArtCount)
        art.GET("/stream", handleArtStream)
    }

    log.Println("[SERVER] Starting on http://0.0.0.0:8080")
    if err := r.Run("0.0.0.0:8080"); err != nil {
        log.Fatalf("[SERVER] Failed to start: %v", err)
    }
}
