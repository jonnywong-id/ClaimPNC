package inboxosclaimpercabanghttp

import (
	"time"

	"claim-pnc/internal/inboxosclaimpercabang"
	"claim-pnc/internal/inboxosclaimpercabang/usecase"
	"claim-pnc/internal/platform/clock"
)

// Berkas ini memuat bentuk jawaban popup Detail.
//
// Nama field JSON tetap berbahasa Indonesia — ia KONTRAK yang dibaca frontend, bukan nama
// internal (`D-80`).

// DetailResponse adalah jawaban lengkap popup.
type DetailResponse struct {
	// Header adalah delapan nilai ringkasan pada kepala popup.
	Header DetailHeaderDTO `json:"ringkasan"`

	// Objects adalah isi grid objek pertanggungan.
	Objects []DetailObjectDTO `json:"objek"`

	// ProgressHistory adalah isi grid riwayat progres, terbaru lebih dulu.
	ProgressHistory []DetailProgressDTO `json:"riwayat_progres"`

	// AdjusterMessages adalah isi grid komunikasi dengan loss adjuster.
	AdjusterMessages []DetailMessageDTO `json:"komunikasi_adjuster"`

	// Branch ikut dikirim supaya layar dapat memastikan popup yang terbuka memang milik
	// cabang yang sedang ditampilkan daftarnya.
	Branch BranchDTO `json:"cabang"`

	// Portal ikut dikirim dengan alasan yang sama seperti pada daftar — memastikan jawabannya
	// bukan sisa cache portal sebelumnya (`R-20`).
	Portal string `json:"portal"`
}

// DetailHeaderDTO adalah delapan nilai ringkasan pada kepala popup.
//
// Nama field-nya mengikuti LABEL yang dibaca pengguna, bukan nama properti Pega — dan itu
// disengaja: `TempDetail.Keterangan` tidak memberi tahu siapa pun bahwa isinya catatan PIC.
type DetailHeaderDTO struct {
	ClaimNumber  string `json:"no_klaim"`
	BusinessName string `json:"cob"`

	Occupation string `json:"occupation"`

	// TotalSumInsured dikirim sebagai TEKS desimal, bukan angka.
	//
	// Angka JSON dibaca JavaScript sebagai float64, dan nilai uang tidak pernah float
	// (`I-12`). Layar memformatnya dengan formatRupiah, sama seperti kolom uang lain.
	TotalSumInsured string `json:"total_sum_insured"`

	Chronology string `json:"kronologi"`

	// EstimationValue dikirim sebagai teks, dengan alasan yang sama.
	EstimationValue string `json:"total_reserve"`

	AgingDays int `json:"aging_hari"`

	// NeedsAttention menyatakan umurnya melewati ambang, sehingga popup dapat menandainya
	// dengan cara yang sama dengan barisnya di grid.
	NeedsAttention bool `json:"perlu_perhatian"`

	DominantFactors      string `json:"dominant_factor"`
	RemarkRecommendation string `json:"claim_recommendation"`
	ProgressNote         string `json:"note_pic"`
}

// DetailObjectDTO adalah satu baris grid objek pertanggungan.
//
// SELURUH kolom ketiga varian dikirim. Yang memilih varian di Pega adalah empat when rule yang
// tidak ada di export (`R-16`), sehingga pemilihannya dikerjakan layar dari lini bisnis —
// keputusan yang dapat diperbaiki tanpa menyentuh backend begitu keempatnya tiba.
type DetailObjectDTO struct {
	Name              string `json:"nama"`
	Location          string `json:"lokasi"`
	Job               string `json:"pekerjaan"`
	DateOfBirth       string `json:"tanggal_lahir"`
	IDCard            string `json:"ktp_paspor"`
	ParticipantStatus string `json:"status_peserta"`
}

// DetailProgressDTO adalah satu baris grid riwayat progres.
type DetailProgressDTO struct {
	RecordedAt     string `json:"tanggal_input"`
	ClaimNumber    string `json:"no_klaim"`
	Status1        string `json:"status_progres_1"`
	Status2        string `json:"status_progres_2"`
	EnteredBy      string `json:"user_input"`
	NextFollowUpAt string `json:"tanggal_next_followup"`
	Status         string `json:"status"`
	Note           string `json:"keterangan"`
}

