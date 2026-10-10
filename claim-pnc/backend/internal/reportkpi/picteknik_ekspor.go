package reportkpi

// PICExportKind adalah pilihan "Pilih Data KPI" pada tab KPI PIC Teknik.
//
// # Nilainya angka, dan itu bukan pilihan kami
//
// Pega menyimpannya sebagai "1".."4" pada `TempLaporan.City`, dan menyusun daftarnya di
// `Activity/EksportDataKPIProgressKlaim-Act.xml` berpasangan `FlagASO` → `NoKTP`. Nilai
// dan labelnya dibawa apa adanya supaya berkas yang diunduh dapat ditumpuk dengan berkas
// Pega pada masa paralel.
type PICExportKind string

// Keempat pilihan, dalam urutan yang sama dengan layar lama.
const (
	PICExportProgress   PICExportKind = "1"
	PICExportSLA        PICExportKind = "2"
	PICExportAcceptance PICExportKind = "3"
	PICExportAnalysis   PICExportKind = "4"
)

// PICExportDefault adalah pilihan bawaan.
//
// Pega menyetelnya lewat `@if(TempLaporan.City==0,"1",TempLaporan.City)` — kosong berarti
// "1". Permintaan tanpa `data_kpi` karena itu DITERIMA, bukan ditolak.
const PICExportDefault = PICExportProgress

// PICExportOption adalah satu baris dropdown.
type PICExportOption struct {
	Code  PICExportKind
	Label string
}

var picExportOptions = []PICExportOption{
	{Code: PICExportProgress, Label: "Export KPI Progress"},
	{Code: PICExportSLA, Label: "Export KPI SLA"},
	{Code: PICExportAcceptance, Label: "Export KPI Akseptasi"},
	{Code: PICExportAnalysis, Label: "Export KPI Analisis"},
}

// PICExportOptions mengembalikan salinan daftar pilihan.
func PICExportOptions() []PICExportOption {
	out := make([]PICExportOption, len(picExportOptions))
	copy(out, picExportOptions)
	return out
}

// FindPICExportKind menerjemahkan nilai dari permintaan.
//
// Kosong berarti bawaan — lihat PICExportDefault. Nilai yang TIDAK dikenal ditolak, bukan
// dijatuhkan ke bawaan: permintaan `data_kpi=9` berarti pemanggil mengira ada pilihan
// kelima, dan menjawabnya dengan berkas pilihan pertama menyembunyikan kekeliruan itu di
// balik berkas yang tampak wajar.
func FindPICExportKind(raw string) (PICExportKind, bool) {
	if raw == "" {
		return PICExportDefault, true
	}
	for _, option := range picExportOptions {
		if string(option.Code) == raw {
			return option.Code, true
		}
	}
	return "", false
}

// QueryName memetakan pilihan ke nama kueri di berkas .sql.
func (k PICExportKind) QueryName() string {
	switch k {
	case PICExportSLA:
		return "ekspor_pic_sla"
	case PICExportAcceptance:
		return "ekspor_pic_akseptasi"
	case PICExportAnalysis:
		return "ekspor_pic_analisis"
	default:
		return "ekspor_pic_progress"
	}
}

// Label mengembalikan nama yang dilihat pengguna.
func (k PICExportKind) Label() string {
	for _, option := range picExportOptions {
		if option.Code == k {
			return option.Label
		}
	}
	return string(k)
}

// PICExportTable adalah hasil satu ekspor: judul kolom beserta barisnya.
//
// Judul kolom datang dari HASIL KUERI, bukan dari daftar di dalam kode. Daftar kedua akan
// menyimpang diam-diam dari berkas .sql begitu satu kolom ditambahkan — dan karena judul
// inilah yang menjadi kepala berkas CSV yang dibandingkan dengan Pega, penyimpangannya
// baru terlihat saat seseorang membandingkan dua berkas.
type PICExportTable struct {
	Header []string
	Rows   [][]string
}
