import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
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
      judul: 'Outstanding per PIC',
      kolom: [
        { kunci: 'pic', judul: 'Nama PIC' },
        { kunci: 'jumlah', judul: 'Jumlah Klaim' },
      ],
    },
  ],
  keputusan: {
    dapat_diputuskan: false,
    alasan_wajib_saat_menolak: false,
  },
  punya_penyaring_periode: false,
}

const TAB_APPROVAL_MASTER: Tab = {
  kode: '4',
  nama: 'Approval Master',
  keterangan: 'Ringkasan seluruh antrean yang menunggu persetujuan Anda.',
  jenis: 'ringkasan',
  keputusan: { dapat_diputuskan: false, alasan_wajib_saat_menolak: false },
  punya_penyaring_periode: false,
}

/** Antrean yang tabelnya PUNYA kolom alasan. */
const TAB_BENGKEL: Tab = {
  kode: '5',
  nama: 'Master Bengkel',
  keterangan: 'Pengajuan data bengkel yang menunggu persetujuan.',
  jenis: 'antrean',
  kolom: [
    { kunci: 'id', judul: 'ID Bengkel' },
    { kunci: 'nama', judul: 'Nama Bengkel' },
  ],
  keputusan: {
    dapat_diputuskan: true,
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
  kolom: [
    { kunci: 'no_klaim', judul: 'No Klaim' },
    { kunci: 'no_akseptasi', judul: 'No Akseptasi' },
  ],
  keputusan: {
    dapat_diputuskan: true,
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
]

/** Isi antrean Master Bengkel. Seluruhnya KARANGAN (`D-69`). */
const BARIS_BENGKEL = [
  { kunci: 'BGK-001', sel: { id: 'BGK-001', nama: 'Bengkel Contoh Satu' } },
  { kunci: 'BGK-002', sel: { id: 'BGK-002', nama: 'Bengkel Contoh Dua' } },
]

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

/** Peladen tiruan yang menjawab bentuk layar, pencacah, dan isi tab. */
function stubDefaultFetch(options?: {
  meta?: MetadataResponse
  decide?: () => Response
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
    const wanted = new URL(url, 'http://uji.invalid').searchParams.get('tab') ?? '1'

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
            : {
                tab: TAB_OUTSTANDING,
                panel: [
                  {
                    kunci: 'pic',
                    judul: 'Outstanding per PIC',
                    kolom: TAB_OUTSTANDING.panel![0]!.kolom,
                    baris: [
                      { pic: { teks: 'ALL' }, jumlah: { jumlah: 354 } },
                      { pic: { teks: 'ANDIKA' }, jumlah: { jumlah: 18 } },
                    ],
                  },
                ],
              }

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
    expect(screen.getByRole('tab', { name: /Master Bengkel/ })).toBeInTheDocument()
  })

  it('membuka tab bawaan yang ditetapkan server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('tab', { name: /Outstanding/ })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('menempelkan angka pencacah pada tabnya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      within(screen.getByRole('tab', { name: /Outstanding/ })).getByText('354'),
    ).toBeInTheDocument()
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

    await userEvent.click(screen.getByRole('tab', { name: /Master Bengkel/ }))

    expect(await screen.findByRole('button', { name: 'Export' })).toBeEnabled()
  })

  it('menampilkan selisih terencana kepada pengguna', async () => {
    // `D-54`: selisih yang hanya tercatat di komentar akan dilaporkan berulang kali sebagai
    // kerusakan oleh orang yang membandingkan layar ini dengan Pega berdampingan.
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByText(/Yang berbeda dari layar lama/)).toBeInTheDocument()
  })
})

describe('dashboard', () => {
  it('menggambar panel beserta barisnya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText('Outstanding per PIC')).toBeInTheDocument()
    expect(await screen.findByText('ANDIKA')).toBeInTheDocument()
  })

  it('TIDAK menggambar penyaring periode pada tab Outstanding', async () => {
    // Ketiga kueri Pega yang memasoknya tidak menyaring tanggal sama sekali. Menggambar
    // penyaring di sana akan menyarankan kemampuan yang tidak ada, dan mengubah arti angkanya
    // bila kemudian benar-benar dipasang.
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText('Outstanding per PIC')).toBeInTheDocument()
    expect(screen.queryByLabelText('Periode')).not.toBeInTheDocument()
  })
})

describe('antrean persetujuan', () => {
  it('menggambar tombol keputusan hanya pada tab antrean', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Tab dashboard: tidak ada tombol keputusan.
    expect(screen.queryByRole('button', { name: 'Setujui' })).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('tab', { name: /Master Bengkel/ }))

    expect(await screen.findByRole('button', { name: 'Setujui' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tolak' })).toBeInTheDocument()
  })

  it('menonaktifkan tombol keputusan selama belum ada baris yang dipilih', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await userEvent.click(screen.getByRole('tab', { name: /Master Bengkel/ }))

    expect(await screen.findByRole('button', { name: 'Setujui' })).toBeDisabled()
  })

  it('mengirim kunci baris APA ADANYA, bukan menyusunnya ulang', async () => {
    // Bentuk kunci berbeda tiap antrean — satu di antaranya gabungan empat kolom. Menyusunnya
    // di layar berarti aturan kuncinya hidup di dua tempat.
    stubDefaultFetch()
    await renderLoaded()
    await userEvent.click(screen.getByRole('tab', { name: /Master Bengkel/ }))

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
    await userEvent.click(screen.getByRole('tab', { name: /Master Bengkel/ }))

    expect(await screen.findByText(/Alasan Status Bengkel/)).toBeInTheDocument()
  })

  it('menahan tombol Setujui beserta ALASANNYA pada antrean pembayaran', async () => {
    // Uji terpenting di berkas ini: menyetujui pembayaran tanpa transfer kasir menandai
    // pembayaran sudah disetujui padahal tidak pernah sampai ke kasir.
    stubDefaultFetch()
    await renderLoaded()
    await userEvent.click(screen.getByRole('tab', { name: /Payment Klaim Akseptasi/ }))

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
    await userEvent.click(screen.getByRole('tab', { name: /Master Bengkel/ }))

    await userEvent.click(await screen.findByLabelText('Pilih baris BGK-001'))
    await userEvent.click(screen.getByLabelText('Pilih baris BGK-002'))
    await userEvent.click(screen.getByRole('button', { name: 'Setujui' }))

    expect(await screen.findByText(/1 dari 2 baris berubah/)).toBeInTheDocument()
  })
})

describe('ringkasan Approval Master', () => {
  it('menyatakan antrean yang sumbernya TIDAK dapat dibaca, bukan menulis nol', async () => {
    // Angka nol yang sesungguhnya berarti "tidak terbaca" membuat penyelia mengira antreannya
    // kosong — kebohongan yang tidak menghasilkan satu pun galat.
    stubDefaultFetch()
    await renderLoaded()
    await userEvent.click(screen.getByRole('tab', { name: /Approval Master/ }))

    expect(await screen.findByText(/SPAREPART_HE/)).toBeInTheDocument()
  })

  it('membuka antreannya saat kartunya diklik', async () => {
    stubDefaultFetch()
    await renderLoaded()
    await userEvent.click(screen.getByRole('tab', { name: /Approval Master/ }))

    await userEvent.click(await screen.findByRole('button', { name: /Master Bengkel/ }))

    expect(await screen.findByRole('button', { name: 'Setujui' })).toBeInTheDocument()
  })
})
