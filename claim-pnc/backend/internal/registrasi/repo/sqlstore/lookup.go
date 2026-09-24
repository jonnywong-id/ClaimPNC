package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/registrasi"
)

// ExchangeRateSource membaca kurs dari POOLDATA.M_CURRENCYSTANDARD.
//
// Ia MENGGANTIKAN function warisan `GETCURRENCYSTANDARD`, dan dua cacatnya tidak dibawa
// (`D-48`, `D-49` butir 4 dan 5):
//
//	warisan                              di sini
//	mengabaikan tanggal yang diterima    kurs dicari pada TANGGAL yang diminta
//	RETURN 1 bila kurs tidak ada         ErrExchangeRateNotFound; klaim ditolak
//
// Cacat kedua yang paling merugikan: kurs `1` membuat klaim valuta asing menyusut
// menjadi seperseribu nilainya, lalu lolos ambang komite tanpa ada yang menyadarinya.
type ExchangeRateSource struct {
	db *sql.DB
}

// NewExchangeRateSource membentuk sumber kurs di atas sebuah koneksi.
func NewExchangeRateSource(db *sql.DB) *ExchangeRateSource {
	return &ExchangeRateSource{db: db}
}

// Find mengembalikan kurs sebuah mata uang pada sebuah tanggal.
func (s *ExchangeRateSource) Find(
	ctx context.Context,
	currency string,
	date time.Time,
) (registrasi.ExchangeRate, error) {
	code := strings.TrimSpace(currency)
	if code == "" {
		return 0, fmt.Errorf("%w: mata uang kosong", registrasi.ErrExchangeRateNotFound)
	}

	exec := executorFrom(ctx, s.db)

	var text string
	err := exec.QueryRowContext(ctx, loadQuery("kurs_pada_tanggal"), code, date).Scan(&text)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Bukan galat teknis, melainkan keadaan yang `D-48` tetapkan menolak klaim.
		return 0, fmt.Errorf("%w: %s pada %s",
			registrasi.ErrExchangeRateNotFound, code, date.Format("2006-01-02"))
	case err != nil:
		return 0, fmt.Errorf("registrasi/sqlstore: membaca kurs %s: %w", code, err)
	}

	rate, err := parseRate(text)
	if err != nil {
		return 0, fmt.Errorf("registrasi/sqlstore: kurs %s pada %s tidak terbaca: %w",
			code, date.Format("2006-01-02"), err)
	}
	return rate, nil
}

// parseRate mengubah nilai kurs berbentuk teks menjadi ExchangeRate (kurs x 10.000).
//
// # Kenapa penguraiannya di sini, bukan di SQL
//
// CURRENCYVALUE bertipe VARCHAR2 dan memakai KOMA sebagai pemisah desimal — "21509,68",
// bukan "21509.68". Menguraikannya di SQL menuntut TO_NUMBER dengan format mask, dan
// hasilnya bergantung pada NLS server: nilai yang sama dapat terbaca 21509 di satu
// lingkungan dan 2150968 di lingkungan lain. Di sini penguraiannya tetap, dapat diuji,
// dan tidak bergantung pada pengaturan apa pun.
//
// Titik sebagai pemisah ribuan TIDAK diterima: "21.509,68" dan "21.509" tidak dapat
// dibedakan tanpa menebak, dan menebak nilai kurs berarti menebak nilai klaim. Bila
// bentuk itu kelak muncul di data, kegagalannya akan terlihat — bukan tersamar.
func parseRate(text string) (registrasi.ExchangeRate, error) {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return 0, errors.New("nilainya kosong")
	}
	if strings.Contains(clean, ".") && strings.Contains(clean, ",") {
		return 0, fmt.Errorf("bentuk %q memuat titik dan koma sekaligus", clean)
	}
	clean = strings.Replace(clean, ",", ".", 1)

	value, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return 0, fmt.Errorf("bentuk %q bukan angka", text)
	}
	if value <= 0 {
		return 0, fmt.Errorf("nilai %q bukan kurs yang sah", text)
	}

	// Dibulatkan ke pecahan terdekat pada skala 10.000, bukan dipotong: memotong membuat
	// setiap kurs sedikit lebih kecil dari yang sebenarnya, dan galat itu menumpuk searah
	// pada setiap klaim valuta asing.
	return registrasi.ExchangeRate(math.Round(value * float64(registrasi.ExchangeRateOne))), nil
}

