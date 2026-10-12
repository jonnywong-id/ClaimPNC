package outstandingclaim

// Bentuk layar Outstanding Claim, dibaca dari `Section/OutstandingClaim-Section.xml`.
//
// # Kenapa bentuk layar tinggal di BACKEND
//
// Karena ia hasil pembacaan export Pega, dan tempat pembacaan itu tercatat adalah di sini.
// Menyalin 97 judul isian dan 10 susunan grid ke frontend berarti daftar yang sama hidup di
// dua tempat, dan yang satu akan tertinggal saat yang lain diperbaiki.
//
// Ia juga yang membuat isian terhalang hilang dengan sendirinya begitu penghalangnya hilang —
// tanpa menyunting frontend.
//
// # Bagaimana judul dan jalurnya diperoleh
//
// Judul diambil dari `pyLabelFieldValue` tiap sel APA ADANYA (`D-13`), termasuk yang salah
// ketik dan yang huruf besarnya tidak konsisten. Jalurnya diturunkan dari `pyValue` sel yang
// sama: `.ClaimData.X` di klipboard adalah `X` di dalam `POOLDATA.JSON_KLAIM.DATA_JSONBLOB`.
//
// Susunan grid diambil dari pasangan baris kepala dan baris isi pada `pyRows` tiap grid,
// berurutan — bukan dari urutan properti di kelasnya, yang berbeda.

// Field adalah satu isian skalar pada layar rincian.
type Field struct {
	// Key adalah nama isian pada kontrak API. Ia berbahasa Indonesia atau menyalin istilah
	// Inggris dari judulnya, mengikuti pengecualian kontrak pada `D-80`.
	Key string

	// Title adalah judul yang dibaca pengguna, mengikuti Pega apa adanya (`D-13`).
	Title string

	// Path adalah jalur isian ini di dalam dokumen JSON klaim, memakai titik sebagai
	// pemisah tingkat — mis. `QuotationData.BusinessName`.
	//
	// KOSONG pada isian yang Blocked: tidak ada jalur yang dapat dibaca.
	Path string

	// Blocked menyatakan isian ini digambar tetapi belum dapat diisi.
	//
	// Ia BUKAN isian yang disembunyikan. Menyembunyikannya membuat pengguna yang
	// membandingkan layar ini dengan Pega mengira isiannya hilang; menggambarnya dengan
	// alasan membuat ia tahu isiannya ada dan kenapa kosong.
	Blocked bool
}

// GridColumn adalah satu kolom pada sebuah grid.
type GridColumn struct {
	Key   string
	Title string

	// Path adalah nama isian di dalam satu baris senarai JSON, relatif terhadap barisnya.
	Path string
}

// Grid adalah satu tabel pada layar rincian.
type Grid struct {
	// Code adalah nama grid pada kontrak API.
	Code string

	// Title adalah judul yang dibaca pengguna. Sebagian grid di Pega tidak punya judul
	// sama sekali — judul di bawah ditetapkan dari label kontainernya atau dari kolom
	// pertamanya, supaya sepuluh tabel dalam satu layar dapat dibedakan pembaca layar.
	Title string

	// Path adalah jalur SENARAI-nya di dalam dokumen JSON klaim.
	//
	// KOSONG pada grid yang Blocked.
	Path string

	Columns []GridColumn

	Blocked       bool
	BlockedReason string
	BlockedOwner  string
}

// Group adalah satu kelompok isian beserta grid yang digambar sesudahnya.
//
// Kelompoknya mengikuti pembagian kontainer di section, termasuk kelompok yang di Pega tidak
// berjudul — judulnya ditetapkan di bawah supaya layar tidak menjadi satu daftar panjang
// berisi 97 isian tanpa pembatas.
type Group struct {
	Code   string
	Title  string
	Fields []Field

	// Grids adalah kode grid yang digambar sesudah isian kelompok ini, berurutan.
	Grids []string
}

