import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type {
  CountsResponse,
  DetailResponse,
  MetadataResponse,
  SalvageRow,
  Tab,
} from './types'

const PATH = '/api/inbox-salvage'
const META_PATH = `${PATH}/daftar`
const COUNTS_PATH = `${PATH}/ringkas`
const EXPORT_PATH = `${PATH}/ekspor`

const SAMPLE_PROFILE = {
  identitas: '90000071',
  nama: 'Contoh Petugas Salvage',
  jenis: 'KARYAWAN',
  login: 'petugassalvagecontoh',
  email: 'contoh.salvage@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Kolom daftar Salvage Outstanding — EMPAT, dan berbeda dari daftar lain.
 *
 * Perbedaan susunan kolom antardaftar itulah yang membuat kolomnya datang dari server:
 * tiga belas daftar memakai empat susunan, dan dua di antaranya berbeda hanya satu kolom.
 */
const TAB_OUTSTANDING: Tab = {
  kode: 'outstanding',
  nama: 'Salvage Outstanding',
  keterangan: 'Klaim yang BELUM ditandai punya salvage sama sekali.',
  kolom: [
    { kunci: 'pic', judul: 'PIC', angka: false },
    { kunci: 'no_klaim', judul: 'No Klaim', angka: false },
    { kunci: 'nama_bisnis', judul: 'COB', angka: false },
    { kunci: 'tanggal_kejadian', judul: 'Tgl Kejadian', angka: false },
  ],
  label_pencarian: 'CARI NO KLAIM',
  pencarian_cocok_persis: false,
  catatan_daftar:
    'Angka pada baris "Outstanding" di tabel ringkas TIDAK sama dengan jumlah baris di sini.',

  // Barisnya KLAIM — rincian dibuka dengan NOMOR KLAIM.
  kunci_rincian: 'klaim',
}

/** Daftar Checker — grid terlebar, dan pencariannya COCOK PERSIS. */
const TAB_CHECKER: Tab = {
  kode: 'checker',
  nama: 'Checker',
  keterangan: 'Pengajuan yang menunggu KEPUTUSAN checker.',
  kolom: [
    { kunci: 'tanggal_input', judul: 'Tanggal Input', angka: false },
    { kunci: 'pic', judul: 'PIC', angka: false },
    { kunci: 'no_klaim', judul: 'No Klaim', angka: false },
    { kunci: 'jenis_salvage', judul: 'Jenis Salvage', angka: false },
    { kunci: 'nilai_pengajuan_pic', judul: 'Nilai Pengajuan PIC', angka: true },
    { kunci: 'catatan', judul: 'Catatan', angka: false },
    { kunci: 'aging', judul: 'Aging', angka: false },
  ],
  label_pencarian: 'CARI NO KLAIM ATAU PIC',
  pencarian_cocok_persis: true,
  catatan_daftar: 'Pencarian di daftar ini COCOK PERSIS, bukan mengandung.',

  // Barisnya PENGAJUAN — rincian dibuka dengan ID PENGAJUAN.
  kunci_rincian: 'pengajuan',
}

const METADATA: MetadataResponse = {
  daftar: [TAB_OUTSTANDING, TAB_CHECKER],
  daftar_bawaan: 'outstanding',
  pilihan_status_salvage: [
    { kode: '2', label: 'Salvage DiTerima' },
    { kode: '4', label: 'Waive Salvage' },
  ],
  kolom_berkas_unggahan: ['item', 'quantity', 'satuan', 'remarks'],
  selisih_terencana: [
    'Angka pada baris "Outstanding" di tabel ringkas TIDAK sama dengan jumlah barisnya.',
    'Kolom "Catatan" SELALU KOSONG.',
  ],
  portal: 'ASM',
}

/**
 * Tabel ringkas contoh.
 *
 * Baris terakhir sengaja TANPA `daftar`: ia baris "Tidak Terjual", yang di Pega pun tidak
 * punya tab. Uji di bawah memastikan ia digambar sebagai teks biasa — bukan tombol yang
 * tidak melakukan apa-apa.
 *
 * Angka baris "Outstanding" sengaja BERBEDA dari jumlah baris daftarnya, karena begitulah
 * keadaannya di Pega.
 */
const COUNTS: CountsResponse = {
  baris: [
    { status_salvage: 'Outstanding', jumlah: 3, daftar: 'outstanding' },
    { status_salvage: 'Checker', jumlah: 2, daftar: 'checker' },
    { status_salvage: 'Tidak Terjual', jumlah: 7 },
  ],
  portal: 'ASM',
}

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 *
 * `catatan` sengaja KOSONG: begitulah yang selalu dikirim server, karena tidak ada satu pun
 * penulisnya di Pega. Uji di bawah memastikan layar menggambarnya sebagai tanda pisah.
 */
const BARIS: SalvageRow = {
  referensi: 'PNC-700071',
  no_klaim: 'PNC-700071',
  id_salvage: '',
  tanggal_input: '',
  tanggal_kejadian: '2026-08-14',
  pic: 'PETUGASCONTOH',
  nama_bisnis: 'Property All Risk',
  nama_objek: '',
  jenis_salvage: '',
  lokasi_salvage: '',
  status_lelang: '',
  nilai_pengajuan_pic: '',
  nilai_request_balai_lelang: '',
  email: '',
  keterangan_pic: '',
  tipe_pengajuan: '',
  aging: '',
  catatan: '',
}

const BARIS_CHECKER: SalvageRow = {
  ...BARIS,
  referensi: '103',
  id_salvage: '103',
  tanggal_input: '2026-09-09',
  tanggal_kejadian: '',
  jenis_salvage: 'Drum Kosong',
  nilai_pengajuan_pic: '1800000',
  aging: '16 day',
  catatan: '',
}

const DETAIL_PATH = `${PATH}/pengajuan/103`

/** Rincian satu pengajuan, sebagaimana dijawab `GET /api/inbox-salvage/pengajuan/{id}`. */
const DETAIL: DetailResponse = {
  id_salvage: '103',
  no_klaim: 'PNC-700071',
  nama_bisnis: 'Property All Risk',
  ada_pengajuan: true,
  pic: 'SITIRAHAYU',
  tanggal_kejadian: '2026-09-01',

  tanggal_input_salvage: '2026-09-09',
  jenis_salvage: 'Drum Kosong',
  quantity_salvage: '24',
  estimasi: '1800000',
  lokasi_salvage: 'Gudang Cakung',

  tanggal_transfer_ga: '2026-09-10',

  // Kode dan labelnya dikirim bersamaan, dan uji di bawah membuktikan yang DIGAMBAR
  // sebagai posisi adalah labelnya.
  kode_posisi_salvage: '3',
  posisi_salvage: 'Checker',

  tanggal_akseptasi: '',
  no_akseptasi: '',
  remark: 'Menunggu keputusan checker',
  mata_uang: 'IDR',

  nama_object: 'Gudang Blok C',
  nama_coverage: 'Property All Risk',
  nilai_salvage: '',

  email: 'salvage.pusat@example.invalid',
  nilai_penawaran: '',
  nama_pemenang: '',
  tanggal_lelang: '',

  nama_pic_survey: 'Petugas Survey Contoh',
  no_telp_pic_survey: '0210000000',
  email_pic_survey: 'survey.contoh@example.invalid',

  lokasi_salvage_di_jabodetabek: true,
  pengajuan_sebelum_juli_2023: false,

  riwayat: [
    {
      id_salvage: '103',
      tanggal_input: '2026-09-09',
      no_klaim: 'PNC-700071',
      pic: 'SITIRAHAYU',
      nilai_minimum: '1800000',
      posisi_salvage: 'Salvage Diterima di Komite',
    },
  ],

  barang: [
    {
      nama_barang: 'Drum 200L',
      jumlah: 12,
      satuan: 'unit',
      total_nilai: '1200000',
      status_terjual: 'Belum terjual',
      nama_pemenang: '',
      no_akseptasi: '-',
      nilai_akseptasi: '',
      remark: '',
    },
  ],

  portal: 'ASM',
}

type Call = { url: string; init?: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(answer: (url: string, init?: RequestInit) => Response) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    // Bilah atas memuat pemilih portal dan menu, sehingga SETIAP layar di balik sesi ikut
    // memanggil keduanya. Dijawab otomatis supaya uji layar ini menguji layarnya.
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    return Promise.resolve(answer(url, init))
  })
}

