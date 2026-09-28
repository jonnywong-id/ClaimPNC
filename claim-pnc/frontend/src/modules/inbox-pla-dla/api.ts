import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI, downloadAPI } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  BalasResponse,
  DetailResponse,
  DokumenResponse,
  ListResponse,
  MetadataResponse,
  RingkasResponse,
  XOLResponse,
} from './types'

const PATH = '/api/inbox-pla-dla'

/**
 * Kunci cache dikumpulkan di satu tempat.
 *
 * Portal ikut menjadi bagian kunci, dan di layar ini taruhannya paling besar di antara
 * seluruh modul: yang membaca jawabannya adalah PIHAK LUAR. Cache yang tertukar antar
 * portal berarti menampilkan klaim satu badan hukum kepada mitra badan hukum lain
 * (`R-20`).
 */
const keys = {
  metadata: (portal: string | null, token: string | null) =>
    ['inbox-pla-dla', 'daftar', portal, token] as const,

  list: (
    portal: string | null,
    token: string | null,
    tab: string,
    page: number,
    search: string,
  ) => ['inbox-pla-dla', 'baris', portal, token, tab, page, search] as const,

  counts: (
    portal: string | null,
    token: string | null,
    tab: string,
    search: string,
  ) => ['inbox-pla-dla', 'ringkas', portal, token, tab, search] as const,

  xol: (portal: string | null, token: string | null) =>
    ['inbox-pla-dla', 'xol', portal, token] as const,

  detail: (portal: string | null, token: string | null, kunci: string) =>
    ['inbox-pla-dla', 'rincian', portal, token, kunci] as const,

  dokumen: (
    portal: string | null,
    token: string | null,
    kunci: string,
    nomor: string,
    jenis: string,
  ) =>
    ['inbox-pla-dla', 'dokumen', portal, token, kunci, nomor, jenis] as const,
}

/** Hook keterangan layar — daftar tab, kolomnya, dan selisih terencana. */
export function useReasMetadata() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.metadata(portal, token),
    queryFn: () => callAPI<MetadataResponse>(`${PATH}/daftar`, { token, portal }),
    enabled: token !== null && portal !== null,
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/** Isian penyaring yang dikirim ke server. */
export type ParameterDaftar = {
  tab: string
  page: number
  cari: string
}

/**
 * Hook isi satu daftar.
 *
 * Paginasi dan pencarian dikerjakan SERVER. Di layar ini itu berbeda dari Pega, dan
 * perbedaannya disengaja: ketiga kueri Pega tidak berpaginasi sama sekali — mereka memuat
 * SELURUH baris yang cocok ke memori sekaligus.
 */
export function useReasList(parameter: ParameterDaftar, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.list(
      portal,
      token,
      parameter.tab,
      parameter.page,
      parameter.cari,
    ),
    queryFn: () =>
      callAPI<ListResponse>(alamatDaftar(parameter), { token, portal }),
    enabled: enabled && token !== null && portal !== null,
    placeholderData: (previous) => previous,
    staleTime: 15 * 1000,
  })
}

/**
 * Hook tabel ringkas "Status / Jumlah".
 *
 * # Kenapa ia permintaan TERSENDIRI
 *
 * Karena isinya tidak berubah saat pengguna berpindah HALAMAN — hanya saat ia berpindah
 * tab atau mengubah pencarian. Menggabungkannya ke dalam jawaban daftar akan menjalankan
 * hitungannya setiap kali halaman berganti, dan hitungannya menyapu tabel klaim yang
 * sama.
 *
 * Penyaringnya IKUT, sedangkan halamannya tidak. Ringkasan yang mengabaikan pencarian
 * akan menampilkan angka yang lebih besar daripada tabel di bawahnya — dan angka yang
 * tidak cocok adalah hal pertama yang dilaporkan pengguna sebagai kerusakan.
 */
export function useReasCounts(parameter: ParameterDaftar, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.counts(portal, token, parameter.tab, parameter.cari),
    queryFn: () =>
      callAPI<RingkasResponse>(alamatRingkas(parameter), { token, portal }),
    enabled: enabled && token !== null && portal !== null,
    staleTime: 60 * 1000,
  })
}

/**
 * Hook grid "DATA PLA DLA XOL KLAIM".
 *
 * Ia TIDAK menerima tab maupun kata kunci: gridnya tidak disaring keduanya, sehingga
 * isinya sama berapa pun tab yang sedang dibuka. Itulah yang membuatnya boleh disimpan
 * jauh lebih lama daripada daftarnya.
 */
export function useReasXOL(enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.xol(portal, token),
    queryFn: () => callAPI<XOLResponse>(`${PATH}/xol`, { token, portal }),
    enabled: enabled && token !== null && portal !== null,
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Hook tombol "Export To Excel".
 *
 * Berkasnya diambil lewat `downloadAPI` karena endpoint ini menuntut header
 * `Authorization` dan `X-Portal`, dan `<a href>` tidak membawa keduanya.
 */
export function useEksporReas() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (parameter: ParameterDaftar) =>
      downloadAPI(
        alamatEkspor(parameter),
        `inbox-pla-dla-${parameter.tab || 'pla'}.csv`,
        { token, portal },
      ),
  })
}

/* ==========================================================================
 * LAYAR RINCIAN — tombol "Detail Claim"
 * ========================================================================== */

/**
 * Hook isi layar rincian satu klaim.
 *
 * Keempat bagiannya — kepala klaim, grid PLA, grid DLA, riwayat komunikasi — datang dalam
 * SATU permintaan. `10-API-STRATEGY.md` §1 menuntutnya: satu layar sebaiknya dilayani satu
 * permintaan, justru supaya layar baru tidak terasa lebih lambat daripada Pega yang
 * merender seluruh layar sekaligus.
 */
