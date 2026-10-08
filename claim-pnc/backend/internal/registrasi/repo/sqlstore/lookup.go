package sqlstore

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"claim-pnc/internal/platform/clock"
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
	err := exec.QueryRowContext(ctx, loadQuery("kurs_pada_tanggal"), code, code, date).Scan(&text)
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

		// Jalur yang diisi Pega ke T_CLAIM_PNC saat klaim dibuat.
		quoBusinessCode, branchName, sob, sobName sql.NullString
		prodKe, policyLeader, typeOfCoins         sql.NullString
		cedingName, facShare                      sql.NullString
		deliveryAddress                           sql.NullString
		tableStart, tableEnd                      sql.NullTime
	)
	err := exec.QueryRowContext(ctx, loadQuery("polis_ambil"), number, number).Scan(
		&no, &panel, &businessCode, &businessName,
		&start, &end, &policyKind, &currency,
		&insured, &qqName, &branch, &spreading,
		&quoBusinessCode, &branchName, &sob, &sobName,
		&prodKe, &policyLeader, &typeOfCoins,
		&cedingName, &facShare, &deliveryAddress,
		&tableStart, &tableEnd,
	)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		return registrasi.Policy{}, fmt.Errorf("%w: %s", registrasi.ErrPolicyNotFound, number)
	case err != nil:
		return registrasi.Policy{}, fmt.Errorf("registrasi/sqlstore: membaca polis %q: %w", number, err)
	}

	// Nama tertanggung: TheInsured lebih dulu, QQName sebagai cadangan. Keduanya ada di
	// dokumen, dan yang pertama kosong pada sebagian polis perorangan.
	name := strings.TrimSpace(insured.String)
	if name == "" {
		name = strings.TrimSpace(qqName.String)
	}

	rows, err := r.coinsurance(ctx, exec, number, strings.TrimSpace(prodKe.String))
	if err != nil {
		return registrasi.Policy{}, err
	}
	fac, facKnown := parsePercent(facShare.String)

	return registrasi.Policy{
		Number:       fallback(no.String, number),
		Line:         registrasi.LineOfBusiness(strings.TrimSpace(panel.String)),
		BusinessType: strings.TrimSpace(businessCode.String),
		// Periode mendahulukan dokumen POLICYDATA, T_GENERAL hanya cadangan: T_GENERAL tidak
		// mengikuti endorsemen — terverifikasi 2026-10-01, satu PRODKE endorsemen tercatat 2019–2020
		// di T_GENERAL sementara dokumennya 2024, dan klaim 2024 atas polis itu hanya sah menurut
		// dokumen.
		CoverageStart: orTime(parsePegaTime(start.String), wibDate(tableStart)),
		CoverageEnd:   orTime(parsePegaTime(end.String), wibDate(tableEnd)),

		// TypeOfPolicy membawa jenis polis; polis DEKLARASI dikenali dari nilainya.
		// Nilai persisnya belum dikonfirmasi pemilik bisnis, sehingga pembandingannya
		// dipusatkan di satu fungsi yang mudah dikoreksi — bukan disebar sebagai literal.
		Declaration: isDeclarationPolicy(policyKind.String),
		Kind:        strings.TrimSpace(policyKind.String),

		Currency:   strings.TrimSpace(currency.String),
		BranchCode: strings.TrimSpace(branch.String),

		InsuredName: name,

		QQName:          strings.TrimSpace(qqName.String),
		DeliveryAddress: strings.TrimSpace(deliveryAddress.String),

		// Spreading dianggap tersedia bila dokumen menyatakannya selesai. Nilai yang
		// tidak dikenali diperlakukan sebagai BELUM tersedia: menganggapnya tersedia
		// akan melewatkan pemeriksaan total 100% (`I-1`), dan itu aturan yang menghitung
		// pembagian uang.
		HasSpreadingAvailable: isSpreadingReady(spreading.String),

		// CreditGuarantee tidak ada di dokumen polis. Ia ditentukan lini bisnis SPK /
		// Asuransi Kredit, dan aturannya milik modul ini — bukan data yang dibaca.
		CreditGuarantee: false,

		// Diisi Pega ke T_CLAIM_PNC saat klaim dibuat (PEGA_CONVERT_JSONKLAIM_PNC.prc
		// baris 317-373). Semuanya disalin apa adanya dari dokumen polis.
		BusinessCode:         strings.TrimSpace(quoBusinessCode.String),
		BusinessName:         strings.TrimSpace(businessName.String),
		BranchName:           strings.TrimSpace(branchName.String),
		SourceOfBusiness:     strings.TrimSpace(sob.String),
		SourceOfBusinessName: strings.TrimSpace(sobName.String),
		ProdKe:               strings.TrimSpace(prodKe.String),
		PolicyLeader:         strings.TrimSpace(policyLeader.String),
		TypeOfCoins:          strings.TrimSpace(typeOfCoins.String),

		Coinsurance: registrasi.DeriveCoinsurance(strings.TrimSpace(typeOfCoins.String), rows,
			strings.TrimSpace(cedingName.String), fac, facKnown),
	}, nil
}

