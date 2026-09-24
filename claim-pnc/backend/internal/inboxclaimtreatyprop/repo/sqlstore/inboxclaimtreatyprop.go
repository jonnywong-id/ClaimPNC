package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/inboxclaimtreatyprop"
)

// Repo membaca antrean klaim treaty proporsional dari SATU basis data entitas.
//
// Tidak ada satu pun operasi yang menulis. Seluruh tabel yang dibacanya milik sistem lama,
// dan selama masa paralel setiap tabel hanya boleh ditulis satu sistem (`P-1`).
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk pembaca antrean di atas satu koneksi.
func NewRepo(db *sql.DB) *Repo {
	return &Repo{db: db}
}

// plan menyebut kueri mana yang melayani sebuah permintaan dan bagaimana argumennya
// disusun.
//
// Argumen paginasi disusun di sini pula, bukan ditambahkan pemanggil, supaya urutan bind
// setiap kueri hidup di satu tempat bersama namanya — dua kueri dengan jumlah bind berbeda
// adalah tempat paling mudah salah urut.
type plan struct {
	// name adalah nama kueri di berkas .sql.
	name string

	// args menyusun argumen bind sesuai urutan `:1`, `:2`, `:3` di kueri itu.
	args func(q inboxclaimtreatyprop.Query, page inboxclaimtreatyprop.Pagination) []any
}

// planFor memilih kueri yang melayani sebuah permintaan.
//
// # Kenapa pemilihannya fungsi, bukan peta dari kode tab
//
// Karena tab pertama dilayani DUA kueri yang berbeda, dan yang memilih di antara keduanya
// bukan kode tabnya melainkan keadaan checkbox "See All Claim". Peta dari kode tab akan
// menyembunyikan percabangan itu di tempat yang tidak terbaca.
func planFor(q inboxclaimtreatyprop.Query) (plan, error) {
	switch q.Tab.Code {
	case inboxclaimtreatyprop.TabWorkList:
		if q.ScopedToCaller() {
			return plan{
				name: "list_worklist",
				args: func(q inboxclaimtreatyprop.Query, p inboxclaimtreatyprop.Pagination) []any {
					return []any{q.Caller.Login, p.Offset(), p.Normalize().Size}
				},
			}, nil
		}
		return plan{
			name: "list_worklist_all",
			args: func(_ inboxclaimtreatyprop.Query, p inboxclaimtreatyprop.Pagination) []any {
				return []any{p.Offset(), p.Normalize().Size}
			},
		}, nil

	case inboxclaimtreatyprop.TabTechnical:
		return plan{
			name: "list_workbasket",
			args: func(_ inboxclaimtreatyprop.Query, p inboxclaimtreatyprop.Pagination) []any {
				return []any{
					inboxclaimtreatyprop.TechnicalWorkbasket,
					p.Offset(),
					p.Normalize().Size,
				}
			},
		}, nil

	default:
		// Tab terhalang seharusnya sudah ditolak NewQuery. Kalau ia sampai ke sini,
		// yang salah adalah kode — bukan permintaan pengguna — dan galatnya menyebut
		// kodenya alih-alih mengembalikan nol baris yang terbaca seperti antrean kosong.
		return plan{}, fmt.Errorf(
			"inboxclaimtreatyprop/sqlstore: tab %q belum punya kueri", q.Tab.Code)
	}
}

