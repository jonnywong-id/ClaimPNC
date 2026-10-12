package archivedokumenklaim

import (
	"strings"
	"time"
)

// SearchColumn adalah kolom yang dicari dropdown "Tipe Pencarian Archive".
//
// # Dari mana ketiganya, dan kenapa tidak dapat dibaca dari export
//
// Dropdown-nya terikat properti ber-`pyListSource=associated`, yang berarti daftar
// pilihannya tersimpan di DEFINISI PROPERTI — dan definisi properti tidak ikut terekspor.
// Propertinya sendiri (`FlagNOLL`) dipakai ulang sembilan layar lain untuk hal yang sama
// sekali berbeda, sehingga menelusurinya pun tidak menolong.
//
// Ketiganya karena itu disebutkan Work Owner langsung dari layar Pega (2026-10-03), dan
// ketiganya cocok dengan ketiga kolom yang dirangkai kueri lama:
//
//	WHERE UPPER(NOKLAIM)='<kata kunci>' or NAMABOX='<kata kunci>'
//	                                    or TERTANGGUNG='<kata kunci>'
type SearchColumn string

const (
	ColumnClaimNumber SearchColumn = "no_klaim"
	ColumnBoxName     SearchColumn = "nama_box"
	ColumnInsuredName SearchColumn = "tertanggung"
)

// SearchColumnOption adalah satu pilihan pada dropdown "Tipe Pencarian Archive".
type SearchColumnOption struct {
	Code  SearchColumn
	Label string
}

// SearchColumns adalah ketiga pilihan, dalam urutan kolom pada kueri lama.
func SearchColumns() []SearchColumnOption {
	return []SearchColumnOption{
		{Code: ColumnClaimNumber, Label: "No Klaim"},
		{Code: ColumnBoxName, Label: "Nama BOX"},
		{Code: ColumnInsuredName, Label: "Tertanggung"},
	}
}

// Criteria adalah isi formulir pencarian arsip yang sudah sah.
//
// # Dua penyaring yang berdiri sendiri, bukan dua mode
//
// Ini koreksi atas rancangan pertama saya, yang menjadikan keduanya satu dropdown mode
// yang saling menggantikan. Bentuk itu tidak pernah ada di Pega.
//
// `Section/SecArchiveDokumen-Section.xml` memberi KEDUA blok ini
// `pyContainerVisibleWhen` yang sama — `FalgArchiveData.FlagASO==2`:
//
//	posisi 164179   "Tipe Pencarian Archive" + "Keyword"
//	posisi 707930   "Tgl Input Dari" + "Tgl Input Sampai"
//
// Keduanya karena itu TAMPIL BERSAMAAN di tab Archive File Klaim, masing-masing dengan
// isiannya sendiri.
//
// Ia hanya dibentuk lewat NewCriteria, sehingga kode yang menerimanya tidak perlu
// memeriksa ulang apakah isiannya terisi.
type Criteria struct {
	// Column adalah kolom yang dicari. Berarti hanya bila Keyword terisi.
	Column SearchColumn

	// Keyword adalah kata kunci, sudah dipangkas dan DIBESARKAN hurufnya.
	//
	// Sistem lama membandingkannya dengan `UPPER(NOKLAIM)` tetapi TIDAK membesarkan
	// huruf nilai yang diketik, sehingga pencarian nomor klaim berhuruf kecil tidak
	// pernah cocok. Yang dibesarkan di sini adalah nilainya — perbaikan yang disebut
	// terang di docs/keputusan-implementasi.md.
	//
	// Kosong berarti penyaring kata kunci tidak dipakai.
	Keyword string

	// From dan To adalah rentang Tgl Input, keduanya tanggal kalender inklusif.
	// Nil berarti penyaring tanggal tidak dipakai.
	From *time.Time
	To   *time.Time
}

// HasKeyword menyatakan penyaring kata kunci dipakai.
func (c Criteria) HasKeyword() bool { return c.Keyword != "" }

// HasDateRange menyatakan penyaring rentang Tgl Input dipakai.
func (c Criteria) HasDateRange() bool { return c.From != nil && c.To != nil }

// CriteriaInput adalah isian mentah dari lapisan transport, sebelum divalidasi.
type CriteriaInput struct {
	Column  string
	Keyword string
	From    *time.Time
	To      *time.Time
}

// NewCriteria memvalidasi isian pencarian arsip dan membentuk Criteria.
//
// Seluruh pelanggaran dikumpulkan sekaligus, tidak berhenti pada yang pertama
// (`11-CROSSCUTTING.md` §1.2).
//
// # Satu aturan yang TIDAK ada di sistem lama
//
// Setidaknya satu penyaring wajib terisi. Pega tidak memeriksanya: menekan Cari dengan
// seluruh isian kosong merangkai `WHERE UPPER(NOKLAIM)=” or NAMABOX=”` — yang tidak
// mengembalikan baris, tetapi tetap memindai seluruh tabel arsip lebih dulu.
func NewCriteria(input CriteriaInput) (Criteria, error) {
	var violations []Violation

	keyword := strings.ToUpper(strings.TrimSpace(input.Keyword))
	column := SearchColumn(strings.TrimSpace(input.Column))

	if keyword != "" {
		known := false
		for _, option := range SearchColumns() {
			if option.Code == column {
				known = true
				break
			}
		}
		if !known {
			violations = append(violations, Violation{
				Field:   FieldSearchColumn,
				Message: "Pilih Tipe Pencarian Archive lebih dulu.",
			})
		}
	}

	from := calendarDate(input.From)
	to := calendarDate(input.To)

	// Rentang setengah terisi DITOLAK, bukan dilengkapi sendiri. Menebak ujung yang
	// kosong — "sampai hari ini", misalnya — menghasilkan hasil yang tidak diminta
	// pengguna dan tidak terbaca dari layar.
	switch {
	case from != nil && to == nil:
		violations = append(violations, Violation{
			Field:   FieldTo,
			Message: "Isi juga Tgl Input Sampai.",
		})
	case from == nil && to != nil:
		violations = append(violations, Violation{
			Field:   FieldFrom,
			Message: "Isi juga Tgl Input Dari.",
		})
	case from != nil && to != nil && to.Before(*from):
		violations = append(violations, Violation{
			Field:   FieldTo,
			Message: "Tgl Input Sampai tidak boleh lebih awal dari Tgl Input Dari.",
		})
	}

	if keyword == "" && from == nil && to == nil {
		violations = append(violations, Violation{
			Field:   FieldKeyword,
			Message: "Isi Keyword atau rentang Tgl Input lebih dulu.",
		})
	}

	if err := NewValidationError(violations); err != nil {
		return Criteria{}, err
	}

	if keyword == "" {
		// Kolom yang tidak dipakai dibuang, supaya kueri tidak menerima kolom terpilih
		// tanpa nilai yang dicari.
		column = ""
	}

	return Criteria{Column: column, Keyword: keyword, From: from, To: to}, nil
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
