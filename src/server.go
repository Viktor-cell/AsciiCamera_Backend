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
)

type user struct {
	Name         string `json:"name"`
	PasswordHash string `json:"password_hash"`
}

type req struct {
	Author  string   `json:"author"`
	ArtName string   `json:"artName"`
	Width   int      `json:"width"`
	Heigth  int      `json:"heigth"`
	Letters []string `json:"letters"`
	Colors  []int32  `json:"colors"`
}

func main() {
	mux := http.NewServeMux()

	createUsersDB()

	mux.HandleFunc("/", root)
	mux.HandleFunc("/debug", debug)
	mux.HandleFunc("/log_in", logIn)
	mux.HandleFunc("/sign_up", signUp)
	mux.HandleFunc("/add_image", addImage)

	handler := http.Handler(mux)

	log.Println("Server starting on http://0.0.0.0:8080")
	err := http.ListenAndServe("0.0.0.0:8080", handler)
	if err != nil {
		log.Printf("[ERROR] Failed to create server: %v", err)
	}
}

func root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Working")
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
	var user user
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[ERROR] Reading body in /sign_in: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(bytes, &user); err != nil {
		log.Printf("[ERROR] Unmarshal in /sign_in: %v", err)
		http.Error(w, "Bad JSON", http.StatusBadRequest)
		return
	}

	if correctPassword(user) {
		log.Printf("[LOGIN] User '%s' authenticated successfully", user.Name)
		w.WriteHeader(http.StatusOK)
	} else {
		log.Printf("[LOGIN] Bad password attempt for user '%s'", user.Name)
		http.Error(w, "bad name or password", http.StatusUnauthorized)
	}
}

func signUp(w http.ResponseWriter, r *http.Request) {
	var user user
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[ERROR] Reading body in /login: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(bytes, &user); err != nil {
		log.Printf("[ERROR] Unmarshal in /login: %v", err)
		http.Error(w, "Bad JSON", http.StatusBadRequest)
		return
	}

	if userExists(user.Name) {
		log.Printf("[SIGNUP] User '%s' already exists", user.Name)
		http.Error(w, "user exists", http.StatusConflict)
	} else {
		log.Printf("[SIGNUP] Creating user '%s'", user.Name)
		addUser(user)
		w.WriteHeader(http.StatusOK)
	}
}

func addImage(w http.ResponseWriter, r *http.Request) {
	var req req
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

	log.Printf("[IMAGE] Adding image: %s by %s", req.ArtName, req.Author)
	createAsciiFile(bytes)
	log.Println("[IMAGE] File created successfully")
	w.WriteHeader(http.StatusCreated)
}

func createAsciiFile(bytes []byte) {
	log.Println("[FILE] Generating filename from hash")
	filenameRaw := hash512(bytes)
	filename := fmt.Sprintf("../images/%v.json", base64.URLEncoding.EncodeToString(filenameRaw))
	log.Printf("[FILE] Creating file: %s", filename)

	file, err := os.Create(filename)
	if err != nil {
		log.Printf("[ERROR] Creating file: %v", err)
		return
	}
	defer file.Close()

	_, err = file.Write(bytes)
	if err != nil {
		log.Printf("[ERROR] Writing to file: %v", err)
	} else {
		log.Printf("[FILE] Write successful")
	}
}

func hash512(bytes []byte) []byte {
	sha := sha512.New()
	_, err := sha.Write(bytes)
	if err != nil {
		log.Printf("[ERROR] Writing to SHA512: %v", err)
	}
	return sha.Sum(nil)
}
