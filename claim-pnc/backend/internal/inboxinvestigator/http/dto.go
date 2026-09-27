// Package inboxinvestigatorhttp adalah lapisan transport modul Inbox Investigator.
//
// Namanya mengikuti `D-81`: folder modul memakai nama modul bisnis apa adanya ("Inbox
// Investigator"), dan paket transportnya menambahkan akhiran `http` tanpa tanda hubung
// karena Go tidak mengizinkannya. Pola yang sama dipakai `masterreashttp`,
// `riwayatklaimhttp`, dan `pelaporanklaimhttp`.
//
// DTO di sini TERPISAH dari tipe modul (`08-TECHNICAL-STRATEGY.md` §4.8). Memakai tipe
// domain langsung sebagai bentuk JSON membuat perubahan internal bocor ke klien dan
// sebaliknya — dan di modul ini bocornya akan nyata: Task membawa Reference, kunci teknis
// Pega yang justru tidak boleh ditampilkan.
//
// Lapisan Transport — ia boleh tahu Domain, dan dilarang tahu SQL maupun nama tabel.
package inboxinvestigatorhttp

import (
	"time"

	"claim-pnc/internal/inboxinvestigator"
)

// TaskDTO adalah satu baris inbox sebagaimana dilihat klien.
//
// # Kenapa nama field JSON-nya bahasa Indonesia
//
// Ia KONTRAK, bukan nama internal (`D-80`). Nama tipe, field Go, dan variabel di modul ini
// seluruhnya bahasa Inggris; yang tetap Indonesia hanyalah yang dipakai di luar kode — dan
// nama field JSON termasuk di dalamnya, karena mengubahnya adalah perubahan yang merusak
// klien, bukan penggantian nama.
//
// # Namanya mengikuti CAPTION GRID layar lama
//
// Bukan nama kolom basis data dan bukan nama properti Pega. `D-13` menetapkan tampilan
// meniru Pega supaya pengguna tidak perlu belajar ulang, dan caption itulah yang selama ini
// mereka baca. Pemetaan lengkapnya ada di kepala inboxinvestigator.sql.
type TaskDTO struct {
	// Reference adalah kunci teknis yang dibutuhkan untuk membuka pekerjaannya.
	//
	// Ia dikirim tetapi TIDAK ditampilkan. Layar kerja Investigator belum dibangun; begitu
	// ia ada, inilah yang dipakai membukanya — sehingga menyalakannya kelak tidak menuntut
	// perubahan kontrak.
	Reference string `json:"referensi"`

	// CaseNumber — caption "Nomor Case".
	CaseNumber string `json:"nomor_case"`

	// PolicyNumber — caption "No Polis".
	PolicyNumber string `json:"nomor_polis"`

	// InsuredName — caption "Nama Tertanggung".
	InsuredName string `json:"nama_tertanggung"`

	// ParticipantName — caption "Nama Peserta", yaitu objek pertanggungan PERTAMA.
	ParticipantName string `json:"nama_peserta"`

	// BusinessName — caption "Nama Bisnis". Section lama menulisnya "Nama Bisinis" pada
	// penyaringnya; salah ketik itu tidak dibawa.
	BusinessName string `json:"nama_bisnis"`

	// BranchName — caption "Nama Cabang".
	BranchName string `json:"nama_cabang"`

	// AdminName — caption "Nama Admin". Ia PEMBUAT kasus, bukan pemegangnya: pekerjaan di
	// workbasket memang belum bertuan (`D-26`).
	AdminName string `json:"nama_admin"`

	// RegisteredAt — caption "Tanggal Pendaftaran". Null bila kosong.
	RegisteredAt *string `json:"tanggal_pendaftaran"`

	// SurveyDate adalah kolom KESEMBILAN, yang di layar lama bercaption
	// **"Lama Masuk Inbox"** — `.ClaimData.SurveyResults(1).SurveyDate`.
	//
	// # Nama field-nya mengikuti ISI, bukan caption — pengecualian yang disengaja
	//
	// Aturan modul ini menamai field JSON menurut caption grid (`D-13`). Kolom ini
	// dikecualikan: captionnya menyebut durasi ("lama"), isinya tanggal. Menamainya
	// `lama_masuk_inbox` akan membuat kontraknya berbohong tentang tipe datanya sendiri,
	// dan klien yang memperlakukannya sebagai angka akan gagal.
	//
	// Yang tetap mengikuti Pega adalah **teks di layar**: kolomnya diberi judul
	// "Lama Masuk Inbox" di frontend. Lihat inboxinvestigator.Task.SurveyDate untuk bukti
	// pemasangan caption-ke-sel-nya.
	//
	// ISO 8601 UTC. Null bila klaimnya belum punya baris survei.
	SurveyDate *string `json:"tanggal_survey"`
}

