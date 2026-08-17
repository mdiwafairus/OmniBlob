# 🚀 OmniBlob

> **A lightweight HTTP storage and file synchronization engine built in Go, designed to decouple applications from direct filesystem and NFS I/O.**

OmniBlob is a high-performance file storage and synchronization layer written in Go. It provides an HTTP-based interface between applications and traditional file systems such as **Local Storage and NFS**, while maintaining file metadata in PostgreSQL for fast lookups and reliable file management.

## Why OmniBlob?

OmniBlob was born from a real production problem.

In a legacy application environment, multiple application servers were accessing shared files directly through NFS. Under high file access, this created significant filesystem I/O pressure, resulting in:

* High I/O wait
* Processes stuck in `D` state
* Filesystem lookup contention
* Increasing system load
* PHP-FPM process accumulation
* Application instability during high I/O activity

Instead of allowing every application process to communicate directly with the underlying filesystem, OmniBlob introduces an **HTTP-based storage layer** between the application and the physical storage.

```text
Application
     │
     │ HTTP
     ▼
┌─────────────────┐
│    OmniBlob     │
│                 │
│ Metadata Index  │
│ HTTP Streaming  │
│ Range Requests  │
│ Checksum        │
│ Background Sync │
└────────┬────────┘
         │
   ┌─────┴─────┐
   ▼           ▼
 Local Disk    NFS
```

This allows applications to interact with files through HTTP rather than directly performing filesystem operations against NFS or shared storage.

## What OmniBlob Provides

* **HTTP File Streaming** — Stream files without loading the entire file into application memory.
* **HTTP Range Requests** — Support partial file downloads and efficient media/file delivery.
* **PostgreSQL Metadata Indexing** — Store and index file metadata for fast file lookup.
* **On-the-fly MD5 Checksum** — Calculate and verify file integrity while processing files.
* **Background File Migration** — Gradually migrate legacy files without requiring a full application rewrite.
* **Local & NFS Storage Support** — Work with existing storage infrastructure instead of requiring a complete storage replacement.
* **Legacy File Synchronization** — Synchronize and manage files that already exist in traditional filesystem structures.
* **Written in Go** — Lightweight, concurrent, and suitable for high-throughput file operations.

## Legacy Migration

One of OmniBlob's primary goals is to help modernize existing applications without forcing an immediate rewrite.

Instead of migrating thousands or millions of legacy files at once, OmniBlob can process them incrementally:

```text
Legacy Files
     │
     ▼
   Scan
     │
     ▼
  Metadata
     │
     ▼
  Checksum
     │
     ▼
 Copy / Move
     │
     ▼
   Verify
     │
     ▼
   Indexed
```

This makes it possible to gradually move from a traditional filesystem-based architecture toward an HTTP-based storage architecture.

## The Goal

OmniBlob is not intended to simply recreate an existing object-storage system.

Its primary goal is to provide a **simple storage abstraction layer for applications that still depend heavily on local filesystems, shared storage, or NFS**.

**Modernize legacy file storage without rewriting your entire application.**

OmniBlob aims to make file storage easier to scale, migrate, monitor, and integrate while reducing the application's direct dependency on filesystem and NFS I/O.
