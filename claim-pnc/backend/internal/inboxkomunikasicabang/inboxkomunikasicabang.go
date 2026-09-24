// Package inboxkomunikasicabang adalah inti modul Inbox Komunikasi Cabang.
//
// # Nama modul ini
//
// Diambil dari nama yang dipakai Work Owner dan dari butir menunya sendiri:
// `Database/m_menu_aplikasi_pnc.csv` baris 65 berbunyi
//
//	1,70,"Inbox Komunikasi Cabang","InboxKomunikasiCabang",1,2,1160
//
// sehingga `MENU_ID 70`, `MENU_PROGRAM` `InboxKomunikasiCabang`. `D-81` menetapkan nama
// modul mengikuti nama yang dipakai Work Owner. Isi modulnya berbahasa Inggris sesuai
// `D-80`; yang berbahasa Indonesia hanya nama modul, nama field JSON, dan teks yang dilihat
// pengguna.
//
// # Artefak Pega yang dibaca
//
//	Harness/InboxKomunikasiCabang-Harness.xml       judul layar, susunan
//	Section/InboxKomunikasi-Section.xml             kedua grid, pencacah, form, tombol
//	Section/PengirimKomunikasi-Section.xml          sel "Pengirim(Dari)"
//	Section/PenjawabKomunikasi-Section.xml          sel "Penjawab(Dari)"
//	Section/BalasKomunikasiCabang-Section.xml       layar Detail Komunikasi + kotak balasan
//	Flow Action/DETAILKOMUNIKASICABANG_11-FA.xml    "DETAIL KOMUNIKASI CABANG"
//	Data Transform/DetailKomunikasi_dt-DT.xml       parameter layar detail
//	Activity/PNCGetInboxKomunikasiCabang_Act.xml    pemuat kedua grid
//	Activity/PNCCountKomunikasiCabang_Act.xml       pencacah + penurunan kode cabang
//	Activity/PNCSendMessageKomunikasiCabang.xml     tombol "Kirim Pesan"
//	Activity/EndKomunikasiCabang-Act.xml            tombol "Selesai Komunikasi"
//	RDB List/GetInboxKomunikasiCabang-SQL.xml       grid "Belum Dijawab"
//	RDB List/GetInboxKomunikasiCabangAnswered.xml   grid "Sudah Dijawab"
//	RDB List/GetCountKomunikasiCabangNotAnswered    pencacah kiri
//	RDB List/GetCountKomunikasiCabangAnswered       pencacah kanan
//	RDB List/GetDocumentKomunikasi1-SQL.xml         lampiran satu percakapan
//	RDB List/GetIDCabang-SQL.xml                    penurunan kode cabang petugas
//	RDB List/ENDMessageCABANG_PNC-SQL.xml           aksi "Selesai Komunikasi"
//	RDB List/ReplyKomunikasiCabang-SQL.xml          aksi "Kirim Pesan"
//
// # Apa itu Inbox Komunikasi Cabang
//
// Kotak percakapan antara KANTOR PUSAT dan CABANG seputar sebuah klaim. Satu baris adalah
// satu percakapan: sebuah pesan beserta balasannya, bila sudah dijawab.
//
// Ia benar-benar Inbox menurut `D-79`: barisnya adalah pekerjaan yang menunggu dijawab,
// "hanya milik saya" adalah aturan kewenangan (batas cabang, lihat BranchFilter), dan
// barisnya HILANG begitu selesai — tombol "Selesai Komunikasi" mengubah `CASEID` menjadi
// `CABANG SELESAI`, dan kedua grid menyaring `CASEID = 'CABANG'`.
//
// # DUA TAB, SATU TABEL, SATU PENYARING YANG MEMBEDAKAN
//
// Keduanya membaca `POOLDATA.M_KOMUNIKASI_PNC` dengan penyaring yang sama persis kecuali
// satu — apakah percakapannya sudah dibalas:
//
//	tab 1  Belum Dijawab   REPLYMESSAGE IS NULL      urut CREATEDDATE ASC
//	tab 2  Sudah Dijawab   REPLYMESSAGE IS NOT NULL  urut CREATEDATEREPLY DESC
//
// Urutannya pun berlawanan, dan itu bukan kelalaian: yang belum dijawab dibaca dari yang
// PALING LAMA menunggu, yang sudah dijawab dari yang PALING BARU dibalas. Keduanya dibawa
// apa adanya (`P-5`).
//
// Kolomnya pula BERBEDA — tab "Belum Dijawab" menggambar tiga kolom, tab "Sudah Dijawab"
// menggambar lima. Lihat tab.go.
//
// # HAL YANG HARUS DISADARI SEBELUM MEMBACA SISA BERKAS INI
//
// Kueri lama memakai SATU ALIAS untuk DUA kolom dalam satu SELECT yang sama:
//
//	sendername       AS "UserName"   <- kolom ketiga
//	COMMUNICATE_FROM AS "UserName"   <- kolom terakhir
//
// Keduanya di `GetInboxKomunikasiCabang-SQL.xml`, berjarak enam baris. Yang menang adalah
// yang TERAKHIR, dan itu terbukti dari pemakaiannya sendiri:
// `PNCGetInboxKomunikasiCabang_Act` langkah 5 memeriksa `.UserName == "1"` lalu menimpanya
// dengan `"PUSAT"` — perbandingan yang hanya masuk akal bila isinya kode asal
// (`COMMUNICATE_FROM`), bukan nama orang (`sendername`).
//
// `sendername` karena itu TIDAK PERNAH sampai ke layar, meski kueri mengambilnya. Modul ini
// tidak mengambilnya sama sekali; lihat catatan pada berkas .sql.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, driver basis data, maupun bentuk JSON.
package inboxkomunikasicabang

