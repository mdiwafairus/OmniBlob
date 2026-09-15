# OmniBlob Migration Guide

> Dokumen ini menjelaskan cara memindahkan file dari storage lama (legacy) ke omniBlob. Pastikan Anda sudah menyelesaikan setup awal (lisensi, database, autentikasi) di `README.md` sebelum mengikuti panduan ini.

## 1. Sebelum Memulai — Baca Ini Dulu

**Data Anda aman.** OmniBlob melakukan **non-destructive copy** — file di `legacy_path` **tidak pernah dihapus otomatis**, baik selama maupun setelah migrasi. File lama tetap ada di lokasinya sampai Anda menghapusnya sendiri secara manual.

**Konsekuensi ruang disk.** Karena file lama tidak dihapus, selama file berada di kedua lokasi (lama + baru), kebutuhan disk space Anda **efektif dua kali lipat** dari volume data yang dimigrasi. Pastikan kapasitas storage baru mencukupi sebelum memulai migrasi skala besar.

**Fitur ganda: migrasi atau distribusi.** Jika Anda tidak mengaktifkan migrasi (`migration.enabled: false`), OmniBlob tetap bisa digunakan sebagai layanan distribusi file terstruktur untuk file-file baru — tanpa menyentuh file legacy sama sekali.

## 2. Prasyarat

- PostgreSQL sudah terpasang dan dapat diakses (lihat `README.md` bagian Prasyarat).
- Lisensi sudah digenerate dan terverifikasi (`license-verify` berhasil).
- Anda tahu lokasi `legacy_path` (storage lama) dan sudah menghitung estimasi ukurannya (`du -sh /path/legacy`).

## 3. Konfigurasi Dasar

Di `configs/config.yaml`:

```yaml
legacy_path: /data/legacy-storage
root_path: /data/omniblob-storage

migration:
  enabled: true
  worker_count: 4      # default: 2, maksimum: 8 (lihat catatan di bawah)
  batch_size: 200       # default: 200
```

### Tentang `worker_count`

| Nilai | Perilaku |
|---|---|
| Tidak diisi | Default `2` |
| 1–8 | Dipakai sesuai input |
| >8 (mis. 100) | Otomatis dipaksa turun ke `8` |

Batas maksimum `8` sengaja dikunci untuk mencegah beban baca/tulis disk berlebihan yang bisa membuat server hang. Sesuaikan nilai lebih rendah dari 8 jika server Anda juga menjalankan proses lain yang sensitif terhadap I/O disk (mis. database produksi di mesin yang sama).

### Tentang `batch_size`

Migrasi diproses per-batch, bukan per-file. Checkpoint (progres yang tersimpan) hanya maju setelah **seluruh file dalam satu batch selesai diproses** (baik sukses maupun gagal). Implikasinya:

- Batch lebih besar → checkpoint lebih jarang ditulis, tapi jika terjadi crash, sistem akan mengecek ulang seluruh batch tersebut saat restart (lebih lama, walau tidak menduplikasi data — lihat bagian Ketahanan terhadap Crash).
- Batch lebih kecil → checkpoint lebih sering, radius pengulangan saat crash lebih kecil.

## 4. Struktur Folder Hasil Migrasi

File yang berhasil dimigrasi akan mengikuti pola:

```
{app}/{year}/{month}/{uuid}/{filename}
```

Contoh: `fe-app/2026/09/3f9a1c22/invoice.pdf`

- `{year}/{month}` diambil dari **tanggal file tersebut pertama kali dibuat di storage lama**, bukan tanggal migrasi dijalankan. Timestamp asli file tidak hilang.
- Setiap file mendapat folder `{uuid}` unik untuk menghindari tabrakan nama antar file yang kebetulan bernama sama.

## 5. Menjalankan Migrasi

### Opsi A — Otomatis (rekomendasi untuk direktori kecil-menengah)

Set `migration.enabled: true`, lalu jalankan OmniBlob seperti biasa. Background worker akan otomatis melakukan scan dan migrasi.

### Opsi B — Discovery Scan Manual (rekomendasi untuk direktori sangat besar)

Jika `legacy_path` berisi jutaan file, jalankan scan awal secara terpisah sebelum mengaktifkan worker migrasi penuh, agar Anda bisa melihat estimasi jumlah file/ukuran total lebih dulu:

```bash
./omniblob --scan-legacy
```

> Catatan: `--scan-legacy` melakukan pendataan (discovery) saja, tidak memindahkan file. Jalankan ini **sebelum** mengaktifkan `migration.enabled: true` untuk direktori besar, agar Anda tahu skala pekerjaan sebelum proses migrasi sesungguhnya berjalan.

