package memory

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"claim-pnc/internal/platform/clock"
	"claim-pnc/internal/registrasi"
)

// ── Polis ────────────────────────────────────────────────────────────────────────

// PolicyStore melayani snapshot polis dari daftar di memori.
//
// Sumber sebenarnya adalah GISFW (`ADR-0006`) lewat modul `B-1`, yang belum ada. Daftar
// contoh di bawah memuat satu polis per lini bisnis yang mempengaruhi alur, supaya
// keempat cabang `IsPA` / `IsTravel` / NonMBU dapat ditelusuri tanpa basis data.
type PolicyStore struct {
	list map[string]registrasi.Policy
}

// NewPolicyStore membentuk repo berisi daftar yang diberikan.
func NewPolicyStore(policy ...registrasi.Policy) *PolicyStore {
	list := make(map[string]registrasi.Policy, len(policy))
	for _, p := range policy {
		list[strings.ToUpper(strings.TrimSpace(p.Number))] = p
	}
	return &PolicyStore{list: list}
}

// Get mengembalikan satu polis.
func (r *PolicyStore) Get(_ context.Context, number string) (registrasi.Policy, error) {
	p, ok := r.list[strings.ToUpper(strings.TrimSpace(number))]
	if !ok {
		return registrasi.Policy{}, fmt.Errorf("%w: %s", registrasi.ErrPolicyNotFound, number)
	}
	return p, nil
}

// SamplePolicies adalah polis karangan untuk pengembangan tanpa basis data.
//
// Nomor polis, nama tertanggung, dan tanggalnya KARANGAN — bukan data nasabah.
func SamplePolicies(now time.Time) []registrasi.Policy {
	start := clock.AddDays(now, -180)
	end := clock.AddDays(now, 185)

	base := func(number string, line registrasi.LineOfBusiness, kind, insured string) registrasi.Policy {
		return registrasi.Policy{
			Number:        number,
			Line:          line,
			BusinessType:  kind,
			CoverageStart: start,
			CoverageEnd:   end,
			Currency:      "IDR",
			InsuredName:   insured,
			BranchCode:    "100081",
		}
	}

	spk := base("POL-SPK-0004", registrasi.LineMiscellaneous, "SPK", "PT Contoh Kredit Sejahtera")
	spk.CreditGuarantee = true

	return []registrasi.Policy{
		base("POL-FIRE-0001", registrasi.LineFire, "Fire", "PT Contoh Industri Nusantara"),
		base("POL-PA-0002", registrasi.LinePersonalAccident, "PersonalAccident", "Contoh Karyawan Bersama"),
		base("POL-TRV-0003", registrasi.LineTravel, "Travel", "Contoh Wisata Mandiri"),
		spk,
		base("POL-MAR-0005", registrasi.LineMarineCargo, "MarineCargo", "PT Contoh Logistik Samudra"),
	}
}

// ── Nomor klaim ──────────────────────────────────────────────────────────────────

// NumberIssuer menerbitkan nomor klaim berformat `PNCN.YY.xxxx` (`D-71`, `ADR-0009`).
//
// # Dua hal yang masih menunggu keputusan
//
// `TKT-F2-006` mencatat dua pertanyaan yang belum dijawab Work Owner: apakah urutan
// direset tiap awal tahun, dan apakah lebar segmen terakhir dibuat tetap. Yang dipakai
// di sini adalah tafsir yang paling langsung dari formatnya sendiri:
//
//   - Urutan DIRESET tiap tahun. Membawa segmen `YY` di dalam nomor tanpa mereset
//     urutannya membuat segmen itu tidak berguna sebagai pembeda.
//   - Lebar minimum EMPAT digit, dan TUMBUH bila terlampaui. Memotong pada empat digit
//     berarti nomor ke-10.001 menabrak nomor yang sudah terbit — kegagalan yang tidak
//     dapat diperbaiki setelah surat terkirim.
//
// Keduanya asumsi yang dicatat, bukan keputusan yang diambil diam-diam.
type NumberIssuer struct {
	mu     sync.Mutex
	seq    map[int]int
	prefix string
}

// NewNumberIssuer membentuk penerbit kosong.
func NewNumberIssuer() *NumberIssuer {
	return &NumberIssuer{seq: map[int]int{}, prefix: "PNCN"}
}

