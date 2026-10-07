package inputacceptation

import (
	"claim-pnc/internal/platform/tabletext"
)

// Bentuk layar Acceptation Claim, dibaca dari `Section/InputAcceptation-Section.xml`.
//
// # Kenapa bentuk layar tinggal di BACKEND
//
// Karena ia hasil pembacaan export Pega, dan tempat pembacaan itu tercatat adalah di sini.
// Menyalin ~90 judul isian dan 13 susunan grid ke frontend berarti daftar yang sama hidup di
// dua tempat, dan yang satu akan tertinggal saat yang lain diperbaiki.
//
// Ia juga yang membuat isian terhalang hilang dengan sendirinya begitu penghalangnya hilang —
// tanpa menyunting frontend.
//
// # Bagaimana judul dan jalurnya diperoleh
//
// Judul isian skalar diambil dari `pyLabelFieldValue` tiap sel APA ADANYA (`D-13`); judul
// kolom grid diambil dari baris KEPALA gridnya (`pyCellHeader=true`), karena sel isi grid
// hanya membawa label bawaan kontrol ("Dropdown", "Number", "Text Input") yang tidak
// menyatakan apa pun.
//
// Jalurnya diturunkan dari `pyValue` sel yang sama: `.ClaimData.X` di klipboard adalah `X` di
// dalam dokumen klaim. Ia TERVERIFIKASI, bukan dugaan: ke-38 kunci tingkat atas dokumen nyata
// pada basis data pengembangan cocok satu-untuk-satu dengan jalur di bawah — `IDMaster`,
// `TreatyName`, `DateOfLoss`, `InterestList`, `ListClaimAmount`, `CNPSpreadLoss`,
// `SpreadingRisk`, `SpreadingClaim`, `SpreadingBreakQS`, `SuggestList`, dan seterusnya.
//
// Jalur SENARAI tiap grid diambil dari `pyPageListProperty` kontainer gridnya, yang terbaca
// utuh di section — ketiga belasnya disebut lengkap di bawah.
//
// # Kolom JSON-nya DATA_JSONBLOB
//
// `POOLDATA.JSON_KLAIM` punya dua kolom JSON. Keempat kueri inbox non-prop membaca
// `DATA_JSON`; layar ini TIDAK mengikutinya, karena pada klaim `CLMNP-%` kolom itu kosong
// seluruhnya. Yang berisi dokumen adalah `DATA_JSONBLOB` — sama dengan layar Prop. Buktinya
// diukur langsung dan dicatat di kepala repo/sqlstore/inputacceptation.sql.

// Field adalah satu isian skalar pada layar.
type Field struct {
	// Key adalah nama isian pada kontrak API.
	Key string

	// Title adalah judul yang dibaca pengguna, mengikuti Pega apa adanya (`D-13`).
	Title string

	// Path adalah jalur isian ini di dalam dokumen JSON klaim, memakai titik sebagai
	// pemisah tingkat — mis. `PolicyData.PolicyNo`.
	//
	// KOSONG pada isian yang Blocked: tidak ada jalur yang dapat dibaca.
	Path string

	// Editable menyatakan sel ini TIDAK bertanda read-only di section.
	//
	// Ia yang memisahkan isian yang benar-benar diisi petugas dari isian yang hanya
	// ditampilkan. Pada layar berisi 120 sel read-only dari 273, pembedaan itu menentukan
	// apa yang boleh ikut terkirim saat Submit.
	Editable bool

	// Blocked menyatakan isian ini digambar tetapi belum dapat diisi.
	//
	// Ia BUKAN isian yang disembunyikan. Menyembunyikannya membuat pengguna yang
	// membandingkan layar ini dengan Pega mengira isiannya hilang; menggambarnya dengan
	// alasan membuat ia tahu isiannya ada dan kenapa kosong.
	Blocked bool

	// BlockedReason dan BlockedOwner menjelaskan penghalangnya beserta alamatnya.
	//
	// Keduanya diisi init(), bukan ditulis ulang pada setiap isian, supaya kedelapan isian
	// yang terhalang oleh sebab yang sama tidak dapat menyebut alasan yang berbeda-beda.
	// Penghalang tanpa pemilik tidak pernah hilang (`D-36`).
	BlockedReason string
	BlockedOwner  string
}