// Kode kelompok isian.
const (
	GroupTreaty      = "treaty"
	GroupClaim       = "klaim"
	GroupInterest    = "interest"
	GroupDeductible  = "deductible"
	GroupClaimAmount = "nilai_klaim"
	GroupResultClaim = "hasil_klaim"
	GroupEstimation  = "estimasi"
	GroupSpreading   = "spreading"
	GroupAttachment  = "lampiran"
	GroupSuggestion  = "saran"
)

// Kode grid.
const (
	GridInterest        = "interest_list"
	GridInterestTotal   = "interest_total"
	GridClaimAmount     = "claim_amount"
	GridSpreadingRisk   = "spreading_risk"
	GridEstimation      = "estimation_list"
	GridEstimationTotal = "estimation_total"
	GridSpreadingClaim  = "spreading_claim"
	GridSpreadingBreak  = "spreading_break_qs"
	GridAttachment      = "attachment"
	GridSuggestion      = "suggestion"
)

// Alasan dan pemilik penghalang blok Treaty Information.
//
// Dikumpulkan sebagai konstanta, bukan diulang pada kedelapan isiannya, supaya kalimatnya
// tidak dapat berbeda-beda antar isian yang penghalangnya satu dan sama.
const (
	treatyMasterBlockedReason = "Blok ini terikat ke halaman `TreatyInMaster` " +
		"(kelas `ASM-FW-GISFW-Int-TREATY_IN`), yang di Pega hidup TERPISAH dari " +
		"`.ClaimData` sehingga tidak ikut tersimpan di dokumen JSON klaim. Seluruh export " +
		"hanya menyebut TREATY_IN di dua berkas, dan keduanya MEMAKAI halaman itu — tidak " +
		"ada satu pun yang mengisinya. Kedua pra-aksi Flow Action pun bukan pemuat data. " +
		"Tanpa rule pemuatnya, tidak ada cara membaca isian ini tanpa menebak."

	treatyMasterBlockedOwner = "Tim Pega — dibutuhkan rule pemuat halaman " +
		"`ASM-FW-GISFW-Int-TREATY_IN` (Connect-SQL, Data Page, atau Data Transform), " +
		"beserta DDL tabel treaty inward yang dibacanya (R-16, R-08)."
)

