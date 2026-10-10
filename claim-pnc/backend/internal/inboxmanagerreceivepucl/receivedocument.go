package inboxmanagerreceivepucl

import "strings"

// Bentuk LAYAR KERJA penerimaan dokumen — flow action `InputReceiveDocument`.
//
// # Layar apa ini, dan bagaimana ia terbuka
//
// Di Pega, sel "CaseID" pada kedua grid Receive bukan teks melainkan TAUTAN
// (`pyUIElement = link`, `pyLabel = .pyID`). Mengkliknya menjalankan tiga perilaku
// berurutan, terbaca apa adanya di `Section/InboxManagerReceive_Section-Section.xml`:
//
//	1. runActivity      SetAssignmentInboxReceive_act, parameter `kunci = .pzInsKey`
//	2. refresh          thisSection
//	3. openAssignment   pyInsKey = TempIns.pyNote
//
// `Activity/SetAssignmentInboxReceive_act-Act.xml` sendiri NYARIS KOSONG: `pyUsage = FLOW`,
// tanpa satu pun method, dengan satu penetapan ke `TempIns.pyNote`. Ia kait pra-proses; yang
// bekerja adalah **Open Assignment** bawaan Pega, yang membuka berkas penerimaan dokumen pada
// tahap alur kerjanya saat itu — dan flow action yang menunggu di sana adalah
// `InputReceiveDocument` (`Flow/InputReceiveDocument.xml`, kelas
// `ASM-FW-GCNMFW-Work-ReceiveDocument`).
//
// Pola yang sama persis sudah ditempuh modul Inbox RCL/PUCL, yang di sana memakai
// `SetAssignmentInboxPUCL_act` dengan parameter `inskey`.
//
// # Dari mana judul dan susunannya
//
// Dibaca dari `Section/InputReceiveDocument-Section.xml` (`pyRuleName = InputReceiveDocument`,
// `pyClassName = ASM-FW-GCNMFW-Work-ReceiveDocument`) — dari `pyLabelFieldValue` tiap sel
// APA ADANYA (`D-13`), berpasangan dengan `pyValue` sel yang sama sebagai properti Pega-nya.
//
// # Temuan yang menentukan bentuk modul ini: SELURUHNYA read-only di Pega
//
// Ke-25 sel ber-properti `.ReceiveDocument.*`, `.Policy.*`, dan `.pxCreateDateTime` pada
// section itu bertanda `<pyReadOnly>true</pyReadOnly>` TANPA KECUALI. Yang bertanda
// `false` hanyalah blok data pelapor (`.ReportHE.*`) dan blok alamatnya (`tempTES.*`).
//
// Artinya layar ini di Pega pun MENAMPILKAN berkas penerimaan dokumen, tidak menyuntingnya;
// yang mengubah data adalah TOMBOL-nya, bukan isiannya. Lihat WriteAction.
//
// # Dari mana ISINYA dibaca
//
// Dua tabel, dan pembagiannya bukan pilihan gaya:
//
//	DATAPEGA.PC_ASM_FW_GCNMFW_WORK    kolom objek kerja yang memang ada
//	POOLDATA.T_CLAIM_RECIVEDCLAIM     tabel cermin berisi isian layar ini
//
// Tabel cermin itulah yang memasok sebagian besar isian, dan nama kolomnya TERVERIFIKASI —
// dibaca dari pernyataan `update` di `Database/PROCINSERTDATARECIVEDKLAIM.prc`, procedure
// yang mengisinya. Ia bukan tebakan dan bukan pula pembacaan DDL, yang memang belum tersedia
// (`R-08`).
//
// Risiko yang sama dengan grid-nya berlaku di sini dan disebut apa adanya: tabel cermin itu
// TIDAK PERNAH DIBACA sistem lama — satu-satunya penyentuhnya adalah procedure yang
// menulisinya — sehingga kelengkapan isinya belum terverifikasi. Gabungannya karena itu
// `LEFT JOIN`, dan berkas tanpa pasangan tetap terbuka dengan isian kosong alih-alih
// dinyatakan tidak ada.
//
// # Isian yang DIGAMBAR tetapi belum dapat diisi
//
// Enam belas isian tidak punya sumber yang dapat dibaca SQL, dan seluruhnya bertanda
// Blocked. Ia BUKAN disembunyikan: menyembunyikannya membuat pengguna yang membandingkan
// layar ini dengan Pega mengira isiannya hilang, sementara menggambarnya dengan alasan
// membuat ia tahu isiannya ada dan kenapa kosong. Itu pula yang dituntut uji kesetaraan
// gerbang 1 — layar yang tampak setara padahal belum, adalah yang paling berbahaya.