// GridColumn adalah satu kolom pada sebuah grid.
type GridColumn struct {
	Key   string
	Title string

	// Path adalah nama isian di dalam satu baris senarai JSON, relatif terhadap barisnya.
	Path string

	// Editable menyatakan sel ini tidak bertanda read-only di section.
	Editable bool
}

// Grid adalah satu tabel pada layar.
type Grid struct {
	// Code adalah nama grid pada kontrak API.
	Code string

	// Title adalah judul yang dibaca pengguna. Sebagian grid di Pega tidak punya judul
	// sama sekali — judul di bawah ditetapkan dari label kontainernya atau dari kolom
	// pertamanya, supaya tiga belas tabel dalam satu layar dapat dibedakan pembaca layar.
	Title string

	// Path adalah jalur SENARAI-nya di dalam dokumen JSON klaim, dari `pyPageListProperty`.
	Path string

	Columns []GridColumn

	Blocked       bool
	BlockedReason string
	BlockedOwner  string
}

// Group adalah satu kelompok isian beserta grid yang digambar sesudahnya.
type Group struct {
	Code   string
	Title  string
	Fields []Field

	// Grids adalah kode grid yang digambar sesudah isian kelompok ini, berurutan.
	Grids []string
}

// Kode kelompok isian.
const (
	GroupTreaty        = "treaty"
	GroupClaim         = "klaim"
	GroupInterest      = "interest"
	GroupDeductible    = "deductible"
	GroupClaimAmount   = "nilai_klaim"
	GroupLossAlloc     = "alokasi_kerugian"
	GroupReinstatement = "reinstatement"
	GroupSpreadClaim   = "claim_spreded"
	GroupAcceptation   = "akseptasi"
	GroupSpreadAdjust  = "claim_spreaded"
	GroupSuggestion    = "konfirmasi_ceding"
)

// Kode grid.
//
// Ketiga belasnya berpasangan satu-untuk-satu dengan `pyPageListProperty` di section, dan
// pasangannya disebut pada Path masing-masing.
const (
	GridInterest        = "interest_list"
	GridInterestTotal   = "interest_total"
	GridClaimAmount     = "claim_amount"
	GridSpreadLoss      = "cnp_spread_loss"
	GridSpreadingRisk   = "spreading_risk"
	GridTotalEstimation = "total_estimation"
	GridReinstatement   = "reinstatement_list"
	GridSpreadingClaim  = "spreading_claim"
	GridSpreadBreakQS   = "spreading_break_qs"
	GridAdjustment      = "adjustment_list"
	GridSpreadAdjust    = "spreading_adjustment"
	GridSpreadAdjustQS  = "spreading_adjustment_qs"
	GridSuggest         = "suggest_list"
)

// blockedTreatyInMaster adalah alasan kedelapan isian `.TreatyInMaster.*` tidak dapat diisi.
//
// Ia dikumpulkan sekali supaya kedelapannya tidak dapat menyebut alasan yang berbeda-beda.
const blockedTreatyInMaster = "Isian ini berada di halaman TreatyInMaster berkelas " +
	"ASM-FW-GISFW-Int-TREATY_IN — halaman TERSENDIRI pada objek kerja, bukan bagian " +
	".ClaimData, sehingga tidak ikut tersimpan di dokumen JSON klaim. Ketiga pra-aksi Flow " +
	"Action ini MEMBACA halaman itu (InputOutStandingClmTNP_PreAct mengambil " +
	".TreatyInMaster.Commencement) tetapi tidak satu pun MENGISINYA, dan tidak ada rule lain " +
	"di export yang mengisinya. Menebaknya dari kolom yang kebetulan mirip berarti " +
	"menampilkan pita share reasuransi milik treaty lain sebagai milik treaty ini — " +
	"kekeliruan yang tidak menghasilkan satu pun galat."

