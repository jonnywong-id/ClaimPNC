// Package masterautoclaim adalah inti modul Master Auto Claim.
//
// # Apa yang dimodelkan di sini
//
// Daftar **Sumber Bisnis yang klaimnya boleh dibuat otomatis**, beserta ke mana ganti
// ruginya dibayarkan. Ia BUKAN master rekening dan bukan master klaim: kuncinya adalah
// kode Sumber Bisnis sebuah polis, dan isinya menjawab satu pertanyaan — "bila polis
// dari sumber bisnis ini mengajukan klaim otomatis, uangnya ke mana".
//
// Buktinya bukan dari namanya melainkan dari pemakaian hilirnya,
// `RDB List/GetReceiverClaimAsuransiKredit-SQL.xml`:
//
//	select nama_penerima, alamat_penerima, bank_penerima, no_rekening, email_lapor,
//	       inisialid, pct_max
//	  from pooldata.m_auto_claim_pnc
//	 where inisialid = (select b.sourceofbusiness from t_general b where b.nopolis = ...)
//	   and claim_allowed = 1
//	   and APPROVAL = '1'
//
// Tiga hal terbaca sekaligus dari kueri itu, dan ketiganya menyetir seluruh modul ini:
//
//  1. `INISIALID` dicocokkan dengan `T_GENERAL.SOURCEOFBUSINESS` — jadi ia kode Sumber
//     Bisnis, bukan nomor apa pun yang diketik petugas. Itu sebabnya ia dipilih dari
//     lookup `POOLDATA.AGENT`, tidak pernah diketik.
//  2. `CLAIM_ALLOWED = 1` adalah PENANDA boleh-tidaknya, bukan pencacah.
//  3. `APPROVAL = '1'` — hanya baris yang sudah disetujui komite yang dipakai membayar.
//
// # Kenapa master ini punya alur persetujuan
//
// Alasannya sama persis dengan Master Rekening, dan bukan kebetulan: baris di sini
// menentukan KE MANA uang klaim dikirim. Salah satu baris berarti pembayaran mendarat
// di rekening yang keliru, dan tidak ada langkah sesudahnya yang menangkapnya. Sistem
// lama pun sudah memperlakukannya begitu — kolom `APPROVAL` dan `KOMITE` ada di
// `POOLDATA.M_AUTO_CLAIM_PNC` sejak awal.
//
// # Asal setiap aturan di berkas ini
//
// Seluruhnya dibaca dari export rule Pega, bukan dikarang:
//
//	Harness/AutoKlaim-Harness.xml                  layar, 4 tab
//	Section/MasterAutoKlaim-Section.xml            judul, tombol Tambah & Refresh
//	Section/BrowseAutoKlaim-Section.xml            tab 1 — grid 9 kolom + form 11 isian
//	Section/BrowseAutoKlaimKomite-Section.xml      tab 2 — tombol Approve & Reject
//	Section/BrowseAutoKlaimApproval-Section.xml    tab 3 — daftar saja
//	Section/BrowseAutoKlaimReject-Section.xml      tab 4 — grid + form
//	RDB List/BrowseAutoKlaim-SQL.xml               daftar, disaring APPROVAL
//	RDB List/InsertAutoClaim-SQL.xml               sisip 14 kolom
//	RDB List/UpdateAutoClaim-SQL.xml               perbarui 12 kolom
//	RDB List/UpdateAutoClaim1-SQL.xml              muat satu baris ke form
//	RDB List/ValidasiAutoClaim-SQL.xml             cek INISIALID sudah dipakai
//	RDB List/GetKomiteAutoKlaim-SQL.xml            penyetuju komite
//	RDB List/GetClientName-SQL.xml                 lookup Sumber Bisnis (POOLDATA.AGENT)
//	RDB List/GetClientName2-SQL.xml                lookup Client (POOLDATA.CLIENT)
//	Activity/InsertMstAutoClaim_act-Act.xml        urutan langkah sisip + validasi
//	Activity/UpdateMstAutoClaim_act-Act.xml        urutan langkah simpan + keputusan
//	Activity/UpdateMstAutoClaim_act1-Act.xml       urutan langkah muat ke form
//	Activity/ValidasiAutoClaim-Act.xml             pesan galat duplikat
//	Activity/BrowseAutoKlaim_act-Act.xml           penyaring tab
//
// # Penamaan ulang yang disengaja (D-19)
//
// Kueri lama mengaliaskan SEPULUH kolom ke nama yang tidak mencerminkan isi sama
// sekali — utang teknis §4.2 `03-CURRENT-ARCHITECTURE.md`. Alias itu TIDAK dibawa:
//
//	INISIALID        AS "CaseID"               -> Initial          (bukan nomor kasus)
//	NAMA_PENERIMA    AS "City"                 -> ReceiverName     (bukan nama kota)
//	BANK_PENERIMA    AS "CityID"               -> BankName         (bukan kode kota)
//	NO_REKENING      AS "District"             -> AccountNumber    (bukan kabupaten)
//	PCT_MAX          AS "Country"              -> MaxPercent       (bukan negara)
//	PIC_LAPOR        AS "AlasanTerlambat"      -> ReporterPIC      (bukan alasan terlambat)
//	EMAIL_LAPOR      AS "DistrictID"           -> ReporterEmail    (bukan kode kabupaten)
//	CLAIM_ALLOWED    AS "AnalystDoctorRemaks"  -> ClaimAllowed     (bukan catatan dokter)
//	ALAMAT_PENERIMA  AS "CountryID"            -> ReceiverAddress  (bukan kode negara)
//	KOMITE           AS "ProdKe"               -> Committee        (bukan nomor produk)
//	CLIENTID         AS "FlagReject"           -> ClientID         (bukan penanda tolak)
//	CLIENTNAME       AS "EmailTertanggung"     -> ClientName       (bukan surel)
//
// DUA PERINGATAN yang membuat alias lama berbahaya dipakai sebagai petunjuk arti, dan
// keduanya nyata di berkas yang sama:
//
//   - `City` berarti NAMA_PENERIMA saat DIBACA, tetapi berarti BANK_PENERIMA saat
//     DITULIS. Satu nama klipboard, dua kolom yang berbeda — bandingkan
//     `BrowseAutoKlaim-SQL.xml` dengan `InsertAutoClaim-SQL.xml`.
//   - `District`/`DistrictID` TIDAK berpasangan: yang satu NO_REKENING, yang lain
//     EMAIL_LAPOR.
//
// Yang dipetakan di seluruh modul ini karena itu adalah KOLOMNYA, bukan aliasnya.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterautoclaim

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ApprovalStatus adalah posisi sebuah baris dalam alur persetujuan komite.
//
// Nilainya sengaja tetap "0", "1", "2" seperti di POOLDATA.M_AUTO_CLAIM_PNC: tabelnya
// masih dibaca dan ditulis sistem lama selama masa paralel (ADR-0004), sehingga
// mengubah sandi nilainya akan membuat kedua sistem membaca baris yang sama secara
// berbeda. Sandinya kebetulan sama persis dengan Master Rekening, dan itu memang pola
// yang berulang di seluruh master bernilai uang pada sistem lama.
type ApprovalStatus string

