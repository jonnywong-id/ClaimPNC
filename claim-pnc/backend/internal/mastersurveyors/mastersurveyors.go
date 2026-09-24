// Package mastersurveyors adalah inti modul Master Surveyors (`F-4`, `MENU_ID 15`).
//
// # Apa yang dimodelkan di sini
//
// Daftar ORANG dan LEMBAGA yang melakukan survei kerugian: surveyor internal ASM, loss
// adjuster, expert, dan survey agent. Ia anak dari Master Tipe Surveyors — tipenya di
// sana, orangnya di sini:
//
//	Tipe Surveyor   POOLDATA.M_SURVEYORS   <- paket mastertipesurveyors
//	  └─ Surveyor   POOLDATA.D_SURVEYORS   <- yang dikerjakan paket ini
//
// Seorang surveyor tidak langsung dapat ditugaskan. Ia menunggu persetujuan komite lebih
// dulu, persis seperti rekening pada `masterrekening`.
//
// # Kenapa master ini punya alur persetujuan, sementara master lain tidak
//
// Surveyor menentukan SIAPA yang menilai besarnya kerugian, dan penilaian itulah yang
// menjadi dasar nilai yang dibayarkan. Ia menyentuh uang dengan cara yang sama seperti
// rekening menyentuh uang — hanya di hulu, bukan di hilir.
//
// Sistem lama sudah memperlakukannya begitu, dan buktinya bukan tafsiran: kolom APPROVAL,
// KOMITE, dan TRFKOMITE ada di POOLDATA.D_SURVEYORS, dan ada EMPAT section terpisah untuk
// keempat posisinya — `BrowseDetailSuveryorsWaiting`, `-Approve`, `-Reject`, `-Komite`.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/DetailSurveyorsInbox-Harness.xml        layar "Detail Surveyors"
//	Section/GridDetailSurveyors-Section.xml          kerangka layar; tombol Tambah & Refresh
//	Section/BrowseDetailSuveryors{,Waiting,Approve,Reject,Komite}-Section.xml
//	                                                 grid per posisi persetujuan
//	Report Definition/BrowseVDSurveyors_RD-RD.xml    19 kolom daftar, pyMaxRecords=500
//	Report Definition/SelectVDSurveyors_RD-RD.xml    ambil satu baris untuk disunting
//	Activity/SetDetailSurveryorsValue_act-Act.xml    aksi Ubah: 16 field disalin ke temp
//	Activity/CNMInsertDetailSurveyors_act-Act.xml    aksi Simpan, 20 langkah
//	Activity/ValidasiMasterSurveyor-Act.xml          uji nama ganda
//	Activity/GCNMCreateOperator-Act.xml              pembuatan akun aplikasi surveyor
//	RDB List/UpdateDetailSurveyors-SQL.xml           memanggil POOLDATA.PEGA_D_SURVEYORS
//	Database/PEGA_D_SURVEYORS.prc                    source procedure-nya
//	RDB List/BrowseSurveyorType{LossAdjuster,Expert,SurveyAgent}-SQL.xml
//	                                                 pembaca V_D_SURVEYORS saat penugasan
//
// # Keputusan komite: satu activity, tiga nilai parameter
//
// Tidak ada rule Approve/Reject tersendiri di sistem lama — ia sempat disangka hilang
// (`R-16`) sampai pemanggilnya ditelusuri pada 2026-09-20. Ketiga tombolnya memanggil
// activity yang SAMA, dibedakan satu parameter:
//
//	Approve  →  CNMInsertDetailSurveyors_act( approval = "1" )
//	Reject   →  CNMInsertDetailSurveyors_act( approval = "2" )
//	Simpan   →  CNMInsertDetailSurveyors_act( approval = "0" )
//
// Kedua tombol keputusan hanya ada di tab "Komite Approval"
// (`Section/BrowseDetailSuveryorsKomite-Section.xml`). Rinciannya di usecase/decide.go.
//
// # SEBAGIAN STRUKTUR D_SURVEYORS BELUM DIVERIFIKASI KE KATALOG BASIS DATA
//
// Work Owner menetapkan modul ini dirancang dari export Pega saja, sehingga panjang kolom,
// nama constraint, dan tipe datanya masih dugaan — lihat MaxNameLength dan PrimaryKeyName.
//
// Satu hal yang SUDAH tertutup: `V_D_SURVEYORS` membaca KOLOM, bukan JSON_DATA (ditegaskan
// Work Owner 2026-09-20). Artinya tulisan modul ini langsung terlihat rule Pega tanpa
// perubahan view apa pun, dan jebakan yang menimpa modul induk tidak berlaku di sini.
// Catatan lengkapnya di repo/sqlstore.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package mastersurveyors

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// InternalTypeCode adalah M_SURVEY_ID milik tipe "INTERNAL SURVEYOR".
//
// Ia dipatok, dan itu BUKAN kelalaian yang menunggu dibersihkan — ia memang dipatok di
// sistem lama, di dua tempat yang keduanya menentukan perilaku:
//
//	Activity/CNMInsertDetailSurveyors_act-Act.xml
//	    param.unitName := @If(TempDetailSurveyors.M_SURVEY_ID=="1001","Internal","Eksternal")
//	RDB List/BrowseSurveyorTypeLossAdjuster-SQL.xml   m_survey_id in ('1002')
//	RDB List/BrowseSurveyorTypeExpert-SQL.xml         m_survey_id in ('1003')
//	RDB List/BrowseSurveyorTypeSurveyAgent-SQL.xml    m_survey_id in ('1004')
//
// `mastertipesurveyors` sudah mencatat akibatnya: kode tipe tidak boleh berubah setelah
// terbit, karena mengubahnya memutus ketiga kueri itu sekaligus. Di sini konsekuensinya
// satu tingkat lebih jauh — kode "1001" menentukan DUA aturan sekaligus: kewajiban login
// aplikasi, dan unit organisasi akun yang dibuat.
const InternalTypeCode = "1001"

