package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

const apiKey = "YOUR_OMDB_API_KEY"

var db *sql.DB

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Movie struct {
	Title  string `json:"Title"`
	Year   string `json:"Year"`
	Genre  string `json:"Genre"`
	Plot   string `json:"Plot"`
	ImdbID string `json:"imdbID"`
}

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./movie.db")
	if err != nil {
		log.Fatal(err)
	}

	createTables()

	http.HandleFunc("/users", createUser)
	http.HandleFunc("/movies", searchMovies)
	http.HandleFunc("/watchlist", addToWatchlist)
	http.HandleFunc("/watchlist/", getWatchlist)

	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

func createTables() {
	userTable := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		email TEXT
	);`

	watchlistTable := `
	CREATE TABLE IF NOT EXISTS watchlist (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER,
		movie_id TEXT,
		rating INTEGER
	);`

	db.Exec(userTable)
	db.Exec(watchlistTable)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var user User
	json.NewDecoder(r.Body).Decode(&user)

	result, err := db.Exec("INSERT INTO users(name,email) VALUES(?,?)", user.Name, user.Email)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	id, _ := result.LastInsertId()
	user.ID = int(id)

	json.NewEncoder(w).Encode(user)
}

func searchMovies(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("search")
	if query == "" {
		http.Error(w, "Missing search parameter", 400)
		return
	}

	url := fmt.Sprintf("http://www.omdbapi.com/?apikey=%s&s=%s", apiKey, query)

	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer resp.Body.Close()

	var data interface{}
	json.NewDecoder(resp.Body).Decode(&data)

	json.NewEncoder(w).Encode(data)
}

func addToWatchlist(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", 405)
		return
	}

	var item struct {
		UserID  int    `json:"user_id"`
		MovieID string `json:"movie_id"`
	}

	json.NewDecoder(r.Body).Decode(&item)

	_, err := db.Exec("INSERT INTO watchlist(user_id,movie_id) VALUES(?,?)",
		item.UserID, item.MovieID)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Write([]byte("Added to watchlist"))
}

func getWatchlist(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 3 {
		http.Error(w, "User ID required", 400)
		return
	}

	userID := parts[2]

	rows, err := db.Query("SELECT movie_id, rating FROM watchlist WHERE user_id=?", userID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	defer rows.Close()

	type WatchlistItem struct {
		MovieID string `json:"movie_id"`
		Rating  int    `json:"rating"`
	}

	var items []WatchlistItem

	for rows.Next() {
		var item WatchlistItem
		rows.Scan(&item.MovieID, &item.Rating)
		items = append(items, item)
	}

	json.NewEncoder(w).Encode(items)
}
