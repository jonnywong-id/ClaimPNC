/**
 * Jalur halaman klaim, dipakai lebih dari satu layar.
 *
 * Ia tinggal di `lib/` dan bukan di salah satu modul, karena aturan frontend melarang satu
 * fitur mengimpor dari fitur lain (`07-TECHNICAL-STRATEGY` §3 aturan 1).
 */

/**
 * claimDetailPath adalah jalur halaman rincian sebuah klaim.
 *
 * Rutenya `/registrasi/klaim/:claimID`, dan `claimID` adalah NOMOR klaim — bukan `pzInsKey`.
 * Kunci teknis Pega tidak pernah dipakai sebagai alamat di aplikasi ini (`D-22`).
 *
 * # Kedua format nomor dilayani, dan itu terbukti dari kuerinya
 *
 * `klaim_ambil_per_nomor` membaca `POOLDATA.T_CLAIM_PNC WHERE CLAIMNO = :1` — tabel klaim
 * WARISAN, yang memuat `PNC-xxxx`. Halaman tujuannya karena itu melayani klaim warisan
 * maupun klaim `PNCN.YY.xxxx` terbitan sistem ini (`D-71`).
 *
 * # Kenapa TIDAK ada `isOpenableHere` di sini
 *
 * Fungsi dengan nama itu ada di `modules/inbox-outstanding/OutstandingPage.tsx`, dan ia
 * menolak klaim `PNC-xxxx`. Itu **kebijakan layar My Inbox**, bukan batas kemampuan
 * halaman tujuannya: My Inbox memilih tidak menautkan klaim yang masih dikerjakan di Pega
 * (`P-3`).
 *
 * Menaikkannya ke sini akan membuat layar lain menyangka batas itu berlaku umum. Layar
 * Inbox Manager Admin justru sebaliknya — di Pega, tombolnya membuka formulir Register_Flow
 * untuk SETIAP baris antrean, apa pun format nomornya (`SetAssignmentInboxReg_act`).
 * Setiap layar karena itu memutuskan sendiri baris mana yang ditautkannya.
 */
export function claimDetailPath(claimNumber: string): string {
  return `/registrasi/klaim/${encodeURIComponent(claimNumber)}`
}
