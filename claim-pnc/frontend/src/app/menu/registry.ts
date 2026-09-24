/**
 * Peta dari MENU_PROGRAM ke rute layar yang sudah dibangun.
 *
 * # Kenapa petanya di frontend, bukan di backend
 *
 * Yang dipetakan adalah RUTE ANTARMUKA, dan backend tidak menyimpannya. Menaruh peta
 * ini di sana berarti server harus tahu bentuk URL React — dan setiap perubahan rute
 * menjadi perubahan di dua tempat.
 *
 * Pembagian tugasnya karena itu tegas:
 *
 *	backend   butir menu mana yang boleh DILIHAT pemanggil (M_OTORISASI_PNC)
 *	frontend  butir menu mana yang sudah punya LAYAR (berkas ini)
 *
 * # Menambah modul = menambah satu baris
 *
 * Itulah satu-satunya yang perlu dikerjakan agar butir menunya hidup. Nama kuncinya
 * harus sama persis dengan MENU_PROGRAM di POOLDATA.M_MENU_APLIKASI_PNC — termasuk
 * huruf besar-kecilnya, karena itulah yang dikirim server apa adanya.
 *
 * # Yang TIDAK ada di sini, dan itu bukan kelalaian
 *
 * 25 dari 75 butir menu belum punya layar. Butirnya tetap tampil di menu, tidak dapat
 * diklik, dan bertanda "belum tersedia" — keputusan Work Owner 2026-09-18. Dengan
 * begitu kemajuan migrasi terbaca langsung dari layar, dan pengguna tidak melaporkan
 * menu yang "hilang".
 *
 * Angka 75 adalah MENU_PROGRAM tidak kosong yang UNIK pada
 * `Database/m_menu_aplikasi_pnc.csv` — dari 80 barisnya, empat adalah judul kelompok
 * (MASTER, INBOX, VIEW, REPORT) dan satu adalah MENU_ID 83 "Report Adjuster" yang
 * MENU_PROGRAM-nya memang kosong.
 *
 * TUJUH di antaranya bahkan menunjuk harness yang TIDAK ADA di export Pega
 * (`InboxCloseClaim_Harness`, `InboxOutstanding_Harness`, `InboxRequestSalvage`,
 * `InboxServiceCenter`, `LostAdjuster_harness`, `PNCViewClaim`,
 * `ReportProduksiPA_harnes`) — memperjelas `K-33`.
 *
 * Dua yang dulu ada di daftar itu SUDAH DITERIMA pada 2026-09-22 dan karena itu
 * dikeluarkan: `DataMemberReas` dan `DetailMasterPasalAI` — keduanya kini punya layar.
 *
 * SATU dari ketujuh yang tersisa KINI SUDAH PUNYA LAYAR, dibangun dari kueri,
 * activity, dan section yang memang ada — bukan dari harness-nya:
 *
 *   `InboxCloseClaim_Harness`    rujukan bentuknya `InboxManagerReopen1_Sec`, section
 *                                yang di dalamnya sendiri berjudul "Inbox Close Claim"
 *
 * `InboxOutstanding_Harness` (MENU_ID 79) TETAP belum punya layar. Yang kini punya
 * layar adalah `InboxRegister_Harness` (MENU_ID 51 "My Inbox") — butir menu yang
 * BERBEDA, dan keduanya sempat tertukar. Lihat catatan pada barisnya di bawah.
 */
