// Package masterxol adalah inti modul Master XOL (`F-4`, butir menu `MENU_ID 19`).
//
// # Apa yang dikerjakan layar ini
//
// Mencatat **struktur treaty Excess of Loss** per tahun: satu induk berisi tahun, kurs
// IDR, dan jenis XOL-nya, lalu tiga daftar anak di bawahnya — grup bisnis yang dicakup,
// lapisan (layer) beserta limit dan excess-nya, dan pembagian share ke para reasuradur
// pada tiap lapisan.
//
// # Bentuknya BERTINGKAT EMPAT, dan itu dibaca dari basis data, bukan diandaikan
//
//	POOLDATA.MST_XOL_PNC        induk       ID PK
//	POOLDATA.MST_XOL_BUSINESS   anak        ID  → grup bisnis yang dicakup
//	POOLDATA.MST_XOL_LAYER      anak        ID  → lapisan, IDLAYER PK
//	POOLDATA.MST_XOL_REAS       cucu        IDLAYER → share reasuradur
//
// Seluruh bentuk kolom diverifikasi langsung ke ALL_TAB_COLUMNS portal ASM pada
// 2026-09-20, bukan disimpulkan dari nama parameter procedure.
//
// # Asal setiap aturan di berkas ini
//
//	Harness/DetailMasterXOL-Harness.xml             kerangka layar
//	Section/DetailXOL-Section.xml                   judul, tombol Tambah dan Refresh
//	Section/DetailXOL_sec-Section.xml               form induk, grid Bisnis, grid Layer
//	Section/InputDetailPanelReasGenerated-Section.xml  grid Reas: ID, Reasuransi, Share (%)
//	Activity/InsertUpdateMasterXOL-Act.xml          urutan simpan beserta validasinya
//	Activity/UpdateMasterXOL-Act.xml                pemuatan satu induk untuk diubah
//	Activity/DeleteFromTabelMst-Act.xml.xml         empat jenis hapus
//	Activity/ShowDetailGroupBisnisXol_Act-Act.xml   penyaring grup bisnis menurut Type XOL
//	Activity/UpdateStatusMasterKomitexol-Act.xml    pengajuan ke komite
//	Activity/SendDataMasterXOLToKomites-Act.xml     pemberitahuan ke komite
//	RDB List/GetDataMasterXOL-SQL.xml               grid induk
//	RDB List/GetDataBisnisXOL-SQL.xml               grid bisnis
//	RDB List/GetDataLayerXOL-SQL.xml                grid layer
//	RDB List/GetDataReasXOL-SQL.xml                 grid reas
//	RDB List/SetMasterXOL-SQL.xml                   pemetaan isian ke parameter procedure
//	RDB List/GetDataBisnisXol_Sql-SQL.xml           pilihan grup bisnis
//	Database/INSERT_UPDATE_MST_XOL.prc              source procedure-nya
//	Database/GET_GROUPBUSINESS_XOL.fnc              perangkai daftar grup bisnis
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterxol

import (
	"strconv"
	"strings"
	"unicode/utf8"
)

