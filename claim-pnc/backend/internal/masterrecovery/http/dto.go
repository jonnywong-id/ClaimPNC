// Package masterrecoveryhttp adalah lapisan transport modul Master Recovery: bentuk
// permintaan dan respons, pemetaan galat, handler, dan pendaftaran rutenya.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola authhttp dan
// portalhttp: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `masterrecoveryhttp` supaya tidak menutupi `net/http`.
//
// Lapisan ini menerjemahkan HTTP menjadi panggilan usecase dan menerjemahkan hasilnya
// kembali. **Tidak ada aturan modul di sini.**
package masterrecoveryhttp

// Nilai uang dikirim sebagai ANGKA JSON, bukan teks.
//
// Keduanya sama-sama masuk akal, dan yang dipilih adalah yang paling sulit disalahgunakan
// di sisi klien: angka JSON di JavaScript adalah float64, yang menyimpan bilangan bulat
// dengan tepat sampai sekitar sembilan ribu triliun — jauh di atas nilai klaim mana pun.
// Teks akan menuntut setiap layar mengurai sendiri, dan satu layar yang lupa akan
// mengirim "1.000.000" sebagai satu.
//
// Yang TIDAK dikirim sebagai angka adalah nomor: nomor virtual account, nomor polis, dan
// Client ID seluruhnya teks, karena nol di depannya bermakna.

// PrincipalDTO adalah satu pilihan pada isian "Nama Principal".
//
// Tipe ini sengaja TERPISAH dari masterrecovery.Principal. Memakai tipe modul langsung
// sebagai bentuk JSON membuat perubahan internal bocor ke klien dan sebaliknya
// (`docs/Steering/08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field-nya Indonesia karena ia KONTRAK, bukan nama internal (`D-80`).
type PrincipalDTO struct {
	ClientID             string `json:"client_id"`
	Name                 string `json:"nama_principal"`
	VirtualAccountNumber string `json:"nomor_virtual_account"`

	// Email adalah surel petugas yang dulu menerbitkan VA ini. Dikirim supaya layar dapat
	// mengisinya kembali saat panel penerbitan dibuka untuk principal yang sama.
	Email string `json:"email_inputor_va"`
}

// PrincipalListResponse adalah jawaban GET /api/master/recovery/principal.
type PrincipalListResponse struct {
	Principal []PrincipalDTO `json:"principal"`

	// Total dikirim eksplisit, bukan dibiarkan dihitung klien dari panjang senarai.
	// Angkanya harus datang dari server supaya yang dilaporkan layar adalah yang
	// benar-benar ada di basis data entitas itu.
	Total int `json:"total"`

	// Portal menyebut entitas yang BENAR-BENAR menjawab permintaan ini.
	//
	// Ia dikirim pada setiap jawaban, bukan diandaikan sama dengan yang diminta: satu
	// aplikasi melayani empat badan hukum dengan basis data terpisah (`ADR-0030`), dan
	// "data siapa ini" tidak boleh hanya ditebak dari keadaan layar (`R-20`).
	Portal string `json:"portal"`
}

// FormResponse adalah bekal awal layar: nomor batch berikutnya dan pilihan tahun.
//
// Keduanya disatukan dalam satu jawaban, bukan dua rute terpisah, karena keduanya selalu
// dibutuhkan bersamaan saat layar dibuka. `10-API-STRATEGY.md` §1 menetapkan satu layar
// sebaiknya dilayani satu permintaan — memecahnya akan membuat aplikasi baru terasa lebih
// lambat daripada yang lama meski servernya lebih cepat.
type FormResponse struct {
	// NextBatch adalah PERKIRAAN nomor batch, untuk ditampilkan saja.
	//
	// Disebut perkiraan karena memang begitu: dua petugas yang membuka layar bersamaan
	// melihat angka yang sama. Yang mengikat adalah nomor yang diterbitkan server saat
	// menyimpan, dan itulah yang dikembalikan pada jawaban simpan.
	NextBatch int64 `json:"nomor_batch_perkiraan"`

	// Year adalah pilihan isian Tahun, terbaru lebih dulu.
	Year []string `json:"tahun"`

	Portal string `json:"portal"`
}

