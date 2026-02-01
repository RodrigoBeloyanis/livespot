package audit

import (
	"database/sql"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/infra/sqlite"
)

func EnsureSchema(db *sql.DB, now time.Time) error {
	return sqlite.Migrate(db, now)
}