// MaxNameLength membatasi panjang nama surveyor.
//
// Angkanya sama dengan master lain yang sudah dibangun (`masterstatus.MaxLabelLength`,
// `masterstatusprogres.MaxNameLength`, `mastertipesurveyors.MaxDescriptionLength`) supaya
// batas yang dilihat pengguna seragam antarlayar master.
//
// BELUM DIVERIFIKASI ke katalog basis data atas keputusan Work Owner — panjang kolom
// POOLDATA.D_SURVEYORS.NAME yang sebenarnya tidak diketahui, dan layar Pega tidak
// membatasi apa pun (`pyMaxLength` kosong). Bila kolomnya ternyata lebih pendek dari 100,
// penyisipan akan ditolak basis data dan angka ini harus turun. Ia asumsi yang dinyatakan,
// bukan fakta yang dibaca.
const MaxNameLength = 100

// ApprovalStatus adalah posisi seorang surveyor dalam alur persetujuan komite.
//
// Nilainya sengaja tetap "0", "1", "2" seperti di POOLDATA.D_SURVEYORS: tabelnya masih
// dibaca dan ditulis sistem lama selama masa paralel (`D-21`, `ADR-0004`), sehingga
// mengubah sandi nilainya akan membuat kedua sistem membaca baris yang sama secara
// berbeda.
type ApprovalStatus string

const (
	// StatusPending — surveyor sudah diajukan, komite belum memutuskan.
	StatusPending ApprovalStatus = "0"
	// StatusApproved — komite menyetujui; surveyor dapat ditugaskan pada klaim.
	StatusApproved ApprovalStatus = "1"
	// StatusRejected — komite menolak.
	StatusRejected ApprovalStatus = "2"
)

// Label mengembalikan sebutan status dalam bahasa yang dibaca pengguna.
//
// Sebutannya disamakan dengan `masterrekening` — "Committee Approve" dan "Committee
// Reject" — karena keduanya layar persetujuan komite yang dibaca peran yang sama, dan dua
// sebutan berbeda untuk keadaan yang sama akan terbaca sebagai dua hal berbeda.
func (s ApprovalStatus) Label() string {
	switch s {
	case StatusPending:
		return "Menunggu"
	case StatusApproved:
		return "Committee Approve"
	case StatusRejected:
		return "Committee Reject"
	default:
		return ""
	}
}

// Known menyatakan status ini termasuk salah satu dari tiga yang sah.
func (s ApprovalStatus) Known() bool {
	return s == StatusPending || s == StatusApproved || s == StatusRejected
}

