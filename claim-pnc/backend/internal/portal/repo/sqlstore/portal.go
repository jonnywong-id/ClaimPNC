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
var berkasKueri embed.FS

// Repo membaca daftar portal dari POOLDATA.M_PORTAL_PNC.
//
// Ia dipasang pada koneksi **portal utama**: daftar portal dibutuhkan sebelum pengguna
// memilih portal, sehingga tidak mungkin dibaca dari basis data portal yang belum
// dipilih.
type Repo struct {
	db    *sql.DB
	kueri string
}

// RepoBaru membentuk repo; db wajib sudah terhubung.
func RepoBaru(db *sql.DB) *Repo {
	isi, err := berkasKueri.ReadFile("portal.sql")
	if err != nil {
		// Berkas disematkan saat kompilasi; ketiadaannya adalah cacat pemrograman yang
		// harus terlihat saat pertama dijalankan, bukan galat yang menunggu pengguna.
		panic("portal/sqlstore: tidak dapat membaca portal.sql: " + err.Error())
	}
	return &Repo{db: db, kueri: potongSetelahPenanda(string(isi))}
}

// Daftar membaca seluruh portal.
func (r *Repo) Daftar(ctx context.Context) ([]portal.Portal, error) {
	baris, err := r.db.QueryContext(ctx, r.kueri)
	if err != nil {
		return nil, fmt.Errorf("portal/sqlstore: membaca daftar portal: %w", err)
	}
	defer func() { _ = baris.Close() }()

	var hasil []portal.Portal
	for baris.Next() {
		var p portal.Portal
		if err := baris.Scan(&p.ID, &p.Nama, &p.Alias); err != nil {
			return nil, fmt.Errorf("portal/sqlstore: membaca baris portal: %w", err)
		}
		p.ID = strings.TrimSpace(p.ID)
		p.Nama = strings.TrimSpace(p.Nama)
		p.Alias = strings.ToUpper(strings.TrimSpace(p.Alias))
		hasil = append(hasil, p)
	}
	if err := baris.Err(); err != nil {
		return nil, fmt.Errorf("portal/sqlstore: menelusuri daftar portal: %w", err)
	}
	return hasil, nil
}

// potongSetelahPenanda mengambil isi setelah baris "-- name: ...", membuang komentar
// kepala berkas yang tidak perlu dikirim ke basis data.
func potongSetelahPenanda(isi string) string {
	const penanda = "-- name:"
	var badan []string
	mulai := false
	for _, baris := range strings.Split(isi, "\n") {
		if strings.HasPrefix(strings.TrimSpace(baris), penanda) {
			mulai = true
			continue
		}
		if mulai {
			badan = append(badan, baris)
		}
	}
	return strings.TrimSpace(strings.Join(badan, "\n"))
}

var _ portal.Repo = (*Repo)(nil)
