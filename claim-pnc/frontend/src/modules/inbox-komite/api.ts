import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  KomiteCaseResponse,
  KomiteDecisionKind,
  KomiteInboxKind,
  KomiteInboxListResponse,
} from '@/api/types'
import { useSession } from '@/app/session'

const PATH = '/api/komite/inbox'

/** Penyaring daftar. Kosong berarti tanpa penyaring. */
export type InboxFilter = {
  kind: KomiteInboxKind
  search?: string
  /** `YYYY-MM-DD`; kosong berarti tanpa batas. */
  from?: string
  to?: string
  offset?: number
}

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * Penyaring ikut menjadi bagian kunci: kotak "outstanding" dan kotak "ditolak" adalah dua
 * hasil berbeda, dan menyimpannya di bawah satu kunci akan membuat perpindahan tab
 * menampilkan isi tab sebelumnya.
 *
 * Token ikut menjadi bagian kunci karena inbox adalah daftar pekerjaan SESEORANG — cache
 * yang bertahan melewati pergantian pengguna akan menampilkan pekerjaan orang sebelumnya.
 */
const keys = {
  all: (token: string | null) => ['inbox-komite', token] as const,
  detail: (token: string | null, caseID: string) =>
    ['inbox-komite', token, 'rincian', caseID] as const,
  list: (token: string | null, f: InboxFilter) =>
    [
      'inbox-komite',
      token,
      f.kind,
      f.search ?? '',
      f.from ?? '',
      f.to ?? '',
      f.offset ?? 0,
    ] as const,
}

function buildPath(f: InboxFilter): string {
  const params = new URLSearchParams()
  params.set('kotak', f.kind)
  if (f.search?.trim()) params.set('cari', f.search.trim())
  if (f.from) params.set('dari', f.from)
  if (f.to) params.set('sampai', f.to)
  if (f.offset) params.set('lewati', String(f.offset))

  return `${PATH}?${params.toString()}`
}

/**
 * Hook daftar Inbox Komite.
 *
 * Menggantikan ketiga kueri inbox sistem lama sekaligus — `GetKomitePAOutstanding`,
 * `GetKomitePAditerima`, dan `ShowKomiteTerimaTolakNonMBU` — yang di sana menjadi tiga
 * rule terpisah karena penyaringnya dirangkai ke dalam teks SQL.
 *
 * # Kenapa penyaringan dikerjakan di SERVER, bukan di peramban
 *
 * Kasus komite tumbuh bersama jumlah klaim, dan klaim baru masuk RIBUAN PER BULAN
 * (`D-10`). Menyaringnya di peramban berarti mengirim seluruh antrean komite ke setiap
 * layar yang dibuka — dan menyaring satu halaman dari sepuluh bukan penyaringan: pengguna
 * mencari sesuatu yang ada di halaman tiga lalu diberi tahu bahwa ia tidak ada.
 */
export function useInboxList(filter: InboxFilter) {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: keys.list(token, filter),
    queryFn: () => callAPI<KomiteInboxListResponse>(buildPath(filter), { token }),
    enabled: token !== null,

    // Hasil tab sebelumnya ditahan selama tab baru dimuat, alih-alih layar berkedip
    // menjadi kosong lalu terisi lagi. Lencana jumlah tetap terbaca selama perpindahan.
    placeholderData: (previous) => previous,

    // Kasus komite baru dapat masuk kapan saja dari petugas lain, dan yang menunggu di
    // sini punya tenggat. Cache-nya karena itu pendek — berbeda dari master yang nyaris
    // tidak berubah dalam satu sesi kerja.
    staleTime: 30 * 1000,
  })
}

/**
 * Hook pencatatan keputusan komite.
 *
 * Menggantikan `KomitePost_Adjustment`, `KomitePost_Reject`, dan `KomitePost_LiableKlaim`.
 *
 * Jenjang, waktu, dan identitas pemutus TIDAK dikirim: ketiganya milik server. Klien yang
 * boleh menyebut jenjangnya dapat menyetujui jenjang yang bukan gilirannya.
 */
export function useDecide() {
  const token = useSession((state) => state.token)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({
      caseID,
      decision,
      note,
    }: {
      caseID: string
      decision: KomiteDecisionKind
      note: string
    }) =>
      callAPI<KomiteCaseResponse>(`${PATH}/${encodeURIComponent(caseID)}/keputusan`, {
        metode: 'POST',
        body: { keputusan: decision, catatan: note },
        token,
      }),
    onSuccess: () => {
      // SELURUH daftar dimuat ulang, bukan hanya kotak yang sedang terbuka: satu
      // keputusan memindahkan kasus dari satu kotak ke kotak lain dan mengubah lencana
      // keduanya. Menyunting cache satu kotak akan membuat angkanya tidak cocok dengan
      // isi kotak lain.
      void client.invalidateQueries({ queryKey: keys.all(token) })
    },
  })
}


/**
 * Hook rincian satu kasus komite.
 *
 * # Apa yang ia gantikan
 *
 * Menekan nomor case di layar lama menjalankan `Activity/SetAssignmentKomite-Act.xml`,
 * yang merangkai kunci assignment `"ASSIGN-WORKLIST " + inskey + "!Komite_Flow"`,
 * membukanya dengan `OBJ-OPEN-BY-HANDLE`, lalu merender flow action `ViewTransferDtl`.
 *
 * Ketiga langkah itu menjadi SATU permintaan di sini. Perangkaian kunci tidak ikut dibawa:
 * ia bentuk kunci internal Pega yang `D-22` larang bocor ke data bisnis, dan yang
 * dibutuhkan server hanyalah nomor case-nya.
 *
 * # "Open Assignment" menjadi pemeriksaan kepemilikan, bukan penguncian
 *
 * `OBJ-OPEN-BY-HANDLE` di Pega membuka SEKALIGUS mengunci objek kerjanya. Di sini tidak ada
 * yang dikunci, dan itu disengaja: layar ini tidak menulis apa pun, dan kunci yang tidak
 * pernah dilepas adalah cacat yang jauh lebih mahal daripada yang dicegahnya. Yang tersisa
 * dari "membuka assignment" adalah pertanyaannya yang sebenarnya — apakah kasus ini milik
 * pemanggil — dan itu dijawab server lewat `BelongsTo`.
 */
export function useKomiteCase(caseID: string) {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: keys.detail(token, caseID),
    queryFn: () =>
      callAPI<KomiteCaseResponse>(`${PATH}/${encodeURIComponent(caseID)}`, { token }),
    enabled: token !== null && caseID !== '',

    // Sama pendeknya dengan daftarnya: keadaan sebuah kasus komite dapat berubah dari
    // Pega kapan saja selama masa paralel.
    staleTime: 30 * 1000,
  })
}
