# Cells

**Daftar tugas pribadi dengan pengingat lewat email dan Discord.**

Cells membantu Anda mencatat pekerjaan, menentukan mana yang perlu didahulukan, dan memantau tugas yang sudah selesai. Tugas ditampilkan berdasarkan prioritas agar pekerjaan penting lebih mudah ditemukan.

Cocok untuk mengelola pekerjaan harian, tugas kuliah, atau rencana pribadi. Anda dapat menjalankan Cells di komputer atau server sendiri, lalu mengaksesnya melalui browser.

## Apa yang bisa dilakukan?

### Mengatur tugas harian

Buat tugas dengan judul, catatan, kategori, prioritas, dan tanggal jatuh tempo. Ubah detailnya saat rencana berubah, tandai selesai setelah dikerjakan, atau hapus tugas yang tidak diperlukan.

### Menentukan fokus

Tugas dikelompokkan menjadi prioritas tinggi, sedang, dan rendah. Pencarian dan filter kategori membantu menemukan pekerjaan tertentu, sementara penanda keterlambatan dan progres penyelesaian membantu memantau pekerjaan yang masih tersisa.

### Mendapat pengingat

Kirim ringkasan tugas aktif melalui email atau ke channel Discord menggunakan webhook. Pengiriman dapat dilakukan langsung atau dijadwalkan setiap hari pada jam pilihan Anda dalam zona WIB.

Email mendukung beberapa penerima, teks biasa, dan HTML. Pesan Discord mendukung teks dan Markdown. Masing-masing memiliki pengaturan pesan serta jadwal sendiri.

### Menyesuaikan pesan

Gunakan pesan biasa atau template yang mengambil informasi dari tugas, seperti nama pengguna, jumlah tugas aktif, dan daftar tugas berdasarkan prioritas. Tersedia pratinjau dan pengiriman tes untuk memeriksa pesan sebelum digunakan. Untuk email, tersedia juga template tabel bawaan.

### Memisahkan data setiap akun

Setiap akun memiliki tugas, kategori, dan pengaturan notifikasinya sendiri. Satu instalasi dapat digunakan oleh beberapa akun dengan daftar tugas masing-masing.

## Contoh penggunaan

Mulai hari dengan mencatat pekerjaan dan memberi prioritas pada tugas yang paling penting. Tambahkan kategori seperti **Kuliah**, **Kerja**, atau **Pribadi**, lalu tentukan tanggal jatuh temponya. Setelah selesai, tandai tugas tersebut untuk memperbarui progres.

Jika ingin pengingat rutin, buka **Setting → Notify**, pilih email atau Discord, lalu simpan pesan dan jadwal pengiriman. Misalnya, kirim ringkasan tugas aktif setiap pukul **08:00 WIB**. Server harus tetap berjalan agar pengingat terjadwal dapat dikirim.

## Menjalankan sendiri

Cells menggunakan React, Go, dan PostgreSQL. Cara menjalankan yang disediakan adalah melalui Docker Compose.

1. Siapkan Docker dan Docker Compose; di Windows, jalankan Docker Desktop.
2. Salin `.env.example` menjadi `.env`, lalu isi `DB_PASSWORD`.
3. Jalankan perintah berikut dari folder proyek:

```bash
docker compose up -d --build
```

Buka [http://localhost:8080](http://localhost:8080). Akun awal menggunakan username **admin** dan password **admin**. Ganti password dengan pilihan Anda, minimal 8 karakter:

```bash
docker compose exec backend /app/server passwd admin "passwordBaruAnda"
```

Untuk email, isi konfigurasi SMTP pada `.env`, lalu jalankan `docker compose up -d backend`. Atur penerima di menu Notify dengan pemisah `;`, misalnya `123@gmail.com; 234@gmail.com`. Untuk Discord, simpan URL webhook melalui panel Discord di menu yang sama.

### Akses melalui Cloudflare Tunnel

Branch `cloudflare` menyediakan Compose tambahan untuk akses melalui `todo.mikanm.me`. Isi `CLOUDFLARE_TUNNEL_TOKEN` di `.env` dan arahkan rute hostname tersebut ke service `http://web:80` pada dashboard Cloudflare, lalu jalankan:

```bash
docker compose -f docker-compose.yml -f docker-compose.cloudflare.yml up -d --build
```

Data aplikasi disimpan di volume PostgreSQL. File `.env` berisi konfigurasi pribadi dan tidak ikut disimpan di Git.
