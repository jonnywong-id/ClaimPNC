// Package inboxlaporanklaimhttp adalah lapisan transport modul Inbox Laporan Klaim.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// portal/http: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `inboxlaporanklaimhttp` supaya tidak menutupi `net/http`.
package inboxlaporanklaimhttp

import (
	"time"

	"claim-pnc/internal/inboxlaporanklaim"
)

// ClaimReportDTO adalah bentuk satu baris daftar yang dikirim ke peramban.
//
// Terpisah dari inboxlaporanklaim.ClaimReport supaya perubahan internal tidak bocor ke
// klien dan sebaliknya (`08-TECHNICAL-STRATEGY.md` §2 aturan 4).
//
// Nama field JSON berbahasa Indonesia — ia kontrak, bukan nama internal (`D-80`).
type ClaimReportDTO struct {
	ID          string `json:"id"`
	NomorKlaim  string `json:"nomor_klaim"`
	NomorPolis  string `json:"nomor_polis"`
	Tertanggung string `json:"tertanggung"`
	NamaBisnis  string `json:"nama_bisnis"`
	NomorRujuk  string `json:"nomor_rujukan"`

	// Tanggal dikirim sebagai `YYYY-MM-DD`, bukan sebagai timestamp berzona.
	//
	// Yang dibaca petugas di kolom "Date of loss" dan "Input Date" adalah TANGGAL, dan
	// mengirimkannya sebagai timestamp UTC membuat peramban di WIB menggesernya mundur
	// sehari pada jam-jam awal. Kelas kesalahan itu tepat yang melahirkan 101
	// penyesuaian tujuh jam di sistem lama; ia tidak dilahirkan kembali di sini.
	//
	// Kosong berarti tanggalnya memang belum ada, bukan nol yang tampak sah.
	TanggalKejadian string `json:"tanggal_kejadian"`
	TanggalMasuk    string `json:"tanggal_masuk"`
	TanggalAging    string `json:"tanggal_aging"`

	// UmurHari adalah kolom "Total Aging", dihitung server terhadap satu waktu acuan
	// untuk seluruh baris — bukan dihitung ulang tiap baris di peramban.
	UmurHari int `json:"umur_hari"`

	Pembuat    string `json:"pembuat"`
	KodeCabang string `json:"kode_cabang"`
	NamaCabang string `json:"nama_cabang"`

	Alasan      string `json:"alasan"`
	SubjekEmail string `json:"subjek_email"`
	PesanAkhir  string `json:"pesan_akhir"`

	// Posisi memakai teks yang sama dengan layar Pega: Outstanding, Not Registered,
	// Not Transferred (`D-13`).
	Posisi string `json:"posisi"`

	// Asal menyebut sistem yang menerbitkan baris ini: "pega" atau "claimpnc".
	Asal string `json:"asal"`

	// RujukanPega adalah kunci penugasan warisan, untuk membuka berkasnya di Pega selama
	// masa paralel. Kosong untuk berkas yang diterbitkan aplikasi ini.
	RujukanPega string `json:"rujukan_pega"`
}

// CategoryDTO adalah satu tab beserta lencana jumlahnya.
type CategoryDTO struct {
	Kode  string `json:"kode"`
	Judul string `json:"judul"`

	// Jumlah bernilai nil untuk tab yang memang tidak dicacah sistem lama — lihat
	// inboxlaporanklaim.Summary.CountOf. Ia sengaja dibedakan dari nol: lencana
	// bertuliskan 0 berarti "tidak ada berkas", ketiadaan lencana berarti "tidak
	// dihitung".
	Jumlah *int `json:"jumlah"`

	// Komunikasi menandai tab yang isinya percakapan; layar memakainya untuk memutuskan
	// apakah kolom "Last message" digambar.
	Komunikasi bool `json:"komunikasi"`
}

// OptionDTO adalah satu pilihan dropdown.
type OptionDTO struct {
	Kode string `json:"kode"`
	Nama string `json:"nama"`
}

// PaginationDTO menyatakan letak halaman yang sedang ditampilkan.
type PaginationDTO struct {
	Halaman      int `json:"halaman"`
	Ukuran       int `json:"ukuran"`
	Total        int `json:"total"`
	TotalHalaman int `json:"total_halaman"`
}

// ListResponse adalah jawaban GET /api/inbox/laporan-klaim.
type ListResponse struct {
	Laporan  []ClaimReportDTO `json:"laporan"`
	Kategori []CategoryDTO    `json:"kategori"`
	Halaman  PaginationDTO    `json:"halaman"`

	// Portal menyebut entitas yang benar-benar menjawab permintaan ini.
	//
	// Ia dikirim balik dengan sengaja: layar dapat memastikan data yang tampil memang
	// milik entitas yang dipilih pengguna. Pada aplikasi yang melayani empat badan
	// hukum, "data siapa ini" tidak boleh hanya diandaikan.
	Portal string `json:"portal"`
}

