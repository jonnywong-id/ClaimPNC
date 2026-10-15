package lodpdf

// Template setiap jenis LOD (`.PDFType`), disalin dari contoh PDF Work Owner di
// `Sample Form/` kalimat per kalimat — termasuk salah ketik dan tanda baca yang janggal pada
// contohnya, karena hasil yang benar selama migrasi adalah hasil yang sama dengan Pega.
//
// Markah teks: *tebal*, ~miring~, ^tebal miring^, =tebal bergaris bawah=. Isian:
// {cur} mata uang · {amt} nilai · {words} terbilang · {wordsbare} terbilang tanpa kata mata
// uang · {pol} nomor polis · {dol} tanggal kejadian · {loc} lokasi kejadian · {addr} alamat
// tertanggung · {insured} nama tertanggung · {long} / {short} tanggal cetak.

type fieldStyle int

const (
	fieldsTable  fieldStyle = iota // "Nama        :" — titik dua satu kolom
	fieldsInline                   // "Nama :"
	fieldsFar                      // titik dua di ujung kanan (contoh jenis 4 dan 5)
	fieldsTight                    // kolom label sempit (contoh Ex Gratia)
)

type recipientStyle int

const (
	recipientTable  recipientStyle = iota // Nama / Email / Bank / No Rekening
	recipientTight                        // idem, kolom label sempit
	recipientBranch                       // Nama / No Rekening / Bank … cabang :
	recipientDotted                       // "Nama : ......" tebal (salinan kedua jenis 6)
)

// surveyStyle adalah bingkai halaman Indeks Kepuasan Pelanggan, yang berbeda antar contoh.
type surveyStyle struct {
	outer    bool   // bingkai seluruh halaman
	inner    bool   // bingkai selebar halaman di sekeliling deret wajah
	centered bool   // deret wajah di tengah (bukan rata kiri)
	closing  string // "- PT. Asuransi Sinar Mas -"; beberapa contoh tanpa spasi sesudah "PT."
}

// letter adalah satu halaman surat LOD. next adalah salinan berikutnya pada jenis yang
// mencetak dua surat (jenis 6).
type letter struct {
	kopBold2  bool // "(KOP SURAT TERTANGGUNG)" tebal
	kopSmall  bool
	underline bool // "SURAT PERNYATAAN" bergaris bawah

	signer  string
	fields  fieldStyle
	address bool // Alamat diisi alamat tertanggung

	amount string // paragraf nilai; bila table, tabel co member menyusul
	table  bool
	body   []string

	intro     string
	recipient recipientStyle
	closing   string
	place     string
	sign      []string
	roles     []string

	footnote      string
	footnotePage2 bool

	survey surveyStyle
	next   *letter
}

const (
	asm           = "PT. Asuransi Sinar Mas"
	asmMember     = "PT. Asuransi Sinar Mas dan co member"
	asmMemberDash = "PT. Asuransi Sinar Mas dan co-member"

	signerSpaced = "Kami yang bertanda tangan di bawah ini :"
	signerTight  = "Kami yang bertanda tangan di bawah ini:"
	introSpaced  = "Pembayaran klaim ini ditujukan kepada :"
	introTight   = "Pembayaran klaim ini ditujukan kepada:"

	closing       = "Demikian Surat Pernyataan ini dibuat dengan sebenar-benarnya, tanpa ada tekanan dari pihak manapun serta dapat dipergunakan sebagaimana mestinya."
	closingTypo   = "Demikian Surat Penyataan ini dibuat dengan sebenar-benarnya, tanpa ada tekanan dari pihak manapun serta dapat dipergunakan sebagaimana mestinya."
	footnoteLower = "co member : ~co member berdasarkan share pada polis~"

	lawSpaced = "dan / atau ketentuan hukum dan perundang-undangan"
	lawTypo   = "dan / atau ketentutan hukum dan perundang-undangan"
	lawTight  = "dan/atau ketentuan hukum dan perundang-undangan"

	evidenceInterim = "Bahwa kami akan memberikan bukti-bukti secara tertulis / dokumen yang diperlukan PT. Asuransi Sinar Mas dan co member, apabila proses klaim ini kami lanjutkan untuk mendapatkan tambahan penggantian sesuai ketentuan di dalam polis."
	interimSolo     = "Bahwa sehubungan dengan pembayaran interim ini, kami dengan ini menyatakan bahwa tidak ada pihak – pihak lain yang berkepentingan dan berhak terhadap harta benda yang dipertanggungkan selain kami."
)

