import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { APIError, callAPI, simpanBerkas, unduhBerkas } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  HasilPermintaanKlaim,
  IzinPermintaanKlaim,
  JenisPermintaanKlaim,
  PICResponse,
  PenyaringDashboard,
  RincianKlaim,
  PermintaanTransfer,
  PenyaringResponse,
  RingkasanResponse,
  TelusurResponse,
  TampunganResponse,
  TransferResponse,
  Tile,
} from './types'

const PATH = '/api/dashboard-claim'

/** Banyaknya baris per halaman. Backend menolak permintaan di atas 100. */
export const PAGE_SIZE = 25

/**
 * Kunci cache dikumpulkan di satu tempat supaya invalidasi tidak salah sasaran.
 *
 * PORTAL ikut menjadi bagian kunci, dan itu bukan kerapian: dua portal adalah dua badan
 * hukum dengan basis data berbeda (`ADR-0030`). Menyimpan keduanya di bawah satu kunci akan
 * membuat perpindahan portal menampilkan angka entitas sebelumnya — kebocoran yang tampil
 * sebagai layar normal, persis `R-20`.
 *
 * Pada layar INI akibatnya khas: yang salah bukan satu baris melainkan seluruh ringkasannya,
 * dan angka yang salah entitas tidak terlihat keliru dengan cara lain apa pun.
 */
const keys = {
  penyaring: (portal: string | null, token: string | null) =>
    ['dashboard-claim', 'penyaring', portal, token] as const,

  /**
   * Kunci ringkasan memuat SELURUH penyaring, bukan hanya lini bisnis dan kotak cari.
   *
   * `buildParams` mengirim kelima isian panel ke endpoint ringkasan juga, sehingga angka
   * pada kartu memang disaring. Bila kuncinya tidak memuatnya, mengubah Nopolis akan
   * menembak ulang daftarnya tetapi TIDAK kartunya — dan pengguna membaca satu angka di
   * kartu lalu menemukan jumlah baris yang lain di bawahnya.
   *
   * Itu persis penyimpangan yang dijaga `TestOutstandingListAndCountShareTheSameWhere` di
   * sisi SQL. Dijaga di sana tetapi dilanggar di lapisan cache tetap menghasilkan layar
   * yang salah.
   */
  ringkasan: (portal: string | null, token: string | null, f: PenyaringDashboard) =>
    [
      'dashboard-claim',
      'ringkasan',
      portal,
      token,
      f.lini_bisnis ?? '',
      f.cari ?? '',
      f.nomor_polis ?? '',
      f.nomor_klaim ?? '',
      f.pic ?? '',
      f.status_transfer ?? '',
      f.status_bayar ?? '',
    ] as const,

  telusur: (portal: string | null, token: string | null, tile: Tile, f: PenyaringDashboard) =>
    [
      'dashboard-claim',
      'telusur',
      portal,
      token,
      tile,
      f.lini_bisnis ?? '',
      f.cari ?? '',
      // Kelima isian panel WAJIB ikut kunci. Tanpa itu, mengubah Nopolis tidak menembak
      // ulang dan pengguna melihat daftar lama — tanpa tanda apa pun bahwa penyaringnya
      // diabaikan.
      f.nomor_polis ?? '',
      f.nomor_klaim ?? '',
      f.pic ?? '',
      f.status_transfer ?? '',
      f.status_bayar ?? '',
      f.halaman ?? 1,
    ] as const,
}

/** Menyusun query string dari penyaring; yang kosong tidak dikirim sama sekali. */
function buildParams(f: PenyaringDashboard): URLSearchParams {
  const params = new URLSearchParams()
  if (f.lini_bisnis?.trim()) params.set('lini_bisnis', f.lini_bisnis.trim())
  if (f.cari?.trim()) params.set('cari', f.cari.trim())

  // Ketiga isian panel penyaring. Yang kosong TIDAK dikirim: server memperlakukan penanda
  // kosong sebagai "tidak menyaring", dan mengirim string kosong akan menyamakannya dengan
  // mencari klaim yang nomor polisnya memang kosong.
  if (f.nomor_polis?.trim()) params.set('nomor_polis', f.nomor_polis.trim())
  if (f.nomor_klaim?.trim()) params.set('nomor_klaim', f.nomor_klaim.trim())
  if (f.pic?.trim()) params.set('pic', f.pic.trim())
  if (f.status_transfer?.trim()) params.set('status_transfer', f.status_transfer.trim())
  if (f.status_bayar?.trim()) params.set('status_bayar', f.status_bayar.trim())

  return params
}

