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
  membuka_form_sunting: false,
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
  membuka_form_sunting: false,
}

/**
 * Daftar "Rejected Checker" — satu-satunya yang membuka FORM SUNTING, bukan panel baca.
 *
 * Isinya pengajuan yang checker kembalikan kepada PIC, dan satu-satunya tindakan yang
 * masuk akal di sana adalah memperbaikinya lalu mengirim ulang.
 */
const TAB_REJECTED: Tab = {
  kode: 'rejected-checker',
  nama: 'Rejected Checker',
  keterangan: 'Pengajuan yang DIKEMBALIKAN checker kepada PIC.',
  kolom: TAB_CHECKER.kolom,
  label_pencarian: TAB_CHECKER.label_pencarian,
  pencarian_cocok_persis: false,
  kunci_rincian: 'pengajuan',
  membuka_form_sunting: true,
}

const METADATA: MetadataResponse = {
  daftar: [TAB_OUTSTANDING, TAB_CHECKER],
  daftar_bawaan: 'outstanding',
  pilihan_status_salvage: [
    { kode: '2', label: 'Salvage DiTerima' },
    { kode: '4', label: 'Waive Salvage' },
  ],
  pilihan_mata_uang: [
    { kode: 'IDR', label: 'IDR' },
    { kode: 'USD', label: 'USD' },
  ],
  kolom_berkas_unggahan: ['item', 'quantity', 'satuan', 'remarks'],
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
  id_object: '1',
  id_coverage: '1',
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

  // Panel rincian dibuka lewat ID PENGAJUAN, dan jalur itu tidak menggambar form Tambah
  // sama sekali — kedua daftar pilihannya karena itu kosong. Lihat DetailResponse.
  pilihan_objek: [],
  pilihan_coverage: [],

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
    const address = new URL(url, 'https://uji.invalid')
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
 * Penantiannya pada tabel ringkas, bukan pada judul layar: judulnya sudah ada sejak
 * penggambaran pertama, sementara tabel ringkas baru tiba bersama jawaban `/ringkas`.
 */
async function renderLoaded() {
  renderPage()
  await screen.findByRole('table', {
    name: 'Ringkasan jumlah pengajuan per status salvage',
  })
}

/**
 * pilihDaftar berpindah daftar lewat TABEL RINGKAS.
 *
 * Bilah tab dihapus 2026-10-03 atas permintaan Work Owner, dan tabel ringkas menjadi
 * satu-satunya navigasi — sebagaimana di Pega, tempat barisnyalah yang mengirim
 * `Param.tipe`/`Param.tipe2`. Uji berpindah daftar lewat jalan yang sama dengan
 * penggunanya; menembak kode daftar langsung ke alamat akan melewatkan justru bagian
 * yang paling mungkin rusak.
 */
async function pilihDaftar(nama: string) {
  const ringkas = screen.getByRole('table', {
    name: 'Ringkasan jumlah pengajuan per status salvage',
  })
  await userEvent.click(within(ringkas).getByRole('button', { name: nama }))
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

  await pilihDaftar('Checker')

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

  await pilihDaftar('Checker')
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

  await pilihDaftar('Checker')
  await screen.findByRole('columnheader', { name: 'Nilai Pengajuan PIC' })

  await userEvent.type(screen.getByLabelText(/CARI NO KLAIM ATAU PIC/i), 'PNC-70')

  expect(await screen.findByText(/COCOK PERSIS/)).toBeInTheDocument()
})

/** Kolom yang SELALU kosong digambar sebagai tanda pisah, bukan sel kosong. */
it('menggambar sel kosong sebagai tanda pisah', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await pilihDaftar('Checker')
  await screen.findByRole('columnheader', { name: 'Catatan' })

  const baris = await screen.findByText('Drum Kosong')
  const sel = baris.closest('tr')
  expect(sel).not.toBeNull()
  expect(within(sel as HTMLElement).getAllByText('—').length).toBeGreaterThan(0)
})

// Keputusan Work Owner 2026-10-06: panel selisih terencana DIHAPUS dari seluruh layar.
//
// Daftarnya tetap hidup di kode Go untuk uji kesetaraan gerbang 1 (`D-54`); yang berubah
// adalah ia berhenti menjadi isi layar.
it('tidak lagi menggambar panel selisih terencana', async () => {
  stubDefaultFetch()
  await renderLoaded()

  expect(
    screen.queryByText(/Perbedaan yang disengaja terhadap layar Pega/),
  ).not.toBeInTheDocument()
  expect(screen.queryByText(/Kolom "Catatan" SELALU KOSONG/)).not.toBeInTheDocument()
})

it('berpindah daftar mengosongkan kata kunci pencarian', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await userEvent.type(screen.getByLabelText(/CARI NO KLAIM/i), 'PNC-700071')
  await vi.waitFor(() => expect(lastListCall()?.url).toContain('cari=PNC-700071'))

  await pilihDaftar('Checker')

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

  // Keduanya WAJIB begitu daftarnya terisi; diisi supaya yang diuji di sini benar-benar
  // penandaan pelanggaran dari server, bukan penahanan oleh peramban.
  await userEvent.selectOptions(within(form).getByLabelText(/Mata Uang/), 'IDR')
  await userEvent.selectOptions(within(form).getByLabelText(/Status Salvage/), '2')

  await userEvent.click(
    within(form).getByRole('button', { name: 'Submit Pengajuan Salvage' }),
  )

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

  await pilihDaftar('Checker')

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
  // Menyembunyikannya membuat keadaan itu tidak dapat dibedakan dari grid yang gagal dimuat.
  const riwayat = within(form).getByRole('region', { name: 'Detail History Salvage' })
  expect(within(riwayat).getByText('Data Tidak Ada')).toBeInTheDocument()
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

  await pilihDaftar('Checker')
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

  await pilihDaftar('Checker')
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

  await pilihDaftar('Checker')
  await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))
  await screen.findByRole('region', { name: 'Detail Salvage' })

  await pilihDaftar('Outstanding')

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

  await pilihDaftar('Checker')
  await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))

  expect(await screen.findByText('Detail tidak dapat dibuka')).toBeInTheDocument()
  expect(screen.getByText(/Periksa pilihan portal/)).toBeInTheDocument()
})

// ============================================================================
// KOLOM NOMOR KLAIM, NAMA OBJECT, DAN NAMA COVERAGE PADA FORM TAMBAH
// ============================================================================
//
// Ketiganya bekerja sebagai SATU RANGKAIAN di layar lama: mengetik Nomor Klaim memicu
// `SetDataDetailSalvage_act`, dan kedua kolom di bawahnya adalah `pxAutoComplete` yang
// membaca hasilnya. Uji di bawah menguji rangkaian itu, bukan ketiga kolomnya
// sendiri-sendiri â€” yang paling mudah salah justru hubungan antarkolomnya.

