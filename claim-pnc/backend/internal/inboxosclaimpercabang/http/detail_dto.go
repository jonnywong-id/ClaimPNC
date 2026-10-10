package inboxosclaimpercabanghttp

import (
	"fmt"
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
	// ID adalah kunci baris. Ia tidak digambar, tetapi layar membutuhkannya untuk
	// menandai baris MANA yang sedang terbuka — tanpa kunci yang stabil, panel yang
	// terbuka berpindah ketika daftar diurutkan ulang.
	ID string `json:"id"`

	Name              string `json:"nama"`
	Location          string `json:"lokasi"`
	Job               string `json:"pekerjaan"`
	DateOfBirth       string `json:"tanggal_lahir"`
	IDCard            string `json:"ktp_paspor"`
	ParticipantStatus string `json:"status_peserta"`

	// Coverages adalah isi panel yang terbuka ketika baris ini dibuka.
	//
	// Dikirim bersama objeknya, bukan lewat permintaan terpisah per baris: layar ini
	// dirancang untuk dibuka-tutup berkali-kali, dan satu permintaan per pembukaan akan
	// terasa lambat tanpa alasan.
	Coverages []DetailCoverageDTO `json:"coverage"`
}

// DetailCoverageDTO adalah satu baris coverage di bawah sebuah objek.
//
// Ketiga kolomnya persis yang digambar `Section/ViewObjectCoverage-Section.xml`, berjudul
// "Coverage", "Mata Uang", dan "TSI". Tidak lebih: menambah kolom di luar itu adalah
// penambahan, bukan penyamaan dengan sistem lama (`P-5`).
type DetailCoverageDTO struct {
	ID string `json:"id"`

	// Name adalah isi kolom "Coverage" — nama jaminan, mis. "FLEXAS".
	Name string `json:"coverage"`

	Currency string `json:"mata_uang"`

	// SumTSI dikirim sebagai TEKS desimal kanonik, sama seperti setiap nilai uang lain di
	// modul ini. Angka JSON akan melewati float64 dan kehilangan ketepatan (`I-12`).
	SumTSI string `json:"tsi"`

	// Items adalah isi grid "Object Item" yang terbuka saat baris ini dibuka.
	Items []DetailItemDTO `json:"object_item"`

	// Spreadings dan CoMembers adalah kedua grid di bawah "Object Item".
	Spreadings []DetailSpreadingDTO `json:"list_spreading"`
	CoMembers  []DetailCoMemberDTO  `json:"co_member"`
}

// DetailSpreadingDTO adalah satu baris grid "List Spreading".
type DetailSpreadingDTO struct {
	// TreatyName adalah kolom "Tipe Treaty" — `ORS`, `QS`, `FAC-OUT`, dan seterusnya.
	TreatyName string `json:"tipe_treaty"`
	Currency   string `json:"currency"`

	// EstimasiValue dan ResultValue teks desimal kanonik; keduanya DIHITUNG, bukan
	// tersimpan. Lihat inboxosclaimpercabang.DetailSpreading.
	EstimasiValue string `json:"estimasi_value"`

	// SharePersen dikirim sebagai teks berdesimal EMPAT, mengikuti layar lama yang
	// menuliskannya `100,0000%`.
	SharePersen string `json:"pembagian_persentase"`
	ResultValue string `json:"result_value"`
}

// DetailCoMemberDTO adalah satu baris grid "CO MEMBER".
type DetailCoMemberDTO struct {
	InsurerName   string `json:"asuransi"`
	Currency      string `json:"currency"`
	EstimasiValue string `json:"estimasi_value"`
	SharePersen   string `json:"pembagian_persentase"`
	ResultValue   string `json:"result_value"`
}

// persenEmpatDesimal memformat persentase berskala 10.000 menjadi teks berdesimal empat.
//
// `1000000` menjadi `"100.0000"`. Pembagiannya BILANGAN BULAT — menempuh float untuk sesuatu
// yang sudah berupa bilangan bulat berskala hanya menambah kemungkinan galat pembulatan.
func persenEmpatDesimal(scaled int64) string {
	tanda := ""
	if scaled < 0 {
		tanda = "-"
		scaled = -scaled
	}
	return fmt.Sprintf("%s%d.%04d", tanda, scaled/10000, scaled%10000)
}