import (
	"context"
	"errors"
	"strings"
)

// Conversation adalah satu baris pada grid — satu percakapan antara pusat dan cabang.
//
// Kelima isian yang digambar TIDAK seluruhnya dipakai kedua tab: tab "Belum Dijawab"
// menggambar tiga, tab "Sudah Dijawab" menggambar lima. Yang menentukan kolom mana yang
// tampil adalah Tab.Columns, bukan ada-tidaknya isian di sini — isian yang kebetulan kosong
// pada seluruh baris halaman tidak boleh membuat kolomnya menghilang.
type Conversation struct {
	// ID adalah nomor percakapan — `KOMUNIKASIID`.
	//
	// Ia kunci yang dipakai tombol "Detail Komunikasi": `DetailKomunikasi_dt` menerimanya
	// sebagai `Param.KOMID`. Ia TIDAK digambar sebagai kolom.
	//
	// Alias Pega-nya "ClaimNo" dan itu MENYESATKAN — ia bukan nomor klaim (`D-19`). Nomor
	// klaim tidak ada di tabel ini sama sekali.
	ID string

	// CreatedAt — kolom **"Tanggal"** <- `CREATEDDATE`.
	//
	// Alias Pega-nya "CloseClaimDate", dan itu pun menyesatkan: tidak ada klaim yang
	// ditutup di sini. Ia tanggal pesannya dikirim.
	CreatedAt string

	// SenderOrigin adalah ASAL pesan — `COMMUNICATE_FROM`, sudah diterjemahkan.
	//
	// Nilainya `"PUSAT"` bila kodenya `1`, dan kode cabangnya apa adanya bila bukan. Itu
	// perilaku `PNCGetInboxKomunikasiCabang_Act` langkah 5 apa adanya — lihat OriginOf.
	SenderOrigin string

	// SenderOperator adalah pengirimnya — `SENDER`, berisi Operator ID.
	//
	// Bersama SenderOrigin ia menyusun sel "Pengirim(Dari)", yang di
	// `Section/PengirimKomunikasi-Section.xml` digambar sebagai `UserName (UserTeknis)` —
	// asal, lalu operator di dalam kurung.
	SenderOperator string

	// Message — kolom **"Pesan"** <- `MESSAGE`.
	//
	// Alias Pega-nya "Email", dan isinya bukan alamat surel melainkan isi pesannya.
	Message string

	// Reply — kolom **"Jawaban Terakhir"** <- `REPLYMESSAGE`.
	//
	// SELALU kosong pada tab "Belum Dijawab" — penyaring tab itu justru `IS NULL`. Kolomnya
	// memang tidak digambar di sana, dan itulah sebabnya kedua tab punya kolom berbeda.
	Reply string

	// ReplierName adalah nama penjawabnya — `REPLYFROMNAME`.
	ReplierName string

	// RecipientOrigin adalah TUJUAN pesan — `COMMUNICATE_TO`, sudah diterjemahkan.
	//
	// Nilainya `"PUSAT"` bila kodenya `1`, dan `"CABANG"` untuk NILAI APA PUN yang lain —
	// termasuk kosong. Perhatikan ia TIDAK simetris dengan SenderOrigin, yang menyimpan
	// kode cabangnya apa adanya. Itu perilaku langkah 6 dan 7 apa adanya; lihat
	// RecipientOf.
	//
	// Bersama ReplierName ia menyusun sel "Penjawab(Dari)" — `UserAdmin (UserTeknisEmail)`
	// pada `Section/PenjawabKomunikasi-Section.xml`.
	RecipientOrigin string

	// Status — `KOMUNIKASISTATUS`.
	//
	// Alias Pega-nya "StatusClaim" dan itu menyesatkan pula: ia bukan Status Klaim ber-33
	// kode (`R-06`). Artinya tidak diketahui — tidak ada master yang menerjemahkannya di
	// export mana pun — sehingga ia dibawa MENTAH dan tidak digambar sebagai kolom.
	Status string

	// RepliedAt — `CREATEDATEREPLY`, tanggal balasannya.
	//
	// Ia DASAR PENGURUTAN tab "Sudah Dijawab" tetapi tidak digambar sebagai kolom di grid
	// mana pun. Dibawa supaya urutan tabel dapat dijelaskan bila dipertanyakan.
	RepliedAt string
}