/** Peladen tiruan yang menjawab bentuk layar, tabel ringkas, dan isi daftarnya. */
function stubDefaultFetch() {
  stubFetch((url) => {
    if (url === META_PATH) return jsonResponse(200, METADATA)
    if (url === COUNTS_PATH) return jsonResponse(200, COUNTS)
    if (url === DETAIL_PATH) return jsonResponse(200, DETAIL)

    // Jalur klaim dijawab dengan pengajuan yang ADA, supaya uji panel di bawah menguji
    // panelnya. Keadaan sebaliknya — klaim tanpa pengajuan — diuji terpisah, dengan
    // peladen tiruannya sendiri.
    if (url.startsWith(`${PATH}/klaim/`)) {
      return jsonResponse(200, { ...DETAIL, no_klaim: BARIS.no_klaim })
    }

    if (url.startsWith(EXPORT_PATH)) {
      return new Response('PIC\nPETUGASCONTOH\n', {
        status: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition': 'attachment; filename="inbox-salvage-outstanding.csv"',
        },
      })
    }

    // Daftar yang diminta menentukan bentuk jawabannya. Menjawab daftar yang sama untuk
    // setiap permintaan akan membuat uji perpindahan daftar lulus tanpa membuktikan apa
    // pun — dan di layar ini bahayanya besar, karena beberapa daftar memang mirip.
    const address = new URL(url, 'http://uji.invalid')
    const wanted = address.searchParams.get('daftar')
    const search = address.searchParams.get('cari') ?? ''

    const answered = wanted === TAB_CHECKER.kode ? TAB_CHECKER : TAB_OUTSTANDING
    const rows =
      answered.kode === TAB_CHECKER.kode
        ? search === '' || search === 'PNC-700071'
          ? [BARIS_CHECKER]
          : []
        : [BARIS]

    return jsonResponse(200, {
      daftar: answered,
      baris: rows,
      paginasi: { halaman: 1, ukuran: 20, total: rows.length, total_halaman: 1 },
      cari: search,
      portal: 'ASM',
    })
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/inbox-salvage']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/**
 * renderLoaded menggambar layar lalu MENUNGGU bentuknya tiba.
 *
 * Penantiannya pada salah satu daftar, bukan pada judul layar: judulnya sudah ada sejak
 * penggambaran pertama, sementara bilah daftar baru tiba bersama jawaban `/daftar`.
 */
async function renderLoaded() {
  renderPage()
  await screen.findByRole('tab', { name: /Salvage Outstanding/ })
}

function lastListCall(): Call | undefined {
  return [...calls].reverse().find((c) => c.url === PATH || c.url.startsWith(`${PATH}?`))
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: SAMPLE_PROFILE,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
})

it('menggambar judul kolom SESUAI daftar yang terbuka, bukan satu susunan tetap', async () => {
  stubDefaultFetch()
  await renderLoaded()

  // Daftar bawaan: empat kolom, di antaranya "COB" dan "Tgl Kejadian".
  expect(await screen.findByRole('columnheader', { name: 'COB' })).toBeInTheDocument()
  expect(screen.getByRole('columnheader', { name: 'Tgl Kejadian' })).toBeInTheDocument()

  // Kolom milik daftar LAIN tidak boleh ikut tergambar.
  expect(
    screen.queryByRole('columnheader', { name: 'Nilai Pengajuan PIC' }),
  ).not.toBeInTheDocument()
})

it('berpindah daftar mengganti susunan kolomnya', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await userEvent.click(screen.getByRole('tab', { name: /Checker/ }))

  expect(
    await screen.findByRole('columnheader', { name: 'Nilai Pengajuan PIC' }),
  ).toBeInTheDocument()
  expect(screen.queryByRole('columnheader', { name: 'COB' })).not.toBeInTheDocument()
})

