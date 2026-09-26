package repository

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"pwni-file-sync/internal/entity"
)

// GetVirtualDirectory is placed here to avoid modifying the main binary_file_repository.go file directly due to size.
func (r *binaryFileRepository) GetVirtualDirectory(ctx context.Context, prefix string) ([]entity.VirtualNode, error) {
	cleanPrefix := strings.TrimPrefix(filepath.ToSlash(prefix), "/")
	if cleanPrefix != "" && !strings.HasSuffix(cleanPrefix, "/") {
		cleanPrefix += "/"
	}

	// HIGH-PERFORMANCE SQL AGGREGATION
	// Offloads string splitting and grouping to PostgreSQL C-engine to prevent Go memory exhaustion.
	query := `
		SELECT 
			split_part(substring(path from length($1::text) + 1), '/', 1) AS node_name,
			BOOL_OR(position('/' in substring(path from length($1::text) + 1)) > 0) AS is_directory,
			COALESCE(SUM(size), 0)::bigint AS total_size,
			MAX(create_date) AS last_modified
		FROM binary_file
		WHERE path LIKE $1::text || '%'
		  AND length(path) > length($1::text)
		GROUP BY node_name
		ORDER BY is_directory DESC, node_name ASC
		LIMIT 500
	`
	rows, err := r.db.Query(ctx, query, cleanPrefix)
	if err != nil {
		return nil, fmt.Errorf("GetVirtualDirectory query: %w", err)
	}
	defer rows.Close()

	var result []entity.VirtualNode

	for rows.Next() {
		var name string
		var isDirectory bool
		var size int64
		var modifiedTime sql.NullTime

		if err := rows.Scan(&name, &isDirectory, &size, &modifiedTime); err != nil {
			return nil, err
		}

		validTime := time.Now()
		if modifiedTime.Valid {
			validTime = modifiedTime.Time
		}

		result = append(result, entity.VirtualNode{
			Name:         name,
			Path:         cleanPrefix + name,
			IsDirectory:  isDirectory,
			Size:         size,
			ModifiedTime: validTime,
		})
	}

	return result, nil
}
