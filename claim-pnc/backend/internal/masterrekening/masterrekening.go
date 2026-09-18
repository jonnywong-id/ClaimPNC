// Package masterrekening adalah inti modul Master Rekening.
//
// # Apa yang dimodelkan di sini
//
// Daftar rekening bank tujuan pembayaran klaim: milik tertanggung, bengkel, rumah
// sakit, dan pihak ketiga lain. Sebuah rekening tidak langsung dapat dipakai — ia
// menunggu persetujuan komite lebih dulu, lalu didaftarkan ke sistem Kasir.
//
// # Kenapa master ini punya alur persetujuan, sementara master lain tidak
//
// Rekening menentukan KE MANA uang klaim dikirim. Salah satu baris di sini berarti
// pembayaran mendarat di rekening yang keliru, dan tidak ada langkah sesudahnya yang
// dapat menangkapnya. Sistem lama sudah memperlakukannya begitu — kolom APPROVAL,
// KOMITE_APPROVAL, dan TANGGALAPPROVEKOMITE ada di POOLDATA.LST_ACCOUNT sejak awal.
//
// TKT-F4-001 mencatat pertanyaan yang masih terbuka: "apakah perubahan master butuh
// alur persetujuan, dan berlaku untuk master yang mana?" Untuk master ini jawabannya
// tidak perlu ditunggu — ia sudah terjawab oleh sistem yang berjalan hari ini.
//
// # Penamaan
//
// Nama field di sini adalah nama domain berbahasa Indonesia, BUKAN alias Pega. Alias
// lama menyesatkan secara aktif dan tidak dibawa masuk:
//
//	CaseID   → STS_AKTIF       (bukan nomor kasus)
//	CoverID  → ACCOUNT_TYPE    (bukan cover)
//	pyCountry→ KOMITE_APPROVAL (bukan negara)
//	KOMISI   → FLAGUPDATE      (bukan komisi)
//	pyID     → USER_INPUT saat dibaca, APPROVAL saat ditulis — satu alias, dua arti
//
// Pemetaan alias→kolom→domain lengkap ada di repo/sqlstore/rekening.sql. Perilakunya
// dipertahankan apa adanya (P-5); yang dibersihkan hanya namanya.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterrekening

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ApprovalStatus adalah posisi sebuah rekening dalam alur persetujuan komite.
//
// Nilainya sengaja tetap "0", "1", "2" seperti di POOLDATA.LST_ACCOUNT: tabelnya masih
// dibaca dan ditulis sistem lama selama masa paralel (ADR-0003), sehingga mengubah
// sandi nilainya akan membuat kedua sistem membaca baris yang sama secara berbeda.
type ApprovalStatus string

const (
	// StatusPending — rekening sudah diajukan, komite belum memutuskan.
	StatusPending ApprovalStatus = "0"
	// StatusApproved — komite menyetujui; rekening dapat dipakai membayar klaim.
	StatusApproved ApprovalStatus = "1"
	// StatusRejected — komite menolak.
	StatusRejected ApprovalStatus = "2"
)

// Label mengembalikan sebutan status dalam bahasa yang dibaca pengguna.
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

