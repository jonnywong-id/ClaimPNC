package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxacceptopenprotection"
)

// Repo membaca antrean akseptasi dan menuliskan keputusannya.
//
// # Tiga kolom yang ditulisnya, dan tidak lebih
//
// `APPROVAL_STATUS`, `RESOLVED_BY`, `RESOLVED_DATE_TIME`. Kolom pembuatan dimiliki modul
// `inputreqprotection` dan tidak pernah disentuh di sini — itulah yang menjaga `P-1` tetap
// berlaku meski dua modul menyentuh satu tabel.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca satu halaman antrean akseptasi.
func (r *Repo) List(
	ctx context.Context,
	f inboxacceptopenprotection.Filter,
) (inboxacceptopenprotection.Page, error) {
	f = f.Normalize()

	// Antrean diterjemahkan menjadi SEPASANG parameter, bukan menjadi dua kueri: penanda
	// sama-atau-tidak, dan kode pembandingnya. Lihat kepala acceptance.sql.
	wantType, equal := f.Queue.TypeFilter()
	sama := 0
	if equal {
		sama = 1
	}

	// Penentu antrean dan kode pembandingnya masing-masing muncul DUA KALI di dalam kueri,
	// sehingga dikirim dua kali — driver mengikat menurut urutan kemunculan penanda, bukan
	// menurut nomornya. Mengirimnya sekali menghasilkan ORA-01008.
	argumen := append([]any{sama, wantType, sama, wantType}, searchArgs(f.Search)...)

	var total int
	if err := r.db.QueryRowContext(ctx, query("acceptance_count"), argumen...).Scan(&total); err != nil {
		return inboxacceptopenprotection.Page{}, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: menghitung antrean akseptasi: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query("acceptance_list"),
		append(append([]any(nil), argumen...), f.Offset, f.Limit)...)
	if err != nil {
		return inboxacceptopenprotection.Page{}, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: membaca antrean akseptasi: %w", err)
	}
	defer rows.Close()

	page := make([]inboxacceptopenprotection.Protection, 0, f.Limit)
	for rows.Next() {
		p, err := scanProtection(rows)
		if err != nil {
			return inboxacceptopenprotection.Page{}, err
		}
		page = append(page, p)
	}
	if err := rows.Err(); err != nil {
		return inboxacceptopenprotection.Page{}, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: membaca baris antrean akseptasi: %w", err)
	}

	return inboxacceptopenprotection.Page{Protections: page, Total: total}, nil
}

// Get membaca satu permintaan untuk form akseptasi.
func (r *Repo) Get(ctx context.Context, number string) (inboxacceptopenprotection.Protection, error) {
	row := r.db.QueryRowContext(ctx, query("acceptance_get"), kunci(number))

	p, err := scanProtection(row)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrNotFound
	}
	if err != nil {
		return inboxacceptopenprotection.Protection{}, err
	}
	return p, nil
}

// Decide menuliskan keputusan akseptasi.
//
// Bila tidak ada baris yang tersentuh, sebabnya DIBEDAKAN lewat pembacaan ulang: tidak ada,
// sudah diputuskan orang lain, atau belum lengkap. Ketiganya menuntut pesan berbeda —
// terutama yang kedua, karena petugas yang kalah cepat pada antrean bersama tidak sedang
// melakukan kesalahan.
func (r *Repo) Decide(
	ctx context.Context,
	number string,
	d inboxacceptopenprotection.Decision,
	by string,
	at time.Time,
) (inboxacceptopenprotection.Protection, error) {
	if !d.Valid() {
		return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrUnknownDecision
	}

	hasil, err := r.db.ExecContext(ctx, query("acceptance_decide"),
		d.Status(), strings.TrimSpace(by), at.UTC(), kunci(number))
	if err != nil {
		return inboxacceptopenprotection.Protection{}, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: menyimpan keputusan akseptasi: %w", err)
	}

	tersentuh, err := hasil.RowsAffected()
	if err != nil {
		return inboxacceptopenprotection.Protection{}, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: membaca jumlah baris tersentuh: %w", err)
	}

	if tersentuh == 0 {
		existing, getErr := r.Get(ctx, number)
		switch {
		case getErr != nil:
			return inboxacceptopenprotection.Protection{}, getErr
		case !existing.Pending():
			return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrAlreadyDecided
		default:
			return inboxacceptopenprotection.Protection{}, inboxacceptopenprotection.ErrIncomplete
		}
	}

	return r.Get(ctx, number)
}