const (
	// StatusPending — sudah diajukan, komite belum memutuskan. Nilai lahir setiap
	// baris baru: `Section/BrowseAutoKlaim-Section.xml` memanggil
	// InsertMstAutoClaim_act dengan Approval="0".
	StatusPending ApprovalStatus = "0"

	// StatusApproved — komite menyetujui. HANYA baris berstatus ini yang dipakai
	// pembuatan klaim otomatis (`GetReceiverClaimAsuransiKredit-SQL.xml`).
	StatusApproved ApprovalStatus = "1"

	// StatusRejected — komite menolak.
	StatusRejected ApprovalStatus = "2"
)

// Label mengembalikan sebutan status dalam bahasa yang dibaca pengguna.
//
// Teksnya mengikuti caption tab pada `Harness/AutoKlaim-Harness.xml` apa adanya —
// "Waiting Approval", "Approve", "Reject" — supaya petugas membaca kata yang sama
// dengan yang dibacanya di Pega hari ini (D-13).
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

// ClaimAllowedYes adalah satu-satunya nilai yang ditulis ke kolom CLAIM_ALLOWED.
//
// # Kenapa ia konstanta, bukan isian
//
// Keputusan Work Owner 2026-09-19: "selalu 1, tidak dapat diubah". Isiannya dibuang
// dari layar, dan nilainya ditulis sistem baik saat menambah maupun saat mengubah.
//
// Sistem lama TIDAK konsisten pada titik ini, dan ketidakkonsistenannya yang membuat
// keputusan itu diambil:
//
//	InsertMstAutoClaim_act step 1  TempInputAutoClaim.District := "1"   <- isian ditimpa
//	UpdateMstAutoClaim_act         CLAIM_ALLOWED = {TempInputAutoClaim.District}
//	                                                                    <- isian dipakai
//
// Artinya baris yang lahir dengan benar dapat "mati" hanya karena disunting: satu
// ketikan yang bukan "1" membuatnya tidak lagi lolos `claim_allowed = 1` pada
// `GetReceiverClaimAsuransiKredit`, dan tidak ada satu pun pesan yang muncul. Klaim
// otomatis untuk sumber bisnis itu berhenti tanpa sebab yang terlihat.
//
// SELISIH YANG DIRENCANAKAN. Pada jalur UBAH, perilaku ini berbeda dari Pega: yang
// tersimpan selalu "1", bukan isian pengguna. Ia wajib dinyatakan di muka pada uji
// kesetaraan gerbang 1, bukan ditemukan sebagai kejutan.
const ClaimAllowedYes = "1"