// PolicyReferenceResponse adalah jawaban pencarian identitas polis.
type PolicyReferenceResponse struct {
	PolicyNo    string `json:"nomor_polis"`
	BusinessID  string `json:"id_lini_bisnis"`
	BranchID    string `json:"id_cabang"`
	AgentID     string `json:"id_agen"`
	MarketingID string `json:"id_marketing"`
	Portal      string `json:"portal"`
}

// VirtualAccountRequestDTO adalah isian panel penerbitan VA.
type VirtualAccountRequestDTO struct {
	ClientID      string `json:"client_id"`
	PrincipalName string `json:"nama_principal"`
	Email         string `json:"email_inputor_va"`
}

// VirtualAccountResponse adalah jawaban penerbitan VA.
type VirtualAccountResponse struct {
	Number  string `json:"nomor_virtual_account"`
	Status  string `json:"status"`
	Message string `json:"pesan"`

	// Reused menyatakan nomor ini DIAMBIL dari master, bukan baru diterbitkan.
	//
	// Ia dikirim sebagai penanda yang dapat dibaca program, bukan disiratkan lewat
	// kalimat di dalam `pesan` seperti sistem lama — layar perlu memutuskan nada
	// pemberitahuannya, dan mencocokkan teks pesan untuk itu adalah cara yang rapuh.
	Reused bool `json:"dipakai_ulang"`

	Portal string `json:"portal"`
}

// ClaimLineDTO adalah satu baris daftar klaim hasil unggahan CSV.
type ClaimLineDTO struct {
	PolicyNo    string `json:"nomor_polis"`
	ClaimAmount int64  `json:"nilai_klaim"`
}

// ClaimLineResponse adalah jawaban pembacaan berkas CSV.
//
// Ia TIDAK menyimpan apa pun: barisnya dikembalikan untuk ditampilkan, lalu dikirim
// kembali bersama permintaan simpan. Itu meniru sistem lama, yang menyusun daftarnya di
// klipboard sebelum menyimpannya sebagai satu dokumen JSON.
type ClaimLineResponse struct {
	ClaimLine []ClaimLineDTO `json:"baris_klaim"`
	Total     int            `json:"total"`

	// TotalClaimAmount adalah jumlah seluruh baris.
	//
	// Dihitung server dan dikirim supaya petugas dapat membandingkannya dengan angka yang
	// diketiknya sendiri SEBELUM menyimpan. Sistem lama tidak punya penjumlahan ini.
	TotalClaimAmount int64 `json:"jumlah_nilai_klaim"`

	// Rejected memuat baris yang DITOLAK beserta alasannya.
	//
	// Dikirim bersama yang diterima, bukan menggantikannya: berkas dengan satu baris cacat
	// tetap berguna, dan petugas berhak tahu persis baris mana yang tidak ikut.
	Rejected []ViolationDTO `json:"baris_ditolak,omitempty"`

	Portal string `json:"portal"`
}

// DocumentResponse adalah jawaban unggahan Bukti Bayar.
type DocumentResponse struct {
	// DocumentID adalah DATAID pada POOLDATA.DATA_ATTACHFILE. Layar menyimpannya dan
	// mengirimkannya kembali saat menyimpan batch.
	DocumentID string `json:"id_dokumen"`
	Name       string `json:"nama_berkas"`
	Portal     string `json:"portal"`
}

// SaveRequest adalah isian form Transfer Recovery.
//
// # Yang sengaja TIDAK diterima
//
//   - `nomor_batch` — diterbitkan penyimpanan di dalam transaksinya sendiri. Menerimanya
//     dari klien berarti dua petugas dapat memperebutkan nomor yang sama.
//   - `sisa` — dihitung server dengan `masterrecovery.Remainder`. Layar tetap
//     menghitungnya untuk diperlihatkan seketika; yang MENGIKAT yang di server, sehingga
//     keduanya tidak dapat berbeda pendapat tentang angka yang tersimpan.
//   - keempat identitas polis — dicari server dari nomor polis. Klien dapat mengirim apa
//     saja; yang tersimpan harus benar-benar milik polis itu.
//   - `username` — diambil dari sesi pemanggil.
//
// Badan yang memuat salah satunya DITOLAK, bukan diabaikan diam-diam: klien yang
// mengirimnya sedang salah paham tentang kontrak ini, dan mengabaikannya akan membuat
// salah paham itu bertahan.
type SaveRequest struct {
	PrincipalName        string `json:"nama_principal"`
	ClientID             string `json:"client_id"`
	VirtualAccountNumber string `json:"nomor_virtual_account"`
	Year                 string `json:"tahun"`

	ClaimAmount     int64 `json:"nilai_klaim"`
	PreviousPayment int64 `json:"pembayaran_sebelumnya"`
	Payment         int64 `json:"pembayaran"`

	Remark       string `json:"keterangan"`
	CasePosition string `json:"posisi_kasus"`

	PolicyNo   string `json:"nomor_polis"`
	DocumentID string `json:"id_dokumen"`

	// ServiceLogID mengisi kolom NOHPLL. Namanya menyiratkan nomor telepon; isinya BUKAN —
	// sistem lama mengisinya dengan nomor catatan log layanan. Ia opsional dan hampir
	// selalu kosong; dibawa supaya kolomnya tidak berubah arti.
	ServiceLogID string `json:"id_log_layanan"`

	ClaimLine []ClaimLineDTO `json:"baris_klaim"`
}

