package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"claim-pnc/internal/inboxbandinghargasalvage"
)

// Repo membaca antrean banding harga salvage dari SATU basis data entitas.
//
// Tidak ada satu pun operasi yang menulis — lihat kepala inboxbandinghargasalvage.sql.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk pembaca antrean di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// List mengambil satu halaman baris beserta jumlah seluruh baris yang cocok.
//
// # Kenapa menghitung lebih dulu, baru mengambil
//
// Layar menggambar "halaman x dari y", dan y menuntut jumlah seluruh baris. Sistem lama pun
// memakai kueri hitung tersendiri, dan menyimpan hasilnya di `Pagination.TotalData`
// (`Activity/SetReqSalvage_Act-Act.xml` langkah 2).
//
// Menghitung lebih dulu juga menghindari satu perjalanan sia-sia: bila tidak ada satu baris
// pun yang cocok, kueri daftarnya tidak perlu dijalankan sama sekali.
func (r *Repo) List(
	ctx context.Context,
	q inboxbandinghargasalvage.Query,
	page inboxbandinghargasalvage.Pagination,
) (inboxbandinghargasalvage.Page, error) {
	total, err := r.Count(ctx, q)
	if err != nil {
		return inboxbandinghargasalvage.Page{}, err
	}

	clean := page.Normalize()
	result := inboxbandinghargasalvage.Page{
		Items:      []inboxbandinghargasalvage.AppealRow{},
		Total:      total,
		Pagination: clean,
	}
	if total == 0 {
		return result, nil
	}

	name, scan := listQueryFor(q.Tab)
	args := append(filterArgs(q), clean.Offset(), clean.Size)

	rows, err := r.db.QueryContext(ctx, query(name), args...)
	if err != nil {
		return inboxbandinghargasalvage.Page{}, fmt.Errorf("menjalankan kueri %s: %w", name, err)
	}
	defer rows.Close()

	for rows.Next() {
		row, err := scan(rows)
		if err != nil {
			return inboxbandinghargasalvage.Page{},
				fmt.Errorf("membaca baris kueri %s: %w", name, err)
		}
		result.Items = append(result.Items, row)
	}
	if err := rows.Err(); err != nil {
		return inboxbandinghargasalvage.Page{},
			fmt.Errorf("menelusuri hasil kueri %s: %w", name, err)
	}

	return result, nil
}

// Count menghitung seluruh baris yang cocok dengan penyaring sebuah tab.
//
// Ia dipakai DUA tempat — List di atas, dan tabel ringkas "Status Salvage / Jumlah" — dan
// itulah yang menjamin angka ringkas sama dengan jumlah baris gridnya. Menyusunnya dua kali
// berarti keduanya dapat berselisih tanpa ketahuan, dan selisih itulah yang justru sedang
// diperbaiki dari sistem lama (lihat selisih 3 di kepala berkas .sql).
func (r *Repo) Count(
	ctx context.Context,
	q inboxbandinghargasalvage.Query,
) (int, error) {
	name := countQueryFor(q.Tab)

	var total int
	if err := r.db.QueryRowContext(ctx, query(name), filterArgs(q)...).Scan(&total); err != nil {
		return 0, fmt.Errorf("menjalankan kueri %s: %w", name, err)
	}
	return total, nil
}

// CheckTable memastikan tabel inti modul ini terbaca dari koneksi yang dipakai, beserta
// setiap kolom yang dipakainya.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun.
//
// Pemeriksaannya sengaja menyebut kolom satu per satu: DDL `T_CLAIM_CHEKER_SALVAGE` belum
// pernah dibaca, sehingga seluruh nama kolom modul ini disimpulkan dari teks kueri Pega — dan
// satu di antaranya pernah KELIRU disimpulkan dari nama propertinya. Bila salah satunya tidak ada, galatnya
// muncul saat start lengkap dengan nama kolom yang salah, bukan sebagai layar kosong yang
// dilaporkan pengguna.
func (r *Repo) CheckTable(ctx context.Context) error {
	var ignored int
	if err := r.db.QueryRowContext(ctx, query("check_table")).Scan(&ignored); err != nil {
		return fmt.Errorf("membaca POOLDATA.T_CLAIM_CHEKER_SALVAGE: %w", err)
	}
	return nil
}

