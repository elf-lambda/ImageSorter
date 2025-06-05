package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	_ "github.com/glebarez/sqlite"
)

var imageRootPath = `C:\Users\elff\Pictures`

type ImageMeta struct {
	Name         string
	SHA256       string
	Tags         []string
	ThumbnailURL string
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
	var tagStr string
	if tags[0] == "" && len(tags) == 2 {
		tagStr = strings.ToLower(tags[1])
	} else {
		tagStr = strings.Join(tags, ",")
	}
	fmt.Println(tagStr)
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
		if err == sql.ErrNoRows {
			return []string{}, nil
		}
		return nil, err
	}
	tags := strings.Split(tagStr, ",")
	var filteredTags []string
	for _, tag := range tags {
		trimmed := strings.TrimSpace(tag)
		if trimmed != "" {
			filteredTags = append(filteredTags, trimmed)
		}
	}
	return filteredTags, nil
}

func deleteTagFromImage(db *sql.DB, sha256 string, tagToRemove string) error {
	var tagStr string
	err := db.QueryRow("SELECT tags FROM images WHERE sha256 = ?", sha256).Scan(&tagStr)
	if err != nil {
		return err
	}

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

				// All Thumbnails are .jpg
				thumbnailFileName := strings.TrimSuffix(name, filepath.Ext(name)) + ".jpg"
				if strings.HasSuffix(lower, ".webm") {
					thumbnailFileName = strings.TrimSuffix(name, filepath.Ext(name)) + ".jpg"
				}

				thumbnailURL := "/thumbnails/" + thumbnailFileName

				if _, err := os.Stat(filepath.Join("/thumbnails", thumbnailFileName)); os.IsNotExist(err) {
					log.Printf("Thumbnail %s not found for %s\n", thumbnailFileName, name)
				}

				meta := ImageMeta{
					Name:         name,
					SHA256:       sha,
					Tags:         []string{},
					ThumbnailURL: thumbnailURL,
				}
				metas = append(metas, meta)
			}
		}
	}
	return metas
	// for _, e := range entries {
	// 	if !e.IsDir() {
	// 		name := e.Name()
	// 		lower := strings.ToLower(name)
	// 		ext := filepath.Ext(lower)
	// 		if ext == ".png" || ext == ".jpg" || ext == ".jpeg" || ext == ".gif" ||
	// 			ext == ".webm" || ext == ".webp" {

	// 			sha := getShaForFile(filepath.Join(dir, name))
	// 			meta := ImageMeta{
	// 				Name:   name,
	// 				SHA256: sha,
	// 				Tags:   []string{},
	// 			}
	// 			metas = append(metas, meta)
	// 		}
	// 	}
	// }
	// return metas
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

func getAllImageMeta(db *sql.DB) ([]ImageMeta, error) {

	rows, err := db.Query("SELECT name, sha256, tags FROM images")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var images []ImageMeta

	localMetas := getImagesPath(imageRootPath)

	imageMap := make(map[string]ImageMeta)
	for _, lm := range localMetas {
		imageMap[lm.SHA256] = lm
	}

	for rows.Next() {
		var name, sha256, tagStr string
		if err := rows.Scan(&name, &sha256, &tagStr); err != nil {
			return nil, err
		}

		var tags []string
		if tagStr != "" {
			// Tags are stored as x,y,z
			// comma separated
			split := strings.Split(tagStr, ",")
			for _, t := range split {
				trimmed := strings.TrimSpace(t)
				if trimmed != "" {
					tags = append(tags, trimmed)
				}
			}
		}

		if meta, ok := imageMap[sha256]; ok {
			images = append(images, ImageMeta{
				Name:         meta.Name,
				SHA256:       meta.SHA256,
				Tags:         tags,
				ThumbnailURL: meta.ThumbnailURL,
			})
		} else {
			log.Printf("Image with SHA256 %s (%s) found in DB but not on disk. Skipping.\n", sha256, name)
		}
	}
	return images, nil
}