// Batas panjang isian teks.
//
// Seluruhnya DIBACA dari ALL_TAB_COLUMNS portal ASM pada 2026-09-20, bukan dikarang —
// itulah sebabnya angkanya tidak seragam:
//
//	MST_XOL_PNC.NAMA              VARCHAR2(50)
//	MST_XOL_PNC.TAHUN             VARCHAR2(10)
//	MST_XOL_PNC.TYPEXOL           VARCHAR2(10)
//	MST_XOL_PNC.REMARKPIC         VARCHAR2(1000)
//	MST_XOL_PNC.REMARKKOMITE      VARCHAR2(1000)
//	MST_XOL_BUSINESS.IDBUSINESS   VARCHAR2(10)
//	MST_XOL_BUSINESS.GROUPBUSINESS VARCHAR2(50)
//	MST_XOL_LAYER.NAMA            VARCHAR2(50)
//	MST_XOL_REAS.IDREAS           VARCHAR2(20)
//	MST_XOL_REAS.NAMA             VARCHAR2(100)
//
// Menolaknya di sini membuat pengguna melihat pesan yang menyebut isian mana yang
// kepanjangan, bukan ORA-12899 yang tidak menyebutnya.
//
// Angka yang sama diulang di frontend supaya pengguna tahu sebelum mengirim; server
// tetap yang berwenang. Bila salah satu berubah, KEDUA tempat wajib ikut berubah.
const (
	MaxNameLength          = 50
	MaxYearLength          = 10
	MaxTypeLength          = 10
	MaxRemarkLength        = 1000
	MaxBusinessIDLength    = 10
	MaxBusinessNameLength  = 50
	MaxLayerNameLength     = 50
	MaxReinsurerIDLength   = 20
	MaxReinsurerNameLength = 100
)

// Amount adalah nilai angka dalam SATUAN UTUH — tanpa pecahan.
//
// # Kenapa bilangan bulat, bukan pecahan
//
// `docs/Steering/09-DATABASE-STRATEGY.md` §5 melarang float untuk nilai uang tanpa
// perkecualian. Yang tersisa adalah bilangan bulat atau titik-tetap, dan basis datanya
// sendiri yang memutuskan mana yang benar:
//
//   - Keempat kolom angka — KURSVALUE, LIMIT, EXCESS, CONVERT_LIMIT — bertipe NUMBER
//     tanpa presisi dan tanpa skala, jadi basis data tidak membatasi apa pun.
//   - Kueri `... <> TRUNC(...)` atas keempatnya pada 2026-09-20 mengembalikan **NOL
//     baris**. Tidak ada satu pun nilai pecahan di seluruh 8 induk dan 18 lapisan.
//   - Nilai terbesar yang ada adalah CONVERT_LIMIT 67.500.000.000 — muat jauh di dalam
//     int64.
//
// SATUANNYA BERBEDA per field, dan itu wajib diperhatikan saat membacanya:
//
//	ExchangeRate    RUPIAH per satu dolar
//	Limit, Excess   DOLAR  — layar lama melabelinya "Limit (USD)" dan "Excess (USD)"
//	ConvertedLimit  RUPIAH — hasil Limit × ExchangeRate
//
// KONSEKUENSI YANG DISADARI: isian bernilai pecahan DITOLAK, sementara layar Pega akan
// menerimanya. Itu selisih perilaku yang disengaja, dipilih karena menerima pecahan
// diam-diam lalu memotongnya jauh lebih berbahaya daripada menolaknya dengan pesan.
type Amount int64

// Tidak ada pembaca angka dari TEKS di modul ini, dan itu keputusan yang diambil setelah
// mencobanya.
//
// Versi pertama berkas ini punya `ParseAmount` yang membuang titik dan koma sebagai
// pemisah ribuan, meniru modul lain. Uji membuktikannya AMBIGU: setelah titik dibuang,
// "13.500" (tiga belas ribu lima ratus) dan "13500.75" (dengan pecahan) menjadi angka
// yang sama — sehingga isian pecahan yang seharusnya ditolak justru diterima sebagai
// 1.350.075.
//
// Ia dibuang, bukan ditambal, karena memang tidak dibutuhkan: kurs, limit, excess, dan
// share seluruhnya tiba sebagai ANGKA di dalam JSON, bukan sebagai teks. Badan permintaan
// yang mengirimkannya sebagai teks ditolak decoder dengan galat yang jelas.