// Caller adalah identitas petugas yang mengirim permintaan.
//
// Ia dibawa sebagai nilai, bukan diambil dari variabel global mana pun: konteks pengguna
// wajib mengalir lewat parameter (`08-TECHNICAL-STRATEGY.md` §4.6).
//
// # Kenapa modul ini benar-benar MEMAKAINYA, bukan sekadar mencatatnya
//
// Berbeda dari modul Inbox RCL/PUCL yang antreannya bersama, layar ini DISARING menurut
// cabang petugasnya — dan cabang itu diturunkan dari Login lewat BranchResolver. Login yang
// tidak terbaca karena itu bukan sekadar kehilangan jejak; ia membuat batas datanya tidak
// dapat ditentukan sama sekali.
type Caller struct {
	// Login adalah nama pengguna yang DIKETIK saat masuk —
	// `OperatorID.pyUserIdentifier` di sistem lama. Bukan NIK.
	//
	// Ia yang dicocokkan `GetIDCabang` ke `V_HRD_MST.login_aplikasi`.
	Login string
}

// Clean memangkas spasi setiap isian identitas.
func (c Caller) Clean() Caller {
	return Caller{Login: strings.TrimSpace(c.Login)}
}

// CaseOpen adalah nilai `CASEID` yang menandai percakapan MASIH BERJALAN.
//
// Kedua kueri grid menyaring `CASEID = 'CABANG'`, dan tombol "Selesai Komunikasi"
// menggantinya menjadi CaseClosed. Karena itulah barisnya hilang dari layar begitu
// percakapannya ditutup — dan karena itulah layar ini benar-benar Inbox menurut `D-79`.
//
// Perhatikan `CASEID` di sini BUKAN kunci kasus. Aliasnya di kueri lama "pzInsKey", dan itu
// menyesatkan: isinya penanda kanal percakapan, bukan kunci objek kerja Pega (`D-19`).
const CaseOpen = "CABANG"

// CaseClosed adalah nilai `CASEID` setelah tombol "Selesai Komunikasi" ditekan.
//
// Dari `RDB List/ENDMessageCABANG_PNC-SQL.xml`:
//
//	UPDATE POOLDATA.M_KOMUNIKASI_PNC
//	   SET CASEID = {TempEndKomunikasi.CaseID}
//	 WHERE KOMUNIKASIID = {TempEndKomunikasi.ClaimID}
//
// dan `Activity/EndKomunikasiCabang-Act.xml` langkah 1 mengisi nilai itu dengan
// `"CABANG SELESAI"` — spasi di tengah, bukan garis bawah.
//
// Ia dicatat meski modul ini TIDAK menulis: ia yang menjelaskan mengapa sebuah percakapan
// dapat lenyap dari kedua tab tanpa terhapus, dan tanpa konstanta ini penjelasannya hanya
// hidup di komentar.
const CaseClosed = "CABANG SELESAI"

// HeadOfficeCode adalah kode asal KANTOR PUSAT pada `COMMUNICATE_FROM`/`COMMUNICATE_TO`.
//
// Nilainya `1`, dan itu BUKAN tebakan: `PNCCountKomunikasiCabang_Act` langkah 9 menyusun
// penyaringnya sebagai
//
//	" (COMMUNICATE_TO = '1' OR COMMUNICATE_FROM = '1') "
//
// untuk petugas kantor pusat, sementara `PNCGetInboxKomunikasiCabang_Act` langkah 5 dan 6
// menerjemahkan nilai `"1"` yang sama menjadi teks `"PUSAT"` di layar. Keduanya menyebut
// nilai yang sama untuk hal yang sama.
const HeadOfficeCode = "1"

// HeadOfficeBranch adalah kode cabang di `POOLDATA.BRANCH` yang berarti KANTOR PUSAT.
//
// Dari precondition `PNCCountKomunikasiCabang_Act` langkah 8 dan 9:
//
//	TempIDCabang.pxResults(1).KodeCabang == "100081" || TempIDCabang.pxResults(1).KodeCabang == ""
//
// Petugas yang cabangnya `100081` — atau yang cabangnya TIDAK TERBACA SAMA SEKALI —
// diperlakukan sebagai kantor pusat. Lihat BranchFilter untuk akibat keduanya, yang
// disatukan di sistem lama dan tetap disatukan di sini atas keputusan Work Owner
// 2026-09-24 (`P-5`).
//
// Angka ini konsisten dengan temuan modul lain: `docs/catatan-pengembangan.md` §19.14
// mencatat 801 baris berkode cabang `100081` pada tabel kerja Pega.
const HeadOfficeBranch = "100081"