// Account adalah satu baris master rekening.
//
// Key alaminya adalah pasangan Number + BankCode — nomor rekening yang sama
// dapat ada di dua bank berbeda, dan sistem lama pun mengunci keduanya bersama-sama
// pada setiap UPDATE.
type Account struct {
	Number      string
	OwnerName   string
	BankName    string
	BankBranch  string
	BankAddress string

	// BankCode adalah LBG_ID pada GENERAL.LST_BANK_GROUP.
	BankCode string

	// AccountType membedakan pemilik rekening: tertanggung, bengkel, rumah sakit,
	// dan seterusnya. Daftarnya data, bukan konstanta.
	AccountType string

	// Active adalah status pakai rekening, terpisah dari status persetujuan. Account
	// yang sudah disetujui masih dapat dinonaktifkan tanpa menghapusnya — data klaim
	// lama tetap merujuknya (ADR-0012).
	Active bool

	Email          string
	SubmitterEmail string
	Phone          string

	// NIK adalah identitas pemilik rekening, bukan identitas petugas yang menginput.
	NIK string

	// DocumentID menunjuk lampiran buku rekening. Wajib terisi sebelum komite dapat
	// menyetujui — komite tidak boleh menyetujui rekening yang tidak dapat dilihat
	// buktinya.
	DocumentID string

	// Note adalah keterangan approval atasan. Wajib terisi saat komite menyetujui.
	Note string

	Status            ApprovalStatus
	CommitteeApproval string
	DecidedAt         *time.Time

	CreatedBy string
	CreatedAt time.Time
	UpdatedBy string

	// ServiceStatus menandai hasil pendaftaran ke sistem Kasir.
	ServiceStatus string
	// CashierAccountID adalah nomor rekening di sisi Kasir, dikembalikan saat
	// pendaftaran berhasil.
	CashierAccountID string
	// CashierResponse adalah pesan terakhir dari Kasir, sudah dipangkas sampai setelah
	// tanda "]" persis seperti yang ditampilkan layar lama.
	CashierResponse string

	// ChangeFlag menandai baris yang lahir dari perubahan rekening klaim berjalan,
	// bukan dari pendaftaran baru.
	ChangeFlag string

	// Tiga field berikut menyimpan nilai sebelum perubahan. Sistem lama memakainya
	// untuk memberi tahu Kasir rekening mana yang digantikan.
	PreviousBankCode  string
	PreviousNumber    string
	PreviousOwnerName string
}

// Bank adalah satu baris GENERAL.LST_BANK_GROUP.
type Bank struct {
	Code string
	Name string
}

// Key adalah identitas alami satu rekening.
type Key struct {
	Number   string
	BankCode string
}

// KeyOf membaca kunci alami sebuah rekening.
func (r Account) KeyOf() Key {
	return Key{Number: r.Number, BankCode: r.BankCode}
}

// AwaitingDecision menyatakan rekening ini belum decided komite.
func (r Account) AwaitingDecision() bool { return r.Status == StatusPending }

// Usable menyatakan rekening ini boleh menjadi tujuan pembayaran klaim.
//
// Dua syarat, bukan satu: disetujui komite DAN masih aktif. Rekening yang dinonaktifkan
// setelah disetujui tidak boleh dipakai lagi.
func (r Account) Usable() bool { return r.Status == StatusApproved && r.Active }

var (
	// ErrNotFound dikembalikan bila kunci yang diminta tidak ada.
	ErrNotFound = errors.New("masterrekening: rekening tidak ditemukan")

	// ErrAlreadyExists dikembalikan bila nomor rekening sudah terdaftar dan barisnya
	// BUKAN bekas penolakan komite. Lihat CanBeResubmitted.
	ErrAlreadyExists = errors.New("masterrekening: nomor rekening sudah terdaftar")

	// ErrAlreadyDecided dikembalikan bila komite hendak memutuskan rekening yang
	// keputusannya sudah pernah diambil. Keputusan komite tidak dianulir lewat layar
	// ini; pengajuan baru adalah jalannya.
	ErrAlreadyDecided = errors.New("masterrekening: keputusan komite sudah pernah diambil")

	// ErrUnknownStatus dikembalikan bila keputusan yang diminta bukan setuju
	// maupun tolak.
	ErrUnknownStatus = errors.New("masterrekening: status keputusan tidak dikenal")
)

// ValidationError menyebut seluruh field yang tidak memenuhi syarat sekaligus.
//
// Disebut sekaligus, bukan satu per satu: pengguna yang mengisi sembilan kolom berhak
// tahu seluruh yang kurang dalam satu kali, bukan menemukan satu kesalahan baru pada
// setiap kali menekan simpan.
type ValidationError struct {
	Field map[string]string
}

func (g *ValidationError) Error() string {
	name := make([]string, 0, len(g.Field))
	for f := range g.Field {
		name = append(name, f)
	}
	// Diurutkan supaya pesannya sama pada setiap pemanggilan — pesan galat yang
	// berubah-ubah urutannya menyulitkan pengujian dan pembacaan log.
	sortRows(name)
	return "masterrekening: isian tidak lengkap: " + strings.Join(name, ", ")
}

// Empty menyatakan tidak ada satu pun field yang bermasalah.
func (g *ValidationError) Empty() bool { return len(g.Field) == 0 }

