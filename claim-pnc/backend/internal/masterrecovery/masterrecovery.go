// Package masterrecovery adalah inti modul Master Recovery (`F-4`, butir menu
// `MENU_ID 16`).
//
// # Apa yang dikerjakan layar ini
//
// Mencatat **pemulihan dana klaim dari pihak penjamin** — satu baris per batch, ke
// POOLDATA.MST_RECOVERY_ASM_PENJAMINAN. Setiap batch menyebut principal, tahun, nilai
// klaim, pembayaran, dan sisanya, ditambah bukti bayar serta daftar polis yang tercakup.
//
// # Bentuknya BUKAN CRUD master, dan itu bukan penyederhanaan
//
// Diperiksa ke seluruh export rule Pega, bukan diandaikan:
//
//   - **Tidak ada satu pun kueri yang MEMBACA tabel itu.** Satu-satunya rule yang
//     menyentuhnya adalah `RDB List/GetMasterRecoveryClaimSPK-SQL.xml`, dan isinya
//     `select nvl(max(BATCH),0)+1` — penerbit nomor, bukan pembaca daftar.
//   - **Tidak ada UPDATE dan tidak ada DELETE.** `Database/INSERTMASTERRECOVERYKLAIM.prc`
//     hanya mengenal INSERT.
//   - Layarnya sendiri, `Section/OutstandingMasterRecovery-Section.xml`, adalah FORM
//     ENTRI; grid yang ada di dalamnya menampilkan baris CSV yang baru diunggah, bukan
//     isi tabel.
//
// Keputusan Work Owner 2026-09-19: ditiru apa adanya — entri saja, tanpa daftar, tanpa
// Ubah, tanpa Hapus.
//
// # Asal setiap aturan di berkas ini
//
//	Harness/MasterRecovery-Harness.xml             kerangka layar; dua panel
//	Section/OutstandingMasterRecovery-Section.xml   form entri beserta urutan isiannya
//	Section/DetailMasterRecovery-Section.xml        panel penerbitan Virtual Account
//	Activity/HitungSisaKlaimRecovery-Act.xml        aturan Sisa — lihat Remainder
//	Activity/Insert_mst_recoveryKlaimASM-Act.xml    aksi Transfer Recovery; isian wajib
//	Activity/GetIDMasterRecoveryKlaim-Act.xml       penerbitan nomor batch
//	Activity/GeneratedVAClaimRecovery-Act.xml       penerbitan VA beserta pakai-ulangnya
//	Activity/FlagRecoveryclaims-Act.xml             sakelar tampilan panel VA
//	RDB List/GetRecoveryClaimData-SQL.xml           pencarian identitas polis
//	RDB List/InsertMasterRecoveryKlaimASM-SQL.xml   pemetaan isian ke kolom
//	Database/INSERTMASTERRECOVERYKLAIM.prc          source procedure-nya
//	Database/ADD_NEWMASTERVIRTUALACCOUNT.prc        master VA
//	Database/SET_ATTACHMENT_64BIT.prc               penyimpanan Bukti Bayar
//
// Seluruh bentuk kolom diverifikasi langsung ke ALL_TAB_COLUMNS portal ASM pada
// 2026-09-19, bukan disimpulkan dari nama parameter procedure.
//
// Lapisan Domain — dilarang mengimpor HTTP, SQL, maupun driver basis data.
package masterrecovery

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"unicode/utf8"
)

// Batas panjang isian teks.
//
// Ketiganya DIBACA dari ALL_TAB_COLUMNS portal ASM pada 2026-09-19, bukan dikarang —
// itulah sebabnya angkanya ganjil dan tidak bulat:
//
//	NAMAPRINCIPAL  VARCHAR2(200)
//	TAHUN          VARCHAR2(20)    → dibatasi lebih ketat; lihat CheckYear
//	KETERANGAN     VARCHAR2(2000)
//	POSISIKASUS    VARCHAR2(100)
//	CLIENTID       VARCHAR2(100)
//	NOVA           VARCHAR2(1000)
//	NOPOLIS        VARCHAR2(200)
//
// Menolaknya di sini membuat pengguna melihat pesan yang dapat ditindaklanjuti, bukan
// ORA-12899 yang tidak menyebut isian mana yang kepanjangan.
//
// Angka yang sama diulang di frontend supaya pengguna tahu sebelum mengirim; server tetap
// yang berwenang. Bila salah satu berubah, KEDUA tempat wajib ikut berubah.
const (
	MaxPrincipalNameLength  = 200
	MaxRemarkLength         = 2000
	MaxCasePositionLength   = 100
	MaxClientIDLength       = 100
	MaxVirtualAccountLength = 1000
	MaxPolicyNoLength       = 200
)