// ── Pemindaian ───────────────────────────────────────────────────────────────────

// pemindai menyatukan *sql.Row dan *sql.Rows supaya scanProtection melayani keduanya.
type pemindai interface {
	Scan(dest ...any) error
}

// scanProtection membaca satu baris menjadi Protection.
//
// Seluruh kolom teks dibaca sebagai sql.NullString: tabelnya membolehkan NULL pada
// semuanya, dan memindainya ke string biasa akan gagal pada baris pertama yang kolomnya
// kosong.
//
// # Tiga field yang SENGAJA tidak terisi di sini
//
// `InsuredName`, `PolicyStart`, dan `PolicyEnd` ditampilkan form akseptasi
// (`Section/AcceptProtectionSection-Section.xml` memuat "Nama Tertanggung", "Start Date
// Time", "End Date Time"), tetapi ketiganya milik SNAPSHOT POLIS — bukan milik tabel ini.
//
// Mengambilnya menuntut pembacaan ke data polis, yang ada di bounded context lain (`D-04`)
// dan modulnya belum terpasang. Sampai itu ada, ketiganya kosong dan form menampilkannya
// sebagai tanda hubung. Itu lebih jujur daripada mengisinya dengan tebakan.
func scanProtection(row pemindai) (inboxacceptopenprotection.Protection, error) {
	var (
		id                        string
		nopolis, noklaim, tipe    sql.NullString
		dibuatPada                sql.NullTime
		notes, dibuatOleh, status sql.NullString
		diputuskanPada            sql.NullTime
		diputuskanOleh            sql.NullString
	)

	if err := row.Scan(&id, &nopolis, &noklaim, &tipe, &dibuatPada,
		&notes, &dibuatOleh, &status, &diputuskanPada, &diputuskanOleh); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return inboxacceptopenprotection.Protection{}, err
		}
		return inboxacceptopenprotection.Protection{}, fmt.Errorf(
			"inboxacceptopenprotection/sqlstore: memindai antrean akseptasi: %w", err)
	}

	p := inboxacceptopenprotection.Protection{
		Number:       strings.TrimSpace(id),
		PolicyNumber: teks(nopolis),
		ClaimNumber:  teks(noklaim),
		Type:         teks(tipe),
		Note:         teks(notes),
		CreatedBy:    teks(dibuatOleh),
		AcceptStatus: teks(status),
		AcceptedBy:   teks(diputuskanOleh),
	}
	if dibuatPada.Valid {
		p.InputDate = dibuatPada.Time
	}
	if diputuskanPada.Valid {
		waktu := diputuskanPada.Time
		p.AcceptedAt = &waktu
	}

	return p, nil
}

// kunci merapikan nomor proteksi menjadi bentuk yang dibandingkan kueri.
//
// Kueri membandingkannya dengan `UPPER(TRIM(ID)) = :n`, sehingga perapiannya harus terjadi
// DI SINI juga — dua tempat yang tidak sepakat menghasilkan pencarian yang selalu gagal
// tanpa satu pun galat.
func kunci(number string) string {
	return strings.ToUpper(strings.TrimSpace(number))
}

// searchArgs menyusun keempat argumen kotak pencarian.
//
// Kata pencarian muncul EMPAT KALI di dalam kueri — sekali sebagai penentu apakah
// pencariannya aktif, tiga kali sebagai pola LIKE — dan driver mengikat menurut urutan
// kemunculan penanda.
//
// Polanya dibentuk di Go supaya tanda persen dan garis bawah yang diketik pengguna tidak
// menjadi wildcard tanpa disengaja.
func searchArgs(search string) []any {
	search = strings.TrimSpace(search)
	if search == "" {
		return []any{nil, nil, nil, nil}
	}

	pola := "%" + escapeLike(strings.ToUpper(search)) + "%"
	return []any{strings.ToUpper(search), pola, pola, pola}
}

// escapeLike menetralkan wildcard LIKE di dalam kata pencarian.
//
// Karakternya dibuang, bukan diloloskan — kueri tidak memakai klausa ESCAPE. Konsekuensinya
// mencari "50%" menemukan yang memuat "50", dan itu perilaku yang masuk akal bagi kotak
// pencarian bebas.
func escapeLike(s string) string {
	s = strings.ReplaceAll(s, "%", "")
	return strings.ReplaceAll(s, "_", "")
}

func teks(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return strings.TrimSpace(v.String)
}
