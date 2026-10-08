import { useMutation, useQuery, type QueryKey } from '@tanstack/react-query'

import { callAPI, HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

/**
 * Potongan bersama lapisan API layar-layar inbox.
 *
 * Setiap modul inbox membaca token sesi dan portal aktif dengan cara yang sama, memuat
 * "bentuk layar" (tab, kolom) sekali seumur aplikasi, memuat daftar berhalaman dengan cache
 * pendek, dan mengunduh berkas ekspor lewat tautan sementara. Bagian-bagian itu tinggal di
 * sini supaya tidak ada dua tafsir tentang, misalnya, kapan sebuah permintaan boleh jalan.
 */

/** Penyusun kunci kueri dari portal dan token aktif. */
export type KeyOf = (portal: string | null, token: string | null) => QueryKey

/** usePortalSession membaca token sesi dan alias portal yang sedang dipilih. */
export function usePortalSession() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  return { token, portal }
}

/**
 * useScreenMetadata memuat bentuk layar (tab, kolom, selisih terencana).
 *
 * Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode server, bukan
 * dari data. Mengambilnya ulang tiap kali tab berpindah hanya menambah perjalanan jaringan
 * tanpa satu pun manfaat — karena itu `staleTime` dan `gcTime` tak terhingga.
 */
export function useScreenMetadata<T>(keyOf: KeyOf, path: string) {
  const { token, portal } = usePortalSession()

  return useQuery({
    queryKey: keyOf(portal, token),
    queryFn: () => callAPI<T>(path, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * useQueueList memuat satu halaman antrean.
 *
 * Hasil sebelumnya ditahan selama halaman berikutnya dimuat, alih-alih tabel berkedip
 * menjadi kosong lalu terisi lagi. Antrean berubah saat petugas lain bekerja, jadi
 * cache-nya pendek.
 */
export function useQueueList<T>(
  keyOf: KeyOf,
  path: string,
  enabled: boolean,
  staleTime: number = 15 * 1000,
) {
  const { token, portal } = usePortalSession()

  return useQuery<T>({
    queryKey: keyOf(portal, token),
    queryFn: () => callAPI<T>(path, { token, portal }),
    enabled: enabled && token !== null && portal !== null,
    placeholderData: (previous) => previous,
    staleTime,
  })
}

/**
 * queryPath menyusun alamat beserta parameternya.
 *
 * Isian yang kosong TIDAK dikirim, bukan dikirim sebagai teks kosong: server membedakan
 * "tidak dikirim" dari "dikirim kosong". Urutan parameter mengikuti urutan `entries`.
 */
export function queryPath(
  path: string,
  entries: [string, string | false | null | undefined][],
): string {
  const params = new URLSearchParams()
  for (const [name, value] of entries) {
    if (value) params.set(name, value)
  }
  return withQuery(path, params)
}

/**
 * usePortalQuery memuat satu sumber data milik portal aktif dengan cache pendek, tanpa
 * menahan hasil sebelumnya (dipakai panel ringkasan dan rincian).
 */
export function usePortalQuery<T>(
  keyOf: KeyOf,
  path: string,
  enabled: boolean,
  staleTime: number = 15 * 1000,
) {
  const { token, portal } = usePortalSession()

  return useQuery({
    queryKey: keyOf(portal, token),
    queryFn: () => callAPI<T>(path, { token, portal }),
    enabled: enabled && token !== null && portal !== null,
    staleTime,
  })
}

/**
 * useRecordDetail memuat satu rekaman menurut kuncinya (`<basePath>/<id>`). Tidak berjalan
 * selama kuncinya belum ada — panel rinciannya memang belum dibuka.
 */
export function useRecordDetail<T>(keyOf: KeyOf, basePath: string, id: string | null) {
  const { token, portal } = usePortalSession()

  return useQuery({
    queryKey: keyOf(portal, token),
    queryFn: () =>
      callAPI<T>(`${basePath}/${encodeURIComponent(id ?? '')}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && id !== null && id !== '',
  })
}

/**
 * offsetSearchPath menyusun alamat daftar berpaginasi offset: pencarian (dipangkas, tidak
 * dikirim bila kosong), jumlah baris yang dilewati (tidak dikirim bila nol), lalu batasnya.
 * `params` membawa parameter yang harus mendahuluinya.
 */
export function offsetSearchPath(
  path: string,
  params: URLSearchParams,
  filter: { search?: string | undefined; offset?: number | undefined },
  limit: number,
): string {
  if (filter.search?.trim()) params.set('cari', filter.search.trim())
  if (filter.offset) params.set('lewati', String(filter.offset))
  params.set('batas', String(limit))

  return `${path}?${params.toString()}`
}

/**
 * usePersonalTaskList memuat satu halaman antrean tugas pribadi (Analyst Doctor, RCL).
 *
 * Antreannya berubah setiap kali sebuah tugas diselesaikan — termasuk oleh Pega, yang masih
 * memiliki penugasannya selama masa paralel. Data dianggap usang seketika, tetapi tidak
 * ditembak ulang sendiri: pengguna yang menekan Muat ulang. Halaman sebelumnya dipertahankan
 * selama halaman berikutnya diambil, supaya tabel tidak berkedip.
 */
export function usePersonalTaskList<T>(
  keyOf: KeyOf,
  path: string,
  search: string,
  offset: number,
  limit: number,
) {
  const { token, portal } = usePortalSession()

  const params = new URLSearchParams()
  if (search.trim()) params.set('cari', search.trim())
  if (offset > 0) params.set('lewati', String(offset))
  params.set('batas', String(limit))

  return useQuery<T>({
    queryKey: keyOf(portal, token),
    queryFn: () => callAPI<T>(`${path}?${params.toString()}`, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: 0,
    placeholderData: (previous) => previous,
  })
}

/**
 * useTabExport adalah tombol ekspor antrean yang disaring satu tab saja.
 *
 * Tab yang kosong TIDAK dikirim: server membedakan "tidak dikirim" dari "dikirim kosong",
 * dan yang pertama berarti tab pertama yang boleh dilihat pemanggil.
 */
export function useTabExport(path: string, fallbackName: string) {
  const { token, portal } = usePortalSession()

  return useMutation({
    mutationFn: async (tab: string) => {
      await downloadExport({
        address: queryPath(`${path}/ekspor`, [['tab', tab]]),
        token,
        portal,
        fallbackName,
      })
    },
  })
}

/** withQuery menempelkan parameter ke alamat, tanpa `?` bila parameternya kosong. */
export function withQuery(path: string, params: URLSearchParams): string {
  const query = params.toString()
  return query ? `${path}?${query}` : path
}

/** filenameOf membaca nama berkas dari header Content-Disposition. */
export function filenameOf(response: Response): string | null {
  const disposition = response.headers.get('Content-Disposition')
  if (!disposition) return null
  const found = /filename="([^"]+)"/.exec(disposition)
  return found?.[1] ?? null
}

/**
 * downloadBlob menyimpan berkas lewat tautan sementara.
 *
 * URL objeknya DICABUT setelah dipakai. Tanpa itu, blob-nya tetap dipegang peramban sampai
 * tab ditutup — dan pada layar yang dipakai sepanjang hari, setiap ekspor menumpuk memori
 * yang tidak pernah dilepas.
 */
export function downloadBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  document.body.appendChild(link)
  link.click()
  link.remove()
  URL.revokeObjectURL(url)
}

/**
 * downloadCSVFile mengunduh berkas lewat permintaan ber-header lalu menyerahkannya ke
 * peramban sebagai objek URL.
 *
 * Ia TIDAK dapat dijadikan `<a href>` biasa, karena permintaannya menuntut dua header —
 * `Authorization` dan `X-Portal` — dan tautan biasa tidak dapat mengirim header. Berkas
 * dimuat seluruhnya ke memori peramban lebih dulu, dan itu diterima: backend sudah membatasi
 * jumlah barisnya.
 */
export async function downloadCSVFile(
  url: string,
  token: string,
  portal: string,
  fallbackName: string,
): Promise<void> {
  const response = await fetch(url, {
    headers: {
      Authorization: `Bearer ${token}`,
      [HEADER_PORTAL]: portal,
    },
  })
  if (!response.ok) {
    throw new Error('Berkas tidak dapat diunduh.')
  }

  const blob = await response.blob()
  const objectURL = URL.createObjectURL(blob)

  try {
    const link = document.createElement('a')
    link.href = objectURL
    link.download = filenameOf(response) ?? fallbackName
    document.body.appendChild(link)
    link.click()
    link.remove()
  } finally {
    // Objek URL menahan blob-nya di memori sampai dicabut. Tanpa ini, setiap unduhan
    // meninggalkan satu salinan berkas yang tidak pernah dilepas.
    URL.revokeObjectURL(objectURL)
  }
}

/** authHeaders menyusun header otorisasi dan portal untuk `fetch` langsung. */
export function authHeaders(token: string | null, portal: string | null): Record<string, string> {
  const header: Record<string, string> = {}
  if (token) header['Authorization'] = `Bearer ${token}`
  if (portal) header[HEADER_PORTAL] = portal
  return header
}

/**
 * downloadExport mengambil berkas ekspor lalu menyimpannya.
 *
 * Galat dijawab sebagai JSON selama header belum terkirim; setelah itu tidak bisa lagi.
 * Yang dibaca di sini adalah kasus pertama — pesannya diteruskan apa adanya.
 */
export async function downloadExport(input: {
  address: string
  token: string | null
  portal: string | null
  fallbackName: string
  fallbackError?: string | undefined
}): Promise<void> {
  const response = await fetch(input.address, { headers: authHeaders(input.token, input.portal) })
  if (!response.ok) {
    const body = (await response.json().catch(() => null)) as { pesan?: string } | null
    throw new Error(body?.pesan ?? input.fallbackError ?? 'Berkas ekspor tidak dapat diambil.')
  }

  const blob = await response.blob()
  downloadBlob(blob, filenameOf(response) ?? input.fallbackName)
}
