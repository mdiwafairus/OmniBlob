# Panduan Migrasi Data Lama ke OmniBlob (Legacy Data Migration Guide)

Dokumen ini ditujukan bagi Administrator atau Pengguna yang mengunduh distribusi OmniBlob dalam bentuk *binary* (seperti `omniblob.exe` atau file `.zip`) dan ingin memindahkan/mensinkronisasi file dari server lama ke sistem penyimpanan OmniBlob.

## Konsep Dasar
OmniBlob bekerja dengan membaca tabel database `binary_file`. OmniBlob memiliki *Background Worker* yang akan otomatis memindahkan (migrasi) file fisik Anda dari folder lama (`legacy_path`) ke struktur *sharding* OmniBlob yang baru (`root_path`), sambil menghitung *checksum* keamanan (SHA-256).

## Langkah 1: Persiapan Konfigurasi (`config.yaml`)
Pastikan file `config.yaml` Anda sudah diatur dengan benar, khususnya pada bagian `storage`, `migration`, dan `database`.

```yaml
storage:
  root_path: "D:/storage_baru"      # Lokasi tujuan (Folder OmniBlob)
  legacy_path: "C:/uploads_lama"    # Lokasi file-file lama Anda saat ini
  sharding_type: "date"             # Metode pembagian folder (rekomendasi: date)

migration:
  enabled: true                     # WAJIB TRUE agar sistem migrasi menyala
  batch_size: 200                   # Jumlah file yang diproses sekaligus
  interval_sec: 3                   # Jeda pengecekan (detik)
  worker_count: 2                   # Jumlah thread/pekerja bersamaan

database:
  # ... Konfigurasi koneksi ke PostgreSQL Anda ...
```

## Langkah 2: Menjalankan "Auto-Discovery" (Sangat Mudah)
Anda **tidak perlu** menginput data satu per satu ke database! OmniBlob sudah dilengkapi dengan alat pelacak otomatis (*Auto-Discovery Scanner*).

Cukup jalankan OmniBlob melalui terminal (Command Prompt / PowerShell) dengan memberikan tambahan argumen `--scan-legacy`:

**Windows:**
```powershell
.\omniblob.exe --scan-legacy
```

**Linux/Mac:**
```bash
./omniblob --scan-legacy
```

**Apa yang akan terjadi?**
1. OmniBlob akan melakukan koneksi ke Database Anda (dan membuatkan tabel secara otomatis jika belum ada).
2. OmniBlob akan merayapi (*crawling*) seluruh isi dari folder `legacy_path` Anda.
3. OmniBlob akan mencatatkan semua file yang ditemukannya ke dalam tabel PostgreSQL `binary_file` dengan status siap migrasi (`flag = '1'`).
4. Setelah selesai, proses *Scanner* akan berhenti sendiri.

## Langkah 3: Mulai Migrasi
Setelah *Auto-Discovery* selesai, Anda tinggal menyalakan OmniBlob secara normal (tanpa argumen).

```powershell
.\omniblob.exe
```

Biarkan menyala. *Background Worker* secara otomatis akan membaca antrean di tabel database dan mulai menyedot file lama Anda ke `root_path`. 

*Selesai! OmniBlob akan menangani sisanya dengan aman.*
