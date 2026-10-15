package memory

import "claim-pnc/internal/mastertipesurveyors"

// SampleList adalah keempat tipe surveyor nyata sesuai isi POOLDATA.V_M_SURVEYORS.
//
// Nilainya DIBACA LANGSUNG dari basis data portal ASM pada 2026-09-19 lewat perkakas
// diagnostik baca-saja, bukan dikarang dan bukan disalin dari dokumen. Kuerinya dicatat di
// `docs/catatan-pengembangan.md` supaya dapat diulang.
//
// # Tiga hal yang terbaca dari daftar ini dan penting untuk tidak dilupakan
//
//  1. Kodenya BERURUTAN `1001`–`1004` tanpa satu pun lompatan, karena ia dibentuk
//     `id_site || lpad(urutan, 3, '0')` — situs `1`, urutan 1 sampai 4.
//
//  2. LegacyCode KOSONG pada keempatnya. Kolom OLD_M_SURVEY_ID memang tidak pernah diisi
//     di master ini, berbeda dari Master Status Klaim yang sebelas kode pertamanya
//     membawa penomoran lama.
//
//  3. Ketiga kode selain `1001` DIPATOK LANGSUNG di kueri Pega, dan itu sebabnya kode
//     tidak boleh berubah:
//
//     1002  LOSS ADJUSTER  RDB List/BrowseSurveyorTypeLossAdjuster-SQL.xml:  in ('1002')
//     1003  EXPERT         RDB List/BrowseSurveyorTypeExpert-SQL.xml:        in ('1003')
//     1004  SURVEY AGENT   RDB List/BrowseSurveyorTypeSurveyAgent-SQL.xml:   in ('1004')
//
// # Kenapa seluruhnya HURUF BESAR
//
// Begitulah isinya di basis data. Ia TIDAK diseragamkan menjadi huruf besar oleh kode mana
// pun yang dapat ditemukan — `Activity/ValidasiMasterSurveyor-Act.xml` memang memanggil
// `@toUpperCase`, tetapi atas `TempDetailSurveyors.NAME`, yaitu nama SURVEYOR di
// D_SURVEYORS, bukan deskripsi tipe di sini.
//
// Karena itu modul ini TIDAK mengubah besar-kecil huruf yang diketik pengguna: menambah
// perilaku yang tidak pernah ada berarti mengarang. Yang diseragamkan hanyalah
// PERBANDINGAN saat menguji keunikan (mastertipesurveyors.DescriptionKey).
//
// # Jumlah surveyor yang bergantung pada tiap tipe, per 2026-09-19
//
//	1001  19 surveyor      1002  18 surveyor      1004  6 surveyor      1003  belum ada
//
// Angka itu yang membuat "menghapus satu tipe" bukan sekadar menghilangkan satu baris.
func SampleList() []mastertipesurveyors.SurveyorType {
	return []mastertipesurveyors.SurveyorType{
		{Code: "1001", Description: "INTERNAL SURVEYOR"},
		{Code: "1002", Description: "LOSS ADJUSTER"},
		{Code: "1003", Description: "EXPERT"},
		{Code: "1004", Description: "SURVEY AGENT"},
	}
}
