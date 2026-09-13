export type Variant = {
  id: string;
  label: string;
  title: string;
  sub: string;
  note: string;
};

export const variants: Variant[] = [
  {
    id: "01",
    label: "Minimal",
    title: "File pindah. Data tetap.",
    sub: "OmniBlob memindahkan dan menyinkronkan file antar share NFS, SMB, dan disk lokal dari dalam infrastruktur Anda. Setiap byte diverifikasi checksum. Tidak ada yang keluar dari jaringan.",
    note: "Paling hemat kata. Cocok untuk audiens yang sudah tahu masalahnya dan hanya butuh konfirmasi.",
  },
  {
    id: "02",
    label: "Langsung",
    title: "Warisan file server itu bisa dipindahkan. Tanpa menyerahkan kontrol.",
    sub: "Mesin migrasi on-premise untuk NFS, SMB, dan storage lama. Permission dipetakan, transfer berjalan delta dan terjadwal, verifikasi checksum per file, audit log lengkap — tanpa data meninggalkan perimeter Anda.",
    note: "Menyebut mekanisme kerja di sub-headline. Cocok sebagai default landing page.",
  },
  {
    id: "03",
    label: "Bold",
    title: "Server lamanya boleh pensiun. Datanya tidak boleh hilang.",
    sub: "OmniBlob mengurai bertahun-tahun sprawl share: inventarisasi jutaan file, transfer terverifikasi, permission utuh, cutover terencana. Semuanya berjalan di rak server Anda — bukan di tempat lain.",
    note: "Menekan rasa sakit operasional. Cocok untuk kampanye ke tim yang sedang menunda migrasi.",
  },
];

export type TermLine = {
  tag?: string;
  text: string;
  tone: "cmd" | "scan" | "plan" | "xfer" | "file" | "warn" | "ok" | "verify" | "done" | "dim";
};

export const termLines: TermLine[] = [
  { text: "omniblob sync --job finance-migration --verify", tone: "cmd" },
  { tag: "scan", text: "nfs://fs01.internal/export/finance", tone: "scan" },
  { tag: "scan", text: "1.248.512 objek · 847,3 GiB · 9.412 direktori", tone: "scan" },
  { tag: "plan", text: "delta 96,1% — 814,4 GiB akan ditransfer", tone: "plan" },
  { tag: "xfer", text: "→ smb://nas02.internal/finance", tone: "xfer" },
  { text: "2019/q4-laporan.xlsx            4,2 MiB   sha256 ✓", tone: "file" },
  { text: "2020/arsip-kontrak/0341.pdf    18,9 MiB   sha256 ✓", tone: "file" },
  { text: "2021/ledger/jurnal-2021.csv   212,4 MiB   sha256 ✓", tone: "file" },
  { text: "2022/audit/bukti-scan/*.tif     1,8 GiB   sha256 ✓", tone: "file" },
  { tag: "retry", text: "2023/vendor-08.zip — SMB_STATUS_NETWORK_BUSY · coba 2/3", tone: "warn" },
  { tag: "resume", text: "2023/vendor-08.zip — dilanjutkan dari byte 44.012.003", tone: "ok" },
  { tag: "xfer", text: "612.904/1.248.512 · 341 MiB/s · ETA 00:31:42", tone: "xfer" },
  { text: "2023/ap/inv-88213.pdf           912 KiB   sha256 ✓", tone: "file" },
  { text: "2024/hr/kontrak/*.pdf            64,1 MiB  sha256 ✓", tone: "file" },
  { tag: "xfer", text: "1.248.512/1.248.512 · 847,3 GiB · 0 gagal", tone: "xfer" },
  { tag: "verify", text: "checksum pass 1.248.512/1.248.512 · 0 mismatch", tone: "verify" },
  { tag: "done", text: "0 error · audit → /var/log/omniblob/run-0482.log", tone: "done" },
];

