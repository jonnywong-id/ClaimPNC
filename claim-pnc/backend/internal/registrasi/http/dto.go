// Package registrasihttp adalah lapisan transport modul registrasi.
//
// Nama paketnya berbeda dari nama foldernya, mengikuti pola auth/http dan portal/http:
// foldernya `http` supaya letaknya seragam antarmodul, nama paketnya `registrasihttp`
// supaya tidak menutupi `net/http`.
package registrasihttp

// # Dua kesepakatan bentuk data yang mengikat seluruh berkas ini
//
//  1. UANG dikirim sebagai bilangan bulat SEN, tidak pernah sebagai pecahan. Nama
//     fieldnya selalu berakhiran `_sen` supaya satuannya tidak dapat salah dibaca.
//     Pecahan biner tidak dapat mewakili rupiah dengan tepat, dan `ADR-0016` menuntut
//     presisi penuh.
//  2. TANGGAL dikirim sebagai `YYYY-MM-DD` dan dibaca sebagai tanggal kalender WIB.
//     Tidak ada jam, tidak ada offset, tidak ada yang perlu ditafsirkan ulang di
//     salah satu ujung.

// TaskDTO adalah satu pekerjaan yang menunggu dikerjakan.
type TaskDTO struct {
	ID          string `json:"id"`
	ClaimID     string `json:"klaim_id"`
	ClaimNumber string `json:"nomor_klaim"`

	Stage     string `json:"tahap"`
	StageName string `json:"nama_tahap"`

	// Queue bernilai "WORKLIST" atau "WORKBASKET".
	Queue      string `json:"antrean"`
	Workbasket string `json:"workbasket"`

	Owner string `json:"pemilik"`

	// Claimable menyatakan tugas ini masih menunggu seseorang mengambilnya. Layar
	// memakainya untuk memilih tombol; kewenangannya tetap diperiksa server.
	Claimable bool `json:"dapat_diambil"`

	// ExitAction adalah nama tindakan yang menutup tahap ini.
	ExitAction string `json:"tindakan_keluar"`

	CreatedAt string `json:"dibuat_pada"`
}

// SpreadingDTO adalah satu baris pembagian risiko.
type SpreadingDTO struct {
	TreatyKind string `json:"jenis_treaty"`
	Name       string `json:"nama"`

	// Share adalah persentase dikali 10.000 — 100% dikirim sebagai 1000000. Empat
	// desimal adalah presisi yang `ADR-0016` pakai untuk memvalidasi totalnya.
	Share Percent `json:"share"`

	Removed      bool   `json:"dihapus"`
	FacOfferItem string `json:"objek_fac_offer"`
}

// Percent adalah persentase dikali 10.000.
type Percent int64

// CoverageDTO adalah satu jaminan pada sebuah objek.
type CoverageDTO struct {
	ID          string         `json:"id"`
	CauseOfLoss string         `json:"penyebab_kerugian"`
	TSICents    int64          `json:"tsi_sen"`
	Spreading   []SpreadingDTO `json:"spreading"`
}

// InsuredItemDTO adalah satu objek pertanggungan.
type InsuredItemDTO struct {
	ID       string        `json:"id"`
	Name     string        `json:"nama"`
	Location string        `json:"lokasi"`
	Coverage []CoverageDTO `json:"coverage"`
}

// ReporterDTO adalah orang yang melaporkan kejadian.
type ReporterDTO struct {
	Name          string `json:"nama"`
	Phone         string `json:"telepon"`
	Email         string `json:"email"`
	Address       string `json:"alamat"`
	Relation      int    `json:"hubungan"`
	OtherRelation string `json:"hubungan_lainnya"`
}

// PolicyDTO adalah bagian polis yang ditampilkan layar registrasi.
type PolicyDTO struct {
	Number          string `json:"nomor"`
	Line            string `json:"lini"`
	LineName        string `json:"nama_lini"`
	BusinessType    string `json:"jenis_bisnis"`
	CoverageStart   string `json:"mulai_pertanggungan"`
	CoverageEnd     string `json:"akhir_pertanggungan"`
	Currency        string `json:"mata_uang"`
	InsuredName     string `json:"nama_tertanggung"`
	Declaration     bool   `json:"deklarasi"`
	CreditGuarantee bool   `json:"penjamin_kredit"`
}