// WriteTargetColumns adalah jumlah kolom yang diharapkan ditemukan CheckWriteTargets.
//
// Enam: tiga pada `SALAVAGEDOCUMENT` dan tiga pada `DETAIL_PNC_SALVAGE`. Angkanya diekspor
// supaya pemeriksa kesiapan dapat menyebut berapa yang kurang, bukan sekadar "tidak lengkap".
const WriteTargetColumns = 6

// CheckWriteTargets memastikan kedua tabel yang DITULIS tombol Approve/Reject punya kolom
// yang ditulisnya. Ia mengembalikan jumlah kolom yang ditemukan.
//
// Tabel inti modul ini tidak ikut diperiksa di sini — CheckTable sudah menyentuh seluruh
// kolomnya, termasuk ketiga kolom yang ditulis putusan.
//
// # Kenapa ia membaca katalog, bukan mencoba menulis
//
// Karena menguji hak tulis dengan benar-benar menulis akan meninggalkan baris percobaan di
// tabel produksi. Yang dapat dipastikan tanpa menulis adalah keberadaan kolomnya; hak
// tulisnya sendiri tetap harus dipastikan DBA, dan pemeriksa kesiapan menyatakan itu.
func (r *Repo) CheckWriteTargets(ctx context.Context) (int, error) {
	var found int
	err := r.db.QueryRowContext(ctx, query("check_write_targets")).Scan(&found)
	if err != nil {
		return 0, fmt.Errorf("membaca katalog kolom tabel tujuan penulisan: %w", err)
	}
	return found, nil
}

// rowScanner memindai satu baris hasil menjadi AppealRow.
type rowScanner func(row scanner) (inboxbandinghargasalvage.AppealRow, error)

// listQueryFor menyerahkan nama kueri daftar sebuah tab beserta pemindainya.
//
// Keduanya diserahkan BERSAMAAN supaya kueri dan pemindainya tidak dapat berpasangan salah —
// kekeliruan yang tidak menghasilkan galat kompilasi dan baru terbaca sebagai kolom yang
// tertukar isinya di layar.
func listQueryFor(tab inboxbandinghargasalvage.Tab) (string, rowScanner) {
	if tab.Decided {
		return "list_history", scanHistoryRow
	}
	return "list_request", scanRequestRow
}

// countQueryFor menyerahkan nama kueri pencacah sebuah tab.
func countQueryFor(tab inboxbandinghargasalvage.Tab) string {
	if tab.Decided {
		return "count_history"
	}
	return "count_request"
}

// filterArgs menyusun argumen penyaring, urut sesuai penanda pada kuerinya.
//
// Jumlahnya berbeda antar tab — lima untuk Request, tiga untuk History — karena penyaring
// giliran komite hanya ada pada yang pertama. `HistoryReqSalvage_SQL` memang tidak
// memuatnya.
//
// Ia dipakai kueri daftar DAN kueri pencacah, dan itulah yang menjamin keduanya menyaring
// hal yang sama.
func filterArgs(q inboxbandinghargasalvage.Query) []any {
	keyword := nilIfEmpty(q.Keyword)

	if q.Tab.Decided {
		return []any{
			strings.ToUpper(q.Reviewer.Name), // :1
			keyword,                          // :2
			keyword,                          // :3
		}
	}

	waitFor := nilIfEmpty(strings.ToUpper(strings.TrimSpace(q.Reviewer.WaitFor)))

	return []any{
		strings.ToUpper(q.Reviewer.Name), // :1
		keyword,                          // :2
		keyword,                          // :3
		waitFor,                          // :4
		waitFor,                          // :5
	}
}

