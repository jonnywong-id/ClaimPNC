import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  Surveyor,
  SurveyorDecisionInput,
  SurveyorInput,
  SurveyorListResponse,
  SurveyorResponse,
  SurveyorStatus,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/surveyor'

/** Saringan daftar, cerminan mastersurveyors.Filter di backend. */
export type SurveyorFilter = {
  status?: SurveyorStatus
  nama?: string
  login?: string
  kode_tipe?: string
  /** Membatasi ke antrean komite pengguna yang sedang masuk. */
  antrean_saya?: boolean
}

/**
 * listKey menyertakan portal, token, DAN saringan.
 *
 * Portal ikut di dalam kunci karena itulah yang menentukan basis data mana yang menjawab
 * (ADR-0030). Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari
 * cache — pengguna melihat daftar yang masuk akal, dan tidak ada apa pun di layar yang
 * menandakan data itu milik badan hukum lain (R-20).
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama. Saringan ikut karena kelima tab memakai saringan berbeda; tanpa itu
 * berpindah tab akan menampilkan isi tab sebelumnya.
 */
function listKey(portal: string | null, token: string | null, filter: SurveyorFilter) {
  return ['master-surveyor', portal, token, filter] as const
}

function detailKey(portal: string | null, token: string | null, id: string) {
  return ['master-surveyor', 'detail', portal, token, id] as const
}

/** Menyusun query string hanya dari saringan yang benar-benar terisi. */
function queryOf(filter: SurveyorFilter): string {
  const params = new URLSearchParams()
  if (filter.status) params.set('status', filter.status)
  if (filter.nama?.trim()) params.set('nama', filter.nama.trim())
  if (filter.login?.trim()) params.set('login', filter.login.trim())
  if (filter.kode_tipe?.trim()) params.set('kode_tipe', filter.kode_tipe.trim())
  if (filter.antrean_saya) params.set('antrean_saya', '1')

  const text = params.toString()
  return text ? `?${text}` : ''
}

/**
 * Hook daftar Master Surveyors.
 *
 * Menggantikan Report Definition `BrowseVDSurveyors_RD` yang mengisi keempat grid layar
 * `DetailSurveyorsInbox`. Keempatnya dilayani SATU endpoint yang dibedakan saringan,
 * bukan empat endpoint — yang berbeda hanyalah penyaringnya.
 *
 * Berbeda dari Master Tipe Surveyors yang memuat seluruh barisnya sekaligus, daftar ini
 * DIPAGINASI SERVER: isinya 43 baris pada portal ASM hari ini dan akan terus bertambah,
 * sementara tipe surveyor adalah golongan yang nyaris tidak pernah bertambah.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 */
export function useSurveyorList(filter: SurveyorFilter) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token, filter),
    queryFn: () => callAPI<SurveyorListResponse>(`${ROUTE}${queryOf(filter)}`, { token, portal }),
    enabled: token !== null && portal !== null,
    // Lebih pendek daripada Master Tipe Surveyors (lima menit): daftar ini berubah setiap
    // kali komite memutuskan, dan menampilkan antrean yang sudah kedaluwarsa membuat dua
    // komite mengerjakan baris yang sama.
    staleTime: 60 * 1000,
  })
}

/** Hook satu surveyor, dipakai form ubah. */
export function useSurveyor(id: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: detailKey(portal, token, id ?? ''),
    queryFn: () =>
      callAPI<SurveyorResponse>(`${ROUTE}/${encodeURIComponent(id ?? '')}`, { token, portal }),
    enabled: token !== null && portal !== null && id !== null && id !== '',
  })
}

type SaveFields = SurveyorInput & {
  /** Kosong berarti mengajukan; terisi berarti mengubah surveyor dengan ID itu. */
  id?: string
}

/**
 * Hook simpan — mengajukan maupun mengubah.
 *
 * Keduanya disatukan karena form-nya memang satu: sistem lama pun memakai satu halaman
 * `TempDetailSurveyors` untuk keduanya, dan membedakannya lewat `pyLabel` yang berisi
 * "Update" saat menyunting (`Activity/SetDetailSurveryorsValue_act-Act.xml`).
 *
 * ID TIDAK PERNAH dikirim di badan permintaan. Pada pengajuan ia dibuat server; pada
 * pengubahan ia berada di jalur URL.
 */
export function useSaveSurveyor() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, ...body }: SaveFields) =>
      callAPI<SurveyorResponse>(id ? `${ROUTE}/${encodeURIComponent(id)}` : ROUTE, {
        metode: id ? 'PUT' : 'POST',
        body,
        token,
        portal,
      }),
    onSuccess: () => invalidateAll(client),
  })
}

/**
 * Hook keputusan komite.
 *
 * Jalur terpisah dari simpan, bukan bagian darinya, dan itu mengikuti bentuk kontraknya:
 * keputusan komite adalah PERISTIWA yang punya invarian sendiri dan wajib tercatat —
 * bukan pembaruan field biasa.
 *
 * Server menolak keputusan dari orang yang bukan komite yang ditunjuk (403), dan menolak
 * keputusan kedua atas baris yang sama (409). Layar tidak perlu menduga keduanya.
 */
export function useDecideSurveyor() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, ...body }: SurveyorDecisionInput & { id: string }) =>
      callAPI<SurveyorResponse>(`${ROUTE}/${encodeURIComponent(id)}/keputusan`, {
        metode: 'POST',
        body,
        token,
        portal,
      }),
    onSuccess: () => invalidateAll(client),
  })
}

/**
 * Memuat ulang SELURUH daftar surveyor, bukan hanya yang sedang dibuka.
 *
 * Satu keputusan komite memindahkan satu baris dari tab "Menunggu" ke tab "Disetujui"
 * sekaligus mengeluarkannya dari "Antrean Komite Saya". Memuat ulang hanya tab yang
 * sedang terbuka akan meninggalkan ketiga tab lain menampilkan keadaan yang sudah tidak
 * benar sampai pengguna menekan muat ulang.
 *
 * Kuncinya sengaja hanya berawalan `'master-surveyor'` — tanpa portal dan token —
 * sehingga ia mencakup seluruh kombinasi saringan sekaligus. Portal yang keliru tidak
 * mungkin ikut terbawa: setiap kunci memuat portalnya sendiri, dan yang dibatalkan hanya
 * ditandai usang, bukan ditukar isinya.
 */
function invalidateAll(client: ReturnType<typeof useQueryClient>) {
  void client.invalidateQueries({ queryKey: ['master-surveyor'] })
}

/** Menyatakan satu surveyor masih menunggu keputusan komite. */
export function isAwaitingDecision(surveyor: Surveyor): boolean {
  return surveyor.status === '0'
}
