import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { Counter, ListResponse, MetadataResponse, Tab } from './types'

const PATH = '/api/inbox-manager'
const TAB_PATH = `${PATH}/tab`
const COUNTER_PATH = `${PATH}/ringkasan`
const DECIDE_PATH = `${PATH}/keputusan`

const SAMPLE_PROFILE = {
  identitas: '90000058',
  nama: 'Contoh Penyelia',
  jenis: 'KARYAWAN',
  login: 'penyelia.manager',
  email: 'contoh.penyelia@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Tab Outstanding — dashboard tanpa penyaring periode.
 *
 * Ketiadaan penyaring itu BUKAN kelalaian data uji: ketiga kueri Pega yang memasoknya tidak
 * menyaring tanggal sama sekali. Menambahkannya mengubah arti angkanya dari "yang sedang
 * berjalan" menjadi "yang terjadi pada suatu periode".
 */
const TAB_OUTSTANDING: Tab = {
  kode: '1',
  nama: 'Outstanding',
  keterangan: 'Klaim yang sedang berjalan dan sudah punya PIC Teknik.',
  jenis: 'dashboard',
  panel: [
    {
      kunci: 'pic',
      judul: '',
      kolom: [
        { kunci: 'pic', judul: 'PIC' },
        { kunci: 'jumlah', judul: 'OS' },
      ],
    },
    {
      kunci: 'grup_bisnis',
      judul: '',
      kolom: [
        { kunci: 'grup_bisnis', judul: 'COB' },
        { kunci: 'jumlah', judul: 'OS' },
      ],
    },
    {
      kunci: 'kategori_os',
      judul: '',
      kolom: [
        { kunci: 'kategori_dol', judul: 'Kategori/DOL' },
        { kunci: 'reinsurer', judul: 'Reinsurer' },
      ],
    },
  ],
  punya_ekspor_detail: true,
  keputusan: {
    dapat_diputuskan: false,
    dapat_massal: false,
    alasan_wajib_saat_menolak: false,
  },
  punya_penyaring_periode: false,
}

const TAB_APPROVAL_MASTER: Tab = {
  kode: '4',
  nama: 'Approval Master',
  keterangan: 'Ringkasan seluruh antrean yang menunggu persetujuan Anda.',
  jenis: 'ringkasan',
  keputusan: { dapat_diputuskan: false, dapat_massal: false, alasan_wajib_saat_menolak: false },
  punya_penyaring_periode: false,
}

/** Antrean yang tabelnya PUNYA kolom alasan. */
const TAB_BENGKEL: Tab = {
  kode: '5',
  nama: 'Master Bengkel',
  keterangan: 'Pengajuan data bengkel yang menunggu persetujuan.',
  jenis: 'antrean',
  induk: '4',
  dalam_bilah_induk: true,
  kolom: [
    { kunci: 'id', judul: 'ID Bengkel' },
    { kunci: 'nama', judul: 'Nama Bengkel' },
  ],
  keputusan: {
    dapat_diputuskan: true,
    dapat_massal: true,
    alasan_wajib_saat_menolak: true,
    label_alasan: 'Alasan Status Bengkel',
  },
  punya_penyaring_periode: false,
}

/**
 * Antrean yang jalur SETUJU-nya ditahan.
 *
 * Alasannya dikirim SERVER, bukan disusun layar. Bila layar menyimpulkannya sendiri dari kode
 * tab, tombolnya dapat tergambar aktif pada antrean yang server justru menolaknya — dan di
 * antrean ini penolakan itu menyangkut uang.
 */
const TAB_PAYMENT: Tab = {
  kode: '12',
  nama: 'Payment Klaim Akseptasi',
  keterangan: 'Nomor akseptasi yang menunggu persetujuan atasan.',
  jenis: 'antrean',
  induk: '4',
  dalam_bilah_induk: true,
  kolom: [
    { kunci: 'no_klaim', judul: 'No Klaim' },
    { kunci: 'no_akseptasi', judul: 'No Akseptasi' },
  ],
  keputusan: {
    dapat_diputuskan: true,
    dapat_massal: true,
    alasan_setuju_ditahan:
      'Menyetujui pembayaran di Pega ikut menjalankan transfer ke kasir, dan rantai itu ' +
      'belum lengkap.',
    alasan_wajib_saat_menolak: true,
    label_alasan: 'Catatan Atasan',
  },
  punya_penyaring_periode: false,
}

const SELISIH = [
  'Dashboard Outstanding kini dihitung dari satu tabel datar, bukan dari gabungan dua tabel ' +
    'Pega.',
]

const METADATA: MetadataResponse = {
  tab: [TAB_OUTSTANDING, TAB_APPROVAL_MASTER, TAB_BENGKEL, TAB_PAYMENT],
  tab_bawaan: TAB_OUTSTANDING.kode,
  lini_bisnis_anda: 'NONMBU',
  selisih_terencana: SELISIH,
}

/**
 * Pencacah, termasuk SATU yang sumbernya tidak dapat dibaca.
 *
 * Keadaan itu nyata sejak 2026-09-28: `POOLDATA.SPAREPART_HE` adalah view berstatus INVALID.
 * Mencontohkannya di uji memastikan jalur yang menanganinya benar-benar tergambar.
 */
const PENCACAH: Counter[] = [
  { tab: '1', label: 'Outstanding', jumlah: 354 },
  { tab: '4', label: 'Approval Master', jumlah: 13 },
  { tab: '5', label: 'Master Bengkel', jumlah: 2, induk: '4' },
  {
    tab: '8',
    label: 'Master Sparepart',
    jumlah: 0,
    induk: '4',
    tidak_tersedia: 'Sumbernya, view POOLDATA.SPAREPART_HE, sedang tidak dapat dibaca.',
  },
  { tab: '12', label: 'Payment Klaim Akseptasi', jumlah: 8, induk: '4' },

  // Penolakan Klaim BERINDUK Approval Master pada pohon pencacah, meski tidak ada di bilah
  // tabnya — dua fakta Pega yang berbeda.
  { tab: '13', label: 'Penolakan Klaim', jumlah: 12, induk: '4' },
]

/** Isi antrean Master Bengkel. Seluruhnya KARANGAN (`D-69`). */
const BARIS_BENGKEL = [
  { kunci: 'BGK-001', sel: { id: 'BGK-001', nama: 'Bengkel Contoh Satu' } },
  { kunci: 'BGK-002', sel: { id: 'BGK-002', nama: 'Bengkel Contoh Dua' } },
]

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

/** listCalls menyebut parameter setiap permintaan isi tab, berurutan. */
function listCalls(): URLSearchParams[] {
  return calls
    .filter((call) => call.url.startsWith(`${PATH}?`) || call.url === PATH)
    .map((call) => new URL(call.url, 'https://uji.invalid').searchParams)
}

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

/** Peladen tiruan yang menjawab bentuk layar, pencacah, dan isi tab. */
/**
 * Jawaban tab Outstanding — tiga grid beserta kedua penyaringnya.
 *
 * Grid ketiga ikut MENYARING menurut Reinsurer yang diminta, supaya uji tombol "Filter"
 * membuktikan permintaannya benar-benar berubah — bukan sekadar tombolnya dapat ditekan.
 */
function outstandingBody(reinsurer: string, kosong = false): ListResponse {
  const semua = [
    {
      kategori_dol: { teks: 'ACCEPTATION' },
      reinsurer: { teks: 'LEADER' },
      '2026': { jumlah: 11 },
    },
    {
      kategori_dol: { teks: 'CLAIM COMMITTEE' },
      reinsurer: { teks: 'MEMBER' },
      '2026': { jumlah: 4 },
    },
  ]

  return {
    tab: TAB_OUTSTANDING,
    panel: [
      {
        kunci: 'pic',
        judul: '',
        kolom: TAB_OUTSTANDING.panel![0]!.kolom,
        baris: [
          { pic: { teks: 'ALL' }, jumlah: { jumlah: 354 } },
          { pic: { teks: 'ANDIKA' }, jumlah: { jumlah: 18 } },
        ],
      },
      {
        kunci: 'grup_bisnis',
        judul: '',
        kolom: TAB_OUTSTANDING.panel![1]!.kolom,
        baris: [{ grup_bisnis: { teks: 'FIRE' }, jumlah: { jumlah: 112 } }],
      },
      {
        kunci: 'kategori_os',
        judul: '',
        kolom: [...TAB_OUTSTANDING.panel![2]!.kolom, { kunci: '2026', judul: '2026' }],
        baris: kosong
          ? []
          : reinsurer
            ? semua.filter((row) => row.reinsurer.teks === reinsurer)
            : semua,
      },
    ],
    penyaring: [
      {
        kunci: 'reinsurer',
        label: 'Reinsurer',
        pilihan: [
          { nilai: '', label: 'All' },
          { nilai: 'LEADER', label: 'Leader' },
          { nilai: 'MEMBER', label: 'Member' },
        ],
      },
      {
        kunci: 'kategori_os',
        label: 'Kategori OS',
        pilihan: [
          { nilai: '', label: 'All' },
          { nilai: 'SURVEY', label: 'SURVEY' },
        ],
      },
    ],
  }
}

function stubDefaultFetch(options?: {
  meta?: MetadataResponse
  decide?: () => Response

  /** Mengosongkan baris grid ketiga, tanpa mengosongkan kolomnya. */
  kosongkanGridKetiga?: boolean
}) {
  const meta = options?.meta ?? METADATA

  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, meta)
    if (url === COUNTER_PATH) return jsonResponse(200, { pencacah: PENCACAH })
    if (url === DECIDE_PATH) {
      return (
        options?.decide?.() ??
        jsonResponse(200, {
          diminta: 1,
          berubah: 1,
          tidak_berubah: 0,
          pesan: '1 baris diputuskan.',
        })
      )
    }

    // Tab yang diminta menentukan bentuk jawabannya. Menjawab tab yang sama untuk setiap
    // permintaan akan membuat uji perpindahan tab lulus tanpa membuktikan apa pun.
    const wanted = new URL(url, 'https://uji.invalid').searchParams.get('tab') ?? '1'

    const body: ListResponse =
      wanted === TAB_BENGKEL.kode
        ? {
            tab: TAB_BENGKEL,
            baris: BARIS_BENGKEL,
            paginasi: { halaman: 1, ukuran: 25, total: 2, total_halaman: 1 },
          }
        : wanted === TAB_PAYMENT.kode
          ? {
              tab: TAB_PAYMENT,
              baris: [
                {
                  kunci: 'AKS-900001',
                  sel: { no_klaim: 'PNC-900001', no_akseptasi: 'AKS-900001' },
                },
              ],
              paginasi: { halaman: 1, ukuran: 25, total: 1, total_halaman: 1 },
            }
          : wanted === TAB_APPROVAL_MASTER.kode
            ? { tab: TAB_APPROVAL_MASTER }
            : outstandingBody(
                new URL(url, 'https://uji.invalid').searchParams.get('reinsurer') ?? '',
                options?.kosongkanGridKetiga === true,
              )

    return jsonResponse(200, body)
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/inbox-manager']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/** renderLoaded menggambar layar lalu MENUNGGU bilah tabnya tiba. */
async function renderLoaded() {
  renderPage()
  await screen.findByRole('tab', { name: /Outstanding/ })
}

/**
 * Membuka satu antrean persetujuan.
 *
 * Dua langkah, dan itu memang bentuknya: kesembilan antrean adalah ANAK "Approval Master",
 * bukan tab sejajar dengannya. Uji yang mengekliknya langsung dari bilah atas akan lolos pada
 * layar yang justru salah jenjang.
 */
async function bukaAntrean(nama: RegExp) {
  await userEvent.click(screen.getByRole('tab', { name: /Approval Master/ }))
  await userEvent.click(await screen.findByRole('tab', { name: nama }))
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

describe('bentuk layar', () => {
  it('menggambar kontainer Pega sebagai tab yang dapat dipilih', async () => {
    // Di Pega ketiga belasnya BUKAN tab melainkan kontainer bersyarat
    // `FlagManager.AlasanKlaim==n`. Pemisahannya menjadi tab adalah selisih terencana.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('tab', { name: /Outstanding/ })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: /Approval Master/ })).toBeInTheDocument()
  })

  /*
    Uji jenjang — dan ia menahan kekeliruan yang BENAR-BENAR pernah tergambar di layar ini:
    ketiga belas tab digambar berjajar, sehingga sembilan antrean persetujuan tampak setara
    dengan induknya sendiri.

    Dasarnya bukan selera: `Activity/CountDashbroardManager` menulis empat pencacah pertama ke
    `TempCountDashboard.pxResults(<APPEND>)` dan sembilan sisanya ke
    `.pxResults(4).pxResults(<APPEND>)`.
  */
  it('tidak menggambar antrean persetujuan di bilah tingkat atas', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      within(screen.getByRole('tablist', { name: 'Bagian Inbox Manager' })).queryByRole('tab', {
        name: /Master Bengkel/,
      }),
    ).not.toBeInTheDocument()
  })

  /*
    Bilah di dalam Approval Master memuat TEPAT tab yang ada di bilah Pega. Penolakan Klaim
    berinduk Approval Master pada pohon pencacah tetapi `InboxManager_Section2` tidak
    menyertakannya di bilah tabnya.

    Dan tidak ada pil "Ringkasan": Pega tidak punya.
  */
  it('TIDAK menaruh Ringkasan maupun Penolakan Klaim di bilah Approval Master', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await userEvent.click(screen.getByRole('tab', { name: /Approval Master/ }))

    const inside = within(
      await screen.findByRole('tablist', { name: 'Antrean Approval Master' }),
    )
    expect(inside.queryByRole('tab', { name: 'Ringkasan' })).not.toBeInTheDocument()
    expect(inside.queryByRole('tab', { name: /Penolakan Klaim/ })).not.toBeInTheDocument()
  })

  /*
    Memilih Approval Master LANGSUNG membuka antrean pertamanya.

    Di Pega tidak ada halaman antara: bilah sub-tabnya muncul dengan sub-tab pertama sudah
    terbuka. Kumpulan kartu berangka yang sempat digambar di sini dicabut atas permintaan
    Work Owner (2026-10-08).
  */
  it('langsung membuka Master Bengkel saat Approval Master dipilih', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await userEvent.click(screen.getByRole('tab', { name: /Approval Master/ }))

    const inside = within(
      await screen.findByRole('tablist', { name: 'Antrean Approval Master' }),
    )
    await waitFor(() =>
      expect(inside.getByRole('tab', { name: /Master Bengkel/ })).toHaveAttribute(
        'aria-selected',
        'true',
      ),
    )

    // Dan tidak ada satu pun kartu ringkasan yang tersisa.
    expect(screen.queryByText('menunggu')).not.toBeInTheDocument()
  })

  it('menggambar antrean persetujuan DI DALAM Approval Master', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await userEvent.click(screen.getByRole('tab', { name: /Approval Master/ }))

    const inside = within(
      await screen.findByRole('tablist', { name: 'Antrean Approval Master' }),
    )
    expect(inside.getByRole('tab', { name: /Master Bengkel/ })).toBeInTheDocument()
    expect(inside.getByRole('tab', { name: /Payment Klaim Akseptasi/ })).toBeInTheDocument()
  })

  /*
    Setelah sebuah antrean terbuka, yang tersorot di bilah atas tetap induknya. Tanpa itu,
    tidak ada satu pun tab tingkat atas yang tersorot — dan pengguna kehilangan letaknya di
    dalam jenjang tepat saat ia masuk ke dalamnya.
  */
  it('tetap menyorot Approval Master saat salah satu antreannya terbuka', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await bukaAntrean(/Master Bengkel/)

    const top = within(screen.getByRole('tablist', { name: 'Bagian Inbox Manager' }))
    expect(top.getByRole('tab', { name: /Approval Master/ })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  /*
    Uji bagian-bagian tab Outstanding, dan ia menahan kekeliruan yang BENAR-BENAR pernah ada:
    grid ketiga tidak digambar sama sekali karena catatan modul menyatakan kueri pemasoknya
    "bukan grid". Work Owner menunjukkannya dari layar Pega yang berjalan.
  */
  it('menggambar KETIGA grid tab Outstanding', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Kepala kolomnya disalin apa adanya dari section — PIC, OS, COB, OS.
    expect(await screen.findByRole('columnheader', { name: 'PIC' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'COB' })).toBeInTheDocument()
    expect(screen.getAllByRole('columnheader', { name: 'OS' })).toHaveLength(2)

    // Grid ketiga: kolom tetap, lalu kolom tahun yang datang dari data.
    expect(screen.getByRole('columnheader', { name: 'Kategori/DOL' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Reinsurer' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: '2026' })).toBeInTheDocument()
    expect(screen.getByText('ACCEPTATION')).toBeInTheDocument()
  })

  /*
    Grid digambar ATAS-BAWAH, bukan berdampingan — ketetapan Work Owner 2026-10-07.

    Berdampingan sempat ditiru dari Pega dan DICABUT: grid di layar ini berkolom banyak
    (sembilan dan sepuluh), sehingga separuh lebar layar memaksa kolomnya berdempetan sampai
    judulnya terpotong.
  */
  it('menggambar grid ATAS-BAWAH, bukan berdampingan', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const pic = await screen.findByRole('columnheader', { name: 'PIC' })
    const pembungkus = pic.closest('table')?.parentElement?.parentElement?.parentElement

    expect(pembungkus?.className).toContain('space-y-6')
    expect(pembungkus?.className).not.toContain('grid-cols-2')
  })

  /*
    Kepala kolom tetap digambar meski tidak ada satu baris pun.

    Sebelumnya grid kosong hanya menampilkan "Tidak ada angka untuk penyaring ini." tanpa satu
    pun kepala kolom, sehingga tidak ada petunjuk angka apa yang sebenarnya dihitung di sana.
    Pega menggambarnya: tabel COB pada layar lama tetap menampilkan kesembilan kepala kolomnya
    di atas tulisan "Data Tidak Ada".
  */
  it('tetap menggambar kepala kolom pada grid yang KOSONG', async () => {
    stubDefaultFetch({ kosongkanGridKetiga: true })
    await renderLoaded()

    expect(await screen.findByRole('columnheader', { name: 'Kategori/DOL' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Reinsurer' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: '2026' })).toBeInTheDocument()
    expect(screen.getByText('Tidak ada angka untuk penyaring ini.')).toBeInTheDocument()
  })

  /*
    Antrean pun tidak kehilangan kolomnya saat permintaannya gagal — prinsip yang sama dengan
    dashboard, dan dengan Pega yang menampilkan alert tanpa membuang tabelnya.
  */
  it('tetap menggambar kolom antrean saat pemuatannya GAGAL', async () => {
    stubFetch((url) => {
      if (url === TAB_PATH) return jsonResponse(200, METADATA)
      if (url === COUNTER_PATH) return jsonResponse(200, { pencacah: PENCACAH })
      return jsonResponse(503, {
        kode: 'sumber_tidak_tersedia',
        pesan: 'Sumber antrean ini sedang tidak dapat dibaca.',
      })
    })
    await renderLoaded()
    await bukaAntrean(/Master Bengkel/)

    expect(await screen.findByText(/Sumber antrean ini sedang tidak dapat dibaca/)).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'ID Bengkel' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Nama Bengkel' })).toBeInTheDocument()
  })

  it('menggambar kedua penyaring beserta tombolnya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByLabelText('Reinsurer')).toBeInTheDocument()
    expect(screen.getByLabelText('Kategori OS')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Filter' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Clear Filter' })).toBeEnabled()
  })

  /*
    Mengubah dropdown TIDAK memuat ulang; tombol Filter yang memuatnya. Itu perilaku layar
    lama, yang memang punya tombolnya sendiri — dan tanpa uji ini, menyatukan keduanya akan
    lolos tanpa terlihat.
  */
  it('menerapkan penyaring hanya setelah tombol Filter ditekan', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.selectOptions(await screen.findByLabelText('Reinsurer'), 'LEADER')
    expect(screen.getByText('CLAIM COMMITTEE')).toBeInTheDocument()
    expect(listCalls().some((params) => params.get('reinsurer') === 'LEADER')).toBe(false)

    await userEvent.click(screen.getByRole('button', { name: 'Filter' }))

    await waitFor(() =>
      expect(listCalls().some((params) => params.get('reinsurer') === 'LEADER')).toBe(true),
    )
    await waitFor(() => expect(screen.queryByText('CLAIM COMMITTEE')).not.toBeInTheDocument())
  })

  it('mengembalikan penyaring ke All lewat Clear Filter', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const reinsurer = await screen.findByLabelText('Reinsurer')
    await userEvent.selectOptions(reinsurer, 'LEADER')
    await userEvent.click(screen.getByRole('button', { name: 'Filter' }))
    await waitFor(() => expect(screen.queryByText('CLAIM COMMITTEE')).not.toBeInTheDocument())

    await userEvent.click(screen.getByRole('button', { name: 'Clear Filter' }))

    expect(reinsurer).toHaveValue('')
    expect(await screen.findByText('CLAIM COMMITTEE')).toBeInTheDocument()
  })

  /*
    Bloknya digambar karena ia memang ada di layar lama; tombolnya dimatikan karena isinya
    belum dapat dibangun dengan jujur. Yang diuji di sini: alasannya ikut tertulis, bukan
    tombol mati tanpa keterangan.
  */
  it('menggambar blok Export Data Detail Klaim beserta alasan penahanannya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText('Export Data Detail Klaim')).toBeInTheDocument()
    expect(screen.getByLabelText('Dari')).toBeDisabled()
    expect(screen.getByLabelText('Sampai')).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Export to Excel' })).toBeDisabled()
    expect(screen.getByText(/belum dapat dijalankan/)).toBeInTheDocument()
  })

  it('membuka tab bawaan yang ditetapkan server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('tab', { name: /Outstanding/ })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('TIDAK menempelkan angka pencacah pada tabnya', async () => {
    // Pega menampilkan angkanya sebagai tabel `Status`/`Jumlah` tersendiri, bukan sebagai
    // lencana pada tombol tabnya.
    stubDefaultFetch()
    await renderLoaded()

    expect(
      within(screen.getByRole('tab', { name: /Outstanding/ })).queryByText('354'),
    ).not.toBeInTheDocument()
  })

  it('menyatakan lini bisnis yang menyaring angka dashboard', async () => {
    // Tanpa ini, dua penyelia yang membandingkan layar masing-masing akan melihat angka
    // berbeda tanpa satu pun petunjuk kenapa.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByText(/NONMBU/)).toBeInTheDocument()
  })

  it('menjelaskan lini bisnis yang BELUM diisi, bukan mendiamkannya', async () => {
    // Keadaan paling umum di produksi: `LINE_BUSINESS` baru terisi pada sebagian petugas.
    stubDefaultFetch({ meta: { ...METADATA, lini_bisnis_anda: '' } })
    await renderLoaded()

    expect(screen.getByText(/belum diisi/)).toBeInTheDocument()
  })

  it('membungkus isinya dengan kerangka halaman yang SAMA dengan modul lain', async () => {
    // Uji ini lahir dari cacat nyata: layar ini sempat dibangun TANPA pembungkus sama
    // sekali, sehingga isinya menempel ke bilah samping tanpa jarak dan terlihat berbeda
    // dari seluruh layar lain meski isinya benar.
    //
    // Penyebabnya struktural: tidak ada pembungkus bersama di `App.tsx`, sehingga setiap
    // modul membungkus halamannya sendiri — dan yang lupa tidak menghasilkan satu pun galat.
    // Yang diperiksa di sini karena itu kelasnya, bukan tampilannya.
    stubDefaultFetch()
    await renderLoaded()

    const frame = screen.getByRole('heading', { name: 'Inbox Manager' }).closest('div.mx-auto')
    expect(frame).not.toBeNull()
    expect(frame).toHaveClass('max-w-[96rem]', 'px-4', 'py-8')
  })

  it('menaruh tombol Export di kepala layar dan mematikannya pada tab dashboard', async () => {
    // Tempat yang TETAP: tombolnya tidak muncul dan hilang saat tab berpindah, sehingga
    // pengguna tidak perlu mencarinya ulang. Yang berubah hanya aktif atau tidaknya.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('button', { name: 'Export' })).toBeDisabled()
    expect(screen.getByText(/Hanya antrean persetujuan yang dapat diekspor/)).toBeInTheDocument()

    await bukaAntrean(/Master Bengkel/)

    expect(await screen.findByRole('button', { name: 'Export' })).toBeEnabled()
  })

  /*
    Kebalikan dari uji sebelumnya, dan itu disengaja: daftar selisih terencana TIDAK digambar
    di layar (ketetapan Work Owner 2026-10-07 — tidak ada tulisan yang Pega tidak punya).

    Ia tetap dikirim server dan tetap dipakai uji kesetaraan gerbang 1 (`D-54`); yang dicabut
    hanya penggambarannya.
  */
  it('TIDAK menggambar daftar selisih terencana di layar', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.queryByText(/Yang berbeda dari layar lama/)).not.toBeInTheDocument()
    SELISIH.forEach((item) => {
      expect(screen.queryByText(item)).not.toBeInTheDocument()
    })
  })
})