// groups adalah kesepuluh kelompok beserta isinya, berurutan seperti tampilnya.
var groups = []Group{
	{
		Code: GroupTreaty,

		// Kontainer pertama di section TIDAK berjudul; yang bertuliskan
		// "<center><b>Outstanding Claim</b></center>" adalah judul LAYARNYA, bukan judul
		// blok ini. Judul di bawah ditetapkan dari isinya.
		Title: "Treaty Information",

		Fields: []Field{
			{Key: "id_master", Title: "Treaty ID", Path: "IDMaster"},
			{Key: "treaty_name", Title: "Treaty Name", Path: "TreatyName"},
			{Key: "ri_type", Title: "R/I Type", Blocked: true},
			{Key: "class_of_business", Title: "Class Of Business",
				Path: "QuotationData.BusinessName"},
			{Key: "ceding_name", Title: "Ceding Name", Blocked: true},
			{Key: "sob_name", Title: "SOB Name", Blocked: true},
			{Key: "bordeaux", Title: "Bordeaux", Blocked: true},
			{Key: "bordereaux_note", Title: "Bordereaux Note", Blocked: true},
			{Key: "year_of_account", Title: "Treaty Year", Path: "YearofAccount"},
			{Key: "start_date_treaty", Title: "StartDateTreaty", Path: "StartDateTreaty"},
			{Key: "end_date_treaty", Title: "EndDateTreaty", Path: "EndDateTreaty"},
			{Key: "accounting_mode", Title: "Accounting Mode", Blocked: true},
			{Key: "teritorial_scope", Title: "TERITORIAL SCOPE", Blocked: true},
			{Key: "treaty_group_id", Title: "Treaty Group", Path: "TreatyGroupID"},

			// Judulnya BENAR-BENAR "Class Of Business", sama dengan isian di atas,
			// meski isinya nama grup treaty. Itu tertulis begitu di section dan
			// dipertahankan apa adanya (`D-13`).
			{Key: "treaty_group_name", Title: "Class Of Business", Path: "TreatyGroupName"},
		},
	},
	{
		Code:  GroupClaim,
		Title: "Claim Information",
		Fields: []Field{
			{Key: "policy_no", Title: "Policy No", Path: "PolicyData.PolicyNo"},

			// Ketiga isian berikut menyusun satu baris "Quarter/Year" di Pega —
			// `Q <Quater> / <YearofQuartal>   U/Y <TreatyYear>`. Di sini ketiganya tetap
			// isian terpisah: merangkainya menjadi satu teks membuat isian yang kosong
			// tidak dapat dibedakan dari yang berisi tanda baca saja.
			{Key: "quater", Title: "Quater", Path: "Quater"},
			{Key: "year_of_quartal", Title: "YearofQuartal", Path: "YearofQuartal"},
			{Key: "treaty_year", Title: "TreatyYear", Path: "TreatyYear"},

			{Key: "policy_no_ceding", Title: "Policy No Ceding", Path: "PolicyNo"},
			{Key: "insured_name", Title: "Insured Name", Path: "InsuredName"},
			{Key: "pla_no_ceding", Title: "Pla No Ceding", Path: "PlaNoCeding"},
			{Key: "policy_start_ceding", Title: "Policy Start Ceding",
				Path: "PolicyData.StartDateTime"},
			{Key: "policy_end_ceding", Title: "Policy End Ceding",
				Path: "PolicyData.EndDateTime"},
			{Key: "date_of_loss", Title: "Date Of Loss", Path: "DateOfLoss"},
			{Key: "report_date", Title: "Report Date", Path: "ReportDate"},
			{Key: "received_date", Title: "Received Date", Path: "DateReceived"},
			{Key: "reporter_name", Title: "Reporter Name", Path: "ReporterName"},
			{Key: "reporter_email", Title: "Reporter Email", Path: "Email"},
			{Key: "cause_of_loss", Title: "Cause Of Loss", Path: "CauseOfLoss"},
			{Key: "report_status", Title: "Report Status", Path: "ReporterStatus"},
			{Key: "report_type", Title: "Report Type", Path: "ReportType"},
			{Key: "insured_relationship_others", Title: "Specify...",
				Path: "InsuredRelationshipOthers"},
			{Key: "report_address", Title: "Report Address", Path: "ReportAddress"},
			{Key: "appointed_adj", Title: "Adjuster / Professional Name",
				Path: "AppointedADJ"},
			{Key: "consultant_name", Title: "Consultant Name", Path: "ConsultantName"},
			{Key: "report_description", Title: "Report Description",
				Path: "ReportDescription"},
			{Key: "location_of_loss", Title: "Location of Loss", Path: "Location"},
			{Key: "province", Title: "Province", Path: "Province"},
			{Key: "zip_code", Title: "Zip Code", Path: "PostalCode"},

			{Key: "asm_share", Title: "ASM Share", Blocked: true},
		},

		// DUA isian di section TIDAK dibawa: `InputData.CARI31` ("Adjuster / Professional
		// ID") dan `InputData.CARI32` ("Consultant ID"). Keduanya kotak autocomplete di
		// atas halaman SEMENTARA `InputData`, bukan bagian klaim — nilainya hanya ada
		// selama pengguna mengetik dan tidak pernah tersimpan. Pada layar baca-saja
		// keduanya pasti kosong, dan menggambar isian yang pasti kosong membuat pengguna
		// mengira datanya hilang. Nama yang dipilih pengguna lewat keduanya TERSIMPAN di
		// `AppointedADJ` dan `ConsultantName`, dan kedua isian itu dibawa.
	},
	{
		Code:  GroupInterest,
		Title: "Insured Interest",
		Fields: []Field{
			{Key: "total_sum_insured_idr", Title: "IDR", Path: "TotalSumInsuredIDR"},
			{Key: "insured_interest", Title: "Description", Path: "InsuredInterest"},
		},
		Grids: []string{GridInterest, GridInterestTotal},
	},
	{
		Code:  GroupDeductible,
		Title: "Deductible",
		Fields: []Field{
			{Key: "share_ceding", Title: "Share Ceding", Path: "ShareCeding"},
			{Key: "deductible_type", Title: "DeductibleType", Path: "DeductibleType"},
			{Key: "form_type", Title: "Format", Path: "FormType"},
			{Key: "currency_deductible", Title: "Currency", Path: "CurrencyDeductible"},
			{Key: "deductible_value", Title: "Amount", Path: "DeductibleValue"},

			// Judulnya di Pega hanya "%" — satu karakter, tanpa keterangan apa pun.
			// Dipertahankan (`D-13`); yang ditambahkan hanyalah nama kuncinya yang
			// menyebut apa isinya.
			{Key: "deductible_percent", Title: "%", Path: "Amount"},

			// Judul sel ini di section adalah `.TypeDeductible` — nama propertinya
			// sendiri, bukan judul. Itu terjadi ketika sel dibuat tanpa mengisi labelnya.
			// Ditulis apa adanya supaya pembandingan dengan layar Pega tetap cocok.
			{Key: "type_deductible", Title: ".TypeDeductible", Path: "TypeDeductible"},

			{Key: "tsi_deductible", Title: "TSI Amount", Path: "TSIDeductible"},
			{Key: "net_deductible_value", Title: "Deductible Value",
				Path: "NetDeductibleValue"},
		},
	},
	{
		Code:   GroupClaimAmount,
		Title:  "Claim Amount",
		Fields: []Field{},
		Grids:  []string{GridClaimAmount},
	},
	{
		Code:   GroupResultClaim,
		Title:  "Result Claim",
		Fields: []Field{},
		Grids:  []string{GridSpreadingRisk},
	},
	{
		Code:  GroupEstimation,
		Title: "Estimation",
		Fields: []Field{
			{Key: "total_gross_estimate_idr", Title: "Total Gross Estimate(100%) in IDR",
				Path: "TotalGrossEstimateIDR"},
			{Key: "total_estimasi_idr", Title: "Total Estimation in IDR",
				Path: "TotalEstimasiIDR"},
			{Key: "ibnr_idr", Title: "IBNR in IDR", Path: "IBNR"},
		},
		Grids: []string{GridEstimation, GridEstimationTotal},
	},
	{
		Code:   GroupSpreading,
		Title:  "Claim Spreaded",
		Fields: []Field{},
		Grids:  []string{GridSpreadingClaim, GridSpreadingBreak},
	},
	{
		Code:   GroupAttachment,
		Title:  "Attachment",
		Fields: []Field{},
		Grids:  []string{GridAttachment},
	},
	{
		Code:   GroupSuggestion,
		Title:  "Suggestion",
		Fields: []Field{},
		Grids:  []string{GridSuggestion},
	},
}

