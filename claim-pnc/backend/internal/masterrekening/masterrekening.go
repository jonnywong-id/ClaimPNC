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

// StatusApproval adalah posisi sebuah rekening dalam alur persetujuan komite.
//
// Nilainya sengaja tetap "0", "1", "2" seperti di POOLDATA.LST_ACCOUNT: tabelnya masih
// dibaca dan ditulis sistem lama selama masa paralel (ADR-0003), sehingga mengubah
// sandi nilainya akan membuat kedua sistem membaca baris yang sama secara berbeda.
type StatusApproval string

const (
	// StatusMenunggu — rekening sudah diajukan, komite belum memutuskan.
	StatusMenunggu StatusApproval = "0"
	// StatusDisetujui — komite menyetujui; rekening dapat dipakai membayar klaim.
	StatusDisetujui StatusApproval = "1"
	// StatusDitolak — komite menolak.
	StatusDitolak StatusApproval = "2"
)

// Label mengembalikan sebutan status dalam bahasa yang dibaca pengguna.
func (s StatusApproval) Label() string {
	switch s {
	case StatusMenunggu:
		return "Menunggu"
	case StatusDisetujui:
		return "Komite Approve"
	case StatusDitolak:
		return "Komite Reject"
	default:
		return ""
	}
}

// Dikenal menyatakan status ini termasuk salah satu dari tiga yang sah.
func (s StatusApproval) Dikenal() bool {
	return s == StatusMenunggu || s == StatusDisetujui || s == StatusDitolak
}

// Rekening adalah satu baris master rekening.
//
// Kunci alaminya adalah pasangan NomorRekening + KodeBank — nomor rekening yang sama
// dapat ada di dua bank berbeda, dan sistem lama pun mengunci keduanya bersama-sama
// pada setiap UPDATE.
type Rekening struct {
	NomorRekening string
	NamaPemilik   string
	NamaBank      string
	CabangBank    string
	AlamatBank    string

	// KodeBank adalah LBG_ID pada GENERAL.LST_BANK_GROUP.
	KodeBank string

	// TipeRekening membedakan pemilik rekening: tertanggung, bengkel, rumah sakit,
	// dan seterusnya. Daftarnya data, bukan konstanta.
	TipeRekening string

	// Aktif adalah status pakai rekening, terpisah dari status persetujuan. Rekening
	// yang sudah disetujui masih dapat dinonaktifkan tanpa menghapusnya — data klaim
	// lama tetap merujuknya (ADR-0012).
	Aktif bool

	Email          string
	EmailPenginput string
	Telepon        string

	// NIK adalah identitas pemilik rekening, bukan identitas petugas yang menginput.
	NIK string

	// IDDokumen menunjuk lampiran buku rekening. Wajib terisi sebelum komite dapat
	// menyetujui — komite tidak boleh menyetujui rekening yang tidak dapat dilihat
	// buktinya.
	IDDokumen string

	// Catatan adalah keterangan approval atasan. Wajib terisi saat komite menyetujui.
	Catatan string

	Status         StatusApproval
	KomiteApproval string
	DiputuskanPada *time.Time

	DiinputOleh string
	DiinputPada time.Time
	DiubahOleh  string

	// StatusLayanan menandai hasil pendaftaran ke sistem Kasir.
	StatusLayanan string
	// IDRekeningKasir adalah nomor rekening di sisi Kasir, dikembalikan saat
	// pendaftaran berhasil.
	IDRekeningKasir string
	// ResponsKasir adalah pesan terakhir dari Kasir, sudah dipangkas sampai setelah
	// tanda "]" persis seperti yang ditampilkan layar lama.
	ResponsKasir string

	// FlagPerubahan menandai baris yang lahir dari perubahan rekening klaim berjalan,
	// bukan dari pendaftaran baru.
	FlagPerubahan string

	// Tiga field berikut menyimpan nilai sebelum perubahan. Sistem lama memakainya
	// untuk memberi tahu Kasir rekening mana yang digantikan.
	KodeBankLama      string
	NomorRekeningLama string
	NamaPemilikLama   string
}

// Bank adalah satu baris GENERAL.LST_BANK_GROUP.
type Bank struct {
	Kode string
	Nama string
}

// Kunci adalah identitas alami satu rekening.
type Kunci struct {
	NomorRekening string
	KodeBank      string
}

// KunciDari membaca kunci alami sebuah rekening.
func (r Rekening) KunciDari() Kunci {
	return Kunci{NomorRekening: r.NomorRekening, KodeBank: r.KodeBank}
}

// MenungguKeputusan menyatakan rekening ini belum diputuskan komite.
func (r Rekening) MenungguKeputusan() bool { return r.Status == StatusMenunggu }