// ConvertedLimit menghitung isi kolom CONVERT_LIMIT.
//
// # Aturannya, dan buktinya
//
// `Activity/InsertUpdateMasterXOL-Act.xml` mengisi parameter `TCONVERT` dengan
// `@toDecimal(.ClaimAmount) * local.kurs` — yaitu **Limit lapisan dikali kurs induk**,
// bukan Excess, dan bukan jumlah keduanya.
//
// Diperiksa ke seluruh 18 lapisan di portal ASM pada 2026-09-20: kueri yang mencari
// baris dengan `CONVERT_LIMIT <> LIMIT * KURSVALUE` mengembalikan **satu** baris, dan
// baris itu adalah lapisan `10017` yang LIMIT-nya NULL — tersimpan 0. Aturannya karena
// itu terbukti berlaku pada setiap baris yang limitnya terisi.
//
// Nilainya DIHITUNG server, tidak pernah diterima dari pemanggil: membiarkan klien
// mengirimnya berarti membiarkan limit rupiah dan limit dolar berbeda diam-diam.
func ConvertedLimit(limit, exchangeRate Amount) Amount {
	return limit * exchangeRate
}

// Share adalah persentase bagian seorang reasuradur pada satu lapisan, dalam PERSEN UTUH.
//
// Bulat, dengan alasan yang sama seperti Amount: kueri `PERCENTSHARE <> TRUNC(...)` atas
// seluruh 42 baris mengembalikan nol. Dan `Activity/InsertUpdateMasterXOL-Act.xml`
// membandingkannya dengan `local.totalshare==100` — perbandingan bilangan bulat, tanpa
// toleransi.
//
// Ia sengaja TIDAK memakai toleransi empat desimal seperti spreading klaim (`D-51`).
// Toleransi itu ditetapkan untuk aturan spreading di modul `B-4`, dan memakainya di sini
// akan memperkenalkan aturan yang tidak pernah ada di layar ini.
type Share int64

// TotalShare menjumlahkan share seluruh reasuradur pada satu lapisan.
func (l Layer) TotalShare() Share {
	var total Share
	for _, r := range l.Reinsurer {
		total += r.Share
	}
	return total
}

// FullShare adalah total share yang dianggap lengkap: 100 persen.
const FullShare Share = 100

// Type adalah kode Type XOL — kolom TYPEXOL.
//
// # Kenapa artinya diturunkan, bukan dibaca
//
// Daftar pilihannya diisi activity `GetTypeofxolclaim`, dan activity itu **tidak ada di
// export** (`R-16`). Tidak ada pula tabel master untuk itu: seluruh objek POOLDATA
// bernama XOL sudah diperiksa pada 2026-09-20, dan yang ada hanyalah keempat tabel data
// di atas.
//
// Yang dapat dibaca hanyalah AKIBATNYA. `Activity/ShowDetailGroupBisnisXol_Act-Act.xml`
// memakai kode ini untuk memilih penyaring grup bisnis, dan ketiga cabangnya bersyarat
// `TempXOL.CNPSupportDoc == "1" | "2" | "3"`. Label di bawah karena itu diturunkan dari
// isi penyaringnya, dan ia menunggu konfirmasi Tim Pega.
//
// Nilai NULL juga ada di produksi — dua dari delapan induk. Ia dipertahankan apa adanya
// sebagai TypeUnknown, bukan dipaksa menjadi salah satu dari ketiganya.
type Type string

// Ketiga kode Type XOL yang benar-benar ada di produksi, ditambah keadaan kosong.
const (
	TypeUnknown  Type = ""
	TypeProperty Type = "1"
	TypeAccident Type = "2"
	TypeMarine   Type = "3"
)

// TypeLabel mengembalikan label yang ditampilkan untuk sebuah kode Type XOL.
//
// Label diturunkan dari isi penyaring grup bisnis di
// `Activity/ShowDetailGroupBisnisXol_Act-Act.xml`, satu-satunya bukti yang tersisa
// setelah `GetTypeofxolclaim` hilang dari export.
func TypeLabel(t Type) string {
	switch t {
	case TypeProperty:
		return "Property / Motor / Engineering"
	case TypeAccident:
		return "PA / GA"
	case TypeMarine:
		return "Marine / Heavy Equipment"
	default:
		return ""
	}
}