/** Klaim contoh beserta objek dan coverage-nya, untuk jalur tombol "Tambah". */
const KLAIM_DENGAN_PILIHAN: DetailResponse = {
  ...DETAIL,
  id_salvage: '',
  no_klaim: 'PNC-700071',
  nama_object: '',
  nama_coverage: '',
  id_object: '',
  id_coverage: '',
  ada_pengajuan: false,
  barang: [],
  riwayat: [],

  // Urutannya sudah MENURUT NAMA, sebagaimana dikirim server — uji di bawah memeriksa
  // daftar saran apa adanya, jadi urutan di sini harus sama dengan yang dijanjikan kueri.
  pilihan_objek: [
    { id: '1', nama: 'Gudang Blok C' },
    { id: '2', nama: 'Isi Gudang' },
  ],
  pilihan_coverage: [
    { id: '10002', nama: 'Gempa Bumi' },
    { id: '10001', nama: 'Kebakaran' },
    { id: '10003', nama: 'Pencurian' },
  ],
}

/**
 * optionsOf MEMBUKA daftar pilihan sebuah autocomplete, lalu membaca isinya.
 *
 * Ia harus membukanya lebih dulu, dan itu perbedaan yang menentukan dari bentuk
 * sebelumnya: dulu kolomnya memakai `<datalist>` yang isinya selalu ada di DOM meski
 * tidak pernah terlihat, sehingga uji dapat membacanya tanpa menyentuh kolomnya sama
 * sekali â€” dan uji itu lulus bahkan seandainya daftarnya tidak pernah dapat dibuka
 * pengguna. Sekarang yang dibaca adalah daftar yang BENAR-BENAR tergambar.
 *
 * Nama diambil dari simpul teks pertama tiap baris; penunjuk di sebelahnya tergambar
 * sebagai `<span>` tersendiri dan sengaja tidak ikut terbaca.
 */
async function optionsOf(input: HTMLElement): Promise<string[]> {
  await userEvent.click(input)

  const listID = input.getAttribute('aria-controls')
  if (listID === null) return []

  const list = document.getElementById(listID)
  if (list === null) return []

  return [...list.querySelectorAll('[role="option"]')].map(
    (option) => option.childNodes[0]?.textContent?.trim() ?? '',
  )
}

/** Peladen tiruan untuk jalur tombol "Tambah": pencarian klaim menjawab pilihan. */
function stubLookupFetch(answerClaim: (no: string) => Response) {
  stubFetch((url) => {
    if (url === META_PATH) return jsonResponse(200, METADATA)
    if (url === COUNTS_PATH) return jsonResponse(200, COUNTS)

    if (url.startsWith(`${PATH}/klaim/`)) {
      return answerClaim(decodeURIComponent(url.slice(`${PATH}/klaim/`.length)))
    }

    return jsonResponse(200, {
      daftar: TAB_OUTSTANDING,
      baris: [BARIS],
      paginasi: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
      cari: '',
      portal: 'ASM',
    })
  })
}

/**
 * Peladen tiruan untuk jalur unggahan dokumen.
 *
 * Pencarian klaim ikut dijawab supaya nomor klaimnya dapat terisi lebih dulu — tanpa itu
 * tombol Submit pada modal tetap mati, dan ujinya tidak pernah sampai ke permintaannya.
 */
function stubUnggahanDokumen(answerUpload: () => Response) {
  stubFetch((url) => {
    if (url === META_PATH) return jsonResponse(200, METADATA)
    if (url === COUNTS_PATH) return jsonResponse(200, COUNTS)
    if (url === `${PATH}/dokumen`) return answerUpload()

    if (url.startsWith(`${PATH}/klaim/`)) {
      return jsonResponse(200, KLAIM_DENGAN_PILIHAN)
    }

    return jsonResponse(200, {
      daftar: TAB_OUTSTANDING,
      baris: [BARIS],
      paginasi: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
      cari: '',
      portal: 'ASM',
    })
  })
}

/** openTambah membuka form "Menambahkan Data Salvage" lewat tombol Tambah. */
async function openTambah(): Promise<HTMLElement> {
  await renderLoaded()
  await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
  return screen.findByRole('form', { name: 'Menambahkan Data Salvage' })
}

it('mengetik Nomor Klaim memuat pilihan objek dan coverage klaim itu', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  // Sebelum nomor klaim diketik, kedua kolom belum menawarkan apa pun â€” belum ada klaim
  // yang menjadi acuannya.
  expect(await optionsOf(within(form).getByLabelText(/Nama Object/))).toEqual([])

  await userEvent.type(within(form).getByLabelText(/Nomor Klaim/), 'PNC-700071')

  // Pencariannya dipicu saat kolomnya DITINGGALKAN, bukan pada setiap ketukan papan
  // ketik â€” sama seperti event `change` pada kotak teks di layar lama.
  await userEvent.tab()

  expect(await screen.findByText(/ditemukan\./)).toBeInTheDocument()
  expect(await optionsOf(within(form).getByLabelText(/Nama Object/))).toEqual([
    'Gudang Blok C',
    'Isi Gudang',
  ])
})

it('pencarian klaim dikirim SEKALI, bukan satu permintaan per huruf', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  await userEvent.type(within(form).getByLabelText(/Nomor Klaim/), 'PNC-700071')
  await userEvent.tab()
  await screen.findByText(/ditemukan\./)

  const lookups = calls.filter((call) => call.url.startsWith(`${PATH}/klaim/`))
  expect(lookups).toHaveLength(1)
  expect(lookups[0]?.url).toBe(`${PATH}/klaim/PNC-700071`)
})

/**
 * Kedua kolom bawah TIDAK saling terikat, dan itu perilaku layar lama apa adanya.
 *
 * `GetDataSalvageObjectCoverageForOs` tidak mengambil `OBJECTID` sama sekali — daftar
 * coverage ditarik per KLAIM, bukan per objek. Uji ini menjaga keadaan itu supaya
 * penyempitan yang tampak masuk akal tidak diam-diam diperkenalkan kembali.
 */
it('memilih objek TIDAK mempersempit daftar coverage', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  await userEvent.type(within(form).getByLabelText(/Nomor Klaim/), 'PNC-700071')
  await userEvent.tab()
  await screen.findByText(/ditemukan\./)

  const coverage = within(form).getByLabelText(/Nama Coverage/)
  const semua = ['Gempa Bumi', 'Kebakaran', 'Pencurian']

  expect(await optionsOf(coverage)).toEqual(semua)

  await userEvent.type(within(form).getByLabelText(/Nama Object/), 'Isi Gudang')

  // Tetap ketiganya. Mempersempitnya akan menyembunyikan pilihan yang di Pega tersedia.
  expect(await optionsOf(coverage)).toEqual(semua)
})