// OptionResponse adalah jawaban GET /api/inbox/laporan-klaim/pilihan.
//
// Ketiga daftar dikirim sekaligus karena layar membutuhkan ketiganya sebelum dapat
// menggambar satu baris pun. Tiga permintaan terpisah hanya menambah perjalanan jaringan
// tanpa menambah apa pun yang dapat dipakai sendiri-sendiri.
type OptionResponse struct {
	Kategori []CategoryDTO `json:"kategori"`
	Bisnis   []OptionDTO   `json:"bisnis"`
	Kanwil   []OptionDTO   `json:"kanwil"`
	Portal   string        `json:"portal"`
}

// SingleResponse adalah jawaban pembuatan berkas baru dan pembacaan satu berkas.
type SingleResponse struct {
	Laporan ClaimReportDTO `json:"laporan"`
	Portal  string         `json:"portal"`
}

// ErrorResponse adalah bentuk galat modul ini.
//
// Bentuknya sama dengan modul auth — `{kode, pesan}` — ditambah `detail` untuk
// pelanggaran per isian. Klien membedakan jenis galat lewat `kode`, tidak pernah dengan
// mencocokkan teks `pesan`.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Detail  []ViolationDTO `json:"detail,omitempty"`
}

// ViolationDTO adalah satu isian yang tidak lolos pemeriksaan.
type ViolationDTO struct {
	Field   string `json:"kolom"`
	Message string `json:"pesan"`
}

// dateLayout adalah bentuk tanggal pada kontrak API modul ini.
const dateLayout = "2006-01-02"

// toDTO mengubah satu baris domain menjadi bentuk yang dikirim ke peramban.
//
// `now` adalah waktu acuan umur berkas, dan ia diberikan pemanggil supaya SELURUH baris
// pada satu jawaban dihitung terhadap acuan yang sama. Memanggil jam per baris membuat
// dua baris berdampingan dapat jatuh di dua hari yang berbeda tepat di tengah malam.
func toDTO(r inboxlaporanklaim.ClaimReport, now time.Time) ClaimReportDTO {
	return ClaimReportDTO{
		ID:              r.ID,
		NomorKlaim:      r.ClaimNumber,
		NomorPolis:      r.PolicyNumber,
		Tertanggung:     r.InsuredName,
		NamaBisnis:      r.BusinessName,
		NomorRujuk:      r.ReferenceNumber,
		TanggalKejadian: dateText(r.DateOfLoss),
		TanggalMasuk:    dateText(r.CreatedAt),
		TanggalAging:    dateText(r.AgingAt),
		UmurHari:        r.AgingDays(now),
		Pembuat:         r.CreatedBy,
		KodeCabang:      r.BranchCode,
		NamaCabang:      r.BranchName,
		Alasan:          r.Reason,
		SubjekEmail:     r.EmailSubject,
		PesanAkhir:      r.LastMessage,
		Posisi:          string(r.Position),
		Asal:            string(r.Origin),
		RujukanPega:     r.LegacyAssignmentKey(),
	}
}

// toListDTO mengubah sekumpulan baris domain.
//
// Slice-nya selalu dibuat, tidak pernah dibiarkan nil, supaya daftar kosong terkirim
// sebagai `[]` dan bukan `null` — layar yang menerima `null` harus menjaganya sendiri,
// dan satu layar yang lupa akan gagal justru saat daftarnya kosong.
func toListDTO(list []inboxlaporanklaim.ClaimReport, now time.Time) []ClaimReportDTO {
	result := make([]ClaimReportDTO, 0, len(list))
	for _, r := range list {
		result = append(result, toDTO(r, now))
	}
	return result
}

// toCategoryDTO menyusun kesembilan tab beserta lencananya.
func toCategoryDTO(summary inboxlaporanklaim.Summary) []CategoryDTO {
	list := inboxlaporanklaim.ListCategories()
	result := make([]CategoryDTO, 0, len(list))

	for _, info := range list {
		item := CategoryDTO{
			Kode:       string(info.Category),
			Judul:      info.Title,
			Komunikasi: info.Message,
		}
		if count, counted := summary.CountOf(info.Category); counted {
			// Salinan diambil per putaran; mengirim alamat variabel perulangan akan
			// membuat kesembilan tab menunjuk angka yang sama.
			value := count
			item.Jumlah = &value
		}
		result = append(result, item)
	}
	return result
}

func dateText(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(dateLayout)
}