// KnownType adalah ketiga kode yang dikenali, berurutan seperti di layar.
func KnownType() []Type { return []Type{TypeProperty, TypeAccident, TypeMarine} }

// CommitteeStatus adalah kolom STSKOMITE — keadaan persetujuan komite atas satu induk.
//
// Nilainya dibaca dari produksi pada 2026-09-20: `'0'` pada lima baris, `'1'` pada dua,
// dan NULL pada satu. Artinya terbaca dari kedua pemakainya:
//
//   - `RDB List/GetDataMasterXOLForKomiteApprove-SQL.xml` menyaring `STSKOMITE='0'`
//     untuk menyusun antrean persetujuan komite — jadi `'0'` berarti MENUNGGU.
//   - `Activity/UpdateStatusMasterKomitexol-Act.xml` menyetelnya ke `'0'` saat PIC
//     mengajukan, lalu ke nilai keputusan komite saat komite menjawab.
//
// NULL berarti induk itu belum pernah diajukan sama sekali.
type CommitteeStatus string

const (
	CommitteeNotSubmitted CommitteeStatus = ""
	CommitteePending      CommitteeStatus = "0"
	CommitteeApproved     CommitteeStatus = "1"
)

// Reinsurer adalah satu reasuradur beserta bagiannya pada sebuah lapisan —
// satu baris POOLDATA.MST_XOL_REAS.
//
// # Kenapa ID dan nama keduanya diketik, bukan dipilih dari master
//
// Section `InputPanelReas` yang memuat pickernya **hilang dari export** (`R-16`). Yang
// tersisa, `Section/InputDetailPanelReasGenerated-Section.xml`, menampilkan keduanya
// sebagai isian teks biasa.
//
// Pencarian sumbernya di basis data pada 2026-09-20 pun tidak menemukan master yang
// cocok: dari 18 nama yang dipakai, hanya 5 ada di POOLDATA.T_REINSURER dan 8 di
// POOLDATA.TREATYREINSURER; "SWISS RE" tidak ada di keduanya. Menebak salah satunya akan
// membuat 13 nama yang sudah dipakai menjadi tidak dapat dipilih lagi.
//
// Karena itu keduanya diterima sebagai teks, persis seperti bukti yang ada, dan dicatat
// terbuka sampai Tim Pega mengirim `InputPanelReas`.
type Reinsurer struct {
	// ID adalah kolom IDREAS.
	ID string

	// Name adalah kolom NAMA.
	Name string

	// Share adalah kolom PERCENTSHARE.
	Share Share
}

// Layer adalah satu lapisan treaty — satu baris POOLDATA.MST_XOL_LAYER.
type Layer struct {
	// ID adalah kolom IDLAYER, kunci utama tabel. Diterbitkan penyimpanan saat lapisan
	// pertama kali disimpan, tidak pernah datang dari pemanggil.
	ID string

	// Name adalah kolom NAMA — label lapisan seperti "Layer 1" atau "Sub Layer".
	Name string

	// Limit adalah kolom LIMIT, dalam DOLAR. Layar lama melabelinya "Limit (USD)".
	Limit Amount

	// Excess adalah kolom EXCESS, dalam DOLAR. Layar lama melabelinya "Excess (USD)".
	Excess Amount

	// ConvertedLimit adalah kolom CONVERT_LIMIT, dalam RUPIAH. TIDAK diterima dari
	// pemanggil — dihitung server dengan ConvertedLimit(), supaya rumusnya hidup di satu
	// tempat.
	ConvertedLimit Amount

	// Reinsurer adalah pembagian share pada lapisan ini.
	Reinsurer []Reinsurer
}