// AutoClaim adalah satu baris master auto claim.
//
// Kuncinya Initial (kolom INISIALID) — tunggal, bukan gabungan. Itu terbaca dari
// `UpdateAutoClaim-SQL.xml` yang menyaring `WHERE INISIALID = ...` saja, dan dari
// `ValidasiAutoClaim-SQL.xml` yang menolak penambahan bila INISIALID-nya sudah ada.
type AutoClaim struct {
	// Initial adalah kode Sumber Bisnis, kolom INISIALID. Ia kunci baris ini dan
	// dicocokkan dengan T_GENERAL.SOURCEOFBUSINESS saat klaim otomatis dibuat.
	//
	// Ia TIDAK PERNAH diketik: nilainya dipilih dari lookup POOLDATA.AGENT, dan itu
	// yang dijaga pesan "Nama penerima klaim tidak ditemukan. Jangan diketik manual."
	// pada Activity/ValidasiAutoClaim.
	Initial string

	// ReceiverName adalah nama Sumber Bisnis, kolom NAMA_PENERIMA.
	//
	// Ia salinan dari POOLDATA.AGENT.CLIENTNAME pada saat baris ini dibuat, bukan
	// hasil join — dan ia TIDAK ikut diperbarui saat baris disunting, karena
	// `UpdateAutoClaim-SQL.xml` memang tidak menyebut kolom itu. Keputusan Work Owner
	// 2026-09-19: perilaku itu dipertahankan, dan layar menampilkannya sebagai
	// keterangan yang tidak dapat diubah.
	ReceiverName string

	// BankName adalah nama bank tujuan, kolom BANK_PENERIMA.
	//
	// Yang tersimpan NAMANYA, bukan kodenya — tabel ini tidak punya kolom kode bank.
	// Kode bank hanya dipakai saat memeriksa isian; lihat Input.BankCode.
	BankName string

	// AccountNumber adalah nomor rekening tujuan, kolom NO_REKENING.
	AccountNumber string

	// MaxPercent adalah kolom PCT_MAX.
	//
	// Disimpan sebagai TEKS, bukan angka, dan itu disengaja: DDL tabelnya belum
	// diterima (R-08), sehingga tipe kolom yang sebenarnya belum diketahui. Mengubahnya
	// menjadi angka di sini berarti memutuskan pembulatan dan presisi tanpa dasar —
	// dan nilai uang maupun persentase tidak boleh ditebak (D-51). Baris lama dibaca
	// apa adanya; yang baru diperiksa berbentuk angka lewat Input.Check.
	MaxPercent string

	// ReporterPIC adalah petugas pelapor di sisi sumber bisnis, kolom PIC_LAPOR.
	ReporterPIC string

	// ReporterEmail adalah surel pelapor, kolom EMAIL_LAPOR.
	ReporterEmail string

	// ClaimAllowed adalah kolom CLAIM_ALLOWED. Lihat ClaimAllowedYes.
	//
	// Ia tetap dibaca apa adanya dari basis data, sehingga baris lama yang bernilai
	// selain "1" terlihat petugas alih-alih tersamar.
	ClaimAllowed string

	// ReceiverAddress adalah alamat penerima, kolom ALAMAT_PENERIMA.
	ReceiverAddress string

	// SubmittedBy adalah operator yang terakhir menyimpan baris ini, kolom USERINPUT.
	//
	// Namanya "submitted by" dan bukan "created by" karena sistem lama menimpanya pada
	// SETIAP penyimpanan — termasuk saat komite menyetujui. Jadi ia menyimpan pelaku
	// terakhir, bukan pembuat pertama. Itu keterbatasan tabelnya, bukan pilihan: tidak
	// ada kolom kedua untuk memisahkan keduanya, dan menambah kolom menuntut
	// persetujuan Work Owner serta pelaksanaan DBA (D-63).
	SubmittedBy string

	// Committee adalah operator komite yang berwenang memutuskan baris ini, kolom
	// KOMITE. Diisi sistem saat penambahan; lihat CommitteeSource.
	Committee string

	// Status adalah kolom APPROVAL.
	Status ApprovalStatus

	// ClientID dan ClientName adalah tertanggung yang ditautkan ke sumber bisnis ini,
	// kolom CLIENTID dan CLIENTNAME. Keduanya dipilih dari lookup POOLDATA.CLIENT.
	//
	// Berbeda dari ReceiverName, keduanya MEMANG ikut diperbarui saat baris disunting
	// (`UpdateAutoClaim-SQL.xml` menyebut keduanya).
	ClientID   string
	ClientName string
}