// Field adalah satu isian pada layar kerja penerimaan dokumen.
type Field struct {
	// Key adalah nama isian pada kontrak API. Ia kontrak yang dibaca layar, sehingga tetap
	// berbahasa Indonesia (`D-80`).
	Key string

	// Title adalah judul yang dibaca pengguna, mengikuti Pega apa adanya (`D-13`).
	Title string

	// Multiline menandai isian yang di Pega digambar sebagai kotak teks bertingkat, bukan
	// satu baris. Layar memakainya untuk memilih bentuk selnya.
	Multiline bool

	// Blocked menyatakan isian ini digambar tetapi belum dapat diisi.
	Blocked bool

	// BlockedReason dan BlockedOwner menjelaskan penghalangnya beserta alamatnya.
	//
	// Keduanya diisi oleh pembentuknya, bukan ditulis ulang pada setiap isian, supaya isian
	// yang terhalang oleh sebab yang sama tidak dapat menyebut alasan yang berbeda-beda.
	// Penghalang tanpa pemilik tidak pernah hilang (`D-36`).
	BlockedReason string
	BlockedOwner  string

	// VisibleWhen adalah syarat tampil isian ini, diterjemahkan dari `<pyCondition>` pada
	// sel yang sama di section.
	//
	// Kosong berarti selalu tampil. Lihat VisibilityRule untuk daftar dan buktinya.
	VisibleWhen VisibilityRule
}

// VisibilityRule adalah syarat tampil sebuah isian pada layar kerja.
//
// # Kenapa ini ada
//
// Karena layar lama TIDAK menggambar seluruh isian pada setiap berkas. Dari 24 isian yang
// modul ini bawa, **14 punya `<pyCondition>`** pada selnya. Versi sebelumnya menggambar
// semuanya tanpa syarat, sehingga layar kita menampilkan isian yang di Pega tidak ada —
// dilaporkan Work Owner 2026-10-10 saat membandingkan kedua layar berdampingan.
//
// # Kenapa berupa kode, bukan fungsi
//
// Supaya bentuk layar tetap berupa DATA yang dapat dibandingkan, diuji, dan dikirim apa
// adanya. Fungsi di dalam senarai paket membuat `DocumentFieldGroups` tidak dapat
// dibandingkan dengan `require.Equal`, dan uji yang membandingkannya sudah ada.
//
// # Yang BELUM dapat diterjemahkan
//
// Sepuluh dari keempat belas syarat itu bergantung pada nilai yang belum kita punya, dan
// keduanya bukan soal kueri melainkan soal artefak yang hilang:
//
//	syarat                                            butuh                    penghalang
//	------------------------------------------------- ------------------------ ----------
//	.ReceiveDocument.pyLabel == / != '…Eksternal'      kolom PYLABEL            R-08
//	OperatorID.pyPosition == 'BROKER' / 'NONMBU'       jabatan operator login   F-3
//
// `PYLABEL` **nol kemunculan** sebagai kolom di seluruh export — tidak satu pun dari ratusan
// kueri yang membaca tabel objek kerja menyentuhnya, sehingga keberadaannya tidak terbukti
// dan menebaknya menghasilkan kueri yang gagal saat pertama dijalankan di produksi.
// `OperatorID.pyPosition` menuntut profil operator yang login, dan itu milik `F-3` yang
// kontrak HCC/HCQ-nya belum ada.
//
// Isian yang syaratnya belum dapat diterjemahkan tetap **digambar tanpa syarat**, sama
// seperti sebelumnya. Itu pilihan yang disengaja: menyembunyikannya berdasarkan tebakan
// akan menghilangkan isian yang seharusnya terlihat, dan isian yang hilang jauh lebih sulit
// dilaporkan daripada isian yang kelebihan.
type VisibilityRule string

const (
	// VisibleAlways berarti tidak ada `<pyCondition>` pada selnya.
	VisibleAlways VisibilityRule = ""

	// VisibleWhenPolicyAndNotPAOrTravel menerjemahkan
	// `.ReceiveDocument.PolicyNo != '' && .Policy.Quotation.GroupPanel != '002'
	// && .Policy.Quotation.GroupPanel != '005'`.
	//
	// `002` Personal Accident dan `005` Travel (`CONTEXT.md`). Keduanya dikecualikan karena
	// tidak punya polis leader maupun broker.
	VisibleWhenPolicyAndNotPAOrTravel VisibilityRule = "polis-ada-bukan-pa-travel"

	// VisibleWhenPolicyAndPA menerjemahkan
	// `.ReceiveDocument.PolicyNo != '' && .Policy.Quotation.GroupPanel == '002'`.
	VisibleWhenPolicyAndPA VisibilityRule = "polis-ada-dan-pa"

	// VisibleWhenPA menerjemahkan `pyWorkPage.Policy.Quotation.GroupPanel == '002'`.
	//
	// Perhatikan ia TIDAK menuntut nomor polis terisi, berbeda dari dua syarat di atas.
	// Itu keadaan di section apa adanya, bukan penyeragaman di sini.
	VisibleWhenPA VisibilityRule = "pa"
)

