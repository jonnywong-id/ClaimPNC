package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"claim-pnc/internal/inboxprogressclaim"
)

// Repo membaca progres klaim dari SATU basis data entitas.
//
// Tidak ada satu pun operasi yang menulis. Seluruh tabel yang dibacanya milik sistem lama,
// dan selama masa paralel setiap tabel hanya boleh ditulis satu sistem (`P-1`).
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk pembaca progres klaim di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// plan menyebut sepasang kueri yang melayani sebuah region klaim dan bagaimana argumennya
// disusun.
//
// # Kenapa daftar dan pencacah selalu berpasangan
//
// Karena penyaring keduanya WAJIB sama persis. Menaruhnya dalam satu struktur membuat
// pasangan yang tidak sepadan terlihat saat dibaca, alih-alih terlihat sebagai halaman
// terakhir yang kosong tanpa sebab.
type plan struct {
	// list adalah nama kueri daftar di berkas .sql.
	list string

	// count adalah nama kueri pencacah.
	count string

	// filterArgs menyusun argumen penyaring, yaitu bind yang dipakai KEDUA kueri.
	// Paginasi ditambahkan di belakangnya hanya untuk kueri daftar.
	filterArgs func(q inboxprogressclaim.ClaimQuery) []any
}

// plans memetakan kode region klaim ke sepasang kuerinya.
//
// Region yang tidak ada di sini menghasilkan galat yang menyebut kodenya — bukan kueri
// kosong yang mengembalikan nol baris dan terbaca seperti antrean yang memang kosong.
var plans = map[string]plan{
	inboxprogressclaim.ViewOutstanding: {
		list:  "claims_outstanding_list",
		count: "claims_outstanding_count",
		filterArgs: func(q inboxprogressclaim.ClaimQuery) []any {
			return []any{keyword(q.Keyword)}
		},
	},
	inboxprogressclaim.ViewNextFollowUp: {
		list:  "claims_next_fu_list",
		count: "claims_next_fu_count",
		filterArgs: func(q inboxprogressclaim.ClaimQuery) []any {
			return []any{keyword(q.Keyword), q.Today}
		},
	},
}

// keyword menyiapkan kata kunci sebagai nilai yang boleh NULL.
//
// Kata kunci kosong dikirim sebagai NULL, dan kueri menjawabnya dengan `:1 IS NULL` yang
// mematikan seluruh saringan pencarian. Mengirimnya sebagai teks kosong akan membuat
// `LIKE '%%'` — yang kebetulan juga cocok dengan semuanya, tetapi memaksa basis data
// memindai setiap baris alih-alih melewati predikatnya.
func keyword(value string) any {
	if value == "" {
		return nil
	}
	return value
}

