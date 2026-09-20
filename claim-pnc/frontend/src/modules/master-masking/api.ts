import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  MaskingBranchListResponse,
  MaskingInput,
  MaskingListResponse,
  MaskingResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/masking'

/**
 * Tipe pencarian yang dikenal server. Nilainya sama persis dengan mastermasking.SearchBy.
 *
 * Keempatnya diambil dari `SearchData.Type` pada layar lama:
 *
 *	''        Type "1"  tidak menyaring apa pun
 *	'cabang'  Type "2"  nama cabang, cocok sebagian
 *	'login'   Type "3"  nama pengguna, cocok sebagian
 *	'status'  Type "4"  status aktif, cocok PERSIS
 */
export type SearchBy = '' | 'cabang' | 'login' | 'status'

/** Dua nilai sah status, sama persis dengan kolom STS_AKTF. */
export const MaskingStatus = {
  aktif: 'AKTIF',
  tidakAktif: 'TIDAK AKTIF',
} as const

export type MaskingFilter = {
  cariDi: SearchBy
  kataKunci: string
}

/**
 * listKey menyertakan portal, token, DAN penyaring.
 *
 * Portal ikut di dalam kunci karena itulah yang menentukan basis data mana yang menjawab
 * (ADR-0030). Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari
 * cache — pengguna melihat daftar yang masuk akal, dan tidak ada apa pun di layar yang
 * menandakan data itu milik badan hukum lain (R-20). Pada layar ini akibatnya paling
 * berat di antara master yang sudah dibangun: isinya daftar siapa yang boleh membuka
 * nomor KTP dan nomor telepon nasabah.
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama.
 *
 * Penyaring ikut supaya hasil pencarian tidak saling menimpa di cache.
 */
function listKey(portal: string | null, token: string | null, filter: MaskingFilter) {
  return ['master-masking', portal, token, filter.cariDi, filter.kataKunci] as const
}

function listPath(filter: MaskingFilter) {
  const query = new URLSearchParams()
  // Tipe hanya dikirim bila nilainya ada. Mengirim tipe tanpa nilai akan menyaring dengan
  // pola kosong — yang cocok dengan segalanya untuk cabang dan login, dan DITOLAK server
  // untuk status. Keduanya bukan yang dimaksud pengguna yang belum mengisi apa pun.
  if (filter.cariDi !== '' && filter.kataKunci.trim() !== '') {
    query.set('cari_di', filter.cariDi)
    query.set('kata_kunci', filter.kataKunci.trim())
  }

  const text = query.toString()
  return text === '' ? ROUTE : `${ROUTE}?${text}`
}

/**
 * Hook daftar Master Masking.
 *
 * Menggantikan `RDB List/SearchMasking_SQL-SQL.xml` yang mengisi grid layar
 * `MasterProteksiVisibilityData`.
 *
 * Seluruh baris dimuat sekaligus, tanpa paginasi server. Itu keputusan yang diambil
 * dengan angka: isinya 25 baris pada portal ASM, dan master ini bertambah hanya ketika
 * seorang petugas diberi kewenangan baru. Layar yang datanya besar — inbox dan laporan —
 * tidak boleh mengikuti pola ini.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 */
export function useMaskingList(filter: MaskingFilter) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token, filter),
    queryFn: () => callAPI<MaskingListResponse>(listPath(filter), { token, portal }),
    enabled: token !== null && portal !== null,
    // Lebih pendek daripada master lain yang lima menit, dan itu disengaja: isi layar ini
    // adalah kewenangan yang aktif. Petugas yang baru saja mencabut hak seseorang harus
    // melihat keadaan terbaru, bukan daftar yang tertinggal beberapa menit.
    staleTime: 60 * 1000,
  })
}

/**
 * Hook daftar pilihan cabang.
 *
 * POOLDATA.BRANCH memuat 803 baris, sehingga daftarnya disaring kata kunci — bukan
 * dimuat seluruhnya. Ini yang menggantikan autocomplete cabang pada layar lama.
 */
export function useBranchOptions(keyword: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const clean = keyword.trim()

  return useQuery({
    queryKey: ['master-masking-cabang', portal, token, clean],
    queryFn: () =>
      callAPI<MaskingBranchListResponse>(
        clean === '' ? `${ROUTE}/cabang` : `${ROUTE}/cabang?kata_kunci=${encodeURIComponent(clean)}`,
        { token, portal },
      ),
    enabled: token !== null && portal !== null,
    // Daftar cabang nyaris tidak pernah berubah; ia data acuan milik sistem lain yang
    // hanya dibaca di sini.
    staleTime: 30 * 60 * 1000,
  })
}

type SaveFields = MaskingInput & {
  /** Kosong berarti menambah; terisi berarti mengubah baris dengan ID itu. */
  id?: string
}

/**
 * Hook simpan — menambah maupun mengubah.
 *
 * Keduanya disatukan karena form-nya memang satu: sistem lama pun memakai satu procedure
 * untuk keduanya, dan membedakannya lewat parameter `T_ACTION`. Perbedaannya di sini hanya
 * pada metode dan jalur, dan itu satu baris.
 *
 * STATUS AKTIF ikut dikirim, meniru form layar lama yang memang memuat isian "STATUS".
 * Yang menjaganya tidak berubah tanpa sengaja adalah LAYAR — tombol Ubah hanya muncul pada
 * baris aktif, sama seperti `ActionMaskingData_Sec` yang bersyarat `.STS_AKTF=='AKTIF'`.
 * Pada penambahan nilainya diabaikan server: baris baru selalu aktif.
 */
export function useSaveMasking() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, ...input }: SaveFields) =>
      callAPI<MaskingResponse>(id ? `${ROUTE}/${encodeURIComponent(id)}` : ROUTE, {
        metode: id ? 'PUT' : 'POST',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      // Seluruh daftar dimuat ulang dari server, bukan disunting di cache. Pada
      // penambahan, ID barunya hanya diketahui server; dan karena penyaring ikut di dalam
      // kunci, baris baru bisa saja tidak cocok dengan penyaring yang sedang aktif.
      void client.invalidateQueries({ queryKey: ['master-masking'] })
    },
  })
}

/**
 * Hook pengaktifan dan penonaktifan.
 *
 * Inilah pengganti tombol DELETE layar lama, yang tidak pernah membuang baris — ia hanya
 * mengubah STS_AKTF (`RDB List/DeleteMstProteksi_SQL-SQL.xml`).
 */
export function useSetMaskingStatus() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, aktif }: { id: string; aktif: boolean }) =>
      callAPI<MaskingResponse>(`${ROUTE}/${encodeURIComponent(id)}/status`, {
        metode: 'PUT',
        body: { aktif },
        token,
        portal,
      }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-masking'] })
    },
  })
}
