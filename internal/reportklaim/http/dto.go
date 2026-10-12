package reportklaimhttp

import (
	"claim-pnc/internal/reportklaim"
)

// # Kenapa nama field JSON berbahasa Indonesia
//
// Ia KONTRAK, bukan nama internal (`D-80`). Mengubahnya adalah perubahan yang merusak
// klien, bukan penggantian nama — dan seluruh modul lain sudah memakai bentuk ini.

// CatalogResponse adalah isi layar Report Klaim sebelum satu tombol pun ditekan.
type CatalogResponse struct {
	// Judul adalah judul layar, disalin dari harness: "Report Claim".
	Judul string `json:"judul"`

	// Laporan adalah ke-28 panel dalam URUTAN LAYAR Pega — DATAR, tanpa pengelompokan.
	//
	// Sebelumnya daftar ini dikelompokkan menjadi lima bagian berjudul (Klaim, PLA/DLA,
	// dan seterusnya). Pengelompokan itu TIDAK ADA di layar lama: harness-nya menumpuk
	// ke-28 panel dalam satu kolom, berurutan. Ia dicabut 2026-10-08 atas ralat Work
	// Owner — pengelompokannya memindahkan panel dari tempat yang dihafal pengguna.
	Laporan []ReportDTO `json:"laporan"`

	// LiniBisnis adalah isi dropdown yang di layar berlabel "Bisnis".
	//
	// Namanya "Bisnis", bukan "Treaty". Yang berlabel "Treaty" di layar lama adalah
	// KOTAK CENTANG pada panel Akseptasi — kontrol yang sama sekali berbeda. Lihat
	// FilterUsageDTO.Rincian.
	LiniBisnis []BusinessLineDTO `json:"lini_bisnis"`

	// StatusCompliance adalah isi dropdown "Status Compliance".
	//
	// Dikirim bersama katalog, bukan lewat permintaan tersendiri: daftarnya tetap — ia
	// berasal dari rule Property, bukan dari data.
	StatusCompliance []ComplianceStatusDTO `json:"status_compliance"`
}

// ComplianceStatusDTO adalah satu pilihan dropdown "Status Compliance".
//
// Nilai dan labelnya BERBEDA, dan keduanya wajib dikirim. Yang disaring adalah kodenya
// ("0", "1", "2"); yang dibaca pengguna adalah labelnya. Mengirim labelnya saja akan
// membuat penyaringnya tidak pernah cocok.
type ComplianceStatusDTO struct {
	Nilai string `json:"nilai"`
	Nama  string `json:"nama"`
}

// ReportDTO adalah satu kartu laporan.
type ReportDTO struct {
	Kode  string `json:"kode"`
	Judul string `json:"judul"`

	Tombol []ActionDTO `json:"tombol"`

	// Penyaring menyebutkan isian mana yang AKTIF pada kartu ini.
	//
	// Layar memakainya untuk menonaktifkan isian yang tidak berpengaruh. Tanpa itu,
	// pengguna mengisi rentang tanggal untuk laporan yang kuerinya tidak menerima tanggal
	// sama sekali, lalu menyimpulkan hasilnya salah.
	Penyaring FilterUsageDTO `json:"penyaring"`

	// Tersedia bernilai false bila laporannya belum dapat dijalankan.
	Tersedia bool `json:"tersedia"`

	// Alasan dan Penghalang hanya terisi bila Tersedia bernilai false.
	//
	// Keduanya ikut dikirim, bukan disembunyikan: panel yang tidak dapat dijalankan tetap
	// TAMPIL bertanda sebabnya, sama seperti butir menu yang belum punya layar.
	Alasan     string `json:"alasan,omitempty"`
	Penghalang string `json:"penghalang,omitempty"`

	// Sumber adalah jejak ke rule Pega asalnya.
	//
	// Ia ikut ke layar dengan sengaja. Modul ini menyalin 28 kueri beserta ratusan kolom,
	// dan satu-satunya cara menjawab "kolom ini dari mana" adalah menunjuk berkasnya.
	Sumber SourceDTO `json:"sumber"`
}