// PolicyRepo membaca snapshot polis dari POOLDATA.JSON_POLIS.
//
// Data polis dimiliki GISFW (`D-03`, `D-04`); modul ini HANYA MEMBACA, dan tidak pernah
// menulis satu kolom pun ke tabel itu.
type PolicyRepo struct {
	db *sql.DB
}

// NewPolicyRepo membentuk repo polis di atas sebuah koneksi.
func NewPolicyRepo(db *sql.DB) *PolicyRepo { return &PolicyRepo{db: db} }

// Get mengembalikan satu polis menurut nomornya.
func (r *PolicyRepo) Get(ctx context.Context, policyNumber string) (registrasi.Policy, error) {
	number := strings.TrimSpace(policyNumber)
	if number == "" {
		return registrasi.Policy{}, fmt.Errorf("registrasi/sqlstore: nomor polis kosong")
	}

	exec := executorFrom(ctx, r.db)

	var (
		no, panel, businessCode, businessName sql.NullString
		start, end, policyKind, currency      sql.NullString
		insured, qqName, branch, spreading    sql.NullString
	)
	err := exec.QueryRowContext(ctx, loadQuery("polis_ambil"), number).Scan(
		&no, &panel, &businessCode, &businessName,
		&start, &end, &policyKind, &currency,
		&insured, &qqName, &branch, &spreading,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return registrasi.Policy{}, fmt.Errorf("registrasi/sqlstore: polis %q tidak ditemukan", number)
	case err != nil:
		return registrasi.Policy{}, fmt.Errorf("registrasi/sqlstore: membaca polis %q: %w", number, err)
	}

	// Nama tertanggung: TheInsured lebih dulu, QQName sebagai cadangan. Keduanya ada di
	// dokumen, dan yang pertama kosong pada sebagian polis perorangan.
	name := strings.TrimSpace(insured.String)
	if name == "" {
		name = strings.TrimSpace(qqName.String)
	}

	return registrasi.Policy{
		Number:        fallback(no.String, number),
		Line:          registrasi.LineOfBusiness(strings.TrimSpace(panel.String)),
		BusinessType:  strings.TrimSpace(businessCode.String),
		CoverageStart: parsePegaTime(start.String),
		CoverageEnd:   parsePegaTime(end.String),

		// TypeOfPolicy membawa jenis polis; polis DEKLARASI dikenali dari nilainya.
		// Nilai persisnya belum dikonfirmasi pemilik bisnis, sehingga pembandingannya
		// dipusatkan di satu fungsi yang mudah dikoreksi — bukan disebar sebagai literal.
		Declaration: isDeclarationPolicy(policyKind.String),

		Currency:   strings.TrimSpace(currency.String),
		BranchCode: strings.TrimSpace(branch.String),

		InsuredName: name,

		// Spreading dianggap tersedia bila dokumen menyatakannya selesai. Nilai yang
		// tidak dikenali diperlakukan sebagai BELUM tersedia: menganggapnya tersedia
		// akan melewatkan pemeriksaan total 100% (`I-1`), dan itu aturan yang menghitung
		// pembagian uang.
		HasSpreadingAvailable: isSpreadingReady(spreading.String),

		// CreditGuarantee tidak ada di dokumen polis. Ia ditentukan lini bisnis SPK /
		// Asuransi Kredit, dan aturannya milik modul ini — bukan data yang dibaca.
		CreditGuarantee: false,
	}, nil
}

// fallback mengembalikan nilai pertama yang tidak kosong.
func fallback(value, other string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return other
}

// parsePegaTime membaca cap waktu berbentuk Pega: `20260924T035421.349 GMT`.
//
// Bentuk yang tidak dikenali menghasilkan waktu kosong, bukan panik: polis lama dapat
// membawa bentuk yang berbeda, dan satu polis yang tidak terbaca tidak boleh
// menghentikan pendaftaran seluruh klaim. Aturan tanggal di Domain-lah yang kemudian
// menolak polis tanpa periode — di sana penolakannya menyebutkan sebabnya.
func parsePegaTime(text string) time.Time {
	clean := strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(text), "GMT"))
	if clean == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		"20060102T150405.000",
		"20060102T150405",
		"20060102",
	} {
		if t, err := time.Parse(layout, clean); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}

