import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { APIError, callAPI } from '@/api/client'
import { useSession } from '@/app/session'

import {
  RegistrationErrorCode,
  type Violation,
  type RegisterRequest,
  type FlowResponse,
  type InboxResponse,
  type ClaimResponse,
  type Task,
} from './types'

// Seluruh panggilan API modul ini lewat hook di berkas ini. Tidak ada `fetch` di dalam
// komponen — aturan 9 pada README repository.

const inboxKey = ['registrasi', 'inbox'] as const
const flowKey = ['registrasi', 'alur'] as const

function claimKey(claimID: string) {
  return ['registrasi', 'klaim', claimID] as const
}

/**
 * Definisi alur Register.
 *
 * Ia tidak memuat data klaim mana pun dan nyaris tidak pernah berubah dalam satu sesi
 * kerja; memuatnya ulang setiap kali komponen dipasang hanya membebani server.
 */
export function useFlow() {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: [...flowKey, token],
    queryFn: () => callAPI<FlowResponse>('/api/registrasi/alur', { token }),
    enabled: token !== null,
    staleTime: 30 * 60 * 1000,
  })
}

/** Daftar pekerjaan yang menunggu pengguna (`D-79`). */
export function useInbox() {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: [...inboxKey, token],
    queryFn: () => callAPI<InboxResponse>('/api/registrasi/inbox', { token }),
    enabled: token !== null,
  })
}

/** Satu klaim beserta tugas terbukanya dan jalur tahap yang akan dilaluinya. */
export function useClaim(claimID: string | undefined) {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: [...claimKey(claimID ?? ''), token],
    queryFn: () => callAPI<ClaimResponse>(`/api/registrasi/klaim/${claimID}`, { token }),
    enabled: token !== null && Boolean(claimID),
  })
}

/** Membuka klaim baru dari sebuah polis. */
export function useStartClaim() {
  const token = useSession((state) => state.token)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: { nomor_polis: string; portal: string }) =>
      callAPI<ClaimResponse>('/api/registrasi/klaim', { metode: 'POST', body: content, token }),
    onSuccess: () => {
      void apiClient.invalidateQueries({ queryKey: inboxKey })
    },
  })
}

/** Menyimpan tahap Input Register, atau mengembalikannya dengan tombol Back. */
export function useSaveRegister() {
  const token = useSession((state) => state.token)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: RegisterRequest) =>
      callAPI<ClaimResponse>('/api/registrasi/register', { metode: 'POST', body: content, token }),
    onSuccess: (result) => {
      void apiClient.invalidateQueries({ queryKey: inboxKey })
      void apiClient.invalidateQueries({ queryKey: claimKey(result.klaim.id) })
    },
  })
}

/** Mengambil tugas dari antrean bersama. */
export function useClaimTask() {
  const token = useSession((state) => state.token)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (taskID: string) =>
      callAPI<Task>(`/api/registrasi/tugas/${taskID}/ambil`, { metode: 'POST', token }),
    onSuccess: (tugas) => {
      void apiClient.invalidateQueries({ queryKey: inboxKey })
      void apiClient.invalidateQueries({ queryKey: claimKey(tugas.klaim_id) })
    },
  })
}

/** Menutup tahap yang aturan isiannya milik modul lain. */
export function useCompleteStage() {
  const token = useSession((state) => state.token)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: { taskID: string; action: string; kembali?: boolean }) =>
      callAPI<ClaimResponse>(`/api/registrasi/tugas/${content.taskID}/selesai`, {
        metode: 'POST',
        body: { action: content.action, kembali: content.kembali ?? false },
        token,
      }),
    onSuccess: (result) => {
      void apiClient.invalidateQueries({ queryKey: inboxKey })
      void apiClient.invalidateQueries({ queryKey: claimKey(result.klaim.id) })
    },
  })
}

/**
 * violationsFrom membaca rincian validasi dari sebuah galat.
 *
 * Ia mengembalikan daftar kosong untuk galat jenis lain, sehingga layar tidak perlu
 * memeriksa kode galat sebelum memanggilnya.
 */
export function violationsFrom(failure: unknown): Violation[] {
  if (!(failure instanceof APIError)) return []
  if (failure.kode !== RegistrationErrorCode.validationFailed) return []
  if (!Array.isArray(failure.detail)) return []

  return failure.detail.filter(
    (p): p is Violation =>
      typeof p === 'object' && p !== null && 'kode' in p && 'field' in p && 'pesan' in p,
  )
}

/**
 * messagesByField mengelompokkan pelanggaran menurut kolom yang harus diperbaiki.
 *
 * Satu kolom dapat melanggar lebih dari satu aturan sekaligus — tanggal kejadian dapat
 * berada di luar periode polis DAN di masa depan. Keduanya ditampilkan, bukan hanya yang
 * pertama: memperbaiki satu lalu menemukan yang lain adalah pengalaman yang sistem lama
 * berikan, dan itu yang sedang diperbaiki.
 */
export function messagesByField(violations: Violation[]): Record<string, string> {
  const result: Record<string, string> = {}
  for (const p of violations) {
    if (!p.field) continue
    result[p.field] = result[p.field] ? `${result[p.field]} ${p.pesan}` : p.pesan
  }
  return result
}
