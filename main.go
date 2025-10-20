package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/daku10/go-lz-string"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"path/filepath"
)

//go:embed index.html
var indexHTML embed.FS

// in-memory store for the save files
var saves = make(map[string][]byte)
var mu sync.Mutex

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	host := os.Getenv("SUGARCUBE_HOST")
	if host == "" {
		host = "0.0.0.0"
	}

	port := os.Getenv("SUGARCUBE_PORT")
	if port == "" {
		port = "8080"
	}

	addr := fmt.Sprintf("%s:%s", host, port)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/", securityHeaders(http.FileServer(http.FS(indexHTML))))
	http.HandleFunc("/api/upload", func(w http.ResponseWriter, r *http.Request) {
		securityHeaders(http.HandlerFunc(uploadHandler)).ServeHTTP(w, r)
	})
	http.HandleFunc("/api/save", func(w http.ResponseWriter, r *http.Request) {
		securityHeaders(http.HandlerFunc(saveHandler)).ServeHTTP(w, r)
	})
	http.HandleFunc("/api/download/", func(w http.ResponseWriter, r *http.Request) {
		securityHeaders(http.HandlerFunc(downloadHandler)).ServeHTTP(w, r)
	})

	fmt.Printf("SugarCube Editor starting on %s...\n", addr)
	http.ListenAndServe(addr, nil)
}

func securityHeaders(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://cdn.tailwindcss.com; style-src 'self' 'unsafe-inline'")
        w.Header().Set("X-Content-Type-Options", "nosniff")
        w.Header().Set("X-Frame-Options", "DENY")
        next.ServeHTTP(w, r)
    })
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	decoded, err := lzstring.DecompressFromBase64(string(body))
	if err != nil {
		http.Error(w, "Failed to decompress data", http.StatusInternalServerError)
		return
	}

	var jsonData interface{}
	if err := json.Unmarshal([]byte(decoded), &jsonData); err != nil {
		http.Error(w, "Failed to parse JSON", http.StatusBadRequest)
		return
	}

	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		http.Error(w, "Failed to format JSON", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(prettyJSON)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	compressed, err := lzstring.CompressToBase64(string(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	id := uuid.New().String()
	mu.Lock()
	saves[id] = []byte(compressed)
	mu.Unlock()

	// a goroutine to delete the save after 1 hour
	go func() {
		time.Sleep(1 * time.Hour)
		mu.Lock()
		delete(saves, id)
		mu.Unlock()
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"id": id})
}

func downloadHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/api/download/"):]

	mu.Lock()
	data, ok := saves[id]
	mu.Unlock()

	if !ok {
		http.Error(w, "File not found or expired", http.StatusNotFound)
		return
	}

	filename := r.URL.Query().Get("filename")
	if filename == "" {
		filename = "edited_save.save"
	} else {
		filename = filepath.Base(filename)
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("Content-Type", "text/plain")
	w.Write(data)
}
