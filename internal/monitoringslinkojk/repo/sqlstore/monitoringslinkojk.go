package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/monitoringslinkojk"
)

// DefaultBranchCode adalah kode kantor cabang pada kolom "Kode Kantor Cabang".
//
// # Ia BELUM SEHARUSNYA berupa konstanta, dan itu diakui di sini
//
// `RDB List/GetDataSlinkAllFOG-SQL.xml` menuliskannya sebagai literal di dalam kueri —
// `'001' as "KodeKantorCabang"`. `D-15` menetapkan tidak ada nilai bisnis yang boleh
// di-hardcode; tempatnya adalah master Cabang pada `F-4`, yang belum ada.
//
// Nilainya karena itu diangkat menjadi parameter repo, bukan ditanam di dalam berkas
// .sql: ketika masternya tiba, yang berubah hanya cara repo ini dibentuk. Ia juga
// dibindkan sebagai parameter, bukan dirangkai ke teks SQL — sehingga entitas yang
// memakai kode cabang lain tidak menuntut kueri kedua.
const DefaultBranchCode = "001"

// tanggalTampil adalah bentuk tanggal yang dibaca pengguna dan ditulis ke CSV.
//
// `dd/mm/yyyy`, sama dengan `to_char(…, 'dd/mm/yyyy')` pada kueri lama. Pemformatannya
// dilakukan DI SINI, bukan di SQL: `08-TECHNICAL-STRATEGY.md` §4.3 melarang `TO_CHAR`
// untuk keperluan tampilan, karena tanggal yang dikembalikan sebagai teks membuat
// pengurutan menjadi pengurutan TEKS — `01/12/2024` terbaca lebih kecil daripada
// `02/01/2020`.
const tanggalTampil = "02/01/2006"

