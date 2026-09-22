package registrasi

import (
	"fmt"
	"sort"
)

// # Alur Register sebagai data
//
// Berkas ini adalah salinan `Flow/Register_Flow.xml` dalam bentuk yang dapat dijalankan
// dan diuji. Diagram lama memuat 23 shape — 1 Start, 13 Assignment, 7 Decision, 2 End —
// dan 30 konektor; seluruhnya ada di sini.
//
// # Kenapa data, bukan if-else
//
// Alur ini akan dibandingkan dengan Pega pada gerbang 1 (`D-54`). Perbandingan itu hanya
// mungkin bila definisinya dapat dibaca sebagai daftar, bukan ditelusuri sebagai kode.
// Bentuk data juga membuat satu hal mustahil: menambah tahap tanpa menyebut dari mana
// klaim datang dan ke mana ia pergi.

// QueueKind membedakan dua model penugasan yang `ADR-0019` tetapkan dipertahankan.
type QueueKind string

const (
	// QueueWorklist — tugas milik satu orang tertentu. Dipakai tahap yang butuh
	// kesinambungan penanganan.
	QueueWorklist QueueKind = "WORKLIST"

	// QueueWorkbasket — antrean bersama, diambil siapa pun yang berwenang. Dipakai
	// tahap yang dikerjakan sebuah tim.
	QueueWorkbasket QueueKind = "WORKBASKET"
)

// NodeKind membedakan ketiga bentuk simpul pada alur.
type NodeKind string

const (
	NodeStage    NodeKind = "TAHAP"
	NodeDecision NodeKind = "KEPUTUSAN"
	NodeEnd      NodeKind = "AKHIR"
)

// Pengenal setiap simpul alur Register.
//
// Nama Indonesia dipakai sebagai pengenal; `PegaID` pada tiap simpul memuat pengenal
// aslinya supaya jejak ke `Flow/Register_Flow.xml` tidak putus.
const (
	StageViewPolicy         = "view-polis"
	StageInputRegister      = "input-register"
	StageEstimatePA         = "estimasi-pa"
	StageEstimateTravel     = "estimasi-travel"
	StageEstimateAdmin      = "estimasi-admin"
	StageInvestigator       = "investigator"
	StageSendToAnalyst      = "kirim-analis"
	StageSendToTechnicalPIC = "kirim-pic-teknik"
	StageChooseSurveyor     = "pilih-surveyor"
	StageRCLPUCL            = "rcl-pucl"
	StageCompliance         = "compliance"
	StageRCLDoctor          = "rcl-dokter"
	StageAnalystDoctor      = "analyst-doctor"

	DecisionReturnFromRegister       = "kembali-dari-register"
	DecisionLinePA                   = "lini-pa"
	DecisionLineTravel               = "lini-travel"
	DecisionReturnFromEstimatePA     = "kembali-dari-estimasi-pa"
	DecisionReturnFromEstimateAdmin  = "kembali-dari-estimasi-admin"
	DecisionReturnFromEstimateTravel = "kembali-dari-estimasi-travel"
	DecisionAfterAnalyst             = "tujuan-setelah-analis"

	EndAfterSurveyor = "selesai-surveyor"
	EndAfterAnalyst  = "selesai-analis"
)

