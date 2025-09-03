package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/cors"
)

type Weight struct {
	ID    int     `json:"id"`
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

type Checklist struct {
	ID      int    `json:"id"`
	Date    string `json:"date"`
	Item    string `json:"item"`
	Checked bool   `json:"checked"`
}

var db *sql.DB

func main() {
	var err error

	// ambil config dari ENV
	dbUser := os.Getenv("DB_USER")   // contoh: u107388512_root
	dbPass := os.Getenv("DB_PASS")   // password user mysql hostinger
	dbHost := os.Getenv("DB_HOST")   // contoh: mysql.hostinger.com
	dbName := os.Getenv("DB_NAME")   // contoh: u107388512_bmi_tracker
	port := os.Getenv("PORT")        // Render kasih otomatis, default fallback 8080

	if port == "" {
		port = "8080"
	}

	// format DSN
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true",
		dbUser, dbPass, dbHost, dbName)

	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatal("DB connect error:", err)
	}
	defer db.Close()

	// routing
	mux := http.NewServeMux()
	mux.HandleFunc("/weights", weightsHandler)
	mux.HandleFunc("/checklist", checklistHandler)

	// cors
	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"}, // bisa diubah ke domain FE lu
		AllowedMethods:   []string{"GET", "POST", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}).Handler(mux)

	log.Println("Server jalan di port", port)
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// ================== Weights Handler ==================
func weightsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query("SELECT id, date, value FROM weights ORDER BY date")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var list []Weight
		for rows.Next() {
			var wt Weight
			if err := rows.Scan(&wt.ID, &wt.Date, &wt.Value); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			list = append(list, wt)
		}
		json.NewEncoder(w).Encode(list)

	case http.MethodPost:
		var wt Weight
		if err := json.NewDecoder(r.Body).Decode(&wt); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if wt.Date == "" {
			wt.Date = time.Now().Format("2006-01-02")
		}
		res, err := db.Exec("INSERT INTO weights (date, value) VALUES (?, ?)", wt.Date, wt.Value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		id, _ := res.LastInsertId()
		wt.ID = int(id)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(wt)

	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			http.Error(w, "id is required", http.StatusBadRequest)
			return
		}

		_, err := db.Exec("DELETE FROM weights WHERE id = ?", id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "Data deleted"})
	}
}

// ================== Checklist Handler ==================
func checklistHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	today := time.Now().Format("2006-01-02")

	switch r.Method {
	case http.MethodGet:
		rows, err := db.Query("SELECT id, date, item, checked FROM meal_checklist WHERE date = ?", today)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var list []Checklist
		for rows.Next() {
			var c Checklist
			if err := rows.Scan(&c.ID, &c.Date, &c.Item, &c.Checked); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			list = append(list, c)
		}
		json.NewEncoder(w).Encode(list)

	case http.MethodPost:
		var c Checklist
		if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		c.Date = today

		_, err := db.Exec(`
			INSERT INTO meal_checklist (date, item, checked) 
			VALUES (?, ?, ?)
			ON DUPLICATE KEY UPDATE checked = VALUES(checked)
		`, c.Date, c.Item, c.Checked)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(c)

	case http.MethodDelete:
		item := r.URL.Query().Get("item")
		if item == "" {
			http.Error(w, "item is required", http.StatusBadRequest)
			return
		}

		_, err := db.Exec("DELETE FROM meal_checklist WHERE date = ? AND item = ?", today, item)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "deleted"})
	}
}