// Issue mengembalikan nomor klaim berikutnya untuk tahun yang berlaku.
//
// Tahun diambil menurut WIB, bukan UTC: klaim yang didaftarkan pukul 06.00 WIB tanggal
// 1 Januari masih 31 Desember di UTC, dan nomornya harus membawa tahun yang dilihat
// petugas.
func (p *NumberIssuer) Issue(_ context.Context, at time.Time) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	year := clock.DateWIB(at).Year()
	p.seq[year]++
	return fmt.Sprintf("%s.%02d.%04d", p.prefix, year%100, p.seq[year]), nil
}

// ── Parameter bisnis ─────────────────────────────────────────────────────────────

// Parameter melayani nilai yang di sistem lama tertanam di dalam rule.
//
// Nilainya berada DI SINI, di luar lapisan aturan, dan bukan di dalam paket registrasi —
// sehingga pemindaian `grep` atas alamat surel di modul domain benar-benar menghasilkan
// nol baris, sesuai kriteria penerimaan `TKT-B02-004`.
//
// # Yang sengaja TIDAK ditiru
//
// Empat blok bertanda `// TESTING` pada `InputRegister_act` langkah 53.5–53.8 menimpa
// penerima produksi dengan alamat penguji, memakai prasyarat yang IDENTIK dengan blok
// produksi di atasnya. `TKT-B02-004` menuntut nol jalur semacam itu, dan tidak ada
// padanannya di sini.
//
// # Yang sengaja TIDAK dibawa
//
// Tujuh alamat surel pimpinan pada langkah 53.1 dan sedikitnya enam akun Gmail pribadi
// di jalur produksi. `D-67` menetapkan penerima pemberitahuan wajib mailbox fungsional.
// Daftar bawaan di bawah memakai alamat fungsional karangan, dan pengisian nyatanya
// menunggu master `TKT-F4-003`.
type Parameter struct {
	mu         sync.Mutex
	threshold  registrasi.Money
	recipients map[registrasi.LineOfBusiness][]string
	general    []string
}

// NewParameter membentuk parameter dengan nilai bawaan.
//
// Ambang bawaan Rp 1.000.000.000 adalah nilai yang di sistem lama di-hardcode sebagai
// `1000000000` pada `InputRegister_act` langkah 53.
func NewParameter() *Parameter {
	return &Parameter{
		threshold: registrasi.Rupiah(1_000_000_000),
		recipients: map[registrasi.LineOfBusiness][]string{
			registrasi.LineFire:          {"uw.fire@contoh.internal"},
			registrasi.LineMiscellaneous: {"uw.aneka@contoh.internal"},
			registrasi.LineMarineCargo:   {"uw.marine@contoh.internal"},
		},
		general: []string{"notifikasi.klaim@contoh.internal"},
	}
}

// SetThreshold mengubah ambang Notice of Large Losses tanpa deployment — perilaku yang
// justru dikejar `TKT-B02-004`.
func (p *Parameter) SetThreshold(u registrasi.Money) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.threshold = u
}

// SetGeneral mengubah penerima yang berlaku untuk seluruh lini.
//
// Dipakai pengujian untuk membuat master BENAR-BENAR kosong. Itu keadaan yang nyata:
// `PNC.PENERIMA_KERUGIAN_BESAR` belum diisi di lingkungan mana pun, dan yang harus
// dibuktikan adalah pendaftaran klaim tetap berjalan.
func (p *Parameter) SetGeneral(address []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.general = append([]string(nil), address...)
}

// SetRecipients mengubah penerima pemberitahuan untuk satu lini.
func (p *Parameter) SetRecipients(line registrasi.LineOfBusiness, address []string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.recipients[line] = append([]string(nil), address...)
}

// LargeLossThreshold mengembalikan ambang yang berlaku.
func (p *Parameter) LargeLossThreshold(_ context.Context) (registrasi.Money, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.threshold, nil
}

// LargeLossRecipients mengembalikan penerima untuk satu lini, selalu digabung dengan
// penerima umum.
func (p *Parameter) LargeLossRecipients(_ context.Context, line registrasi.LineOfBusiness) ([]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := append([]string(nil), p.general...)
	result = append(result, p.recipients[line]...)
	sort.Strings(result)
	return result, nil
}

// ── Kurs ─────────────────────────────────────────────────────────────────────────

// ExchangeRateSource melayani kurs dari daftar di memori.
//
// `ADR-0015` menetapkan kurs yang dipakai adalah kurs TANGGAL KEJADIAN, dan kurs yang
// tidak ditemukan MENOLAK klaim. Karena itu tidak ada nilai bawaan di sini: mata uang
// yang tidak terdaftar menghasilkan ErrExchangeRateNotFound, bukan angka 1.
//
// Rupiah adalah pengecualian yang sah — kurs rupiah terhadap rupiah adalah 1,0000 secara
// definisi, bukan karena tidak ditemukan.
type ExchangeRateSource struct {
	mu    sync.Mutex
	value map[string]registrasi.ExchangeRate
}