// Stage adalah satu perhentian alur yang menghasilkan Tugas bagi seseorang.
type Stage struct {
	ID   string
	Name string

	// PegaID adalah pengenal shape pada Flow/Register_Flow.xml — `Assignment1` dan
	// seterusnya. Ia tidak dipakai logika mana pun; ia ada supaya penelusuran ke
	// sumber tidak bergantung pada ingatan orang.
	PegaID string

	Queue QueueKind

	// Router adalah nama aturan penugasan yang menentukan SIAPA menerima tugas ini.
	// Tiga di antaranya — PNCAdminRouter, PNCTeknikRouter, RouterRCLDoctor — tidak ada
	// di export (`R-04`); pengisiannya ada di seam Penugasan, bukan di sini.
	Router string

	// Workbasket terisi hanya bila Antrean == QueueWorkbasket.
	Workbasket string

	// Operator terisi hanya bila tahap merutekan ke ORANG BERNAMA, bukan ke aturan.
	// Satu-satunya yang begitu adalah Analyst Doctor. Lihat catatan pada definisi.
	Operator string

	// ExitAction adalah nama Flow Action yang mengakhiri tahap ini di sistem lama.
	// Ia menjadi nama tindakan yang dikirim klien saat menyelesaikan tugas.
	ExitAction string

	// LateralJump adalah nama Ticket rule yang menjadikan tahap ini TUJUAN lompatan
	// lateral. Kosong berarti tahap hanya dapat dicapai lewat jalur normal.
	LateralJump string

	// Next adalah simpul yang dituju setelah ExitAction dijalankan.
	Next string
}

// Branch adalah satu jalan keluar bersyarat dari sebuah Keputusan.
type Branch struct {
	// Name adalah nama rule When di Pega — dipertahankan apa adanya, termasuk
	// kapitalisasinya yang tidak konsisten antar-rule.
	Name string

	// Test adalah aturannya. Ia menerima seluruh konteks karena satu di antaranya —
	// IsAnalystDoctor — ternyata menguji PERAN PEMANGGIL, bukan data klaim.
	Test func(FlowContext) bool

	Target string
}

// Decision adalah percabangan. Cabang diuji BERURUTAN; yang pertama benar menang.
// Bila tidak satu pun benar, klaim mengambil Selainnya.
type Decision struct {
	ID     string
	Name   string
	PegaID string

	Branch []Branch

	// Otherwise adalah cabang ELSE. Ia SELALU ada pada ketujuh keputusan alur ini.
	Otherwise string

	// OtherwiseName adalah label cabang ELSE pada diagram — `Not PA`, `Non MBU`,
	// `ElseRCLMSIG`, `Continue`.
	//
	// TEMUAN: ketiga nama pertama sempat tercatat sebagai rule When yang hilang dari
	// export (`R-16` pada `docs/ticketing/B-2-Input-Register/spec.md`). Keduanya bukan
	// rule: `pyConditionType` ketiganya `Else`, dan `pyFromTasks` menandainya `ELSE`.
	// Ia label diagram, bukan aturan. Tidak ada yang perlu diminta ke Tim Pega untuk
	// ini.
	OtherwiseName string
}

// End adalah simpul penutup alur.
type End struct {
	ID     string
	Name   string
	PegaID string

	// ProcessStatus yang diberikan klaim saat mencapai simpul ini.
	ProcessStatus ProcessStatus

	LateralJump string
}

// Node adalah satu titik pada alur: tahap, keputusan, atau akhir.
type Node struct {
	Kind     NodeKind
	Stage    Stage
	Decision Decision
	End      End
}

// ID mengembalikan pengenal simpul apa pun jenisnya.
func (s Node) ID() string {
	switch s.Kind {
	case NodeStage:
		return s.Stage.ID
	case NodeDecision:
		return s.Decision.ID
	default:
		return s.End.ID
	}
}

// FlowContext adalah bahan yang dipakai keputusan alur untuk memilih cabang.
type FlowContext struct {
	// Claim adalah klaim yang sedang berjalan.
	Claim Claim

	// CallerRoles adalah daftar peran pengguna yang sedang menekan tombol.
	//
	// Ia ada di sini karena satu keputusan alur benar-benar bergantung padanya, bukan
	// karena rancangan ini menginginkannya. Lihat DecisionAfterAnalyst.
	CallerRoles []string
}

// HasRole menyatakan pemanggil memegang salah satu peran yang disebut.
func (k FlowContext) HasRole(roles ...string) bool {
	for _, p := range roles {
		for _, owned := range k.CallerRoles {
			if equalFold(owned, p) {
				return true
			}
		}
	}
	return false
}

// Definition adalah seluruh alur: simpul-simpulnya dan dari mana ia dimulai.
type Definition struct {
	Name  string
	Start string

	node map[string]Node
}