/**
 * Hook bentuk layar — isi dropdown lini bisnis dan daftar kartu.
 *
 * # Kenapa bentuk penyaring datang dari server
 *
 * Karena kelima pilihan lini bisnis adalah HASIL PEMBACAAN activity Pega
 * (`Activity/SetDashboardClaim-Act.xml`), dan tempat pembacaan itu tercatat adalah backend.
 * Menyalinnya ke layar berarti daftar yang sama hidup di dua tempat, dan yang satu akan
 * tertinggal saat yang lain diperbaiki.
 *
 * Di layar ini akibatnya nyata: nilainya dibandingkan server sebagai TEKS, sehingga satu
 * ejaan yang berbeda menghasilkan penyaring yang tidak pernah cocok — dan backend
 * menolaknya dengan 422, bukan diam-diam menampilkan semuanya.
 */
export function usePenyaringDashboard() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: keys.penyaring(portal, token),
    queryFn: () => callAPI<PenyaringResponse>(`${PATH}/penyaring`, { token, portal }),
    enabled: token !== null && portal !== null,

    // Bentuk layar tidak berubah selama aplikasi berjalan: ia dibaca dari kode, bukan dari
    // data. Mengambilnya ulang tiap kali penyaring berubah hanya menambah perjalanan
    // jaringan tanpa satu pun manfaat.
    staleTime: Infinity,
    gcTime: Infinity,
  })
}

/**
 * Hook keempat angka kartu.
 *
 * Menggantikan keempat kueri hitung sistem lama, yang di sana dijalankan satu per satu oleh
 * `SetDashboardClaim` menurut `param.tipe`.
 *
 * # Satu permintaan untuk keempatnya
 *
 * Layar membaca keempat angka sekaligus, bukan satu per kartu. Memecahnya menjadi empat
 * permintaan akan membuat keempat angka datang pada saat yang berbeda — dan pada layar
 * manajerial, empat angka yang tidak sezaman lebih buruk daripada empat angka yang datang
 * sedikit lebih lambat.
 *
 * # Tidak dijalankan sebelum portal dipilih
 *
 * Backend memang menolak permintaan tanpa portal (`TKT-F6-002`), tetapi menembaknya lebih
 * dulu hanya untuk menerima penolakan adalah perjalanan jaringan yang sia-sia.
 */
export function useRingkasanDashboard(filter: PenyaringDashboard) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const params = buildParams(filter)

  return useQuery({
    queryKey: keys.ringkasan(portal, token, filter),
    queryFn: () => {
      const query = params.toString()
      return callAPI<RingkasanResponse>(query ? `${PATH}/ringkasan?${query}` : `${PATH}/ringkasan`, {
        token,
        portal,
      })
    },
    enabled: token !== null && portal !== null,

    // Angkanya berubah setiap kali sebuah klaim berpindah tahap. Dianggap usang seketika,
    // tetapi tidak ditembak ulang sendiri — pengguna yang menekan Muat ulang.
    staleTime: 0,
  })
}

/**
 * Hook telusur satu tile.
 *
 * `enabled` ikut menunggu `tile` terisi: layar baru menembaknya SETELAH sebuah kartu
 * dipilih, bukan saat dibuka. Tanpa itu, membuka layar akan menarik daftar yang belum tentu
 * dilihat siapa pun.
 */
export function useTelusurDashboard(tile: Tile | null, filter: PenyaringDashboard) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const params = buildParams(filter)
  params.set('halaman', String(filter.halaman ?? 1))
  params.set('ukuran', String(PAGE_SIZE))

  return useQuery({
    queryKey: keys.telusur(portal, token, tile ?? 'outstanding', filter),
    queryFn: () =>
      callAPI<TelusurResponse>(`${PATH}/${tile}?${params.toString()}`, { token, portal }),
    enabled: token !== null && portal !== null && tile !== null,

    // Halaman sebelumnya dipertahankan selama halaman berikutnya dimuat, sehingga tabel
    // tidak berkedip kosong setiap kali pengguna berpindah halaman.
    placeholderData: (previous) => previous,
    staleTime: 0,
  })
}

