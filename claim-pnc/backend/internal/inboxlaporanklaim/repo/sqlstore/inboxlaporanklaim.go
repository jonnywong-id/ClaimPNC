// Package sqlstore memenuhi seam inboxlaporanklaim.Repo dengan SQL.
//
// Satu instans Repo terikat pada SATU koneksi basis data, yaitu satu portal entitas.
// Pemisahan data antarentitas karena itu ada di tingkat koneksi, bukan di tingkat
// penyaringan baris (ADR-0030 Opsi 1) — tidak ada satu pun kueri di sini yang menyaring
// berdasarkan entitas, dan memang tidak boleh ada.
package sqlstore

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"claim-pnc/internal/inboxlaporanklaim"
	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/platform/db"
)

// ownTable menerjemahkan kegagalan pada tabel MILIK APLIKASI INI menjadi galat domain
// yang menyebutkan perbaikannya.
//
// Dipakai hanya pada jalur yang menyentuh POOLDATA.CPNC_LAPORAN_KLAIM beserta
// sequence-nya — bukan pada jalur yang membaca tabel warisan Pega. Bedanya penting:
// tabel warisan yang hilang berarti basis datanya salah portal atau hak akses akun
// aplikasi kurang, dan itu perbaikan yang sama sekali berbeda dari menjalankan migrasi.
func ownTable(err error) error {
	if db.IsMissingObject(err) {
		return fmt.Errorf("%w: %v", inboxlaporanklaim.ErrStorageNotReady, err)
	}
	return err
}

// cutBytes memotong teks ke paling banyak max byte tanpa memecah satu karakter UTF-8 —
// kolom VARCHAR2 tabel lama diukur dalam byte.
func cutBytes(text string, max int) string {
	if len(text) <= max {
		return text
	}
	cut := max
	for cut > 0 && !utf8.RuneStart(text[cut]) {
		cut--
	}
	return strings.TrimSpace(text[:cut])
}

// tooLongPattern membaca ORA-12899: value too large for column "OWNER"."TABEL"."KOLOM"
// (actual: N, maximum: M).
var tooLongPattern = regexp.MustCompile(`ORA-12899:.*"([A-Z0-9_]+)"\s*\(actual:\s*(\d+),\s*maximum:\s*(\d+)\)`)

// columnField memetakan kolom berkas ke isian form dan labelnya.
var columnField = map[string][2]string{
	"NAMAPELAPOR":          {"nama_pelapor", "Nama Pengirim / Pelapor Dokumen"},
	"REPORTERNAME":         {"nama_pelapor", "Nama Pengirim / Pelapor Dokumen"},
	"EMAILPENGIRIM":        {"email_pelapor", "Email Pengirim"},
	"TLPPENGIRIM":          {"telepon_pelapor", "No. HP Pengirim"},
	"NO_HP":                {"telepon_pelapor", "No. HP Pengirim"},
	"NAMAKURIRASM":         {"nama_kurir", "Nama Kurir ASM"},
	"NOPOLIS":              {"nomor_polis", "Nomor Polis"},
	"NAMATERTANGGUNG":      {"tertanggung", "Nama Tertanggung"},
	"QQNAME":               {"tertanggung", "Nama Tertanggung"},
	"NOREFERENSI":          {"nomor_rujukan", "No. Referensi/Placing Slip"},
	"LOKASIKEJADIAN":       {"lokasi_kejadian", "Lokasi Kejadian"},
	"LOCATION":             {"lokasi_kejadian", "Lokasi Kejadian"},
	"SUBJECTEMAIL":         {"subjek_email", "Subject Email"},
	"KRONOLOGIKEJADIAN":    {"kronologis", "Kronologis Kejadian"},
	"KRONOLOGI":            {"kronologis", "Kronologis Kejadian"},
	"RINCIANKERUSAKAN":     {"rincian_kerusakan", "Rincian Kerusakan"},
	"ALASANBLMTRANSFER":    {"alasan", "Keterangan Belum Transfer"},
	"KETERANGANBLMREGIST":  {"keterangan_belum_registrasi", "Keterangan Belum Registrasi"},
	"GROUPPANEL":           {"group_panel", "Group Panel"},
	"EMAILTERTANGGUNG":     {"email_tertanggung", "Email Tertanggung"},
	"SIMPENGENDARA":        {"sim_pengendara", "SIM Pengendara"},
	"RESOURCES":            {"sumber_laporan", "Source Of Reports"},
	"TANGGALTERIMADOKUMEN": {"tanggal_terima_dokumen", "Tanggal Terima Dokumen"},
}

// valueTooLarge menerjemahkan ORA-12899 menjadi pelanggaran isian yang menyebut isiannya dan
// batasnya, bukan galat sistem (RCVN.26.90/.91, 2026-10-10). Nil bila galatnya bukan itu.
func valueTooLarge(err error) error {
	if err == nil {
		return nil
	}
	m := tooLongPattern.FindStringSubmatch(err.Error())
	if m == nil {
		return nil
	}
	field, label := "", m[1]
	if f, ok := columnField[m[1]]; ok {
		field, label = f[0], f[1]
	}
	return &inboxlaporanklaim.ValidationError{Violation: []inboxlaporanklaim.Violation{{
		Field:   field,
		Message: fmt.Sprintf("%s terlalu panjang: %s karakter, paling banyak %s.", label, m[2], m[3]),
	}}}
}

// isDuplicateKey menyatakan apakah penyisipan gagal karena nomornya sudah dipakai.
func isDuplicateKey(err error) bool { return db.IsDuplicateKey(err) }

