package main
import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v4"
)
func main() {
	conn, err := pgx.Connect(context.Background(), "postgres://postgres:1234@localhost:5432/omniblob?sslmode=disable")
	if err != nil {
		fmt.Println("DB error:", err)
		return
	}
	defer conn.Close(context.Background())
	
	rows, err := conn.Query(context.Background(), "SELECT file_name, path FROM binary_file WHERE flag != 'M' LIMIT 5")
	if err != nil {
		fmt.Println("Query error:", err)
		return
	}
	defer rows.Close()
	fmt.Println("Failed files:")
	for rows.Next() {
		var name, path string
		if err := rows.Scan(&name, &path); err == nil {
			fmt.Printf("- %s (Path: %s)\n", name, path)
		}
	}
}