// Amount adalah nilai uang dalam RUPIAH UTUH.
//
// # Kenapa bilangan bulat, bukan pecahan
//
// `docs/Steering/09-DATABASE-STRATEGY.md` §5 melarang float untuk nilai uang tanpa
// perkecualian — pembulatan floating point membuat perbandingan gagal secara acak dan
// tidak dapat direproduksi. Yang tersisa adalah bilangan bulat atau titik-tetap.
//
// Rupiah utuh dipilih setelah memeriksa basis datanya, bukan karena selera:
//
//   - Kolom NILAIKLAIM, NILAIRECOVERY, PEMBAYARAN, dan SISAKLAIM bertipe NUMBER tanpa
//     presisi dan tanpa skala — basis data MENERIMA pecahan, jadi ia tidak membatasi apa
//     pun.
//   - Ketiga baris yang ada sejak 2025 SELURUHNYA bilangan bulat. Kueri
//     `NILAIKLAIM <> TRUNC(NILAIKLAIM) OR …` pada 2026-09-19 mengembalikan NOL baris.
//   - Nilai klaim rupiah tidak pernah dicatat sampai sen dalam praktik.
//
// KONSEKUENSI YANG DISADARI: isian bernilai pecahan DITOLAK, sementara layar Pega akan
// menerimanya. Itu selisih perilaku yang disengaja, dan ia dipilih karena menerima
// pecahan diam-diam lalu memotongnya jauh lebih berbahaya daripada menolaknya dengan
// pesan. Bila kelak sen benar-benar dibutuhkan, yang berubah adalah tipe ini menjadi
// titik-tetap dua desimal — kolomnya sendiri tidak perlu disentuh.
type Amount int64

// ParseAmount membaca nilai uang dari teks dan menolak yang bukan rupiah utuh.
//
// Ia menerima bentuk yang benar-benar diketik orang — spasi tepi, dan pemisah ribuan
// berupa titik maupun koma — karena isian di layar lama pun memformat angkanya. Yang
// ditolak hanyalah PECAHAN, dan penolakannya menyebut alasannya.
func ParseAmount(text string) (Amount, error) {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return 0, ErrAmountEmpty
	}

	// Pemisah ribuan dibuang lebih dulu. Keduanya dibuang, bukan salah satu: layar lama
	// tidak menyeragamkan format, sehingga "1.000.000" dan "1,000,000" sama-sama muncul.
	clean = strings.ReplaceAll(clean, ".", "")
	clean = strings.ReplaceAll(clean, ",", "")

	value, err := strconv.ParseInt(clean, 10, 64)
	if err != nil {
		return 0, ErrAmountInvalid
	}
	return Amount(value), nil
}

// ClaimLine adalah satu polis beserta nilai klaimnya di dalam sebuah batch recovery.
//
// Daftarnya diunggah sebagai CSV, ditampilkan sebagai grid, lalu disimpan UTUH sebagai
// dokumen JSON di kolom JSON_POLIS — bukan sebagai tabel anak. Itu mengikuti sistem lama:
// `Activity/Insert_mst_recoveryKlaimASM-Act.xml` menyusun `TempRecovery.ObjectList` lalu
// mengubahnya menjadi teks dengan `@GCNM.GetPageJSONString()` dan mengirimnya sebagai
// parameter `tJSON_POLIS`.
type ClaimLine struct {
	// PolicyNo adalah nomor polis yang tercakup batch ini.
	PolicyNo string

	// ClaimAmount adalah nilai klaim polis tersebut.
	ClaimAmount Amount
}

