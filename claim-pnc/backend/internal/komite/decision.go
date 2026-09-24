package komite

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Pencatatan keputusan komite — `TKT-B07-002`.
//
// # Kenapa keputusan ditulis ke tabel MILIK APLIKASI INI
//
// Kasusnya dibaca dari tabel warisan (lihat InboxRepo), tetapi keputusannya TIDAK boleh
// ditulis ke sana. `POOLDATA.T_CLAIM_KOMITE_LIST` masih ditulis Pega lewat
// `INSERTDATAKOMITELIST` dan dibaca puluhan kueri di sana; `P-1` menetapkan satu tabel
// hanya boleh ditulis satu sistem.
//
// `TKT-B07-002` sudah menetapkan jalan keluarnya pada bagian migrasi skema: **"Menambah
// tabel jejak komite. Backward-compatible."** Itulah yang dikerjakan — migrasi `0004`.
//
// # Akibat yang HARUS diketahui sebelum modul ini menyala di produksi
//
// Selama masa paralel, sebuah kasus yang sudah diputuskan di sini **tetap terbuka di
// Pega**: `PYSTATUSWORK` di sana tidak berubah, dan penugasan worklist-nya tidak dicabut.
// Layar ini menutupinya dengan menumpangkan keputusan kita di atas baris warisan —
// kasusnya keluar dari kotak Outstanding dan masuk ke riwayat — tetapi **Pega tidak tahu
// apa-apa tentang itu.**
//
// Konsekuensi praktisnya ada dua, dan keduanya bukan cacat melainkan sifat dari
// menjalankan dua sistem sekaligus:
//
//  1. Orang yang sama dapat memutuskan kasus itu LAGI di Pega. Keputusan mana yang sah
//     adalah pertanyaan yang tidak dapat dijawab kode ini.
//  2. Alur Pega TIDAK berlanjut ke jenjang berikutnya karena persetujuan kita. Selama
//     `B-5`/`B-6` belum ada, perpindahan jenjang hidup di sistem baru saja.
//
// Keduanya dicatat sebagai pertanyaan terbuka di `docs/keputusan-implementasi.md`, dan
// jawabannya milik Work Owner — bukan sesuatu yang dapat diputuskan dari export.

// DecisionKind adalah salah satu dari tiga keputusan yang dapat diberikan komite.
//
// Nilainya berbahasa Indonesia karena ia **kontrak API** (`D-80`).
type DecisionKind string

const (
	// DecisionApprove meneruskan kasus ke jenjang berikutnya, atau menyelesaikannya bila
	// ini jenjang terakhir.
	DecisionApprove DecisionKind = "setuju"

	// DecisionReject menghentikan seluruh komite, bukan hanya jenjang ini.
	//
	// Diambil apa adanya dari `Activity/KomitePost_Adjustment-Act.xml` step 65 dan
	// `KomitePost_LiableKlaim` step 22–23: saat `AcceptStatus = 2`, `KomiteCount` dipaksa
	// sama dengan `KomiteLoop` sehingga perulangan `IsKomiteLoop` berhenti. Penolakan
	// satu jenjang membatalkan seluruh sisa jenjang.
	DecisionReject DecisionKind = "tolak"

	// DecisionReturn mengembalikan kasus kepada pengaju untuk diperbaiki.
	//
	// Ia diminta `TKT-B07-002` sebagai keputusan ketiga. Berbeda dari dua yang lain, ia
	// TIDAK punya padanan yang terbaca di export — tidak ada nilai `AcceptStatus` ketiga
	// dan tidak ada Ticket rule yang tersisa untuknya (`KomiteAssign_ticket` dan
	// `komiteAccept_ticket` keduanya hilang, `R-16`).
	//
	// Karena itu perilakunya di sini dibuat SEMINIMAL mungkin yang masih berguna:
	// komitenya berhenti, kasusnya keluar dari antrean, dan alasannya tercatat. Ke mana
	// tepatnya kasus itu berpindah sesudahnya adalah lompatan lateral yang menuntut
	// artefak yang belum ada, dan tidak ditebak di sini.
	DecisionReturn DecisionKind = "kembalikan"
)

// Valid menyatakan keputusannya dikenali.
func (k DecisionKind) Valid() bool {
	switch k {
	case DecisionApprove, DecisionReject, DecisionReturn:
		return true
	default:
		return false
	}
}

