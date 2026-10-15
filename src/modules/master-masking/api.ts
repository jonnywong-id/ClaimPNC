import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import { callAPI } from '@/api/client'
import type {
  MaskingBulkInput,
  MaskingBulkResponse,
  MaskingBranchListResponse,
  MaskingInput,
  MaskingListResponse,
  MaskingOperatorListResponse,
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

/**
 * Pilihan MODUL beserta label yang dibaca pengguna.
 *
 * `nilai` adalah yang tersimpan di kolom MODUL; `label` adalah yang tertulis di layar.
 * Keduanya berbeda, dan itu bukan kelalaian — dialog ViewDataModul di Pega menuliskan
 * "VIEW HISTORY KLAIM" untuk baris yang kolomnya berisi `PNCSearchKlaim`.
 *
 * Daftarnya TIDAK dapat dibaca dari export: rule `MODULKLAIMMASKING` dan section
 * `Emb_ModulForMaskingData` keduanya hilang (`R-16`). Isi di bawah dipastikan dari dua
 * sumber yang saling menguatkan — tangkapan layar aplikasi Pega yang berjalan, dan
 * pembacaan langsung `POOLDATA.MST_PROTEKSI_DATA_PNC` yang hanya memuat satu nilai MODUL.
 *
 * Bila Tim Pega kelak mengirim rule-nya dan daftarnya lebih panjang, YANG BERUBAH HANYA
 * berkas ini.
 */
export const MASKING_MODULES = [{ nilai: 'PNCSearchKlaim', label: 'VIEW HISTORY KLAIM' }] as const

/**
 * Pilihan SUB MODUL beserta labelnya.
 *
 * Ketiganya adalah atom yang benar-benar ada di kolom SUBMODUL portal ASM, dan urutannya
 * mengikuti dialog ViewDataModul.
 *
 * Perhatikan selisih satu huruf pada yang pertama: layar menulis "PENERIMA", kolomnya
 * menyimpan "Penerimaan". Keduanya memang berbeda di sistem lama — satu baris produksi
 * bahkan menyimpan "Penerima" tanpa `-an`. Pencocokannya karena itu menerima kedua bentuk.
 */
export const MASKING_SUB_MODULES = [
  { nilai: 'Penerimaan Pembayaran Klaim', label: 'PENERIMA PEMBAYARAN KLAIM' },
  { nilai: 'Registrasi', label: 'REGISTRASI' },
  { nilai: 'Dokumen', label: 'DOKUMEN' },
] as const

/** Membandingkan nilai sub modul tanpa peduli besar-kecil huruf, spasi tepi, dan `-an`. */
function subModuleKey(value: string) {
  return value.trim().toUpperCase().replace('PENERIMAAN', 'PENERIMA')
}

/**
 * Memecah isi kolom SUBMODUL menjadi daftar atom.
 *
 * Nilainya dipisah koma DENGAN koma di ujung ("Registrasi,Dokumen,"), sehingga pemecahan
 * biasa menghasilkan satu entri kosong di belakang.
 */
export function splitSubModules(value: string): string[] {
  return value
    .split(',')
    .map((item) => item.trim())
    .filter((item) => item !== '')
}

/** Satu baris kotak centang pada dialog VIEW maupun pada form. */
export type ChecklistItem = {
  /** Nilai yang tersimpan di kolom. */
  nilai: string
  /** Teks yang dibaca pengguna. */
  label: string
  checked: boolean
  /** False bila nilainya tidak ada di daftar pilihan. */
  dikenal: boolean
}

/**
 * Menyusun daftar centang SUB MODUL.
 *
 * Nilai tersimpan yang TIDAK dikenal tetap ditampilkan — tercentang, ditandai sebagai di
 * luar daftar. Menyembunyikannya akan membuat data yang perlu diperbaiki tidak pernah
 * terlihat; satu baris produksi memang menyimpan sub modul yang salah ketik. Dan pada
 * FORM, menyembunyikannya jauh lebih berbahaya: menyimpan ulang baris itu akan membuang
 * nilainya tanpa ada yang meminta.
 */
export function subModuleChecklist(value: string): ChecklistItem[] {
  const stored = splitSubModules(value)
  const storedKey = new Set(stored.map(subModuleKey))

  const known: ChecklistItem[] = MASKING_SUB_MODULES.map((option) => ({
    nilai: option.nilai,
    label: option.label,
    checked: storedKey.has(subModuleKey(option.nilai)),
    dikenal: true,
  }))

  const knownKey = new Set(MASKING_SUB_MODULES.map((o) => subModuleKey(o.nilai)))
  const extra: ChecklistItem[] = stored
    .filter((item) => !knownKey.has(subModuleKey(item)))
    .map((item) => ({ nilai: item, label: item, checked: true, dikenal: false }))

  return [...known, ...extra]
}

/** Menyusun daftar centang MODUL dengan aturan yang sama. */
export function moduleChecklist(value: string): ChecklistItem[] {
  const clean = value.trim().toUpperCase()

  const known: ChecklistItem[] = MASKING_MODULES.map((option) => ({
    nilai: option.nilai,
    label: option.label,
    checked: option.nilai.toUpperCase() === clean,
    dikenal: true,
  }))

  const matched = MASKING_MODULES.some((o) => o.nilai.toUpperCase() === clean)
  if (!matched && clean !== '') {
    return [...known, { nilai: value.trim(), label: value.trim(), checked: true, dikenal: false }]
  }
  return known
}

/**
 * Menyusun kembali isi kolom MODUL dari pilihan yang dicentang.
 *
 * TANPA koma di ujung — kolom MODUL di portal ASM menyimpan satu nilai polos
 * (`PNCSearchKlaim`), bukan daftar berkoma.
 */
export function joinModules(nilai: string[]): string {
  return nilai.join(',')
}

/**
 * Menyusun kembali isi kolom SUBMODUL dari pilihan yang dicentang.
 *
 * DENGAN koma di ujung — itulah bentuk yang tersimpan di sistem lama
 * (`Registrasi,Dokumen,`), dan Pega masih membaca tabel yang sama selama masa paralel
 * (`D-21`). Menghilangkan koma penutup akan membuat bentuknya menyimpang dari seluruh
 * baris yang sudah ada.
 *
 * Urutannya mengikuti urutan pilihan, bukan urutan pengguna mencentang — supaya dua baris
 * berisi kewenangan yang sama tersimpan dengan teks yang sama pula.
 */
export function joinSubModules(nilai: string[]): string {
  if (nilai.length === 0) return ''
  return nilai.join(',') + ','
}

/** Menyalakan atau memadamkan satu pilihan, lalu menyusun ulang nilai kolomnya. */
export function toggleChecklist(
  item: ChecklistItem[],
  nilai: string,
  checked: boolean,
  join: (nilai: string[]) => string,
): string {
  return join(
    item
      .map((row) => (row.nilai === nilai ? { ...row, checked } : row))
      .filter((row) => row.checked)
      .map((row) => row.nilai),
  )
}

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

/**
 * Hook daftar petugas sebuah cabang — mengisi tabel pada form Tambah.
 *
 * Menggantikan `RDB List/GetDataLogin-SQL.xml`. Tidak dijalankan sebelum cabang dipilih:
 * daftar petugas tanpa cabang tidak punya arti, dan server pun menolaknya.
 */
export function useOperators(branchID: string) {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const clean = branchID.trim()

  return useQuery({
    queryKey: ['master-masking-pengguna', portal, token, clean],
    queryFn: () =>
      callAPI<MaskingOperatorListResponse>(
        `${ROUTE}/pengguna?cabang=${encodeURIComponent(clean)}`,
        { token, portal },
      ),
    enabled: token !== null && portal !== null && clean !== '',
    // Daftar petugas berubah saat seseorang pindah cabang atau berganti jabatan — jarang,
    // tetapi tidak sejarang daftar cabang.
    staleTime: 5 * 60 * 1000,
  })
}

/**
 * Hook simpan form Tambah — MASSAL.
 *
 * Satu cabang, banyak baris, satu tombol SIMPAN; penyimpanannya berulang per baris di
 * server, sama seperti `Activity/InsermaskingDataKlaimPnc_-Act.xml` yang mengulang
 * `TempLogin.pxResults`.
 *
 * Jawabannya melaporkan hasil PER BARIS, bukan satu kata berhasil/gagal.
 */
export function useSaveMaskingBulk() {
  const token = useSession((state) => state.token)
  const portal = useSelectedPortal((state) => state.alias)
  const client = useQueryClient()

  return useMutation({
    mutationFn: (input: MaskingBulkInput) =>
      callAPI<MaskingBulkResponse>(ROUTE, { metode: 'POST', body: input, token, portal }),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['master-masking'] })
    },
  })
}

type SaveFields = MaskingInput & {
  /** Wajib: hook ini hanya melayani PENGUBAHAN. Penambahan memakai useSaveMaskingBulk. */
  id: string
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
      callAPI<MaskingResponse>(`${ROUTE}/${encodeURIComponent(id)}`, {
        metode: 'PUT',
        body: input,
        token,
        portal,
      }),
    onSuccess: () => {
      // Seluruh daftar dimuat ulang dari server, bukan disunting di cache. Pada
      // penambahan, ID barunya hanya diketahui server; dan karena penyaring ikut di dalam
      // kunci, baris baru bisa saja tidak cocok dengan penyaring yang sedang aktif.
      client.invalidateQueries({ queryKey: ['master-masking'] })
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
      client.invalidateQueries({ queryKey: ['master-masking'] })
    },
  })
}