// Usable menyatakan baris ini benar-benar dipakai pembuatan klaim otomatis.
//
// Kedua syaratnya disatukan di sini supaya tidak diulang di setiap layar, dan supaya
// keduanya persis sama dengan penyaring `GetReceiverClaimAsuransiKredit-SQL.xml`:
// `claim_allowed = 1 AND APPROVAL = '1'`.
func (a AutoClaim) Usable() bool {
	return a.Status == StatusApproved && strings.TrimSpace(a.ClaimAllowed) == ClaimAllowedYes
}

// Input adalah nilai yang dikirim pengguna dari layar, sebelum diperiksa.
//
// Sebelas isian, mengikuti `Section/BrowseAutoKlaim-Section.xml`. Tiga nilai yang ada
// di tabel TIDAK ada di sini karena ketiganya diturunkan sistem:
//
//	USERINPUT      operator yang sedang masuk
//	KOMITE         hasil CommitteeSource
//	CLAIM_ALLOWED  selalu ClaimAllowedYes
type Input struct {
	// Initial dan ReceiverName datang berpasangan dari lookup Sumber Bisnis.
	Initial      string
	ReceiverName string

	// BankName adalah nama bank yang tersimpan, kolom BANK_PENERIMA.
	//
	// KODE BANK TIDAK ADA DI SINI, dan itu keputusan yang disengaja. Pega mengirim
	// kode bank dari autocomplete lewat `TempInputAutoClaim.Location`, lalu menolak
	// dengan "Nama bank jangan diketik manual" bila kode itu kosong
	// (InsertMstAutoClaim_act step 3). Kodenya sendiri tidak pernah disimpan — tabel
	// ini tidak punya kolomnya.
	//
	// Menerima kode dari layar di sini berarti mempercayai peramban atas nilai yang
	// sudah ada di basis data, dan membuat tombol Approve bergantung pada nilai yang
	// TIDAK dapat dibaca kembali dari baris yang tersimpan. Sebagai gantinya, lapisan
	// aplikasi mencocokkan NAMA bank ke GENERAL.LST_BANK_GROUP — pemeriksaan yang
	// lebih kuat daripada aslinya: Pega memastikan "ada sesuatu yang dipilih", di sini
	// dipastikan "yang tersimpan benar-benar ada di master bank".
	BankName string

	AccountNumber   string
	MaxPercent      string
	ReporterPIC     string
	ReporterEmail   string
	ReceiverAddress string

	ClientID   string
	ClientName string

	// Status adalah posisi persetujuan yang dikehendaki pemanggil.
	//
	// Ia ADA di sini, dan itu bukan kelalaian keamanan melainkan bentuk sistem lama.
	// Satu activity — UpdateMstAutoClaim_act — melayani tiga tombol yang berbeda,
	// dibedakan HANYA oleh parameternya:
	//
	//	tombol Update (tab Master & Reject)  stsapprove="0"  -> kembali menunggu
	//	tombol Approve (tab Komite)          stsapprove="1"
	//	tombol Reject  (tab Komite)          stsapprove="2"
	//
	// Keputusan Work Owner 2026-09-19 mempertahankannya: menyetujui dan mengubah
	// adalah operasi yang sama, dengan status yang berbeda. Karena itu pula menyunting
	// baris yang sudah disetujui MENGEMBALIKANNYA ke status menunggu — persetujuan
	// lama tidak berlaku atas isi yang sudah berubah.
	//
	// Pada penambahan, nilai ini diabaikan: baris baru selalu lahir StatusPending.
	Status ApprovalStatus
}

