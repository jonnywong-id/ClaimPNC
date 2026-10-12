package riwayatklaim

import (
	"strings"
	"time"
)

// Column adalah kolom tambahan yang hanya dibawa sebagian tipe pencarian.
//
// Sebelas kolom dibawa SELURUH kueri dan karena itu tidak disebut di sini; yang disebut
// hanyalah empat kolom yang kemunculannya bergantung pada tipe pencarian.
type Column string

const (
	ColumnAcceptanceNumber Column = "nomor_akseptasi"
	ColumnAuctionHouseID   Column = "nomor_balai_lelang"
	ColumnInsuredItemName  Column = "nama_objek"
	ColumnBirthDate        Column = "tanggal_lahir"
)

// SearchType adalah satu pilihan pada dropdown "Tipe Pencarian".
//
// # Tiga isian, bukan satu
//
// Formulir lama memuat TIGA isian, dan tipe pencarian menentukan mana yang tampak
// (`Section/PNCSearchKlaim-Section.xml`):
//
//	"Nama Pencarian"   teks     tipe 1,2,3,4,5,7,8,11,12,13   properti SearchName
//	"Tanggal Pencarian" tanggal tipe 6,12                     properti DateOfSendInputor
//	"Tanggal Lahir"    tanggal  tipe 9                        properti SearchDate
//
// Bentuk itu ditiru persis, termasuk tipe 12 yang menampilkan DUA isian sekaligus.
// Menyederhanakannya menjadi satu isian akan menghapus cacat yang Work Owner putuskan
// untuk direplikasi — lihat Criteria.QueryValue.
type SearchType struct {
	// Code adalah kode yang dipakai sistem lama, misalnya "7" untuk No Klaim.
	//
	// Kodenya DIPERTAHANKAN apa adanya, termasuk lompatan dari "9" ke "11". Menomori
	// ulang menjadi 1..12 akan membuat rujukan mana pun ke "tipe 13" — di percakapan
	// tim maupun di dokumen — menunjuk tipe yang berbeda.
	Code string

	// Label adalah teks yang dibaca pengguna di dropdown.
	//
	// SUMBERNYA TERBATAS, dan itu perlu diketahui siapa pun yang membacanya: daftar
	// pilihan dropdown sistem lama tersimpan pada definisi properti `SearchType`
	// (`pyListSource=associated`), dan definisi properti TIDAK ADA di export. Label di
	// sini karena itu diturunkan dari deskripsi langkah
	// `Activity/PNCSearchHistoryKlaim_Act-Act.xml` — "Search Type 7 NO KLAIM" dan
	// saudara-saudaranya — lalu disepakati Work Owner 2026-09-20.
	Label string

	// ShowsText menyatakan isian "Nama Pencarian" tampak untuk tipe ini.
	ShowsText bool

	// ShowsSearchDate menyatakan isian "Tanggal Pencarian" tampak untuk tipe ini.
	ShowsSearchDate bool

	// ShowsBirthDate menyatakan isian "Tanggal Lahir" tampak untuk tipe ini.
	ShowsBirthDate bool

	// ExtraColumns adalah kolom tambahan yang dibawa kueri tipe ini.
	ExtraColumns []Column

	// Available bernilai false bila tipenya belum dapat dijalankan.
	//
	// Ia TETAP tampil di dropdown dalam keadaan tidak dapat dipilih, mengikuti perlakuan
	// yang sama pada butir menu yang layarnya belum ada: dengan begitu kemajuan migrasi
	// terbaca langsung dari layar, dan pengguna tidak melaporkan pilihan yang "hilang".
	Available bool

	// Unavailable menjelaskan MENGAPA tipenya belum dapat dijalankan, untuk ditampilkan
	// di layar. Kosong bila Available.
	Unavailable string
}

