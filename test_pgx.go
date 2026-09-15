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
	
	query := "SELECT 0 as vol"
	var vol int64
	err = conn.QueryRow(context.Background(), query).Scan(&vol)
	fmt.Println("Error:", err)
}