// Recovery adalah satu batch pemulihan dana yang akan dicatat.
//
// Nama field mengikuti ARTINYA (`D-19`, `D-80`), bukan nama properti Pega maupun nama
// kolomnya. Itu bukan kerapian: properti di sistem lama dipakai untuk hal yang sama
// sekali berbeda dari namanya, dan membawa namanya berarti membawa kekeliruannya.
//
// Tiga yang paling menyesatkan, seluruhnya terbukti di
// `RDB List/InsertMasterRecoveryKlaimASM-SQL.xml`:
//
//	TempRecovery.NoKTPMasking  → POSISIKASUS   ia POSISI KASUS, bukan NIK tersamar
//	TempRecovery.TPLAmount     → SISAKLAIM     ia SISA, bukan nilai tanggung jawab pihak ketiga
//	TempRecovery.Idlogservice  → NOHPLL        ia ID LOG LAYANAN, bukan nomor telepon
//
// Pemetaan ke kolomnya ada di repo/sqlstore, satu tempat saja.
type Recovery struct {
	// Batch adalah kolom BATCH, kunci utama tabel. Diterbitkan penyimpanan, tidak pernah
	// datang dari pemanggil.
	Batch int64

	// PrincipalName adalah NAMAPRINCIPAL — pihak penjamin yang mengembalikan dana.
	PrincipalName string

	// ClientID adalah CLIENTID, identitas principal di master Virtual Account.
	ClientID string

	// VirtualAccountNumber adalah NOVA, rekening virtual tempat dana diterima.
	VirtualAccountNumber string

	// Year adalah TAHUN — tahun buku batch ini.
	Year string

	// ClaimAmount adalah NILAIKLAIM.
	ClaimAmount Amount

	// PreviousPayment adalah NILAIRECOVERY, berlabel "Nilai Pembayaran Sebelumnya" di
	// layar. Namanya di basis data justru kebalikan dari perannya, dan perannya itulah
	// yang menentukan rumus Sisa — lihat Remainder.
	PreviousPayment Amount

	// Payment adalah PEMBAYARAN — yang dibayarkan pada batch ini.
	Payment Amount

	// Remainder adalah SISAKLAIM. TIDAK diterima dari pemanggil: ia dihitung server
	// dengan Remainder(), supaya rumusnya hidup di satu tempat.
	Remainder Amount

	// Remark adalah KETERANGAN.
	Remark string

	// CasePosition adalah POSISIKASUS — keterangan posisi perkara. Teks bebas; dua nilai
	// yang ada di portal ASM membuktikannya bukan daftar kode.
	CasePosition string

	// DocumentID adalah DOKUMENID, menunjuk baris POOLDATA.DATA_ATTACHFILE berisi Bukti
	// Bayar. Kosong berarti bukti bayar belum diunggah.
	DocumentID string

	// InputBy adalah USERNAME — identitas petugas yang mencatat. Diisi server dari sesi,
	// tidak pernah dari badan permintaan.
	InputBy string

	// ServiceLogID adalah kolom NOHPLL.
	//
	// Namanya menyiratkan nomor telepon; isinya BUKAN. `Insert_mst_recoveryKlaimASM`
	// mengisinya dari `TempRecovery.Idlogservice`, yaitu nomor catatan log layanan.
	// Dibawa apa adanya supaya kolomnya tidak berubah arti, dan dinamai menurut isinya
	// supaya kekeliruannya berhenti di sini.
	ServiceLogID string

	// PolicyNo adalah NOPOLIS — polis acuan yang dipakai mencari identitas di bawah.
	PolicyNo string

	// Keempat berikut adalah LBU_ID, LDC_ID, LAG_AGEN_ID, dan LMO_ID: identitas lini
	// bisnis, cabang, agen, dan marketing officer. Tidak diketik pengguna — dicari dari
	// nomor polis lewat Repo.LookupPolicy, meniru `GetRecoveryClaimData-SQL.xml`.
	BusinessID  string
	BranchID    string
	AgentID     string
	MarketingID string

	// ClaimLine adalah isi JSON_POLIS.
	ClaimLine []ClaimLine
}