export const tickerItems = [
  "NFSv3",
  "NFSv4.1",
  "SMB 2.1",
  "SMB 3.1.1",
  "POSIX ACL",
  "NTFS ACL",
  "XATTR",
  "SHA-256",
  "BLAKE2b",
  "DELTA TRANSFER",
  "DRY-RUN",
  "RESUME OTOMATIS",
  "AUDIT LOG",
  "AIR-GAP READY",
  "PROMETHEUS",
  "SYSTEMD",
  "TANPA AGEN",
  "TANPA TELEMETRI",
];

export type Problem = { idx: string; tag: string; title: string; body: string };

export const problems: Problem[] = [
  {
    idx: "01",
    tag: "inventaris",
    title: "Share tersebar, peta tidak ada.",
    body: "NFS di tiga server, SMB di dua lagi, satu NAS yang mulai berbunyi aneh. Tidak ada satu pun dokumen yang menyebut semua mount point — dan satu orang yang hafal, sedang cuti panjang.",
  },
  {
    idx: "02",
    tag: "permission",
    title: "Tiga rezim permission dalam satu pohon file.",
    body: "POSIX mode di satu sisi, ACL NFSv4 di sisi lain, NTFS ACL lewat SMB. Ditambah satu service account yang semua orang takut hapus karena “dulu pernah ada gunanya”.",
  },
  {
    idx: "03",
    tag: "integritas",
    title: "Backup terakhir yang terverifikasi: tidak ada yang tahu.",
    body: "Job backup memang jalan tiap malam. Tapi kapan terakhir kali restore diuji, dan apakah hasilnya lengkap — itu pertanyaan yang membuat rapat jadi hening.",
  },
  {
    idx: "04",
    tag: "lifecycle",
    title: "Hardware lama, vendor hilang.",
    body: "Server filer generasi lama sudah lewat masa dukung. Spare part tidak ada, dokumentasi vendor tidak ditemukan. Datanya harus keluar dari sana — dengan selamat, dan dengan jadwal Anda.",
  },
  {
    idx: "05",
    tag: "proses",
    title: "Percobaan pindah sebelumnya berhenti di spreadsheet.",
    body: "Seseorang pernah menulis daftar share di spreadsheet, memperkirakan durasi copy manual, lalu menyerah di baris ke-40. Sejak itu tidak ada yang menyentuh topik ini.",
  },
];

export type Step = { n: string; title: string; body: string; out: string };

export const steps: Step[] = [
  {
    n: "01",
    title: "Inventarisasi",
    body: "Scan seluruh mount sumber: jumlah file, ukuran, tipe, umur, dan struktur permission. Hasilnya inventori yang bisa dibaca manusia dan mesin — dasar semua keputusan berikutnya.",
    out: "omniblob scan nfs://fs01/export/finance --out inventaris.csv",
  },
  {
    n: "02",
    title: "Perencanaan",
    body: "Kelompokkan file ke dalam job, hitung estimasi durasi dan kebutuhan bandwidth, deteksi bentrok permission dan path yang terlalu panjang sebelum menjadi masalah di tengah malam.",
    out: "omniblob plan inventaris.csv --job finance-migration",
  },
  {
    n: "03",
    title: "Transfer",
    body: "Transfer delta multi-thread dengan resume otomatis saat koneksi putus, batas bandwidth agar produksi tidak tercekik, dan jendela jadwal — misalnya hanya 22:00–05:00.",
    out: "omniblob run --job finance-migration --bw 400MiB/s",
  },
  {
    n: "04",
    title: "Verifikasi",
    body: "Checksum per file dibandingkan sumber lawan target. File yang mismatch masuk antrean ulang otomatis, bukan ke dalam email laporan yang harus Anda baca manual.",
    out: "omniblob verify --job finance-migration --algo sha256",
  },
  {
    n: "05",
    title: "Cutover & audit",
    body: "Sinkronisasi final singkat saat jendela perubahan, lalu laporan per job: apa yang pindah, apa yang diverifikasi, apa yang ditahan. Audit log siap untuk tim compliance.",
    out: "omniblob report --job finance-migration --format pdf",
  },
];