// Check memeriksa kelengkapan sebuah rekening sebelum disimpan.
//
// Daftar field wajibnya diambil apa adanya dari prasyarat langkah "Set Err Msg" pada
// activity CNMUpdateMasterRekening_act:
//
//	TempBank.NoAccount=="" || TempBank.NameOfBank=="" || TempBank.Name=="" ||
//	TempBank.Address=="" || TempBank.EmailReceiver=="" || TempBank.CoverID=="" ||
//	TempBank.BranchOfBank==""
//
// ditambah satu langkah terpisah sesudahnya untuk TempBank.Nik, dan satu lagi untuk
// TempBank.IDBank. Ketiganya digabung di sini karena pengguna melihatnya sebagai satu
// formulir, bukan tiga.
func (r Account) Check() error {
	issues := &ValidationError{Field: map[string]string{}}

	wajib := []struct {
		name    string
		value   string
		message string
	}{
		{"nomor_rekening", r.Number, "Nomor rekening wajib diisi."},
		{"nama_pemilik", r.OwnerName, "Nama pemilik rekening wajib diisi."},
		{"nama_bank", r.BankName, "Nama bank wajib diisi."},
		{"cabang_bank", r.BankBranch, "Nama cabang bank wajib diisi."},
		{"alamat_bank", r.BankAddress, "Alamat bank wajib diisi."},
		{"kode_bank", r.BankCode, "Bank wajib dipilih dari daftar."},
		{"tipe_rekening", r.AccountType, "Tipe rekening wajib dipilih."},
		{"email", r.Email, "Email wajib diisi."},
		{"nik", r.NIK, "NIK pemilik rekening wajib diisi."},
	}
	for _, w := range wajib {
		if strings.TrimSpace(w.value) == "" {
			issues.Field[w.name] = w.message
		}
	}

	// Email diperiksa bentuknya hanya bila terisi; kalau kosong, pesan "wajib diisi"
	// di atas sudah cukup dan menambah pesan kedua untuk field yang sama hanya
	// membingungkan.
	if address := strings.TrimSpace(r.Email); address != "" && !EmailLooksValid(address) {
		issues.Field["email"] = "Format email tidak benar."
	}
	if address := strings.TrimSpace(r.SubmitterEmail); address != "" && !EmailLooksValid(address) {
		issues.Field["email_penginput"] = "Format email penginput tidak benar."
	}

	if issues.Empty() {
		return nil
	}
	return issues
}

// CheckBeforeApproval memeriksa dua syarat tambahan yang hanya berlaku saat komite
// MENYETUJUI — bukan saat menolak, dan bukan saat rekening disimpan.
//
// Keduanya diambil dari dua langkah Page-Set-Messages pada CNMUpdateMasterRekening_act
// yang prasyaratnya Param.komite=="ya" && Param.APPROVAL=="1":
//
//	"wajib upload file/buku rekening"        → local.flagdok=="1" || TempBank.CoverInsKey!=""
//	"Wajib isi KETERANGAN APPROVAL ATASAN"   → TempBank.pyContext==""
//
// Alasannya masuk akal dan dipertahankan: komite tidak boleh menyetujui rekening yang
// buktinya tidak dapat dilihat, dan alasan persetujuan harus tercatat.
func (r Account) CheckBeforeApproval() error {
	issues := &ValidationError{Field: map[string]string{}}

	if strings.TrimSpace(r.DocumentID) == "" {
		issues.Field["id_dokumen"] = "Buku rekening wajib diunggah sebelum disetujui."
	}
	if strings.TrimSpace(r.Note) == "" {
		issues.Field["catatan"] = "Keterangan approval atasan wajib diisi."
	}

	if issues.Empty() {
		return nil
	}
	return issues
}

// CanBeResubmitted menyatakan nomor rekening yang sudah ada masih boleh diajukan
// lagi karena pengajuan sebelumnya DITOLAK komite.
//
// Ini menjaga perilaku sistem lama apa adanya (P-5). Prasyarat aslinya:
//
//	@SizeOfPropertyList(TempValidasi.pxResults)==0 || TempValidasi.pxResults(1).StsAp=="Committee Reject"
//
// CATATAN UTANG TEKNIS. Sistem lama melaksanakannya dengan MENGHAPUS baris yang
// ditolak lalu menyisipkan baris baru (RDB-List DelDataRejectMasterRekening). Itu
// menghilangkan jejak penolakan sebelumnya, dan persis pola yang ADR-0013 perintahkan
// diganti. Penggantinya tidak dikerjakan di sini karena akan mengubah perilaku, dan
// Work Owner memilih paritas lebih dulu. Ia dicatat di docs/keputusan-implementasi.md.
func CanBeResubmitted(yangAda []Account) bool {
	if len(yangAda) == 0 {
		return true
	}
	return yangAda[0].Status == StatusRejected
}