// Terminal menyatakan keputusan ini menghentikan komite seluruhnya.
func (k DecisionKind) Terminal() bool {
	return k == DecisionReject || k == DecisionReturn
}

// MaxNoteLength membatasi panjang catatan keputusan.
//
// Angkanya mengikuti lebar kolom `NOTEKOMITE` yang dipakai sistem lama sejauh yang dapat
// dibaca dari pemakaiannya. DDL tabelnya tidak ada di export (`R-08`), sehingga batas ini
// adalah batas YANG KITA TETAPKAN untuk tabel kita sendiri — bukan hasil pembacaan.
const MaxNoteLength = 1000

// Decision adalah satu keputusan komite yang sudah tercatat.
//
// Ia tidak pernah diubah dan tidak pernah dihapus (`ADR-0012`). Perubahan pikiran
// dinyatakan dengan keputusan baru, bukan dengan menyunting yang lama — dan karena
// `D-59` menghapus pemisahan tugas, catatan inilah **satu-satunya bukti** bahwa
// persetujuan diberikan orang yang berwenang.
type Decision struct {
	ID string

	CaseID      string
	ClaimNumber string

	// Tier adalah jenjang keberapa yang diputuskan — `KOMITEKE` di sistem lama.
	//
	// Ia ditetapkan SERVER dari keadaan kasus, tidak pernah dikirim klien. Klien yang
	// boleh menyebut jenjangnya sendiri dapat menyetujui jenjang yang bukan gilirannya.
	Tier int

	Kind DecisionKind
	Note string

	// ActorLogin adalah login yang diketik pengguna, dinormalkan lewat OperatorKey.
	//
	// Inilah kunci yang Work Owner tetapkan untuk mencocokkan identitas sesi dengan
	// `OPERATOR_ID` sistem lama (`docs/keputusan-implementasi.md` §16.5).
	ActorLogin string

	// ActorName disimpan bersama keputusannya, bukan dirujuk ke tabel pengguna.
	//
	// Jejak yang namanya diambil lewat join akan BERUBAH ketika orangnya berganti nama
	// atau catatannya dihapus — dan jejak yang dapat berubah bukan jejak.
	ActorName string

	DecidedAt time.Time
}

// Outcome adalah kesimpulan atas seluruh keputusan pada satu kasus.
type Outcome string

const (
	OutcomePending  Outcome = "menunggu"
	OutcomeApproved Outcome = "disetujui"
	OutcomeRejected Outcome = "ditolak"
	OutcomeReturned Outcome = "dikembalikan"
)

// Progress adalah keadaan penjenjangan satu kasus menurut sistem baru.
type Progress struct {
	// TierCount adalah banyaknya jenjang yang harus menyetujui — `KomiteLoop` di sistem
	// lama, yaitu jumlah baris master yang cocok dengan nilai klaimnya.
	//
	// Nol berarti penjenjangannya BELUM DIKETAHUI, bukan berarti nol jenjang. Lihat
	// TierCountUnknown.
	TierCount int

	// CurrentTier adalah jenjang yang sedang menunggu — `KomiteCount` di sistem lama.
	// Nol berarti tidak ada yang menunggu lagi.
	CurrentTier int

	// ApprovedTiers adalah banyaknya jenjang yang sudah menyetujui.
	ApprovedTiers int

	Outcome Outcome

	// Decisions adalah seluruh keputusan pada kasus ini, terurut.
	Decisions []Decision
}

// TierCountUnknown menyatakan banyaknya jenjang belum dapat dihitung.
//
// Ia terjadi saat nilai klaim atau lini bisnisnya belum terbaca dari sumber mana pun —
// keadaan yang nyata hari ini, karena `B-5` (nilai penyelesaian) belum ada di aplikasi
// ini.
//
// Keadaan itu DITAMPILKAN, bukan disamarkan menjadi angka. Jenjang yang tidak diketahui
// lalu digambar sebagai "1 dari 1" akan membuat persetujuan pertama tampak menyelesaikan
// seluruh komite.
func (p Progress) TierCountUnknown() bool { return p.TierCount <= 0 }

// Closed menyatakan kasus ini tidak menerima keputusan lagi.
func (p Progress) Closed() bool {
	return p.Outcome == OutcomeApproved ||
		p.Outcome == OutcomeRejected ||
		p.Outcome == OutcomeReturned
}