// grids adalah kesepuluh grid beserta kolomnya.
//
// Urutan kolom tiap grid diambil dari pasangan baris kepala dan baris isi pada `pyRows` grid
// itu — kolom ke-n pada baris kepala berpasangan dengan isian ke-n pada baris isi.
var grids = map[string]Grid{
	GridInterest: {
		Code:  GridInterest,
		Title: "Insured Interest",
		Path:  "InterestList",
		Columns: []GridColumn{
			{Key: "object_name", Title: "Insured Interest", Path: "ObjectName"},
			{Key: "currency", Title: "Currency", Path: "CurrencyID"},

			// PERINGATAN: pasangan dua kolom terakhir TAMPAK tertukar, dan itu dibawa apa
			// adanya. Pada baris kepala urutannya "Value in IDR" lalu "Value"; pada baris
			// isi urutannya `.KursObjectItem` lalu `.TSIPerObject` — sehingga kolom
			// berjudul "Value in IDR" menampilkan KURS, dan kolom berjudul "Value"
			// menampilkan nilai pertanggungan per objek.
			//
			// Itu bukan salah baca: kedua baris diambil dari `pyRows` grid yang sama dan
			// jumlah selnya cocok satu lawan satu. `P-5` menetapkan perilaku dipertahankan
			// lebih dulu, dan menukarnya di sini berarti angka di layar baru berbeda dari
			// Pega tanpa satu pun butir perbaikan yang menjelaskannya.
			//
			// Diangkat sebagai pertanyaan terbuka ke Work Owner, bukan diperbaiki sepihak.
			{Key: "kurs", Title: "Value in IDR", Path: "KursObjectItem"},
			{Key: "tsi_per_object", Title: "Value", Path: "TSIPerObject"},
		},
	},
	GridInterestTotal: {
		Code:  GridInterestTotal,
		Title: "Total Insured Interest",
		Path:  "TotalInterestInsured",
		Columns: []GridColumn{
			{Key: "currency", Title: "Total", Path: "Currency"},
			{Key: "value", Title: "Value", Path: "Value"},
		},
	},
	GridClaimAmount: {
		Code:  GridClaimAmount,
		Title: "Claim Amount",
		Path:  "ListClaimAmount",
		Columns: []GridColumn{
			{Key: "currency", Title: "Currency", Path: "CurrencyID"},
			{Key: "claim_amount", Title: "Claim Amount 100%", Path: "ClaimAmount"},
			{Key: "net_deductible", Title: "Net Deductible", Path: "NetDeductibleValue"},
			{Key: "claim_amount_ceding", Title: "Claim Amount Ceding", Path: "Value"},

			// Judul kolomnya "Claim Amount in IDR" sementara propertinya bernama `.USD`.
			// Nama properti itu sisa penamaan lama; yang menentukan arti kolom adalah
			// judulnya, dan judul itulah yang dibawa (`D-13`).
			{Key: "claim_amount_idr", Title: "Claim Amount in IDR", Path: "USD"},
		},
	},
	GridSpreadingRisk: {
		Code:  GridSpreadingRisk,
		Title: "Result Claim",
		Path:  "SpreadingRisk",
		Columns: []GridColumn{
			{Key: "currency", Title: "Currency", Path: "CurrencyID"},
			{Key: "treaty_type", Title: "Treaty Type", Path: "TreatyName"},
			{Key: "share_percentage", Title: "Share (%)", Path: "SharePercentage"},
			{Key: "result_claim", Title: "Result Claim", Path: "ClaimSpreaded"},
			{Key: "ibnr", Title: "IBNR", Path: "IBNR"},
			{Key: "result_claim_idr", Title: "Result Claim in IDR", Path: "ClaimEstimation"},
		},
	},
	GridEstimation: {
		Code:  GridEstimation,
		Title: "Estimation",
		Path:  "EstimationList",
		Columns: []GridColumn{
			// Kolom pertama TIDAK berjudul di Pega — sel kepalanya kosong. Judul di bawah
			// ditambahkan dari nama propertinya supaya kolomnya punya nama bagi pembaca
			// layar; ia satu-satunya judul di berkas ini yang tidak berasal dari section.
			{Key: "type_loss", Title: "Type Loss", Path: "TypeLoss"},

			{Key: "estimation_date", Title: "Estimation Date", Path: "EstimationDate"},
			{Key: "type", Title: "Type", Path: "Type"},
			{Key: "currency", Title: "Currency", Path: "CurrencyID"},

			// Pasangan yang sama anehnya dengan grid Insured Interest: kolom berjudul
			// "Value in IDR" menampilkan KURS.
			{Key: "kurs", Title: "Value in IDR", Path: "KursValue"},

			{Key: "gross_estimation", Title: "Gross Estimate Treaty (100%)",
				Path: "GrossEstimationPct"},
			{Key: "estimation_asm", Title: "Estimation ASM", Path: "EstimationValue"},
			{Key: "ibnr", Title: "IBNR", Path: "IBNR"},
			{Key: "estimation_asm_idr", Title: "Estimation ASM in IDR", Path: "ConvertValue"},
		},
	},
	GridEstimationTotal: {
		Code:  GridEstimationTotal,
		Title: "Total Estimation",
		Path:  "ListTotalEstimation",
		Columns: []GridColumn{
			{Key: "currency", Title: "Currency", Path: "Currency"},

			// Kedua judul di bawah salah ketik di Pega — "Geoss" dan "Esstimation".
			// Dipertahankan apa adanya (`D-13`): membetulkannya membuat pembandingan
			// berdampingan dengan layar lama gagal pada kolom yang sebenarnya benar.
			{Key: "gross_estimate", Title: "Geoss Estimate Treaty (100%)", Path: "IDR"},
			{Key: "estimation_asm", Title: "Esstimation ASM", Path: "Value"},
		},
	},
	GridSpreadingClaim: {
		Code:  GridSpreadingClaim,
		Title: "Claim Spreaded",
		Path:  "SpreadingClaim",
		Columns: []GridColumn{
			{Key: "treaty_type", Title: "Treaty Type", Path: "TreatyName"},
			{Key: "share_percentage", Title: "Share (%)", Path: "SharePercentage"},
			{Key: "currency", Title: "Currency", Path: "Currency"},
			{Key: "claim_spreaded", Title: "Claim Spreaded", Path: "ClaimSpreaded"},
		},
	},
	GridSpreadingBreak: {
		Code: GridSpreadingBreak,

		// Kedua grid ini KEMBAR di layar — judul kolomnya sama persis, dan hanya sumber
		// senarainya yang berbeda. Judulnya dibedakan di sini supaya dua tabel yang
		// bersebelahan tidak dibacakan pembaca layar dengan nama yang sama.
		Title: "Claim Spreaded — Break QS",
		Path:  "SpreadingBreakQS",
		Columns: []GridColumn{
			{Key: "treaty_type", Title: "Treaty Type", Path: "TreatyName"},
			{Key: "share_percentage", Title: "Share (%)", Path: "SharePercentage"},
			{Key: "currency", Title: "Currency", Path: "Currency"},
			{Key: "claim_spreaded", Title: "Claim Spreaded", Path: "ClaimSpreaded"},
		},
	},
	GridAttachment: {
		Code:  GridAttachment,
		Title: "Attachment",
		Columns: []GridColumn{
			{Key: "category", Title: "Category"},
			{Key: "count_attach", Title: "Count Attach"},
		},

		// Satu-satunya grid yang sumbernya BUKAN dokumen JSON klaim. Ia diisi
		// `AttachCategory.pxResults`, hasil Report Definition `BrowseUpRegisterDoc_rd` —
		// yang TIDAK ADA di export.
		Blocked: true,
		BlockedReason: "Daftar lampiran diisi Report Definition `BrowseUpRegisterDoc_rd`, " +
			"yang tidak ada di export (R-16). Tanpa rule itu, tidak diketahui tabel mana " +
			"yang dibacanya maupun bagaimana lampiran dikelompokkan menjadi kategori. " +
			"Kolom \"View File\" pada layar lama pun membuka berkasnya lewat jalur yang " +
			"sama, sehingga keduanya terhalang bersama.",
		BlockedOwner: "Tim Pega — dibutuhkan export Report Definition " +
			"`BrowseUpRegisterDoc_rd` beserta rule yang dipanggil tombol View File.",
	},
	GridSuggestion: {
		Code:  GridSuggestion,
		Title: "Suggestion",
		Path:  "SuggestList",
		Columns: []GridColumn{
			{Key: "name", Title: "Name", Path: "PICSuggest"},
			{Key: "date", Title: "Date", Path: "DateSuggest"},
			{Key: "noted", Title: "Noted", Path: "CommentSuggest"},
		},
	},
}