// Business adalah satu grup bisnis yang dicakup sebuah induk XOL —
// satu baris POOLDATA.MST_XOL_BUSINESS.
type Business struct {
	// ID adalah kolom IDBUSINESS, menunjuk POOLDATA.BUSINESSGROUP.ID.
	//
	// Ia BOLEH kosong, dan itu bukan kelonggaran yang dikarang: dua baris produksi milik
	// induk 10009 menyimpan NULL di kolom ini. Penyebabnya terbaca di
	// `Activity/ShowDetailGroupBisnisXol_Act-Act.xml`, yang menambahkan baris
	// "TREATY INWARD" ke daftar pilihan tanpa memberinya ID.
	ID string

	// Name adalah kolom GROUPBUSINESS.
	Name string
}

// Master adalah satu induk XOL — satu baris POOLDATA.MST_XOL_PNC beserta anaknya.
//
// Nama field mengikuti ARTINYA (`D-19`, `D-80`), bukan nama properti Pega. Itu bukan
// kerapian: properti di layar lama dipakai untuk hal yang sama sekali berbeda dari
// namanya, dan membawa namanya berarti membawa kekeliruannya. Tiga yang paling
// menyesatkan, seluruhnya terbukti di `RDB List/SetMasterXOL-SQL.xml`:
//
//	TempXOL.UserName        → NAMA         ia NAMA MASTER, bukan nama pengguna
//	TempXOL.Amount          → KURSVALUE    ia KURS, bukan nilai klaim
//	TempXOL.AreaClaimId     → REMARKPIC    ia CATATAN PIC, bukan id area klaim
//	TempXOL.CNPSupportDoc   → TYPEXOL      ia JENIS XOL, bukan dokumen pendukung
//
// Pemetaan ke kolomnya ada di repo/sqlstore, satu tempat saja.
type Master struct {
	// ID adalah kolom ID, kunci utama. Diterbitkan penyimpanan, tidak pernah datang dari
	// pemanggil.
	ID string

	// Name adalah kolom NAMA.
	Name string

	// Year adalah kolom TAHUN — tahun treaty.
	Year string

	// ExchangeRate adalah kolom KURSVALUE, rupiah per satu dolar. Dipakai mengubah Limit
	// tiap lapisan menjadi ConvertedLimit.
	ExchangeRate Amount

	// Type adalah kolom TYPEXOL.
	Type Type

	// RemarkPIC adalah kolom REMARKPIC — catatan PIC saat mengajukan ke komite.
	RemarkPIC string

	// PIC adalah kolom PIC, identitas petugas yang mengajukan. Diisi server dari sesi,
	// tidak pernah dari badan permintaan.
	PIC string

	// CommitteeStatus adalah kolom STSKOMITE.
	CommitteeStatus CommitteeStatus

	// Committee adalah kolom KOMITE, identitas anggota komite yang menjawab.
	Committee string

	// RemarkCommittee adalah kolom REMARKKOMITE — catatan komite. Hanya dibaca di modul
	// ini; yang mengisinya adalah layar persetujuan komite (`MENU_ID 53`), yang belum
	// dibangun.
	RemarkCommittee string

	// Business adalah grup bisnis yang dicakup induk ini.
	Business []Business

	// Layer adalah lapisan treaty beserta pembagian share-nya.
	Layer []Layer
}