// Clean mengembalikan salinan dengan spasi tepi seluruh isian teks dibuang.
//
// Perapian ini nyata gunanya: isian layar lama tidak pernah dirapikan, sehingga nama
// principal yang sama dapat tersimpan dengan dan tanpa spasi tepi lalu gagal dicocokkan
// saat VA-nya dicari kembali.
func (r Recovery) Clean() Recovery {
	r.PrincipalName = strings.TrimSpace(r.PrincipalName)
	r.ClientID = strings.TrimSpace(r.ClientID)
	r.VirtualAccountNumber = strings.TrimSpace(r.VirtualAccountNumber)
	r.Year = strings.TrimSpace(r.Year)
	r.Remark = strings.TrimSpace(r.Remark)
	r.CasePosition = strings.TrimSpace(r.CasePosition)
	r.DocumentID = strings.TrimSpace(r.DocumentID)
	r.InputBy = strings.TrimSpace(r.InputBy)
	r.ServiceLogID = strings.TrimSpace(r.ServiceLogID)
	r.PolicyNo = strings.TrimSpace(r.PolicyNo)
	r.BusinessID = strings.TrimSpace(r.BusinessID)
	r.BranchID = strings.TrimSpace(r.BranchID)
	r.AgentID = strings.TrimSpace(r.AgentID)
	r.MarketingID = strings.TrimSpace(r.MarketingID)

	line := make([]ClaimLine, 0, len(r.ClaimLine))
	for _, l := range r.ClaimLine {
		l.PolicyNo = strings.TrimSpace(l.PolicyNo)
		line = append(line, l)
	}
	r.ClaimLine = line
	return r
}

// Remainder menghitung Sisa Klaim.
//
// # Aturannya, dan buktinya
//
// Diturunkan dari `Activity/HitungSisaKlaimRecovery-Act.xml`, yang berisi TEPAT DUA
// langkah Property-Set dengan prasyarat yang saling meniadakan:
//
//	:387  TPLAmount := ClaimAmountAdjust - TotalListClaimAmountIDR   bila NilaiDeductible == 0
//	:518  TPLAmount := ClaimAmountAdjust - NilaiDeductible           bila NilaiDeductible  > 0
//
// Dalam istilah layar:
//
//	Nilai Pembayaran Sebelumnya = 0  →  Sisa = Nilai Klaim − Pembayaran
//	Nilai Pembayaran Sebelumnya > 0  →  Sisa = Nilai Klaim − Nilai Pembayaran Sebelumnya
//
// # Kenapa cabang kedua mengabaikan Pembayaran, dan kenapa itu TIDAK "diperbaiki"
//
// Membacanya sekilas, cabang kedua tampak keliru: pembayaran batch berjalan seolah tidak
// mengurangi sisa. Ketiga baris produksi membuktikan itu memang perilakunya, bukan salah
// baca — diperiksa langsung ke portal ASM pada 2026-09-19:
//
//	klaim 160.000 · sebelumnya 0     · bayar   5.000 → sisa 155.000 = 160.000 − 5.000
//	klaim 160.000 · sebelumnya 5.000 · bayar   2.000 → sisa 155.000 = 160.000 − 5.000
//	klaim 160.000 · sebelumnya 7.000 · bayar 100.000 → sisa 153.000 = 160.000 − 7.000
//
// `P-5` menetapkan perilaku dipertahankan lebih dulu dan diperbaiki kemudian, dan aturan
// ini TIDAK ada di daftar 13 perbaikan eksplisit `D-49`. Mengubahnya di sini akan
// membuat setiap selisih pada uji kesetaraan tidak dapat dijelaskan. Bila ia memang
// cacat, perbaikannya menempuh keputusan tertulis — bukan diputuskan diam-diam saat
// menulis kode.
func Remainder(claimAmount, previousPayment, payment Amount) Amount {
	if previousPayment == 0 {
		return claimAmount - payment
	}
	return claimAmount - previousPayment
}

// Principal adalah satu baris master Virtual Account
// (POOLDATA.MST_VIRTUAL_ACCOUNT_PNC) — pilihan pada isian "Nama Principal".
type Principal struct {
	ClientID             string
	Name                 string
	VirtualAccountNumber string

	// Email adalah EMAILVA, surel petugas yang dulu menerbitkan VA ini.
	Email string

	// Status dan Message adalah jawaban terakhir layanan penerbit VA, disimpan apa adanya
	// pada saat penerbitan. Keduanya dibawa supaya layar dapat menerangkan keadaan sebuah
	// VA tanpa menembak layanan luar lagi.
	Status  string
	Message string
}

// PolicyReference adalah identitas yang menempel pada sebuah nomor polis.
//
// Menggantikan `RDB List/GetRecoveryClaimData-SQL.xml`, yang membaca empat kolom dari
// baris polis dan menyalinnya ke batch recovery.
type PolicyReference struct {
	BusinessID  string
	BranchID    string
	AgentID     string
	MarketingID string
}

