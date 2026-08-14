package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	// Driver MySQL
	_ "github.com/go-sql-driver/mysql"
)

// Helper untuk membaca Environment Variable dengan nilai default (fallback)
func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// Helper untuk membuat DSN MySQL (Mendukung MYSQL_URL, DATABASE_URL, dan variabel terpisah)
func getConnStr() string {
	rawURL := os.Getenv("MYSQL_URL")
	if rawURL == "" {
		rawURL = os.Getenv("DATABASE_URL")
	}

	if rawURL != "" {
		u := strings.TrimPrefix(rawURL, "mysql://")
		u = strings.TrimPrefix(u, "mysql2://")
		lastAt := strings.LastIndex(u, "@")
		if lastAt != -1 {
			userPass := u[:lastAt]
			hostPortDb := u[lastAt+1:]
			slashIdx := strings.Index(hostPortDb, "/")
			if slashIdx != -1 {
				hostPort := hostPortDb[:slashIdx]
				dbName := hostPortDb[slashIdx+1:]
				return fmt.Sprintf("%s@tcp(%s)/%s?parseTime=true", userPass, hostPort, dbName)
			}
		}
	}

	dbUser := getEnv("MYSQLUSER", getEnv("MYSQL_USER", getEnv("DB_USER", "root")))
	dbPass := getEnv("MYSQLPASSWORD", getEnv("MYSQL_PASSWORD", getEnv("DB_PASSWORD", "")))
	dbHost := getEnv("MYSQLHOST", getEnv("MYSQL_HOST", getEnv("DB_HOST", "127.0.0.1")))
	dbPort := getEnv("MYSQLPORT", getEnv("MYSQL_PORT", getEnv("DB_PORT", "3306")))
	dbName := getEnv("MYSQLDATABASE", getEnv("MYSQL_DATABASE", getEnv("DB_NAME", "toko_bangunan")))

	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPass, dbHost, dbPort, dbName)
}

// Struktur Data Produk
type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
}

var db *sql.DB

func main() {
	var err error

	connStr := getConnStr()
	log.Println("Menggunakan string koneksi DB:", connStr)

	// Buka Koneksi
	db, err = sql.Open("mysql", connStr)
	if err != nil {
		log.Println("Gagal sql.Open:", err)
	} else {
		if err = db.Ping(); err != nil {
			log.Println("Warning: Belum dapat terhubung ke MySQL:", err)
		} else {
			fmt.Println("Berhasil terhubung ke database MySQL!")
			// Otomatis buat tabel products jika belum ada di database
			createTableQuery := `
			CREATE TABLE IF NOT EXISTS products (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				price DOUBLE NOT NULL
			);`
			if _, errTable := db.Exec(createTableQuery); errTable != nil {
				log.Println("Warning: Gagal membuat tabel products:", errTable)
			} else {
				log.Println("Tabel 'products' terverifikasi & siap digunakan.")
			}
		}
	}

	// Inisialisasi Router (Fitur Go 1.22+)
	mux := http.NewServeMux()

	// ----------------------------------------------------
	// ENDPOINT CRUD
	// ----------------------------------------------------

	// 0. HOME / HEALTH CHECK
	mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		dbStatus := "connected"
		if db == nil || db.Ping() != nil {
			dbStatus = "disconnected / checking..."
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":          "online",
			"database_status": dbStatus,
			"message":         "REST API Golang is running on Railway!",
		})
	})

	// 1. READ ALL (Mendapatkan semua produk)
	mux.HandleFunc("GET /products", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, name, price FROM products")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var products []Product
		for rows.Next() {
			var p Product
			if err := rows.Scan(&p.ID, &p.Name, &p.Price); err != nil {
				continue
			}
			products = append(products, p)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(products)
	})

	// 2. READ BY ID (Mendapatkan 1 produk berdasarkan ID)
	mux.HandleFunc("GET /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))

		var p Product
		err := db.QueryRow("SELECT id, name, price FROM products WHERE id = ?", id).Scan(&p.ID, &p.Name, &p.Price)

		if err == sql.ErrNoRows {
			http.Error(w, "Produk tidak ditemukan", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	})

	// 3. CREATE (Menambah produk baru)
	mux.HandleFunc("POST /products", func(w http.ResponseWriter, r *http.Request) {
		var p Product
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Format request salah", http.StatusBadRequest)
			return
		}

		res, err := db.Exec("INSERT INTO products (name, price) VALUES (?, ?)", p.Name, p.Price)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		lastId, _ := res.LastInsertId()
		p.ID = int(lastId)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(p)
	})

	mux.HandleFunc("POST /products/bulk", func(w http.ResponseWriter, r *http.Request) {
		var products []Product

		if err := json.NewDecoder(r.Body).Decode(&products); err != nil {
			http.Error(w, "Format request salah: "+err.Error(), http.StatusBadRequest)
			return
		}

		if len(products) == 0 {
			http.Error(w, "Data produk tidak boleh kosong (minimal 1 data)", http.StatusBadRequest)
			return
		}

		// Gunakan Transaction agar semua data berhasil atau tidak sama sekali (atomic)
		tx, err := db.Begin()
		if err != nil {
			http.Error(w, "Gagal memulai transaksi: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		stmt, err := tx.Prepare("INSERT INTO products (name, price) VALUES (?, ?)")
		if err != nil {
			http.Error(w, "Gagal menyiapkan query: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer stmt.Close()

		for _, p := range products {
			_, err := stmt.Exec(p.Name, p.Price)
			if err != nil {
				http.Error(w, fmt.Sprintf("Gagal menyimpan produk '%s': %s", p.Name, err.Error()), http.StatusInternalServerError)
				return
			}
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Gagal menyimpan transaksi: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"status": "success", "message": "Data bulk berhasil ditambahkan"})
	})

	// 4. UPDATE (Mengubah produk yang sudah ada)
	mux.HandleFunc("PUT /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		var p Product

		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "Format request salah", http.StatusBadRequest)
			return
		}

		res, err := db.Exec("UPDATE products SET name = ?, price = ? WHERE id = ?", p.Name, p.Price, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			http.Error(w, "Produk tidak ditemukan", http.StatusNotFound)
			return
		}

		p.ID = id
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(p)
	})

	// 5. DELETE (Menghapus produk)
	mux.HandleFunc("DELETE /products/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))

		res, err := db.Exec("DELETE FROM products WHERE id = ?", id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			http.Error(w, "Produk tidak ditemukan", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	})

	// Jalankan Server (Mendukung port dinamis dari Railway/Heroku atau default 8080)
	port := getEnv("PORT", "8080")
	fmt.Printf("Server API berjalan di port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}