// isoDate menyiapkan tanggal untuk kolom TEKS pada tabel lama.
//
// TANGGALTERIMADOKUMEN bertipe VARCHAR2 — bukan pilihan kita, melainkan bentuk tabel yang
// sudah ada. Bentuk ISO dipakai supaya pengurutan teksnya sama dengan urutan tanggal, dan
// supaya pembacaannya kembali lewat TO_DATE tidak bergantung pada NLS server.
func isoDate(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.Format("2006-01-02")
}

// Repo membaca POOLDATA.T_CLAIMLIST_ADMIN dan membaca-menulis
// POOLDATA.CPNC_LAPORAN_KLAIM.
type Repo struct {
	db    *sql.DB
	clock inboxlaporanklaim.Clock
}

// NewRepo membentuk repo; db wajib sudah terhubung ke basis data portal yang dimaksud.
func NewRepo(db *sql.DB, clock inboxlaporanklaim.Clock) *Repo {
	return &Repo{db: db, clock: clock}
}

// criteriaWidth adalah banyaknya kode yang dikirim untuk setiap daftar kode.
//
// Bentuk kueri tetap, sehingga jumlah bind-nya harus tetap pula. Daftar yang lebih
// pendek dipadatkan dengan mengulang kode pertamanya — `IN ('002','002','002','002')`
// berarti sama persis dengan `IN ('002')`, dan itu yang membuat satu bentuk kueri dapat
// melayani kelima pilihan lini bisnis tanpa merangkai teks SQL sama sekali.
const criteriaWidth = 4

// List membaca satu halaman beserta jumlah seluruh baris yang cocok.
func (r *Repo) List(
	ctx context.Context,
	filter inboxlaporanklaim.Filter,
	page inboxlaporanklaim.Pagination,
) (inboxlaporanklaim.Page, error) {
	clean := page.Clean()

	if message, isMessage := inboxlaporanklaim.MessageFilterOf(filter.Category); isMessage {
		return r.listMessage(ctx, filter, message, clean)
	}
	return r.listReport(ctx, filter, clean)
}

// listReport melayani keenam tab yang menyaring keadaan berkas.
func (r *Repo) listReport(
	ctx context.Context,
	filter inboxlaporanklaim.Filter,
	page inboxlaporanklaim.Pagination,
) (inboxlaporanklaim.Page, error) {
	category := categoryArguments(filter.Category)

	countArgument := append(scopeArguments(filter), category...)
	total, err := r.count(ctx, "claim_report_count_body", countArgument)
	if err != nil {
		return inboxlaporanklaim.Page{}, err
	}

	listArgument := append(append([]any(nil), countArgument...), page.Offset(), page.Size)
	rows, err := r.query(ctx, "claim_report_list_body", listArgument)
	if err != nil {
		return inboxlaporanklaim.Page{}, err
	}

	return inboxlaporanklaim.Page{Report: rows, Total: total, Pagination: page}, nil
}

// listMessage melayani ketiga tab komunikasi.
func (r *Repo) listMessage(
	ctx context.Context,
	filter inboxlaporanklaim.Filter,
	message inboxlaporanklaim.MessageFilter,
	page inboxlaporanklaim.Pagination,
) (inboxlaporanklaim.Page, error) {
	// Tanpa identitas pemanggil, ketiga tab ini tidak punya arti: seluruhnya menyaring
	// berkas milik pemanggil sendiri. Menjawab daftar kosong lebih benar daripada
	// menjawab percakapan milik orang lain.
	if filter.Operator == "" {
		return inboxlaporanklaim.Page{Report: nil, Total: 0, Pagination: page}, nil
	}

	messageArgument := messageArguments(filter.Operator, message)

	countArgument := append(scopeArguments(filter), messageArgument...)
	total, err := r.count(ctx, "claim_report_message_count_body", countArgument)
	if err != nil {
		return inboxlaporanklaim.Page{}, err
	}

	// Keenam bind percakapan dikirim DUA KALI: sekali untuk mengambil pesan terakhirnya di
	// SELECT, sekali untuk EXISTS pada WHERE. Keduanya memakai penyaring yang sama persis
	// — bila berbeda, baris dapat lolos EXISTS lalu menampilkan pesan kosong.
	//
	// Yang di SELECT dikirim LEBIH DULU karena di teks kuerinya ia muncul lebih dulu.
	// Oracle mengikat menurut urutan kemunculan, bukan menurut nomor pada `:n`.
	//
	// `messageArgument[2:]` — enam nilai: status percakapan, pengirim yang dicari, dan
	// pengirim yang dihindari, masing-masing penjaga dan pembandingnya. Dua yang pertama
	// (identitas pembuat berkas) hanya dipakai WHERE.
	listArgument := make([]any, 0, len(countArgument)+8)
	listArgument = append(listArgument, messageArgument[2:]...)
	listArgument = append(listArgument, countArgument...)
	listArgument = append(listArgument, page.Offset(), page.Size)

	rows, err := r.query(ctx, "claim_report_message_body", listArgument)
	if err != nil {
		return inboxlaporanklaim.Page{}, err
	}

	return inboxlaporanklaim.Page{Report: rows, Total: total, Pagination: page}, nil
}

