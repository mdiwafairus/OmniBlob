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
	
	rows, err := conn.Query(context.Background(), "SELECT conname, pg_get_constraintdef(c.oid) FROM pg_constraint c JOIN pg_namespace n ON n.oid = c.connamespace WHERE conrelid = 'binary_file'::regclass;")
	if err != nil {
		fmt.Println("Query error:", err)
		return
	}
	defer rows.Close()
	fmt.Println("Constraints:")
	for rows.Next() {
		var name, def string
		if err := rows.Scan(&name, &def); err == nil {
			fmt.Printf("- %s: %s\n", name, def)
		}
	}
}
