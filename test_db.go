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
	
	var maxLogBin int64
	err = conn.QueryRow(context.Background(), "SELECT COALESCE(MAX(last_bin_id), 0) FROM log_file_rsync").Scan(&maxLogBin)
	fmt.Println("Max last_bin_id in log_file_rsync:", maxLogBin)

	var maxBinFile int64
	err = conn.QueryRow(context.Background(), "SELECT COALESCE(MAX(bin_id), 0) FROM binary_file").Scan(&maxBinFile)
	fmt.Println("Max bin_id in binary_file:", maxBinFile)

	var pending int
	err = conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM binary_file WHERE bin_id > $1", maxLogBin).Scan(&pending)
	fmt.Println("Files with bin_id > maxLogBin:", pending)
	
	var pendingFlag int
	err = conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM binary_file WHERE flag != 'M'").Scan(&pendingFlag)
	fmt.Println("Files with flag != 'M':", pendingFlag)
}
