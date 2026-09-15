# OmniBlob Migration Guide

## 1. Prepare Legacy Path
Ensure your `legacy_path` is set to your old storage directory in `config.yaml`.

## 2. Enable Migration
Set `migration.enabled: true` in `config.yaml`. The background worker will automatically scan and migrate files to the `root_path`.

## 3. Best Practices
- Run the initial discovery scan if you have a massive directory:
  `./omniblob --scan-legacy`
- Adjust `worker_count` based on your disk I/O limits.
