import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  DokumenPenunjangItemResponse,
  DokumenPenunjangListResponse,
} from './types'

/**
 * Jalurnya bersarang di bawah KLAIM, bukan di bawah salah satu layar proteksi.
 *
 * Work Owner menetapkan dokumennya menempel *"Ke klaim, seperti Pega"*, dan bentuk datanya
 * menuntut hal yang sama. Akibatnya kedua layar proteksi memanggil jalur yang SAMA untuk
 * klaim yang sama — dan dokumen yang diunggah dari satu layar terlihat dari layar lainnya.
 */
function path(nomorKlaim: string): string {
  return `/api/klaim/${encodeURIComponent(nomorKlaim)}/dokumen-penunjang`
}

/**
 * Kunci cache dikumpulkan di satu tempat.
 *
 * PORTAL menjadi bagian kunci: dokumen sebuah klaim dibaca lewat basis data portalnya
 * sendiri (`ADR-0030`). Tanpa portal di kunci, berpindah portal akan menampilkan dokumen
 * entitas sebelumnya sampai permintaan baru selesai — dan itu kebocoran antar badan hukum
 * yang tampak normal di layar (`R-20`).
 */
const keys = {
  list: (portal: string | null, token: string | null, nomorKlaim: string) =>
    ['dokumen-penunjang', portal, token, nomorKlaim] as const,
}

/** Hook daftar dokumen penunjang sebuah klaim. */
export function useDokumenPenunjang(nomorKlaim: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const nomor = nomorKlaim?.trim() ?? ''

  return useQuery({
    queryKey: keys.list(portal, token, nomor),
    queryFn: () =>
      callAPI<DokumenPenunjangListResponse>(path(nomor), { token, portal }),

    // Tidak ditembak tanpa nomor klaim. Klaim kosong akan dijawab daftar kosong oleh
    // server, tetapi menembaknya tetap sia-sia — dan pada form yang klaimnya belum
    // dipilih, itu satu permintaan per ketikan.
    enabled: token !== null && portal !== null && nomor !== '',
  })
}

/** Bahan satu unggahan. */
export interface UnggahDokumenInput {
  nomorKlaim: string
  berkas: File
  jenisDokumen?: string
}

/**
 * Hook unggah satu dokumen.
 *
 * # Dikirim sebagai multipart, bukan JSON base64
 *
 * Pega mengirim base64 karena Connect REST hanya bicara JSON. Antara peramban dan backend
 * kita tidak ada batasan itu, dan multipart lebih baik pada dua hal yang nyata: muatannya
 * tidak membengkak sepertiga, dan berkasnya tidak perlu dirakit utuh sebagai teks di memori
 * peramban lebih dulu.
 *
 * Penyandian base64 tetap terjadi — di server, tepat sebelum dikirim ke layanan
 * penyimpanan, karena di sanalah ia memang dituntut.
 */
export function useUnggahDokumen(nomorKlaim: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const queryClient = useQueryClient()
  const nomor = nomorKlaim?.trim() ?? ''

  return useMutation({
    mutationFn: (input: UnggahDokumenInput) => {
      const form = new FormData()
      // Nama bagian `berkas` HARUS sama dengan yang dibaca handler
      // (`r.FormFile("berkas")`). Bila berbeda, server menjawab "berkas tidak ditemukan"
      // meski berkasnya jelas terkirim.
      form.append('berkas', input.berkas, input.berkas.name)
      if (input.jenisDokumen?.trim()) {
        form.append('jenis_dokumen', input.jenisDokumen.trim())
      }

      return callAPI<DokumenPenunjangItemResponse>(path(input.nomorKlaim), {
        metode: 'POST',
        body: form,
        token,
        portal,
      })
    },

    onSuccess: () => {
      // Daftar disegarkan dari SERVER, bukan disisipi hasil unggah.
      //
      // Alasannya bukan kerapian: `url` dan `berlaku_sampai` lahir saat metadata tercatat,
      // dan respons unggah dapat mengembalikannya kosong. Menyisipkan hasil itu apa adanya
      // menampilkan baris tanpa tautan yang tidak akan pernah terisi sampai halaman dimuat
      // ulang.
      queryClient.invalidateQueries({
        queryKey: keys.list(portal, token, nomor),
      })
    },
  })
}
