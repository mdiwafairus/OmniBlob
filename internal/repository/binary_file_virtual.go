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

	query := `
		SELECT path, size, create_date
		FROM binary_file
		WHERE path LIKE $1 || '%'
	`
	rows, err := r.db.Query(ctx, query, cleanPrefix)
	if err != nil {
		return nil, fmt.Errorf("GetVirtualDirectory query: %w", err)
	}
	defer rows.Close()

	nodesMap := make(map[string]*entity.VirtualNode)

	for rows.Next() {
		var path sql.NullString
		var size sql.NullInt64
		var createDate sql.NullTime
		if err := rows.Scan(&path, &size, &createDate); err != nil {
			return nil, err
		}

		if !path.Valid || path.String == "" {
			continue // Skip records with no path
		}

		validSize := int64(0)
		if size.Valid {
			validSize = size.Int64
		}

		validTime := time.Now()
		if createDate.Valid {
			validTime = createDate.Time
		}

		relPath := strings.TrimPrefix(path.String, cleanPrefix)
		if relPath == "" || (cleanPrefix != "" && relPath == path.String) {
			continue // Should not happen given LIKE, but safe check
		}

		parts := strings.Split(relPath, "/")
		name := parts[0]
		isDirectory := len(parts) > 1

		node, exists := nodesMap[name]
		if !exists {
			node = &entity.VirtualNode{
				Name:         name,
				Path:         cleanPrefix + name,
				IsDirectory:  isDirectory,
				Size:         0,
				ModifiedTime: validTime,
			}
			nodesMap[name] = node
		}

		if isDirectory {
			node.IsDirectory = true
			node.Size += validSize
			if validTime.After(node.ModifiedTime) {
				node.ModifiedTime = validTime
			}
		} else {
			node.Size = validSize
			node.ModifiedTime = validTime
		}
	}

	var result []entity.VirtualNode
	for _, node := range nodesMap {
		result = append(result, *node)
	}
	return result, nil
}
