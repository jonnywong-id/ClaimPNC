import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  ReceiveTKACompleteRequest,
  ReceiveTKACompleteResponse,
  ReceiveTKAInboxResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/inbox/receive-tka'
const COMPLETE_ROUTE = '/api/inbox/receive-tka/kelengkapan-dokumen'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (`ADR-0030`).
 * Tanpa itu, berpindah entitas akan menampilkan daftar entitas sebelumnya dari cache —
 * petugas melihat pekerjaan yang masuk akal, dan tidak ada apa pun di layar yang menandakan
 * pekerjaan itu milik badan hukum lain (`R-20`).
 *
 * Pada modul ini akibatnya melampaui kebocoran baca: baris yang salah portal dapat DIISI,
 * dan pengisiannya mengubah tanggal pada klaim badan hukum lain.
 *
 * TANPA kata kunci di dalam kunci cache: penyaringan dikerjakan PERAMBAN atas baris yang
 * sudah di tangan, sehingga mengetik tidak menembak server dan tidak melahirkan entri cache
 * baru per huruf.
 */
function listKey(portal: string | null, token: string | null) {
  return ['inbox-receive-tka', portal, token] as const
}

/**
 * Hook daftar Inbox Receive TKA.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (`TKT-F6-002`), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 *
 * # Penyaring `cari` milik endpoint ini TIDAK dipakai layar
 *
 * Endpoint-nya menerimanya, dan ia tetap disediakan supaya perpindahan ke penyaringan sisi
 * server kelak (`TKT-U2-001`) tidak menuntut perubahan kontrak. Yang dipakai layar sekarang
 * adalah pencarian bawaan `DataTable` — sama seperti Inbox Investigator dan seluruh layar
 * master, dan sesuai pilihan Work Owner bahwa penyaringan dikerjakan peramban seperti grid
 * Pega (`pyGridFiltering = true`).
 *
 * # Kenapa TIDAK di-refetch berkala
 *
 * Daftar ini berubah tanpa tindakan pengguna — petugas lain mengisi tanggal, dan proses
 * pengisi tabelnya menambahkan baris. Penyegaran otomatis karena itu menggoda, tetapi ia
 * memindahkan baris di bawah kursor orang yang sedang mengetik tanggal. Yang dipilih adalah
 * tombol Refresh yang ditekan sendiri, sama seperti tombol Refresh pada harness lamanya.
 */
export function useReceiveTKAInbox() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token),
    queryFn: () => callAPI<ReceiveTKAInboxResponse>(ROUTE, { token, portal }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook tombol **Submit** — mengisi tanggal kelengkapan dokumen satu baris.
 *
 * Ia padanan `Activity/SubmitTanggalLengkapTKA-Act.xml`.
 *
 * # Daftar disegarkan setelah berhasil, bukan diubah di tempat
 *
 * Baris yang terisi HILANG dari inbox, dan yang menentukan hilangnya adalah penyaring di
 * server (`TGL_DOC_LENGKAP IS NULL`) — bukan layar. Membuang barisnya dari cache secara
 * manual akan membuat layar menebak hasil yang seharusnya dijawab server, dan tebakannya
 * dapat berbeda: baris lain mungkin ikut berubah, atau daftar yang tadi terpotong kini
 * memuat satu baris tambahan yang sebelumnya tidak terkirim.
 *
 * # Kegagalan TIDAK menyegarkan daftar
 *
 * Penyegaran hanya pada keberhasilan. Pada penolakan, barisnya memang masih di sana dan
 * tanggal yang sudah diketik pengguna tidak boleh hilang dari layar.
 */
export function useCompleteReceiveTKA() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: (body: ReceiveTKACompleteRequest) =>
      callAPI<ReceiveTKACompleteResponse>(COMPLETE_ROUTE, {
        metode: 'POST',
        body,
        token,
        portal,
      }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: listKey(portal, token) })
    },
  })
}
