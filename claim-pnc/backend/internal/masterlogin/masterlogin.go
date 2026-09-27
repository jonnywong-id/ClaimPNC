// Package masterlogin adalah inti modul Master Login.
//
// # Apa yang dimodelkan di sini
//
// Daftar **login surveyor** — satu baris per orang yang boleh masuk sebagai surveyor,
// beserta alamat surel, telepon, dan alamatnya. Tabelnya
// `POOLDATA.MST_LOGIN_SURVEYOR`, dan modul ini menjadi penulisnya (`P-1`).
//
// Ia master paling sederhana di rumpunnya: **tanpa alur persetujuan**, **tanpa penghapusan**,
// dan **tanpa tabel acuan**. Tujuh kolom, lima di antaranya diketik pengguna.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/MasterLoginSurvey-Harness.xml             layar, MENU_ID 37
//	Section/LoginSurveyor-Section.xml                 judul, tombol Tambah dan Refresh
//	Section/BrowseLoginSurveyor-Section.xml           grid 5 kolom + form 5 isian
//	RDB List/GetLoginMemberSurveyor-SQL.xml           daftar dan pemuatan satu baris
//	RDB List/GetLoginLeaderSurveyor-SQL.xml           sisip, dan pencarian leader
//	RDB List/UpdateMasterLoginSurvey-SQL.xml          simpan, WHERE login = ...
//	Activity/CNMInsertMstLoginSurveyor_act-Act.xml    urutan langkah simpan
//	Activity/SetLoginSurveyor_act-Act.xml             penurunan Login dari Nama
//	Activity/SetLoginSurveyorValue_act-Act.xml        pemuatan baris ke form (tombol Ubah)
//	Database/m_menu_aplikasi_pnc.csv                  MENU_ID 37 "Master Login"
//
// # Empat hal yang membedakannya dari modul master lain
//
//  1. **Kuncinya diturunkan, bukan diterbitkan.** Tidak ada sequence dan tidak ada
//     `MAX(...)+1`. `LOGIN` dihitung dari `NAMA`; lihat DeriveLogin.
//  2. **Tanpa kolom APPROVAL.** Tabelnya tidak punya, dan tidak satu pun rule Pega yang
//     menyentuhnya menyebut persetujuan. Karena itu layar ini tidak bertab, dan tidak ada
//     jalur Decide sama sekali.
//  3. **Dua kolom ditulis tetapi tidak pernah digambar.** `STSLOGIN` dan `LOGINLEADER`
//     tidak muncul sekali pun di `Section/BrowseLoginSurveyor-Section.xml`; keduanya
//     diturunkan saat menyimpan. Lihat LoginStatusMember dan Input.
//  4. **Tidak ada pencatat pelaku maupun stempel waktu.** Ketujuh kolomnya terbaca lengkap
//     dari ketiga rule SQL-nya, dan tidak satu pun menampung siapa atau kapan.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterlogin

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// LoginStatusMember adalah nilai `STSLOGIN` yang ditulis setiap penambahan.
//
// Ia KONSTANTA, bukan pilihan pengguna: `Activity/CNMInsertMstLoginSurveyor_act`
// menetapkan `TempLoginSurvey.ObjectName := "Member"` tanpa syarat apa pun, dan isian itu
// tidak pernah digambar di layar.
//
// Nilai lain memang ada di tabelnya — `GetLoginLeaderSurveyor` membaca `LOGINLEADER`, yang
// berisi login seseorang yang berperan sebagai leader — tetapi **tidak ada satu pun rule di
// export yang menuliskan nilai selain "Member"** (`R-16`). Karena itu modul ini tidak
// menawarkan pilihan: mengarang daftar nilai sah pada kolom yang menentukan peran seseorang
// adalah tebakan yang paling mahal.
//
// Baris lama yang sudah berisi nilai lain DIBACA apa adanya dan DIPERTAHANKAN saat baris itu
// disunting; lihat usecase.Service.Save.
const LoginStatusMember = "Member"