// Teks asal percakapan sebagaimana digambar layar.
//
// Keduanya berbahasa Indonesia karena ia TEKS YANG DILIHAT PENGGUNA, dan `D-13` menetapkan
// teks layar mengikuti layar lama apa adanya.
const (
	// OriginHeadOffice digambar untuk kode `1`.
	OriginHeadOffice = "PUSAT"

	// OriginBranch digambar untuk nilai lain pada KOLOM TUJUAN saja.
	OriginBranch = "CABANG"
)

// OriginOf menerjemahkan `COMMUNICATE_FROM` menjadi teks kolom "Pengirim(Dari)".
//
// Dari `PNCGetInboxKomunikasiCabang_Act` langkah 5:
//
//	when .UserName == "1"  ->  .UserName := "PUSAT"
//
// dan TIDAK ADA langkah lain yang menyentuhnya. Nilai selain `1` karena itu tetap tampil
// APA ADANYA — yaitu kode cabangnya, sebuah angka seperti `100081`.
//
// # Kenapa ia TIDAK disamakan dengan RecipientOf
//
// Karena sistem lama memang tidak menyamakannya, dan menyamakannya akan MENGHILANGKAN
// keterangan: kolom "Pengirim(Dari)" adalah satu-satunya tempat di layar ini yang
// menyebutkan cabang MANA yang mengirim. Menggantinya dengan kata "CABANG" membuat seluruh
// cabang terbaca sama.
//
// Ia ada di paket domain, bukan di SQL, karena kedua pengisi seam wajib menghasilkan teks
// yang sama persis — uji aturan modul yang berjalan di atas memori hanya menyatakan sesuatu
// tentang Oracle bila keduanya memakai penerjemah yang sama.
func OriginOf(code string) string {
	clean := strings.TrimSpace(code)
	if clean == HeadOfficeCode {
		return OriginHeadOffice
	}
	return clean
}

// RecipientOf menerjemahkan `COMMUNICATE_TO` menjadi teks kolom "Penjawab(Dari)".
//
// Dari `PNCGetInboxKomunikasiCabang_Act` langkah 6 dan 7, berurutan:
//
//	when .UserTeknisEmail == "1"       ->  .UserTeknisEmail := "PUSAT"
//	when .UserTeknisEmail != "PUSAT"   ->  .UserTeknisEmail := "CABANG"
//
// Langkah kedua menyapu SISANYA — termasuk nilai kosong. Akibatnya kolom tujuan hanya punya
// dua nilai yang mungkin, sementara kolom asal punya sebanyak jumlah cabang.
//
// Asimetri itu terbaca seperti kelalaian penulisnya, dan mungkin memang begitu. Ia tetap
// direplikasi (`P-5`): memperbaikinya berarti menampilkan kode cabang di tempat pengguna
// hari ini membaca kata "CABANG", dan itu selisih yang tidak diputuskan siapa pun.
func RecipientOf(code string) string {
	if strings.TrimSpace(code) == HeadOfficeCode {
		return OriginHeadOffice
	}
	return OriginBranch
}

