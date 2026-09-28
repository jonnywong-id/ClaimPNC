package archivedokumenklaim

import (
	"strings"
	"time"
)

// SearchMode adalah salah satu dari dua cara mencari berkas arsip.
//
// Keduanya dibaca dari `Activity/SearchDataArchiveFilling-Act.xml`, yang menyusun klausa
// WHERE-nya sendiri sebagai teks lalu menempelkannya ke kueri:
//
//	langkah 2   WHERE UPPER(NOKLAIM)='<kata kunci>' or NAMABOX='<kata kunci>'
//	                                                 or TERTANGGUNG='<kata kunci>'
//	langkah 10  WHERE trunc(TGLINPUT) >= <awal> and trunc(TGLINPUT) <= <akhir>
//
// Keduanya SALING MENGGANTIKAN, bukan saling melengkapi: klausa yang kedua menimpa yang
// pertama, bukan menambahinya. Bentuk itu ditiru persis, dan itulah sebabnya mode
// dinyatakan sebagai pilihan tunggal alih-alih empat isian yang boleh diisi bersamaan.
type SearchMode string

const (
	// ModeKeyword adalah "Tipe Pencarian Archive" dengan satu isian Keyword.
	ModeKeyword SearchMode = "kata_kunci"

	// ModeInputDate adalah pencarian rentang Tgl Input Dari–Sampai.
	ModeInputDate SearchMode = "tanggal_input"
)

// Criteria adalah isi formulir pencarian arsip yang sudah sah.
//
// Ia hanya dibentuk lewat NewCriteria, sehingga kode yang menerimanya tidak perlu
// memeriksa ulang apakah modenya dikenal atau isiannya terisi.
type Criteria struct {
	Mode SearchMode

	// Keyword adalah kata kunci, sudah dipangkas dan DIBESARKAN hurufnya.
	//
	// Sistem lama membandingkannya dengan `UPPER(NOKLAIM)` tetapi TIDAK membesarkan
	// huruf nilai yang diketik, sehingga pencarian nomor klaim berhuruf kecil tidak
	// pernah cocok. Yang dibesarkan di sini adalah nilainya, dan itu perbaikan yang
	// disebut terang di docs/keputusan-implementasi.md — bukan diam-diam.
	Keyword string

	// From dan To adalah rentang Tgl Input, keduanya tanggal kalender inklusif.
	From *time.Time
	To   *time.Time
}

// CriteriaInput adalah isian mentah dari lapisan transport, sebelum divalidasi.
type CriteriaInput struct {
	Mode    string
	Keyword string
	From    *time.Time
	To      *time.Time
}

// NewCriteria memvalidasi isian pencarian arsip dan membentuk Criteria.
//
// Seluruh pelanggaran dikumpulkan sekaligus, tidak berhenti pada yang pertama
// (`11-CROSSCUTTING.md` §1.2).
func NewCriteria(input CriteriaInput) (Criteria, error) {
	mode := SearchMode(strings.TrimSpace(input.Mode))
	if mode == "" {
		mode = ModeKeyword
	}

	if mode != ModeKeyword && mode != ModeInputDate {
		return Criteria{}, NewValidationError([]Violation{{
			Field:   FieldSearchMode,
			Message: "Tipe pencarian tidak dikenal. Pilih Keyword atau Tgl Input.",
		}})
	}

	var violations []Violation

	keyword := strings.ToUpper(strings.TrimSpace(input.Keyword))
	from := calendarDate(input.From)
	to := calendarDate(input.To)

	switch mode {
	case ModeKeyword:
		if keyword == "" {
			violations = append(violations, Violation{
				Field:   FieldKeyword,
				Message: "Isi Keyword yang dicari — No Klaim, Nama BOX, atau Nama Tertanggung.",
			})
		}
		// Isian yang tidak berlaku bagi mode ini dibuang, tidak dibawa diam-diam. Nilai
		// sisa dari mode sebelumnya yang ikut masuk kueri adalah kelas cacat yang tidak
		// ada di sistem lama, karena di sana tiap isian punya propertinya sendiri.
		from, to = nil, nil

	case ModeInputDate:
		if from == nil {
			violations = append(violations, Violation{
				Field:   FieldFrom,
				Message: "Pilih Tgl Input Dari.",
			})
		}
		if to == nil {
			violations = append(violations, Violation{
				Field:   FieldTo,
				Message: "Pilih Tgl Input Sampai.",
			})
		}
		if from != nil && to != nil && to.Before(*from) {
			violations = append(violations, Violation{
				Field:   FieldTo,
				Message: "Tgl Input Sampai tidak boleh lebih awal dari Tgl Input Dari.",
			})
		}
		keyword = ""
	}

	if err := NewValidationError(violations); err != nil {
		return Criteria{}, err
	}

	return Criteria{Mode: mode, Keyword: keyword, From: from, To: to}, nil
}

// ClaimSearchType adalah salah satu dari tiga pilihan "Tipe Input Archive".
//
// Ketiganya dibaca dari `Activity/ShowInsertArchiveKlaim_Act-Act.xml`, yang menerima
// parameter `noklaim`, `nopolis`, `namatertanggung`, dan `tipeinsert` lalu menaruh salah
// satunya ke satu properti pencarian yang sama.
type ClaimSearchType string

const (
	ClaimByNumber      ClaimSearchType = "no_klaim"
	ClaimByPolicy      ClaimSearchType = "no_polis"
	ClaimByInsuredName ClaimSearchType = "nama_tertanggung"
)