// ClaimDTO adalah klaim sebagaimana dilihat layar.
type ClaimDTO struct {
	ID     string    `json:"id"`
	Number string    `json:"nomor"`
	Portal string    `json:"portal"`
	Policy PolicyDTO `json:"polis"`

	DateOfLoss   string `json:"tanggal_kejadian"`
	ReportDate   string `json:"tanggal_lapor"`
	DateReceived string `json:"tanggal_terima_dokumen"`

	Location   string      `json:"lokasi"`
	Chronology string      `json:"kronologi"`
	Reporter   ReporterDTO `json:"pelapor"`

	EstimateValueCents int64  `json:"nilai_estimasi_sen"`
	Currency           string `json:"mata_uang"`
	SLIKNumber         string `json:"nomor_slik"`
	ExGratia           bool   `json:"ex_gratia"`
	TechnicalPIC       string `json:"user_teknis"`
	RCVID              string `json:"rcv_id"`

	InsuredItem []InsuredItemDTO `json:"objek"`

	PUCLStatus         int  `json:"status_pucl"`
	ComplianceTransfer bool `json:"transfer_compliance"`

	// Keempat status di bawah sengaja dikirim terpisah dan dinamai berbeda. `ADR-0018`
	// menetapkan keempatnya konsep yang berbeda, dan yang membuatnya sering tertukar di
	// sistem lama adalah namanya — bukan jumlahnya.
	ProcessStatus          string `json:"status_proses"`
	ClaimStatus            string `json:"status_klaim"`
	ClaimFlag              string `json:"flag_klaim"`
	ProgressPositionStatus string `json:"status_posisi_progres"`

	CurrentStage string `json:"tahap_kini"`
}

// StageDTO adalah satu tahap alur.
type StageDTO struct {
	ID         string `json:"id"`
	Name       string `json:"nama"`
	Queue      string `json:"antrean"`
	Workbasket string `json:"workbasket"`
	Router     string `json:"router"`

	ExitAction string `json:"tindakan_keluar"`

	// PegaID adalah pengenal shape pada Flow/Register_Flow.xml. Ia dikirim ke layar
	// hanya untuk penelusuran saat uji kesetaraan; layar tidak menampilkannya.
	PegaID string `json:"pega_id"`
}

// FlowResponse adalah jawaban GET /api/registrasi/alur.
type FlowResponse struct {
	Name  string     `json:"nama"`
	Start string     `json:"mulai"`
	Stage []StageDTO `json:"tahap"`
}

// InboxResponse adalah jawaban GET /api/registrasi/inbox.
type InboxResponse struct {
	Task []TaskDTO `json:"tugas"`
}

// StartRequest adalah badan POST /api/registrasi/klaim.
type StartRequest struct {
	PolicyNumber string `json:"nomor_polis"`
	Portal       string `json:"portal"`
}

// RegisterRequest adalah badan POST /api/registrasi/register.
type RegisterRequest struct {
	TaskID string `json:"tugas_id"`

	DateOfLoss   string `json:"tanggal_kejadian"`
	ReportDate   string `json:"tanggal_lapor"`
	DateReceived string `json:"tanggal_terima_dokumen"`

	Location   string      `json:"lokasi"`
	Chronology string      `json:"kronologi"`
	Reporter   ReporterDTO `json:"pelapor"`

	EstimateValueCents int64  `json:"nilai_estimasi_sen"`
	Currency           string `json:"mata_uang"`
	SLIKNumber         string `json:"nomor_slik"`
	ExGratia           bool   `json:"ex_gratia"`
	TechnicalPIC       string `json:"user_teknis"`
	RCVID              string `json:"rcv_id"`

	InsuredItem []InsuredItemDTO `json:"objek"`

	PUCLStatus         int  `json:"status_pucl"`
	ComplianceTransfer bool `json:"transfer_compliance"`

	// Return menandai tombol Back, bukan Simpan.
	Return bool `json:"kembali"`
}

// CompleteRequest adalah badan POST /api/registrasi/tugas/{id}/selesai.
type CompleteRequest struct {
	Action string `json:"tindakan"`
	Return bool   `json:"kembali"`
}

// ClaimResponse adalah klaim beserta keadaan alurnya.
type ClaimResponse struct {
	Claim ClaimDTO `json:"klaim"`

	// Task bernilai nil bila klaim sudah selesai.
	Task *TaskDTO `json:"tugas"`

	// Path adalah rangkaian pengenal tahap yang akan dilalui klaim dari tahap
	// sekarang. Ia gambaran menurut data yang berlaku sekarang, bukan janji.
	Path []string `json:"jalur"`

	// DecisionTrace menyebut keputusan alur yang baru saja dilewati beserta cabang
	// yang dipilih, misalnya "IsPA → isPA_PNC". Ia terisi hanya pada respons yang
	// memindahkan klaim.
	DecisionTrace []string `json:"jejak_keputusan"`

	// LargeLoss menyatakan Notice of Large Losses ikut diterbitkan.
	LargeLoss bool `json:"large_loss"`
}

// ViolationDTO adalah satu aturan validasi yang dilanggar.
type ViolationDTO struct {
	Code    string `json:"kode"`
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul auth — `{kode, pesan}` — ditambah satu field opsional
// untuk rincian validasi. Klien membedakan jenis galat lewat `kode`, tidak pernah dengan
// mencocokkan teks `pesan`.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`

	// Kuncinya `detail`, sama dengan masterstatus dan masterstatusprogres, karena
	// itulah satu-satunya kunci yang dibaca APIError di sisi klien. Sebelum
	// penyelarasan ini modul ini mengirim `pelanggaran` sementara layar membaca
	// `violations`, sehingga tidak ada satu pun pesan validasi yang sampai ke layar.
	Violation []ViolationDTO `json:"detail,omitempty"`
}
