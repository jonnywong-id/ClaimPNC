package sqlstore

import (
	"context"
	"database/sql"
	"fmt"

	"claim-pnc/internal/inboxmanagerreceivepucl"
)

// Repo membaca kedua antrean layar Inbox Manager Receive / PUCL dari SATU basis data
// entitas.
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
// adalah tempat paling mudah salah urut. Di modul ini bedanya nyata: kedua kueri Receive
// memakai tiga bind, kueri RCL/PUCL memakai empat.
type plan struct {
	// name adalah nama kueri di berkas .sql.
	name string

	// args menyusun argumen bind sesuai urutan `:1`, `:2`, … di kueri itu.
	args func(p inboxmanagerreceivepucl.Pagination) []any
}

// planFor memilih kueri yang melayani sebuah permintaan.
//
// # Kenapa pemilihannya fungsi, bukan peta dari kode tab
//
// Karena yang dipilih bukan hanya NAMA kuerinya melainkan juga susunan bind-nya, dan
// keduanya harus berpindah bersama. Peta dari kode tab ke nama kueri akan menyimpan
// separuhnya di satu tempat dan separuh lagi di tempat lain.
func planFor(q inboxmanagerreceivepucl.Query) (plan, error) {
	switch q.Tab.Code {
	case inboxmanagerreceivepucl.TabReceivePA:
		return plan{
			name: "list_receive_pa",
			args: func(p inboxmanagerreceivepucl.Pagination) []any {
				return []any{
					inboxmanagerreceivepucl.GroupPanelPA,
					p.Offset(),
					p.Normalize().Size,
				}
			},
		}, nil

	case inboxmanagerreceivepucl.TabReceiveNonMBU:
		return plan{
			name: "list_receive_non_mbu",
			args: func(p inboxmanagerreceivepucl.Pagination) []any {
				// Bind yang SAMA dengan kueri PA, dan itu disengaja: yang berbeda adalah
				// arah pembandingnya di dalam kueri (`=` versus `<>`), bukan nilainya.
				// Dengan begitu kode Group Panel PA hidup di satu tempat saja.
				return []any{
					inboxmanagerreceivepucl.GroupPanelPA,
					p.Offset(),
					p.Normalize().Size,
				}
			},
		}, nil

	case inboxmanagerreceivepucl.TabRCLPUCL:
		return plan{
			name: "list_rclpucl",
			args: func(p inboxmanagerreceivepucl.Pagination) []any {
				return []any{
					inboxmanagerreceivepucl.WorkStatusCompleted,
					inboxmanagerreceivepucl.RCLPUCLWorkbasket,
					p.Offset(),
					p.Normalize().Size,
				}
			},
		}, nil

	default:
		// Tab yang tidak dikenal seharusnya sudah ditolak NewQuery. Kalau ia sampai ke
		// sini, yang salah adalah kode — bukan permintaan pengguna — dan galatnya menyebut
		// kodenya alih-alih mengembalikan nol baris yang terbaca seperti antrean kosong.
		return plan{}, fmt.Errorf(
			"inboxmanagerreceivepucl/sqlstore: tab %q belum punya kueri", q.Tab.Code)
	}
}

// List mengambil satu halaman baris beserta jumlah seluruh baris yang cocok.
//
// Paginasi dipotong BASIS DATA, bukan di aplikasi — lihat catatan paginasi di kepala
// inboxmanagerreceivepucl.sql. Jumlah seluruhnya datang dari kolom TOTAL_ROWS pada baris
// mana pun; ia sama di seluruh baris karena dihitung `COUNT(*) OVER ()`.
func (r *Repo) List(
	ctx context.Context,
	q inboxmanagerreceivepucl.Query,
	page inboxmanagerreceivepucl.Pagination,
) (inboxmanagerreceivepucl.Page, error) {
	selected, err := planFor(q)
	if err != nil {
		return inboxmanagerreceivepucl.Page{}, err
	}

	clean := page.Normalize()
	result := inboxmanagerreceivepucl.Page{
		Items:      []inboxmanagerreceivepucl.WorkItem{},
		Pagination: clean,
	}

	rows, err := r.db.QueryContext(ctx, query(selected.name), selected.args(clean)...)
	if err != nil {
		return inboxmanagerreceivepucl.Page{},
			fmt.Errorf("menjalankan kueri %s: %w", selected.name, err)
	}
	defer rows.Close()

	for rows.Next() {
		item, total, err := scanWorkItem(rows)
		if err != nil {
			return inboxmanagerreceivepucl.Page{},
				fmt.Errorf("membaca baris kueri %s: %w", selected.name, err)
		}
		result.Items = append(result.Items, item)
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxmanagerreceivepucl.Page{},
			fmt.Errorf("menelusuri hasil kueri %s: %w", selected.name, err)
	}

	// Halaman kosong menyisakan Total nol, dan itu BENAR untuk halaman pertama yang memang
	// tidak punya baris. Ia TIDAK benar untuk halaman kelima dari antrean berisi tiga
	// baris — tetapi keadaan itu hanya tercapai lewat parameter yang diketik sendiri, dan
	// layar tidak pernah memintanya. Menambah satu kueri penghitung hanya untuk itu berarti
	// satu perjalanan tambahan pada setiap permintaan yang normal.

	return result, nil
}

