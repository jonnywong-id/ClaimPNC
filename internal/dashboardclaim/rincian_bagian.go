package dashboardclaim

import "time"

// Bagian-bagian isi popup rincian klaim, dibaca dari TABEL RELASIONAL.
//
// # Kenapa bertipe, bukan peta bebas
//
// Sampai 2026-10-12 seluruh isi popup berupa `map[string]any` hasil urai dokumen JSON, dan
// alasannya tercatat: bentuk dokumen itu belum pernah diperiksa, sehingga struct akan
// menyatakan bentuk yang belum terbukti.
//
// Alasan itu **tidak berlaku lagi**. Sumbernya kini tabel yang kolomnya dibaca dari katalog
// satu per satu, dan kunci penautnya diukur. Bentuknya diketahui, jadi dinyatakan — peta
// bebas sekarang hanya akan menyembunyikan salah ketik nama kolom sampai layar tampil kosong.
//
// # Kosong bukan galat
//
// Setiap bagian boleh kosong. Klaim yang belum punya objek, belum punya penerima, atau belum
// punya lampiran tetap dapat dibuka — hanya bagiannya yang kosong. Diukur 2026-10-12: dari
// 2.758 klaim, yang punya adjustment 1.752, objek 1.434, penerima 1.352, lampiran 779.

// Receiver adalah satu penerima pembayaran klaim — `POOLDATA.T_CLAIM_RECEIVER`.
//
// Nomor rekening dan nama bank ADA di tabel tetapi TIDAK dibawa: grid layar lama hanya
// menampilkan Nama dan Alamat, dan nomor rekening adalah data nasabah (`D-69`).
type Receiver struct {
	Name    string
	Address string

	// Dibaca untuk sub-popup "Detail Penerima", bukan untuk grid.
	BankName      string
	AccountNumber string
}

// InsuredItem adalah satu objek pertanggungan — `POOLDATA.T_CLAIM_OBJECTLIST`.
//
// Kedelapan isiannya mengikuti grid layar lama. Isian kendaraan dan isian orang berada pada
// grid yang SAMA di sana, sehingga yang tidak terpakai pada sebuah lini bisnis tampil
// kosong — itu perilaku yang diwarisi, bukan kekurangan data.
type InsuredItem struct {
	Name              string
	Location          string
	ParticipantStatus string
	IDCard            string
	DateOfBirth       *time.Time
	Job               string
	VehicleType       string
	ChassisNumber     string
}

// Coverage adalah satu jaminan pada objek — `POOLDATA.T_CLAIM_OBJECTCOVERAGE`.
//
// `PlanCode` memuat KODE, bukan nama. Grid layar lama memberinya judul "Nama Plan" sementara
// yang ditampilkannya `COVERAGEID`; itu diwarisi apa adanya (`P-5`), dan namanya di sini
// dibuat jujur supaya yang membaca kode tidak ikut tertipu (`D-19`).
type Coverage struct {
	Name        string
	PlanCode    string
	SumInsured  *float64
	Currency    string
	CauseOfLoss string
	ItemName    string
}

// SettlementLine adalah satu baris Adjustment & Akseptasi — `POOLDATA.T_CLAIM_ADJUSTMENT`.
//
// Nilai uang bertipe pecahan dan dibawa PRESISI PENUH. `D-51` menetapkan pembulatan hanya
// terjadi saat ditampilkan; membulatkannya di sini memindahkan keputusan itu ke tempat yang
// tidak terlihat pembacanya.
type SettlementLine struct {
	AcceptanceNumber string
	AcceptedAt       *time.Time
	AcceptanceStatus string
	PaymentType      string
	ReceiverName     string
	Currency         string
	AcceptedValue    *float64
	ProposedValue    *float64
	TotalClaim       *float64
	TransferredAt    *time.Time
	PaidStatus       string
}

// SurveyStep adalah satu LANGKAH berkas survei — `POOLDATA.T_SURVEYORLIST`.
//
// Seluruh langkah dibawa, bukan langkah terakhir saja. Berbeda dari tile Loss Adjuster yang
// menampilkan satu baris per berkas: di popup, riwayatnya yang ditampilkan.
type SurveyStep struct {
	SurveyedAt *time.Time
	Surveyor   string
	ItemName   string
	Location   string
	Status     string
	StepIndex  string

	// Empat berikut dibaca untuk SUB-POPUP yang membuka baris ini, bukan untuk gridnya.
	//
	// SurveyLocation berbeda dari Location: yang pertama lokasi survei (LOCATION_SURVEY),
	// yang kedua lokasi penjadwalan ulang (RESCHEDULE_LOCATION). Grid menggambar yang kedua
	// karena layar lama begitu; sub-popup menggambar yang pertama.
	ReferenceNumber string
	Note            string
	SurveyLocation  string
	LossType        string
}

