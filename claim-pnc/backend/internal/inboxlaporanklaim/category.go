package inboxlaporanklaim

import "strings"

// Category adalah satu tab pada Inbox Laporan Klaim.
//
// # Kenapa tabnya sembilan, dan dari mana angka itu
//
// `Activity/SetListRCV_Act-Act.xml` memilih kueri lewat satu rantai @if atas
// `param.Note`:
//
//	.SQLName := @if(param.Note=="1","ViewTableBrowseClaimRegister",
//	            @if(param.Note=="9","ViewTableBrowseClaimNotRegister",
//	            @if(param.Note==2,"ViewTableBrowseRCVInProcess",
//	            @if(param.Note==3,"ViewTableBrowseRCVAcc",
//	            @if(param.Note==4,"ViewTableBrowseRCVReject",
//	            @if(param.Note==5 || param.Note==7 || param.Note==8,
//	                "ViewRejectKomunikasiUser","ViewAllCase"))))))
//
// Enam kueri, tetapi SEMBILAN tab: ketiga nilai 5, 7, dan 8 memakai kueri yang sama
// dengan penyaring percakapan yang berbeda. Judul kesembilannya dibaca apa adanya dari
// `Section/ViewStatusReceiveDocument-Section.xml`.
//
// # Kenapa kodenya teks, bukan angka warisan
//
// Angka `1`, `2`, `3`, `4`, `5`, `7`, `8`, `9` tidak menyatakan apa pun bagi pembacanya,
// dan urutannya pun tidak berarti — `9` adalah tab kedua di layar, `1` tab pertama.
// Yang dipakai di kontrak API adalah kode yang terbaca; angka warisannya tetap disimpan
// (LegacyCode) supaya uji kesetaraan `S-8` dapat memanggil kueri Pega yang sama persis.
type Category string

const (
	// CategoryOutstanding — tab "Outstanding Data" (param.Note 1).
	// Laporan yang sudah menjadi klaim bernomor dan masih berjalan.
	CategoryOutstanding Category = "outstanding"

	// CategoryUnregistered — tab "Unregistered data" (param.Note 9).
	// Berkas sudah diserahkan ke klaim, tetapi belum diregistrasi.
	CategoryUnregistered Category = "belum-registrasi"

	// CategoryNotTransferred — tab "Data hasn't been transferred" (param.Note 2).
	// Laporan baru masuk dan belum diserahkan ke petugas klaim.
	CategoryNotTransferred Category = "belum-diserahkan"

	// CategoryAccepted — tab "Data has been accepted" (param.Note 3).
	// Klaimnya sudah memiliki nomor akseptasi.
	CategoryAccepted Category = "sudah-akseptasi"

	// CategoryRejected — tab "Data rejected" (param.Note 4).
	CategoryRejected Category = "ditolak"

	// CategoryMessageUnanswered — tab "Not answered communication" (param.Note 5).
	CategoryMessageUnanswered Category = "komunikasi-belum-dijawab"

	// CategoryMessageWaiting — tab "Not replied from ASM" (param.Note 7).
	CategoryMessageWaiting Category = "komunikasi-menunggu-asm"

	// CategoryMessageReplied — tab "Replied from ASM" (param.Note 8).
	CategoryMessageReplied Category = "komunikasi-dijawab-asm"

	// CategoryAll — tab "All data", cabang terakhir rantai @if.
	CategoryAll Category = "semua"
)

// categoryDefinition memuat seluruh yang membedakan satu tab dari tab lain.
//
// Ia satu tabel data, bukan rangkaian switch yang tersebar. Bentuk itu dipilih dengan
// alasan yang sama seperti alur pada modul registrasi: definisi baru dan diagram lama
// dapat dibandingkan baris per baris saat uji kesetaraan `S-8` dijalankan.
type categoryDefinition struct {
	category Category
	title    string
	legacy   string
	query    string
}

// categoryOrder adalah urutan tab persis seperti di layar Pega.
//
// Urutannya BUKAN urutan angka warisan, dan itu memang yang terjadi di sana: tab kedua
// adalah param.Note 9. Menata ulangnya menjadi terurut angka akan mengubah tempat yang
// sudah dihafal petugas.
var categoryOrder = []categoryDefinition{
	{CategoryOutstanding, "Outstanding Data", "1", "ViewTableBrowseClaimRegister"},
	{CategoryUnregistered, "Unregistered data", "9", "ViewTableBrowseClaimNotRegister"},
	{CategoryNotTransferred, "Data hasn't been transferred", "2", "ViewTableBrowseRCVInProcess"},
	{CategoryAccepted, "Data has been accepted", "3", "ViewTableBrowseRCVAcc"},
	{CategoryRejected, "Data rejected", "4", "ViewTableBrowseRCVReject"},
	{CategoryMessageUnanswered, "Not answered communication", "5", "ViewRejectKomunikasiUser"},
	{CategoryMessageWaiting, "Not replied from ASM", "7", "ViewRejectKomunikasiUser"},
	{CategoryMessageReplied, "Replied from ASM", "8", "ViewRejectKomunikasiUser"},
	{CategoryAll, "All data", "0", "ViewAllCase"},
}

