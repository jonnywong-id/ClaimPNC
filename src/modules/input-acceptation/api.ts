import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { DetailResponse, SubmitRequest, SubmitResponse } from './types'

const PATH = '/api/input-acceptation'

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Portal ikut menjadi bagian kunci, dan itu BUKAN kerapian: akseptasi satu badan hukum bukan
 * akseptasi badan hukum lain, dan menyimpan keduanya di bawah satu kunci akan membuat
 * perpindahan portal menampilkan nilai klaim entitas sebelumnya (`R-20`).
 */
const keys = {
  detail: (portal: string | null, token: string | null, claimID: string) =>
    ['input-acceptation', portal, token, claimID] as const,
}

/**
 * Hook rincian akseptasi satu klaim.
 *
 * # Kenapa bentuk layar TIDAK diambil terpisah
 *
 * Karena layar ini selalu dibuka untuk satu klaim tertentu, dan server mengirim bentuk beserta
 * isinya dalam satu jawaban. Mengambil bentuknya terpisah berarti dua perjalanan untuk satu
 * layar, dan kemungkinan keduanya menjawab keadaan yang berbeda.
 */
export function useInputAcceptation(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.detail(portal, token, claimID),
    queryFn: () =>
      callAPI<DetailResponse>(`${PATH}/${encodeURIComponent(claimID)}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && claimID !== '',

    // Akseptasi berubah saat petugas lain menyimpannya, jadi cache-nya pendek. Ia tidak
    // dibuat nol: membuka ulang layar yang sama dalam hitungan detik — yang terjadi setiap
    // kali pengguna menekan kembali dari sebuah tautan — tidak perlu menembak server lagi.
    staleTime: 15 * 1000,
  })
}

/**
 * Hook tombol Submit.
 *
 * # Ia MENOLAK hari ini, dan itu bukan cacat
 *
 * Selama Pega dan sistem baru berjalan berdampingan, tabel objek kerja dan `JSON_KLAIM` hanya
 * boleh ditulis satu sistem (`P-1`), dan keduanya masih dimiliki Pega. Server karena itu
 * menjawab `409` ber-kode `belum_dapat_disimpan`, dengan pesan yang menyebut sebab dan jalan
 * keluarnya.
 *
 * Penolakan itu terjadi SESUDAH validasi muatan. Pengguna yang mengirim isian keliru tetap
 * diberi tahu apa yang keliru — bukan hanya diberi tahu bahwa penyimpanannya belum tersedia.
 *
 * # Rinciannya disimpan pada galat
 *
 * Jawaban `422` membawa daftar pelanggaran per isian, dan layar menandai isiannya. Membuang
 * daftar itu akan memaksa pengguna menebak isian mana yang ditolak dari satu kalimat.
 */
export function useSubmitInputAcceptation(claimID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (body: SubmitRequest) =>
      callAPI<SubmitResponse>(`${PATH}/${encodeURIComponent(claimID)}`, {
        metode: 'POST',
        token,
        portal,
        body,
      }),

    // Berhasil menyimpan berarti nilai yang dihitung server berubah — Nomor Akseptasi,
    // status, dan seluruh grid hasil hitungan. Rinciannya diambil ulang alih-alih ditebak
    // dari muatan yang dikirim.
    onSuccess: () => {
      client.invalidateQueries({
        queryKey: keys.detail(portal, token, claimID),
      })
    },
  })
}
