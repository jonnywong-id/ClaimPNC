// Package inboxreceivetkahttp adalah lapisan transport modul Inbox Receive TKA.
//
// Namanya mengikuti `D-81`: folder modul memakai nama modul bisnis apa adanya ("Inbox
// Receive TKA"), dan paket transportnya menambahkan akhiran `http` tanpa tanda hubung karena
// Go tidak mengizinkannya. Pola yang sama dipakai `inboxinvestigatorhttp`, `masterreashttp`,
// dan `riwayatklaimhttp`.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: Task membawa Reference, kunci teknis
// Pega yang justru tidak boleh ditampilkan.
//
// Lapisan Transport — ia boleh tahu Domain, dan dilarang tahu SQL maupun nama tabel.
package inboxreceivetkahttp

import (
	"strings"
	"time"

	"claim-pnc/internal/inboxreceivetka"
)

// dateLayout adalah bentuk tanggal pada kontrak API.
//
// # TANGGAL saja, tanpa jam, dan itu keputusan yang disengaja
//
// Kedua kolomnya bertipe `DATE` pada basis data, dan keduanya memang tanggal kalender —
// tanggal kejadian dan tanggal kelengkapan dokumen. Mengirimkannya sebagai stempel waktu
// lengkap berarti mengarang bagian jam yang tidak pernah ada, dan jam karangan itu punya
// akibat nyata: peramban yang menerima `2026-09-14T00:00:00Z` lalu menampilkannya dalam WIB
// akan menuliskan **15 September**.
//
// Itu persis kelas cacat yang `R-12` catat dan yang `F-5` dibangun untuk menutupnya. Bentuk
// tanggal murni membuatnya tidak mungkin terjadi: tidak ada jam, maka tidak ada yang dapat
// bergeser.
//
// Ia tetap ISO 8601, sehingga dapat diurutkan sebagai teks — yang berformat `dd/mm/yyyy`
// tidak. Pemformatan untuk layar dilakukan di frontend.
const dateLayout = "2006-01-02"

// TaskDTO adalah satu baris inbox sebagaimana dilihat klien.
//
// # Kenapa nama field JSON-nya bahasa Indonesia
//
// Ia KONTRAK, bukan nama internal (`D-80`). Nama tipe, field Go, dan variabel di modul ini
// seluruhnya bahasa Inggris; yang tetap Indonesia hanyalah yang dipakai di luar kode.
//
// # Namanya mengikuti CAPTION GRID layar lama, dengan satu pengecualian
//
// `D-13` menetapkan tampilan meniru Pega supaya pengguna tidak perlu belajar ulang, dan
// caption itulah yang selama ini mereka baca. Pengecualiannya kolom kelima; lihat
// DateOfLoss.
type TaskDTO struct {
	// Reference adalah kunci teknis Pega (`pzInsKey`), dan **kunci baris** layar ini.
	//
	// Ia dikirim tetapi tidak ditampilkan. Selalu terisi — sumbernya tabel yang
	// menggerakkan kuerinya, bukan hasil gabungan.
	Reference string `json:"referensi"`

	// ClaimAvailable menyatakan klaimnya ditemukan di data klaim utama.
	//
	// # Kenapa layar perlu mengetahuinya SEBELUM Submit ditekan
	//
	// Pekerjaan yang ada di antrean Pega tetapi belum ada di tabel klaim bisnis TETAP
	// TAMPIL — gabungannya sengaja LEFT, sama seperti Pega yang juga menampilkannya.
	// Tetapi tanggalnya tidak dapat disimpan: tidak ada baris klaim yang dapat diperbarui.
	//
	// Bernilai false, layar mematikan isian dan tombolnya beserta alasannya — alih-alih
	// membiarkan pengguna mengetik tanggal lalu ditolak.
	ClaimAvailable bool `json:"klaim_tersedia"`

	// ClaimNumber — caption "Nomor Klaim", yaitu `pyID`.
	ClaimNumber string `json:"nomor_klaim"`

	// PolicyNumber — caption "No Polis".
	PolicyNumber string `json:"nomor_polis"`

	// InsuredName — caption "Nama Tertanggung".
	//
	// Pemasangannya sempat diragukan karena Report Definition melabelinya terbalik;
	// badan surel `NotificationKelengkapanTKA` menyelesaikannya. Lihat
	// inboxreceivetka.Task.
	InsuredName string `json:"nama_tertanggung"`

	// ParticipantName — caption "Nama Peserta".
	ParticipantName string `json:"nama_peserta"`

	// DateOfLoss adalah kolom kelima, yang di layar lama bercaption **"Date Of Loss"**.
	//
	// # Nama field-nya bahasa Indonesia meski captionnya bahasa Inggris
	//
	// Inilah pengecualian atas aturan "nama field mengikuti caption". `D-80` menetapkan
	// nama field JSON memakai bahasa Indonesia, dan `CONTEXT.md` sudah menetapkan padanan
	// resminya: **Tanggal Kejadian**. Menamainya `date_of_loss` akan membawa satu-satunya
	// nama berbahasa Inggris ke dalam kontrak yang seluruhnya Indonesia.
	//
	// Yang tetap mengikuti Pega adalah **teks di layar**: kolomnya diberi judul
	// "Date Of Loss" di frontend, persis seperti yang dibaca pengguna selama ini.
	//
	// Null bila kolomnya kosong.
	DateOfLoss *string `json:"tanggal_kejadian"`

	// RegisteredOn mengisi kolom yang di layar lama bercaption **"Aging"**.
	//
	// # Yang dikirim adalah TANGGAL, bukan lama menunggu
	//
	// Pega menampilkan kolom itu sebagai waktu relatif — "2 years 6 months ago" — tetapi
	// yang disimpannya adalah `.ClaimData.RegisterDate`. Terverifikasi: klaim
	// ber-`REGISTERDATE_1 = 20240319` ditampilkan Pega sebagai "2 years 6 months ago",
	// tepat selisihnya terhadap hari pembacaan.
	//
	// Server mengirim tanggalnya; **layar yang menyusun kalimatnya**. Alasannya satu:
	// "berapa lama menunggu" bergantung pada KAPAN ia dibaca. Menghitungnya di server
	// berarti nilainya membeku pada saat permintaan, dan modul ini akan membutuhkan seam
	// Clock hanya demi satu label tampilan.
	//
	// Nama field mengikuti ISI supaya kontraknya tidak berbohong tentang tipe datanya;
	// yang mengikuti Pega adalah judul kolom di layar (`D-13`).
	//
	// Null bila kosong atau bentuk teks sumbernya tidak dikenali.
	RegisteredOn *string `json:"tanggal_registrasi"`
}

