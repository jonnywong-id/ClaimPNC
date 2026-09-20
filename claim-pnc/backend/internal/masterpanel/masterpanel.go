// Package masterpanel adalah inti modul Master Panel.
//
// # Apa yang dimodelkan di sini
//
// Daftar **panel bodi kendaraan berat** beserta perlakuan klaim yang berlaku atasnya:
// boleh diperbaiki atau tidak, boleh diubah kuantitasnya atau tidak, kena premium repair
// atau tidak, dan seterusnya. Setiap panel punya daftar **lokasi** — kiri, kanan, depan,
// belakang, lain-lain — beserta **sisi**-nya.
//
// Ia master pertama di aplikasi ini yang punya BARIS ANAK. Seluruh master sebelumnya
// rata: satu baris layar sama dengan satu baris tabel.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/MasterPanel_HE-Harness.xml            layar, judul "Master Panel HE"
//	Section/ListPanelHE-Section.xml               judul + tombol tambah
//	Section/BrowsePanelHE-Section.xml             3 tab: Approval · Approve · Reject
//	Section/BrowsePanelHEApproval-Section.xml     grid + form, 10 isian berlabel
//	Section/BrowsePanelHEApprove-Section.xml      tab disetujui
//	Section/BrowsePanelHEReject-Section.xml       tab ditolak
//	Section/ApprovalMasterPanelHE-Section.xml     persetujuan borongan di Inbox Manager
//	Report Definition/BrowseMasterPanel_HE_RD-RD.xml 14 kolom POOLDATA.PANEL_HE
//	RDB List/ValidationMasterPanel-SQL.xml        tolak nama panel ganda
//	RDB List/UpdatePanel_HE-SQL.xml               simpan lewat PEGA_M_PANEL_HE
//	RDB List/GetLokasiSisiPanel-SQL.xml           baris anak LOKASI_PANEL_HE
//	RDB List/GetDataSisiPanel-SQL.xml             sisi satu lokasi (dipakai modul lain)
//	RDB List/CountMasterPanelManager-SQL.xml      pencacah antrean persetujuan
//	RDB List/GetIDDokumenPanel-SQL.xml            id dokumen lampiran
//	Database/PEGA_M_PANEL_HE.prc                  pembentukan ID dan penyimpanan JSON
//	Activity/CNMUpdatePanelHE_act-Act.xml         urutan langkah simpan
//	Activity/ValidateMasterPanel-Act.xml          pesan galat nama ganda
//	Activity/SetPanelHEValue-Act.xml              pemuatan baris ke form
//	Activity/SetLokasiSisiPanel-Act.xml           daftar pilihan Lokasi dan Sisi
//	Database/m_menu_aplikasi_pnc.csv              MENU_ID 30 "Master Panel"
//
// # Satu nama yang memikul dua arti, dan di sini ia dipisah
//
// `STS_SISI` adalah kolom pada PANEL_HE dengan caption layar **"STATUS SISI"**. Nama yang
// SAMA dipakai sebagai alias kolom lain: `RDB List/GetLokasiSisiPanel-SQL.xml` menulis
// `SISI_PANEL as "STS_SISI"` — kolom tabel ANAK, artinya sama sekali berbeda.
//
// Di modul ini keduanya punya nama sendiri: Panel.SideStatus untuk kolom induk, dan
// PanelLocation.Side untuk kolom anak. Ini persis bentuk utang yang
// `03-CURRENT-ARCHITECTURE.md` §4.2 catat, dan yang sudah dicatat dua kali di
// masterbengkel.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterpanel

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// ApprovalStatus adalah posisi sebuah baris dalam alur persetujuan.
//
// Nilainya tetap "0", "1", "2" seperti kolom `APPROVAL` pada `POOLDATA.PANEL_HE`:
// tabelnya masih dibaca sistem lama selama masa paralel (ADR-0004), sehingga mengubah
// sandi nilainya akan membuat kedua sistem membaca baris yang sama secara berbeda.
//
// Ketiga sandinya terbaca dari `Section/BrowsePanelHE-Section.xml`, yang memuat tiga
// section dengan nilai penyaring `"0"`, `"1"`, dan `"2"`.
//
// Sandinya kebetulan sama dengan Master Bengkel, Master Auto Claim, dan Master Rekening.
// Keempatnya sengaja TIDAK dipakai bersama: tabelnya berbeda, dan tipe bersama membuat
// perubahan di satu master menyeret master lain.
type ApprovalStatus string

