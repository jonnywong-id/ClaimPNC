import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { APIError, callAPI, unduhBerkas } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import {
  AreaLevel,
  RegistrationErrorCode,
  type AreaOptionsResponse,
  type CauseOfLossOptionsResponse,
  type AcceptanceDefaults,
  type Violation,
  type RegisterRequest,
  type FlowResponse,
  type InboxResponse,
  type ClaimResponse,
  type CommitteeNote,
  type DiagnosisResponse,
  type CurrenciesResponse,
  type EstimateRequest,
  type ItemOptionsResponse,
  type CoverageOptionsResponse,
  type SettlementHistoryResponse,
  type PremiumAgingResponse,
  type SurveysResponse,
  type DocumentLink,
  type DocumentsResponse,
  type InsuredResponse,
  type ProgressResponse,
  type FaceSheetRequest,
  type PLARequest,
  type PLAListResponse,
  type DLARequest,
  type DLAListResponse,
  type CashierRequest,
  type CashierPreview,
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
      apiClient.invalidateQueries({ queryKey: inboxKey })
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
      apiClient.invalidateQueries({ queryKey: inboxKey })
      apiClient.invalidateQueries({ queryKey: claimKey(result.klaim.id) })
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
      apiClient.invalidateQueries({ queryKey: claimKey(result.klaim.id) })
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
 * Pilihan Penyebab Kerugian menurut kode bisnis polis — pengganti autocomplete Pega
 * BrowseCouseOfLoss_Business. Kode bisnis kosong tidak memanggil server.
 */
export function useCauseOfLossOptions(businessCode: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['registrasi', 'penyebab-kerugian', businessCode, token],
    enabled: businessCode !== '',
    staleTime: 10 * 60 * 1000,
    queryFn: () =>
      callAPI<CauseOfLossOptionsResponse>(
        `/api/registrasi/penyebab-kerugian?bisnis=${encodeURIComponent(businessCode)}`,
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
      apiClient.invalidateQueries({ queryKey: inboxKey })
      apiClient.invalidateQueries({ queryKey: claimKey(result.klaim.id) })
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

/**
 * Pilihan dropdown "Tambah coverage" untuk satu objek klaim. Hanya membaca — coverage
 * tersimpan bersama klaim saat Save/Submit, tidak ke tabel lain.
 */
export function useCoverageOptions(claimID: string, objectID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['registrasi', 'pilihan-coverage', claimID, objectID, token],
    enabled: claimID !== '' && objectID !== '',
    staleTime: 10 * 60 * 1000,
    queryFn: () =>
      callAPI<CoverageOptionsResponse>(
        `/api/registrasi/klaim/${encodeURIComponent(claimID)}/pilihan-coverage?objek=${encodeURIComponent(objectID)}`,
        { token, portal },
      ),
  })
}

/**
 * Riwayat satu baris Adjustment: Status Penerimaan Komite dan Histori Transfer Kasir.
 * Baris dialamatkan lewat urutan objek, jaminan, dan adjustment (berbasis 1).
 */
/** Awalan kunci riwayat adjustment satu klaim — dipakai juga untuk menyegarkannya. */
export const settlementHistoryKey = (claimID: string) => ['registrasi', 'riwayat-adjustment', claimID] as const

export function useSettlementHistory(claimID: string, object: number, coverage: number, adjustment: number) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const query = new URLSearchParams({
    objek: String(object),
    coverage: String(coverage),
    adjustment: String(adjustment),
  }).toString()

  return useQuery({
    queryKey: [...settlementHistoryKey(claimID), object, coverage, adjustment, token],
    enabled: claimID !== '',
    queryFn: () =>
      callAPI<SettlementHistoryResponse>(
        `/api/registrasi/klaim/${encodeURIComponent(claimID)}/adjustment/riwayat?${query}`,
        { token, portal },
      ),
  })
}

/**
 * Aging Amount polis klaim (`.PaymentData.AgingAmount`), dibaca dari layanan premi. Layanan
 * yang tidak dapat dihubungi dijawab `tersedia: false`, bukan galat.
 */
