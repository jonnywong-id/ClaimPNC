package inboxanalystdoctorhttp

import (
	"time"

	"claim-pnc/internal/inboxanalystdoctor"
	"claim-pnc/internal/inboxanalystdoctor/usecase"
)

// Bentuk jawaban modul Inbox Analyst Doctor.
//
// Nama field JSON berbahasa Indonesia karena ia KONTRAK yang dibaca layar, bukan nama
// internal (`D-80`); yang berbahasa Inggris hanyalah nama tipe dan nama field Go-nya.

// TaskDTO adalah satu baris antrean.
type TaskDTO struct {
	// KlaimID adalah kunci teknis `PZINSKEY`.
	//
	// TIDAK digambar sebagai kolom — isinya memuat nama kelas internal Pega — tetapi dikirim
	// karena tautan baris membutuhkannya untuk membuka klaimnya.
	KlaimID string `json:"klaim_id"`

	NomorCase       string `json:"nomor_case"`
	NomorPolis      string `json:"nomor_polis"`
	NamaTertanggung string `json:"nama_tertanggung"`
	NamaCabang      string `json:"nama_cabang"`
	NamaAdmin       string `json:"nama_admin"`

	// KomentarPICTeknis masih KOSONG terhadap Oracle sampai kolomnya dikonfirmasi DBA.
	//
	// Ia tetap dikirim, tidak dihilangkan dari kontrak: kolom yang hilang dari jawaban akan
	// membuat layar tidak menggambarnya sama sekali, dan isian yang belum terbawa menjadi
	// tidak terlihat oleh siapa pun.
	KomentarPICTeknis string `json:"komentar_pic_teknis"`

	// PICTeknis diambil Report Definition tetapi TIDAK digambar sebagai kolom (`D-13`).
	PICTeknis string `json:"pic_teknis"`

	// TanggalPendaftaran berbentuk YYYY-MM-DD, sudah dalam WIB.
	//
	// Dikonversi SERVER, bukan di peramban. Membiarkan tiap tempat di frontend memutuskan
	// sendiri cara mengubahnya ke WIB adalah cara paling cepat mengulang cacat `Set7Hours`
	// sistem lama, yang menambah tujuh jam manual di 118 titik pada 36 activity (`R-12`).
	TanggalPendaftaran string `json:"tanggal_pendaftaran"`

	// LamaHari adalah kolom "Lama Waktu Klaim" — umur tugas dalam hari, dihitung server.
	LamaHari int `json:"lama_hari"`

	StatusProses string `json:"status_proses"`

	// OperatorPenerima adalah pemilik penugasan. Selalu sama dengan pemanggil pada keadaan
	// biasa; dikirim supaya antrean kosong dapat ditelusuri tanpa membuka basis data.
	OperatorPenerima string `json:"operator_penerima"`
}

// ColumnDTO adalah satu judul kolom.
type ColumnDTO struct {
	Kunci      string `json:"kunci"`
	Judul      string `json:"judul"`
	Keterangan string `json:"keterangan,omitempty"`
}

// MetadataResponse adalah keterangan layar.
type MetadataResponse struct {
	Portal string `json:"portal"`

	Kolom []ColumnDTO `json:"kolom"`

	// SelisihTerencana adalah perbedaan yang DISENGAJA terhadap layar Pega (`D-54`).
	SelisihTerencana []string `json:"selisih_terencana"`

	// Keterbatasan adalah penghalang yang masih menunggu pihak lain, dan akan hilang dengan
	// sendirinya begitu penghalangnya hilang.
	Keterbatasan []string `json:"keterbatasan"`

	UkuranHalaman int `json:"ukuran_halaman"`
}

