package komitehttp

// Bentuk respons layar Inbox Komite.
//
// Seluruh tag `json` berbahasa Indonesia dan tetap demikian: ia **kontrak API**, bukan
// nama internal (`D-80`). Mengubahnya adalah perubahan yang merusak klien.

// CommitteeCaseDTO adalah satu baris Inbox Komite.
//
// # Kesembilan kolomnya diambil dari InboxRegisterKomite_RD, bukan dikarang
//
// Work Owner menetapkan 2026-09-28 bahwa data komite dimunculkan `InboxRegisterKomite_RD`
// dan `SetDataKomitePNC_Act`. Grid pada `Section/InboxKomite_section` menampilkan tepat
// kolom-kolom di bawah:
//
//	Nomor Case     .pyID
//	Nomor Klaim    .CoverID
//	Tgl Komite     .Komite.DateOfComitee
//	Nomor Polis    .POLICYNO
//	Nama Bisnis    .BUSINESSNAME
//	Cabang         .BranchName
//	Sumber Bisnis  .SOBNAME
//	Tertanggung    .QQNAME
//	Aging Komite   pxDifferenceInDays(.pxCreateDateTime, sekarang)
//
// # Yang SENGAJA tidak ada di sini
//
// Nilai klaim, Nilai ASM Share, Nilai OR ASM, Tipe Komite, PIC Klaim, dan penilaian AI.
// Tidak satu pun ada di RD; yang menampilkannya di sistem lama adalah
// `ShowKomiteTerimaTolakNonMBU` — jalur **Non-MBU**, yang `SetDataKomitePNC_Act` serahkan
// ke `SetDataKomiteNonMBU_Act` dan yang bukan sumber layar ini.
//
// Menghapusnya bukan pemangkasan fitur melainkan koreksi: menampilkan nilai uang yang
// tidak pernah ada di layar aslinya berarti menambah angka yang tidak dapat diuji
// kesetaraannya terhadap apa pun.
type CommitteeCaseDTO struct {
	CaseID      string `json:"nomor_case"`
	ClaimNumber string `json:"nomor_klaim"`

	PolicyNumber     string `json:"nomor_polis"`
	InsuredName      string `json:"nama_tertanggung"`
	BusinessName     string `json:"nama_bisnis"`
	SourceOfBusiness string `json:"sumber_bisnis"`
	BranchName       string `json:"cabang"`

	// CommitteeDate dan CreatedAt dikirim sebagai RFC 3339 dalam UTC.
	//
	// Pemformatan untuk manusia dikerjakan di layar, bukan di SQL maupun di sini —
	// 411 pemakaian `TO_CHAR` pada sistem lama mengembalikan tanggal sebagai teks
	// `dd/mm/yyyy`, sehingga pengurutannya menjadi pengurutan TEKS dan penyaringan
	// rentang tanggal berhenti dapat memakai index (`docs/Steering/09-DATABASE-STRATEGY.md`
	// §3.2).
	CommitteeDate string `json:"tanggal_komite,omitempty"`
	CreatedAt     string `json:"tanggal_input,omitempty"`

	// AgingDays dihitung SERVER, bukan diserahkan ke layar.
	//
	// Menghitungnya di peramban berarti memakai jam peramban, dan jam peramban dapat
	// berbeda dari jam server. Satu kenyataan tidak boleh punya dua umur.
	AgingDays int `json:"aging_komite"`

	WorkStatus string `json:"status_kerja,omitempty"`

	// LegacyOutcome adalah keputusan yang tercatat DI PEGA, diturunkan dari
	// `T_CLAIM_KOMITE_LIST.STATUSAPPROVE` persis seperti `GetKomitePAditerima`.
	//
	// Ia bukan sekadar keterangan: inilah yang menentukan isi kotak Diterima dan Ditolak.
	LegacyOutcome string `json:"keputusan_pega,omitempty"`

	Progress ProgressDTO `json:"penjenjangan"`
}

