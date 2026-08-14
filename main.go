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

// Inisialisasi Koneksi DB: Otomatis buat Database & Tabel jika belum ada
func initDBConnection() (*sql.DB, error) {
	rawURL := os.Getenv("MYSQL_URL")
	if rawURL == "" {
		rawURL = os.Getenv("DATABASE_URL")
	}

	var dbUser, dbPass, dbHost, dbPort, dbName string

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
				dbName = hostPortDb[slashIdx+1:]

				colonIdx := strings.Index(userPass, ":")
				if colonIdx != -1 {
					dbUser = userPass[:colonIdx]
					dbPass = userPass[colonIdx+1:]
				} else {
					dbUser = userPass
				}

				colonHostIdx := strings.Index(hostPort, ":")
				if colonHostIdx != -1 {
					dbHost = hostPort[:colonHostIdx]
					dbPort = hostPort[colonHostIdx+1:]
				} else {
					dbHost = hostPort
					dbPort = "3306"
				}
			}
		}
	}

	if dbUser == "" {
		dbUser = getEnv("MYSQLUSER", getEnv("MYSQL_USER", getEnv("DB_USER", "root")))
		dbPass = getEnv("MYSQLPASSWORD", getEnv("MYSQL_PASSWORD", getEnv("DB_PASSWORD", "")))
		dbHost = getEnv("MYSQLHOST", getEnv("MYSQL_HOST", getEnv("DB_HOST", "127.0.0.1")))
		dbPort = getEnv("MYSQLPORT", getEnv("MYSQL_PORT", getEnv("DB_PORT", "3306")))
		dbName = getEnv("MYSQLDATABASE", getEnv("MYSQL_DATABASE", getEnv("DB_NAME", "toko_bangunan")))
	}

	if dbName == "" {
		dbName = "toko_bangunan"
	}

	// 1. Hubungi Server MySQL tanpa DB Name untuk otomatis membuat database jika belum ada
	serverDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/?parseTime=true", dbUser, dbPass, dbHost, dbPort)
	log.Println("Memeriksa Server MySQL di:", dbHost+":"+dbPort)

	if serverDB, err := sql.Open("mysql", serverDSN); err == nil {
		if _, errCreate := serverDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`;", dbName)); errCreate != nil {
			log.Println("Warning: Gagal mengeksekusi CREATE DATABASE:", errCreate)
		} else {
			log.Printf("Database '%s' dipastikan siap di server MySQL.", dbName)
		}
		serverDB.Close()
	}

	// 2. Hubungkan ke database target
	targetDSN := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true", dbUser, dbPass, dbHost, dbPort, dbName)
	return sql.Open("mysql", targetDSN)
}

// Struktur Data Produk
type Product struct {
	ID    int     `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	UserID int `json:"user_id"`
}

//  Struktur Data Users
type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password,omitempty"`
}

var db *sql.DB

func main() {
	var err error

	// Buka Koneksi & Buat DB Otomatis
	db, err = initDBConnection()
	if err != nil {
		log.Println("Gagal sql.Open:", err)
	} else {
		if err = db.Ping(); err != nil {
			log.Println("Warning: Belum dapat terhubung ke MySQL:", err)
		} else {
			fmt.Println("Berhasil terhubung ke database MySQL!")

			// Otomatis buat tabel users jika belum ada
			createUsersTable := `
			CREATE TABLE IF NOT EXISTS users (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				email VARCHAR(255) NOT NULL UNIQUE,
				password VARCHAR(255) NOT NULL
			);`
			if _, errUsers := db.Exec(createUsersTable); errUsers != nil {
				log.Println("Warning: Gagal membuat tabel users:", errUsers)
			} else {
				log.Println("Tabel 'users' terverifikasi & siap digunakan.")
			}

			// Otomatis buat tabel products dengan relasi user_id jika belum ada
			createProductsTable := `
			CREATE TABLE IF NOT EXISTS products (
				id INT AUTO_INCREMENT PRIMARY KEY,
				name VARCHAR(255) NOT NULL,
				price DOUBLE NOT NULL,
				user_id INT,
				FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
			);`
			if _, errTable := db.Exec(createProductsTable); errTable != nil {
				log.Println("Warning: Gagal membuat tabel products:", errTable)
			} else {
				log.Println("Tabel 'products' terverifikasi & siap digunakan.")
			}

			// Pastikan kolom id bertipe AUTO_INCREMENT di tabel users & products
			db.Exec("ALTER TABLE users MODIFY id INT AUTO_INCREMENT;")
			db.Exec("ALTER TABLE products MODIFY id INT AUTO_INCREMENT;")

		
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
			"message":         "API is running on Railway!",
		})
	})

	// Bikin data users
	mux.HandleFunc("POST /users", func(w http.ResponseWriter, r *http.Request) {
		var u User
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, "Format request user salah", http.StatusBadRequest)
			return
		}

		res, err := db.Exec("INSERT INTO users (name, email, password) VALUES (?, ?, ?)", u.Name, u.Email, u.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		lastId, _ := res.LastInsertId()
		u.ID = int(lastId)
		u.Password = ""
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(u)
	})

	// Read all users
	mux.HandleFunc("GET /users", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, name, email FROM users")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var users []User
		for rows.Next() {
			var u User
			if err := rows.Scan(&u.ID, &u.Name, &u.Email); err != nil {
				continue
			}
			users = append(users, u)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	})

	// Read User berdasarkan ID
	mux.HandleFunc("GET /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))

		var u User
		err := db.QueryRow("SELECT id, name, email FROM users WHERE id = ?", id).Scan(&u.ID, &u.Name, &u.Email)

		if err == sql.ErrNoRows {
			http.Error(w, "User tidak ditemukan", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(u)
	})

	// Update user (PUT /users/{id})
	mux.HandleFunc("PUT /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))
		var u User
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, "Format request salah", http.StatusBadRequest)
			return
		}
		res, err := db.Exec("UPDATE users SET name = ?, email = ? WHERE id = ?", u.Name, u.Email, id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			http.Error(w, "User tidak ditemukan", http.StatusNotFound)
			return
		}
		u.ID = id
		u.Password = ""
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(u)
	})

	// Delete User (DELETE /users/{id})
	mux.HandleFunc("DELETE /users/{id}", func(w http.ResponseWriter, r *http.Request) {
		id, _ := strconv.Atoi(r.PathValue("id"))

		res, err := db.Exec("DELETE FROM users WHERE id = ?", id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		rowsAffected, _ := res.RowsAffected()
		if rowsAffected == 0 {
			http.Error(w, "User tidak ditemukan", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	// 1. READ ALL (Mendapatkan semua produk)
	mux.HandleFunc("GET /products", func(w http.ResponseWriter, r *http.Request) {
		rows, err := db.Query("SELECT id, name, price, user_id FROM products")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var products []Product
		for rows.Next() {
			var p Product
			if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.UserID); err != nil {
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
		err := db.QueryRow("SELECT id, name, price, user_id FROM products WHERE id = ?", id).Scan(&p.ID, &p.Name, &p.Price, &p.UserID)

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

		res, err := db.Exec("INSERT INTO products (name, price, user_id) VALUES (?, ?, ?)", p.Name, p.Price, p.UserID)
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

		stmt, err := tx.Prepare("INSERT INTO products (name, price, user_id) VALUES (?, ?, ?)")
		if err != nil {
			http.Error(w, "Gagal menyiapkan query: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer stmt.Close()

		for _, p := range products {
			_, err := stmt.Exec(p.Name, p.Price, p.UserID)
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
