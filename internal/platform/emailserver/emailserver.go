// Package emailserver membaca akun server surel dari POOLDATA.M_EMAIL_SERVER_PNC.
//
// # Apa yang digantikan
//
// Rule Pega mengirim surel lewat **Email Account** Pega (`EmailAccount = "Admin-PNC"`,
// `"ASM"`, `"GCNM_ASI"`, …), atau — seperti `SubmitTanggalLengkapTKA` — dengan host dan sandi
// SMTP yang ditulis langsung di dalam rule. Di aplikasi ini akun itu dibaca dari tabel
// M_EMAIL_SERVER_PNC menurut kolom EMAIL_ACCOUNT (Work Owner 2026-10-10). Modul yang di Pega
// memakai `"Admin-PNC"` — PLA/DLA (`UpdateDetailPLA2`, `UpdateDetailDLA2`) dan peringatan
// Master Rekening (`SendEmailAlertRekening`) — membaca akun itu (EMAIL_ACCOUNT_ADMIN_PNC);
// modul lain memakai `ClaimPNC` (EMAIL_ACCOUNT).
//
// # Dibaca setiap kali surel dikirim
//
// Bukan sekali saat start: perubahan isi tabel — sandi yang dirotasi, host yang dipindah —
// berlaku pada surel berikutnya tanpa restart, sama seperti Email Account Pega yang dibaca
// saat rule berjalan. Surel jarang dikirim, sehingga satu kueri per surel tidak berarti.
package emailserver

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// DefaultAccount adalah EMAIL_ACCOUNT bawaan.
const DefaultAccount = "ClaimPNC"

// AdminPNCAccount adalah Email Account Pega `"Admin-PNC"`.
const AdminPNCAccount = "Admin-PNC"

// Account adalah satu baris M_EMAIL_SERVER_PNC.
type Account struct {
	Name        string // EMAIL_ACCOUNT
	Host        string // HOST
	Port        int    // PORT (teks di tabel)
	Address     string // EMAIL_ADDRESS — alamat pengirim sekaligus nama pengguna SMTP
	Password    string // PASS
	DisplayName string // DISPLAY_NAME
}

// ErrNotFound berarti tidak ada baris dengan EMAIL_ACCOUNT itu.
var ErrNotFound = errors.New("emailserver: akun surel tidak ada di POOLDATA.M_EMAIL_SERVER_PNC")

//go:embed emailserver.sql
var accountQuery string

// Store membaca satu akun dari M_EMAIL_SERVER_PNC.
type Store struct {
	db      *sql.DB
	account string
}

// NewStore membentuk pembaca akun. account kosong berarti DefaultAccount.
func NewStore(db *sql.DB, account string) *Store {
	if strings.TrimSpace(account) == "" {
		account = DefaultAccount
	}
	return &Store{db: db, account: strings.TrimSpace(account)}
}

// With membentuk pembaca akun lain di basis data yang sama. account kosong berarti akun ini.
func (s *Store) With(account string) *Store {
	if strings.TrimSpace(account) == "" {
		return s
	}
	return NewStore(s.db, account)
}

// AccountName adalah EMAIL_ACCOUNT yang dibaca.
func (s *Store) AccountName() string { return s.account }

// Account membaca baris akun dan memeriksa kelengkapannya.
func (s *Store) Account(ctx context.Context) (Account, error) {
	var name, host, port, address, password, display sql.NullString
	err := s.db.QueryRowContext(ctx, accountQuery, s.account).
		Scan(&name, &host, &port, &address, &password, &display)
	if errors.Is(err, sql.ErrNoRows) {
		return Account{}, fmt.Errorf("%w: EMAIL_ACCOUNT %q", ErrNotFound, s.account)
	}
	if err != nil {
		return Account{}, fmt.Errorf("emailserver: membaca akun %q: %w", s.account, err)
	}
	a := Account{
		Name:        strings.TrimSpace(name.String),
		Host:        strings.TrimSpace(host.String),
		Address:     strings.TrimSpace(address.String),
		Password:    password.String,
		DisplayName: strings.TrimSpace(display.String),
	}
	a.Port, err = strconv.Atoi(strings.TrimSpace(port.String))
	if err != nil || a.Port <= 0 {
		return Account{}, fmt.Errorf("emailserver: PORT akun %q bukan angka yang sah", s.account)
	}
	if a.Host == "" || a.Address == "" {
		return Account{}, fmt.Errorf("emailserver: HOST atau EMAIL_ADDRESS akun %q kosong", s.account)
	}
	return a, nil
}