// ActionDTO adalah satu tombol Export pada sebuah kartu.
type ActionDTO struct {
	// Kode kosong pada kartu bertombol tunggal.
	Kode  string `json:"kode"`
	Label string `json:"label"`
}

// FilterUsageDTO menyatakan isian mana yang berlaku pada satu laporan.
type FilterUsageDTO struct {
	RentangTanggal   bool `json:"rentang_tanggal"`
	LiniBisnis       bool `json:"lini_bisnis"`
	StatusCompliance bool `json:"status_compliance"`
	Bisnis           bool `json:"bisnis"`
	Rincian          bool `json:"rincian"`
}

// SourceDTO adalah jejak ke rule Pega.
type SourceDTO struct {
	Activity string   `json:"activity"`
	RuleSQL  []string `json:"rule_sql,omitempty"`
}

// BusinessLineDTO adalah satu pilihan dropdown "Treaty".
type BusinessLineDTO struct {
	Nilai string `json:"nilai"`
	Nama  string `json:"nama"`
}

// BusinessOptionDTO adalah satu pilihan autocomplete "Bisnis".
type BusinessOptionDTO struct {
	Kode string `json:"kode"`
	Nama string `json:"nama"`
}

// BusinessOptionResponse membungkus daftar pilihan bisnis.
type BusinessOptionResponse struct {
	Bisnis []BusinessOptionDTO `json:"bisnis"`
}

// toCatalogDTO menyusun isi layar dari katalog.
//
// Urutan laporannya dipakai APA ADANYA dari pemanggil — yaitu urutan layar Pega. Tidak ada
// pengurutan ulang di sini: susunan layar adalah hal yang dihafal pengguna, dan satu-satunya
// tempat yang boleh menentukannya adalah katalog.
func toCatalogDTO(reports []reportklaim.Report) CatalogResponse {
	laporan := make([]ReportDTO, 0, len(reports))
	for _, r := range reports {
		laporan = append(laporan, toReportDTO(r))
	}

	lines := reportklaim.BusinessLineOptions()
	lineDTO := make([]BusinessLineDTO, 0, len(lines))
	for _, l := range lines {
		lineDTO = append(lineDTO, BusinessLineDTO{Nilai: string(l.Value), Nama: l.Name})
	}

	statusDTO := make([]ComplianceStatusDTO, 0, len(reportklaim.ComplianceStatusOptions))
	for _, o := range reportklaim.ComplianceStatusOptions {
		statusDTO = append(statusDTO, ComplianceStatusDTO{Nilai: o.Code, Nama: o.Label})
	}

	return CatalogResponse{
		// Disalin apa adanya dari harness. Judulnya memang berbahasa Inggris di sistem
		// lama, dan `D-13` menetapkan teks layar ditiru (`D-80`).
		Judul:            "Report Claim",
		Laporan:          laporan,
		LiniBisnis:       lineDTO,
		StatusCompliance: statusDTO,
	}
}

func toReportDTO(r reportklaim.Report) ReportDTO {
	actions := make([]ActionDTO, 0, len(r.Actions))
	for _, a := range r.Actions {
		actions = append(actions, ActionDTO{Kode: a.Code, Label: a.Label})
	}

	return ReportDTO{
		Kode:   string(r.Code),
		Judul:  r.Title,
		Tombol: actions,
		Penyaring: FilterUsageDTO{
			RentangTanggal:   r.Uses.Has(reportklaim.FilterDateRange),
			LiniBisnis:       r.Uses.Has(reportklaim.FilterBusinessLine),
			StatusCompliance: r.Uses.Has(reportklaim.FilterComplianceStatus),
			Bisnis:           r.Uses.Has(reportklaim.FilterBusinessCode),
			Rincian:          r.Uses.Has(reportklaim.FilterDetail),
		},
		Tersedia:   r.Availability.Ready,
		Alasan:     r.Availability.Reason,
		Penghalang: r.Availability.Blocker,
		Sumber: SourceDTO{
			Activity: r.Source.Activity,
			RuleSQL:  r.Source.SQLRule,
		},
	}
}