// RecoveryDTO adalah batch yang tersimpan, dikembalikan apa adanya sesudah disimpan.
type RecoveryDTO struct {
	Batch                int64  `json:"nomor_batch"`
	PrincipalName        string `json:"nama_principal"`
	ClientID             string `json:"client_id"`
	VirtualAccountNumber string `json:"nomor_virtual_account"`
	Year                 string `json:"tahun"`

	ClaimAmount     int64 `json:"nilai_klaim"`
	PreviousPayment int64 `json:"pembayaran_sebelumnya"`
	Payment         int64 `json:"pembayaran"`

	// Remainder adalah angka yang BENAR-BENAR tersimpan, hasil hitungan server.
	//
	// Dikembalikan supaya layar dapat memperlihatkan yang tersimpan, bukan yang
	// dihitungnya sendiri — bila keduanya berbeda, yang benar adalah yang ini.
	Remainder int64 `json:"sisa"`

	Remark       string `json:"keterangan"`
	CasePosition string `json:"posisi_kasus"`
	DocumentID   string `json:"id_dokumen"`
	InputBy      string `json:"dicatat_oleh"`
	PolicyNo     string `json:"nomor_polis"`

	BusinessID  string `json:"id_lini_bisnis"`
	BranchID    string `json:"id_cabang"`
	AgentID     string `json:"id_agen"`
	MarketingID string `json:"id_marketing"`

	ClaimLine []ClaimLineDTO `json:"baris_klaim"`
}

// SaveResponse adalah jawaban Transfer Recovery.
type SaveResponse struct {
	Recovery RecoveryDTO `json:"recovery"`

	// PolicyResolved menyatakan keempat identitas polis berhasil dilengkapi.
	//
	// Ia dikirim TERPISAH, bukan disimpulkan layar dari kolom yang kosong: kosong dapat
	// berarti nomor polisnya memang tidak diisi, dan itu hal yang berbeda dari nomor polis
	// yang diisi tetapi tidak ditemukan. Batch tetap tersimpan pada kedua keadaan —
	// sistem lama pun demikian — dan penanda ini yang membuat layar dapat mengatakannya.
	PolicyResolved bool `json:"identitas_polis_terisi"`

	Portal string `json:"portal"`
}

// ViolationDTO adalah satu aturan yang dilanggar beserta isian yang melanggarnya.
//
// Field dikirim supaya layar dapat menandai kolom yang salah, bukan sekadar menampilkan
// satu pesan di atas form — yang persis itulah yang dilakukan sistem lama dengan satu
// kalimat "Wajib ISI semua field" tanpa menyebut isian mana.
type ViolationDTO struct {
	Field   string `json:"field"`
	Message string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Kode dimaksudkan untuk dibaca program, Message untuk dibaca manusia. Klien membedakan
// jenis galat lewat Kode — bukan dengan mencocokkan teks Message.
//
// Bentuknya sengaja dibuat sama dengan modul master lain. Menyatukannya menjadi satu tipe
// bersama adalah lingkup TKT-F1-004, kontrak galat yang mengikat seluruh aplikasi — dan
// tiket itu masih terhalang keputusan Work Owner. Sampai itu diputuskan, tipe yang
// berbentuk sama lebih jujur daripada satu tipe bersama yang menyiratkan kontraknya sudah
// ada.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`

	// Detail hanya terisi pada galat validasi, dan memuat SELURUH pelanggaran sekaligus.
	Detail []ViolationDTO `json:"detail,omitempty"`
}
