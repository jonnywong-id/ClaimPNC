import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  SurveyorTypeInput,
  SurveyorTypeListResponse,
  SurveyorTypeResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/tipe-surveyor'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut di dalam kunci karena itulah yang menentukan basis data mana yang menjawab
 * (ADR-0030). Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari
 * cache — pengguna melihat daftar yang masuk akal, dan tidak ada apa pun di layar yang
 * menandakan data itu milik badan hukum lain (R-20).
 *
 * Token ikut supaya cache pengguna sebelumnya tidak terwarisi pengguna berikutnya di
 * peramban yang sama.
 */
function listKey(portal: string | null, token: string | null) {
  return ['master-tipe-surveyor', portal, token] as const
}

/**
 * Hook daftar Master Tipe Surveyors.
 *
 * Menggantikan Report Definition `BrowseVMSurveyors_RD` yang mengisi grid layar
 * `SurveyorsInbox`.
 *
 * Seluruh baris dimuat sekaligus, tanpa paginasi server. Itu keputusan yang diambil
 * dengan angka: isinya EMPAT baris, dan tipe surveyor adalah golongan yang nyaris tidak
 * pernah bertambah. Layar yang datanya besar — inbox dan laporan — tidak boleh mengikuti
 * pola ini.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka — layar yang
 * menuntun pengguna memilih portal lebih berguna daripada pesan galat.
 */
export function useSurveyorTypeList() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<SurveyorTypeListResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
    // Master nyaris tidak pernah berubah dalam satu sesi kerja. Lima menit menahan
    // pemuatan ulang yang tidak perlu, sementara tombol Refresh tetap tersedia bagi
    // pengguna yang tahu datanya baru saja diubah orang lain.
    staleTime: 5 * 60 * 1000,
  })
}

type SaveFields = SurveyorTypeInput & {
  /** Kosong berarti menambah; terisi berarti mengubah tipe dengan kode itu. */
  kode?: string
}

/**
 * Hook simpan — menambah maupun mengubah.
 *
 * Keduanya disatukan karena form-nya memang satu: sistem lama pun memakai satu halaman
 * `TempSurveyors` untuk keduanya, dan membedakannya dengan ada-tidaknya `M_SURVEY_ID`
 * (`Activity/CNMInsertSurveyors_act-Act.xml` mengirim sentinel `"UnknownID"` bila kosong).
 * Perbedaannya hanya pada metode dan jalur, dan itu satu baris.
 *
 * KODE TIDAK PERNAH dikirim di badan permintaan. Pada penambahan ia dibuat server; pada
 * pengubahan ia berada di jalur URL. Server bahkan menolak badan yang memuat `kode`.
 */
export function useSaveSurveyorType() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ kode, deskripsi }: SaveFields) =>
      callAPI<SurveyorTypeResponse>(kode ? `${ROUTE}/${encodeURIComponent(kode)}` : ROUTE, {
        metode: kode ? 'PUT' : 'POST',
        body: { deskripsi },
        token,
        portal,
      }),
    onSuccess: () => {
      // Daftar dimuat ulang dari server, bukan disunting di cache. Pada penambahan, kode
      // barunya hanya diketahui server — menebaknya di klien akan menampilkan kode yang
      // salah sampai muat ulang berikutnya.
      void client.invalidateQueries({ queryKey: listKey(portal, token) })
    },
  })
}