it('mengklik baris tabel ringkas membuka daftarnya', async () => {
  stubDefaultFetch()
  await renderLoaded()

  const ringkas = await screen.findByRole('table', {
    name: /Ringkasan jumlah pengajuan/,
  })
  await userEvent.click(within(ringkas).getByRole('button', { name: 'Checker' }))

  expect(
    await screen.findByRole('columnheader', { name: 'Nilai Pengajuan PIC' }),
  ).toBeInTheDocument()
})

/**
 * Satu baris pencacah TIDAK menuju daftar mana pun — di Pega pun tidak ada tab yang
 * menerimanya. Ia harus digambar sebagai teks, bukan tombol yang tidak melakukan apa-apa.
 */
// Peladen sekarang TIDAK mengirim baris tanpa daftar — satu-satunya yang begitu ("Tidak
// Terjual") tidak lagi digambar, sebab layar Pega sungguhan pun tidak menampilkannya.
//
// Uji ini tetap ada sebagai kontrak komponen: StatusSummary harus menggambar baris tanpa
// daftar sebagai teks biasa, bukan sebagai tombol yang menuju kekosongan. Bila baris
// seperti itu kembali — dan tiga daftar memang akan kembali begitu kewenangan berbasis
// peran ada — perilakunya sudah terjaga.
it('baris ringkas yang tidak punya daftar TIDAK dapat diklik', async () => {
  stubDefaultFetch()
  await renderLoaded()

  const ringkas = await screen.findByRole('table', {
    name: /Ringkasan jumlah pengajuan/,
  })

  expect(
    within(ringkas).queryByRole('button', { name: 'Tidak Terjual' }),
  ).not.toBeInTheDocument()
  expect(within(ringkas).getByText('Tidak Terjual')).toBeInTheDocument()
})

