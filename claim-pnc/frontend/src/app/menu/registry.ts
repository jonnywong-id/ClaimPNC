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
 * 62 dari 75 butir menu belum punya layar. Butirnya tetap tampil di menu, tidak dapat
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
  // MENU_ID 29 "Master Supplier". Satu butir menu, satu layar, TANPA tab — layar lamanya
  // memang satu grid dengan tiga tombol (New Supplier, Edit, Refresh) dan tidak punya
  // penyaring status apa pun (`Section/InboxMasterSupplier-Section.xml`).
  //
  // Satu-satunya master yang seluruh isinya tinggal di SATU kolom JSONDATA: `M_SUPPLIER`
  // hanya punya ID, OLDID, dan JSONDATA.
  MasterSupplier: '/master/supplier',

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