// coinsurance membaca baris T_COINSLIST polis pada PRODKE-nya. Nol baris adalah keadaan
// biasa: sebagian besar polis tidak berkoasuransi, dan DeriveCoinsurance memberi bawaan Pega.
func (r *PolicyRepo) coinsurance(ctx context.Context, exec executor, number, prodKe string) ([]registrasi.CoinsuranceRow, error) {
	baris, err := exec.QueryContext(ctx, loadQuery("polis_koasuransi"), number, prodKe)
	if err != nil {
		return nil, fmt.Errorf("registrasi/sqlstore: membaca koasuransi polis %q: %w", number, err)
	}
	defer func() { _ = baris.Close() }()

	var out []registrasi.CoinsuranceRow
	for baris.Next() {
		var leader, name, share sql.NullString
		if err := baris.Scan(&leader, &name, &share); err != nil {
			return nil, fmt.Errorf("registrasi/sqlstore: membaca baris koasuransi: %w", err)
		}
		p, ok := parsePercent(share.String)
		out = append(out, registrasi.CoinsuranceRow{
			Leader:       strings.TrimSpace(leader.String),
			CoinsName:    strings.TrimSpace(name.String),
			PercentShare: p,
			HasShare:     ok,
		})
	}
	return out, baris.Err()
}