/**
 * Angka ringkas yang BERBEDA dari jumlah baris daftarnya bukan kerusakan — ia selisih yang
 * direplikasi (`P-5`). Uji ini memastikan keduanya memang digambar apa adanya, tanpa
 * "diperbaiki" agar cocok.
 */
it('menggambar angka ringkas apa adanya meski berbeda dari jumlah baris daftarnya', async () => {
  stubDefaultFetch()
  await renderLoaded()

  const ringkas = await screen.findByRole('table', {
    name: /Ringkasan jumlah pengajuan/,
  })
  const baris = within(ringkas)
    .getByRole('button', { name: 'Outstanding' })
    .closest('tr')

  expect(baris).not.toBeNull()
  expect(within(baris as HTMLElement).getByText('3')).toBeInTheDocument()
})

it('pencarian dikirim ke server, bukan disaring di peramban', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await userEvent.click(screen.getByRole('tab', { name: /Checker/ }))
  await screen.findByRole('columnheader', { name: 'Nilai Pengajuan PIC' })

  const kotak = screen.getByLabelText(/CARI NO KLAIM ATAU PIC/i)
  await userEvent.type(kotak, 'PNC-700071')

  await vi.waitFor(() => {
    const terakhir = lastListCall()
    expect(terakhir?.url).toContain('cari=PNC-700071')
  })
})

/**
 * Pada daftar yang pencariannya cocok persis, kekosongan hampir selalu berarti pengguna
 * mengetik separuh nomor klaim. Layar harus MENGATAKANNYA — tanpa itu, pengguna
 * menyimpulkan datanya hilang.
 */
it('menjelaskan kekosongan pada daftar yang pencariannya cocok persis', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await userEvent.click(screen.getByRole('tab', { name: /Checker/ }))
  await screen.findByRole('columnheader', { name: 'Nilai Pengajuan PIC' })

  await userEvent.type(screen.getByLabelText(/CARI NO KLAIM ATAU PIC/i), 'PNC-70')

  expect(await screen.findByText(/COCOK PERSIS/)).toBeInTheDocument()
})

/** Kolom yang SELALU kosong digambar sebagai tanda pisah, bukan sel kosong. */
it('menggambar sel kosong sebagai tanda pisah', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await userEvent.click(screen.getByRole('tab', { name: /Checker/ }))
  await screen.findByRole('columnheader', { name: 'Catatan' })

  const baris = await screen.findByText('Drum Kosong')
  const sel = baris.closest('tr')
  expect(sel).not.toBeNull()
  expect(within(sel as HTMLElement).getAllByText('—').length).toBeGreaterThan(0)
})

