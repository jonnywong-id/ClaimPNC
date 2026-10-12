import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  PanelDocumentResponse,
  PanelImportResponse,
  PanelInput,
  PanelListResponse,
  PanelOptionsResponse,
  PanelResponse,
} from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/panel'
const ROUTE_OPTIONS = '/api/master/panel/pilihan'

/**
 * listKey menyertakan portal DAN token.
 *
 * Portal ikut karena itulah yang menentukan basis data mana yang menjawab (ADR-0030).
 * Tanpa itu, berpindah entitas akan menampilkan data entitas sebelumnya dari cache —
 * pengguna melihat angka yang masuk akal, dan tidak ada apa pun di layar yang menandakan
 * data itu milik badan hukum lain (R-20).
 *
 * Penyaring statusnya ikut pula: ketiga tab layar adalah tiga kombinasi penyaring atas
 * satu endpoint, dan tanpa status di kunci cache, berpindah tab akan menampilkan isi tab
 * sebelumnya sampai permintaan barunya tiba.
 */
function listKey(portal: string | null, token: string | null, status: string) {
  return ['master-panel', portal, token, status] as const
}

/**
 * optionsKey TIDAK menyertakan portal.
 *
 * Isinya konstanta yang ditanam di `Activity/SetLokasiSisiPanel-Act.xml`, bukan bacaan
 * basis data — sehingga jawabannya sama untuk keempat entitas. Menyertakan portal di
 * kuncinya berarti memuat ulang daftar yang sama setiap kali pengguna berpindah entitas.
 */
function optionsKey(token: string | null) {
  return ['master-panel-pilihan', token] as const
}

/**
 * Hook daftar Master Panel.
 *
 * Tidak dijalankan sebelum portal dipilih. Backend memang menolak permintaan tanpa portal
 * (TKT-F6-002), tetapi menembaknya lebih dulu hanya untuk menerima penolakan akan
 * menampilkan pesan galat pada layar yang sebenarnya belum siap dibuka.
 *
 * # Penyaring `cari` milik endpoint ini TIDAK dipakai layar
 *
 * Endpoint-nya menerimanya, dan ia memang dibuat sebagai pengganti `pyMaxRecords=500` yang
 * memotong daftar Pega tanpa satu pun cara mempersempitnya. Yang dipakai layar sekarang
 * adalah pencarian bawaan `DataTable`, sama seperti seluruh layar master lain — pencarian
 * kedua di kepala halaman hanya akan membingungkan.
 *
 * Perpindahan ke penyaringan sisi server dilakukan bersama paginasi sisi server
 * (`TKT-U2-001`), bukan sendirian.
 */
export function usePanelList(status: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: listKey(portal, token, status),
    queryFn: () =>
      callAPI<PanelListResponse>(`${ROUTE}?status=${encodeURIComponent(status)}`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null,
  })
}

/**
 * Hook daftar pilihan Lokasi dan Sisi.
 *
 * Ia TIDAK menunggu portal dipilih, karena endpoint-nya pun tidak menuntutnya: isinya
 * konstanta, bukan data entitas. Form karena itu dapat menggambar dropdown-nya bahkan
 * sebelum pengguna memilih entitas.
 */
export function usePanelOptions() {
  const token = useSession((state) => state.token)

  return useQuery({
    queryKey: optionsKey(token),
    queryFn: () => callAPI<PanelOptionsResponse>(ROUTE_OPTIONS, { token }),
    enabled: token !== null,
  })
}

/**
 * Hook penambahan.
 *
 * Seluruh daftar dibuang dari cache setelah berhasil, bukan hanya daftar tab yang sedang
 * dibuka: baris baru lahir berstatus menunggu, sehingga yang berubah adalah tab Waiting
 * Approval — tab yang justru TIDAK sedang dilihat pengguna saat ia menambah dari tab
 * Approve.
 */
export function useCreatePanel() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: PanelInput) =>
      callAPI<PanelResponse>(ROUTE, { metode: 'POST', body: input, token, portal }),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['master-panel'] })
    },
  })
}

/**
 * Hook penyimpanan.
 *
 * Seluruh daftar dibuang dari cache setelah berhasil. Menyimpan MEMINDAHKAN baris ke tab
 * Waiting Approval — `Activity/CNMUpdatePanelHE_act` menetapkan APPROVAL := "0" tanpa
 * syarat apa pun — sehingga daftar yang tidak sedang dilihat pun sudah basi.
 */
export function useSavePanel() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: PanelInput }) =>
      callAPI<PanelResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['master-panel'] })
    },
  })
}


/**
 * documentKey menyertakan portal, token, dan ID panelnya.
 *
 * Portal ikut dengan alasan yang sama seperti daftar: dokumen sebuah panel hidup di basis
 * data entitasnya, dan cache yang tidak membedakannya akan menampilkan dokumen badan hukum
 * lain tanpa satu pun tanda di layar (R-20).
 */
function documentKey(portal: string | null, token: string | null, id: string) {
  return ['master-panel-dokumen', portal, token, id] as const
}

/**
 * Hook pembaca dokumen satu panel.
 *
 * 404 adalah jawaban yang WAJAR di sini — panel yang belum punya dokumen menjawab
 * `dokumen_belum_ada` — sehingga kegagalannya tidak diperlakukan sebagai galat layar
 * melainkan sebagai "belum ada". Karena itu pula ia tidak dicoba ulang: mengulangi tiga
 * kali hanya memperlambat panel tanpa mengubah jawabannya.
 */
export function usePanelDocument(id: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: documentKey(portal, token, id ?? ''),
    queryFn: () =>
      callAPI<PanelDocumentResponse>(`${ROUTE}/${encodeURIComponent(id ?? '')}/dokumen`, {
        token,
        portal,
      }),
    enabled: token !== null && portal !== null && id !== null && id !== '',
    retry: false,
  })
}

