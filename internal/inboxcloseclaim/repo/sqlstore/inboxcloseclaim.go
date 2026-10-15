package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"claim-pnc/internal/inboxcloseclaim"
)

// Repo membaca klaim yang sudah tutup dari DATAPEGA.PC_ASM_FW_GCNMFW_WORK.
//
// Tabel itu MILIK SISTEM LAMA dan ditulis olehnya. Repo ini hanya membaca, dan tidak punya
// satu pun method tulis — `P-1` menetapkan satu tabel ditulis satu sistem. Yang menulis
// pada modul ini adalah RequestRepo, ke tabel milik aplikasi ini sendiri.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// filterBindCount adalah banyaknya penanda parameter penyaring pada kedua kueri.
//
// Ia konstanta supaya kode dan berkas `.sql` tidak dapat berbeda pendapat diam-diam:
// menambah satu penyaring di SQL tanpa menambah argumennya di sini menghasilkan
// **ORA-01008 not all variables bound** pada permintaan pertama yang datang — bukan saat
// kode ditulis. Dijaga `query_test.go`.
const filterBindCount = 22

// List membaca satu halaman klaim tutup beserta jumlah seluruh yang cocok.
func (r *Repo) List(ctx context.Context, f inboxcloseclaim.Filter) (inboxcloseclaim.Page, error) {
	f = f.Normalize()
	filters := filterArgs(f)

	var total int
	if err := r.db.QueryRowContext(ctx, query("close_claim_count"), filters...).Scan(&total); err != nil {
		return inboxcloseclaim.Page{}, fmt.Errorf("inboxcloseclaim/sqlstore: menghitung klaim tutup: %w", err)
	}

	listArgs := append(append([]any(nil), filters...), f.Offset, f.Limit)

	rows, err := r.db.QueryContext(ctx, query("close_claim_list"), listArgs...)
	if err != nil {
		return inboxcloseclaim.Page{}, fmt.Errorf("inboxcloseclaim/sqlstore: membaca daftar klaim tutup: %w", err)
	}
	defer func() { _ = rows.Close() }()

	claims := make([]inboxcloseclaim.ClosedClaim, 0, f.Limit)
	for rows.Next() {
		claim, err := scanClaim(rows)
		if err != nil {
			return inboxcloseclaim.Page{}, fmt.Errorf("inboxcloseclaim/sqlstore: membaca baris klaim: %w", err)
		}
		claims = append(claims, claim)
	}
	if err := rows.Err(); err != nil {
		return inboxcloseclaim.Page{}, fmt.Errorf("inboxcloseclaim/sqlstore: menelusuri daftar klaim: %w", err)
	}

	return inboxcloseclaim.Page{Claims: claims, Total: total}, nil
}

// ClosedClaimNumber mengembalikan nomor klaim bila klaim itu ada DAN sudah tutup.
//
// Dipakai sebelum permintaan dicatat. Mengembalikan `inboxcloseclaim.ErrClaimNotFound` bila
// tidak ada — termasuk bila klaimnya ada tetapi MASIH BERJALAN, dan penyamaan itu disengaja
// (lihat komentar galatnya).
func (r *Repo) ClosedClaimNumber(ctx context.Context, claimID string) (string, error) {
	var number sql.NullString

	err := r.db.QueryRowContext(ctx, query("close_claim_exists"), strings.TrimSpace(claimID)).
		Scan(&number)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return "", inboxcloseclaim.ErrClaimNotFound
	case err != nil:
		return "", fmt.Errorf("inboxcloseclaim/sqlstore: memeriksa klaim tutup: %w", err)
	}
	return strings.TrimSpace(number.String), nil
}

// filterArgs menyusun 22 argumen penyaring, sama untuk kedua kueri.
//
// # Kenapa nilainya dikirim BERULANG
//
// Lini bisnis muncul lima kali di dalam SQL, status transfer tiga kali, status bayar lima
// kali. Masing-masing kemunculan punya NOMOR penanda tersendiri, dan karena itu menuntut
// argumennya sendiri.
//
// Ini bukan kelalaian melainkan pola yang TERBUKTI jalan terhadap Oracle: penanda yang
// diulang dengan nomor yang sama menghasilkan ORA-01008 pada modul Inbox Outstanding.
// Penjelasannya ada di kepala `inboxcloseclaim.sql`.
//
// # Pola "NULL berarti tidak menyaring"
//
// Dipakai supaya SATU teks SQL melayani seluruh kombinasi penyaring. Menyusun WHERE-nya di
// Go akan mengembalikan SQL ke dalam kode — persis yang aturan §4.3 larang.
func filterArgs(f inboxcloseclaim.Filter) []any {
	search := nilIfEmpty(f.Search)
	searchPattern := nilIfEmpty(likePattern(f.Search))

	policy := nilIfEmpty(f.PolicyNumber)
	policyPattern := nilIfEmpty(likePattern(f.PolicyNumber))

	claimNo := nilIfEmpty(f.ClaimNumber)
	claimPattern := nilIfEmpty(likePattern(f.ClaimNumber))

	pic := nilIfEmpty(f.TechnicalPIC)
	picPattern := nilIfEmpty(likePattern(f.TechnicalPIC))

	business := string(f.Business)
	transfer := nilIfEmpty(string(f.Transfer))
	transferValue := string(f.Transfer)
	payment := nilIfEmpty(string(f.Payment))
	paymentValue := string(f.Payment)

	return []any{
		search,        // :1  kotak cari aktif?
		searchPattern, // :2  POLICYNO
		searchPattern, // :3  PYID

		policy,        // :4  penyaring No Polis aktif?
		policyPattern, // :5  POLICYNO

		claimNo,      // :6  penyaring No Klaim aktif?
		claimPattern, // :7  PYID

		pic,        // :8  penyaring PIC aktif?
		picPattern, // :9  USERTEKNIS_1

		business, // :10 ALL
		business, // :11 NONMBU
		business, // :12 BONDING
		business, // :13 PA
		business, // :14 TRAVEL

		transfer,      // :15 penyaring transfer aktif?
		transferValue, // :16 SUDAH TRANSFER
		transferValue, // :17 BELUM TRANSFER

		payment,                         // :18 penyaring bayar aktif?
		paymentValue,                    // :19 LUNAS
		inboxcloseclaim.KodeStatusLunas, // :20 kode '1163'
		paymentValue,                    // :21 BELUM LUNAS
		inboxcloseclaim.KodeStatusLunas, // :22 kode '1163'
	}
}