it('menyatakan selisih terencana kepada pengguna', async () => {
  stubDefaultFetch()
  await renderLoaded()

  expect(
    await screen.findByText(/Perbedaan yang disengaja terhadap layar Pega/),
  ).toBeInTheDocument()
  expect(screen.getByText(/Kolom "Catatan" SELALU KOSONG/)).toBeInTheDocument()
})

it('berpindah daftar mengosongkan kata kunci pencarian', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await userEvent.type(screen.getByLabelText(/CARI NO KLAIM/i), 'PNC-700071')
  await vi.waitFor(() => expect(lastListCall()?.url).toContain('cari=PNC-700071'))

  await userEvent.click(screen.getByRole('tab', { name: /Checker/ }))

  await vi.waitFor(() => {
    const terakhir = lastListCall()
    expect(terakhir?.url).toContain('daftar=checker')
    expect(terakhir?.url).not.toContain('cari=')
  })
})

it('membuka form Tambah dan dapat menutupnya kembali', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

  const form = await screen.findByRole('form', { name: 'Menambahkan Data Salvage' })
  expect(within(form).getByLabelText(/Nomor Klaim/)).toBeInTheDocument()
  expect(within(form).getByLabelText(/Nama Object/)).toBeInTheDocument()

  await userEvent.click(within(form).getByRole('button', { name: 'Batal' }))
  expect(
    screen.queryByRole('form', { name: 'Menambahkan Data Salvage' }),
  ).not.toBeInTheDocument()
})

/**
 * Pelanggaran validasi ditandai DI ISIANNYA, bukan sebagai satu pesan di atas form. Pada
 * form berisi tujuh belas isian, satu pesan umum memaksa pengguna menebak.
 */
it('menandai isian yang ditolak server pada isiannya masing-masing', async () => {
  stubFetch((url, init) => {
    if (url === META_PATH) return jsonResponse(200, METADATA)
    if (url === COUNTS_PATH) return jsonResponse(200, COUNTS)

    if (url === PATH && init?.method === 'POST') {
      return jsonResponse(422, {
        kode: 'validasi_gagal',
        pesan: 'Permintaan belum benar.',
        // Kunci `field`, bukan `isian`: itulah satu-satunya nama yang dibaca
        // `APIError.violations()`. Nama lain sampai ke layar tetapi tidak pernah terbaca,
        // dan pelanggarannya hilang tanpa satu pun galat.
        detail: [
          { field: 'nama_object', pesan: 'Nama object harus diisi' },
          { field: 'nama_coverage', pesan: 'Nama coverage harus diisi' },
        ],
      })
    }

    return jsonResponse(200, {
      daftar: TAB_OUTSTANDING,
      baris: [BARIS],
      paginasi: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
      cari: '',
      portal: 'ASM',
    })
  })

  await renderLoaded()
  await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

  const form = await screen.findByRole('form', { name: 'Menambahkan Data Salvage' })

  await userEvent.type(within(form).getByLabelText(/Nomor Klaim/), 'PNC-700071')
  await userEvent.type(within(form).getByLabelText(/Nama Object/), 'x')
  await userEvent.type(within(form).getByLabelText(/Nama Coverage/), 'y')
  await userEvent.type(within(form).getByLabelText(/Jenis Salvage/), 'Besi Tua')
  await userEvent.click(within(form).getByRole('button', { name: 'Submit' }))

  // KEDUA pesan tampil sekaligus, bukan satu lalu satu lagi (`P-5`).
  expect(await screen.findByText('Nama object harus diisi')).toBeInTheDocument()
  expect(screen.getByText('Nama coverage harus diisi')).toBeInTheDocument()
})

it('permintaan daftar membawa header portal', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await vi.waitFor(() => {
    const terakhir = lastListCall()
    expect(terakhir).toBeDefined()

    const header = new Headers(terakhir?.init?.headers)
    expect(header.get('X-Portal')).toBe('ASM')
  })
})