const blockedOwnerTreatyInMaster = "Tim Pega — dibutuhkan rule yang mengisi halaman " +
	"TreatyInMaster, atau DBA untuk DDL tabel TREATY_IN beserta pemetaan propertinya (R-08)."

// blockedOfferFacIn adalah alasan isian "Class Of Business" tidak dapat diisi.
//
// Ia terpisah dari blockedTreatyInMaster karena halamannya memang berbeda — dan menyamakan
// keduanya akan mengirim Tim Pega mencari rule yang salah.
const blockedOfferFacIn = "Isian ini terikat pyWorkPage.OfferFacIn.QuotationData.BusinessName " +
	"— halaman OfferFacIn, bukan .ClaimData, sehingga tidak ikut tersimpan di dokumen JSON " +
	"klaim. Banyak rule lain di export MEMBACA halaman itu, tetapi tidak satu pun dari " +
	"ketiga pra-aksi Flow Action ini mengisinya, sehingga saat layar ini terbuka halamannya " +
	"kosong."

// groups adalah seluruh kelompok isian, berurutan seperti tampilnya di section.
var groups = []Group{
	{
		Code:  GroupTreaty,
		Title: "Treaty Information",
		Fields: tabletext.Rows[Field](`
			Key               | Title             | Path            | Blocked
			claim_no          | Claim No          | NoClaim         |
			treaty_id         | Treaty ID         | IDMaster        |
			treaty_name       | Treaty Name       | TreatyName      |
			ri_type           | R/I Type          |                 | true
			// Isian ini terikat 'pyWorkPage.OfferFacIn.QuotationData.BusinessName' —
			// halaman 'OfferFacIn', bukan '.ClaimData'. Seperti TreatyInMaster, ia tidak
			// diisi satu pun pra-aksi Flow Action ini, sehingga tidak dapat dibaca dari
			// dokumen klaim.
			class_of_business | Class Of Business |                 | true
			ceding_name       | Ceding Name       |                 | true
			sob_name          | SOB Name          |                 | true
			bordereaux        | Bordereaux        |                 | true
			bordereaux_note   | Bordereaux Note   |                 | true
			treaty_year       | Treaty Year       | YearofAccount   |
			treaty_start_date | Treaty Start Date | StartDateTreaty |
			treaty_end_date   | Treaty End Date   | EndDateTreaty   |
			accounting_mode   | Accounting Mode   |                 | true
			teritorial_scope  | Teritorial Scope  |                 | true
		`),
	},
	{
		Code:  GroupClaim,
		Title: "Claim Information",
		Fields: tabletext.Rows[Field](`
			Key                         | Title                        | Path                      | Editable
			policy_no                   | Policy No                    | PolicyData.PolicyNo       |
			date_of_loss                | Date of Loss                 | DateOfLoss                |
			report_date                 | Report Date                  | ReportDate                |
			policy_no_ceding            | Policy No Ceding             | PolicyNo                  |
			received_date               | Received Date                | DateReceived              |
			insured_name                | Insured Name                 | InsuredName               |
			reporter_name               | Reporter Name                | ReporterName              |
			pla_no_ceding               | Pla No Ceding                | PlaNoCeding               |
			claim_no_ceding             | Claim No Ceding              | CNPClmNoCedant            |
			// Salah satu dari sedikit isian yang BOLEH diubah di layar ini.
			dla_no_ceding               | DLA No Ceding                | DLANoCeding               | true
			reporter_phone              | Reporter Phone Number        | ReporterTelp              |
			policy_start                | Policy Start                 | PolicyData.StartDateTime  |
			policy_end                  | Policy End                   | PolicyData.EndDateTime    |
			cause_of_loss               | Cause Of Loss                | CauseOfLoss               |
			// Label selnya di section berbunyi "Dropdown" — label bawaan kontrol, bukan
			// judul. Judul di bawah mengikuti modul Outstanding Claim, yang isian sama
			// pada layar saudaranya bernama begitu.
			report_status               | Report Status                | ReporterStatus            | true
			report_type                 | Report Type                  | ReportType                |
			insured_relationship_others | Specify...                   | InsuredRelationshipOthers |
			location_of_loss            | Location of Loss             | Location                  |
			reporter_address            | Reporter Address             | ReportAddress             |
			report_description          | Report Description           | ReportDescription         |
			appointed_adj               | Adjuster / Professional Name | AppointedADJ              |
			consultant_name             | Consultant Name              | ConsultantName            |
			circumtances                | Circumtances                 | CNPCircumtances           | true
			supporting_document         | Supporting Document          | CNPSupportDoc             | true
		`),

		// DUA isian di section TIDAK dibawa: `InputData.CARI31` dan `InputData.CARI32`.
		// Keduanya kotak pencari di atas halaman SEMENTARA `InputData` — pencari ID
		// adjuster dan ID konsultan — bukan bagian klaim. Nilainya hanya ada selama layar
		// terbuka dan tidak pernah tersimpan. Perlakuan yang sama dipakai modul Outstanding
		// Claim atas kedua kotak yang sama.
	},
	{
		Code:  GroupInterest,
		Title: "Insured Interest",
		Grids: []string{GridInterest, GridInterestTotal},
		Fields: []Field{
			{Key: "total_sum_insured_idr", Title: "Total Sum Insured (IDR)",
				Path: "TotalSumInsuredIDR"},
			{Key: "insured_interest", Title: "Description", Path: "InsuredInterest"},
		},
	},
	{
		Code:  GroupDeductible,
		Title: "Deductible",
		Fields: tabletext.Rows[Field](`
			Key                 | Title           | Path               | Editable
			share_ceding        | Share Ceding(%) | ShareCeding        |
			// Labelnya di section "Checkbox" — label bawaan kontrol. Ia yang MEMILIH di
			// antara dua susunan deductible yang digambar berdampingan di section
			// (persentase-dari-TSI versus jumlah tetap), dan keduanya membaca isian yang
			// sama. Yang digambar di sini satu susunan, dengan penanda jenisnya.
			deductible_type     | Deductible Type | DeductibleType     | true
			form_type           | Format          | FormType           |
			currency_deductible | Currency        | CurrencyDeductible | true
			deductible_value    | Amount          | DeductibleValue    |
			deductible_percent  | %               | Amount             |
			type_deductible     | of              | TypeDeductible     |
			tsi_deductible      | TSI Amount      | TSIDeductible      |
		`),
	},
	{
		Code:  GroupClaimAmount,
		Title: "Claim Amount",
		Grids: []string{GridClaimAmount},
		Fields: []Field{
			{Key: "asm_share", Title: "ASM Share", Blocked: true},
		},
	},
	{
		Code:  GroupLossAlloc,
		Title: "Loss Allocation",
		Grids: []string{GridSpreadLoss, GridSpreadingRisk, GridTotalEstimation},
	},
	{
		Code:  GroupReinstatement,
		Title: "Reinstatement",
		Grids: []string{GridReinstatement},
	},
	{
		Code:  GroupSpreadClaim,
		Title: "Claim Spreded",
		Grids: []string{GridSpreadingClaim, GridSpreadBreakQS},
	},
	{
		Code:  GroupAcceptation,
		Title: "Acceptation",
		Grids: []string{GridAdjustment},
	},
	{
		Code:  GroupSpreadAdjust,
		Title: "Claim Spreaded",
		Grids: []string{GridSpreadAdjust, GridSpreadAdjustQS},
	},
	{
		Code:  GroupSuggestion,
		Title: "Ceding Confirmation",
		Grids: []string{GridSuggest},
	},
}

