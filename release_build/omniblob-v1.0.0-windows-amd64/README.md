# OmniBlob v1.0.0

OmniBlob is a high-performance Object Storage & Sync Service designed to manage, store, and migrate files dynamically with built-in dashboard capabilities.

## System Requirements
- **Windows**: Windows 7 / Windows Server 2008 R2 or newer (32-bit & 64-bit)
- **Linux**: Kernel 2.6.32 or newer (32-bit & 64-bit), compatible with Ubuntu, Debian, CentOS, RHEL, etc.
- **RAM**: Minimum 512 MB (1 GB recommended for large migrations)

## Features
- Dynamic Configuration via `configs/config.yaml`
- Legacy File Migration to sharded storage formats
- Background Job Process
- In-memory dashboard analytics

## Usage
Run the executable directly:
**Windows:**
```cmd
omniblob.exe --config configs/config.yaml
```

**Linux:**
```bash
./omniblob --config configs/config.yaml
```

Check `configs/config.yaml` to customize your database, storage paths, and clients.