// parsePercent membaca persentase bertipe teks pada dokumen polis ("60", "33.3333")
// menjadi satuan 1/10000 persen, dibulatkan ke satuan terdekat. Teks yang bukan angka
// dianggap tidak diketahui — bukan nol.
func parsePercent(raw string) (registrasi.Percent, bool) {
	teks := strings.TrimSpace(raw)
	if teks == "" {
		return 0, false
	}
	v, ok := new(big.Rat).SetString(teks)
	if !ok {
		return 0, false
	}
	v.Mul(v, big.NewRat(10_000, 1))
	half := big.NewRat(1, 2)
	if v.Sign() < 0 {
		half.Neg(half)
	}
	v.Add(v, half)
	return registrasi.Percent(new(big.Int).Quo(v.Num(), v.Denom()).Int64()), true
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

// wibDate membaca kolom DATE jam dinding WIB (T_GENERAL.STARTDATE/ENDDATE, mis. 12:00 WIB)
// sebagai instan UTC — setara teks Pega "...T050000.000 GMT" pada dokumen polis. Komponen
// jamnya dibaca apa adanya, tidak bergantung zona waktu sesi driver.
func wibDate(t sql.NullTime) time.Time {
	if !t.Valid || t.Time.IsZero() {
		return time.Time{}
	}
	v := t.Time
	return time.Date(v.Year(), v.Month(), v.Day(), v.Hour(), v.Minute(), v.Second(), 0, clock.ZoneWIB).UTC()
}

// orTime mengembalikan a bila terisi, selain itu b.
func orTime(a, b time.Time) time.Time {
	if !a.IsZero() {
		return a
	}
	return b
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

// Assigner memilih penerima tugas menurut router Pega (`Activity/*Router-act.xml`, diterima
// 2026-10-01):
//
//	PNCAdminRouter    Param.AssignTo = ClaimData.UserAdmin (pembuat klaim, ADMINKLAIM)
//	PNCTeknikRouter   Param.AssignTo = ClaimData.UserTeknis; kosong -> "ServicePNC"
//	RouterRCLDokter   Param.AssignTo = ClaimData.NamaDokterRCL
//
// UserTeknis di Pega diisi lebih awal oleh `getRandomTeam_act`; aturannya ada di
// registrasi.PlanTechnicalPIC dan dijalankan chooseTechnicalPIC — saat Claim Face Sheet
// diunduh, atau saat tahap teknis pertama bila klaim masih belum punya PIC.
type Assigner struct {
	db         *sql.DB
	attendance registrasi.AttendanceSource
	now        func() time.Time
	logger     *slog.Logger
}

// NewAssigner membentuk pemilih penugasan di atas sebuah koneksi.
func NewAssigner(db *sql.DB) *Assigner {
	return &Assigner{db: db, now: time.Now, logger: slog.Default()}
}

// WithAttendance memasang sumber absensi PIC (ServiceGetDataAbsenPIC). Tanpa sumber, setiap
// kandidat dianggap hadir — sama dengan Pega saat layanannya tidak menjawab.
func (a *Assigner) WithAttendance(src registrasi.AttendanceSource, logger *slog.Logger) *Assigner {
	a.attendance = src
	if logger != nil {
		a.logger = logger
	}
	return a
}

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

	// ToCurrentOperator menugaskan ke PEMANGGIL — itu arti namanya, dan pengisi memori
	// sudah melakukannya. Tanpa cabang ini, tahap View Polis jatuh ke kueri beban PIC
	// Teknik dan berakhir di tangan orang lain.
	if stage.Router == registrasi.RouterCurrentOperator {
		if operator := strings.TrimSpace(caller); operator != "" {
			return registrasi.Assignee{Operator: operator}, nil
		}
		return registrasi.Assignee{}, fmt.Errorf(
			"registrasi/sqlstore: tahap %q menuntut pemanggil, tetapi pemanggilnya kosong", stage.ID)
	}

	// PNCAdminRouter menugaskan ke ADMIN KLAIMNYA, bukan ke petugas teknis.
	//
	// Rule-nya tidak ada di export (`R-04`), tetapi dua hal menunjukkan perilakunya.
	// Pertama, `Activity/CreateRegisterKlaimPNC_act.xml` langkah 14 mengisi
	// `ClaimData.UserAdmin` dengan `OperatorID.pyUserIdentifier` — petugas yang menekan
	// tombolnya. Kedua, kembarannya yang ADA di export, `PNCAdminRouterRCV`, menugaskan
	// ke `ReceiveDocument.UserAdmin` dan jatuh ke pembuat kasus bila kosong.
	//
	// Memakai kueri beban PIC Teknik untuk tahap ini akan melempar klaim ke orang lain
	// tepat setelah petugas menekan Register Klaim — dan petugas itu lalu tidak dapat
	// membuka klaim yang baru saja dibuatnya sendiri.
	//
	// ADMINKLAIM di basis data adalah kolom yang sama dengan CreatedBy di sini.
	// RouterRCLDokter menugaskan ke ClaimData.NamaDokterRCL. Tidak ada rule di export yang
	// mengisinya (hanya laporan yang membacanya; Pega menyimpannya di
	// T_CLAIMLIST_ADMIN.NAMADOKTERRCL_1), dan layar pemilihan dokter RCL belum ada di aplikasi
	// ini. Sampai itu ada, tugas diparkir di ServicePNC — antrean "belum ditugaskan" — bukan
	// diberikan ke PIC Teknik yang bukan dokter.
	if stage.Router == registrasi.RouterRCLDoctor {
		return registrasi.Assignee{Operator: registrasi.OperatorUnassigned}, nil
	}

	if stage.Router == registrasi.RouterPNCAdmin {
		admin := strings.TrimSpace(claim.CreatedBy)
		if admin == "" {
			admin = strings.TrimSpace(caller)
		}
		if admin == "" {
			return registrasi.Assignee{}, fmt.Errorf(
				"registrasi/sqlstore: tahap %q tidak punya admin klaim maupun pemanggil", stage.ID)
		}
		return registrasi.Assignee{Operator: admin}, nil
	}

	// PIC Teknik yang sudah tercatat di klaim dihormati (registrasi.AssignedTechnicalPIC).
	// Bebannya tidak dinaikkan di sini: ia tidak dipilih oleh kueri beban.
	if pic := registrasi.AssignedTechnicalPIC(stage, claim); pic != "" {
		return registrasi.Assignee{Operator: pic}, nil
	}

	operator, err := a.chooseTechnicalPIC(ctx, claim, caller)
	if err != nil {
		return registrasi.Assignee{}, err
	}
	if operator == "" {
		// Tidak ada kandidat: PNCTeknikRouter langkah 2 — UserTeknis kosong — menugaskan ke
		// ServicePNC, antrean "belum ditugaskan" yang dibagikan ulang agent
		// TransferAllCaseNotAssigned. Tugas tidak dibuang dan tidak jatuh ke pemanggil.
		return registrasi.Assignee{Operator: registrasi.OperatorUnassigned}, nil
	}
	return registrasi.Assignee{Operator: operator}, nil
}

// chooseTechnicalPIC menjalankan jalur registrasi.PlanTechnicalPIC terhadap
// POOLDATA.MST_USER_TEKNIK. Kosong berarti tidak ada kandidat.
//
// # Beban petugas
//
// Mengikuti Pega apa adanya. Prosedur PA/Travel menaikkan MST_USER_TEKNIK.COUNTER_QUOTA.
// Jalur NONMBU TIDAK menaikkan beban apa pun saat memilih; bebannya dinaikkan
// `AddTJobCQuota_SQL` saat Claim Face Sheet pertama — di MST_USER_TEKNIS, bukan di
// MST_USER_TEKNIK yang dipakai mengurutkan kandidat (lihat usecase/facesheet.go).
func (a *Assigner) chooseTechnicalPIC(ctx context.Context, claim registrasi.Claim, caller string) (string, error) {
	estimate, err := a.estimateIDR(ctx, claim)
	if err != nil {
		return "", err
	}
	plan := registrasi.PlanTechnicalPIC(claim, estimate)
	if plan.Operator != "" {
		return plan.Operator, nil
	}

	exec := executorFrom(ctx, a.db)
	var operator string
	switch plan.Pool {
	case registrasi.PoolNone:
		return "", nil
	case registrasi.PoolProcedure:
		operator, err = a.procedurePIC(ctx, exec, plan, caller)
	case registrasi.PoolNonMBU:
		candidates := plan.Preferred
		if len(candidates) == 0 {
			candidates, err = a.nonMBUCandidates(ctx, exec, plan)
		}
		if err == nil {
			operator = a.firstPresent(ctx, claim, candidates)
		}
	}
	if err != nil {
		return "", fmt.Errorf("registrasi/sqlstore: memilih petugas teknis: %w", err)
	}
	if operator == "" {
		return "", nil
	}

	// Hanya cabang prosedur yang menaikkan beban MST_USER_TEKNIK (UPDATE di akhir
	// GETDATA_PICTEKNIK). Jalur NONMBU tidak: di Pega step 15.8 keluar ke EX sebelum step
	// 15.9–15.10, sehingga pencacahnya tidak pernah jalan.
	if plan.Pool == registrasi.PoolProcedure {
		if _, err := exec.ExecContext(ctx, loadQuery("pic_teknik_naikkan_beban"), operator); err != nil {
			return "", fmt.Errorf("registrasi/sqlstore: menaikkan beban petugas %q: %w", operator, err)
		}
	}
	return strings.TrimSpace(operator), nil
}

// procedurePIC adalah `POOLDATA.GETDATA_PICTEKNIK(txtBusinessType, txtTKI)` dengan
// txtBusinessType = jabatan operator yang sedang bekerja (`getRandomTeam_act` step 19/24).
func (a *Assigner) procedurePIC(ctx context.Context, exec executor, plan registrasi.TechnicalPICPlan, caller string) (string, error) {
	var position sql.NullString
	err := exec.QueryRowContext(ctx, loadQuery("pic_teknik_jabatan"), strings.ToUpper(strings.TrimSpace(caller))).Scan(&position)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("membaca jabatan operator %q: %w", caller, err)
	}

	var operator string
	if strings.TrimSpace(position.String) == registrasi.PositionPA {
		err = exec.QueryRowContext(ctx, loadQuery("pic_teknik_pa"), registrasi.PATechnicalPIC(plan.TKI)).Scan(&operator)
	} else {
		err = exec.QueryRowContext(ctx, loadQuery("pic_teknik_travel")).Scan(&operator)
	}
	if errors.Is(err, sql.ErrNoRows) {
		// Prosedur mengisi errmsg dan keluar tanpa PIC.
		return "", nil
	}
	return operator, err
}

