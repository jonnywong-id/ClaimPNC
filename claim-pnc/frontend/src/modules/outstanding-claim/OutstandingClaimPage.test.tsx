import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import type { DetailResponse, LayoutResponse } from './types'

const PATH = '/api/outstanding-claim'
const LAYOUT_PATH = `${PATH}/tata-letak`
const CLAIM_ID = 'CLMP-1001'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'admintreaty1',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

/**
 * Susunan layar contoh — dipangkas menjadi tiga kelompok.
 *
 * Bentuknya yang diuji, bukan kelengkapannya: layar menggambar apa pun yang server kirimkan,
 * dan menyalin seluruh 97 isian ke sini hanya memindahkan daftar yang sama ke tempat kedua.
 */
const LAYOUT: LayoutResponse = {
  kelompok: [
    {
      kode: 'treaty',
      judul: 'Treaty Information',
      isian: [
        { kunci: 'id_master', judul: 'Treaty ID' },
        { kunci: 'teritorial_scope', judul: 'TERITORIAL SCOPE', terhalang: true },
      ],
      grid: [],
    },
    {
      kode: 'klaim',
      judul: 'Claim Information',
      isian: [
        { kunci: 'insured_name', judul: 'Insured Name' },
        { kunci: 'date_of_loss', judul: 'Date Of Loss' },
        { kunci: 'pla_no_ceding', judul: 'Pla No Ceding' },
      ],
      grid: [],
    },
    {
      kode: 'interest',
      judul: 'Insured Interest',
      isian: [],
      grid: ['interest_list', 'attachment'],
    },
  ],
  grid: [
    {
      kode: 'interest_list',
      judul: 'Insured Interest',
      kolom: [
        { kunci: 'object_name', judul: 'Insured Interest' },
        { kunci: 'currency', judul: 'Currency' },
        { kunci: 'tsi_per_object', judul: 'Value' },
      ],
    },
    {
      kode: 'attachment',
      judul: 'Attachment',
      kolom: [
        { kunci: 'category', judul: 'Category' },
        { kunci: 'count_attach', judul: 'Count Attach' },
      ],
      terhalang: true,
      alasan_terhalang:
        'Daftar lampiran diisi Report Definition BrowseUpRegisterDoc_rd, yang tidak ada ' +
        'di export (R-16).',
      pemilik_penghalang: 'Tim Pega — dibutuhkan export Report Definition tersebut.',
    },
  ],
  portal: 'ASM',
}

/**
 * Rincian contoh. Seluruh isinya KARANGAN — `D-69` melarang data nasabah ditulis di berkas
 * yang di-commit, dan larangan itu berlaku untuk data uji sama seperti untuk dokumen.
 */
const DETAIL: DetailResponse = {
  no_klaim: CLAIM_ID,
  status_kerja: 'Pending-Teknik',
  operator_pengubah: 'PICTREATY1',
  isian: {
    id_master: 'TRP-2026-01',
    insured_name: 'PT Contoh Sejahtera',
    date_of_loss: '2026-08-14',
    // `pla_no_ceding` SENGAJA tidak ada kuncinya: ia meniru isian yang jalurnya tidak
    // ditemukan di dokumen klaim.
  },
  baris: {
    interest_list: [
      { object_name: 'Bangunan Gudang Contoh', currency: 'IDR', tsi_per_object: '10000000000' },
      { object_name: 'Isi Gudang Contoh', currency: 'IDR', tsi_per_object: '5000000000' },
    ],
  },
  portal: 'ASM',
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

function stubDefaultFetch(detail: DetailResponse = DETAIL) {
  stubFetch((url) => {
    if (url === LAYOUT_PATH) return jsonResponse(200, LAYOUT)
    return jsonResponse(200, detail)
  })
}

function renderPage(claimID: string = CLAIM_ID) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[`/outstanding-claim/${claimID}`]}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

async function renderLoaded() {
  renderPage()
  await screen.findByRole('heading', { name: /Outstanding Claim/ })
  await screen.findByText('Treaty Information')
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
  it('menggambar nomor klaim dari alamatnya', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(
      screen.getByRole('heading', { name: `Outstanding Claim — ${CLAIM_ID}` }),
    ).toBeInTheDocument()
  })

  it('menggambar kelompok yang ditetapkan server, bukan daftar tetap di layar', async () => {
    stubDefaultFetch()
    await renderLoaded()

    // Dicari sebagai JUDUL, bukan sebagai teks mana pun: "Insured Interest" muncul tiga
    // kali di layar ini — judul kelompok, judul gridnya, dan judul kolom pertama grid itu —
    // dan pencarian berbasis teks akan menemukan ketiganya sekaligus.
    for (const judul of ['Treaty Information', 'Claim Information', 'Insured Interest']) {
      expect(screen.getAllByRole('heading', { name: judul }).length).toBeGreaterThan(0)
    }
  })

  it('menyediakan jalan kembali ke antrean', async () => {
    // Layar ini hanya dapat dicapai dari antrean, sehingga jalan pulang adalah hal pertama
    // yang dicari pengguna ketika ia membuka klaim yang keliru.
    stubDefaultFetch()
    await renderLoaded()

    expect(
      screen.getByRole('link', { name: /Kembali ke Claim Treatyin In Progress/ }),
    ).toHaveAttribute('href', '/inbox-claim-treaty-prop')
  })
})