// Surveyor adalah satu baris master surveyor.
//
// Nama field adalah nama domain berbahasa Inggris (`D-80`), BUKAN nama kolom. Pemetaan
// nama domain ke kolom hidup di satu tempat saja, repo/sqlstore — sehingga membaca paket
// ini tidak menuntut tahu apa pun tentang bentuk tabelnya.
//
// Kedelapan belas field di bawah adalah SELURUH kolom yang dibaca `BrowseVDSurveyors_RD`
// dan `SelectVDSurveyors_RD`. Tidak ada yang ditambahkan, tidak ada yang dibuang.
type Surveyor struct {
	// ID adalah D_SURVEY_ID. Dibuat sistem saat surveyor ditambahkan dan TIDAK PERNAH
	// berubah sesudahnya.
	//
	// Bentuknya kode situs ditambah enam digit — `Database/PEGA_D_SURVEYORS.prc:19`:
	//
	//	id_surv_ins := id_site || lpad(to_Char(D_SURVEYORS_SEQ.nextval),6,'0')
	//
	// Perhatikan bedanya dari tipe surveyor, yang memakai TIGA digit. Keduanya memakai
	// urutan yang berlainan (`D_SURVEYORS_SEQ` versus `M_SURVEYORS_SEQ`) dan tidak boleh
	// tertukar.
	ID string

	// LegacyID adalah OLD_D_SURVEY_ID, jejak penomoran sistem sebelumnya.
	//
	// Dibawa karena ia bagian dari kontrak view yang dibaca Pega, dan karena ia dipakai
	// `UpdateDetailSurveyors-SQL.xml` sebagai pembawa muatan JSON — bukan sebagai kode
	// lama. Tidak pernah diisi untuk surveyor baru: ia jejak sejarah, bukan field yang
	// dikelola.
	LegacyID string

	// TypeCode adalah M_SURVEY_ID, merujuk baris di POOLDATA.M_SURVEYORS.
	//
	// Ia bukan sekadar penggolongan tampilan: nilainya menentukan apakah AppLogin wajib
	// (lihat RequiresAppLogin) dan unit organisasi akun yang dibuat.
	TypeCode string

	// TypeDescription adalah deskripsi tipe, hasil gabungan dari M_SURVEYORS. Hanya
	// dibaca, tidak pernah ditulis modul ini — pemiliknya `mastertipesurveyors`.
	TypeDescription string

	// Name adalah NAME, nama orang atau lembaga surveyor.
	//
	// Keunikannya diuji dengan mengabaikan huruf besar-kecil DAN seluruh spasi — lihat
	// NameKey, dan alasannya ada di sana.
	Name string

	Address      string
	PostalCode   string // KDPOS
	State        string
	Phone        string // TELEPHONE
	Fax          string // FAKSIMILE
	Email        string
	OtherContact string

	// BranchCode adalah BRANCH, berlabel "Cabang" di layar lama.
	//
	// Isian teks, bukan pilihan dari daftar — master cabang belum ada di aplikasi ini, dan
	// layar Pega pun tidak menyediakan daftar pilihan untuknya. Ia diganti daftar pilihan
	// saat master cabang dibangun; sampai itu terjadi, menirunya sebagai isian teks adalah
	// yang setia pada perilaku sekarang.
	BranchCode string

	// BranchName adalah BRANCHNAME, berlabel "Nama Cabang".
	BranchName string

	// AppLogin adalah LOGIN_APLIKASI, nama pengguna akun aplikasi surveyor.
	//
	// WAJIB bila TypeCode adalah InternalTypeCode, dan keunikannya diuji terhadap
	// identitas pengguna yang sudah ada. Kedua aturan itu dari langkah 8 dan langkah
	// 10–14 `CNMInsertDetailSurveyors_act`, bukan tambahan.
	AppLogin string

	// DocumentID adalah DOCID, menunjuk lampiran yang diunggah saat pengajuan.
	//
	// Di sistem lama ia hasil `Call PNCSaveAttachmentToDB` pada langkah 2, sebelum apa pun
	// yang lain — lampirannya disimpan LEBIH DULU, lalu ID-nya ditaruh di sini.
	DocumentID string

	// Status adalah APPROVAL.
	Status ApprovalStatus

	// Committee adalah KOMITE, identitas komite yang ditunjuk memutuskan.
	//
	// Diisi sistem, bukan pengguna. Sistem lama mengisinya di langkah 6–7 dari hasil
	// `GetKomiteApproval`:
	//
	//	TempDetailSurveyors.KOMITE := TempRDBSearchEmailKomite.pxResults(1).BUSINESS_CODE
	//
	// dengan prasyarat `TempDetailSurveyors.KOMITE==""` — artinya komite ditetapkan SEKALI
	// saat pertama, dan tidak ditetapkan ulang pada penyuntingan berikutnya.
	Committee string

	// CommitteeTransferred adalah TRFKOMITE, penanda baris sudah diteruskan ke komite.
	//
	// Dibawa apa adanya. Nilai yang benar-benar dipakai sistem lama tidak dapat dibaca
	// dari export — ia hanya disalin, tidak pernah dibandingkan di rule mana pun yang ada.
	CommitteeTransferred string

	// DecidedAt adalah waktu keputusan komite. Nil berarti belum diputuskan.
	//
	// TIDAK ADA KOLOM PADANANNYA di D_SURVEYORS sejauh yang terbaca dari kedua Report
	// Definition. Ia ditambahkan modul ini, bukan dibawa — karena `D-59` menjadikan jejak
	// audit satu-satunya kontrol pengimbang, dan keputusan komite tanpa waktu tidak dapat
	// ditelusuri. Penyimpanannya diatur repo/sqlstore.
	DecidedAt *time.Time

	// Note adalah keterangan yang menyertai keputusan komite.
	//
	// Sumbernya `TempDetailSurveyors.pyNote`, yang di sistem lama dipakai ganda: langkah
	// terakhir `CNMInsertDetailSurveyors_act` menimpanya dengan pesan keluaran procedure
	// (`OutputData.NAME`), yaitu kalimat "Data Sudah Disimpan dengan ID : ...".
	//
	// Pemakaian ganda itu TIDAK dibawa: satu field yang kadang berisi catatan komite dan
	// kadang berisi pesan sistem tidak dapat dibaca siapa pun. Di sini ia catatan komite,
	// dan hanya itu.
	Note string

	CreatedBy string
	CreatedAt time.Time
	UpdatedBy string
}