// Summarize menghitung kedelapan pencacah dalam satu kueri.
func (r *Repo) Summarize(
	ctx context.Context,
	filter inboxlaporanklaim.Filter,
) (inboxlaporanklaim.Summary, error) {
	operator := emptyToNil(filter.Operator)

	// Sembilan bind pencacah komunikasi DIKIRIM LEBIH DULU, karena di teks kuerinya
	// merekalah yang muncul lebih dulu — ada di SELECT, sedangkan penyaring ada di WHERE.
	//
	// Oracle mengikat menurut URUTAN KEMUNCULAN, bukan menurut nomor pada `:n`. Sebelum
	// ini argumennya dikirim dalam urutan nomor, sehingga penyaring cabang menerima NULL
	// dan pencacahnya menerima kode cabang: lencana di atas tab menyebut 115 sementara
	// tabel di bawahnya kosong. Gagal tanpa satu pun galat.
	argument := make([]any, 0, 30)
	for i := 0; i < 9; i++ {
		argument = append(argument, operator)
	}
	argument = append(argument, scopeArguments(filter)...)

	row := r.db.QueryRowContext(ctx, sourced("claim_report_summary_body"), argument...)

	var summary inboxlaporanklaim.Summary
	// SUM atas tabel kosong menghasilkan NULL, bukan nol; sql.NullInt64 yang
	// menampungnya. COUNT tidak pernah NULL, tetapi dibaca sama supaya satu perubahan
	// kueri tidak menuntut dua gaya pembacaan.
	var total, notTransferred, unregistered, outstanding, accepted sql.NullInt64
	var unanswered, waiting, replied sql.NullInt64

	if err := row.Scan(&total, &notTransferred, &unregistered, &outstanding, &accepted,
		&unanswered, &waiting, &replied); err != nil {
		return inboxlaporanklaim.Summary{}, fmt.Errorf("inboxlaporanklaim/sqlstore: menghitung pencacah: %w", err)
	}

	summary.Total = int(total.Int64)
	summary.NotTransferred = int(notTransferred.Int64)
	summary.Unregistered = int(unregistered.Int64)
	summary.Outstanding = int(outstanding.Int64)
	summary.Accepted = int(accepted.Int64)
	summary.MessageUnanswered = int(unanswered.Int64)
	summary.MessageWaiting = int(waiting.Int64)
	summary.MessageReplied = int(replied.Int64)
	return summary, nil
}

// ListRegions membaca isi dropdown "Pilih Kanwil".
func (r *Repo) ListRegions(ctx context.Context) ([]inboxlaporanklaim.Region, error) {
	rows, err := r.db.QueryContext(ctx, getQuery("claim_report_region_list"))
	if err != nil {
		return nil, fmt.Errorf("inboxlaporanklaim/sqlstore: membaca daftar kanwil: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []inboxlaporanklaim.Region
	for rows.Next() {
		var code sql.NullString
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("inboxlaporanklaim/sqlstore: membaca kanwil: %w", err)
		}
		clean := strings.TrimSpace(code.String)
		if clean == "" {
			continue
		}
		// Nama kanwil TIDAK ada di basis data — POOLDATA.BRANCH hanya menyimpan
		// kodenya di BASTERRITORY, dan tabel nama kanwil tidak ikut dikirim bersama
		// export (R-08). Kodenya dipakai sebagai nama supaya dropdown tetap dapat
		// dipakai, dan itu dicatat sebagai keterbatasan — bukan ditambal nama karangan
		// yang tampak sah tetapi tidak dapat dicocokkan dengan apa pun.
		result = append(result, inboxlaporanklaim.Region{Code: clean, Name: clean})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inboxlaporanklaim/sqlstore: menelusuri daftar kanwil: %w", err)
	}
	return result, nil
}

// Get membaca satu berkas laporan dari tabel mana pun asalnya, LENGKAP dengan isian
// formnya.
//
// Ia memakai badan kuerinya sendiri dan pembaca barisnya sendiri: kueri detail memilih
// sepuluh kolom lebih banyak daripada kueri daftar. Lihat catatan pada
// claim_report_get_body.
func (r *Repo) Get(ctx context.Context, id string) (inboxlaporanklaim.ClaimReport, error) {
	clean := strings.TrimSpace(id)

	// Dua jalur, dipilih menurut AWALAN NOMOR.
	//
	// Sejak daftar ditarik dari T_CLAIMLIST_ADMIN (Work Owner, 2026-09-23), berkas yang
	// baru dibuat belum ada di sana sampai proses pengisinya berjalan. Membacanya lewat
	// jalur daftar akan menjawab "tidak ditemukan" tepat sesudah tombol "Buat Baru"
	// ditekan — tombol yang tampak rusak, kelas kegagalan yang sudah dua kali menimpa
	// layar ini.
	//
	// Awalan `RCVN` hanya diterbitkan aplikasi ini, sehingga pemilihannya pasti.
	// Itu pula alasan awalan itu ditetapkan: asal sebuah berkas terbaca dari nomornya
	// tanpa tabel pemetaan.
	query := sourced("claim_report_get_body")
	own := inboxlaporanklaim.IssuedHere(clean)
	if own {
		query = getQuery("claim_report_get_own_body")
	}

	// Hanya jalur berkas sendiri yang diterjemahkan; jalur warisan dibiarkan apa adanya,
	// karena tabel Pega yang hilang punya sebab dan perbaikan yang berbeda.
	translate := func(err error) error { return err }
	if own {
		translate = ownTable
	}

	rows, err := r.db.QueryContext(ctx, query, clean)
	if err != nil {
		return inboxlaporanklaim.ClaimReport{}, fmt.Errorf("inboxlaporanklaim/sqlstore: membaca berkas: %w", translate(err))
	}
	defer func() { _ = rows.Close() }()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return inboxlaporanklaim.ClaimReport{}, fmt.Errorf("inboxlaporanklaim/sqlstore: membaca berkas: %w", err)
		}
		return inboxlaporanklaim.ClaimReport{}, inboxlaporanklaim.ErrNotFound
	}

	report, err := scanDetailRow(rows)
	if err != nil {
		return inboxlaporanklaim.ClaimReport{}, err
	}
	return report, nil
}