// Kode kedua belas tipe pencarian.
//
// Kode "10" TIDAK ADA, dan ketiadaannya bukan kelalaian penulisan: tidak ada satu pun
// langkah `Search Type 10` di activity lama, dan tidak ada kueri yang menganggur
// menunggunya. Ia dilewati apa adanya (keputusan Work Owner 2026-09-20).
const (
	TypePolicyNumber     = "1"
	TypeInsuredName      = "2"
	TypeInsuredItemName  = "3"
	TypePLANumber        = "4"
	TypeDLANumber        = "5"
	TypeLossDate         = "6"
	TypeClaimNumber      = "7"
	TypeAcceptanceNumber = "8"
	TypeBirthDate        = "9"
	TypeSurveyNumber     = "11"
	TypeAccountNumber    = "12"
	TypeAuctionHouseID   = "13"
)

// searchTypes adalah kedua belas tipe, dalam urutan kodenya.
//
// Ia tidak diekspor supaya tidak ada pemanggil yang dapat mengubah isinya; yang diekspor
// adalah SearchTypes() yang mengembalikan salinan.
var searchTypes = []SearchType{
	{Code: TypePolicyNumber, Label: "No Polis", ShowsText: true, Available: true},
	{Code: TypeInsuredName, Label: "Nama Customer", ShowsText: true, Available: true},
	{Code: TypeInsuredItemName, Label: "Nama Objek", ShowsText: true, Available: true},
	{Code: TypePLANumber, Label: "No PLA", ShowsText: true, Available: true},
	{Code: TypeDLANumber, Label: "No DLA", ShowsText: true, Available: true},
	{Code: TypeLossDate, Label: "Tgl Kejadian", ShowsSearchDate: true, Available: true},
	{Code: TypeClaimNumber, Label: "No Klaim", ShowsText: true, Available: true},
	{Code: TypeAcceptanceNumber, Label: "No Akseptasi", ShowsText: true, Available: true},
	{
		Code:  TypeBirthDate,
		Label: "Tanggal Lahir",
		// Isian yang tampak adalah "Tanggal Lahir" — BUKAN "Tanggal Pencarian" yang
		// justru dipakai kueri. Di situlah cacatnya. Lihat Criteria.QueryValue.
		ShowsBirthDate: true,
		// Kedua kolom ini HANYA dibawa kueri tipe ini — `t_person.FullName` dan
		// `t_person.ASMDateOfBirth`.
		ExtraColumns: []Column{ColumnInsuredItemName, ColumnBirthDate},
		Available:    true,
	},
	{Code: TypeSurveyNumber, Label: "No Survey", ShowsText: true, Available: true},
	{
		Code:  TypeAccountNumber,
		Label: "No Rekening",
		// DUA isian sekaligus, dan itu memang yang digambar layar lama: isian teks
		// tampil untuk tipe 12, dan isian tanggal juga tampil untuk tipe 12.
		ShowsText:       true,
		ShowsSearchDate: true,
		Available:       false,
		Unavailable: "Menunggu API pengganti DB Link ke basis data pembayaran (R-03). " +
			"Di sistem lama pun nomor rekening yang diketik tidak pernah sampai ke kueri.",
	},
	{
		Code:      TypeAuctionHouseID,
		Label:     "ID Balai Lelang",
		ShowsText: true,
		// Kedua kolom ini HANYA dibawa kueri tipe ini — `detail_pnc_salvage`.
		ExtraColumns: []Column{ColumnAcceptanceNumber, ColumnAuctionHouseID},
		Available:    true,
	},
}

// SearchTypes mengembalikan kedua belas tipe pencarian dalam urutan kodenya.
func SearchTypes() []SearchType {
	list := make([]SearchType, len(searchTypes))
	copy(list, searchTypes)
	return list
}

// FindSearchType mencari tipe berdasarkan kodenya.
func FindSearchType(code string) (SearchType, bool) {
	clean := strings.TrimSpace(code)
	for _, candidate := range searchTypes {
		if candidate.Code == clean {
			return candidate, true
		}
	}
	return SearchType{}, false
}

