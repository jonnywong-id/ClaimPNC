import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { MetadataResponse, Tab, WorkItem } from './types'

const PATH = '/api/inbox-manager-admin'
const TAB_PATH = `${PATH}/tab`
const EXPORT_PATH = `${PATH}/ekspor`

const SAMPLE_PROFILE = {
  identitas: '90000057',
  nama: 'Contoh Penyelia Admin',
  jenis: 'KARYAWAN',
  login: 'penyelia.admin',
  email: 'contoh.penyelia@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Kedelapan kolom, dan ketiga tab MEMAKAI YANG SAMA.
 *
 * Kesamaan itu bukan kebetulan: ketiganya digambar Report Definition yang sama
 * (`ManagementAdminView`), dijalankan dengan parameter `OrgUnit` yang berbeda.
 *
 * Judulnya disalin dari `<pyCaption …>` pada section. Yang paling mudah "dirapikan" tanpa
 * sengaja adalah yang pertama — "ID", yang di layar inbox lain berbunyi "Case ID".
 */
const KOLOM = [
  { kunci: 'id', judul: 'ID' },
  { kunci: 'no_polis', judul: 'No Polis' },
  { kunci: 'nama_tertanggung', judul: 'Nama Tertanggung' },
  { kunci: 'nama_bisnis', judul: 'Nama Bisnis' },
  { kunci: 'nama_sumber_bisnis', judul: 'Nama Sumber Bisnis' },
  { kunci: 'tanggal_pendaftaran', judul: 'Tanggal Pendaftaran' },
  { kunci: 'lama_waktu_klaim', judul: 'Lama Waktu Klaim' },
  { kunci: 'admin_pnc', judul: 'Admin PNC' },
] as Tab['kolom']

const TAB_NONMBU: Tab = {
  kode: '1',
  nama: 'Manajemen Admin - Non MBU',
  keterangan: 'Antrean registrasi klaim di luar Personal Accident dan Travel.',
  kolom: KOLOM,
  unit_organisasi: 'AdminPNC',
  lini_bisnis: 'NONMBU',
}

const TAB_PA: Tab = {
  kode: '2',
  nama: 'Manajemen Admin - PA',
  keterangan: 'Antrean registrasi klaim Personal Accident.',
  kolom: KOLOM,
  unit_organisasi: 'AdminPA',
  lini_bisnis: 'PA',
}

const TAB_TRAVEL: Tab = {
  kode: '3',
  nama: 'Manajemen Admin - Travel',
  keterangan: 'Antrean registrasi klaim Travel.',
  kolom: KOLOM,
  unit_organisasi: 'AdminTRAVEL',
  lini_bisnis: 'TRAVEL',
}

const SELISIH = [
  'Daftar tidak lagi terpotong di 500 baris.',
  'Ketiga kontainer grid menjadi TAB yang dapat dipilih.',
]

/** Jawaban bagi pengguna yang berhak atas ketiga tab — unit organisasi Development di Pega. */
const METADATA: MetadataResponse = {
  tab: [TAB_NONMBU, TAB_PA, TAB_TRAVEL],
  tab_bawaan: '1',
  semua_tab: [TAB_NONMBU, TAB_PA, TAB_TRAVEL],
  lini_bisnis_yang_diharapkan: ['NONMBU', 'PA', 'TRAVEL'],
  lini_bisnis_anda: '',
  selisih_terencana: SELISIH,
  portal: 'ASM',
}

/**
 * Jawaban bagi pengguna yang lini bisnisnya TERISI tetapi tidak membuka satu tab pun.
 *
 * Nilainya sengaja jabatan kepegawaian HCQ. Sampai 2026-09-27 nilai SEPERTI INILAH yang
 * dibandingkan dengan ketiga tab — dan karena itu tidak seorang pun melihat satu tab pun.
 * Sejak koreksi, yang dibandingkan `M_LOGIN_PNC.LINE_BUSINESS`; bila kolom itu kebetulan
 * diisi jabatan, hasilnya tetap nol tab dan layar mengatakannya.
 */
const METADATA_TANPA_TAB: MetadataResponse = {
  ...METADATA,
  tab: [],
  tab_bawaan: '',
  lini_bisnis_anda: 'IT SPECIALIST',
}

/**
 * Jawaban bagi pengguna yang lini bisnisnya BELUM DIISI sama sekali.
 *
 * Ini keadaan yang PALING UMUM di produksi hari ini: `LINE_BUSINESS` baru terisi pada
 * sebagian petugas (`migrations/0004_DICABUT.md`). Ia dibedakan dari yang di atas karena
 * tindakan yang dituntutnya berbeda — melengkapi, bukan membetulkan.
 */
const METADATA_LINI_KOSONG: MetadataResponse = {
  ...METADATA,
  tab: [],
  tab_bawaan: '',
  lini_bisnis_anda: '',
}

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
const BARIS: WorkItem = {
  referensi: 'ASM-FW-GCNMFW-WORK PNC-900001',
  id: 'PNC-900001',
  no_polis: 'CONTOH-NONMBU-0001',
  nama_tertanggung: 'PT Contoh Sejahtera',
  nama_bisnis: 'Property All Risk',
  nama_sumber_bisnis: 'Broker Contoh',
  tanggal_pendaftaran: '2026-09-23',
  lama_waktu_klaim: '3 days ago',
  admin_pnc: 'Admin Contoh Satu',
  status_klaim: 'Register',
}

/**
 * Baris yang tanggal pendaftarannya KOSONG.
 *
 * Ia membuktikan dua hal sekaligus: sel tanggalnya digambar sebagai tanda pisah, dan kolom
 * Lama Waktu Klaim ikut kosong alih-alih berbunyi "0 minutes ago" (`P-5` butir 13).
 */
const BARIS_TANPA_TANGGAL: WorkItem = {
  ...BARIS,
  referensi: 'ASM-FW-GCNMFW-WORK PNC-900002',
  id: 'PNC-900002',
  nama_sumber_bisnis: '',
  tanggal_pendaftaran: null,
  lama_waktu_klaim: '',
}

type Call = { url: string; init: RequestInit | undefined }

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

/** Peladen tiruan yang menjawab bentuk layar dan barisnya. */
function stubDefaultFetch(rows: WorkItem[] = [BARIS], meta: MetadataResponse = METADATA) {
  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, meta)
    if (url.startsWith(EXPORT_PATH)) {
      return new Response('ID\nPNC-900001\n', {
        status: 200,
        headers: {
          'Content-Type': 'text/csv',
          'Content-Disposition': 'attachment; filename="Data AdminPNC.csv"',
        },
      })
    }

    // Tab yang diminta menentukan bentuk jawabannya. Menjawab tab yang sama untuk setiap
    // permintaan akan membuat uji perpindahan tab lulus tanpa membuktikan apa pun.
    const wanted = new URL(url, 'http://uji.invalid').searchParams.get('tab')
    const answered =
      wanted === TAB_TRAVEL.kode
        ? TAB_TRAVEL
        : wanted === TAB_PA.kode
          ? TAB_PA
          : TAB_NONMBU

    return jsonResponse(200, {
      tab: answered,
      baris: rows,
      paginasi: { halaman: 1, ukuran: 50, total: rows.length, total_halaman: 1 },
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
      <MemoryRouter initialEntries={['/inbox-manager-admin']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/**
 * renderLoaded menggambar layar lalu MENUNGGU bentuknya tiba.
 *
 * Penantiannya pada salah satu tab, bukan pada judul layar: judulnya sudah ada sejak
 * penggambaran pertama, sementara bilah tab baru tiba bersama jawaban `/tab`.
 */
async function renderLoaded() {
  renderPage()
  await screen.findByRole('tab', { name: /Non MBU/ })
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

describe('bentuk layar', () => {
  it('menggambar ketiga kontainer Pega sebagai tab yang dapat dipilih', async () => {
    // Di Pega ketiganya BUKAN tab melainkan kontainer terpisah yang tampil menurut lini bisnis
    // pengguna. Pemisahannya menjadi tab adalah selisih terencana, dan uji ini yang
    // menjaganya tidak diam-diam dikembalikan menjadi tiga tabel bertumpuk.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('tab', { name: 'Manajemen Admin - Non MBU' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Manajemen Admin - PA' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Manajemen Admin - Travel' })).toBeInTheDocument()
  })

  it('membuka tab bawaan yang ditetapkan server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('tab', { name: 'Manajemen Admin - Non MBU' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('menggambar kolom yang ditetapkan server, bukan daftar tetap di layar', async () => {
    stubDefaultFetch()
    await renderLoaded()

    for (const column of KOLOM) {
      expect(
        await screen.findByRole('columnheader', { name: column.judul }),
      ).toBeInTheDocument()
    }
  })

  it('TIDAK menggambar Status Klaim sebagai kolom, karena grid Pega pun tidak', async () => {
    // Isian itu dikirim server — berkas ekspor memuatnya — tetapi tidak digambar. Menariknya
    // ke grid akan membuat layar baru punya kolom yang tidak ada di layar lama.
    stubDefaultFetch()
    await renderLoaded()

    expect(
      screen.queryByRole('columnheader', { name: 'Status Klaim' }),
    ).not.toBeInTheDocument()
  })

  it('menyatakan bahwa isinya pekerjaan seluruh petugas, bukan milik pemanggil', async () => {
    // Tidak satu pun tab di layar ini menyaring menurut pengguna yang login. Tanpa
    // keterangan itu, petugas yang terbiasa dengan inbox lain akan mengira daftarnya
    // keliru karena memuat pekerjaan orang lain.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByText(/seluruh petugas/)).toBeInTheDocument()
  })

  it('menampilkan selisih terencana apa adanya dari server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    for (const line of SELISIH) {
      expect(await screen.findByText(line)).toBeInTheDocument()
    }
  })
})

describe('isi antrean', () => {
  it('menggambar baris yang dikirim server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText('PNC-900001')).toBeInTheDocument()
    expect(screen.getByText('PT Contoh Sejahtera')).toBeInTheDocument()
    expect(screen.getByText('Broker Contoh')).toBeInTheDocument()
  })

  it('menggambar Lama Waktu Klaim sebagai teks apa adanya, bukan diformat ulang', async () => {
    // Bentuknya disusun SERVER, meniru format bawaan Pega `DateTime-Frame`. Layar yang
    // menyusunnya sendiri berarti aturan bentuknya hidup di dua tempat.
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText('3 days ago')).toBeInTheDocument()
  })

  it('menggambar sel kosong sebagai tanda pisah, bukan sel yang benar-benar kosong', async () => {
    // Sel kosong tidak dapat dibedakan dari kolom yang gagal dimuat. Di layar ini itu
    // penting khusus: dua kolom memang dapat kosong pada data yang sah.
    stubDefaultFetch([BARIS_TANPA_TANGGAL])
    await renderLoaded()

    await screen.findByText('PNC-900002')
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(2)
  })

  it('meminta tab yang baru dipilih dan kembali ke halaman pertama', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: 'Manajemen Admin - Travel' }))

    const call = lastListCall()
    expect(call?.url).toContain('tab=3')
    // Halaman tidak ikut dikirim berarti halaman pertama — server yang menetapkannya.
    expect(call?.url).not.toContain('halaman=')
  })

  it('menjelaskan antrean kosong dengan menyebut unit organisasinya', async () => {
    // "Tidak ada data" tidak cukup: antrean kosong dapat berarti unitnya memang sepi, atau
    // kolom PXASSIGNEDORGUNIT tidak pernah terisi di produksi — dan yang kedua adalah
    // kekeliruan konfigurasi yang tidak menghasilkan satu pun galat.
    stubDefaultFetch([])
    await renderLoaded()

    expect(await screen.findByText(/AdminPNC/)).toBeInTheDocument()
  })
})