// DapatDipakai menyatakan rekening ini boleh menjadi tujuan pembayaran klaim.
//
// Dua syarat, bukan satu: disetujui komite DAN masih aktif. Rekening yang dinonaktifkan
// setelah disetujui tidak boleh dipakai lagi.
func (r Rekening) DapatDipakai() bool { return r.Status == StatusDisetujui && r.Aktif }

var (
	// ErrTidakDitemukan dikembalikan bila kunci yang diminta tidak ada.
	ErrTidakDitemukan = errors.New("masterrekening: rekening tidak ditemukan")

	// ErrSudahAda dikembalikan bila nomor rekening sudah terdaftar dan barisnya
	// BUKAN bekas penolakan komite. Lihat BolehDidaftarkanUlang.
	ErrSudahAda = errors.New("masterrekening: nomor rekening sudah terdaftar")

	// ErrSudahDiputuskan dikembalikan bila komite hendak memutuskan rekening yang
	// keputusannya sudah pernah diambil. Keputusan komite tidak dianulir lewat layar
	// ini; pengajuan baru adalah jalannya.
	ErrSudahDiputuskan = errors.New("masterrekening: keputusan komite sudah pernah diambil")

	// ErrStatusTidakDikenal dikembalikan bila keputusan yang diminta bukan setuju
	// maupun tolak.
	ErrStatusTidakDikenal = errors.New("masterrekening: status keputusan tidak dikenal")
)

// GalatValidasi menyebut seluruh field yang tidak memenuhi syarat sekaligus.
//
// Disebut sekaligus, bukan satu per satu: pengguna yang mengisi sembilan kolom berhak
// tahu seluruh yang kurang dalam satu kali, bukan menemukan satu kesalahan baru pada
// setiap kali menekan simpan.
type GalatValidasi struct {
	Field map[string]string
}

func (g *GalatValidasi) Error() string {
	nama := make([]string, 0, len(g.Field))
	for f := range g.Field {
		nama = append(nama, f)
	}
	// Diurutkan supaya pesannya sama pada setiap pemanggilan — pesan galat yang
	// berubah-ubah urutannya menyulitkan pengujian dan pembacaan log.
	urutkan(nama)
	return "masterrekening: isian tidak lengkap: " + strings.Join(nama, ", ")
}

// Kosong menyatakan tidak ada satu pun field yang bermasalah.
func (g *GalatValidasi) Kosong() bool { return len(g.Field) == 0 }

// Periksa memeriksa kelengkapan sebuah rekening sebelum disimpan.
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
func (r Rekening) Periksa() error {
	galat := &GalatValidasi{Field: map[string]string{}}

	wajib := []struct {
		nama  string
		nilai string
		pesan string
	}{
		{"nomor_rekening", r.NomorRekening, "Nomor rekening wajib diisi."},
		{"nama_pemilik", r.NamaPemilik, "Nama pemilik rekening wajib diisi."},
		{"nama_bank", r.NamaBank, "Nama bank wajib diisi."},
		{"cabang_bank", r.CabangBank, "Nama cabang bank wajib diisi."},
		{"alamat_bank", r.AlamatBank, "Alamat bank wajib diisi."},
		{"kode_bank", r.KodeBank, "Bank wajib dipilih dari daftar."},
		{"tipe_rekening", r.TipeRekening, "Tipe rekening wajib dipilih."},
		{"email", r.Email, "Email wajib diisi."},
		{"nik", r.NIK, "NIK pemilik rekening wajib diisi."},
	}
	for _, w := range wajib {
		if strings.TrimSpace(w.nilai) == "" {
			galat.Field[w.nama] = w.pesan
		}
	}

	// Email diperiksa bentuknya hanya bila terisi; kalau kosong, pesan "wajib diisi"
	// di atas sudah cukup dan menambah pesan kedua untuk field yang sama hanya
	// membingungkan.
	if alamat := strings.TrimSpace(r.Email); alamat != "" && !EmailMasukAkal(alamat) {
		galat.Field["email"] = "Format email tidak benar."
	}
	if alamat := strings.TrimSpace(r.EmailPenginput); alamat != "" && !EmailMasukAkal(alamat) {
		galat.Field["email_penginput"] = "Format email penginput tidak benar."
	}

	if galat.Kosong() {
		return nil
	}
	return galat
}

// PeriksaSebelumDisetujui memeriksa dua syarat tambahan yang hanya berlaku saat komite
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
func (r Rekening) PeriksaSebelumDisetujui() error {
	galat := &GalatValidasi{Field: map[string]string{}}

	if strings.TrimSpace(r.IDDokumen) == "" {
		galat.Field["id_dokumen"] = "Buku rekening wajib diunggah sebelum disetujui."
	}
	if strings.TrimSpace(r.Catatan) == "" {
		galat.Field["catatan"] = "Keterangan approval atasan wajib diisi."
	}

	if galat.Kosong() {
		return nil
	}
	return galat
}

