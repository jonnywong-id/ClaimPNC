package inboxadmin

import "strings"

// BusinessLine adalah penyaring lini bisnis pada bilah atas layar.
//
// # Dari mana nilainya, dan kenapa hanya lima
//
// Dari `Activity/SetTempClaimRegistandNotRegist-Act.xml` langkah 12–16, yang bercabang atas
// `TempView2.Remark` — properti yang diisi dropdown "Business" di layar. Kelima cabangnya
// menyusun potongan SQL yang berbeda, dan tidak ada cabang keenam.
//
// Predikat aslinya, apa adanya:
//
//	NONMBU   GROUPPANEL_1 IN ('003','004','006','009')
//	         AND BUSINESSGROUPID NOT IN ('10008','10010','10015','10023')
//	BONDING  BUSINESSGROUPID IN ('10008','10010','10015','10023')
//	PA       GROUPPANEL_1 = '002'
//	TRAVEL   GROUPPANEL_1 = '005'
//	ALL      tanpa saringan
//
// Perhatikan NONMBU: ia bukan "semua selain PA dan Travel", melainkan empat Group Panel
// tertentu DIKURANGI kelompok Bonding. Klaim ber-Group Panel di luar keenam kode itu tidak
// muncul di pilihan mana pun kecuali ALL.
type BusinessLine string

// Kelima nilai penyaring lini bisnis.
//
// Nilainya huruf besar persis seperti yang dibandingkan activity lama, dan itu bukan gaya
// penulisan melainkan kontrak: ia dikirim layar sebagai parameter query.
const (
	BusinessAll     BusinessLine = "ALL"
	BusinessNonMBU  BusinessLine = "NONMBU"
	BusinessBonding BusinessLine = "BONDING"
	BusinessPA      BusinessLine = "PA"
	BusinessTravel  BusinessLine = "TRAVEL"
)

// businessLines adalah kelimanya dalam urutan tampilnya di dropdown.
var businessLines = []BusinessLine{
	BusinessAll, BusinessNonMBU, BusinessBonding, BusinessPA, BusinessTravel,
}

// BusinessLines mengembalikan isi dropdown "Business".
func BusinessLines() []BusinessLine {
	result := make([]BusinessLine, len(businessLines))
	copy(result, businessLines)
	return result
}

// Label adalah teks yang dibaca pengguna pada dropdown.
func (b BusinessLine) Label() string {
	switch b {
	case BusinessAll:
		return "Semua Lini Bisnis"
	case BusinessNonMBU:
		return "Non-MBU"
	case BusinessBonding:
		return "Bonding"
	case BusinessPA:
		return "Personal Accident"
	case BusinessTravel:
		return "Travel"
	default:
		return string(b)
	}
}

// parseBusinessLine membaca pilihan lini bisnis dari isian layar.
//
// Isian kosong berarti ALL, bukan galat: layar yang baru dibuka belum memilih apa pun, dan
// "belum memilih" di sistem lama memang berarti tanpa saringan.
func parseBusinessLine(raw string) (BusinessLine, bool) {
	value := BusinessLine(strings.ToUpper(strings.TrimSpace(raw)))
	if value == "" {
		return BusinessAll, true
	}
	for _, known := range businessLines {
		if known == value {
			return value, true
		}
	}
	return "", false
}

// QueryInput adalah isian mentah dari layar, belum divalidasi.
//
// Ia dipisah dari Query supaya yang sudah tervalidasi tidak dapat dibentuk begitu saja:
// Query hanya lahir lewat NewQuery.
type QueryInput struct {
	// Tab adalah kode tab yang diminta. Kosong berarti DefaultTab.
	Tab string

	// Business adalah pilihan dropdown lini bisnis. Kosong berarti ALL.
	Business string

	// Keyword adalah isi kotak cari.
	Keyword string
}

// Query adalah permintaan isi satu tab yang sudah tervalidasi.
type Query struct {
	// Tab adalah tab yang diminta, lengkap dengan kolom dan kemampuannya.
	Tab Tab

	// Business adalah penyaring lini bisnis yang BENAR-BENAR dipakai.
	//
	// Ia selalu BusinessAll pada tab yang tidak mendukungnya — bukan nilai yang dikirim
	// layar. Tanpa penetapan itu, dua permintaan yang hasilnya pasti sama akan tampak
	// berbeda di log dan di kunci cache.
	Business BusinessLine

	// Keyword adalah kata kunci pencarian yang BENAR-BENAR dipakai, sudah dipangkas.
	//
	// Ia selalu kosong pada tab yang tidak mendukung pencarian, dengan alasan yang sama.
	Keyword string

	// Caller adalah identitas pemanggil. Tiga tab menyaring menurut nilai ini.
	Caller Caller
}

// NewQuery membentuk permintaan yang sah, atau menyatakan apa yang salah.
func NewQuery(input QueryInput, caller Caller) (Query, error) {
	cleanCaller := caller.Clean()
	if cleanCaller.Login == "" {
		return Query{}, ErrCallerUnknown
	}

	code := strings.TrimSpace(input.Tab)
	if code == "" {
		code = DefaultTab
	}

	tab, known := FindTab(code)
	if !known {
		// Tab Komunikasi dijawab dengan pesan yang menyebut alasannya, bukan dengan
		// "tab tidak dikenal" yang membuat orang mengira modulnya belum selesai. Tautan
		// lama yang masih menyimpan `CityID=4` di riwayat peramban akan sampai ke sini.
		for _, disabled := range DisabledTabs {
			if disabled.Code == code {
				return Query{}, NewValidationError([]Violation{{
					Field: FieldTab,
					Message: "Tab \"" + disabled.Name + "\" " + disabled.Reason +
						", sehingga tidak dibawa ke sistem baru.",
				}})
			}
		}

		return Query{}, NewValidationError([]Violation{{
			Field:   FieldTab,
			Message: "Tab tidak dikenal.",
		}})
	}

	business, valid := parseBusinessLine(input.Business)
	if !valid {
		return Query{}, NewValidationError([]Violation{{
			Field:   FieldBusiness,
			Message: "Pilihan lini bisnis tidak dikenal.",
		}})
	}

	query := Query{Tab: tab, Business: BusinessAll, Caller: cleanCaller}

	if tab.SupportsBusinessFilter {
		query.Business = business
	}
	if tab.SupportsSearch {
		query.Keyword = strings.TrimSpace(input.Keyword)
	}

	return query, nil
}
