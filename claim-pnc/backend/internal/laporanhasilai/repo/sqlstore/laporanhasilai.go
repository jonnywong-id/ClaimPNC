// Package sqlstore adalah pengisi seam laporanhasilai.Repo terhadap basis data.
//
// Satu instans selalu terikat pada SATU koneksi entitas; pemisahan antarentitas ada di
// tingkat koneksi, bukan di tingkat kueri (`ADR-0030` Opsi 1).
package sqlstore

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/laporanhasilai"
)

//go:embed *.sql
var queryFiles embed.FS

// query memuat seluruh pernyataan SQL modul ini, dikunci dengan namanya.
var query = loadAllQueries()

// Repo membaca POOLDATA.T_CLAIM_DATA_RESULTS_AI yang digabung dengan
// POOLDATA.T_CLAIM_KOMITE_LIST.
//
// Ia TIDAK punya method tulis, dan itu bukan kelalaian — layar lamanya baca-saja.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List mencacah lalu membaca satu halaman.
//
// # Kenapa cacah lebih dulu, bukan sesudah
//
// Supaya halaman di luar jangkauan tidak perlu menembak basis data sama sekali: cacahnya
// sudah di tangan, dan `OFFSET` di luar jangkauan menghasilkan daftar kosong yang sama —
// hanya setelah basis data memindainya lebih dulu.
//
// # Kenapa TIDAK di dalam satu transaksi
//
// Keduanya pembacaan, dan sistem lama pun menjalankannya sebagai pernyataan lepas.
// Membungkusnya dalam transaksi akan menahan kunci baca pada dua tabel yang dibaca setiap
// kali layar dibuka, demi ketepatan cacah yang sudah tidak berarti begitu halamannya
// tergambar.
func (r *Repo) List(
	ctx context.Context,
	filter laporanhasilai.Filter,
	page laporanhasilai.Pagination,
) (laporanhasilai.Page, error) {
	from, to := bounds(filter)

	total, err := r.count(ctx, "report_count", from, to)
	if err != nil {
		return laporanhasilai.Page{}, err
	}

	result := laporanhasilai.Page{Total: total}

	clean := page.Normalize()
	offset := clean.Offset()
	if offset >= total {
		return result, nil
	}

	rows, err := r.db.QueryContext(ctx, getQuery("report_list"), from, to, offset, clean.Size)
	if err != nil {
		return laporanhasilai.Page{}, fmt.Errorf(
			"laporanhasilai/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		one, err := scanRow(rows)
		if err != nil {
			return laporanhasilai.Page{}, err
		}
		result.Rows = append(result.Rows, one)
	}
	if err := rows.Err(); err != nil {
		return laporanhasilai.Page{}, fmt.Errorf(
			"laporanhasilai/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// Summarize mencacah seluruh baris yang cocok, dipecah menurut keputusan AI dan komite.
//
// Satu kueri, bukan lima: kelima angkanya dibaca dari himpunan baris yang sama dalam satu
// kali pindai. Menjalankannya sebagai lima kueri terpisah membuka kemungkinan kelimanya
// dihitung atas keadaan basis data yang berbeda-beda — dan ringkasan yang angkanya tidak
// saling menjumlah adalah ringkasan yang tidak dipercaya siapa pun.
func (r *Repo) Summarize(
	ctx context.Context,
	filter laporanhasilai.Filter,
) (laporanhasilai.Summary, error) {
	from, to := bounds(filter)

	var (
		aiAccepted        sql.NullInt64
		aiRejected        sql.NullInt64
		committeeAccepted sql.NullInt64
		committeeRejected sql.NullInt64
		rowCount          sql.NullInt64
	)

	// SUM menghasilkan NULL bila tidak ada satu baris pun yang cocok, sementara COUNT
	// menghasilkan 0. Keduanya dibaca lewat NullInt64 supaya perbedaan itu tidak menjadi
	// galat pemindaian pada rentang tanggal yang memang kosong — keadaan yang normal,
	// bukan kegagalan.
	err := r.db.QueryRowContext(ctx, getQuery("report_summary"), from, to).Scan(
		&aiAccepted,
		&aiRejected,
		&committeeAccepted,
		&committeeRejected,
		&rowCount,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Kueri agregat tanpa GROUP BY selalu mengembalikan satu baris, sehingga ini
			// mestinya mustahil. Ia ditangani sebagai ringkasan kosong, bukan galat:
			// layar yang kehilangan seluruh isinya karena kejanggalan di satu angka jauh
			// lebih buruk daripada layar yang menampilkan nol.
			return laporanhasilai.Summary{
				Committee: laporanhasilai.Tally{Subject: laporanhasilai.SubjectCommittee},
				AI:        laporanhasilai.Tally{Subject: laporanhasilai.SubjectAI},
			}, nil
		}
		return laporanhasilai.Summary{}, fmt.Errorf(
			"laporanhasilai/sqlstore: membaca ringkasan: %w", err)
	}

	total := int(rowCount.Int64)

	return laporanhasilai.Summary{
		Committee: tallyOf(
			laporanhasilai.SubjectCommittee,
			int(committeeAccepted.Int64),
			int(committeeRejected.Int64),
			total,
		),
		AI: tallyOf(
			laporanhasilai.SubjectAI,
			int(aiAccepted.Int64),
			int(aiRejected.Int64),
			total,
		),
	}, nil
}

// CountAll mencacah seluruh baris pada satu rentang. Dipakai `claimpnc -periksa`.
func (r *Repo) CountAll(ctx context.Context, filter laporanhasilai.Filter) (int, error) {
	from, to := bounds(filter)
	return r.count(ctx, "report_count", from, to)
}

// CheckTable membuktikan kedua tabel beserta seluruh kolom yang dibaca ada dan dapat
// diakses akun aplikasi.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, getQuery("report_check_table"))
	if err != nil {
		return fmt.Errorf("laporanhasilai/sqlstore: memeriksa tabel Laporan Hasil AI: %w", err)
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// count menjalankan satu kueri pencacah.
func (r *Repo) count(ctx context.Context, name string, argument ...any) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, getQuery(name), argument...).Scan(&total); err != nil {
		return 0, fmt.Errorf("laporanhasilai/sqlstore: %s: %w", name, err)
	}
	return total, nil
}

// bounds menyiapkan kedua batas tanggal sebagaimana dikirim ke basis data.
//
// Batas atasnya EKSKLUSIF — lihat doc `laporanhasilai.Filter.ToExclusive`.
func bounds(filter laporanhasilai.Filter) (time.Time, time.Time) {
	clean := filter.Clean()
	return clean.From, clean.ToExclusive()
}

// tallyOf menyusun satu baris ringkasan dari cacah mentahnya.
//
// Menunggu dihitung sebagai SISA, bukan sebagai kondisi tersendiri. Dengan begitu nilai
// tak terduga di kolom statusnya — kode yang tidak dikenal, NULL, spasi — tetap terhitung
// di suatu tempat alih-alih menghilang tanpa jejak.
//
// Sisa yang negatif mustahil secara aritmetika, tetapi tetap dijaga: bila kelak kueri
// ringkasannya berubah dan pembilangnya melampaui cacah barisnya, angka negatif di layar
// akan terbaca sebagai kerusakan data, bukan sebagai kekeliruan kueri.
func tallyOf(subject string, accepted, rejected, rowCount int) laporanhasilai.Tally {
	pending := rowCount - accepted - rejected
	if pending < 0 {
		pending = 0
	}
	return laporanhasilai.Tally{
		Subject:  subject,
		Accepted: accepted,
		Rejected: rejected,
		Pending:  pending,
	}
}

// rowScanner menyatukan *sql.Row dan *sql.Rows.
//
// Keduanya punya Scan dengan tanda tangan yang sama tetapi tidak berbagi interface apa pun
// di pustaka standar, dan tanpa ini pembacaan barisnya harus ditulis dua kali — dua tempat
// yang dapat berbeda urutan kolomnya tanpa satu pun yang memberi tahu.
type rowScanner interface {
	Scan(target ...any) error
}

// scanRow membaca satu baris menjadi laporanhasilai.Row.
//
// Seluruh kolom teks dibaca lewat sql.NullString lalu dipangkas. Dua sebab: kolom bertipe
// CHAR berlebar tetap memadatkan nilainya dengan spasi tanpa memberi tanda apa pun, dan
// baris lama dapat memuat NULL karena kedua tabel ini tidak punya constraint NOT NULL yang
// diketahui (`R-08`).
//
// Urutan kolomnya WAJIB sama dengan urutan SELECT pada report_list dan report_check_table.
// Dijaga query_test.go.
//
// # Lima field sengaja dibiarkan kosong
//
// ObjectName, AcceptNote, RejectNote, CoverageFinal, dan ChronologyCategory tidak diisi
// karena kuerinya memang tidak memilih kolomnya — replikasi keputusan Work Owner
// 2026-09-26. Lihat doc paket `laporanhasilai`.
func scanRow(row rowScanner) (laporanhasilai.Row, error) {
	var (
		committeeID   sql.NullString
		committeeStep sql.NullString
		claimNumber   sql.NullString
		approveCode   sql.NullString
		committeeDate sql.NullTime
		objectID      sql.NullString
		coverageID    sql.NullString
		aiResult      sql.NullString
		aiDate        sql.NullTime
	)

	if err := row.Scan(
		&committeeID,
		&committeeStep,
		&claimNumber,
		&approveCode,
		&committeeDate,
		&objectID,
		&coverageID,
		&aiResult,
		&aiDate,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return laporanhasilai.Row{}, err
		}
		return laporanhasilai.Row{}, fmt.Errorf(
			"laporanhasilai/sqlstore: membaca baris: %w", err)
	}

	caseID := strings.TrimSpace(committeeID.String)
	step := strings.TrimSpace(committeeStep.String)
	code := strings.TrimSpace(approveCode.String)

	return laporanhasilai.Row{
		ID:                  rowKey(caseID, step, objectID.String, coverageID.String),
		ClaimNumber:         claimNumberOf(step, claimNumber.String),
		CommitteeStatus:     laporanhasilai.CommitteeLabel(code),
		CommitteeStatusCode: code,
		CommitteeDate:       committeeDate.Time,
		AIStatus:            strings.TrimSpace(aiResult.String),
		AIDate:              aiDate.Time,
	}, nil
}

// claimNumberOf meniru `CASE WHEN B.KOMITEKE = '1' THEN B.NO_KLAIM ELSE '' END`.
//
// # Kenapa diturunkan di Go, bukan di SQL
//
// Dua sebab, keduanya nyata. Pertama, `KOMITEKE` tetap dibutuhkan utuh untuk menyusun
// kunci baris, sehingga kolomnya tetap dipilih apa pun yang terjadi — dan menaruh `CASE`
// di kueri berarti kolom yang sama dipilih dua kali dalam dua bentuk.
//
// Kedua, aturannya dapat diuji tanpa basis data. Ia satu-satunya aturan tampilan yang
// benar-benar mengubah isi sel di modul ini, dan ia berhak atas uji yang berjalan dalam
// milidetik.
//
// Perbandingannya sebagai TEKS, bukan angka: kueri lama menulis `'1'` dengan tanda kutip,
// dan tipe kolomnya belum diketahui (`R-08`).
func claimNumberOf(step, claimNumber string) string {
	if strings.TrimSpace(step) != "1" {
		return ""
	}
	return strings.TrimSpace(claimNumber)
}

// rowKey menyusun kunci baris dari empat kolom kunci alami.
//
// Pemisahnya `|` karena tidak satu pun dari keempat nilai itu diketahui dapat memuatnya —
// keempatnya pengenal, bukan teks bebas. Tanpa pemisah, "1" + "23" dan "12" + "3"
// menghasilkan kunci yang sama.
func rowKey(committeeID, step, objectID, coverageID string) string {
	return strings.Join([]string{
		committeeID,
		step,
		strings.TrimSpace(objectID),
		strings.TrimSpace(coverageID),
	}, "|")
}

// getQuery mengambil pernyataan SQL menurut namanya.
//
// Ia PANIC bila namanya tidak ada, dan itu disengaja: nama kueri adalah konstanta yang
// ditulis programmer, bukan masukan pengguna. Salah ketik harus gagal saat uji pertama
// dijalankan, bukan menjadi galat runtime di hadapan petugas.
func getQuery(name string) string {
	text, exists := query[name]
	if !exists {
		panic(fmt.Sprintf(
			"laporanhasilai/sqlstore: kueri %q tidak ditemukan di berkas .sql", name))
	}
	return text
}

func loadAllQueries() map[string]string {
	result := map[string]string{}
	list, err := queryFiles.ReadDir(".")
	if err != nil {
		panic("laporanhasilai/sqlstore: tidak dapat membaca berkas kueri: " + err.Error())
	}
	for _, file := range list {
		content, err := queryFiles.ReadFile(file.Name())
		if err != nil {
			panic("laporanhasilai/sqlstore: tidak dapat membaca " +
				file.Name() + ": " + err.Error())
		}
		for name, text := range splitByName(string(content)) {
			if _, clash := result[name]; clash {
				panic("laporanhasilai/sqlstore: nama kueri ganda: " + name)
			}
			result[name] = text
		}
	}
	return result
}

// splitByName memecah isi berkas pada penanda "-- name: <nama>", lalu membuang baris
// komentar dari badan kueri supaya yang dikirim ke basis data hanya pernyataannya.
func splitByName(content string) map[string]string {
	const marker = "-- name:"
	result := map[string]string{}
	name := ""
	var body []string

	save := func() {
		if name == "" {
			return
		}
		var statement []string
		for _, rows := range body {
			if strings.HasPrefix(strings.TrimSpace(rows), "--") {
				continue
			}
			statement = append(statement, rows)
		}
		if text := strings.TrimSpace(strings.Join(statement, "\n")); text != "" {
			result[name] = text
		}
	}

	for _, rows := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(rows)
		if strings.HasPrefix(trimmed, marker) {
			save()
			name = strings.TrimSpace(strings.TrimPrefix(trimmed, marker))
			body = nil
			continue
		}
		body = append(body, rows)
	}
	save()
	return result
}

var _ laporanhasilai.Repo = (*Repo)(nil)
