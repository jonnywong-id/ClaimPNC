package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"claim-pnc/internal/registrasi"
)

// GroupStore membaca keanggotaan grup pengguna dari POOLDATA.M_LOGIN_GROUP_PNC — tabel yang
// sama dengan yang dipakai modul menu. Hanya dibaca.
type GroupStore struct {
	db *sql.DB
}

// NewGroupStore membentuk pembaca grup di atas sebuah koneksi.
func NewGroupStore(db *sql.DB) *GroupStore { return &GroupStore{db: db} }

// GroupsOf mengembalikan GROUP_ID sebuah login.
func (g *GroupStore) GroupsOf(ctx context.Context, login string) ([]string, error) {
	rows, err := executorFrom(ctx, g.db).QueryContext(ctx, loadQuery("grup_login"), login)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca grup pengguna: %w", err)
	}
	defer func() { _ = rows.Close() }()
	var result []string
	for rows.Next() {
		var group sql.NullString
		if err := rows.Scan(&group); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris grup pengguna: %w", err)
		}
		if v := strings.TrimSpace(group.String); v != "" {
			result = append(result, v)
		}
	}
	return result, rows.Err()
}

var _ registrasi.GroupSource = (*GroupStore)(nil)