export type SpecRow = { k: string; v: string };
export type SpecGroup = { title: string; rows: SpecRow[] };

export const specGroups: SpecGroup[] = [
  {
    title: "Sumber & target",
    rows: [
      { k: "Protokol sumber", v: "NFSv3, NFSv4/4.1, SMB 2.1/3.x, CIFS, mount lokal (ext4, XFS, ZFS, Btrfs), FTP read-only" },
      { k: "Protokol target", v: "NFS, SMB, filesystem lokal — target campuran dalam satu job diperbolehkan" },
      { k: "Skala teruji", v: "Miliaran file per lingkungan; snapshot inventori per run; file tunggal di atas 16 TiB" },
    ],
  },
  {
    title: "Integritas & metadata",
    rows: [
      { k: "Verifikasi", v: "SHA-256 atau BLAKE2b per file, pra dan pasca transfer; mode perbandingan byte-for-byte opsional" },
      { k: "Permission", v: "POSIX mode, uid/gid, ACL NFSv4, NTFS ACL via SMB, xattr, hard link, sparse file, timestamp" },
      { k: "Penanganan error", v: "Retry berpola backoff, antrean gagal terpisah, exit code eksplisit — tidak ada kegagalan diam-diam" },
    ],
  },
  {
    title: "Operasi",
    rows: [
      { k: "Antarmuka", v: "CLI, job file YAML, REST API; cocok untuk cron, pipeline internal, atau operasi manual" },
      { k: "Kontrol akses", v: "RBAC per job, token berumur pendek, audit log terstruktur untuk setiap perintah" },
      { k: "Observabilitas", v: "Endpoint metrik format Prometheus, log JSON terstruktur, status job real-time" },
    ],
  },
  {
    title: "Deployment",
    rows: [
      { k: "Bentuk", v: "Satu binary statis Linux (amd64, arm64); paket .deb dan .rpm; image Docker; unit systemd disertakan" },
      { k: "Dependensi", v: "Tidak ada runtime tambahan; tidak ada agen di server sumber maupun target" },
      { k: "Lingkungan", v: "Bare metal, VM, cluster Kubernetes internal, dan lingkungan air-gapped penuh" },
    ],
  },
];

export type CliTab = { id: string; label: string; file: string; code: string };

export const cliTabs: CliTab[] = [
  {
    id: "yaml",
    label: "Job file",
    file: "finance-migration.yaml",
    code: `job: finance-migration
source:
  type: nfs
  host: fs01.internal
  path: /export/finance
target:
  type: smb
  host: nas02.internal
  share: finance
options:
  verify: sha256
  preserve: [mode, uid, gid, acl, xattr, times]
  bandwidth: 400MiB/s
  schedule: "0 22 * * *"
on_error: retry-3-then-hold`,
  },
  {
    id: "cli",
    label: "CLI",
    file: "shell",
    code: `$ omniblob scan   nfs://fs01.internal/export/finance \\
                  --out inventaris.csv

$ omniblob plan   inventaris.csv --job finance-migration

$ omniblob run    --job finance-migration --dry-run
  → 814,4 GiB akan ditransfer · 0 konflik terdeteksi

$ omniblob run    --job finance-migration --verify

$ omniblob status --watch`,
  },
  {
    id: "api",
    label: "REST API",
    file: "HTTP",
    code: `POST /api/v1/jobs/finance-migration/run
Authorization: Bearer ••••••••••••
Content-Type: application/json

{ "verify": "sha256", "dry_run": false }

→ 202 Accepted
{
  "run_id": "run-0483",
  "state": "queued",
  "eta_seconds": 2520,
  "metrics_url": "/api/v1/runs/run-0483/metrics"
}`,
  },
];