// Document adalah Bukti Bayar yang diunggah petugas.
//
// Isinya disimpan sebagai BLOB pada POOLDATA.DATA_ATTACHFILE — kolom ATTACHFILE, yang
// keberadaannya diverifikasi langsung ke katalog pada 2026-09-19. Karena itu modul ini
// TIDAK bergantung pada API penyimpanan dokumen luar (`D-16`, modul `S-1`) yang belum
// dibangun: berkasnya cukup ditulis ke tabel yang sudah ada.
type Document struct {
	// Name adalah nama berkas apa adanya dari peramban. Kolom ATTACHNAME VARCHAR2(255).
	Name string

	// MimeType adalah jenis isinya. Kolom ATTACHMIMETYPE VARCHAR2(30).
	MimeType string

	// Note adalah keterangan singkat. Kolom ATTACHNOTE VARCHAR2(255).
	Note string

	// Content adalah isi berkasnya.
	Content []byte

	// UploadedBy adalah identitas pengunggah. Kolom INPUTOPERATOR VARCHAR2(150).
	UploadedBy string
}

// Batas unggahan Bukti Bayar.
//
// MaxDocumentBytes ditetapkan, bukan dibaca: kolom ATTACHFILE bertipe BLOB dan tidak
// membatasi apa pun yang berguna. Lima megabita memadai untuk pindaian bukti transfer,
// dan menolak yang lebih besar menjaga satu unggahan tidak menghabiskan memori proses
// yang sedang melayani pengguna lain.
//
// MaxDocumentNameLength dan MaxDocumentNoteLength DIBACA dari ALL_TAB_COLUMNS.
const (
	MaxDocumentBytes      = 5 << 20
	MaxDocumentNameLength = 255
	MaxDocumentNoteLength = 255
	MaxMimeTypeLength     = 30
)