export const MENU_ROUTES: Record<string, string> = {
  StatusClaimInbox: '/master/status-klaim',
  MasterRekening: '/master/rekening',
  StatusProgress: '/master/status-progres-1',
  StatusProgress2: '/master/status-progres-2',
  // MENU_ID 22 "Master Dokumen Travel", di bawah kelompok MASTER.
  //
  // JANGAN tertukar dengan MENU_ID 39 di bawahnya: yang ini master INDUK — hanya DOCID
  // dan judul dokumen.
  BrowseMasterDocumentTravel_Harness: '/master/dokumen-travel',
  // MENU_ID 39 "Daftar Detail Dokumen Travel", di bawah kelompok MASTER.
  //
  // Master TURUNAN atas V_LST_DOC_TRAVEL: ia merujuk DOCID milik MENU_ID 22 di atas dan
  // menambahkan aturannya — wajib atau tidak, jumlah unggahan minimum, dan pembatasan per
  // plan serta jaminan pada V_LST_DOC_TRAVEL_COVERAGE.
  ListDocumentTravel: '/master/daftar-detail-dokumen-travel',
  // MENU_ID 21 "Master COL SIMAS ONLNE", di bawah kelompok MASTER.
  //
  // Salah ketik "ONLNE" pada MENU_DESC ada di basis datanya, bukan di sini. Ia TIDAK
  // diperbaiki dari kode: butir menu dibaca apa adanya dari
  // POOLDATA.M_MENU_APLIKASI_PNC, dan memperbaikinya di sini akan membuat layar
  // menampilkan teks yang berbeda dari isi tabel — perbaikannya menempuh `D-63`.
  //
  // JANGAN tertukar dengan MENU_ID 20 "Master Penyebab Kerugian" (`CauseOfLossInbox`)
  // di bawah, layar Master COL biasa atas tabel yang SAMA tetapi tanpa isian ID Master
  // Kerugian dan tanpa pemetaan bisnis. Keduanya sudah dibangun dan menulis tabel yang
  // sama — perbedaannya hanya pada isian yang ditampilkan.
  CauseOfLossInboxSimasOnline: '/master/col-simas-online',
  // MENU_ID 40 "Daftar Tipe Dokumen", di bawah kelompok MASTER.
  //
  // JANGAN tertukar dengan kedua master TURUNANNYA, yang merujuk ID milik layar ini:
  //
  //   `ListDetTypeDocument`   "Daftar Detail Tipe Dokumen"   (V_LST_DET_TYPE_DOC)  ada di bawah
  //   `DetTypeDocumenBisnis`  "Daftar Tipe Dokumen Bisnis"   (LST_TYPE_DOC_BUSINESS) ada di bawah
  ListDocumentTypeInbox: '/master/tipe-dokumen',
  // MENU_ID 41 "Daftar Detail Tipe Dokumen", di bawah kelompok MASTER.
  //
  // Master TURUNAN atas V_LST_DET_TYPE_DOC: ia merujuk ID milik MENU_ID 40 di atas dan
  // menambahkan rinciannya — dokumen apa persisnya yang diminta, melekat pada objek apa,
  // dipicu penyebab kerugian mana, dan wajib pada lini bisnis mana.
  //
  // JANGAN tertukar dengan MENU_ID 42 tepat di bawahnya. Namanya mirip, tetapi tabelnya
  // sama sekali berbeda: yang ini V_LST_DET_TYPE_DOC, yang itu LST_TYPE_DOC_BUSINESS —
  // dan yang itu justru MEMBACA ID baris layar ini lewat DOC_TYPE_DT_ID.
  //
  // Rutenya `detail-tipe-dokumen`, bukan nama modulnya utuh, supaya hubungan ketiganya
  // terbaca dari URL: /master/tipe-dokumen (induk), /master/detail-tipe-dokumen (ini),
  // /master/objek-dokumen (master yang dirujuknya).
  ListDetTypeDocument: '/master/detail-tipe-dokumen',
  // MENU_ID 42 "Daftar Tipe Dokumen Bisnis", di bawah kelompok MASTER.
  //
  // Master TURUNAN atas V_LST_DOC_TYPE: ia merujuk ID milik MENU_ID 40 di atas dan
  // menjawab pertanyaan yang lebih sempit — dokumen apa yang harus diunggah, untuk lini
  // bisnis mana, pada tahap klaim mana.
  //
  // Judul layarnya berbunyi "Detail Tipe Dokumen Bisnis", berbeda dari MENU_DESC di
  // tabel menu yang berbunyi "Daftar Tipe Dokumen Bisnis". Keduanya ditiru apa adanya:
  // yang di menu dibaca dari POOLDATA.M_MENU_APLIKASI_PNC, yang di layar dari
  // `Section/DetTypeDocumenBisnis_Portal-Section.xml` (`D-13`).
  //
  // Rutenya mengikuti nama BUTIR MENU, bukan nama harness (`D-81`), dan jalur itu sudah
  // dicadangkan sejak modul MENU_ID 40 dibangun.
  DetTypeDocumenBisnis: '/master/tipe-dokumen-bisnis',
  // MENU_ID 43 "Daftar Objek Dokumen", di bawah kelompok MASTER.
  //
  // JANGAN tertukar dengan `ListDocumentTypeInbox` tepat di atasnya. Keduanya master acuan
  // yang dirujuk BERSAMAAN oleh satu tabel yang sama, dan bedanya ada di pertanyaan yang
  // dijawabnya:
  //
  //   Daftar Tipe Dokumen    dokumennya JENISNYA apa   LST_TYPE_DOC_BUSINESS.DOCUMENT_TYPE_ID
  //   Daftar Objek Dokumen   dokumennya MELEKAT PADA APA  LST_TYPE_DOC_BUSINESS.OBJECT_DOC_ID
  //
  // Keduanya terbaca berdampingan di `Database/PEGA_LST_DET_TYPE_DOC_BUSINESS.prc:26`.
  ListDocumentObject: '/master/objek-dokumen',
  // MENU_ID 51 **"My Inbox"** — layar daftar klaim yang masih berjalan.
  //
  // # Kuncinya sempat salah, dan menunya karena itu tidak pernah muncul
  //
  // Semula didaftarkan sebagai `InboxOutstanding_Harness`, karena judul DI DALAM
  // `Section/InboxRegister_Section-Section.xml:2150` berbunyi "Inbox Outstanding".
  // Judul itu memang ada, tetapi ia judul section — bukan nama butir menu.
  //
  // `POOLDATA.M_MENU_APLIKASI_PNC` memuat KEDUANYA sebagai butir yang BERBEDA:
  //
  //	MENU_ID 51 · "My Inbox"          · InboxRegister_Harness      <- layar ini
  //	MENU_ID 79 · "Inbox Outstanding" · InboxOutstanding_Harness   <- layar LAIN
  //
  // Jadi kunci lama bukan sekadar salah nama: ia menempelkan layar ini pada butir
  // menu MILIK LAYAR LAIN. Pengguna yang menekan "Inbox Outstanding" akan mendapat
  // layar My Inbox, sedangkan "My Inbox" sendiri tidak mengarah ke mana-mana.
  //
  // MENU_ID 79 tetap belum punya layar — harness-nya tidak ada di export (`K-33`),
  // dan isinya belum pernah diketahui.
  //
  // Nama modul dan alamat rutenya sengaja DIBIARKAN `inbox-outstanding` sampai Work
  // Owner memutuskan — mengganti nama modul menyentuh backend, frontend, dan tiket
  // sekaligus (`D-81`), sedangkan memperbaiki kunci ini memulihkan menunya sekarang.
  InboxRegister_Harness: '/inbox-outstanding',
  InboxAutoClaim: '/inbox-auto-claim',
  // MENU_ID 14 "Master Tipe Surveyors" — GOLONGAN petugas survei.
  SurveyorsInbox: '/master/tipe-surveyor',
  // MENU_ID 15 "Master Surveyors" — daftar ORANGNYA, anak dari butir di atas. Keduanya
  // bernama mirip dan mudah tertukar; yang membedakan adalah `Detail` di awal nama
  // programnya.
  DetailSurveyorsInbox: '/master/surveyor',
  // MENU_ID 13 "Master PIC Teknik". Namanya mengandung "Inbox" tetapi ia layar MASTER,
  // bukan daftar pekerjaan (`D-79`) — barisnya data acuan, tidak hilang setelah
  // ditindaklanjuti, dan tidak punya tenggat.
  UserTeknisInbox: '/master/pic-teknik',
  // MENU_ID 16 "Master Recovery". Berbeda dari butir master lain di daftar ini: layarnya
  // FORM ENTRI, bukan pengelola data acuan — tidak ada satu pun kueri di export yang
  // membaca kembali tabelnya. Letaknya tetap di bawah kelompok MASTER karena di situlah
  // butir menunya berada (`MENU_ID_LEADER 1`).
  MasterRecovery: '/master/recovery',
  // MENU_ID 18 "Master Dominan Factor". Namanya diawali `Detail` seperti
  // `DetailSurveyorsInbox`, tetapi di sini awalan itu TIDAK menandakan tingkat kedua —
  // tidak ada master "Dominan Factor" di atasnya. Ia layar master yang berdiri sendiri.
  DetailDominanFactor: '/master/dominan-factor',
  // MENU_ID 20 "Master Penyebab Kerugian" — tingkat GOLONGAN (`M_CAUSE_OF_LOSS`).
  //
  // Satu butir menu bersaudara sengaja TIDAK dipetakan di sini, dan ia mudah tertukar
  // dengannya:
  //
  //   MENU_ID 38  DetailCauseOfLoss  rinciannya (`D_CAUSE_OF_LOSS`), belum ada
  //
  // MENU_ID 21 `CauseOfLossInboxSimasOnline` — varian Simas Online — sudah dipetakan di
  // atas. Ia menulis TABEL YANG SAMA dengan layar ini, sehingga keduanya harus dijaga
  // tetap sepakat soal aturan isiannya. Lihat
  // backend/migrations/0005_master_penyebab_kerugian.up.sql.
  CauseOfLossInbox: '/master/penyebab-kerugian',
  // MENU_ID 17 "Master Masking". Nama programnya panjang dan tidak menyebut "masking"
  // sama sekali — rutenya mengikuti nama BUTIR MENU, bukan nama harness (`D-81`), karena
  // itulah nama yang dipakai Work Owner dan yang tertulis di menu.
  //
  // Isinya kewenangan melihat data pribadi nasabah, sehingga layar ini yang paling berat
  // akibatnya bila terbuka oleh peran yang tidak berhak. Penegakan izin per menu masih
  // TKT-F3-005 dan belum ada.
  MasterProteksiVisibilityData: '/master/masking',
  // MENU_ID 19 "Master XOL". Satu-satunya butir master yang layarnya BERTINGKAT EMPAT —
  // induk, grup bisnis, layer, dan reas tiap layer — dan satu-satunya yang menyimpannya
  // sekaligus mengajukan ke komite.
  DetailMasterXOL: '/master/xol',

  // ── Butir yang layar DAN backend-nya baru tersambung ────────────────────────────
  //
  // Kesepuluh butir di bawah sudah punya layar lengkap beserta ujinya sejak lama, tetapi
  // perakitan backend-nya hilang pada penggabungan cabang — rutenya tidak pernah
  // terdaftar, sehingga setiap layarnya menjawab 404. Perakitannya dipulihkan di
  // `backend/cmd/claimpnc/modules.go`, dan `TestExtraModulesMounted` menjaganya tidak
  // lepas lagi.
  //
  // Setiap nama kunci di bawah DICOCOKKAN ke `Database/m_menu_aplikasi_pnc.csv`, bukan
  // ditebak dari nama modulnya: kunci yang meleset tidak menimbulkan galat apa pun —
  // butirnya sekadar tetap "belum tersedia" — sehingga mencocokkannya ke tabel adalah
  // satu-satunya cara memastikannya benar.
  //
  // `StatusProgress2` (MENU_ID 24) dulu sengaja TIDAK ada di sini karena backend-nya belum
  // pernah ditulis. Itu berubah pada penggabungan 2026-09-24: `/api/master/status-progres-2`
  // kini ada di `internal/masterstatusprogres/http/routes2.go`, sehingga butirnya dipetakan
  // di atas bersama `StatusProgress`.

  // MENU_ID 25. SATU layar untuk DUA master — Penolakan Klaim dan Penolakan Komite —
  // karena di Pega pun keduanya satu butir menu. Pemilihannya tab di dalam layar.
  PNC_MasterTolakKlaim: '/master/penolakan-klaim',
  // MENU_ID 26. Empat tab atas tabel yang sama, hanya berbeda saringan.
  AutoKlaim: '/master/auto-claim',
  // MENU_ID 27 "Master Pasal Kerugian". Nama programnya menyebut "Rejected" tetapi
  // layarnya master pasal, bukan daftar penolakan — nama itu dibaca apa adanya dari tabel
  // menu, dan memperbaikinya menempuh `D-63`.
  DetailMasterPasalRejected: '/master/pasal-kerugian',
  // MENU_ID 36 "Master Pasal AI". Kembaran Master Pasal Kerugian di atas — section-nya
  // memang Save-As darinya — tetapi sudah dipangkas menjadi layar PENCARIAN BACA-SAJA:
  // tanpa tab, tanpa Tambah/Simpan/Hapus, hanya Cari dan Refresh.
  //
  // Ia satu-satunya layar master yang paginasinya dikerjakan SERVER, dan itu bukan pilihan
  // kami: grid Pega-nya ber-`pyPageMode = None` dengan jendela dihitung activity.
  //
  // Tabelnya `POOLDATA.MST_PASAL_AI`; kolomnya `WP_PASAL`, `WP_AYAT`, dan `WP_KEJADIAN` —
  // nama yang baru terbaca setelah activity dan kedua Connect-SQL-nya diterima, karena
  // propertinya di layar Pega bernama warisan (`.City`, `.CityID`, `.District`).
  DetailMasterPasalAI: '/master/pasal-ai',

  // MENU_ID 28, 30, 31 — keluarga alat berat, berbagi satu activity persetujuan yang sama
  // di Pega (`Activity/SetApprovalAllMaster`). Ketiganya beserta Master Supplier di bawah
  // membaca tabel yang `D-34` keluarkan dari lingkup migrasi; Work Owner memutuskan pada
  // 2026-09-22 bahwa seluruh modul dari cabang `fran-masuk-master` harus ada. Lihat
  // catatan di kepala `backend/cmd/claimpnc/modules.go`.
  //
  // MENU_ID 28 "Master Bengkel". Satu butir menu, satu layar, TIGA tab — Approve,
  // Waiting Approval, dan Reject — persis seperti ketiga tab pada
  // `Section/BrowseMasterHE-Section.xml`.
  BengkelHE: '/master/bengkel',
  MasterPanel_HE: '/master/panel',
  SparePart_HE: '/master/sparepart',
  // MENU_ID 32 "Master Grouping Sparepart". Satu butir menu, satu layar, TIGA tab —
  // Approve, Reject, dan Waiting Approval — persis seperti ketiga section pada
  // `Section/PNCMasterGroupingSparepartHE-Section.xml`.
  //
  // Yang dikelolanya BUKAN penggolongan suku cadang melainkan penautan suku cadang ke
  // panel bodi pada sebuah kendaraan, dikelompokkan menurut nomor rangka.
  //
  // Nama kuncinya `GroupingSparePart_HE` dengan P besar di tengah, persis seperti yang
  // tertulis di POOLDATA.M_MENU_APLIKASI_PNC. Huruf besar-kecilnya dikirim server apa
  // adanya.
  GroupingSparePart_HE: '/master/grouping-sparepart',
  // MENU_ID 33 "Master Kategori Sparepart". Satu butir menu, satu layar, TIGA tab —
  // Approve, Reject, dan Waiting Approval — persis seperti ketiga section pada
  // `Section/MasterKategoriSparepartHE-Section.xml`.
  //
  // Nama kuncinya `GCNMCatSparepart`, persis seperti yang tertulis di
  // POOLDATA.M_MENU_APLIKASI_PNC — termasuk "Cat" yang merupakan singkatan dari Category,
  // bukan salah ketik. Huruf besar-kecilnya dikirim server apa adanya.
  GCNMCatSparepart: '/master/kategori-sparepart',
  // MENU_ID 34 "Master Tipe Sparepart". Satu butir menu, satu layar, TIGA tab — Approve,
  // Reject, dan Waiting Approval — persis seperti ketiga section pada
  // `Section/MasterTipeSparepartHE-Section.xml`.
  //
  // Nama kuncinya `GCNMMasterSparepartType`, persis seperti yang tertulis di
  // POOLDATA.M_MENU_APLIKASI_PNC. Perhatikan bahwa nama programnya memakai "SparepartType"
  // sementara seluruh caption layarnya menyebut "Tipe Sparepart"; rutenya mengikuti nama
  // bisnis, kunci petanya mengikuti basis data.
  GCNMMasterSparepartType: '/master/tipe-sparepart',
  // MENU_ID 29 "Master Supplier". Satu butir menu, satu layar, TANPA tab — layar lamanya
  // memang satu grid dengan tiga tombol (New Supplier, Edit, Refresh) dan tidak punya
  // penyaring status apa pun (`Section/InboxMasterSupplier-Section.xml`).
  //
  // Satu-satunya master yang seluruh isinya tinggal di SATU kolom JSONDATA: `M_SUPPLIER`
  // hanya punya ID, OLDID, dan JSONDATA.
  MasterSupplier: '/master/supplier',
  // MENU_ID 37 "Master Login". Satu butir menu, satu layar, TANPA tab — layar lamanya
  // memang satu grid dengan dua tombol (Tambah, Refresh) dan tidak punya penyaring status
  // apa pun, karena POOLDATA.MST_LOGIN_SURVEYOR tidak punya kolom APPROVAL.
  //
  // Nama kuncinya `MasterLoginSurvey` — menyebut "Survey", sementara MENU_DESC-nya hanya
  // "Master Login". Rutenya mengikuti nama menu, kunci petanya mengikuti basis data.
  MasterLoginSurvey: '/master/login',
  // MENU_ID 35 "Master Reas". Satu butir menu, satu layar, TANPA tab dan TANPA tombol
  // simpan — harness lamanya memang satu grid dengan satu tombol Refresh, dan
  // POOLDATA.T_REINSURER tidak punya kolom persetujuan.
  //
  // Satu-satunya layar master yang BACA-SAJA. Tabelnya ditulis alur PLA/DLA lewat
  // `Database/UPDATEREAS.prc` — dipanggil `UpdateDetailPLA2` dan `UpdateDetailDLA2` —
  // bukan oleh layar ini.
  //
  // Harness-nya dulu termasuk yang dicatat di atas sebagai TIDAK ADA di export; ia
  // diterima pada 2026-09-22. Yang MASIH hilang adalah section gridnya,
  // `BrowseListMemberReas` — sehingga daftar kolom layarnya tetap rekonstruksi (`R-16`).
  DataMemberReas: '/master/reas',
  // MENU_ID 38 "Detail Penyebab Kerugian". Satu butir menu, satu layar, TANPA tab —
  // harness lamanya memang satu grid dengan form penyuntingan di bawahnya, dan
  // POOLDATA.D_CAUSE_OF_LOSS tidak punya kolom persetujuan.
  //
  // Ia ANAK dari "Master Penyebab Kerugian" (MENU_ID 20, `CauseOfLossInbox`) yang belum
  // punya layar. Layar ini hanya MEMBACA master itu sebagai daftar pilihan; induk baru
  // belum dapat dibuat dari sini.
  //
  // Report Definition pengisi gridnya, `BrowseVDCauseOfLoss_RD`, HILANG dari export
  // (`R-16`) — sehingga cakupan daftarnya rekonstruksi dari dua rule lain atas view yang
  // sama. Lihat banner paket `detailpenyebab`.
  DetailCauseOfLoss: '/master/detail-penyebab-kerugian',

  // MENU_ID 48 "Inbox Investigator". Layar INBOX pertama yang dibangun, dan yang pertama
  // berada di bawah awalan `/inbox/...` — INBOX adalah kelompok menu tersendiri di sistem
  // lama (`MENU_ID 2`, induk dari 30 butir).
  //
  // Isinya antrean bersama workbasket `InvestigatorPNC`: barisnya PEKERJAAN, hilang setelah
  // selesai dikerjakan, dan punya tenggat — keempat ciri Inbox pada `D-79`. Itu yang
  // membedakannya dari layar master dan dari View History Claim.
  //
  // Baca-saja. Mengambil pekerjaan dari antrean dan mencatat hasil investigasi ada di layar
  // kerja yang tidak digambar harness ini dan belum dibangun.
  InboxInvestigator_Harness: '/inbox/investigator',

  // MENU_ID 49 "Inbox Receive TKA". Layar INBOX kedua, dan yang PERTAMA yang menulis:
  // pengguna mengisi Tanggal Dokumen Lengkap langsung di dalam tabel lalu menekan Submit,
  // dan barisnya hilang dari daftar.
  //
  // Sumbernya BUKAN antrean penugasan Pega melainkan POOLDATA.T_CLAIM_TKA_H — tabel yang
  // ketujuh kolomnya sama persis dengan ketujuh kolom grid layar lama. Report Definition
  // lamanya mendeklarasikan halaman workbasket tetapi tidak pernah merujuknya; itu sisa
  // Save-As, dan penanda TKA-lah yang menentukan keanggotaan daftar.
  //
  // Submit menulis DUA tabel dalam satu transaksi: T_CLAIM_PNC.TGLDOKLENGKAP agar
  // tanggalnya sampai ke klaim, dan T_CLAIM_TKA_H.TGL_DOC_LENGKAP agar barisnya hilang.
  InboxTKA_Harness: '/inbox/receive-tka',

  // MENU_ID 47 "Inbox Compliance", di bawah kelompok INBOX, urutan 1137 — tepat sebelum
  // Inbox Investigator (1138). Ia Inbox sungguhan menurut `D-79`: barisnya pekerjaan yang
  // menunggu di workbasket `CompliancePNC`, hilang setelah klaimnya selesai, dan punya
  // tenggat berupa kolom Aging.
  //
  // Ini mengoreksi `22-INVENTARIS-HARNESS.md`, yang menandainya JANGGAL dengan alasan
  // `RD 0`. Report Definition-nya ADA dan dua buah — keduanya tinggal di dalam section,
  // bukan di harness, sehingga tidak terhitung pada tingkat harness.
  inboxCompliance_Harness: '/inbox-compliance',
  // MENU_ID 63, di bawah kelompok INBOX — bukan MASTER. Ia Inbox sungguhan menurut `D-79`:
  // barisnya pekerjaan, hilang setelah ditindaklanjuti, dan punya tenggat.
  PNCInboxAdmin: '/inbox-admin',
  // MENU_ID 64 "Inbox Laporan Klaim" TIDAK dipetakan di sini — lihat entri
  // `InboxRCVApp_Harness` di bawah, yang menunjuk `/inbox/laporan-klaim`.
  // MENU_ID 76 "View History Claim", di bawah kelompok VIEW.
  PNCSearchKlaim: '/riwayat-klaim',
  // MENU_ID 53 "Inbox XOL". Akumulasi klaim per perjanjian Excess of Loss beserta
  // pemberitahuan PLA/DLA kepada reasuradur. MEMBACA SAJA untuk sekarang — keempat
  // tabel yang ditulis sistem lama masih dimiliki Pega selama masa paralel (`P-1`).
  //
  // Bedakan dari `DetailMasterXOL` di atas: itu Master XOL (`MENU_ID 19`) yang MENULIS
  // struktur treaty-nya. Keduanya menyentuh MST_XOL_PNC dan kerabatnya, dan hanya satu
  // di antaranya yang boleh menulis.
  Inbox_XOL_Harness: '/inbox-xol',
  // MENU_ID 54 "Inbox Claim Treaty Prop". Antrean klaim treaty PROPORSIONAL — klaim yang
  // dialihkan perusahaan asuransi lain kepada ASM sebagai penanggung ulang.
  //
  InboxClaimTreaty_Harness: '/inbox-claim-treaty-prop',
  // MENU_ID 55 "Inbox Claim Treaty Non Prop". Layar SAUDARA dari yang di atas, dan
  // rutenya sengaja BERBEDA — bukan sekadar karena namanya berbeda.
  //
  // Keduanya membaca tabel, kolom, dan penanda objek kerja yang berbeda: yang ini
  // menyaring `PXREFOBJECTINSNAME LIKE 'CLMNP-%'` atas gabungan TIGA tabel, sedangkan
  // yang di atas menyaring kunci objek kerjanya atas gabungan dua tabel. Menunjuk
  // keduanya ke satu rute akan menampilkan antrean lini bisnis yang salah, dan tidak ada
  // apa pun di layar yang menandakannya.
  InboxClaimNonProp_Harness: '/inbox-claim-treaty-non-prop',

  // MENU_ID 65 "Inbox Progress Claim". Inbox sungguhan menurut `D-79`: barisnya klaim
  // yang menunggu ditindaklanjuti, hilang begitu klaimnya tutup, dan punya tenggat
  // berupa Next Follow Up.
  //
  // Dua butir lain — `PNCInboxAdmin` (63) dan `PNCSearchKlaim` (76) — SENGAJA tidak
  // dipetakan di sini. Kedua modulnya ada di repo, tetapi tidak satu pun rute API-nya
  // terpasang di `cmd/claimpnc`, sehingga memetakannya berarti menghidupkan butir menu
  // yang mengantar pengguna ke layar tanpa backend. Menyalakannya menuntut perakitan
  // kedua modul itu lebih dulu — keputusan tersendiri, bukan bagian dari merge ini.
  //
  // `InboxRCVApp_Harness` (64) dulu ikut ditahan karena alasan yang sama; ia kini SUDAH
  // dipetakan di bawah, karena rute API-nya sudah terpasang.
  ProgressClaim_Harness: '/inbox-progress-claim',

  // MENU_ID 64 "Inbox Laporan Klaim", kelompok INBOX.
  //
  // Ia SUDAH dipetakan — berbeda dari kedua butir di atas — karena merge ini memasang
  // rute API-nya di `cmd/claimpnc`. Alasan menahannya pada merge sebelumnya karena itu
  // sudah tidak berlaku untuk butir ini.
  //
  // Butir ini sempat TERDAFTAR DUA KALI. Cabang tujuan memetakannya pula ke
  // `/pelaporan-klaim` di dekat kepala daftar, sementara baris ini memetakannya ke
  // `/inbox/laporan-klaim`. Yang kedua itu yang benar — `/pelaporan-klaim` tidak ada di
  // `App.tsx` maupun di berkas mana pun — dan yang pertama dibuang saat merge Inbox
  // Manager Receive / PUCL (2026-09-23).
  //
  // Kunci ganda pada object literal TIDAK menghasilkan galat saat dijalankan: yang
  // terakhir menang, sehingga perilakunya kebetulan benar. Yang menangkapnya adalah
  // `tsc` (TS1117), bukan uji mana pun.
  InboxRCVApp_Harness: '/inbox/laporan-klaim',

  // MENU_ID 56 "Inbox Manager Receive / PUCL", kelompok INBOX.
  //
  // Nama programnya menyimpan salah ketik yang dipertahankan — `ReceiveDoucument`, bukan
  // `ReceiveDocument`. Ia disalin apa adanya dari POOLDATA.M_MENU_APLIKASI_PNC: kuncinya
  // harus sama persis dengan yang dikirim server, dan membetulkannya di sini akan membuat
  // butir menunya tampak belum tersedia selamanya.
  //
  // Bedakan dari `InboxRCVApp_Harness` tepat di atasnya. Keduanya menyentuh berkas
  // penerimaan dokumen, dan hanya itu kesamaannya:
  //
  //   Inbox Laporan Klaim (64)   berkas MILIK petugas, lengkap dengan komunikasi cabang
  //   layar ini (56)             pandangan PENYELIA atas berkas SELURUH petugas,
  //                              ditambah antrean klaim RCL/PUCL yang tidak ada di sana
  //
  // Rutenya karena itu terpisah, dan tidak boleh disatukan: yang satu menyaring menurut
  // pembuat berkas, yang lain tidak menyaring menurut pemanggil sama sekali.
  ReceiveDoucument_Harness: '/inbox-manager-receive-pucl',

  // MENU_ID 52 "Inbox Komite" — case ASM-FW-GCNMFW-Work-Komite.
  //
  // Di data contoh `m_otorisasi_pnc.csv`, butir ini hanya diberikan kepada grup `IT`.
  // Anggota komite yang sesungguhnya — peran PNCKomite dan PNCKomiteTeknik — belum ada
  // barisnya, sehingga mereka tidak akan melihat butirnya sampai otorisasinya diisi.
  // Itu keadaan DATA, bukan cacat kode.
  InboxKomite_Harness: '/komite/inbox',

  // MENU_ID 59 "Inbox Close Claim", kelompok INBOX. Harness-nya TIDAK ADA di export
  // (`K-33`); layarnya dibangun dari `GcnmBrowseReopenCase_SQL`, `GCNMCountCloseClaim`,
  // `GCNMGetManagerReopenCase_Act`, dan `InboxManagerReopen1_Sec` yang memang ada.
  //
  // Bedakan dari `InboxOutstanding_Harness` di dekat kepala daftar. Keduanya menyaring DUA
  // NILAI PYSTATUSWORK YANG SAMA dengan arah yang BERLAWANAN — yang satu klaim berjalan,
  // yang lain klaim tutup — sehingga menunjuk keduanya ke satu rute akan menampilkan
  // kebalikan dari yang diminta pengguna, dan tidak ada apa pun di layar yang menandakannya.
  //
  // Di Pega butir ini dijaga `When/IsManagerPNC_CLOSE-When.xml`: empat access group ditambah
  // TIGA Operator ID perorangan yang tertanam di dalam rule. Ketiga nama itu tidak dibawa
  // (`D-15`); yang menentukan siapa melihat butirnya sekarang adalah `M_OTORISASI_PNC`.
  InboxCloseClaim_Harness: '/inbox-close-claim',

  // MENU_ID 60 "Inbox Analyst Doctor", kelompok INBOX.
  //
  // Berbeda dari dua butir di atasnya, harness-nya ADA di export
  // (`Harness/inboxAnalystDoctor_Harness-Harness.xml`) beserta section dan Report
  // Definition-nya — sehingga kedelapan judul kolom dan ketiga penyaringnya terbaca dari
  // bukti, bukan disusun ulang.
  //
  // Di Pega butir ini dijaga `When/IsAnalystDoctor-When.xml` berkelas `@baseclass`:
  // `(Administrators OR PncAnalystDoctor) AND NOT ViewClaimPNC`. Aturan itu BELUM ditegakkan
  // (`TKT-F3-005`); yang menentukan siapa melihat butirnya sekarang adalah `M_OTORISASI_PNC`.
  //
  // Yang meredam akibatnya untuk sementara adalah penyaring identitas di server: antreannya
  // milik satu orang, sehingga pengguna yang tidak punya tugas Analyst Doctor melihat layar
  // kosong — bukan antrean orang lain. Itu peredam, bukan kendali.
  inboxAnalystDoctor_Harness: '/inbox-analyst-doctor',

  // MENU_ID 61 "Inbox RCL/PUCL", kelompok INBOX.
  //
  // Harness-nya ADA di export (`Harness/RCLPUCL_Harness-Harness.xml`) beserta keempat
  // section dan ketiga Report Definition-nya — sehingga kesembilan judul kolom dan keenam
  // penyaringnya terbaca dari bukti, bukan disusun ulang.
  //
  // Bedakan dari `ReceiveDoucument_Harness` di atas. Keduanya membaca antrean bersama yang
  // SAMA (`RCLPUCL`) pada tabel yang sama, dan hanya itu yang perlu diingat agar tidak
  // menyatukannya:
  //
  //   Inbox Manager Receive / PUCL (56)  SATU tab RCL/PUCL tanpa penyaring halus —
  //                                      superset layar ini, untuk penyelia
  //   layar ini (61)                     TIGA tab menurut perjalanan surat PUCL,
  //                                      untuk petugas yang mengerjakannya
  //
  // Pega pun memisahkannya menjadi dua menu dan dua harness, ditujukan pada peran yang
  // berbeda. Menunjuk keduanya ke satu rute akan menghilangkan partisi yang justru menjadi
  // inti layar ini.
  //
  // Bedakan pula dari `MENU_ID 62` "Inbox RCL" (`RCL_Harness`), yang BELUM dipetakan: ia
  // harness tersendiri dan belum dianalisis sama sekali.
  //
  // Di Pega butir ini dijaga `When/IsRCLPUCL-When.xml`:
  // `(Administrators OR PncRCLPUCL) AND NOT ViewClaimPNC`. Aturan itu BELUM ditegakkan
  // (`TKT-F3-004`); yang menentukan siapa melihat butirnya sekarang adalah
  // `M_OTORISASI_PNC`.
  //
  // Berbeda dari Inbox Analyst Doctor, TIDAK ADA peredam sementara di sini: antreannya
  // bersama, sehingga pengguna yang tidak berhak melihat isi penuhnya — bukan layar
  // kosong. Yang tersisa hanyalah jejak di sisi peladen (`D-59`).
  RCLPUCL_Harness: '/inbox-rcl-pucl',
  // Butir menu "Input Req Protection" — permintaan pembukaan proteksi beserta form
  // inputnya, langkah PERTAMA pada `Flow/CreateProtection_Flow.xml`.
  InputReqProtection_Harness: '/input-req-protection',

  // Butir menu "Inbox Open Protection" — antrean AKSEPTASI, langkah kedua alur yang sama.
  //
  // Perhatikan silangan namanya, dan jangan diperbaiki menjadi "seragam": butir menu
  // bernama "Inbox Open Protection" membuka harness `InputProtection_Harness`, yang judul
  // di dalamnya justru berbunyi "Inbox Accept Open Protection". Sebaliknya, butir menu
  // "Input Req Protection" membuka harness yang judulnya "Inbox Open Protection".
  //
  // Rute di bawah memakai nama dari JUDUL harness-nya, karena nama butir menunya
  // bertabrakan dengan judul layar di atas. Menukar keduanya akan mengantar petugas
  // akseptasi ke layar pemohon, dan tidak ada apa pun di layar yang menandakannya.
  InputProtection_Harness: '/inbox-accept-open-protection',
}

/**
 * routeFor mengembalikan rute layar sebuah MENU_PROGRAM, atau null bila belum ada.
 *
 * Nama program dipangkas lebih dulu: kolomnya VARCHAR2 tanpa penyeragaman, dan satu
 * spasi di ujung akan membuat butir yang sebenarnya sudah jadi tampak belum tersedia.
 */
export function routeFor(program: string): string | null {
  return MENU_ROUTES[program.trim()] ?? null
}