// ListResponse adalah jawaban daftar.
//
// Alias portal ikut dikirim, sama seperti modul lain: satu aplikasi melayani empat badan
// hukum dengan basis data terpisah, dan layar menyebut terang-terangan antrean siapa yang
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

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul lain — `{kode, pesan}` — supaya klien tidak menghadapi dua
// bentuk galat yang berbeda. Penyatuannya menjadi satu tipe bersama adalah lingkup
// `TKT-F1-004`, yang masih terhalang keputusan Work Owner.
type ErrorResponse struct {
	Code    string `json:"kode"`
	Message string `json:"pesan"`
}

// dateTimeLayout adalah bentuk waktu pada kontrak API.
//
// ISO 8601 UTC, bukan `dd/mm/yyyy` seperti sistem lama. Pemformatan untuk layar dilakukan di
// frontend, dan waktu berformat ISO dapat diurutkan sebagai teks — yang berformat
// `dd/mm/yyyy` tidak.
const dateTimeLayout = "2006-01-02T15:04:05Z"

// toTaskDTO memetakan satu baris domain menjadi bentuk yang dikirim.
func toTaskDTO(task inboxinvestigator.Task) TaskDTO {
	return TaskDTO{
		Reference:       task.Reference,
		CaseNumber:      task.CaseNumber,
		PolicyNumber:    task.PolicyNumber,
		InsuredName:     task.InsuredName,
		ParticipantName: task.ParticipantName,
		BusinessName:    task.BusinessName,
		BranchName:      task.BranchName,
		AdminName:       task.AdminName,
		RegisteredAt:    toTimeString(task.RegisteredAt),
		SurveyDate:      toTimeString(task.SurveyDate),
	}
}

// toListDTO memetakan seluruh baris.
//
// Senarai kosong dikembalikan sebagai `[]`, bukan `null`: klien yang memetakan hasilnya
// tanpa memeriksa nil akan gagal pada `null`, dan antrean yang memang kosong adalah keadaan
// yang WAJAR di sini — inbox yang bersih justru yang diharapkan.
func toListDTO(page inboxinvestigator.Page) []TaskDTO {
	result := make([]TaskDTO, 0, len(page.Tasks))
	for _, task := range page.Tasks {
		result = append(result, toTaskDTO(task))
	}
	return result
}

// toListResponse merakit jawaban daftar.
func toListResponse(page inboxinvestigator.Page, portalAlias string) ListResponse {
	return ListResponse{
		Task:      toListDTO(page),
		Truncated: page.Truncated,
		MaxRows:   inboxinvestigator.MaxRows,
		Portal:    portalAlias,
	}
}

// toTimeString memformat waktu, atau nil bila kosong.
//
// Pointer, bukan teks kosong: `null` menyatakan "tidak ada tanggalnya", sedangkan `""` akan
// terbaca layar sebagai tanggal yang gagal diformat.
func toTimeString(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(dateTimeLayout)
	return &formatted
}