// visible menjawab apakah sebuah isian digambar untuk berkas ini.
//
// Nilai pembandingnya diambil dari berkas yang sedang dibuka, bukan dari clipboard Pega:
// Group Panel dari `GROUPPANEL_1` dan nomor polis dari `POLICYNO`, keduanya sudah dibaca
// kueri `detail_receive_document`.
func (f Field) visible(d ReceiveDocument) bool {
	panel := strings.TrimSpace(d.GroupPanel)
	hasPolicy := strings.TrimSpace(d.PolicyNumber) != ""

	switch f.VisibleWhen {
	case VisibleWhenPolicyAndNotPAOrTravel:
		return hasPolicy && panel != GroupPanelPA && panel != GroupPanelTravel
	case VisibleWhenPolicyAndPA:
		return hasPolicy && panel == GroupPanelPA
	case VisibleWhenPA:
		return panel == GroupPanelPA
	default:
		return true
	}
}

// FieldGroup adalah satu kelompok isian, digambar sebagai satu panel.
//
// Pengelompokannya mengikuti urutan sel di section, bukan disusun ulang menurut selera:
// petugas yang membandingkan kedua layar berdampingan membaca isian pada urutan yang sama.
type FieldGroup struct {
	Title  string
	Fields []Field
}

// Kunci isian layar kerja.
//
// Dikumpulkan sebagai konstanta supaya judul di berkas ini, penyusun nilai di valuesOf,
// penyusun DTO, dan penggambar sel di layar tidak dapat berselisih tanpa ketahuan.
const (
	DocFieldCreatedAt        = "tanggal_input_dokumen"
	DocFieldReceivedAt       = "tanggal_terima_dokumen"
	DocFieldSenderName       = "nama_pengirim"
	DocFieldSenderEmail      = "email_pengirim"
	DocFieldSenderPhone      = "telepon_pengirim"
	DocFieldCourierName      = "nama_kurir_asm"
	DocFieldInsuredName      = "nama_tertanggung"
	DocFieldPolicyNumber     = "nomor_polis"
	DocFieldLossDate         = "tanggal_kejadian"
	DocFieldPolicyLeader     = "polis_leader"
	DocFieldBusinessName     = "nama_bisnis"
	DocFieldReferenceNumber  = "no_referensi"
	DocFieldLossEstimate     = "estimasi_kerugian"
	DocFieldBrokerReference  = "no_ref_broker"
	DocFieldSourceOfReports  = "source_of_reports"
	DocFieldInsuredEmail     = "email_tertanggung"
	DocFieldLossLocation     = "lokasi_kejadian"
	DocFieldAdjusterNote     = "rekomendasi_adjuster"
	DocFieldDriverLicence    = "sim_pengendara"
	DocFieldChronology       = "kronologis_kejadian"
	DocFieldDamageDetail     = "rincian_kerusakan"
	DocFieldTransferReason   = "alasan_belum_transfer"
	DocFieldTransferNote     = "keterangan_belum_transfer"
	DocFieldEmailSubject     = "subjek_email"
	DocFieldNotRegisteredNot = "keterangan_belum_registrasi"
	DocFieldDocumentTotal    = "total_jumlah_dokumen"

	DocFieldReporterIDCard   = "pelapor_no_ktp"
	DocFieldReporterName     = "pelapor_nama_lengkap"
	DocFieldReporterPosition = "pelapor_jabatan"
	DocFieldReporterCompany  = "pelapor_nama_perusahaan"
	DocFieldReporterJob      = "pelapor_pekerjaan"
	DocFieldReporterCountry  = "pelapor_negara"
	DocFieldReporterProvince = "pelapor_provinsi"
	DocFieldReporterCity     = "pelapor_kota"
	DocFieldReporterDistrict = "pelapor_kabupaten"
	DocFieldReporterVillage  = "pelapor_kelurahan"
	DocFieldReporterPostcode = "pelapor_kodepos"
)

// Dua sebab terhalang, dan keduanya berbeda.
//
// Membedakannya penting bagi yang akan menghilangkannya: yang pertama menuntut kolomnya
// DIBUKA di Pega (properti yang hidup di dalam blob objek kerja), yang kedua menuntut
// isiannya ada di suatu tempat sama sekali.
const (
	blockedUnexposed = "Isian ini hidup di dalam blob objek kerja Pega dan tidak punya " +
		"kolom basis data, sehingga tidak dapat dibaca kueri biasa selama objek kerjanya " +
		"masih dimiliki Pega. Ia digambar di tempatnya supaya ketiadaannya terlihat, bukan " +
		"tersamar sebagai isian yang memang belum diisi."

	blockedNoSource = "Tidak ada satu pun kolom untuk isian ini di seluruh export, dan " +
		"tabel penerimaan dokumen tidak menyimpannya. Menebak nama kolomnya menghasilkan " +
		"kueri yang gagal saat pertama dijalankan di produksi — jauh lebih mahal daripada " +
		"satu isian yang kosong dan dijelaskan."

	blockedStaleSection = "Isian ini ada di layar Pega yang berjalan tetapi TIDAK ADA di " +
		"export section yang kita punya, sehingga properti dan kolomnya belum diketahui. " +
		"Yang dibutuhkan bukan kolom basis data melainkan export ulang " +
		"`Rule-Obj-Section` `InputReceiveDocument` versi terkini."

	blockedOwnerPega = "Tim Pega + DBA"
)

