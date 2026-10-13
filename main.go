package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	db, _ := sql.Open("postgres", os.Getenv("PGURL"))
	defer db.Close()

	var total, comVec int
	db.QueryRow("select count(*) from chunks where repository='repo_central'").Scan(&total)
	db.QueryRow("select count(*) from chunks where repository='repo_central' and embedding is not null").Scan(&comVec)
	fmt.Printf("repo_central: %d chunks | %d com embedding\n", total, comVec)

	// Amostra: o vetor e realmente o embedding do texto, e nao zero?
	var txt string
	err := db.QueryRow(`select embedding::text from chunks where repository='repo_central' and embedding is not null limit 1`).Scan(&txt)
	if err != nil {
		fmt.Println("sem amostra:", err)
		return
	}
	zero := true
	for i := 0; i < len(txt); i++ {
		if txt[i] >= '1' && txt[i] <= '9' {
			zero = false
			break
		}
	}
	fmt.Println("vetor amostrado e todo zero?", zero)
	fmt.Println("inicio:", txt[:100], "...")
}