// Update menyimpan isian form ke atas berkas yang sudah ada.
func (r *Repo) Update(ctx context.Context, report inboxlaporanklaim.ClaimReport) error {
	// Baris milik Pega ditolak SEBELUM menyentuh basis data. Pernyataan UPDATE-nya
	// memang hanya mengenai tabel milik aplikasi ini, sehingga baris warisan akan
	// menghasilkan "nol baris" yang terbaca sebagai ErrNotFound — pesan yang menyesatkan
	// untuk berkas yang sebenarnya ADA tetapi bukan milik kita.
	if report.Origin == inboxlaporanklaim.OriginLegacy {
		return inboxlaporanklaim.ErrReadOnlyOrigin
	}

	// Berkas dan barisnya di T_CLAIM_PNC ditulis dalam SATU transaksi. Berkas yang
	// tersimpan tanpa barisnya di sana — atau sebaliknya — adalah keadaan yang tampak
	// benar di satu layar dan salah di layar lain.
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("inboxlaporanklaim/sqlstore: memulai transaksi: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Tanggal terima dokumen disimpan sebagai TEKS berbentuk ISO. Kolomnya bertipe
	// VARCHAR2 di tabel lama — Pega bahkan mengisinya dengan ReferenceId, bukan tanggal
	// (Rcv_ProcInsertRecivedDocument). Baris terbitan aplikasi ini selalu ISO, sehingga
	// pembacaannya kembali lewat TO_DATE pasti.
	//
	// Pernyataan ini berjalan LEBIH DULU: ia mengunci baris berkasnya, sehingga dua
	// penyimpanan bersamaan atas berkas yang sama tidak dapat sama-sama menyisipkan baris
	// di T_CLAIM_PNC — tabel itu tidak punya kunci utama yang akan menolaknya.
	result, err := tx.ExecContext(ctx, getQuery("claim_report_update"),
		isoDate(report.ReceivedDate),
		nullTime(report.DateOfLoss),
		emptyToNil(report.ReporterName),
		emptyToNil(report.ReporterEmail),
		emptyToNil(report.ReporterPhone),
		emptyToNil(report.CourierName),
		emptyToNil(report.PolicyNumber),
		// Kolomnya 100 byte; nama lengkap tetap di T_CLAIM_PNC.QQNAME (upsertClaimRow).
		emptyToNil(cutBytes(report.InsuredName, inboxlaporanklaim.StoredInsuredNameLength)),
		emptyToNil(report.ReferenceNumber),
		int64(report.EstimateValue),
		emptyToNil(report.LossLocation),
		emptyToNil(report.EmailSubject),
		emptyToNil(report.Chronology),
		emptyToNil(report.DamageDetail),
		emptyToNil(report.Reason),
		emptyToNil(report.NotRegisteredNote),
		emptyToNil(report.GroupPanel),
		emptyToNil(report.InsuredEmail),
		emptyToNil(report.DriverLicense),
		emptyToNil(report.ReportSource),
		report.ID,
	)
	if err != nil {
		if tooLong := valueTooLarge(err); tooLong != nil {
			return tooLong
		}
		return fmt.Errorf("inboxlaporanklaim/sqlstore: menyimpan %q ke POOLDATA.T_CLAIM_RECIVEDCLAIM: %w", report.ID, ownTable(err))
	}

	// Driver yang tidak dapat melaporkan jumlah baris tidak boleh diartikan sebagai
	// kegagalan: pernyataannya sendiri sudah berhasil.
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return inboxlaporanklaim.ErrNotFound
	}

	if err := upsertClaimRow(ctx, tx, report); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("inboxlaporanklaim/sqlstore: menutup transaksi simpan: %w", err)
	}
	return nil
}

// upsertClaimRow menuliskan baris berkas ini di POOLDATA.T_CLAIM_PNC.
//
// Work Owner, 2026-09-26: pembuatan RCVN ikut masuk ke T_CLAIM_PNC — seperti Pega, yang
// menyimpan 436 baris ber-CLAIMID RCV di tabel itu. Isinya ditaruh di kolom warisan yang
// juga dipakai baris klaim, lihat catatan pada inboxlaporanklaim.sql.
//
// # UPDATE lebih dulu, INSERT hanya bila tidak ada
//
// T_CLAIM_PNC tidak punya kunci utama, sehingga penyisipan kedua tidak akan ditolak —
// ia akan diam-diam menjadi baris ganda. UPDATE lebih dulu membuat penyimpanan berulang
// memperbarui baris yang sama, dan sekaligus melengkapi berkas yang dibuat sebelum
// perubahan ini: barisnya lahir pada penyimpanan berikutnya.
//
// # Hanya berkas milik aplikasi ini
//
// Berkas Pega tidak pernah sampai ke sini (Update menolaknya lebih awal), dan kedua
// pernyataan tetap memagari CLAIMID `RCVN%` — pagar `P-1` tidak diserahkan pada urutan
// pemanggilan.
func upsertClaimRow(ctx context.Context, tx *sql.Tx, report inboxlaporanklaim.ClaimReport) error {
	if !inboxlaporanklaim.IssuedHere(report.ID) {
		return nil
	}

	args := []any{
		emptyToNil(report.PolicyNumber),
		emptyToNil(report.InsuredName),
		calendarDate(report.DateOfLoss),
		// Langkah 14: ClaimData.ReportDate <- ReceiveDocument.ReceivedDate.
		calendarDate(report.ReceivedDate),
		emptyToNil(report.ReporterName),
		emptyToNil(report.ReporterPhone),
		emptyToNil(report.LossLocation),
		emptyToNil(report.Chronology),
		emptyToNil(report.BranchCode),
		// Langkah 14: ClaimData.SubjectEmail <- ReceiveDocument.SubjectEmail.
		emptyToNil(report.EmailSubject),
		report.ID,
	}

	result, err := tx.ExecContext(ctx, getQuery("claim_report_pnc_update"), args...)
	if tooLong := valueTooLarge(err); tooLong != nil {
		return tooLong
	}
	if err != nil {
		return fmt.Errorf("inboxlaporanklaim/sqlstore: memperbarui baris %q di POOLDATA.T_CLAIM_PNC: %w", report.ID, err)
	}
	if affected, err := result.RowsAffected(); err == nil && affected > 0 {
		return nil
	}
	if _, err := tx.ExecContext(ctx, getQuery("claim_report_pnc_insert"), args...); err != nil {
		if tooLong := valueTooLarge(err); tooLong != nil {
			return tooLong
		}
		return fmt.Errorf("inboxlaporanklaim/sqlstore: menyisipkan baris %q di POOLDATA.T_CLAIM_PNC: %w", report.ID, err)
	}
	return nil
}

