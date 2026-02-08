package main

import (
	"database/sql"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func generateID() string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	id := ""

	for i := 0; i < 6; i++ {
		randomIndex := rand.Intn(len(chars))
		id += string(chars[randomIndex])
	}

	return id
}

func main() {
	rand.Seed(time.Now().UnixNano())

	//  Connection to the  database
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Connected to database")

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/create", createHandler)
	http.HandleFunc("/view/", viewHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Println("Server running on port", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))

}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "templates/home.html")
}

func createHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		http.ServeFile(w, r, "templates/create.html")
		return
	}

	if r.Method == "POST" {
		r.ParseForm()
		content := r.FormValue("content")

		id := generateID()
		expiry := time.Now().Add(24 * time.Hour)

		_, err := db.Exec(
			"INSERT INTO pastes (id, content, created_at, expires_at) VALUES ($1, $2, $3, $4)",
			id,
			content,
			time.Now(),
			expiry,
		)

		if err != nil {
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/view/"+id, http.StatusSeeOther)
	}
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/view/")

	var content string
	var expiresAt time.Time

	err := db.QueryRow(
		"SELECT content, expires_at FROM pastes WHERE id = $1",
		id,
	).Scan(&content, &expiresAt)

	if err == sql.ErrNoRows {
		http.NotFound(w, r)
		return
	}

	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}

	// Expiration check
	if time.Now().After(expiresAt) {
		db.Exec("DELETE FROM pastes WHERE id = $1", id)
		http.NotFound(w, r)
		return
	}

	t, err := template.ParseFiles("templates/view.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	fullURL := "https://" + r.Host + "/view/" + id

	t.Execute(w, struct {
		Content string
		URL     string
	}{
		Content: content,
		URL:     fullURL,
	})

}
