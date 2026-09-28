import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { MetadataResponse, PageInfo, ServiceClaim, Tab } from './types'

const PATH = '/api/inbox-service-center'
const TAB_PATH = `${PATH}/tab`

const SAMPLE_PROFILE = {
  identitas: '90000002',
  nama: 'Contoh Petugas Service Center',
  jenis: 'KARYAWAN',
  login: 'picsc',
  email: 'contoh.picsc@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Keenam kolom, sama untuk keempat tab.
 *
 * Judulnya diambil dari `pyCaption` pada keempat section tab — bukan diterjemahkan (`D-13`).
 */
const KOLOM: Tab['kolom'] = [
  { kunci: 'id', judul: 'ID' },
  { kunci: 'tanggal_input', judul: 'Tanggal Input' },
  { kunci: 'no_polis', judul: 'No Polis' },
  { kunci: 'nasabah', judul: 'Nasabah' },
  { kunci: 'tipe', judul: 'Tipe' },
  { kunci: 'pic', judul: 'PIC' },
]

const TAB_REGISTRASI: Tab = {
  kode: 'registrasi-sc',
  nama: 'Registrasi SC',
  keterangan: 'Klaim portal rekanan yang belum pernah diajukan ke komite.',
  parameter_pega: '',
  kolom: KOLOM,
}

const TAB_REJECTED: Tab = {
  kode: 'rejected',
  nama: 'Rejected',
  keterangan: 'Ditolak komite, termasuk yang diputuskan Total Loss Only (TLO).',
  parameter_pega: '2',
  kolom: KOLOM,
}

const METADATA: MetadataResponse = {
  tab: [TAB_REGISTRASI, TAB_REJECTED],
  tab_bawaan: 'registrasi-sc',
  portal: 'ASM',
  keterbatasan: [
    'Layar ini baru MEMBACA. Menyimpan rincian dan memutuskan komite belum dibangun.',
  ],
}

/**
 * Baris contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas yang
 * di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
const BARIS: ServiceClaim = {
  id: 'SC-000101',
  tanggal_input: '2026-09-22',
  no_polis: '90-001-2026-00000101',
  nasabah: 'Contoh Nasabah Satu',
  tipe: 'Gadget',
  pic: 'PICSC',
  repair_id: '1000101',
  no_klaim: 'PNCN.26.0101',
  imei: 'IMEI-CONTOH-001',
  status_perbaikan: '1',
  status_perbaikan_label: 'Repair Submitted',
  status_persetujuan: '',
  status_persetujuan_label: 'Belum diajukan',
}

const PAGINASI_AKTIF: PageInfo = {
  halaman: 1,
  ukuran: 25,
  total: 1,
  total_halaman: 1,
  aktif: true,
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

/** Peladen tiruan yang menjawab bentuk layar dan satu baris antrean. */
function stubDefaultFetch(
  baris: ServiceClaim[] = [BARIS],
  paginasi: PageInfo = PAGINASI_AKTIF,
  tab: Tab = TAB_REGISTRASI,
) {
  stubFetch((url) => {
    if (url === TAB_PATH) return jsonResponse(200, METADATA)
    return jsonResponse(200, {
      tab,
      baris,
      paginasi: { ...paginasi, total: paginasi.total || baris.length },
      penyaring: { cari: '' },
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
      <MemoryRouter initialEntries={['/inbox-service-center']}>
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
  await screen.findByRole('tab', { name: 'Registrasi SC' })
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
  it('membuka tab bawaan yang ditetapkan server, bukan tab pertama yang kebetulan ada', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(screen.getByRole('tab', { name: 'Registrasi SC' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('menggambar keenam kolom yang ditetapkan server, bukan daftar tetap di layar', async () => {
    stubDefaultFetch()
    await renderLoaded()

    for (const judul of ['ID', 'Tanggal Input', 'No Polis', 'Nasabah', 'Tipe', 'PIC']) {
      expect(await screen.findByRole('columnheader', { name: judul })).toBeInTheDocument()
    }
  })

  it('menampilkan keterbatasan yang dikirim server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText(/baru MEMBACA/i)).toBeInTheDocument()
  })
})

describe('perpindahan tab', () => {
  it('mengirim kode tab yang dipilih, bukan nilai Pega-nya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.click(screen.getByRole('tab', { name: 'Rejected' }))

    // `rejected`, bukan `2`. Nilai Pega dibawa terpisah pada `parameter_pega` demi
    // ketelusuran, dan tidak pernah dipakai sebagai kode di kontrak API.
    expect(lastListCall()?.url).toContain('tab=rejected')
  })

  it('membersihkan kotak cari saat berpindah tab', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const kotak = screen.getByLabelText('Cari')
    await userEvent.type(kotak, '90-001')
    expect(kotak).toHaveValue('90-001')

    await userEvent.click(screen.getByRole('tab', { name: 'Rejected' }))

    // Kata kunci yang terbawa ke tab lain akan membuat antrean tampak kosong padahal
    // isinya ada — dan pengguna tidak punya cara melihat sebabnya.
    expect(screen.getByLabelText('Cari')).toHaveValue('')
  })
})

describe('pencarian', () => {
  // Jam sungguhan, bukan jam tiruan. Layar ini menunggu jeda ketikan SEKALIGUS menunggu
  // janji fetch dan React Query; mengganti jamnya membuat penantian kedua tidak pernah
  // selesai. `waitFor` menanti keduanya tanpa perlu tahu berapa jedanya.
  it('mengirim kata kunci setelah ketikan berhenti', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await userEvent.type(screen.getByLabelText('Cari'), '90-001')

    await vi.waitFor(() => {
      expect(lastListCall()?.url).toContain('cari=90-001')
    })
  })

  it('menyatakan jangkauan pencarian di sebelah kotaknya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Penyaring pencarian sistem lama adalah IRISAN dua kelompok, sehingga mengetik ID
    // saja tidak menghasilkan baris. Tanpa keterangan ini, pengguna akan menyimpulkan
    // antreannya kosong.
    expect(
      screen.getByText(/Mencari dengan ID atau No Klaim saja tidak menghasilkan baris/i),
    ).toBeInTheDocument()
  })
})

describe('paginasi', () => {
  it('menyembunyikan bilah halaman saat server menyatakan paginasinya mati', async () => {
    const banyak = Array.from({ length: 3 }, (_, index) => ({
      ...BARIS,
      id: `SC-00010${index + 1}`,
      repair_id: `100010${index + 1}`,
    }))

    stubDefaultFetch(banyak, { ...PAGINASI_AKTIF, total: 3, aktif: false })
    await renderLoaded()

    await screen.findByText('SC-000101')

    // Saat mencari, sistem lama mengirim seluruh baris yang cocok sekaligus. Menggambar
    // "Berikutnya" di sana berarti tombol yang tidak pernah bisa ditekan.
    expect(screen.queryByRole('button', { name: /berikutnya/i })).not.toBeInTheDocument()
    expect(screen.getByText(/Menampilkan seluruh 3 baris yang cocok/i)).toBeInTheDocument()
  })
})

describe('antrean kosong', () => {
  it('menjelaskan sebabnya, bukan sekadar "tidak ada data"', async () => {
    stubDefaultFetch([], { ...PAGINASI_AKTIF, total: 0 })
    await renderLoaded()

    expect(
      await screen.findByText(/Tidak ada klaim milik Anda pada antrean Registrasi SC/i),
    ).toBeInTheDocument()
  })
})