describe('kewenangan tab', () => {
  it('menjelaskan keadaannya saat lini bisnis pengguna tidak membuka satu tab pun', async () => {
    stubDefaultFetch([BARIS], METADATA_TANPA_TAB)
    renderPage()

    expect(
      await screen.findByText(/Tidak ada antrean yang menjadi hak lini bisnis Anda/),
    ).toBeInTheDocument()

    // Nilai yang terbaca sistem disebut apa adanya: sering berbeda dari yang dikira
    // pengguna, dan itulah petunjuk yang ia sampaikan saat melapor.
    expect(screen.getByText('IT SPECIALIST')).toBeInTheDocument()

    // Ketiga lini bisnis yang membuka tab ikut disebut.
    expect(screen.getByText(/NONMBU, PA, TRAVEL/)).toBeInTheDocument()
  })

  it('membedakan lini bisnis yang BELUM DIISI dari yang terisi nilai lain', async () => {
    // Keduanya menuntut tindakan yang berbeda — melengkapi, bukan membetulkan — dan hanya
    // pengguna yang dapat menyampaikannya ke orang yang tepat. Pesan yang sama untuk
    // keduanya membuat laporannya tidak dapat ditindaklanjuti.
    stubDefaultFetch([BARIS], METADATA_LINI_KOSONG)
    renderPage()

    expect(await screen.findByText(/\(belum diisi\)/)).toBeInTheDocument()
    expect(screen.getByText(/belum diisi pada master pengguna/)).toBeInTheDocument()
  })

  it('TIDAK meminta isi antrean saat tidak ada tab yang boleh dilihat', async () => {
    // Permintaannya pasti ditolak server dengan 403. Galat yang sudah diketahui pasti
    // terjadi bukan galat yang layak ditampilkan.
    stubDefaultFetch([BARIS], METADATA_TANPA_TAB)
    renderPage()

    await screen.findByText(/Tidak ada antrean yang menjadi hak lini bisnis Anda/)
    expect(lastListCall()).toBeUndefined()
  })

  it('menggambar hanya tab yang dikirim server, bukan ketiganya', async () => {
    // Penyaringannya terjadi di server. Layar hanya mengikuti hasilnya — menyusun daftar
    // sendiri berarti penyembunyian di layar menjadi satu-satunya kendali.
    stubDefaultFetch([BARIS], {
      ...METADATA,
      tab: [TAB_PA],
      tab_bawaan: TAB_PA.kode,
      lini_bisnis_anda: 'PA',
    })
    renderPage()

    await screen.findByRole('tab', { name: 'Manajemen Admin - PA' })
    expect(screen.queryByRole('tab', { name: /Non MBU/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('tab', { name: /Travel/ })).not.toBeInTheDocument()
  })
})

describe('ekspor', () => {
  it('mengirim tab yang sedang terbuka ke endpoint ekspor', async () => {
    // Berkas yang isinya tidak dapat dicocokkan dengan layar adalah berkas yang
    // menyesatkan.
    stubDefaultFetch()
    await renderLoaded()
    await screen.findByText('PNC-900001')

    await userEvent.click(screen.getByRole('tab', { name: 'Manajemen Admin - PA' }))
    await userEvent.click(screen.getByRole('button', { name: /Export To Excel/ }))

    const ekspor = [...calls].reverse().find((c) => c.url.startsWith(EXPORT_PATH))
    expect(ekspor?.url).toContain('tab=2')
  })

  it('mematikan tombol ekspor saat tidak ada yang dapat diekspor', async () => {
    // Berkas kosong yang tetap terunduh tidak dapat dibedakan dari ekspor yang gagal
    // diam-diam.
    stubDefaultFetch([])
    await renderLoaded()

    expect(screen.getByRole('button', { name: /Export To Excel/ })).toBeDisabled()
  })

  it('mempertahankan judul tombol Pega meski berkasnya CSV', async () => {
    // `pyButtonLabel Export To Excel` (`D-13`). Activity lamanya pun memanggil
    // `pxConvertResultsToCSV`, bukan penulis Excel mana pun.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('button', { name: /Export To Excel/ })).toBeInTheDocument()
  })
})

describe('portal', () => {
  it('meminta pengguna memilih entitas lebih dulu', async () => {
    // `R-20`: antrean satu badan hukum bukan antrean badan hukum lain.
    stubDefaultFetch()
    useSelectedPortal.getState().clear()
    renderPage()

    expect(await screen.findByText(/Pilih entitas lebih dulu/)).toBeInTheDocument()
  })

  it('mengirim portal aktif pada setiap permintaan', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const call = lastListCall()
    const headers = new Headers(call?.init?.headers)
    expect(headers.get('X-Portal')).toBe('ASM')
  })
})