/**
 * Hook tab **Inbox Tampungan PIC**.
 *
 * Ia TIDAK mengirim penyaring lini bisnis, dan itu bukan kelalaian: kueri lamanya tidak
 * punya penandanya, dan layar lamanya tidak menggambar dropdown Bisnis pada tab ini.
 *
 * `enabled` ikut menunggu tab itu benar-benar dibuka, supaya membuka layar tidak menarik
 * daftar yang belum tentu dilihat siapa pun.
 */
export function useTampunganPIC(aktif: boolean, filter: PenyaringDashboard) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  const params = new URLSearchParams()
  if (filter.cari?.trim()) params.set('cari', filter.cari.trim())
  params.set('halaman', String(filter.halaman ?? 1))
  params.set('ukuran', String(PAGE_SIZE))

  return useQuery({
    queryKey: [
      'dashboard-claim',
      'tampungan',
      portal,
      token,
      filter.cari ?? '',
      filter.halaman ?? 1,
    ] as const,
    queryFn: () =>
      callAPI<TampunganResponse>(`${PATH}/tampungan?${params.toString()}`, { token, portal }),
    enabled: token !== null && portal !== null && aktif,
    placeholderData: (previous) => previous,
    staleTime: 0,
  })
}

/**
 * Mengunduh satu tile sebagai CSV.
 *
 * Menggantikan tombol "Export to Excel" pada grid layar lama. Bentuknya CSV, bukan XLSX:
 * `D-11` menetapkan pembuatan berkas dikerjakan sendiri, dan CSV terbaca Excel tanpa pustaka
 * tambahan sekaligus dapat dialirkan baris demi baris tanpa menahan memori.
 *
 * Bukan `<a download>` biasa karena permintaannya butuh header `Authorization` dan
 * `X-Portal` — tanpa keduanya unduhan dijawab "sesi tidak sah", atau lebih buruk: dilayani
 * entitas yang salah.
 */
export async function unduhTile(
  tile: Tile,
  filter: PenyaringDashboard,
  token: string | null,
  portal: string | null,
): Promise<void> {
  const params = buildParams(filter)
  const query = params.toString()
  const path = `${PATH}/${tile}/unduh${query ? `?${query}` : ''}`

  simpanBerkas(await unduhBerkas(path, { token, portal }))
}

// Tidak ada `unduhTampungan`. Tab Inbox Tampungan PIC TIDAK punya tombol unduh di layar
// lama, dan yang pernah ada di sini dicabut 2026-10-07 — lihat keterangan di
// TampunganPIC.tsx.

/**
 * Hook pengajuan permintaan transfer.
 *
 * # Ia mengajukan PERMINTAAN, bukan memindahkan penugasan
 *
 * `P-1` menetapkan `PC_ASSIGN_WORKLIST` masih ditulis Pega selama masa paralel, sehingga
 * yang tercatat adalah permintaan beserta pemohonnya; pelaksanaannya tetap di Pega.
 * Penugasannya TIDAK berpindah, dan barisnya tetap ada di layar — itulah sebabnya layar
 * wajib menyatakannya, bukan membiarkan pengguna menduga.
 */
export function useAjukanTransfer() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    /*
      Penyaring ikut sebagai PARAMETER KUERI, bukan di badan permintaan.

      Server membacanya dengan `readFilter` — pembaca yang SAMA dengan yang dipakai daftar.
      Itu yang membuat lingkup `saring` memindahkan tepat klaim yang terlihat pengguna; dua
      pembaca berbeda akan menyimpang diam-diam begitu salah satunya berubah.

      Pada lingkup lain parameternya diabaikan server, jadi mengirimkannya selalu aman.
    */
    mutationFn: ({ body, penyaring }: { body: PermintaanTransfer; penyaring?: PenyaringDashboard }) =>
      callAPI<TransferResponse>(
        `${PATH}/transfer${jalurPenyaring(penyaring)}`,
        { metode: 'POST', body, token, portal },
      ),

    // Daftar disegarkan supaya penanda "permintaan terkirim" muncul tanpa muat ulang manual.
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['dashboard-claim'] })
    },
  })
}