// calendarDate mengikat sebuah TANGGAL KALENDER ke kolom Oracle bertipe DATE.
//
// DATE tidak menyimpan zona; ia menyimpan jam dinding apa adanya, dan driver memasang
// zona sesi (WIB) saat membacanya kembali. Waktu UTC yang diikat langsung akan kehilangan
// sehari — tengah malam WIB selalu jatuh di hari sebelumnya menurut UTC (`R-12`, bab 57
// catatan pengembangan). Karena itu tanggalnya dipotong ke tengah malam WIB lebih dulu.
func calendarDate(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return clock.DateWIB(t)
}

// Insert menerbitkan nomor lalu menyimpan berkas baru — ke tabel milik aplikasi ini.
//
// Keduanya berjalan di dalam SATU transaksi. Ini pengecualian yang disadari terhadap
// `08-TECHNICAL-STRATEGY.md` §4.5 yang menempatkan batas transaksi di lapisan aplikasi:
// nomor yang sudah diambil dari sequence tidak dapat dikembalikan, sehingga penyisipan
// yang gagal setelahnya meninggalkan lubang pada deret nomor. Membungkusnya tidak
// menutup lubang itu seluruhnya — sequence Oracle memang tidak ikut di-rollback — tetapi
// ia memastikan tidak ada berkas separuh jadi yang tersimpan.
func (r *Repo) Insert(
	ctx context.Context,
	report inboxlaporanklaim.ClaimReport,
) (inboxlaporanklaim.ClaimReport, error) {
	year := r.clock.Now().Year()

	// Percobaan diulang karena nomornya diturunkan dari isi tabel, bukan dari sequence
	// (lihat claim_report_next_sequence). Dua permintaan bersamaan dapat membaca nomor
	// yang sama; yang menjaganya adalah kunci utama tabel, dan tabrakan itu diselesaikan
	// dengan mengambil nomor berikutnya.
	//
	// Batasnya kecil dengan sengaja: tabrakan berulang sampai lima kali berarti ada yang
	// lain yang salah, dan berputar lebih lama hanya menunda kegagalan itu terlihat.
	const maxAttempt = 5
	var lastErr error
	for attempt := 0; attempt < maxAttempt; attempt++ {
		saved, err := r.insertOnce(ctx, report, year)
		switch {
		case err == nil:
			return saved, nil
		case isDuplicateKey(err):
			lastErr = err
			continue
		default:
			return inboxlaporanklaim.ClaimReport{}, err
		}
	}
	return inboxlaporanklaim.ClaimReport{}, fmt.Errorf(
		"inboxlaporanklaim/sqlstore: nomor laporan bertabrakan %d kali berturut-turut: %w",
		maxAttempt, lastErr)
}

// insertOnce menjalankan satu percobaan penerbitan nomor beserta penyisipannya.
func (r *Repo) insertOnce(
	ctx context.Context,
	report inboxlaporanklaim.ClaimReport,
	year int,
) (inboxlaporanklaim.ClaimReport, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return inboxlaporanklaim.ClaimReport{}, fmt.Errorf("inboxlaporanklaim/sqlstore: memulai transaksi: %w", err)
	}
	// Rollback dipanggil tanpa syarat; setelah Commit berhasil ia tidak berakibat apa
	// pun. Tanpa ini, satu jalur galat yang terlewat meninggalkan transaksi menggantung
	// dan menahan kunci baris sampai koneksinya didaur ulang.
	defer func() { _ = tx.Rollback() }()

	var sequence int64
	if err := tx.QueryRowContext(ctx, getQuery("claim_report_next_sequence"),
		fmt.Sprintf("%02d", year%100)).Scan(&sequence); err != nil {
		return inboxlaporanklaim.ClaimReport{}, fmt.Errorf("inboxlaporanklaim/sqlstore: mengambil nomor urut: %w", ownTable(err))
	}

	saved := report
	saved.Origin = inboxlaporanklaim.OriginNew
	saved.ID = inboxlaporanklaim.FormatReportNumber(year, sequence)
	saved.Position = inboxlaporanklaim.DerivePosition(saved.Registered(), saved.Transferred)

	if _, err := tx.ExecContext(ctx, getQuery("claim_report_insert"),
		saved.ID, saved.CreatedAt, saved.BranchCode,
		saved.CreatedBy, emptyToNil(saved.ReporterName),
	); err != nil {
		return inboxlaporanklaim.ClaimReport{}, fmt.Errorf("inboxlaporanklaim/sqlstore: menyisipkan %q ke POOLDATA.T_CLAIM_RECIVEDCLAIM: %w", saved.ID, ownTable(err))
	}

	// Barisnya di T_CLAIM_PNC lahir bersama berkasnya, di transaksi yang sama. Bila
	// nomornya bertabrakan dan percobaan diulang, transaksi ini dibatalkan seluruhnya —
	// tidak ada baris tertinggal atas nomor yang tidak jadi dipakai.
	if err := upsertClaimRow(ctx, tx, saved); err != nil {
		return inboxlaporanklaim.ClaimReport{}, err
	}

	if err := tx.Commit(); err != nil {
		return inboxlaporanklaim.ClaimReport{}, fmt.Errorf("inboxlaporanklaim/sqlstore: menutup transaksi sisip: %w", err)
	}
	return saved, nil
}

