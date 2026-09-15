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
	
	query := `
		SELECT 
			id::VARCHAR, 
			'Background Migration' as name, 
			'Migrasi' as type, 
			0 as volume_bytes, 
			TO_CHAR(created_at, 'YYYY-MM-DD HH24:MI') as date, 
			error_logs as status
		FROM log_file_rsync
		ORDER BY id DESC LIMIT 20
	`
	rows, err := conn.Query(context.Background(), query)
	if err != nil {
		fmt.Println("Query error:", err)
		return
	}
	defer rows.Close()
	fmt.Println("Results:")
	var count int
	for rows.Next() {
		count++
		var id, name, typ, date, status *string
		var vol *int64
		if err := rows.Scan(&id, &name, &typ, &vol, &date, &status); err != nil {
			fmt.Println("Scan error:", err)
		} else {
			fmt.Printf("id=%v date=%v status=%v\n", *id, *date, *status)
		}
	}
	fmt.Println("Total rows:", count)
}
