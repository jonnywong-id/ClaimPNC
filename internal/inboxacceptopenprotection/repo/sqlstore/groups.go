package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// GroupRepo membaca access group yang diikuti sebuah login.
//
// Ia memenuhi seam `inboxacceptopenprotection.GroupReader`.
//
// # Kenapa TERPISAH dari Repo proteksi
//
// Keduanya memakai koneksi yang BERBEDA. Proteksi tinggal di basis data portal yang sedang
// dibuka; keanggotaan group tinggal di basis data utama bersama identitas (`D-78`).
// Menyatukannya ke dalam satu struct akan membuat salah satunya dipasang pada koneksi yang
// keliru — dan keliru di sini tidak menghasilkan galat, hanya kewenangan yang salah.
type GroupRepo struct {
	db *sql.DB
}

// NewGroupRepo membentuk repo; db wajib koneksi UTAMA, bukan koneksi portal.
func NewGroupRepo(db *sql.DB) *GroupRepo { return &GroupRepo{db: db} }

// GroupsOf membaca GROUP_ID yang diikuti sebuah login.
//
// Login kosong mengembalikan daftar kosong tanpa menembak basis data. Daftar kosong berarti
// "tidak berwenang" bagi pemanggilnya, dan itu jawaban yang benar — bukan galat.
func (r *GroupRepo) GroupsOf(ctx context.Context, login string) ([]string, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return nil, nil
	}

	rows, err := r.db.QueryContext(ctx, query("groups_of_login"), login)
	if err != nil {
		return nil, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: membaca access group login: %w", err)
	}
	defer rows.Close()

	var groups []string
	for rows.Next() {
		var g sql.NullString
		if err := rows.Scan(&g); err != nil {
			return nil, fmt.Errorf(
				"inboxacceptopenprotection/sqlstore: memindai access group: %w", err)
		}
		if nama := teks(g); nama != "" {
			groups = append(groups, nama)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: membaca baris access group: %w", err)
	}

	return groups, nil
}
