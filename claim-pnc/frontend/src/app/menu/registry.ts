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
 * 72 dari 75 butir menu belum punya layar. Butirnya tetap tampil di menu, tidak dapat
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
  // JANGAN tertukar dengan MENU_ID 20 "Master Penyebab Kerugian" (`CauseOfLossInbox`),
  // layar Master COL biasa atas tabel yang SAMA tetapi tanpa isian ID Master Kerugian
  // dan tanpa pemetaan bisnis. Ia belum dibangun, dan butirnya tetap tampil sebagai
  // "belum tersedia".
  CauseOfLossInboxSimasOnline: '/master/col-simas-online',
  // MENU_ID 40 "Daftar Tipe Dokumen", di bawah kelompok MASTER.
  //
  // JANGAN tertukar dengan dua master TURUNANNYA, yang merujuk ID milik layar ini dan
  // keduanya belum dibangun:
  //
  //   `ListDetTypeDocument`   "Daftar Detail Tipe Dokumen"      (V_LST_DET_TYPE_DOC)
  //   `DetTypeDocumenBisnis`  "Detail Tipe Dokumen per Bisnis"  (LST_TYPE_DOC_BUSINESS)
  //
  // Keduanya tetap tampil sebagai "belum tersedia".
  ListDocumentTypeInbox: '/master/tipe-dokumen',
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