// NewExchangeRateSource membentuk sumber kurs dengan rupiah sudah terisi.
func NewExchangeRateSource() *ExchangeRateSource {
	return &ExchangeRateSource{value: map[string]registrasi.ExchangeRate{"IDR": registrasi.ExchangeRateOne}}
}

// Set menetapkan kurs sebuah mata uang.
func (s *ExchangeRateSource) Set(currency string, k registrasi.ExchangeRate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.value[strings.ToUpper(strings.TrimSpace(currency))] = k
}

// Find mengembalikan kurs sebuah mata uang pada sebuah tanggal.
//
// Daftar di memori belum membedakan tanggal; seam-nya sudah membawa tanggal supaya
// adapter SQL dapat memenuhinya tanpa mengubah satu pun pemanggil.
func (s *ExchangeRateSource) Find(_ context.Context, currency string, _ time.Time) (registrasi.ExchangeRate, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	k, ok := s.value[strings.ToUpper(strings.TrimSpace(currency))]
	if !ok {
		return 0, fmt.Errorf("%w: mata uang %q", registrasi.ErrExchangeRateNotFound, currency)
	}
	return k, nil
}

// ── Penugasan ────────────────────────────────────────────────────────────────────

// Assigner membagi tugas ke petugas dengan beban paling sedikit.
//
// # Algoritmanya bukan tebakan
//
// Tiga aturan routing yang dipakai alur Register tidak ada di export (`R-04`), tetapi
// algoritmanya terbaca dari tempat lain: `RDB List/BrowsePICRandomTeam-SQL.xml:39-40`
// mengurutkan calon petugas dengan `ORDER BY counter_quota ASC`, dan
// `AddTJobCounterPIC_SQL` menaikkan pencacahnya setelah tugas diberikan. Yang belum
// diketahui adalah SIAPA saja yang menjadi calon pada tiap aturan — dan itu data, bukan
// algoritma.
//
// # Satu hal yang sengaja tidak dibawa
//
// Kueri yang sama memuat `operator_ID != 'ELLENSUPRIYATI'` — satu nama pegawai yang
// dikecualikan, tertanam di dalam SQL. `TKT-B06-002` mendaftarkannya sebagai keputusan
// Work Owner yang belum diambil. Ia tidak ditiru di sini; pengecualian semacam itu
// menjadi data bila kelak diputuskan tetap ada.
type Assigner struct {
	mu    sync.Mutex
	teams map[string][]string
	load  map[string]int
}

// NewAssigner membentuk penugasan dengan tim per aturan routing.
func NewAssigner(teams map[string][]string) *Assigner {
	copy := make(map[string][]string, len(teams))
	for k, v := range teams {
		copy[k] = append([]string(nil), v...)
	}
	return &Assigner{teams: copy, load: map[string]int{}}
}

// SampleTeams adalah petugas karangan per aturan routing, untuk pengembangan tanpa basis
// data. Nama-namanya bukan pegawai nyata.
func SampleTeams() map[string][]string {
	return map[string][]string{
		registrasi.RouterPNCAdmin:     {"ADMINPNC01", "ADMINPNC02", "ADMINPNC03"},
		registrasi.RouterPNCTechnical: {"TEKNIK01", "TEKNIK02"},
		registrasi.RouterRCLDoctor:    {"RCLDOKTER01"},
	}
}