// blocked membentuk isian yang digambar tetapi belum dapat diisi.
func blocked(key, title, reason string) Field {
	return Field{
		Key:           key,
		Title:         title,
		Blocked:       true,
		BlockedReason: reason,
		BlockedOwner:  blockedOwnerPega,
	}
}

// DocumentFieldGroups adalah susunan isian layar kerja, berurutan seperti di Pega.
//
// # Judulnya DIAMBIL dari section, bukan dikarang
//
// Versi sebelumnya memakai lima judul karangan — "Penerimaan Dokumen", "Data Polis",
// "Kejadian", "Transfer dan Registrasi", "Data Pelapor" — dengan alasan tertulis bahwa
// "section itu memang tidak punya judul panel satu pun". **Itu salah.**
// `Section/InputReceiveDocument-Section.xml` punya tepat enam `<pyTitle>`, dan tidak satu pun
// dari kelima judul karangan itu ada di sana. Dilaporkan Work Owner 2026-10-10 dan dicabut.
//
// Keenam judul itu, beserta rentang blok XML-nya — dihitung dengan menyeimbangkan
// `<rowdata>` pembuka dan penutupnya, bukan ditebak dari jarak teks:
//
//	judul                        rentang blok            peran
//	---------------------------- ----------------------- -------------------------------
//	Data Pelaporan Klaim          151866 .. 1564427       PEMBUNGKUS SELURUH layar
//	Data Tertanggung Klaim        448447 ..  526549       blok di dalamnya
//	Alamat                        533044 ..  644780       blok lipat (COLLAPSIBLE)
//	Daftar Penerimaan Dokumen     820886 ..  994948       blok di dalamnya
//	Progress Pelaporan           1303518 .. 1403866       blok di dalamnya
//	History Komunikasi           1446410 .. 1557534       blok di dalamnya
//
// Yang pertama membungkus kelima sisanya. Itu sebabnya layar Pega menampilkannya sebagai
// SATU judul di puncak layar — persis seperti yang terlihat pada tangkapan layar Work Owner.
//
// # Bagaimana tiap isian ditempatkan
//
// Secara mekanis: sebuah isian masuk ke blok yang rentangnya memuat posisi selnya. Tidak ada
// penilaian selera di sini, dan hasilnya dapat dihitung ulang kapan saja.
//
//	isian                              posisi sel   blok
//	---------------------------------- ------------ -------------------------
//	Tanggal Input Dokumen … SIM Peng.    255709..423000  Data Pelaporan Klaim
//	No KTP … Pekerjaan                   473946..518053  Data Tertanggung Klaim
//	Kronologis … Subjek Email            766057..797574  Data Pelaporan Klaim
//	Ket. Belum Registrasi, Total Dok.   1269222..1291045 Data Pelaporan Klaim
//
// Perhatikan dua baris terakhir: keduanya berada SETELAH blok "Alamat" tertutup di 644780
// dan DI LUAR blok "Daftar Penerimaan Dokumen", sehingga keduanya kembali menjadi milik
// pembungkus. Pemeriksaan jarak teks saja akan menempatkannya keliru — itu sebabnya
// penempatannya dihitung dari sarang XML.
//
// # Urutan isian TIDAK berubah
//
// Urutan yang dipakai versi sebelumnya ternyata sudah sama persis dengan urutan sel di
// section. Yang berubah hanya pengelompokan dan judulnya.
//
// # Yang tidak digambar
//
// Tiga blok — "Daftar Penerimaan Dokumen", "Progress Pelaporan", "History Komunikasi" —
// tidak punya satu pun isian yang modul ini baca, sehingga ketiganya tidak digambar.
// Keduanya berisi grid dan riwayat yang sumbernya belum dibangun.
var DocumentFieldGroups = []FieldGroup{
	{
		// Judul ini ADA di layar lama, apa adanya — `<pyTitle>Data Pelaporan Klaim</pyTitle>`.
		Title: "Data Pelaporan Klaim",
		Fields: []Field{
			{Key: DocFieldCreatedAt, Title: "Tanggal Input Dokumen"},
			{Key: DocFieldReceivedAt, Title: "Tanggal Terima Dokumen"},
			{Key: DocFieldSenderName, Title: "Nama Pengirim / Pelapor Dokumen"},
			{Key: DocFieldSenderEmail, Title: "Email Pengirim"},
			{Key: DocFieldSenderPhone, Title: "No. HP Pengirim"},
			{Key: DocFieldCourierName, Title: "Nama Kurir ASM"},

			// Sel ini kehilangan judulnya di Pega — `pyLabelFieldValue` berisi label bawaan
			// kontrol ("Text Input"), yang tidak menyatakan apa pun. Yang dipakai adalah
			// judul kolom grid untuk properti yang sama (`.ReceiveDocument.QQName` →
			// "Nama Tertanggung"), supaya berkas dan layar menyebut hal yang sama dengan
			// kata yang sama.
			{Key: DocFieldInsuredName, Title: "Nama Tertanggung"},

			{Key: DocFieldPolicyNumber, Title: "Nomor Polis"},
			{Key: DocFieldLossDate, Title: "Tanggal Kejadian"},

			withVisibility(
				blocked(DocFieldPolicyLeader, "Polis Leader", blockedUnexposed),
				VisibleWhenPolicyAndNotPAOrTravel),
			blocked(DocFieldBusinessName, "Nama Bisnis", blockedUnexposed),

			{Key: DocFieldReferenceNumber, Title: "No. Referensi/Placing Slip"},

			blocked(DocFieldLossEstimate, "Estimasi Kerugian", blockedUnexposed),
			withVisibility(
				blocked(DocFieldBrokerReference, "No Ref Broker", blockedUnexposed),
				VisibleWhenPolicyAndNotPAOrTravel),
			blocked(DocFieldSourceOfReports, "Source Of Reports", blockedUnexposed),

			{
				Key:         DocFieldInsuredEmail,
				Title:       "Email Tertanggung",
				VisibleWhen: VisibleWhenPolicyAndPA,
			},

			{Key: DocFieldLossLocation, Title: "Lokasi Kejadian"},

			// Isian ini ADA di layar Pega yang berjalan — terlihat pada tangkapan layar
			// Work Owner 2026-10-10, berpasangan dengan "Lokasi Kejadian" dan berupa kotak
			// teks bertingkat — tetapi **nol kemunculan** di
			// `Section/InputReceiveDocument-Section.xml` yang kita punya.
			//
			// Itu bukan isian yang terlewat dibaca: ia bukti bahwa export section kita
			// LEBIH TUA daripada layar yang berjalan. Bukti keduanya pada berkas yang sama:
			// tombol "Upload Email" juga nol kemunculan, padahal ia tergambar di layar.
			//
			// Digambar di tempatnya supaya selisihnya terlihat, bukan tersamar sebagai
			// isian yang memang tidak ada.
			blocked(DocFieldAdjusterNote, "Rekomendasi Loss Adjuster/Surveyor",
				blockedStaleSection),

			// Pega memberi sel ini judul "Lokasi Kejadian" pula — dua sel berturut-turut
			// dengan judul yang sama persis, padahal propertinya berbeda
			// (`.ReceiveDocument.LokasiKejadian` dan `.ReceiveDocument.SIM`). Itu
			// kekeliruan penamaan di Pega, bukan kesengajaan.
			//
			// Judulnya di sini DIPERBAIKI menjadi "SIM Pengendara", mengikuti nama parameter
			// procedure yang menyimpannya (`tSIMPENGENDARA`). Dua isian berjudul sama persis
			// pada satu layar bukan "tampilan yang meniru Pega" melainkan tampilan yang
			// tidak dapat dibaca, dan `D-13` tidak menuntut membawa kekeliruan penamaan.
			// Selisihnya dinyatakan ke pengguna, bukan disembunyikan.
			{
				Key:         DocFieldDriverLicence,
				Title:       "SIM Pengendara",
				VisibleWhen: VisibleWhenPA,
			},

			{Key: DocFieldChronology, Title: "Kronologis Kejadian", Multiline: true},
			{Key: DocFieldDamageDetail, Title: "Rincian Kerusakan", Multiline: true},

			{Key: DocFieldTransferReason, Title: "Alasan Belum Transfer", Multiline: true},

			// Di Pega, sel ini dan sel di atasnya menunjuk properti yang SAMA
			// (`.ReceiveDocument.Keterangan`) dengan dua judul berbeda, sehingga keduanya
			// menampilkan isi yang sama persis. Keadaan itu dibawa apa adanya — lihat
			// valuesOf — dan dinyatakan ke pengguna.
			{Key: DocFieldTransferNote, Title: "Keterangan Belum Transfer", Multiline: true},

			{Key: DocFieldEmailSubject, Title: "Subjek Email"},
			{
				Key:       DocFieldNotRegisteredNot,
				Title:     "Keterangan Belum Registrasi",
				Multiline: true,
			},

			blocked(DocFieldDocumentTotal, "Total Jumlah Dokumen", blockedNoSource),
		},
	},
	{
		// Judul ini ADA di layar lama, apa adanya —
		// `<pyTitle>Data Tertanggung Klaim</pyTitle>`, blok 448447..526549.
		//
		// SELURUH isian di sini terhalang, dan justru blok inilah satu-satunya yang di Pega
		// dapat DISUNTING (`pyReadOnly=false`). Kedua hal itu berkaitan langsung: isian yang
		// disunting petugas disimpan di dalam blob objek kerja, dan blob tidak punya kolom
		// yang dapat dibaca maupun ditulis SQL.
		//
		// Itu pula sebab tombol tulis layar ini belum dapat dihidupkan; lihat WriteAction.
		Title: "Data Tertanggung Klaim",
		Fields: []Field{
			blocked(DocFieldReporterIDCard, "No KTP", blockedUnexposed),
			blocked(DocFieldReporterName, "Nama Lengkap", blockedUnexposed),
			blocked(DocFieldReporterPosition, "Jabatan", blockedUnexposed),
			blocked(DocFieldReporterCompany, "Nama Perusahaan", blockedUnexposed),
			blocked(DocFieldReporterJob, "Pekerjaan", blockedUnexposed),
		},
	},
	{
		// Judul ini ADA di layar lama, apa adanya — `<pyTitle>Alamat</pyTitle>`, blok
		// 533044..644780, satu-satunya blok ber-`pyHeaderType=COLLAPSIBLE`.
		//
		// Keenam isian di bawah adalah alamat pelapor. Hanya `.ASMAddressType` yang posisi
		// selnya terbukti berada DI DALAM blok ini; sisanya menyusul sesudahnya di deretan
		// sel alamat yang sama. Keenamnya terhalang, sehingga penempatan yang keliru di
		// antara kedua blok alamat TIDAK mengubah apa pun yang terlihat pengguna — tidak
		// satu pun dari keenamnya punya nilai untuk digambar.
		Title: "Alamat",
		Fields: []Field{
			blocked(DocFieldReporterCountry, "Negara", blockedUnexposed),
			blocked(DocFieldReporterProvince, "Provinsi", blockedUnexposed),
			blocked(DocFieldReporterCity, "Kota", blockedUnexposed),
			blocked(DocFieldReporterDistrict, "Kabupaten", blockedUnexposed),
			blocked(DocFieldReporterVillage, "Kelurahan", blockedUnexposed),
			blocked(DocFieldReporterPostcode, "Kodepos", blockedUnexposed),
		},
	},
}