it('mengganti objek TIDAK membuang coverage yang sudah dipilih', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  await userEvent.type(within(form).getByLabelText(/Nomor Klaim/), 'PNC-700071')
  await userEvent.tab()
  await screen.findByText(/ditemukan\./)

  const objek = within(form).getByLabelText(/Nama Object/)
  const coverage = within(form).getByLabelText(/Nama Coverage/)

  await userEvent.type(objek, 'Gudang Blok C')
  await userEvent.type(coverage, 'Gempa Bumi')

  await userEvent.clear(objek)
  await userEvent.type(objek, 'Isi Gudang')

  // Mengosongkannya akan menghapus ketikan orang atas dasar keterikatan yang tidak
  // pernah ada di layar lama.
  expect(coverage).toHaveValue('Gempa Bumi')
})

it('nama di LUAR daftar tetap diterima, sesuai autocomplete layar lama', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  await userEvent.type(within(form).getByLabelText(/Nomor Klaim/), 'PNC-700071')
  await userEvent.tab()
  await screen.findByText(/ditemukan\./)

  // `pyAllowFreeFormInput=true` pada kedua autocomplete layar lama: daftarnya membantu,
  // tetapi tidak mengunci.
  const objek = within(form).getByLabelText(/Nama Object/)
  await userEvent.type(objek, 'Objek Di Luar Daftar')
  expect(objek).toHaveValue('Objek Di Luar Daftar')
})

it('nomor klaim yang tidak ada dijelaskan, bukan digambar sebagai kerusakan', async () => {
  stubLookupFetch(() =>
    jsonResponse(404, { kode: 'tidak_ditemukan', pesan: 'Klaim tidak ditemukan.' }),
  )
  const form = await openTambah()

  await userEvent.type(within(form).getByLabelText(/Nomor Klaim/), 'PNC-999999')
  await userEvent.tab()

  // Salah ketik nomor klaim adalah hal yang lazim. Ia dijawab sebagai keterangan, dan
  // keterangannya menyebut nomornya kembali supaya "klaim ini tidak ada" dapat dibedakan
  // dari "saya salah satu huruf".
  const keterangan = await screen.findByText(/tidak ditemukan di portal ini/)

  // Nomornya disebut KEMBALI di dalam keterangannya — tanpa itu, "klaim ini tidak ada"
  // tidak dapat dibedakan dari "saya salah satu huruf".
  expect(keterangan).toHaveTextContent('PNC-999999')

  // Formnya TIDAK ditutup, dan kedua kolom di bawahnya tetap dapat diketik.
  expect(within(form).getByLabelText(/Nama Object/)).toBeEnabled()
})

it('klaim tanpa objek dinyatakan apa adanya, dan formnya tetap dapat diisi', async () => {
  stubLookupFetch(() =>
    jsonResponse(200, {
      ...KLAIM_DENGAN_PILIHAN,
      pilihan_objek: [],
      pilihan_coverage: [],
    }),
  )
  const form = await openTambah()

  await userEvent.type(within(form).getByLabelText(/Nomor Klaim/), 'PNC-700071')
  await userEvent.tab()

  expect(
    await screen.findByText(/objek dan coverage-nya belum terisi/),
  ).toBeInTheDocument()

  const objek = within(form).getByLabelText(/Nama Object/)
  await userEvent.type(objek, 'Diketik Sendiri')
  expect(objek).toHaveValue('Diketik Sendiri')
})