// nonMBUCandidates adalah seluruh daftar `BrowsePICRandomTeam` / `BrowsePICRandomTeam2`,
// berurutan menurut beban.
func (a *Assigner) nonMBUCandidates(ctx context.Context, exec executor, plan registrasi.TechnicalPICPlan) ([]string, error) {
	query := "pic_teknik_nonmbu"
	if plan.Large {
		query = "pic_teknik_nonmbu_besar"
	}
	rows, err := exec.QueryContext(ctx, loadQuery(query), registrasi.ExcludedTechnicalPIC, yesNo(plan.TeamC))
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// firstPresent adalah loop `getRandomTeam_act` step 15 atas daftar kandidat:
//
//   - akhir pekan atau hari libur (jawaban absensi kandidat yang sedang diperiksa) →
//     pemilihan berhenti TANPA PIC (step 15.7);
//   - `RuleTimeIn` memuat "000000" → kandidat dilewati (step 15.8);
//   - selain itu kandidat itu yang dipilih.
//
// Kegagalan layanan absensi diperlakukan seperti Pega: jawabannya kosong, sehingga kandidat
// itu dipilih. Kegagalannya dicatat, bukan ditelan diam-diam.
func (a *Assigner) firstPresent(ctx context.Context, claim registrasi.Claim, candidates []string) string {
	date := a.now().In(clock.ZoneWIB)
	check := a.attendance != nil && registrasi.AttendanceApplies(claim)
	for _, candidate := range candidates {
		id := strings.TrimSpace(candidate)
		if id == "" {
			continue
		}
		if !check {
			return id
		}
		att, err := a.attendance.Attendance(ctx, id, date, claim.Number)
		if err != nil {
			a.logger.WarnContext(ctx, "absensi PIC tidak terbaca; kandidat dianggap hadir seperti Pega",
				slog.String("pic", id), slog.String("nomor_klaim", claim.Number), slog.String("galat", err.Error()))
			return id
		}
		if att.Closed() {
			return ""
		}
		if att.Absent() {
			continue
		}
		return id
	}
	return ""
}

// estimateIDR adalah `ClaimEstimate × DollarCurrencyVal` — estimasi klaim dalam rupiah,
// dengan kurs tanggal kejadian (`D-48`), sama seperti registrasi menghitung ambang Large
// Losses. Klaim tanpa mata uang dianggap rupiah.
func (a *Assigner) estimateIDR(ctx context.Context, claim registrasi.Claim) (registrasi.Money, error) {
	currency := strings.TrimSpace(claim.Currency)
	if currency == "" {
		return claim.EstimateValue, nil
	}
	rate, err := NewExchangeRateSource(a.db).Find(ctx, currency, claim.DateOfLoss)
	if err != nil {
		return 0, fmt.Errorf("registrasi/sqlstore: kurs estimasi untuk memilih PIC Teknik: %w", err)
	}
	return claim.EstimateValue.Convert(rate), nil
}

var (
	_ registrasi.Parameter = (*Parameter)(nil)
	_ registrasi.Assigner  = (*Assigner)(nil)
)