// BranchFilter adalah batas data layar ini — cabang mana percakapan yang boleh dilihat.
//
// # Dari mana bentuknya
//
// `PNCCountKomunikasiCabang_Act` menurunkan kode cabang petugas lewat `GetIDCabang`, lalu
// merangkai penyaringnya sebagai teks:
//
//	langkah 8  " (COMMUNICATE_TO = '"+IDCABANG+"' OR COMMUNICATE_FROM = '"+IDCABANG+"') "
//	langkah 9  " (COMMUNICATE_TO = '1' OR COMMUNICATE_FROM = '1') "   <- kantor pusat
//
// Di sini keduanya menjadi SATU nilai — Code — dan perangkaian teksnya diganti parameter
// binding tanpa perkecualian (`08-TECHNICAL-STRATEGY.md` §4.3).
//
// # Kenapa petugas yang cabangnya TIDAK TERBACA tetap dilayani
//
// Ini keputusan Work Owner 2026-09-24: **replikasi apa adanya (`P-5`)**.
//
// Precondition sistem lama menyatukan dua keadaan yang berbeda:
//
//	KodeCabang == "100081"   petugas kantor pusat
//	KodeCabang == ""         cabangnya tidak dapat diturunkan sama sekali
//
// dan memperlakukan KEDUANYA sebagai kantor pusat. Modul Inbox Laporan Klaim mengambil
// keputusan yang BERLAWANAN untuk keadaan kedua — ia menolak permintaannya
// (`keputusan-implementasi.md` §20) — dan perbedaan itu disengaja: keputusan §20 diambil
// untuk layar itu, dan Work Owner menetapkan layar INI mereplikasi Pega.
//
// # Akibatnya, dan ia harus dinyatakan alih-alih disembunyikan
//
// Petugas yang tidak terdaftar di HRD — broker dan surveyor independen masuk lewat
// `POOLDATA.M_LOGIN_PNC` dan memang tidak pernah ada di sana — melihat percakapan KANTOR
// PUSAT. Itu pelebaran batas data yang tidak menghasilkan satu pun galat, dan karena itu ia
// dinyatakan lewat PlannedDifferences serta dicatat pada setiap permintaan di usecase.
type BranchFilter struct {
	// Code adalah kode yang dibandingkan dengan `COMMUNICATE_FROM` dan `COMMUNICATE_TO`.
	//
	// Untuk kantor pusat ia HeadOfficeCode (`1`); untuk cabang ia kode cabangnya.
	Code string

	// HeadOffice menyatakan penyaringnya jatuh ke jalur kantor pusat.
	HeadOffice bool

	// Resolved menyatakan kode cabang petugas benar-benar terbaca.
	//
	// Ia dibedakan dari HeadOffice supaya kedua keadaan yang sistem lama satukan tetap
	// dapat dibedakan di log dan di layar. `HeadOffice && !Resolved` berarti petugasnya
	// TIDAK terdaftar dan sedang melihat percakapan kantor pusat karena itu.
	Resolved bool
}

// BranchResolver menerjemahkan login petugas menjadi kode cabang klaimnya.
//
// # Kenapa ia seam tersendiri
//
// Karena penurunannya menyentuh dua basis data lain lewat DB Link, dan `D-25` menetapkan
// seluruh DB Link kelak diganti pemanggilan API (`R-03`). Menaruhnya di balik seam berarti
// penggantian itu kelak tidak menyentuh satu baris pun aturan modul ini.
//
// Jalurnya tiga tabel dan dua DB Link (`RDB List/GetIDCabang-SQL.xml`):
//
//	login petugas -> HRDASM.V_HRD_MST.login_aplikasi
//	              -> NIK
//	              -> LST_USER_ASURANSI.cab_id
//	              -> BRANCH.oldid
//	              -> BRANCH.id            <- inilah kodenya
//
// Sambungan terakhir lewat `oldid`, BUKAN `id`. Tidak ada jalan pintas dari profil HCC/HCQ
// ke nilai itu — modul Inbox Laporan Klaim pernah memakai `Placement.BranchCode` dan
// hasilnya daftar kosong tanpa satu pun galat.
//
// # Kegagalan BUKAN galat
//
// Nilai kedua false berarti cabangnya tidak dapat ditentukan. Di layar INI pemanggil
// memperlakukannya sebagai kantor pusat (`P-5`, lihat BranchFilter) — berbeda dari modul
// Inbox Laporan Klaim, yang menolaknya.
type BranchResolver interface {
	Resolve(ctx context.Context, login string) (code string, resolved bool, err error)
}

// Attachment adalah satu lampiran pada sebuah percakapan.
//
// Sumbernya `RDB List/GetDocumentKomunikasi1-SQL.xml`, yang menggabungkan
// `POOLDATA.D_KOMUNIKASI_PNC` dengan dua master jenis dokumen.
//
// # Setiap alias pada kueri itu menyebut hal yang lain sama sekali
//
// Ini contoh utang teknis §4.2 yang paling padat di modul ini — tujuh alias, tujuh kali
// salah:
//
//	alias Pega       kolom sebenarnya            isi sebenarnya
//	---------------- --------------------------- ----------------------------
//	BranchCode       A.KOMUNIKASI_ID             nomor percakapan
//	BranchName       A.CATEGORYID                kode kategori dokumen
//	ClaimID          A.TYPEID                    kode jenis dokumen
//	ObjectName       C.DETAIL_DOCUMENT           nama rincian dokumen
//	ObjectLocation   D.TYPE_DOCUMENT             nama jenis dokumen
//	Comment          A.NOTE                      catatan
//	Date             A.UPLOADDATE                tanggal unggah
//	IdCompliance     A.DOCUMENTID                id dokumen di penyimpanan
//
// Tidak satu pun dibawa (`D-19`). Nama di bawah menyebut isinya.
type Attachment struct {
	// DocumentID adalah `DOCUMENTID` — rujukan ke penyimpanan dokumen (`D-16`).
	//
	// Ia dibawa tetapi TIDAK dipakai mengunduh apa pun: modul ini membaca metadata, dan
	// pengambilan berkasnya menempuh seam DocumentStore yang belum dibangun di sini.
	DocumentID string

	// TypeName adalah nama JENIS dokumen — `V_LST_DOC_TYPE.TYPE_DOCUMENT`.
	TypeName string

	// DetailName adalah nama RINCIAN dokumen — `V_LST_DET_TYPE_DOC.DETAIL_DOCUMENT`.
	DetailName string

	// Note adalah catatan unggahan — `NOTE`.
	Note string

	// UploadedAt adalah tanggal unggah — `UPLOADDATE`.
	//
	// Kosong berarti BELUM diunggah, dan itu arti yang dipakai sistem lama sendiri:
	// `RDB List/BrowseKomunikasiDokumenCabang-SQL.xml` menyimpulkan "Belum Upload" dari
	// `uploaddate is null`. Lihat Uploaded.
	UploadedAt string
}