it('form dari baris klaim TIDAK mengunci Nama Object dan Nama Coverage', async () => {
  stubFetch((url) => {
    if (url === META_PATH) return jsonResponse(200, METADATA)
    if (url === COUNTS_PATH) return jsonResponse(200, COUNTS)

    // Klaim yang belum pernah diajukan salvage: kedua namanya KOSONG, karena memang
    // belum ada pengajuan yang pernah memilihnya.
    if (url.startsWith(`${PATH}/klaim/`)) {
      return jsonResponse(200, { ...KLAIM_DENGAN_PILIHAN, no_klaim: BARIS.no_klaim })
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

  // Nomor klaimnya tetap terkunci â€” pengajuan tidak boleh tersimpan atas klaim yang
  // berbeda dari baris yang diklik.
  expect(within(form).getByLabelText(/Nomor Klaim/)).toHaveAttribute('readonly')

  // Kedua kolom lainnya TIDAK. Mengunci keduanya dalam keadaan kosong membuat form ini
  // tidak pernah dapat disimpan, sementara keduanya wajib diisi di server.
  const objek = within(form).getByLabelText(/Nama Object/)
  expect(objek).not.toHaveAttribute('readonly')
  expect(await optionsOf(objek)).toEqual(['Gudang Blok C', 'Isi Gudang'])

  await userEvent.type(objek, 'Gudang Blok C')
  expect(objek).toHaveValue('Gudang Blok C')
})

it('pilihan pada jalur baris klaim diambil dari jawaban yang SUDAH di tangan', async () => {
  stubFetch((url) => {
    if (url === META_PATH) return jsonResponse(200, METADATA)
    if (url === COUNTS_PATH) return jsonResponse(200, COUNTS)
    if (url.startsWith(`${PATH}/klaim/`)) {
      return jsonResponse(200, { ...KLAIM_DENGAN_PILIHAN, no_klaim: BARIS.no_klaim })
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
  await screen.findByRole('form', { name: 'Menambahkan Data Salvage' })

  // SATU permintaan, bukan dua. Halaman sudah menembak rincian klaimnya untuk memutuskan
  // form inilah yang digambar; form yang mencarinya sendiri lagi berarti permintaan kedua
  // untuk jawaban yang sama persis.
  const lookups = calls.filter((call) => call.url.startsWith(`${PATH}/klaim/`))
  expect(lookups).toHaveLength(1)
})


it('coverage kembar ditawarkan DUA KALI, sebanyak barisnya', async () => {
  stubLookupFetch(() =>
    jsonResponse(200, {
      ...KLAIM_DENGAN_PILIHAN,

      // Baris kembar ini NYATA, bukan karangan: kueri coverage tidak menyaring objek,
      // dan `COVERAGEID` adalah kode JENIS jaminan — sehingga klaim yang dua objeknya
      // sama-sama berjaminan kebakaran mengembalikan dua baris yang identik seluruhnya.
      pilihan_coverage: [
        { id: '10001', nama: 'Kebakaran' },
        { id: '10001', nama: 'Kebakaran' },
      ],
    }),
  )
  const form = await openTambah()

  await userEvent.type(within(form).getByLabelText(/Nomor Klaim/), 'PNC-700071')
  await userEvent.tab()
  await screen.findByText(/ditemukan\./)

  // Dua baris yang terbaca sama persis tidak menambah satu pun keterangan — selain
  // membuat React menandai kunci ganda pada datalist-nya.
  // Keduanya digambar APA ADANYA, dan itu kebalikan dari perilaku sebelumnya.
  //
  // Dulu nama yang berulang dibuang supaya `<datalist>` tidak memuat dua pilihan
  // bernilai sama. Yang hilang bukan hanya barisnya: pada objek, dua baris bernama sama
  // membawa PENUNJUK yang berbeda, dan membuang salah satunya membuat objek itu tidak
  // pernah dapat dipilih. Daftar sekarang memilih barisnya, bukan namanya, sehingga
  // keduanya berdiri sendiri.
  expect(await optionsOf(within(form).getByLabelText(/Nama Coverage/))).toEqual([
    'Kebakaran',
    'Kebakaran',
  ])
})


// ============================================================================
// GRID "DETAIL ITEM SALVAGE"
// ============================================================================
//
// Barisnya diisi LANGSUNG di dalam tabel â€” bentuk aslinya di
// `Section/TambahData_Salvage-Section.xml:8691`, tempat page list `DetailSalvage.Data`
// digambar sebagai grid ber-`pxTextInput` dengan tautan Tambah dan Hapus di atasnya.
// Unggahan CSV adalah jalur KEDUA, bukan satu-satunya.

/** isiWajib mengisi seluruh kolom bertanda wajib supaya form dapat dikirim. */
async function isiWajib(form: HTMLElement) {
  await userEvent.type(within(form).getByLabelText(/Nomor Klaim/), 'PNC-700071')
  await userEvent.type(within(form).getByLabelText(/Nama Object/), 'Gudang Blok C')
  await userEvent.type(within(form).getByLabelText(/Nama Coverage/), 'Kebakaran')
  await userEvent.type(within(form).getByLabelText(/Jenis Salvage/), 'Besi Tua')

  // Mata Uang dan Status Salvage bertanda WAJIB begitu daftarnya terisi, sehingga
  // peramban menahan pengiriman selama keduanya kosong.
  await userEvent.selectOptions(within(form).getByLabelText(/Mata Uang/), 'IDR')
  await userEvent.selectOptions(within(form).getByLabelText(/Status Salvage/), '2')
}

it('tombol Tambah menyisipkan baris yang dapat langsung diketik', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  const grid = within(form).getByRole('table', { name: 'Detail Item Salvage' })

  // Sebelum ditambah, gridnya tetap digambar â€” kekosongannya adalah keterangan, bukan
  // ketiadaan.
  expect(within(grid).getByText('Data Tidak Ada')).toBeInTheDocument()

  await userEvent.click(within(form).getByRole('button', { name: '+ Tambah' }))

  await userEvent.type(within(grid).getByLabelText('Nama Item baris 1'), 'Drum 200L')
  await userEvent.type(within(grid).getByLabelText('Jumlah Item baris 1'), '12')
  await userEvent.type(within(grid).getByLabelText('Remark baris 1'), 'kondisi penyok')

  expect(within(grid).getByLabelText('Nama Item baris 1')).toHaveValue('Drum 200L')
  expect(within(grid).getByLabelText('Jumlah Item baris 1')).toHaveValue('12')
})

it('baris yang diketik di grid IKUT TERKIRIM saat disimpan', async () => {
  stubFetch((url, init) => {
    if (url === META_PATH) return jsonResponse(200, METADATA)
    if (url === COUNTS_PATH) return jsonResponse(200, COUNTS)

    if (url === PATH && init?.method === 'POST') {
      return jsonResponse(200, {
        id_salvage: '501',
        jumlah_detail_item: 1,
        pesan: 'Pengajuan tersimpan.',
        portal: 'ASM',
      })
    }

    if (url.startsWith(`${PATH}/klaim/`)) {
      return jsonResponse(200, KLAIM_DENGAN_PILIHAN)
    }

    return jsonResponse(200, {
      daftar: TAB_OUTSTANDING,
      baris: [BARIS],
      paginasi: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
      cari: '',
      portal: 'ASM',
    })
  })

  const form = await openTambah()
  await isiWajib(form)

  const grid = within(form).getByRole('table', { name: 'Detail Item Salvage' })
  await userEvent.click(within(form).getByRole('button', { name: '+ Tambah' }))
  await userEvent.type(within(grid).getByLabelText('Nama Item baris 1'), 'Drum 200L')
  await userEvent.type(within(grid).getByLabelText('Jumlah Item baris 1'), '12')

  await userEvent.click(
    within(form).getByRole('button', { name: 'Submit Pengajuan Salvage' }),
  )

  const simpan = calls.find((call) => call.url === PATH && call.init?.method === 'POST')
  expect(simpan).toBeDefined()

  const badan = JSON.parse(String(simpan?.init?.body)) as {
    detail_item_salvage: { nama_item: string; jumlah_item: string }[]
    id_object: string
    id_coverage: string
  }

  expect(badan.detail_item_salvage).toEqual([
    { nama_item: 'Drum 200L', jumlah_item: '12', satuan: '', remark: '' },
  ])

  // Sekalian membuktikan penunjuk objek dan coverage ikut terkirim â€” sebelum perbaikan
  // ini keduanya selalu kosong.
  expect(badan.id_object).toBe('1')
  expect(badan.id_coverage).toBe('10001')
})

it('tautan Hapus membuang baris terakhir', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  const grid = within(form).getByRole('table', { name: 'Detail Item Salvage' })
  const tambah = within(form).getByRole('button', { name: '+ Tambah' })

  await userEvent.click(tambah)
  await userEvent.click(tambah)
  await userEvent.type(within(grid).getByLabelText('Nama Item baris 1'), 'Pertama')
  await userEvent.type(within(grid).getByLabelText('Nama Item baris 2'), 'Kedua')

  await userEvent.click(within(form).getByRole('button', { name: 'Hapus' }))

  // Yang tersisa adalah baris PERTAMA; baris terakhir yang dibuang.
  expect(within(grid).getByLabelText('Nama Item baris 1')).toHaveValue('Pertama')
  expect(within(grid).queryByLabelText('Nama Item baris 2')).not.toBeInTheDocument()
})

it('grid punya LIMA kolom, dua di antaranya mati', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  const grid = within(form).getByRole('table', { name: 'Detail Item Salvage' })
  const judul = within(grid)
    .getAllByRole('columnheader')
    .map((kolom) => kolom.textContent)

  // Persis ketiganya. "Satuan" ADA di berkas CSV tetapi tidak digambar — begitu pula di
  // layar lama, dan menambah kolom di sini berarti menyimpang darinya.
  // "Harga Total" dan "Upload file" ADA di layar yang berjalan, dan karena itu digambar.
  //
  // Keduanya tetap MATI: `INSERT_SALVAGE_DETAILS` tidak punya satu pun parameter untuk
  // keduanya. Menghilangkannya membuat ketiadaannya tidak terlihat; membuatnya aktif
  // membuat orang mengetik nilai yang diam-diam hilang saat disimpan.
  //
  // "Satuan" tetap tidak digambar: ia ada di berkas CSV, tetapi tidak di layar mana pun.
  expect(judul).toEqual([
    'Nama Item',
    'Jumlah Item',
    'Remark',
    'Harga Total',
    'Upload file',
  ])
})

it('kedua kolom mati tidak dapat diisi', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  const grid = within(form).getByRole('table', { name: 'Detail Item Salvage' })
  await userEvent.click(within(form).getByRole('button', { name: '+ Tambah' }))

  expect(within(grid).getByLabelText('Harga Total baris 1')).toBeDisabled()
  expect(within(grid).getByRole('button', { name: 'Upload file' })).toBeDisabled()
})

// ============================================================================
// MODAL "UploadDocument_Salvage"
// ============================================================================
//
// Tombol "Upload file" MEMBUKA MODAL, tidak langsung memilih berkas — ia local action
// `UploadDocument_Salvage` di layar lama. Flow Action bernama itu tidak ada di export,
// sehingga bentuk modalnya disalin dari tangkapan layar sistem yang berjalan.

it('tombol Upload file membuka modal beserta kedua batas unggahannya', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

  await userEvent.click(within(form).getByRole('button', { name: 'Upload File Pendukung Lain' }))

  const modal = await screen.findByRole('dialog', { name: 'UploadDocument_Salvage' })

  // Kedua batas disalin kata demi kata dari `Section/SalvageUploadDocumentAll`. Huruf
  // besarnya datang dari gaya layar, bukan dari teksnya — teks yang diuji di sini teks
  // rule-nya.
  expect(within(modal).getByText('Max Doc : 5 Doc.')).toBeInTheDocument()
  expect(within(modal).getByText('Max Size /file : 1 MB')).toBeInTheDocument()

  expect(within(modal).getByRole('button', { name: 'Select file(s)' })).toBeEnabled()
  expect(within(modal).getByRole('button', { name: 'Cancel' })).toBeEnabled()

  // Submit MATI selama belum ada berkas terpilih.
  expect(within(modal).getByRole('button', { name: 'Submit' })).toBeDisabled()
})

it('Cancel menutup modal unggahan tanpa menyentuh form', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  await userEvent.type(within(form).getByLabelText(/Jenis Salvage/), 'Besi Tua')
  await userEvent.click(within(form).getByRole('button', { name: 'Upload File Pendukung Lain' }))

  const modal = await screen.findByRole('dialog', { name: 'UploadDocument_Salvage' })
  await userEvent.click(within(modal).getByRole('button', { name: 'Cancel' }))

  expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

  // Isian yang sudah diketik TIDAK hilang — modalnya tidak menyentuh form.
  expect(within(form).getByLabelText(/Jenis Salvage/)).toHaveValue('Besi Tua')
})

