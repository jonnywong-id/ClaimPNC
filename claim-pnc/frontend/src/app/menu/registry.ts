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
 */
export const MENU_ROUTES: Record<string, string> = {
  StatusClaimInbox: '/master/status-klaim',
  MasterRekening: '/master/rekening',
  StatusProgress: '/master/status-progres-1',
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
  // Dua butir menu bersaudara sengaja TIDAK dipetakan di sini, dan keduanya mudah
  // tertukar dengannya:
  //
  //   MENU_ID 38  DetailCauseOfLoss            rinciannya (`D_CAUSE_OF_LOSS`), belum ada
  //   MENU_ID 21  CauseOfLossInboxSimasOnline  varian Simas Online, belum ada
  //
  // Yang kedua patut diperhatikan khusus: ia menulis TABEL YANG SAMA dengan layar ini
  // lewat Pega, sehingga selama ia belum dipindahkan, `P-1` belum terpenuhi utuh. Lihat
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
  // MENU_ID 55 "Inbox Claim Treaty Non Prop" (`InboxClaimNonProp_Harness`) adalah layar
  // saudara yang berdiri sendiri dan BELUM dibangun; ia sengaja tidak dipetakan ke rute
  // yang sama, karena keduanya membaca kueri dan tabel yang berbeda.
  InboxClaimTreaty_Harness: '/inbox-claim-treaty-prop',

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
  InboxRCVApp_Harness: '/inbox/laporan-klaim',
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
