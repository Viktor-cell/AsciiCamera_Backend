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
	Colors  []int32 `json:"colors"`
}

func main() {
	mux := http.NewServeMux()

	create_users_db()

	mux.HandleFunc("/", root)
	mux.HandleFunc("/debug", debug)
	mux.HandleFunc("/login", login)
	mux.HandleFunc("/sign_in", signIn)
	mux.HandleFunc("/add_image", addImage)

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

	err = json.Unmarshal(bytes, &user)
	check(err)

	if correctPassword(user) {
		log.Println("user ", user.Name, " entered right password")
		w.WriteHeader(http.StatusOK)
	} else {
		log.Println("Bad password for ", user.Name)
		http.Error(w, "bad name or password", http.StatusUnauthorized)
	}
}

func addImage(w http.ResponseWriter, r *http.Request) {
	var req req
	bytes, err := io.ReadAll(r.Body)
	check(err)

	err = json.Unmarshal(bytes, &req)
	check(err)

	log.Println("Creating file")
	create_ascii_file(bytes)
	log.Println("File created")
}

func create_ascii_file(bytes []byte) {
	filenameRaw := hash512(bytes)
	filename := fmt.Sprintf("../images/%v.json", base64.URLEncoding.EncodeToString(filenameRaw))

	file, err := os.Create(filename)
	check(err)
	defer file.Close()

	_, err = file.Write(bytes)
	check(err)
}

func hash512(bytes []byte) []byte {
	sha := sha512.New()
	_, err := sha.Write(bytes)
	check(err)

	out := sha.Sum(nil)
	return out
}