// nilIfEmpty mengubah teks kosong menjadi NULL, supaya penjaga `:n IS NULL` pada SQL
// mematikan penyaring yang bersangkutan.
//
// Mengirimnya sebagai teks kosong akan membuat penyaringnya TETAP berjalan dan mencocokkan
// nomor klaim dengan teks kosong — yang tidak pernah cocok, sehingga layarnya kosong tanpa
// alasan yang terbaca.
func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// scanner adalah bentuk minimal yang dibutuhkan pemindai, sehingga keduanya dapat diuji tanpa
// basis data.
type scanner interface {
	Scan(dest ...any) error
}

// scanRequestRow memindai satu baris grid "Request Banding Harga".
//
// Urutannya WAJIB sama dengan requestColumns dan dengan urutan kolom pada list_request.
// Ketiganya dijaga query_test.go.
//
// Seluruh kolom teks dipindai lewat sql.NullString: kolomnya nullable, dan `NOTEAPPROVE` yang
// NULL justru keadaan biasa — ia baru terisi setelah komite menuliskan catatannya.
func scanRequestRow(row scanner) (inboxbandinghargasalvage.AppealRow, error) {
	var (
		claimNo, detailObject, itemName       sql.NullString
		itemPrice, requestPrice, requestNote  sql.NullString
		checkerNote, salvageID, committeeName sql.NullString
		requestDate                           sql.NullTime
		agingDays                             sql.NullFloat64
	)

	err := row.Scan(
		&claimNo, &requestDate, &detailObject, &itemName,
		&itemPrice, &requestPrice, &requestNote, &agingDays,
		&checkerNote, &salvageID, &committeeName,
	)
	if err != nil {
		return inboxbandinghargasalvage.AppealRow{}, err
	}

	return inboxbandinghargasalvage.AppealRow{
		ClaimNo:       strings.TrimSpace(claimNo.String),
		RequestDate:   timeOrNil(requestDate),
		DetailObject:  strings.TrimSpace(detailObject.String),
		ItemName:      strings.TrimSpace(itemName.String),
		ItemPrice:     strings.TrimSpace(itemPrice.String),
		RequestPrice:  strings.TrimSpace(requestPrice.String),
		RequestNote:   strings.TrimSpace(requestNote.String),
		AgingDays:     wholeDays(agingDays),
		CheckerNote:   strings.TrimSpace(checkerNote.String),
		SalvageID:     strings.TrimSpace(salvageID.String),
		CommitteeName: strings.TrimSpace(committeeName.String),
	}, nil
}

// scanHistoryRow memindai satu baris grid "History Cheker".
//
// Urutannya WAJIB sama dengan historyColumns dan dengan urutan kolom pada list_history.
func scanHistoryRow(row scanner) (inboxbandinghargasalvage.AppealRow, error) {
	var claimNo, salvageType, salvageLocation, pic sql.NullString

	if err := row.Scan(&claimNo, &salvageType, &salvageLocation, &pic); err != nil {
		return inboxbandinghargasalvage.AppealRow{}, err
	}

	return inboxbandinghargasalvage.AppealRow{
		ClaimNo:         strings.TrimSpace(claimNo.String),
		SalvageType:     strings.TrimSpace(salvageType.String),
		SalvageLocation: strings.TrimSpace(salvageLocation.String),
		PIC:             strings.TrimSpace(pic.String),
	}, nil
}