// BolehDidaftarkanUlang menyatakan nomor rekening yang sudah ada masih boleh diajukan
// lagi karena pengajuan sebelumnya DITOLAK komite.
//
// Ini menjaga perilaku sistem lama apa adanya (P-5). Prasyarat aslinya:
//
//	@SizeOfPropertyList(TempValidasi.pxResults)==0 || TempValidasi.pxResults(1).StsAp=="Komite Reject"
//
// CATATAN UTANG TEKNIS. Sistem lama melaksanakannya dengan MENGHAPUS baris yang
// ditolak lalu menyisipkan baris baru (RDB-List DelDataRejectMasterRekening). Itu
// menghilangkan jejak penolakan sebelumnya, dan persis pola yang ADR-0013 perintahkan
// diganti. Penggantinya tidak dikerjakan di sini karena akan mengubah perilaku, dan
// Work Owner memilih paritas lebih dulu. Ia dicatat di docs/keputusan-implementasi.md.
func BolehDidaftarkanUlang(yangAda []Rekening) bool {
	if len(yangAda) == 0 {
		return true
	}
	return yangAda[0].Status == StatusDitolak
}

// Filter mempersempit daftar rekening yang dibaca.
//
// Seluruh field boleh kosong; yang kosong tidak ikut mempersempit. Bentuknya sengaja
// meniru apa yang benar-benar dipakai layar lama — GetDataMasterBank menyusun empat
// potongan WHERE dari properti TempDataBank, ditambah satu cabang per nilai
// Param.stsapprove ("0", "1", "2") dan satu cabang untuk Param.komiteapprove.
type Filter struct {
	// Status membatasi ke satu posisi persetujuan. Kosong berarti seluruhnya.
	Status StatusApproval

	// NomorRekening, NamaPemilik, NamaBank adalah pencarian sebagian, tanpa peduli
	// besar-kecil huruf.
	NomorRekening string
	NamaPemilik   string
	NamaBank      string

	// HanyaKomiteSaya membatasi ke rekening yang menunggu keputusan komite yang
	// sedang masuk. Dipakai tab "Komite Approval".
	HanyaKomiteSaya bool
	// IdentitasKomite adalah komite yang sedang masuk; hanya dipakai bila
	// HanyaKomiteSaya bernilai true.
	IdentitasKomite string

	// Batas dan Lewati adalah paginasi dari server (TKT-U6-001). Batas 0 berarti
	// memakai nilai baku repo, bukan berarti tanpa batas — daftar rekening tumbuh
	// terus dan tidak pernah aman dibaca seluruhnya.
	Batas  int
	Lewati int
}

// Repo adalah seam ke penyimpanan master rekening.
//
// Pengisinya ada di repo/sqlstore (POOLDATA.LST_ACCOUNT) dan repo/memori.
type Repo interface {
	// Daftar membaca rekening yang cocok dengan filter, beserta jumlah seluruh baris
	// yang cocok sebelum dipotong paginasi.
	Daftar(ctx context.Context, f Filter) (baris []Rekening, jumlah int, err error)

	// Ambil membaca satu rekening. Mengembalikan ErrTidakDitemukan bila tidak ada.
	Ambil(ctx context.Context, k Kunci) (Rekening, error)

	// CariNomor membaca seluruh baris dengan nomor rekening tertentu, tanpa peduli
	// banknya. Dipakai pemeriksaan duplikasi, yang di sistem lama memang hanya
	// membandingkan nomornya.
	CariNomor(ctx context.Context, nomor string) ([]Rekening, error)

	// Simpan menyisipkan rekening baru.
	Simpan(ctx context.Context, r Rekening) error

	// Perbarui menulis ulang rekening yang sudah ada.
	Perbarui(ctx context.Context, r Rekening) error

	// HapusYangDitolak membuang baris bekas penolakan komite sebelum pengajuan ulang
	// disisipkan.
	//
	// Namanya menyebut syaratnya supaya tidak pernah terbaca sebagai penghapusan
	// umum: pengisi seam WAJIB menolak menghapus baris yang statusnya bukan
	// StatusDitolak.
	HapusYangDitolak(ctx context.Context, k Kunci) error
}

// BankRepo adalah seam ke daftar bank.
//
// Terpisah dari Repo karena sumbernya tabel lain (GENERAL.LST_BANK_GROUP) yang hanya
// DIBACA aplikasi ini — pemiliknya sistem lain (ADR-0004).
type BankRepo interface {
	Daftar(ctx context.Context) ([]Bank, error)
}