### Mengaktifkan/Menonaktifkan Migrasi Tanpa Restart

Migrasi dapat di-pause kapan saja lewat **Migration Toggle** tanpa menghentikan seluruh service — berguna saat maintenance window atau ketika beban I/O server sedang tinggi karena kebutuhan lain.

## 6. Memverifikasi Migrasi Berhasil

Buka dashboard admin di browser:

```
http://localhost:<PORT>
```

(Port sesuai konfigurasi `dashboard.port` di `config.yaml`.)

Dashboard menampilkan:
- Jumlah dan daftar file yang **berhasil** dimigrasi, beserta lokasi folder barunya.
- Tanggal file **dibuat** (timestamp asli) terpisah dari tanggal **dimigrasi**.
- Progres migrasi secara live (persentase batch yang sudah selesai).
- Daftar file yang **di-skip** beserta alasannya (lihat bagian 7).

> **Penting — Keamanan Dashboard:** Akses ke dashboard tunduk pada mekanisme autentikasi dan pembatasan IP yang sama dengan API utama (lihat `README.md` bagian Konfigurasi Keamanan). Jangan mengekspos port dashboard ke jaringan yang lebih luas dari yang seharusnya.

## 7. Penanganan File Bermasalah — Skip & Continue

Jika satu file gagal dimigrasi (mis. file sumber corrupt atau tidak terbaca), OmniBlob **tidak menghentikan seluruh proses migrasi**. File tersebut ditandai gagal dan proses lanjut ke file berikutnya, agar ribuan file sehat lainnya tidak ikut tertahan.

- Daftar file yang di-skip selalu bisa dicek di dashboard beserta alasan kegagalannya.
- File yang di-skip **tidak dihapus atau diubah** di `legacy_path` — Anda bisa menanganinya secara manual kapan saja.
- Jalankan ulang proses migrasi (restart service atau toggle enable/disable) untuk mencoba memigrasi ulang file yang sebelumnya gagal karena sebab sementara (mis. gangguan jaringan NFS sesaat).

## 8. Ketahanan terhadap Crash (Checkpoint & Resume)

OmniBlob mencatat progres migrasi ke tabel `log_file_rsync` di PostgreSQL secara **per-batch**, bukan per-file individual:

- Checkpoint hanya diperbarui setelah **seluruh file dalam satu batch** selesai diproses.
- Jika server crash/restart di tengah sebuah batch, OmniBlob akan mengambil ulang **seluruh batch tersebut** saat menyala kembali — bukan mulai dari nol secara keseluruhan.
- File yang sebenarnya **sudah** berhasil dimigrasi sebelum crash **tidak akan dikopi ulang** — sistem mengenali status file tersebut sudah `Migrated` di database dan langsung melewatinya (proses ini idempotent, hanya makan waktu milidetik per file, tanpa beban I/O disk tambahan).

Artinya: restart aman kapan saja, tanpa risiko duplikasi maupun kehilangan progres di luar batch yang sedang berjalan saat crash terjadi.

> **Catatan lisensi trial:** berakhirnya masa trial (lihat `README.md` bagian Lisensi & Masa Percobaan) menghentikan service dengan cara yang sama seperti restart biasa — bukan crash yang merusak data. Jika migrasi sedang berjalan saat trial habis, cukup perpanjang lisensi dan nyalakan ulang service; migrasi akan melanjutkan dari checkpoint terakhir seperti skenario di atas.

## 9. Troubleshooting

| Gejala | Kemungkinan Penyebab | Langkah |
|---|---|---|
| Migrasi terlihat "macet" (progres tidak naik lama) | Batch besar sedang diproses, atau salah satu worker lambat (mis. file besar/NFS lambat) | Cek progres live di dashboard, bukan hanya checkpoint resmi |
| Disk penuh saat migrasi berjalan | Non-destructive copy butuh 2x ruang | Tambah kapasitas storage baru, atau hapus manual file legacy yang sudah terverifikasi aman |
| Banyak file di-skip dengan alasan sama | Kemungkinan masalah sistemik (mis. NFS mount bermasalah), bukan file individual yang corrupt | Cek koneksi ke `legacy_path`, baru migrasi ulang batch terkait |
| Service tidak start setelah update config | Kesalahan format YAML atau nilai IP/CIDR tidak valid | Cek log startup — service didesain gagal start (bukan jalan dengan config rusak) jika ada entri konfigurasi tidak valid |

Log detail migrasi tersimpan di lokasi yang dikonfigurasi lewat `logging.path` di `config.yaml`.
