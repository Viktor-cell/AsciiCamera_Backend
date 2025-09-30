package main

import (
	_ "crypto/sha256"
	_ "encoding/base64"
	"encoding/json"
	_ "encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	_ "os"
)

type asciiData struct {
	Name    string   `json:"name"`
	Width   int      `json:"width"`
	Height  int      `json:"height"`
	Letters []string `json:"letters"`
	Colors  []uint32 `json:"colors"`
}

type user struct {
	Name         string `json:"name"`
	PasswordHash string `json:"password_hash"`
}

func main() {
	mux := http.NewServeMux()

	create_users_db()

	mux.HandleFunc("/", root)
	mux.HandleFunc("/debug", debug)
	mux.HandleFunc("/login", login)
	mux.HandleFunc("/sign_in", signIn)

	log.Println("server at 0.0.0.0:8080")
	handler := http.Handler(mux)

	err := http.ListenAndServe("0.0.0.0:8080", handler)
	check(err)
}

func check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func root(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Working")
}

func debug(w http.ResponseWriter, r *http.Request) {
	bytes, err := io.ReadAll(r.Body)
	check(err)
	fmt.Println(string(bytes))
}

func login(w http.ResponseWriter, r *http.Request) {
	var user user
	bytes, err := io.ReadAll(r.Body)
	check(err)

	json.Unmarshal(bytes, &user)

	if user_exists(user.Name) {
		log.Println("user ", user.Name, " exists")
		http.Error(w, "user exists", http.StatusConflict)
	} else {
		log.Println("creating ", user.Name)
		w.WriteHeader(http.StatusOK)
		add_user(user)
	}
}

func signIn(w http.ResponseWriter, r *http.Request) {
	var user user
	bytes, err := io.ReadAll(r.Body)
	check(err)

	json.Unmarshal(bytes, &user)

	if checkUserWithPassword(user) {
		log.Println("user ", user.Name, " entered right password")
		w.WriteHeader(http.StatusOK)
	} else {
		log.Println("Bad password for ", user.Name)
		http.Error(w, "bad name or password", http.StatusUnauthorized)
		add_user(user)
	}
}