describe('isian', () => {
  it('memformat tanggal yang berbentuk YYYY-MM-DD', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText('14 Agustus 2026')).toBeInTheDocument()
  })

  it('menampilkan nilai yang bentuknya tidak dikenali APA ADANYA', async () => {
    // Nilainya dibaca dari dokumen JSON yang bentuknya tidak dapat diperiksa (`R-08`).
    // Memaksakan pemformatan akan mengubahnya menjadi teks yang salah.
    stubDefaultFetch({ ...DETAIL, isian: { ...DETAIL.isian, date_of_loss: '14/08/2026' } })
    await renderLoaded()

    expect(await screen.findByText('14/08/2026')).toBeInTheDocument()
  })

  it('menggambar tanda pisah untuk isian yang jalurnya tidak ditemukan', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const label = screen.getByText('Pla No Ceding')
    expect(label.parentElement).not.toBeNull()
    expect(within(label.parentElement as HTMLElement).getByText('—')).toBeInTheDocument()
  })
})

describe('isian dan grid yang belum punya sumber', () => {
  it('menggambar isian terhalang dengan penanda, bukan tanda pisah biasa', async () => {
    // Tanpa penanda, isian yang tidak punya sumber sama sekali tampak sama dengan isian
    // yang datanya memang belum diisi — dan yang pertama adalah hal yang harus dilaporkan.
    stubDefaultFetch()
    await renderLoaded()

    const label = screen.getByText('TERITORIAL SCOPE')
    expect(
      within(label.parentElement as HTMLElement).getByText('belum tersedia'),
    ).toBeInTheDocument()
  })

  it('menjelaskan alasan dan pemilik penghalang grid, bukan tabel kosong', async () => {
    stubDefaultFetch()
    await renderLoaded()

    expect(await screen.findByText(/BrowseUpRegisterDoc_rd/)).toBeInTheDocument()
    expect(screen.getByText(/Tim Pega/)).toBeInTheDocument()

    // Hanya SATU tabel di layar — grid lampiran tidak digambar sebagai tabel kosong, yang
    // terbaca sebagai "tidak ada lampiran" padahal yang benar adalah "belum dapat dibaca".
    expect(screen.getAllByRole('table')).toHaveLength(1)
  })
})

describe('grid', () => {
  it('menggambar kolom dan baris yang ditetapkan server', async () => {
    stubDefaultFetch()
    await renderLoaded()

    const table = await screen.findByRole('table')
    expect(within(table).getByText('Bangunan Gudang Contoh')).toBeInTheDocument()
    expect(within(table).getByText('Isi Gudang Contoh')).toBeInTheDocument()

    for (const judul of ['Insured Interest', 'Currency', 'Value']) {
      expect(within(table).getByRole('columnheader', { name: judul })).toBeInTheDocument()
    }
  })

  it('menggambar kepala kolom meski tidak ada satu pun baris', async () => {
    // Grid yang senarainya ADA tetapi kosong berbeda dari grid yang terhalang. Kepala
    // kolomnya tetap digambar supaya keduanya tidak terlihat sama.
    stubDefaultFetch({ ...DETAIL, baris: { interest_list: [] } })
    await renderLoaded()

    const table = await screen.findByRole('table')
    expect(
      within(table).getByRole('columnheader', { name: 'Currency' }),
    ).toBeInTheDocument()
  })
})

describe('galat', () => {
  it('membedakan klaim yang tidak ada dari kegagalan lain', async () => {
    stubFetch((url) => {
      if (url === LAYOUT_PATH) return jsonResponse(200, LAYOUT)
      return jsonResponse(404, {
        kode: 'tidak_ditemukan',
        pesan:
          'Klaim treaty dengan nomor itu tidak ada di entitas yang sedang dipilih. ' +
          'Periksa nomornya, atau pilih entitas lain di bilah atas.',
      })
    })

    renderPage('CLMP-9999')

    expect(await screen.findByText('Klaim tidak ditemukan')).toBeInTheDocument()

    // Pesannya menyebut PORTAL, karena sebab paling sering dari "tidak ditemukan" pada
    // aplikasi empat badan hukum bukan salah ketik melainkan salah portal.
    expect(screen.getByText(/pilih entitas lain di bilah atas/)).toBeInTheDocument()
  })
})

describe('portal', () => {
  it('menolak membuka layar sebelum entitas dipilih', async () => {
    stubDefaultFetch()
    useSelectedPortal.getState().clear()

    renderPage()

    expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()
  })

  it('menyebut portal pada setiap permintaan rincian', async () => {
    stubDefaultFetch()
    await renderLoaded()

    await vi.waitFor(() => {
      const detailCall = [...calls]
        .reverse()
        .find((c) => c.url === `${PATH}/${CLAIM_ID}`)
      const header = detailCall?.init?.headers as Record<string, string> | undefined
      expect(header?.['X-Portal']).toBe('ASM')
    })
  })
})
