package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// ensureDatabaseExists tries to auto-create the target database if possible (e.g. on localhost/XAMPP).
func ensureDatabaseExists(dsn string) {
	parsedCfg, err := mysql.ParseDSN(dsn)
	if err != nil || parsedCfg.DBName == "" {
		return
	}
	dbName := parsedCfg.DBName
	parsedCfg.DBName = ""
	rootDB, err := sql.Open("mysql", parsedCfg.FormatDSN())
	if err != nil {
		return
	}
	defer rootDB.Close()
	_, _ = rootDB.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", dbName))
}

// Connect opens a MySQL connection, verifies it with Ping, and returns the *sql.DB.
func Connect(dsn string) *sql.DB {
	ensureDatabaseExists(dsn)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("[db] Failed to open database: %v", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	const maxAttempts = 10
	for i := 1; i <= maxAttempts; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		log.Printf("[db] Ping attempt %d/%d failed: %v — retrying...", i, maxAttempts, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Printf("\n=======================================================")
		log.Printf("⚠️  ไม่สามารถเชื่อมต่อฐานข้อมูล MySQL ได้!")
		log.Printf("1. หากรันในเครื่อง (Localhost): โปรดเปิด XAMPP Control Panel แล้วกด 'Start' ที่ MySQL")
		log.Printf("2. ตรวจสอบไฟล์ .env ว่า DB_DSN ถูกต้องหรือไม่: %s", dsn)
		log.Printf("3. หากรันบน Cloud/Host: ตรวจสอบ Environment Variable 'DATABASE_URL' หรือ 'DB_DSN'")
		log.Printf("=======================================================\n")
		log.Fatalf("[db] Cannot connect to database: %v", err)
	}

	log.Println("[db] Connected to MySQL successfully")
	return db
}

// RunMigrations executes all SQL migration files in the given directory or single file.
func RunMigrations(db *sql.DB, migrationTarget string) {
	info, err := os.Stat(migrationTarget)
	if err != nil {
		log.Fatalf("[db] Cannot access migration path %s: %v", migrationTarget, err)
	}

	var files []string
	if info.IsDir() {
		entries, err := os.ReadDir(migrationTarget)
		if err != nil {
			log.Fatalf("[db] Cannot read migration dir: %v", err)
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
				files = append(files, filepath.Join(migrationTarget, entry.Name()))
			}
		}
		sort.Strings(files)
	} else {
		files = []string{migrationTarget}
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("[db] Cannot read migration file %s: %v", file, err)
		}

		statements := strings.Split(string(data), ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.Exec(stmt); err != nil {
				errMsg := strings.ToLower(err.Error())
				// Ignore harmless re-run errors such as duplicate column or duplicate key
				if strings.Contains(errMsg, "duplicate column") || strings.Contains(errMsg, "already exists") {
					continue
				}
				log.Fatalf("[db] Migration failed on file %s, statement:\n%s\nError: %v", file, stmt, err)
			}
		}
		log.Printf("[db] Applied migration: %s", filepath.Base(file))
	}
	log.Println("[db] All migrations completed successfully")
}