// ListResponse adalah jawaban daftar.
//
// Alias portal ikut dikirim, sama seperti modul lain: satu aplikasi melayani empat badan
// hukum dengan basis data terpisah, dan layar menyebut terang-terangan daftar siapa yang
// sedang ditampilkan (`ADR-0030`, `R-20`).
type ListResponse struct {
	Task []TaskDTO `json:"tugas"`

	// Truncated menyatakan masih ada pekerjaan yang cocok tetapi TIDAK terkirim.
	//
	// Ia ada karena sistem lama memotong pada 500 baris **tanpa memberi tahu siapa pun**
	// (`pyMaxRecords`). Batas yang diketahui adalah batas; batas yang senyap adalah data
	// yang hilang.
	Truncated bool `json:"terpotong"`

	// MaxRows adalah batas yang berlaku, dikirim supaya layar dapat menyebut angkanya
	// alih-alih menuliskannya sendiri — dua tempat yang dapat berbeda tanpa ada yang tahu.
	MaxRows int `json:"batas_baris"`

	Portal string `json:"portal"`
}

// CompleteRequest adalah badan permintaan pengisian tanggal kelengkapan dokumen.
//
// # Satu baris, satu permintaan
//
// Grid layar lama menggambar isian tanggal pada SETIAP baris dengan satu tombol Submit di
// bawahnya, sehingga secara tampilan ia tampak dapat mengisi banyak baris sekaligus.
// Activity yang dijalankannya tidak begitu: `SubmitTanggalLengkapTKA` menerima `Param.Inskey`
// dan `Param.Tanggal` dalam bentuk TUNGGAL, membuka satu case, dan menyimpan satu tanggal.
//
// Bentuk ini mengikuti activity-nya, bukan tampilannya. Mengirim banyak baris sekaligus
// akan menuntut keputusan yang belum pernah diambil siapa pun — apakah kegagalan pada baris
// ketiga membatalkan dua yang pertama.
type CompleteRequest struct {
	// ClaimNumber menunjuk baris yang diisi.
	ClaimNumber string `json:"nomor_klaim"`

	// CompletedAt adalah tanggal kelengkapan dokumen, berformat `YYYY-MM-DD`.
	CompletedAt string `json:"tanggal_dokumen_lengkap"`
}

