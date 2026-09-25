import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  Decision,
  ProtectionDetail,
  ProtectionFilter,
  ProtectionListResponse,
} from './types'

const PATH = '/api/inbox-accept-open-protection'

/** Banyaknya baris per halaman; mengikuti `pyPageSize` layar lama. */
export const PAGE_SIZE = 50

/**
 * Kunci cache dikumpulkan di satu tempat.
 *
 * PORTAL dan ANTREAN keduanya menjadi bagian kunci. Portal karena dua portal adalah dua
 * badan hukum dengan basis data berbeda (`ADR-0030`); antrean karena PREMI dan NON PREMI
 * adalah dua daftar yang berbeda isinya — menyatukannya akan membuat perpindahan tab
 * menampilkan isi tab sebelumnya sampai permintaan baru selesai.
 */
const keys = {
  all: ['inbox-accept-open-protection'] as const,
  list: (portal: string | null, token: string | null, f: ProtectionFilter) =>
    [
      'inbox-accept-open-protection',
      'daftar',
      portal,
      token,
      f.queue,
      f.search ?? '',
      f.offset ?? 0,
    ] as const,
  detail: (portal: string | null, token: string | null, number: string) =>
    ['inbox-accept-open-protection', 'detail', portal, token, number] as const,
}

function buildPath(f: ProtectionFilter): string {
  const params = new URLSearchParams()
  params.set('antrean', f.queue)
  if (f.search?.trim()) params.set('cari', f.search.trim())
  if (f.offset) params.set('lewati', String(f.offset))
  params.set('batas', String(PAGE_SIZE))

  return `${PATH}?${params.toString()}`
}

/**
 * Hook antrean akseptasi.
 *
 * Menggantikan `InboxOpenProtection2_RD` (NON PREMI) dan
 * `InboxOpenProtection2_RD_collection` (PREMI). Ketiga syaratnya —
 * `CaseID IS NOT NULL AND PolicyNo IS NOT NULL AND AcceptStatus IS NULL` — ditegakkan
 * server dan tidak dapat dimatikan dari sini.
 */
export function useAcceptQueue(filter: ProtectionFilter) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(portal, token, filter),
    queryFn: () => callAPI<ProtectionListResponse>(buildPath(filter), { token, portal }),
    enabled: token !== null && portal !== null,

    // Antrean BERSAMA: baris dapat hilang kapan saja karena diputuskan petugas lain. Data
    // dianggap usang seketika supaya perpindahan tab dan Muat ulang selalu menembak server.
    staleTime: 0,
  })
}

/** Hook pembacaan satu proteksi untuk form akseptasi. */
export function useProtectionDetail(number: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.detail(portal, token, number ?? ''),
    queryFn: () =>
      callAPI<ProtectionDetail>(`${PATH}/${encodeURIComponent(number ?? '')}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && number !== null && number !== '',
  })
}

/**
 * Hook keputusan akseptasi: menyetujui atau menolak satu permintaan proteksi.
 *
 * Pelaku dan waktunya TIDAK dikirim dari sini — keduanya diterbitkan server dari sesi dan
 * jam aplikasi. `D-59` menetapkan tidak ada pemisahan tugas formal, sehingga kolom
 * `DIAKSEP_OLEH` adalah satu-satunya kontrol pengimbang atas persetujuan ini; membiarkan
 * klien menentukannya akan menghapus kontrol itu.
 *
 * Backend menjawab `409` bila proteksi sudah diputuskan petugas lain. Itu BUKAN kesalahan
 * pengguna — ia hanya kalah cepat pada antrean bersama, dan layar menyampaikannya begitu.
 */
export function useDecideProtection() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ number, decision }: { number: string; decision: Decision }) =>
      callAPI<ProtectionDetail>(
        `${PATH}/${encodeURIComponent(number)}/akseptasi`,
        { metode: 'PUT', body: { keputusan: decision }, token, portal },
      ),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.all })
    },
  })
}