it('modal unggahan menolak berkas yang melewati 1 MB', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  await userEvent.click(within(form).getByRole('button', { name: 'Upload File Pendukung Lain' }))
  const modal = await screen.findByRole('dialog', { name: 'UploadDocument_Salvage' })

  // Berkas 2 MB. Menolaknya DI LAYAR menghemat perjalanan kirim yang sudah pasti ditolak
  // penyimpanan dokumen — pada koneksi kantor cabang itu berarti menunggu lama untuk
  // sebuah penolakan yang dapat diketahui sejak awal.
  const terlaluBesar = new File(['x'.repeat(2 * 1024 * 1024)], 'besar.pdf', {
    type: 'application/pdf',
  })

  const pemilih = modal.querySelector('input[type="file"]')
  expect(pemilih).not.toBeNull()
  await userEvent.upload(pemilih as HTMLInputElement, terlaluBesar)

  expect(within(modal).getByRole('alert')).toHaveTextContent('besar.pdf')
  expect(within(modal).getByRole('button', { name: 'Submit' })).toBeDisabled()
})

it('modal unggahan menolak lebih dari lima dokumen', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  await userEvent.click(within(form).getByRole('button', { name: 'Upload File Pendukung Lain' }))
  const modal = await screen.findByRole('dialog', { name: 'UploadDocument_Salvage' })

  const enam = Array.from(
    { length: 6 },
    (_, index) => new File(['isi'], `berkas-${index + 1}.pdf`, { type: 'application/pdf' }),
  )

  const pemilih = modal.querySelector('input[type="file"]') as HTMLInputElement
  await userEvent.upload(pemilih, enam)

  expect(within(modal).getByRole('alert')).toHaveTextContent('Paling banyak 5 dokumen')
})

/**
 * Submit pada modal unggahan MENGIRIM berkasnya, dan menutup modal setelah tersimpan.
 *
 * Work Owner memutuskan 2026-10-03 bahwa jalur ini mengikuti Pega apa adanya — base64 ke
 * dalam Oracle — sehingga Submit tidak lagi menolak. Yang diperiksa di sini bentuk
 * permintaannya: satu POST `multipart/form-data` berisi nomor klaim, id pengajuan, dan
 * berkasnya; bukan satu permintaan per berkas.
 */
it('Submit pada modal unggahan MENGIRIM berkasnya sebagai satu permintaan', async () => {
  stubUnggahanDokumen(() =>
    jsonResponse(200, {
      dokumen: [
        {
          data_id: '900001',
          image_id: 'A1B2C3',
          nama_tersimpan: 'PNC-700071_SALVAGE_1.pdf',
          tertaut_salvage: false,
        },
      ],
      pesan: '1 dokumen tersimpan.',
      portal: 'ASM',
    }),
  )
  const form = await openTambah()
  await userEvent.type(within(form).getByLabelText(/Nomor Klaim/), 'PNC-700071')
  await userEvent.tab()
  await screen.findByText(/ditemukan\./)

  await userEvent.click(within(form).getByRole('button', { name: 'Upload File Pendukung Lain' }))
  const modal = await screen.findByRole('dialog', { name: 'UploadDocument_Salvage' })

  const pemilih = modal.querySelector('input[type="file"]') as HTMLInputElement
  await userEvent.upload(pemilih, [
    new File(['isi'], 'foto.pdf', { type: 'application/pdf' }),
    new File(['isi'], 'nota.pdf', { type: 'application/pdf' }),
  ])

  await userEvent.click(within(modal).getByRole('button', { name: 'Submit' }))

  // Modal menutup sendiri setelah tersimpan, dan kabarnya tertinggal di form — modalnya
  // sudah tidak ada, jadi kabar di dalamnya akan ikut hilang bersamanya.
  await screen.findByText('1 dokumen tersimpan.')
  expect(
    screen.queryByRole('dialog', { name: 'UploadDocument_Salvage' }),
  ).not.toBeInTheDocument()

  const unggahan = calls.filter((call) => call.url === `${PATH}/dokumen`)
  expect(unggahan).toHaveLength(1)

  const badan = unggahan[0]?.init?.body as FormData
  expect(badan.get('nomor_klaim')).toBe('PNC-700071')
  expect(badan.get('id_salvage')).toBe('')
  expect(badan.getAll('berkas')).toHaveLength(2)
})

/**
 * Penolakan server dinyatakan DI DALAM modal, dan berkasnya tetap terpilih.
 *
 * Menutup modal pada kegagalan akan membuang daftar berkas yang baru saja dipilih, dan
 * pengguna harus memilih ulang satu per satu untuk sesuatu yang mungkin hanya perlu
 * dicoba lagi.
 */