// CheckTable memastikan keempat tabel yang disentuh modul ini terbaca dari koneksi yang
// dipakai.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah
// hak baca dan keberadaan tabelnya.
func (r *Repo) CheckTable(ctx context.Context) error {
	var ignored int

	if err := r.db.QueryRowContext(ctx, query("check_receive")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca DATAPEGA.PC_ASM_FW_GCNMFW_WORK, DATAPEGA.PC_ASSIGN_WORKLIST, "+
				"atau POOLDATA.T_CLAIM_RECIVEDCLAIM: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, query("check_rclpucl")).Scan(&ignored); err != nil {
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
// inboxmanagerreceivepucl.sql. Ketiganya dijaga query_test.go.
//
// Seluruh kolom dipindai lewat tipe yang mengizinkan NULL, dan itu bukan kehati-hatian
// berlebih: kesembilan kolom yang tidak berlaku bagi sebuah tab memang dikirim sebagai
// `CAST(NULL …)`, gabungan ke tabel cermin adalah `LEFT JOIN`, dan penerjemahan
// `RCL_PUCL_1` adalah `CASE` tanpa `ELSE`.
//
// # Kenapa waktu ikut dipindai sebagai teks
//
// `PXCREATEDATETIME` bertipe waktu di basis data, tetapi bentuk yang dikembalikan driver
// bergantung pada tipe kolomnya — dan DDL tabel Pega tidak tersedia (`R-08`). Memindainya
// sebagai `sql.NullString` membuat nilainya sampai ke layar apa adanya alih-alih gagal
// dipindai pada baris pertama di produksi. Pemformatannya dikerjakan layar, dan hanya bila
// bentuknya memang dikenali.
func scanWorkItem(row scanner) (inboxmanagerreceivepucl.WorkItem, int, error) {
	var (
		reference, caseID, policyNumber     sql.NullString
		claimNumber, insuredName, lossDate  sql.NullString
		groupPanel, senderName              sql.NullString
		documentReceivedAt, sheetCount      sql.NullString
		inboxEntryAt, analystNote           sql.NullString
		track, trackStatus, letterPrintedAt sql.NullString
		claimAge, expiryStatus              sql.NullString
		total                               sql.NullInt64
	)

	err := row.Scan(
		&reference, &caseID, &policyNumber, &claimNumber, &insuredName,
		&lossDate, &groupPanel, &senderName, &documentReceivedAt,
		&sheetCount, &inboxEntryAt, &analystNote,
		&track, &trackStatus, &letterPrintedAt, &claimAge, &expiryStatus,
		&total,
	)
	if err != nil {
		return inboxmanagerreceivepucl.WorkItem{}, 0, err
	}

	item := inboxmanagerreceivepucl.WorkItem{
		Reference:            reference.String,
		CaseID:               caseID.String,
		PolicyNumber:         policyNumber.String,
		ClaimNumber:          claimNumber.String,
		InsuredName:          insuredName.String,
		LossDate:             lossDate.String,
		SenderName:           senderName.String,
		DocumentReceivedDate: documentReceivedAt.String,
		DocumentSheetCount:   sheetCount.String,
		InboxEntryAt:         inboxEntryAt.String,
		AnalystNote:          analystNote.String,
		Track:                track.String,
		TrackStatus:          trackStatus.String,
		LetterPrintedAt:      letterPrintedAt.String,
		ClaimAge:             claimAge.String,
		ExpiryStatus:         expiryStatus.String,
	}

	// Jenis Klaim DITURUNKAN di sini, bukan di dalam kueri.
	//
	// Penerjemahannya milik domain (`ClaimTypeOf`), sehingga penyimpanan SQL dan penyimpanan
	// memori menghasilkan teks yang sama persis. Menuliskannya sebagai `CASE` di dalam SQL
	// akan membuat kedua pengisi seam punya dua penerjemah yang dapat menyimpang tanpa
	// ketahuan.
	//
	// Baris tab RCL/PUCL tidak punya Group Panel sama sekali — kolomnya `CAST(NULL …)` di
	// sana — sehingga isian ini dibiarkan kosong alih-alih diisi "NONMBU" yang tidak berarti
	// apa pun bagi sebuah klaim.
	if groupPanel.Valid {
		item.ClaimType = inboxmanagerreceivepucl.ClaimTypeOf(groupPanel.String)
	}

	return item, int(total.Int64), nil
}