// CompleteResponse adalah jawaban pengisian yang berhasil.
type CompleteResponse struct {
	// ClaimNumber adalah baris yang barusan terisi.
	ClaimNumber string `json:"nomor_klaim"`

	// CompletedAt adalah tanggal yang tersimpan, dikirim kembali supaya layar menampilkan
	// apa yang BENAR-BENAR tersimpan, bukan apa yang tadi dikirimnya.
	CompletedAt string `json:"tanggal_dokumen_lengkap"`

	// NotificationAttempted bernilai false bila pemberitahuan memang tidak dipasang di
	// lingkungan ini.
	//
	// Ia dipisahkan dari NotificationSent supaya layar dapat membedakan "belum dipasang"
	// dari "gagal dikirim". Menyatukan keduanya akan membuat lingkungan pengembangan yang
	// memang tidak punya server surel terus-menerus menampilkan peringatan gagal — dan
	// peringatan yang selalu muncul berhenti dibaca.
	NotificationAttempted bool `json:"pemberitahuan_dicoba"`

	// NotificationSent bernilai true hanya bila surelnya benar-benar terkirim.
	//
	// # Gagal kirim TIDAK membatalkan penyimpanan
	//
	// Itu perbedaan yang disengaja terhadap sistem lama, tempat surel dikirim sebelum
	// `Commit` sehingga kegagalannya membatalkan seluruh pekerjaan. Lihat usecase.Complete.
	NotificationSent bool `json:"pemberitahuan_terkirim"`

	Portal string `json:"portal"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul lain — `{kode, pesan}` — supaya klien tidak menghadapi dua
// bentuk galat yang berbeda. Penyatuannya menjadi satu tipe bersama adalah lingkup
// `TKT-F1-004`, yang masih terhalang keputusan Work Owner.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}

// toTaskDTO memetakan satu baris domain menjadi bentuk yang dikirim.
func toTaskDTO(task inboxreceivetka.Task) TaskDTO {
	return TaskDTO{
		Reference:       task.Reference,
		ClaimAvailable:  strings.TrimSpace(task.ClaimKey) != "",
		ClaimNumber:     task.ClaimNumber,
		PolicyNumber:    task.PolicyNumber,
		InsuredName:     task.InsuredName,
		ParticipantName: task.ParticipantName,
		DateOfLoss:      toDateString(task.DateOfLoss),
		RegisteredOn:    toDateString(task.RegisteredOn),
	}
}

// toListDTO memetakan seluruh baris.
//
// Senarai kosong dikembalikan sebagai `[]`, bukan `null`: klien yang memetakan hasilnya
// tanpa memeriksa nil akan gagal pada `null`, dan daftar yang memang kosong adalah keadaan
// yang WAJAR di sini — inbox yang bersih justru yang diharapkan.
func toListDTO(page inboxreceivetka.Page) []TaskDTO {
	result := make([]TaskDTO, 0, len(page.Tasks))
	for _, task := range page.Tasks {
		result = append(result, toTaskDTO(task))
	}
	return result
}

// toListResponse merakit jawaban daftar.
func toListResponse(page inboxreceivetka.Page, portalAlias string) ListResponse {
	return ListResponse{
		Task:      toListDTO(page),
		Truncated: page.Truncated,
		MaxRows:   inboxreceivetka.MaxRows,
		Portal:    portalAlias,
	}
}

// parseCompletion mengubah badan permintaan menjadi nilai domain.
//
// Galat yang dihasilkannya adalah galat DOMAIN, bukan galat bentuk — supaya transport
// memetakannya ke 422 dan bukan 400. Tanggal yang tidak dapat diurai adalah satu-satunya
// perkecualian; lihat errors.go.
func parseCompletion(body CompleteRequest) (inboxreceivetka.Completion, error) {
	claimNumber := strings.TrimSpace(body.ClaimNumber)
	raw := strings.TrimSpace(body.CompletedAt)

	if claimNumber == "" {
		return inboxreceivetka.Completion{}, inboxreceivetka.ErrClaimNumberRequired
	}
	if raw == "" {
		// Padanan langsung precondition `Param.Tanggal==""` pada activity lama, yang
		// menuliskan "Silahkan isi tanggal terlebih dahulu".
		return inboxreceivetka.Completion{}, inboxreceivetka.ErrDateRequired
	}

	moment, err := time.Parse(dateLayout, raw)
	if err != nil {
		return inboxreceivetka.Completion{}, errMalformedDate
	}

	return inboxreceivetka.Completion{
		ClaimNumber: claimNumber,
		CompletedAt: moment,
	}.Clean(), nil
}

// toCompleteResponse merakit jawaban pengisian.
func toCompleteResponse(
	claimNumber string,
	completedAt time.Time,
	attempted, sent bool,
	portalAlias string,
) CompleteResponse {
	return CompleteResponse{
		ClaimNumber:           claimNumber,
		CompletedAt:           completedAt.Format(dateLayout),
		NotificationAttempted: attempted,
		NotificationSent:      sent,
		Portal:                portalAlias,
	}
}

// toDateString memformat tanggal, atau nil bila kosong.
//
// Pointer, bukan teks kosong: `null` menyatakan "tidak ada tanggalnya", sedangkan `""` akan
// terbaca layar sebagai tanggal yang gagal diformat.
func toDateString(value *time.Time) *string {
	if value == nil || value.IsZero() {
		return nil
	}
	formatted := value.Format(dateLayout)
	return &formatted
}