// ListResponse adalah satu halaman antrean.
type ListResponse struct {
	Portal string `json:"portal"`

	Data []TaskDTO `json:"data"`

	// Total adalah jumlah SELURUH baris yang cocok, bukan jumlah baris di halaman ini.
	Total int `json:"total"`

	// Lewati dan Batas adalah paginasi yang BENAR-BENAR dipakai, bukan yang diminta.
	//
	// Permintaan `batas=5000` dipangkas menjadi 100, dan tanpa mengembalikan angka yang
	// dipakai, bilah halaman di layar akan menghitung jumlah halaman yang salah.
	Lewati int `json:"lewati"`
	Batas  int `json:"batas"`

	Cari string `json:"cari"`
}

// ViolationDTO adalah satu pelanggaran pada satu isian.
type ViolationDTO struct {
	Isian string `json:"isian"`
	Pesan string `json:"pesan"`
}

// ErrorResponse adalah bentuk galat yang dibaca klien.
type ErrorResponse struct {
	Code    string         `json:"kode"`
	Message string         `json:"pesan"`
	Details []ViolationDTO `json:"detail,omitempty"`
}

// toMetadataResponse menyusun keterangan layar.
func toMetadataResponse(meta usecase.Metadata, portalAlias string) MetadataResponse {
	columns := make([]ColumnDTO, 0, len(meta.Columns))
	for _, c := range meta.Columns {
		columns = append(columns, ColumnDTO{Kunci: c.Key, Judul: c.Title, Keterangan: c.Note})
	}

	return MetadataResponse{
		Portal:           portalAlias,
		Kolom:            columns,
		SelisihTerencana: meta.PlannedDifferences,
		Keterbatasan:     meta.Limitations,
		UkuranHalaman:    meta.PageSize,
	}
}

// toListResponse menyusun satu halaman antrean.
//
// `now` dan `loc` diserahkan pemanggil, bukan dibaca dari jam sistem di sini: keduanya yang
// menentukan kolom "Lama Waktu Klaim", dan seam waktu (`F-5`) ada justru supaya kolom itu
// dapat diuji tanpa bergantung pada mesin yang menjalankannya.
func toListResponse(
	listed usecase.Listed,
	portalAlias string,
	now time.Time,
	loc *time.Location,
) ListResponse {
	data := make([]TaskDTO, 0, len(listed.Page.Tasks))
	for _, task := range listed.Page.Tasks {
		data = append(data, toTaskDTO(task, now, loc))
	}

	return ListResponse{
		Portal: portalAlias,
		Data:   data,
		Total:  listed.Page.Total,
		Lewati: listed.Filter.Offset,
		Batas:  listed.Filter.Limit,
		Cari:   listed.Filter.Search,
	}
}

// toTaskDTO mengubah satu tugas menjadi bentuk yang dibaca layar.
func toTaskDTO(
	task inboxanalystdoctor.AnalystDoctorTask,
	now time.Time,
	loc *time.Location,
) TaskDTO {
	return TaskDTO{
		KlaimID:            task.ClaimID,
		NomorCase:          task.ClaimNumber,
		NomorPolis:         task.PolicyNumber,
		NamaTertanggung:    task.InsuredName,
		NamaCabang:         task.BranchName,
		NamaAdmin:          task.AdminName,
		KomentarPICTeknis:  task.TechnicalPICNote,
		PICTeknis:          task.TechnicalPIC,
		TanggalPendaftaran: formatDate(task.RegisteredAt, loc),
		LamaHari:           task.DurationDays(now, loc),
		StatusProses:       task.ProcessStatus,
		OperatorPenerima:   task.AssignedOperator,
	}
}

// formatDate mengubah waktu UTC menjadi tanggal WIB berbentuk YYYY-MM-DD.
//
// Inilah SATU-SATUNYA tempat konversi zona waktu terjadi pada modul ini. Menyebarkannya akan
// mengulang persis cacat `Set7Hours` sistem lama, yang menambah tujuh jam manual di 118 titik
// pada 36 activity sehingga satu tempat yang lupa menggeser tanggal tanpa terdeteksi.
func formatDate(t time.Time, loc *time.Location) string {
	if t.IsZero() {
		return ""
	}
	if loc == nil {
		loc = time.UTC
	}
	return t.In(loc).Format("2006-01-02")
}