// ProgressDTO adalah keadaan penjenjangan satu kasus menurut sistem ini.
type ProgressDTO struct {
	// Outcome bernilai "menunggu", "disetujui", "ditolak", atau "dikembalikan".
	Outcome string `json:"kesimpulan"`

	// TierCount nol berarti BELUM DIKETAHUI, bukan nol jenjang — lihat TierCountUnknown.
	TierCount     int `json:"jumlah_jenjang"`
	CurrentTier   int `json:"jenjang_kini"`
	ApprovedTiers int `json:"jenjang_disetujui"`

	// TierCountUnknown dikirim sebagai KESIMPULAN, bukan dibiarkan disimpulkan layar
	// dari TierCount yang bernilai nol.
	//
	// Aturannya milik server, dan menduplikasinya di layar membuat dua salinan yang
	// dapat berbeda pendapat. Yang dipertaruhkan bukan kosmetik: jumlah jenjang yang
	// tidak diketahui lalu digambar sebagai "1 dari 1" akan membuat persetujuan pertama
	// tampak menutup seluruh komite.
	TierCountUnknown bool `json:"jumlah_jenjang_belum_diketahui"`

	// Closed menyatakan kasus ini tidak menerima keputusan lagi.
	Closed bool `json:"selesai"`

	// DecidedByMe menyatakan pemanggil sudah memberi keputusan pada kasus ini.
	//
	// Layar memakainya untuk menyembunyikan tombol keputusan. Penyembunyian itu
	// KENYAMANAN TAMPILAN; penegakan yang sebenarnya ada di server, yang menolak
	// keputusan kedua dengan 409.
	DecidedByMe bool `json:"sudah_saya_putuskan"`

	Decisions []DecisionDTO `json:"keputusan"`
}

// DecisionDTO adalah satu keputusan komite yang tercatat.
type DecisionDTO struct {
	ID   string `json:"id"`
	Tier int    `json:"jenjang"`

	// Kind bernilai "setuju", "tolak", atau "kembalikan".
	Kind string `json:"keputusan"`
	Note string `json:"catatan,omitempty"`

	ActorLogin string `json:"oleh"`
	ActorName  string `json:"nama_pemutus,omitempty"`
	DecidedAt  string `json:"pada"`
}

// InboxSummaryDTO adalah jumlah baris per kotak, untuk lencana pada tab.
//
// Ia dihitung dalam perjalanan yang sama dengan isi tabelnya, sehingga angka lencana
// tidak dapat berselisih dengan isi tabel di bawahnya. Dan ia mengikuti pencarian yang
// sedang berlaku — tanpa itu, pengguna melihat angka pada tab lain, berpindah ke sana,
// lalu menemukan tabel kosong.
type InboxSummaryDTO struct {
	Outstanding int `json:"outstanding"`
	Accepted    int `json:"diterima"`
	Rejected    int `json:"ditolak"`
}