// CheckTable memastikan kedua tabel dapat dibaca akun aplikasi.
//
// Ia tidak mengambil satu baris pun, sehingga aman dijalankan terhadap produksi.
// Keduanya diperiksa terpisah supaya pesannya menyebut tabel mana yang bermasalah —
// satu pesan "tidak dapat dibaca" untuk dua tabel tidak menolong siapa pun.
func (r *Repo) CheckTable(ctx context.Context) error {
	for name, table := range map[string]string{
		"claim_report_check_table":        "POOLDATA.T_CLAIM_RECIVEDCLAIM",
		"claim_report_check_legacy_table": "POOLDATA.T_CLAIMLIST_ADMIN",
		"claim_report_check_claim_row":    "POOLDATA.T_CLAIM_PNC (kolom baris berkas)",
	} {
		rows, err := r.db.QueryContext(ctx, getQuery(name))
		if err != nil {
			return fmt.Errorf("inboxlaporanklaim/sqlstore: %s tidak dapat dibaca: %w", table, err)
		}
		err = rows.Err()
		_ = rows.Close()
		if err != nil {
			return fmt.Errorf("inboxlaporanklaim/sqlstore: %s tidak dapat dibaca: %w", table, err)
		}
	}
	return nil
}

// count menjalankan salah satu kueri pencacah.
func (r *Repo) count(ctx context.Context, name string, argument []any) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, sourced(name), argument...).Scan(&total); err != nil {
		return 0, fmt.Errorf("inboxlaporanklaim/sqlstore: menghitung baris: %w", err)
	}
	return total, nil
}