// List mengambil satu halaman baris beserta jumlah seluruh baris yang cocok.
//
// Paginasi dipotong BASIS DATA, bukan di aplikasi — lihat catatan paginasi di kepala
// inboxclaimtreatyprop.sql. Jumlah seluruhnya datang dari kolom TOTAL_ROWS pada baris mana
// pun; ia sama di seluruh baris karena dihitung `COUNT(*) OVER ()`.
func (r *Repo) List(
	ctx context.Context,
	q inboxclaimtreatyprop.Query,
	page inboxclaimtreatyprop.Pagination,
) (inboxclaimtreatyprop.Page, error) {
	selected, err := planFor(q)
	if err != nil {
		return inboxclaimtreatyprop.Page{}, err
	}

	clean := page.Normalize()
	result := inboxclaimtreatyprop.Page{
		Items:      []inboxclaimtreatyprop.WorkItem{},
		Pagination: clean,
	}

	rows, err := r.db.QueryContext(ctx, query(selected.name), selected.args(q, clean)...)
	if err != nil {
		return inboxclaimtreatyprop.Page{},
			fmt.Errorf("menjalankan kueri %s: %w", selected.name, err)
	}
	defer rows.Close()

	for rows.Next() {
		item, total, err := scanWorkItem(rows)
		if err != nil {
			return inboxclaimtreatyprop.Page{},
				fmt.Errorf("membaca baris kueri %s: %w", selected.name, err)
		}
		result.Items = append(result.Items, item)
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxclaimtreatyprop.Page{},
			fmt.Errorf("menelusuri hasil kueri %s: %w", selected.name, err)
	}

	// Halaman kosong menyisakan Total nol, dan itu BENAR untuk halaman pertama yang
	// memang tidak punya baris. Ia TIDAK benar untuk halaman kelima dari antrean
	// berisi tiga baris — tetapi keadaan itu hanya tercapai lewat parameter yang
	// diketik sendiri, dan layar tidak pernah memintanya. Menambah satu kueri
	// penghitung hanya untuk itu berarti satu perjalanan tambahan pada setiap
	// permintaan yang normal.

	return result, nil
}

// CheckTable memastikan ketiga tabel yang disentuh modul ini terbaca dari koneksi yang
// dipakai.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah
// hak baca dan keberadaan tabelnya.
func (r *Repo) CheckTable(ctx context.Context) error {
	var ignored int

	if err := r.db.QueryRowContext(ctx, query("check_worklist")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca DATAPEGA.PC_ASSIGN_WORKLIST atau POOLDATA.JSON_KLAIM: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, query("check_workbasket")).Scan(&ignored); err != nil {
		return fmt.Errorf("membaca DATAPEGA.PC_ASSIGN_WORKBASKET: %w", err)
	}
	return nil
}

// scanner adalah bentuk minimal yang dibutuhkan scanWorkItem, sehingga ia dapat diuji tanpa
// basis data.
type scanner interface {
	Scan(dest ...any) error
}

// scanWorkItem memindai satu baris menjadi WorkItem beserta jumlah seluruh baris.
//
// Urutannya WAJIB sama dengan resultColumns dan dengan urutan kolom di
// inboxclaimtreatyprop.sql. Ketiganya dijaga query_test.go.
//
// Seluruh kolom teks dipindai lewat tipe yang mengizinkan NULL. Itu bukan kehati-hatian
// berlebih: gabungan ke JSON_KLAIM adalah LEFT JOIN, sehingga penugasan yang klaimnya belum
// punya baris di sana mengembalikan NULL pada SELURUH kolom JSON sekaligus.
func scanWorkItem(row scanner) (inboxclaimtreatyprop.WorkItem, int, error) {
	var (
		workKey, reference, claimID, assignedOperator sql.NullString
		masterID, policyNumber, lossDate              sql.NullString
		businessName, businessSource                  sql.NullString
		cedingCompany, insuredName, subjectivity      sql.NullString
		total                                         sql.NullInt64
	)

	err := row.Scan(
		&workKey, &reference, &claimID, &assignedOperator,
		&masterID, &policyNumber, &lossDate,
		&businessName, &businessSource, &cedingCompany, &insuredName,
		&subjectivity, &total,
	)
	if err != nil {
		return inboxclaimtreatyprop.WorkItem{}, 0, err
	}

	return inboxclaimtreatyprop.WorkItem{
		WorkKey:          workKey.String,
		Reference:        reference.String,
		ClaimID:          claimID.String,
		AssignedOperator: assignedOperator.String,
		MasterID:         masterID.String,
		PolicyNumber:     policyNumber.String,
		LossDate:         lossDate.String,
		BusinessName:     businessName.String,
		BusinessSource:   businessSource.String,
		CedingCompany:    cedingCompany.String,
		InsuredName:      insuredName.String,
		Subjectivity:     subjectivity.String,
	}, int(total.Int64), nil
}
