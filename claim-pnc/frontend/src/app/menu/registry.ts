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
 * 50 dari 75 butir menu belum punya layar. Butirnya tetap tampil di menu, tidak dapat
 * 70 dari 75 butir menu belum punya layar. Butirnya tetap tampil di menu, tidak dapat
 * diklik, dan bertanda "belum tersedia" — keputusan Work Owner 2026-09-18. Dengan
 * begitu kemajuan migrasi terbaca langsung dari layar, dan pengguna tidak melaporkan
 * menu yang "hilang".
 *
 * Sembilan di antaranya bahkan menunjuk harness yang TIDAK ADA di export Pega
 * (`DataMemberReas`, `DetailMasterPasalAI`, `InboxCloseClaim_Harness`,
 * `InboxOutstanding_Harness`, `InboxRequestSalvage`, `InboxServiceCenter`,
 * `LostAdjuster_harness`, `PNCViewClaim`, `ReportProduksiPA_harnes`) — memperjelas
 * `K-33`. Ditambah MENU_ID 83 "Report Adjuster" yang MENU_PROGRAM-nya memang kosong.
 *
 * DUA dari sembilan itu KINI SUDAH PUNYA LAYAR, dan keduanya dibangun dengan cara yang
 * sama — dari kueri, activity, dan section yang memang ada, bukan dari harness-nya:
 *
 *   `InboxOutstanding_Harness`   rujukan bentuknya `InboxRegister_Harness`
 *   `InboxCloseClaim_Harness`    rujukan bentuknya `InboxManagerReopen1_Sec`, section yang
 *                                di dalamnya sendiri berjudul "Inbox Close Claim"
 */
export const MENU_ROUTES: Record<string, string> = {
  StatusClaimInbox: '/master/status-klaim',
  MasterRekening: '/master/rekening',
  StatusProgress: '/master/status-progres-1',
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
  // Butir menu "Inbox Outstanding". Harness-nya tidak ada di export (`K-33`); layarnya
  // dibangun dari kueri BrowseInboxOutstanding1 beserta activity dan section-nya.
  InboxOutstanding_Harness: '/inbox-outstanding',
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
  // `StatusProgress2` (MENU_ID 24) sengaja TIDAK ada di sini meski layarnya sudah ada:
  // backend-nya belum pernah ditulis di commit mana pun, dan `/api/master/status-progres-2`
  // tidak ada. Memetakannya hanya akan mengubah label jujur "belum tersedia" menjadi layar
  // yang tampak rusak.

  // MENU_ID 25. SATU layar untuk DUA master — Penolakan Klaim dan Penolakan Komite —
  // karena di Pega pun keduanya satu butir menu. Pemilihannya tab di dalam layar.
  PNC_MasterTolakKlaim: '/master/penolakan-klaim',
  // MENU_ID 26. Empat tab atas tabel yang sama, hanya berbeda saringan.
  AutoKlaim: '/master/auto-claim',
  // MENU_ID 27 "Master Pasal Kerugian". Nama programnya menyebut "Rejected" tetapi
  // layarnya master pasal, bukan daftar penolakan — nama itu dibaca apa adanya dari tabel
  // menu, dan memperbaikinya menempuh `D-63`.
  DetailMasterPasalRejected: '/master/pasal-kerugian',

  // MENU_ID 28, 30, 31 — keluarga alat berat, berbagi satu activity persetujuan yang sama
  // di Pega (`Activity/SetApprovalAllMaster`). Ketiganya beserta Master Supplier di bawah
  // membaca tabel yang `D-34` keluarkan dari lingkup migrasi; Work Owner memutuskan pada
  // 2026-09-22 bahwa seluruh modul dari cabang `fran-masuk-master` harus ada. Lihat
  // catatan di kepala `backend/cmd/claimpnc/modules.go`.
  BengkelHE: '/master/bengkel',
  MasterPanel_HE: '/master/panel',
  SparePart_HE: '/master/sparepart',
  // MENU_ID 29. Seluruh isinya tinggal di satu kolom JSONDATA.
  MasterSupplier: '/master/supplier',

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
  // MENU_ID 84 "Report KPI PNC", di bawah kelompok REPORT.
  //
  // Rutenya `/report-kpi`, bukan `/report-kpi-pnc`: akhiran "PNC" dibuang karena seluruh
  // aplikasi ini adalah Claim PNC — sama seperti `masterstatus` membuang "Klaim" dari
  // "Master Status Klaim" (`D-81`).
  //
  // JANGAN tertukar dengan MENU_ID 83 "Report Adjuster", yang MENU_PROGRAM-nya memang
  // KOSONG di basis data dan karena itu tidak dapat dipetakan sama sekali.
  //
  // CATATAN PEMULIHAN: baris ini sempat TERHAPUS pada 2026-09-25 oleh `git checkout`
  // yang dijalankan sesi lain untuk membatalkan pemformatan ulang Prettier. Kodenya
  // dipulihkan apa adanya; komentar aslinya tidak dapat dipulihkan utuh.
  ReportKPIHarness: '/report-kpi',
  // MENU_ID 85 "Report Klaim", di bawah kelompok REPORT.
  //
  // Harness-nya `PNCTATReport` — dan namanya menyesatkan: ia BUKAN layar laporan TAT
  // melainkan **halaman peluncur berisi 28 panel laporan**, yang REPORT TAT hanya salah
  // satunya. Nama harness itu tampaknya tertinggal dari saat panelnya masih satu.
  //
  // "REPORT ADJUSTER" juga muncul sebagai salah satu dari 28 panel di dalam layar ini,
  // dan itu hal yang BERBEDA dari butir menu MENU_ID 83. Panel itu pun terhalang: kedua
  // Report Definition-nya tidak ada di export (`R-16`).
  //
  // Rutenya mengikuti nama BUTIR MENU, bukan nama harness (`D-81`).
  PNCTATReport: '/report-klaim',
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