// Repo membaca data Monitoring SLINK OJK dari POOLDATA.
//
// Seluruh tabelnya MILIK SISTEM LAMA dan HANYA DIBACA di sini.
type Repo struct {
	db *sql.DB

	// branchCode mengisi kolom "Kode Kantor Cabang". Lihat DefaultBranchCode.
	branchCode string
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo {
	return NewRepoWithBranchCode(db, DefaultBranchCode)
}

// NewRepoWithBranchCode membentuk repo dengan kode kantor cabang yang ditentukan
// pemanggil.
//
// Nilai kosong jatuh ke DefaultBranchCode — bukan ke teks kosong, yang akan membuat
// setiap baris laporan kehilangan kode cabangnya tanpa sebab yang terbaca.
func NewRepoWithBranchCode(db *sql.DB, branchCode string) *Repo {
	clean := strings.TrimSpace(branchCode)
	if clean == "" {
		clean = DefaultBranchCode
	}
	return &Repo{db: db, branchCode: clean}
}

// Search mengembalikan satu halaman baris pada segmen yang diminta.
//
// Dua kueri, bukan satu: jumlah seluruh baris dibutuhkan bilah halaman, dan menghitungnya
// di Go berarti membaca seluruh hasilnya lebih dulu — persis yang dihindari paginasi.
func (r *Repo) Search(
	ctx context.Context,
	segment monitoringslinkojk.Segment,
	filter monitoringslinkojk.Filter,
) (monitoringslinkojk.Page, error) {
	plan, err := r.planFor(segment)
	if err != nil {
		return monitoringslinkojk.Page{}, err
	}

	total, err := r.count(ctx, plan, filter)
	if err != nil {
		return monitoringslinkojk.Page{}, err
	}

	// Halaman di luar jangkauan dijawab senarai KOSONG beserta totalnya, bukan galat.
	// Pengguna yang menekan "halaman berikutnya" tepat ketika baris terakhir baru saja
	// berpindah tidak melakukan kesalahan apa pun.
	if total == 0 || filter.Offset() >= total {
		return monitoringslinkojk.Page{Rows: []monitoringslinkojk.Row{}, Total: total}, nil
	}

	args := plan.rowArgs(r.branchCode, filter)
	rows, err := r.db.QueryContext(ctx, query(plan.rowsQuery), args...)
	if err != nil {
		return monitoringslinkojk.Page{}, fmt.Errorf(
			"monitoringslinkojk/sqlstore: %s: %w", plan.rowsQuery, err)
	}
	defer rows.Close()

	collected := make([]monitoringslinkojk.Row, 0, filter.Size)
	if err := scanAll(rows, plan, func(row monitoringslinkojk.Row) error {
		collected = append(collected, row)
		return nil
	}); err != nil {
		return monitoringslinkojk.Page{}, fmt.Errorf(
			"monitoringslinkojk/sqlstore: %s: %w", plan.rowsQuery, err)
	}

	return monitoringslinkojk.Page{Rows: collected, Total: total}, nil
}

// Stream membaca SELURUH baris yang cocok, satu per satu.
//
// Galat yang dikembalikan `emit` menghentikan pembacaan dan diteruskan APA ADANYA — itu
// yang memungkinkan pemanggil berhenti di tengah, misalnya ketika batas baris ekspor
// tercapai, tanpa kesalahan itu tersamar sebagai kegagalan basis data.
func (r *Repo) Stream(
	ctx context.Context,
	segment monitoringslinkojk.Segment,
	filter monitoringslinkojk.Filter,
	emit func(monitoringslinkojk.Row) error,
) error {
	plan, err := r.planFor(segment)
	if err != nil {
		return err
	}

	args := plan.exportArgs(r.branchCode, filter)
	rows, err := r.db.QueryContext(ctx, query(plan.exportQuery), args...)
	if err != nil {
		return fmt.Errorf("monitoringslinkojk/sqlstore: %s: %w", plan.exportQuery, err)
	}
	defer rows.Close()

	return scanAll(rows, plan, emit)
}

// count menghitung seluruh baris yang cocok.
func (r *Repo) count(
	ctx context.Context,
	plan segmentPlan,
	filter monitoringslinkojk.Filter,
) (int, error) {
	var total sql.NullInt64
	args := plan.countArgs(filter)
	if err := r.db.QueryRowContext(ctx, query(plan.countQuery), args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("monitoringslinkojk/sqlstore: %s: %w", plan.countQuery, err)
	}
	if !total.Valid {
		return 0, nil
	}
	return int(total.Int64), nil
}

// scanAll membaca seluruh baris hasil dan menyerahkannya satu per satu.
//
// # Kenapa nilainya dibaca sebagai `any`, bukan sebagai tipe yang ditentukan di muka
//
// Karena tipe kolomnya BELUM DIKETAHUI. DDL `POOLDATA.T_CLAIM_SLIK_OJK` tidak ada di
// export dan belum diterima DBA (`R-08`), dan bukti yang ada saling bertentangan: kueri
// segmen D01 memilih kolom tanggalnya TANPA konversi, sedangkan kueri segmen F06
// membungkus kolom tanggalnya dengan `to_char(…,'dd/mm/yyyy')`. Yang pertama menunjukkan
// kolom TEKS, yang kedua kolom TANGGAL.
//
// Menebak salah satu berarti seluruh kolom tanggal gagal di-scan pada separuh
// kemungkinan — dan gagalnya baru terlihat di produksi. Membacanya sebagai `any` lalu
// memformatnya di satu tempat melayani keduanya, dan bentuk keluarannya tetap sama.
// Begitu DDL-nya diterima, yang berubah hanya fungsi text di bawah.
func scanAll(
	rows *sql.Rows,
	plan segmentPlan,
	emit func(monitoringslinkojk.Row) error,
) error {
	names, err := rows.Columns()
	if err != nil {
		return err
	}

	values := make([]any, len(names))
	pointers := make([]any, len(names))
	for i := range values {
		pointers[i] = &values[i]
	}

	for rows.Next() {
		for i := range values {
			values[i] = nil
		}
		if err := rows.Scan(pointers...); err != nil {
			return err
		}

		raw := make(map[string]string, len(names))
		for i, name := range names {
			raw[strings.ToUpper(name)] = text(values[i])
		}

		if err := emit(plan.toRow(raw)); err != nil {
			return err
		}
	}
	return rows.Err()
}

// text mengubah satu nilai kolom menjadi teks siap tampil.
//
// Satu tempat untuk seluruh pemformatan — tanggal, angka, dan teks — sehingga layar dan
// berkas CSV tidak akan pernah menampilkan bentuk yang berbeda untuk nilai yang sama.
func text(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(v)
	case []byte:
		return strings.TrimSpace(string(v))
	case time.Time:
		if v.IsZero() {
			return ""
		}
		return v.Format(tanggalTampil)
	case bool:
		if v {
			return "1"
		}
		return "0"
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		// `'f', -1` menulis angka apa adanya tanpa notasi ilmiah. Tanpa itu, nilai
		// klaim besar tampil sebagai `2.25e+07` di layar maupun di berkas laporan.
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		// godror mengembalikan sebagian kolom NUMBER sebagai tipenya sendiri yang
		// berupa string. Ia — dan tipe tak terduga lainnya — dicetak apa adanya alih-alih
		// dibuang menjadi teks kosong: nilai yang tampil aneh akan ditanyakan pengguna,
		// nilai yang hilang tidak.
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

// ============================================================================
// RENCANA PER SEGMEN
// ============================================================================

// segmentPlan menyatukan nama kueri, penyusun argumen, dan pemetaan kolom satu segmen.
//
// Satu struct, bukan percabangan `switch` yang tersebar di empat method: penambahan
// segmen ketiga kelak cukup menambah satu rencana, dan tidak ada method yang dapat
// tertinggal.
type segmentPlan struct {
	rowsQuery   string
	countQuery  string
	exportQuery string

	// takesBranchCode menandai kuerinya memakai parameter kode kantor cabang di posisi
	// pertama. Hanya segmen F06 — lihat DefaultBranchCode.
	takesBranchCode bool

	// takesBusinessScope menandai kuerinya menyaring "Business Name". Hanya segmen D01.
	//
	// Layar F06 menampilkan dropdown-nya, tetapi `GetAllDataSumbisSlink` tidak pernah
	// menurunkannya menjadi `BusinessID` dan kuerinya tidak memuat placeholder itu —
	// sehingga di Pega dropdown tersebut tidak mengubah hasil apa pun pada segmen F06.
	takesBusinessScope bool

	// toRow memetakan kolom hasil kueri menjadi baris berkunci Column.Key.
	toRow func(raw map[string]string) monitoringslinkojk.Row
}

func (r *Repo) planFor(segment monitoringslinkojk.Segment) (segmentPlan, error) {
	switch segment {
	case monitoringslinkojk.SegmentD01:
		return segmentPlan{
			// Segmen D01 menampilkan IDENTITAS DEBITUR, dan sumbernya
			// `T_CLAIM_OBJECTLIST` — tabel yang memuat CIF, jenis kelamin, tanggal
			// lahir, alamat, dan telepon debitur.
			//
			// Nama kuerinya tetap `f06_*` mengikuti berkas asalnya di export
			// (`GetDataSlinkAllFOG`). Penamaan terbalik itu dicatat di columns.go;
			// pemetaan ke segmen layar terjadi di sini.
			rowsQuery:       "f06_rows",
			countQuery:      "f06_count",
			exportQuery:     "f06_export",
			takesBranchCode: true,
			toRow:           f06Row,
		}, nil
	case monitoringslinkojk.SegmentF06:
		return segmentPlan{
			// Segmen F06 menampilkan FASILITAS KREDIT yang sudah tersusun sebagai
			// laporan, dan sumbernya `T_CLAIM_SLIK_OJK`.
			rowsQuery:          "d01_rows",
			countQuery:         "d01_count",
			exportQuery:        "d01_export",
			takesBusinessScope: true,
			toRow:              d01Row,
		}, nil
	}
	return segmentPlan{}, monitoringslinkojk.ErrUnknownSegment
}

// rowArgs menyusun argumen kueri satu halaman.
func (p segmentPlan) rowArgs(branchCode string, filter monitoringslinkojk.Filter) []any {
	args := p.baseArgs(branchCode, filter)
	return append(args, filter.Offset(), filter.Size)
}

// countArgs menyusun argumen kueri penghitung.
//
// Kode kantor cabang TIDAK ikut: kueri penghitung tidak memilih kolom itu, sehingga
// menyertakannya akan menggeser seluruh penomoran parameter.
func (p segmentPlan) countArgs(filter monitoringslinkojk.Filter) []any {
	return p.filterArgs(filter)
}

// exportArgs menyusun argumen kueri ekspor — sama dengan rowArgs tanpa paginasi.
func (p segmentPlan) exportArgs(branchCode string, filter monitoringslinkojk.Filter) []any {
	return p.baseArgs(branchCode, filter)
}

func (p segmentPlan) baseArgs(branchCode string, filter monitoringslinkojk.Filter) []any {
	args := make([]any, 0, 16)
	if p.takesBranchCode {
		args = append(args, branchCode)
	}
	return append(args, p.filterArgs(filter)...)
}

// filterArgs menyusun argumen penyaring sesuai urutan `:n` di berkas .sql.
//
// # Kenapa nilai yang sama dikirim berkali-kali
//
// Karena pola `(:n IS NULL OR …)` menyebut nilainya lebih dari sekali, dan Oracle
// memperlakukan setiap `:n` sebagai parameter POSISIONAL tersendiri — bukan parameter
// bernama yang dapat dipakai ulang. Mengirimnya sekali akan menggeser seluruh
// penomoran, dan pergeseran itu TIDAK menghasilkan galat: kueri tetap berjalan dengan
// nilai yang tertukar. query_test.go memeriksa jumlahnya cocok dengan berkas .sql.
//
// # Jumlahnya berbeda antar segmen, dan itu mengikuti Pega
//
// Hanya segmen D01 yang menyaring "Business Name". Kueri segmen F06 —
// `GetDataSlinkAllFOG-SQL.xml` — TIDAK memuat placeholder `BusinessID` sama sekali,
// sehingga dropdown-nya tampil tetapi tidak menyaring apa pun. Lihat takesBusinessScope.
func (p segmentPlan) filterArgs(filter monitoringslinkojk.Filter) []any {
	from, to := dateBounds(filter)

	args := []any{
		from, from, // :1 :2  — batas bawah tanggal registrasi
		to, to, // :3 :4  — batas atas, EKSKLUSIF; lihat dateBounds
	}
	if !p.takesBusinessScope {
		return args
	}

	mode, businessType := scopeArgs(filter.BusinessScope)
	return append(args, mode, mode, businessType, mode, businessType) // :5 … :9
}

// dateBounds mengubah kedua isian tanggal menjadi batas yang dapat dibandingkan langsung.
//
// Batas atasnya dimajukan SATU HARI dan dibandingkan dengan `<`, bukan `<=`. Alasannya:
// `REGISTERDATE_1` menyimpan tanggal BESERTA jamnya, sehingga `<= '31/03/2026'` berarti
// `<= 31/03/2026 00:00:00` — dan seluruh klaim yang diregistrasi pada hari terakhir
// rentang hilang dari laporan tanpa satu pun tanda. Pada laporan ke regulator, baris yang
// hilang diam-diam adalah kegagalan yang paling mahal.
func dateBounds(filter monitoringslinkojk.Filter) (any, any) {
	var from, to any

	if filter.DateOfLoss != nil {
		start := *filter.DateOfLoss
		from = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, start.Location())
	}
	if filter.DateOfRequestDocument != nil {
		end := *filter.DateOfRequestDocument
		midnight := time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, end.Location())
		to = midnight.AddDate(0, 0, 1)
	}
	return from, to
}