export function usePremiumAging(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['registrasi', 'aging', claimID, portal, token],
    enabled: claimID !== '',
    staleTime: 5 * 60 * 1000,
    queryFn: () =>
      callAPI<PremiumAgingResponse>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/aging`, { token, portal }),
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

/** Cari Kode / Desc Diagnose (modal Transfer Claim ke Komite, PA). Kosong: tidak mencari. */
export function useDiagnosisSearch(term: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['registrasi', 'diagnosa', term, token, portal],
    enabled: term !== '',
    retry: false,
    queryFn: () =>
      callAPI<DiagnosisResponse>(`/api/registrasi/diagnosa?cari=${encodeURIComponent(term)}`, { token, portal }),
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
      apiClient.invalidateQueries({ queryKey: inboxKey })
      apiClient.invalidateQueries({ queryKey: claimKey(tugas.klaim_id) })
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
      apiClient.invalidateQueries({ queryKey: inboxKey })
      apiClient.invalidateQueries({ queryKey: claimKey(result.klaim.id) })
    },
  })
}

/**
 * Tombol Kirim pada modal "Kirim ke Inputor" (`AnalystRemarks_sect`): menutup tugas berjalan dan
 * melompatkan klaim ke Input Register milik Inputor, dengan catatan analis.
 */
export function useSendToInputor(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: { taskID: string; catatan: string }) =>
      callAPI<ClaimResponse>(`/api/registrasi/tugas/${content.taskID}/kirim-inputor`, {
        metode: 'POST',
        body: { catatan: content.catatan },
        token,
        portal,
      }),
    onSuccess: () => {
      apiClient.invalidateQueries({ queryKey: inboxKey })
      apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
      // Catatan baru tampil di tab Progress dan sebagai Catatan dari Analyst.
      apiClient.invalidateQueries({ queryKey: ['registrasi', 'progres', claimID] })
    },
  })
}

/**
 * Pilihan "Perihal" modal Kirim ke RCL/PUCL — `POOLDATA.M_PERIHAL_RCLPUCL`, disaring
 * menurut jalur yang sedang dipilih. Jalur 0 (belum dipilih) tidak memanggil apa pun.
 */
export function usePUCLSubjects(jalur: number) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['registrasi', 'rclpucl', 'perihal', jalur, token],
    enabled: jalur > 0,
    staleTime: 10 * 60 * 1000,
    queryFn: () =>
      callAPI<{ pilihan: { id: number; nama: string }[] }>(
        `/api/registrasi/rclpucl/perihal?jalur=${jalur}`,
        { token, portal },
      ),
  })
}

/**
 * Grid alasan penolakan — `POOLDATA.M_REASON_REJECT_REPRO`, 2.116 baris.
 *
 * Pencariannya dikirim ke server, bukan disaring di layar: menarik seluruh tabel ke
 * browser lalu menyaringnya di sana adalah persis cacat yang `NFR-12` larang.
 */
export function usePUCLReasons(cari: string, aktif: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['registrasi', 'rclpucl', 'alasan', cari, token],
    // Grid ini tidak tampil pada jalur Notification maupun di luar lini PA; tanpa
    // sakelar ini layar tetap menarik 200 baris yang tidak pernah digambar.
    enabled: aktif,
    staleTime: 5 * 60 * 1000,
    queryFn: () =>
      callAPI<{ pilihan: { id: string; nama: string; deskripsi: string }[] }>(
        `/api/registrasi/rclpucl/alasan?cari=${encodeURIComponent(cari)}`,
        { token, portal },
      ),
  })
}

/**
 * Pilihan dropdown "Nama Dokter" — `POOLDATA.T_ACCESS_GROUP_PNC`, disaring dengan ketiga
 * grup akses yang sama dengan yang dipakai Inbox RCL mencari identitas lama pemanggilnya.
 *
 * Yang dipilih di sini menentukan SIAPA yang melihat klaimnya, jadi daftarnya tidak boleh
 * lebih luas maupun lebih sempit daripada himpunan nilai yang dapat dicocokkan penyaring
 * itu. Alasan lengkapnya ada di `rclpucl.sql`.
 */
export function useRCLDoctors(aktif: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['registrasi', 'rclpucl', 'dokter', token],
    // Isiannya hanya tampil pada jalur RCL atau Notification lini PA; tanpa sakelar ini
    // layar menarik daftarnya pada setiap modal yang tidak pernah menggambarnya.
    enabled: aktif,
    staleTime: 10 * 60 * 1000,
    queryFn: () =>
      callAPI<{ pilihan: { id: string }[] }>('/api/registrasi/rclpucl/dokter', { token, portal }),
  })
}

/** Isi modal "Kirim ke RCL/PUCL" — `Section/SectionPUCL-sect.xml`. */
export type SendToRCLPUCLContent = {
  taskID: string
  jalur: number
  catatan: string
  perihal: string
  keterangan_pembuka: string
  keterangan_isi: string
  keterangan_penutup: string
  nama_dokter: string
}

/**
 * Tombol Kirim pada modal "Kirim ke RCL/PUCL": menyimpan suratnya, menutup tugas
 * berjalan, dan melompatkan klaim ke RCL/PUCL (jalur 2) atau RCLDokter (jalur 1).
 */
export function useSendToRCLPUCL(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: ({ taskID, ...body }: SendToRCLPUCLContent) =>
      callAPI<ClaimResponse>(`/api/registrasi/tugas/${taskID}/kirim-rclpucl`, {
        metode: 'POST',
        body,
        token,
        portal,
      }),
    onSuccess: () => {
      apiClient.invalidateQueries({ queryKey: inboxKey })
      apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
      apiClient.invalidateQueries({ queryKey: ['registrasi', 'progres', claimID] })
    },
  })
}

/** Isian dialog "Prevent Close Claim" yang dikirim ke server. */
export interface CloseClaimBody {
  catatan_tutup: string
  usulan: string
  effort_tutup: string
  kendala_tutup: string
  tutup_sementara: boolean
}

/** Tombol Ya pada dialog "Prevent Close Claim" — `CloseClaim`. */
export function useCloseClaim(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: { taskID: string; body: CloseClaimBody }) =>
      callAPI<ClaimResponse>(`/api/registrasi/tugas/${content.taskID}/tutup-klaim`, {
        metode: 'POST',
        body: content.body,
        token,
        portal,
      }),
    onSuccess: () => {
      apiClient.invalidateQueries({ queryKey: inboxKey })
      apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
    },
  })
}

/** Tombol Kirim Analyst pada modal "Transfer Claim ke Komite" — `setTicketToAnalyst`. */
export function useTransferToAnalyst(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: { taskID: string; objekID: string; coverageID: string }) =>
      callAPI<ClaimResponse>(`/api/registrasi/tugas/${content.taskID}/transfer-analis`, {
        metode: 'POST',
        body: { objek_id: content.objekID, coverage_id: content.coverageID },
        token,
        portal,
      }),
    onSuccess: () => {
      apiClient.invalidateQueries({ queryKey: inboxKey })
      apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
      apiClient.invalidateQueries({ queryKey: ['registrasi', 'progres', claimID] })
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
      typeof p === 'object' && p != null && 'kode' in p && 'field' in p && 'pesan' in p,
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
function useClaimRecord<T>(claimID: string, path: 'survey' | 'dokumen' | 'progres' | 'tertanggung', enabled: boolean) {
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
      apiClient.invalidateQueries({ queryKey: ['registrasi', 'dokumen', claimID] })
    },
  })
}

/**
 * Tombol Lihat dokumen: alamat baca satu lampiran dari metadata penyimpanan. Diminta saat
 * tombol ditekan, bukan dimuat bersama daftar — alamatnya bermasa berlaku.
 */
export function useDocumentLink(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (attachmentID: string) =>
      callAPI<DocumentLink>(
        `/api/registrasi/klaim/${encodeURIComponent(claimID)}/dokumen/${encodeURIComponent(attachmentID)}/tautan`,
        { token, portal },
      ),
  })
}

/**
 * Tombol Delete pada daftar berkas: hapus PERMANEN (keputusan-implementasi §171). Jawabannya
 * checklist yang sudah diperbarui, sehingga "Lihat dokumen (n)" langsung berkurang.
 */
export function useDeleteDocument(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (attachmentID: string) =>
      callAPI<DocumentsResponse>(
        `/api/registrasi/klaim/${encodeURIComponent(claimID)}/dokumen/${encodeURIComponent(attachmentID)}/hapus`,
        { metode: 'POST', token, portal },
      ),
    onSuccess: () => {
      apiClient.invalidateQueries({ queryKey: ['registrasi', 'dokumen', claimID] })
    },
  })
}

/** Data tertanggung dari CIF polis — tab Register (DATA TERTANGGUNG KLAIM, Alamat). */
export function useInsuredProfile(claimID: string, enabled = true) {
  return useClaimRecord<InsuredResponse>(claimID, 'tertanggung', enabled)
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
      apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
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

/**
 * Membuka layar PrintDLA: menerbitkan DLA adjustment itu (bila belum) lalu mengembalikan
 * daftarnya. Mutasi, bukan kueri, karena membukanya dapat menerbitkan nomor.
 */
export function useDLAList(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (content: DLARequest) =>
      callAPI<DLAListResponse>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/dla/daftar`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
  })
}

