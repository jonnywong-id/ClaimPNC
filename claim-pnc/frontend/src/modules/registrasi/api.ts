import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { APIError, callAPI, unduhBerkas } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import {
  AreaLevel,
  RegistrationErrorCode,
  type AreaOptionsResponse,
  type Violation,
  type RegisterRequest,
  type FlowResponse,
  type InboxResponse,
  type ClaimResponse,
  type CurrenciesResponse,
  type EstimateRequest,
  type ItemOptionsResponse,
  type SurveysResponse,
  type DocumentsResponse,
  type ProgressResponse,
  type FaceSheetRequest,
  type PLARequest,
  type PLAListResponse,
  type SettlementRequest,
  type SettlementPreviewResponse,
  type BankAccount,
  type ReceiverRequest,
  type Committee,
  type CommitteeListResponse,
  type CommitteeTransferRequest,
  type CommitteeTransferResponse,
  type Task,
} from './types'

// Seluruh panggilan API modul ini lewat hook di berkas ini. Tidak ada `fetch` di dalam
// komponen — aturan 9 pada README repository.
//
// SETIAP panggilan membawa portal aktif. Modul ini ditulis sebelum portal ada, dan
// ketiadaannya membuat seluruh layarnya menjawab "Portal entitas belum dipilih" begitu
// ia dipasang di belakang middleware ActivePortal (2026-09-24). Portal menentukan
// basis data mana yang dibaca (`D-75`); panggilan tanpa portal DITOLAK, bukan jatuh ke
// portal bawaan — itulah yang mencegah `R-20`.

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
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: [...flowKey, token],
    queryFn: () => callAPI<FlowResponse>('/api/registrasi/alur', { token, portal }),
    enabled: token !== null,
    staleTime: 30 * 60 * 1000,
  })
}

/** Daftar pekerjaan yang menunggu pengguna (`D-79`). */
export function useInbox() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: [...inboxKey, token],
    queryFn: () => callAPI<InboxResponse>('/api/registrasi/inbox', { token, portal }),
    enabled: token !== null,
  })
}

/** Satu klaim beserta tugas terbukanya dan jalur tahap yang akan dilaluinya. */
export function useClaim(claimID: string | undefined) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: [...claimKey(claimID ?? ''), token],
    queryFn: () => callAPI<ClaimResponse>(`/api/registrasi/klaim/${claimID}`, { token, portal }),
    enabled: token !== null && Boolean(claimID),
  })
}

/** Membuka klaim baru dari sebuah polis. */
export function useStartClaim() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    // Portal TIDAK ikut di badan permintaan: server mengambilnya dari portal aktif,
    // supaya pemanggil tidak dapat menuliskan klaim atas nama entitas lain (`R-20`).
    mutationFn: (content: { nomor_polis: string }) =>
      callAPI<ClaimResponse>('/api/registrasi/klaim', { metode: 'POST', body: content, token, portal }),
    onSuccess: () => {
      void apiClient.invalidateQueries({ queryKey: inboxKey })
    },
  })
}

/** Menyimpan tahap Input Register, atau mengembalikannya dengan tombol Back. */
export function useSaveRegister() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: RegisterRequest) =>
      callAPI<ClaimResponse>('/api/registrasi/register', { metode: 'POST', body: content, token, portal }),
    onSuccess: (result) => {
      void apiClient.invalidateQueries({ queryKey: inboxKey })
      void apiClient.invalidateQueries({ queryKey: claimKey(result.klaim.id) })
    },
  })
}

/**
 * Tombol Save: menyimpan isian Input Register TANPA menutup tahapnya dan tanpa
 * menjalankan gerbang validasi — seperti Save pada layar tahap Pega.
 */
export function useSaveDraft() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: RegisterRequest) =>
      callAPI<ClaimResponse>('/api/registrasi/register/simpan', { metode: 'POST', body: content, token, portal }),
    onSuccess: (result) => {
      void apiClient.invalidateQueries({ queryKey: claimKey(result.klaim.id) })
    },
  })
}

/**
 * Satu tingkat daftar pilihan wilayah. Tingkat di bawah negara baru diminta setelah
 * induknya dipilih; tanpa induk, daftarnya memang kosong.
 */
export function useAreaOptions(level: AreaLevel, parent: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const needsParent = level !== AreaLevel.Country

  return useQuery({
    queryKey: ['registrasi', 'wilayah', level, parent, token],
    enabled: !needsParent || parent !== '',
    staleTime: 10 * 60 * 1000,
    queryFn: () =>
      callAPI<AreaOptionsResponse>(
        `/api/registrasi/wilayah/${level}?induk=${encodeURIComponent(parent)}`,
        { token, portal },
      ),
  })
}