// Clean mengembalikan salinan dengan spasi tepi seluruh isian teks dibuang, dan
// ConvertedLimit tiap lapisan dihitung ulang.
//
// Perapian ini nyata gunanya: isian layar lama tidak pernah dirapikan, sehingga nama
// grup bisnis yang sama dapat tersimpan dengan dan tanpa spasi tepi lalu gagal dicocokkan
// saat pemeriksaan ganda dijalankan.
//
// ConvertedLimit sengaja dihitung DI SINI, bukan di penyimpanan: dengan begitu nilai yang
// divalidasi, yang disimpan, dan yang dikembalikan ke layar dijamin sama.
func (m Master) Clean() Master {
	m.ID = strings.TrimSpace(m.ID)
	m.Name = strings.TrimSpace(m.Name)
	m.Year = strings.TrimSpace(m.Year)
	m.Type = Type(strings.TrimSpace(string(m.Type)))
	m.RemarkPIC = strings.TrimSpace(m.RemarkPIC)
	m.PIC = strings.TrimSpace(m.PIC)
	m.CommitteeStatus = CommitteeStatus(strings.TrimSpace(string(m.CommitteeStatus)))
	m.Committee = strings.TrimSpace(m.Committee)
	m.RemarkCommittee = strings.TrimSpace(m.RemarkCommittee)

	business := make([]Business, 0, len(m.Business))
	for _, b := range m.Business {
		business = append(business, Business{
			ID:   strings.TrimSpace(b.ID),
			Name: strings.TrimSpace(b.Name),
		})
	}
	m.Business = business

	layer := make([]Layer, 0, len(m.Layer))
	for _, l := range m.Layer {
		reinsurer := make([]Reinsurer, 0, len(l.Reinsurer))
		for _, r := range l.Reinsurer {
			reinsurer = append(reinsurer, Reinsurer{
				ID:    strings.TrimSpace(r.ID),
				Name:  strings.TrimSpace(r.Name),
				Share: r.Share,
			})
		}
		layer = append(layer, Layer{
			ID:             strings.TrimSpace(l.ID),
			Name:           strings.TrimSpace(l.Name),
			Limit:          l.Limit,
			Excess:         l.Excess,
			ConvertedLimit: ConvertedLimit(l.Limit, m.ExchangeRate),
			Reinsurer:      reinsurer,
		})
	}
	m.Layer = layer

	return m
}

// CheckMaster mengumpulkan SELURUH pelanggaran isian, bukan berhenti pada yang pertama.
//
// Mengumpulkan semuanya adalah kesetaraan perilaku, bukan selera: layar lama menampilkan
// pesannya sekaligus, dan mengembalikannya satu per satu akan membuat pengguna menekan
// Simpan berkali-kali untuk menemukan kesalahan berikutnya.
//
// # Apa yang DIBLOKIR, dan kenapa hanya itu
//
// Layar Pega nyaris tidak memblokir apa pun: keempat isian induk bertanda
// `pyRequired=false`, dan kolomnya pun nullable — dua dari delapan induk produksi
// menyimpan TYPEXOL NULL. Menuntut isian yang layar lama tidak menuntut akan menolak
// data yang hari ini sah.
//
// Yang diblokir karena itu hanya dua golongan, dan keduanya punya alasan yang dapat
// diperiksa:
//
//  1. **Panjang teks** — bukan aturan bisnis melainkan penjaga terhadap penolakan basis
//     data. Tanpa ini, isian yang melebihi lebar kolom sampai ke pengguna sebagai galat
//     500 beserta nomor galat Oracle yang tidak menyebut isian mana.
//  2. **Angka negatif** — DITOLAK meski layar lama menerimanya. Kurs, limit, excess, dan
//     share yang negatif tidak punya arti bisnis apa pun, dan akibatnya baru terlihat
//     jauh kemudian pada perhitungan PLA/DLA. Ini selisih terencana, dicatat supaya
//     terbaca sebagai keputusan.
//
// Total share yang tidak 100% TIDAK diblokir — lihat ShareWarning.
func CheckMaster(m Master) []Violation {
	clean := m.Clean()
	var violation []Violation

	add := func(field, message string) {
		violation = append(violation, Violation{Field: field, Message: message})
	}

	if tooLong(clean.Name, MaxNameLength) {
		add(FieldName, lengthMessage("Nama master XOL", MaxNameLength))
	}
	if tooLong(clean.Year, MaxYearLength) {
		add(FieldYear, lengthMessage("Tahun", MaxYearLength))
	}
	if tooLong(string(clean.Type), MaxTypeLength) {
		add(FieldType, lengthMessage("Type XOL", MaxTypeLength))
	}
	if tooLong(clean.RemarkPIC, MaxRemarkLength) {
		add(FieldRemarkPIC, lengthMessage("Remark PIC", MaxRemarkLength))
	}
	if clean.ExchangeRate < 0 {
		add(FieldExchangeRate, "Kurs IDR tidak boleh negatif.")
	}

	violation = append(violation, checkBusiness(clean.Business)...)
	violation = append(violation, checkLayer(clean.Layer)...)

	return violation
}