// DecidedBy menyatakan orang itu sudah memberi keputusan pada kasus ini.
//
// # Kenapa "sekali per orang", bukan "sekali per jenjang"
//
// Penugasan komite bersifat per ORANG, bukan per antrean bersama (`T-7`): satu jenjang
// dimiliki satu operator bernama. Selama jumlah jenjang sebuah kasus belum dapat dihitung
// — dan hari ini memang belum, karena nilai klaimnya datang dari `B-5` yang belum ada —
// "sudah selesai" tidak dapat disimpulkan dari jumlah persetujuan.
//
// Yang MASIH dapat dijamin tanpa mengetahui jumlah jenjang adalah ini: seorang anggota
// komite memutuskan sekali. Itulah yang mengeluarkan kasus dari kotak Outstanding
// miliknya, persis seperti menyelesaikan assignment mengeluarkannya dari worklist di
// sistem lama — sementara jenjang lain tetap menunggu pemiliknya masing-masing.
func (p Progress) DecidedBy(login string) bool {
	key := OperatorKey(login)
	if key == "" {
		return false
	}
	for _, d := range p.Decisions {
		if OperatorKey(d.ActorLogin) == key {
			return true
		}
	}
	return false
}

// LastDecision mengembalikan keputusan terakhir, bila ada.
func (p Progress) LastDecision() (Decision, bool) {
	if len(p.Decisions) == 0 {
		return Decision{}, false
	}
	return p.Decisions[len(p.Decisions)-1], true
}

// Evaluate menyimpulkan keadaan penjenjangan dari seluruh keputusan yang tercatat.
//
// # Kenapa dihitung ulang, bukan disimpan
//
// Keadaan yang disimpan dapat berselisih dengan kejadian yang membentuknya, dan
// selisihnya tidak terlihat sampai seseorang membandingkan keduanya. Menghitungnya dari
// daftar keputusan membuat keduanya tidak mungkin berbeda — daftar keputusan adalah
// satu-satunya kebenaran, dan ia append-only.
//
// # Aturan yang diterapkan, dan dari mana asalnya
//
//	setuju      → jenjang bertambah satu; selesai bila sudah mencapai TierCount
//	tolak       → berhenti seluruhnya  (KomitePost_Adjustment:65)
//	kembalikan  → berhenti seluruhnya  (TKT-B07-002, tanpa padanan di export)
func Evaluate(decisions []Decision, tierCount int) Progress {
	sorted := append([]Decision(nil), decisions...)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Tier != sorted[j].Tier {
			return sorted[i].Tier < sorted[j].Tier
		}
		return sorted[i].DecidedAt.Before(sorted[j].DecidedAt)
	})

	progress := Progress{TierCount: tierCount, Decisions: sorted, Outcome: OutcomePending}

	for _, d := range sorted {
		switch d.Kind {
		case DecisionApprove:
			progress.ApprovedTiers++
		case DecisionReject:
			progress.Outcome = OutcomeRejected
			progress.CurrentTier = 0
			return progress
		case DecisionReturn:
			progress.Outcome = OutcomeReturned
			progress.CurrentTier = 0
			return progress
		}
	}

	// Jenjang yang belum diketahui TIDAK pernah dinyatakan selesai. Menyimpulkan
	// "seluruh jenjang sudah menyetujui" dari angka yang tidak diketahui berarti membuka
	// akseptasi tanpa dasar — persis yang invarian `I-5` larang.
	if !progress.TierCountUnknown() && progress.ApprovedTiers >= progress.TierCount {
		progress.Outcome = OutcomeApproved
		progress.CurrentTier = 0
		return progress
	}

	progress.CurrentTier = progress.ApprovedTiers + 1
	return progress
}

// DecisionCommand adalah permintaan mencatat satu keputusan.
//
// Ia sengaja TIDAK memuat Tier maupun DecidedAt: keduanya milik server. Klien yang boleh
// menyebut jenjang dapat menyetujui jenjang yang bukan gilirannya, dan klien yang boleh
// menyebut waktu dapat mencatat keputusan bertanggal kemarin.
type DecisionCommand struct {
	CaseID string
	Kind   DecisionKind
	Note   string
}