/**
 * Tahap Input Estimasi. `simpan` menyimpan tanpa menutup tahap (Save); tanpanya tahap
 * ditutup — Next, atau Back bila `kembali`.
 */
export function useSaveEstimate(simpan: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: EstimateRequest) =>
      callAPI<ClaimResponse>(simpan ? '/api/registrasi/estimasi/simpan' : '/api/registrasi/estimasi', {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
    onSuccess: (result) => {
      void apiClient.invalidateQueries({ queryKey: inboxKey })
      void apiClient.invalidateQueries({ queryKey: claimKey(result.klaim.id) })
    },
  })
}

/** Pilihan Objek untuk item estimasi satu objek klaim. */
export function useItemOptions(claimID: string, objectID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['registrasi', 'pilihan-item', claimID, objectID, token],
    staleTime: 10 * 60 * 1000,
    queryFn: () =>
      callAPI<ItemOptionsResponse>(
        `/api/registrasi/klaim/${encodeURIComponent(claimID)}/pilihan-item?objek=${encodeURIComponent(objectID)}`,
        { token, portal },
      ),
  })
}

/** Pilihan Mata Uang dari master POOLDATA.CURRENCY. */
export function useCurrencies() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['registrasi', 'mata-uang', token],
    staleTime: 10 * 60 * 1000,
    queryFn: () => callAPI<CurrenciesResponse>('/api/registrasi/mata-uang', { token, portal }),
  })
}

/** Mengambil tugas dari antrean bersama. */
export function useClaimTask() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (taskID: string) =>
      callAPI<Task>(`/api/registrasi/tugas/${taskID}/ambil`, { metode: 'POST', token, portal }),
    onSuccess: (tugas) => {
      void apiClient.invalidateQueries({ queryKey: inboxKey })
      void apiClient.invalidateQueries({ queryKey: claimKey(tugas.klaim_id) })
    },
  })
}

/** Menutup tahap yang aturan isiannya milik modul lain. */
export function useCompleteStage() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: { taskID: string; action: string; kembali?: boolean }) =>
      callAPI<ClaimResponse>(`/api/registrasi/tugas/${content.taskID}/selesai`, {
        metode: 'POST',
        body: { action: content.action, kembali: content.kembali ?? false },
        token,
        portal,
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

/**
 * useClaimRecord membaca satu tab pendamping Input Estimasi: survey, dokumen, atau
 * progres. Ketiganya hanya membaca.
 */
function useClaimRecord<T>(claimID: string, path: 'survey' | 'dokumen' | 'progres', enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['registrasi', path, claimID, token, portal],
    enabled,
    queryFn: () =>
      callAPI<T>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/${path}`, { token, portal }),
  })
}

/** Tab Survey — T_SURVEYORLIST. */
export function useSurveys(claimID: string, enabled = true) {
  return useClaimRecord<SurveysResponse>(claimID, 'survey', enabled)
}

/** Tab Unggah Dokumen — checklist jenis dokumen dan berkas yang sudah diunggah. */
export function useDocuments(claimID: string, enabled = true) {
  return useClaimRecord<DocumentsResponse>(claimID, 'dokumen', enabled)
}

/** Bahan satu unggahan dari baris checklist. */
export interface UploadDocumentInput {
  /** DOC_TYPE_DT_ID baris checklist. */
  jenisDokumen: string
  berkas: File
  catatan?: string
}

/**
 * Tombol Unggah Dokumen pada satu baris checklist. Dikirim multipart; jawabannya checklist
 * yang sudah diperbarui, dan tab Unggah Dokumen dimuat ulang dari server.
 */
export function useUploadDocument(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (input: UploadDocumentInput) => {
      const form = new FormData()
      // Nama bagian `berkas`, `jenis_dokumen`, dan `catatan` harus sama dengan yang dibaca
      // handler (UploadDocument).
      form.append('berkas', input.berkas, input.berkas.name)
      form.append('jenis_dokumen', input.jenisDokumen)
      if (input.catatan?.trim()) form.append('catatan', input.catatan.trim())
      return callAPI<DocumentsResponse>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/dokumen`, {
        metode: 'POST',
        body: form,
        token,
        portal,
      })
    },
    onSuccess: () => {
      void apiClient.invalidateQueries({ queryKey: ['registrasi', 'dokumen', claimID] })
    },
  })
}