// Groups mengembalikan kesepuluh kelompok dalam urutan tampilnya.
//
// Salinan sampai ke daftar isiannya: Group memuat senarai, dan salinan dangkal masih berbagi
// lariknya — pemanggil yang menulisinya akan mengubah bentuk layar bagi seluruh permintaan
// berikutnya.
func Groups() []Group {
	result := make([]Group, 0, len(groups))
	for _, group := range groups {
		clone := group
		clone.Fields = make([]Field, len(group.Fields))
		copy(clone.Fields, group.Fields)
		clone.Grids = make([]string, len(group.Grids))
		copy(clone.Grids, group.Grids)
		result = append(result, clone)
	}
	return result
}

// Grids mengembalikan kesepuluh grid dalam urutan tampilnya.
//
// Urutannya diambil dari Groups, bukan dari peta `grids` — urutan peta di Go tidak
// ditetapkan, dan grid yang berpindah tempat setiap permintaan adalah layar yang tidak dapat
// dipercaya.
func GridList() []Grid {
	result := make([]Grid, 0, len(grids))
	for _, group := range groups {
		for _, code := range group.Grids {
			grid := grids[code]
			clone := grid
			clone.Columns = make([]GridColumn, len(grid.Columns))
			copy(clone.Columns, grid.Columns)
			result = append(result, clone)
		}
	}
	return result
}

