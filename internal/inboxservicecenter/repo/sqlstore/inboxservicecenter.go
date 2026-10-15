package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxservicecenter"
)

// Repo membaca klaim portal rekanan dari SATU basis data entitas.
//
// Tidak ada satu pun operasi yang menulis. Alasannya bukan gaya melainkan penghalang nyata:
// seluruh jalur tulis layar ini bermuara pada `POOLDATA.PEGA_PORTAL_REKANAN`, stored
// procedure ber-90 parameter yang sumbernya tidak ada di export (`R-01`), sementara `D-02`
// menetapkan logikanya ditulis ulang di Go.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk pembaca klaim di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// List mengambil satu halaman klaim beserta jumlah seluruh baris yang cocok.
//
// # Kenapa menghitung lebih dulu, baru mengambil
//
// Dua alasan, dan keduanya perlu:
//
//   - Layar menggambar "halaman x dari y", dan y menuntut jumlah seluruh baris. Sistem lama
//     pun memakai kueri hitung tersendiri (`RDB List/CountDataServiceCenter-SQL.xml`).
//   - Saat pengguna MENCARI, paginasi dimatikan (lihat Query.Paginated), dan batas
//     `FETCH NEXT` karena itu harus sebesar jumlah baris yang cocok. Tanpa hitungan lebih
//     dulu, angka itu tidak diketahui.
func (r *Repo) List(
	ctx context.Context,
	q inboxservicecenter.Query,
	page inboxservicecenter.Pagination,
) (inboxservicecenter.Page, error) {
	filters := filterArgs(q)

	total, err := r.count(ctx, filters)
	if err != nil {
		return inboxservicecenter.Page{}, err
	}

	clean := page.Normalize()
	paginated := q.Paginated()

	offset, limit := clean.Offset(), clean.Size
	if !paginated {
		// Seluruh baris yang cocok, persis seperti sistem lama saat kotak cari terisi.
		//
		// Batasnya minimal 1: `FETCH NEXT 0 ROWS ONLY` sah di Oracle tetapi tidak di setiap
		// dialek, dan mengirim nol saat hasilnya memang kosong hanya menambah satu cara
		// gagal tanpa menambah satu pun baris.
		offset = 0
		limit = total
		if limit < 1 {
			limit = 1
		}
	}

	result := inboxservicecenter.Page{
		Items:      []inboxservicecenter.ServiceClaim{},
		Total:      total,
		Pagination: clean,
		Paginated:  paginated,
	}

	if total == 0 {
		return result, nil
	}

	args := append(append([]any{}, filters...), offset, limit)

	rows, err := r.db.QueryContext(ctx, query("list_claims"), args...)
	if err != nil {
		return inboxservicecenter.Page{}, fmt.Errorf("menjalankan kueri list_claims: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		claim, err := scanClaim(rows)
		if err != nil {
			return inboxservicecenter.Page{}, fmt.Errorf("membaca baris kueri list_claims: %w", err)
		}
		result.Items = append(result.Items, claim)
	}
	if err := rows.Err(); err != nil {
		return inboxservicecenter.Page{}, fmt.Errorf("menelusuri hasil kueri list_claims: %w", err)
	}

	return result, nil
}

// count menghitung seluruh baris yang cocok dengan penyaring yang sama persis.
func (r *Repo) count(ctx context.Context, filters []any) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, query("count_claims"), filters...).Scan(&total); err != nil {
		return 0, fmt.Errorf("menjalankan kueri count_claims: %w", err)
	}
	return total, nil
}

// CheckTable memastikan tabel inti modul ini terbaca dari koneksi yang dipakai.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah hak
// baca dan keberadaan tabelnya.
func (r *Repo) CheckTable(ctx context.Context) error {
	var ignored int
	if err := r.db.QueryRowContext(ctx, query("check_table")).Scan(&ignored); err != nil {
		return fmt.Errorf("membaca POOLDATA.T_KLAIM_PORTAL_REKANAN: %w", err)
	}
	return nil
}

// filterArgs menyusun ke-15 argumen penyaring, urut sesuai `:1`…`:15`.
//
// Ia dipakai DUA kueri — list_claims dan count_claims — dan itulah yang menjamin keduanya
// menyaring hal yang sama. Menyusunnya dua kali berarti keduanya dapat berselisih tanpa
// ketahuan, dan selisih itu muncul sebagai "halaman 1 dari 7" yang halaman ketujuhnya kosong.
func filterArgs(q inboxservicecenter.Query) []any {
	matchNull := flag(q.Approval.MatchNull)
	matchCodes := flag(len(q.Approval.Codes) > 0)

	// Tab berkode tunggal mengirim kode yang sama dua kali. Bentuk `IN (:3, :4)` dengan
	// jumlah penanda tetap dipilih supaya setiap penanda muncul tepat sekali dan jumlah
	// argumennya tidak pernah berubah — daftar IN yang panjangnya berubah menuntut teks SQL
	// yang dirakit, dan itu jalan kembali ke perangkaian string.
	first, second := codePair(q.Approval.Codes)

	pattern := searchPattern(q.Keyword)
	keyword := nilIfEmpty(pattern)

	return []any{
		matchNull,     // :1
		matchCodes,    // :2
		first, second, // :3 :4
		strings.ToUpper(q.Caller.Login),    // :5
		keyword,                            // :6
		pattern, pattern, pattern, pattern, // :7 :8 :9 :10
		keyword,                            // :11
		pattern, pattern, pattern, pattern, // :12 :13 :14 :15
	}
}