// Uploaded menyatakan lampirannya sudah benar-benar diunggah.
//
// Sistem lama menyimpulkannya dari `uploaddate is null`, dan status yang dihasilkannya
// hanya dua: "Sudah Upload" dan "Belum Upload".
//
// Perhatikan kueri lama itu punya cacat yang tidak dibawa: cabang pertama `CASE`-nya
// berbunyi `when COUNT(...) < 0 then 'Sudah Upload'`, dan sebuah `COUNT` tidak pernah
// negatif — sehingga cabang itu TIDAK PERNAH dipilih dan seluruh baris selalu berbunyi
// "Belum Upload". Di sini kesimpulannya diambil per lampiran dari kolomnya sendiri, bukan
// dari pencacah yang tidak dapat benar.
func (a Attachment) Uploaded() bool {
	return strings.TrimSpace(a.UploadedAt) != ""
}

// ThreadMessage adalah satu baris pada layar **Detail Komunikasi**.
//
// # Layar apa ini
//
// Yang terbuka saat tombol "Detail Komunikasi" ditekan. Tombolnya menjalankan
// `Data Transform/DetailKomunikasi_dt-DT.xml`, yang menyiapkan dua parameter —
//
//	TempView2       := Param.KOMID        nomor percakapan
//	TempView2.City  := Param.KODECABANG   kode cabangnya
//
// — lalu flow action `DETAILKOMUNIKASICABANG_11` ("DETAIL KOMUNIKASI CABANG") menyisipkan
// `Section/BalasKomunikasiCabang-Section.xml`. Section itu menggambar tiga kolom —
// **Tanggal**, **Pengirim**, **Pesan** — beserta kotak "Masukkan Balasan" dan tombol
// "Balas".
//
// # DUA ARTEFAKNYA HILANG DARI EXPORT
//
// Kueri grid-nya (`GetInboxKomunikasiCabang_detail`) dan activity tombol Balas
// (`PNCReplyMessageCabang`) TIDAK ADA di export mana pun — keduanya kelas `R-16`.
//
// Yang dibangun di sini karena itu adalah bentuk yang terbaca dari section-nya sendiri
// (tiga kolom, urut tanggal) atas percakapan yang nomornya diminta. Ia TIDAK mengarang
// penyaring yang tidak dapat dibaca dari mana pun: yang dipakai hanyalah nomor percakapan,
// satu-satunya parameter yang data transform-nya benar-benar kirimkan.
type ThreadMessage struct {
	// CreatedAt — kolom **"Tanggal"**.
	CreatedAt string

	// SenderOrigin adalah asal pesan, sudah diterjemahkan lewat OriginOf.
	SenderOrigin string

	// SenderOperator adalah Operator ID pengirimnya.
	//
	// Bersama SenderOrigin ia menyusun kolom **"Pengirim"**, digambar dengan pola yang
	// sama seperti di grid — asal, lalu operator di dalam kurung.
	SenderOperator string

	// Message — kolom **"Pesan"**.
	Message string

	// Reply adalah balasan pada baris yang sama, bila ada.
	//
	// Section lama TIDAK menggambar kolom balasan di layar detail; ia hanya menggambar
	// Tanggal, Pengirim, dan Pesan. Isian ini tetap dibawa karena satu baris
	// `M_KOMUNIKASI_PNC` menyimpan pesan DAN balasannya, dan layar yang membuang balasannya
	// akan menampilkan percakapan yang separuh isinya hilang.
	//
	// Ia digambar sebagai baris tersendiri di bawah pesannya, bukan sebagai kolom keempat —
	// sehingga ketiga kolom section lama tetap utuh (`D-13`).
	Reply string

	// ReplierName adalah nama penjawabnya, digambar bersama balasannya.
	ReplierName string

	// RepliedAt adalah tanggal balasannya.
	RepliedAt string
}