/** PRINT (satu nomor) dan Print All DLA — PDF, atau ZIP bila lebih dari satu. */
export function usePrintDLA(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (content: DLARequest) =>
      unduhBerkas(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/dla`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
  })
}

/**
 * Tombol PRINT di samping Nomor Akseptasi — PDF Draft Persetujuan (`PrintPDFAcceptanceNote`).
 * Badannya sama dengan Transfer Kasir: tugas beserta objek, jaminan, dan adjustment.
 */
export function usePrintAcceptanceNote(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (content: CashierRequest) =>
      unduhBerkas(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/akseptasi/draft`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
  })
}

/** Isi dialog konfirmasi Transfer Kasir (penerima, rekening, nilai nett, galat pertama). */
export function useCashierPreview(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (content: CashierRequest) =>
      callAPI<CashierPreview>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/kasir/pratinjau`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
  })
}

/** Submit Transfer Kasir — mengirim pembayaran ke sistem Kasir (TransferToKasir_act). */
export function useTransferCashier(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: CashierRequest) =>
      callAPI<ClaimResponse>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/kasir`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
    onSuccess: () => {
      apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
      // Histori Transfer Kasir membaca TRF_KASIR_LOG lewat kuncinya sendiri; tanpa ini grid
      // riwayat tetap menampilkan keadaan sebelum transfer sampai halaman dimuat ulang.
      apiClient.invalidateQueries({ queryKey: settlementHistoryKey(claimID) })
    },
  })
}