// Node mengambil satu simpul berdasarkan pengenalnya.
func (d Definition) Node(id string) (Node, bool) {
	s, ok := d.node[id]
	return s, ok
}

// Stage mengambil satu tahap berdasarkan pengenalnya.
func (d Definition) Stage(id string) (Stage, bool) {
	s, ok := d.node[id]
	if !ok || s.Kind != NodeStage {
		return Stage{}, false
	}
	return s.Stage, true
}

// Stages mengembalikan seluruh tahap, terurut tetap berdasarkan pengenalnya.
//
// Urutannya sengaja tidak mengikuti urutan alur: alur ini bercabang, sehingga "urutan"
// tunggal tidak ada. Layar yang ingin menampilkan jalur klaim memakai Jalur().
func (d Definition) Stages() []Stage {
	result := make([]Stage, 0, len(d.node))
	for _, s := range d.node {
		if s.Kind == NodeStage {
			result = append(result, s.Stage)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

// Next menentukan tahap mana yang menerima klaim setelah sebuah tahap selesai.
//
// Ia menelusuri keputusan sebanyak yang diperlukan sampai bertemu tahap atau akhir.
// Keputusan tidak boleh membentuk lingkaran; batas penelusuran menjadikan cacat itu
// galat yang terlihat, bukan permintaan yang menggantung selamanya.
func (d Definition) Next(fromStage string, fctx FlowContext) (Node, []string, error) {
	stage, ok := d.Stage(fromStage)
	if !ok {
		return Node{}, nil, fmt.Errorf("%w: %q", ErrUnknownStage, fromStage)
	}
	return d.walk(stage.Next, fctx)
}

// maxWalkDepth adalah jumlah keputusan berantai terbanyak yang masih dianggap sah.
// Alur Register terpanjang melewati dua keputusan berturut-turut (IsBack lalu IsPA);
// delapan memberi ruang berlebih tanpa membiarkan lingkaran berjalan selamanya.
const maxWalkDepth = 8

func (d Definition) walk(id string, fctx FlowContext) (Node, []string, error) {
	var trace []string
	for i := 0; i < maxWalkDepth; i++ {
		node, ok := d.node[id]
		if !ok {
			return Node{}, trace, fmt.Errorf("%w: %q", ErrUnknownNode, id)
		}
		if node.Kind != NodeDecision {
			return node, trace, nil
		}

		k := node.Decision
		target := k.Otherwise
		chosen := k.OtherwiseName
		for _, c := range k.Branch {
			if c.Test(fctx) {
				target = c.Target
				chosen = c.Name
				break
			}
		}
		trace = append(trace, k.Name+" → "+chosen)
		id = target
	}
	return Node{}, trace, fmt.Errorf("%w: mulai dari %q", ErrFlowLoop, id)
}

// Path menyusun rangkaian tahap yang akan dilalui klaim dari sebuah tahap sampai
// akhir, dengan konteks yang berlaku sekarang.
//
// Ia dipakai layar untuk menunjukkan "sesudah ini ke mana" — bukan untuk mengambil
// keputusan. Jalur dapat berubah bila data klaim berubah, dan itu memang benar: di
// sistem lama pun tujuan klaim baru diketahui saat tombol ditekan.
func (d Definition) Path(fromStage string, fctx FlowContext) ([]string, error) {
	path := []string{fromStage}
	current := fromStage
	for i := 0; i < len(d.node); i++ {
		node, _, err := d.Next(current, fctx)
		if err != nil {
			return path, err
		}
		if node.Kind == NodeEnd {
			return path, nil
		}
		// Perpindahan mundur — tombol Back — akan membuat jalur berputar. Ia dihentikan
		// di sini: yang ditampilkan layar adalah jalur MAJU.
		if contains(path, node.Stage.ID) {
			return path, nil
		}
		path = append(path, node.Stage.ID)
		current = node.Stage.ID
	}
	return path, nil
}

func contains(list []string, value string) bool {
	for _, d := range list {
		if d == value {
			return true
		}
	}
	return false
}