// likePattern membentuk pola LIKE.
//
// Diseragamkan menjadi huruf besar karena sisi SQL memakai UPPER(...). Perbandingan yang
// hanya satu sisinya diseragamkan tidak pernah cocok, dan gagalnya DIAM — pengguna mengetik
// dengan huruf kecil lalu diberi tahu bahwa klaimnya tidak ada.
//
// Karakter khusus LIKE di-escape supaya pencarian "100%" tidak berubah menjadi pola yang
// mencocokkan apa saja. ESCAPE-nya dinyatakan di sisi SQL.
func likePattern(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(trimmed)
	return "%" + strings.ToUpper(escaped) + "%"
}

// scanClaim membaca satu baris.
//
// Seluruh kolom teks dibaca sebagai sql.NullString: kolomnya nullable di skema warisan, dan
// membacanya langsung ke string akan gagal dengan galat konversi pada baris pertama yang
// kosong.
func scanClaim(rows *sql.Rows) (inboxcloseclaim.ClosedClaim, error) {
	var (
		claimID         string
		number          sql.NullString
		policyNumber    sql.NullString
		insuredName     sql.NullString
		businessName    sql.NullString
		businessSource  sql.NullString
		branchName      sql.NullString
		groupPanel      sql.NullString
		businessGroupID sql.NullString
		createdAt       sql.NullTime
		lossDate        sql.NullTime
		closedAt        sql.NullTime
		resolvedAt      sql.NullTime
		processStatus   sql.NullString
		claimStatus     sql.NullString
		claimStatusText sql.NullString
		technicalPIC    sql.NullString
		adminPNC        sql.NullString
		transferred     sql.NullInt64
	)

	// Urutannya WAJIB sama persis dengan daftar kolom pada close_claim_list. Menambah kolom
	// di SQL tanpa menambahnya di sini menghasilkan galat jumlah kolom; MENUKAR urutannya
	// menghasilkan data yang tertukar TANPA galat — dan di layar ini dua kolom bersebelahan
	// sama-sama bertipe teks nama orang (PIC Teknik dan Admin PNC), sehingga tertukarnya
	// tidak akan terlihat.
	if err := rows.Scan(
		&claimID, &number, &policyNumber, &insuredName, &businessName, &businessSource,
		&branchName, &groupPanel, &businessGroupID,
		&createdAt, &lossDate, &closedAt, &resolvedAt,
		&processStatus, &claimStatus, &claimStatusText,
		&technicalPIC, &adminPNC, &transferred,
	); err != nil {
		return inboxcloseclaim.ClosedClaim{}, err
	}

	claim := inboxcloseclaim.ClosedClaim{
		ClaimID:              claimID,
		ClaimNumber:          strings.TrimSpace(number.String),
		PolicyNumber:         strings.TrimSpace(policyNumber.String),
		InsuredName:          insuredName.String,
		BusinessName:         businessName.String,
		BusinessSource:       businessSource.String,
		BranchName:           branchName.String,
		GroupPanel:           strings.TrimSpace(groupPanel.String),
		BusinessGroupID:      strings.TrimSpace(businessGroupID.String),
		ProcessStatus:        strings.TrimSpace(processStatus.String),
		ClaimStatusCode:      strings.TrimSpace(claimStatus.String),
		ClaimStatusLabel:     strings.TrimSpace(claimStatusText.String),
		TechnicalPIC:         strings.TrimSpace(technicalPIC.String),
		AdminPNC:             strings.TrimSpace(adminPNC.String),
		TransferredToCashier: transferred.Valid && transferred.Int64 == 1,
	}

	if createdAt.Valid {
		claim.RegisteredAt = createdAt.Time.UTC()
	}

	// Salinan lokal pada tiap nilai bertipe pointer: mengambil alamat field struct
	// sql.NullTime akan membuat seluruh baris berbagi pointer yang sama saat di-loop.
	if lossDate.Valid {
		date := lossDate.Time.UTC()
		claim.LossDate = &date
	}
	if closedAt.Valid {
		date := closedAt.Time.UTC()
		claim.ClosedAt = &date
	}
	if resolvedAt.Valid {
		date := resolvedAt.Time.UTC()
		claim.ResolvedAt = &date
	}
	return claim, nil
}

var _ inboxcloseclaim.Repo = (*Repo)(nil)
