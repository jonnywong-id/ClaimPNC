// Package masterreas adalah inti modul Master Reas.
//
// # Apa yang dimodelkan di sini
//
// Daftar **member reasuransi** — satu baris per pihak yang menerima pemberitahuan PLA,
// Pre-DLA, dan DLA, beserta login portal dan alamat surelnya. Tabelnya
// `POOLDATA.T_REINSURER`.
//
// # Modul ini TIDAK MENULIS, dan itu keputusan yang berdasar bukti
//
// Layar lamanya — `Harness/DataMemberReas-harness.xml`, MENU_ID 35 — memuat **satu grid dan
// satu tombol Refresh**, dan tidak ada satu pun jalur tulis yang sampai ke sana. Satu-satunya
// penulis tabel ini di sistem lama adalah **alur PLA/DLA**, bukan layar master:
//
//	Database/UPDATEREAS.prc              prosedur upsert-nya
//	RDB List/UpdateEmailReas-SQL.xml     satu-satunya pemanggil prosedur itu
//	Activity/UpdateDetailPLA2-Act.xml    memanggil UpdateEmailReas  ← layar detail PLA
//	Activity/UpdateDetailDLA2-Act.xml    memanggil UpdateEmailReas  ← layar detail DLA
//
// Jadi baris reasuransi lahir dan berubah sebagai **efek samping pengiriman PLA/DLA**
// (`B-9`), bukan lewat pemeliharaan master. Modul ini karena itu hanya membaca — meniru
// sistem lama apa adanya (`P-5`), bukan menambah kewenangan yang tidak pernah ada.
//
// Bukti kedua, dan ia HANYA berlaku untuk tombol Tambah: indeks rule di dalam harness
// menyebut **satu** label tombol saja, `pyButtonLabel Refresh`. Dikalibrasi terhadap
// `Harness/MasterLoginSurvey-Harness.xml` — yang layarnya terbukti punya dua tombol — indeks
// itu memang menyebut **keduanya**, `REFRESH` dan `TAMBAH`. Jadi ketiadaan `TAMBAH` di sini
// berarti sesuatu.
//
// # Yang TIDAK dapat dibuktikan: tombol Ubah per baris
//
// Kalibrasi yang sama menunjukkan indeks harness hanya menjangkau **section teratas**. Pada
// Master Login ia memuat rujukan milik `LOGINSURVEYOR` tetapi **tidak** milik
// `BROWSELOGINSURVEYOR` — padahal di section gridlah tombol **Ubah** dan form-nya berada.
//
// Susunannya di sini identik: indeks memuat `LISTMEMBERREAS`, dan **tidak** memuat
// `BROWSELISTMEMBERREAS` yang memang hilang dari export.
//
// Penelusuran activity pun tidak dapat menolong: activity Pega **tidak menyebut nama
// section**. Ketiga activity Master Login memuat **nol** kemunculan `BrowseLoginSurveyor`;
// yang mereka sebut adalah halaman klipboardnya (`TempLoginSurvey`, 7–18 kali). Nama halaman
// klipboard layar Master Reas tidak diketahui, karena section yang mendeklarasikannya hilang.
//
// **Jadi tombol Ubah per baris TIDAK dapat disingkirkan.** Yang menahan modul ini tetap
// baca-saja adalah bukti pertama — tidak ada satu pun jalur tulis ke `T_REINSURER` di seluruh
// export yang berpangkal di layar ini — bukan ketiadaan tombolnya.
//
// **Yang harus disadari:** section grid `BrowseListMemberReas` **tidak ada di export**
// (`R-16`), sehingga "tanpa tombol tambah dan ubah" adalah **rekonstruksi dari bukti yang
// ada**, bukan hal yang terbukti mustahil. Bila kelak terbukti layar lamanya punya tombol
// simpan, yang perlu ditambahkan adalah Repo.Insert/Update beserta rutenya — bentuk domain
// di berkas ini tidak perlu berubah.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/DataMemberReas-harness.xml         layar, MENU_ID 35, grid + tombol Refresh
//	RDB List/BrowseEmailReas-SQL.xml           5 kolom terbaca + insert 5 kolom + filter TYPE
//	RDB List/GetListDataLoginReas-SQL.xml      4 kolom terbaca + insert 4 kolom
//	RDB List/GetDataPreDLA-SQL.xml             arti TYPE: substr(NODLA,0,1) = TYPE
//	RDB List/GetPNCList_PLA1-SQL.xml           LOGIN menyaring klaim yang dilihat mitra
//	RDB List/GetPNCList_PLADLA-SQL.xml         idem
//	RDB List/BrowseCommunicationReas-SQL.xml   idem
//	Database/UPDATEREAS.prc                    kunci alami, dan kolom COUNTRYID
//	Database/INSERT_PLADLA.prc                 urutan pencarian baris saat PLA/DLA terbit
//	Database/m_menu_aplikasi_pnc.csv           MENU_ID 35 "Master Reas"
//	Database/m_otorisasi_pnc.csv               satu-satunya grup berwenang: IT
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterreas

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// MaxEmailLength adalah panjang maksimum Email.
//
// ASUMSI YANG DISADARI, bukan angka dari DDL: `POOLDATA.T_REINSURER` tidak ada DDL-nya di
// export (`R-08`), dan layar lamanya tidak dapat dibaca karena section gridnya hilang
// (`R-16`).
//
// Batasnya tetap dipasang karena tanpa itu penolakan datang dari basis data sebagai
// ORA-12899 — galat teknis yang tidak menuntun pengguna ke mana pun. Seratus dipilih agar
// sama dengan batas surel pada modul master lain di aplikasi ini.
const MaxEmailLength = 100

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang hendak diubah tidak ada.
	//
	// Pada modul ini ia lebih mungkin terjadi daripada di modul master lain: kunci alaminya
	// TIGA kolom, dan baris reasuransi dapat berubah di belakang layar oleh alur PLA/DLA
	// yang menulis ke tabel yang sama (`Database/UPDATEREAS.prc`).
	ErrNotFound = errors.New("masterreas: member reas tidak ditemukan")
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

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "masterreas: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Input adalah nilai yang dikirim pengguna dari form ubah.
//
// # SATU isian, dan itu bukan penyederhanaan
//
// `Database/UPDATEREAS.prc` pada baris yang SUDAH ADA hanya menyentuh satu kolom:
//
//	UPDATE POOLDATA.T_REINSURER SET EMAIl = tEMAIl
//	 WHERE REINSURERID = tREINSID AND REINSURERNAME = tREINSNAME AND TYPE = tTYPE
//
// `LOGIN`, `COUNTRY`, dan `COUNTRYID` hanya ditulis pada jalur **sisip**, tidak pernah pada
// jalur ubah. Menawarkan keduanya sebagai isian berarti menambah kewenangan yang sistem lama
// tidak pernah punya — dan `LOGIN` bukan kolom sembarangan: ia menentukan klaim mana yang
// dilihat seorang mitra reasuransi (lihat Member.Login).
//
// Ketiga kolom kunci TIDAK ada di sini. Ia dikirim terpisah sebagai Key; lihat Repo.Update.
type Input struct {
	Email string
}

