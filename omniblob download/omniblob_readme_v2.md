# OmniBlob v1.0.0

OmniBlob adalah Object Storage & Sync Service on-premise berperforma tinggi untuk mengelola, menyimpan, dan memigrasikan file secara dinamis antar aplikasi, lengkap dengan dashboard monitoring bawaan.

## Daftar Isi

1. [Prasyarat](#prasyarat)
2. [Instalasi & Quick Start](#instalasi--quick-start)
3. [System Requirements](#system-requirements)
4. [Konfigurasi Keamanan](#konfigurasi-keamanan)
5. [Fitur](#fitur)
6. [Monitoring & Log](#monitoring--log)
7. [Panduan Lanjutan](#panduan-lanjutan)

## Prasyarat

Sebelum menjalankan OmniBlob, pastikan hal berikut sudah tersedia:

- **PostgreSQL** (versi minimum: *isi versi yang sudah diuji tim dev*) — digunakan untuk indexing metadata file, log migrasi, dan status lisensi. OmniBlob **tidak akan berjalan** tanpa koneksi database yang valid.
- **Storage backend** — direktori lokal dan/atau NFS mount yang sudah ter-mount dan bisa diakses (baca/tulis) oleh user yang menjalankan OmniBlob.
- **Lisensi produk** — lihat langkah generate di bawah. OmniBlob memvalidasi lisensi saat startup; tanpa lisensi valid, service tidak akan start.

## Lisensi & Masa Percobaan (Trial)

OmniBlob yang diunduh pertama kali berjalan dalam **mode trial selama 3 bulan** sejak lisensi digenerate.

- **Selama masa trial:** seluruh fitur berfungsi penuh, tanpa batasan.
- **Setelah 3 bulan:** OmniBlob **berhenti menerima request baru** (service tidak aktif) sampai lisensi diperpanjang/diaktifkan penuh.
- **Yang PALING PENTING — tidak ada data yang hilang atau dihapus.** Saat trial berakhir, OmniBlob hanya menghentikan *service*-nya (proses aplikasi berhenti melayani), bukan menyentuh data. File yang sudah tersimpan di storage (`root_path`) dan file legacy di `legacy_path` tetap utuh, tidak dihapus, tidak dimodifikasi. Begitu lisensi diperpanjang dan service dinyalakan kembali, seluruh data — termasuk progres migrasi yang tersimpan di PostgreSQL — langsung tersedia kembali seperti sebelum service berhenti.
- **Migrasi yang sedang berjalan saat trial berakhir:** proses migrasi akan berhenti di tengah batch yang sedang diproses (service mati sepenuhnya, bukan cuma fitur migrasi). Ini aman — mengikuti mekanisme checkpoint per-batch yang sama seperti skenario crash/restart biasa (lihat `MIGRATION_GUIDE.md` bagian Ketahanan terhadap Crash): begitu lisensi diperpanjang dan service dinyalakan lagi, migrasi otomatis melanjutkan dari checkpoint batch terakhir, tanpa duplikasi maupun file yang terlewat.
- **Perpanjangan/aktivasi lisensi:** gunakan kembali alur `license-gen` → `license-verify` dengan lisensi baru/lisensi berbayar untuk mengaktifkan kembali service.

## Instalasi & Quick Start

Ikuti urutan berikut — melewati salah satu langkah akan menyebabkan service gagal start atau tidak bisa diakses aplikasi klien.

### Langkah 1 — Generate & Verifikasi Lisensi

```bash
./omniblob key-gen
./omniblob license-gen --key <path-to-key>
./omniblob license-verify --license <path-to-license>
```

### Langkah 2 — Siapkan Database

Buat database PostgreSQL kosong dan catat kredensialnya untuk langkah berikutnya.

### Langkah 3 — Konfigurasi `configs/config.yaml`

Salin dari `configs/config.example.yaml` (jika tersedia) lalu sesuaikan minimal bagian berikut:

```yaml
database:
  host: localhost
  port: 5432
  name: omniblob
  user: omniblob_user
  password: ********

root_path: /data/omniblob-storage

clients:
  - name: fe-app
    auth_type: api_key
    api_key: ********
  - name: be-app
    auth_type: basic
    username: be-app
    password: ********
```

Lihat bagian [Konfigurasi Keamanan](#konfigurasi-keamanan) untuk mengatur pembatasan akses jaringan sebelum menjalankan service di lingkungan produksi.

### Langkah 4 — Jalankan Service

**Windows:**
```cmd
omniblob.exe --config configs/config.yaml
```

**Linux:**
```bash
./omniblob --config configs/config.yaml
```

### Langkah 5 — Verifikasi Service Berjalan

Buka dashboard admin:

```
http://localhost:<PORT>
```

Port sesuai nilai `dashboard.port` di `config.yaml`.

## System Requirements

| Komponen | Minimum | Catatan |
|---|---|---|
| **Windows** | Windows 7 / Server 2008 R2 | ⚠️ OS ini sudah *end-of-life* dan tidak lagi menerima security patch dari Microsoft. Menjalankan OmniBlob di OS ini menambah risiko keamanan di level sistem operasi, terlepas dari keamanan OmniBlob itu sendiri. Gunakan hanya jika benar-benar diperlukan untuk kompatibilitas lingkungan legacy, dan pastikan mesin terisolasi secara jaringan. |
| **Linux** | Kernel 2.6.32+ | Kompatibel dengan Ubuntu, Debian, CentOS, RHEL, dll. |
| **RAM** | 512 MB | Cukup untuk beban ringan/single-client. Untuk migrasi skala besar atau `worker_count` mendekati maksimum (8), disarankan minimal 1–2 GB tergantung ukuran file rata-rata yang diproses. |
| **Arsitektur** | 32-bit & 64-bit | *Catatan: konfirmasi batas ukuran file maksimum yang didukung pada arsitektur 32-bit sebelum digunakan untuk file berukuran sangat besar.* |

## Konfigurasi Keamanan

OmniBlob dirancang untuk bisa menjaga keamanannya sendiri di level aplikasi, tanpa bergantung penuh pada konfigurasi firewall/infra eksternal. Atur mode berikut sesuai lingkungan deployment Anda.

```yaml
security:
  mode: direct_socket        # direct_socket (air-gapped/internal) atau behind_proxy (internet-facing)
  allowed_socket_ips:
    - 10.0.0.0/24             # mendukung format CIDR
    - 192.168.1.5
  auth_type: both             # ip_only, token_only, atau both
```

- **`direct_socket`** — gunakan untuk server tertutup/air-gapped tanpa reverse proxy di depannya. Koneksi difilter langsung di level socket (Layer 4) sebelum masuk ke HTTP handler.
- **`behind_proxy`** — gunakan jika OmniBlob berada di belakang reverse proxy/WAF. `allowed_socket_ips` diisi IP/CIDR milik proxy tersebut, bukan IP klien akhir.
- Header `X-Forwarded-For` hanya dibaca untuk keperluan audit log, dan **hanya dipercaya jika koneksi berasal dari IP yang terdaftar** di `allowed_socket_ips`.

> **Rekomendasi keamanan tambahan** (di luar cakupan konfigurasi ini, tapi penting): jika `mode: behind_proxy` digunakan, pastikan juga ada pembatasan di level jaringan/firewall agar server OmniBlob tidak bisa diakses langsung dari luar tanpa melalui proxy tersebut. Whitelist IP di level aplikasi adalah lapisan kedua, bukan pengganti isolasi jaringan.

## Fitur

- **Dynamic Configuration** lewat `configs/config.yaml` — tidak perlu rebuild untuk mengubah pengaturan.
- **Legacy File Migration** ke struktur folder terstandarisasi (`{app}/{year}/{month}/{uuid}/{filename}`) — lihat `MIGRATION_GUIDE.md` untuk detail lengkap.
- **Migration Toggle** — aktifkan/nonaktifkan proses migrasi kapan saja tanpa restart service.
- **Checkpoint & Resume** — migrasi aman di-restart kapan saja tanpa duplikasi maupun kehilangan progres di luar batch yang sedang berjalan.
- **Multi-Client Authentication** — API Key, Basic Auth, atau Custom Header per aplikasi klien.
- **HTTP Range Requests** — mendukung resume download dan streaming media.
- **On-the-fly Checksum** — verifikasi integritas file saat transfer.
- **IP Whitelist berbasis CIDR + Rate Limiting + Dynamic Blocklist** — proteksi jaringan bawaan tanpa bergantung infra eksternal.
- **Dashboard Analytics** (in-memory) — status migrasi, breakdown jenis file, deteksi duplikat, dan file tidak terpakai.

> **Catatan:** data dashboard bersifat *in-memory* — riwayat statistik akan ter-reset setiap kali service di-restart. Data migrasi permanen (status per file) tetap tersimpan aman di PostgreSQL; yang hilang saat restart hanya agregat tampilan dashboard-nya.

## Monitoring & Log

- Dashboard: `http://localhost:<PORT>` (lihat `dashboard.port` di config).
- Log file: lokasi sesuai `logging.path` di `config.yaml`.
- Status koneksi yang ditolak (IP tidak diizinkan, autentikasi gagal) dicatat terpisah untuk keperluan audit — lihat `logging.security_audit_path`.

## Panduan Lanjutan

- Migrasi file legacy: lihat `MIGRATION_GUIDE.md`.
- Update/upgrade versi OmniBlob: *(tambahkan bagian ini setelah prosedur upgrade resmi tersedia)*.
