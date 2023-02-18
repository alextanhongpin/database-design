package main

import (
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"net/http"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed html/*.html
var files embed.FS

var templateFunc = map[string]any{
	"unescapeHTML": func(s string) template.HTML {
		return template.HTML(s)
	},
	"inc": func(i int) int {
		return i + 1
	},
}

var layout = template.Must(template.ParseFS(files, "html/layout.html")).Funcs(templateFunc)
var templates = template.Must(layout.ParseFS(files, "html/home.html"))

func main() {
	db, err := sql.Open("sqlite3", "file:search.db?cache=shared")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.Handle("/", home(db))

	fmt.Println("Listening to port *:8080. Press ctrl + c to cancel.")
	http.ListenAndServe(":8080", mux)
}

func home(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var data map[string]any
		q := r.URL.Query().Get("q")
		if q != "" {
			res, err := search(db, q)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			data = map[string]any{
				"Data": res,
				"Q":    q,
			}
		}

		err := templates.ExecuteTemplate(w, "home.html", data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

type SearchResult struct {
	Path  string
	Match string
}

func search(db *sql.DB, q string) ([]SearchResult, error) {
	rows, err := db.Query(`
		select
			path, 
			snippet(docs_idx, 1, '<b>', '</b>', '...', 32) 
		from docs_idx 
		where docs_idx match ? 
		order by bm25(docs_idx)
		limit 10`, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var res []SearchResult
	for rows.Next() {
		var row SearchResult
		if err := rows.Scan(&row.Path, &row.Match); err != nil {
			return nil, err
		}

		res = append(res, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return res, nil
}
