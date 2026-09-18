import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { panggilAPI } from '@/api/klien'
import type {
  Rekening,
  ResponsDaftarBank,
  ResponsDaftarRekening,
  StatusRekening,
} from '@/api/tipe'
import { gunakanSesi } from '@/app/sesi'

const JALUR = '/api/master-rekening'

/** Saringan daftar rekening; seluruh field boleh kosong. */
export type SaringanRekening = {
  status?: StatusRekening | ''
  nomorRekening?: string
  namaPemilik?: string
  namaBank?: string
  /** Membatasi ke antrean komite yang sedang masuk. Dipakai tab Komite Approval. */
  komiteSaya?: boolean
  batas?: number
  lewati?: number
}

/** Isian formulir rekening. */
export type IsianRekening = {
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

function badanDari(isian: IsianRekening) {
  return {
    nomor_rekening: isian.nomorRekening,
    nama_pemilik: isian.namaPemilik,
    nama_bank: isian.namaBank,
    cabang_bank: isian.cabangBank,
    alamat_bank: isian.alamatBank,
    kode_bank: isian.kodeBank,
    tipe_rekening: isian.tipeRekening,
    email: isian.email,
    telepon: isian.telepon,
    nik: isian.nik,
    id_dokumen: isian.idDokumen,
    catatan: isian.catatan,
    aktif: isian.aktif,
    kode_bank_lama: isian.kodeBankLama ?? '',
    nomor_rekening_lama: isian.nomorRekeningLama ?? '',
    nama_pemilik_lama: isian.namaPemilikLama ?? '',
  }
}

function kueriDari(saringan: SaringanRekening): string {
  const q = new URLSearchParams()
  if (saringan.status) q.set('status', saringan.status)
  if (saringan.nomorRekening?.trim()) q.set('nomor_rekening', saringan.nomorRekening.trim())
  if (saringan.namaPemilik?.trim()) q.set('nama_pemilik', saringan.namaPemilik.trim())
  if (saringan.namaBank?.trim()) q.set('nama_bank', saringan.namaBank.trim())
  if (saringan.komiteSaya) q.set('komite_saya', '1')
  if (saringan.batas !== undefined) q.set('batas', String(saringan.batas))
  if (saringan.lewati !== undefined) q.set('lewati', String(saringan.lewati))
  const teks = q.toString()
  return teks === '' ? '' : `?${teks}`
}

/**
 * Hook daftar rekening.
 *
 * Kelima tab layar memakai hook yang sama dengan saringan berbeda; `queryKey` memuat
 * saringannya sehingga berpindah tab tidak menampilkan hasil tab sebelumnya sesaat.
 */
export function gunakanDaftarRekening(saringan: SaringanRekening) {
  const token = gunakanSesi((keadaan) => keadaan.token)

  return useQuery({
    queryKey: ['master-rekening', saringan, token],
    queryFn: () =>
      panggilAPI<ResponsDaftarRekening>(`${JALUR}${kueriDari(saringan)}`, { token }),
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
export function gunakanDaftarBank() {
  const token = gunakanSesi((keadaan) => keadaan.token)

  return useQuery({
    queryKey: ['master-rekening', 'bank', token],
    queryFn: () => panggilAPI<ResponsDaftarBank>(`${JALUR}/bank`, { token }),
    enabled: token !== null,
    staleTime: 30 * 60 * 1000,
  })
}

/** Hook pengajuan rekening baru. */
export function gunakanAjukanRekening() {
  const token = gunakanSesi((keadaan) => keadaan.token)
  const klien = useQueryClient()

  return useMutation({
    mutationFn: (isian: IsianRekening) =>
      panggilAPI<Rekening>(JALUR, { metode: 'POST', badan: badanDari(isian), token }),
    onSuccess: () => {
      void klien.invalidateQueries({ queryKey: ['master-rekening'] })
    },
  })
}

/** Hook perubahan rekening yang masih menunggu keputusan. */
export function gunakanUbahRekening() {
  const token = gunakanSesi((keadaan) => keadaan.token)
  const klien = useQueryClient()

  return useMutation({
    mutationFn: ({ kodeBank, nomorRekening, isian }: {
      kodeBank: string
      nomorRekening: string
      isian: IsianRekening
    }) =>
      panggilAPI<Rekening>(
        `${JALUR}/${encodeURIComponent(kodeBank)}/${encodeURIComponent(nomorRekening)}`,
        { metode: 'PUT', badan: badanDari(isian), token },
      ),
    onSuccess: () => {
      void klien.invalidateQueries({ queryKey: ['master-rekening'] })
    },
  })
}

/** Hook keputusan komite: menyetujui atau menolak satu rekening. */
export function gunakanPutuskanRekening() {
  const token = gunakanSesi((keadaan) => keadaan.token)
  const klien = useQueryClient()

  return useMutation({
    mutationFn: ({ kodeBank, nomorRekening, status, catatan, idDokumen }: {
      kodeBank: string
      nomorRekening: string
      status: StatusRekening
      catatan: string
      idDokumen?: string
    }) =>
      panggilAPI<Rekening>(
        `${JALUR}/${encodeURIComponent(kodeBank)}/${encodeURIComponent(nomorRekening)}/keputusan`,
        {
          metode: 'POST',
          badan: { status, catatan, id_dokumen: idDokumen ?? '' },
          token,
        },
      ),
    onSuccess: () => {
      // Seluruh tab ikut disegarkan: satu keputusan memindahkan baris dari tab
      // "Waiting Approval" ke tab "Approve" atau "Reject" sekaligus.
      void klien.invalidateQueries({ queryKey: ['master-rekening'] })
    },
  })
}
