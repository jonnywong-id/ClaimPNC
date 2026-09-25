import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { CountsResponse, MetadataResponse, SalvageRow, Tab } from './types'

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
