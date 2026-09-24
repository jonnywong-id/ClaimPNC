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
 * diklik, dan bertanda "belum tersedia" — keputusan Work Owner 2026-09-18. Dengan
 * begitu kemajuan migrasi terbaca langsung dari layar, dan pengguna tidak melaporkan
 * menu yang "hilang".
 *
 * Sembilan di antaranya bahkan menunjuk harness yang TIDAK ADA di export Pega
 * (`DataMemberReas`, `DetailMasterPasalAI`, `InboxCloseClaim_Harness`,
 * `InboxOutstanding_Harness`, `InboxRequestSalvage`, `InboxServiceCenter`,
 * `LostAdjuster_harness`, `PNCViewClaim`, `ReportProduksiPA_harnes`) — memperjelas
 * `K-33`. Ditambah MENU_ID 83 "Report Adjuster" yang MENU_PROGRAM-nya memang kosong.
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
  // MENU_ID 64 "Inbox Laporan Klaim". Nama programnya `InboxRCVApp_Harness` dan tidak
  // menyebut laporan sama sekali; rutenya mengikuti nama BUTIR MENU (`D-81`).
  InboxRCVApp_Harness: '/pelaporan-klaim',
  // MENU_ID 76 "View History Claim", di bawah kelompok VIEW.
  PNCSearchKlaim: '/riwayat-klaim',
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
