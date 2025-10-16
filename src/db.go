package main

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

const DB_NAME = "../ascii.db"

func addImageDB(author string, artName string, filename string) {
	log.Println("[IMAGE] Adding image to arts")
	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatal("[FATAL] Failed to open DB:", err)
	}
	defer db.Close()

	id, err := getUserID(author)
	if err != nil {
		log.Fatal("[FATAL] Failed to find user ", author, ":, err")
	}

	log.Println("[DB] Inserting into DB file:", filename, " for user: ", author)

	_, err = db.Exec("INSERT INTO Arts (artName, author, json_data) VALUES (?, ?, ?)", artName, id, filename)
	if err != nil {
		log.Fatal("[FATAL] Failed to insert into Arts db: ", err)
	}

}

func createUsersDB() {
	log.Println("[DB] Opening database for table creation")
	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatal("[FATAL] Failed to open DB:", err)
	}
	defer db.Close()

	log.Println("[DB] Creating users table if not exists")
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS Users (
		user_id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		password_hash TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatal("[FATAL] Failed to create users table:", err)
	}
	log.Println("[DB] Users table ensured")
}

func createArtBD() {
	log.Println("[DB] Opening database for table creation")
	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatal("[FATAL] Failed to open DB:", err)
	}
	defer db.Close()

	log.Println("[DB] Creating Arts table if not exists")
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS Arts (
		art_id INTEGER PRIMARY KEY AUTOINCREMENT,
		artName TEXT NOT NULL,
		author INTEGER,
		json_data TEXT NOT NULL,
		FOREIGN KEY (author) REFERENCES Users(user_id)
	)`)
	if err != nil {
		log.Fatal("[FATAL] Failed to create arts table:", err)
	}
	log.Println("[DB] Arts table ensured")
}

func correctPassword(user user) bool {
	log.Printf("[AUTH] Verifying password for user: '%s'", user.Name)
	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatal("[FATAL] Failed to open DB:", err)
	}
	defer db.Close()

	var existingPasswordHash string
	err = db.QueryRow("SELECT password_hash FROM Users WHERE name = ?", user.Name).Scan(&existingPasswordHash)

	if err == sql.ErrNoRows {
		log.Printf("[AUTH] User '%s' not found", user.Name)
		return false
	}

	if err != nil {
		log.Fatalf("[FATAL] Query error for user '%s': %v", user.Name, err)
	}

	match := user.PasswordHash == existingPasswordHash
	if match {
		log.Printf("[AUTH] Password match for user '%s'", user.Name)
	} else {
		log.Printf("[AUTH] Password mismatch for user '%s'", user.Name)
	}
	return match
}

func userExistsDB(name string) bool {
	log.Printf("[DB] Checking if user '%s' exists", name)
	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatal("[FATAL] Failed to open DB:", err)
	}
	defer db.Close()

	var existingName string
	err = db.QueryRow("SELECT name FROM Users WHERE name = ?", name).Scan(&existingName)

	if err == sql.ErrNoRows {
		log.Printf("[DB] User '%s' does not exist", name)
		return false
	}

	if err != nil {
		log.Fatalf("[FATAL] Query error for user '%s': %v", name, err)
	}

	log.Printf("[DB] User '%s' exists", name)
	return true
}

func getUserID(user string) (int, error) {
	log.Println("[USER] Finding user: ", user)
	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatal("[FATAL] Failed to open DB:", err)
	}
	defer db.Close()

	var id int
	err = db.QueryRow("SELECT user_id FROM Users WHERE name = ?", user).Scan(&id)

	if err != nil {
		return -1, err
	}

	return id, nil
}

func addUserDB(user user) {
	log.Printf("[DB] Adding user '%s'", user.Name)
	db, err := sql.Open("sqlite3", DB_NAME)
	if err != nil {
		log.Fatal("[FATAL] Failed to open DB:", err)
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO Users (name, password_hash) VALUES (?, ?)", user.Name, user.PasswordHash)
	if err != nil {
		log.Fatalf("[FATAL] Failed to insert user '%s': %v", user.Name, err)
	}

	log.Printf("[DB] User '%s' added successfully", user.Name)
}