/** Alamat satu baris adjustment untuk dialog Print LOD. Indeks berbasis 1. */
export type LODRequest = {
  tugas_id: string
  objek: number
  jaminan: number
  adjustment: number
  tipe_pdf?: string
}

export type LODType = { id: string; nama: string; tersedia: boolean }

/** Isi awal dialog Print LOD — pilihan Tipe PDF dan isian section PrintLODdanEmail. */
export type LODDialog = { tipe: LODType[]; email_lod: string; nama_tertanggung: string }

/** Isi awal dialog Print LOD (`SetTypePDFAdjustment`, `SetDataEmailTertanggung`). */
export function useLODTypes(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (content: LODRequest) =>
      callAPI<LODDialog>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/lod/tipe`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
  })
}

/**
 * Pilihan dropdown Tipe LOD kolom Adjustment — daftar yang sama dengan dialog Print LOD
 * (`SetTypePDFAdjustment`), dimuat sekali per tugas karena tidak berbeda antarbaris.
 */
export function useLODTypeOptions(claimID: string, taskID: string, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['registrasi', 'lod-tipe', claimID, taskID, token],
    enabled: enabled && taskID !== '',
    staleTime: 5 * 60_000,
    queryFn: () =>
      callAPI<LODDialog>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/lod/tipe`, {
        metode: 'POST',
        body: { tugas_id: taskID, objek: 1, jaminan: 1, adjustment: 1 },
        token,
        portal,
      }),
  })
}