// RequiresAppLogin menyatakan surveyor ini wajib punya nama login aplikasi.
//
// Aturannya dari langkah 8 `CNMInsertDetailSurveyors_act`, yang deskripsinya menyebutkan
// syaratnya apa adanya:
//
//	"error if \"internal surveyor\" & LOGIN_APLIKASI is null"
//
// Hanya surveyor internal yang memakai aplikasi ini untuk mengisi hasil survei; surveyor
// eksternal mengirimkan hasilnya lewat jalur lain dan tidak pernah masuk.
func (s Surveyor) RequiresAppLogin() bool {
	return strings.TrimSpace(s.TypeCode) == InternalTypeCode
}

// AwaitingDecision menyatakan surveyor ini belum diputuskan komite.
func (s Surveyor) AwaitingDecision() bool { return s.Status == StatusPending }

// Assignable menyatakan surveyor ini boleh ditugaskan pada sebuah klaim.
//
// Satu syarat saja, berbeda dari `masterrekening.Usable` yang punya dua: D_SURVEYORS
// TIDAK punya kolom penanda aktif. Menambahkannya di sini berarti mengarang keadaan yang
// tidak dapat disimpan di mana pun.
func (s Surveyor) Assignable() bool { return s.Status == StatusApproved }

// Clean mengembalikan salinan dengan spasi tepi dibuang dari setiap isian teks.
//
// Perapian ini nyata gunanya, bukan kosmetik: kode situs pada D_SURVEY_ID dibentuk dengan
// perangkaian teks, dan setiap kolom kode yang bertipe CHAR akan dipadatkan Oracle dengan
// spasi tanpa memberi tanda apa pun. Membiarkannya membuat perbandingan kode membawa spasi
// yang tidak dimaksudkan siapa pun.
func (s Surveyor) Clean() Surveyor {
	s.ID = strings.TrimSpace(s.ID)
	s.LegacyID = strings.TrimSpace(s.LegacyID)
	s.TypeCode = strings.TrimSpace(s.TypeCode)
	s.TypeDescription = strings.TrimSpace(s.TypeDescription)
	s.Name = strings.TrimSpace(s.Name)
	s.Address = strings.TrimSpace(s.Address)
	s.PostalCode = strings.TrimSpace(s.PostalCode)
	s.State = strings.TrimSpace(s.State)
	s.Phone = strings.TrimSpace(s.Phone)
	s.Fax = strings.TrimSpace(s.Fax)
	s.Email = strings.TrimSpace(s.Email)
	s.OtherContact = strings.TrimSpace(s.OtherContact)
	s.BranchCode = strings.TrimSpace(s.BranchCode)
	s.BranchName = strings.TrimSpace(s.BranchName)
	s.AppLogin = strings.TrimSpace(s.AppLogin)
	s.DocumentID = strings.TrimSpace(s.DocumentID)
	s.Committee = strings.TrimSpace(s.Committee)
	s.CommitteeTransferred = strings.TrimSpace(s.CommitteeTransferred)
	s.Note = strings.TrimSpace(s.Note)
	return s
}

