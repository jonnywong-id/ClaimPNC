import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI, downloadAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  Advice,
  AdviceForm,
  ApprovalResponse,
  Breakdown,
  CauseOfLoss,
  ClaimListItem,
  ClaimSummaryResponse,
  InsertDolColForm,
  InsertDolColResult,
  MasterXOL,
  SummaryBusiness,
  UploadResult,
} from './types'

const PATH = '/api/inbox-xol'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian setiap kunci, dan itu BUKAN kerapian: perjanjian XOL dan
 * nilai klaimnya milik satu badan hukum, dan menyimpan dua entitas di bawah satu kunci
 * akan membuat perpindahan portal menampilkan data entitas sebelumnya (`R-20`).
 */
const keys = {
  masters: (portal: string | null, token: string | null) =>
    ['inbox-xol', 'perjanjian', portal, token] as const,

  claims: (portal: string | null, token: string | null, masterID: string) =>
    ['inbox-xol', 'klaim', portal, token, masterID] as const,

  breakdown: (
    portal: string | null,
    token: string | null,
    masterID: string,
    lossDate: string,
    cause: string,
  ) => ['inbox-xol', 'rincian', portal, token, masterID, lossDate, cause] as const,

  advices: (portal: string | null, token: string | null, form: AdviceForm) =>
    ['inbox-xol', 'pla-dla', portal, token, form.tahun, form.sebab_kerugian, form.tipe] as const,

  approvals: (portal: string | null, token: string | null) =>
    ['inbox-xol', 'persetujuan', portal, token] as const,

  causes: (portal: string | null, token: string | null) =>
    ['inbox-xol', 'sebab-kerugian', portal, token] as const,
}

/**
 * useSessionPortal menyatukan dua nilai yang SELALU dibutuhkan bersama.
 *
 * Keduanya dibaca di setiap hook di bawah, dan memisahkannya hanya akan mengulang dua
 * baris yang sama enam kali — beserta risiko salah satu yang tertinggal saat hook baru
 * ditambahkan.
 */
function useSessionPortal() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  return { token, portal, ready: token !== null && portal !== null }
}

/**
 * Hook daftar perjanjian XOL.
 *
 * Dipakai dua tempat: daftar pilihan perjanjian di tab 1, dan penyedia kurs bagi
 * perhitungan grid utama. Satu kunci cache melayani keduanya.
 *
 * Perjanjian XOL berubah setahun sekali; `staleTime` lima menit mencegah permintaan
 * berulang setiap kali tab berpindah tanpa membuat perubahan master tertahan lama.
 */