// Kolom yang berulang di banyak grid, disusun sekali supaya judulnya tidak dapat berbeda
// antar grid tanpa disengaja.
var (
	colCurrencyID  = GridColumn{Key: "currency", Title: "Currency", Path: "CurrencyID"}
	colCurrency    = GridColumn{Key: "currency", Title: "Currency", Path: "Currency"}
	colTreatyName  = GridColumn{Key: "treaty_name", Title: "Treaty Name", Path: "TreatyName"}
	colTreatyType  = GridColumn{Key: "treaty_type", Title: "Treaty Type", Path: "TreatyName"}
	colSharePct    = GridColumn{Key: "share_pct", Title: "Share (%)", Path: "SharePercentage"}
	colClaimSpread = GridColumn{Key: "claim_spreded", Title: "Claim Spreded",
		Path: "ClaimSpreaded"}
	colAdjusterFee = GridColumn{Key: "adjuster_fee", Title: "Adjuster Fee",
		Path: "AdjusterFee"}
	colSalvage     = GridColumn{Key: "salvage", Title: "Salvage", Path: "Salvage"}
	colOthersFee   = GridColumn{Key: "fee", Title: "Fee", Path: "CNPOthersFee"}
	colTotalClaim  = GridColumn{Key: "total_claim", Title: "Total Claim", Path: "TotalClaim"}
	colPremiSpread = GridColumn{Key: "reinstatement_premium",
		Title: "Reinstatement Premium", Path: "PremiumSpreaded"}
)

