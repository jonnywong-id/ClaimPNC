import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  Account,
  BankListResponse,
  AccountListResponse,
  AccountStatus,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTES = '/api/master-rekening'

/** Saringan daftar rekening; seluruh field boleh kosong. */
export type AccountFilter = {
  status?: AccountStatus | ''
  nomorRekening?: string
  namaPemilik?: string
  namaBank?: string
  /** Membatasi ke antrean komite yang sedang masuk. Dipakai tab Komite Approval. */
  komiteSaya?: boolean
  batas?: number
  lewati?: number
}

/** Isian formulir rekening. */
export type AccountFields = {
  nomorRekening: string
  namaPemilik: string
  namaBank: string
  cabangBank: string
  alamatBank: string
  kodeBank: string
  tipeRekening: string
  email: string
  telepon: string
  nik: string
  idDokumen: string
  catatan: string
  aktif: boolean
  kodeBankLama?: string
  nomorRekeningLama?: string
  namaPemilikLama?: string
}

function bodyOf(values: AccountFields) {
  return {
    nomor_rekening: values.nomorRekening,
    nama_pemilik: values.namaPemilik,
    nama_bank: values.namaBank,
    cabang_bank: values.cabangBank,
    alamat_bank: values.alamatBank,
    kode_bank: values.kodeBank,
    tipe_rekening: values.tipeRekening,
    email: values.email,
    telepon: values.telepon,
    nik: values.nik,
    id_dokumen: values.idDokumen,
    catatan: values.catatan,
    aktif: values.aktif,
    kode_bank_lama: values.kodeBankLama ?? '',
    nomor_rekening_lama: values.nomorRekeningLama ?? '',
    nama_pemilik_lama: values.namaPemilikLama ?? '',
  }
}

function queryFrom(filter: AccountFilter): string {
  const q = new URLSearchParams()
  if (filter.status) q.set('status', filter.status)
  if (filter.nomorRekening?.trim()) q.set('nomor_rekening', filter.nomorRekening.trim())
  if (filter.namaPemilik?.trim()) q.set('nama_pemilik', filter.namaPemilik.trim())
  if (filter.namaBank?.trim()) q.set('nama_bank', filter.namaBank.trim())
  if (filter.komiteSaya) q.set('komite_saya', '1')
  if (filter.batas !== undefined) q.set('batas', String(filter.batas))
  if (filter.lewati !== undefined) q.set('lewati', String(filter.lewati))
  const text = q.toString()
  return text === '' ? '' : `?${text}`
}

/**
 * Hook daftar rekening.
 *
 * Kelima tab layar memakai hook yang sama dengan saringan berbeda; `queryKey` memuat
 * saringannya sehingga berpindah tab tidak menampilkan hasil tab sebelumnya sesaat.
 */
export function useAccountList(filter: AccountFilter) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['master-rekening', portal, filter, token],
    queryFn: () =>
      callAPI<AccountListResponse>(`${ROUTES}${queryFrom(filter)}`, { token, portal }),
    enabled: token !== null,
    // Antrean komite berubah karena tindakan orang lain; hasil yang basi di layar
    // persetujuan berarti dua komite memutuskan rekening yang sama.
    staleTime: 10 * 1000,
  })
}

/**
 * Hook daftar bank.
 *
 * Daftarnya nyaris tidak pernah berubah dalam satu sesi kerja, sehingga disimpan lama —
 * memuatnya ulang setiap kali formulir dibuka hanya membebani basis data.
 */
export function useBankList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['master-rekening', portal, 'bank', token],
    queryFn: () => callAPI<BankListResponse>(`${ROUTES}/bank`, { token, portal }),
    enabled: token !== null,
    staleTime: 30 * 60 * 1000,
  })
}

/** Hook pengajuan rekening baru. */
export function useSubmitAccount() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (values: AccountFields) =>
      callAPI<Account>(ROUTES, { metode: 'POST', body: bodyOf(values), token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-rekening'] })
    },
  })
}

/** Hook perubahan rekening yang masih menunggu keputusan. */
export function useUpdateAccount() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ kodeBank, nomorRekening, values }: {
      kodeBank: string
      nomorRekening: string
      values: AccountFields
    }) =>
      callAPI<Account>(
        `${ROUTES}/${encodeURIComponent(kodeBank)}/${encodeURIComponent(nomorRekening)}`,
        { metode: 'PUT', body: bodyOf(values), token, portal },
      ),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-rekening'] })
    },
  })
}

/** Hook keputusan komite: menyetujui atau menolak satu rekening. */
export function useDecideAccount() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ kodeBank, nomorRekening, status, catatan, idDokumen }: {
      kodeBank: string
      nomorRekening: string
      status: AccountStatus
      catatan: string
      idDokumen?: string
    }) =>
      callAPI<Account>(
        `${ROUTES}/${encodeURIComponent(kodeBank)}/${encodeURIComponent(nomorRekening)}/keputusan`,
        {
          metode: 'POST',
          body: { status, catatan, id_dokumen: idDokumen ?? '' },
          token,
          portal,
        },
      ),
    onSuccess: () => {
      // Seluruh tab ikut disegarkan: satu keputusan memindahkan baris dari tab
      // "Waiting Approval" ke tab "Approve" atau "Reject" sekaligus.
      void client.invalidateQueries({ queryKey: ['master-rekening'] })
    },
  })
}