// Criteria adalah isi formulir pencarian yang sudah sah.
//
// Ia hanya dibentuk lewat NewCriteria, sehingga kode yang menerimanya tidak perlu
// memeriksa ulang apakah tipenya dikenal atau isiannya terisi.
type Criteria struct {
	// Type adalah tipe pencarian yang dipilih.
	Type SearchType

	// Text adalah isian "Nama Pencarian", sudah dipangkas dan DIBESARKAN hurufnya.
	//
	// Sistem lama menaikkan hurufnya lewat `@toUpperCase(TempSearch.SearchName)` sebelum
	// nilainya masuk kueri, sehingga pencarian nama tidak peka besar-kecil huruf.
	// Perilaku itu dipertahankan.
	Text string

	// SearchDate adalah isian "Tanggal Pencarian" — properti `DateOfSendInputor`.
	//
	// Ia TANGGAL KALENDER, bukan titik waktu: kueri lama membandingkannya dengan
	// `trunc(DATEOFLOSS)`, sehingga jamnya tidak pernah ikut menentukan.
	SearchDate *time.Time

	// BirthDate adalah isian "Tanggal Lahir" — properti `SearchDate` di sistem lama.
	//
	// Ia DIKUMPULKAN tetapi TIDAK PERNAH dipakai kueri. Lihat QueryValue.
	BirthDate *time.Time
}

// QueryValue mengembalikan nilai yang BENAR-BENAR dikirim ke kueri.
//
// # Kenapa ada method ini, alih-alih kueri membaca langsung isiannya
//
// Karena sistem lama memilih nilainya dengan aturan yang tidak sama dengan "isian yang
// tampak di layar", dan perbedaan itu harus terbaca di satu tempat, bukan tersebar.
//
// Aturannya di `Activity/PNCSearchHistoryKlaim_Act-Act.xml`:
//
//	langkah 6  CARI4 := @toUpperCase(TempSearch.SearchName)
//	           prakondisi: SearchType!="6" || SearchType!="9"   ← SELALU BENAR
//	langkah 7  CARI4 := TempSearch.DateOfSendInputor
//	           prakondisi: SearchType=="6" || =="9" || =="12"
//
// Prakondisi langkah 6 adalah TAUTOLOGI — sebuah nilai mustahil bukan "6" DAN bukan "9"
// secara bersamaan salah, sehingga kondisinya tidak pernah gagal. Akibatnya langkah 6
// selalu berjalan, lalu langkah 7 menimpanya untuk tipe 6, 9, dan 12.
//
// # Cacat yang direplikasi dengan sengaja
//
// Untuk tipe 9 Tanggal Lahir, isian yang TAMPAK adalah "Tanggal Lahir" tetapi yang
// DIPAKAI adalah "Tanggal Pencarian" — yang justru tersembunyi untuk tipe itu, sehingga
// selalu kosong. Pencarian Tanggal Lahir karena itu tidak pernah mengembalikan baris.
//
// Work Owner memutuskan 2026-09-20 ketiga cacat layar ini DIREPLIKASI apa adanya demi
// kesetaraan `P-5`, bukan diperbaiki. Perilakunya diuji di riwayatklaim_test.go supaya
// ia tetap terlihat sebagai keputusan, bukan tertimbun menjadi kelalaian.
//
// Nilai kedua menyatakan apakah yang dikirim berupa tanggal.
func (c Criteria) QueryValue() (text string, date *time.Time, isDate bool) {
	switch c.Type.Code {
	case TypeLossDate, TypeBirthDate, TypeAccountNumber:
		return "", c.SearchDate, true
	default:
		return c.Text, nil, false
	}
}

// CriteriaInput adalah isian mentah dari lapisan transport, sebelum divalidasi.
type CriteriaInput struct {
	Type       string
	Text       string
	SearchDate *time.Time
	BirthDate  *time.Time
}

