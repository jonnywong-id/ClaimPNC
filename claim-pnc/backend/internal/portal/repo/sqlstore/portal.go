// Package sqlstore memenuhi seam portal.Repo dengan SQL.
//
// Dua aturan mengikat: teks SQL berada di berkas .sql terpisah, dan nilai selalu lewat
// parameter binding.
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"strings"

	"claim-pnc/internal/portal"
)

//go:embed *.sql
var queryFiles embed.FS

// Repo membaca daftar portal dari POOLDATA.M_PORTAL_PNC.
//
// Ia dipasang pada koneksi **portal utama**: daftar portal dibutuhkan sebelum pengguna
// memilih portal, sehingga tidak mungkin dibaca dari basis data portal yang belum
// dipilih.
type Repo struct {
	db    *sql.DB
	query string
}

// NewRepo membentuk repo; db wajib sudah terhubung.
func NewRepo(db *sql.DB) *Repo {
	content, err := queryFiles.ReadFile("portal.sql")
	if err != nil {
		// Berkas disematkan saat kompilasi; ketiadaannya adalah cacat pemrograman yang
		// harus terlihat saat pertama dijalankan, bukan galat yang menunggu pengguna.
		panic("portal/sqlstore: tidak dapat membaca portal.sql: " + err.Error())
	}
	return &Repo{db: db, query: contentAfterMarker(string(content))}
}

// List membaca seluruh portal.
func (r *Repo) List(ctx context.Context) ([]portal.Portal, error) {
	rows, err := r.db.QueryContext(ctx, r.query)
	if err != nil {
		return nil, fmt.Errorf("portal/sqlstore: membaca daftar portal: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []portal.Portal
	for rows.Next() {
		var p portal.Portal
		if err := rows.Scan(&p.ID, &p.Name, &p.Alias); err != nil {
			return nil, fmt.Errorf("portal/sqlstore: membaca baris portal: %w", err)
		}
		p.ID = strings.TrimSpace(p.ID)
		p.Name = strings.TrimSpace(p.Name)
		p.Alias = strings.ToUpper(strings.TrimSpace(p.Alias))
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("portal/sqlstore: menelusuri daftar portal: %w", err)
	}
	return result, nil
}

// contentAfterMarker mengambil isi setelah baris "-- name: ...", membuang komentar
// kepala berkas yang tidak perlu dikirim ke basis data.
func contentAfterMarker(content string) string {
	const marker = "-- name:"
	var body []string
	started := false
	for _, line := range strings.Split(content, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), marker) {
			started = true
			continue
		}
		if started {
			body = append(body, line)
		}
	}
	return strings.TrimSpace(strings.Join(body, "\n"))
}

var _ portal.Repo = (*Repo)(nil)
