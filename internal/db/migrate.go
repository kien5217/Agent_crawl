package db

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Migrate reads all SQL files from the specified migrations directory and executes them in order to set up or update the database schema. It sorts the files alphabetically to ensure they are applied in the correct sequence. If any migration fails, it returns an error.
func Migrate(ctx context.Context, conn *pgx.Conn, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir) // đọc tất cả file trong thư mục migrations, lọc ra những file có đuôi .sql và sắp xếp theo tên để đảm bảo thứ tự thực thi đúng
	if err != nil {
		return err
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(migrationsDir, e.Name()))
		}
	}
	sort.Strings(files)

	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := conn.Exec(ctx, string(b)); err != nil {
			return err
		}
	}
	return nil
}