// Clean memangkas spasi di kedua ujung isian.
//
// Dipisahkan dari Check supaya nilai yang TERSIMPAN adalah nilai yang sudah dipangkas —
// bukan nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
func (i Input) Clean() Input {
	return Input{Email: strings.TrimSpace(i.Email)}
}

// Check menjalankan seluruh aturan isian dan mengembalikan SEMUA pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
//
// # Surel WAJIB terisi, dan itu DITAMBAHKAN terhadap sistem lama
//
// `UPDATEREAS` tidak memeriksa apa pun — ia menerima surel kosong dan menuliskannya. Akibat
// yang ditimbulkannya tidak terlihat sebagai galat: dokumen PLA/DLA tetap terbit, tetap
// tercatat terkirim, dan tidak pernah sampai ke siapa pun.
//
// Baris lama yang surelnya SUDAH kosong tetap DIBACA apa adanya dan tetap tampil di daftar;
// penolakan hanya terjadi saat seseorang menyimpan baris itu dengan surel kosong.
//
// # Bentuk surel TIDAK diperiksa
//
// Tidak ada satu pun rule di export yang memeriksanya. Menambahkan pemeriksaan bentuk
// berarti menolak alamat yang selama ini diterima sistem lama, dan itu selisih perilaku yang
// tidak diminta siapa pun (`P-5`). Yang diperiksa hanyalah keberadaannya.
func (i Input) Check() error {
	var violation []Violation

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

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// Key adalah kunci alami satu baris — KETIGA kolomnya.
//
// Ia tipe tersendiri, bukan tiga argumen berjajar, karena ketiganya harus bergerak bersama.
// Tiga string berurutan adalah undangan untuk tertukar: `REINSURERID` dan `TYPE` sama-sama
// pendek, dan kompilator tidak akan menolong bila keduanya bertukar tempat.
//
// Lihat Member.NaturalKey untuk asal kuncinya.
type Key struct {
	ReinsurerID   string
	ReinsurerName string
	Type          string
}

// Clean memangkas spasi di ketiga bagian kunci.
func (k Key) Clean() Key {
	return Key{
		ReinsurerID:   strings.TrimSpace(k.ReinsurerID),
		ReinsurerName: strings.TrimSpace(k.ReinsurerName),
		Type:          strings.TrimSpace(k.Type),
	}
}

// IsEmpty menyatakan kunci tidak cukup untuk menunjuk satu baris.
//
// `Type` BOLEH kosong — baris yang lahir dari `GetListDataLoginReas` disisipkan tanpa kolom
// itu sama sekali, sehingga kunci ber-Type kosong adalah kunci yang sah. Yang tidak boleh
// kosong adalah kode dan nama reasuransinya.
func (k Key) IsEmpty() bool {
	return k.ReinsurerID == "" || k.ReinsurerName == ""
}

// FallbackType adalah nilai TYPE yang diperlakukan sistem lama sebagai baris CADANGAN.
//
// Ia terbaca dari dua tempat yang saling menguatkan:
//
//	BrowseEmailReas   where reinsurerid = {...} and (type = {TempReasPLA.NoPLA} or type = '1')
//	UPDATEREAS.prc    ... AND TYPE = '1' ; lalu UPDATE ... SET TYPE = tTYPE
//
// Bacaannya: saat PLA/DLA dikirim, sistem mencari baris yang TYPE-nya cocok dengan jenis
// dokumen itu; bila tidak ada, ia memakai baris ber-TYPE `'1'`. Dan saat menyimpan,
// `UPDATEREAS` **mengubah baris `'1'` itu menjadi tipe yang diminta** alih-alih menyisipkan
// baris baru — sehingga baris cadangan dapat habis terpakai.
//
// Konstanta ini TIDAK dipakai untuk menyaring apa pun di modul ini; ia ada supaya layar
// dapat menandai baris cadangan, dan supaya arti `'1'` tidak perlu ditemukan ulang oleh
// pembaca berikutnya.
const FallbackType = "1"

// Member adalah satu member reasuransi — satu baris `POOLDATA.T_REINSURER`.
//
// # Tujuh kolom, dan hanya ENAM yang dibaca siapa pun
//
// Ketujuh kolomnya terbaca lengkap dari `UPDATEREAS.prc`, yang menyisipkan semuanya
// sekaligus:
//
//	INSERT INTO POOLDATA.T_REINSURER
//	  (REINSURERID, REINSURERNAME, LOGIN, EMAIl, COUNTRY, COUNTRYID, TYPE)
//
// **`COUNTRYID` ditulis, tetapi tidak dibaca satu pun rule di seluruh export.** Ia
// diterjemahkan dari COUNTRY lewat `SELECT ID FROM COUNTRY WHERE COUNTRY = tCOUNTRY` di
// dalam prosedur itu, lalu tidak pernah dipakai lagi — tidak oleh SELECT mana pun, tidak
// oleh laporan, dan tidak oleh dokumen PLA/DLA. Karena itu ia TIDAK ada di sini: membawa
// kolom yang tidak ada pembacanya berarti mengarang kegunaan yang tidak dapat ditunjukkan.
//
// # Nama field Go dan alias Pega sengaja BERBEDA
//
// Rule lama mengalias setiap kolom menjadi properti klipboard yang namanya tidak ada
// hubungannya dengan isinya — persis bentuk utang yang `03-CURRENT-ARCHITECTURE.md` §4.2
// catat:
//
//	email          as "City"          reinsurerid  as "CityID"
//	reinsurername  as "District"       country      as "Country"
//	login          as "DistrictID"
//
// Alamat surel dialiaskan menjadi "City", dan nama perusahaan reasuransi menjadi "District".
// Membaca rule lama berarti menelusuri kelimanya sampai ke pemanggilnya untuk tahu isian
// mana yang mana. Di sini kolomnya disebut menurut ISINYA (`D-19`, `D-80`).
type Member struct {
	// ReinsurerID adalah kolom REINSURERID — kode perusahaan reasuransi.
	//
	// Ia yang menghubungkan baris ini ke dokumen PLA/DLA: `T_PLALIST.REINSCODE`,
	// `T_DLALIST.REINSCODE`, `T_PLA_XOL.IDREAS`, dan `T_DLA_XOL.IDREAS` seluruhnya
	// menunjuk ke sini.
	ReinsurerID string

	// ReinsurerName adalah kolom REINSURERNAME — nama perusahaan reasuransi.
	//
	// Ia bagian dari KUNCI ALAMI, bukan sekadar keterangan; lihat NaturalKey.
	ReinsurerName string

	// Login adalah kolom LOGIN — identitas yang dipakai mitra reasuransi saat masuk.
	//
	// # Kolom yang taruhannya paling tinggi di tabel ini
	//
	// Lima kueri inbox menyaring klaim yang boleh dilihat seseorang dengan membandingkan
	// kolom ini terhadap identitas pemanggil:
	//
	//	GetPNCList_PLA1          where login = {OperatorID.pyUserIdentifier}
	//	GetPNCList_PLADLA        idem
	//	GetPNCList_PLADLAClose   idem
	//	BrowseCommunicationReas  idem
	//	SetDataPLADLA            where login = '<login reas>'
	//
	// Artinya nilai di kolom ini menentukan **klaim milik siapa yang tampil di layar
	// seorang mitra**. Satu nilai yang keliru memindahkan visibilitas klaim ke mitra lain,
	// dan kegagalannya tidak terlihat sebagai galat — layarnya tampil normal.
	//
	// # Ia TIDAK pernah diketik orang
	//
	// `Activity/FindDataReinsurer-Act.xml` menurunkannya dari nama reasuransi dengan
	// membuang spasi, titik, koma, tanda hubung, dan tanda kurung tutup:
	//
	//	@replaceAll(@replaceAll(@replaceAll(@replaceAll(@replaceAll(
	//	  TempReasPLA.PLAReinsurer," ",""),".",""),",",""),"-",""),")","")
	//
	// lalu menambahkan akhiran `_<index>` bila bentrok dengan nama reasuransi lain. Itu
	// berjalan di alur PLA/DLA, bukan di layar master — dan itulah sebab modul ini tidak
	// menawarkan cara mengubahnya.
	Login string

	// Email adalah kolom EMAIL — alamat tujuan pemberitahuan PLA/DLA.
	//
	// Satu-satunya kolom yang `UPDATEREAS` ubah pada baris yang sudah ada; ketiga cabang
	// UPDATE-nya menyentuh EMAIL, dan hanya satu yang ikut menyentuh TYPE.
	Email string

	// Country adalah kolom COUNTRY — negara asal perusahaan reasuransi.
	//
	// Dibaca `GetDataPreDLA` dan `BrowseAllDataXOL_PLA` untuk ditempatkan pada dokumen
	// PLA/DLA, jadi isinya benar-benar sampai ke pihak luar.
	//
	// Nilainya TEKS NEGARA, bukan kode: `UPDATEREAS` mencarinya dengan
	// `WHERE COUNTRY = tCOUNTRY`. **Daftar negara yang sah tidak diketahui** — tabel
	// `COUNTRY` tidak dibaca satu pun rule di seluruh export (`R-16`), dan DDL-nya tidak
	// ada (`R-08`).
	Country string

	// Type adalah kolom TYPE.
	//
	// # Artinya terbaca, kegunaannya terbaca, PENAMAANNYA tidak
	//
	// Yang terbukti: ia dicocokkan dengan **karakter pertama nomor dokumen PLA/DLA**.
	//
	//	GetDataPreDLA   WHERE REINSURERID = a.REINSCODE and substr(a.NODLA,0,1) = TYPE
	//	BrowseEmailReas where reinsurerid = {...} and (type = {NoPLA} or type = '1')
	//
	// Jadi satu perusahaan reasuransi dapat punya BEBERAPA baris — satu per jenis dokumen —
	// dengan surel yang berbeda-beda, dan `FallbackType` dipakai bila jenisnya tidak ada.
	//
	// Yang TIDAK terbukti adalah apa arti tiap nilainya dalam bahasa bisnis. Tidak ada
	// master, tidak ada daftar nilai sah, dan tidak ada satu pun rule yang menerjemahkannya
	// menjadi label. Karena itu namanya di sini mengikuti nama kolomnya apa adanya alih-alih
	// diberi nama yang mengaku tahu artinya.
	Type string
}

// NaturalKey mengembalikan kunci alami baris ini.
//
// # Kuncinya TIGA kolom, bukan satu
//
// `Database/UPDATEREAS.prc` memeriksa keberadaan baris dengan
//
//	WHERE REINSURERID = tREINSID AND REINSURERNAME = tREINSNAME AND TYPE = tTYPE
//
// Ketiganya sekaligus. Ini BERBEDA dari Master Login, yang kuncinya satu kolom — dan
// perbedaannya menentukan bentuk layar: satu nama perusahaan reasuransi dapat muncul
// beberapa kali di daftar, dan yang membedakannya adalah TYPE.
//
// # Ia tidak dijamin unik oleh basis data
//
// Tidak ada DDL-nya (`R-08`), sehingga tidak diketahui apakah ada constraint unik. Bukti
// justru mengarah sebaliknya: `GetPNCList_PLA1-SQL.xml` mengambil ReinsurerID dari LOGIN
// dengan `order by reinsurerid desc fetch next 1 row only` — sistem lama **tahu** satu login
// dapat menunjuk beberapa baris, dan menyelesaikannya dengan memilih yang terbesar.
//
// Fungsi ini karena itu dipakai sebagai kunci BARIS DI LAYAR, bukan sebagai jaminan
// keunikan. Pemisahnya `\x1f` (unit separator) — bukan tanda baca biasa yang dapat muncul
// di dalam nama perusahaan dan membuat dua baris berbeda menghasilkan kunci yang sama.
func (m Member) NaturalKey() string {
	return strings.Join([]string{m.ReinsurerID, m.ReinsurerName, m.Type}, "\x1f")
}

// IsFallback menyatakan baris ini adalah baris cadangan; lihat FallbackType.
func (m Member) IsFallback() bool { return strings.TrimSpace(m.Type) == FallbackType }

// Filter menyaring daftar yang dibaca layar.
//
// # Cakupan daftarnya adalah REKONSTRUKSI, dan itu harus disadari
//
// Section grid layar lama (`BrowseListMemberReas`) **tidak ada di antara 2.634 berkas
// export** (`R-16`), sehingga tidak dapat dipastikan apakah gridnya menampilkan seluruh
// baris atau disaring lebih dulu.
//
// Kueri yang benar-benar ada atas tabel ini seluruhnya BERPENYARING, dan penyaringnya
// selalu kode atau nama satu reasuransi tertentu — keduanya milik alur PLA/DLA, bukan milik
// layar master:
//
//	BrowseEmailReas        where reinsurerid = {TempReasPLA.ReinsCode} and (type = ... )
//	GetListDataLoginReas   where reinsurername = {TempReasPLA.PLAReinsurer}
//
// Layar yang berjudul "Data Member" dan bertombol Refresh tidak punya keduanya untuk diisi.
// Yang dipakai modul ini karena itu adalah **seluruh baris entitas**, yaitu bentuk kueri itu
// tanpa klausa WHERE-nya.
type Filter struct {
	// Keyword mempersempit daftar pada ReinsurerID, ReinsurerName, Login, dan Email.
	//
	// DITAMBAHKAN terhadap sistem lama, yang tidak punya kotak pencarian sama sekali di
	// layar ini. Kosong berarti tanpa penyaring.
	//
	// Country tidak ikut dicari: ia penggolongan, bukan cara seseorang menyebut sebuah
	// perusahaan. Type juga tidak — nilainya satu karakter, sehingga mencarinya akan
	// mencocokkan hampir setiap baris.
	Keyword string
}

// Clean memangkas spasi di kedua ujung penyaring.
func (f Filter) Clean() Filter {
	return Filter{Keyword: strings.TrimSpace(f.Keyword)}
}

// Repo adalah seam ke penyimpanan member reasuransi SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans selalu terikat pada satu
// basis data entitas — pemisahan antarentitas ada di tingkat koneksi, bukan di tingkat
// kueri (`ADR-0030` Opsi 1).
//
// # List dan Update saja — tanpa Insert, tanpa Delete
//
// **Tanpa Insert.** Layar lamanya tidak punya tombol Tambah, dan itu terkalibrasi: indeks
// rule `Harness/MasterLoginSurvey-Harness.xml` — layar yang terbukti punya dua tombol —
// menyebut `PYBUTTONLABEL!REFRESH` **dan** `!TAMBAH`; `DataMemberReas` hanya menyebut
// `REFRESH`. Baris baru lahir dari alur PLA/DLA, lewat jalur sisip `UPDATEREAS` dan kedua
// `pySaveSQL` pada `BrowseEmailReas`/`GetListDataLoginReas`.
//
// **Tanpa Delete.** Tidak satu pun rule di export menghapus baris tabel ini, dan `D-66`
// melarang penghapusan fisik data bernilai bisnis.
//
// **Tanpa Get.** Form ubah dimuat dari baris yang SUDAH ada di daftar, sama seperti layar
// lamanya memuat grid lebih dulu. Menambahkan pengambilan satu baris berarti menyediakan
// jalur yang tidak ada pemakainya, dengan kunci tiga kolom yang harus dipaksakan ke jalur URL.
type Repo interface {
	// List mengembalikan baris yang cocok dengan penyaring.
	List(ctx context.Context, filter Filter) ([]Member, error)

	// Update mengubah SURAT ELEKTRONIK satu baris; ErrNotFound bila barisnya tidak ada.
	//
	// Padanan cabang pertama `Database/UPDATEREAS.prc`:
	//
	//	UPDATE POOLDATA.T_REINSURER SET EMAIl = tEMAIl
	//	 WHERE REINSURERID = tREINSID AND REINSURERNAME = tREINSNAME AND TYPE = tTYPE
	//
	// # Yang TIDAK dibawa dari prosedur itu
	//
	// `UPDATEREAS` punya sembilan cabang: ia menyisipkan bila barisnya tidak ada, dan
	// **menaikkan baris ber-TYPE `'1'` menjadi tipe yang diminta** bila tipe yang diminta
	// belum ada. Keduanya adalah penulisan yang diminta alur PLA/DLA, bukan yang diminta
	// petugas yang menekan Simpan di layar master.
	//
	// Di sini perilakunya tegas: baris yang tidak ada **ditolak**, tidak disisipkan diam-diam
	// — dan baris cadangan tidak pernah berubah tipenya. Petugas yang mengubah surel satu
	// baris tidak boleh tanpa sadar menciptakan baris baru maupun memindahkan baris cadangan
	// milik jenis dokumen lain.
	//
	// `COMMIT` di dalam prosedur juga tidak dibawa; kepemilikan transaksi ada di Go (`D-68`).
	Update(ctx context.Context, key Key, email string) error
}

// RepoSelector memilih Repo milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada saat
// permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menampilkan daftar mitra
// reasuransi satu badan hukum kepada pengguna badan hukum lain tanpa satu pun pesan galat
// (`R-20`).
type RepoSelector func(portalAlias string) (Repo, error)