/**
 * Jalur modul Inbox Close Claim.
 *
 * # Kenapa menembak modul lain, dan kenapa lewat HTTP
 *
 * Tile Close Claim pada dashboard menampilkan populasi yang **sama persis** dengan Inbox
 * Close Claim, dan kedua tombolnya — ReOpen dan Copy Klaim — sudah punya endpoint yang
 * berjalan beserta aturan otorisasi dan pencatatannya.
 *
 * Membangunnya ulang di sini berarti dua tempat yang memutuskan hal yang sama, dan cepat
 * atau lambat keduanya berbeda. Yang dipakai karena itu endpoint yang sudah ada.
 *
 * Yang TIDAK dilakukan: mengimpor dari `src/modules/inbox-close-claim`. Aturan frontend
 * melarang satu fitur mengimpor fitur lain — kebutuhan bersama naik ke `shared/`. Yang
 * dipakai di sini adalah **kontrak HTTP**-nya, bukan kodenya, sehingga modul itu sama
 * sekali tidak disentuh.
 */
const PATH_CLOSE_CLAIM = '/api/inbox-close-claim'

/**
 * Izin mengajukan ReOpen dan Copy Klaim.
 *
 * Jawabannya datang dari DOMAIN modul Inbox Close Claim, bukan ditebak di layar ini. Di
 * sistem lama penjaganya adalah When rule `IsManagerPNC_CLOSE`; menyalin aturannya ke sini
 * berarti dua salinan yang dapat berbeda.
 */
export function useIzinPermintaanKlaim(aktif: boolean) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['dashboard-claim', 'izin-permintaan', portal, token],
    enabled: aktif && portal !== null,
    queryFn: () =>
      callAPI<IzinPermintaanKlaim>(`${PATH_CLOSE_CLAIM}/penyaring`, { token, portal }),

    // Izin berubah hanya saat peran pengguna berubah, dan itu menuntut masuk ulang.
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Pengajuan ReOpen atau Copy Klaim atas baris-baris yang dicentang.
 *
 * # Satu permintaan per klaim, bukan satu permintaan berisi banyak klaim
 *
 * Endpoint `POST /permintaan` menerima **satu** `klaim_id`. Mencentang lima baris karena itu
 * mengirim lima permintaan — persis seperti pengguna mengklik barisnya satu per satu di
 * layar lama, yang juga tidak punya pengajuan massal.
 *
 * Akibatnya sebagiannya dapat berhasil dan sebagiannya gagal, dan itu TIDAK disembunyikan:
 * hasilnya dikembalikan per baris. Menyatukannya menjadi satu "gagal" akan membuat pengguna
 * mengirim ulang seluruhnya, dan yang sudah tercatat akan ditolak 409 — penolakan yang lalu
 * terbaca sebagai kegagalan baru.
 *
 * Dikirim BERURUTAN, bukan serentak. Lima permintaan tulis serentak ke tabel yang sama
 * tidak mempercepat apa pun yang berarti, dan urutan tercatatnya menjadi tidak dapat
 * ditebak saat dibaca kembali di jejak audit.
 */
export function useAjukanPermintaanKlaim() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: async (input: {
      jenis: JenisPermintaanKlaim
      alasan: string
      baris: { klaim_id: string; nomor_klaim: string }[]
    }): Promise<HasilPermintaanKlaim[]> => {
      const hasil: HasilPermintaanKlaim[] = []

      for (const baris of input.baris) {
        try {
          await callAPI(`${PATH_CLOSE_CLAIM}/permintaan`, {
            token,
            portal,
            metode: 'POST',
            body: { jenis: input.jenis, klaim_id: baris.klaim_id, alasan: input.alasan },
          })
          hasil.push({ ...baris, berhasil: true })
        } catch (failure) {
          hasil.push({ ...baris, berhasil: false, pesan: pesanGalat(failure) })
        }
      }

      return hasil
    },

    onSuccess: () => {
      // Daftar tile diperbarui karena yang berubah bukan klaimnya melainkan penanda
      // permintaan tertunda pada barisnya. Daftar modul Inbox Close Claim ikut diperbarui:
      // baris yang sama tampil di sana, dan membiarkannya basi membuat pengguna mengajukan
      // dua kali dari dua layar.
      client.invalidateQueries({ queryKey: ['dashboard-claim', 'telusur'] })
      client.invalidateQueries({ queryKey: ['inbox-close-claim', 'daftar'] })
    },
  })
}