// DetailMessageDTO adalah satu baris grid komunikasi dengan loss adjuster.
type DetailMessageDTO struct {
	SenderName string `json:"nama_user"`
	SentAt     string `json:"tanggal_proses"`
	Message    string `json:"pesan"`
	RepliedAt  string `json:"tanggal_balas"`
	Reply      string `json:"jawaban"`

	// Internal menyatakan pengirimnya petugas teknis, bukan adjuster luar.
	//
	// Ia tidak digambar sebagai kolom, tetapi layar memakainya untuk menempatkan pesan di
	// sisi yang benar — dan tanpa itu percakapan terbaca seolah satu pihak saja.
	Internal bool `json:"internal"`
}

// toDetailResponse menyusun jawaban popup.
func toDetailResponse(detailed usecase.Detailed, portalAlias string) DetailResponse {
	detail := detailed.Detail

	objects := make([]DetailObjectDTO, 0, len(detail.Objects))
	for _, o := range detail.Objects {
		objects = append(objects, DetailObjectDTO{
			Name:              o.Name,
			Location:          o.Location,
			Job:               o.Job,
			DateOfBirth:       isoDate(o.DateOfBirth),
			IDCard:            o.IDCard,
			ParticipantStatus: o.ParticipantStatus,
		})
	}

	progress := make([]DetailProgressDTO, 0, len(detail.ProgressHistory))
	for _, p := range detail.ProgressHistory {
		progress = append(progress, DetailProgressDTO{
			RecordedAt:     isoDateTime(p.RecordedAt),
			ClaimNumber:    p.ClaimNumber,
			Status1:        p.Status1,
			Status2:        p.Status2,
			EnteredBy:      p.EnteredBy,
			NextFollowUpAt: isoDate(p.NextFollowUpAt),
			Status:         p.Status,
			Note:           p.Note,
		})
	}

	messages := make([]DetailMessageDTO, 0, len(detail.AdjusterMessages))
	for _, m := range detail.AdjusterMessages {
		messages = append(messages, DetailMessageDTO{
			SenderName: m.SenderName,
			SentAt:     isoDateTime(m.SentAt),
			Message:    m.Message,
			RepliedAt:  isoDateTime(m.RepliedAt),
			Reply:      m.Reply,
			Internal:   m.Internal,
		})
	}

	return DetailResponse{
		Header: DetailHeaderDTO{
			ClaimNumber:          detail.ClaimNumber,
			BusinessName:         detail.BusinessName,
			Occupation:           detail.Occupation,
			TotalSumInsured:      detail.TotalSumInsured.String(),
			Chronology:           detail.Chronology,
			EstimationValue:      detail.EstimationValue.String(),
			AgingDays:            detail.AgingDays,
			NeedsAttention:       detail.AgingDays > inboxosclaimpercabang.AgingThreshold,
			DominantFactors:      detail.DominantFactors,
			RemarkRecommendation: detail.RemarkRecommendation,
			ProgressNote:         detail.ProgressNote,
		},
		Objects:            objects,
		ProgressHistory:    progress,
		AdjusterMessages:   messages,
		Branch:             BranchDTO{Code: detailed.Query.Branch.Code, Name: detailed.Query.Branch.Name},
		Portal:             portalAlias,
	}
}

// isoDateTime menuliskan cap waktu WIB lengkap dengan jamnya, atau kosong bila tidak ada.
//
// Ia BERBEDA dari isoDate, dan perbedaannya disengaja: kolom "TANGGAL INPUT" pada riwayat
// progres dan "Tanggal Proses" pada komunikasi keduanya memuat jam di layar lama, dan jam itu
// yang membedakan dua catatan pada hari yang sama. Membuangnya membuat urutan barisnya tampak
// acak.
func isoDateTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return ""
	}
	return value.In(clock.ZoneWIB).Format("2006-01-02 15:04")
}