export type Faq = { q: string; a: string };

export const faqs: Faq[] = [
  {
    q: "Apakah data kami keluar dari jaringan?",
    a: "Tidak, dan ini bukan fitur yang bisa dimatikan — ini arsitekturnya. OmniBlob berjalan di infrastruktur Anda dan memindahkan data langsung antara sistem yang Anda tentukan. Tidak ada komponen yang menghubungi layanan eksternal, dan telemetri mati secara default.",
  },
  {
    q: "Apa bedanya dengan rsync?",
    a: "rsync memindahkan isi satu path dengan baik. OmniBlob memakai pendekatan serupa untuk transfer delta, lalu menambahkan yang dibutuhkan migrasi skala produksi: inventarisasi, perencanaan job, pemetaan permission antar rezim ACL, verifikasi per file, penjadwalan, resume, dan audit log. Untuk satu folder, rsync sudah cukup. Untuk 400 share warisan, Anda butuh yang kedua.",
  },
  {
    q: "Bagaimana permission dan ACL ditangani?",
    a: "Setiap job punya daftar atribut yang dipertahankan: mode, uid/gid, ACL NFSv4, NTFS ACL via SMB, xattr, timestamp, hard link, dan sparse file. Jalankan dry-run dan OmniBlob melaporkan atribut yang tidak bisa dipetakan ke target sebelum satu byte pun dipindahkan.",
  },
  {
    q: "Apakah perlu memasang agen di server sumber?",
    a: "Tidak. OmniBlob berbicara lewat protokol yang memang sudah ada — NFS dan SMB — atau lewat mount lokal. Satu binary di satu host yang punya akses ke sumber dan target. Server lama tidak perlu disentuh sama sekali.",
  },
  {
    q: "Bisa jalan di lingkungan air-gapped?",
    a: "Bisa. Binary-nya statis tanpa dependensi runtime, tidak butuh koneksi keluar saat instalasi maupun operasi, dan semua job dijalankan dari job file lokal. Beberapa instalasi kami berjalan di jaringan yang tidak pernah menyentuh internet.",
  },
  {
    q: "Berapa lama migrasi skala besar biasanya berjalan?",
    a: "Tergantung volume dan bandwidth yang Anda izinkan — karena itu ada fase perencanaan. Untuk ratusan GiB sampai beberapa TiB, sebagian besar waktu habis di transfer pertama. Setelahnya, sinkronisasi delta harian hanya memindahkan perubahan, dan cutover final biasanya selesai dalam satu jendela perubahan pendek.",
  },
];

export type Dl = { platform: string; arch: string; kind: string; size: string };

export const downloads: Dl[] = [
  { platform: "Linux", arch: "amd64", kind: ".tar.gz", size: "18,4 MB" },
  { platform: "Linux", arch: "arm64", kind: ".tar.gz", size: "17,1 MB" },
  { platform: "Debian / Ubuntu", arch: "amd64", kind: ".deb", size: "19,0 MB" },
  { platform: "RHEL / Rocky", arch: "x86_64", kind: ".rpm", size: "19,2 MB" },
  { platform: "Docker image", arch: "multi", kind: "OCI", size: "42,7 MB" },
];

export const heroStats = [
  { value: "1.248.512", label: "file terverifikasi dalam satu run" },
  { value: "0", label: "error tak tertangani" },
  { value: "341 MiB/s", label: "throughput sustained di 10GbE" },
  { value: "100%", label: "checksum pass, sumber ↔ target" },
];

export const navLinks = [
  { href: "#masalah", label: "Masalah" },
  { href: "#arsitektur", label: "Arsitektur" },
  { href: "#cara-kerja", label: "Cara Kerja" },
  { href: "#spec", label: "Spec" },
  { href: "#deploy", label: "Deploy" },
  { href: "#faq", label: "FAQ" },
];