/**
 * Hook unggah dokumen panel — jalurnya berujung di GCS.
 *
 * Backend meneruskannya ke modul `dokumenpenunjang` lewat adapter
 * `masterpanel/repo/dokumenlink`: konversi gambar, izin akses, lalu
 * `POST /api/v1/upload` ke layanan penyimpanan internal. Yang tersimpan di basis data kita
 * hanya metadata beserta `IMAGEID` (`D-16`).
 *
 * Badannya `FormData`, bukan JSON base64 — base64 membengkakkan muatan sekitar sepertiga,
 * dan `callAPI` sudah menangani FormData termasuk membiarkan peramban yang menulis
 * `Content-Type` beserta boundary-nya.
 *
 * Dua kunci cache dibuang sesudah berhasil, dan keduanya perlu: dokumen panel itu karena
 * isinya memang berganti, dan seluruh daftar karena `DOKUMENID` adalah kolom pada baris
 * panelnya sendiri.
 */
export function useUploadPanelDocument() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: ({ id, file, note }: { id: string; file: File; note: string }) => {
      const body = new FormData()
      body.append('berkas', file)
      if (note.trim() !== '') body.append('catatan', note.trim())
      return callAPI<PanelDocumentResponse>(`${ROUTE}/${encodeURIComponent(id)}/dokumen`, {
        metode: 'POST',
        body,
        token,
        portal,
      })
    },
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['master-panel-dokumen'] })
      client.invalidateQueries({ queryKey: ['master-panel'] })
    },
  })
}
/**
 * Hook unggah CSV master panel — padanan `Activity/PNCUploadMasterPanel_Act`.
 *
 * Kuncinya NAMA (`upper(trim(name))` pada `ValidationMasterPanel`), dan setiap baris
 * menentukan sendiri apakah ia menambah atau memperbarui. Baris hasil unggah masuk antrean
 * persetujuan — `TempPanel.APPROVAL := "0"` tanpa syarat.
 *
 * Berkasnya DIKIRIM SEKETIKA, berbeda dari unggah dokumen yang menunggu Simpan: ia membawa
 * kuncinya sendiri, dan tidak ada satu panel pun yang sedang disunting saat berkas berisi
 * tiga ratus baris diunggah.
 */
export function useImportPanelCSV() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (file: File) => {
      const body = new FormData()
      body.append('berkas', file)
      return callAPI<PanelImportResponse>(`${ROUTE}/unggah-csv`, {
        metode: 'POST',
        body,
        token,
        portal,
      })
    },
    onSuccess: () => {
      // SELURUH daftar dibuang, bukan tab yang sedang dibuka saja: baris hasil unggah masuk
      // antrean persetujuan, sehingga yang berubah justru tab Waiting Approval — tab yang
      // TIDAK sedang dilihat pengguna saat ia mengunggah dari tab Approve.
      client.invalidateQueries({ queryKey: ['master-panel'] })
    },
  })
}

/**
 * Hook unggah CSV lokasi panel — padanan `Activity/PNCUploadLokasiSisiPanel_Act`.
 *
 * Ia MENAMBAH lokasi, tidak mengganti: Pega memakai `LOKASI(<APPEND>)` dan menyalin seluruh
 * isian induk dari panel yang ditemukan, sehingga lokasi lama tetap utuh. Tautan dokumennya
 * pun dipertahankan (`CoverID` disalin), berbeda dari jalur master.
 */
export function useImportLocationCSV() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (file: File) => {
      const body = new FormData()
      body.append('berkas', file)
      return callAPI<PanelImportResponse>(`${ROUTE}/unggah-csv-lokasi`, {
        metode: 'POST',
        body,
        token,
        portal,
      })
    },
    onSuccess: () => {
      client.invalidateQueries({ queryKey: ['master-panel'] })
      client.invalidateQueries({ queryKey: ['master-panel-dokumen'] })
    },
  })
}
/*
  Ketiga unggahan layar Pega kini lengkap di sini: dokumen, master panel, dan lokasi panel.

  Aturan ketiganya diturunkan dari activity yang ADA di export — `UploadDocument`,
  `PNCUploadMasterPanel_Act`, dan `PNCUploadLokasiSisiPanel_Act` — bukan dari Flow Action
  pemanggilnya, yang memang hilang tetapi hanya memuat pemilih berkas.

  Satu perilaku Pega SENGAJA tidak ditiru, dan itu keputusan terbuka: unggah CSV master
  menautkan berkas CSV-nya sendiri sebagai dokumen ke SETIAP baris yang disentuhnya
  (`TempPanel.CoverID := tempDocumentPanel.CaseID`), sehingga baris yang sudah punya dokumen
  kehilangan tautannya. Jalur lokasi justru MEMPERTAHANKAN tautan itu. Asimetrinya dicatat
  di docs/keputusan-implementasi.md; Work Owner yang memutuskan apakah ia ditiru.
*/

/*
  TIDAK ADA hook keputusan di sini, dan itu disengaja.

  Layar Master Panel di Pega tidak punya persetujuan sama sekali: ketiga tabnya hanya
  punya Simpan, Ubah, dan Upload Document, serta nol `pySelected`. Approve dan Reject ada
  di `Section/ApprovalMasterPanelHE`, yang dimuat layar Inbox Manager.

  Endpoint `POST /api/master/panel/keputusan` TETAP ADA di backend — ia padanan
  `Activity/SetApprovalAllMaster` dengan `Param.TIPE2 = "M_PANEL_HE"`, dan Inbox Manager
  akan memakainya begitu dibangun. Yang dihapus hanyalah pemanggilnya dari layar ini.
*/