var (
	surveyPlain  = surveyStyle{inner: true, closing: "- PT. Asuransi Sinar Mas -"}
	surveyTight  = surveyStyle{inner: true, closing: "- PT.Asuransi Sinar Mas -"}
	surveyCenter = surveyStyle{inner: true, centered: true, closing: "- PT. Asuransi Sinar Mas -"}
	surveyFramed = surveyStyle{outer: true, inner: true, closing: "- PT. Asuransi Sinar Mas -"}
)

// refund adalah paragraf pengembalian uang bila penanggung ternyata tidak wajib membayar.
func refund(law, party, tail string) string {
	return "Bahwa apabila di kemudian hari terbukti berdasarkan ketentuan polis beserta endorsement dan klausula yang melekat " +
		law + " yang berlaku ternyata " + party + " tidak mempunyai kewajiban untuk membayar klaim tersebut di atas, maka uang " +
		"dengan jumlah tersebut di atas yang telah kami terima akan dikembalikan kepada " + tail
}

// release adalah paragraf pembebasan penuh dan final.
func release(party, bahwa string) string {
	return "Bahwa sehubungan dengan pembayaran klaim ini, kami dengan ini membebaskan " + party + " dari seluruh tanggung jawab " +
		"selanjutnya atas kerugian yang dipertanggungkan di bawah polis tersebut di atas, baik sekarang maupun di masa yang " +
		"akan datang berkaitan dengan kerugian yang dimaksud, dan selanjutnya menyatakan " + bahwa + " tidak ada pihak-pihak " +
		"lain yang berkepentingan terhadap harta benda yang dipertanggungkan."
}

// subrogation adalah paragraf hak subrogasi LOD Marine Cargo.
func subrogation(party, end string) string {
	return "Dengan telah dibayarkannya klaim ini, maka " + party + " memperoleh segala hak kami terhadap pihak ketiga sehubungan " +
		"dengan klaim ini berdasarkan hak subrogasi, dan kami akan membantu sepenuhnya segala usaha dan tindakan yang dilakukan " +
		party + " untuk mendapatkannya, dan akan menghindari setiap perbuatan yang merugikan hak " + end
}

// indemnityRelease dan indemnityEvidence adalah paragraf LOD indemnity / reinstatement.
func indemnityRelease(party, indemnity, reinstatement, fulfilled string) string {
	return "Bahwa sehubungan dengan pembayaran klaim secara " + indemnity + " ini, kami dengan ini membebaskan " + party +
		" dari seluruh tanggung jawab selanjutnya atas kerugian yang dipertanggungkan di bawah polis tersebut di atas, baik " +
		"sekarang maupun di masa yang akan datang berkaitan dengan kerugian yang dimaksud, dan melepaskan hak untuk " +
		"mendapatkan tambahan penggantian kerugian secara " + reinstatement + " apabila proses klaim sesuai ketentuan " +
		"reinstatement di dalam polis tidak " + fulfilled + "/dilanjutkan oleh kami, dan selanjutnya menyatakan bahwa tidak " +
		"ada pihak-pihak lain yang berkepentingan terhadap harta benda yang dipertanggungkan."
}

func indemnityEvidence(who, reinstatement string) string {
	return "Bahwa kami akan memberikan bukti-bukti secara tertulis / dokumen yang diperlukan " + who + ", apabila proses klaim " +
		"kami lanjutkan untuk mendapatkan tambahan penggantian kerugian secara " + reinstatement + " sesuai dengan tenggang " +
		"waktu mengenai " + reinstatement + " sebagaimana yang tercantum dalam polis."
}

// indemnityCopy2 adalah salinan kedua LOD jenis 6 — contohnya mencetak dua surat yang
// berbeda sedikit. "( lima ratus ribu , )" pada contoh adalah terbilang tanpa kata mata
// uang diikuti koma; disalin apa adanya.
var indemnityCopy2 = letter{
	kopBold2: true, signer: signerSpaced, fields: fieldsTable,
	amount: "Menyatakan setuju dan telah menerima uang sejumlah *{cur} {amt} .-* ( {wordsbare} , ) secara indemnity basis dari " +
		"*PT. Asuransi Sinar Mas* sebagai pembayaran atas klaim kerugian / kerusakan yang kami yang ajukan pada polis *no. {pol}* " +
		"sehubungan dengan kerugian / kerusakan yang terjadi pada tanggal *{dol} DOL*, di {loc} *(lokasi kejadian)*.",
	body: []string{
		indemnityRelease(asm, "indemnity", "reinstatement", "dipenuihi"),
		indemnityEvidence("PT. Asuransi Sinar Mas ", "reinstatement"),
		refund(lawSpaced, asm, asm+" ."),
	},
	intro: introSpaced, recipient: recipientDotted, closing: closing,
	place: "*{long}*",
	sign:  []string{"Untuk dan atas nama", "Meterai Rp. 10.000,-"},
	roles: []string{"(______________________)", "Direktur"},
}