// withVisibility memasang syarat tampil pada isian yang sudah terbentuk.
//
// Ia ada supaya isian terhalang tetap dibentuk `blocked` — dengan alasan dan pemilik yang
// sama seperti saudaranya — alih-alih ditulis ulang lengkap hanya untuk menambah satu isian.
func withVisibility(f Field, rule VisibilityRule) Field {
	f.VisibleWhen = rule
	return f
}

// DocumentFieldGroupsFor menyusun bentuk layar untuk SATU berkas, dengan isian yang syarat
// tampilnya tidak terpenuhi sudah dibuang.
//
// # Kenapa penyaringan terjadi di sini, bukan di layar
//
// Karena syaratnya dibaca dari section Pega, dan di sinilah buktinya tercatat. Mengirim
// seluruh isian beserta syaratnya ke layar berarti aturan yang sama hidup di dua tempat, dan
// yang di layar akan tertinggal pada hari syaratnya berubah.
//
// Kelompok yang seluruh isiannya tersaring habis ikut dibuang — panel berjudul tanpa satu
// pun isian lebih membingungkan daripada panel yang tidak ada.
func DocumentFieldGroupsFor(detail ReceiveDocument) []FieldGroup {
	result := make([]FieldGroup, 0, len(DocumentFieldGroups))

	for _, group := range DocumentFieldGroups {
		fields := make([]Field, 0, len(group.Fields))
		for _, field := range group.Fields {
			if field.visible(detail) {
				fields = append(fields, field)
			}
		}
		if len(fields) == 0 {
			continue
		}
		result = append(result, FieldGroup{Title: group.Title, Fields: fields})
	}

	return result
}

