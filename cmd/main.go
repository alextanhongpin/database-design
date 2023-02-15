package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	cmd := os.Args[1]
	if cmd == "" {
		panic("e.g. init")
	}

	var query string
	flag.StringVar(&query, "q", "", "search keyword")
	// NOTE: If we have a command before the flag, it will not be interpreted.
	// We need to start parsing after the first arg.
	//flag.Parse()
	flag.CommandLine.Parse(os.Args[2:])

	db, err := sql.Open("sqlite3", "file:search.db?cache=shared")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	switch cmd {
	case "init":
		fmt.Println("initializing...")
		if err := initDB(db); err != nil {
			log.Fatal(err)
		}
	case "index":
		fmt.Println("indexing...")
		if err := index(db); err != nil {
			log.Fatal(err)
		}
	case "search":
		fmt.Println("searching...", query)
		res, err := search(db, query)
		if err != nil {
			log.Fatal(err)
		}
		for i, r := range res {
			fmt.Printf("%d) %s\n\n%s\n\n", i+1, r.Path, r.Match)
		}
	default:
		log.Fatalf("invalid command: %s", cmd)
	}
}

type SearchResult struct {
	Path  string
	Match string
}

func search(db *sql.DB, q string) ([]SearchResult, error) {
	rows, err := db.Query(`select path, snippet(docs_idx, 1, '<b>', '</b>', '...', 32) from docs_idx where docs_idx match ? order by rank`, q)
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

func initDB(db *sql.DB) error {
	//https://www.sqlite.org/fts5.html#external_content_and_contentless_tables
	_, err := db.Exec(`create table docs (id integer primary key, path text not null unique, markdown text not null)`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`create virtual table docs_idx using fts5(path, markdown, content='docs', content_rowid='id')`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TRIGGER docs_ai AFTER INSERT ON docs BEGIN
  INSERT INTO docs_idx(rowid, path, markdown) VALUES (new.id, new.path, new.markdown);
END;`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TRIGGER docs_ad AFTER DELETE ON docs BEGIN
  INSERT INTO docs_idx(docs_idx, rowid, path, markdown) VALUES('delete', old.id, old.path, old.markdown);
END;`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`CREATE TRIGGER docs_au AFTER UPDATE ON docs BEGIN
  INSERT INTO docs_idx(docs_idx, rowid, path, markdown) VALUES('delete', old.id, old.path, old.markdown);
  INSERT INTO docs_idx(rowid, path, markdown) VALUES (new.id, new.path, new.markdown);
END;`)
	return err
}

func index(db *sql.DB) error {
	return filepath.Walk(".",
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip non-markdown file.
			if filepath.Ext(path) != ".md" {
				return nil
			}

			b, err := os.ReadFile(path)
			if err != nil {
				return fmt.Errorf("%w: failed to read file %s", err, path)
			}

			markdown := string(b)

			_, err = db.Exec(`insert into docs(path, markdown) values (?, ?) on conflict (path) do update set markdown = excluded.markdown`, path, markdown)
			if err != nil {
				return fmt.Errorf("%w: failed to insert %s", err, path)
			}

			return nil
		})
}