// InboxListResponse adalah jawaban GET /api/komite/inbox.
type InboxListResponse struct {
	Cases []CommitteeCaseDTO `json:"kasus"`

	// Total adalah banyaknya baris yang cocok dengan penyaring — bukan banyaknya baris
	// pada halaman ini. Ia dikirim supaya layar dapat memberi tahu bahwa masih ada yang
	// belum tampak, alih-alih memotong diam-diam seperti `pyMaxRecords=500` (`T-12`).
	Total int `json:"total"`

	Offset int `json:"lewati"`
	Limit  int `json:"batas"`

	Summary InboxSummaryDTO `json:"ringkasan"`

	// Kind adalah kotak yang benar-benar dipakai server.
	//
	// Ia dipantulkan kembali karena kotak yang tidak dikenali JATUH ke Outstanding
	// alih-alih ditolak; tanpa pantulan ini layar tidak dapat tahu bahwa permintaannya
	// diperlakukan berbeda dari yang ia minta.
	Kind string `json:"kotak"`

	// Operator adalah pemilik inbox yang benar-benar dipakai menyaring.
	//
	// Dipantulkan supaya inbox yang kosong dapat dibedakan sebabnya: tidak ada pekerjaan,
	// versus identitas sesi tidak cocok dengan satu pun OPERATOR_ID di data warisan —
	// keadaan yang sangat mungkin terjadi selama pemetaan identitas HCC/HCQ ke
	// OPERATOR_ID belum ada (`ADR-0024`).
	Operator string `json:"operator"`

	// Now adalah jam server yang dipakai menghitung Aging.
	Now string `json:"sekarang"`

	// DecisionsAvailable menyatakan apakah keputusan dapat dicatat saat ini.
	//
	// Salah berarti jejak keputusan belum dapat dipakai — `POOLDATA.CPNC_KOMITE_KEPUTUSAN`
	// dibuat migrasi `0004`, dan migrasi menempuh `D-63` sehingga hanya DBA yang dapat
	// menjalankannya.
	//
	// Dalam keadaan itu daftar di atas TETAP berisi pekerjaan yang sebenarnya, dibaca dari
	// tabel warisan; yang tidak tersedia hanyalah pencatatan keputusannya. Layar WAJIB
	// menyatakannya kepada pengguna sebelum tombol ditekan — bukan membiarkannya
	// ditemukan sebagai kegagalan setelah keputusan diambil.
	//
	// Tanpa `omitempty`: penandanya harus selalu ada. Nilai `false` yang hilang dari
	// respons tidak dapat dibedakan dari versi server lama oleh klien mana pun.
	DecisionsAvailable bool `json:"jejak_keputusan_tersedia"`

	// OwnerFilterActive menyatakan daftar di atas benar-benar milik `Operator`.
	//
	// `false` berarti penyaring pemilik sedang DIMATIKAN, dan daftarnya adalah SELURUH
	// antrean komite — termasuk pekerjaan orang lain beserta nama tertanggung dan nomor
	// polisnya. Keadaan itu hanya mungkin di `APP_ENV=development`; konfigurasi menolak
	// menyalakannya di luar sana.
	//
	// Layar WAJIB menyatakannya. Daftar pekerjaan orang lain yang tampak seperti daftar
	// pekerjaan sendiri adalah kekeliruan yang tidak terlihat sebagai kekeliruan — dan
	// pada layar yang menyetujui uang, itu kelas kesalahan yang paling mahal.
	//
	// Tanpa `omitempty`, dengan alasan yang sama seperti penanda di atas.
	OwnerFilterActive bool `json:"penyaring_pemilik_aktif"`
}

// CommitteeCaseResponse adalah jawaban satu kasus — detail maupun sesudah keputusan.
type CommitteeCaseResponse struct {
	Case CommitteeCaseDTO `json:"kasus"`
	Now  string           `json:"sekarang"`

	// Transfer null pada jawaban yang BUKAN layar rincian.
	//
	// Ia hanya diisi `GET /api/komite/inbox/{nomor}`. Jawaban lain yang memakai bentuk ini
	// tidak membacanya, dan mengirim objek kosong di sana akan menyiratkan rincian yang
	// memang tidak diminta.
	Transfer *TransferDetailDTO `json:"transfer,omitempty"`
}

// DecisionRequest adalah badan POST /api/komite/inbox/{nomor}/keputusan.
//
// # Yang TIDAK ada di sini, dan itu disengaja
//
// Tidak ada jenjang, tidak ada waktu, dan tidak ada identitas pemutus. Ketiganya milik
// server: klien yang boleh menyebut jenjangnya dapat menyetujui jenjang yang bukan
// gilirannya, klien yang boleh menyebut waktunya dapat mencatat keputusan bertanggal
// kemarin, dan klien yang boleh menyebut pemutusnya dapat menyetujui atas nama orang lain.
type DecisionRequest struct {
	Decision string `json:"keputusan"`
	Note     string `json:"catatan"`
}

