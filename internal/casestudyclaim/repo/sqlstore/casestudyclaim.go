package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"

	"claim-pnc/internal/casestudyclaim"
	"claim-pnc/internal/platform/money"
)

// Repo membaca POOLDATA.PEGA_DASHBOARDPNC, T_CLAIM_PNC, BUSINESS, dan T_CLAIM_ADJUSTMENT.
//
// Keempatnya MILIK SISTEM LAMA. Repo ini membaca keempatnya dan menulis **satu kolom**
// pada salah satunya — lihat SaveRemark.
type Repo struct {
	db *sql.DB
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dituju.
func NewRepo(db *sql.DB) *Repo { return &Repo{db: db} }

// List membaca satu halaman klaim telaah beserta jumlah seluruh yang cocok.
//
// Rentang tahun WAJIB terisi, dan kueri tidak dijalankan sama sekali tanpanya. Di Pega
// ketiadaannya menghasilkan nol baris secara diam — lihat casestudyclaim.ErrPeriodRequired.
func (r *Repo) List(ctx context.Context, f casestudyclaim.Filter) (casestudyclaim.Page, error) {
	f = f.Normalize()

	if err := casestudyclaim.ValidatePeriod(f.FromYear, f.ToYear); err != nil {
		return casestudyclaim.Page{}, err
	}

	filters := filterArgs(f)

	var total int
	if err := r.db.QueryRowContext(ctx, query("case_study_count"), filters...).Scan(&total); err != nil {
		return casestudyclaim.Page{}, fmt.Errorf("casestudyclaim/sqlstore: menghitung klaim: %w", err)
	}

	// Paginasi memakai :12 dan :13 — dua parameter terakhir, disisipkan sesudah penyaring.
	listArgs := append(append([]any(nil), filters...), f.Offset, f.Limit)

	rows, err := r.db.QueryContext(ctx, query("case_study_list"), listArgs...)
	if err != nil {
		return casestudyclaim.Page{}, fmt.Errorf("casestudyclaim/sqlstore: membaca daftar klaim: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := make([]casestudyclaim.CaseStudyRow, 0, f.Limit)
	for rows.Next() {
		row, err := scanRow(rows)
		if err != nil {
			return casestudyclaim.Page{}, fmt.Errorf("casestudyclaim/sqlstore: membaca baris klaim: %w", err)
		}
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return casestudyclaim.Page{}, fmt.Errorf("casestudyclaim/sqlstore: menelusuri daftar klaim: %w", err)
	}

	return casestudyclaim.Page{Rows: result, Total: total}, nil
}

// filterArgs menyusun kesebelas argumen pertama, sama untuk kueri hitung dan kueri daftar.
//
// # Kenapa penyaring yang sama dikirim BERKALI-KALI
//
// Kode bisnis muncul lima kali di dalam SQL dan kode status tiga kali — sekali pada
// `IS NULL`, sisanya pada perbandingan tiap cabang. Driver mengikat argumen menurut urutan
// KEMUNCULAN penanda, bukan menurut nomornya, sehingga mengirimnya sekali menghasilkan
// **ORA-01008**. Lihat kepala casestudyclaim.sql.
func filterArgs(f casestudyclaim.Filter) []any {
	business := nilIfEmpty(string(f.Business))
	status := nilIfEmpty(string(f.Status))

	// Ambang dikirim dalam RUPIAH, bukan satuan terkecil: kolom yang dibandingkan
	// (`TOTAL_CLAIM * CURRENCYVALUE`) bersatuan rupiah, dan kueri lama membandingkannya
	// terhadap `5000000000` — angka rupiah.
	threshold := casestudyclaim.LargeClaimThreshold.MinorUnits() / 100

	return []any{
		f.FromYear, // :1
		f.ToYear,   // :2
		business,   // :3  IS NULL
		business,   // :4  NONMBU
		business,   // :5  PA
		business,   // :6  TRAVEL
		business,   // :7  BONDING
		status,     // :8  IS NULL
		status,     // :9  progress-accept
		status,     // :10 reject
		threshold,  // :11 ambang nilai klaim, dalam rupiah
	}
}

// SaveRemark menuliskan catatan telaah satu klaim.
//
// Nilai pertama `false` berarti nomor klaimnya tidak ada. Ia BUKAN galat: baris dapat
// hilang di antara saat daftar dibaca dan saat Save ditekan.
//
// # Kenapa jumlah baris terpengaruh diperiksa
//
// Keunikan `CLAIMNO` tidak dibuktikan DDL mana pun (`R-08`). Bila UPDATE menyentuh lebih
// dari satu baris, catatan telaah satu klaim akan diam-diam tertulis pada klaim lain —
// kegagalan yang tidak menghasilkan satu pun galat dan tidak terlihat di layar. Di sini ia
// dijadikan galat supaya terlihat pada kejadian pertama.
func (r *Repo) SaveRemark(ctx context.Context, claimNumber, remark string) (bool, error) {
	number := strings.TrimSpace(claimNumber)
	if number == "" {
		return false, casestudyclaim.ErrClaimRequired
	}

	// Catatan kosong disimpan sebagai NULL, bukan sebagai teks kosong.
	//
	// Keduanya tampil sama di layar, tetapi hanya NULL yang berarti "belum pernah diisi"
	// bagi kueri mana pun yang kelak menyaring `REMARKRECOMENDATION IS NULL`. Pega pun
	// mengirimkan properti yang kosong, yang pada `RDB-List` menjadi NULL.
	var value any
	if trimmed := strings.TrimSpace(remark); trimmed != "" {
		value = trimmed
	}

	outcome, err := r.db.ExecContext(ctx, query("case_study_save_remark"), value, number)
	if err != nil {
		if isValueTooLarge(err) {
			// Kolomnya lebih sempit daripada MaxRemarkLength. Dikembalikan sebagai galat
			// domain supaya pengguna diberi tahu bahwa catatannya terlalu panjang —
			// bukan "terjadi kesalahan pada sistem". Lihat ErrRemarkRejectedByColumn.
			return false, fmt.Errorf("%w: %v", casestudyclaim.ErrRemarkRejectedByColumn, err)
		}
		return false, fmt.Errorf("casestudyclaim/sqlstore: menyimpan catatan telaah: %w", err)
	}

	affected, err := outcome.RowsAffected()
	if err != nil {
		// Driver yang tidak dapat menyebutkan jumlah baris membuat pemeriksaan di bawah
		// mustahil. Yang dikembalikan adalah galat, bukan "berhasil": menyatakan berhasil
		// atas penulisan yang tidak dapat dipastikan adalah persis yang hendak dicegah.
		return false, fmt.Errorf("casestudyclaim/sqlstore: membaca jumlah baris terpengaruh: %w", err)
	}

	switch {
	case affected == 0:
		return false, nil
	case affected > 1:
		return false, fmt.Errorf(
			"casestudyclaim/sqlstore: nomor klaim %q menyentuh %d baris — CLAIMNO ternyata tidak unik",
			number, affected)
	}
	return true, nil
}

// CheckTable membuktikan keempat tabel beserta kolomnya dapat dibaca.
//
// Dipakai `claimpnc -periksa`, bukan oleh jalur permintaan.
func (r *Repo) CheckTable(ctx context.Context) error {
	rows, err := r.db.QueryContext(ctx, query("case_study_check_table"))
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()
	return rows.Err()
}

// isValueTooLarge mengenali penolakan Oracle atas nilai yang melebihi lebar kolomnya.
//
// # Kenapa dicocokkan lewat TEKS, bukan lewat kode galat bertipe
//
// Driver `go-ora` mengembalikan galatnya sebagai tipe miliknya sendiri, dan mengimpor tipe
// itu di sini berarti lapisan penyimpanan modul ini terikat pada satu driver. Aturan
// proyek justru mengurung seluruh sentuhan driver di satu berkas platform
// (`keputusan-implementasi.md` §3.2), supaya pertukaran kembali ke `godror` menyentuh satu
// tempat saja.
//
// Yang dicocokkan karena itu adalah **nomor galatnya**, bukan pesannya: `ORA-12899` stabil
// lintas versi Oracle dan lintas bahasa pesan, sedangkan teks "value too large" berubah
// bila basis datanya berbahasa lain.
//
// # Kenapa pengenalan ini ada sama sekali
//
// Karena `MaxRemarkLength` adalah DUGAAN — DDL kolomnya belum pernah diterima (`R-08`),
// dan Work Owner memutuskan tidak mengejarnya. Bila dugaan itu terlalu longgar, tanpa
// pengenalan ini pengguna menerima "terjadi kesalahan pada sistem" setelah mengetik satu
// halaman penuh, dan tidak ada apa pun yang memberi tahu bahwa yang salah hanyalah
// panjangnya.
func isValueTooLarge(err error) bool {
	return err != nil && strings.Contains(err.Error(), "ORA-12899")
}

// nilIfEmpty mengubah teks kosong menjadi NULL, supaya penyaring "NULL berarti semua" pada
// SQL bekerja.
func nilIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

// scanRow membaca satu baris hasil.
//
// Seluruh kolom teks dibaca sebagai sql.NullString: kolomnya nullable di skema, dan
// membacanya langsung ke string akan gagal dengan galat konversi pada baris pertama yang
// kosong — misalnya klaim yang kronologinya belum diisi, yang justru keadaan biasa.
//
// Kesembilan nilai uang dan satu persentase dibaca sebagai `any`, bukan sebagai angka:
// bentuk yang diserahkan driver untuk kolom NUMBER berbeda-beda menurut presisinya, dan
// readMinorUnits yang menyatukannya. Lihat catatan di sana.
func scanRow(rows *sql.Rows) (casestudyclaim.CaseStudyRow, error) {
	var (
		claimNumber    sql.NullString
		policyNumber   sql.NullString
		insuredName    sql.NullString
		businessName   sql.NullString
		policyPeriod   sql.NullString
		claimMonth     any
		lossDate       sql.NullTime
		businessSource sql.NullString
		reinsurerCode  sql.NullString
		causeOfLoss    sql.NullString
		tsi            any
		branchName     sql.NullString
		claimStatus    sql.NullString
		chronology     sql.NullString
		remark         sql.NullString

		sharePercent  any
		deductible    any
		asmShareValue any
		claimValue    any
		adjusterFee   any
		lackOfDoc     any
		netClaim      any
		netClaimASM   any
	)

	// Urutannya WAJIB sama persis dengan daftar kolom pada case_study_list. Menambah kolom
	// di SQL tanpa menambahnya di sini menghasilkan galat jumlah kolom; menukar urutannya
	// menghasilkan data yang tertukar TANPA galat.
	if err := rows.Scan(
		&claimNumber, &policyNumber, &insuredName, &businessName,
		&policyPeriod, &claimMonth, &lossDate, &businessSource,
		&reinsurerCode, &causeOfLoss, &tsi,
		&branchName, &claimStatus, &chronology, &remark,
		&sharePercent, &deductible, &asmShareValue, &claimValue,
		&adjusterFee, &lackOfDoc, &netClaim, &netClaimASM,
	); err != nil {
		return casestudyclaim.CaseStudyRow{}, err
	}

	month, err := readInteger(claimMonth)
	if err != nil {
		return casestudyclaim.CaseStudyRow{}, fmt.Errorf("kolom %q: %w", "Bulan Klaim", err)
	}

	row := casestudyclaim.CaseStudyRow{
		ClaimNumber:  strings.TrimSpace(claimNumber.String),
		PolicyNumber: strings.TrimSpace(policyNumber.String),
		InsuredName:  strings.TrimSpace(insuredName.String),
		BusinessName: strings.TrimSpace(businessName.String),
		PolicyPeriod: strings.TrimSpace(policyPeriod.String),

		// Nol di depan ditambahkan DI SINI, bukan di SQL.
		//
		// Pega mengirimkan `to_char(…,'mm')` yaitu `"01"`…`"12"`; `EXTRACT` mengirimkan
		// angka. Keluaran akhirnya dibuat sama persis, supaya kolom ini tidak berubah
		// bentuk dari "03" menjadi "3" saat uji kesetaraan dijalankan.
		ClaimMonth: monthText(month),

		BusinessSource: strings.TrimSpace(businessSource.String),
		ReinsurerRole:  casestudyclaim.ReinsurerRoleOf(reinsurerCode.String),
		CauseOfLoss:    strings.TrimSpace(causeOfLoss.String),
		BranchName:     strings.TrimSpace(branchName.String),
		ClaimStatus:    casestudyclaim.ClaimStatusOf(claimStatus.String),

		// Kronologi dan catatan TIDAK dipangkas spasinya di ujung baris, hanya di ujung
		// teksnya — keduanya teks panjang yang ditulis manusia, dan pemenggalan barisnya
		// adalah bagian dari isinya.
		Chronology: strings.TrimSpace(chronology.String),
		Remark:     strings.TrimSpace(remark.String),
	}

	// Salinan lokal pada nilai bertipe pointer: mengambil alamat field sql.NullTime akan
	// membuat seluruh baris berbagi pointer yang sama saat di-loop.
	if lossDate.Valid {
		date := lossDate.Time.UTC()
		row.LossDate = &date
	}

	values := []struct {
		raw    any
		target **money.Money
		column string
	}{
		{tsi, &row.TSI, "TSI (100%)"},
		{deductible, &row.Deductible, "Deductible"},
		{asmShareValue, &row.ASMShareValue, "Nilai share ASM"},
		{claimValue, &row.ClaimValue100, "NILAI KLAIM 100%"},
		{adjusterFee, &row.AdjusterFee, "ADJUSTER FEE 100% SHARE"},
		{netClaim, &row.NetClaim100, "NILAI KLAIM NET 100%"},
		{netClaimASM, &row.NetClaimASM, "NILAI KLAIM NET ASM SHARE"},
		{lackOfDoc, &row.LackOfDoc, "LACK OF DOC/SALVAGE / RECOVERY / SUBROGARATION"},
	}
	for _, item := range values {
		amount, err := readMinorUnits(item.raw)
		if err != nil {
			return casestudyclaim.CaseStudyRow{}, fmt.Errorf("kolom %q: %w", item.column, err)
		}
		*item.target = amount
	}

	share, err := readInteger(sharePercent)
	if err != nil {
		return casestudyclaim.CaseStudyRow{}, fmt.Errorf("kolom %q: %w", "ASM SHARE", err)
	}
	row.ASMSharePercent = share

	return row, nil
}

// monthText menuliskan nomor bulan sebagai dua digit, seperti `to_char(…,'mm')` di Pega.
//
// Nilai di luar 1–12 ditulis APA ADANYA, tidak dipaksa masuk rentang: kolom sumbernya
// bertipe tanggal, sehingga nilai di luar rentang berarti ada yang tidak dipahami — dan
// itu harus terlihat, bukan disamarkan menjadi bulan yang tampak wajar.
func monthText(month *int64) string {
	if month == nil {
		return ""
	}
	if *month < 1 || *month > 12 {
		return strconv.FormatInt(*month, 10)
	}
	return fmt.Sprintf("%02d", *month)
}

// readMinorUnits membaca nilai uang yang SUDAH dikalikan 100 oleh SQL.
//
// # Kenapa TIDAK memakai money.FromSQLValue
//
// Fungsi itu menafsirkan angka bulat sebagai RUPIAH, sedangkan yang datang ke sini adalah
// SATUAN TERKECIL — dan ia juga menolak `float64` yang berdesimal, karena ketepatannya
// tidak dapat dijamin. Kedua sifat itu benar untuk pemakainya di modul Komite, dan
// keduanya tidak cocok di sini.
//
// Yang dikerjakan SQL-lah yang membuat pembacaan ini aman: setiap nilai sudah dikalikan
// 100 dan dibulatkan di Oracle, sehingga yang sampai ke sini selalu BILANGAN BULAT. Bila
// ternyata tidak, itu ditolak sebagai galat — bukan dibulatkan diam-diam, karena
// pembulatan diam adalah cara selisih rupiah masuk tanpa ada yang menyadarinya.
//
// NULL mengembalikan nil, bukan nol. Keduanya berbeda: `SUM` atas himpunan kosong
// menghasilkan NULL, dan "belum ada nilainya" bukan "nilainya nol".
func readMinorUnits(value any) (*money.Money, error) {
	minor, err := readInteger(value)
	if err != nil || minor == nil {
		return nil, err
	}
	amount := money.FromMinorUnits(*minor)
	return &amount, nil
}

// readInteger membaca bilangan bulat dari nilai apa pun yang diserahkan driver.
//
// Driver `go-ora` menyerahkan NUMBER sebagai int64, float64, atau string bergantung pada
// presisi kolomnya, dan kolom hasil `ROUND(...)` tidak punya presisi yang dideklarasikan.
// Ketiga bentuk itu ditangani; bentuk lain ditolak dengan menyebut tipenya, supaya
// pergantian driver kelak terlihat sebagai galat yang menjelaskan dirinya sendiri.
func readInteger(value any) (*int64, error) {
	switch v := value.(type) {
	case nil:
		return nil, nil
	case int64:
		return &v, nil
	case int:
		number := int64(v)
		return &number, nil
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("bukan bilangan: %v", v)
		}
		if v != math.Trunc(v) {
			return nil, fmt.Errorf(
				"%v berdesimal padahal SQL sudah membulatkannya; ketepatannya tidak dapat dijamin", v)
		}
		const exactLimit = 1 << 53
		if v >= exactLimit || v <= -exactLimit {
			return nil, fmt.Errorf("%v melampaui rentang yang float64 masih tepat", v)
		}
		number := int64(v)
		return &number, nil
	case []byte:
		return parseInteger(string(v))
	case string:
		return parseInteger(v)
	default:
		if text, canText := value.(fmt.Stringer); canText {
			return parseInteger(text.String())
		}
		return nil, fmt.Errorf("tipe %T tidak dikenali", value)
	}
}

// parseInteger membaca bilangan bulat dari teks yang diserahkan driver.
//
// Oracle menuliskan angka positif dengan satu spasi di depan pada sebagian jalur, dan
// sebagian lagi menuliskan `-0`. Keduanya dipangkas lebih dulu; sisanya wajib benar-benar
// bilangan bulat.
func parseInteger(text string) (*int64, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, nil
	}
	number, err := strconv.ParseInt(trimmed, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("%q bukan bilangan bulat: %w", trimmed, err)
	}
	return &number, nil
}