it('kolom aksi Detail digambar pada SETIAP daftar, bukan hanya sebagian', async () => {
  stubDefaultFetch()
  await renderLoaded()

  // Salvage Outstanding berbaris KLAIM, dan tombolnya tetap ada — di layar lama pun
  // tombol Detail tersebar di seluruh grid.
  expect(await screen.findByRole('columnheader', { name: 'Aksi' })).toBeInTheDocument()
  expect(await screen.findByRole('button', { name: 'Detail' })).toBeInTheDocument()

  await userEvent.click(screen.getByRole('tab', { name: /Checker/ }))

  expect(await screen.findByRole('columnheader', { name: 'Aksi' })).toBeInTheDocument()
  expect(await screen.findByRole('button', { name: 'Detail' })).toBeInTheDocument()
})

it('daftar berbasis klaim membuka rincian dengan NOMOR KLAIM', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))
  await screen.findByRole('region', { name: 'Detail Salvage' })

  // Rute yang ditembak menyebut jenis kuncinya. Nomor klaim dan ID pengajuan adalah dua
  // ruang nilai yang berbeda; satu rute untuk keduanya akan menjawab "tidak ditemukan"
  // untuk nilai yang sebenarnya sah pada ruang yang lain.
  const hit = [...calls].reverse().find((c) => c.url.startsWith(`${PATH}/klaim/`))
  expect(hit?.url).toBe(`${PATH}/klaim/${encodeURIComponent(BARIS.no_klaim)}`)
})

it('klaim yang belum punya pengajuan membuka form Tambah, bukan panel rincian', async () => {
  stubFetch((url) => {
    if (url === META_PATH) return jsonResponse(200, METADATA)
    if (url === COUNTS_PATH) return jsonResponse(200, COUNTS)

    if (url.startsWith(`${PATH}/klaim/`)) {
      return jsonResponse(200, {
        ...DETAIL,
        id_salvage: '',
        no_klaim: BARIS.no_klaim,
        nama_object: 'Gudang Blok C',
        nama_coverage: 'Property All Risk',
        ada_pengajuan: false,
        barang: [],
        riwayat: [],
      })
    }

    return jsonResponse(200, {
      daftar: TAB_OUTSTANDING,
      baris: [BARIS],
      paginasi: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
      cari: '',
      portal: 'ASM',
    })
  })

  await renderLoaded()
  await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))

  // Bukan panel rincian — form pengajuan, seperti di layar lama.
  const form = await screen.findByRole('form', { name: 'Menambahkan Data Salvage' })
  expect(screen.queryByRole('region', { name: 'Detail Salvage' })).not.toBeInTheDocument()

  // Isian yang berasal dari klaimnya sudah terisi dan TIDAK dapat diubah: pengajuan tidak
  // boleh tersimpan atas klaim yang berbeda dari baris yang diklik.
  const nomor = within(form).getByLabelText(/Nomor Klaim/)
  expect(nomor).toHaveValue(BARIS.no_klaim)
  expect(nomor).toHaveAttribute('readonly')

  expect(within(form).getByLabelText(/Nama Object/)).toHaveValue('Gudang Blok C')

  // Grid riwayat tetap digambar meski kosong — kekosongannya adalah keterangan.
  const riwayat = within(form).getByRole('region', { name: 'Detail History Salvage' })
  expect(within(riwayat).getByText(/belum pernah diajukan salvage/)).toBeInTheDocument()
})

it('grid Detail History Salvage menggambar pengajuan sebelumnya milik klaim itu', async () => {
  stubFetch((url) => {
    if (url === META_PATH) return jsonResponse(200, METADATA)
    if (url === COUNTS_PATH) return jsonResponse(200, COUNTS)

    if (url.startsWith(`${PATH}/klaim/`)) {
      return jsonResponse(200, {
        ...DETAIL,
        id_salvage: '',
        no_klaim: BARIS.no_klaim,
        ada_pengajuan: false,
        barang: [],
        riwayat: [
          {
            id_salvage: '77',
            tanggal_input: '2026-04-19',
            no_klaim: BARIS.no_klaim,
            pic: 'ELLENSUPRIYATI',
            nilai_minimum: '10000',
            posisi_salvage: 'Salvage Waive',
          },
        ],
      })
    }

    return jsonResponse(200, {
      daftar: TAB_OUTSTANDING,
      baris: [BARIS],
      paginasi: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
      cari: '',
      portal: 'ASM',
    })
  })

  await renderLoaded()
  await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))

  const form = await screen.findByRole('form', { name: 'Menambahkan Data Salvage' })
  const riwayat = within(form).getByRole('region', { name: 'Detail History Salvage' })

  expect(within(riwayat).getByText('ELLENSUPRIYATI')).toBeInTheDocument()

  // Posisinya berupa KALIMAT, bukan kode. Pemetaannya berbeda dari "Posisi Salvage" pada
  // panel rincian, meski keduanya berasal dari kolom yang sama.
  expect(within(riwayat).getByText('Salvage Waive')).toBeInTheDocument()
})

