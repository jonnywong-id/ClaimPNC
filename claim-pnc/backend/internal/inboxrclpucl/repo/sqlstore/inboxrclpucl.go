package sqlstore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"claim-pnc/internal/inboxrclpucl"
)

// Repo membaca antrean RCL/PUCL dari SATU basis data entitas.
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
// setiap kueri hidup di satu tempat bersama namanya. Di modul ini bedanya nyata: dua kueri
// memakai enam bind, satu memakai tujuh.
type plan struct {
	// name adalah nama kueri di berkas .sql.
	name string

	// args menyusun argumen bind sesuai urutan `:1`, `:2`, … di kueri itu.
	args func(p inboxrclpucl.Pagination) []any
}

// planFor memilih kueri yang melayani sebuah permintaan.
//
// # Kenapa pemilihannya fungsi, bukan peta dari kode tab
//
// Karena yang dipilih bukan hanya NAMA kuerinya melainkan juga susunan bind-nya, dan
// keduanya harus berpindah bersama. Peta dari kode tab ke nama kueri akan menyimpan
// separuhnya di satu tempat dan separuh lagi di tempat lain.
//
// # Kenapa nilai penyaringnya diikat, bukan ditulis di dalam SQL
//
// Ketiga nilai — akun antrean bersama, status kerja yang dikecualikan, dan penanda
// persetujuan — adalah NILAI BISNIS, dan `D-15` melarangnya tertanam di dalam kode. Di sini
// alasannya lebih tajam daripada kerapian: satu nilai yang salah mengosongkan seluruh layar
// tanpa satu pun galat, dan tidak ada apa pun di antarmuka yang menandakannya.
func planFor(q inboxrclpucl.Query) (plan, error) {
	switch q.Tab.Code {
	case inboxrclpucl.TabCetakSurat:
		return plan{
			name: "list_cetak_surat",
			args: func(p inboxrclpucl.Pagination) []any {
				return []any{
					inboxrclpucl.WorkClassClaim,
					inboxrclpucl.RCLPUCLWorkbasket,
					inboxrclpucl.WorkStatusCompleted,
					inboxrclpucl.ExpiryStatusActive,
					p.Offset(),
					p.Normalize().Size,
				}
			},
		}, nil

	case inboxrclpucl.TabKelengkapanDokumen:
		return plan{
			name: "list_kelengkapan_dokumen",
			args: func(p inboxrclpucl.Pagination) []any {
				return []any{
					inboxrclpucl.WorkClassClaim,
					inboxrclpucl.RCLPUCLWorkbasket,
					inboxrclpucl.WorkStatusCompleted,
					inboxrclpucl.PUCLApproved,
					p.Offset(),
					p.Normalize().Size,
				}
			},
		}, nil

	case inboxrclpucl.TabKlaimMSIG:
		return plan{
			name: "list_klaim_msig",
			args: func(p inboxrclpucl.Pagination) []any {
				// Satu bind LEBIH BANYAK daripada kueri di atasnya, dan itulah satu-satunya
				// perbedaannya: penanda jalur MSIG. Urutannya disisipkan SEBELUM paginasi,
				// mengikuti urutan `:n` di berkas .sql.
				return []any{
					inboxrclpucl.WorkClassClaim,
					inboxrclpucl.RCLPUCLWorkbasket,
					inboxrclpucl.WorkStatusCompleted,
					inboxrclpucl.PUCLApproved,
					inboxrclpucl.MSIGMarker,
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
			"inboxrclpucl/sqlstore: tab %q belum punya kueri", q.Tab.Code)
	}
}

// List mengambil satu halaman baris beserta jumlah seluruh baris yang cocok.
//
// Paginasi dipotong BASIS DATA, bukan di aplikasi — lihat catatan paginasi di kepala
// inboxrclpucl.sql. Jumlah seluruhnya datang dari kolom TOTAL_ROWS pada baris mana pun; ia
// sama di seluruh baris karena dihitung `COUNT(*) OVER ()`.
func (r *Repo) List(
	ctx context.Context,
	q inboxrclpucl.Query,
	page inboxrclpucl.Pagination,
) (inboxrclpucl.Page, error) {
	selected, err := planFor(q)
	if err != nil {
		return inboxrclpucl.Page{}, err
	}

	clean := page.Normalize()
	result := inboxrclpucl.Page{
		Items:      []inboxrclpucl.WorkItem{},
		Pagination: clean,
	}

	rows, err := r.db.QueryContext(ctx, query(selected.name), selected.args(clean)...)
	if err != nil {
		return inboxrclpucl.Page{},
			fmt.Errorf("menjalankan kueri %s: %w", selected.name, err)
	}
	defer rows.Close()

	for rows.Next() {
		item, total, err := scanWorkItem(rows)
		if err != nil {
			return inboxrclpucl.Page{},
				fmt.Errorf("membaca baris kueri %s: %w", selected.name, err)
		}
		result.Items = append(result.Items, item)
		result.Total = total
	}
	if err := rows.Err(); err != nil {
		return inboxrclpucl.Page{},
			fmt.Errorf("menelusuri hasil kueri %s: %w", selected.name, err)
	}

	// Halaman kosong menyisakan Total nol, dan itu BENAR untuk halaman pertama yang memang
	// tidak punya baris. Ia TIDAK benar untuk halaman kelima dari antrean berisi tiga
	// baris — tetapi keadaan itu hanya tercapai lewat parameter yang diketik sendiri, dan
	// layar tidak pernah memintanya. Menambah satu kueri penghitung hanya untuk itu berarti
	// satu perjalanan tambahan pada setiap permintaan yang normal.

	return result, nil
}

// DailyReport mengambil satu halaman LAPORAN HARIAN RCL/PUCL.
//
// Ia terpisah dari List karena kuerinya memang berbeda — bukan hanya penyaringnya melainkan
// kolomnya, gabungannya, dan jumlah tabel yang dibacanya. Lihat catatan pada
// inboxrclpucl.DailyReportRow dan pada kueri `daily_report`.
//
// Rentang tanggal DIIKAT DUA KALI karena kueri ber-`UNION` dan kedua cabangnya menyaring
// rentang yang sama. Menulis penanda bind yang sama dua kali akan bergantung pada cara
// driver menafsirkan penanda berulang — perbedaan yang tidak terlihat saat membaca kueri,
// dan yang akibatnya adalah rentang tanggal yang salah pada salah satu cabang.
func (r *Repo) DailyReport(
	ctx context.Context,
	rng inboxrclpucl.DateRange,
	page inboxrclpucl.Pagination,
) ([]inboxrclpucl.DailyReportRow, int, error) {
	clean := page.Normalize()
	rows := []inboxrclpucl.DailyReportRow{}
	total := 0

	cursor, err := r.db.QueryContext(ctx, query("daily_report"),
		// Cabang antrean bersama.
		inboxrclpucl.WorkClassClaim,
		inboxrclpucl.RCLPUCLWorkbasket,
		rng.From,
		rng.To,
		// Cabang Personal Accident — TANPA gabungan antrean bersama, mengikuti kueri lama.
		inboxrclpucl.WorkClassClaim,
		inboxrclpucl.GroupPanelPA,
		rng.From,
		rng.To,
		// Paginasi.
		clean.Offset(),
		clean.Size,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("menjalankan kueri daily_report: %w", err)
	}
	defer cursor.Close()

	for cursor.Next() {
		row, rowTotal, err := scanReportRow(cursor)
		if err != nil {
			return nil, 0, fmt.Errorf("membaca baris kueri daily_report: %w", err)
		}
		rows = append(rows, row)
		total = rowTotal
	}
	if err := cursor.Err(); err != nil {
		return nil, 0, fmt.Errorf("menelusuri hasil kueri daily_report: %w", err)
	}

	return rows, total, nil
}

// Detail mengambil isi layar kerja RCL/PUCL untuk satu klaim.
//
// # Kenapa kelas objek kerja ikut disaring
//
// Karena `PZINSKEY` memang unik, tetapi penyaring kelas menutup satu kelas kekeliruan yang
// tidak menghasilkan galat: kunci milik kelas objek kerja LAIN yang kebetulan sampai ke
// sini akan mengembalikan baris yang kolom PUCL-nya seluruhnya kosong — terbaca persis
// seperti klaim RCL/PUCL yang belum diisi.
//
// # Kenapa "tidak ditemukan" dibedakan dari "kosong"
//
// Klaim yang tidak ada dan klaim yang seluruh isiannya kosong terlihat SAMA di layar, dan
// hanya yang pertama yang merupakan kekeliruan. Yang paling mungkin menyebabkannya: kunci
// yang benar dibuka pada PORTAL YANG SALAH — dan itu keterangan yang harus sampai ke
// pengguna, bukan layar kosong tanpa sebab (`R-20`).
func (r *Repo) Detail(
	ctx context.Context,
	reference string,
) (inboxrclpucl.ClaimDetail, error) {
	row := r.db.QueryRowContext(ctx, query("detail"),
		reference,
		inboxrclpucl.WorkClassClaim,
	)

	detail, err := scanDetail(row)
	if errors.Is(err, sql.ErrNoRows) {
		return inboxrclpucl.ClaimDetail{}, inboxrclpucl.ErrClaimNotFound
	}
	if err != nil {
		return inboxrclpucl.ClaimDetail{}, fmt.Errorf("membaca layar kerja klaim: %w", err)
	}
	return detail, nil
}

// CheckTable memastikan tabel DAN kolom yang disentuh modul ini terbaca dari koneksi yang
// dipakai.
//
// Dipanggil perintah `-periksa`. Ia tidak menyentuh satu baris pun: yang diperiksa adalah
// hak baca, keberadaan tabelnya, dan keberadaan kelima kolom penyaringnya.
//
// Pemeriksaan kolom terpisah dari pemeriksaan tabel dengan sengaja — lihat catatan pada
// kueri `check_columns`. Kolom yang tidak ada dan kolom yang ada tetapi kosong menghasilkan
// layar yang sama-sama kosong, dan hanya yang pertama yang merupakan kerusakan.
func (r *Repo) CheckTable(ctx context.Context) error {
	var ignored int

	if err := r.db.QueryRowContext(ctx, query("check_rclpucl")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca DATAPEGA.PC_ASM_FW_GCNMFW_WORK atau "+
				"DATAPEGA.PC_ASSIGN_WORKBASKET: %w", err)
	}
	if err := r.db.QueryRowContext(ctx, query("check_columns")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca kolom penyaring PUCL (TANGGALCETAKDOKUMENPUCL_1, STATUSCASE_1, "+
				"PUCLAPPROVE_1, MSIG_1, TANGGALKIRIMPUCL_1): %w", err)
	}

	// Kedua tabel anak diperiksa TERPISAH: tanpa keduanya, layar kerja tetap terbuka
	// tetapi "Nama Peserta", "UP", dan "Jumlah Tagihan" diam-diam kosong — dan kosong
	// adalah keadaan yang sah bagi klaim tanpa objek, sehingga tidak dapat dibedakan dari
	// kerusakan.
	if err := r.db.QueryRowContext(ctx, query("check_detail")).Scan(&ignored); err != nil {
		return fmt.Errorf(
			"membaca POOLDATA.T_CLAIM_OBJECTLIST atau "+
				"POOLDATA.T_CLAIM_ADJUSTMENT: %w", err)
	}
	return nil
}

// scanner adalah bentuk minimal yang dibutuhkan pemindai, sehingga keduanya dapat diuji
// tanpa basis data.
type scanner interface {
	Scan(dest ...any) error
}

// scanWorkItem memindai satu baris grid menjadi WorkItem beserta jumlah seluruh baris.
//
// Urutannya WAJIB sama dengan listColumns dan dengan urutan kolom di inboxrclpucl.sql.
// Ketiganya dijaga query_test.go.
//
// # Kenapa SELURUH kolom dipindai lewat tipe yang mengizinkan NULL
//
// Bukan kehati-hatian berlebih. `LETTER_PRINTED_AT` memang SELALU NULL pada tab Cetak Surat
// — penyaringnya `IS NULL`. `TRACK_CODE` kosong pada klaim yang jalurnya belum ditetapkan.
// Dan kelima kolom PUCL lain hanya punya 2–86 nilai berbeda di produksi, yang berarti
// sebagian besar barisnya kosong.
//
// # Kenapa waktu ikut dipindai sebagai teks
//
// Bentuk yang dikembalikan driver bergantung pada tipe kolomnya, dan DDL tabel Pega tidak
// tersedia (`R-08`). Memindainya sebagai `sql.NullString` membuat nilainya sampai ke layar
// apa adanya alih-alih gagal dipindai pada baris pertama di produksi. Pemformatannya
// dikerjakan layar, dan hanya bila bentuknya memang dikenali.
func scanWorkItem(row scanner) (inboxrclpucl.WorkItem, int, error) {
	var (
		reference, caseID, policyNumber sql.NullString
		insuredName, inboxEntryAt       sql.NullString
		analystNote, trackCode          sql.NullString
		letterPrintedAt, claimAge       sql.NullString
		expiryStatus                    sql.NullString
		total                           sql.NullInt64
	)

	err := row.Scan(
		&reference, &caseID, &policyNumber, &insuredName, &inboxEntryAt,
		&analystNote, &trackCode, &letterPrintedAt, &claimAge, &expiryStatus,
		&total,
	)
	if err != nil {
		return inboxrclpucl.WorkItem{}, 0, err
	}

	return inboxrclpucl.WorkItem{
		Reference:    reference.String,
		CaseID:       caseID.String,
		PolicyNumber: policyNumber.String,
		InsuredName:  insuredName.String,
		InboxEntryAt: inboxEntryAt.String,
		AnalystNote:  analystNote.String,

		// Jalur DITERJEMAHKAN di sini, bukan di dalam kueri.
		//
		// Penerjemahannya milik domain (`TrackOf`), sehingga penyimpanan SQL dan
		// penyimpanan memori menghasilkan teks yang sama persis. Menuliskannya sebagai
		// `CASE` di dalam SQL akan membuat kedua pengisi seam punya dua penerjemah yang
		// dapat menyimpang tanpa ketahuan.
		//
		// Hasilnya identik dengan `CASE` tanpa `ELSE` di sistem lama: kode yang tidak
		// dikenali menghasilkan teks kosong.
		Track: inboxrclpucl.TrackOf(trackCode.String),

		LetterPrintedAt: letterPrintedAt.String,
		ClaimAge:        claimAge.String,
		ExpiryStatus:    expiryStatus.String,
	}, int(total.Int64), nil
}

// scanDetail memindai satu baris layar kerja.
//
// Urutannya WAJIB sama dengan detailColumns dan dengan urutan kolom kueri `detail`.
//
// Kedua isian TURUNAN dipindai sebagai teks yang mengizinkan NULL, dan keduanya memang
// sering kosong: klaim tanpa objek, atau objek tanpa adjustment, menghasilkan subkueri yang
// tidak mengembalikan baris. Itu keadaan yang sah — bukan kegagalan.
func scanDetail(row scanner) (inboxrclpucl.ClaimDetail, error) {
	var (
		reference, claimNumber, trackCode sql.NullString
		analystNote, policyNumber         sql.NullString
		lossDate, puclNote                sql.NullString
		firstObjectName, firstPropose     sql.NullString
	)

	err := row.Scan(
		&reference, &claimNumber, &trackCode, &analystNote, &policyNumber,
		&lossDate, &puclNote,
		&firstObjectName, &firstPropose,
	)
	if err != nil {
		return inboxrclpucl.ClaimDetail{}, err
	}

	return inboxrclpucl.ClaimDetail{
		Reference:   reference.String,
		ClaimNumber: claimNumber.String,

		Letter: inboxrclpucl.LetterDraft{
			Track:        inboxrclpucl.TrackOf(trackCode.String),
			TrackCode:    trackCode.String,
			AnalystNote:  analystNote.String,
			PolicyNumber: policyNumber.String,
			LossDate:     lossDate.String,

			// Kedua isian diisi dari SATU sumber, dan itu memang benar.
			//
			// `SetDataLampiranSuratRCLPUCL_Act` menetapkan `.UP` dan `.NamaPeserta` dari
			// ekspresi yang sama persis, dan Work Owner menegaskan 2026-09-24 bahwa kolom
			// "UP" pada surat RCL/PUCL memang berisi NAMA OBJEK — bukan nilai
			// pertanggungan.
			//
			// Ditulis berdampingan dengan sengaja: keduanya terbaca sebagai salin-tempel
			// yang keliru, dan pernah "diperbaiki" atas dasar itu. Lihat
			// `inboxrclpucl.LetterDraft.SumInsured`.
			InsuredName: firstObjectName.String,
			SumInsured:  firstObjectName.String,

			BillAmount: firstPropose.String,
		},

		DocumentReceipt: inboxrclpucl.DocumentReceipt{
			PUCLNote: puclNote.String,
		},
	}, nil
}

// scanReportRow memindai satu baris laporan harian beserta jumlah seluruh baris.
//
// Urutannya WAJIB sama dengan reportColumns dan dengan urutan kolom kueri `daily_report`.
// Ia TERPISAH dari scanWorkItem karena kolomnya memang berbeda — menyatukan keduanya akan
// menuntut satu pemindai yang separuh kolomnya selalu kosong, dan itu menyembunyikan
// perbedaan yang justru harus terlihat.
func scanReportRow(row scanner) (inboxrclpucl.DailyReportRow, int, error) {
	var (
		reference, caseID, policyNumber sql.NullString
		insuredName, sentAt             sql.NullString
		analystNote, letterPrintedAt    sql.NullString
		trackCode, claimStatus          sql.NullString
		total                           sql.NullInt64
	)

	err := row.Scan(
		&reference, &caseID, &policyNumber, &insuredName, &sentAt,
		&analystNote, &letterPrintedAt, &trackCode, &claimStatus,
		&total,
	)
	if err != nil {
		return inboxrclpucl.DailyReportRow{}, 0, err
	}

	return inboxrclpucl.DailyReportRow{
		Reference:       reference.String,
		CaseID:          caseID.String,
		PolicyNumber:    policyNumber.String,
		InsuredName:     insuredName.String,
		SentAt:          sentAt.String,
		AnalystNote:     analystNote.String,
		LetterPrintedAt: letterPrintedAt.String,

		// Kueri lama menuliskan kode mentah `1`/`2` ke dalam berkas. Di sini ia
		// diterjemahkan supaya berkas dan layar menyebut hal yang sama dengan kata yang
		// sama — selisih terencana, dinyatakan lewat PlannedDifferences.
		Track: inboxrclpucl.TrackOf(trackCode.String),

		ClaimStatus: claimStatus.String,
	}, int(total.Int64), nil
}