// Filter mempersempit daftar rekening yang dibaca.
//
// Seluruh field boleh kosong; yang kosong tidak ikut mempersempit. Bentuknya sengaja
// meniru apa yang benar-benar dipakai layar lama — GetDataMasterBank menyusun empat
// potongan WHERE dari properti TempDataBank, ditambah satu cabang per nilai
// Param.stsapprove ("0", "1", "2") dan satu cabang untuk Param.komiteapprove.
type Filter struct {
	// Status membatasi ke satu posisi persetujuan. Empty berarti seluruhnya.
	Status ApprovalStatus

	// Number, OwnerName, BankName adalah pencarian sebagian, tanpa peduli
	// besar-kecil huruf.
	Number    string
	OwnerName string
	BankName  string

	// MyCommitteeOnly membatasi ke rekening yang menunggu keputusan komite yang
	// sedang masuk. Dipakai tab "Komite Approval".
	MyCommitteeOnly bool
	// CommitteeIdentity adalah komite yang sedang masuk; hanya dipakai bila
	// MyCommitteeOnly bernilai true.
	CommitteeIdentity string

	// Limit dan Lewati adalah paginasi dari server (TKT-U6-001). Batas 0 berarti
	// memakai nilai baku repo, bukan berarti tanpa batas — daftar rekening tumbuh
	// terus dan tidak pernah aman dibaca seluruhnya.
	Limit  int
	Offset int
}

// Repo adalah seam ke penyimpanan master rekening.
//
// Pengisinya ada di repo/sqlstore (POOLDATA.LST_ACCOUNT) dan repo/memory.
type Repo interface {
	// List membaca rekening yang cocok dengan filter, beserta jumlah seluruh baris
	// yang cocok sebelum dipotong paginasi.
	List(ctx context.Context, f Filter) (rows []Account, total int, err error)

	// Get membaca satu rekening. Mengembalikan ErrNotFound bila tidak ada.
	Get(ctx context.Context, k Key) (Account, error)

	// FindByNumber membaca seluruh baris dengan nomor rekening tertentu, tanpa peduli
	// banknya. Dipakai pemeriksaan duplikasi, yang di sistem lama memang hanya
	// membandingkan nomornya.
	FindByNumber(ctx context.Context, nomor string) ([]Account, error)

	// Save menyisipkan rekening baru.
	Save(ctx context.Context, r Account) error

	// Update menulis ulang rekening yang sudah ada.
	Update(ctx context.Context, r Account) error

	// ClearRejected membuang baris bekas penolakan komite sebelum pengajuan ulang
	// disisipkan.
	//
	// Namanya menyebut syaratnya supaya tidak pernah terbaca sebagai penghapusan
	// umum: pengisi seam WAJIB menolak menghapus baris yang statusnya bukan
	// StatusRejected.
	ClearRejected(ctx context.Context, k Key) error
}

// BankRepo adalah seam ke daftar bank.
//
// Terpisah dari Repo karena sumbernya tabel lain (GENERAL.LST_BANK_GROUP) yang hanya
// DIBACA aplikasi ini — pemiliknya sistem lain (ADR-0004).
type BankRepo interface {
	List(ctx context.Context) ([]Bank, error)
}

// CashierResult adalah jawaban sistem Kasir atas pendaftaran satu rekening.
type CashierResult struct {
	// Succeeded menyatakan Kasir menerima rekening ini.
	Succeeded bool
	// AccountID adalah nomor rekening di sisi Kasir bila pendaftaran berhasil.
	AccountID string
	// Message adalah keterangan dari Kasir, dipakai apa adanya untuk CashierResponse.
	Message string
	// Code adalah kode respons mentah. "1" berarti gagal dan "9" berarti gagal yang
	// menuntut PIC diberi tahu — keduanya dibaca dari activity lama.
	Code string
}

