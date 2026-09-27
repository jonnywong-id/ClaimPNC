import { useQuery } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
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