// Assign memilih penerima tugas untuk sebuah tahap.
func (p *Assigner) Assign(_ context.Context, stage registrasi.Stage, _ registrasi.Claim, caller string) (registrasi.Assignee, error) {
	// Tahap Workbasket tidak memilih orang sama sekali — itu yang membuatnya antrean
	// bersama.
	if stage.Queue == registrasi.QueueWorkbasket {
		return registrasi.Assignee{Workbasket: stage.Workbasket}, nil
	}

	// Tahap yang merutekan ke ORANG BERNAMA. Satu-satunya yang begitu adalah Analyst
	// Doctor; lihat catatan pada OperatorAnalystDoctor.
	if stage.Operator != "" {
		return registrasi.Assignee{Operator: stage.Operator}, nil
	}

	if stage.Router == registrasi.RouterCurrentOperator {
		return registrasi.Assignee{Operator: caller}, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	candidates := p.teams[stage.Router]
	if len(candidates) == 0 {
		// Aturan routing yang belum punya daftar petugas mengembalikan tugas kepada
		// caller, bukan membuang tugasnya. Klaim yang tidak punya penerima adalah
		// klaim yang mandek tanpa ada yang tahu — kelas cacat yang `ADR-0021` sebut
		// paling mahal ditemukan setelah rilis.
		return registrasi.Assignee{Operator: caller}, nil
	}

	chosen := candidates[0]
	for _, c := range candidates {
		if p.load[c] < p.load[chosen] {
			chosen = c
		}
	}
	p.load[chosen]++
	return registrasi.Assignee{Operator: chosen}, nil
}

// Load mengembalikan pencacah beban tiap petugas. Dipakai pengujian.
func (p *Assigner) Load() map[string]int {
	p.mu.Lock()
	defer p.mu.Unlock()
	copy := make(map[string]int, len(p.load))
	for k, v := range p.load {
		copy[k] = v
	}
	return copy
}

// ── Pengenal internal ────────────────────────────────────────────────────────────

// IDGenerator membangkitkan pengenal internal acak.
type IDGenerator struct{}

// New mengembalikan pengenal 128 bit dalam heksadesimal.
//
// Ia acak, bukan berurut: pengenal internal tidak boleh membocorkan berapa banyak klaim
// yang sudah dibuat, dan tidak boleh dapat ditebak dari pengenal lain.
func (IDGenerator) New() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand tidak gagal pada sistem yang sehat. Bila ia gagal, melanjutkan
		// dengan pengenal yang dapat ditebak lebih berbahaya daripada berhenti.
		panic("registrasi/memori: sumber acak tidak tersedia: " + err.Error())
	}
	return hex.EncodeToString(b[:])
}

func sameText(a, b string) bool {
	return strings.EqualFold(strings.Join(strings.Fields(a), " "), strings.Join(strings.Fields(b), " "))
}

var (
	_ registrasi.PolicyRepo         = (*PolicyStore)(nil)
	_ registrasi.NumberIssuer       = (*NumberIssuer)(nil)
	_ registrasi.Parameter          = (*Parameter)(nil)
	_ registrasi.ExchangeRateSource = (*ExchangeRateSource)(nil)
	_ registrasi.Assigner           = (*Assigner)(nil)
	_ registrasi.IDGenerator        = IDGenerator{}
)

// ── Tautan ke berkas laporan ─────────────────────────────────────────────────────

// ClaimReportLink merekam tautan klaim ke berkas laporan di memori.
//
// Ia MEREKAM, bukan sekadar mengabaikan: pengujian perlu membuktikan tautannya benar
// terjadi, dan pada urutan yang benar. Tanpa rekaman, "berkas tidak berpindah tab" —
// keluhan yang melahirkan seam ini — tidak dapat dijaga oleh satu uji pun.
type ClaimReportLink struct {
	mu          sync.Mutex
	handedOver  map[string]time.Time
	claimNumber map[string]string
}

// NewClaimReportLink membentuk penaut kosong.
func NewClaimReportLink() *ClaimReportLink {
	return &ClaimReportLink{
		handedOver:  map[string]time.Time{},
		claimNumber: map[string]string{},
	}
}

// MarkHandedOver menandai berkas sudah diserahkan; pemanggilan ulang tidak menggeser
// waktunya, sama seperti pengisi SQL.
func (p *ClaimReportLink) MarkHandedOver(_ context.Context, reportID string, at time.Time) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, sudah := p.handedOver[reportID]; !sudah {
		p.handedOver[reportID] = at
	}
	return nil
}

// AttachClaimNumber memasang nomor klaim, dan MENOLAK bila berkasnya belum diserahkan —
// urutan yang sama dengan pengisi SQL, supaya uji yang lulus di sini tidak gagal di sana.
func (p *ClaimReportLink) AttachClaimNumber(_ context.Context, reportID, claimNumber string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, sudah := p.handedOver[reportID]; !sudah {
		return fmt.Errorf("registrasi/memori: laporan %s belum ditandai diserahkan", reportID)
	}
	p.claimNumber[reportID] = claimNumber
	return nil
}

// HandedOver menyebut apakah sebuah berkas sudah ditandai diserahkan.
func (p *ClaimReportLink) HandedOver(reportID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	_, ok := p.handedOver[reportID]
	return ok
}

// ClaimNumber menyebut nomor klaim yang terpasang pada sebuah berkas.
func (p *ClaimReportLink) ClaimNumber(reportID string) string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.claimNumber[reportID]
}

var _ registrasi.ClaimReportLink = (*ClaimReportLink)(nil)
