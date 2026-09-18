package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"pwni-file-sync/internal/config"
	"pwni-file-sync/internal/database"
	"pwni-file-sync/internal/logger"
)

func main() {
	configPath := "configs/config.yaml"
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	l := logger.NewLogger()
	dbPool, err := database.NewPostgres(cfg, &l)
	if err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	defer dbPool.Close()

	ctx := context.Background()

	rows, err := dbPool.Query(ctx, "SELECT bin_id, path FROM binary_file WHERE path != ''")
	if err != nil {
		log.Fatalf("Failed to query: %v", err)
	}
	defer rows.Close()

	type fileInfo struct {
		id   int64
		path string
	}
	var files []fileInfo
	for rows.Next() {
		var f fileInfo
		if err := rows.Scan(&f.id, &f.path); err != nil {
			continue
		}
		files = append(files, f)
	}
	rows.Close()

	count := 0
	for _, f := range files {
		absPath := filepath.Join(cfg.Storage.RootPath, filepath.FromSlash(f.path))
		info, err := os.Stat(absPath)
		if err != nil {
			continue
		}
		modTime := info.ModTime()

		_, err = dbPool.Exec(ctx, "UPDATE binary_file SET create_date = $1 WHERE bin_id = $2", modTime, f.id)
		if err == nil {
			count++
		}
	}

	fmt.Printf("Successfully repaired %d file dates!\n", count)
}
