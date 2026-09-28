package inboxmanager

import "strings"

// Verdict adalah keputusan yang dikirim penyelia atas sebuah baris antrean.
type Verdict string

const (
	// VerdictApprove menyetujui baris.
	VerdictApprove Verdict = "setujui"

	// VerdictReject menolak baris.
	VerdictReject Verdict = "tolak"
)

// Nilai kolom persetujuan di basis data.
//
// # Ketiganya terbukti, bukan ditebak
//
// Dua sumber yang berdiri sendiri menghasilkan nilai yang sama:
//
//   - Export rule. `Activity/UpdateStatusAksepRangka_HE-Act.xml:34214` menetapkan
//     `TempRangka.TSI := "1"` sebelum memanggil `SetStatusAksepNoRangka`, dan
//     `Activity/UpdateStatusRejectRangka_HE-Act.xml:26885` menetapkan `"2"` pada jalur
//     penolakan. Keenam antrean master memakai pasangan yang sama, dan enam modul Master
//     yang sudah dibangun mencatatnya sebagai StatusPending/Approved/Rejected = "0"/"1"/"2".
//
//   - Isi basis data. Saat diperiksa 2026-09-28: `NOTIF_RANGKA_HE.STS_AKSEP` berisi
//     `1` (10 baris), `2` (5), NULL (2); `T_CLAIM_AKSEPTASI_CHECKER.STSAPP` berisi
//     `1` (42), `2` (33), `0` (8).
//
// Kedua sumber itu penting bersama-sama: yang pertama menyatakan apa yang DITULIS sistem
// lama, yang kedua membuktikan nilai itu memang yang ada di kolomnya hari ini.
const (
	// StatusPending adalah nilai baris yang belum diputuskan.
	//
	// Ia dipakai sebagai PENJAGA pada setiap pernyataan keputusan, bukan sekadar sebagai
	// penyaring daftar. Lihat catatan di kepala repo/sqlstore/inboxmanager.sql.
	StatusPending = "0"

	// StatusApproved adalah nilai baris yang disetujui.
	StatusApproved = "1"

	// StatusRejected adalah nilai baris yang ditolak.
	StatusRejected = "2"
)

// Value mengembalikan nilai kolom yang ditulis sebuah keputusan.
func (v Verdict) Value() string {
	if v == VerdictApprove {
		return StatusApproved
	}
	return StatusRejected
}

// Valid menyatakan apakah keputusannya dikenal.
func (v Verdict) Valid() bool {
	return v == VerdictApprove || v == VerdictReject
}

// DecisionRule menyatakan apa yang boleh dilakukan sebuah antrean, dan apa yang tidak.
//
// Ia DATA, bukan percabangan yang tersebar di usecase dan di layar. Dengan begitu layar dapat
// menggambar tombol yang benar tanpa mengetahui satu pun nama tabel, dan penegakannya tetap
// terjadi di server.
type DecisionRule struct {
	// Decidable menyatakan apakah antrean ini dapat diputuskan sama sekali.
	Decidable bool

	// ApproveBlockedReason menjelaskan kenapa jalur SETUJU ditahan, bila ditahan.
	//
	// Kosong berarti tidak ditahan. Ia dipisah dari Decidable karena satu antrean dapat
	// menolak dengan benar sementara jalur setujunya belum dapat diselesaikan — dan itu
	// bukan keadaan hipotetis, lihat tab Payment Klaim Akseptasi.
	ApproveBlockedReason string

	// ReasonRequiredOnReject menuntut alasan diisi saat menolak.
	//
	// Ia hanya dinyalakan pada antrean yang tabelnya PUNYA kolom alasan. Pada antrean yang
	// tidak punya, isian alasannya tidak digambar sama sekali — bukan digambar lalu isinya
	// dibuang.
	ReasonRequiredOnReject bool

	// ReasonLabel adalah judul isian alasan yang dibaca pengguna.
	ReasonLabel string
}

// Decision adalah permintaan keputusan yang sudah tervalidasi.
//
// Ia hanya lahir lewat NewDecision, dan di situlah kewenangan tab, kesahihan keputusan, dan
// kelengkapan alasan diperiksa.
type Decision struct {
	// Tab adalah antrean yang sedang diputuskan.
	Tab Tab

	// Verdict adalah keputusannya.
	Verdict Verdict

	// Keys adalah kunci baris yang diputuskan, sudah dibersihkan dan tanpa duplikat.
	Keys []string

	// Reason adalah alasan yang diketik penyelia. Kosong bila antreannya tidak punya kolom
	// alasan.
	Reason string

	// Caller adalah identitas penyelia, dicatat pada baris yang diubah bila tabelnya punya
	// kolom untuk itu.
	Caller Caller
}

// DecisionResult adalah hasil satu permintaan keputusan.
type DecisionResult struct {
	// Requested adalah jumlah kunci yang dikirim.
	Requested int

	// Changed adalah jumlah baris yang BENAR-BENAR berubah.
	Changed int
}

// Stale adalah jumlah baris yang tidak berubah karena sudah diputuskan lebih dulu oleh orang
// lain.
//
// # Kenapa angka ini wajib sampai ke layar
//
// Karena setiap pernyataan keputusan ikut menyaring status menunggu di samping kuncinya —
// `AND <kolom> = '0'`. Tanpa penyaring itu, penyelia yang membuka daftar lama akan MENIMPA
// keputusan orang lain, dan jumlah baris yang berubah akan tetap sama dengan jumlah yang
// dipilih sehingga tidak ada yang tahu.
//
// Dengan penyaring itu, selisihnya menjadi terlihat — dan layar menyatakannya apa adanya
// alih-alih melaporkan keberhasilan penuh.
func (r DecisionResult) Stale() int {
	if r.Requested <= r.Changed {
		return 0
	}
	return r.Requested - r.Changed
}

