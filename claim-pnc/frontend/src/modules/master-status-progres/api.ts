import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { panggilAPI } from '@/api/klien'
import type {
  IsianStatusProgres,
  ResponsDaftarPosisiKlaim,
  ResponsDaftarStatusProgres,
  ResponsSatuStatusProgres,
} from '@/api/tipe'
import { gunakanPortalTerpilih } from '@/app/portal'
import { gunakanSesi } from '@/app/sesi'

const JALUR = '/api/master/status-progres-1'
const JALUR_POSISI = '/api/master/posisi-klaim'

/**
 * kunciDaftar menyertakan portal DAN token.
 *
 * Portal ikut di dalam kunci karena itulah yang menentukan basis data mana yang
 * menjawab (ADR-0030). Tanpa itu, berpindah entitas akan menampilkan data entitas
 * sebelumnya dari cache — pengguna melihat angka yang masuk akal, dan tidak ada apa pun
 * di layar yang menandakan data itu milik badan hukum lain (R-20).
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama, mengikuti pola gunakanDaftarPortal.
 */
function kunciDaftar(portal: string | null, token: string | null) {
  return ['master-status-progres-1', portal, token] as const
}

/**
 * Hook daftar master status progres 1.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa
 * portal (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan
 * akan menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka — layar
 * yang menuntun pengguna memilih portal lebih berguna daripada pesan galat.
 */
export function gunakanDaftarStatusProgres() {
  const token = gunakanSesi((keadaan) => keadaan.token)
  const portal = gunakanPortalTerpilih((keadaan) => keadaan.alias)

  return useQuery({
    queryKey: kunciDaftar(portal, token),
    queryFn: () => panggilAPI<ResponsDaftarStatusProgres>(JALUR, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook daftar posisi klaim untuk dropdown.
 *
 * Tidak menuntut portal: keempat posisi adalah daftar milik aplikasi, bukan isi basis
 * data entitas mana pun (lihat internal/statusprogres/posisi.go).
 *
 * Daftarnya diambil dari server, tidak disalin ke sini. Menyalinnya berarti keempat
 * nilai itu hidup di dua tempat, dan tempat kedua akan terlupa ketika daftarnya kelak
 * pindah menjadi master data `F-4`.
 */
export function gunakanDaftarPosisiKlaim() {
  const token = gunakanSesi((keadaan) => keadaan.token)

  return useQuery({
    queryKey: ['posisi-klaim', token],
    queryFn: () => panggilAPI<ResponsDaftarPosisiKlaim>(JALUR_POSISI, { token }),
    enabled: token !== null,
    // Daftarnya tetap selama aplikasi berjalan; memuatnya ulang setiap kali komponen
    // dipasang hanya menambah permintaan tanpa menambah apa pun.
    staleTime: 60 * 60 * 1000,
  })
}

/** Hook penambahan status progres. */
export function gunakanTambahStatusProgres() {
  const token = gunakanSesi((keadaan) => keadaan.token)
  const portal = gunakanPortalTerpilih((keadaan) => keadaan.alias)
  const klien = useQueryClient()

  return useMutation({
    mutationFn: (isian: IsianStatusProgres) =>
      panggilAPI<ResponsSatuStatusProgres>(JALUR, {
        metode: 'POST',
        badan: isian,
        token,
        portal,
      }),
    // Daftar dimuat ulang dari server, BUKAN ditambahi barisnya di sisi klien. ID baru
    // diterbitkan server dari isi tabel, dan petugas lain dapat menambah baris pada
    // saat yang sama — daftar yang disusun sendiri di peramban akan berbeda dari isi
    // tabel yang sebenarnya.
    onSuccess: () => {
      void klien.invalidateQueries({ queryKey: kunciDaftar(portal, token) })
    },
  })
}

/** Hook penyuntingan status progres. */
export function gunakanUbahStatusProgres() {
  const token = gunakanSesi((keadaan) => keadaan.token)
  const portal = gunakanPortalTerpilih((keadaan) => keadaan.alias)
  const klien = useQueryClient()

  return useMutation({
    mutationFn: ({ id, isian }: { id: string; isian: IsianStatusProgres }) =>
      panggilAPI<ResponsSatuStatusProgres>(`${JALUR}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        badan: isian,
        token,
        portal,
      }),
    onSuccess: () => {
      void klien.invalidateQueries({ queryKey: kunciDaftar(portal, token) })
    },
  })
}