// ConversationDetail adalah isi layar Detail Komunikasi untuk satu percakapan.
type ConversationDetail struct {
	// ID adalah nomor percakapan yang dibuka.
	ID string

	// Messages adalah utas pesannya, terurut menaik menurut tanggal.
	Messages []ThreadMessage

	// Attachments adalah lampiran percakapan ini.
	//
	// Keputusan Work Owner 2026-09-24: lampiran ikut dibangun, BACA-SAJA. Mengunggah
	// lampiran baru menulis ke `D_KOMUNIKASI_PNC`, dan tabel itu milik Pega selama masa
	// paralel (`P-1`).
	Attachments []Attachment

	// Origin adalah asal percakapan, sudah diterjemahkan — dipakai judul layar detail.
	Origin string
}

// ErrConversationNotFound dikembalikan saat nomor percakapan tidak ada di portal terpilih.
//
// Ia dibedakan dari galat teknis dengan sengaja: nomor yang benar pada portal yang SALAH
// menghasilkan keadaan ini, dan itu keterangan yang harus sampai ke pengguna (`R-20`).
var ErrConversationNotFound = errors.New("inboxkomunikasicabang: percakapan tidak ditemukan")

// Pagination menyatakan halaman keberapa yang diminta dan sebesar apa.
//
// # Kenapa ukuran bawaannya 20
//
// Karena itu ukuran halaman grid layar ini — `<pyPageSize>20</pyPageSize>` pada
// `Section/InboxKomunikasi-Section.xml`, muncul TIGA kali dengan nilai yang sama: kedua
// grid dan grid layar detail.
//
// Angka itu BERBEDA dari modul Inbox RCL/PUCL yang memakai 50 dan dari sebagian modul lain
// yang memakai 25. Perbedaannya TIDAK diseragamkan: yang dipakai adalah angka layarnya
// sendiri, karena ukuran halaman menentukan baris mana yang terlihat tanpa menggulir — dan
// itu bagian dari tampilan yang `D-13` tetapkan mengikuti Pega.
type Pagination struct {
	// Page dimulai dari 1.
	Page int

	// Size adalah jumlah baris per halaman.
	Size int
}

// Batas paginasi.
//
// MaxSize 100 mengikuti `10-API-STRATEGY.md` §4: permintaan yang lebih besar DITOLAK, bukan
// dipenuhi diam-diam — memenuhinya membuat batas menjadi saran, bukan batas.
const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

// Normalize mengembalikan paginasi yang sudah dibetulkan ke rentang yang sah.
//
// Nilai di luar rentang DIBETULKAN, tidak ditolak: halaman dan ukuran datang dari parameter
// query yang mudah salah ketik, dan menolak seluruh permintaan karena `halaman=0` akan
// membuat layar gagal tanpa alasan yang terbaca pengguna.
func (p Pagination) Normalize() Pagination {
	clean := p
	if clean.Page < 1 {
		clean.Page = 1
	}
	if clean.Size < 1 {
		clean.Size = DefaultPageSize
	}
	if clean.Size > MaxPageSize {
		clean.Size = MaxPageSize
	}
	return clean
}

// Offset adalah jumlah baris yang dilewati sebelum halaman yang diminta.
func (p Pagination) Offset() int {
	clean := p.Normalize()
	return (clean.Page - 1) * clean.Size
}

// Page adalah satu halaman hasil beserta jumlah seluruh baris yang cocok.
type Page struct {
	Items []Conversation

	// Total adalah jumlah SELURUH baris yang cocok di penyimpanan, bukan yang tampil.
	Total int

	// Pagination adalah paginasi yang BENAR-BENAR dipakai setelah dibetulkan.
	Pagination Pagination
}

// TotalPages adalah jumlah halaman, minimal 1 supaya layar tidak pernah menggambar
// "halaman 1 dari 0" saat hasilnya kosong.
func (p Page) TotalPages() int {
	size := p.Pagination.Normalize().Size
	if p.Total <= 0 {
		return 1
	}
	pages := p.Total / size
	if p.Total%size != 0 {
		pages++
	}
	return pages
}

// Slice memotong satu halaman dari seluruh baris yang sudah di tangan.
//
// Ia dipakai penyimpanan MEMORI saja. Penyimpanan SQL memotongnya di basis data dengan
// `OFFSET … FETCH NEXT`, dan perbedaan itu disengaja — lihat catatan paginasi di kepala
// repo/sqlstore/inboxkomunikasicabang.sql.
func Slice(all []Conversation, page Pagination) Page {
	clean := page.Normalize()

	result := Page{Total: len(all), Pagination: clean, Items: []Conversation{}}

	offset := clean.Offset()
	if offset >= len(all) {
		return result
	}

	end := offset + clean.Size
	if end > len(all) {
		end = len(all)
	}

	result.Items = all[offset:end]
	return result
}

