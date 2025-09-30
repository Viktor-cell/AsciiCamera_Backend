package main

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

const DB_NAME = "ascii.db"

func create_users_db() {
	db, err := sql.Open("sqlite3", DB_NAME)
	check(err)
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users(id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, password_hash TEXT NOT NULL) ")
	check(err)
}

func checkUserWithPassword(user user) bool {
	db, err := sql.Open("sqlite3", DB_NAME)
	check(err)
	defer db.Close()

	var existingPasswordHash string

	err = db.QueryRow("SELECT password_hash FROM users WHERE name = ?", user.Name).Scan(&existingPasswordHash)

	if err == sql.ErrNoRows {
		return false
	}

	if err != nil {
		check(err)
	}

	return user.PasswordHash == existingPasswordHash
}

func user_exists(name string) bool {
	db, err := sql.Open("sqlite3", DB_NAME)
	check(err)
	defer db.Close()

	var existingName string
	err = db.QueryRow("SELECT name FROM users WHERE name = ?", name).Scan(&existingName)

	if err == sql.ErrNoRows {
		return false
	}

	if err != nil {
		check(err)
	}

	return true
}

func add_user(user user) {
	db, err := sql.Open("sqlite3", DB_NAME)
	check(err)
	defer db.Close()

	_, err = db.Exec("INSERT INTO users (name, password_hash) VALUES (?, ?)", user.Name, user.PasswordHash)
	check(err)
}
