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
 * 71 dari 75 butir menu belum punya layar. Butirnya tetap tampil di menu, tidak dapat
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
 * Salah satu dari sembilan itu — `InboxOutstanding_Harness` — KINI SUDAH PUNYA LAYAR.
 * Harness-nya tetap tidak ada di export; yang dipakai sebagai rujukan bentuk adalah
 * `InboxRegister_Harness`, sedangkan perilakunya diambil dari kueri, activity, dan section
 * Outstanding yang memang ada.
 */
export const MENU_ROUTES: Record<string, string> = {
  StatusClaimInbox: '/master/status-klaim',
  MasterRekening: '/master-rekening',
  StatusProgress: '/master/status-progres-1',
  // MENU_ID 64 "Inbox Laporan Klaim" — case ASM-FW-GCNMFW-Work-ReceiveDocument.
  InboxRCVApp_Harness: '/pelaporan-klaim',
  // Butir menu "Inbox Outstanding". Harness-nya tidak ada di export (`K-33`); layarnya
  // dibangun dari kueri BrowseInboxOutstanding1 beserta activity dan section-nya.
  InboxOutstanding_Harness: '/inbox-outstanding',
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