// Attachment adalah satu baris riwayat lampiran — `POOLDATA.DATA_ATTACHFILE`.
//
// Isi berkas dan rujukan storage TIDAK dibawa. Popup hanya menggambar daftar, dan mengirim
// isi berkas untuk menggambar beberapa baris memindahkan megabita tanpa satu pun manfaat.
type Attachment struct {
	Category string
	Name     string
	AddedAt  *time.Time
	AddedBy  string
}

// ProgressStep adalah satu tahapan progres klaim — `POOLDATA.GCNM_PROGRESS_CLAIM`.
//
// KUNCINYA NOMOR KLAIM, bukan kunci Pega — lihat `rincian_bagian.sql`.
//
// `StageName` datang dari master lewat LEFT JOIN dan BOLEH kosong: baris progres yang kodenya
// belum terdaftar di master tetap ditampilkan. Membuangnya akan menyembunyikan progres yang
// benar-benar terjadi hanya karena masternya belum lengkap.
type ProgressStep struct {
	RecordedAt *time.Time
	StageCode  string
	Status     string
	Note       string
	RecordedBy string
	StageName  string

	// Dibaca untuk sub-popup "Detail Progress".
	NextFollowUp string
}

// RegistrationFields adalah isian tab "Registrasi" pada popup rincian.
//
// # Kenapa ada di sini, bukan dibaca ulang dari modul registrasi
//
// Kolom-kolomnya SAMA PERSIS dengan yang dibaca `registrasi` pada `klaim_ambil`. Yang berbeda
// hanya BENTUKNYA: di sana ia bagian dari formulir kerja yang dapat disunting, di sini ia
// nilai baca-saja untuk ditampilkan.
//
// Memanggil modul registrasi dari sini akan menautkan dua modul domain yang seharusnya tidak
// saling tahu, dan membawa serta seluruh model formulirnya. Yang dijaga agar keduanya tidak
// berbeda adalah NAMA KOLOMNYA — bukan berbagi kode.
//
// # Tiga isian yang tidak ada di sini
//
// "Klaim Estimasi" dan "Nilai Dalam IDR" hidup di `POOLDATA.T_CLAIM_ESTIMASI` — per objek dan
// per coverage, bukan satu nilai per klaim. Menjumlahkannya menjadi satu angka adalah
// keputusan bisnis yang belum diambil, dan angka uang yang salah jumlah lebih buruk daripada
// angka yang tidak ditampilkan.
//
// "Catatan dari Analyst" ada sebagai PERCAKAPAN di `POOLDATA.M_KOMUNIKASI_PNC`, bukan sebagai
// satu nilai. Menampilkan satu pesan terakhir sebagai "catatan" akan menyembunyikan sisanya.
type RegistrationFields struct {
	DocumentReceivedAt  *time.Time
	LossAt              *time.Time
	HospitalDischarge   *time.Time
	ReportedAt          *time.Time
	ReporterPhone       string
	ReporterEmail       string
	ReporterRelation    string
	ReporterRelationEtc string
	LossLocation        string
	Village             string
	InsuredIsTKI        string
	NoteToAnalyst       string
	NoteToTechnicalPIC  string
}

// SurveyRequestFields adalah isian tab "Hasil Survey" bagian atas — permintaan surveinya,
// bukan hasilnya. Hasil survei adalah grid tersendiri (`SurveyStep`).
//
// Kedelapan field di bawah berpadanan SATU LAWAN SATU dengan kolom `POOLDATA.T_REQ_SURVEY`.
// Kecocokan 8-dari-8 itulah yang membuktikan tabel itu rumahnya; tabel lain yang namanya mirip
// (`T_REQSURVEY`, tanpa garis bawah) hanya punya sebagian.
type SurveyRequestFields struct {
	RequestorName string
	Location      string
	Phone         string
	RequestedAt   *time.Time
	Branch        string
	SurveyorName  string
	SurveyorEmail string
	ItemName      string
}

// PUCLFields adalah dua isian tab "Catatan Untuk Analis" yang punya kolom.
type PUCLFields struct {
	DocumentCompleteAt *time.Time
	DoctorRejectReason string
}
