import { useMutation, useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  Advice,
  AdviceForm,
  ApprovalResponse,
  Breakdown,
  CauseOfLoss,
  ClaimSummaryResponse,
  InsertDolColForm,
  MasterXOL,
  SummaryBusiness,
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
 * # Permintaannya benar-benar DIKIRIM, dan memang harus
 *
 * Jawaban penolakannya datang dari server, bukan dikarang layar. Alasannya ada tiga:
 *
 *  1. `internal/inboxxol/http/handler.go` menyediakan rute ini KHUSUS untuk itu —
 *     `RejectWrite` menjawab `409` beserta sebabnya, bukan `404` yang terbaca seperti
 *     salah alamat.
 *  2. Pada hari kewenangan menulis berpindah ke sini (`P-1` gugur), yang berubah cukup
 *     satu handler. Layar yang menolak sendiri harus ikut disunting, dan itulah jenis
 *     suntingan yang terlupakan.
 *  3. Penolakan yang tercatat di log server dapat ditelusuri; penolakan yang hanya hidup
 *     di peramban tidak meninggalkan jejak apa pun.
 *
 * Isi badan permintaan dikirim apa adanya supaya penolakannya tercatat bersama data yang
 * hendak disimpan — itu yang membedakan "pengguna mencoba menyimpan" dari "pengguna
 * menekan tombol tanpa mengisi apa-apa".
 */
export function useInsertDolCol() {
  const { token, portal } = useSessionPortal()

  return useMutation({
    mutationFn: (body: InsertDolColForm) =>
      callAPI<void>(`${PATH}/dol-col`, { metode: 'POST', body, token, portal }),
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
 * Rutenya sama dengan INSERT DOL DAN COL karena TABELNYA sama — `routes.go` memetakan
 * `dol-col` ke `POOLDATA.XOL_TABLE_ALL_KLAIM`. Selama masa paralel tabel itu dimiliki
 * Pega (`P-1`), sehingga server menolak dengan menyebutkan sebabnya.
 */
export function useRemoveDolCol() {
  const { token, portal } = useSessionPortal()

  return useMutation({
    mutationFn: (body: { tanggal_kejadian: string; sebab_kerugian: string }) =>
      callAPI<void>(`${PATH}/dol-col`, { metode: 'POST', body, token, portal }),
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
