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
