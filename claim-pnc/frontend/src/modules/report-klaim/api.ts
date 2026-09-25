import { useMutation, useQuery } from '@tanstack/react-query'

import { callAPI, simpanBerkas, unduhBerkas } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { BusinessOptionResponse, CatalogResponse, ExportRequest } from './types'

const ROUTE = '/api/report-klaim'

/**
 * Kunci cache menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (`ADR-0030`).
 * Tanpa itu, berpindah entitas akan menampilkan pilihan bisnis entitas sebelumnya dari
 * cache — dan tidak ada apa pun di layar yang menandakan daftar itu milik badan hukum
 * lain (`R-20`).
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama.
 */
function catalogKey(portal: string | null, token: string | null) {
  return ['report-klaim', portal, token] as const
}

function businessKey(portal: string | null, token: string | null) {
  return ['report-klaim-bisnis', portal, token] as const
}

/** Katalog 28 panel beserta keadaannya. */
export function useReportCatalog() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: catalogKey(portal, token),
    // Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa
    // portal (`TKT-F6-002`), tetapi menembaknya lebih dulu hanya untuk menerima
    // penolakan akan menampilkan pesan galat pada layar yang belum siap dibuka.
    enabled: portal !== null && token !== null,
    queryFn: () => callAPI<CatalogResponse>(ROUTE, { token, portal }),
  })
}

/**
 * Isi autocomplete "Bisnis".
 *
 * Ia dibaca TERPISAH dari katalog, dan hanya saat dibutuhkan: daftarnya dapat berisi
 * ratusan baris sementara hanya SATU dari 28 panel yang memakainya. Menyatukannya dengan
 * katalog berarti setiap kali layar dibuka ikut menarik daftar yang 27 panel lain tidak
 * membutuhkannya.
 */
export function useBusinessOptions(enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: businessKey(portal, token),
    enabled: enabled && portal !== null && token !== null,
    queryFn: () => callAPI<BusinessOptionResponse>(`${ROUTE}/pilihan-bisnis`, { token, portal }),
  })
}

/**
 * searchParams menyusun parameter kueri, dan HANYA mengirim penyaring yang berlaku.
 *
 * Mengirim rentang tanggal untuk laporan yang tidak memakainya tidak membuat hasilnya
 * salah — backend mengabaikannya — tetapi ia membuat alamat permintaannya berbohong
 * tentang apa yang menentukan isi berkas. Pada laporan yang hasilnya dipertanyakan, itu
 * jejak pertama yang dibaca orang.
 */
function searchParams(request: ExportRequest): string {
  const param = new URLSearchParams()
  const { filter, penyaring } = request

  if (request.aksi !== '') param.set('aksi', request.aksi)
  if (penyaring.rentang_tanggal) {
    if (filter.dari !== '') param.set('dari', filter.dari)
    if (filter.sampai !== '') param.set('sampai', filter.sampai)
  }
  if (penyaring.lini_bisnis && filter.lini !== '') param.set('lini', filter.lini)
  if (penyaring.status_compliance && filter.status_compliance !== '') {
    param.set('status_compliance', filter.status_compliance)
  }
  if (penyaring.bisnis && filter.bisnis !== '') param.set('bisnis', filter.bisnis)
  if (penyaring.rincian && filter.rincian) param.set('rincian', '1')

  return param.toString()
}

/**
 * Tombol Export pada satu kartu.
 *
 * Unduhannya menempuh `unduhBerkas`, bukan `<a download>`: permintaannya butuh header
 * `Authorization` dan `X-Portal` yang tidak dapat disertakan pada navigasi peramban
 * biasa — dan tanpa keduanya berkasnya akan ditolak, atau lebih buruk, dilayani entitas
 * yang salah.
 */
export function useExportReport() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: async (request: ExportRequest) => {
      const query = searchParams(request)
      const path = `${ROUTE}/${encodeURIComponent(request.kode)}/ekspor${query ? `?${query}` : ''}`

      const berkas = await unduhBerkas(path, { token, portal })
      simpanBerkas(berkas)
      return berkas
    },
  })
}