// ClaimSearchTypeOption adalah satu pilihan pada dropdown "Tipe Input Archive".
type ClaimSearchTypeOption struct {
	Code  ClaimSearchType
	Label string
}

// ClaimSearchTypes adalah ketiga pilihan dalam urutan layar lama.
func ClaimSearchTypes() []ClaimSearchTypeOption {
	return []ClaimSearchTypeOption{
		{Code: ClaimByNumber, Label: "No Klaim"},
		{Code: ClaimByPolicy, Label: "No Polis"},
		{Code: ClaimByInsuredName, Label: "Nama Tertanggung"},
	}
}

// ClaimCriteria adalah isi formulir pencarian klaim yang sudah sah.
type ClaimCriteria struct {
	Type ClaimSearchType

	// Value adalah nilai yang dicari, sudah dipangkas.
	//
	// Ia TIDAK dibesarkan hurufnya: kedua kueri lama membandingkannya apa adanya dengan
	// CLAIMID, NOPOLIS, dan QQNAME — tanpa UPPER di kedua sisi. Membesarkannya di sini
	// akan membuat pencarian nama tertanggung berhuruf campuran berhenti cocok, yaitu
	// mengubah hasil, bukan memperbaikinya.
	Value string
}

// NewClaimCriteria memvalidasi isian pencarian klaim.
func NewClaimCriteria(rawType, rawValue string) (ClaimCriteria, error) {
	searchType := ClaimSearchType(strings.TrimSpace(rawType))

	known := false
	for _, option := range ClaimSearchTypes() {
		if option.Code == searchType {
			known = true
			break
		}
	}
	if !known {
		return ClaimCriteria{}, NewValidationError([]Violation{{
			Field:   FieldClaimSearchType,
			Message: "Pilih Tipe Input Archive lebih dulu.",
		}})
	}

	value := strings.TrimSpace(rawValue)
	if value == "" {
		return ClaimCriteria{}, NewValidationError([]Violation{{
			Field:   FieldClaimKeyword,
			Message: "Isi Keyword klaim yang dicari.",
		}})
	}

	return ClaimCriteria{Type: searchType, Value: value}, nil
}

// BranchScope adalah lini bisnis yang boleh dilihat seorang petugas pada daftar kirim ke
// cabang.
//
// # Bentuknya persis sistem lama, termasuk yang tampak terbalik
//
// `Activity/GetDataArchiveCabangKlaim-Act.xml` menyusun saringannya dari
// `OperatorID.pyPosition`:
//
//	NONMBU   AND GROUPPANEL not in ('002','005')
//	PA       AND GROUPPANEL not in ('002')
//	TRAVEL   AND GROUPPANEL not in ('005')
//	lainnya  tanpa saringan lini bisnis sama sekali
//
// Dua baris tengahnya TAMPAK TERBALIK: Group Panel `002` adalah Personal Accident dan
// `005` adalah Travel, sehingga petugas berjabatan PA justru TIDAK melihat berkas PA, dan
// petugas Travel tidak melihat berkas Travel.
//
// Ia direplikasi apa adanya atas keputusan Work Owner 2026-09-24 (`P-5` murni), dan
// diuji supaya tetap terlihat sebagai keputusan alih-alih tertimbun menjadi kelalaian.
// Pertanyaan apakah ini memang yang dikehendaki tercatat di
// docs/permintaan-artefak-pega.md.
type BranchScope struct {
	// ExcludedGroupPanels adalah lini bisnis yang TIDAK boleh tampak. Kosong berarti
	// seluruh lini tampak.
	ExcludedGroupPanels []string
}

// Jabatan yang menentukan saringan lini bisnis, sebagaimana dibandingkan sistem lama.
const (
	PositionNonMBU = "NONMBU"
	PositionPA     = "PA"
	PositionTravel = "TRAVEL"
)

// Kode Group Panel yang disebut saringan itu.
const (
	GroupPanelPersonalAccident = "002"
	GroupPanelTravel           = "005"
)

// BranchScopeFor menerjemahkan jabatan petugas menjadi lini bisnis yang boleh dilihatnya.
//
// Perbandingannya tidak peka besar-kecil huruf, berbeda dari sistem lama yang memakai
// `==` apa adanya. Alasannya sama dengan penormalan kapitalisasi peran pada `D-58`:
// jabatan tersimpan dalam dua bentuk, dan membandingkannya mentah membuat sebagian
// petugas jatuh ke cabang "lainnya" tanpa ada yang menyadarinya.
func BranchScopeFor(position string) BranchScope {
	switch strings.ToUpper(strings.TrimSpace(position)) {
	case PositionNonMBU:
		return BranchScope{ExcludedGroupPanels: []string{
			GroupPanelPersonalAccident,
			GroupPanelTravel,
		}}
	case PositionPA:
		return BranchScope{ExcludedGroupPanels: []string{GroupPanelPersonalAccident}}
	case PositionTravel:
		return BranchScope{ExcludedGroupPanels: []string{GroupPanelTravel}}
	default:
		return BranchScope{}
	}
}

// calendarDate memotong sebuah waktu menjadi tanggal kalender UTC.
//
// Kueri lama membandingkannya dengan `trunc(TGLINPUT)`, dan membawa serta jam akan
// membuat pencarian rentang gagal tepat pada baris yang jamnya bukan tengah malam.
func calendarDate(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	truncated := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	return &truncated
}