// Panjang maksimum isian.
//
// SELURUHNYA ASUMSI YANG DISADARI, bukan angka yang diterima dari Work Owner maupun
// dibaca dari DDL: `POOLDATA.M_AUTO_CLAIM_PNC` tidak ada DDL-nya di export (R-08), dan
// sistem lama tidak memeriksa panjang satu pun isian.
//
// Batasnya tetap dipasang karena tanpa itu penolakan datang dari basis data sebagai
// ORA-12899 — galat teknis yang tidak menuntun pengguna ke mana pun. Angkanya dipilih
// longgar tetapi tidak sembarangan: mengikuti panjang lazim kolom sejenis pada tabel
// yang sudah diketahui DDL-nya, dan diturunkan begitu DDL yang sebenarnya tiba.
//
// Angka yang sama diulang di `AutoClaimForm.tsx`. Bila berubah, KEDUA tempat harus ikut
// berubah — utang yang disadari dari menduplikasi sebuah angka, dijaga terlihat oleh
// uji di masterautoclaim_test.go.
const (
	MaxInitialLength       = 20
	MaxReceiverNameLength  = 150
	MaxBankNameLength      = 100
	MaxAccountNumberLength = 30
	MaxReporterPICLength   = 100
	MaxEmailLength         = 100
	MaxAddressLength       = 250
	MaxClientIDLength      = 20
	MaxClientNameLength    = 150
)

// Galat modul ini. Transport yang memetakannya ke kode HTTP; domain tidak tahu HTTP.
var (
	// ErrNotFound: baris yang diminta tidak ada.
	ErrNotFound = errors.New("masterautoclaim: master auto claim tidak ditemukan")

	// ErrInitialTaken: INISIALID yang akan disisipkan sudah dipakai baris lain.
	//
	// Padanan langsung pesan "Data sudah pernah diinput." pada
	// Activity/ValidasiAutoClaim step 5.
	ErrInitialTaken = errors.New("masterautoclaim: sumber bisnis ini sudah ada di master")

	// ErrUnknownStatus: status persetujuan di luar "0", "1", "2".
	ErrUnknownStatus = errors.New("masterautoclaim: status persetujuan tidak dikenal")
)

// Violation adalah satu isian yang tidak lolos pemeriksaan.
type Violation struct {
	// Field adalah nama isian dalam bentuk yang dikenali layar, bukan nama kolom basis
	// data — layar yang menyorot isiannya memakai nilai ini.
	Field   string
	Message string
}

// ValidationError memuat SELURUH pelanggaran sekaligus, bukan yang pertama saja.
//
// Ini kesetaraan perilaku, bukan selera (P-5). Yang berbeda dari sistem lama hanyalah
// KEHALUSANNYA: Pega menjawab satu kalimat untuk delapan isian sekaligus — "Mohon Isi
// Data Dengan Lengkap, Data Tidak Boleh Kosong" — sehingga pengguna harus menebak isian
// mana yang dimaksud. Di sini setiap isian menyebut dirinya sendiri.
//
// Itu perbaikan tampilan, bukan perubahan aturan: himpunan isian yang ditolak sama
// persis, dan permintaan yang ditolak Pega juga ditolak di sini.
type ValidationError struct {
	Violation []Violation
}

// OneViolation membungkus satu pelanggaran menjadi ValidationError.
//
// Dipakai lapisan aplikasi untuk pemeriksaan yang menuntut pembacaan basis data —
// pencocokan nama bank — supaya galatnya sampai ke layar dalam bentuk yang SAMA dengan
// pelanggaran isian lain, dan menempel pada isiannya. Tanpa ini, satu pemeriksaan akan
// menjawab 500 sementara pemeriksaan lain menjawab 422 pada isian.
func OneViolation(field, message string) error {
	return &ValidationError{Violation: []Violation{{Field: field, Message: message}}}
}

func (g *ValidationError) Error() string {
	parts := make([]string, 0, len(g.Violation))
	for _, p := range g.Violation {
		parts = append(parts, p.Field+": "+p.Message)
	}
	return "masterautoclaim: isian tidak sah (" + strings.Join(parts, "; ") + ")"
}

// Clean memangkas spasi di kedua ujung setiap isian.
//
// Dipisahkan dari Check supaya nilai yang tersimpan adalah nilai yang sudah dipangkas
// — bukan nilai mentah yang lolos pemeriksaan karena kebetulan spasinya ikut terhitung.
func (i Input) Clean() Input {
	return Input{
		Initial:         strings.TrimSpace(i.Initial),
		ReceiverName:    strings.TrimSpace(i.ReceiverName),
		BankName:        strings.TrimSpace(i.BankName),
		AccountNumber:   strings.TrimSpace(i.AccountNumber),
		MaxPercent:      strings.TrimSpace(i.MaxPercent),
		ReporterPIC:     strings.TrimSpace(i.ReporterPIC),
		ReporterEmail:   strings.TrimSpace(i.ReporterEmail),
		ReceiverAddress: strings.TrimSpace(i.ReceiverAddress),
		ClientID:        strings.TrimSpace(i.ClientID),
		ClientName:      strings.TrimSpace(i.ClientName),
		Status:          ApprovalStatus(strings.TrimSpace(string(i.Status))),
	}
}

