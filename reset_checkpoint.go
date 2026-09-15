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
	
	_, err = conn.Exec(context.Background(), "INSERT INTO log_file_rsync (server, last_bin_id, bin_executed, error_logs) VALUES ('local-storage-134', 0, '{}', 'Manual Reset by AI')")
	if err != nil {
		fmt.Println("Reset error:", err)
		return
	}
	fmt.Println("Checkpoint successfully reset to 0!")
}
