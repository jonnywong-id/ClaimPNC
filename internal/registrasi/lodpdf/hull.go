package lodpdf

// LOD jenis 11 "Marine Hull - LOD", disalin dari `Sample Form/LOD Marine Hull1.pdf`: surat
// dwibahasa dua kolom (Inggris | Indonesia) di atas kop perusahaan tertanggung, tiga halaman
// surat ditambah Indeks Kepuasan Pelanggan.
//
// Dua hal pada contoh disalin apa adanya dan menunggu konfirmasi Work Owner: isian Policy
// Number, Subject Matter Insured, Claim, dan Date of Loss KOSONG, dan kalimat pembuka
// menulis "the sum of IDR from" / "sejumlah dari" tanpa nilai. Nilainya tercetak pada
// "Claim Ammount" dan tabel co member.

// hullPairs adalah pasangan paragraf halaman pertama dan kedua.
var hullPairs = [][][2]string{
	{
		{"We, the undersigned, are the parties who have received the sum of {cur} from PT. Asuransi Sinar Mas and Co Member " +
			"(hereinafter referred to as *“the Insurer”* )",
			"Kami, yang bertanda tangan di bawah ini, menyatakan bahwa kami adalah pihak yang telah menerima uang sejumlah dari " +
				"PT. Asuransi Sinar Mas dan Co Member (selanjutnya disebut sebagai *“Penanggung”* )"},
		{"By receiving the Claim Settlement above, we agree not to file any demands or claim against the insurer that may occur " +
			"presently or in the future in respect of the aforementioned Claim Settlement.",
			"Dengan menerima Pembayaran Klaim di atas, maka kami sepakat untuk tidak akan mengajukan tuntutan atau gugatan apapun " +
				"terhadap Penanggung yang mungkin terjadi saat ini maupun di kemudian hari sehubungan dengan Pembayaran Klaim tersebut"},
		{"We hereby declare that we are the rightful claimant and are legally entitled to receive the Claim Settlement mentioned above.",
			"Kami dengan ini menyatakan bahwa kami adalah pihak yang sah dan berhak secara hukum untuk menerima Pembayaran Klaim " +
				"tersebut di atas."},
		{"We are willing to provide appropriate assistance (as far as reasonably possible) to the Insurer in taking action " +
			"reasonably necessary for all efforts made by the Insurer against other parties regarding this Claim Settlement, " +
			"including lawsuits, provided that the Insurer releases us from liability for, and reimburses us for, all fees, costs " +
			"and expenses to carry out the action or claim.",
			"Kami bersedia memberi bantuan yang patut (sejauh memungkinkan) kepada Penanggung dalam melakukan setiap tindakan yang " +
				"diperlukan atas segala upaya yang dilakukan oleh Penanggung terhadap pihak lain berkenaan dengan Pembayaran Klaim " +
				"ini, termasuk untuk melakukan gugatan hukum, sepanjang Penanggung membebaskan kami dan memberi penggantian kepada " +
				"kami atas seluruh ongkos, biaya, dan beban untuk melakukan tindakan atau gugatan tersebut."},
	},
	{
		{"If in the future, based a final and binding court judgement in Indonesia, it is found by the Insurer that the Insurer " +
			"is not liable for the full Claim Settlement, and/or that the amount of actual loss we suffered is lower than the " +
			"amount of the claim (“ *Overpayment*”) then we agree and are willing to return the Overpayment to the Insurer within " +
			"21 business days of the receipt of such court judgement by us (or such earlier time stipulated by the court).",
			"Apabila dikemudian hari, berdasarkan putusan pengadilan yang final dan mengikat di Indonesia, Penanggung menemukan " +
				"bahwa tidak ada tanggung jawab Penanggung atas Pembayaran Klaim ini, dan/atau ternyata jumlah kerugian yang " +
				"sebenarnya dialami oleh kami ternyata lebih rendah daripada jumlah klaim (“ *Kelebihan Pembayaran*”), maka kami " +
				"setuju dan bersedia mengembalikan Kelebihan Pembayaran kepada Penanggung dalam jangka waktu 21 hari kerja sejak " +
				"diterimanya putusan pengadilan tersebut oleh kami (atau dalam jangka waktu lebih awal sebagaimana ditentukan oleh " +
				"pengadilan)."},
		{"This letter may be executed in any number of counterparts, and this has the same effect as if the signatures of the " +
			"counterparts were on a single copy of this letter. The date of this letter will be the date on which the last " +
			"counterpart is signed.",
			"Surat ini dapat ditandatangani dalam beberapa salinan dan akan memiliki dampak yang sama sebagaimana apabila surat ini " +
				"ditandatangani pada satu salinan tanggal. Tanggal dari surat ini adalah tanggal dimana salinan terakhir " +
				"ditandatangani."},
	},
}

func marineHull(w *writer) {
	pdf := w.pdf
	heading := func() {
		pdf.SetFont(font, "B", 10)
		pdf.CellFormat(0, 5, "COMPANY LETTER HEAD", "", 1, "C", false, 0, "")
	}
	title := func() {
		pdf.SetFont(font, "BU", 10)
		pdf.CellFormat(0, 5, "SURAT PERNYATAAN", "", 1, "C", false, 0, "")
		pdf.SetFont(font, "B", 10)
		pdf.CellFormat(0, 5, "(Letter of Discharge)", "", 1, "C", false, 0, "")
		w.gap(6)
	}

	pdf.AddPage()
	heading()
	w.gap(6)
	for _, l := range []string{"To: PT. Asuransi Sinar Mas", "     JL. KH FACHRUDIN NO 18", "     JAKARTA 10250"} {
		w.para(l)
	}
	w.gap(5)
	w.pdf.SetFont(font, "", body)
	for _, row := range [][2]string{
		{"Insured", w.vals["insured"]},
		{"Policy Number", ""},
		{"Subject Matter Insured", ""},
		{"Claim", ""},
		{"Date of Loss", ""},
		{"Claim Ammount", w.fill("{cur} {amt}.-")},
	} {
		pdf.CellFormat(45, 5.5, w.tr(row[0]), "", 0, "L", false, 0, "")
		pdf.MultiCell(w.text-45, 5.5, w.tr(": "+row[1]), "", "L", false)
	}
	w.gap(4)
	w.members()
	w.gap(6)
	title()
	w.columns(hullPairs[0])

	pdf.AddPage()
	heading()
	w.gap(4)
	title()
	w.columns(hullPairs[1])
	w.para("The above amount will be transfered to the below account;")
	w.recipient(recipientTable)

	pdf.AddPage()
	w.para("*Location and Date*")
	w.gap(2)
	w.para("(JAKARTA, {long} )")
	w.gap(4)
	w.center("*Name, Signature and Company Stamp*")
	w.gap(16)
	w.para("({insured} )")

	w.survey(surveyPlain)
}

// columns menulis pasangan paragraf Inggris | Indonesia berdampingan.
func (w *writer) columns(pairs [][2]string) {
	pdf := w.pdf
	colW := (w.text - 8) / 2
	for _, pair := range pairs {
		top := pdf.GetY()
		bottom := top
		for i, tpl := range pair {
			x := margin + float64(i)*(colW+8)
			pdf.SetLeftMargin(x)
			pdf.SetRightMargin(w.width - x - colW)
			pdf.SetXY(x, top)
			w.flow(w.parse(tpl))
			pdf.Ln(line)
			if y := pdf.GetY(); y > bottom {
				bottom = y
			}
		}
		pdf.SetLeftMargin(margin)
		pdf.SetRightMargin(margin)
		pdf.SetXY(margin, bottom)
		w.gap(5)
	}
}