// Penanda mode penyaring Business Name, sebagaimana dibaca berkas .sql.
//
// Teks, bukan boolean: ia harus dapat bernilai "tidak menyaring sama sekali" sebagai
// NULL, dan `(:n IS NULL OR …)` menuntut ketiga keadaan itu dapat dibedakan dalam satu
// parameter.
const (
	scopeInclude = "INCLUDE"
	scopeExclude = "EXCLUDE"
)

// creditInsuranceNamePattern memilih lini Asuransi Kredit pada `BUSINESSNAME`.
//
// ============================================================================
// KENAPA POLA, BUKAN PERBANDINGAN PERSIS — dan kenapa kolomnya bukan BUSINESSTYPE
// ============================================================================
//
// Kueri Pega menyaring `b.businesstype = 'AsuransiKredit'` pada tabel kerja Pega. Tabel
// datar `POOLDATA.T_CLAIMLIST_ADMIN` yang menggantikannya **tidak punya kolom
// `BUSINESSTYPE` sama sekali** — terbukti dari katalog Oracle, dan sebelumnya membuat
// seluruh kueri segmen F06 gagal dengan `ORA-00904`.
//
// Yang ada `BUSINESSNAME`, dan isinya nama lini yang terbaca — bukan kode. Asuransi
// Kredit muncul sebagai EMPAT varian:
//
//	ASURANSI KREDIT 2             ASURANSI KREDIT PEMBIAYAAN
//	ASURANSI KREDIT               ASURANSI KREDIT PERDAGANGAN
//
// Perbandingan persis karena itu tidak mungkin; yang dipakai adalah awalan. Pada data
// hari ini, dari 119 baris laporan: **74 tercakup awalan ini**, 45 sisanya menjadi
// "Surety Bond".
//
// # Ini PILIHAN yang perlu dikonfirmasi, bukan kesetaraan yang terbukti
//
// Pencocokan awalan adalah kelas pola yang sama dengan toleransi spreading yang `D-49`
// butir 1 perbaiki. Dipakai di sini karena tidak ada kolom kode yang dapat dibandingkan
// persis — bukan karena dianggap aman. Bila ternyata salah satu varian seharusnya TIDAK
// termasuk, baris itu hilang dari laporan regulator tanpa satu pun galat.
//
// Tercatat sebagai pertanyaan terbuka di `docs/permintaan-artefak-pega.md`.
const creditInsuranceNamePattern = "ASURANSI KREDIT%"

