# OmniBlob Future Improvements

## 1. Graceful License Degradation (High Priority)
- **Issue:** Currently, if the 90-day trial expires, the app calls `os.Exit(1)` (Fatal panic) via `enforceLicensing()`, killing all server operations.
- **Solution:** Change `enforceLicensing()` to return a specific state (e.g., `LicenseState: EXPIRED`). Middleware should intercept all `POST/PUT/DELETE` (Upload/Migration) requests and block them, but `GET` (Download) requests MUST continue to serve files so that client production apps do not crash. 

## 2. Asynchronous Materialized View for Dashboard (Medium Priority)
- **Issue:** The 5-minute memory cache is currently populated synchronously via an HTTP request. On a 50M+ row database, the first user hitting the dashboard will trigger a massive `SELECT COUNT(*)` causing an HTTP 504 Gateway Timeout.
- **Solution:** Implement a background CRON ticker (e.g., in `dashboard_repo.go`) that computes the stats every 5 minutes completely independently of HTTP requests, and stores the results directly in RAM (`sync.RWMutex`). The HTTP handler just reads from RAM in O(1) time.
