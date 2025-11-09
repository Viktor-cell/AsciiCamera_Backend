package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

const DB_NAME = "../ascii.db"

// -------------------- User Functions --------------------

func addUserDB(user user) {
	log.Printf("[DB] Adding user '%s'", user.Name)

	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatalf("[FATAL] Failed to open DB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO Users (name, password_hash) VALUES (?, ?)", user.Name, user.PasswordHash)
	if err != nil {
		log.Fatalf("[FATAL] Failed to insert user '%s': %v", user.Name, err)
	}

	log.Printf("[DB] User '%s' added successfully", user.Name)
}

func getUserID(name string) (int, error) {
	log.Printf("[USER] Finding user: '%s'", name)

	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatalf("[FATAL] Failed to open DB: %v", err)
	}
	defer db.Close()

	var id int
	err = db.QueryRow("SELECT user_id FROM Users WHERE name = ?", name).Scan(&id)
	if err != nil {
		log.Printf("[USER] User '%s' not found: %v", name, err)
		return -1, err
	}

	log.Printf("[USER] Found user '%s' with ID %d", name, id)
	return id, nil
}

func userExistsDB(name string) bool {
	log.Printf("[DB] Checking if user '%s' exists", name)

	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatalf("[FATAL] Failed to open DB: %v", err)
	}
	defer db.Close()

	var existingName string
	err = db.QueryRow("SELECT name FROM Users WHERE name = ?", name).Scan(&existingName)

	if err == sql.ErrNoRows {
		log.Printf("[DB] User '%s' does not exist", name)
		return false
	} else if err != nil {
		log.Fatalf("[FATAL] Query error for user '%s': %v", name, err)
	}

	log.Printf("[DB] User '%s' exists", name)
	return true
}

func correctPassword(user user) bool {
	log.Printf("[AUTH] Verifying password for user '%s'", user.Name)

	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatalf("[FATAL] Failed to open DB: %v", err)
	}
	defer db.Close()

	var hash string
	err = db.QueryRow("SELECT password_hash FROM Users WHERE name = ?", user.Name).Scan(&hash)
	if err == sql.ErrNoRows {
		log.Printf("[AUTH] User '%s' not found", user.Name)
		return false
	} else if err != nil {
		log.Fatalf("[FATAL] Query error for user '%s': %v", user.Name, err)
	}

	if user.PasswordHash == hash {
		log.Printf("[AUTH] Password match for user '%s'", user.Name)
		return true
	}

	log.Printf("[AUTH] Password mismatch for user '%s'", user.Name)
	return false
}

// -------------------- ASCII Art Functions --------------------

func addImageDB(author, artName, filename string) {
	log.Printf("[IMAGE] Adding image '%s' for author '%s'", artName, author)

	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatalf("[FATAL] Failed to open DB: %v", err)
	}
	defer db.Close()

	id, err := getUserID(author)
	if err != nil {
		log.Fatalf("[FATAL] Failed to find user '%s': %v", author, err)
	}

	_, err = db.Exec("INSERT INTO Arts (artName, author, json_data) VALUES (?, ?, ?)", artName, id, filename)
	if err != nil {
		log.Fatalf("[FATAL] Failed to insert into Arts: %v", err)
	}

	log.Printf("[DB] Image '%s' added successfully", artName)
}

func getAsciiArtFilePath(id int) string {
	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatalf("[FATAL] Failed to open DB: %v", err)
	}
	defer db.Close()

	var path string
	err = db.QueryRow("SELECT json_data FROM Arts WHERE art_id = ?", id).Scan(&path)
	if err != nil {
		log.Printf("[DB] No ASCII art found with ID %d", id)
		return ""
	}

	log.Printf("[DB] Found ASCII art file: %s", path)
	return path
}

// -------------------- DB Creation --------------------

func createUsersDB() {
	log.Println("[DB] Ensuring Users table exists")

	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatalf("[FATAL] Failed to open DB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS Users (
		user_id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		password_hash TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create Users table: %v", err)
	}

	log.Println("[DB] Users table ensured")
}

func createArtsDB() {
	log.Println("[DB] Ensuring Arts table exists")

	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatalf("[FATAL] Failed to open DB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS Arts (
		art_id INTEGER PRIMARY KEY AUTOINCREMENT,
		artName TEXT NOT NULL,
		author INTEGER,
		json_data TEXT NOT NULL,
		FOREIGN KEY (author) REFERENCES Users(user_id)
	)`)
	if err != nil {
		log.Fatalf("[FATAL] Failed to create Arts table: %v", err)
	}

	log.Println("[DB] Arts table ensured")
}
