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

	Kelompok []GroupDTO `json:"kelompok"`

	// LiniBisnis adalah isi dropdown yang di layar berlabel "Treaty".
	LiniBisnis []BusinessLineDTO `json:"lini_bisnis"`
}

// GroupDTO adalah satu kelompok kartu di layar.
type GroupDTO struct {
	Kode    string      `json:"kode"`
	Judul   string      `json:"judul"`
	Laporan []ReportDTO `json:"laporan"`
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
// Urutan kelompoknya TETAP — ia ditulis di sini, bukan diambil dari peta, supaya susunan
// layar tidak berubah-ubah antar permintaan.
func toCatalogDTO(reports []reportklaim.Report) CatalogResponse {
	order := []reportklaim.Group{
		reportklaim.GroupKlaim,
		reportklaim.GroupReasuransi,
		reportklaim.GroupPenyelesaian,
		reportklaim.GroupLiniBisnis,
		reportklaim.GroupOperasional,
	}

	byGroup := make(map[reportklaim.Group][]ReportDTO, len(order))
	for _, r := range reports {
		byGroup[r.Group] = append(byGroup[r.Group], toReportDTO(r))
	}

	groups := make([]GroupDTO, 0, len(order))
	for _, g := range order {
		item := byGroup[g]
		if len(item) == 0 {
			continue
		}
		groups = append(groups, GroupDTO{
			Kode:    string(g),
			Judul:   reportklaim.GroupLabel(g),
			Laporan: item,
		})
	}

	lines := reportklaim.BusinessLineOptions()
	lineDTO := make([]BusinessLineDTO, 0, len(lines))
	for _, l := range lines {
		lineDTO = append(lineDTO, BusinessLineDTO{Nilai: string(l.Value), Nama: l.Name})
	}

	return CatalogResponse{
		// Disalin apa adanya dari harness. Judulnya memang berbahasa Inggris di sistem
		// lama, dan `D-13` menetapkan teks layar ditiru (`D-80`).
		Judul:      "Report Claim",
		Kelompok:   groups,
		LiniBisnis: lineDTO,
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