// TransferDetailDTO adalah isi "Lihat Detail Transfer" — pengganti `ShowTransferDetail`.
//
// # Kenapa ada di sini padahal daftarnya sengaja tanpa nilai uang
//
// Bukan pertentangan, melainkan dua layar yang berbeda. `InboxRegisterKomite_RD` — sumber
// DAFTAR — memang tidak memuat satu pun nilai uang, dan `§69` mengikutinya. Yang memuatnya
// adalah `ShowTransferDetail`, yang hanya tergambar setelah sebuah case ditekan.
//
// Nilai uangnya karena itu muncul di tempat layar lama menempatkannya: pada rincian, bukan
// pada daftar.
type TransferDetailDTO struct {
	// Judul adalah judul layar yang SUDAH JADI, mis. "CLAIM COMMITTEE - ADJUSTMENT".
	//
	// Ia dikirim jadi, bukan dirakit layar dari kode mentah, karena aturannya adalah enam
	// syarat Pega — termasuk `IsTravel` yang menyembunyikan "- ADJUSTMENT" pada Group
	// Panel 005. Menaruh aturan itu di React berarti ia hidup di dua tempat begitu ada
	// klien kedua, dan itu persis kegagalan yang dihindari `D-19`.
	Judul string `json:"judul"`

	// HEDapatDinilai menyatakan apakah cabang `IsHE` — yang memunculkan
	// `ShowTransferDetailHE` — dapat dinilai sama sekali.
	//
	// `BUSINESSTYPE` yang dibandingkan `IsHE` terisi 0 dari 610 case komite. Selama ia
	// kosong, layar TIDAK BOLEH menyatakan "case ini bukan HE" — yang benar adalah "hal
	// itu tidak dapat diketahui dari sini". Keduanya berbeda, dan hanya satu yang jujur.
	HEDapatDinilai bool `json:"he_dapat_dinilai"`

	Lines []AdjustmentLineDTO `json:"baris"`

	// Claim null bila klaimnya tidak terbaca. Terbaca pada 189 dari 189 case inbox, jadi
	// null di sini menandakan keadaan yang layak diperiksa, bukan keadaan biasa.
	Claim *ClaimSummaryDTO `json:"klaim"`

	// Coverages adalah blok analisis komite — 1 sampai 3 baris per case.
	Coverages []CoverageAnalysisDTO `json:"coverage"`

	// MoneyEmpty dipisahkan dari Empty sejak blok klaim masuk.
	//
	// Sebuah case dapat punya data klaim lengkap tanpa satu pun baris adjustment — 148
	// dari 189 berada dalam keadaan itu. Menyamakan keduanya akan menyembunyikan seluruh
	// layar hanya karena angkanya belum ada.
	MoneyEmpty bool `json:"nilai_uang_kosong"`

	// Committee null bila case ini belum punya baris di `T_CLAIM_KOMITE_LIST`.
	//
	// `null`, bukan objek kosong: "belum ada keputusan" dan "ada keputusan yang isinya
	// kosong" adalah dua keadaan yang berbeda, dan layar menggambarnya berbeda.
	Committee *CommitteeRecordDTO `json:"komite"`

	// Empty menyatakan tidak ada satu pun rincian yang dapat ditampilkan.
	//
	// Dikirim sebagai KESIMPULAN, bukan dibiarkan disimpulkan layar dari senarai yang
	// kosong — supaya aturannya hidup di satu tempat.
	Empty bool `json:"kosong"`
}

