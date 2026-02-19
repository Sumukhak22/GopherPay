package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"gopherpay/internal/config"

	"database/sql"
)

func main() {

	// Basic logger setup
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if len(os.Args) < 2 {
		log.Println("[ERROR] Expected subcommand: report")
		os.Exit(1)
	}

	switch os.Args[1] {

	case "report":
		runReport()

	default:
		log.Println("[ERROR] Unknown command")
		os.Exit(1)
	}
}

func runReport() {

	reportCmd := flag.NewFlagSet("report", flag.ExitOnError)
	userIDFlag := reportCmd.String("user", "", "User account ID (required)")

	if err := reportCmd.Parse(os.Args[2:]); err != nil {
		log.Println("[ERROR] Failed to parse flags:", err)
		os.Exit(1)
	}

	if *userIDFlag == "" {
		log.Println("[ERROR] --user flag is required")
		os.Exit(1)
	}

	userID, err := strconv.ParseUint(*userIDFlag, 10, 64)
	if err != nil {
		log.Println("[ERROR] Invalid user ID:", err)
		os.Exit(1)
	}

	log.Printf("[INFO] Generating transaction report for User ID: %d\n", userID)

	// // Load config
	// cfg, err := config.Load()
	// if err != nil {
	// 	log.Println("[ERROR] Config load failed:", err)
	// 	os.Exit(1)
	// }

	// // Connect DB
	// db, err := config.NewDBConnection(cfg)
	// if err != nil {
	// 	log.Println("[ERROR] Database connection failed:", err)
	// 	os.Exit(1)
	// }
	cfg, err := config.LoadDBConfig()
	if err != nil {
		log.Println("[ERROR] Config load failed:", err)
		os.Exit(1)
	}

	db, err := config.ConnectDB(cfg)
	if err != nil {
		log.Println("[ERROR] Database connection failed:", err)
		os.Exit(1)
	}

	defer db.Close()

	// Context with timeout (important to prevent hanging queries)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Query transactions (READ ONLY)
	query := `
	SELECT 
		id,
		request_id,
		from_account_id,
		to_account_id,
		amount,
		status,
		error_message,
		created_at
	FROM transactions
	WHERE from_account_id = ? OR to_account_id = ?
	ORDER BY created_at DESC
	`

	rows, err := db.QueryContext(ctx, query, userID, userID)
	if err != nil {
		log.Println("[ERROR] Query failed:", err)
		os.Exit(1)
	}
	defer rows.Close()

	fileName := fmt.Sprintf("user_%d_report.csv", userID)
	file, err := os.Create(fileName)
	if err != nil {
		log.Println("[ERROR] Failed to create file:", err)
		os.Exit(1)
	}
	defer file.Close()

	bufferedWriter := bufio.NewWriter(file)
	csvWriter := csv.NewWriter(bufferedWriter)

	// Write CSV header
	err = csvWriter.Write([]string{
		"ID",
		"RequestID",
		"FromAccountID",
		"ToAccountID",
		"Amount(Cents)",
		"Status",
		"ErrorMessage",
		"CreatedAt",
	})
	if err != nil {
		log.Println("[ERROR] Failed to write CSV header:", err)
		os.Exit(1)
	}

	var rowCount int

	for rows.Next() {

		var (
			id            uint64
			requestID     string
			fromAccountID uint64
			toAccountID   uint64
			amount        int64
			status        string
			errorMessage  sql.NullString
			createdAt     time.Time
		)

		if err := rows.Scan(
			&id,
			&requestID,
			&fromAccountID,
			&toAccountID,
			&amount,
			&status,
			&errorMessage,
			&createdAt,
		); err != nil {
			log.Println("[ERROR] Failed scanning row:", err)
			os.Exit(1)
		}

		errorMsg := ""
		if errorMessage.Valid {
			errorMsg = errorMessage.String
		}

		record := []string{
			strconv.FormatUint(id, 10),
			requestID,
			strconv.FormatUint(fromAccountID, 10),
			strconv.FormatUint(toAccountID, 10),
			strconv.FormatInt(amount, 10),
			status,
			errorMsg,
			createdAt.Format("2006-01-02 15:04:05"),
		}

		if err := csvWriter.Write(record); err != nil {
			log.Println("[ERROR] Failed writing CSV row:", err)
			os.Exit(1)
		}

		rowCount++
	}

	if err := rows.Err(); err != nil {
		log.Println("[ERROR] Row iteration error:", err)
		os.Exit(1)
	}

	csvWriter.Flush()
	bufferedWriter.Flush()

	if err := csvWriter.Error(); err != nil {
		log.Println("[ERROR] CSV writer error:", err)
		os.Exit(1)
	}

	log.Printf("[SUCCESS] Report generated successfully. Rows exported: %d\n", rowCount)
	log.Printf("[INFO] Output file: %s\n", fileName)

	os.Exit(0)
}
