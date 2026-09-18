<!-- go build -ldflags="-s -w" -o omniblob.exe ./cmd/file-sync -->
# OmniBlob Future Improvements

## 1. Graceful License Degradation (High Priority)
- **Issue:** Currently, if the 90-day trial expires, the app calls `os.Exit(1)` (Fatal panic) via `enforceLicensing()`, killing all server operations.
- **Solution:** Change `enforceLicensing()` to return a specific state (e.g., `LicenseState: EXPIRED`). Middleware should intercept all `POST/PUT/DELETE` (Upload/Migration) requests and block them, but `GET` (Download) requests MUST continue to serve files so that client production apps do not crash. 

## 2. Asynchronous Materialized View for Dashboard (Medium Priority)
- **Issue:** The 5-minute memory cache is currently populated synchronously via an HTTP request. On a 50M+ row database, the first user hitting the dashboard will trigger a massive `SELECT COUNT(*)` causing an HTTP 504 Gateway Timeout.
- **Solution:** Implement a background CRON ticker (e.g., in `dashboard_repo.go`) that computes the stats every 5 minutes completely independently of HTTP requests, and stores the results directly in RAM (`sync.RWMutex`). The HTTP handler just reads from RAM in O(1) time.
## 3. Storage Capacity Management / Disk Full Prevention (High Priority - On Premise)
- **Issue:** On-premise servers have strict physical disk limits. If OmniBlob writes data until the disk reaches 100%, it will cause OS kernel panics and database corruption.
- **Solution:** Implement a `High-Watermark Threshold` in the configuration (e.g., `max_disk_usage_percent: 95`). Before processing any upload or migration, check the OS disk usage. If it exceeds 95%, gracefully reject new uploads with an HTTP 507 Insufficient Storage error and show a critical alert on the Dashboard, preserving system stability.

## 4. Cold Data Management / Tiered Storage (Medium Priority - On Premise)
- **Issue:** Enterprise SSDs are expensive. Storing 10-year-old unused files on the same high-speed SSDs as active data wastes IT budgets.
- **Solution:** Introduce an `Auto-Archiving` background worker. Allow sysadmins to configure a secondary `cold_storage_path` (e.g., a mounted HDD or NFS). OmniBlob will periodically scan the database for files older than X years and silently move them to the slower storage, transparently serving them from the cold path when requested.

## 5. Active Directory / LDAP Integration (Medium Priority - Security)
- **Issue:** The Dashboard is currently secured by a single static password in `config.yaml`, which is often rejected by enterprise IT auditors.
- **Solution:** Add an option to authenticate Dashboard users against the company's existing Microsoft Active Directory (LDAP).