// DocumentFieldGroupList mengembalikan susunan isian dalam urutan tampilnya.
//
// Salinan bertingkat, bukan senarai aslinya: pemanggil tidak boleh dapat mengubah bentuk
// layar dengan menulisi hasilnya.
func DocumentFieldGroupList() []FieldGroup {
	result := make([]FieldGroup, 0, len(DocumentFieldGroups))
	for _, group := range DocumentFieldGroups {
		fields := make([]Field, len(group.Fields))
		copy(fields, group.Fields)
		result = append(result, FieldGroup{Title: group.Title, Fields: fields})
	}
	return result
}

// WriteAction adalah satu tombol yang di layar lama MENGUBAH data.
//
// # Kenapa tombolnya tetap digambar
//
// Karena ia pekerjaan NYATA yang dilakukan pengguna layar ini setiap hari. Layar yang
// kehilangan tombolnya tanpa penjelasan akan dilaporkan sebagai kerusakan, dan penggunanya
// tidak akan tahu ia masih harus mengerjakannya lewat Pega.
//
// # Kenapa belum satu pun dapat dihidupkan
//
// Bukan karena keputusan yang belum diambil, melainkan karena dua hal yang terbaca dari
// artefaknya sendiri:
//
//   - **Isian yang disunting layar ini tidak punya kolom basis data.** Satu-satunya blok yang
//     `pyReadOnly=false` di section adalah data pelapor (`.ReportHE.*`) dan alamatnya
//     (`tempTES.*`), dan tidak satu pun punya kolom — keduanya hidup di dalam blob objek
//     kerja Pega. Menyimpan perubahannya dengan SQL karena itu tidak mungkin, bukan sekadar
//     tidak diizinkan.
//   - **Tombolnya memanggil pekerjaan milik modul lain.** `CreateRegisterKlaimPNC`
//     MEMBUAT KLAIM — itu modul `B-2`, bukan layar ini. `SendAttachmentToPNC` adalah
//     integrasi keluar (`S-4`), `InputParamUpload_act` dan `SetCategoryAttachment` adalah
//     unggah dokumen (`S-1`), `PNCSendMessageKomunikasi` dan `PNCReplyMessage` adalah
//     notifikasi (`S-3`). Tidak satu pun sudah dibangun.
//
// Ditambah `P-1`: objek kerjanya masih dimiliki Pega selama kedua sistem berjalan
// berdampingan.
//
// Penekanannya karena itu menjawab ALASAN, bukan halaman kosong — lihat ErrWriteNotAvailable.
type WriteAction struct {
	// Code adalah kode tombol pada kontrak API.
	Code string

	// Label adalah teks tombol, mengikuti `pyLabel` di section apa adanya (`D-13`).
	Label string

	// Activity adalah activity Pega yang dijalankannya. Ia disebut supaya penelusuran balik
	// ke export tetap mungkin, dan supaya modul yang kelak mengambil alih pekerjaannya dapat
	// ditunjuk dengan nama.
	Activity string

	// Owner menyebut modul yang kelak memiliki pekerjaan ini.
	Owner string

	// AtTop menyatakan tombol ini digambar DI ATAS isian, bukan di bawahnya.
	//
	// # Kenapa posisinya data, bukan urusan layar
	//
	// Karena ia dibaca dari section, dan buktinya posisi sel di dalamnya. Dua tombol berada
	// di dalam blok utama dekat puncaknya; sisanya berada SETELAH blok itu tertutup di
	// 1564427, yaitu di tempat Pega menggambar tombol flow action:
	//
	//	tombol                        activity                     posisi sel   letak
	//	----------------------------- ---------------------------- ------------ ------
	//	Upload Form Klaim             SetTempCategoryAttachment         180799   atas
	//	View Form Klaim               GetUploadData                     200348   atas
	//	Simpan & Transfer data ke ASM SendAttachmentToPNC              1587331   bawah
	//	Register Klaim                CreateRegisterKlaimPNC           1618486   bawah
	//
	// Tangkapan layar Work Owner 2026-10-10 memperlihatkan hal yang sama: tombol unggah dan
	// lihat dokumen berada di puncak blok, sebelum isian pertama.
	AtTop bool
}

