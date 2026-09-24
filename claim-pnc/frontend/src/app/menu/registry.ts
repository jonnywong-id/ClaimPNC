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
 * 57 dari 75 butir menu belum punya layar. Butirnya tetap tampil di menu, tidak dapat
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
 */
export const MENU_ROUTES: Record<string, string> = {
  StatusClaimInbox: '/master/status-klaim',
  MasterRekening: '/master-rekening',
  StatusProgress: '/master/status-progres-1',
  StatusProgress2: '/master/status-progres-2',
  // MENU_ID 25 "Master Penolakan Klaim". Satu butir menu, satu layar, DUA master —
  // Penolakan Klaim dan Penolakan Komite dipilih lewat tab di dalamnya, persis seperti
  // dua tombol pada harness aslinya.
  PNC_MasterTolakKlaim: '/master/penolakan-klaim',
  // MENU_ID 26 "Master Auto Claim". Satu butir menu, satu layar, EMPAT tab — Master
  // Auto Klaim, Komite Approval, Waiting Approval, dan Reject — persis seperti empat
  // section pada `Harness/AutoKlaim-Harness.xml`.
  AutoKlaim: '/master/auto-claim',
  // MENU_ID 27 "Master Pasal Kerugian". Nama harness-nya menyebut "Rejected" tetapi
  // judul di layarnya "Detail Pasal Kerugian", dan yang dikelolanya bukan penolakan
  // melainkan butir ketentuan polis — jaminan, pengecualian, dan notifikasi.
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
  // MENU_ID 28 "Master Bengkel". Satu butir menu, satu layar, TIGA tab — Approve,
  // Waiting Approval, dan Reject — persis seperti ketiga tab pada
  // `Section/BrowseMasterHE-Section.xml`.
  BengkelHE: '/master/bengkel',
  // MENU_ID 30 "Master Panel". Satu butir menu, satu layar, TIGA tab — Approve, Waiting
  // Approval, dan Reject — persis seperti ketiga section pada
  // `Section/BrowsePanelHE-Section.xml`.
  //
  // Layar master pertama yang mengelola BARIS ANAK: daftar lokasi pada setiap panel,
  // tersimpan di POOLDATA.LOKASI_PANEL_HE.
  MasterPanel_HE: '/master/panel',
  // MENU_ID 31 "Master Sparepart". Satu butir menu, satu layar, TIGA tab — Approve,
  // Reject, dan Waiting Approval — persis seperti ketiga section pada
  // `Section/BrowseMasterSparepartHE-Section.xml`.
  //
  // Nama kuncinya `SparePart_HE` dengan P besar di tengah, persis seperti yang tertulis
  // di POOLDATA.M_MENU_APLIKASI_PNC. Huruf besar-kecilnya dikirim server apa adanya.
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

  // MENU_ID 64 "Inbox Laporan Klaim" — case ASM-FW-GCNMFW-Work-ReceiveDocument.
  InboxRCVApp_Harness: '/pelaporan-klaim',

  // MENU_ID 76 "View History Claim". Layar pencarian riwayat klaim, bukan inbox —
  // pembedaannya ditetapkan `D-79` dan menentukan modul pemiliknya.
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
