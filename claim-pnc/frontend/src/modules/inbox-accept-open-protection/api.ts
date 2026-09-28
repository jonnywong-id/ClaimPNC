import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  Decision,
  ProtectionDetail,
  ProtectionFilter,
  ProtectionListResponse,
  QueuesResponse,
} from './types'

const PATH = '/api/inbox-accept-open-protection'

/**
 * Banyaknya baris per halaman.
 *
 * Mengikuti `<pyPageSize>20</pyPageSize>` pada ketiga grid
 * `Section/InputProtection_Section-Section.xml`. Nilainya WAJIB sama dengan `DefaultLimit`
 * di `backend/internal/inboxacceptopenprotection` — layar menghitung nomor halaman dari
 * angka ini, sedangkan yang benar-benar memotong hasil adalah server. Bila keduanya
 * berbeda, penomoran halaman meleset tanpa satu pun galat.
 */
export const PAGE_SIZE = 20

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

  // Kewenangan TIDAK berkunci portal: satu identitas berlaku di keempat portal (`D-78`),
  // dan tabelnya pun tinggal di basis data utama. Memasukkan portal ke kunci hanya akan
  // menembak server berkali-kali untuk jawaban yang sama.
  queues: (token: string | null) =>
    ['inbox-accept-open-protection', 'antrean', token] as const,
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
/**
 * Hook antrean yang boleh dibuka pemanggil.
 *
 * # Kenapa server yang menentukan, bukan layar
 *
 * Layar lama tidak punya pemilihan sama sekali: grid yang bukan hak seseorang **tidak
 * pernah dirender** baginya (`Section/InputProtection_Section-Section.xml:1592` dan `:6036`
 * menyaring lewat `AccessGroup.pyAccessGroup`).
 *
 * Menggambar kedua tab lalu membiarkan salah satunya dijawab 403 akan menampilkan pilihan
 * yang pasti gagal — dan pengguna tetap akan mencobanya. Menebaknya di peramban lebih buruk
 * lagi: kewenangan yang ditentukan klien bukan kewenangan.
 *
 * Tidak menuntut portal aktif — kewenangan sama di keempat portal (`D-78`).
 */
export function useAllowedQueues() {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: keys.queues(token),
    queryFn: () => callAPI<QueuesResponse>(`${PATH}/antrean`, { token }),
    enabled: token !== null,

    // Tidak dicoba ulang saat 403: penolakan kewenangan BUKAN gangguan sementara, dan
    // mengulanginya hanya menunda pesan yang sudah pasti.
    retry: false,
  })
}

export function useAcceptQueue(filter: ProtectionFilter, options?: { enabled?: boolean }) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(portal, token, filter),
    queryFn: () => callAPI<ProtectionListResponse>(buildPath(filter), { token, portal }),
    // Ditambah syarat pemanggil: selama antrean yang boleh dibuka belum diketahui, daftar
    // TIDAK ditembak. Menembaknya lebih dulu hanya menghasilkan 403 yang pasti.
    enabled: token !== null && portal !== null && (options?.enabled ?? true),

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
 * `RESOLVED_BY` adalah satu-satunya kontrol pengimbang atas persetujuan ini; membiarkan
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
