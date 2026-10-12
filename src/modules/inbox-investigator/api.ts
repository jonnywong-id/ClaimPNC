import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI, HEADER_PORTAL } from '@/api/client'
import type { InvestigatorInboxResponse } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/inbox/investigator'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (`ADR-0030`).
 * Tanpa itu, berpindah entitas akan menampilkan antrean entitas sebelumnya dari cache —
 * petugas melihat daftar pekerjaan yang masuk akal, dan tidak ada apa pun di layar yang
 * menandakan pekerjaan itu milik badan hukum lain (`R-20`).
 *
 * TANPA kata kunci di dalam kunci cache: penyaringan dikerjakan PERAMBAN atas baris yang
 * sudah di tangan (keputusan Work Owner 2026-09-23), sehingga mengetik tidak menembak server
 * dan tidak melahirkan entri cache baru per huruf.
 */
function listKey(portal: string | null, token: string | null) {
  return ['inbox-investigator', portal, token] as const
}

/**
 * Hook daftar Inbox Investigator.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (`TKT-F6-002`), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 *
 * # HANYA hook baca, dan itu bukan kelalaian
 *
 * Tidak ada `useAmbilTugas...` maupun `useSimpan...` di berkas ini. Layar ini tidak mengubah
 * apa pun: mengambil pekerjaan dari antrean dan mencatat hasil investigasi
 * (`SetStatusInvestigator_Act`) terjadi di layar kerja yang belum dibangun, dan endpoint
 * tulisnya pun tidak ada di server.
 *
 * # Penyaring `cari` milik endpoint ini TIDAK dipakai layar
 *
 * Endpoint-nya menerimanya, dan ia tetap disediakan supaya perpindahan ke penyaringan sisi
 * server kelak (`TKT-U2-001`) tidak menuntut perubahan kontrak. Yang dipakai layar sekarang
 * adalah pencarian bawaan `DataTable` — sama seperti layar master lain, dan sesuai pilihan
 * Work Owner bahwa penyaringan dikerjakan peramban seperti grid Pega.
 *
 * # Kenapa TIDAK di-refetch berkala
 *
 * Antrean bersama berubah tanpa tindakan pengguna — orang lain mengambil pekerjaan, dan job
 * terjadwal menambahkannya. Penyegaran otomatis karena itu menggoda, tetapi ia memindahkan
 * baris di bawah kursor orang yang sedang membaca. Yang dipilih adalah tombol Refresh yang
 * ditekan sendiri, sama seperti tombol Refresh pada harness lamanya.
 */
export function useInvestigatorInbox() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<InvestigatorInboxResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/** Ketiga kendali tombol Export Data Investigation. */
export type ExportInvestigationInput = {
  /** Isian "Dari", bentuk `YYYY-MM-DD`. */
  dari: string
  /** Isian "Sampai", bentuk `YYYY-MM-DD`; hari itu IKUT terbawa. */
  sampai: string
  /** Dropdown "Pilih Investigation": `"1"` atau `"0"`. */
  investigasi: string
}

/**
 * Hook tombol **Export Data Investigation**.
 *
 * # Ia `fetch` langsung, bukan `callAPI`
 *
 * `callAPI` menguraikan jawabannya sebagai JSON; yang datang di sini berkas CSV. Memakainya
 * akan menggagalkan setiap unduhan yang berhasil — dan kegagalannya berbunyi seperti galat
 * server, bukan seperti salah alat.
 *
 * # Ia mutation, meski endpoint-nya GET
 *
 * Bukan karena ada yang berubah di server — tidak ada. Yang dibutuhkan adalah pemicuan oleh
 * tombol beserta keadaan "sedang berjalan" dan "gagal"-nya; `useQuery` dirancang untuk
 * memuat sendiri saat layar dibuka, dan mengunduh berkas saat layar dibuka adalah hal
 * terakhir yang diinginkan siapa pun.
 *
 * # Galat dibaca SELAMA masih dapat dibaca
 *
 * Server menjawab galat sebagai JSON hanya sampai header terkirim; sesudahnya yang sampai
 * adalah berkas separuh jadi. Pesan yang dibaca di sini adalah kasus pertama — dan itulah
 * seluruh penolakan penyaring, yang terjadi sebelum satu byte pun ditulis.
 */
export function useExportInvestigation() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (input: ExportInvestigationInput) => {
      const header: Record<string, string> = {}
      if (token) header['Authorization'] = `Bearer ${token}`
      if (portal) header[HEADER_PORTAL] = portal

      const params = new URLSearchParams({
        dari: input.dari,
        sampai: input.sampai,
        investigasi: input.investigasi,
      })

      const response = await fetch(`${ROUTE}/ekspor?${params.toString()}`, {
        headers: header,
      })
      if (!response.ok) {
        const body = (await response.json().catch(() => null)) as { pesan?: string } | null
        throw new Error(body?.pesan ?? 'Berkas data investigasi tidak dapat diambil.')
      }

      const blob = await response.blob()
      downloadBlob(blob, filenameOf(response) ?? 'data-investigasi.csv')
    },
  })
}