// FindGrid mencari susunan sebuah grid menurut kodenya.
func FindGrid(code string) (Grid, bool) {
	grid, found := grids[code]
	if !found {
		return Grid{}, false
	}
	clone := grid
	clone.Columns = make([]GridColumn, len(grid.Columns))
	copy(clone.Columns, grid.Columns)
	return clone, true
}

// Fields mengembalikan SELURUH isian skalar dari seluruh kelompok, berurutan.
//
// Dipakai pengisi seam untuk mengetahui isian mana yang harus dipetik dari dokumen JSON, dan
// dipakai NewDetail untuk menolak kunci yang tidak dikenal.
func Fields() []Field {
	result := make([]Field, 0, 64)
	for _, group := range groups {
		result = append(result, group.Fields...)
	}
	return result
}

// knownFields adalah himpunan kunci isian yang sah, disusun sekali saat start.
var knownFields = func() map[string]struct{} {
	set := map[string]struct{}{}
	for _, field := range Fields() {
		set[field.Key] = struct{}{}
	}
	return set
}()

// KnownField menyatakan sebuah kunci isian memang digambar layar.
func KnownField(key string) bool {
	_, found := knownFields[key]
	return found
}

// NewDetail merakit satu rincian dari isian dan grid yang sudah dibaca penyimpanan.
//
// Kunci isian yang TIDAK dikenal section.go dibuang, bukan dibawa diam-diam. Isian yang tidak
// pernah digambar tetapi ikut dikirim adalah data yang keluar dari server tanpa satu pun
// alasan — dan di layar ini isinya memuat nama tertanggung dan nilai klaim.
func NewDetail(
	claimID, reference, statusWork, lastUpdateOperator string,
	values map[string]string,
	gridRows map[string][]GridRow,
) Detail {
	clean := make(map[string]string, len(values))
	for key, value := range values {
		if KnownField(key) {
			clean[key] = value
		}
	}

	rows := make(map[string][]GridRow, len(gridRows))
	for code, list := range gridRows {
		if _, known := grids[code]; known {
			rows[code] = list
		}
	}

	return Detail{
		ClaimID:            claimID,
		Reference:          reference,
		StatusWork:         statusWork,
		LastUpdateOperator: lastUpdateOperator,
		Values:             clean,
		Grids:              rows,
	}
}
