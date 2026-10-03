package memory

import (
	"context"
	"strconv"
	"strings"

	"claim-pnc/internal/registrasi"
)

// ClaimRecords adalah catatan pendamping klaim di memori, untuk pengujian dan pengembangan
// lokal. Isinya dikunci dengan nomor klaim; jenis dokumen berlaku untuk seluruh lini.
type ClaimRecords struct {
	Survey        map[string][]registrasi.Survey
	DocumentType  []registrasi.DocumentType
	Attachment    map[string][]registrasi.Attachment
	ProgressEntry map[string][]registrasi.ProgressEntry
	Communication map[string][]registrasi.Communication
}

// SampleClaimRecords mengembalikan jenis dokumen contoh untuk setiap kategori tab.
func SampleClaimRecords() *ClaimRecords {
	return &ClaimRecords{
		DocumentType: []registrasi.DocumentType{
			{Category: "REGISTER", CategoryID: "10064", ID: "14901", Name: "PELAPORAN KLAIM", RequiredRaw: "1", ObjectDocID: "OTHERS", MinDoc: "1"},
			{Category: "REGISTER", CategoryID: "10064", ID: "14902", Name: "FOTO KERUSAKAN", RequiredRaw: "0", MinDoc: "0"},
			{Category: "SURVEY", CategoryID: "10065", ID: "14903", Name: "LAPORAN SURVEY", RequiredRaw: "1", ObjectDocID: "BUILDING(S)", MinDoc: "1"},
			{Category: "COLLECTING DOCUMENT", CategoryID: "10066", ID: "14893", Name: "FORMULIR KLAIM", RequiredRaw: "1", ObjectDocID: "OTHERS", MinDoc: "1"},
			{Category: "PAYMENT", CategoryID: "10067", ID: "14904", Name: "KWITANSI", RequiredRaw: "0", MinDoc: "0"},
			{Category: "COMMITEE", CategoryID: "10068", ID: "14905", Name: "NOTA KOMITE", RequiredRaw: "0", MinDoc: "0"},
		},
	}
}

// Surveys mengembalikan hasil survey klaim.
func (r *ClaimRecords) Surveys(_ context.Context, keys registrasi.RecordKeys) ([]registrasi.Survey, error) {
	return r.Survey[keys.Number], nil
}

// DocumentTypes mengembalikan jenis dokumen contoh.
func (r *ClaimRecords) DocumentTypes(_ context.Context, businessCode string) ([]registrasi.DocumentType, error) {
	if businessCode == "" {
		return nil, nil
	}
	return r.DocumentType, nil
}

// Attachments mengembalikan berkas klaim.
func (r *ClaimRecords) Attachments(_ context.Context, keys registrasi.RecordKeys) ([]registrasi.Attachment, error) {
	return r.Attachment[keys.Number], nil
}

// Progress mengembalikan riwayat progres klaim.
func (r *ClaimRecords) Progress(_ context.Context, keys registrasi.RecordKeys) ([]registrasi.ProgressEntry, error) {
	return r.ProgressEntry[keys.Number], nil
}

// Communications mengembalikan percakapan klaim.
func (r *ClaimRecords) Communications(_ context.Context, keys registrasi.RecordKeys) ([]registrasi.Communication, error) {
	return r.Communication[keys.Number], nil
}

// AddCommunication menambahkan pesan di depan daftar — terbaru lebih dulu, seperti kueri SQL.
func (r *ClaimRecords) AddCommunication(_ context.Context, c registrasi.NewCommunication) error {
	if r.Communication == nil {
		r.Communication = map[string][]registrasi.Communication{}
	}
	key := strings.TrimSpace(c.ClaimNumber)
	r.Communication[key] = append([]registrasi.Communication{{
		CaseID: c.ClaimID, ID: strconv.Itoa(len(r.Communication[key]) + 1), SentAt: c.At,
		Sender: c.Sender, SenderName: c.SenderName, Message: c.Message, Status: c.Status, Channel: c.Channel,
	}}, r.Communication[key]...)
	return nil
}
