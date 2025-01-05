package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

func initDB() {
	var err error
	db, err = sql.Open("sqlite3", "todos.db")
	if err != nil {
		log.Fatal(err)
	}

	// Create todos table if not exists
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS todos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		content TEXT NOT NULL,
		status TEXT NOT NULL,
		deadline DATETIME NOT NULL,
		completed BOOLEAN DEFAULT FALSE,
		created_at DATETIME DEFAULT (datetime('now', 'localtime'))
	);`

	_, err = db.Exec(createTableSQL)
	if err != nil {
		log.Fatal(err)
	}
}

func loadTodos() []TodoItem {
	rows, err := db.Query(`
		SELECT content, status, deadline, completed 
		FROM todos 
		WHERE status != '√'
		ORDER BY 
			created_at DESC
	`)
	if err != nil {
		log.Printf("Error loading todos: %v", err)
		return nil
	}
	defer rows.Close()

	var todos []TodoItem
	for rows.Next() {
		var todo TodoItem
		var statusStr string
		var deadline string
		err := rows.Scan(&todo.content, &statusStr, &deadline, &todo.completed)
		if err != nil {
			log.Printf("Error scanning todo: %v", err)
			continue
		}
		// Convert empty string to space character for status
		if statusStr == "" {
			todo.status = ' '
		} else {
			todo.status = []rune(statusStr)[0]
		}
		// Parse time in local timezone
		todo.deadline = time.Now()  // Default to current time if parsing fails
		if t, err := time.Parse("01-02 15:04", deadline); err == nil {
			todo.deadline = t
		} else {
			log.Printf("Error parsing deadline: %v", err)
		}
		todos = append(todos, todo)
	}
	return todos
}

func saveTodo(todo TodoItem) error {
	if strings.TrimSpace(todo.content) == "" {
		return fmt.Errorf("content cannot be empty")
	}
	
	now := time.Now()
	_, err := db.Exec(`
		INSERT INTO todos (content, status, deadline, completed)
		VALUES (?, ?, ?, ?)
	`, todo.content, string(todo.status), now.Format("01-02 15:04"), todo.completed)
	return err
}

func updateTodo(todo TodoItem) error {
	now := time.Now()
	_, err := db.Exec(`
		UPDATE todos 
		SET status = ?, deadline = ?, completed = ?
		WHERE content = ?
	`, string(todo.status), now.Format("01-02 15:04"), todo.completed, todo.content)
	return err
}

func cleanupOldTodos() {
	// Remove todos that have been completed for more than 7 days
	_, err := db.Exec(`
		DELETE FROM todos 
		WHERE status = '√' 
		AND datetime(deadline) < datetime('now', '-7 days')
	`)
	if err != nil {
		log.Printf("Error cleaning up old todos: %v", err)
	}
}

func closeDB() {
	if db != nil {
		db.Close()
	}
}