// grids adalah ketiga belas grid beserta kolomnya, berurutan seperti tampilnya.
//
// Setiap Path senarai diambil dari `pyPageListProperty` kontainer gridnya di section; nomor
// posisinya disebut supaya dapat ditelusuri balik.
var grids = []Grid{
	{
		Code: GridInterest, Title: "Insured Interest",
		Path: "InterestList", // section :984363
		Columns: []GridColumn{
			{Key: "object_name", Title: "Insured Interest", Path: "ObjectName"},
			colCurrencyID,
			{Key: "value_idr", Title: "Value in IDR", Path: "KursObjectItem"},
			{Key: "value", Title: "Value", Path: "TSIPerObject"},
		},
	},
	{
		Code: GridInterestTotal, Title: "Total Insured Interest",
		Path: "TotalInterestInsured", // section :1104993
		Columns: []GridColumn{
			colCurrency,
			{Key: "value", Title: "Value", Path: "Value"},
		},
	},
	{
		Code: GridClaimAmount, Title: "Claim Amount",
		Path: "ListClaimAmount", // section :1693149
		Columns: []GridColumn{
			colCurrencyID,
			{Key: "rate_of_exchange", Title: "Rate Of Exchange", Path: "AltValue",
				Editable: true},
			{Key: "claim_amount", Title: "Claim Amount", Path: "Value"},
			{Key: "tpl", Title: "TPL", Path: "TPL"},
			{Key: "adjuster_fee", Title: "Adjuster Fee", Path: "AdjusterFee",
				Editable: true},
			{Key: "salvage", Title: "Salvage", Path: "Salvage", Editable: true},
			{Key: "fee", Title: "Fee", Path: "CNPOthersFee", Editable: true},
			{Key: "proportion_pct", Title: "Proportion (%)", Path: "PctProrateClaim"},
			{Key: "claim_amount_idr", Title: "Claim Amount in IDR", Path: "USD"},
			{Key: "claim_amount_cedant", Title: "Claim Amount Cedant",
				Path: "ClaimAmountCedant"},
		},
	},
	{
		Code: GridSpreadLoss, Title: "Loss Allocation",
		Path: "CNPSpreadLoss", // section :1956544
		Columns: []GridColumn{
			colCurrencyID,
			{Key: "treaty_name", Title: "Treaty Name", Path: "TreatyName", Editable: true},
			{Key: "share_pct", Title: "Share (%)", Path: "ClaimPercentage",
				Editable: true},
			{Key: "claim_amount", Title: "Claim Amount", Path: "ClaimAmountAdjust",
				Editable: true},
			{Key: "adjuster_fee", Title: "Adjuster Fee", Path: "AdjusterFee",
				Editable: true},
			{Key: "salvage", Title: "Salvage", Path: "Salvage", Editable: true},
			{Key: "fee", Title: "Fee", Path: "CNPOthersFee", Editable: true},
			{Key: "to_xol", Title: "To XOL", Path: "CNPFlagXOL", Editable: true},
		},
	},
	{
		Code: GridSpreadingRisk, Title: "Claim Amount ASM",
		Path: "SpreadingRisk", // section :2132206
		Columns: []GridColumn{
			colCurrencyID,
			colTreatyName,
			{Key: "claim_amount", Title: "Claim Amount", Path: "TotalClaim",
				Editable: true},
			{Key: "asm_share_pct", Title: "ASM Share (%)", Path: "ClaimPercentage"},
			{Key: "claim_amount_asm", Title: "Claim Amount ASM", Path: "ClaimSpreaded"},
			colAdjusterFee, colSalvage, colOthersFee,
		},
	},
	{
		Code: GridTotalEstimation, Title: "Total Claim",
		Path: "ListTotalEstimation", // section :2283095
		Columns: []GridColumn{
			colCurrency,
			{Key: "claim_amount", Title: "Claim Amount", Path: "CNPTotalClaim"},
			{Key: "claim_amount_asm", Title: "Claim Amount ASM", Path: "Value"},
			colAdjusterFee, colSalvage, colOthersFee,
		},
	},
	{
		Code: GridReinstatement, Title: "Reinstatement",
		Path: "ReinstatementList", // section :2407573
		Columns: []GridColumn{
			{Key: "layer", Title: "Layer", Path: "Layer"},
			colCurrencyID,
			{Key: "claim_amount", Title: "Claim Amount", Path: "IDR"},
			colAdjusterFee, colSalvage,
			{Key: "limit", Title: "Limit", Path: "CNPLimit"},
			{Key: "premi_mdp", Title: "Premi MDP", Path: "CNPMDP"},
			{Key: "reinstatement_pct", Title: "Reinstatement (%)",
				Path: "CNPPctReinstate"},
			{Key: "reinstatement_premium", Title: "Reinstatement Premium",
				Path: "CNPReinstatePremium"},
			{Key: "reinstatement_premium_asm", Title: "Reinstatement Premium ASM",
				Path: "CNPReinsPremiRNM"},
		},
	},
	{
		Code: GridSpreadingClaim, Title: "Claim Spreded",
		Path: "SpreadingClaim", // section :2600413
		Columns: []GridColumn{
			colTreatyType, colSharePct, colCurrency, colClaimSpread,
			colAdjusterFee, colSalvage, colOthersFee,
		},
	},
	{
		Code: GridSpreadBreakQS, Title: "Claim Spreded — Break QS",
		Path: "SpreadingBreakQS", // section :2740956
		Columns: []GridColumn{
			colTreatyType, colSharePct, colCurrency, colClaimSpread,
			colAdjusterFee, colSalvage, colOthersFee,
		},
	},
	{
		Code: GridAdjustment, Title: "Acceptation",
		Path: "AdjustmentList", // section :2938678
		Columns: tabletext.Rows[GridColumn](`
			Key              | Title            | Path             | Editable
			type             | Type             | Type             | true
			acceptation_no   | Acceptation No   | AcceptedNo       |
			acceptation_date | Acceptation Date | AcceptedDate     |
			status           | Status           | AcceptanceStatus |
		`),
	},
	{
		Code: GridSpreadAdjust, Title: "Claim Spreaded",
		Path: "SpreadingAdjustment", // section :3075037
		Columns: []GridColumn{
			colTreatyType,
			// Judulnya "Share(%)" TANPA spasi di grid ini, berbeda dari "Share (%)" pada
			// grid Claim Spreded di atas. Keduanya dibawa apa adanya (`D-13`).
			{Key: "share_pct", Title: "Share(%)", Path: "SharePercentage"},
			colCurrency,
			{Key: "claim_spreaded", Title: "Claim Spreaded", Path: "ClaimSpreaded"},
			colAdjusterFee, colSalvage, colOthersFee, colTotalClaim, colPremiSpread,
		},
	},
	{
		Code: GridSpreadAdjustQS, Title: "Claim Spreaded — QS",
		Path: "SpreadingAdjustmentQS", // section :3241298
		Columns: []GridColumn{
			colTreatyType,
			{Key: "share_pct", Title: "Share(%)", Path: "SharePercentage"},
			colCurrency,
			{Key: "claim_spreaded", Title: "Claim Spreaded", Path: "ClaimSpreaded"},
			colAdjusterFee, colSalvage, colOthersFee, colTotalClaim, colPremiSpread,
		},
	},
	{
		Code: GridSuggest, Title: "Ceding Confirmation",
		Path: "SuggestList", // section :3448089
		Columns: tabletext.Rows[GridColumn](`
			Key       | Title     | Path            | Editable
			confirmed | Confirmed | IsCedingConfirm | true
			name      | Name      | Initial         |
			date      | Date      | DateSuggest     | true
			noted     | Noted     | CommentSuggest  | true
		`),
	},
}