// Check menjalankan seluruh aturan isian dan mengembalikan SEMUA pelanggarannya.
//
// Nil berarti isian sah. Input sudah harus melewati Clean lebih dulu.
//
// # Yang diperiksa, dan dari mana asalnya
//
// Delapan isian wajib, persis daftar pada prasyarat InsertMstAutoClaim_act step 2:
//
//	Local.INISIAL == "" || Local.NAMA == "" || Local.BANK == "" || Local.EMAIL == ""
//	|| Local.ALAMAT == "" || Local.PIC == "" || Local.PCT_MAX == "" || Local.NO_REK == ""
//
// # Satu pemeriksaan yang TIDAK ada di sini, dan kenapa
//
// Padanan prasyarat step 3 — "Nama bank jangan diketik manual" — menuntut pencocokan ke
// GENERAL.LST_BANK_GROUP, dan itu pembacaan basis data. Domain tidak boleh
// melakukannya, sehingga ia dikerjakan lapisan aplikasi lewat LookupRepo.FindBankByName
// dan dilaporkan sebagai pelanggaran pada isian `nama_bank` yang sama. Lihat
// AutoClaim.BankName.
//
// # Yang SENGAJA tidak diwajibkan
//
// ClientID dan ClientName. Keduanya tidak ada di daftar prasyarat mana pun, dan
// `InsertAutoClaim-SQL.xml` menyisipkannya apa adanya — termasuk kosong. Mewajibkannya
// di sini akan menolak penambahan yang hari ini diterima.
//
// Keberadaan Sumber Bisnis dan Client di masternya masing-masing juga TIDAK diperiksa
// di sini: itu menuntut pembacaan basis data, dan domain tidak boleh melakukannya.
func (i Input) Check() error {
	violation := i.checkEditable()

	// Kedua isian ini hanya diperiksa pada PENAMBAHAN, dan itulah sebabnya ia terpisah
	// dari checkEditable. Pada penyimpanan keduanya tidak datang dari permintaan sama
	// sekali: INISIALID diambil dari jalur URL, dan NAMA_PENERIMA dibaca dari baris yang
	// tersimpan karena memang tidak dapat diubah.
	for _, r := range []struct {
		field, label, value string
		max                 int
	}{
		{"inisial", "Sumber Bisnis", i.Initial, MaxInitialLength},
		{"nama_penerima", "Nama penerima", i.ReceiverName, MaxReceiverNameLength},
	} {
		violation = append(violation, checkText(r.field, r.label, r.value, r.max)...)
	}

	if len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// CheckEditable memeriksa HANYA isian yang dapat diubah lewat penyimpanan.
//
// Ia dipakai jalur simpan dan keputusan komite. Bedanya dari Check hanya dua isian:
// Sumber Bisnis dan nama penerima, yang keduanya tidak dapat diubah setelah baris
// dibuat — lihat auto_claim_update pada berkas .sql.
//
// Keduanya berbagi checkEditable, bukan disalin, supaya aturan yang sama tidak pernah
// berbeda antara menambah dan menyimpan.
func (i Input) CheckEditable() error {
	if violation := i.checkEditable(); len(violation) > 0 {
		return &ValidationError{Violation: violation}
	}
	return nil
}

// checkEditable mengembalikan pelanggaran pada isian yang dapat diubah.
func (i Input) checkEditable() []Violation {
	var violation []Violation

	for _, r := range []struct {
		field, label, value string
		max                 int
	}{
		{"nama_bank", "Bank penerima", i.BankName, MaxBankNameLength},
		{"no_rekening", "Nomor rekening", i.AccountNumber, MaxAccountNumberLength},
		{"pic_lapor", "PIC lapor", i.ReporterPIC, MaxReporterPICLength},
		{"email_lapor", "Email lapor", i.ReporterEmail, MaxEmailLength},
		{"alamat_penerima", "Alamat penerima", i.ReceiverAddress, MaxAddressLength},
	} {
		violation = append(violation, checkText(r.field, r.label, r.value, r.max)...)
	}

	violation = append(violation, i.checkMaxPercent()...)
	violation = append(violation, i.checkClient()...)
	return violation
}

// checkText memeriksa satu isian wajib beserta panjangnya.
func checkText(field, label, value string, max int) []Violation {
	switch {
	case value == "":
		return []Violation{{Field: field, Message: label + " wajib diisi."}}
	case len(value) > max:
		return []Violation{{
			Field:   field,
			Message: fmt.Sprintf("%s paling panjang %d karakter.", label, max),
		}}
	}
	return nil
}

// checkMaxPercent memeriksa PCT_MAX.
//
// Wajib diisi — itu dari daftar prasyarat Pega. Yang DITAMBAHKAN di sini adalah
// pemeriksaan bahwa isinya berbentuk angka 0–100.
//
// # Kenapa pemeriksaan itu ditambahkan, padahal sistem lama tidak punya
//
// Kolomnya bernama PCT_MAX dan dipakai sebagai persentase pada pembuatan klaim
// otomatis. Sistem lama menerima teks apa pun — termasuk "abc" — dan akibatnya baru
// muncul jauh di hilir, pada perhitungan yang memakainya, tanpa satu pun tanda bahwa
// asalnya dari baris master ini.
//
// SELISIH YANG DIRENCANAKAN: isian yang dulu lolos kini ditolak. Ia disebut di muka
// pada uji kesetaraan gerbang 1, dan tidak mengubah satu pun baris yang sudah ada —
// baris lama tetap dibaca apa adanya.
func (i Input) checkMaxPercent() []Violation {
	if i.MaxPercent == "" {
		return []Violation{{Field: "pct_max", Message: "PCT max wajib diisi."}}
	}

	value, err := parsePercent(i.MaxPercent)
	if err != nil {
		return []Violation{{
			Field:   "pct_max",
			Message: "PCT max harus berupa angka, misalnya 100 atau 82,5.",
		}}
	}
	if value < 0 || value > 100 {
		return []Violation{{
			Field:   "pct_max",
			Message: "PCT max harus di antara 0 dan 100.",
		}}
	}
	return nil
}

// checkClient memeriksa pasangan ClientID dan ClientName.
//
// Keduanya boleh kosong — lihat Check. Yang tidak boleh adalah SETENGAH terisi:
// `UpdateAutoClaim-SQL.xml` menulis keduanya, sehingga menyimpan nama tanpa ID
// meninggalkan baris yang tidak dapat ditautkan ke master client mana pun.
func (i Input) checkClient() []Violation {
	switch {
	case i.ClientID == "" && i.ClientName == "":
		return nil
	case i.ClientID == "" || i.ClientName == "":
		return []Violation{{
			Field:   "id_client",
			Message: "Client harus dipilih dari daftar; ID dan namanya tidak boleh terpisah.",
		}}
	case len(i.ClientID) > MaxClientIDLength:
		return []Violation{{
			Field:   "id_client",
			Message: fmt.Sprintf("ID client paling panjang %d karakter.", MaxClientIDLength),
		}}
	case len(i.ClientName) > MaxClientNameLength:
		return []Violation{{
			Field:   "nama_client",
			Message: fmt.Sprintf("Nama client paling panjang %d karakter.", MaxClientNameLength),
		}}
	}
	return nil
}

// parsePercent membaca PCT_MAX sebagai angka.
//
// Koma DAN titik keduanya diterima sebagai pemisah desimal: petugas Indonesia mengetik
// "82,5" sementara nilai yang tersimpan di basis data memakai "82.5". Menolak salah
// satunya berarti menolak isian yang benar hanya karena papan ketiknya.
//
// ParseFloat dipakai, bukan Sscanf: Sscanf berhenti pada karakter pertama yang tidak
// cocok dan TETAP melapor sukses, sehingga "82abc" akan lolos sebagai 82. ParseFloat
// menolak seluruh teks yang tidak habis terbaca.
//
// Nilainya TIDAK dipakai untuk apa pun selain pemeriksaan — yang tersimpan tetap teks
// apa adanya, sehingga pembulatan tidak pernah terjadi di jalur ini (D-51).
func parsePercent(text string) (float64, error) {
	normalised := strings.ReplaceAll(text, ",", ".")
	value, err := strconv.ParseFloat(normalised, 64)
	if err != nil {
		return 0, fmt.Errorf("masterautoclaim: %q bukan angka: %w", text, err)
	}
	// NaN dan Inf lolos ParseFloat lewat teks "NaN" dan "Inf". Keduanya bukan
	// persentase, dan perbandingan rentang di pemanggil tidak menangkapnya: setiap
	// perbandingan dengan NaN bernilai false.
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, fmt.Errorf("masterautoclaim: %q bukan angka", text)
	}
	return value, nil
}

