package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Dictionary struct {
	ID          int    `json:"id"`
	Text        string `json:"text"`
	Description string `json:"description"`
}

var dbPool *pgxpool.Pool

func main() {
	godotenv.Load(".env")

	connString := os.Getenv("DATABASE_URL")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	var err error
	dbPool, err = pgxpool.New(context.Background(), connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer dbPool.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		fmt.Fprint(w, "Hello there")
	})

	http.HandleFunc("/dict", dictHandler)
	http.HandleFunc("/dict/{id}", entryHandler)

	fmt.Println("Server starting on :" + port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

func dictHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	switch r.Method {
	case http.MethodGet:
		rows, err := dbPool.Query(context.Background(), "select id, text, description from dictionary")
		if err != nil {
			log.Printf("Database failed: %v\n", err)
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		if err := dbPool.Ping(context.Background()); err != nil {
			log.Fatalf("Unable to connect to database: %v\n", err)
		}

		defer rows.Close()

		dicts := []Dictionary{}
		for rows.Next() {
			var e Dictionary
			if err := rows.Scan(&e.ID, &e.Text, &e.Description); err != nil {
				log.Printf("Query failed: %v\n", err)
				http.Error(w, "Internal server error", http.StatusInternalServerError)
				return
			}
			dicts = append(dicts, e)
		}
		json.NewEncoder(w).Encode(dicts)

	case http.MethodPost:
		var e Dictionary
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		err := dbPool.QueryRow(context.Background(), "insert into dictionary (text, description) values ($1, $2) returning id", e.Text, e.Description).Scan(&e.ID)
		if err != nil {
			log.Printf("Query failed: %v\n", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(e)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func entryHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		var e Dictionary
		err := dbPool.QueryRow(context.Background(), "select id, text, description from dictionary where id = $1", id).Scan(&e.ID, &e.Text, &e.Description)
		if err != nil {
			http.Error(w, "Entry not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(e)

	case http.MethodPut:
		var e Dictionary
		if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		commandTag, err := dbPool.Exec(context.Background(), "update dictionary set text = $1, description = $2 where id = $3", e.Text, e.Description, id)
		if err != nil {
			log.Printf("Query failed: %v\n", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if commandTag.RowsAffected() == 0 {
			http.Error(w, "Entry not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Entry updated successfully")

	case http.MethodDelete:
		commandTag, err := dbPool.Exec(context.Background(), "delete from dictionary where id = $1", id)
		if err != nil {
			log.Printf("Query failed: %v\n", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if commandTag.RowsAffected() == 0 {
			http.Error(w, "Entry not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "Entry deleted successfully")

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