// query menjalankan salah satu kueri daftar dan membaca seluruh barisnya.
func (r *Repo) query(ctx context.Context, name string, argument []any) ([]inboxlaporanklaim.ClaimReport, error) {
	rows, err := r.db.QueryContext(ctx, sourced(name), argument...)
	if err != nil {
		return nil, fmt.Errorf("inboxlaporanklaim/sqlstore: membaca daftar: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var result []inboxlaporanklaim.ClaimReport
	for rows.Next() {
		report, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, report)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("inboxlaporanklaim/sqlstore: menelusuri daftar: %w", err)
	}
	return result, nil
}

// scopeArguments menyusun dua puluh satu bind pertama, sama untuk setiap kueri.
//
// Setiap penyaring muncul dua kali di dalam SQL — sekali pada pemeriksaan IS NULL,
// sekali pada perbandingannya — sehingga nilainya dikirim dua kali pula. Nomor bind
// sengaja dibedakan, bukan diulang, supaya tidak bergantung pada tafsir driver terhadap
// bind bernomor sama. Pola yang sama dipakai modul Master Rekening.
func scopeArguments(filter inboxlaporanklaim.Filter) []any {
	branch := emptyToNil(filter.BranchCode)
	region := emptyToNil(filter.RegionCode)
	keyword := emptyToNil(filter.Keyword)

	groupPanel, businessGroupIn, businessGroupNotIn := filter.BusinessLine.Criteria()

	argument := []any{
		branch, branch, // :1 :2
		region, region, // :3 :4
		keyword, keyword, // :5 :6
	}
	argument = append(argument, codeList(groupPanel)...)         // :7 … :11
	argument = append(argument, codeList(businessGroupIn)...)    // :12 … :16
	argument = append(argument, codeList(businessGroupNotIn)...) // :17 … :21
	return argument
}

// codeList menyusun satu penjaga diikuti empat kode.
//
// Daftar kosong menghasilkan lima NULL, dan penjaganya yang NULL itulah yang membuat
// seluruh penyaring tidak berlaku. Daftar yang lebih pendek dari empat dipadatkan
// dengan mengulang kode terakhirnya — lihat criteriaWidth. Daftar yang LEBIH PANJANG
// dipotong, dan itu tidak dapat terjadi dengan isi ListBusinessLines hari ini; bila
// kelak bertambah, uji pada berkas ini yang akan menangkapnya lebih dulu.
func codeList(code []string) []any {
	if len(code) == 0 {
		result := make([]any, criteriaWidth+1)
		return result
	}

	result := make([]any, 0, criteriaWidth+1)
	result = append(result, code[0])
	for i := 0; i < criteriaWidth; i++ {
		if i < len(code) {
			result = append(result, code[i])
			continue
		}
		result = append(result, code[len(code)-1])
	}
	return result
}

// categoryArguments menyusun bind :22 … :26 untuk keenam tab non-komunikasi.
//
// Aturan tiap tab diterjemahkan menjadi penanda, bukan menjadi potongan SQL. Dengan
// begitu bentuk kuerinya tetap satu dan dapat dibaca utuh — dan aturan tabnya tetap
// hidup di Go, tempat ia dapat diuji tanpa basis data.
func categoryArguments(category inboxlaporanklaim.Category) []any {
	var position any
	var accepted, rejected any

	// Berkas selesai dan ditolak dikecualikan di seluruh tab KECUALI dua: tab akseptasi
	// tidak menyaring PYSTATUSWORK sama sekali di kueri lama, dan tab penolakan justru
	// mencari yang ditolak.
	excludeResolved := "1"

	switch category {
	case inboxlaporanklaim.CategoryOutstanding:
		position = string(inboxlaporanklaim.PositionOutstanding)
	case inboxlaporanklaim.CategoryUnregistered:
		position = string(inboxlaporanklaim.PositionNotRegistered)
	case inboxlaporanklaim.CategoryNotTransferred:
		position = string(inboxlaporanklaim.PositionNotTransferred)
	case inboxlaporanklaim.CategoryAccepted:
		position = string(inboxlaporanklaim.PositionOutstanding)
		accepted = "1"
		excludeResolved = "0"
	case inboxlaporanklaim.CategoryRejected:
		rejected = "1"
		excludeResolved = "0"
	case inboxlaporanklaim.CategoryAll:
		// Tanpa penyaring tambahan.
	}

	return []any{excludeResolved, position, position, accepted, rejected}
}

// messageArguments menyusun bind :22 … :29 untuk ketiga tab komunikasi.
//
// Keenam bind terakhir dikirim ulang oleh pemanggil untuk mengambil pesan terakhirnya;
// karena itu keempatnya disusun sebagai satu blok yang dapat dipotong ulang.
func messageArguments(operator string, filter inboxlaporanklaim.MessageFilter) []any {
	value := emptyToNil(operator)

	var senderEqual, senderNotEqual any
	if filter.FromSelf {
		senderEqual = value
	} else {
		senderNotEqual = value
	}

	return []any{
		value, value, // :22 :23 — berkas milik pemanggil
		filter.Status, filter.Status, // :24 :25 — status percakapan
		senderEqual, senderEqual, // :26 :27
		senderNotEqual, senderNotEqual, // :28 :29
	}
}

// scanRow membaca satu baris hasil kueri menjadi ClaimReport.
//
// Seluruh kolom teks dibaca lewat sql.NullString lalu dipangkas. Dua sebab, dan
// keduanya nyata pada tabel warisan: kolom bertipe CHAR berlebar tetap memadatkan
// nilainya dengan spasi tanpa memberi tanda apa pun, dan barisnya dapat memuat NULL
// karena DDL-nya tidak diketahui (R-08).
func scanRow(rows *sql.Rows) (inboxlaporanklaim.ClaimReport, error) {
	var (
		id, claimNumber, assignmentRef          sql.NullString
		policyNumber, insuredName, businessName sql.NullString
		reporterName, referenceNumber           sql.NullString
		dateOfLoss, createdAt, agingAt          sql.NullTime
		createdBy, branchCode, branchName       sql.NullString
		reason, emailSubject                    sql.NullString
		position, origin, lastMessage           sql.NullString
		agingValue                              sql.NullInt64
	)

	// Urutan Scan mengikuti urutan kolom pada ketiga badan kueri pembaca, dan ketiganya
	// menyebut kolom yang sama persis dalam urutan yang sama. Menambah satu kolom di
	// salah satunya tanpa menambahkannya di sini akan menggeser SELURUH nilai setelahnya
	// — dan pergeseran itu tidak menghasilkan galat, hanya kolom yang berisi isi kolom
	// sebelahnya.
	if err := rows.Scan(
		&id, &claimNumber, &assignmentRef,
		&policyNumber, &insuredName, &reporterName, &businessName, &referenceNumber,
		&dateOfLoss, &createdAt, &createdBy, &branchCode, &branchName,
		&agingAt, &reason, &emailSubject, &position, &origin, &agingValue, &lastMessage,
	); err != nil {
		return inboxlaporanklaim.ClaimReport{}, fmt.Errorf("inboxlaporanklaim/sqlstore: membaca baris: %w", err)
	}

	report := inboxlaporanklaim.ClaimReport{
		ID:              text(id),
		ClaimNumber:     text(claimNumber),
		AssignmentRef:   text(assignmentRef),
		PolicyNumber:    text(policyNumber),
		InsuredName:     text(insuredName),
		ReporterName:    text(reporterName),
		BusinessName:    text(businessName),
		ReferenceNumber: text(referenceNumber),
		DateOfLoss:      moment(dateOfLoss),
		CreatedAt:       moment(createdAt),
		CreatedBy:       text(createdBy),
		BranchCode:      text(branchCode),
		BranchName:      text(branchName),
		AgingAt:         moment(agingAt),
		AgingValue:      number(agingValue),
		Reason:          text(reason),
		EmailSubject:    text(emailSubject),
		LastMessage:     text(lastMessage),
		Position:        inboxlaporanklaim.Position(text(position)),
		Origin:          inboxlaporanklaim.Origin(text(origin)),
	}

	// Transferred diturunkan dari ada-tidaknya rujukan penugasan, sama seperti kueri
	// lama yang menguji `statuslock_1 IS NOT NULL`. Untuk baris milik aplikasi ini,
	// posisinya sudah dihitung kueri dari kolomnya sendiri.
	report.Transferred = report.Position != inboxlaporanklaim.PositionNotTransferred
	return report, nil
}

// scanDetailRow membaca satu baris kueri detail — kolom daftar ditambah sepuluh isian
// form.
//
// Ia terpisah dari scanRow karena kueri detail memang memilih lebih banyak kolom. Dua
// pembaca untuk dua bentuk kueri, bukan satu pembaca yang menebak: `Scan` membaca secara
// posisi, dan satu kolom yang bergeser TIDAK menghasilkan galat — hanya kolom yang berisi
// isi kolom sebelahnya.
func scanDetailRow(rows *sql.Rows) (inboxlaporanklaim.ClaimReport, error) {
	var (
		id, claimNumber, assignmentRef          sql.NullString
		policyNumber, insuredName, businessName sql.NullString
		reporterName, referenceNumber           sql.NullString
		dateOfLoss, createdAt, agingAt          sql.NullTime
		createdBy, branchCode, branchName       sql.NullString
		reason, emailSubject                    sql.NullString
		position, origin                        sql.NullString
		agingValue                              sql.NullInt64

		receivedDate                 sql.NullTime
		reporterEmail, reporterPhone sql.NullString
		courierName, lossLocation    sql.NullString
		estimateValue, documentCount sql.NullInt64
		chronology, damageDetail     sql.NullString
		notRegisteredNote            sql.NullString

		groupPanel, insuredEmail    sql.NullString
		driverLicense, reportSource sql.NullString

		lastMessage sql.NullString
	)

	if err := rows.Scan(
		&id, &claimNumber, &assignmentRef,
		&policyNumber, &insuredName, &reporterName, &businessName, &referenceNumber,
		&dateOfLoss, &createdAt, &createdBy, &branchCode, &branchName,
		&agingAt, &reason, &emailSubject, &position, &origin, &agingValue,
		&receivedDate, &reporterEmail, &reporterPhone, &courierName,
		&estimateValue, &lossLocation, &chronology, &damageDetail,
		&notRegisteredNote, &documentCount,
		&groupPanel, &insuredEmail, &driverLicense, &reportSource,
		&lastMessage,
	); err != nil {
		return inboxlaporanklaim.ClaimReport{}, fmt.Errorf("inboxlaporanklaim/sqlstore: membaca baris detail: %w", err)
	}

	report := inboxlaporanklaim.ClaimReport{
		ID:                id.String,
		ClaimNumber:       text(claimNumber),
		AssignmentRef:     text(assignmentRef),
		PolicyNumber:      text(policyNumber),
		InsuredName:       text(insuredName),
		ReporterName:      text(reporterName),
		BusinessName:      text(businessName),
		ReferenceNumber:   text(referenceNumber),
		DateOfLoss:        moment(dateOfLoss),
		CreatedAt:         moment(createdAt),
		CreatedBy:         text(createdBy),
		BranchCode:        text(branchCode),
		BranchName:        text(branchName),
		AgingAt:           moment(agingAt),
		AgingValue:        number(agingValue),
		Reason:            text(reason),
		EmailSubject:      text(emailSubject),
		Position:          inboxlaporanklaim.Position(text(position)),
		Origin:            inboxlaporanklaim.Origin(text(origin)),
		ReceivedDate:      moment(receivedDate),
		ReporterEmail:     text(reporterEmail),
		ReporterPhone:     text(reporterPhone),
		CourierName:       text(courierName),
		EstimateValue:     inboxlaporanklaim.Money(estimateValue.Int64),
		LossLocation:      text(lossLocation),
		Chronology:        text(chronology),
		DamageDetail:      text(damageDetail),
		NotRegisteredNote: text(notRegisteredNote),
		DocumentCount:     int(documentCount.Int64),
		GroupPanel:        text(groupPanel),
		InsuredEmail:      text(insuredEmail),
		DriverLicense:     text(driverLicense),
		ReportSource:      text(reportSource),
		LastMessage:       text(lastMessage),
	}
	report.ID = strings.TrimSpace(report.ID)
	report.Transferred = report.Position != inboxlaporanklaim.PositionNotTransferred
	return report, nil
}

func text(value sql.NullString) string { return strings.TrimSpace(value.String) }

// number mengubah angka yang boleh NULL menjadi teks, dan NULL menjadi teks kosong.
//
// Ia dipakai kolom yang hanya DIGAMBAR, tidak pernah dihitung. Mengembalikan 0 untuk NULL
// akan membuat kolom yang memang belum diisi tergambar sebagai angka nol — dua keadaan
// berbeda yang tidak lagi dapat dibedakan pembacanya.
func number(value sql.NullInt64) string {
	if !value.Valid {
		return ""
	}
	return strconv.FormatInt(value.Int64, 10)
}

// nullTime mengubah waktu nol menjadi NULL.
//
// Tanpa ini, tanggal yang memang belum diisi tersimpan sebagai tahun 1 — dan tanggal itu
// tampak sah, lolos setiap pemeriksaan, lalu muncul di layar sebagai berkas paling
// tertunggak yang pernah ada.
func nullTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}