// ListCategories mengembalikan seluruh tab beserta judulnya, dalam urutan layar.
//
// Salinan dibuat setiap kali: senarai paket yang dikembalikan apa adanya dapat diubah
// pemanggil, dan perubahan itu akan terbawa ke seluruh permintaan berikutnya.
func ListCategories() []CategoryInfo {
	result := make([]CategoryInfo, 0, len(categoryOrder))
	for _, d := range categoryOrder {
		result = append(result, CategoryInfo{
			Category: d.category,
			Title:    d.title,
			Message:  d.category.Message(),
		})
	}
	return result
}

// CategoryInfo adalah satu tab sebagaimana dibutuhkan layar.
type CategoryInfo struct {
	Category Category
	Title    string

	// Message menandai tab yang isinya percakapan, bukan berkas laporan. Layar
	// memakainya untuk memutuskan apakah kolom "Last message" perlu digambar.
	Message bool
}

// FindCategory mencari tab dari kodenya. Nilai kedua false bila kodenya tidak dikenal.
//
// Kode yang tidak dikenal DITOLAK, tidak diam-diam diartikan sebagai "semua". Di sistem
// lama, rantai @if menjadikan setiap nilai tak dikenal jatuh ke `ViewAllCase` — sehingga
// salah ketik pada parameter menghasilkan daftar yang tampak wajar tetapi bukan yang
// diminta. Perilaku itu tidak dibawa: ia bukan aturan bisnis, melainkan akibat bentuk
// rantai @if yang memang selalu punya cabang terakhir.
func FindCategory(code string) (Category, bool) {
	clean := Category(strings.ToLower(strings.TrimSpace(code)))
	for _, d := range categoryOrder {
		if d.category == clean {
			return d.category, true
		}
	}
	return "", false
}

// Title mengembalikan judul tab sebagaimana tertulis di layar Pega.
func (c Category) Title() string {
	if d, known := c.definition(); known {
		return d.title
	}
	return string(c)
}

// LegacyCode mengembalikan nilai `param.Note` yang dipakai `SetListRCV_Act`.
//
// Ia tidak dipakai jalur normal aplikasi ini; ia ada supaya perkakas uji kesetaraan
// `S-8` dapat menembak kueri Pega yang sama dengan tab yang sedang dibandingkan.
func (c Category) LegacyCode() string {
	if d, known := c.definition(); known {
		return d.legacy
	}
	return ""
}

// LegacyQuery mengembalikan nama RDB List yang dipakai tab ini di sistem lama.
func (c Category) LegacyQuery() string {
	if d, known := c.definition(); known {
		return d.query
	}
	return ""
}

// Message menyatakan tab ini menampilkan percakapan, bukan berkas laporan.
func (c Category) Message() bool {
	switch c {
	case CategoryMessageUnanswered, CategoryMessageWaiting, CategoryMessageReplied:
		return true
	default:
		return false
	}
}

func (c Category) definition() (categoryDefinition, bool) {
	for _, d := range categoryOrder {
		if d.category == c {
			return d, true
		}
	}
	return categoryDefinition{}, false
}

// MessageFilter adalah penyaring percakapan yang membedakan ketiga tab komunikasi.
//
// # Ketidakcocokan yang DISENGAJA dibiarkan terlihat
//
// Kueri grid dan kueri pencacah di sistem lama TIDAK sepakat tentang tab pertama.
// `SetListRCV_Act` menyusun penyaring gridnya begini:
//
//	tempQuery.ClaimNo = "1" ; tempQuery.CaseID = "and b.sender!='<saya>'"
//	tempQuery.ClaimNo = "0" ; tempQuery.CaseID = "and b.sender='<saya>'"
//	tempQuery.ClaimNo = "0" ; tempQuery.CaseID = "and b.sender!='<saya>'"
//
// sedangkan `BrowseClaimRCV_Aksep` mencacah pasangan yang berbeda untuk status "1" —
// di sana `kom.sender = <saya>`, bukan `!=`. Artinya di layar Pega hari ini, angka pada
// lencana tab "Replied from ASM" dan isi tabelnya dihitung dari aturan yang berlainan.
//
// Yang dipakai di sini adalah aturan GRID, karena itulah yang benar-benar dilihat
// pengguna saat tabnya dibuka. Selisih terhadap pencacah lama karena itu akan muncul
// pada uji kesetaraan, dan ia BUKAN cacat modul ini — ia cacat yang sudah ada, dan
// dicatat di sini supaya tidak dikira baru.
type MessageFilter struct {
	// Status adalah nilai KOMUNIKASISTATUS yang dicari: "0" belum dijawab, "1" dijawab.
	Status string

	// FromSelf true berarti hanya pesan yang DIKIRIM pemanggil; false berarti hanya
	// pesan dari pihak lain.
	FromSelf bool
}

// MessageFilterOf mengembalikan penyaring percakapan sebuah tab.
//
// Nilai kedua false untuk tab yang bukan komunikasi.
func MessageFilterOf(c Category) (MessageFilter, bool) {
	switch c {
	case CategoryMessageUnanswered:
		return MessageFilter{Status: "1", FromSelf: false}, true
	case CategoryMessageWaiting:
		return MessageFilter{Status: "0", FromSelf: true}, true
	case CategoryMessageReplied:
		return MessageFilter{Status: "0", FromSelf: false}, true
	default:
		return MessageFilter{}, false
	}
}