// init melengkapi alasan terhalang setiap isian yang Blocked.
//
// Ia disusun di sini, bukan diulang sembilan kali di atas, supaya seluruhnya tidak dapat
// menyebut alasan yang berbeda-beda saat salah satunya disunting. Sekaligus ia memastikan
// isian terhalang TIDAK punya jalur baca — jalur pada isian terhalang adalah undangan untuk
// membacanya dari tempat yang salah.
func init() {
	for gi := range groups {
		for fi := range groups[gi].Fields {
			f := &groups[gi].Fields[fi]
			if !f.Blocked {
				continue
			}
			f.Path = ""
			f.Editable = false
			f.BlockedOwner = blockedOwnerTreatyInMaster
			if f.Key == "class_of_business" {
				f.BlockedReason = blockedOfferFacIn
				continue
			}
			f.BlockedReason = blockedTreatyInMaster
		}
	}
}

// Groups mengembalikan seluruh kelompok isian dalam urutan tampilnya.
//
// Salinan, bukan senarai aslinya: pemanggil tidak boleh dapat mengubah bentuk layar dengan
// menulisi hasilnya.
func Groups() []Group {
	result := make([]Group, len(groups))
	copy(result, groups)
	return result
}

// GridList mengembalikan seluruh grid dalam urutan tampilnya.
func GridList() []Grid {
	result := make([]Grid, len(grids))
	copy(result, grids)
	return result
}

