import { useQuery } from '@tanstack/react-query'

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
  MasterXOL,
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
 * Hook akumulasi klaim satu perjanjian — grid "DATA XOL BASED ON DOL AND COL".
 *
 * Tidak ditembak sebelum sebuah perjanjian dipilih. Backend memang menolak permintaan
 * tanpa `id_master`, tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan validasi sebelum pengguna sempat memilih apa pun.
 */
export function useClaimSummary(masterID: string) {
  const { token, portal, ready } = useSessionPortal()

  return useQuery({
    queryKey: keys.claims(portal, token, masterID),
    queryFn: () =>
      callAPI<ClaimSummaryResponse>(
        `${PATH}/klaim?id_master=${encodeURIComponent(masterID)}`,
        { token, portal },
      ),
    enabled: ready && masterID !== '',
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
