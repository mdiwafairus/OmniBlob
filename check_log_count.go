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
	
	var count int
	err = conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM log_file_rsync").Scan(&count)
	fmt.Println("Count in log_file_rsync:", count)
}