// DocumentWriteActions adalah kedelapan tombol layar kerja, berurutan seperti di section.
//
// Labelnya dibaca dari `<pyLabel>` pada `Section/InputReceiveDocument-Section.xml`; nama
// activity-nya dari `<pyActivity>` pada action set masing-masing.
var DocumentWriteActions = []WriteAction{
	{
		Code:     "simpan",
		Label:    "Simpan",
		Activity: "(flow action save)",
		Owner:    "modul ini, setelah kepemilikan objek kerja berpindah dari Pega (`P-1`)",
	},
	{
		Code:     "simpan-transfer-asm",
		Label:    "Simpan & Transfer data ke ASM",
		Activity: "SendAttachmentToPNC",
		Owner:    "`S-4` Integrasi Eksternal",
	},
	{
		Code:     "register-klaim",
		Label:    "Register Klaim",
		Activity: "CreateRegisterKlaimPNC",
		Owner:    "`B-2` Registrasi Klaim",
	},
	{
		Code:     "reject-pelaporan",
		Label:    "Reject Pelaporan",
		Activity: "(flow action save)",
		Owner:    "modul ini, setelah kepemilikan objek kerja berpindah dari Pega (`P-1`)",
	},
	{
		Code:  "unggah-form-klaim",
		Label: "Upload Form Klaim",

		// Dikoreksi 2026-10-10. Versi sebelumnya menyebut `InputParamUpload_act`; activity
		// itu berada di posisi sel 955323, jauh di dalam blok "Daftar Penerimaan Dokumen",
		// dan melayani tombol unggah pada grid dokumen — bukan tombol ini. Tombol
		// "Upload Form Klaim" di puncak layar (sel 179928) menjalankan dua activity
		// berurutan.
		//
		// Yang kedua inilah yang menyimpan: `InsertOCRData` membaca hasil OCR formulir klaim
		// lalu mengisi `EstimationList`. Rantainya SEDANG RUSAK di Pega hari ini — step 4-nya
		// memanggil `ReviewOCRData` yang sudah tidak ada (lihat catatan di `R-09`), sehingga
		// tombol ini gagal di sisi Pega pun.
		Activity: "SetTempCategoryAttachment lalu InsertOCRData",
		Owner:    "`S-1` Dokumen & Lampiran",
		AtTop:    true,
	},
	{
		Code:     "lihat-form-klaim",
		Label:    "View Form Klaim",
		Activity: "GetUploadData",
		Owner:    "`S-1` Dokumen & Lampiran",
		AtTop:    true,
	},
	{
		Code:     "kirim-pesan",
		Label:    "Kirim Pesan",
		Activity: "PNCSendMessageKomunikasi",
		Owner:    "`S-3` Notifikasi",
	},
	{
		Code:     "balas-pesan",
		Label:    "Balas Pesan",
		Activity: "PNCReplyMessage",
		Owner:    "`S-3` Notifikasi",
	},
}

// DocumentWriteActionList mengembalikan kedelapan tombol dalam urutan tampilnya.
func DocumentWriteActionList() []WriteAction {
	result := make([]WriteAction, len(DocumentWriteActions))
	copy(result, DocumentWriteActions)
	return result
}