// Repo adalah seam ke penyimpanan Master Recovery SATU portal.
//
// Satu instans Repo selalu terikat pada satu basis data entitas — pemisahan antarentitas
// ada di tingkat KONEKSI, bukan di tingkat penyaringan baris (`ADR-0030` Opsi 1). Tidak
// ada satu pun kueri di pengisinya yang menyaring berdasarkan entitas, dan memang tidak
// boleh ada.
type Repo interface {
	// NextBatch mengembalikan nomor batch berikutnya, meniru
	// `select nvl(max(BATCH),0)+1`. Ia hanya PERKIRAAN untuk ditampilkan di layar; yang
	// mengikat adalah nomor yang diterbitkan Insert di dalam transaksinya sendiri.
	NextBatch(ctx context.Context) (int64, error)

	// Insert menyimpan satu batch dan mengembalikannya lengkap dengan nomor batch yang
	// diterbitkan penyimpanan.
	Insert(ctx context.Context, recovery Recovery) (Recovery, error)

	// ListPrincipal mengembalikan seluruh principal di master Virtual Account.
	ListPrincipal(ctx context.Context) ([]Principal, error)

	// FindPrincipal mencari satu principal menurut ClientID DAN nama, tanpa membedakan
	// besar-kecil huruf — pencocokan yang sama dipakai
	// `Database/ADD_NEWMASTERVIRTUALACCOUNT.prc`. ErrPrincipalNotFound bila tidak ada.
	FindPrincipal(ctx context.Context, clientID, name string) (Principal, error)

	// SavePrincipal mencatat principal beserta VA yang baru diterbitkan.
	SavePrincipal(ctx context.Context, principal Principal) error

	// LookupPolicy mencari identitas yang menempel pada sebuah nomor polis.
	// ErrPolicyNotFound bila polisnya tidak dikenal.
	LookupPolicy(ctx context.Context, policyNo string) (PolicyReference, error)

	// SaveDocument menyimpan Bukti Bayar dan mengembalikan DATAID-nya.
	SaveDocument(ctx context.Context, document Document) (string, error)
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

// VirtualAccountRequest adalah permintaan penerbitan satu rekening virtual.
type VirtualAccountRequest struct {
	ClientID      string
	PrincipalName string

	// Email adalah surel petugas yang menerbitkan — isian "Email Inputor VA", satu-satunya
	// isian yang ditandai WAJIB pada panel VA di layar lama.
	Email string
}

// VirtualAccount adalah jawaban layanan penerbit.
type VirtualAccount struct {
	Number  string
	Status  string
	Message string

	// Reused menyatakan nomor ini DIAMBIL dari master, bukan baru diterbitkan. Ia
	// dibedakan supaya layar dapat mengatakannya terang-terangan; sistem lama
	// menyampaikannya lewat kalimat di dalam pesan, yang tidak dapat dibaca program.
	Reused bool
}

// VirtualAccountIssuer adalah seam ke layanan penerbit rekening virtual.
//
// Pengisinya ada di virtualaccount/ — adapter nyata yang menembak alamat pada
// POOLDATA.GCNM_CONNECT_REST (`TYPESERVICE = 'GENERATEDVA'`), dan sebuah fake untuk
// pengujian serta pengembangan tanpa jaringan. Dua adapter nyata itulah yang membuat seam
// ini benar-benar seam, bukan abstraksi hipotetis
// (`docs/Steering/04-FUTURE-ARCHITECTURE.md` §3).
type VirtualAccountIssuer interface {
	Issue(ctx context.Context, portalAlias string, request VirtualAccountRequest) (VirtualAccount, error)
}

// CheckRecovery mengumpulkan SELURUH pelanggaran isian, bukan berhenti pada yang pertama.
//
// Mengumpulkan semuanya adalah kesetaraan perilaku, bukan selera: sistem lama memeriksa
// ketujuh isian wajib sekaligus dan menjawabnya dengan satu pesan
// (`Insert_mst_recoveryKlaimASM-Act.xml:802`, "Wajib ISI semua field"). Mengembalikannya
// satu per satu akan membuat pengguna menekan Simpan berkali-kali untuk menemukan
// kesalahan berikutnya.
//
// # Apa yang WAJIB, dan dari mana daftarnya
//
// Prasyarat langkah penolakan di activity lama menyebut tepat tujuh properti:
//
//	NamaPrincipal · TreatyYear · ClaimAmountAdjust · TotalListClaimAmountIDR ·
//	TPLAmount · Remark · NoKTPMasking
//
// Sisa (TPLAmount) tidak diperiksa di sini karena ia DIHITUNG server, sehingga ia tidak
// pernah dapat kosong. Keenam lainnya diperiksa.
//
// # Yang sengaja diperketat
//
// Nilai uang NEGATIF ditolak. Layar lama menerimanya — prasyaratnya hanya menguji
// `== ""`, sehingga "-5000" lolos. Menerima nilai klaim negatif tidak punya arti bisnis
// apa pun, dan akibatnya baru terlihat jauh di kemudian hari pada laporan. Ini selisih
// terencana, dan ia dicatat supaya terbaca sebagai keputusan.
//
// Nilai NOL tetap DITERIMA, mengikuti sistem lama: "0" bukan "", dan baris produksi
// pertama memang bernilai sebelumnya nol.
func CheckRecovery(recovery Recovery) []Violation {
	clean := recovery.Clean()
	var violation []Violation

	add := func(field, message string) {
		violation = append(violation, Violation{Field: field, Message: message})
	}

	if clean.PrincipalName == "" {
		add(FieldPrincipalName, "Nama principal wajib diisi.")
	} else if tooLong(clean.PrincipalName, MaxPrincipalNameLength) {
		add(FieldPrincipalName, lengthMessage("Nama principal", MaxPrincipalNameLength))
	}

	violation = append(violation, CheckYear(clean.Year)...)

	if clean.Remark == "" {
		add(FieldRemark, "Keterangan wajib diisi.")
	} else if tooLong(clean.Remark, MaxRemarkLength) {
		add(FieldRemark, lengthMessage("Keterangan", MaxRemarkLength))
	}

	if clean.CasePosition == "" {
		add(FieldCasePosition, "Posisi kasus wajib diisi.")
	} else if tooLong(clean.CasePosition, MaxCasePositionLength) {
		add(FieldCasePosition, lengthMessage("Posisi kasus", MaxCasePositionLength))
	}

	if tooLong(clean.ClientID, MaxClientIDLength) {
		add(FieldClientID, lengthMessage("Client ID", MaxClientIDLength))
	}
	if tooLong(clean.VirtualAccountNumber, MaxVirtualAccountLength) {
		add(FieldVirtualAccount, lengthMessage("Nomor virtual account", MaxVirtualAccountLength))
	}
	if tooLong(clean.PolicyNo, MaxPolicyNoLength) {
		add(FieldPolicyNo, lengthMessage("Nomor polis", MaxPolicyNoLength))
	}

	if clean.ClaimAmount < 0 {
		add(FieldClaimAmount, "Nilai klaim tidak boleh negatif.")
	}
	if clean.PreviousPayment < 0 {
		add(FieldPreviousPayment, "Nilai pembayaran sebelumnya tidak boleh negatif.")
	}
	if clean.Payment < 0 {
		add(FieldPayment, "Pembayaran tidak boleh negatif.")
	}

	for index, line := range clean.ClaimLine {
		position := strconv.Itoa(index + 1)
		if strings.TrimSpace(line.PolicyNo) == "" {
			add(FieldClaimLine, "Baris "+position+" pada daftar klaim tidak menyebut nomor polis.")
		}
		if line.ClaimAmount < 0 {
			add(FieldClaimLine, "Baris "+position+" pada daftar klaim bernilai negatif.")
		}
	}

	return violation
}

// CheckYear memeriksa isian Tahun.
//
// # Kenapa aturannya ditetapkan, bukan dibaca
//
// Layar lama memakai dropdown yang diisi Data Transform `GetListYear` — dan rule itu
// TIDAK ADA di export. Ia satu dari ±242 rule yang hilang (`R-16`), dan isinya tidak
// dapat dibaca dari mana pun.
//
// Yang diketahui pasti hanyalah bentuknya: kolom TAHUN bertipe VARCHAR2(20), dan ketiga
// baris yang ada berisi "2018" — empat angka. Karena itu yang ditegakkan di sini adalah
// BENTUKNYA, bukan daftar nilainya: empat angka, dalam rentang yang masuk akal untuk
// tahun buku.
//
// Menebak isi daftarnya akan mengarang aturan bisnis; menerima apa saja akan membiarkan
// "20188" tersimpan tanpa ada yang menyadarinya. Memeriksa bentuk adalah yang paling
// jujur di antara keduanya, dan batas rentangnya dicatat supaya dapat dikoreksi Work
// Owner begitu daftar sebenarnya tiba.
func CheckYear(year string) []Violation {
	clean := strings.TrimSpace(year)
	if clean == "" {
		return []Violation{{Field: FieldYear, Message: "Tahun wajib diisi."}}
	}

	value, err := strconv.Atoi(clean)
	if err != nil || len(clean) != 4 {
		return []Violation{{Field: FieldYear, Message: "Tahun harus empat angka, misalnya 2026."}}
	}
	if value < MinYear || value > MaxYear {
		return []Violation{{
			Field:   FieldYear,
			Message: "Tahun harus antara " + strconv.Itoa(MinYear) + " dan " + strconv.Itoa(MaxYear) + ".",
		}}
	}
	return nil
}

// Rentang tahun yang diterima.
//
// DITETAPKAN, bukan dibaca — lihat CheckYear. Batas bawahnya mengikuti baris tertua yang
// benar-benar ada (2018) dengan kelonggaran ke belakang untuk batch lama yang belum
// dicatat; batas atasnya memberi ruang satu tahun buku ke depan.
const (
	MinYear = 2000
	MaxYear = 2100
)

// CheckDocument memeriksa Bukti Bayar sebelum ia menyentuh penyimpanan.
func CheckDocument(document Document) []Violation {
	var violation []Violation
	add := func(message string) {
		violation = append(violation, Violation{Field: FieldDocument, Message: message})
	}

	if len(document.Content) == 0 {
		add("Berkas bukti bayar kosong.")
	}
	if len(document.Content) > MaxDocumentBytes {
		add("Berkas bukti bayar paling besar 5 MB.")
	}
	if strings.TrimSpace(document.Name) == "" {
		add("Nama berkas bukti bayar tidak terbaca.")
	} else if tooLong(document.Name, MaxDocumentNameLength) {
		add(lengthMessage("Nama berkas", MaxDocumentNameLength))
	}
	if tooLong(document.Note, MaxDocumentNoteLength) {
		add(lengthMessage("Keterangan berkas", MaxDocumentNoteLength))
	}
	if tooLong(document.MimeType, MaxMimeTypeLength) {
		// Bukan kerapian: kolom ATTACHMIMETYPE hanya VARCHAR2(30), dan jenis isi yang
		// panjang — sebagian bentuk Office memakai lebih dari 60 karakter — akan ditolak
		// basis data dengan ORA-12899 yang tidak menyebut sebabnya.
		add("Jenis berkas tidak dikenali penyimpanan dokumen.")
	}

	return violation
}

// CheckVirtualAccountRequest memeriksa isian panel penerbitan VA.
//
// Ketiganya wajib. Di layar lama hanya "Email Inputor VA" yang ditandai `pyRequired`,
// tetapi Client ID dan Nama Principal keduanya dipakai sebagai KUNCI pencarian VA yang
// sudah ada (`upper(CLIENTID)` dan `upper(CLIENTNAME)` pada
// `ADD_NEWMASTERVIRTUALACCOUNT.prc`). Salah satu yang kosong membuat pencocokan itu
// mencocokkan hal yang salah, dan akibatnya VA ganda untuk principal yang sama.
func CheckVirtualAccountRequest(request VirtualAccountRequest) []Violation {
	var violation []Violation
	add := func(field, message string) {
		violation = append(violation, Violation{Field: field, Message: message})
	}

	if strings.TrimSpace(request.ClientID) == "" {
		add(FieldClientID, "Client ID wajib diisi.")
	} else if tooLong(strings.TrimSpace(request.ClientID), MaxClientIDLength) {
		add(FieldClientID, lengthMessage("Client ID", MaxClientIDLength))
	}

	if strings.TrimSpace(request.PrincipalName) == "" {
		add(FieldPrincipalName, "Nama principal wajib diisi.")
	} else if tooLong(strings.TrimSpace(request.PrincipalName), MaxPrincipalNameLength) {
		add(FieldPrincipalName, lengthMessage("Nama principal", MaxPrincipalNameLength))
	}

	email := strings.TrimSpace(request.Email)
	switch {
	case email == "":
		add(FieldEmail, "Email inputor VA wajib diisi.")
	case !strings.Contains(email, "@"), strings.HasPrefix(email, "@"), strings.HasSuffix(email, "@"):
		// Pemeriksaan sengaja dangkal. Satu-satunya cara membuktikan sebuah alamat surel
		// benar adalah mengirim surat ke sana; pola yang rumit hanya menolak alamat sah
		// yang tidak umum. Yang dicegah di sini adalah salah ketik yang nyata.
		add(FieldEmail, "Email inputor VA belum berupa alamat surel.")
	case tooLong(email, 100):
		// Kolom EMAILVA VARCHAR2(100), dibaca dari katalog.
		add(FieldEmail, lengthMessage("Email inputor VA", 100))
	}

	return violation
}

// PrincipalKey adalah bentuk pembanding principal yang dipakai menguji "sudah ada atau
// belum".
//
// Mengabaikan besar-kecil huruf dan spasi tepi, mengikuti pencocokan di
// `ADD_NEWMASTERVIRTUALACCOUNT.prc` yang memakai `upper(...)` pada kedua kolom. Menirunya
// persis penting: bila di sini lebih ketat, aplikasi akan menerbitkan VA kedua untuk
// principal yang menurut basis data sudah punya.
func PrincipalKey(clientID, name string) string {
	return strings.ToUpper(strings.TrimSpace(clientID)) + "\x00" + strings.ToUpper(strings.TrimSpace(name))
}

// tooLong menghitung dalam rune, bukan byte: satu huruf beraksen memakan dua byte dan
// akan membuat batas terasa berubah-ubah bagi pengguna.
func tooLong(value string, limit int) bool {
	return utf8.RuneCountInString(value) > limit
}

func lengthMessage(label string, limit int) string {
	return label + " paling panjang " + strconv.Itoa(limit) + " karakter."
}

// ErrAmountEmpty dan ErrAmountInvalid dikembalikan ParseAmount. Keduanya dideklarasikan
// di domain supaya transport dapat membedakan "isian tidak diisi" dari "isian bukan
// angka" tanpa membaca teks pesan.
var (
	ErrAmountEmpty   = errors.New("masterrecovery: nilai uang kosong")
	ErrAmountInvalid = errors.New("masterrecovery: nilai uang bukan rupiah utuh")
)