/** Membaca pesan galat yang layak dibaca pengguna. */
function pesanGalat(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}

/**
 * Daftar PIC Teknik untuk layar Transfer.
 *
 * # Tanpa penyaring lini bisnis
 *
 * `BrowseVMstUserTeknis_RD` menyaring `TYPE_BUSINESS = Param.type_business`, dan parameter
 * itu diisi `OperatorID.pyPosition` — properti Pega yang rupanya memuat "NONMBU", "TRAVEL",
 * "BONDING", atau "PA". Nilai itu **tidak ada** di sistem baru: HCC/HCQ mengembalikan jabatan
 * sebenarnya (`Placement.PositionName`).
 *
 * Tiga bentuk sudah dicoba dan ketiganya keliru — memblokir daftar dengan peringatan,
 * memakai jabatan sesi (daftar kosong tanpa penjelasan), lalu meminta pengguna memilih lini
 * bisnis (langkah yang di layar lama tidak ada).
 *
 * Yang berlaku: tidak ada penyaring lini sama sekali, mengikuti layar lama. Selisihnya —
 * daftar yang lebih luas — dicatat di `picteknik.sql`.
 */
export function useDaftarPIC(aktif: boolean, cari: string, halaman = 1) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['dashboard-claim', 'pic-teknik', portal, token, cari, halaman],
    enabled: aktif && portal !== null,
    queryFn: () => {
      const q = new URLSearchParams({
        ukuran: String(PAGE_SIZE),
        halaman: String(halaman),
      })
      if (cari) q.set('cari', cari)
      return callAPI<PICResponse>(`${PATH}/pic-teknik?${q.toString()}`, { token, portal })
    },

    // Master petugas berubah jarang; menembaknya ulang setiap ketikan tidak memberi apa pun.
    staleTime: 60 * 1000,
  })
}

/**
 * Mengubah penyaring menjadi parameter kueri.
 *
 * Ditulis sendiri, bukan `new URLSearchParams(penyaring)`: penyaringnya memuat `halaman`
 * yang bertipe angka, dan URLSearchParams hanya menerima teks. Isian kosong DIBUANG supaya
 * alamatnya tidak dipenuhi parameter hampa yang menyulitkan pembacaan log.
 */
function jalurPenyaring(penyaring?: PenyaringDashboard): string {
  if (penyaring === undefined) return ''

  const query = paramPenyaring(penyaring)

  // Tanpa penjagaan ini, penyaring yang seluruh isiannya kosong meninggalkan `?` menggantung
  // di alamatnya — benar secara teknis, dan menyesatkan di log.
  return query === '' ? '' : `?${query}`
}

function paramPenyaring(penyaring: PenyaringDashboard): string {
  const params = new URLSearchParams()
  for (const [kunci, nilai] of Object.entries(penyaring)) {
    if (nilai === undefined || nilai === '') continue
    params.set(kunci, String(nilai))
  }
  return params.toString()
}

/**
 * Rincian satu klaim, untuk popup yang terbuka dari nomor klaim.
 *
 * Kuncinya `klaim_id` (PZINSKEY), bukan nomor klaim — itu yang dipakai Pega
 * (`GetJsonKlaimPNC` menyaring `idpega`), dan pada data warisan yang nomornya kembar,
 * mencari lewat nomor dapat membuka klaim yang salah.
 *
 * `aktif` memastikan permintaannya hanya berangkat saat popup-nya dibuka. Di Pega pun
 * pengambilannya terjadi SAAT diklik, bukan saat gridnya dimuat.
 */
export function useRincianKlaim(klaimID: string | null) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)

  return useQuery({
    queryKey: ['dashboard-claim', 'rincian', portal, token, klaimID],
    enabled: klaimID !== null && klaimID !== '',
    queryFn: () =>
      callAPI<RincianKlaim>(`${PATH}/klaim/${encodeURIComponent(klaimID ?? '')}`, {
        token,
        portal,
      }),
  })
}