// HasilKasir adalah jawaban sistem Kasir atas pendaftaran satu rekening.
type HasilKasir struct {
	// Berhasil menyatakan Kasir menerima rekening ini.
	Berhasil bool
	// IDRekening adalah nomor rekening di sisi Kasir bila pendaftaran berhasil.
	IDRekening string
	// Pesan adalah keterangan dari Kasir, dipakai apa adanya untuk ResponsKasir.
	Pesan string
	// Kode adalah kode respons mentah. "1" berarti gagal dan "9" berarti gagal yang
	// menuntut PIC diberi tahu — keduanya dibaca dari activity lama.
	Kode string
}

// Kasir adalah seam ke sistem Kasir.
//
// Sistem lama memanggilnya lewat dua Connect-REST berbeda: InjectDataRekeningToKasir
// untuk rekening yang baru disetujui, dan UpdateSearchDataRekeningToKasir untuk
// rekening yang menggantikan rekening lama. Keduanya hanya dipanggil untuk portal ASM
// dan SIMASNET — lihat usecase.Putuskan.
type Kasir interface {
	// Daftarkan mendaftarkan rekening baru ke Kasir.
	Daftarkan(ctx context.Context, r Rekening) (HasilKasir, error)

	// Perbarui memberi tahu Kasir bahwa sebuah rekening menggantikan rekening lama.
	Perbarui(ctx context.Context, r Rekening) (HasilKasir, error)
}

// Penerima adalah orang yang dituju sebuah pemberitahuan.
type Penerima struct {
	Nama  string
	Email string
}

// Peringatan adalah isi pemberitahuan kegagalan pendaftaran ke Kasir.
type Peringatan struct {
	Rekening Rekening

	// Pesan adalah keterangan kegagalan dari Kasir, apa adanya.
	Pesan string

	// Kode adalah kode respons Kasir. "9" adalah satu-satunya yang di sistem lama
	// memicu surel.
	Kode string

	// Diputuskan adalah komite yang baru saja mengambil keputusan.
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
	// docs/keputusan-implementasi.md §11.10.
	Diputuskan Penerima
}

// Notifier adalah seam ke pemberitahuan.
//
// Satu-satunya pemakaiannya di modul ini meniru SendEmailAlertRekening: memberi tahu
// komite ketika pendaftaran ke Kasir gagal dengan kode "9".
type Notifier interface {
	PeringatkanKegagalanKasir(ctx context.Context, p Peringatan) error
}

// PangkasResponsKasir mengambil bagian pesan setelah tanda "]".
//
// Sistem lama melakukannya di dalam SQL:
//
//	SUBSTR(response_kasir, INSTR(response_kasir, ']') + 1) AS "Notes"
//
// Ia dipindahkan ke Go karena itu urusan penyajian, bukan urusan basis data — dan
// karena INSTR bukan fungsi standar yang tersedia sama di PostgreSQL kelak
// (docs/Steering/09-DATABASE-STRATEGY.md §4).
func PangkasResponsKasir(mentah string) string {
	if i := strings.IndexByte(mentah, ']'); i >= 0 {
		return strings.TrimSpace(mentah[i+1:])
	}
	return strings.TrimSpace(mentah)
}

// EmailMasukAkal memeriksa bentuk alamat surel sekadarnya.
//
// Sengaja longgar, dan itu disengaja: satu-satunya cara membuktikan sebuah alamat
// benar adalah mengirim surel ke sana. Validasi yang terlalu ketat justru menolak
// alamat sah — dan layar lama pun hanya memeriksa keberadaan "@" lewat activity
// ValidasiEmailRekening.
func EmailMasukAkal(alamat string) bool {
	alamat = strings.TrimSpace(alamat)
	i := strings.IndexByte(alamat, '@')
	if i <= 0 || i == len(alamat)-1 {
		return false
	}
	// Tidak boleh ada "@" kedua, dan bagian setelahnya harus memuat titik di tengah.
	domain := alamat[i+1:]
	if strings.ContainsRune(domain, '@') {
		return false
	}
	j := strings.IndexByte(domain, '.')
	return j > 0 && j < len(domain)-1
}

// urutkan mengurutkan nama field. Ditulis di sini supaya paket domain tidak perlu
// mengimpor sort hanya untuk satu pemakaian pada jalur galat.
func urutkan(nama []string) {
	for i := 1; i < len(nama); i++ {
		for j := i; j > 0 && nama[j] < nama[j-1]; j-- {
			nama[j], nama[j-1] = nama[j-1], nama[j]
		}
	}
}
