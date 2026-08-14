# Toko Bangunan Backend Service

RESTful API berperforma tinggi yang dibangun menggunakan **Golang** (native `net/http` Go 1.22+) dan **MySQL**. REST API ini mengelola data Pengguna (*Users*) dan Produk (*Products*) dengan dukungan relasi antar-tabel (*One-to-Many*), transaksi database (*Bulk Insert*), serta otomatisasi pembuatan database & tabel (*Auto-Migration*).

---

## ✨ Fitur Utama

- 🚀 **Go 1.22+ Native Routing (`net/http` ServeMux)**: Memanfaatkan HTTP Method Matching (`GET /`, `POST /products`, `GET /products/{id}`).
- 👤 **CRUD Users**: Registrasi, baca data, perbarui profil, dan hapus user.
- 🛒 **CRUD Products**: Manajemen produk (Nama, Harga, `user_id`).
- ⚡ **Bulk Insert Transaction (`POST /products/bulk`)**: Menggunakan **MySQL Transaction** (`db.Begin`, `tx.Commit`, `tx.Rollback`) untuk penginputan data massal secara *atomic*.
- 🛠️ **Auto Database & Table Creation**: Otomatis membuat database `toko_bangunan` dan tabel `users` & `products` saat pertama kali dijalankan.
- 🌐 **Cross-Environment Ready**: Kompatibel secara otomatis di lingkungan **Localhost (Laragon)** maupun **Cloud Deployment (Railway.app)**.
- 💓 **Health Check Endpoint (`GET /`)**: Endpoint pemantau status aktif server & status koneksi database.

---

## 🛠️ Teknologi & Tools

- **Bahasa Pemrograman:** [Golang 1.22+](https://go.dev/)
- **Database:** MySQL
- **Driver Database:** `github.com/go-sql-driver/mysql`
- **Testing API:** Postman / Insomnia / cURL
- **Deployment Platform:** [Railway.app](https://railway.app/)

---

## 📌 Dokumentasi Endpoint API

### 1. Health Check
| Method | Endpoint | Deskripsi |
| :--- | :--- | :--- |
| `GET` | `/` | Mengecek status aktif server & koneksi database |

### 2. Management Users
| Method | Endpoint | Deskripsi | Status Code |
| :--- | :--- | :--- | :--- |
| `GET` | `/users` | Mengambil seluruh data user | `200 OK` |
| `GET` | `/users/{id}` | Mengambil 1 data user berdasarkan ID | `200 OK` / `404` |
| `POST` | `/users` | Menambahkan user baru | `201 Created` |
| `PUT` | `/users/{id}` | Mengubah data user berdasarkan ID | `200 OK` / `404` |
| `DELETE` | `/users/{id}` | Menghapus user berdasarkan ID | `204 No Content` |

### 3. Management Products
| Method | Endpoint | Deskripsi | Status Code |
| :--- | :--- | :--- | :--- |
| `GET` | `/products` | Mengambil seluruh data produk | `200 OK` |
| `GET` | `/products/{id}` | Mengambil 1 data produk berdasarkan ID | `200 OK` / `404` |
| `POST` | `/products` | Menambahkan 1 produk baru | `201 Created` |
| `POST` | `/products/bulk` | Menambahkan banyak produk sekaligus (Atomic) | `201 Created` |
| `PUT` | `/products/{id}` | Mengubah data produk berdasarkan ID | `200 OK` / `404` |
| `DELETE` | `/products/{id}` | Menghapus produk berdasarkan ID | `204 No Content` |

---

## 📝 Contoh Request Body (JSON)

### A. Tambah User (`POST /users`)
```json
{
  "name": "Budi Santoso",
  "email": "budi@gmail.com",
  "password": "secretpassword"
}
```

### B. Tambah Produk (`POST /products`)
```json
{
  "name": "Semen Tiga Roda 50kg",
  "price": 75000,
  "user_id": 1
}
```

### C. Tambah Produk Massal (`POST /products/bulk`)
```json
[
  {
    "name": "Cat Tembok 5kg",
    "price": 120000,
    "user_id": 1
  },
  {
    "name": "Paku Kayu 2 Inch",
    "price": 20000,
    "user_id": 1
  }
]
```

---

## 💻 Panduan Jalankan di Local (Localhost)

### 1. Prasyarat
- [Go 1.22+](https://go.dev/dl/) sudah terinstal di komputer.
- MySQL Server (Laragon / XAMPP / Native MySQL) sudah menyala di port `3306`.

### 2. Clone Repository
```bash
git clone https://github.com/Rerey155-del/REST_API-golang.git
cd REST_API-golang
```

### 3. Jalankan Aplikasi
```bash
go run main.go
```
*Aplikasi akan otomatis terhubung ke MySQL local, membuat database `toko_bangunan` dan tabel `users` & `products`, lalu berjalan di `http://localhost:8080`.*

## 🚀 Deployment ke Railway.app

1. Push repository ini ke akun GitHub Anda.
2. Login ke [Railway.app](https://railway.app/) dan buat **New Project**.
3. Tambahkan service **MySQL** (Provision MySQL).
4. Tambahkan service dari **GitHub Repo** ini.
5. Hubungkan variabel MySQL ke service Golang (Variable Reference).
6. Generate Domain Publik di bagian **Settings -> Networking**.

---

## 📄 Lisensi

Distributed under the MIT License. See `LICENSE` for more information.