const (
	// StatusPending — diajukan, belum diputuskan. Nilai lahir setiap baris baru DAN
	// setiap baris yang disunting: `Activity/CNMUpdatePanelHE_act` menetapkan
	// `TempStsClaim.APPROVAL := "0"` tanpa syarat.
	StatusPending ApprovalStatus = "0"

	// StatusApproved — disetujui. Hanya baris berstatus ini yang dipakai sistem hilir.
	StatusApproved ApprovalStatus = "1"

	// StatusRejected — ditolak.
	StatusRejected ApprovalStatus = "2"
)

// Label mengembalikan sebutan status dalam bahasa yang dibaca pengguna.
//
// Layar lama tidak memberi nama pada ketiga tabnya — nama sectionnya `…Approval`,
// `…Approve`, dan `…Reject`. Teks di sini mengikuti kata yang sama, dengan "Waiting
// Approval" untuk yang pertama supaya terbaca sebagai antrean dan bukan sebagai
// tindakan (D-13: kata layar lama diikuti, bukan salah bacanya).
func (s ApprovalStatus) Label() string {
	switch s {
	case StatusPending:
		return "Waiting Approval"
	case StatusApproved:
		return "Approve"
	case StatusRejected:
		return "Reject"
	default:
		return ""
	}
}

// Known menyatakan status ini termasuk salah satu dari tiga yang sah.
func (s ApprovalStatus) Known() bool {
	return s == StatusPending || s == StatusApproved || s == StatusRejected
}

// Side adalah nilai kolom SISI_PANEL pada tabel anak POOLDATA.LOKASI_PANEL_HE.
//
// Ketiga sandinya adalah satu-satunya daftar pilihan modul ini yang BENAR-BENAR terbaca
// dari export, dan terbacanya dari dua tempat yang sepakat:
//
//	Activity/SetLokasiSisiPanel-Act.xml  "-" → "-", "1" → KIRI, "2" → KANAN
//	Activity/GetSisiPanel-Act.xml        @If(.NAME=="-","-",@If(.NAME=="1","KIRI","KANAN"))
//
// Perhatikan cara yang kedua membacanya: apa pun yang BUKAN "-" dan BUKAN "1" dibaca
// sebagai "KANAN". Perlakuan itu TIDAK ditiru — ia membuat nilai rusak tampil sebagai
// nilai yang sah. Di sini sandi di luar ketiganya ditolak saat disimpan, dan baris lama
// yang memuatnya tetap dibaca apa adanya.
type Side string

const (
	// SideNone — panel ini tidak dibedakan sisinya.
	SideNone Side = "-"

	// SideLeft — sisi kiri.
	SideLeft Side = "1"

	// SideRight — sisi kanan.
	SideRight Side = "2"
)

// Label mengembalikan sebutan sisi dalam bahasa yang dibaca pengguna.
func (s Side) Label() string {
	switch s {
	case SideNone:
		return "-"
	case SideLeft:
		return "KIRI"
	case SideRight:
		return "KANAN"
	default:
		return ""
	}
}

// Known menyatakan sisi ini termasuk salah satu dari tiga yang sah.
func (s Side) Known() bool {
	return s == SideNone || s == SideLeft || s == SideRight
}

// LocationOptions adalah daftar pilihan Lokasi Panel.
//
// Kelimanya dibaca dari `Activity/SetLokasiSisiPanel-Act.xml`, yang menyusunnya sebagai
// lima langkah Property-Set berturut-turut. Urutannya di sini sama dengan urutan di sana
// — itulah urutan yang dilihat petugas pada dropdown Pega hari ini.
//
// # Kenapa ia konstanta di kode, padahal D-15 melarang nilai bisnis di-hardcode
//
// Karena di sistem lama pun ia bukan master data: kelimanya ditulis satu per satu di
// dalam activity, bukan dibaca dari tabel mana pun. Mengangkatnya menjadi master berarti
// membuat tabel baru yang tidak ada padanannya — perubahan skema yang menempuh `D-63`,
// dan yang tidak diminta siapa pun.
//
// Yang TIDAK dilakukan sebagai gantinya: nilai di luar kelima ini tidak ditolak. Baris
// lama dapat memuat lokasi lain, dan menolaknya berarti baris yang hari ini sah menjadi
// tidak dapat disimpan ulang. Lihat Input.Check.
var LocationOptions = []string{"KIRI", "KANAN", "DEPAN", "BELAKANG", "LAIN-LAIN"}