// isDeclarationPolicy menyatakan apakah jenis polis adalah Polis Deklarasi.
//
// Polis Deklarasi tidak dapat diklaim kecuali lini Aneka — aturan itu ada di Domain;
// yang di sini hanya pembacaan penandanya.
func isDeclarationPolicy(kind string) bool {
	return strings.EqualFold(strings.TrimSpace(kind), "DECLARATION")
}

// isSpreadingReady menyatakan apakah spreading polis sudah selesai disusun.
func isSpreadingReady(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "FINISHED", "COMPLETE", "COMPLETED", "1", "Y", "YES":
		return true
	default:
		return false
	}
}

var (
	_ registrasi.ExchangeRateSource = (*ExchangeRateSource)(nil)
	_ registrasi.PolicyRepo         = (*PolicyRepo)(nil)
)

// ── Parameter ────────────────────────────────────────────────────────────────────

// Kunci parameter pada POOLDATA.M_PARAMETER.
//
// Awalan `PNC.` memisahkan parameter modul ini dari parameter domain lain yang kelak
// mungkin ikut memakai tabel yang sama.
const (
	paramLargeLossThreshold  = "PNC.AMBANG_KERUGIAN_BESAR"
	paramLargeLossRecipients = "PNC.PENERIMA_KERUGIAN_BESAR"
)

// Parameter membaca nilai bisnis dari POOLDATA.M_PARAMETER.
//
// # Kenapa dari tabel, bukan dari konstanta
//
// `D-15` menetapkan TIDAK ADA nilai bisnis yang boleh di-hardcode. Ambang Notice of
// Large Losses dan daftar penerimanya di sistem lama tertanam di dalam activity — 66
// alamat surel, termasuk enam akun pribadi di jalur produksi — dan itulah yang
// keputusan itu ada untuk menghapusnya.
//
// Parameter yang belum diisi menghasilkan GALAT, bukan nilai bawaan. Ambang bawaan yang
// diam-diam salah lebih berbahaya daripada kegagalan yang terlihat: klaim bernilai besar
// akan lewat tanpa Notice of Large Losses, dan tidak ada yang tahu sampai ada yang
// memeriksanya.
type Parameter struct {
	db *sql.DB
}

// NewParameter membentuk pembaca parameter di atas sebuah koneksi.
func NewParameter(db *sql.DB) *Parameter { return &Parameter{db: db} }

// LargeLossThreshold mengembalikan ambang Notice of Large Losses.
func (p *Parameter) LargeLossThreshold(ctx context.Context) (registrasi.Money, error) {
	var body struct {
		NilaiSen json.Number `json:"nilai_sen"`
	}
	if err := p.read(ctx, paramLargeLossThreshold, &body); err != nil {
		return 0, err
	}
	value, err := body.NilaiSen.Int64()
	if err != nil {
		return 0, fmt.Errorf("registrasi/sqlstore: parameter %s tidak berisi angka: %w",
			paramLargeLossThreshold, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("registrasi/sqlstore: parameter %s bernilai %d; ambang harus lebih dari nol",
			paramLargeLossThreshold, value)
	}
	return registrasi.Money(value), nil
}

// LargeLossRecipients mengembalikan penerima Notice of Large Losses untuk sebuah lini.
//
// Penerima dipisah per lini bisnis karena sistem lama pun memisahkannya: Notice of Large
// Losses dikirim ke Underwriting SESUAI Group Panel, bukan ke satu daftar tunggal.
func (p *Parameter) LargeLossRecipients(
	ctx context.Context,
	line registrasi.LineOfBusiness,
) ([]string, error) {
	var body struct {
		PerLini map[string][]string `json:"per_lini"`
		Umum    []string            `json:"umum"`
	}
	if err := p.read(ctx, paramLargeLossRecipients, &body); err != nil {
		return nil, err
	}

	// Penerima lini tertentu ditambahkan ke penerima umum, bukan menggantikannya:
	// jajaran pimpinan menerima seluruh Notice, sedangkan Underwriting hanya menerima
	// lini yang ditanganinya.
	result := append([]string(nil), body.Umum...)
	result = append(result, body.PerLini[string(line)]...)

	if len(result) == 0 {
		return nil, fmt.Errorf("registrasi/sqlstore: parameter %s tidak memuat penerima untuk lini %q",
			paramLargeLossRecipients, line)
	}
	return result, nil
}

// read membaca satu parameter dan menguraikannya.
func (p *Parameter) read(ctx context.Context, id string, into any) error {
	exec := executorFrom(ctx, p.db)

	var body string
	err := exec.QueryRowContext(ctx, loadQuery("parameter_ambil"), id).Scan(&body)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return fmt.Errorf("registrasi/sqlstore: parameter %s belum diisi di POOLDATA.M_PARAMETER", id)
	case err != nil:
		return fmt.Errorf("registrasi/sqlstore: membaca parameter %s: %w", id, err)
	}
	if err := json.Unmarshal([]byte(body), into); err != nil {
		return fmt.Errorf("registrasi/sqlstore: parameter %s tidak dapat diuraikan: %w", id, err)
	}
	return nil
}