// letters memetakan `.PDFType` ke templatenya. Jenis 11 (Marine Hull) dwibahasa dan dua
// kolom — lihat marineHull. Jenis 16 tidak punya contoh sehingga tidak ada di sini.
var letters = map[string]letter{
	// Property - Interim dengan Co Member
	"1": {
		kopBold2: true, underline: true, signer: signerTight, fields: fieldsTable,
		amount: "Menyatakan setuju dan telah menerima uang sejumlah *{cur} {amt} ( {words} )* dari:",
		table:  true,
		body: []string{
			"sebagai pembayaran interim atas klaim kerugian / kerusakan yang kami ajukan pada polis no *{pol}* sehubungan dengan " +
				"kerugian yang terjadi pada tanggal *{dol}* di *{loc}* .",
			"Bahwa sehubungan dengan pembayaran klaim interim ini, kami dengan ini membebaskan PT. Asuransi Sinar Mas dan co member " +
				"dari sebagian tanggung jawab selanjutnya atas kerugian yang dipertanggungkan di bawah polis tersebut di atas, baik " +
				"sekarang maupun di masa yang akan datang berkaitan dengan kerugian yang dimaksud, dan melepaskan hak untuk " +
				"mendapatkaan tambahan penggantian klaim lebih lanjut apabila proses klaim sesuai ketentuan di dalam polis tidak " +
				"dipenuhi / dilanjutkan oleh kami, dan selanjutnya menyatakan bahwa tidak ada pihak-pihak lain yang berkepentingan " +
				"terhadap harta benda yang dipertanggungkan.",
			evidenceInterim,
			refund(lawTypo, asmMember, asmMember+"."),
		},
		intro: introTight, recipient: recipientTable, closing: closingTypo,
		place:    "...................., {long}",
		sign:     []string{"Untuk dan atas nama", "Meterai Rp 10.000"},
		roles:    []string{"Jabatan"},
		footnote: footnoteLower,
		survey:   surveyStyle{outer: true, inner: true, centered: true, closing: "- PT. Asuransi Sinar Mas -"},
	},
	// Property - Interim tanpa Co Member
	"2": {
		kopBold2: true, underline: true, signer: signerTight, fields: fieldsTable,
		amount: "Menyatakan setuju dan telah menerima uang sejumlah *{cur} {amt}.- ( {words} )* dari *PT. Asuransi Sinar Mas* " +
			"sebagai *pembayaran interim* atas klaim kerugian / kerusakan perusahaan kami yang dipertanggungkan di bawah polis no " +
			"*{pol}* sehubungan dengan kerugian / kerusakan yang terjadi pada tanggal *{dol}* di *{loc}* .",
		body: []string{
			"Bahwa sehubungan dengan pembayaran klaim interim ini, kami dengan ini membebaskan PT. Asuransi Sinar Mas dari sebagian " +
				"tanggung jawab sesuai jumlah tersebut diatas untuk kerugian yang dipertanggungkan di dalam polis tersebut diatas, " +
				"baik sekarang maupun di masa yang akan datang, dan akan berusaha memenuhi kelengkapan dokumen / bukti-bukti secara " +
				"tertulis atau data lain serta bersedia untuk membantu kelancaran proses klaim dengan Penanggung, untuk mendapatkan " +
				"tambahan klaim lebih lanjut, dan kami juga menyatakan bahwa tidak ada pihak-pihak lain yang berkepentingan terhadap " +
				"harta benda yang dipertanggungkan.",
			refund(lawTypo, asm, asm+" ."),
		},
		intro: introTight, recipient: recipientTable, closing: closingTypo,
		place:  "................, {long}",
		sign:   []string{"Untuk dan atas nama", "Meterai Rp 10.000"},
		roles:  []string{"Jabatan"},
		survey: surveyStyle{outer: true, inner: true, centered: true, closing: "- PT. Asuransi Sinar Mas -"},
	},
	// Property - Final dengan Co Member
	"3": {
		underline: true, signer: signerSpaced, fields: fieldsInline,
		amount: "Menyatakan setuju dan telah menerima uang sejumlah *{cur} {amt} .- ( {words} )* dari :",
		table:  true,
		body: []string{
			"sebagai *pembayaran penuh dan final* atas klaim kerugian / kerusakan yang kami yang ajukan pada polis no. {pol} " +
				"sehubungan dengan kerugian / kerusakan yang terjadi pada tanggal *{dol}*",
			release(asmMember, "bahwa"),
			refund(lawSpaced, asmMember, asm),
		},
		intro: introSpaced, recipient: recipientTable, closing: closing,
		place:         "*Jakarta, {long}*",
		sign:          []string{"~Untuk dan atas nama~", "~Meterai Rp 10.000~"},
		roles:         []string{"Jabatan"},
		footnote:      "Co Member : ^co member berdasarkan share pada polis^",
		footnotePage2: true,
		survey:        surveyPlain,
	},
	// Property - Final tanpa Co Member
	"4": {
		underline: true, signer: signerSpaced, fields: fieldsFar,
		amount: "Menyatakan setuju dan telah menerima uang sejumlah *{cur} {amt}- ( {words} )* dari *PT. Asuransi Sinar Mas* " +
			"sebagai *pembayaran penuh dan final* atas klaim kerugian / kerusakan yang kami yang ajukan pada polis no.{pol} " +
			"sehubungan dengan kerugian / kerusakan yang terjadi pada tanggal *{dol}*.",
		body:  []string{release(asm, "bahwa"), refund(lawSpaced, asm, asm+" .")},
		intro: introSpaced, recipient: recipientTable, closing: closing,
		place:  "*..............., {long}*",
		sign:   []string{"~Untuk dan atas nama~", "~Materai Rp 10.000~"},
		roles:  []string{"Jabatan"},
		survey: surveyPlain,
	},
	// Property - Final Reinstatement tanpa Co Member
	"5": {
		underline: true, signer: signerSpaced, fields: fieldsFar,
		amount: "Menyatakan setuju dan telah menerima uang sejumlah *{cur} {amt}.- ( {words} )* dari *PT. Asuransi Sinar Mas* " +
			"sebagai *pembayaran penuh dan final* atas klaim kerugian / kerusakan yang kami yang ajukan pada polis no.{pol} " +
			"sehubungan dengan kerugian / kerusakan yang terjadi pada tanggal *{dol}* .",
		body:  []string{release(asm, "bahwa"), refund(lawSpaced, asm, asm+" .")},
		intro: introSpaced, recipient: recipientTable, closing: closing,
		place:  "*..............., {short}*",
		sign:   []string{"~Untuk dan atas nama~", "~Meterai Rp 10.000~"},
		roles:  []string{"Jabatan"},
		survey: surveyTight,
	},
	// Properti - Pembayaran Indemnity dan Setelah Tertanggung Reinstatement — dua surat
	"6": {
		kopBold2: true, signer: signerSpaced, fields: fieldsTable,
		amount: "Menyatakan setuju dan telah menerima uang sejumlah *{cur} {amt} .- ( {words} )* secara indemnity basis dari " +
			"*PT. Asuransi Sinar Mas* sebagai pembayaran atas klaim kerugian / kerusakan yang kami yang ajukan pada polis *no. {pol}* " +
			"sehubungan dengan kerugian / kerusakan yang terjadi pada tanggal *{dol}* , di {loc} *(lokasi kejadian)*.",
		body: []string{
			indemnityRelease(asm, "indemnity", "reinstatement", "dipenuihi"),
			indemnityEvidence("PT. Asuransi Sinar Mas ", "reinstatement"),
			refund(lawSpaced, asm, asm+" ."),
		},
		intro: introSpaced, recipient: recipientTable, closing: closing,
		place:  "*{long}*",
		sign:   []string{"Untuk dan atas nama", "Meterai Rp. 10.000,-"},
		roles:  []string{"=Nama=", "Jabatan"},
		survey: surveyStyle{centered: true, closing: "- PT. Asuransi Sinar Mas -"},
		next:   &indemnityCopy2,
	},
	// Marine Cargo - Interim dengan Co Member
	"7": {
		kopBold2: true, signer: signerSpaced, fields: fieldsInline,
		amount: "Menyatakan setuju dan telah menerima uang sejumlah *{cur} {amt} .-  ( {words} )* dari :",
		table:  true,
		body: []string{
			"sebagai *pembayaran interim* atas klaim kerugian perusahaan kami yang dipertanggungkan di bawah polis *no. {pol}* " +
				"sehubungan dengan kerugian yang terjadi pada tanggal *{dol}* Bahwa sehubungan dengan pembayaran interim ini, kami " +
				"dengan ini menyatakan bahwa tidak ada pihak – pihak lain yang berkepentingan dan berhak terhadap harta benda yang " +
				"dipertanggungkan selain kami.",
			subrogation(asmMember, asmMember+". Subrogasi ini tidak akan berakhir oleh hal-hal yang disebut dalam pasal 1813 KUH Perdata"),
			refund("dan/atau ketentuan hukum dan perundang undangan", asmMember, asmMember+"."),
		},
		intro: introSpaced, recipient: recipientTable, closing: closing,
		place:    "*..............., {long}*",
		sign:     []string{"Untuk dan atas nama", "Meterai Rp 10.000,-"},
		roles:    []string{"Jabatan"},
		footnote: "co member : ^co member berdasarkan share pada polis^",
		survey:   surveyCenter,
	},
	// Marine Cargo - Interim tanpa Co Member
	"8": {
		kopBold2: true, kopSmall: true, signer: signerTight, fields: fieldsInline,
		amount: "Menyatakan setuju dan telah menerima uang sejumlah *{cur} {amt}.- ( {words} )* dari *PT. Asuransi Sinar Mas* " +
			"sebagai pembayaran interim atas klaim kerugian perusahaan kami yang dipertanggungkan di bawah polis no. {pol} " +
			"sehubungan dengan kerugian yang terjadi pada tanggal *{dol}*.",
		body: []string{
			interimSolo,
			subrogation(asm, asm+" . Subrogasi ini tidak akan berakhir oleh hal-hal yang disebut dalam pasal 1813 KUH Perdata."),
			refund(lawTight, asm, asm+" ."),
		},
		intro: introSpaced, recipient: recipientTable, closing: closing,
		place:  "*..............., {long}*",
		sign:   []string{"~Untuk dan atas nama~", "~Meterai Rp 10.000,-~"},
		roles:  []string{"Jabatan"},
		survey: surveyCenter,
	},
	// Marine Cargo - Final dengan Co Member
	"9": {
		kopBold2: true, kopSmall: true, signer: signerTight, fields: fieldsInline,
		amount: "Menyatakan setuju dan telah menerima uang sejumlah *{cur} {amt} .- ( {words} )* dari :",
		table:  true,
		body: []string{
			"sebagai pembayaran penuh dan final atas klaim kerugian perusahaan kami yang dipertanggungkan di bawah polis no. {pol} " +
				"sehubungan dengan kerugian yang terjadi pada tanggal *{dol}* .",
			release(asmMember, "bahwa"),
			subrogation(asmMember, asmMember+". Subrogasi ini tidak akan berakhir oleh hal-hal yang disebut dalam pasal 1813 KUH Perdata"),
			refund(lawTight, asmMember, asmMember+"."),
		},
		intro: introSpaced, recipient: recipientTight, closing: closing,
		place:    "..............., {long}",
		sign:     []string{"Untuk dan atas nama", "Meterai Rp 10.000,-"},
		roles:    []string{"Jabatan"},
		footnote: "*co member :* ^co member berdasarkan share pada polis^",
		survey:   surveyStyle{outer: true, inner: true, centered: true, closing: "- PT. Asuransi Sinar Mas -"},
	},
	// Marine Cargo - Final tanpa Co Member
	"10": {
		kopBold2: true, signer: signerSpaced, fields: fieldsTable,
		amount: "Menyatakan setuju dan telah menerima uang sejumlah *{cur} {amt} -( {words} )* dari *PT. Asuransi Sinar Mas* " +
			"sebagai *pembayaran penuh dan final* atas klaim kerugian perusahaan kami yang dipertanggungkan di bawah polis " +
			"*no. {pol}* sehubungan dengan kerugian yang terjadi pada tanggal *{dol} DOL*.",
		body: []string{
			release(asm, "bahwan"),
			subrogation(asm, asm+" . Subrogasi ini tidak akan berakhir oleh hal-hal yang disebut dalam pasal 1813 KUH Perdata."),
			refund("dan/atau ketentuan hukum dan perundang- undangan", asm, asm+" ."),
		},
		intro: introSpaced, recipient: recipientTable, closing: closing,
		place:  "*..............., {long}*",
		sign:   []string{"Untuk dan atas nama", "Meterai Rp 10.000,-"},
		roles:  []string{"Jabatan"},
		survey: surveyCenter,
	},
	// Pembayaran Final Tanpa Co Member Ex Gratia
	"12": {
		kopBold2: true, kopSmall: true, signer: signerTight, fields: fieldsTight, address: true,
		amount: "Menyatakan setuju dan telah menerima uang sejumlah {cur} {amt}.- ( {words} ) dari : *PT. Asuransi Sinar Mas* " +
			"sebagai pembayaran penuh dan final secara ex-gratia atas klaim kerugian perusahaan kami yang dipertanggungkan di " +
			"bawah polis no. *{pol}* sehubungan dengan kerugian yang terjadi pada tanggal *{dol}*.",
		body:  []string{release(asm, "bahwa"), refund(lawTight, asm, asm+" .")},
		intro: introSpaced, recipient: recipientTable, closing: closing,
		place:  "Jakarta, {long}",
		sign:   []string{"Untuk dan atas nama", "Meterai Rp 10.000,-"},
		roles:  []string{"Jabatan"},
		survey: surveyFramed,
	},
	// Property - Indemnity Terlebih Dahulu Dengan Member
	"13": {
		underline: true, signer: signerSpaced, fields: fieldsTable,
		amount: "Menyatakan setuju dan telah menerima uang sejumlah *{cur} {amt}.- ( {words} )* secara ~indemnity~ basis dari :",
		table:  true,
		body: []string{
			"sebagai pembayaran atas klaim kerugian / kerusakan yang kami yang ajukan pada polis no.{pol} sehubungan dengan " +
				"kerugian / kerusakan yang terjadi pada tanggal *{dol}*, di *{loc}*.",
			indemnityRelease(asmMemberDash, "~indemnity~", "~reinstatement~", "dipenuhi"),
			indemnityEvidence("PT Asuransi Sinar Mas dan co-member", "~reinstatement~"),
			refund(lawSpaced, asmMemberDash, asmMemberDash+"."),
		},
		intro: introSpaced, recipient: recipientBranch, closing: closing,
		place:  "*..............., {short}*",
		sign:   []string{"~Untuk dan atas nama~", "~Meterai Rp 10.000~"},
		roles:  []string{"Direktur"},
		survey: surveyTight,
	},
	// Property - Indemnity Terlebih Dahulu Tanpa Member
	"14": {
		underline: true, signer: signerSpaced, fields: fieldsTable,
		amount: "Menyatakan setuju dan telah menerima uang sejumlah *{cur} {amt}.- ( {words} )* secara ~indemnity~ basis dari " +
			"*PT ASURANSI SINAR MAS* sebagai pembayaran atas klaim kerugian / kerusakan yang kami yang ajukan pada polis no.{pol} " +
			"sehubungan dengan kerugian / kerusakan yang terjadi pada tanggal *{dol}* , di *{loc}*.",
		body: []string{
			indemnityRelease(asm, "~indemnity~", "~reinstatement~", "dipenuhi"),
			indemnityEvidence("PT Asuransi Sinar Mas", "~reinstatement~"),
			refund(lawSpaced, asm, asm+" ."),
		},
		intro: introSpaced, recipient: recipientBranch, closing: closing,
		place:  "*..............., {short}*",
		sign:   []string{"~Untuk dan atas nama~", "~Meterai Rp 10.000~"},
		roles:  []string{"Direktur"},
		survey: surveyTight,
	},
	// Pembayaran Final dengan Co Member Ex Gratia
	"15": {
		kopBold2: true, kopSmall: true, signer: signerTight, fields: fieldsTight, address: true,
		amount: "Menyatakan setuju dan telah menerima uang sejumlah {cur} {amt}.- ( {words} ) dari :",
		table:  true,
		body: []string{
			"sebagai pembayaran penuh dan final secara ex-gratia atas klaim kerugian perusahaan kami yang dipertanggungkan di " +
				"bawah polis no. *{pol}* sehubungan dengan kerugian yang terjadi pada tanggal *{dol}*.",
			release(asmMemberDash, "bahwa"),
			refund(lawTight, asmMemberDash, asmMemberDash+"."),
		},
		intro: introSpaced, recipient: recipientTable, closing: closing,
		place:  "Jakarta, {long}",
		sign:   []string{"Untuk dan atas nama", "Meterai Rp 10.000,-"},
		roles:  []string{"Jabatan"},
		survey: surveyFramed,
	},
}
