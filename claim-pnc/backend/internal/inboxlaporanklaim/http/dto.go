// Package inboxlaporanklaimhttp adalah lapisan transport modul Inbox Laporan Klaim.
//
// Nama paketnya sengaja berbeda dari nama foldernya, mengikuti pola auth/http dan
// portal/http: foldernya `http` supaya letaknya seragam antarmodul, nama paketnya
// `inboxlaporanklaimhttp` supaya tidak menutupi `net/http`.
package inboxlaporanklaimhttp

import (
	"strings"
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

	/*
		BatasCabang menyebut cabang yang membatasi daftar ini.

		Kosong punya SATU arti: pengguna memilih kanwil, dan pilihan itu menggantikan
		batas cabangnya. Ia tidak pernah berarti "batasnya hilang" — permintaan yang
		cabangnya tidak dapat ditentukan dijawab `403 cabang_tidak_dikenali`, bukan
		dijawab daftar tanpa batas.

		Ia dikirim supaya layar dapat MENYATAKAN batasnya. Petugas yang tidak tahu
		daftarnya sedang disaring akan menyimpulkan tidak ada pekerjaan, padahal yang
		benar adalah tidak ada pekerjaan DI CABANGNYA.
	*/
	BatasCabang string `json:"batas_cabang"`

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

	// Isian adalah isi form Input Receive Document. Terisi hanya pada jawaban yang
	// memuat satu berkas — daftar tidak pernah membawanya.
	Isian *DetailDTO `json:"isian,omitempty"`

	// DapatDisunting menyatakan berkas ini boleh disimpan dari sini.
	//
	// Ia dihitung server, bukan disimpulkan layar dari `asal`. Aturannya — siapa yang
	// berwenang menulis sebuah baris selama masa paralel — milik server, dan menyalinnya
	// ke layar berarti satu aturan hidup di dua tempat.
	DapatDisunting bool `json:"dapat_disunting"`

	Portal string `json:"portal"`
}

// DetailDTO adalah isian form Input Receive Document.
//
// Nama field mengikuti label yang dibaca petugas di layar lama, bukan nama properti Pega
// yang mengikatnya — `.ReceiveDocument.Keterangan` misalnya adalah "Keterangan Belum
// Transfer", dan namanya di sini `alasan` sesuai kolom yang menyimpannya.
type DetailDTO struct {
	// Tanggal dikirim sebagai `YYYY-MM-DD`; kosong berarti belum diisi.
	TanggalTerimaDokumen string `json:"tanggal_terima_dokumen"`
	TanggalKejadian      string `json:"tanggal_kejadian"`

	NamaPelapor    string `json:"nama_pelapor"`
	EmailPelapor   string `json:"email_pelapor"`
	TeleponPelapor string `json:"telepon_pelapor"`
	NamaKurir      string `json:"nama_kurir"`

	NomorPolis   string `json:"nomor_polis"`
	Tertanggung  string `json:"tertanggung"`
	NamaBisnis   string `json:"nama_bisnis"`
	NomorRujukan string `json:"nomor_rujukan"`

	// EstimasiKerugian dikirim sebagai bilangan bulat SEN, tidak pernah sebagai pecahan
	// (`ADR-0016`). Rp 1.000 = 100000.
	EstimasiKerugian int64 `json:"estimasi_kerugian"`

	LokasiKejadian string `json:"lokasi_kejadian"`
	SubjekEmail    string `json:"subjek_email"`

	Kronologis                string `json:"kronologis"`
	RincianKerusakan          string `json:"rincian_kerusakan"`
	Alasan                    string `json:"alasan"`
	KeteranganBelumRegistrasi string `json:"keterangan_belum_registrasi"`

	JumlahDokumen int `json:"jumlah_dokumen"`
}

// SaveRequest adalah badan permintaan penyimpanan form.
//
// Bentuknya sama persis dengan DetailDTO, dan itu disengaja: form mengirim kembali
// seluruh isian setiap kali disimpan, sehingga permintaannya menggantikan dan idempoten.
type SaveRequest = DetailDTO

// toDetailDTO mengubah isian domain menjadi bentuk yang dikirim ke peramban.
func toDetailDTO(d inboxlaporanklaim.Detail) DetailDTO {
	return DetailDTO{
		TanggalTerimaDokumen:      dateText(d.ReceivedDate),
		TanggalKejadian:           dateText(d.DateOfLoss),
		NamaPelapor:               d.ReporterName,
		EmailPelapor:              d.ReporterEmail,
		TeleponPelapor:            d.ReporterPhone,
		NamaKurir:                 d.CourierName,
		NomorPolis:                d.PolicyNumber,
		Tertanggung:               d.InsuredName,
		NamaBisnis:                d.BusinessName,
		NomorRujukan:              d.ReferenceNumber,
		EstimasiKerugian:          int64(d.EstimateValue),
		LokasiKejadian:            d.LossLocation,
		SubjekEmail:               d.EmailSubject,
		Kronologis:                d.Chronology,
		RincianKerusakan:          d.DamageDetail,
		Alasan:                    d.Reason,
		KeteranganBelumRegistrasi: d.NotRegisteredNote,
		JumlahDokumen:             d.DocumentCount,
	}
}

// toDetail membaca isian dari badan permintaan.
//
// Tanggal yang tidak dapat dibaca menjadi NOL, bukan galat penguraian: isian tanggal
// yang dikosongkan pengguna memang sah, dan membedakan "kosong" dari "salah bentuk" pada
// tingkat ini hanya memindahkan pesan galat ke tempat yang tidak dapat menunjuk isiannya.
func toDetail(r SaveRequest) inboxlaporanklaim.Detail {
	return inboxlaporanklaim.Detail{
		ReceivedDate:      parseDate(r.TanggalTerimaDokumen),
		DateOfLoss:        parseDate(r.TanggalKejadian),
		ReporterName:      r.NamaPelapor,
		ReporterEmail:     r.EmailPelapor,
		ReporterPhone:     r.TeleponPelapor,
		CourierName:       r.NamaKurir,
		PolicyNumber:      r.NomorPolis,
		InsuredName:       r.Tertanggung,
		BusinessName:      r.NamaBisnis,
		ReferenceNumber:   r.NomorRujukan,
		EstimateValue:     inboxlaporanklaim.Money(r.EstimasiKerugian),
		LossLocation:      r.LokasiKejadian,
		EmailSubject:      r.SubjekEmail,
		Chronology:        r.Kronologis,
		DamageDetail:      r.RincianKerusakan,
		Reason:            r.Alasan,
		NotRegisteredNote: r.KeteranganBelumRegistrasi,
		DocumentCount:     r.JumlahDokumen,
	}
}

func parseDate(text string) time.Time {
	clean := strings.TrimSpace(text)
	if clean == "" {
		return time.Time{}
	}
	// Dibaca sebagai tanggal UTC, bukan tanggal lokal peladen: yang dikirim layar adalah
	// TANGGAL tanpa zona, dan menafsirkannya di zona peladen akan menggesernya saat
	// peladen dan pengguna berada di zona yang berbeda.
	parsed, err := time.ParseInLocation(dateLayout, clean, time.UTC)
	if err != nil {
		return time.Time{}
	}
	return parsed
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
