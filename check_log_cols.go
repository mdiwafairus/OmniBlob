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
	
	rows, err := conn.Query(context.Background(), "SELECT column_name FROM information_schema.columns WHERE table_name = 'log_file_rsync'")
	if err != nil {
		fmt.Println("Query error:", err)
		return
	}
	defer rows.Close()
	fmt.Println("Columns:")
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			fmt.Println("-", name)
		}
	}
}