// wholeDays mengubah selisih hari menjadi bilangan bulat.
//
// # Kenapa dipindai sebagai pecahan lalu dibulatkan
//
// Karena `TRUNC(SYSDATE) - TRUNC(TGLREQUEST)` bertipe NUMBER di Oracle, dan driver memetakan
// NUMBER ke float64. Memindainya langsung ke int membuat kueri gagal pada driver yang
// menyerahkannya sebagai pecahan — meski nilainya selalu bulat, sebab kedua sisinya sudah
// dipotong ke hari.
//
// NULL menjadi nol, dan itu keadaan yang sah: baris yang `TGLREQUEST`-nya kosong tidak punya
// umur yang dapat dihitung. Ia tidak dapat tertukar dengan "baru saja masuk", karena baris
// hari ini pun bernilai nol — dan keduanya memang sama-sama berarti "belum ada yang
// menunggu lama".
func wholeDays(value sql.NullFloat64) int {
	if !value.Valid {
		return 0
	}
	return int(math.Round(value.Float64))
}

// timeOrNil mengubah kolom tanggal yang boleh NULL menjadi pointer.
//
// Pointer, bukan time.Time kosong: tanggal nol tahun 1 tidak dapat dibedakan dari "belum
// diisi" saat ditampilkan.
func timeOrNil(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	at := value.Time
	return &at
}

// ListDecisions mengambil keputusan banding harga satu klaim.
//
// Tidak berhalaman, dan itu disengaja: panel rincian menampilkan keputusan atas satu klaim
// saja, dan jumlahnya dibatasi banyaknya barang salvage pada klaim itu. Kueri aslinya pun
// tidak memaginasinya.
func (r *Repo) ListDecisions(
	ctx context.Context,
	q inboxbandinghargasalvage.DecisionQuery,
) ([]inboxbandinghargasalvage.Decision, error) {
	rows, err := r.db.QueryContext(ctx, query("list_decisions"),
		strings.ToUpper(q.ClaimNo), strings.ToUpper(q.Reviewer.Name))
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri list_decisions: %w", err)
	}
	defer rows.Close()

	// Senarai kosong, bukan nil: klaim tanpa keputusan adalah keadaan yang SAH di panel
	// ini, dan `nil` memaksa setiap pemanggil memeriksanya lebih dulu.
	result := []inboxbandinghargasalvage.Decision{}

	for rows.Next() {
		decision, err := scanDecision(rows)
		if err != nil {
			return nil, fmt.Errorf("membaca baris kueri list_decisions: %w", err)
		}
		result = append(result, decision)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri list_decisions: %w", err)
	}

	return result, nil
}

// scanDecision memindai satu baris panel rincian.
//
// Urutannya WAJIB sama dengan decisionColumns pada query.go dan dengan urutan kolom pada
// list_decisions. Ketiganya dijaga query_test.go.
//
// `STATUSAPPROVE` dipindai sebagai TEKS meski kuerinya membandingkannya sebagai angka
// (`STATUSAPPROVE = 1`), sehingga kolomnya kemungkinan NUMBER. Driver menyerahkan NUMBER
// sebagai pecahan, dan `DecisionLabel` merapikan bentuk `"1.0"` yang mungkin muncul — lihat
// normalizeDecisionCode.
func scanDecision(row scanner) (inboxbandinghargasalvage.Decision, error) {
	var (
		approvedAt                          sql.NullTime
		detailObject, itemName, itemPrice   sql.NullString
		requestPrice, status, committeeName sql.NullString
	)

	err := row.Scan(
		&approvedAt, &detailObject, &itemName, &itemPrice,
		&requestPrice, &status, &committeeName,
	)
	if err != nil {
		return inboxbandinghargasalvage.Decision{}, err
	}

	return inboxbandinghargasalvage.Decision{
		ApprovedAt:    timeOrNil(approvedAt),
		DetailObject:  strings.TrimSpace(detailObject.String),
		ItemName:      strings.TrimSpace(itemName.String),
		ItemPrice:     strings.TrimSpace(itemPrice.String),
		RequestPrice:  strings.TrimSpace(requestPrice.String),
		Status:        strings.TrimSpace(status.String),
		CommitteeName: strings.TrimSpace(committeeName.String),
	}, nil
}