// FindGrid mencari grid menurut kodenya.
func FindGrid(code string) (Grid, bool) {
	for _, g := range grids {
		if g.Code == code {
			return g, true
		}
	}
	return Grid{}, false
}

// Fields mengembalikan seluruh isian skalar dari seluruh kelompok, berurutan.
func Fields() []Field {
	result := []Field{}
	for _, g := range groups {
		result = append(result, g.Fields...)
	}
	return result
}

// KnownField menyatakan sebuah kunci isian dikenal bentuk layar ini.
//
// Ia yang menjaga Detail.Values tidak dapat menampung isian yang tidak pernah digambar —
// isian seperti itu tidak akan pernah terlihat, dan keberadaannya hanya menyesatkan pembaca
// jawaban API.
func KnownField(key string) bool {
	for _, f := range Fields() {
		if f.Key == key {
			return true
		}
	}
	return false
}

// EditableFields mengembalikan kunci isian skalar yang BOLEH diubah petugas.
//
// Ia dipakai memeriksa muatan Submit: isian di luar daftar ini ditolak, bukan diabaikan
// diam-diam. Mengabaikannya membuat pengguna mengira perubahannya tersimpan.
func EditableFields() []string {
	result := []string{}
	for _, f := range Fields() {
		if f.Editable && !f.Blocked {
			result = append(result, f.Key)
		}
	}
	return result
}