it('membuka panel Detail Salvage dan menutupnya kembali', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await userEvent.click(screen.getByRole('tab', { name: /Checker/ }))
  await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))

  const panel = await screen.findByRole('region', { name: 'Detail Salvage' })

  const hit = [...calls].reverse().find((c) => c.url.startsWith(`${PATH}/pengajuan/`))
  expect(hit?.url).toBe(`${PATH}/pengajuan/103`)

  expect(within(panel).getByText('Gudang Cakung')).toBeInTheDocument()
  expect(within(panel).getByText('Menunggu keputusan checker')).toBeInTheDocument()

  // Grid barang ikut tergambar, dan jumlah serta satuannya dirangkai DI LAYAR — di layar
  // lama keduanya dirangkai di dalam SQL, dan sejak itu jumlahnya berhenti menjadi angka.
  expect(within(panel).getByText('Drum 200L')).toBeInTheDocument()
  expect(within(panel).getByText('12 unit')).toBeInTheDocument()

  await userEvent.click(within(panel).getByRole('button', { name: 'Tutup' }))
  expect(screen.queryByRole('region', { name: 'Detail Salvage' })).not.toBeInTheDocument()
})

it('menggambar posisi salvage sebagai NAMA, bukan sebagai kodenya', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await userEvent.click(screen.getByRole('tab', { name: /Checker/ }))
  await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))

  const panel = await screen.findByRole('region', { name: 'Detail Salvage' })

  // Namanya yang dibaca; kodenya tetap tampil kecil di sebelahnya untuk penelusuran ke
  // Pega — bukan menggantikan namanya.
  expect(within(panel).getByText('Checker')).toBeInTheDocument()
  expect(within(panel).getByText('kode 3')).toBeInTheDocument()
})

it('berpindah daftar menutup panel Detail yang sedang terbuka', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await userEvent.click(screen.getByRole('tab', { name: /Checker/ }))
  await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))
  await screen.findByRole('region', { name: 'Detail Salvage' })

  await userEvent.click(screen.getByRole('tab', { name: /Salvage Outstanding/ }))

  // Rincian yang tetap terbuka di bawah daftar lain akan terbaca seolah milik baris yang
  // sekarang tampil.
  expect(screen.queryByRole('region', { name: 'Detail Salvage' })).not.toBeInTheDocument()
})

it('menjelaskan pengajuan yang tidak ada pada entitas yang sedang dipilih', async () => {
  stubFetch((url) => {
    if (url === META_PATH) return jsonResponse(200, METADATA)
    if (url === COUNTS_PATH) return jsonResponse(200, COUNTS)

    if (url === DETAIL_PATH) {
      return jsonResponse(404, {
        kode: 'pengajuan_tidak_ditemukan',
        pesan:
          'Pengajuan salvage tidak ditemukan pada entitas yang sedang dipilih. ' +
          'Periksa pilihan portal di bilah atas.',
      })
    }

    return jsonResponse(200, {
      daftar: TAB_CHECKER,
      baris: [BARIS_CHECKER],
      paginasi: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
      cari: '',
      portal: 'ASM',
    })
  })

  await renderLoaded()

  await userEvent.click(screen.getByRole('tab', { name: /Checker/ }))
  await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))

  expect(await screen.findByText('Detail tidak dapat dibuka')).toBeInTheDocument()
  expect(screen.getByText(/Periksa pilihan portal/)).toBeInTheDocument()
})