// NewCriteria memvalidasi isian pencarian dan membentuk Criteria.
//
// Seluruh pelanggaran dikumpulkan sekaligus, tidak berhenti pada yang pertama
// (`11-CROSSCUTTING.md` §1.2). Pada formulir sependek ini bedanya kecil, tetapi aturannya
// berlaku seragam di seluruh aplikasi — dan pengecualian yang dibuat "karena formulirnya
// pendek" adalah pengecualian yang kemudian ditiru di formulir panjang.
//
// # Satu aturan yang TIDAK ada di sistem lama
//
// Isian yang tampak untuk tipe terpilih WAJIB diisi. Sistem lama tidak memeriksanya sama
// sekali, dan akibatnya nyata: menekan Cari dengan isian kosong pada tipe "Nama Customer"
// menghasilkan `QQNAME like '%%'` — seluruh isi T_CLAIM_PNC, puluhan juta baris (`D-10`).
//
// Ia dipasang sebagai PENGAMAN, bukan perubahan aturan bisnis, dan dicatat di
// docs/keputusan-implementasi.md supaya tidak terbaca sebagai perbaikan diam-diam.
func NewCriteria(input CriteriaInput) (Criteria, error) {
	searchType, known := FindSearchType(input.Type)
	if !known {
		// Tipe yang tidak dikenal menghentikan pemeriksaan selanjutnya: tanpa tipe, tidak
		// ada dasar untuk menilai isian mana yang wajib.
		return Criteria{}, NewValidationError([]Violation{{
			Field:   FieldSearchType,
			Message: "Pilih tipe pencarian lebih dulu.",
		}})
	}

	if !searchType.Available {
		return Criteria{}, NewValidationError([]Violation{{
			Field:   FieldSearchType,
			Message: "Tipe pencarian " + searchType.Label + " belum tersedia. " + searchType.Unavailable,
		}})
	}

	var violations []Violation

	text := strings.ToUpper(strings.TrimSpace(input.Text))
	if searchType.ShowsText && text == "" {
		violations = append(violations, Violation{
			Field:   FieldSearchValue,
			Message: "Isi " + searchType.Label + " yang dicari.",
		})
	}

	searchDate := calendarDate(input.SearchDate)
	if searchType.ShowsSearchDate && searchDate == nil {
		violations = append(violations, Violation{
			Field:   FieldSearchDate,
			Message: "Pilih Tanggal Pencarian.",
		})
	}

	birthDate := calendarDate(input.BirthDate)
	if searchType.ShowsBirthDate && birthDate == nil {
		violations = append(violations, Violation{
			Field:   FieldBirthDate,
			Message: "Pilih Tanggal Lahir.",
		})
	}

	if err := NewValidationError(violations); err != nil {
		return Criteria{}, err
	}

	// Isian yang TIDAK tampak untuk tipe ini dibuang, tidak dibawa diam-diam. Membawanya
	// berarti nilai sisa dari tipe yang dipilih sebelumnya ikut masuk kueri — kelas cacat
	// yang berbeda dari cacat warisan di QueryValue, dan yang ini tidak ada di sistem
	// lama karena di sana setiap isian punya properti klipboardnya sendiri.
	if !searchType.ShowsText {
		text = ""
	}
	if !searchType.ShowsSearchDate {
		searchDate = nil
	}
	if !searchType.ShowsBirthDate {
		birthDate = nil
	}

	return Criteria{
		Type:       searchType,
		Text:       text,
		SearchDate: searchDate,
		BirthDate:  birthDate,
	}, nil
}

// calendarDate memotong sebuah waktu menjadi tanggal kalender UTC.
//
// Kueri lama membandingkannya dengan `trunc(...)`, dan membawa serta jam akan membuat
// pencarian tanggal gagal tepat pada baris yang jamnya bukan tengah malam.
func calendarDate(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	truncated := time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	return &truncated
}