it('unggahan yang DITOLAK server tetap membuka modal beserta berkasnya', async () => {
  stubUnggahanDokumen(() =>
    jsonResponse(422, { kode: 'berkas_ditolak', pesan: 'Jenis berkas tidak diterima.' }),
  )
  const form = await openTambah()
  await userEvent.type(within(form).getByLabelText(/Nomor Klaim/), 'PNC-700071')
  await userEvent.tab()
  await screen.findByText(/ditemukan\./)

  await userEvent.click(within(form).getByRole('button', { name: 'Upload File Pendukung Lain' }))
  const modal = await screen.findByRole('dialog', { name: 'UploadDocument_Salvage' })

  const pemilih = modal.querySelector('input[type="file"]') as HTMLInputElement
  await userEvent.upload(pemilih, new File(['isi'], 'foto.pdf', { type: 'application/pdf' }))
  await userEvent.click(within(modal).getByRole('button', { name: 'Submit' }))

  expect(await within(modal).findByText('Dokumen tidak tersimpan')).toBeInTheDocument()
  expect(within(modal).getByText('Jenis berkas tidak diterima.')).toBeInTheDocument()
  expect(within(modal).getByText('foto.pdf')).toBeInTheDocument()
})

/**
 * Submit TIDAK dapat ditekan selama nomor klaimnya belum terisi.
 *
 * Dokumen melekat pada klaim — tanpa nomornya, tidak ada yang dapat dilampiri. Pengiriman
 * yang pasti ditolak server lebih baik dicegah sebelum isi berkasnya naik.
 */
it('Submit pada modal unggahan mati selama Nomor Klaim kosong', async () => {
  stubUnggahanDokumen(() => jsonResponse(200, { dokumen: [], pesan: '', portal: 'ASM' }))
  const form = await openTambah()

  await userEvent.click(within(form).getByRole('button', { name: 'Upload File Pendukung Lain' }))
  const modal = await screen.findByRole('dialog', { name: 'UploadDocument_Salvage' })

  const pemilih = modal.querySelector('input[type="file"]') as HTMLInputElement
  await userEvent.upload(pemilih, new File(['isi'], 'foto.pdf', { type: 'application/pdf' }))

  expect(within(modal).getByText('foto.pdf')).toBeInTheDocument()
  expect(within(modal).getByRole('button', { name: 'Submit' })).toBeDisabled()
  expect(calls.filter((call) => call.url === `${PATH}/dokumen`)).toHaveLength(0)
})

it('Mata Uang adalah daftar pilihan, bukan kotak ketik', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  const mataUang = within(form).getByLabelText(/Mata Uang/)
  expect(mataUang.tagName).toBe('SELECT')

  // Isinya datang dari server — `POOLDATA.CURRENCY`, tabel yang sama yang dibaca
  // `SelectCurrency_RD` di layar lama — bukan ditulis di layar.
  await userEvent.selectOptions(mataUang, 'USD')
  expect(mataUang).toHaveValue('USD')
})

it('kolom Tanggal Input tidak dapat diubah, sama seperti layar lama', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  // Di layar lama ia teks biasa, bukan kotak isian â€” jadi tidak ada isian berlabel itu
  // yang dapat ditemukan, hanya tulisannya.
  expect(within(form).queryByLabelText(/Tanggal Input/)).not.toBeInTheDocument()
  expect(within(form).getByText('Tanggal Input')).toBeInTheDocument()
})

it('peringatan Lokasi Salvage Di Jabodatabek digambar apa adanya', async () => {
  stubLookupFetch(() => jsonResponse(200, KLAIM_DENGAN_PILIHAN))
  const form = await openTambah()

  expect(
    within(form).getByText(
      /Perhatian Lokasi Salvage Di Jabodatabek diInput Jika Lokasi Salvage DI Jabodatabek/,
    ),
  ).toBeInTheDocument()
})


// ============================================================================
// DAFTAR "REJECTED CHECKER" â€” SATU-SATUNYA YANG MEMBUKA FORM SUNTING
// ============================================================================
//
// Isinya pengajuan yang checker kembalikan kepada PIC. Membukanya sebagai panel baca
// menjadikan daftar itu jalan buntu: barisnya menunggu diperbaiki, dan layarnya tidak
// menyediakan cara memperbaikinya.

/** Rincian pengajuan yang DITOLAK checker, lengkap dengan pilihan dan barangnya. */
const DITOLAK: DetailResponse = {
  ...DETAIL,
  id_salvage: '103',
  no_klaim: 'PNC-700071',
  ada_pengajuan: true,
  kode_posisi_salvage: '4',
  posisi_salvage: 'Rejected Checker',
  nama_object: 'Gudang Blok C',
  nama_coverage: 'Kebakaran',
  id_object: '1',
  id_coverage: '10001',
  jenis_salvage: 'Besi Tua',
  estimasi: '1800000',
  mata_uang: 'IDR',

  pilihan_objek: [
    { id: '1', nama: 'Gudang Blok C' },
    { id: '2', nama: 'Isi Gudang' },
  ],
  pilihan_coverage: [
    { id: '10001', nama: 'Kebakaran' },
    { id: '10002', nama: 'Gempa Bumi' },
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
      remark: 'penyok',
    },
  ],
}

/** Peladen tiruan yang menawarkan daftar Rejected Checker dan menjawab rinciannya. */
function stubRejectedFetch(onPost?: (body: string) => Response) {
  stubFetch((url, init) => {
    if (url === META_PATH) {
      return jsonResponse(200, {
        ...METADATA,
        daftar: [TAB_OUTSTANDING, TAB_REJECTED],
        daftar_bawaan: TAB_REJECTED.kode,
      })
    }
    if (url === COUNTS_PATH) return jsonResponse(200, COUNTS)

    if (url === PATH && init?.method === 'POST' && onPost !== undefined) {
      return onPost(String(init.body))
    }

    if (url.startsWith(`${PATH}/pengajuan/`)) return jsonResponse(200, DITOLAK)

    return jsonResponse(200, {
      daftar: TAB_REJECTED,
      baris: [BARIS_CHECKER],
      paginasi: { halaman: 1, ukuran: 20, total: 1, total_halaman: 1 },
      cari: '',
      portal: 'ASM',
    })
  })
}

/** openRejected membuka baris pertama daftar Rejected Checker. */
async function openRejected(): Promise<HTMLElement> {
  renderPage()
  await screen.findByRole('table', { name: 'Ringkasan jumlah pengajuan per status salvage' })
  await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))
  return screen.findByRole('form', { name: 'Menambahkan Data Salvage' })
}