// Normalize merapikan perintah sebelum divalidasi.
func (c DecisionCommand) Normalize() DecisionCommand {
	c.CaseID = strings.TrimSpace(c.CaseID)
	c.Kind = DecisionKind(strings.ToLower(strings.TrimSpace(string(c.Kind))))
	c.Note = strings.TrimSpace(c.Note)
	return c
}

// Validate mengumpulkan SELURUH pelanggaran sekaligus, bukan berhenti pada yang pertama.
//
// Ini kesetaraan perilaku, bukan selera: sistem lama menampilkan seluruh pesan validasi
// bersamaan (`docs/Steering/12-CROSSCUTTING.md` §1.2 butir 1).
//
// # Kenapa catatan WAJIB pada tolak dan kembalikan, tetapi tidak pada setuju
//
// Penolakan tanpa alasan tidak dapat ditindaklanjuti siapa pun: pengaju tidak tahu apa
// yang harus diperbaiki, dan pemeriksa kelak tidak tahu mengapa uang tidak jadi dibayar.
// Persetujuan tidak punya masalah itu — yang disetujui adalah nilai yang sudah tertulis.
//
// Ini PENAMBAHAN terhadap sistem lama, yang tidak mewajibkan `NOTEKOMITE` pada keadaan
// mana pun. Ia dicatat sebagai penyimpangan yang disengaja, bukan peniruan.
func (c DecisionCommand) Validate() error {
	var violations []Violation

	if !c.Kind.Valid() {
		violations = append(violations, Violation{
			Field:   FieldDecision,
			Message: "Pilih salah satu: setuju, tolak, atau kembalikan.",
		})
	}
	if c.Kind.Terminal() && c.Note == "" {
		violations = append(violations, Violation{
			Field:   FieldNote,
			Message: "Catatan wajib diisi supaya alasannya dapat ditindaklanjuti.",
		})
	}
	// Dihitung per RUNE, bukan per byte. Catatan komite ditulis dalam bahasa Indonesia
	// dan sering memuat karakter di luar ASCII; membatasinya per byte akan menolak
	// kalimat yang panjangnya wajar hanya karena hurufnya kebetulan beraksen.
	//
	// Angka pada pesannya diambil dari konstantanya, tidak ditulis ulang sebagai teks,
	// sehingga keduanya tidak dapat berselisih saat batasnya berubah.
	if len([]rune(c.Note)) > MaxNoteLength {
		violations = append(violations, Violation{
			Field:   FieldNote,
			Message: "Catatan paling panjang " + strconv.Itoa(MaxNoteLength) + " karakter.",
		})
	}
	return NewValidationError(violations)
}

// DecisionRepo adalah seam ke tabel keputusan MILIK APLIKASI INI.
//
// Berbeda dari InboxRepo yang membaca tabel warisan, tabel di balik seam ini dibuat
// migrasi `0004` dan tidak dibaca maupun ditulis Pega. `P-1` terpenuhi tanpa negosiasi
// kepemilikan karena tidak ada sistem lain yang menyentuhnya.
//
// Ia **append-only**: tidak ada metode ubah dan tidak ada metode hapus, dan ketiadaannya
// disengaja. Keputusan komite tidak dapat dihapus (`ADR-0012`); penegakan yang
// sesungguhnya ada di hak akses basis data — akun aplikasi tidak diberi UPDATE maupun
// DELETE pada tabel ini (`docs/Steering/09-DATABASE-STRATEGY.md` §8).
type DecisionRepo interface {
	// ListForCases mengembalikan keputusan seluruh kasus yang disebut, dikunci CaseID.
	//
	// Banyak kasus sekaligus, bukan satu per satu: inbox menampilkan satu halaman penuh,
	// dan meminta keputusannya per baris adalah kueri di dalam perulangan — yang
	// `docs/Steering/15-NFR-PERFORMANCE-SCALABILITY.md` §3.2 larang.
	ListForCases(ctx context.Context, caseIDs []string) (map[string][]Decision, error)

	// Record menyimpan satu keputusan. Ia tidak pernah menimpa apa pun.
	Record(ctx context.Context, d Decision) error
}

// IDGenerator adalah seam ke pembangkit pengenal keputusan.
//
// Pengenalnya acak, bukan berurut: ia tidak punya makna bisnis dan tidak boleh
// membocorkan berapa banyak keputusan yang sudah tercatat.
type IDGenerator interface {
	New() string
}