// scopeArgs menerjemahkan pilihan "Business Name" menjadi mode dan nilai pembanding.
//
// Lihat monitoringslinkojk.BusinessScope: "SURETY BOND" MENIADAKAN Asuransi Kredit alih-
// alih memilih Surety Bond, dan itu perilaku warisan yang sengaja direplikasi.
func scopeArgs(scope monitoringslinkojk.BusinessScope) (any, any) {
	switch scope {
	case monitoringslinkojk.ScopeCreditInsurance:
		return scopeInclude, creditInsuranceNamePattern
	case monitoringslinkojk.ScopeSuretyBond:
		return scopeExclude, creditInsuranceNamePattern
	default:
		return nil, nil
	}
}

// ============================================================================
// PEMETAAN KOLOM HASIL KUERI → KATALOG
// ============================================================================

// d01Row memetakan hasil kueri segmen D01 menjadi satu baris grid.
//
// Kunci baris dirangkai dari nomor klaim DAN nomor kontrak, bukan dari nomor klaim saja:
// satu klaim dapat memuat beberapa fasilitas kredit, dan kunci yang sama pada dua baris
// membuat tabel di layar menampilkan salah satunya dua kali.
func d01Row(raw map[string]string) monitoringslinkojk.Row {
	row := monitoringslinkojk.Row{
		monitoringslinkojk.RowKeyColumn: rowKey(raw["NO_KLAIM"], raw["CONTRACT_NO"]),

		"no_klaim":                 raw["NO_KLAIM"],
		"contract_no":              raw["CONTRACT_NO"],
		"nomor_rekening_fasilitas": raw["NOMOR_REKENING_FASILITAS"],
		"no_cif_debitur":           raw["NO_CIF_DEBITUR"],
		"kode_jenis_fasilitas":     raw["KODE_JENIS_FASILITAS"],
		"sumber_dana":              raw["SUMBER_DANA"],
		"start_polis":              raw["START_POLIS"],
		"end_polis":                raw["END_POLIS"],
		"suku_bunga":               raw["SUKU_BUNGA"],
		"kode_valuta":              raw["KODE_VALUTA"],
		"nilai_mata_uang_asal":     raw["NILAI_MATA_UANG_ASAL"],
		"tanggal_pembayaran":       raw["TANGGAL_PEMBAYARAN"],
		"kode_kolektibilitas":      raw["KODE_KOLEKTIBILITAS"],
		"tanggal_macet":            raw["TANGGAL_MACET"],
		"kode_sebab_macet":         raw["KODE_SEBAB_MACET"],
		"tunggakan":                raw["TUNGGAKAN"],
		"jumlah_kewajiban":         raw["JUMLAH_KEWAJIBAN"],
		"tanggal_kondisi":          raw["TANGGAL_KONDISI"],
		"kode_kondisi":             raw["KODE_KONDISI"],
		"keterangan":               raw["KETERANGAN"],

		// Ketiga kolom berikut TIDAK tampil di grid; keduanya hanya dipakai berkas
		// ekspor, yang memang punya 27 kolom sementara grid punya 20.
		"kode_kantor_cabang": raw["KODE_KANTOR_CABANG"],
		"operasi_data":       raw["OPERASI_DATA"],
		"no_ktp":             raw["NO_KTP"],
		"npwp_perusahaan":    raw["NPWP_PERUSAHAAN"],
		"no_polis":           raw["NO_POLIS"],
	}
	return row
}

