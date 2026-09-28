package registrasi

import (
	"context"
	"sort"
	"strings"
	"time"
)

// Catatan pendamping klaim pada tahap Input Estimasi: tab Survey, Unggah Dokumen, dan
// Progress Claim & Komunikasi.
//
// Seluruhnya DIBACA dari tabel milik sistem lama (T_SURVEYORLIST, LST_TYPE_DOC_BUSINESS,
// DATA_ATTACHFILE, GCNM_PROGRESS_CLAIM, M_KOMUNIKASI_PNC). Tidak ada satu pun yang
// ditulis modul ini: penulisannya milik modul lain (survey `B-8`, dokumen `S-1`, progres)
// dan belum dibangun.

// legacyWorkPrefix adalah awalan kunci kelas Pega yang tertanam pada tabel warisan
// (`03-CURRENT-ARCHITECTURE.md` §4.1). Klaim terbitan aplikasi ini tidak memakainya
// (`D-22`), tetapi klaim warisan menyimpannya di kolom kunci.
const legacyWorkPrefix = "ASM-FW-GCNMFW-WORK "

// RecordKeys adalah kunci-kunci yang dipakai tabel warisan untuk menunjuk sebuah klaim.
//
// Tabel warisan tidak seragam: GCNM_PROGRESS_CLAIM memakai nomor klaim polos,
// T_SURVEYORLIST dan DATA_ATTACHFILE memakai nomor berawalan kelas Pega. Ketiga bentuk
// karena itu dicocokkan sekaligus, dan satu fungsi inilah satu-satunya tempat bentuk
// kunci itu diketahui (`07-MIGRATION-STRATEGY.md` §3.3).
type RecordKeys struct {
	Number   string
	ID       string
	Prefixed string
}

// Keys menyusun kunci pencocokan tabel warisan untuk klaim ini.
func (c Claim) Keys() RecordKeys {
	number := strings.TrimSpace(c.Number)
	return RecordKeys{Number: number, ID: strings.TrimSpace(c.ID), Prefixed: legacyWorkPrefix + number}
}

// SurveyTypeInternal dan SurveyTypeAdjuster adalah nilai SURVEYTYPE T_SURVEYORLIST.
const (
	SurveyTypeInternal = "1"
	SurveyTypeAdjuster = "2"
)

// Survey adalah satu baris hasil survey sebuah klaim — grid `.ClaimData.SurveyResults`
// pada `Section/ViewHasilSurvey-Section.xml`.
type Survey struct {
	// CaseID adalah nomor kasus survey (SRV-…). Percakapan dengan surveyor menempel ke
	// nomor ini, bukan ke nomor klaim.
	CaseID         string
	Type           string
	SurveyorName   string
	Date           time.Time
	SurveyLocation string
	ObjectName     string
	ObjectLocation string
	Index          string
	Status         string
	Note           string
	InputDate      time.Time
}

// DocumentType adalah satu jenis dokumen yang dapat diunggah untuk sebuah lini bisnis —
// baris LST_TYPE_DOC_BUSINESS.
type DocumentType struct {
	// Category adalah TYPE_DOCUMENT jenis induknya: REGISTER, SURVEY, COMMITEE, PAYMENT,
	// COLLECTING DOCUMENT, SALVAGE.
	Category    string
	CategoryID  string
	ID          string // DOC_TYPE_DT_ID
	Name        string // DETAIL_DOKUMEN
	RequiredRaw string // STS_WAJIB apa adanya
	ObjectDocID string // OBJECT_DOC_ID
	MinDoc      string

	// Coverage adalah kode coverage yang mewajibkan dokumen ini pada lini Personal
	// Accident (COVERAGE_DOC_BUSINESS).
	Coverage []string
}

// Attachment adalah satu berkas yang sudah diunggah — baris DATA_ATTACHFILE.
type Attachment struct {
	ID          string
	Name        string
	MimeType    string
	Note        string
	Category    string
	SubCategory string
	ImageID     string
	UploadedBy  string
	UploadedAt  time.Time
}

// ProgressEntry adalah satu catatan progres klaim — baris GCNM_PROGRESS_CLAIM.
type ProgressEntry struct {
	Seq          int
	InputAt      time.Time
	Status1      string
	Status1Name  string
	Status2      string
	Status2Name  string
	Note         string
	NextFollowUp time.Time
	InputBy      string
	PositionID   string
}

// Communication adalah satu percakapan pada klaim — baris M_KOMUNIKASI_PNC.
type Communication struct {
	CaseID      string
	ID          string
	SentAt      time.Time
	Sender      string
	SenderName  string
	Message     string
	Reply       string
	ReplierName string
	RepliedAt   time.Time
	Status      string
}

