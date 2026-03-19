package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/go-sql-driver/mysql"

	"github.com/dabom/simulator-usage/internal/config"
	"github.com/dabom/simulator-usage/internal/generator"
)

// Connect opens a MySQL connection using the given config.
func Connect(cfg config.DatabaseConfig) (*sql.DB, error) {
	mysqlCfg := mysql.Config{
		User:                 cfg.User,
		Passwd:               cfg.Password,
		Net:                  "tcp",
		Addr:                 fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		DBName:               cfg.DBName,
		ParseTime:            true,
		AllowNativePasswords: true,
	}
	dsn := mysqlCfg.FormatDSN()

	slog.Info("connecting to database",
		"host", cfg.Host,
		"port", cfg.Port,
		"user", cfg.User,
		"dbName", cfg.DBName,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	slog.Info("database connected successfully")
	return db, nil
}

// LoadFamilies fetches family and member data from the database.
func LoadFamilies(ctx context.Context, db *sql.DB) ([]generator.Family, error) {
	query := `
		SELECT fm.family_id, fm.customer_id
		FROM family_member fm
		JOIN family f ON f.id = fm.family_id AND f.deleted_at IS NULL
		JOIN customer c ON c.id = fm.customer_id AND c.deleted_at IS NULL
		WHERE fm.deleted_at IS NULL
		ORDER BY fm.family_id, fm.customer_id
	`
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query family members: %w", err)
	}
	defer rows.Close()

	familyMap := make(map[int64][]int64)
	var order []int64
	for rows.Next() {
		var familyID, customerID int64
		if err := rows.Scan(&familyID, &customerID); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		if _, exists := familyMap[familyID]; !exists {
			order = append(order, familyID)
		}
		familyMap[familyID] = append(familyMap[familyID], customerID)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate rows: %w", err)
	}

	families := make([]generator.Family, 0, len(order))
	for _, fid := range order {
		families = append(families, generator.Family{
			ID:      fid,
			Members: familyMap[fid],
		})
	}

	slog.Info("families loaded from database",
		"families", len(families),
		"members", countMembers(families),
	)
	return families, nil
}

func countMembers(families []generator.Family) int {
	total := 0
	for i := range families {
		total += len(families[i].Members)
	}
	return total
}