// PanelLocation adalah satu baris `POOLDATA.LOKASI_PANEL_HE` — satu lokasi pada panel.
type PanelLocation struct {
	// Name adalah kolom LOKASI_PANEL.
	//
	// `RDB List/GetLokasiSisiPanel-SQL.xml` membacanya dengan alias `"NAME"`, dan
	// `Activity/SetPanelHEValue-Act.xml` menaruhnya di `TempStsClaim.LOKASI().pyNote`.
	Name string

	// Side adalah kolom SISI_PANEL. Lihat tipe Side.
	Side Side
}

// Panel adalah satu baris master panel — satu baris `POOLDATA.PANEL_HE` beserta baris
// anaknya.
//
// Keempat belas kolomnya dibaca dari `Report Definition/BrowseMasterPanel_HE_RD-RD.xml`,
// ditambah DOKUMENID yang dibaca terpisah oleh `RDB List/GetIDDokumenPanel-SQL.xml` atas
// tabel yang sama. Label yang dipakai layar dibaca dari caption
// `Section/BrowsePanelHEApproval-Section.xml`.
//
// # Kenapa hampir semuanya bertipe teks
//
// Termasuk sembilan kolom yang namanya jelas penanda — STS_REPAIR sampai EXCLUSION_C.
// Alasannya satu: **daftar nilai sahnya tidak ada di export**. Kesembilannya dirender
// `pxDropdown` atau `pxRadioButtons` dengan `pyListSource=associated`, artinya pilihannya
// datang dari rule Field Value pada propertinya — dan tidak ada satu pun direktori
// Property maupun Field Value di export (`R-16`).
//
// Menebaknya menjadi "Ya/Tidak" berarti memutuskan domain kolom yang menentukan boleh
// tidaknya sebuah panel diperbaiki. Baris lama karena itu dibaca apa adanya, dan layar
// menawarkan nilai yang SUDAH DIPAKAI baris lain sebagai saran — jawaban terbaik yang
// tersedia atas pertanyaan yang export tidak jawab. Perlakuan yang sama dipakai Master
// Bengkel untuk ketujuh penandanya.
type Panel struct {
	// ID adalah kolom ID_PANEL — kunci baris ini.
	//
	// Ia TIDAK diketik pengguna. `Database/PEGA_M_PANEL_HE.prc:21` menerbitkannya sebagai
	// kode situs ditambah nomor urut ENAM digit; lihat IDSource.
	ID string

	// Name adalah kolom NAME. Ia kunci alami: penambahan ditolak bila namanya sudah
	// dipakai baris lain (`RDB List/ValidationMasterPanel-SQL.xml`).
	Name string

	// Kesembilan penanda berikut adalah kolom STS_REPAIR, STS_EDIT_QTY,
	// STS_PREMIUM_REPAIR, STS_PECAH, STS_STICKER, STS_SISI, STS_RUSAK_PARAH, STS_AKTIF,
	// dan EXCLUSION_C.
	//
	// NILAI SAHNYA TIDAK DIKETAHUI; lihat catatan pada tipe ini. Kesembilannya bertanda
	// `pyRequired=true` di layar Pega, dan kewajiban itu ditiru — lihat Input.Check.
	RepairStatus        string // STS_REPAIR          · "STATUS REPAIR"
	EditQuantityStatus  string // STS_EDIT_QTY        · "STATUS EDIT QUANTITY"
	PremiumRepairStatus string // STS_PREMIUM_REPAIR  · "STATUS PREMIUM REPAIR"
	ShatterStatus       string // STS_PECAH           · "STATUS PECAH"
	StickerStatus       string // STS_STICKER         · "STATUS STICKER"
	SideStatus          string // STS_SISI            · "STATUS SISI" — kolom INDUK
	SevereDamageStatus  string // STS_RUSAK_PARAH     · "STATUS RUSAK PARAH"
	ActiveStatus        string // STS_AKTIF           · "STATUS AKTIF"
	ExclusionC          string // EXCLUSION_C         · "Exclusion C"

	// ApprovalMark adalah kolom STS_APPROVAL — dan ia BUKAN kolom APPROVAL.
	//
	// Keduanya ada berdampingan di `BrowseMasterPanel_HE_RD`, dan hanya APPROVAL yang
	// dipakai menyaring ketiga tab. Arti STS_APPROVAL tidak disebut di mana pun dalam
	// export: ia tidak punya caption, tidak dipakai penyaring mana pun, dan tidak
	// dibandingkan di rule mana pun.
	//
	// Ia tetap dibaca dan ditulis kembali apa adanya, supaya penyimpanan lewat layar baru
	// tidak mengosongkan kolom yang tidak dilihat siapa pun.
	ApprovalMark string

	// RejectReason adalah kolom ALASAN_TOLAK.
	//
	// `Activity/SetPanelHEValue-Act.xml` memuatnya ke form, dan layar lama menyediakan
	// isian "Catatan" (`TempStsClaim.pyNote`) berdampingan dengan tombol Reject. Ia diisi
	// pada jalur KEPUTUSAN, bukan pada jalur simpan — lihat usecase.Service.Decide.
	RejectReason string

	// DocumentID adalah kolom DOKUMENID — lampiran yang ditautkan ke baris ini.
	//
	// Diisi `Activity/CNMUpdatePanelHE_act` dari hasil `PNCSaveAttachmentToDB`. Unggah
	// lampirannya TIDAK dibawa modul ini — lihat catatan pada usecase.Service.
	DocumentID string

	// Status adalah kolom APPROVAL.
	Status ApprovalStatus

	// Location adalah baris anak pada `POOLDATA.LOKASI_PANEL_HE`.
	//
	// Nil dan slice kosong keduanya berarti panel tanpa lokasi. Itu keadaan yang SAH:
	// layar lama tidak mewajibkan satu pun baris lokasi, dan `GetLokasiSisiPanel` yang
	// mengembalikan nol baris tidak diperlakukan sebagai galat di mana pun.
	Location []PanelLocation
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// Sepuluh isian induk mengikuti caption `Section/BrowsePanelHEApproval-Section.xml`,
// ditambah daftar lokasi. Yang ADA di tabel tetapi TIDAK di sini, karena keempatnya
// diturunkan sistem:
//
//	ID_PANEL      diterbitkan saat penambahan; lihat IDSource
//	APPROVAL      selalu StatusPending pada penyimpanan lewat layar ini
//	ALASAN_TOLAK  diisi pada jalur keputusan, bukan pada jalur simpan
//	DOKUMENID     hasil unggah lampiran, jalur yang tidak dibawa modul ini
//
// STS_APPROVAL juga tidak ada di sini: ia tidak punya caption di layar mana pun, sehingga
// tidak ada isian yang dapat mengisinya. Nilainya dipertahankan dari baris yang tersimpan.
type Input struct {
	Name string

	RepairStatus        string
	EditQuantityStatus  string
	PremiumRepairStatus string
	ShatterStatus       string
	StickerStatus       string
	SideStatus          string
	SevereDamageStatus  string
	ActiveStatus        string
	ExclusionC          string

	// Location adalah seluruh baris lokasi yang dikehendaki, BUKAN hanya yang berubah.
	//
	// Bentuknya sengaja begitu: layar lama mengirim seluruh halaman klipboard sebagai
	// satu dokumen JSON (`@GCNM.GetPageJSONString()`), sehingga daftar yang dikirim
	// memang selalu daftar yang utuh. Baris yang dibuang pengguna cukup tidak ikut
	// dikirim.
	Location []PanelLocation
}

// Panjang maksimum isian teks.
//
// SELURUHNYA ASUMSI YANG DISADARI, bukan angka yang diterima dari Work Owner maupun
// dibaca dari DDL: `POOLDATA.PANEL_HE` dan `POOLDATA.LOKASI_PANEL_HE` tidak ada DDL-nya
// di export (`R-08`), dan sistem lama tidak memeriksa panjang satu pun isian.
//
// Batasnya tetap dipasang karena tanpa itu penolakan datang dari basis data sebagai
// ORA-12899 — galat teknis yang tidak menuntun pengguna ke mana pun.
//
// Angka yang sama diulang di `PanelForm.tsx`. Bila berubah, KEDUA tempat harus ikut
// berubah — utang yang disadari dari menduplikasi sebuah angka, dijaga terlihat oleh uji
// di masterpanel_test.go.
const (
	MaxNameLength     = 100
	MaxCodeLength     = 20
	MaxLocationLength = 50
	MaxReasonLength   = 250

	// MaxLocationRows membatasi banyaknya baris lokasi pada satu panel.
	//
	// Sistem lama tidak membatasinya. Batas di sini bukan aturan bisnis melainkan penjaga
	// sumber daya: setiap baris menjadi satu pernyataan INSERT di dalam satu transaksi.
	// Lima puluh jauh di atas kelima pilihan yang ditawarkan layar, sehingga ia tidak
	// akan pernah tersentuh pemakaian yang wajar.
	MaxLocationRows = 50
)

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("masterpanel: panel tidak ditemukan")

	// ErrNameTaken: NAME yang akan disisipkan sudah dipakai baris lain.
	//
	// Padanan langsung pesan `Activity/ValidateMasterPanel`:
	// "Nama tersebut telah digunakan. Silakan ganti dengan nama yang lain."
	ErrNameTaken = errors.New("masterpanel: nama panel sudah dipakai")

	// ErrUnknownStatus: status persetujuan di luar "0", "1", "2".
	ErrUnknownStatus = errors.New("masterpanel: status persetujuan tidak dikenal")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field adalah nama isian dalam bentuk yang dikenali layar, bukan nama kolom basis
	// data — layar yang menyorot isiannya memakai nilai ini.
	//
	// Pelanggaran pada baris lokasi memakai bentuk `lokasi.<indeks>.<isian>`, sehingga
	// layar dapat menyorot baris yang tepat pada daftar yang panjangnya berubah-ubah.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Ini kesetaraan perilaku, bukan selera (`P-5`). Pega pun menampilkan pesannya lewat
// `Page-Set-Messages` yang menempel pada isiannya masing-masing; yang berbeda hanyalah
// di sini seluruhnya dikirim dalam satu jawaban.
type ValidationError struct {
	Violation []Violation
}

// OneViolation membungkus satu pelanggaran menjadi ValidationError.
//
// Dipakai lapisan aplikasi untuk pemeriksaan yang menuntut pembacaan basis data —
// keunikan nama — supaya galatnya sampai ke layar dalam bentuk yang SAMA dengan
// pelanggaran isian lain, dan menempel pada isiannya.
func OneViolation(field, message string) error {
	return &ValidationError{Violation: []Violation{{Field: field, Message: message}}}
}

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "masterpanel: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Clean memangkas spasi di kedua ujung setiap isian, termasuk setiap baris lokasi.
//
// Dipisahkan dari Check supaya nilai yang tersimpan adalah nilai yang sudah dipangkas —
// bukan nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
//
// Nama lokasi ikut di-UPPERCASE. Itu bukan kerapian: kelima pilihan yang ditawarkan layar
// lama seluruhnya huruf besar (`SetLokasiSisiPanel-Act`), dan `GetDataSisiPanel` —
// pembaca tabel anak ini dari modul Grouping Sparepart — mencocokkannya dengan `=` biasa
// tanpa UPPER di kedua sisi. Satu baris bertuliskan "Kiri" karena itu tidak akan pernah
// ditemukan modul itu.
func (i Input) Clean() Input {
	trim := strings.TrimSpace

	location := make([]PanelLocation, 0, len(i.Location))
	for _, one := range i.Location {
		clean := PanelLocation{
			Name: strings.ToUpper(trim(one.Name)),
			Side: Side(trim(string(one.Side))),
		}
		// Baris yang kosong seluruhnya dibuang, tidak dilaporkan sebagai pelanggaran:
		// layar yang menambah baris lalu membiarkannya kosong adalah hal yang wajar, dan
		// menolaknya memaksa pengguna menghapus baris yang tidak pernah ia isi.
		if clean.Name == "" && clean.Side == "" {
			continue
		}
		location = append(location, clean)
	}

	return Input{
		Name:                trim(i.Name),
		RepairStatus:        trim(i.RepairStatus),
		EditQuantityStatus:  trim(i.EditQuantityStatus),
		PremiumRepairStatus: trim(i.PremiumRepairStatus),
		ShatterStatus:       trim(i.ShatterStatus),
		StickerStatus:       trim(i.StickerStatus),
		SideStatus:          trim(i.SideStatus),
		SevereDamageStatus:  trim(i.SevereDamageStatus),
		ActiveStatus:        trim(i.ActiveStatus),
		ExclusionC:          trim(i.ExclusionC),
		Location:            location,
	}
}

// Check menjalankan seluruh aturan isian dan mengembalikan SEMUA pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
//
// # Yang diwajibkan, dan dari mana asalnya
//
// KESEPULUH isian induk wajib, dan itu bukan tambahan: kesepuluhnya bertanda
// `pyRequired=true` pada `Section/BrowsePanelHEApproval-Section.xml`. Ini berbeda dari
// Master Bengkel, yang tidak punya satu pun isian wajib pada layarnya.
//
// # Yang DITAMBAHKAN terhadap sistem lama
//
//  1. **Sisi lokasi wajib salah satu dari tiga sandi yang dikenal.** `GetSisiPanel-Act`
//     membaca apa pun yang bukan "-" dan bukan "1" sebagai "KANAN", sehingga nilai rusak
//     tampil sebagai nilai yang sah. SELISIH YANG DIRENCANAKAN; baris lama tetap dibaca
//     apa adanya.
//  2. **Lokasi yang sama tidak boleh muncul dua kali pada sisi yang sama.** Sistem lama
//     tidak memeriksanya, dan dua baris kembar membuat `GetDataSisiPanel` — yang membaca
//     satu nilai saja — mengembalikan baris yang mana pun lebih dulu ditemukan.
//
// # Yang TIDAK ditambahkan, meski tampak wajar
//
// Nama lokasi TIDAK dibatasi pada kelima pilihan LocationOptions. Baris lama dapat memuat
// lokasi lain, dan menolaknya berarti baris yang hari ini sah tidak dapat disimpan ulang.
func (i Input) Check() error {
	var violation []Violation

	for _, r := range []struct {
		field, label, value string
		max                 int
	}{
		{"nama_panel", "Nama panel", i.Name, MaxNameLength},
		{"status_repair", "Status repair", i.RepairStatus, MaxCodeLength},
		{"status_edit_quantity", "Status edit quantity", i.EditQuantityStatus, MaxCodeLength},
		{"status_premium_repair", "Status premium repair", i.PremiumRepairStatus, MaxCodeLength},
		{"status_pecah", "Status pecah", i.ShatterStatus, MaxCodeLength},
		{"status_sticker", "Status sticker", i.StickerStatus, MaxCodeLength},
		{"status_sisi", "Status sisi", i.SideStatus, MaxCodeLength},
		{"status_rusak_parah", "Status rusak parah", i.SevereDamageStatus, MaxCodeLength},
		{"status_aktif", "Status aktif", i.ActiveStatus, MaxCodeLength},
		{"exclusion_c", "Exclusion C", i.ExclusionC, MaxCodeLength},
	} {
		violation = append(violation, checkRequired(r.field, r.label, r.value, r.max)...)
	}

	violation = append(violation, i.checkLocation()...)

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// checkLocation memeriksa seluruh baris lokasi.
//
// Nama isiannya berbentuk `lokasi.<indeks>.<isian>` supaya layar dapat menyorot baris
// yang tepat. Indeksnya dihitung dari nol, mengikuti posisi pada daftar yang dikirim —
// bukan nomor urut yang terlihat pengguna, karena daftar yang sedang disunting dapat
// bergeser di antara pengiriman dan jawaban.
func (i Input) checkLocation() []Violation {
	if len(i.Location) > MaxLocationRows {
		return []Violation{{
			Field: "lokasi",
			Message: fmt.Sprintf("Lokasi panel paling banyak %d baris dalam satu panel.",
				MaxLocationRows),
		}}
	}

	var violation []Violation
	seen := make(map[string]int, len(i.Location))

	for index, one := range i.Location {
		prefix := "lokasi." + strconv.Itoa(index) + "."

		switch {
		case one.Name == "":
			violation = append(violation, Violation{
				Field:   prefix + "lokasi_panel",
				Message: "Lokasi panel wajib dipilih.",
			})
		case len(one.Name) > MaxLocationLength:
			violation = append(violation, Violation{
				Field: prefix + "lokasi_panel",
				Message: fmt.Sprintf("Lokasi panel paling panjang %d karakter.",
					MaxLocationLength),
			})
		}

		if !one.Side.Known() {
			violation = append(violation, Violation{
				Field:   prefix + "sisi_panel",
				Message: `Sisi panel hanya boleh "-", KIRI, atau KANAN.`,
			})
		}

		if one.Name == "" {
			continue
		}
		key := one.Name + "\x00" + string(one.Side)
		if first, already := seen[key]; already {
			violation = append(violation, Violation{
				Field: prefix + "lokasi_panel",
				Message: fmt.Sprintf("Lokasi %s sisi %s sudah ada di baris %d.",
					one.Name, one.Side.Label(), first+1),
			})
			continue
		}
		seen[key] = index
	}

	return violation
}

// checkRequired memeriksa satu isian wajib beserta panjangnya.
func checkRequired(field, label, value string, max int) []Violation {
	if value == "" {
		return []Violation{{Field: field, Message: label + " wajib diisi."}}
	}
	return checkLength(field, label, value, max)
}

// checkLength memeriksa panjang satu isian yang boleh kosong.
func checkLength(field, label, value string, max int) []Violation {
	if len(value) > max {
		return []Violation{{
			Field:   field,
			Message: fmt.Sprintf("%s paling panjang %d karakter.", label, max),
		}}
	}
	return nil
}

// Filter menyaring daftar yang dibaca layar.
//
// Ia cerminan ketiga tab `Section/BrowsePanelHE-Section.xml`, yang ketiganya membaca
// `BrowseMasterPanel_HE_RD` yang sama dan hanya berbeda pada nilai APPROVAL-nya.
//
// Keyword DITAMBAHKAN terhadap sistem lama. Alasannya bukan kelengkapan: `pyMaxRecords`
// pada report definition-nya bernilai **500** (`BrowseMasterPanel_HE_RD-RD.xml`),
// sehingga daftar Pega memang terpotong di 500 baris tanpa satu pun cara mempersempitnya
// dari layar. Pencarian di sini yang menggantikan pemotongan itu.
type Filter struct {
	// Status wajib salah satu dari tiga yang dikenal.
	Status ApprovalStatus

	// Keyword mempersempit daftar pada nama panel. Kosong berarti tanpa penyaring.
	Keyword string
}

// IDSource menerbitkan ID_PANEL baru.
//
// Ia seam tersendiri, bukan method pada Repo, karena isinya bukan urusan master panel
// melainkan urusan **penomoran**: kode situs dan pola yang sama dipakai keluarga
// procedure `PEGA_M_*` lain pada basis data yang sama.
//
// # Bentuknya, dibaca dari Database/PEGA_M_PANEL_HE.prc:12,21
//
//	SELECT ID INTO id_site from M_SITE_DATABASE WHERE CURRENT_SITE = '1';
//	id_panel_he_ins := id_site || lpad(to_Char(PANEL_HE_SEQ.nextval),6,'0');
//
// Ditiru persis, termasuk pembandingnya yang berupa TEKS '1' dan bukan angka.
//
// PERHATIKAN LEBARNYA: **enam** digit, bukan sepuluh seperti Master Bengkel. Kedua
// procedure ditulis orang yang sama dengan pola yang sama, dan lebarnya tetap berbeda.
// Menyeragamkannya akan menerbitkan ID yang tidak sebentuk dengan ID yang sudah ada.
type IDSource interface {
	// NextID mengembalikan ID_PANEL berikutnya.
	NextID(ctx context.Context) (string, error)
}

// ComposeID merangkai ID_PANEL dari kode situs dan nomor urut.
//
// Ia berada di paket domain, bukan di adapter, karena BENTUK KUNCI adalah aturan domain:
// ia yang menentukan bagaimana sebuah panel dikenali, dan ia harus sama persis pada
// adapter SQL maupun adapter memori. Satu tempat, satu bentuk — dan satu uji yang
// menjaganya.
//
// Nomor urut yang LEBIH PANJANG dari lebar yang diminta tidak dipotong. `LPAD` Oracle
// memotongnya dari kanan, sehingga urutan ke-1.000.000 akan menghasilkan kunci yang
// bertabrakan dengan urutan lain — diam-diam. Pada lebar enam digit, batas itu jauh lebih
// dekat daripada pada Master Bengkel: sejuta panel memang tidak masuk akal, tetapi
// sequence yang dipakai ulang atau direset tinggi tidak menuntut sejuta baris untuk
// sampai ke sana. Di sini ia dibiarkan tumbuh: kuncinya menjadi lebih panjang, dan itu
// terlihat, alih-alih salah tanpa terlihat.
func ComposeID(site string, sequence int64, width int) string {
	number := strconv.FormatInt(sequence, 10)
	if pad := width - len(number); pad > 0 {
		number = strings.Repeat("0", pad) + number
	}
	return strings.TrimSpace(site) + number
}

// Repo adalah seam ke penyimpanan master panel SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat
// kueri (ADR-0030 Opsi 1).
type Repo interface {
	// List mengembalikan baris yang cocok dengan penyaring, LENGKAP dengan lokasinya.
	//
	// Lokasi ikut dibaca karena layar menampilkannya sebagai ringkasan pada setiap baris
	// daftar. Membacanya per baris saat form dibuka akan menghasilkan satu perjalanan
	// basis data per panel — pola N+1 yang `15-NFR-PERFORMANCE-SCALABILITY.md` §3.1
	// letakkan di peringkat ketiga hambatan nyata.
	List(ctx context.Context, filter Filter) ([]Panel, error)

	// Get mengembalikan satu baris beserta lokasinya; ErrNotFound bila tidak ada.
	Get(ctx context.Context, id string) (Panel, error)

	// FindByName mencari baris menurut NAME-nya; ErrNotFound bila tidak ada.
	//
	// Padanan `RDB List/ValidationMasterPanel-SQL.xml`, yang mencocokkan
	// `upper(trim(name))` — perlakuan yang ditiru apa adanya.
	//
	// Lokasinya TIDAK ikut dibaca: pemanggilnya hanya perlu tahu barisnya ada dan apa
	// kuncinya.
	FindByName(ctx context.Context, name string) (Panel, error)

	// Insert menyisipkan baris baru beserta seluruh lokasinya.
	//
	// Pemeriksaan nama ganda berada DI DALAM operasi repo, bukan dipecah menjadi "cek"
	// lalu "sisip" di lapisan aplikasi. Sistem lama memecahnya — `ValidateMasterPanel`
	// dipanggil dari layar, `CNMUpdatePanelHE_act` menyimpan jauh sesudahnya — dan jarak
	// di antara keduanya adalah lubang balapan yang tidak dijaga apa pun.
	//
	// KETERBATASAN YANG DISADARI. Tanpa constraint unik pada NAME, lubang itu hanya
	// dipersempit, tidak ditutup. Penutupnya adalah constraint di basis data, dan itu
	// menunggu DDL (`R-08`) beserta prosedur perubahan skema (`D-63`).
	Insert(ctx context.Context, p Panel) error

	// Update menyimpan perubahan pada baris yang sudah ada beserta seluruh lokasinya;
	// ErrNotFound bila barisnya hilang di antara pemuatan layar dan penyimpanan.
	Update(ctx context.Context, p Panel) error

	// SetStatus menetapkan APPROVAL sejumlah baris sekaligus, beserta alasannya.
	//
	// Ia terpisah dari Update karena di sistem lama pun terpisah, dan bentuknya memang
	// borongan: `Activity/SetApprovalAllMaster` menelusuri baris yang `.pySelected=="true"`
	// lalu menetapkan `APPROVAL := Param.approval` pada masing-masing — tanpa menyentuh
	// satu pun kolom lain selain alasannya.
	//
	// Yang dikembalikan adalah jumlah baris yang benar-benar berubah, supaya pemanggil
	// dapat membedakan "tidak ada yang dipilih" dari "yang dipilih sudah tidak ada".
	SetStatus(ctx context.Context, id []string, status ApprovalStatus, reason string) (int, error)
}

// RepoSelector memilih Store milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Store, error)

// Store menyatukan kedua seam yang dipakai layanan modul ini.
//
// Keduanya tetap DIDEKLARASIKAN terpisah — Repo untuk tabel master beserta anaknya,
// IDSource untuk penomoran — karena keduanya menjawab pertanyaan yang berbeda dan dapat
// berubah sendiri-sendiri. Yang disatukan hanyalah CARA MEMILIHNYA: keduanya selalu
// berasal dari koneksi entitas yang sama.
//
// Berbeda dari Master Bengkel, modul ini TIDAK punya LookupRepo: kelima pilihan Lokasi
// dan ketiga pilihan Sisi ditanam di dalam activity Pega, bukan dibaca dari tabel acuan
// mana pun. Lihat LocationOptions.
type Store interface {
	Repo
	IDSource
}