// ClaimRecordSource membaca catatan pendamping klaim dari tabel warisan.
type ClaimRecordSource interface {
	Surveys(ctx context.Context, keys RecordKeys) ([]Survey, error)
	DocumentTypes(ctx context.Context, businessCode string) ([]DocumentType, error)
	Attachments(ctx context.Context, keys RecordKeys) ([]Attachment, error)
	Progress(ctx context.Context, keys RecordKeys) ([]ProgressEntry, error)

	// Communications membaca percakapan yang menempel ke kasus-kasus survey klaim, atau
	// yang CASECLAIM-nya menunjuk klaim itu sendiri.
	Communications(ctx context.Context, keys RecordKeys) ([]Communication, error)
}

// DocumentCategories adalah urutan kategori tab Unggah Dokumen, mengikuti
// `Section/ViewUploadDocument-Section.xml`: TempRegister, TempSurvey, TempCommitee,
// TempPayment, TempCollectingDoc. SALVAGE punya layarnya sendiri dan tidak tampil di sini.
var DocumentCategories = []struct{ Code, Label string }{
	{"REGISTER", "Register"},
	{"SURVEY", "Survey"},
	{"COMMITEE", "Committee"},
	{"PAYMENT", "Payment"},
	{"COLLECTING DOCUMENT", "Collecting Document"},
}

// DocumentRow adalah satu baris checklist dokumen.
type DocumentRow struct {
	Type     DocumentType
	Required bool
	Uploaded int
}

// DocumentCategory adalah satu kelompok checklist dokumen.
type DocumentCategory struct {
	Code  string
	Label string
	Row   []DocumentRow
}

// DocumentChecklist menyusun checklist tab Unggah Dokumen — `RequiredDocument_act`
// ditambah `SetCountAttach_act`.
//
// # Kapan sebuah dokumen wajib
//
// STS_WAJIB '1' saja tidak cukup. Di luar Personal Accident (`BrowseRegister_upload`
// dan saudaranya), dokumen wajib hanya bila OBJECT_DOC_ID-nya termasuk kelompok item
// (PropertyItemGroup) klaim. Item yang namanya memuat OTHERS dan tidak berkelompok
// dianggap berkelompok OTHERS. Kelompok kosong tidak pernah cocok: kueri lama
// menyisipkan string kosong (dua tanda kutip tunggal), yang di Oracle adalah NULL.
//
// Pada Personal Accident (`RequiredDocPA` → `BrowseRegisterCvg`), dokumen wajib bila
// salah satu coverage klaim tercatat di COVERAGE_DOC_BUSINESS dokumen itu.
//
// # Jumlah yang sudah diunggah
//
// `CountUpload` membaca DATA_ATTACHFILE yang IMAGEID-nya terisi, lalu `SetCountAttach_act`
// mencocokkan SUB_CATEGORY-nya dengan DOC_TYPE_DT_ID baris checklist.
func DocumentChecklist(claim Claim, types []DocumentType, files []Attachment) []DocumentCategory {
	uploaded := map[string]int{}
	for _, f := range files {
		if strings.TrimSpace(f.ImageID) == "" {
			continue
		}
		uploaded[strings.TrimSpace(f.SubCategory)]++
	}

	pa := claim.Policy.Line == LinePersonalAccident
	groups := itemGroups(claim)
	coverages := map[string]bool{}
	for _, o := range claim.InsuredItem {
		for _, c := range o.Coverage {
			if id := strings.TrimSpace(c.ID); id != "" {
				coverages[id] = true
			}
		}
	}

	byCategory := map[string][]DocumentRow{}
	for _, t := range types {
		required := false
		if strings.TrimSpace(t.RequiredRaw) == "1" {
			if pa {
				for _, c := range t.Coverage {
					if coverages[strings.TrimSpace(c)] {
						required = true
						break
					}
				}
			} else {
				required = groups[strings.TrimSpace(t.ObjectDocID)]
			}
		}
		code := strings.ToUpper(strings.TrimSpace(t.Category))
		byCategory[code] = append(byCategory[code], DocumentRow{
			Type: t, Required: required, Uploaded: uploaded[strings.TrimSpace(t.ID)],
		})
	}

	result := make([]DocumentCategory, 0, len(DocumentCategories))
	for _, c := range DocumentCategories {
		rows := byCategory[c.Code]
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].Type.Name < rows[j].Type.Name })
		result = append(result, DocumentCategory{Code: c.Code, Label: c.Label, Row: rows})
	}
	return result
}

// itemGroups mengumpulkan kelompok item seluruh coverage klaim.
func itemGroups(claim Claim) map[string]bool {
	groups := map[string]bool{}
	for _, o := range claim.InsuredItem {
		for _, c := range o.Coverage {
			for _, it := range c.Item {
				group := strings.TrimSpace(it.Group)
				if group == "" && strings.Contains(strings.ToUpper(it.Name), "OTHERS") {
					group = "OTHERS"
				}
				if group != "" {
					groups[group] = true
				}
			}
		}
	}
	return groups
}