// Kasir adalah seam ke sistem Kasir.
//
// Sistem lama memanggilnya lewat dua Connect-REST berbeda: InjectDataRekeningToKasir
// untuk rekening yang baru disetujui, dan UpdateSearchDataRekeningToKasir untuk
// rekening yang menggantikan rekening lama. Keduanya hanya dipanggil untuk portal ASM
// dan SIMASNET — lihat usecase.Decide.
type Cashier interface {
	// Register mendaftarkan rekening baru ke Kasir.
	Register(ctx context.Context, r Account) (CashierResult, error)

	// Update memberi tahu Kasir bahwa sebuah rekening menggantikan rekening lama.
	Update(ctx context.Context, r Account) (CashierResult, error)
}

// Recipients adalah orang yang dituju sebuah pemberitahuan.
type Recipients struct {
	Name  string
	Email string
}

// Alert adalah isi pemberitahuan kegagalan pendaftaran ke Kasir.
type Alert struct {
	Account Account

	// Message adalah keterangan kegagalan dari Kasir, apa adanya.
	Message string

	// Code adalah kode respons Kasir. "9" adalah satu-satunya yang di sistem lama
	// memicu surel.
	Code string

	// DecidedBy adalah komite yang baru saja mengambil keputusan.
	//
	// Ia BUKAN penerima peringatan — ia keterangan, supaya penerima tahu kepada siapa
	// harus bertanya. Penerimanya ditetapkan konfigurasi dan diketahui pengisi seam,
	// bukan dikirim dari sini.
	//
	// SATU-SATUNYA PENYIMPANGAN DARI SISTEM LAMA DI MODUL INI, dan disengaja. Rule
	// lama mengirim ke operator yang sedang masuk:
	//
	//	select email from pooldata.mst_user_teknik where operator_id = {TempIns.pyID}
	//
	// Work Owner menetapkan penerimanya adalah mailbox Tim IT (2026-09-17), karena
	// peringatan kegagalan integrasi ditujukan ke pihak yang dapat MEMPERBAIKINYA,
	// bukan ke orang yang kebetulan menekan tombol approve. Dicatat di
	// docs/keputusan-implementasi.md §10.10.
	DecidedBy Recipients
}

// Notifier adalah seam ke pemberitahuan.
//
// Satu-satunya pemakaiannya di modul ini meniru SendEmailAlertRekening: memberi tahu
// komite ketika pendaftaran ke Kasir gagal dengan kode "9".
type Notifier interface {
	WarnCashierFailure(ctx context.Context, p Alert) error
}

// TrimCashierResponse mengambil bagian pesan setelah tanda "]".
//
// Sistem lama melakukannya di dalam SQL:
//
//	SUBSTR(response_kasir, INSTR(response_kasir, ']') + 1) AS "Notes"
//
// Ia dipindahkan ke Go karena itu urusan penyajian, bukan urusan basis data — dan
// karena INSTR bukan fungsi standar yang tersedia sama di PostgreSQL kelak
// (docs/Steering/09-DATABASE-STRATEGY.md §4).
func TrimCashierResponse(mentah string) string {
	if i := strings.IndexByte(mentah, ']'); i >= 0 {
		return strings.TrimSpace(mentah[i+1:])
	}
	return strings.TrimSpace(mentah)
}

// EmailLooksValid memeriksa bentuk alamat surel sekadarnya.
//
// Sengaja longgar, dan itu disengaja: satu-satunya cara membuktikan sebuah alamat
// benar adalah mengirim surel ke sana. Validasi yang terlalu ketat justru menolak
// alamat sah — dan layar lama pun hanya memeriksa keberadaan "@" lewat activity
// ValidasiEmailRekening.
func EmailLooksValid(address string) bool {
	address = strings.TrimSpace(address)
	i := strings.IndexByte(address, '@')
	if i <= 0 || i == len(address)-1 {
		return false
	}
	// Tidak boleh ada "@" kedua, dan bagian setelahnya harus memuat titik di tengah.
	domain := address[i+1:]
	if strings.ContainsRune(domain, '@') {
		return false
	}
	j := strings.IndexByte(domain, '.')
	return j > 0 && j < len(domain)-1
}

// sortRows mengurutkan nama field. Ditulis di sini supaya paket domain tidak perlu
// mengimpor sort hanya untuk satu pemakaian pada jalur galat.
func sortRows(name []string) {
	for i := 1; i < len(name); i++ {
		for j := i; j > 0 && name[j] < name[j-1]; j-- {
			name[j], name[j-1] = name[j-1], name[j]
		}
	}
}