// codePair menyerahkan dua kode status persetujuan yang diterima sebuah tab.
//
// Kosong menghasilkan dua NULL — aman, karena cabang yang memakainya dimatikan penanda `:2`.
func codePair(codes []string) (any, any) {
	switch len(codes) {
	case 0:
		return nil, nil
	case 1:
		return codes[0], codes[0]
	default:
		return codes[0], codes[1]
	}
}

// flag mengubah keadaan menjadi penanda 'Y'/'N' yang dibandingkan SQL.
//
// Teks, bukan angka atau boolean: Oracle tidak punya tipe boolean di SQL, dan driver yang
// berbeda memetakan bool ke hal yang berbeda pula.
func flag(on bool) string {
	if on {
		return "Y"
	}
	return "N"
}

// searchPattern membentuk pola LIKE.
//
// Diseragamkan menjadi huruf besar karena sisi SQL memakai `UPPER(...)`. Perbandingan yang
// hanya satu sisinya diseragamkan tidak pernah cocok, dan gagalnya diam — pengguna mengetik
// dengan huruf kecil lalu diberi tahu bahwa klaimnya tidak ada.
//
// Karakter khusus LIKE di-escape supaya pencarian "100%" tidak berubah menjadi pola yang
// mencocokkan apa saja. ESCAPE-nya dinyatakan di sisi SQL, dan keduanya harus cocok.
func searchPattern(keyword string) string {
	trimmed := strings.TrimSpace(keyword)
	if trimmed == "" {
		return ""
	}
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(trimmed)
	return "%" + strings.ToUpper(escaped) + "%"
}

// nilIfEmpty mengubah teks kosong menjadi NULL, supaya penjaga `:n IS NULL` pada SQL
// mematikan seluruh kelompok penyaring pencarian.
//
// Mengirimnya sebagai teks kosong akan membentuk `LIKE '%%'` — yang kebetulan juga cocok
// dengan semuanya, tetapi memaksa basis data memindai setiap baris alih-alih melewati
// predikatnya.
func nilIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// scanner adalah bentuk minimal yang dibutuhkan scanClaim, sehingga ia dapat diuji tanpa
// basis data.
type scanner interface {
	Scan(dest ...any) error
}

// scanClaim memindai satu baris menjadi ServiceClaim.
//
// Urutannya WAJIB sama dengan resultColumns dan dengan urutan kolom di
// inboxservicecenter.sql. Ketiganya dijaga query_test.go.
//
// Seluruh kolom teks dipindai lewat sql.NullString: kolomnya nullable di skema, dan
// `STS_APPROVAL` yang NULL justru keadaan biasa — itulah isi tab Registrasi SC.
func scanClaim(row scanner) (inboxservicecenter.ServiceClaim, error) {
	var (
		id, repairID, claimNumber, policyNumber sql.NullString
		customerName, claimType, technicalPIC   sql.NullString
		inputDate                               sql.NullTime
		imei, approvalStatus, repairStatus      sql.NullString
		owner, committeeApprover                sql.NullString
	)

	err := row.Scan(
		&id, &repairID, &claimNumber, &policyNumber,
		&customerName, &claimType, &technicalPIC,
		&inputDate, &imei, &approvalStatus, &repairStatus,
		&owner, &committeeApprover,
	)
	if err != nil {
		return inboxservicecenter.ServiceClaim{}, err
	}

	return inboxservicecenter.ServiceClaim{
		ID:                strings.TrimSpace(id.String),
		RepairID:          strings.TrimSpace(repairID.String),
		ClaimNumber:       strings.TrimSpace(claimNumber.String),
		PolicyNumber:      strings.TrimSpace(policyNumber.String),
		CustomerName:      strings.TrimSpace(customerName.String),
		Type:              strings.TrimSpace(claimType.String),
		TechnicalPIC:      strings.TrimSpace(technicalPIC.String),
		InputDate:         timeOrNil(inputDate),
		IMEI:              strings.TrimSpace(imei.String),
		ApprovalStatus:    strings.TrimSpace(approvalStatus.String),
		RepairStatus:      strings.TrimSpace(repairStatus.String),
		Owner:             strings.TrimSpace(owner.String),
		CommitteeApprover: strings.TrimSpace(committeeApprover.String),
	}, nil
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
