package memory

import (
	"context"
	"sort"
	"strings"
	"sync"

	"claim-pnc/internal/registrasi"
)

// PUCL adalah penyimpan surat RCL/PUCL di memori — adapter kedua seam
// `PUCLLetterStore` dan `PUCLOptionSource`, yang membuat aturan modal "Kirim ke
// RCL/PUCL" dapat diuji tanpa Oracle sama sekali.
type PUCL struct {
	mu sync.Mutex

	// Letter adalah surat yang tersimpan, SATU per klaim. Uji membacanya langsung
	// untuk memeriksa apa yang benar-benar ditulis.
	Letter []registrasi.PUCLLetter

	// Subject, Reason, dan Doctor adalah data contoh ketiga daftar pilihan.
	Subject []registrasi.PUCLSubjectOption
	Reason  []registrasi.PUCLRejectReason
	Doctor  []registrasi.RCLDoctorOption
}

var (
	_ registrasi.PUCLLetterStore  = (*PUCL)(nil)
	_ registrasi.PUCLOptionSource = (*PUCL)(nil)
)

// NewPUCL membentuk penyimpan surat RCL/PUCL di memori beserta dua pilihan contoh.
//
// Isi contohnya meniru pembelahan master yang sebenarnya: Perihal jalur RCL berbunyi
// "Tolakan…", jalur PUCL berbunyi "Kelengkapan…". Data KARANGAN (`D-69`).
func NewPUCL() *PUCL {
	return &PUCL{
		Subject: []registrasi.PUCLSubjectOption{
			{ID: 1, Name: "Kelengkapan Data dan Dokumen Klaim", Track: registrasi.PUCLTrackPUCL},
			{ID: 2, Name: "Pemberitahuan Penundaan Proses Klaim", Track: registrasi.PUCLTrackPUCL},
			{ID: 9, Name: "Tolakan klaim polis", Track: registrasi.PUCLTrackRCL},
		},
		Reason: []registrasi.PUCLRejectReason{
			{ID: "001", Name: "Dikecualikan polis", Description: "Perawatan yang Anda ajukan dikecualikan dalam polis."},
			{ID: "002", Name: "Masa tunggu belum terlampaui", Description: "Pengajuan berada di dalam masa tunggu polis."},
		},
		// Dua identitas lama contoh — bentuknya meniru `OLD_OPERATOR_ID` yang
		// sebenarnya (huruf besar, boleh berspasi). Data KARANGAN (`D-69`).
		Doctor: []registrasi.RCLDoctorOption{
			{ID: "DOKTERCONTOHSATU"},
			{ID: "DOKTER CONTOH DUA"},
		},
	}
}

// SaveLetter menyimpan satu surat, MENIMPA surat klaim yang sama bila sudah ada.
//
// Perilakunya sengaja sama dengan adapter Oracle-nya: primary key `TC_PNC_PUCL_PK`
// adalah `CLAIMID` tunggal, sehingga satu klaim hanya punya satu surat. Adapter memori
// yang menumpuk baris akan membuat uji lulus pada keadaan yang di Oracle justru ditolak
// ORA-00001 — kepercayaan palsu yang justru paling mahal.
func (p *PUCL) SaveLetter(_ context.Context, letter registrasi.PUCLLetter) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i, existing := range p.Letter {
		if existing.ClaimID == letter.ClaimID {
			p.Letter[i] = letter
			return nil
		}
	}
	p.Letter = append(p.Letter, letter)
	return nil
}

// SubjectOptions mengembalikan pilihan Perihal jalur itu.
//
// Notification dan jalur yang tidak dikenal mengembalikan SELURUHNYA — perilaku yang
// sama dengan adapter Oracle-nya, karena master Perihal tidak punya kelompok untuk
// Notification.
func (p *PUCL) SubjectOptions(_ context.Context, track int) ([]registrasi.PUCLSubjectOption, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	known := track == registrasi.PUCLTrackRCL || track == registrasi.PUCLTrackPUCL
	result := []registrasi.PUCLSubjectOption{}
	for _, option := range p.Subject {
		if !known || option.Track == track {
			result = append(result, option)
		}
	}
	return result, nil
}

// RCLDoctors menyerahkan salinan pilihan "Nama Dokter", terurut menaik.
//
// Terurut, bukan apa adanya: adapter Oracle-nya mengurutkan di basis data, dan adapter
// memori yang mengembalikan urutan sisip membuat uji lulus pada urutan yang tidak pernah
// terjadi di produksi.
func (p *PUCL) RCLDoctors(_ context.Context) ([]registrasi.RCLDoctorOption, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	result := make([]registrasi.RCLDoctorOption, len(p.Doctor))
	copy(result, p.Doctor)
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

// RejectReasons mencari alasan penolakan menurut nama atau kodenya.
func (p *PUCL) RejectReasons(_ context.Context, keyword string, limit int) ([]registrasi.PUCLRejectReason, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	needle := strings.ToUpper(strings.TrimSpace(keyword))
	result := []registrasi.PUCLRejectReason{}
	for _, reason := range p.Reason {
		if limit > 0 && len(result) >= limit {
			break
		}
		if needle != "" &&
			!strings.Contains(strings.ToUpper(reason.Name), needle) &&
			!strings.Contains(strings.ToUpper(reason.ID), needle) {
			continue
		}
		result = append(result, reason)
	}
	return result, nil
}