/** filenameOf membaca nama berkas dari header Content-Disposition. */
function filenameOf(response: Response): string | null {
  const disposition = response.headers.get('Content-Disposition')
  if (!disposition) return null
  const found = /filename="([^"]+)"/.exec(disposition)
  return found?.[1] ?? null
}

/**
 * downloadBlob menyimpan berkas yang sudah diterima ke folder unduhan peramban.
 *
 * Tautannya dibuat, ditekan, lalu DIBUANG beserta URL objeknya. Tanpa pembuangan itu,
 * setiap unduhan meninggalkan satu berkas di memori tab sampai halaman ditutup — dan pada
 * berkas berisi data medis, "masih ada di memori" bukan hal yang dibiarkan tanpa alasan.
 */
function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
}

/** Bentuk formulir investigasi pada kontrak API. */
export type InvestigationForm = {
  referensi: string
  urutan_survei: number
  urutan: number

  tanggal_investigasi: string | null
  dapat_diinvestigasi: string
  tempat_kejadian: string
  nama_rumah_sakit: string
  nama_tempat_lainnya: string
  alamat_rs_klinik: string
  nomor_rekam_medik: string
  nama_pasien: string
  tanggal_lahir: string | null
  verifikasi_tanggal_lahir: string
  keterangan_tanggal_lahir: string
  peserta_terdaftar: string
  keterangan_pendaftaran: string
  tanggal_perawatan: string | null
  tanggal_selesai_perawatan: string | null
  total_pengajuan: string
  tagihan_lunas: string
  bayar_pasien: string
  bayar_perusahaan: string
  bayar_asuransi_lain: string
  tidak_ada_pembayaran: string
  nama_asuransi_lain: string
  konfirmasi_kwitansi: string
  nama_pic_rs: string
  nama_penelepon: string
  nama_karyawan: string
  kode_area_telepon: string
  nomor_telepon: string
  ekstensi_telepon: string
  hasil_investigasi: string
}

/**
 * Keadaan tampil keenam isian bersyarat.
 *
 * Datang dari SERVER, tidak dihitung ulang di sini. Keenam syaratnya dibaca dari
 * `pyCondition` section lama — ia aturan, bukan selera tata letak, dan aturan yang hidup di
 * dua tempat akan berbeda pada perubahan berikutnya.
 */
export type InvestigationVisibility = {
  nama_rumah_sakit: boolean
  nama_tempat_lainnya: boolean
  alamat_rs_klinik: boolean
  tanggal_selesai_perawatan: boolean
  nama_asuransi_lain: boolean
}

export type InvestigationResponse = {
  investigasi: InvestigationForm
  tampil: InvestigationVisibility
  portal: string
}

export type InvestigationTransition = {
  referensi: string
  status_survei: string
  status_pnc: string
  status_klaim: string
  pada: string
}

export type SubmitInvestigationResponse = {
  pindah: InvestigationTransition
}

/** Kunci cache formulir satu pekerjaan. */
function formKey(reference: string, portal: string | null, token: string | null) {
  return ['inbox-investigator-form', reference, portal, token] as const
}

/**
 * Hook membuka formulir investigasi satu pekerjaan.
 *
 * # Ia GET, meski servernya mengisi Tanggal Investigasi
 *
 * Yang diisi TIDAK disimpan: nilainya hanya dikembalikan sebagai isian awal, dan baru
 * tersimpan bila pengguna menekan Simpan. Membuka formulir berkali-kali karena itu tidak
 * mengubah apa pun.
 *
 * # `enabled` menunggu referensinya ada
 *
 * Formulir dibuka dari modal yang tertutup saat layar dimuat. Tanpa penjagaan ini, setiap
 * pembukaan layar akan menembak server untuk pekerjaan yang belum dipilih siapa pun.
 */
export function useInvestigation(reference: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: formKey(reference ?? '', portal, token),
    queryFn: () =>
      callAPI<InvestigationResponse>(
        `${ROUTE}/${encodeURIComponent(reference ?? '')}/investigasi`,
        { token, portal },
      ),
    enabled: token !== null && portal !== null && Boolean(reference),
  })
}

/**
 * Hook menyimpan formulir investigasi.
 *
 * # Apa yang terjadi setelah ini
 *
 * Klaimnya berpindah ke Analyst dan pekerjaannya HILANG dari antrean. Karena itu daftar
 * inbox ikut di-invalidasi — tanpa itu, baris yang sudah dikerjakan tetap tampil sampai
 * pengguna menekan Refresh, dan ia akan menyimpulkan simpannya gagal.
 */
export function useSubmitInvestigation() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const apiClient = useQueryClient()

  return useMutation({
    mutationFn: (input: { referensi: string; isian: Partial<InvestigationForm> }) =>
      callAPI<SubmitInvestigationResponse>(
        `${ROUTE}/${encodeURIComponent(input.referensi)}/investigasi`,
        { metode: 'POST', body: input.isian, token, portal },
      ),
    onSuccess: (_hasil, input) => {
      apiClient.invalidateQueries({ queryKey: listKey(portal, token) })
      apiClient.invalidateQueries({ queryKey: formKey(input.referensi, portal, token) })
    },
  })
}
