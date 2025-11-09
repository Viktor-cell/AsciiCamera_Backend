package main

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/websocket"
)

type user struct {
	Name         string `json:"name"`
	PasswordHash string `json:"password_hash"`
}

type ASCIIart struct {
	Author   string   `json:"author"`
	ArtName  string   `json:"artName"`
	Width    int      `json:"width"`
	Height   int      `json:"height"`
	Letters  []string `json:"letters"`
	Colors   []int32  `json:"colors"`
}

type ASCIIartRequest struct {
	ID    int `json:"id"`
	Count int `json:"msg"`
}

type ASCIIartResponse struct {
	ID  int         `json:"id"`
	Msg []ASCIIart  `json:"msg"`
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// -------------------- WebSocket Handler --------------------

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade error: %v", err)
		return
	}
	defer conn.Close()

	log.Println("[WS] Client connected")

	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[WS] Read error: %v", err)
			break
		}

		var req ASCIIartRequest
		if err := json.Unmarshal(msg, &req); err != nil {
			log.Printf("[WS] JSON unmarshal error: %v", err)
			break
		}

		log.Printf("[WS] Request received: %+v", req)

		var arts []ASCIIart
		for i := 1; i <= req.Count; i++ {
			path := getAsciiArtFilePath(i)
			if path == "" {
				log.Printf("[WS] No file for art ID %d", i)
				break
			}
			data, err := readFile(path)
			if err != nil {
				log.Printf("[WS] Read file error: %v", err)
				break
			}
			var tmp ASCIIart
			if err := json.Unmarshal(data, &tmp); err != nil {
				log.Printf("[WS] JSON unmarshal error for file %s: %v", path, err)
				break
			}
			arts = append(arts, tmp)
		}

		res := ASCIIartResponse{
			ID:  req.ID,
			Msg: arts,
		}

		log.Printf("[WS] Sending response with %v items", len(res.Msg))
		if err := conn.WriteJSON(res); err != nil {
			log.Printf("[WS] Write error: %v", err)
			break
		}
	}

	log.Println("[WS] Client disconnected")
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

func root(w http.ResponseWriter, r *http.Request) {
	// log.Println("[HTTP] / root accessed")
	fmt.Fprintln(w, "Working")
}

func debug(w http.ResponseWriter, r *http.Request) {
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[ERROR] Reading body in /debug: %v", err)
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	log.Println("[DEBUG]", string(bytes))
	fmt.Fprintln(w, "Debug logged")
}

func logIn(w http.ResponseWriter, r *http.Request) {
	var u user
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[ERROR] Reading body in /log_in: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(bytes, &u); err != nil {
		log.Printf("[ERROR] Unmarshal in /log_in: %v", err)
		http.Error(w, "Bad JSON", http.StatusBadRequest)
		return
	}

	if correctPassword(u) {
		log.Printf("[LOGIN] User '%s' authenticated successfully", u.Name)
		w.WriteHeader(http.StatusOK)
	} else {
		log.Printf("[LOGIN] Bad password attempt for user '%s'", u.Name)
		http.Error(w, "Bad name or password", http.StatusUnauthorized)
	}
}

func signUp(w http.ResponseWriter, r *http.Request) {
	var u user
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[ERROR] Reading body in /sign_up: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(bytes, &u); err != nil {
		log.Printf("[ERROR] Unmarshal in /sign_up: %v", err)
		http.Error(w, "Bad JSON", http.StatusBadRequest)
		return
	}

	if userExistsDB(u.Name) {
		log.Printf("[SIGNUP] User '%s' already exists", u.Name)
		http.Error(w, "User exists", http.StatusConflict)
	} else {
		log.Printf("[SIGNUP] Creating user '%s'", u.Name)
		addUserDB(u)
		w.WriteHeader(http.StatusOK)
	}
}

func addImage(w http.ResponseWriter, r *http.Request) {
	var req ASCIIart
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[ERROR] Reading body in /add_image: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(bytes, &req); err != nil {
		log.Printf("[ERROR] Unmarshal in /add_image: %v", err)
		http.Error(w, "Bad JSON", http.StatusBadRequest)
		return
	}

	log.Printf("[IMAGE] Adding image '%s' by '%s'", req.ArtName, req.Author)
	filename := generateASCIIFilePath(bytes)
	createASCIIFile(filename, bytes)
	addImageDB(req.Author, req.ArtName, filename)
	w.WriteHeader(http.StatusCreated)
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
	mux := http.NewServeMux()
	mux.HandleFunc("/", root)
	mux.HandleFunc("/debug", debug)
	mux.HandleFunc("/log_in", logIn)
	mux.HandleFunc("/sign_up", signUp)
	mux.HandleFunc("/add_image", addImage)
	mux.HandleFunc("/ws", handleWS)

	log.Println("[SERVER] Starting on http://0.0.0.0:8080")
	if err := http.ListenAndServe("0.0.0.0:8080", mux); err != nil {
		log.Fatalf("[SERVER] Failed to start: %v", err)
	}
}
