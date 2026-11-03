# cells

Aplikasi daftar tugas dengan React, Go, dan PostgreSQL 17.

## Menjalankan

```bash
docker compose up -d --build
```

Buka http://localhost:8080. Akun default dibuat otomatis: username **admin**, password **admin**. Password disimpan sebagai hash bcrypt. Startup berikutnya tidak menimpa password akun yang sudah ada.

Database hanya memakai PostgreSQL. Volume `postgres_data` menyimpan data. Data dari volume MySQL lama tidak otomatis dipindahkan.

## Kelola akun

```bash
docker compose exec backend /app/server adduser budi passwordku123 'Budi Santoso'
docker compose exec backend /app/server passwd admin passwordBaru456
```

Password akun tambahan minimal 8 karakter. Aplikasi saat ini tidak memiliki pembagian peran; admin adalah akun default.

Konfigurasi password database ada di `.env` (`DB_PASSWORD`). Port web memakai `WEB_PORT` (default 8080).
