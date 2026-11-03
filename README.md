# Cells

Aplikasi daftar tugas dengan pengingat melalui email dan Discord, menggunakan React, Go, dan PostgreSQL.

## Fungsi aplikasi

- Menambah, mengubah, menghapus, dan menandai tugas selesai.
- Mengatur kategori, prioritas, catatan, dan tanggal jatuh tempo.
- Mencari serta memfilter tugas dan melihat progres penyelesaian.
- Menyimpan tugas dan pengaturan secara terpisah untuk setiap akun.
- Mengirim pengingat melalui email atau Discord Webhook, secara manual maupun terjadwal setiap hari dalam zona WIB.
- Mengatur template pesan, melihat pratinjau, dan mengirim pesan tes. Email mendukung teks biasa dan HTML.
- Menyimpan beberapa penerima email dengan pemisah `;`, misalnya `123@gmail.com; 234@gmail.com`.

## Cara menjalankan

Siapkan Docker dengan Docker Compose. Di Windows, jalankan Docker Desktop terlebih dahulu.
Buka terminal di folder proyek.

### 1. Siapkan konfigurasi

Salin `.env.example` menjadi `.env`. Jika `.env` sudah ada, gunakan file tersebut.

Windows (CMD atau PowerShell):

```text
copy .env.example .env
```

Linux atau macOS:

```bash
cp .env.example .env
```

Isi `DB_PASSWORD` di `.env`. `WEB_PORT` menentukan port aplikasi, dengan nilai awal `8080`.

### 2. Jalankan aplikasi

```bash
docker compose up -d --build
```

Buka [http://localhost:8080](http://localhost:8080), atau sesuaikan dengan `WEB_PORT`.
Saat pertama dijalankan, aplikasi membuat akun dengan username **admin** dan password **admin**.

Ganti password akun awal dengan password pilihan Anda, minimal 8 karakter:

```bash
docker compose exec backend /app/server passwd admin "passwordBaruAnda"
```

Untuk membuat akun tambahan:

```bash
docker compose exec backend /app/server adduser budi "passwordAnda123" "Budi Santoso"
```

### 3. Aktifkan notifikasi bila diperlukan

Untuk email, isi `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, dan `SMTP_FROM_NAME` di `.env` sesuai mailbox Anda. `NOTIFY_RECIPIENTS` menentukan penerima awal. Contoh konfigurasi tersedia di `.env.example`.

Terapkan perubahan konfigurasi:

```bash
docker compose up -d backend
```

Buka **Setting → Notify → Notify By Mail**, isi penerima dengan pemisah `;`, lalu simpan. Atur pesan dan jadwal, atau gunakan tombol kirim tes.

Untuk Discord, buat webhook melalui **Pengaturan channel → Integrations → Webhooks**. Masukkan URL di **Setting → Notify → Notify By Discord Webhook**, lalu simpan dan kirim tes.

Server harus tetap berjalan agar pengingat terjadwal dapat dikirim.

### 4. Jalankan melalui Cloudflare Tunnel (opsional)

Untuk membuka aplikasi melalui `https://todo.mikanm.me`, buat Cloudflare Tunnel di dashboard Cloudflare, lalu salin token konektornya ke `.env`:

```dotenv
CLOUDFLARE_TUNNEL_TOKEN=isi-token-tunnel-anda
```

Tambahkan published application route pada tunnel tersebut:

- Hostname: `todo.mikanm.me` (subdomain `todo`, domain `mikanm.me`).
- Service: **HTTP**, URL **`web:80`**.

Alamat tujuan lengkapnya `http://web:80`, yaitu service web di jaringan Docker yang sama. Pastikan password akun awal sudah diganti sebelum membuka akses publik.

Jalankan aplikasi beserta tunnel:

```bash
docker compose -f docker-compose.yml -f docker-compose.cloudflare.yml up -d --build
```

Lihat status koneksi tunnel:

```bash
docker compose -f docker-compose.yml -f docker-compose.cloudflare.yml logs --tail=50 cloudflared
```

Setelah mengganti token di `.env`, jalankan kembali perintah `up -d` dengan kedua file Compose. Untuk menghentikan aplikasi beserta tunnel:

```bash
docker compose -f docker-compose.yml -f docker-compose.cloudflare.yml down
```

Token hanya disimpan di `.env`, yang sudah diabaikan Git. Cara membuat tunnel dan rute tersedia di [panduan Cloudflare](https://developers.cloudflare.com/tunnel/get-started/).

### Memperbarui kode dan menghentikan aplikasi

Setelah mengubah frontend:

```bash
docker compose up -d --build --no-deps web
```

Setelah mengubah backend:

```bash
docker compose up -d --build backend
```

Untuk menghentikan aplikasi:

```bash
docker compose down
```

Data tetap tersimpan di volume database selama volume tersebut tidak dihapus.
