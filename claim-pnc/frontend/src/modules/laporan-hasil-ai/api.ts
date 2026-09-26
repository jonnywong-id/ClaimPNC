import { useMutation, useQuery } from '@tanstack/react-query'

import { callAPI, simpanBerkas, unduhBerkas } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { isComplete, type FilterInput, type SearchResponse } from './types'

const ROUTE = '/api/laporan-hasil-ai'

/**
 * Kunci cache menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (`ADR-0030`).
 * Tanpa itu, berpindah entitas akan menampilkan laporan entitas sebelumnya dari cache —
 * dan tidak ada apa pun di layar yang menandakan baris itu milik badan hukum lain
 * (`R-20`). Di layar ini akibatnya nyata: barisnya memuat keputusan uang.
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama.
 */
function searchKey(
  portal: string | null,
  token: string | null,
  filter: FilterInput,
  page: number,
) {
  return ['laporan-hasil-ai', portal, token, filter.dari, filter.sampai, page] as const
}

/** Menyusun parameter kueri; nama parameternya sama dengan yang dibaca handler. */
function searchParams(filter: FilterInput, page: number): string {
  const param = new URLSearchParams()
  if (filter.dari !== '') param.set('dari', filter.dari)
  if (filter.sampai !== '') param.set('sampai', filter.sampai)
  param.set('halaman', String(page))
  return param.toString()
}

/**
 * Isi kedua grid — ringkasan dan rincian — dalam SATU permintaan.
 *
 * Keduanya bersama karena layar lama pun mengisinya dalam satu kali jalan
 * (`SearchDataLaporanAI` mengisi `TempTotal` dan `DatasearchLaporan` berurutan). Memecahnya
 * menjadi dua permintaan membuka kemungkinan angka ringkasan dan isi grid dibaca dari
 * keadaan basis data yang berbeda — dan selisihnya akan terbaca sebagai salah hitung.
 *
 * `enabled` menahan permintaan sampai tombol ditekan DAN kedua tanggal terisi. Tanpa itu
 * layar akan menembak server pada pemuatan pertama hanya untuk menerima 422, lalu
 * menampilkan pesan galat pada layar yang belum pernah disentuh pengguna.
 */
export function useLaporanHasilAI(filter: FilterInput, page: number, searched: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: searchKey(portal, token, filter, page),
    enabled: searched && isComplete(filter) && portal !== null && token !== null,
    queryFn: () =>
      callAPI<SearchResponse>(`${ROUTE}?${searchParams(filter, page)}`, { token, portal }),
  })
}

/**
 * Tombol "Export To Excel".
 *
 * # Di Pega ia BUKAN tombol yang berbeda
 *
 * Tombol itu memanggil activity yang SAMA dengan "Cari Data" —
 * `SearchDataLaporanAI(flagss=2)` — hanya dibuka di jendela baru sehingga jawabannya
 * terunduh sebagai berkas. Di sini keduanya dua rute karena bentuk keluarannya memang
 * berbeda, tetapi penyaringnya sama persis.
 *
 * Unduhannya menempuh `unduhBerkas`, bukan `<a download>`: permintaannya butuh header
 * `Authorization` dan `X-Portal` yang tidak dapat disertakan pada navigasi peramban biasa —
 * dan tanpa keduanya berkasnya akan ditolak, atau lebih buruk, dilayani entitas yang salah.
 *
 * Berkasnya memuat SELURUH baris yang cocok, bukan halaman yang sedang terbuka. Layar lama
 * pun begitu: grid-nya memuat semuanya sekaligus, dan CSV-nya dibuat dari muatan itu.
 */
export function useExportLaporanHasilAI() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (filter: FilterInput) => {
      const param = new URLSearchParams()
      if (filter.dari !== '') param.set('dari', filter.dari)
      if (filter.sampai !== '') param.set('sampai', filter.sampai)

      const berkas = await unduhBerkas(`${ROUTE}/ekspor?${param.toString()}`, { token, portal })
      simpanBerkas(berkas)
      return berkas
    },
  })
}