// ── Assigner ─────────────────────────────────────────────────────────────────────

// Assigner memilih penerima tugas dari POOLDATA.MST_USER_TEKNIK.
//
// Algoritmanya dipulihkan dari `RDB List/BrowsePICRandomTeam-SQL.xml` — beban paling
// sedikit mendapat tugas berikutnya — karena ketiga router yang dipakai alur ini
// (PNCAdminRouter, PNCTeknikRouter, RouterRCLDokter) tidak ada di export (`R-04`).
type Assigner struct {
	db *sql.DB
}

// NewAssigner membentuk pemilih penugasan di atas sebuah koneksi.
func NewAssigner(db *sql.DB) *Assigner { return &Assigner{db: db} }

// Assign memilih penerima tugas untuk sebuah tahap.
func (a *Assigner) Assign(
	ctx context.Context,
	stage registrasi.Stage,
	claim registrasi.Claim,
	caller string,
) (registrasi.Assignee, error) {
	// Tahap antrean bersama tidak memilih orang. Mengembalikan seseorang di sini akan
	// mengubah Workbasket menjadi Worklist tanpa ada yang memutuskannya (`D-26`).
	if stage.Queue == registrasi.QueueWorkbasket {
		return registrasi.Assignee{Workbasket: stage.Workbasket}, nil
	}

	// Tahap yang menyebut operatornya sendiri dihormati apa adanya — misalnya tahap yang
	// kembali ke petugas yang sedang mengerjakannya.
	if operator := strings.TrimSpace(stage.Operator); operator != "" {
		return registrasi.Assignee{Operator: operator}, nil
	}

	line := businessGroupOf(claim.Policy.Line)
	exec := executorFrom(ctx, a.db)

	var operator string
	err := exec.QueryRowContext(ctx, loadQuery("pic_teknik_paling_ringan"), line).Scan(&operator)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// Tidak ada petugas aktif untuk lini ini. Mengembalikan pemanggil sebagai
		// penerima akan menyembunyikan lubang master di balik perilaku yang tampak
		// normal; yang benar adalah kegagalan yang menyebutkan lininya.
		return registrasi.Assignee{}, fmt.Errorf(
			"registrasi/sqlstore: tidak ada petugas teknis aktif untuk lini %q di POOLDATA.MST_USER_TEKNIK", line)
	case err != nil:
		return registrasi.Assignee{}, fmt.Errorf("registrasi/sqlstore: memilih petugas teknis: %w", err)
	}

	if _, err := exec.ExecContext(ctx, loadQuery("pic_teknik_naikkan_beban"), operator); err != nil {
		return registrasi.Assignee{}, fmt.Errorf(
			"registrasi/sqlstore: menaikkan beban petugas %q: %w", operator, err)
	}
	return registrasi.Assignee{Operator: strings.TrimSpace(operator)}, nil
}

// businessGroupOf memetakan lini bisnis ke nilai TYPE_BUSINESS pada master petugas.
//
// Nilai yang dipakai master terverifikasi 2026-09-24: NONMBU (15 petugas aktif),
// BONDING (5), PA (3), TRAVEL (1). Lini yang tidak punya kelompoknya sendiri masuk
// NONMBU — itulah kelompok terbesar dan memang berarti "Non-Motor selain yang khusus".
func businessGroupOf(line registrasi.LineOfBusiness) string {
	switch line {
	case registrasi.LinePersonalAccident:
		return "PA"
	case registrasi.LineTravel:
		return "TRAVEL"
	default:
		return "NONMBU"
	}
}

var (
	_ registrasi.Parameter = (*Parameter)(nil)
	_ registrasi.Assigner  = (*Assigner)(nil)
)