// nullableDate menyiapkan tanggal yang boleh kosong sebagai nilai bind.
func nullableDate(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

// ListClaims mengambil satu halaman baris klaim beserta jumlah seluruh baris yang cocok.
//
// Tiga perjalanan ke basis data, dan ketiganya memang diperlukan:
//
//	pencacah  jumlah baris yang cocok, untuk penomoran halaman
//	daftar    satu halaman baris klaim
//	posisi    seluruh posisi berjalan milik baris pada halaman itu
//
// Yang ketiga menggantikan `GET_POSISI_PROGRESS_PNC`, yang di sistem lama dipanggil empat
// kali untuk SETIAP baris — 60 pemanggilan pada satu halaman 15 baris.
func (r *Repo) ListClaims(
	ctx context.Context,
	q inboxprogressclaim.ClaimQuery,
	page inboxprogressclaim.Pagination,
) (inboxprogressclaim.ClaimPage, error) {
	selected, known := plans[q.View]
	if !known {
		return inboxprogressclaim.ClaimPage{}, fmt.Errorf(
			"inboxprogressclaim/sqlstore: bagian %q belum punya kueri", q.View)
	}

	clean := page.Normalize()
	result := inboxprogressclaim.ClaimPage{
		Items:      []inboxprogressclaim.ClaimRow{},
		Pagination: clean,
	}

	filters := selected.filterArgs(q)

	if err := r.db.QueryRowContext(ctx, query(selected.count), filters...).
		Scan(&result.Total); err != nil {
		return inboxprogressclaim.ClaimPage{}, fmt.Errorf(
			"menghitung baris kueri %s: %w", selected.count, err)
	}

	// Halaman di luar jangkauan tidak perlu ditembakkan ke basis data. Totalnya sudah
	// diketahui, dan layar menggambar "halaman n dari m" dari angka itu.
	if result.Total == 0 || clean.Offset() >= result.Total {
		return result, nil
	}

	listArgs := append(append([]any{}, filters...), clean.Offset(), clean.Size)

	rows, err := r.db.QueryContext(ctx, query(selected.list), listArgs...)
	if err != nil {
		return inboxprogressclaim.ClaimPage{}, fmt.Errorf(
			"menjalankan kueri %s: %w", selected.list, err)
	}
	defer rows.Close()

	for rows.Next() {
		item, err := scanClaimRow(rows)
		if err != nil {
			return inboxprogressclaim.ClaimPage{}, fmt.Errorf(
				"membaca baris kueri %s: %w", selected.list, err)
		}
		result.Items = append(result.Items, item)
	}
	if err := rows.Err(); err != nil {
		return inboxprogressclaim.ClaimPage{}, fmt.Errorf(
			"menelusuri hasil kueri %s: %w", selected.list, err)
	}

	if err := r.attachPositions(ctx, result.Items); err != nil {
		return inboxprogressclaim.ClaimPage{}, err
	}

	return result, nil
}

// attachPositions mengisi posisi berjalan setiap baris pada satu halaman.
//
// Ia mengubah `items` di tempat. Baris yang tidak punya posisi berjalan dibiarkan bersenarai
// kosong — bukan nil — supaya penggabungannya di lapisan transport menghasilkan teks kosong,
// bukan panik.
func (r *Repo) attachPositions(
	ctx context.Context,
	items []inboxprogressclaim.ClaimRow,
) error {
	if len(items) == 0 {
		return nil
	}

	numbers := make([]any, 0, len(items))
	index := make(map[string]int, len(items))
	for i := range items {
		items[i].Positions = []inboxprogressclaim.Position{}

		number := items[i].ClaimNumber
		if number == "" {
			continue
		}
		if _, seen := index[number]; seen {
			// Nomor klaim ganda pada satu halaman tidak seharusnya terjadi, tetapi bila
			// terjadi ia tidak boleh membuat bind-nya ganda pula. Posisi akan menempel ke
			// baris pertama saja, dan itu lebih baik daripada kueri yang gagal.
			continue
		}
		index[number] = i
		numbers = append(numbers, number)
	}

	if len(numbers) == 0 {
		return nil
	}

	text := strings.Replace(query("positions"), "%s", inList(1, len(numbers)), 1)

	rows, err := r.db.QueryContext(ctx, text, numbers...)
	if err != nil {
		return fmt.Errorf("menjalankan kueri positions: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			claimNumber sql.NullString
			name        sql.NullString
			status1     sql.NullString
			status2     sql.NullString
			followUp    sql.NullTime
		)

		if err := rows.Scan(&claimNumber, &name, &status1, &status2, &followUp); err != nil {
			return fmt.Errorf("membaca baris kueri positions: %w", err)
		}

		at, exists := index[claimNumber.String]
		if !exists {
			continue
		}

		items[at].Positions = append(items[at].Positions, inboxprogressclaim.Position{
			Name:         name.String,
			Status1:      status1.String,
			Status2:      status2.String,
			NextFollowUp: timeOrNil(followUp),
		})
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("menelusuri hasil kueri positions: %w", err)
	}

	return nil
}

// ListPICSummary mengambil rekap beban dan ketepatan tindak lanjut.
//
// Tanpa paginasi, mengikuti sistem lama: `Activity/GetProgressPerPIC-Act.xml` tidak memuat
// satu pun langkah Pagination, dan barisnya satu per petugas — bukan satu per klaim.
func (r *Repo) ListPICSummary(
	ctx context.Context,
	q inboxprogressclaim.PICQuery,
) ([]inboxprogressclaim.PICSummary, error) {
	rows, err := r.db.QueryContext(ctx, query("pic_summary"),
		string(q.Business),
		q.Caller.Login,
		q.Today,
		nullableDate(q.From),
		nullableDate(q.To),
	)
	if err != nil {
		return nil, fmt.Errorf("menjalankan kueri pic_summary: %w", err)
	}
	defer rows.Close()

	result := []inboxprogressclaim.PICSummary{}
	for rows.Next() {
		var (
			pic      sql.NullString
			claims   sql.NullInt64
			updates  sql.NullInt64
			dueToday sql.NullInt64
			onTime   sql.NullInt64
			late     sql.NullInt64
		)

		if err := rows.Scan(&pic, &claims, &updates, &dueToday, &onTime, &late); err != nil {
			return nil, fmt.Errorf("membaca baris kueri pic_summary: %w", err)
		}

		result = append(result, inboxprogressclaim.PICSummary{
			PIC:           pic.String,
			ClaimCount:    int(claims.Int64),
			UpdateCount:   int(updates.Int64),
			DueTodayCount: int(dueToday.Int64),
			OnTimeCount:   int(onTime.Int64),
			LateCount:     int(late.Int64),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("menelusuri hasil kueri pic_summary: %w", err)
	}

	return result, nil
}

// CheckTable memastikan tabel inti modul ini terbaca dari koneksi yang dipakai.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah
// hak baca dan keberadaan tabelnya.
func (r *Repo) CheckTable(ctx context.Context) error {
	var ignored int
	if err := r.db.QueryRowContext(ctx, query("check_table")).Scan(&ignored); err != nil {
		return fmt.Errorf("membaca POOLDATA.PEGA_DASHBOARDPNC: %w", err)
	}
	return nil
}

// scanner adalah bentuk minimal yang dibutuhkan scanClaimRow, sehingga ia dapat diuji tanpa
// basis data.
type scanner interface {
	Scan(dest ...any) error
}

// scanClaimRow memindai satu baris klaim.
//
// Urutannya WAJIB sama dengan claimColumns dan dengan urutan kolom di
// inboxprogressclaim.sql. Ketiganya dijaga query_test.go.
//
// Seluruh kolom dipindai lewat tipe yang mengizinkan NULL — termasuk nomor klaim, yang
// pada tabel ringkasan warisan tidak dijamin terisi.
func scanClaimRow(row scanner) (inboxprogressclaim.ClaimRow, error) {
	var (
		claimNumber, policyNumber, insuredName sql.NullString
		registerDate, lossDate                 sql.NullTime
		lgbNote, technicalPIC                  sql.NullString
		earliestFollowUp, processDate          sql.NullTime
		prodKe                                 sql.NullString
	)

	err := row.Scan(
		&claimNumber, &policyNumber, &insuredName,
		&registerDate, &lossDate, &lgbNote, &technicalPIC,
		&earliestFollowUp, &processDate, &prodKe,
	)
	if err != nil {
		return inboxprogressclaim.ClaimRow{}, err
	}

	return inboxprogressclaim.ClaimRow{
		ClaimNumber:      claimNumber.String,
		PolicyNumber:     policyNumber.String,
		InsuredName:      insuredName.String,
		RegisterDate:     timeOrNil(registerDate),
		LossDate:         timeOrNil(lossDate),
		LGBNote:          lgbNote.String,
		TechnicalPIC:     technicalPIC.String,
		EarliestFollowUp: timeOrNil(earliestFollowUp),
		ProcessDate:      timeOrNil(processDate),
		ProdKe:           prodKe.String,
		Positions:        []inboxprogressclaim.Position{},
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