// ReceiveDocument adalah isi layar kerja penerimaan dokumen untuk SATU berkas.
//
// Isian yang tidak punya sumber TIDAK ada di sini — ia bertanda Blocked di
// DocumentFieldGroups, dan tidak akan pernah terisi sampai penghalangnya hilang. Menyediakan
// tempatnya di sini akan membuat orang mengira ia tinggal diisi.
type ReceiveDocument struct {
	// Reference adalah kunci teknis Pega — `PZINSKEY`. Ia yang dipakai membuka layar ini,
	// dan yang ditampilkan supaya petugas dapat menyebutkannya saat mengerjakan berkas ini
	// di Pega.
	Reference string

	// CaseID adalah nomor berkas yang dibaca pengguna — `PYID`.
	CaseID string

	// ClaimNumber adalah nomor klaim PNC yang terbit dari berkas ini — `PNCCASEID`.
	//
	// Kosong selama berkasnya belum diregistrasi menjadi klaim, dan itu keadaan yang sah —
	// bukan data hilang. Justru itulah yang dikerjakan tombol "Register Klaim".
	ClaimNumber string

	// ClaimType adalah Jenis Klaim, diturunkan dari Group Panel — lihat ClaimTypeOf.
	ClaimType string

	// GroupPanel adalah Group Panel mentah — `GROUPPANEL_1`.
	//
	// Ia dibawa BERDAMPINGAN dengan ClaimType, bukan menggantikannya, karena keduanya
	// menjawab pertanyaan yang berbeda. ClaimType hanya membedakan PA dari selainnya;
	// syarat tampil isian membutuhkan kodenya sendiri, dan salah satu syarat mengecualikan
	// `005` Travel yang pada ClaimType tidak terbedakan dari lini lain.
	//
	// Ia TIDAK digambar sebagai isian — lihat Field.visible.
	GroupPanel string

	// WorkStatus adalah status kerja berkas ini — `PYSTATUSWORK`.
	//
	// Ia TIDAK digambar sebagai isian di layar lama. Ia dibawa karena layar ini dibuka lewat
	// Open Assignment: petugas perlu tahu berkas ini masih berjalan atau sudah selesai
	// sebelum mengerjakannya di Pega.
	WorkStatus string

	// Isian yang punya sumber, berpasangan satu-untuk-satu dengan kunci di atas.
	CreatedAt        string
	ReceivedAt       string
	SenderName       string
	SenderEmail      string
	SenderPhone      string
	CourierName      string
	InsuredName      string
	PolicyNumber     string
	LossDate         string
	ReferenceNumber  string
	InsuredEmail     string
	LossLocation     string
	DriverLicence    string
	Chronology       string
	DamageDetail     string
	TransferReason   string
	EmailSubject     string
	NotRegisteredNot string
}

// Values menyusun isian layar sebagai peta berkunci Field.Key.
//
// # Kenapa peta, bukan struct yang dikirim apa adanya
//
// Karena yang menentukan isian mana yang digambar adalah DocumentFieldGroups, bukan bentuk
// baris ini. Peta membuat keduanya tidak dapat berselisih: isian yang tidak disebut bentuk
// layar tidak akan tergambar, dan isian yang disebut tetapi belum punya sumber tergambar
// kosong dengan alasannya.
//
// Isian bertanda Blocked sengaja TIDAK muncul di peta ini. Mengisinya dengan teks kosong akan
// membuat layar tidak dapat membedakan "belum ada sumbernya" dari "sumbernya ada tetapi
// kosong" — dua hal yang tindak lanjutnya berbeda.
func (d ReceiveDocument) Values() map[string]string {
	return map[string]string{
		DocFieldCreatedAt:       d.CreatedAt,
		DocFieldReceivedAt:      d.ReceivedAt,
		DocFieldSenderName:      d.SenderName,
		DocFieldSenderEmail:     d.SenderEmail,
		DocFieldSenderPhone:     d.SenderPhone,
		DocFieldCourierName:     d.CourierName,
		DocFieldInsuredName:     d.InsuredName,
		DocFieldPolicyNumber:    d.PolicyNumber,
		DocFieldLossDate:        d.LossDate,
		DocFieldReferenceNumber: d.ReferenceNumber,
		DocFieldInsuredEmail:    d.InsuredEmail,
		DocFieldLossLocation:    d.LossLocation,
		DocFieldDriverLicence:   d.DriverLicence,
		DocFieldChronology:      d.Chronology,
		DocFieldDamageDetail:    d.DamageDetail,
		DocFieldTransferReason:  d.TransferReason,

		// Kedua judul ini menunjuk properti yang SAMA di Pega, sehingga keduanya menampilkan
		// isi yang sama persis. Dibawa apa adanya (`D-13`) dan dinyatakan ke pengguna lewat
		// PlannedDifferences — bukan diperbaiki diam-diam dengan menebak kolom kedua.
		DocFieldTransferNote: d.TransferReason,

		DocFieldEmailSubject:     d.EmailSubject,
		DocFieldNotRegisteredNot: d.NotRegisteredNot,
	}
}