describe('dashboard', () => {
  it('menggambar panel beserta barisnya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Kedua grid pertama TIDAK berjudul di Pega — yang membedakannya kepala kolomnya.
    expect(await screen.findByRole('columnheader', { name: 'PIC' })).toBeInTheDocument()
    expect(screen.queryByText('Outstanding per PIC')).not.toBeInTheDocument()
    expect(await screen.findByText('ANDIKA')).toBeInTheDocument()
  })

  it('TIDAK menggambar penyaring periode pada tab Outstanding', async () => {
    // Ketiga kueri Pega yang memasoknya tidak menyaring tanggal sama sekali. Menggambar
    // penyaring di sana akan menyarankan kemampuan yang tidak ada, dan mengubah arti angkanya
    // bila kemudian benar-benar dipasang.
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByRole('columnheader', { name: 'PIC' })).toBeInTheDocument()
    expect(screen.queryByLabelText('Periode')).not.toBeInTheDocument()
  })
})

describe('antrean persetujuan', () => {
  it('menggambar tombol keputusan hanya pada tab antrean', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Tab dashboard: tidak ada tombol keputusan.
    expect(screen.queryByRole('button', { name: 'Setujui' })).not.toBeInTheDocument()

    await bukaAntrean(/Master Bengkel/)

    expect(await screen.findByRole('button', { name: 'Setujui' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tolak' })).toBeInTheDocument()
  })

  it('menonaktifkan tombol keputusan selama belum ada baris yang dipilih', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await bukaAntrean(/Master Bengkel/)

    expect(await screen.findByRole('button', { name: 'Setujui' })).toBeDisabled()
  })

  it('mengirim kunci baris APA ADANYA, bukan menyusunnya ulang', async () => {
    // Bentuk kunci berbeda tiap antrean — satu di antaranya gabungan empat kolom. Menyusunnya
    // di layar berarti aturan kuncinya hidup di dua tempat.
    stubDefaultFetch()
    await renderLoaded()
    await bukaAntrean(/Master Bengkel/)

    await userEvent.click(await screen.findByLabelText('Pilih baris BGK-001'))
    await userEvent.click(screen.getByRole('button', { name: 'Setujui' }))

    const sent = [...calls].reverse().find((call) => call.url === DECIDE_PATH)
    expect(sent).toBeDefined()

    const body = JSON.parse(String(sent!.init?.body)) as { kunci: string[]; tab: string }
    expect(body.kunci).toEqual(['BGK-001'])
    expect(body.tab).toBe('5')
  })

  it('menggambar isian alasan hanya bila tabelnya punya kolomnya', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await bukaAntrean(/Master Bengkel/)

    expect(await screen.findByText(/Alasan Status Bengkel/)).toBeInTheDocument()
  })

  it('menahan tombol Setujui beserta ALASANNYA pada antrean pembayaran', async () => {
    // Uji terpenting di berkas ini: menyetujui pembayaran tanpa transfer kasir menandai
    // pembayaran sudah disetujui padahal tidak pernah sampai ke kasir.
    stubDefaultFetch()
    await renderLoaded()
    await bukaAntrean(/Payment Klaim Akseptasi/)

    await userEvent.click(await screen.findByLabelText('Pilih baris AKS-900001'))

    expect(screen.getByRole('button', { name: 'Setujui' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Tolak' })).toBeEnabled()
    expect(screen.getByText(/Persetujuan sedang ditahan/)).toBeInTheDocument()
  })

  it('menyatakan baris yang sudah diputuskan orang lain', async () => {
    // Di Pega keadaan ini tidak dapat diketahui sama sekali: keputusan terakhir menimpa yang
    // sebelumnya tanpa jejak.
    stubDefaultFetch({
      decide: () =>
        jsonResponse(200, {
          diminta: 2,
          berubah: 1,
          tidak_berubah: 1,
          pesan:
            '1 dari 2 baris berubah. Sisanya sudah diputuskan orang lain sementara daftar ' +
            'ini masih yang lama — muat ulang untuk melihat keadaan terkini.',
        }),
    })
    await renderLoaded()
    await bukaAntrean(/Master Bengkel/)

    await userEvent.click(await screen.findByLabelText('Pilih baris BGK-001'))
    await userEvent.click(screen.getByLabelText('Pilih baris BGK-002'))
    await userEvent.click(screen.getByRole('button', { name: 'Setujui' }))

    expect(await screen.findByText(/1 dari 2 baris berubah/)).toBeInTheDocument()
  })
})

describe('Approval Master tanpa kartu ringkasan', () => {
  /*
    Antrean yang sumbernya belum ada TIDAK lagi menjelaskan dirinya di layar.

    Keterangan itu — nama view yang tidak sah di basis data — ditujukan ke tim teknis, dan
    sebabnya keadaan pengembangan: tabelnya memang belum dibuat. Ia dicabut dari badan tab
    maupun dari tooltip pilnya (Work Owner, 2026-10-08). Server tetap menjawab 503 dan tetap
    mencatatnya di log.
  */
  it('tidak menempelkan keterangan sumber pada pil antreannya', async () => {
    const sparepart: Tab = {
      ...TAB_BENGKEL,
      kode: '8',
      nama: 'Master Sparepart',
      keterangan: 'Pengajuan data sparepart yang menunggu persetujuan.',
    }
    stubDefaultFetch({ meta: { ...METADATA, tab: [...METADATA.tab, sparepart] } })
    await renderLoaded()
    await userEvent.click(screen.getByRole('tab', { name: /Approval Master/ }))

    const inside = within(
      await screen.findByRole('tablist', { name: 'Antrean Approval Master' }),
    )
    expect(inside.getByRole('tab', { name: /Master Sparepart/ })).toHaveAttribute(
      'title',
      sparepart.keterangan,
    )
    expect(screen.queryByText(/SPAREPART_HE/)).not.toBeInTheDocument()
  })
})