// f06Row memetakan hasil kueri segmen F06 menjadi satu baris grid.
//
// Hanya kedelapan kolom bersumber yang diisi. Ketiga puluh sisanya sengaja TIDAK
// dimasukkan ke dalam peta — bukan diisi teks kosong — sehingga ketiadaannya tetap
// terbaca sebagai "tidak ada sumber" dan bukan sebagai "sudah dibaca, ternyata kosong".
// Yang menerjemahkan keduanya menjadi sel kosong di layar adalah Row.Get.
// f06Row memetakan hasil `f06_rows` menjadi baris grid.
//
// Kedua puluh kunci pertama SAMA dengan d01Row — grid kedua segmen memang identik di
// Pega (lihat f06Columns). Yang berbeda hanya asal nilainya: D01 membaca tabel SLIK yang
// sudah tersusun, F06 membaca data klaim sumbernya.
//
// Kunci sesudahnya tidak tampil di grid; ia dipakai berkas ekspor F06, yang memasangkan
// `NOMOR_CIF_DEBITUR`, `JENIS_KELAMIN`, `TANGGAL_LAHIR`, `ALAMAT`, `KODE_POS`, `TELEPON`,
// `KODE_KANTOR_CABANG`, dan `OPERASI_DATA` ke judul-judul CIF-nya.
func f06Row(raw map[string]string) monitoringslinkojk.Row {
	return monitoringslinkojk.Row{
		monitoringslinkojk.RowKeyColumn: rowKey(
			raw["NO_KLAIM"], raw["CONTRACT_NO"]),

		"no_klaim":                 raw["NO_KLAIM"],
		"contract_no":              raw["CONTRACT_NO"],
		"nomor_rekening_fasilitas": raw["NOMOR_REKENING_FASILITAS"],
		"no_cif_debitur":           raw["NO_CIF_DEBITUR"],
		"kode_jenis_fasilitas":     raw["KODE_JENIS_FASILITAS"],
		"sumber_dana":              raw["SUMBER_DANA"],
		"start_polis":              raw["START_POLIS"],
		"end_polis":                raw["END_POLIS"],
		"suku_bunga":               raw["SUKU_BUNGA"],
		"kode_valuta":              raw["KODE_VALUTA"],
		"tanggal_pembayaran":       raw["TANGGAL_PEMBAYARAN"],
		"nilai_mata_uang_asal":     raw["NILAI_MATA_UANG_ASAL"],
		"kode_kolektibilitas":      raw["KODE_KOLEKTIBILITAS"],
		"tanggal_macet":            raw["TANGGAL_MACET"],
		"kode_sebab_macet":         raw["KODE_SEBAB_MACET"],
		"tunggakan":                raw["TUNGGAKAN"],
		"jumlah_kewajiban":         raw["JUMLAH_KEWAJIBAN"],
		"tanggal_kondisi":          raw["TANGGAL_KONDISI"],
		"kode_kondisi":             raw["KODE_KONDISI"],
		"keterangan":               raw["KETERANGAN"],

		// Hanya untuk berkas ekspor F06 — tidak tampil di grid.
		"object_name":        raw["OBJECT_NAME"],
		"customer_type":      raw["CUSTOMER_TYPE"],
		"nomor_cif_debitur":  raw["NO_CIF_DEBITUR"],
		"jenis_kelamin":      raw["JENIS_KELAMIN"],
		"tanggal_lahir":      raw["TANGGAL_LAHIR"],
		"alamat":             raw["ALAMAT"],
		"kode_pos":           raw["KODE_POS"],
		"telepon":            raw["TELEPON"],
		"kode_kantor_cabang": raw["KODE_KANTOR_CABANG"],
		"operasi_data":       raw["OPERASI_DATA"],
	}
}

// rowKey merangkai kunci baris dari beberapa bagian.
//
// Pemisahnya `|` — karakter yang tidak muncul di nomor klaim maupun nomor kontrak —
// supaya dua baris berbeda tidak dapat menghasilkan kunci yang sama karena bagiannya
// kebetulan bersambung.
func rowKey(parts ...string) string {
	cleaned := make([]string, 0, len(parts))
	for _, part := range parts {
		cleaned = append(cleaned, strings.TrimSpace(part))
	}
	return strings.Join(cleaned, "|")
}
