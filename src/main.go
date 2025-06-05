package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/glebarez/sqlite"
)

type ImageMeta struct {
	Name   string
	SHA256 string
	Tags   []string
}

func discard(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func initDB(path string) *sql.DB {
	db, err := sql.Open("sqlite", path)
	discard(err)

	createTable := `
	CREATE TABLE IF NOT EXISTS images (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT,
		sha256 TEXT UNIQUE,
		tags TEXT
	);`
	_, err = db.Exec(createTable)
	discard(err)

	return db
}

func updateImageTags(db *sql.DB, sha256 string, tags []string) error {
	tagStr := strings.Join(tags, ",")
	_, err := db.Exec("UPDATE images SET tags = ? WHERE sha256 = ?", tagStr, sha256)
	if err == nil {
		fmt.Println("Updated ", sha256, " with tags ", tags)
	}
	return err
}

func updateImageTagsByName(db *sql.DB, name string, tags []string) error {
	tagStr := strings.Join(tags, ",")
	_, err := db.Exec("UPDATE images SET tags = ? WHERE name = ?", tagStr, name)
	if err == nil {
		fmt.Println("Updated ", name, " with tags ", tags)
	}
	return err
}

func getTagsForImage(db *sql.DB, sha256 string) ([]string, error) {
	var tagStr string
	err := db.QueryRow("SELECT tags FROM images WHERE sha256 = ?", sha256).Scan(&tagStr)
	if err != nil {
		return nil, err
	}
	tags := strings.Split(tagStr, ",")
	return tags, nil
}

func deleteTagFromImage(db *sql.DB, sha256 string, tagToRemove string) error {
	var tagStr string
	err := db.QueryRow("SELECT tags FROM images WHERE sha256 = ?", sha256).Scan(&tagStr)
	if err != nil {
		return err
	}

	// Split tags, remove unwanted one
	tags := strings.Split(tagStr, ",")
	var newTags []string
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" && trimmed != tagToRemove {
			newTags = append(newTags, trimmed)
		}
	}

	// Join and update
	updated := strings.Join(newTags, ",")
	_, err = db.Exec("UPDATE images SET tags = ? WHERE sha256 = ?", updated, sha256)
	if err == nil {
		fmt.Println("Updated ", sha256, " with tags ", updated)
	}
	return err
}

func storeImages(db *sql.DB, images []ImageMeta) {
	for _, img := range images {
		var existingName string
		err := db.QueryRow("SELECT name FROM images WHERE sha256 = ?", img.SHA256).Scan(&existingName)
		if err == sql.ErrNoRows {
			// Insert new image
			_, err := db.Exec("INSERT INTO images (name, sha256, tags) VALUES (?, ?, ?)", img.Name, img.SHA256, "")
			discard(err)
			fmt.Println("Inserted:", img.Name)
		} else if err == nil {
			// Exists, maybe update name
			if existingName != img.Name {
				_, err := db.Exec("UPDATE images SET name = ? WHERE sha256 = ?", img.Name, img.SHA256)
				discard(err)
				fmt.Println("Updated name:", existingName, "->", img.Name)
			}
		} else {
			discard(err)
		}
	}
}

func getImagesPath(dir string) []ImageMeta {
	var metas []ImageMeta
	entries, err := os.ReadDir(dir)
	discard(err)

	for _, e := range entries {
		if !e.IsDir() {
			name := e.Name()
			lower := strings.ToLower(name)
			ext := filepath.Ext(lower)
			if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" ||
				ext == ".webm" || ext == ".webp" {

				sha := getShaForFile(filepath.Join(dir, name))
				meta := ImageMeta{
					Name:   name,
					SHA256: sha,
					Tags:   []string{},
				}
				metas = append(metas, meta)
			}
		}
	}
	return metas
}

func getShaForFile(file string) string {
	f, err := os.Open(file)
	discard(err)
	defer f.Close()

	h := sha256.New()
	_, err = io.Copy(h, f)
	discard(err)

	return fmt.Sprintf("%x", h.Sum(nil))
}

func main() {
	db := initDB("images.db")
	defer db.Close()

	images := getImagesPath(".")
	storeImages(db, images)

	fs := http.FileServer(http.Dir("src/web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Host != "localhost:8080" {
			fmt.Println("Not Me...")
			fmt.Fprintf(w, "%s", "Not Me...")
		}
		tmplPath := filepath.Join("src", "web", "templates", "index.html")
		tmpl, err := template.ParseFiles(tmplPath)
		if err != nil {
			http.Error(w, "Template error", http.StatusInternalServerError)
			log.Println("Template error:", err)
			return
		}
		tmpl.Execute(w, nil)
	})

	tags, err := getTagsForImage(db, getShaForFile("b.png"))
	discard(err)
	fmt.Println("tags: ", tags)

	err = updateImageTagsByName(db, "b.png", []string{"HELLO", "World!", "Three"})
	discard(err)

	tags, err = getTagsForImage(db, getShaForFile("b.png"))
	discard(err)
	fmt.Println("tags: ", tags)

	err = deleteTagFromImage(db, getShaForFile("b.png"), "HELLO")
	discard(err)

	tags, err = getTagsForImage(db, getShaForFile("b.png"))
	discard(err)
	fmt.Println("tags: ", tags)

	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
