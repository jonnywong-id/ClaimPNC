package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/inboxclaimtreatynonprop"
)

// Repo membaca antrean klaim treaty non-proporsional dari SATU basis data entitas.
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
	args func(q inboxclaimtreatynonprop.Query, page inboxclaimtreatynonprop.Pagination) []any
}

// scopedPlan menyusun rencana untuk kueri yang menyaring menurut login pemanggil.
func scopedPlan(name string) plan {
	return plan{
		name: name,
		args: func(
			q inboxclaimtreatynonprop.Query,
			p inboxclaimtreatynonprop.Pagination,
		) []any {
			return []any{q.Caller.Login, p.Offset(), p.Normalize().Size}
		},
	}
}

// openPlan menyusun rencana untuk kueri yang TIDAK menyaring menurut login pemanggil.
func openPlan(name string) plan {
	return plan{
		name: name,
		args: func(
			_ inboxclaimtreatynonprop.Query,
			p inboxclaimtreatynonprop.Pagination,
		) []any {
			return []any{p.Offset(), p.Normalize().Size}
		},
	}
}

// planFor memilih kueri yang melayani sebuah permintaan.
//
// # Kenapa pemilihannya fungsi, bukan peta dari kode tab
//
// Karena tab Admin dilayani EMPAT kueri yang berbeda, dan yang memilih di antaranya bukan
// kode tabnya melainkan kombinasi dua checkbox. Peta dari kode tab akan menyembunyikan
// percabangan itu di tempat yang tidak terbaca.
//
// Keempat kombinasinya ditulis lengkap, tanpa satu pun yang disimpulkan dari yang lain:
//
//	milik saya, polis apa adanya   list_admin
//	semua,      polis apa adanya   list_admin_all
//	milik saya, hanya TBA          list_admin_tba       <- tidak ada di sistem lama
//	semua,      hanya TBA          list_admin_all_tba
func planFor(q inboxclaimtreatynonprop.Query) (plan, error) {
	switch q.Tab.Code {
	case inboxclaimtreatynonprop.TabAdmin:
		switch {
		case q.ScopedToCaller() && q.TBAOnly:
			return scopedPlan("list_admin_tba"), nil
		case q.ScopedToCaller():
			return scopedPlan("list_admin"), nil
		case q.TBAOnly:
			return openPlan("list_admin_all_tba"), nil
		default:
			return openPlan("list_admin_all"), nil
		}

	case inboxclaimtreatynonprop.TabTechnical:
		return plan{
			name: "list_technical",
			args: func(
				_ inboxclaimtreatynonprop.Query,
				p inboxclaimtreatynonprop.Pagination,
			) []any {
				return []any{
					inboxclaimtreatynonprop.TechnicalWorkbasket,
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
			"inboxclaimtreatynonprop/sqlstore: tab %q belum punya kueri", q.Tab.Code)
	}
}

// List mengambil satu halaman baris beserta jumlah seluruh baris yang cocok.
//
// Paginasi dipotong BASIS DATA, bukan di aplikasi — lihat catatan paginasi di kepala
// inboxclaimtreatynonprop.sql. Jumlah seluruhnya datang dari kolom TOTAL_ROWS pada baris
// mana pun; ia sama di seluruh baris karena dihitung `COUNT(*) OVER ()`.
func (r *Repo) List(
	ctx context.Context,
	q inboxclaimtreatynonprop.Query,
	page inboxclaimtreatynonprop.Pagination,
) (inboxclaimtreatynonprop.Page, error) {
	selected, err := planFor(q)
	if err != nil {
		return inboxclaimtreatynonprop.Page{}, err
	}

	clean := page.Normalize()
	result := inboxclaimtreatynonprop.Page{
		Items:      []inboxclaimtreatynonprop.WorkItem{},
		Pagination: clean,
	}

	rows, err := r.db.QueryContext(ctx, query(selected.name), selected.args(q, clean)...)
	if err != nil {
		return inboxclaimtreatynonprop.Page{},
			fmt.Errorf("menjalankan kueri %s: %w", selected.name, err)
	}
	defer rows.Close()

	for rows.Next() {
		item, total, err := scanWorkItem(rows)
		if err != nil {
			return inboxclaimtreatynonprop.Page{},
				fmt.Errorf("membaca baris kueri %s: %w", selected.name, err)
		}
		result.Items = append(result.Items, item)
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxclaimtreatynonprop.Page{},
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

// CheckTable memastikan keempat tabel yang disentuh modul ini terbaca dari koneksi yang
// dipakai.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah
// hak baca dan keberadaan tabelnya.
func (r *Repo) CheckTable(ctx context.Context) error {
	var ignored int

	if err := r.db.QueryRowContext(ctx, query("check_admin")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca DATAPEGA.PC_ASSIGN_WORKLIST, DATAPEGA.PC_ASM_FW_GCNMFW_WORK, "+
				"atau POOLDATA.JSON_KLAIM: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, query("check_technical")).Scan(&ignored); err != nil {
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
// inboxclaimtreatynonprop.sql. Ketiganya dijaga query_test.go.
//
// Seluruh kolom teks dipindai lewat tipe yang mengizinkan NULL. Itu bukan kehati-hatian
// berlebih: kedua gabungan adalah LEFT JOIN, sehingga penugasan yang objek kerjanya atau
// baris JSON_KLAIM-nya belum ada mengembalikan NULL pada seluruh kolom tabel itu sekaligus.
//
// AGING_DAYS ikut dipindai lewat tipe ber-NULL dengan alasan yang sama: ia dihitung dari
// `b.PXCREATEDATETIME`, sehingga baris tanpa pasangan objek kerja menghasilkan NULL, bukan
// nol. Keduanya dibedakan di layar — umur nol hari berbeda artinya dari umur yang tidak
// diketahui.
func scanWorkItem(row scanner) (inboxclaimtreatynonprop.WorkItem, int, error) {
	var (
		reference, claimID, assignedOperator   sql.NullString
		masterID, jsonMasterID, policyNumber   sql.NullString
		lossDate, businessName, businessSource sql.NullString
		cedingCompany, insuredName, status     sql.NullString
		createOperator, lastUpdateOperator     sql.NullString
		agingDays                              sql.NullInt64
		total                                  sql.NullInt64
	)

	err := row.Scan(
		&reference, &claimID, &assignedOperator,
		&masterID, &jsonMasterID, &policyNumber, &lossDate,
		&businessName, &businessSource, &cedingCompany, &insuredName,
		&status, &agingDays, &createOperator, &lastUpdateOperator,
		&total,
	)
	if err != nil {
		return inboxclaimtreatynonprop.WorkItem{}, 0, err
	}

	return inboxclaimtreatynonprop.WorkItem{
		Reference:          reference.String,
		ClaimID:            claimID.String,
		AssignedOperator:   assignedOperator.String,
		MasterID:           masterID.String,
		JSONMasterID:       jsonMasterID.String,
		PolicyNumber:       policyNumber.String,
		LossDate:           lossDate.String,
		BusinessName:       businessName.String,
		BusinessSource:     businessSource.String,
		CedingCompany:      cedingCompany.String,
		InsuredName:        insuredName.String,
		Status:             status.String,
		AgingDays:          int(agingDays.Int64),
		CreateOperator:     createOperator.String,
		LastUpdateOperator: lastUpdateOperator.String,
	}, int(total.Int64), nil
}