// MaxNameLength adalah panjang maksimum Nama.
//
// ASUMSI YANG DISADARI, bukan angka dari DDL: `POOLDATA.MST_LOGIN_SURVEYOR` tidak ada
// DDL-nya di export (`R-08`), dan layar lamanya tidak memasang satu pun `pyMaxLength` pada
// kelima isiannya.
//
// Batasnya tetap dipasang karena tanpa itu penolakan datang dari basis data sebagai
// ORA-12899 — galat teknis yang tidak menuntun pengguna ke mana pun. Seratus dipilih agar
// sama dengan batas nama pada modul master lain di aplikasi ini.
//
// Angka yang sama diulang di `SurveyorLoginForm.tsx`. Bila berubah, KEDUA tempat harus ikut
// berubah — utang yang disadari dari menduplikasi sebuah angka, dijaga oleh uji di
// masterlogin_test.go.
const MaxNameLength = 100

// MaxEmailLength adalah panjang maksimum Email. Asumsi, dengan alasan yang sama seperti
// MaxNameLength.
const MaxEmailLength = 100

// MaxPhoneLength adalah panjang maksimum Telp.
//
// Nama kolomnya `TELP`, tetapi properti klipboard yang memetakannya bernama `Ekst` —
// singkatan dari *ekstensi*. Itu bentuk utang yang `03-CURRENT-ARCHITECTURE.md` §4.2 catat:
// properti lama dipakai ulang untuk isi yang berbeda. Yang disimpan tetap nomor telepon,
// sesuai label "Telp" pada gridnya.
//
// Lima puluh, cukup untuk nomor berkode negara beserta ekstensinya.
const MaxPhoneLength = 50

// MaxAddressLength adalah panjang maksimum Alamat.
//
// Properti klipboardnya `BodyLetterTo` — nama yang berasal dari badan surat, bukan dari
// alamat. Lihat catatan pada MaxPhoneLength.
const MaxAddressLength = 250