// 	var images []ImageMeta
// 	for rows.Next() {
// 		var name, sha256, tagStr string
// 		if err := rows.Scan(&name, &sha256, &tagStr); err != nil {
// 			return nil, err
// 		}
// 		var tags []string
// 		if tagStr != "" {
// 			tags = strings.Split(tagStr, ",")
// 		}
// 		images = append(images, ImageMeta{Name: name, SHA256: sha256, Tags: tags})
// 	}
// 	return images, nil
// }

func renameAllFilesAndUpdateDB(db *sql.DB, root string) error {
	files := getImagesPath(root)

	var timeList []int64
	for range len(files) {
		timeList = append(timeList, generateUnixFakeTime())
	}
	slices.Sort(timeList)

	thumbDir := filepath.Join(root, "thumbnails")
	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		return fmt.Errorf("failed to create thumbnails directory: %w", err)
	}
	for i, img := range files {
		oldPath := filepath.Join(root, img.Name)
		ext := filepath.Ext(img.Name)
		timestamp := strconv.FormatInt(timeList[i], 10)
		newName := timestamp + ext
		newPath := filepath.Join(root, newName)

		if img.Name == newName {
			continue
		}

		err := os.Rename(oldPath, newPath)
		if err != nil {
			log.Printf("Failed to rename %s to %s: %v", img.Name, newName, err)
			continue
		}

		oldThumb := filepath.Join(thumbDir, strings.TrimSuffix(img.Name, filepath.Ext(img.Name))+".jpg")
		newThumb := filepath.Join(thumbDir, timestamp+".jpg")

		if _, err := os.Stat(oldThumb); err == nil {
			if err := os.Rename(oldThumb, newThumb); err != nil {
				log.Printf("Warning: couldn't rename thumbnail for %s: %v", img.Name, err)
			}
		} else {
			if err := generateThumbnail(newPath, newThumb); err != nil {
				log.Printf("Thumbnail generation failed for %s: %v", newPath, err)
			}
		}

		_, err = db.Exec("UPDATE images SET name = ? WHERE name = ?", newName, img.Name)
		if err != nil {
			log.Printf("DB update failed for %s: %v", newName, err)
		}
	}

	return nil
}

func main() {
	db := initDB("images.db")
	defer db.Close()
	images := getImagesPath(imageRootPath)
	storeImages(db, images)

	fs := http.FileServer(http.Dir("src/web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	http.HandleFunc("/images-meta", func(w http.ResponseWriter, r *http.Request) {
		images, err := getAllImageMeta(db)
		if err != nil {
			http.Error(w, "Failed to get image metadata", http.StatusInternalServerError)
			log.Println("Error getting image metadata:", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(images)
	})

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

	http.HandleFunc("/add-tag", func(w http.ResponseWriter, r *http.Request) {
		sha := r.URL.Query().Get("sha")
		tag := r.URL.Query().Get("tag")
		if sha == "" || tag == "" {
			http.Error(w, "Missing params", http.StatusBadRequest)
			return
		}
		tags, err := getTagsForImage(db, sha)
		fmt.Println("/add-tag tags: ", tags)
		if err != nil {
			http.Error(w, "Image not found", http.StatusNotFound)
			return
		}
		for _, t := range tags {
			if t == tag {
				w.WriteHeader(http.StatusNoContent)
				return // already present
			}
		}
		tags = append(tags, tag)
		updateImageTags(db, sha, tags)
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/delete-tag", func(w http.ResponseWriter, r *http.Request) {
		sha := r.URL.Query().Get("sha")
		tag := r.URL.Query().Get("tag")
		if sha == "" || tag == "" {
			http.Error(w, "Missing params", http.StatusBadRequest)
			return
		}
		err := deleteTagFromImage(db, sha, tag)
		if err != nil {
			http.Error(w, "Could not delete tag", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	http.HandleFunc("/rename-all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		err := renameAllFilesAndUpdateDB(db, imageRootPath)
		if err != nil {
			http.Error(w, "Failed to rename", http.StatusInternalServerError)
			log.Println("Rename error:", err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	fsThumbnails := http.FileServer(http.Dir(filepath.Join(imageRootPath, "thumbnails")))
	http.Handle("/thumbnails/", http.StripPrefix("/thumbnails/", fsThumbnails))

	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir(imageRootPath))))

	log.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