// AdjustmentLineDTO adalah satu baris `POOLDATA.T_CLAIM_ADJUSTMENT`.
type AdjustmentLineDTO struct {
	ClaimNumber string `json:"nomor_klaim"`
	ObjectID    string `json:"id_objek,omitempty"`
	CoverageID  string `json:"id_coverage,omitempty"`

	AcceptanceNo string `json:"nomor_akseptasi,omitempty"`
	AcceptedAt   string `json:"tanggal_akseptasi,omitempty"`

	Currency    string `json:"mata_uang,omitempty"`
	PaymentType string `json:"jenis_pembayaran,omitempty"`

	// Keenam nilai uang dikirim sebagai TEKS desimal kanonik, bukan angka JSON. Angka JSON
	// adalah floating point ganda di hampir seluruh peramban, dan mengirim nilai uang
	// lewatnya berarti menyerahkan ketepatannya kepada pembulatan biner (`I-12`).
	GrossValue     string `json:"nilai_gross"`
	ProposeValue   string `json:"nilai_usulan"`
	AcceptedValue  string `json:"nilai_akseptasi"`
	SalvageValue   string `json:"nilai_salvage"`
	ASMShareValue  string `json:"nilai_asm_share"`
	IndividualRisk string `json:"nilai_risiko_sendiri"`

	// ASMSharePercent persentase, bukan nilai uang — karena itu tidak ikut aturan di atas.
	ASMSharePercent string `json:"persen_asm_share,omitempty"`

	ExGratia    bool   `json:"ex_gratia"`
	Notes       string `json:"catatan,omitempty"`
	CauseOfLoss string `json:"sebab_kerugian,omitempty"`
}

// CommitteeRecordDTO adalah keputusan komite MENURUT PEGA.
type CommitteeRecordDTO struct {
	MemberName string `json:"nama_komite,omitempty"`
	Tier       int    `json:"jenjang,omitempty"`
	Kind       string `json:"tipe_komite,omitempty"`
	Note       string `json:"catatan,omitempty"`

	ClaimValue string `json:"nilai_klaim"`
	ASMShare   string `json:"persen_asm_share,omitempty"`

	DecidedAt string `json:"tanggal_komite,omitempty"`
	Outcome   string `json:"kesimpulan,omitempty"`
}

// ClaimSummaryDTO adalah blok `.KomiteClaimData.*` pada `ShowTransferDetail`.
type ClaimSummaryDTO struct {
	DateOfLoss   string `json:"tanggal_kejadian,omitempty"`
	RegisterDate string `json:"tanggal_register,omitempty"`

	Location    string `json:"lokasi,omitempty"`
	Chronology  string `json:"kronologi,omitempty"`
	ClaimStatus string `json:"status_klaim,omitempty"`

	Recommendation string `json:"rekomendasi,omitempty"`

	ASMShare  string `json:"persen_asm_share,omitempty"`
	CoinsName string `json:"koasuransi,omitempty"`
	Currency  string `json:"mata_uang,omitempty"`
	ExGratia  string `json:"ex_gratia,omitempty"`
}

// CoverageAnalysisDTO adalah satu baris blok analisis komite.
//
// `analisis_terisi` dikirim sebagai KESIMPULAN dari server, bukan disimpulkan layar dari
// enam medan yang mungkin kosong: aturannya satu, dan tempatnya satu.
type CoverageAnalysisDTO struct {
	ObjectID   string `json:"id_objek,omitempty"`
	CoverageID string `json:"id_coverage,omitempty"`

	ObjectName   string `json:"nama_objek,omitempty"`
	CoverageName string `json:"nama_coverage,omitempty"`
	CauseOfLoss  string `json:"sebab_kerugian,omitempty"`

	SumInsured string `json:"nilai_tsi"`
	Currency   string `json:"mata_uang,omitempty"`

	Circumstances  string `json:"keadaan_kerugian,omitempty"`
	ExtentOfLoss   string `json:"luas_kerugian,omitempty"`
	LegalLiability string `json:"tanggung_jawab_hukum,omitempty"`
	Remarks        string `json:"catatan,omitempty"`
	Diagnose       string `json:"diagnosa,omitempty"`
	InitialName    string `json:"nama_initial,omitempty"`

	CommitteeDate string `json:"tanggal_komite,omitempty"`

	AnalysisFilled bool `json:"analisis_terisi"`
}