export function useMasters() {
  const { token, portal, ready } = useSessionPortal()

  return useQuery({
    queryKey: keys.masters(portal, token),
    queryFn: () =>
      callAPI<{ perjanjian: MasterXOL[] }>(`${PATH}/perjanjian`, { token, portal }),
    enabled: ready,
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Hook grid "DATA XOL BASED ON DOL AND COL" — akumulasi SELURUH perjanjian XOL.
 *
 * # Kenapa tanpa `id_master`
 *
 * Karena layar lama tidak pernah memilih satu perjanjian. `Activity/GetClaimXOL-Act.xml`
 * step 4 me-loop `MstXOL.pxResults` — seluruh perjanjian, tanpa batas awal maupun akhir —
 * lalu meng-APPEND hasil tiap perjanjian ke satu daftar, masing-masing dibagi kursnya
 * sendiri.
 *
 * Versi sebelumnya mengirim satu `id_master` dan memilihkannya sendiri di layar. Itu
 * mengarang perilaku yang tidak ada, dan akibatnya grid menampilkan satu perjanjian saja
 * — pada data nyata, perjanjian yang kebetulan tidak punya klaim.
 */
export function useClaimSummary() {
  const { token, portal, ready } = useSessionPortal()

  return useQuery({
    queryKey: keys.claims(portal, token, ''),
    queryFn: () => callAPI<ClaimSummaryResponse>(`${PATH}/klaim`, { token, portal }),
    enabled: ready,
  })
}

/**
 * Hook rincian di balik satu baris grid utama.
 *
 * Ditembak hanya saat sebuah baris dibuka. Memuat rincian SELURUH baris di muka akan
 * mengirim satu permintaan per baris — pola N+1 yang dipindahkan dari basis data ke
 * jaringan, dan sama mahalnya.
 */
export function useBreakdown(masterID: string, lossDate: string, cause: string) {
  const { token, portal, ready } = useSessionPortal()
  const active = masterID !== '' && lossDate !== '' && cause !== ''

  return useQuery({
    queryKey: keys.breakdown(portal, token, masterID, lossDate, cause),
    queryFn: () => {
      const query = new URLSearchParams({
        id_master: masterID,
        tanggal_kejadian: lossDate,
        sebab_kerugian: cause,
      })
      return callAPI<{ baris: Breakdown[] }>(`${PATH}/klaim/rincian?${query}`, {
        token,
        portal,
      })
    },
    enabled: ready && active,
  })
}

/**
 * Hook pencarian PLA/DLA yang sudah diterbitkan.
 *
 * Melayani "Generated DLA PLA XOL" dan "Cari Data DLA PLA XOL" sekaligus — keduanya
 * menjalankan aktivitas yang sama dengan parameter yang sama di sistem lama.
 *
 * `enabled` dikendalikan pemanggil, bukan disimpulkan dari isi formulir: tombol Cari
 * yang menentukan kapan permintaan berangkat, sehingga mengetik di isian tidak menembak
 * satu permintaan per ketikan.
 */
export function useAdviceSearch(form: AdviceForm, enabled: boolean) {
  const { token, portal, ready } = useSessionPortal()

  return useQuery({
    queryKey: keys.advices(portal, token, form),
    queryFn: () => {
      const query = new URLSearchParams({
        tahun: form.tahun,
        sebab_kerugian: form.sebab_kerugian,
        tipe: form.tipe,
      })
      return callAPI<{ pemberitahuan: Advice[] }>(`${PATH}/pla-dla?${query}`, {
        token,
        portal,
      })
    },
    enabled: ready && enabled,
  })
}

/** Hook antrean persetujuan — tab "Inbox XOL Komite". */
export function useApprovals(enabled: boolean) {
  const { token, portal, ready } = useSessionPortal()

  return useQuery({
    queryKey: keys.approvals(portal, token),
    queryFn: () => callAPI<ApprovalResponse>(`${PATH}/persetujuan`, { token, portal }),
    enabled: ready && enabled,
  })
}

/**
 * Hook daftar Penyebab Kerugian.
 *
 * Master ini nyaris tidak pernah berubah; `staleTime` lima menit membuatnya dibaca sekali
 * per kunjungan alih-alih setiap kali modal dibuka.
 */
export function useCauseOfLoss() {
  const { token, portal, ready } = useSessionPortal()

  return useQuery({
    queryKey: keys.causes(portal, token),
    queryFn: () =>
      callAPI<{ sebab_kerugian: CauseOfLoss[] }>(`${PATH}/sebab-kerugian`, {
        token,
        portal,
      }),
    enabled: ready,
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Hook tombol "Simpan" pada modal INSERT DOL DAN COL.
 *
 * # Ia MENYIMPAN, sejak 2026-10-08
 *
 * Sebelumnya rute ini selalu menolak, karena `POOLDATA.XOL_TABLE_ALL_KLAIM` masih dimiliki
 * Pega selama masa paralel (`P-1`). Work Owner meminta aksinya dipindahkan, dan seluruh
 * rantai rule-nya ada di export sehingga tidak ada yang ditebak — lihat
 * `internal/inboxxol/http/handler.go`, InsertDolCol.
 *
 * Satu simpan menuliskan SATU BARIS PER GROUP BUSINESS perjanjian yang dipilih, bukan satu
 * baris. Jawabannya karena itu membawa jumlahnya: itu satu-satunya cara pengguna
 * mengetahui berapa banyak yang ditulis atas namanya.
 *
 * # Kenapa cache grid dibatalkan di sini
 *
 * Karena baris baru TIDAK akan terlihat tanpa itu. Grid utama dilayani `useClaimSummary`
 * dengan kunci yang sama sepanjang sesi, sehingga tanpa invalidasi pengguna menyimpan,
 * melihat gridnya tidak berubah, lalu menyimpan sekali lagi — dan baris gandanya nyata,
 * karena sisipannya memang tidak memeriksa duplikat.
 */
export function useInsertDolCol() {
  const { token, portal } = useSessionPortal()
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (body: InsertDolColForm) =>
      callAPI<InsertDolColResult>(`${PATH}/dol-col`, { metode: 'POST', body, token, portal }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: keys.claims(portal, token, '') })
    },
  })
}

/**
 * Hook tombol "Remove All Data".
 *
 * # Ia MENGHAPUS, dan itu terbaca dari rule-nya
 *
 * `RDB List/DeleteDataInXOLSummarybasedondol-SQL.xml` berbunyi
 *
 *	delete POOLDATA.XOL_TABLE_ALL_KLAIM
 *	 where dol={…} and CAUSEOFLOSS={…} and GROUPBUSINESS={…}
 *
 * Jadi tombol itu membuang baris dari tabel yang dibaca grid di atasnya — bukan
 * mengosongkan tampilan. Versi sebelumnya menebak yang kedua dan menggambarnya sebagai
 * tombol yang bekerja; tebakan itu keliru, dan keliru ke arah yang berbahaya.
 *
 * # Rutenya TERPISAH dari INSERT DOL DAN COL, dan harus begitu
 *
 * Keduanya menyentuh tabel yang sama, tetapi sejak `POST /inbox-xol/dol-col` benar-benar
 * MENYISIPKAN, mengirim penghapusan ke jalur itu berarti menekan "Remove All Data" lalu
 * memperoleh baris BARU. Penghapusan karena itu punya jalurnya sendiri,
 * `dol-col/hapus`, dan jalur itulah yang masih dijawab penolakan.
 *
 * Penolakannya bukan karena `P-1` semata: jalur Pega yang benar-benar menjalankan DELETE
 * itu belum tertelusuri — tombol "Remove All Data" di section memanggil
 * `ToFlaggingDataXOLByRequest`, yang hanya Property-Set dan Page-New dan tidak menghapus
 * apa pun. Memindahkan penghapusan atas dasar tebakan berarti membuang baris produksi
 * tanpa tahu aturan aslinya.
 */
export function useRemoveDolCol() {
  const { token, portal } = useSessionPortal()

  return useMutation({
    mutationFn: (body: { tanggal_kejadian: string; sebab_kerugian: string }) =>
      callAPI<void>(`${PATH}/dol-col/hapus`, { metode: 'POST', body, token, portal }),
  })
}

/**
 * downloadURL menyusun alamat unduhan perhitungan PLA/DLA.
 *
 * # Kenapa alamat, bukan hook
 *
 * Karena yang diminta adalah BERKAS, bukan data yang digambar layar. Mengunduhnya lewat
 * `callAPI` berarti menahan seluruh isinya di memori peramban lebih dulu, lalu
 * membangkitkan tautan buatan — dua langkah yang tidak menambah apa pun dibanding
 * membiarkan peramban mengunduhnya sendiri.
 *
 * Token TIDAK ikut di alamat: ia dikirim sebagai cookie oleh peramban, dan nilai di URL
 * ikut tercatat di log peramban, log proxy, dan header Referer.
 */
export function downloadURL(form: AdviceForm): string {
  const query = new URLSearchParams({
    tahun: form.tahun,
    sebab_kerugian: form.sebab_kerugian,
    tipe: form.tipe,
  })
  return `${PATH}/pla-dla/unduh?${query}`
}

/**
 * exportClaimDetailURL menyusun alamat tombol "Export to Excel" pada grid rincian.
 *
 * Alasan memakai alamat, bukan hook, sama dengan `downloadURL` di atas.
 *
 * Keempat isian di bawah persis parameter tombol lama — `dol`, `coldesc`, `grpbzid`, dan
 * `biz` (`Section/DetailValueClaimXOL-Section.xml:4530-4570`), seluruhnya diambil dari
 * BARIS yang diekspor:
 *
 * - `kode_group_business` menentukan KUERI MANA yang dipakai server. Sistem lama memilih
 *   cabangnya dari `Param.grpbzid`, dengan `'10004'` untuk MBU dan satu cabang tersendiri
 *   untuk treaty inward — baris treaty karena itu mengirim `'treaty'`.
 * - `nama_group_business` tidak menyaring apa pun; ia hanya nama berkasnya, sama seperti
 *   `Param.biz` yang di seluruh activity lama hanya dipakai di satu tempat: menyusun
 *   `"Detail Claim XOL " + Param.biz`.
 */
export function exportClaimDetailURL(
  lossDate: string,
  cause: string,
  businessGroupID: string,
  businessGroupName: string,
): string {
  const query = new URLSearchParams({
    tanggal_kejadian: lossDate,
    sebab_kerugian: cause,
    kode_group_business: businessGroupID,
    nama_group_business: businessGroupName,
  })
  return `${PATH}/klaim/rincian/unduh?${query}`
}

/**
 * uploadTemplateURL menyusun alamat tautan "Format MBU Salvage" dan "Format Inward".
 *
 * `jenis` persis `Param.jenis` tombol lama: `'1'` untuk MBU Salvage, `'2'` untuk Inward
 * (`Section/DetailValueClaimXOL-Section.xml:7094` dan `:7480`).
 */
export function uploadTemplateURL(jenis: '1' | '2'): string {
  return `${PATH}/format-unggah?jenis=${jenis}`
}

/**
 * Hook tombol "Upload MBU Salvage" — unggahan yang sudah berjalan penuh.
 *
 * Seluruh rantai rule-nya ada di export, sehingga tidak ada pemetaan kolom yang ditebak:
 * `PNCUploadClaimCSV_MBUSalvage-FA.xml` → `ConvertDataCsvSalvageMBUToPage-Act.xml` →
 * `InsertDataSalvageMBU-SQL.xml`.
 *
 * Berkasnya dikirim sebagai `FormData`, bukan JSON ber-base64: isi CSV tidak perlu
 * membengkak sepertiga hanya untuk melewati jalur yang sama.
 */
export function useUploadSalvageMBU() {
  const { token, portal } = useSessionPortal()

  return useMutation({
    mutationFn: (berkas: File) => {
      const body = new FormData()
      body.append('berkas', berkas)
      return callAPI<UploadResult>(`${PATH}/unggah/mbu-salvage`, {
        metode: 'POST',
        body,
        token,
        portal,
      })
    },
  })
}

/**
 * Hook tombol "Upload Inward" — masih dijawab penolakan server.
 *
 * # Kenapa ia tetap ditembakkan padahal pasti ditolak
 *
 * Alasannya sama dengan "Remove All Data": yang menolak adalah server beserta sebabnya,
 * bukan layar yang diam-diam menyembunyikan tombolnya.
 *
 * Berkasnya TIDAK ikut dikirim, karena pemetaan kolomnya memang belum dapat dibaca
 * siapa pun — activity `ConvertDataCsvInwardToPage` tidak ada di export. Mengirim isi
 * berkas ke server yang pasti menolaknya hanya memindahkan data nasabah tanpa satu pun
 * kegunaan.
 */
export function useUploadInward() {
  const { token, portal } = useSessionPortal()

  return useMutation({
    mutationFn: () =>
      callAPI<void>(`${PATH}/unggah/inward`, { metode: 'POST', body: {}, token, portal }),
  })
}

/**
 * Hook grid "Summary Data XOL" pada layar rincian.
 *
 * Penyaringnya Tanggal Kejadian dan Penyebab Kerugian — keduanya milik baris yang sedang
 * dibuka. Ditembak hanya saat baris benar-benar terbentang, sehingga membuka layar tidak
 * memanggilnya sama sekali.
 */
export function useSummaryBusiness(lossDate: string, cause: string) {
  const { token, portal, ready } = useSessionPortal()

  return useQuery({
    queryKey: ['inbox-xol', 'summary', portal, token, lossDate, cause],
    queryFn: () => {
      const query = new URLSearchParams({
        tanggal_kejadian: lossDate,
        sebab_kerugian: cause,
      })
      return callAPI<{ baris: SummaryBusiness[] }>(`${PATH}/klaim/summary?${query}`, {
        token,
        portal,
      })
    },
    enabled: ready && lossDate !== '' && cause !== '',
  })
}

/**
 * Hook grid "No Klaim" pada layar rincian.
 *
 * Di sistem lama ia diisi `ShowDataKlaimXOLKlaimBeforeGenerated` — activity di balik DUA
 * tombol, "Pilih" dan "Show All Data". Keduanya memuat daftar yang SAMA: penyaringnya
 * hanya Tanggal Kejadian dan Penyebab Kerugian, bukan perjanjian yang dipilih.
 *
 * Karena itu di sini ia dimuat begitu barisnya terbentang, tanpa menunggu tombol —
 * menyembunyikan daftar di balik tombol yang tidak menyaring apa pun hanya menambah satu
 * langkah tanpa menambah pilihan.
 */
export function useClaimList(lossDate: string, cause: string) {
  const { token, portal, ready } = useSessionPortal()

  return useQuery({
    queryKey: ['inbox-xol', 'daftar-klaim', portal, token, lossDate, cause],
    queryFn: () => {
      const query = new URLSearchParams({
        tanggal_kejadian: lossDate,
        sebab_kerugian: cause,
      })
      return callAPI<{ baris: ClaimListItem[] }>(`${PATH}/klaim/daftar?${query}`, {
        token,
        portal,
      })
    },
    enabled: ready && lossDate !== '' && cause !== '',
  })
}

/**
 * Hook unduhan berkas — dipakai seluruh tautan unduh modul ini.
 *
 * # Kenapa tidak cukup `<a href>`
 *
 * Karena sesi dikirim sebagai header `Authorization`, BUKAN sebagai cookie, dan peramban
 * tidak mengirim header apa pun pada navigasi biasa. Tautan polos karena itu selalu
 * dijawab `sesi_tidak_sah` — bukan berkas.
 *
 * Komentar pada `downloadURL` sebelumnya menyatakan sebaliknya, dan itu KELIRU: tidak ada
 * satu pun cookie sesi di aplikasi ini.
 *
 * Berkasnya diambil sebagai blob lalu diserahkan ke peramban sebagai unduhan — itulah
 * yang dikerjakan `downloadAPI` bersama seluruh modul lain.
 */
export function useUnduhBerkas() {
  const { token, portal } = useSessionPortal()

  return useMutation({
    mutationFn: ({ alamat, namaBerkas }: { alamat: string; namaBerkas: string }) =>
      downloadAPI(alamat, namaBerkas, { token, portal }),
  })
}

/**
 * Hook tombol "Generate PLA" dan "Generate DLA" pada layar rincian.
 *
 * Keduanya memanggil activity yang SAMA di sistem lama — `GenerateXOLByType`, dengan
 * parameter `type` berisi `"PLA"` atau `"DLA"`
 * (`Section/DetailValueClaimXOL-Section.xml:23215` dan `:24351`).
 *
 * Isinya belum dipindahkan: activity itu 257 KB dan memanggil dua activity lain sebesar
 * 543 KB dan 524 KB, ditambah delapan rule SQL. Seluruhnya ADA di export — yang belum ada
 * adalah pembacaannya. Sampai itu selesai, server menolak beserta sebabnya.
 */
export function useGenerateAdvice(tipe: 'pla' | 'dla') {
  const { token, portal } = useSessionPortal()

  return useMutation({
    mutationFn: (body: { tanggal_kejadian: string; sebab_kerugian: string }) =>
      callAPI<void>(`${PATH}/generate/${tipe}`, { metode: 'POST', body, token, portal }),
  })
}
