import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  CheckerResponse,
  ListResponse,
  MetadataResponse,
  SendPostAuditRequest,
  SendPostAuditResponse,
  SubmitDecisionRequest,
  SubmitDecisionResponse,
} from './types'

const PATH = '/api/inbox-compliance'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: antrean kepatuhan satu badan
 * hukum bukan antrean badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan
 * membuat perpindahan portal menampilkan pekerjaan entitas sebelumnya (`R-20`).
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['inbox-compliance', 'tab', portal, token] as const,

  list: (portal: string | null, token: string | null, tab: string, page: number) =>
    ['inbox-compliance', 'daftar', portal, token, tab, page] as const,

  checker: (portal: string | null, token: string | null, reference: string) =>
    ['inbox-compliance', 'checker', portal, token, reference] as const,
}

/**
 * Hook keterangan layar — daftar tab beserta kolomnya.
 *
 * # Kenapa bentuk layar datang dari server
 *
 * Karena kolom tiap tab adalah HASIL PEMBACAAN export Pega, dan tempat pembacaan itu
 * tercatat adalah backend (`internal/inboxcompliance/tab.go`). Menyalinnya ke layar berarti
 * daftar yang sama hidup di dua tempat, dan yang satu akan tertinggal saat yang lain
 * diperbaiki.
 *
 * Hal yang sama berlaku untuk penanda `tersedia` dan kalimat `penghalang` pada tab Post
 * Audit: begitu daftar kolom `POOLDATA.T_CLAIM_COMPLIANCE_H` tiba dan kuerinya ditulis,
 * tab itu hidup tanpa satu baris pun di layar ini disunting.
 */
export function useInboxComplianceMetadata() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.metadata(portal, token),
    queryFn: () => callAPI<MetadataResponse>(`${PATH}/tab`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan dari
    // data. Mengambilnya ulang tiap kali tab berpindah hanya menambah perjalanan jaringan
    // tanpa satu pun manfaat.
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * Hook isi satu tab.
 *
 * # Kenapa paginasi dikerjakan di SERVER
 *
 * Karena yang dibaca adalah `DATAPEGA.PC_ASM_FW_GCNMFW_WORK` yang berisi puluhan juta baris
 * (`D-10`). Berbeda dari layar Inbox Admin — yang atas keputusan Work Owner menarik seluruh
 * baris lalu memotongnya di server — layar ini memotong halamannya di basis data, sehingga
 * permintaan pertama tidak menjadi lebih lambat pada antrean yang panjang.
 *
 * `enabled` dipakai menahan permintaan pada tab yang belum dapat dilayani. Memanggilnya
 * tetap aman — server menjawab 503 dengan penjelasan — tetapi memanggil sesuatu yang sudah
 * pasti gagal hanya menambah galat di log tanpa menambah keterangan apa pun.
 */
export function useInboxComplianceList(tab: string, page: number, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(portal, token, tab, page),
    queryFn: () => callAPI<ListResponse>(buildPath(tab, page), { token, portal }),
    enabled: enabled && token !== null && portal !== null,

    // Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
    // menjadi kosong lalu terisi lagi.
    placeholderData: (previous) => previous,

    // Antrean kerja berubah saat petugas lain menyelesaikan pekerjaannya, jadi cache-nya
    // pendek — sama dengan layar antrean kerja lain.
    staleTime: 15 * 1000,
  })
}

/**
 * buildPath menyusun parameter permintaan.
 *
 * Isian yang kosong TIDAK dikirim, bukan dikirim sebagai teks kosong: server membedakan
 * "tidak dikirim" dari "dikirim kosong", dan pada `tab` yang kedua akan menjadi tab bawaan
 * sementara yang pertama memang itu yang diinginkan.
 */
function buildPath(tab: string, page: number): string {
  const params = new URLSearchParams()

  if (tab) params.set('tab', tab)
  if (page > 1) params.set('halaman', String(page))

  const query = params.toString()
  return query ? `${PATH}?${query}` : PATH
}

/**
 * Hook pengiriman satu klaim dari antrean Compliance ke Post Audit.
 *
 * # Kenapa dua daftar sekaligus yang disegarkan
 *
 * Karena satu pengiriman mengubah keduanya: barisnya MUNCUL di tab Post Audit, dan di Pega
 * klaimnya berhenti menunggu di antrean Compliance. Menyegarkan satu saja membuat layar
 * menampilkan keadaan yang tidak pernah ada.
 *
 * Kuncinya disegarkan menurut awalannya, bukan satu per satu, supaya halaman keberapa pun
 * yang sedang terbuka ikut termuat ulang.
 */
export function useSendToPostAudit() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (body: SendPostAuditRequest) =>
      callAPI<SendPostAuditResponse>(`${PATH}/post-audit`, {
        token,
        portal,
        metode: 'POST',
        body,
      }),

    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['inbox-compliance', 'daftar'] })
    },
  })
}

/**
 * Hook pembukaan form Compliance Checker atas satu klaim.
 *
 * # Kenapa `retry: false`
 *
 * Karena galat yang paling mungkin terjadi di sini BUKAN gangguan jaringan melainkan
 * `ErrClaimNotInQueue` — klaimnya sudah diputuskan petugas lain, atau sudah selesai.
 * Mengulanginya tiga kali tidak akan mengubah jawabannya; yang ia lakukan hanyalah menunda
 * pesan yang perlu segera dibaca petugas.
 *
 * # Kenapa `staleTime` nol
 *
 * Berbeda dari daftarnya, form ini dibuka untuk DIKERJAKAN. Menampilkan keputusan yang
 * tersimpan beberapa detik lalu oleh petugas lain — lalu menimpanya — adalah kelas
 * kesalahan yang tidak sepadan dengan satu perjalanan jaringan yang dihemat.
 */
export function useComplianceChecker(reference: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.checker(portal, token, reference),
    queryFn: () =>
      callAPI<CheckerResponse>(`${PATH}/${encodeURIComponent(reference)}`, {
        token,
        portal,
      }),
    enabled: reference !== '' && token !== null && portal !== null,
    retry: false,
    staleTime: 0,
  })
}

/**
 * Hook penyimpanan keputusan Compliance — padanan tombol "Simpan Data".
 *
 * # Tiga cache yang disegarkan, dan kenapa ketiganya
 *
 *   - `checker`  — form itu sendiri, supaya keputusan yang baru tersimpan terbaca saat
 *                  form dibuka ulang.
 *   - `daftar`   — kedua tab. Pada pilihan Bayar/PostAudit barisnya MUNCUL di tab Post
 *                  Audit, dan menyegarkan satu saja membuat layar menampilkan keadaan
 *                  yang tidak pernah ada.
 *
 * Yang TIDAK berubah adalah antrean Compliance di Pega: klaimnya tetap menunggu di sana,
 * karena tabel klaim masih dimiliki Pega (`P-1`). Jadi baris yang baru diputuskan akan
 * tetap tampil di tab Compliance setelah disegarkan — itu BUKAN cache yang basi,
 * melainkan keterbatasan yang dinyatakan di `keterbatasan`.
 */
export function useSubmitComplianceDecision(reference: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (body: SubmitDecisionRequest) =>
      callAPI<SubmitDecisionResponse>(
        `${PATH}/${encodeURIComponent(reference)}/keputusan`,
        { token, portal, metode: 'POST', body },
      ),

    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['inbox-compliance', 'checker'] })
      client.invalidateQueries({ queryKey: ['inbox-compliance', 'daftar'] })
    },
  })
}