func checkBusiness(list []Business) []Violation {
	var violation []Violation
	for index, b := range list {
		position := strconv.Itoa(index + 1)
		if tooLong(b.ID, MaxBusinessIDLength) {
			violation = append(violation, Violation{
				Field:   FieldBusiness,
				Message: "Baris " + position + " pada daftar bisnis: " + lengthMessage("ID bisnis", MaxBusinessIDLength),
			})
		}
		if tooLong(b.Name, MaxBusinessNameLength) {
			violation = append(violation, Violation{
				Field:   FieldBusiness,
				Message: "Baris " + position + " pada daftar bisnis: " + lengthMessage("Nama bisnis", MaxBusinessNameLength),
			})
		}
		if b.ID == "" && b.Name == "" {
			violation = append(violation, Violation{
				Field:   FieldBusiness,
				Message: "Baris " + position + " pada daftar bisnis kosong seluruhnya.",
			})
		}
	}
	return violation
}

func checkLayer(list []Layer) []Violation {
	var violation []Violation
	for index, l := range list {
		position := strconv.Itoa(index + 1)
		prefix := "Baris " + position + " pada daftar layer: "

		if tooLong(l.Name, MaxLayerNameLength) {
			violation = append(violation, Violation{
				Field:   FieldLayer,
				Message: prefix + lengthMessage("Nama layer", MaxLayerNameLength),
			})
		}
		if l.Limit < 0 {
			violation = append(violation, Violation{Field: FieldLayer, Message: prefix + "Limit tidak boleh negatif."})
		}
		if l.Excess < 0 {
			violation = append(violation, Violation{Field: FieldLayer, Message: prefix + "Excess tidak boleh negatif."})
		}

		for reasIndex, r := range l.Reinsurer {
			reasPosition := strconv.Itoa(reasIndex + 1)
			reasPrefix := prefix + "reas baris " + reasPosition + ": "

			if tooLong(r.ID, MaxReinsurerIDLength) {
				violation = append(violation, Violation{
					Field:   FieldReinsurer,
					Message: reasPrefix + lengthMessage("ID reas", MaxReinsurerIDLength),
				})
			}
			if tooLong(r.Name, MaxReinsurerNameLength) {
				violation = append(violation, Violation{
					Field:   FieldReinsurer,
					Message: reasPrefix + lengthMessage("Nama reas", MaxReinsurerNameLength),
				})
			}
			if r.Share < 0 {
				violation = append(violation, Violation{
					Field:   FieldReinsurer,
					Message: reasPrefix + "Share tidak boleh negatif.",
				})
			}
			if r.ID == "" && r.Name == "" {
				violation = append(violation, Violation{
					Field:   FieldReinsurer,
					Message: reasPrefix + "baris kosong seluruhnya.",
				})
			}
		}
	}
	return violation
}

