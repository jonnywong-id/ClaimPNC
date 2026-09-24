package komitehttp

// Bentuk respons layar Inbox Komite.
//
// Seluruh tag `json` berbahasa Indonesia dan tetap demikian: ia **kontrak API**, bukan
// nama internal (`D-80`). Mengubahnya adalah perubahan yang merusak klien.

// CommitteeCaseDTO adalah satu baris Inbox Komite.
//
// # Nama field di sini SUDAH benar, berbeda dari property Pega yang digantikannya
//
// Layar lama menampilkan kolom yang sama lewat property yang namanya tidak mencerminkan
// isinya sama sekali — `.IBNR` untuk Nilai ASM Share, `.pyScore` untuk Nilai OR ASM,
// `.DraftWordingID` untuk PIC Klaim, `.StatusKlaim` untuk Tipe Komite. Kontrak ini
// memutus warisan itu (`D-19`); pemetaan ke kolom aslinya hanya ada di repo/sqlstore.
type CommitteeCaseDTO struct {
	CaseID      string `json:"nomor_case"`
	ClaimNumber string `json:"nomor_klaim"`

	PolicyNumber     string `json:"nomor_polis"`
	InsuredName      string `json:"nama_tertanggung"`
	BusinessName     string `json:"nama_bisnis"`
	SourceOfBusiness string `json:"sumber_bisnis"`
	BranchName       string `json:"cabang"`
	GroupPanel       string `json:"group_panel,omitempty"`
	ClaimPIC         string `json:"pic_klaim,omitempty"`

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

	WorkStatus    string `json:"status_kerja,omitempty"`
	CommitteeKind string `json:"tipe_komite,omitempty"`

	// Ketiga nilai uang dikirim sebagai TEKS desimal kanonik — "45000000.00" — bukan
	// angka JSON. Angka JSON adalah floating point ganda di hampir seluruh peramban, dan
	// mengirim nilai uang lewatnya berarti menyerahkan ketepatannya kepada pembulatan
	// biner. `I-12` menetapkan nilai uang disimpan presisi penuh; kontrak ini menjaganya
	// sampai ke layar.
	ClaimValue    string `json:"nilai_klaim"`
	ASMShareValue string `json:"nilai_asm_share"`
	ORValue       string `json:"nilai_or_asm"`

	CommitteeNote string `json:"note_komite,omitempty"`

	// Penilaian AI. HasAIAssessment membedakan "belum dinilai" dari "dinilai dengan
	// hasil kosong" — dua keadaan yang tidak boleh terbaca sama, karena kasus tanpa
	// penilaian AI tetap wajib dikerjakan komite.
	HasAIAssessment bool   `json:"ada_penilaian_ai"`
	AIResult        string `json:"jawaban_ai,omitempty"`
	AINoteAccepted  string `json:"note_ai_diterima,omitempty"`
	AINoteRejected  string `json:"note_ai_ditolak,omitempty"`
	AIAssessedAt    string `json:"tanggal_ai,omitempty"`

	// LegacyOutcome adalah keputusan yang tercatat DI PEGA, dan LegacyTier jenjang yang
	// tercatat di sana.
	//
	// Keduanya dikirim berdampingan dengan Progress — bukan menggantikannya — karena
	// selama masa paralel keduanya dapat berbeda, dan perbedaan itu harus TERLIHAT.
	LegacyOutcome string `json:"keputusan_pega,omitempty"`
	LegacyTier    int    `json:"jenjang_pega,omitempty"`

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
}

// CommitteeCaseResponse adalah jawaban satu kasus — detail maupun sesudah keputusan.
type CommitteeCaseResponse struct {
	Case CommitteeCaseDTO `json:"kasus"`
	Now  string           `json:"sekarang"`
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
