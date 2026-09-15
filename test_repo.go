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
	
	var totalJobs int64
	err = conn.QueryRow(context.Background(), "SELECT COUNT(*) FROM log_file_rsync").Scan(&totalJobs)
	if err != nil {
		fmt.Println("Count error:", err)
		return
	}
	fmt.Println("Total Jobs:", totalJobs)

	query := `
		SELECT 
			id::VARCHAR, 
			'Background Migration' as name, 
			'Migrasi' as type, 
			0::BIGINT as volume_bytes, 
			COALESCE(TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI'), TO_CHAR(NOW(), 'YYYY-MM-DD HH24:MI')) as date, 
			COALESCE(error_logs, 'Success') as status
		FROM log_file_rsync
		ORDER BY id DESC LIMIT 20
	`
	rows, err := conn.Query(context.Background(), query)
	if err != nil {
		fmt.Println("Query error:", err)
		return
	}
	defer rows.Close()
	
	for rows.Next() {
		var id, name, typ, date, status string
		var vol int64
		if err := rows.Scan(&id, &name, &typ, &vol, &date, &status); err != nil {
			fmt.Println("Scan error EXACT:", err)
			return
		}
		fmt.Println("Row:", id, name, typ, vol, date, status)
	}
}