// ShareWarning menyusun peringatan untuk setiap lapisan yang total share-nya bukan 100%.
//
// # Kenapa PERINGATAN, bukan penolakan
//
// Layar lama memeriksa hal yang sama tetapi **tidak memblokir penyimpanan**. Itu terbaca
// dari urutan langkahnya di `Activity/InsertUpdateMasterXOL-Act.xml`:
//
//	pySteps(2)     ulangi tiap lapisan
//	  pySteps(2.1) local.totalshare := 0
//	  pySteps(2.2) ulangi tiap reas → local.totalshare += .IndividualRiskPercentage
//	  pySteps(2.3) SETEL PESAN bila total bukan 100
//	pySteps(3)     SIMPAN INDUK        ← tanpa prasyarat apa pun
//	pySteps(7)     SIMPAN LAPISAN & REAS
//	pySteps(9)     tampilkan pesan bila ada
//
// Langkah 3 dan 7 tidak punya prasyarat, sehingga penyimpanan berjalan lebih dulu dan
// pesannya baru muncul sesudahnya. Buktinya ada di produksi: lapisan `10004` milik induk
// `10002` tersimpan dengan **nol** reasuradur, sehingga totalnya 0.
//
// Keputusan Work Owner 2026-09-20: perilaku itu ditiru apa adanya — data tetap tersimpan,
// pesannya ditampilkan sebagai peringatan.
//
// Prasyarat langkah 2.3 berbunyi `local.totalshare==100` dengan kode cabang
// `true=3` (lewati) dan `false=2` (jalankan). Jadi pesannya muncul justru ketika total
// BUKAN 100 — dibaca sekilas ia tampak terbalik, dan itu sebabnya dicatat di sini.
//
// Lapisan tanpa satu pun reasuradur ikut diperingatkan. Di sistem lama ia lolos tanpa
// pesan, karena totalnya 0 dan pesannya pun berbunyi "belum 100%" — sama saja artinya.
func ShareWarning(m Master) []string {
	clean := m.Clean()
	var warning []string

	for index, l := range clean.Layer {
		total := l.TotalShare()
		if total == FullShare {
			continue
		}
		// Nama lapisan dipakai bila ada, karena itulah yang dipakai pesan lama
		// (`local.layer := .ObjectName`). Bila kosong, nomor barisnya yang disebut —
		// pesan yang menunjuk lapisan tanpa nama tidak dapat ditindaklanjuti.
		label := l.Name
		if label == "" {
			label = "layer baris " + strconv.Itoa(index+1)
		}
		warning = append(warning,
			"Total share pada "+label+" belum 100% (sekarang "+strconv.FormatInt(int64(total), 10)+"%).")
	}
	return warning
}

// NextID mengembalikan nomor berikutnya untuk induk maupun lapisan.
//
// # Kenapa `max + 1`, dan kenapa dimulai dari 10001
//
// Ditiru dari `Database/INSERT_UPDATE_MST_XOL.prc`, yang memakai pola yang sama untuk
// keduanya:
//
//	:16-22  SELECT max(to_number(id)) ... → null atau 0 maka 10001, selain itu +1
//	:73-79  idem untuk idlayer
//
// Angka awal 10001 bukan pilihan bebas — data produksi memang dimulai dari sana, dan
// nomor di bawahnya tidak pernah ada.
//
// Nilai yang bukan angka DIABAIKAN, tidak membuat penerbitan gagal: `to_number` pada
// procedure lama akan melempar galat, tetapi menolak seluruh penyimpanan hanya karena
// satu baris warisan berisi ID aneh jauh lebih merugikan daripada melewatinya.
func NextID(existing []string) string {
	highest := int64(0)
	for _, id := range existing {
		value, err := strconv.ParseInt(strings.TrimSpace(id), 10, 64)
		if err != nil {
			continue
		}
		if value > highest {
			highest = value
		}
	}
	if highest == 0 {
		return "10001"
	}
	return strconv.FormatInt(highest+1, 10)
}

// tooLong menghitung dalam rune, bukan byte: satu huruf beraksen memakan dua byte dan
// akan membuat batas terasa berubah-ubah bagi pengguna.
func tooLong(value string, limit int) bool {
	return utf8.RuneCountInString(value) > limit
}

func lengthMessage(label string, limit int) string {
	return label + " paling panjang " + strconv.Itoa(limit) + " karakter."
}