// Summary adalah kedua pencacah di atas grid.
//
// Sumbernya `GetCountKomunikasiCabangNotAnswered` dan `GetCountKomunikasiCabangAnswered`,
// yang di layar lama memasok diagram lingkaran (`.pyTemplateChart` pada section) berlabel
// "Answered" dan "Not Answered".
//
// # Kenapa dua angka, bukan satu per tab yang sedang dibuka
//
// Karena keduanya digambar BERSAMAAN di layar lama, di atas kedua grid. Petugas melihat
// berapa yang menunggu jawaban TANPA berpindah tab — dan itulah satu-satunya hal di layar
// ini yang menyatakan ada pekerjaan di tab sebelah.
type Summary struct {
	// NotAnswered adalah jumlah percakapan yang BELUM dibalas.
	//
	// Penyaringnya `REPLYFROM IS NULL AND REPLYMESSAGE IS NULL` — DUA kolom, bukan satu.
	// Kueri grid hanya memeriksa `REPLYMESSAGE`. Selisih itu dibawa apa adanya; lihat
	// catatan pada berkas .sql.
	NotAnswered int

	// Answered adalah jumlah percakapan yang SUDAH dibalas.
	Answered int
}

// Total adalah jumlah seluruh percakapan berjalan pada batas cabang yang berlaku.
func (s Summary) Total() int {
	return s.NotAnswered + s.Answered
}

// Repo adalah seam ke penyimpanan komunikasi cabang pada SATU portal.
//
// Dideklarasikan DI SINI, di paket yang memakainya — bukan di paket yang memenuhinya. Diisi
// `repo/sqlstore` terhadap Oracle dan `repo/memory` untuk pengujian.
//
// # Tidak ada satu pun operasi yang menulis, dan itu keputusan
//
// Layar lama punya TIGA tindakan yang menulis, dan ketiganya menyentuh tabel yang selama
// masa paralel dimiliki Pega (`P-1`):
//
//	Kirim Pesan         PNCSendMessageKomunikasiCabang -> ReplyKomunikasiCabang (INSERT)
//	Balas               PNCReplyMessageCabang          -> HILANG dari export
//	Selesai Komunikasi  EndKomunikasiCabang            -> ENDMessageCABANG_PNC (UPDATE)
//
// Yang kedua tidak dapat direplikasi sama sekali — activity-nya tidak ada di export mana
// pun, sehingga tidak ada yang dapat dibaca untuk ditulis ulang. Mengarang logikanya
// dilarang: aturan kerja proyek ini melarang logika karangan bila proses aslinya dapat
// dipelajari, dan di sini ia justru TIDAK dapat dipelajari.
//
// Tombolnya tetap digambar dan penekanannya dijawab dengan alasan; lihat
// ErrWriteNotAvailable dan preseden `RejectWrite` pada modul Inbox RCL/PUCL.
type Repo interface {
	// List mengembalikan SATU HALAMAN percakapan yang cocok beserta jumlah seluruhnya.
	//
	// Paginasi diserahkan ke pengisi seam, bukan dikerjakan pemanggil, supaya pengisi SQL
	// dapat memotongnya di basis data. Pengisi memori memakai Slice untuk hasil yang sama.
	List(ctx context.Context, query Query, page Pagination) (Page, error)

	// Summarize mengembalikan kedua pencacah pada batas cabang yang sama.
	//
	// Ia TIDAK menerima Tab: kedua angkanya dihitung sekaligus, dan memisahkannya per tab
	// akan membuat layar menampilkan dua angka dari dua saat yang berbeda.
	Summarize(ctx context.Context, filter BranchFilter) (Summary, error)

	// Detail mengembalikan isi layar Detail Komunikasi untuk satu percakapan.
	//
	// Nomor yang tidak ditemukan menghasilkan ErrConversationNotFound, BUKAN nilai kosong:
	// percakapan yang tidak ada dan percakapan yang isinya kosong terlihat sama di layar,
	// dan hanya yang pertama yang merupakan kekeliruan.
	Detail(ctx context.Context, id string, filter BranchFilter) (ConversationDetail, error)
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// # Kenapa per portal, bukan satu penyimpanan
//
// `ADR-0030` menetapkan satu database per entitas, bukan satu database bersama dengan
// penanda entitas. Percakapan milik Asuransi Sinar Mas dan percakapan milik Simas Insurtech
// karena itu tidak pernah berada di tabel yang sama.
//
// # Kenapa galat, bukan cadangan
//
// Portal yang tidak dikenal atau koneksinya belum hidup menghasilkan galat — TIDAK PERNAH
// dialihkan ke koneksi utama. Jatuh ke koneksi default berarti menampilkan percakapan satu
// badan hukum kepada petugas badan hukum lain tanpa satu pun pesan galat (`R-20`,
// `TKT-F6-002`).
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
type RepoSelector func(portalAlias string) (Repo, error)