// MaxDecisionKeys adalah batas jumlah baris yang boleh diputuskan dalam satu permintaan.
//
// Ia ada karena pernyataan keputusan dijalankan SATU BARIS PER PERNYATAAN di dalam satu
// transaksi — bentuk yang dipilih supaya teks SQL tetap tetap dan dapat dibaca utuh di berkas
// `.sql` (`08-TECHNICAL-STRATEGY.md` §4.3). Biayanya satu perjalanan per baris, dan batas ini
// yang menjaga biaya itu tetap wajar.
//
// Seratus jauh di atas kebutuhan nyata: saat diperiksa 2026-09-28 antrean terpanjang berisi
// 11 baris.
const MaxDecisionKeys = 100

// NewDecision membentuk permintaan keputusan yang sah, atau menyatakan apa yang salah.
//
// # Urutan pemeriksaannya disengaja
//
// Identitas, lalu kewenangan tab, lalu apakah antreannya dapat diputuskan, baru isi
// permintaannya. Dengan urutan itu penyelia yang salah alamat memperoleh jawaban yang
// menjelaskan keadaannya — bukan "alasan wajib diisi" untuk antrean yang sebenarnya memang
// tidak boleh ia sentuh.
func NewDecision(
	tabCode string,
	verdict Verdict,
	keys []string,
	reason string,
	caller Caller,
) (Decision, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Decision{}, ErrCallerUnknown
	}

	tab, known := FindTab(strings.TrimSpace(tabCode))
	if !known {
		return Decision{}, NewValidationError([]Violation{{
			Field:   FieldTab,
			Message: "Tab tidak dikenal.",
		}})
	}
	if !CanSee(cleanCaller, tab) {
		return Decision{}, ErrTabNotAllowed
	}

	rule := tab.Decision
	if tab.Kind != KindQueue || !rule.Decidable {
		return Decision{}, ErrQueueNotDecidable
	}
	if verdict == VerdictApprove && rule.ApproveBlockedReason != "" {
		return Decision{}, ErrApproveBlocked
	}

	violations := []Violation{}

	if !verdict.Valid() {
		violations = append(violations, Violation{
			Field:   FieldVerdict,
			Message: "Keputusan hanya boleh \"setujui\" atau \"tolak\".",
		})
	}

	cleanKeys := dedupe(keys)
	switch {
	case len(cleanKeys) == 0:
		violations = append(violations, Violation{
			Field:   FieldKeys,
			Message: "Pilih dulu baris yang hendak diputuskan.",
		})
	case len(cleanKeys) > MaxDecisionKeys:
		violations = append(violations, Violation{
			Field: FieldKeys,
			Message: "Terlalu banyak baris dalam satu permintaan. " +
				"Putuskan paling banyak 100 baris sekaligus.",
		})
	}

	cleanReason := strings.TrimSpace(reason)

	// Alasan dituntut HANYA pada antrean yang tabelnya punya kolomnya, dan HANYA pada
	// penolakan. Ini LEBIH KETAT daripada Pega, yang menerima penolakan tanpa alasan apa
	// pun — dan itu disengaja: kolom itu satu-satunya hal yang memberi tahu pengaju kenapa
	// barisnya ditolak, dan baris yang ditolak tanpa keterangan akan diajukan ulang apa
	// adanya.
	if rule.ReasonRequiredOnReject && verdict == VerdictReject && cleanReason == "" {
		violations = append(violations, Violation{
			Field:   FieldReason,
			Message: "Alasan penolakan wajib diisi.",
		})
	}

	// Alasan yang dikirim untuk antrean yang tidak punya kolomnya DIBUANG, bukan ditolak:
	// layar tidak menggambar isiannya, sehingga nilai yang sampai ke sini hanya dapat
	// berasal dari permintaan yang disusun tangan. Menolaknya tidak melindungi apa pun.
	if !rule.ReasonRequiredOnReject && rule.ReasonLabel == "" {
		cleanReason = ""
	}

	if len(violations) > 0 {
		return Decision{}, NewValidationError(violations)
	}

	return Decision{
		Tab:     tab,
		Verdict: verdict,
		Keys:    cleanKeys,
		Reason:  cleanReason,
		Caller:  cleanCaller,
	}, nil
}

// dedupe memangkas spasi, membuang yang kosong, dan membuang kunci ganda dengan tetap
// mempertahankan urutan kirimnya.
//
// # Kenapa kunci ganda dibuang, bukan dibiarkan
//
// Karena jumlah baris yang berubah dibandingkan dengan jumlah kunci yang dikirim untuk
// menghitung DecisionResult.Stale. Kunci yang dikirim dua kali akan berubah sekali, dan
// selisihnya akan terbaca sebagai "sudah diputuskan orang lain" — laporan yang keliru tentang
// hal yang justru paling perlu dipercaya.
func dedupe(keys []string) []string {
	seen := map[string]struct{}{}
	result := []string{}

	for _, key := range keys {
		clean := strings.TrimSpace(key)
		if clean == "" {
			continue
		}
		if _, repeat := seen[clean]; repeat {
			continue
		}
		seen[clean] = struct{}{}
		result = append(result, clean)
	}

	return result
}