/** Isian awal form AcceptationLOD satu baris adjustment (POST …/akseptasi/awal). */
export function useAcceptanceDefaults(
  claimID: string,
  address: { tugas_id: string; objek: number; jaminan: number; adjustment: number },
) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['registrasi', 'akseptasi-awal', claimID, address, token],
    staleTime: 0,
    queryFn: () =>
      callAPI<AcceptanceDefaults>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/akseptasi/awal`, {
        metode: 'POST',
        body: address,
        token,
        portal,
      }),
  })
}

/** Menyimpan pilihan dropdown Tipe LOD (PDFTYPE) satu baris adjustment, lalu memuat ulang klaim. */
export function useSetLODType(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: LODRequest) =>
      callAPI<unknown>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/lod/pilih`, {
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
 * Print LOD — mengunduh PDF Letter of Discharge satu baris adjustment. Mencetak menulis
 * PDFTYPE dan PRINTLOD_DATE baris itu, sehingga klaim dimuat ulang: kolom Adjustment grid dan
 * form AcceptationLOD menampilkan Tipe PDF dan Tanggal Cetak LOD yang baru.
 */
export function usePrintLOD(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: LODRequest) =>
      unduhBerkas(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/lod`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
    onSuccess: () => {
      apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
    },
  })
}

/** Isian form Persetujuan / Akseptasi (AcceptationLOD_Sect). Tanggal YYYY-MM-DD. */
export type AcceptanceInput = {
  tugas_id: string
  objek: number
  jaminan: number
  adjustment: number
  /** `.TipeAkseptasi` — terisi dari AcceptationLOD_PreAct, dapat diubah. */
  tipe_akseptasi?: string
  persetujuan_tertanggung: string
  tanggal_terima_lod: string
  tanggal_boleh_bayar: string
  nilai_lod_sen?: number
  penerima: string
  nama_komite_akseptasi: string
  catatan_penerima: string
  berita_acara: string
}

/** Satu berkas "Unggah Dokumen Persetujuan LOD" beserta jenis dokumennya (DOC_TYPE_DT_ID). */
export type AcceptanceFileInput = { berkas: File; jenisDokumen: string }

/** Simpan form Persetujuan / Akseptasi — menerbitkan nomor akseptasi (SetAdjustmentAcceptation). */
export function useAcceptSettlement(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: ({ isian, berkas }: { isian: AcceptanceInput; berkas: AcceptanceFileInput[] }) => {
      const form = new FormData()
      // Nama bagian harus sama dengan yang dibaca handler (AcceptSettlement).
      form.append('isian', JSON.stringify(isian))
      for (const f of berkas) {
        form.append('berkas', f.berkas, f.berkas.name)
        form.append('jenis_dokumen', f.jenisDokumen)
      }
      return callAPI<ClaimResponse>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/akseptasi`, {
        metode: 'POST',
        body: form,
        token,
        portal,
      })
    },
    onSuccess: () => {
      apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
      apiClient.invalidateQueries({ queryKey: ['registrasi', 'dokumen', claimID] })
    },
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
      apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
    },
  })
}

/** Tombol Tambah grid Adjustment — ValidationAdjustment; untuk PA menambahkan estimasi NewEstimationPA. */
export function usePrepareSettlement(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: { tugas_id: string; objek: number; jaminan: number }) =>
      callAPI<ClaimResponse>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/adjustment/tambah`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
    onSuccess: () => {
      apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
    },
  })
}

/** Menyimpan ulang baris Adjustment yang sudah ada setelah isiannya berubah (SetNilaiResikoSendiri). */
export function useUpdateSettlement(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: SettlementRequest) =>
      callAPI<ClaimResponse>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/adjustment/ubah`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
    onSuccess: () => {
      apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
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
      apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
      apiClient.invalidateQueries({ queryKey: committeeKey })
    },
  })
}

/** Simpan isian modal "Transfer Claim ke Komite" satu jaminan. Objek dan jaminan berbasis 1. */
export function useSaveCommitteeNote(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (content: { tugas_id: string; objek: number; jaminan: number } & CommitteeNote) =>
      callAPI<ClaimResponse>(`/api/registrasi/klaim/${encodeURIComponent(claimID)}/jaminan/isian-komite`, {
        metode: 'POST',
        body: content,
        token,
        portal,
      }),
    onSuccess: () => {
      apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
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
      apiClient.invalidateQueries({ queryKey: committeeKey })
      apiClient.invalidateQueries({ queryKey: ['registrasi', 'klaim'] })
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
      apiClient.invalidateQueries({ queryKey: claimKey(claimID) })
    },
  })
}