// Filter menyaring daftar yang dibaca layar.
//
// Ia cerminan langsung dari kedua parameter `Activity/BrowseAutoKlaim_act`:
//
//	Param.stsapprove -> Status
//	Param.komite="ya" -> CommitteeOnly, dengan CommitteeID diisi operator yang masuk
//
// Yang BERBEDA adalah caranya. Pega merangkai penyaring komite menjadi teks SQL:
//
//	TempApproval.CityID := "and KOMITE = '" + OperatorID.pyUserIdentifier + "'"
//	... {ASIS:TempApproval.CityID}
//
// Itu pola `{ASIS:...}` yang `08-TECHNICAL-STRATEGY.md` §4.3 larang tanpa perkecualian:
// nilai dirangkai langsung ke teks SQL. Di sini ia menjadi parameter terikat pada kueri
// terpisah. Hasil yang dikembalikan sama; celah injeksinya tidak ikut.
type Filter struct {
	// Status wajib salah satu dari tiga yang dikenal.
	Status ApprovalStatus

	// CommitteeOnly membatasi daftar pada baris yang KOMITE-nya adalah CommitteeID.
	CommitteeOnly bool

	// CommitteeID adalah operator komite yang sedang masuk. Wajib bila CommitteeOnly.
	CommitteeID string
}

