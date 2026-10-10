package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxadmin"
)

var _ inboxadmin.ScopeRepo = (*Repo)(nil)

// branchTimeout membatasi penerjemahan cabang, yang melintasi DB Link ke HRD.
//
// Tanpa batas, DB Link yang macet menahan seluruh layar sampai batas waktu permintaan.
const branchTimeout = 5 * time.Second

// BranchOfLogin membaca cabang petugas lewat DB Link HRD.
func (r *Repo) BranchOfLogin(ctx context.Context, login string) (string, bool, error) {
	clean := strings.TrimSpace(login)
	if clean == "" {
		return "", false, nil
	}
	ctx, cancel := context.WithTimeout(ctx, branchTimeout)
	defer cancel()

	var code sql.NullString
	err := r.db.QueryRowContext(ctx, query("branch_of_login"), clean).Scan(&code)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", false, nil
	case errors.Is(err, context.DeadlineExceeded):
		return "", false, fmt.Errorf(
			"inboxadmin/sqlstore: cabang petugas %q tidak terbaca dalam %s; DB Link ke HRD "+
				"kemungkinan tidak hidup", clean, branchTimeout)
	case err != nil:
		return "", false, fmt.Errorf("inboxadmin/sqlstore: membaca cabang petugas: %w", err)
	}
	return strings.TrimSpace(code.String), code.Valid, nil
}

// GroupsOf membaca access group petugas.
func (r *Repo) GroupsOf(ctx context.Context, login string) ([]string, error) {
	return r.texts(ctx, "groups_of_login", strings.TrimSpace(login))
}

// Regions membaca isi dropdown "Pilih Kanwil".
func (r *Repo) Regions(ctx context.Context) ([]string, error) {
	return r.texts(ctx, "region_list")
}

// texts menjalankan kueri satu kolom teks.
func (r *Repo) texts(ctx context.Context, name string, args ...any) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, query(name), args...)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri %s: %w", name, err)
	}
	defer rows.Close()
	result := []string{}
	for rows.Next() {
		var value sql.NullString
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("membaca baris kueri %s: %w", name, err)
		}
		if v := strings.TrimSpace(value.String); v != "" {
			result = append(result, v)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri %s: %w", name, err)
	}
	return result, nil
}