func moment(value sql.NullTime) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

// emptyToNil mengubah teks kosong menjadi NULL.
//
// Itulah yang membuat satu bentuk kueri melayani seluruh gabungan penyaring tanpa
// merangkai teks SQL: penyaring yang tidak diisi menjadi NULL, dan `:n IS NULL OR …`
// membuatnya tidak berlaku.
func emptyToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

var _ inboxlaporanklaim.Repo = (*Repo)(nil)

// FindPolicy membaca data polis pengisi form dari POOLDATA.T_GENERAL.
//
// Tabel itu milik sistem polis dan hanya dibaca; tidak ada satu pun penulisan dari sini.
func (r *Repo) FindPolicy(ctx context.Context, number string) (inboxlaporanklaim.Policy, bool, error) {
	var insured, businessCode, businessName, reference, panel, syariah, leader sql.NullString
	err := r.db.QueryRowContext(ctx, getQuery("claim_report_policy_find"), number).
		Scan(&insured, &businessCode, &businessName, &reference, &panel, &syariah, &leader)
	if err == sql.ErrNoRows {
		return inboxlaporanklaim.Policy{}, false, nil
	}
	if err != nil {
		return inboxlaporanklaim.Policy{}, false,
			fmt.Errorf("inboxlaporanklaim/sqlstore: membaca polis dari T_GENERAL: %w", err)
	}
	return inboxlaporanklaim.Policy{
		Number:          number,
		InsuredName:     text(insured),
		BusinessCode:    text(businessCode),
		BusinessName:    text(businessName),
		ReferenceNumber: text(reference),
		GroupPanel:      text(panel),
		Leader:          text(leader),
		// Pega membandingkan SYARIAHSTATUS==1; kolomnya teks, dan nilai kosong berarti
		// bukan Syariah.
		Syariah: text(syariah) == "1",
	}, true, nil
}