// DetailItemDTO adalah satu baris grid "Object Item".
//
// Nama dan deskripsinya hampir selalu kosong, dan itu BUKAN cacat — sumbernya memang tidak
// memuatnya. Lihat inboxosclaimpercabang.DetailItem.
type DetailItemDTO struct {
	ID          string `json:"id"`
	Name        string `json:"object_item"`
	Description string `json:"deskripsi_item"`

	// Estimations adalah isi grid "Estimasi" yang terbuka saat baris ini dibuka.
	Estimations []DetailEstimationDTO `json:"estimasi"`
}

// DetailEstimationDTO adalah satu baris grid "Estimasi" — tingkat terdalam.
type DetailEstimationDTO struct {
	Sequence   string `json:"estimasi_ke"`
	RecordedAt string `json:"tanggal_estimasi"`
	Type       string `json:"tipe_estimasi"`
	Currency   string `json:"mata_uang"`

	// Rate dan Value dikirim sebagai TEKS desimal kanonik. Value DAPAT negatif.
	Rate  string `json:"nilai_kurs"`
	Value string `json:"nilai_estimasi"`
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
		// Irisan kosong, bukan nil: ia diserialkan menjadi `[]` alih-alih `null`, sehingga
		// layar tidak perlu membedakan "tidak punya coverage" dari "field tidak ada".
		coverages := make([]DetailCoverageDTO, 0, len(o.Coverages))
		for _, c := range o.Coverages {
			items := make([]DetailItemDTO, 0, len(c.Items))
			for _, it := range c.Items {
				estimations := make([]DetailEstimationDTO, 0, len(it.Estimations))
				for _, e := range it.Estimations {
					estimations = append(estimations, DetailEstimationDTO{
						Sequence:   e.Sequence,
						RecordedAt: isoDateTime(e.RecordedAt),
						Type:       e.Type,
						Currency:   e.Currency,
						Rate:       e.Rate.String(),
						Value:      e.Value.String(),
					})
				}
				items = append(items, DetailItemDTO{
					ID:          it.ID,
					Name:        it.Name,
					Description: it.Description,
					Estimations: estimations,
				})
			}

			spreadings := make([]DetailSpreadingDTO, 0, len(c.Spreadings))
			for _, s := range c.Spreadings {
				spreadings = append(spreadings, DetailSpreadingDTO{
					TreatyName:    s.TreatyName,
					Currency:      s.Currency,
					EstimasiValue: s.EstimationValue.String(),
					SharePersen:   persenEmpatDesimal(s.SharePercentScaled),
					ResultValue:   s.ResultValue.String(),
				})
			}

			coMembers := make([]DetailCoMemberDTO, 0, len(c.CoMembers))
			for _, m := range c.CoMembers {
				coMembers = append(coMembers, DetailCoMemberDTO{
					InsurerName:   m.InsurerName,
					Currency:      m.Currency,
					EstimasiValue: m.EstimationValue.String(),
					SharePersen:   persenEmpatDesimal(m.SharePercentScaled),
					ResultValue:   m.ResultValue.String(),
				})
			}

			coverages = append(coverages, DetailCoverageDTO{
				ID:         c.ID,
				Name:       c.Name,
				Currency:   c.Currency,
				SumTSI:     c.SumTSI.String(),
				Items:      items,
				Spreadings: spreadings,
				CoMembers:  coMembers,
			})
		}

		objects = append(objects, DetailObjectDTO{
			ID:                o.ID,
			Name:              o.Name,
			Location:          o.Location,
			Job:               o.Job,
			DateOfBirth:       isoDate(o.DateOfBirth),
			IDCard:            o.IDCard,
			ParticipantStatus: o.ParticipantStatus,
			Coverages:         coverages,
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