export function useRincianKlaim(kunci: string, enabled: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.detail(portal, token, kunci),
    queryFn: () =>
      callAPI<DetailResponse>(`${PATH}/klaim/${encodeURIComponent(kunci)}`, {
        token,
        portal,
      }),
    enabled: enabled && kunci !== '' && token !== null && portal !== null,
    staleTime: 15 * 1000,
  })
}

/**
 * Hook dokumen satu nomor pemberitahuan — tombol "Dokumen" pada baris PLA/DLA.
 *
 * Ia TIDAK ikut permintaan rincian, dan itu disengaja: kuncinya adalah nomor
 * pemberitahuan, bukan klaimnya. Satu klaim dapat punya banyak PLA dan banyak DLA, dan
 * mengambil dokumen seluruhnya di muka berarti membaca isi tabel lampiran untuk baris yang
 * mungkin tidak pernah dibuka siapa pun.
 */
export function useDokumenPemberitahuan(
  kunci: string,
  nomor: string,
  jenis: string,
) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.dokumen(portal, token, kunci, nomor, jenis),
    queryFn: () => {
      const query = new URLSearchParams({ nomor, jenis })
      return callAPI<DokumenResponse>(
        `${PATH}/klaim/${encodeURIComponent(kunci)}/dokumen?${query.toString()}`,
        { token, portal },
      )
    },
    enabled:
      kunci !== '' &&
      nomor !== '' &&
      jenis !== '' &&
      token !== null &&
      portal !== null,
    staleTime: 60 * 1000,
  })
}

/**
 * Hook unduh satu dokumen.
 *
 * Berkasnya diambil lewat `downloadAPI` karena endpoint ini menuntut header
 * `Authorization` dan `X-Portal`, dan `<a href>` tidak membawa keduanya.
 */
export function useUnduhDokumen(kunci: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (dokumen: { id: string; nama: string }) =>
      downloadAPI(
        `${PATH}/klaim/${encodeURIComponent(kunci)}/dokumen/` +
          encodeURIComponent(dokumen.id),
        dokumen.nama || 'dokumen',
        { token, portal },
      ),
  })
}

/**
 * Hook tombol **"Balas Pesan"** — satu-satunya operasi TULIS di modul ini.
 *
 * Rinciannya dimuat ULANG setelah balasannya tersimpan, bukan ditambal di sisi layar.
 * Yang berubah bukan hanya isi balasannya: percakapannya berpindah menjadi "sudah
 * dijawab", tombol balasnya hilang, dan pada daftar induk barisnya berpindah tab. Menebak
 * seluruh akibat itu di layar berarti menyalin aturan peladen ke tempat kedua.
 */
export function useBalasKomunikasi(kunci: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (isian: { percakapan: string; balasan: string }) =>
      callAPI<BalasResponse>(
        `${PATH}/klaim/${encodeURIComponent(kunci)}/komunikasi/balas`,
        {
          token,
          portal,
          metode: 'POST',
          body: isian,
        },
      ),

    onSuccess: () => {
      void client.invalidateQueries({
        queryKey: keys.detail(portal, token, kunci),
      })
      // Daftar induk ikut berubah: percakapan yang dijawab berpindah dari tab
      // "Terkirim — Belum Dijawab" ke "Terkirim — Sudah Dijawab".
      void client.invalidateQueries({ queryKey: ['inbox-pla-dla', 'baris'] })
      void client.invalidateQueries({ queryKey: ['inbox-pla-dla', 'ringkas'] })
    },
  })
}

/**
 * Hook kedua tombol yang belum dibangun — "Download ALL PLA" dan "Download ALL DLA".
 *
 * Ia SELALU gagal; yang dibacanya adalah alasannya. Tombolnya tetap digambar karena tombol
 * yang mati tidak menjelaskan apa pun, dan alasan yang tidak sampai ke pengguna sama saja
 * dengan tidak ada alasan.
 */
export function useTindakanBelumTersedia() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useMutation({
    mutationFn: (tindakan: string) => {
      const query = new URLSearchParams({ tindakan })
      return callAPI<unknown>(`${PATH}/tindakan?${query.toString()}`, {
        token,
        portal,
        metode: 'POST',
      })
    },
  })
}

/** alamatDaftar menyusun alamat permintaan daftar beserta penyaringnya. */
function alamatDaftar(parameter: ParameterDaftar): string {
  const query = parameterPenyaring(parameter)
  if (parameter.page > 1) query.set('halaman', String(parameter.page))

  const teks = query.toString()
  return teks ? `${PATH}?${teks}` : PATH
}

/** alamatRingkas menyusun alamat tabel ringkas — tanpa halaman. */
function alamatRingkas(parameter: ParameterDaftar): string {
  const teks = parameterPenyaring(parameter).toString()
  return teks ? `${PATH}/ringkas?${teks}` : `${PATH}/ringkas`
}

/**
 * alamatEkspor menyusun alamat unduhan.
 *
 * Halaman sengaja TIDAK ikut: ekspor menyalin SELURUH daftar yang cocok, bukan halaman
 * yang sedang tampil.
 */
function alamatEkspor(parameter: ParameterDaftar): string {
  const teks = parameterPenyaring(parameter).toString()
  return teks ? `${PATH}/ekspor?${teks}` : `${PATH}/ekspor`
}

/** parameterPenyaring menyusun bagian alamat yang sama bagi ketiga permintaan. */
function parameterPenyaring(parameter: ParameterDaftar): URLSearchParams {
  const query = new URLSearchParams()
  if (parameter.tab) query.set('daftar', parameter.tab)
  if (parameter.cari) query.set('cari', parameter.cari)
  return query
}
