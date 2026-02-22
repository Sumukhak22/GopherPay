# GopherPay 💳

GopherPay is a concurrent, ACID-compliant transaction processing system
built in Go.\
It simulates a fintech wallet backend with atomic transfers,
asynchronous processing, audit logging, CLI reporting, and a real-time
admin dashboard.

------------------------------------------------------------------------

## ✨ Features

-   Atomic money transfers using MySQL transactions
-   Worker pool for asynchronous processing
-   Real-time admin dashboard (charts + metrics)
-   Complete audit logging for all transfer attempts
-   CLI tool to generate CSV transaction reports
-   Health endpoint for system monitoring
-   Safe currency handling (stored in paise as int64)

------------------------------------------------------------------------

## 🏗 Architecture

HTTP → Worker Pool → Service Layer → Repository → MySQL

-   Deadlock prevention using consistent row locking
-   SELECT ... FOR UPDATE for balance locking
-   Transaction rollback on failure
-   Backpressure handling (HTTP 429 when overloaded)
-   Graceful shutdown support

------------------------------------------------------------------------

## 🗄 Database Design

### Accounts

-   id
-   balance (BIGINT, stored in paise)
-   timestamps

### Transactions

-   request_id
-   from_account_id
-   to_account_id
-   amount (paise)
-   status (PENDING / SUCCESS / FAILED)
-   error_message
-   balance snapshots
-   timestamps

### Audit Logs

-   request_id
-   action
-   status
-   message
-   timestamp

All amounts are stored in paise to prevent floating point precision
issues.

------------------------------------------------------------------------

## 🚀 Running the Project

### 1. Configure .env

DB_USER=root\
DB_PASSWORD=yourpassword\
DB_HOST=localhost\
DB_PORT=3306\
DB_NAME=gopherpay

### 2. Run migrations

Execute SQL inside migrations/001_init_schema.sql.

### 3. Start server

go run cmd/server/main.go

Dashboard available at:

http://localhost:8080/

------------------------------------------------------------------------

## 🖥 CLI Reporting

Generate CSV report:

go run cmd/admin/main.go report --user=1

------------------------------------------------------------------------

## 📡 API Endpoints

POST /transfer\
GET /accounts\
GET /transactions\
GET /audit\
GET /health

------------------------------------------------------------------------

## 📌 Tech Stack

-   Go (net/http, database/sql)
-   MySQL
-   HTML/CSS + Chart.js

------------------------------------------------------------------------

# 🏁 Conclusion

GopherPay demonstrates how to build a concurrent, fault-tolerant transaction processing system in Go using ACID database transactions, worker pools, and layered architecture principles.

This project simulates real-world fintech backend design patterns.