// Repo adalah seam ke penyimpanan master auto claim SATU portal.
//
// Pengisinya ada di repo/sqlstore dan repo/memory. Satu instans selalu terikat pada
// satu basis data entitas — pemisahan antarentitas ada di tingkat koneksi, bukan di
// tingkat kueri (ADR-0030 Opsi 1).
type Repo interface {
	// List mengembalikan baris yang cocok dengan penyaring.
	List(ctx context.Context, filter Filter) ([]AutoClaim, error)

	// Get mengembalikan satu baris; ErrNotFound bila tidak ada.
	Get(ctx context.Context, initial string) (AutoClaim, error)

	// Insert menyisipkan baris baru, dan menolak dengan ErrInitialTaken bila
	// INISIALID-nya sudah dipakai.
	//
	// Pemeriksaan duplikat berada DI DALAM operasi repo, bukan dipecah menjadi "cek"
	// lalu "sisip" di lapisan aplikasi. Sistem lama memecahnya — Call ValidasiAutoClaim
	// pada langkah 4, INSERT pada langkah 8 — dan jarak di antara keduanya adalah
	// lubang balapan yang tidak dijaga apa pun.
	//
	// KETERBATASAN YANG DISADARI. Tanpa constraint unik pada INISIALID, lubang itu
	// hanya dipersempit, tidak ditutup: dua penambahan bersamaan atas kode yang sama
	// masih dapat lolos keduanya. Penutupnya adalah constraint di basis data, dan itu
	// menunggu DDL (R-08) beserta prosedur perubahan skema (D-63).
	Insert(ctx context.Context, ac AutoClaim) error

	// Update menyimpan perubahan pada baris yang sudah ada; ErrNotFound bila barisnya
	// hilang di antara pemuatan layar dan penyimpanan.
	//
	// Ia TIDAK menulis NAMA_PENERIMA — lihat AutoClaim.ReceiverName.
	Update(ctx context.Context, ac AutoClaim) error
}

// RepoSelector memilih Store milik satu portal entitas.
//
// Ia fungsi, bukan map yang sudah jadi, supaya kegagalan memilih portal terbaca pada
// saat permintaan datang — bukan diputuskan sekali saat aplikasi start.
//
// Portal yang tidak dikenal atau koneksinya belum hidup WAJIB menghasilkan galat.
// Mengembalikan repo portal utama sebagai jalan pintas berarti menulis data satu badan
// hukum ke basis data badan hukum lain tanpa satu pun pesan galat (R-20).
type RepoSelector func(portalAlias string) (Store, error)

// Store menyatukan kedua seam yang dipakai layanan modul ini.
//
// Keduanya tetap DIDEKLARASIKAN terpisah — Repo untuk tabel master, LookupRepo untuk
// keempat tabel acuan yang hanya dibaca — karena keduanya menjawab pertanyaan yang
// berbeda dan dapat berubah sendiri-sendiri. Yang disatukan hanyalah CARA MEMILIHNYA:
// keduanya selalu berasal dari koneksi entitas yang sama, sehingga dua pemilih terpisah
// hanya akan membuka kemungkinan keduanya menunjuk entitas yang berbeda.
type Store interface {
	Repo
	LookupRepo
}