// SurveyorLogin adalah satu login surveyor — satu baris `POOLDATA.MST_LOGIN_SURVEYOR`.
//
// # Ketujuh kolomnya terbaca lengkap, dan itu sudah dipastikan
//
// Ketiga rule SQL yang menyentuh tabel ini menyebut kolom yang sama persis, dan tidak satu
// pun menyebut kolom di luar ketujuh ini:
//
//	GetLoginMemberSurveyor   select nama, login, email, telp, alamat, stslogin, loginleader
//	GetLoginLeaderSurveyor   insert into ... (nama,login,email,telp,alamat,stslogin,loginleader)
//	UpdateMasterLoginSurvey  update ... set nama,login,email,telp,alamat,stslogin,loginleader
//
// Tidak ada kolom pencatat pelaku, tidak ada stempel waktu, dan tidak ada penanda aktif.
// Akibat yang harus disadari: **siapa yang menambah atau mengubah sebuah login tidak
// tersimpan di mana pun.** Itu keterbatasan tabelnya, bukan kelalaian modul ini.
//
// # Nama field Go dan nama alias Pega sengaja BERBEDA
//
// Rule lama mengalias setiap kolom menjadi properti klipboard yang namanya tidak ada
// hubungannya dengan isinya:
//
//	nama         as "SurveyName"     alamat       as "BodyLetterTo"
//	login        as "SurveyorID"     stslogin     as "ObjectName"
//	email        as "Email"          loginleader  as "NamaPasien"
//	telp         as "Ekst"
//
// `NamaPasien` untuk login seorang leader, dan `ObjectName` untuk status login. Membaca rule
// lama berarti menelusuri ketujuhnya sampai ke pemanggilnya untuk tahu isian mana yang mana.
// Di sini kolomnya disebut menurut ISINYA (`D-19`, `D-80`).
type SurveyorLogin struct {
	// Name adalah kolom NAMA — isian "Nama" pada layar.
	//
	// Ia **bukan sekadar keterangan**: Login diturunkan darinya, sehingga mengubah Name
	// berarti mengubah kunci barisnya. Itulah sebabnya layar Pega mengunci isian ini saat
	// menyunting (`pyDisabledWhen = TempLoginSurvey.pyLabel='Update'`), dan sebabnya modul
	// ini menolaknya di server pula — lihat ErrNameLocked.
	Name string

	// Login adalah kolom LOGIN — kunci baris ini, dan satu-satunya penyaring pada WHERE
	// setiap pernyataan simpan.
	//
	// TIDAK diketik pengguna: ia diturunkan dari Name; lihat DeriveLogin.
	Login string

	// Email adalah kolom EMAIL. Wajib di layar (`pyRequired=true`).
	Email string

	// Phone adalah kolom TELP. Wajib di layar (`pyRequired=true`).
	Phone string

	// Address adalah kolom ALAMAT. Tidak wajib (`pyRequired=false`).
	Address string

	// LoginStatus adalah kolom STSLOGIN.
	//
	// TIDAK digambar di layar mana pun. Setiap penambahan menuliskan LoginStatusMember;
	// penyuntingan mempertahankan nilai yang sudah ada apa adanya.
	LoginStatus string

	// LeaderLogin adalah kolom LOGINLEADER — login orang yang menjadi leader baris ini.
	//
	// TIDAK digambar di layar mana pun. Pada penambahan ia diturunkan dari leader milik
	// pengguna yang menyimpan; lihat usecase.Service.Create.
	LeaderLogin string
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// LIMA isian, satu-lawan-satu dengan kelima kontrol pada
// `Section/BrowseLoginSurveyor-Section.xml`. Yang ADA di tabel tetapi TIDAK di sini:
//
//	LOGIN        diturunkan dari Name; lihat DeriveLogin
//	STSLOGIN     selalu LoginStatusMember pada penambahan
//	LOGINLEADER  diturunkan dari leader milik pengguna yang menyimpan
//
// Ketiganya sengaja tidak dapat dikirim klien. Menerimanya berarti membuka jalan menetapkan
// kunci baris, peran, dan induk tim lewat permintaan HTTP biasa — tiga hal yang di sistem
// lama pun tidak pernah berada di tangan pengguna.
type Input struct {
	Name    string
	Email   string
	Phone   string
	Address string
}

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("masterlogin: login surveyor tidak ditemukan")

	// ErrLoginTaken: LOGIN yang akan disimpan sudah dipakai baris lain.
	//
	// # Pemeriksaan ini DIREKONSTRUKSI, bukan disalin
	//
	// `Activity/SetLoginSurveyor_act` memang memeriksa login ganda, tetapi ia menembak
	// **tabel operator Pega** — `Param.pyReportClass := "Data-Admin-Operator-ID"` lewat
	// report definition `GCNMGetListOfOperators`, disaring `pyUserIdentifier = Param.UserId`.
	// Tabel itu **tidak ada di sistem baru**, dan tidak akan pernah ada: `ADR-0002` menolak
	// membawa engine Pega, dan kontrak identitas `F-3` belum ditetapkan (`R-14`).
	//
	// Yang dipakai sebagai gantinya adalah keunikan `LOGIN` pada
	// `POOLDATA.MST_LOGIN_SURVEYOR` sendiri. Itu bukan pilihan sembarang: `LOGIN` memang
	// kunci alaminya — `UpdateMasterLoginSurvey` menyaring `where login = {...}` — sehingga
	// dua baris berlogin sama membuat penyimpanan yang satu menimpa yang lain tanpa satu pun
	// pesan galat.
	//
	// Pesannya diambil dari `local.msg` pada activity itu apa adanya:
	//
	//	"Login sudah terdaftar dengan nama yang sama"
	//
	// Rekonstruksi ini dinyatakan di sini supaya ia dapat diuji ulang begitu kontrak `F-3`
	// tiba — bukan tersamar sebagai fakta.
	ErrLoginTaken = errors.New("masterlogin: login surveyor sudah dipakai")

	// ErrNameLocked: Nama yang dikirim berbeda dari Nama yang tersimpan.
	//
	// Layar Pega mengunci isian Nama saat menyunting — `pyDisabledWhen` pada kontrolnya
	// berbunyi `TempLoginSurvey.pyLabel='Update'`, dan `Activity/SetLoginSurveyorValue_act`
	// menetapkan `pyLabel := "Update"` tepat saat tombol Ubah menekan baris ke dalam form.
	//
	// Penguncian di antarmuka adalah KENYAMANAN TAMPILAN; permintaan yang tidak datang dari
	// layar itu tidak tersentuh olehnya. Karena itu server ikut menolaknya — perlakuan yang
	// sama dengan isian NAMA pada Master Supplier, yang juga terkunci lewat `pyReadOnly`.
	//
	// Akibatnya bukan kerapian melainkan keutuhan data: Login diturunkan dari Nama, dan
	// Login adalah kunci baris. Nama yang berubah akan menghasilkan Login yang berubah,
	// sedangkan pernyataan simpannya menyaring `where login = <login lama>` — barisnya tidak
	// akan pernah ditemukan, dan penyimpanan "berhasil" tanpa mengubah apa pun.
	ErrNameLocked = errors.New("masterlogin: nama login surveyor tidak dapat diubah")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field adalah nama isian dalam bentuk yang dikenali layar, bukan nama kolom basis
	// data — layar yang menyorot isiannya memakai nilai ini.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja (`P-5`).
type ValidationError struct {
	Violation []Violation
}

// OneViolation membungkus satu pelanggaran menjadi ValidationError.
//
// Dipakai lapisan aplikasi untuk pemeriksaan yang menuntut pembacaan basis data — keunikan
// login — supaya galatnya sampai ke layar dalam bentuk yang SAMA dengan pelanggaran isian
// lain, dan menempel pada isiannya.
func OneViolation(field, message string) error {
	return &ValidationError{Violation: []Violation{{Field: field, Message: message}}}
}

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "masterlogin: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// loginStripped adalah karakter yang dibuang saat Login diturunkan dari Nama.
//
// Keempatnya — spasi, titik, koma, tanda hubung — dan URUTANNYA dibaca apa adanya dari
// `Activity/SetLoginSurveyor_act`:
//
//	local.login := @replaceAll(@replaceAll(@replaceAll(@replaceAll(
//	                 TempLoginSurvey.SurveyName," ",""),".",""),",",""),"-","")
//
// Urutannya tidak berpengaruh pada hasil — keempatnya dibuang, bukan diganti — tetapi ia
// ditulis ulang di sini supaya perbandingan terhadap rule aslinya dapat dilakukan tanpa
// membuka dua berkas sekaligus.
var loginStripped = strings.NewReplacer(" ", "", ".", "", ",", "", "-", "")

// DeriveLogin menurunkan LOGIN dari NAMA.
//
// # Ia BUKAN pilihan modul ini
//
// Sistem lama menurunkannya di layar, pada setiap perubahan isian Nama:
// `Section/BrowseLoginSurveyor-Section.xml` memasang aksi `refresh` bereven `change` pada
// kontrol Nama, yang menjalankan `Activity/SetLoginSurveyor_act`, yang menetapkan
// `TempLoginSurvey.SurveyorID := local.login`.
//
// Penurunannya dipindahkan ke SERVER di sini, dan itu perbedaan yang disengaja: di Pega
// nilainya dihitung di layar lalu dikirim kembali sebagai isian biasa, sehingga permintaan
// yang tidak datang dari layar dapat mengirim Login apa pun. Kunci baris tidak boleh
// bergantung pada kejujuran klien. Layar baru tetap memperlihatkan hasilnya saat pengguna
// mengetik — ia menghitung hal yang sama untuk ditampilkan, bukan untuk dikirim.
//
// # Huruf besar-kecil TIDAK diubah
//
// Rule lamanya tidak memanggil `@toUpperCase` maupun `@toLowerCase` sama sekali. Memaksa
// huruf besar akan mengubah bentuk setiap login yang diterbitkan sesudah ini, dan login
// lama yang sudah ada di tabel tidak akan sebentuk dengannya.
//
// # Kosong adalah hasil yang MUNGKIN
//
// Nama yang seluruhnya terdiri atas keempat karakter yang dibuang menghasilkan teks kosong.
// Itu bukan keadaan yang dijaga fungsi ini — ia hanya menghitung — melainkan keadaan yang
// ditolak Input.Check lewat ErrLoginEmpty, persis seperti langkah kedua
// `CNMInsertMstLoginSurveyor_act`.
func DeriveLogin(name string) string {
	return loginStripped.Replace(strings.TrimSpace(name))
}

// Clean memangkas spasi di kedua ujung setiap isian.
//
// Dipisahkan dari Check supaya nilai yang TERSIMPAN adalah nilai yang sudah dipangkas —
// bukan nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
//
// Tidak ada yang di-UPPERCASE di sini; lihat DeriveLogin.
func (i Input) Clean() Input {
	return Input{
		Name:    strings.TrimSpace(i.Name),
		Email:   strings.TrimSpace(i.Email),
		Phone:   strings.TrimSpace(i.Phone),
		Address: strings.TrimSpace(i.Address),
	}
}

// Check menjalankan seluruh aturan isian dan mengembalikan SEMUA pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
//
// # Yang diwajibkan, dan dari mana asalnya
//
// Ketiganya dibaca dari `pyRequired` pada kontrolnya masing-masing di
// `Section/BrowseLoginSurveyor-Section.xml`:
//
//	Nama   (SurveyName)    pyRequired = true
//	Email  (Email)         pyRequired = true
//	Telp   (Ekst)          pyRequired = true
//	Alamat (BodyLetterTo)  pyRequired = false  → tidak wajib
//
// # Kewajiban itu DITEGAKKAN DI SERVER, dan di Pega tidak
//
// Di sistem lama ketiganya hanya ditandai di layar; `CNMInsertMstLoginSurveyor_act` sendiri
// hanya menolak Login yang kosong. Artinya permintaan yang tidak melewati layar dapat
// menyimpan baris tanpa surel dan tanpa telepon.
//
// Penegakan di server karena itu **DITAMBAHKAN** terhadap sistem lama. Alasannya bukan
// kerapian: surel pada baris ini adalah alamat yang dipakai memberi tahu surveyor tentang
// penugasannya, dan baris tanpa surel gagal diam-diam — tidak ada galat, hanya
// pemberitahuan yang tidak pernah sampai. Baris lama yang sudah kosong tetap DIBACA apa
// adanya; penolakan hanya terjadi saat barisnya disimpan ulang.
//
// # Bentuk surel TIDAK diperiksa
//
// Tidak ada satu pun rule di export yang memeriksanya — bukan pada layar, bukan pada
// activity, dan bukan pada kueri. Menambahkan pemeriksaan bentuk berarti menolak alamat
// yang selama ini diterima sistem lama, dan itu selisih perilaku yang tidak diminta siapa
// pun (`P-5`). Yang diperiksa hanyalah keberadaannya.
func (i Input) Check() error {
	var violation []Violation

	switch {
	case i.Name == "":
		violation = append(violation, Violation{
			Field:   "nama",
			Message: "Nama wajib diisi.",
		})
	case len(i.Name) > MaxNameLength:
		violation = append(violation, Violation{
			Field:   "nama",
			Message: fmt.Sprintf("Nama paling panjang %d karakter.", MaxNameLength),
		})
	case DeriveLogin(i.Name) == "":
		// Padanan langsung langkah kedua `Activity/CNMInsertMstLoginSurveyor_act`, yang
		// menolak penyimpanan saat `TempLoginSurvey.SurveyorID == ""` dengan pesan
		// `local.err := "User Name masih kosong"`.
		//
		// Ia BERBEDA dari "Nama wajib diisi", dan itu sebabnya ia cabang tersendiri: nama
		// yang seluruhnya terdiri atas spasi, titik, koma, dan tanda hubung — misalnya
		// "- . -" — lolos sebagai nama yang terisi, tetapi menghasilkan Login kosong.
		//
		// Ia dilaporkan sebagai pelanggaran ISIAN, bukan sebagai galat tersendiri seperti
		// di Pega: pengguna memperbaikinya dengan mengetik ulang Nama, dan satu-satunya
		// tempat yang berguna menyorotnya adalah isian itu.
		//
		// Pesannya diperluas dengan SEBABNYA. Pesan asli Pega tidak dapat dijelaskan kepada
		// pengguna yang isian Namanya jelas terisi — ia menyebut "User Name", yang bahkan
		// bukan nama isian mana pun di layar itu.
		violation = append(violation, Violation{
			Field: "nama",
			Message: "Nama harus memuat setidaknya satu huruf atau angka. " +
				"Spasi, titik, koma, dan tanda hubung dibuang saat Login dibentuk.",
		})
	}

	switch {
	case i.Email == "":
		violation = append(violation, Violation{
			Field:   "email",
			Message: "Email wajib diisi.",
		})
	case len(i.Email) > MaxEmailLength:
		violation = append(violation, Violation{
			Field:   "email",
			Message: fmt.Sprintf("Email paling panjang %d karakter.", MaxEmailLength),
		})
	}

	switch {
	case i.Phone == "":
		violation = append(violation, Violation{
			Field:   "telp",
			Message: "Telp wajib diisi.",
		})
	case len(i.Phone) > MaxPhoneLength:
		violation = append(violation, Violation{
			Field:   "telp",
			Message: fmt.Sprintf("Telp paling panjang %d karakter.", MaxPhoneLength),
		})
	}

	if len(i.Address) > MaxAddressLength {
		violation = append(violation, Violation{
			Field:   "alamat",
			Message: fmt.Sprintf("Alamat paling panjang %d karakter.", MaxAddressLength),
		})
	}

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// Filter menyaring daftar yang dibaca layar.
//
// # Cakupan daftarnya TIDAK TERBACA dari export, dan itu harus disadari
//
// Grid layar lama terikat page list klipboard `LoginMemberSurvey.pxResults`
// (`pyPageListProperty` pada `Section/BrowseLoginSurveyor-Section.xml`), dan **rule yang
// MENGISI page list itu tidak ada di antara 2.634 berkas export** (`R-16`). Yang ada
// hanyalah pemuat satu baris — `Activity/SetLoginSurveyorValue_act`, yang menembak
// `pyReportContentPage` dan menyaring `where login = <satu login>` untuk tombol Ubah.
//
// Kueri yang benar-benar ada karena itu dibaca apa adanya:
//
//	select nama, login, email, telp, alamat, stslogin, loginleader
//	  from pooldata.mst_login_surveyor {ASIS:InputLogin.IDIndex}
//
// Tanpa penyaring, ia mengembalikan SELURUH baris — dan itulah yang dipakai modul ini.
//
// **Bacaan lain yang mungkin**, dan tidak dapat dibantah maupun dibuktikan dari export:
// daftarnya disaring `LOGINLEADER` = leader milik pengguna yang membukanya, sehingga
// seorang leader hanya melihat anggotanya sendiri. Nama page list-nya
// (`LoginMemberSurvey`), `STSLOGIN` yang selalu "Member", dan keberadaan
// `GetLoginLeaderSurveyor` ketiganya menunjuk ke arah itu.
//
// Perbedaan keduanya menentukan siapa yang boleh menyunting login milik tim lain — dan itu
// pertanyaan terbuka untuk Work Owner, bukan sesuatu yang boleh ditebak modul ini. Sampai
// terjawab, cakupannya mengikuti kueri yang benar-benar ada, dan layar menyatakannya
// terang-terangan.
type Filter struct {
	// Keyword mempersempit daftar pada Nama, Login, dan Email.
	//
	// DITAMBAHKAN terhadap sistem lama, yang memuat seluruh baris ke klipboard lalu
	// memaginasinya 15 baris per halaman tanpa satu pun kotak pencarian. Kosong berarti
	// tanpa penyaring.
	//
	// Ketiganya dicari sekaligus karena ketiganya dihafal orang: Nama yang dicari petugas,
	// Login yang dipakai surveyor, dan Email yang ada di surat. Telp dan Alamat tidak ikut
	// — keduanya tidak pernah menjadi cara orang menyebut seseorang.
	Keyword string
}

// Repo adalah seam ke penyimpanan master login SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat
// kueri (`ADR-0030` Opsi 1).
//
// **Tanpa Delete.** Tidak satu pun rule di export menghapus baris tabel ini, dan `D-66`
// melarang penghapusan fisik data bernilai bisnis. Login yang tidak lagi dipakai tetap ada
// — dan tabelnya tidak punya penanda nonaktif, sehingga tidak ada cara menyatakannya pun.
// Itu keterbatasan yang dicatat, bukan yang ditutupi.
type Repo interface {
	// List mengembalikan baris yang cocok dengan penyaring.
	List(ctx context.Context, filter Filter) ([]SurveyorLogin, error)

	// Get mengembalikan satu baris menurut LOGIN-nya; ErrNotFound bila tidak ada.
	//
	// Padanan `RDB List/GetLoginMemberSurveyor-SQL.xml` sebagaimana dipakai
	// `Activity/SetLoginSurveyorValue_act` untuk memuat baris ke form — dengan satu
	// perbedaan yang menentukan: di sana penyaringnya DIRANGKAI dari teks,
	// `"where login = '" + Param.operatorid + "'"`, tanpa satu pun pelolosan. Itu persis
	// celah `{ASIS:...}` yang `03-CURRENT-ARCHITECTURE.md` §4.5 catat. Di sini ia parameter.
	Get(ctx context.Context, login string) (SurveyorLogin, error)

	// FindLeaderOf mengembalikan isi LOGINLEADER milik satu login; ErrNotFound bila
	// barisnya tidak ada.
	//
	// Padanan `RDB List/GetLoginLeaderSurveyor-SQL.xml`:
	//
	//	select loginleader as "NamaPasien" from pooldata.mst_login_surveyor
	//	 where login = {OperatorID.pyUserIdentifier}
	//
	// Perhatikan APA yang dicari: bukan leader dari baris yang sedang dibuat, melainkan
	// leader dari **pengguna yang sedang menyimpan**. Lihat usecase.Service.Create.
	FindLeaderOf(ctx context.Context, login string) (string, error)

	// Insert menyisipkan baris baru dan mengembalikan baris yang benar-benar tersimpan.
	//
	// Pemeriksaan keunikan LOGIN berada DI DALAM operasi ini, bukan sebagai langkah
	// terpisah sebelumnya. Sistem lama memecahnya — pemeriksaan dilakukan di layar saat
	// Nama diketik, penyisipannya menyusul saat Simpan ditekan — dan jarak di antara
	// keduanya tidak dijaga apa pun. Lihat ErrLoginTaken.
	Insert(ctx context.Context, one SurveyorLogin) (SurveyorLogin, error)

	// Update menyimpan perubahan pada baris yang sudah ada; ErrNotFound bila barisnya
	// hilang di antara pemuatan layar dan penyimpanan.
	//
	// Padanan `RDB List/UpdateMasterLoginSurvey-SQL.xml`, yang menulis KETUJUH kolomnya
	// sekaligus dan menyaring `where login = {...}`.
	Update(ctx context.Context, one SurveyorLogin) error
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)