// NameKey adalah bentuk nama yang dipakai menguji keunikan.
//
// Ia MEMBUANG SELURUH SPASI, bukan hanya spasi tepi, lalu menyeragamkan huruf. Itu bukan
// pilihan gaya — ia meniru `Activity/ValidasiMasterSurveyor-Act.xml` persis:
//
//	@toUpperCase(@replaceAll(.NAME," ","")) == @toUpperCase(@replaceAll(TempDetailSurveyors.NAME," ",""))
//
// Akibatnya "BUDI SANTOSO", "Budi Santoso", dan "budisantoso" adalah orang yang SAMA bagi
// sistem, dan surveyor kedua bernama demikian ditolak.
//
// Perhatikan bedanya dari modul induk: `mastertipesurveyors.DescriptionKey` hanya
// menyeragamkan huruf dan membuang spasi TEPI. Menyalinnya ke sini akan melonggarkan
// aturan yang sistem lama tegakkan, dan membuat nama ganda lolos.
func NameKey(name string) string {
	var b strings.Builder
	b.Grow(len(name))
	for _, r := range strings.ToUpper(name) {
		// Seluruh ruang putih dibuang, bukan hanya U+0020. Tab dan spasi tanpa-putus
		// menghasilkan nama yang terlihat sama di layar tetapi lolos perbandingan bila
		// hanya spasi biasa yang dibuang.
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == ' ' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

var (
	// ErrNotFound dikembalikan bila surveyor yang diminta tidak ada.
	ErrNotFound = errors.New("mastersurveyors: surveyor tidak ditemukan")

	// ErrNameTaken dikembalikan bila nama surveyor sudah terdaftar. Lihat NameKey untuk
	// arti "sama".
	ErrNameTaken = errors.New("mastersurveyors: nama surveyor sudah terdaftar")

	// ErrLoginTaken dikembalikan bila nama login aplikasi sudah dipakai identitas lain.
	//
	// Sistem lama memeriksanya terhadap Operator ID Pega — langkah 10–14
	// `CNMInsertDetailSurveyors_act`, yang deskripsinya berbunyi "Panggil RD cek operator
	// ID = login aplikasi, kalo sama error".
	ErrLoginTaken = errors.New("mastersurveyors: nama login aplikasi sudah dipakai")

	// ErrAlreadyDecided dikembalikan bila komite hendak memutuskan surveyor yang
	// keputusannya sudah pernah diambil. Keputusan komite tidak dianulir lewat layar ini.
	ErrAlreadyDecided = errors.New("mastersurveyors: keputusan komite sudah pernah diambil")

	// ErrUnknownStatus dikembalikan bila keputusan yang diminta bukan setuju maupun tolak.
	ErrUnknownStatus = errors.New("mastersurveyors: status keputusan tidak dikenal")

	// ErrNotAssignedCommittee dikembalikan bila yang memutuskan bukan komite yang
	// ditunjuk untuk surveyor itu.
	//
	// Ia satu-satunya kontrol kewenangan yang benar-benar ada di modul ini: `D-59`
	// menetapkan satuan izin adalah MENU dan tidak ada pemisahan tugas formal, sehingga
	// siapa pun yang memiliki menu Master Data dapat membuka layarnya.
	ErrNotAssignedCommittee = errors.New("mastersurveyors: bukan komite yang ditunjuk untuk surveyor ini")

	// ErrNoSite dikembalikan bila kode tidak dapat dibentuk karena tabel situs tidak
	// memuat baris aktif. Ia dideklarasikan di domain supaya transport dapat membedakannya
	// dari kegagalan basis data biasa dan menjawabnya dengan pesan yang dapat
	// ditindaklanjuti.
	ErrNoSite = errors.New("mastersurveyors: POOLDATA.M_SITE_DATABASE tidak memuat baris CURRENT_SITE aktif")
)

// ValidationError menyebut seluruh field yang tidak memenuhi syarat sekaligus.
//
// Disebut sekaligus, bukan satu per satu: formulir surveyor punya dua belas isian, dan
// pengguna berhak tahu seluruh yang kurang dalam satu kali. Ini kesetaraan perilaku, bukan
// selera — sistem lama menampilkan seluruh pesan validasi bersamaan
// (`docs/Steering/12-CROSSCUTTING.md` §1.2 butir 1).
type ValidationError struct {
	Field map[string]string
}

func (v *ValidationError) Error() string {
	name := make([]string, 0, len(v.Field))
	for f := range v.Field {
		name = append(name, f)
	}
	// Diurutkan supaya pesannya sama pada setiap pemanggilan — pesan galat yang
	// berubah-ubah urutannya menyulitkan pengujian dan pembacaan log.
	sortNames(name)
	return "mastersurveyors: isian tidak lengkap: " + strings.Join(name, ", ")
}

// Empty menyatakan tidak ada satu pun field yang bermasalah.
func (v *ValidationError) Empty() bool { return len(v.Field) == 0 }

// Check memeriksa kelengkapan seorang surveyor sebelum disimpan.
//
// # Dari mana daftar wajibnya
//
// Empat aturan wajib terbaca dari `CNMInsertDetailSurveyors_act`, dan keempatnya dibawa:
//
//	langkah 5   EMAIL wajib          → pesan "Email harus diisi"
//	langkah 8   LOGIN_APLIKASI wajib bila tipe = internal surveyor
//	langkah 9   nama tidak boleh sama dengan yang sudah ada
//	(tipe)      M_SURVEY_ID wajib — tanpa tipe, PNCIsInternalSurveyors tidak dapat dinilai
//
// EMAIL sempat diperlakukan OPSIONAL di sini, dan itu KELIRU. Yang membetulkannya adalah
// langkah 4, yang menyiapkan teks galatnya sebagai variabel lokal:
//
//	local.email := "Email harus diisi"
//
// lalu langkah 5 memancarkannya dengan prasyarat `TempDetailSurveyors.EMAIL==""`. Pesan
// itulah buktinya — bukan dugaan dari bentuk formulirnya.
//
// Formulir masukannya sendiri memang TIDAK ADA di export, sehingga tanda wajib di layar
// tidak dapat dibaca. Karena itu yang diperlakukan wajib hanyalah yang punya PESAN GALAT
// atau ATURAN yang bergantung padanya. Sisanya dibiarkan opsional: menambahkan kewajiban
// yang tidak terbaca di mana pun akan menolak data yang hari ini sah.
func (s Surveyor) Check() error {
	clean := s.Clean()
	issues := &ValidationError{Field: map[string]string{}}

	if clean.TypeCode == "" {
		issues.Field["kode_tipe"] = "Tipe surveyor wajib dipilih."
	}
	if clean.Name == "" {
		issues.Field["nama"] = "Nama surveyor wajib diisi."
	}
	// Dihitung dalam rune, bukan byte: satu huruf beraksen memakan dua byte dan akan
	// membuat batas terasa berubah-ubah bagi pengguna.
	if utf8.RuneCountInString(clean.Name) > MaxNameLength {
		issues.Field["nama"] = "Nama surveyor paling panjang " + strconv.Itoa(MaxNameLength) + " karakter."
	}

	if clean.RequiresAppLogin() && clean.AppLogin == "" {
		issues.Field["login_aplikasi"] = "Login aplikasi wajib diisi untuk Internal Surveyor."
	}

	// Email WAJIB — meniru langkah 5 beserta pesannya yang terbaca apa adanya di langkah 4
	// (`local.email := "Email harus diisi"`).
	//
	// Bentuknya diperiksa hanya bila terisi: kalau kosong, pesan "wajib diisi" sudah cukup
	// dan menambah pesan kedua untuk field yang sama hanya membingungkan.
	switch address := clean.Email; {
	case address == "":
		issues.Field["email"] = "Email harus diisi."
	case !EmailLooksValid(address):
		issues.Field["email"] = "Format email tidak benar."
	}

	if issues.Empty() {
		return nil
	}
	return issues
}

// Filter mempersempit daftar surveyor yang dibaca.
//
// Seluruh field boleh kosong; yang kosong tidak ikut mempersempit. Bentuknya meniru
// keempat section grid yang ada — masing-masing membaca `BrowseVDSurveyors_RD` dengan
// penyaring posisi persetujuan yang berbeda.
type Filter struct {
	// Status membatasi ke satu posisi persetujuan. Kosong berarti seluruhnya.
	Status ApprovalStatus

	// Name dan AppLogin adalah pencarian sebagian, tanpa peduli besar-kecil huruf.
	Name     string
	AppLogin string

	// TypeCode membatasi ke satu tipe surveyor.
	TypeCode string

	// MyCommitteeOnly membatasi ke surveyor yang menunggu keputusan komite yang sedang
	// masuk. Dipakai tab "Antrean Komite Saya".
	MyCommitteeOnly bool
	// CommitteeIdentity adalah komite yang sedang masuk; hanya dipakai bila
	// MyCommitteeOnly bernilai true.
	CommitteeIdentity string

	// Limit dan Offset adalah paginasi dari server. Limit 0 berarti memakai nilai baku
	// repo, BUKAN berarti tanpa batas — `BrowseVDSurveyors_RD` pun memasang
	// `pyMaxRecords=500`, dan daftar surveyor tumbuh terus.
	Limit  int
	Offset int
}

// Repo adalah seam ke penyimpanan master surveyor SATU portal.
//
// Satu instans Repo selalu terikat pada satu basis data entitas — pemisahan antarentitas
// ada di tingkat KONEKSI, bukan di tingkat penyaringan baris (`ADR-0030` Opsi 1). Tidak
// ada satu pun kueri di pengisinya yang menyaring berdasarkan entitas, dan memang tidak
// boleh ada.
type Repo interface {
	// List membaca surveyor yang cocok dengan filter, beserta jumlah seluruh baris yang
	// cocok sebelum dipotong paginasi.
	List(ctx context.Context, f Filter) (rows []Surveyor, total int, err error)

	// Get membaca satu surveyor. Mengembalikan ErrNotFound bila tidak ada.
	Get(ctx context.Context, id string) (Surveyor, error)

	// FindByNameKey membaca seluruh baris yang NameKey-nya sama persis. Dipakai
	// pemeriksaan nama ganda.
	FindByNameKey(ctx context.Context, key string) ([]Surveyor, error)

	// FindByAppLogin membaca seluruh baris dengan nama login tertentu. Dipakai
	// pemeriksaan login ganda di dalam master ini sendiri.
	FindByAppLogin(ctx context.Context, login string) ([]Surveyor, error)

	// Insert menyimpan surveyor baru dan mengembalikannya LENGKAP DENGAN ID yang dibuat
	// penyimpanan. ID tidak pernah datang dari pemanggil.
	Insert(ctx context.Context, s Surveyor) (Surveyor, error)

	// Update menulis ulang surveyor yang sudah ada.
	Update(ctx context.Context, s Surveyor) error
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

// CommitteeResolver adalah seam ke penetapan komite.
//
// Ia memenuhi langkah 6 `CNMInsertDetailSurveyors_act`, `Call GetKomiteApproval` yang
// deskripsinya "get nama komite". Dipisahkan menjadi seam sendiri karena sumbernya tabel
// POOLDATA.EMAILKOMITE — master yang sama yang dipakai penjenjangan komite klaim (`B-7`),
// dan yang pemiliknya bukan modul ini.
//
// Prasyarat aslinya `TempDetailSurveyors.KOMITE==""` dibawa di usecase, bukan di sini:
// komite ditetapkan SEKALI saat pengajuan pertama dan tidak ditetapkan ulang pada
// penyuntingan berikutnya.
type CommitteeResolver interface {
	// Resolve mengembalikan identitas komite yang berwenang memutuskan surveyor ini.
	//
	// Mengembalikan string kosong TANPA galat bila tidak ada komite yang cocok — itu
	// keadaan yang sah dan ditangani usecase, bukan kegagalan.
	Resolve(ctx context.Context, portalAlias string, s Surveyor) (string, error)
}

// AccountRequest adalah permintaan pembuatan akun aplikasi untuk seorang surveyor.
//
// Kedelapan field di bawah adalah PERSIS kedelapan parameter yang
// `CNMInsertDetailSurveyors_act` kirimkan ke `GCNMCreateOperator`, dibaca apa adanya dari
// export:
//
//	param.orgName        := "ASM"
//	param.divName        := "PNC"
//	param.unitName       := @If(M_SURVEY_ID=="1001","Internal","Eksternal")
//	param.userId         := LOGIN_APLIKASI
//	param.userName       := NAME
//	param.password       := LOGIN_APLIKASI + "123456"
//	param.accessGroup    := "GCNMFW:PNCSurveyor"
//	param.changePassword := "true"
type AccountRequest struct {
	Organization string
	Division     string

	// Unit adalah "Internal" atau "Eksternal".
	//
	// Ia BUKAN keterangan tampilan. `docs/Steering/11-SECURITY.md` §3.2 mencatat batas
	// data yang ditegakkan dengan membandingkan `OperatorID.pyOrgUnit != "Eksternal"` —
	// jadi nilai ini menentukan data siapa yang boleh dilihat pemilik akun.
	Unit string

	UserID   string
	UserName string

	// Password adalah sandi SEMENTARA.
	//
	// `GCNMCreateOperator` menyetel `pyChangePasswordOnNextLogin = "True"`, sehingga ia
	// wajib diganti pada login pertama dan tidak pernah menjadi sandi tetap. Itu yang
	// membuat pola `LOGIN_APLIKASI + "123456"` dapat dibawa tanpa menjadi sandi lemah
	// permanen — tetapi ia tetap dapat ditebak selama jeda sebelum login pertama, dan itu
	// dicatat sebagai risiko yang diterima, bukan risiko yang tidak terlihat.
	Password string

	// MustChangePassword selalu true, mengikuti `pyChangePasswordOnNextLogin`. Ia
	// dijadikan field, bukan dipatok di pengisi seam, supaya syaratnya terbaca dari sini.
	MustChangePassword bool

	// AccessGroup adalah "GCNMFW:PNCSurveyor", satu dari 22 peran pada `D-58`.
	AccessGroup string
}

// AccountRegistrar adalah seam ke pembuatan akun aplikasi surveyor.
//
// # Kenapa seam, dan bukan penulisan langsung
//
// Sistem lama membuat instans `Data-Admin-Operator-ID`, yaitu baris pada tabel operator
// milik ENGINE PEGA. `P-1` (`D-21`) menetapkan satu tabel hanya boleh ditulis satu sistem
// selama masa paralel, dan tabel itu dimiliki Pega sampai `F-3` memindahkannya.
//
// Menulisnya dari sini akan melanggar aturan itu dengan cara yang tidak menimbulkan pesan
// galat apa pun — pelanggarannya baru terlihat sebagai data rusak.
//
// Karena itu modul ini menyatakan KEHENDAKNYA lewat seam ini, dengan kedelapan parameter
// Pega apa adanya, dan menyerahkan pelaksanaannya kepada pengisi seam. Saat `F-3` siap,
// yang berubah hanya adapternya — tidak satu baris pun di domain maupun di layar.
//
// Keputusan Work Owner 2026-09-19: perilaku Pega dibawa; tabel Pega tidak ditulis.
type AccountRegistrar interface {
	// Register membuat akun aplikasi bagi seorang surveyor.
	//
	// Mengembalikan ErrLoginTaken bila nama login sudah dipakai identitas lain — itulah
	// pemeriksaan yang di sistem lama dilakukan terhadap Operator ID Pega, dan pengisi
	// seam-lah yang tahu identitas apa saja yang sudah ada.
	Register(ctx context.Context, portalAlias string, req AccountRequest) error
}

// AccountRequestFor menyusun permintaan akun bagi seorang surveyor.
//
// Ia fungsi murni dan berada di domain supaya kedelapan nilainya dapat diuji tanpa basis
// data, tanpa jaringan, dan tanpa apa pun — dan supaya sandi sementaranya dibentuk di satu
// tempat saja.
func AccountRequestFor(s Surveyor) AccountRequest {
	clean := s.Clean()
	unit := "Eksternal"
	if clean.RequiresAppLogin() {
		unit = "Internal"
	}
	return AccountRequest{
		Organization:       "ASM",
		Division:           "PNC",
		Unit:               unit,
		UserID:             clean.AppLogin,
		UserName:           clean.Name,
		Password:           clean.AppLogin + "123456",
		MustChangePassword: true,
		AccessGroup:        "GCNMFW:PNCSurveyor",
	}
}

// EmailLooksValid memeriksa bentuk alamat surel sekadarnya.
//
// Sengaja longgar, dan itu disengaja: satu-satunya cara membuktikan sebuah alamat benar
// adalah mengirim surel ke sana. Validasi yang terlalu ketat justru menolak alamat sah.
//
// Aturannya disamakan persis dengan `masterrekening.EmailLooksValid` — dua master yang
// menolak alamat berbeda akan membuat pengguna menyimpulkan salah satunya rusak.
func EmailLooksValid(address string) bool {
	address = strings.TrimSpace(address)
	i := strings.IndexByte(address, '@')
	if i <= 0 || i == len(address)-1 {
		return false
	}
	domain := address[i+1:]
	if strings.ContainsRune(domain, '@') {
		return false
	}
	j := strings.IndexByte(domain, '.')
	return j > 0 && j < len(domain)-1
}

// sortNames mengurutkan nama field. Ditulis di sini supaya paket domain tidak perlu
// mengimpor sort hanya untuk satu pemakaian pada jalur galat.
func sortNames(name []string) {
	for i := 1; i < len(name); i++ {
		for j := i; j > 0 && name[j] < name[j-1]; j-- {
			name[j], name[j-1] = name[j-1], name[j]
		}
	}
}