it('baris Rejected Checker membuka FORM, bukan panel rincian', async () => {
  stubRejectedFetch()
  const form = await openRejected()

  expect(screen.queryByRole('region', { name: 'Detail Salvage' })).not.toBeInTheDocument()

  // Isian pengajuannya sudah terisi â€” itu yang membedakannya dari form pengajuan baru.
  expect(within(form).getByLabelText(/Nomor Klaim/)).toHaveValue('PNC-700071')
  expect(within(form).getByLabelText(/Jenis Salvage/)).toHaveValue('Besi Tua')
  expect(within(form).getByLabelText(/Total Nilai Salvage/)).toHaveValue('1800000')
})

it('Nama Object dan Nama Coverage TERISI beserta daftar pilihannya', async () => {
  stubRejectedFetch()
  const form = await openRejected()

  const objek = within(form).getByLabelText(/Nama Object/)
  const coverage = within(form).getByLabelText(/Nama Coverage/)

  // Inilah keluhan yang berulang: keduanya kosong karena rincian berbasis ID PENGAJUAN
  // tidak pernah memuat pilihannya. Sekarang jalur itu memuatnya pula.
  expect(objek).toHaveValue('Gudang Blok C')
  expect(coverage).toHaveValue('Kebakaran')

  expect(await optionsOf(objek)).toEqual(['Gudang Blok C', 'Isi Gudang'])
  expect(await optionsOf(coverage)).toEqual(['Kebakaran', 'Gempa Bumi'])
})

it('grid Detail Item Salvage terisi barang yang sudah tersimpan', async () => {
  stubRejectedFetch()
  const form = await openRejected()

  const grid = within(form).getByRole('table', { name: 'Detail Item Salvage' })
  expect(within(grid).getByLabelText('Nama Item baris 1')).toHaveValue('Drum 200L')
  expect(within(grid).getByLabelText('Jumlah Item baris 1')).toHaveValue('12')
  expect(within(grid).getByLabelText('Remark baris 1')).toHaveValue('penyok')
})

it('menyimpan dari Rejected Checker memakai mode ubah beserta ID pengajuannya', async () => {
  let dikirim = ''
  stubRejectedFetch((body) => {
    dikirim = body
    return jsonResponse(200, {
      id_salvage: '103',
      jumlah_detail_item: 1,
      pesan: 'Pengajuan diperbarui.',
      portal: 'ASM',
    })
  })

  const form = await openRejected()

  const grid = within(form).getByRole('table', { name: 'Detail Item Salvage' })
  await userEvent.clear(within(grid).getByLabelText('Jumlah Item baris 1'))
  await userEvent.type(within(grid).getByLabelText('Jumlah Item baris 1'), '9')

  // Status Salvage SENGAJA tidak terisi saat form sunting dibuka — lihat toFormState.
  // PIC harus memilihnya sendiri, dan tanpa itu peramban menahan pengiriman.
  expect(within(form).getByLabelText(/Status Salvage/)).toHaveValue('')
  await userEvent.selectOptions(within(form).getByLabelText(/Status Salvage/), '2')

  await userEvent.click(within(form).getByRole('button', { name: 'Submit Pengajuan Salvage' }))

  await vi.waitFor(() => expect(dikirim).not.toBe(''))

  const badan = JSON.parse(dikirim) as {
    mode: string
    id_salvage: string
    detail_item_salvage: { nama_item: string; jumlah_item: string }[]
  }

  // Mode UBAH, bukan insert â€” tanpa ini pengajuan yang diperbaiki akan tersimpan sebagai
  // pengajuan KEDUA, dan yang ditolak checker tetap tertinggal di antreannya.
  expect(badan.mode).toBe('ubah')
  expect(badan.id_salvage).toBe('103')

  // Daftar barang dikirim UTUH, bukan hanya yang berubah: server membuang seluruh barang
  // lama lebih dulu, sehingga isi grid inilah yang menjadi daftar barangnya.
  expect(badan.detail_item_salvage).toHaveLength(1)
  expect(badan.detail_item_salvage[0]?.jumlah_item).toBe('9')
})

it('form sunting mengunci Nomor Klaim dan tidak mencari klaim lagi', async () => {
  stubRejectedFetch()
  const form = await openRejected()

  // Klaimnya sudah pasti â€” ia milik pengajuan yang sedang diperbaiki.
  expect(within(form).getByLabelText(/Nomor Klaim/)).toHaveAttribute('readonly')

  // Dan tidak ada permintaan pencarian klaim sama sekali: pilihannya sudah dibawa
  // jawaban rincian yang baru saja diambil.
  expect(calls.filter((call) => call.url.startsWith(`${PATH}/klaim/`))).toHaveLength(0)
})

// ============================================================================
// KOTAK `DataDetail_Salvage` DAN DONAT RINGKASAN
// ============================================================================

/**
 * Kotak utama panel memuat LIMA BELAS isian, dalam urutan layar lama.
 *
 * Urutannya bukan selera: ia disalin dari urutan caption di
 * `Section/DataDetail_Salvage-Section.xml`. Petugas yang berpindah dari Pega mencari
 * isian di tempatnya, bukan dengan membaca seluruh kotak.
 */
it('panel rincian menggambar lima belas isian DataDetail_Salvage menurut urutannya', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await pilihDaftar('Checker')
  await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))

  const panel = await screen.findByRole('region', { name: 'Detail Salvage' })

  // `dl` PERTAMA di dalam panel adalah kotak `DataDetail_Salvage`; kelompok tambahan di
  // bawahnya punya judulnya sendiri dan tidak ikut terbaca di sini.
  const kotak = panel.querySelector('dl')
  expect(kotak).not.toBeNull()

  const label = [...(kotak?.querySelectorAll('dt') ?? [])].map((dt) =>
    (dt.textContent ?? '').trim(),
  )

  expect(label).toEqual([
    'Tanggal Input Salvage',
    'Tanggal Transfer GA',
    'Tanggal Akseptasi',
    'No Klaim',
    'No Akseptasi',
    'Nama Object',
    'Nama Coverage',
    'Jenis Salvage',
    'Quantity Salvage',
    'Lokasi Salvage',
    'Mata Uang',
    'Estimasi',
    'Nilai Salvage',
    'Remark',
    'Posisi Salvage',
  ])
})

/**
 * Grid "Detail History Salvage" ikut digambar DI PANEL, bukan hanya di form Tambah.
 *
 * Di Pega satu section yang sama — `DetailPengajuanSalvage` — disisipkan keduanya, dan ia
 * menjawab pertanyaan yang tidak dapat dijawab daftar mana pun: klaim ini sudah pernah
 * diajukan berapa kali, dan masing-masing berakhir di mana.
 */
it('panel rincian ikut menggambar grid Detail History Salvage', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await pilihDaftar('Checker')
  await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))

  const panel = await screen.findByRole('region', { name: 'Detail Salvage' })
  const riwayat = within(panel).getByRole('region', { name: 'Detail History Salvage' })

  expect(within(riwayat).getByText('Salvage Diterima di Komite')).toBeInTheDocument()
})