/** Tab Progress Claim & Komunikasi. */
export function useProgressRecords(claimID: string, enabled = true) {
  return useClaimRecord<ProgressResponse>(claimID, 'progres', enabled)
}

/**
 * Tombol Download Claim Face Sheet: membentuk PDF satu jaminan, mencatat revisinya, dan
 * mengunci estimasinya. Jawabannya berkas, bukan JSON.
 */
export function useFaceSheet(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: FaceSheetRequest) =>
      unduhBerkas(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/cfs`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
    onSuccess: () => {
      void apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
    },
  })
}

/**
 * Membuka layar PrintPLA_dtl: menerbitkan PLA koasuransi revisi CFS terakhir (bila belum) lalu
 * mengembalikan daftarnya. Mutasi, bukan kueri, karena membukanya dapat menerbitkan nomor.
 */
export function usePLAList(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (content: PLARequest) =>
      callAPI<PLAListResponse>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/pla/daftar`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
  })
}

/** Menyimpan isian REMARKS PLA. */
export function useSavePLANotes(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (content: PLARequest) =>
      callAPI<PLAListResponse>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/pla/catatan`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
  })
}

/** Print PLA (satu nomor) dan Print All PLA — PDF, atau ZIP bila lebih dari satu. */
export function usePrintPLA(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (content: PLARequest) =>
      unduhBerkas(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/pla`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
  })
}

/** Tombol Tambah pada grid Adjustment (tab Adjustment & Akseptasi, layar InputSurveyor). */
export function useAddSettlement(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: SettlementRequest) =>
      callAPI<ClaimResponse>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/adjustment`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
    onSuccess: () => {
      void apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
    },
  })
}

/** Menghitung ulang nilai tampilan baris Adjustment tanpa menyimpan (SetNilaiResikoSendiri). */
export function usePreviewSettlement(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (content: SettlementRequest) =>
      callAPI<SettlementPreviewResponse>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/adjustment/hitung`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
  })
}

/**
 * Isian No Rekening InputReceiver: rekening Master Rekening bernomor itu — pengganti
 * `GetDataBankMaster` Pega, yang dipanggil setiap nomor berubah. Nomor kosong tidak dibaca.
 */
export function useBankAccount(number: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const trimmed = number.trim()

  return useQuery({
    queryKey: ['registrasi', 'rekening', trimmed, token, portal],
    enabled: trimmed !== '',
    retry: false,
    queryFn: () =>
      callAPI<BankAccount>(`/api/registrasi/rekening/${encodeURIComponent(trimmed)}`, { token, portal }),
  })
}

const committeeKey = ['registrasi', 'komite'] as const

/** Tombol Transfer Komite pada satu baris Adjustment. */
export function useTransferCommittee(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: CommitteeTransferRequest) =>
      callAPI<CommitteeTransferResponse>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/adjustment/komite`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
    onSuccess: () => {
      void apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
      void apiClient.invalidateQueries({ queryKey: committeeKey })
    },
  })
}

/** Status satu kasus komite per jenjang. */
export function useCommittee(committeeID: string | undefined) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: [...committeeKey, committeeID ?? '', token, portal],
    enabled: !!committeeID,
    queryFn: () => callAPI<Committee>(`/api/registrasi/komite/${encodeURIComponent(committeeID ?? '')}`, { token, portal }),
  })
}

/** Putusan komite klaim PNCN yang menunggu pengguna ini. */
export function usePendingCommittees() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: [...committeeKey, 'menunggu', token, portal],
    queryFn: () => callAPI<CommitteeListResponse>('/api/registrasi/komite', { token, portal }),
  })
}

/** Tombol Setuju/Tolak anggota komite. */
export function useDecideCommittee() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: { komiteID: string; keputusan: string; catatan: string }) =>
      callAPI<Committee>(`/api/registrasi/komite/${encodeURIComponent(content.komiteID)}/putusan`, {
        metode: 'POST',
        body: { keputusan: content.keputusan, catatan: content.catatan },
        token,
        portal,
      }),
    onSuccess: () => {
      void apiClient.invalidateQueries({ queryKey: committeeKey })
      void apiClient.invalidateQueries({ queryKey: ['registrasi', 'klaim'] })
    },
  })
}

/** Tombol Simpan InputReceiver — penerima baru (Tambah) atau yang sedang dibuka. */
export function useSaveReceiver(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: ReceiverRequest) =>
      callAPI<ClaimResponse>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/penerima`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
    onSuccess: () => {
      void apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
    },
  })
}
