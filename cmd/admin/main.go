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
	"gopherpay/internal/reporting"
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

	log.Printf("[INFO] Generating comprehensive report for User ID: %d\n", userID)

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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Use the new multi-table query
	details, err := reporting.GetUserTransactionsWithAudit(ctx, db, userID)
	if err != nil {
		log.Println("[ERROR] Failed to fetch transactions:", err)
		os.Exit(1)
	}

	fileName := fmt.Sprintf("user_%d_report.csv", userID)
	file, err := os.Create(fileName)
	if err != nil {
		log.Println("[ERROR] Failed to create file:", err)
		os.Exit(1)
	}
	defer file.Close()

	bufferedWriter := bufio.NewWriter(file)
	csvWriter := csv.NewWriter(bufferedWriter)

	// ===== SECTION 1: Transaction Overview =====
	err = csvWriter.Write([]string{
		"=== TRANSACTION OVERVIEW ===",
	})
	if err != nil {
		log.Println("[ERROR] Failed to write header:", err)
		os.Exit(1)
	}

	csvWriter.Write([]string{})

	err = csvWriter.Write([]string{
		"TransactionID",
		"RequestID",
		"FromAccountID",
		"ToAccountID",
		"Amount(Cents)",
		"Status",
		"ErrorMessage",
		"FromAccountBalance",
		"ToAccountBalance",
		"CreatedAt",
	})
	if err != nil {
		log.Println("[ERROR] Failed to write transaction header:", err)
		os.Exit(1)
	}

	for _, detail := range details {
		errorMsg := ""
		if detail.ErrorMessage != nil {
			errorMsg = *detail.ErrorMessage
		}

		record := []string{
			strconv.FormatUint(detail.TransactionID, 10),
			detail.RequestID,
			strconv.FormatUint(detail.FromAccountID, 10),
			strconv.FormatUint(detail.ToAccountID, 10),
			strconv.FormatInt(detail.Amount, 10),
			detail.Status,
			errorMsg,
			strconv.FormatInt(detail.FromBalance, 10),
			strconv.FormatInt(detail.ToBalance, 10),
			detail.CreatedAt,
		}

		if err := csvWriter.Write(record); err != nil {
			log.Println("[ERROR] Failed to write transaction row:", err)
			os.Exit(1)
		}
	}

	csvWriter.Write([]string{})
	csvWriter.Write([]string{"=== AUDIT TRAIL ===", "", "", "", "", "", "", "", "", ""})
	csvWriter.Write([]string{})

	// ===== SECTION 2: Audit Trail =====
	csvWriter.Write([]string{
		"RequestID",
		"Action",
		"Status",
		"Message",
		"Timestamp",
	})

	for _, detail := range details {
		for _, audit := range detail.AuditLogs {
			auditMsg := ""
			if audit.Message != nil {
				auditMsg = *audit.Message
			}

			auditRecord := []string{
				detail.RequestID,
				audit.Action,
				audit.Status,
				auditMsg,
				audit.Timestamp,
			}

			if err := csvWriter.Write(auditRecord); err != nil {
				log.Println("[ERROR] Failed to write audit row:", err)
				os.Exit(1)
			}
		}
	}

	csvWriter.Flush()
	bufferedWriter.Flush()

	if err := csvWriter.Error(); err != nil {
		log.Println("[ERROR] CSV writer error:", err)
		os.Exit(1)
	}

	log.Printf("[SUCCESS] Report generated successfully with %d transactions\n", len(details))
	log.Printf("[INFO] Output file: %s\n", fileName)

	os.Exit(0)
}