/**
 * Bagian yang KOSONG tidak digambar, sehingga kotaknya terlihat seperti aslinya.
 *
 * Inilah keadaan baris Salvage Buyback: pengajuannya tidak punya nilai lelang, tidak punya
 * PIC survey, dan tidak punya rincian barang. Menggambar ketiga bagian itu dalam keadaan
 * kosong akan membuat panel memanjang oleh tiga kotak berisi tanda pisah belaka.
 */
it('bagian lelang, PIC survey, dan barang HILANG ketika tidak ada isinya', async () => {
  stubFetch((url) => {
    if (url === META_PATH) return jsonResponse(200, METADATA)
    if (url === COUNTS_PATH) return jsonResponse(200, COUNTS)

    if (url.startsWith(`${PATH}/pengajuan/`)) {
      return jsonResponse(200, {
        ...DETAIL,
        nilai_penawaran: '',
        nama_pemenang: '',
        tanggal_lelang: '',
        nama_pic_survey: '',
        no_telp_pic_survey: '',
        email_pic_survey: '',
        barang: [],
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

  await pilihDaftar('Checker')
  await userEvent.click(await screen.findByRole('button', { name: 'Detail' }))

  const panel = await screen.findByRole('region', { name: 'Detail Salvage' })

  expect(within(panel).queryByText('Lelang')).not.toBeInTheDocument()
  expect(within(panel).queryByText('PIC Survey')).not.toBeInTheDocument()
  expect(within(panel).queryByText('Detail Pengajuan Salvage')).not.toBeInTheDocument()

  // Kotak utamanya tetap utuh — yang hilang hanya bagian tambahannya.
  expect(within(panel).getByText('Gudang Cakung')).toBeInTheDocument()
})

/**
 * TIDAK ada bilah tab di layar ini, dan tabel ringkas yang menggantikannya.
 *
 * Bilah itu dihapus 2026-10-03. Uji ini menjaga keadaan itu: bilah tab menduakan
 * navigasi yang sudah ada, dan menambahkannya kembali berarti dua tempat yang
 * memutuskan daftar mana yang terbuka — yang satu akan tertinggal.
 */
it('tidak ada bilah tab; perpindahan daftar lewat tabel ringkas', async () => {
  stubDefaultFetch()
  await renderLoaded()

  expect(screen.queryByRole('tablist')).not.toBeInTheDocument()
  expect(screen.queryAllByRole('tab')).toHaveLength(0)

  // Dan tabel ringkas benar-benar berpindah daftar, bukan sekadar menandai barisnya.
  await pilihDaftar('Checker')
  expect(lastListCall()?.url).toContain('daftar=checker')
})

/**
 * Baris yang sedang terbuka DITANDAI di tabel ringkas.
 *
 * Tanpa bilah tab, inilah satu-satunya petunjuk daftar mana yang sedang digambar —
 * dan empat pasang daftar punya kolom yang identik, sehingga grid-nya sendiri tidak
 * memberi tahu apa pun.
 */
it('tabel ringkas menandai daftar yang sedang terbuka', async () => {
  stubDefaultFetch()
  await renderLoaded()

  await pilihDaftar('Checker')

  const ringkas = screen.getByRole('table', {
    name: 'Ringkasan jumlah pengajuan per status salvage',
  })
  expect(within(ringkas).getByRole('button', { name: 'Checker' })).toHaveAttribute(
    'aria-current',
    'true',
  )
  expect(
    within(ringkas).getByRole('button', { name: 'Outstanding' }),
  ).not.toHaveAttribute('aria-current')
})

/** Donat ringkasan menggambar satu irisan beserta angkanya untuk tiap baris pencacah. */
it('donat ringkasan menggambar angka setiap baris pencacah', async () => {
  stubDefaultFetch()
  await renderLoaded()

  const donat = screen.getByRole('img', { name: /Sebaran status salvage/ })

  // Bacaan untuk pembaca layar memuat ketiga baris beserta angkanya — grafik tanpa ini
  // hanya terbaca sebagai gambar tanpa isi.
  expect(donat).toHaveAccessibleName(
    'Sebaran status salvage, total 12: Outstanding 3, Checker 2, Tidak Terjual 7.',
  )

  // Dan angkanya benar-benar tercetak di sebelah irisannya, sebagaimana di layar lama.
  expect(within(donat).getByText('3')).toBeInTheDocument()
  expect(within(donat).getByText('2')).toBeInTheDocument()
  expect(within(donat).getByText('7')).toBeInTheDocument()
})

/**
 * Donat dan tabel dibaca dari SATU daftar yang sama.
 *
 * Di Pega keduanya diisi dua page berbeda dengan label yang tidak sama, sehingga grafik
 * dan tabel yang berdampingan dapat menyebut hal yang berbeda. Uji ini menjaga keadaan
 * yang dipilih di sini: label di legenda selalu label yang sama dengan di tabel.
 */
it('legenda donat memakai label yang sama dengan tabel ringkas', async () => {
  stubDefaultFetch()
  await renderLoaded()

  const ringkas = screen.getByRole('table', {
    name: 'Ringkasan jumlah pengajuan per status salvage',
  })
  const diTabel = [...ringkas.querySelectorAll('tbody th')].map((th) =>
    (th.textContent ?? '').trim(),
  )

  const donat = screen.getByRole('img', { name: /Sebaran status salvage/ })
  const legenda = donat.parentElement?.querySelector('ul')
  const diLegenda = [...(legenda?.querySelectorAll('li') ?? [])].map((li) =>
    (li.textContent ?? '').trim(),
  )

  expect(diLegenda).toEqual(diTabel)
})

/**
 * Baris bernilai NOL tidak punya irisan, dan karena itu tidak punya legenda.
 *
 * Irisan bersapuan nol derajat tidak menggambar apa pun; menyisakan warnanya di legenda
 * berarti menunjuk sesuatu yang tidak ada di grafiknya. Angkanya tetap terbaca di tabel.
 */
it('baris bernilai nol tidak masuk ke donat', async () => {
  stubFetch((url) => {
    if (url === META_PATH) return jsonResponse(200, METADATA)
    if (url === COUNTS_PATH) {
      return jsonResponse(200, {
        baris: [
          { status_salvage: 'Outstanding', jumlah: 4, daftar: 'outstanding' },
          { status_salvage: 'Checker', jumlah: 0, daftar: 'checker' },
        ],
        portal: 'ASM',
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

  const donat = screen.getByRole('img', { name: /Sebaran status salvage/ })
  expect(donat).toHaveAccessibleName('Sebaran status salvage, total 4: Outstanding 4.')

  // Tetapi barisnya TIDAK hilang dari tabel — kekosongan adalah angka, bukan ketiadaan.
  const ringkas = screen.getByRole('table', {
    name: 'Ringkasan jumlah pengajuan per status salvage',
  })
  expect(within(ringkas).getByRole('button', { name: 'Checker' })).toBeInTheDocument()
})

