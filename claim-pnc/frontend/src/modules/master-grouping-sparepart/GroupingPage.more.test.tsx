import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { GroupingPage } from './GroupingPage'

/**
 * Uji tambahan Master Grouping Sparepart: cabang galat muat/simpan/keputusan, mode ubah,
 * dan keadaan isian yang tidak tersentuh uji utama. Seluruh data KARANGAN (`D-69`).
 */

const ROUTE = '/api/master/grouping-sparepart'

const OPTIONS = {
  panel: [
    { kode: 'PNL01', nama: 'BUMPER DEPAN' },
    { kode: 'PNL09', nama: 'PANEL TANPA SISI' },
  ],
  tipe_kendaraan: [{ kode: 'TY01', nama: 'EXCAVATOR' }],
  portal: 'ASM',
}

const PART = {
  nomor_sparepart: 'SP-1001',
  nama_sparepart: 'FILTER OLI',
  kategori_sparepart: 'KAT01',
  tipe_sparepart: 'TIP01',
  kode_sparepart: 'KD-1001',
  tanggal_produksi: '12/05/2024',
  portal: 'ASM',
}

const ROW = {
  id_grouping: '7',
  nomor_sparepart: 'SP-1001',
  nama_sparepart: '',
  kategori_sparepart: 'KAT01',
  tipe_sparepart: 'TIP01',
  kode_sparepart: 'KD-1001',
  tanggal_produksi: '12/05/2024',
  id_panel: 'PNL01',
  nama_panel: 'BUMPER DEPAN',
  sisi: '1',
  sisi_label: '',
  no_rangka: '',
  tipe_kendaraan: 'EXCAVATOR',
  grouping_dengan_no_rangka: '',
  nomor_grup: '',
  catatan: '',
  status: '1',
  status_label: 'Approve',
}

const ROW_B = { ...ROW, id_grouping: '8', nomor_sparepart: 'SP-0001', no_rangka: 'MHF1', nomor_grup: '0004' }

type Call = { url: string; method: string; body: unknown }

let calls: Call[] = []

type Answer = Response | Promise<Response> | 'putus' | 'rusak' | undefined

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function brokenResponse(): Response {
  return {
    get status(): number {
      throw new TypeError('jawaban rusak')
    },
  } as unknown as Response
}

function installFetch(answer: (call: Call) => Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
      body: init?.body ? JSON.parse(init.body as string) : undefined,
    }
    calls.push(call)
    const custom = answer(call)
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
    if (custom === 'rusak') return Promise.resolve(brokenResponse())
    if (custom) return Promise.resolve(custom)
    if (url.startsWith(`${ROUTE}/pilihan`)) return Promise.resolve(json(200, OPTIONS))
    if (url.startsWith(`${ROUTE}/sisi`)) {
      const panel = new URL(url, 'https://x').searchParams.get('id_panel')
      const sisi = panel === 'PNL01' ? [{ kode: '1', nama: 'KIRI' }] : []
      return Promise.resolve(json(200, { sisi, portal: 'ASM' }))
    }
    if (url.startsWith(`${ROUTE}/sparepart`)) return Promise.resolve(json(200, PART))
    if (url.endsWith('/keputusan')) {
      return Promise.resolve(
        json(200, { jumlah_berubah: 1, status: '2', status_label: 'Reject', portal: 'ASM' }),
      )
    }
    if (call.method === 'GET') {
      return Promise.resolve(json(200, { grouping: [ROW, ROW_B], status: '1', portal: 'ASM' }))
    }
    return Promise.resolve(json(200, { grouping: ROW, portal: 'ASM' }))
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <GroupingPage />
    </QueryClientProvider>,
  )
}

const isList = (call: Call) => call.method === 'GET' && call.url.startsWith(`${ROUTE}?`)

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('daftar', () => {
  it.each([
    { name: 'jaringan putus', answer: (): Answer => 'putus', title: 'Server Claim PNC tidak dapat dihubungi' },
    {
      name: 'portal tidak disebut',
      answer: (): Answer => json(400, { kode: 'portal_tidak_disebut', pesan: 'x' }),
      title: 'Portal entitas belum dipilih',
    },
    {
      name: 'portal tidak dikenal',
      answer: (): Answer => json(400, { kode: 'portal_tidak_dikenal', pesan: 'x' }),
      title: 'Portal entitas belum dipilih',
    },
    {
      name: 'portal belum siap',
      answer: (): Answer => json(503, { kode: 'portal_belum_siap', pesan: 'x' }),
      title: 'Basis data entitas ini belum tersedia',
    },
    {
      name: 'galat API lain',
      answer: (): Answer => json(500, { kode: 'galat_internal', pesan: 'x' }),
      title: 'Daftar grouping tidak dapat dimuat',
      description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
    },
    {
      name: 'galat bukan API',
      answer: (): Answer => 'rusak',
      title: 'Daftar grouping tidak dapat dimuat',
      description: 'Coba beberapa saat lagi.',
    },
  ])('menampilkan galat muat: $name', async ({ answer, title, description }) => {
    installFetch((call) => (isList(call) ? answer() : undefined))
    show()

    expect(screen.getByText('Memuat daftar grouping…')).toBeInTheDocument()
    expect(await screen.findByText(title)).toBeInTheDocument()
    if (description) expect(screen.getByText(description)).toBeInTheDocument()
  })

  it('menampilkan tanda hubung untuk isian kosong dan pesan kosong per tab', async () => {
    installFetch((call) =>
      isList(call) && call.url.includes('status=2')
        ? json(200, { grouping: [], status: '2', portal: 'ASM' })
        : undefined,
    )
    show()

    const row = (await screen.findByText('SP-1001')).closest('tr')!
    expect(within(row).getAllByText('—')).toHaveLength(4)

    await userEvent.click(screen.getByRole('button', { name: 'Reject' }))
    expect(await screen.findByText('Belum ada grouping pada tab Reject.')).toBeInTheDocument()
  })

  it('mengurutkan menurut setiap kolom', async () => {
    installFetch()
    show()
    const table = await screen.findByRole('table')

    for (const title of [
      'ID',
      'No Sparepart',
      'Nama Sparepart',
      'Nama Panel',
      'Sisi Panel',
      'No Rangka',
      'Nomor Grup',
    ]) {
      const column = within(table).getByRole('columnheader', { name: new RegExp(`^${title}`) })
      await userEvent.click(within(column).getByRole('button'))
      expect(column).toHaveAttribute('aria-sort', 'ascending')
    }
    const nomor = within(table).getByRole('columnheader', { name: /^No Sparepart/ })
    await userEvent.click(within(nomor).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('SP-0001')
  })

  it('memuat ulang daftar lewat Refresh', async () => {
    installFetch()
    show()
    await screen.findByText('SP-1001')
    const before = calls.filter(isList).length

    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))

    await waitFor(() => expect(calls.filter(isList).length).toBe(before + 1))
  })
})

describe('keputusan', () => {
  async function openPending() {
    show()
    await screen.findByText('SP-1001')
    await userEvent.click(screen.getByRole('button', { name: 'Waiting Approval' }))
    await waitFor(() => expect(screen.getAllByRole('checkbox')).toHaveLength(2))
  }

  it('mengatur pilihan lalu menolak dan menyebut hasilnya', async () => {
    installFetch()
    await openPending()

    expect(screen.getByText('Centang grouping yang akan diputuskan.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Reject terpilih' })).toBeDisabled()

    const [first, second] = screen.getAllByRole('checkbox')
    await userEvent.click(first!)
    await userEvent.click(second!)
    expect(screen.getByText('2 grouping')).toBeInTheDocument()
    await userEvent.click(first!)
    expect(screen.getByText('1 grouping')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Bersihkan' }))
    expect(screen.getByText('Centang grouping yang akan diputuskan.')).toBeInTheDocument()

    await userEvent.click(second!)
    await userEvent.click(screen.getByRole('button', { name: 'Reject terpilih' }))

    expect(await screen.findByText(/1 grouping dipindahkan ke/)).toHaveTextContent(
      '1 grouping dipindahkan ke Reject.',
    )
    expect(calls.find((c) => c.url.endsWith('/keputusan'))?.body).toEqual({
      id_grouping: ['8'],
      status: '2',
    })
    // Pilihan dibersihkan setelah keputusan tersimpan.
    expect(screen.getByText('Centang grouping yang akan diputuskan.')).toBeInTheDocument()
  })

  it('menampilkan pesan server bila keputusan ditolak', async () => {
    installFetch((call) =>
      call.url.endsWith('/keputusan')
        ? json(409, { kode: 'konflik', pesan: 'Grouping sudah diputuskan.' })
        : undefined,
    )
    await openPending()
    await userEvent.click(screen.getAllByRole('checkbox')[0]!)
    await userEvent.click(screen.getByRole('button', { name: 'Approve terpilih' }))

    expect(await screen.findByText('Keputusan belum tersimpan')).toBeInTheDocument()
    expect(screen.getByText('Grouping sudah diputuskan.')).toBeInTheDocument()

    // Berpindah tab membuang galat keputusan.
    await userEvent.click(screen.getByRole('button', { name: 'Approve' }))
    expect(screen.queryByText('Keputusan belum tersimpan')).not.toBeInTheDocument()
  })

  it('menampilkan pesan umum bila keputusan gagal karena jaringan', async () => {
    installFetch((call) => (call.url.endsWith('/keputusan') ? 'putus' : undefined))
    await openPending()
    await userEvent.click(screen.getAllByRole('checkbox')[0]!)
    await userEvent.click(screen.getByRole('button', { name: 'Approve terpilih' }))

    expect(
      await screen.findByText(
        'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
      ),
    ).toBeInTheDocument()
  })
})

describe('form ubah', () => {
  it('menampilkan ID dan nomor grup lalu mengirim PUT ke grouping itu', async () => {
    installFetch()
    show()
    const row = (await screen.findByText('SP-1001')).closest('tr')!
    await userEvent.click(within(row).getByRole('button', { name: 'Ubah' }))

    expect(
      await screen.findByRole('heading', { name: 'Ubah grouping SP-1001 — BUMPER DEPAN' }),
    ).toBeInTheDocument()
    expect(screen.getByText('ID grouping').parentElement).toHaveTextContent('7')
    expect(screen.getByText('Nomor grup kendaraan').parentElement).toHaveTextContent('—')
    // Isian turunan diambil dari pencarian sparepart yang berjalan saat form dibuka.
    expect(await screen.findByText('FILTER OLI')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tambah' })).toBeDisabled()

    await userEvent.type(screen.getByLabelText('No Rangka'), 'MHF9')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: /Ubah grouping/ })).not.toBeInTheDocument(),
    )
    const sent = calls.find((c) => c.method === 'PUT')
    expect(sent?.url).toBe(`${ROUTE}/7`)
    expect(sent?.body).toMatchObject({ no_rangka: 'MHF9', sisi: '1', nama_panel: 'BUMPER DEPAN' })
  })

  it('menutup form saat berpindah tab', async () => {
    installFetch()
    show()
    const row = (await screen.findByText('SP-1001')).closest('tr')!
    await userEvent.click(within(row).getByRole('button', { name: 'Ubah' }))
    await screen.findByRole('heading', { name: /Ubah grouping/ })

    await userEvent.click(screen.getByRole('button', { name: 'Reject' }))
    expect(screen.queryByRole('heading', { name: /Ubah grouping/ })).not.toBeInTheDocument()
  })
})

describe('form tambah', () => {
  async function openAdd() {
    show()
    await screen.findByText('SP-1001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    return screen.findByRole('heading', { name: 'Tambah Master Grouping Sparepart' })
  }

  async function fillValid() {
    await userEvent.type(screen.getByLabelText('Nomor Sparepart'), 'SP-1001')
    await userEvent.selectOptions(await screen.findByLabelText('Nama Panel'), 'PNL01')
    await screen.findByRole('option', { name: 'KIRI' })
    await userEvent.selectOptions(screen.getByLabelText('Sisi'), '1')
    await userEvent.type(screen.getByLabelText('No Rangka'), 'MHF5')
  }

  it('menyebut panel yang belum punya sisi', async () => {
    installFetch()
    await openAdd()

    await userEvent.selectOptions(await screen.findByLabelText('Nama Panel'), 'PNL09')
    expect(
      await screen.findByText(/Panel ini belum punya sisi yang terdaftar\./),
    ).toBeInTheDocument()
  })

  it('menyebut pencarian sparepart yang sedang berjalan', async () => {
    let release: (r: Response) => void = () => {}
    installFetch((call) =>
      call.url.startsWith(`${ROUTE}/sparepart`)
        ? new Promise<Response>((resolve) => {
            release = resolve
          })
        : undefined,
    )
    await openAdd()

    await userEvent.type(screen.getByLabelText('Nomor Sparepart'), 'SP-1001')
    await userEvent.tab()
    expect(await screen.findByText('Mencari data sparepart…')).toBeInTheDocument()
    release(json(200, PART))
    expect(await screen.findByText('FILTER OLI')).toBeInTheDocument()
  })

  it('menyebut daftar panel yang gagal dimuat', async () => {
    installFetch((call) =>
      call.url.startsWith(`${ROUTE}/pilihan`) ? json(500, { kode: 'x', pesan: 'x' }) : undefined,
    )
    await openAdd()

    expect(
      await screen.findByText('Daftar Panel dan Tipe Kendaraan tidak dapat dimuat'),
    ).toBeInTheDocument()
  })

  it('menyorot pelanggaran server pada isiannya masing-masing', async () => {
    installFetch((call) =>
      call.method === 'POST'
        ? json(422, {
            kode: 'validasi_gagal',
            pesan: 'x',
            detail: [
              { kolom: 'no_rangka', pesan: 'No rangka ditolak server.' },
              { kolom: 'catatan', pesan: 'Catatan ditolak server.' },
              { kolom: 'kolom_asing', pesan: 'Tidak disorot.' },
            ],
          })
        : undefined,
    )
    await openAdd()
    await fillValid()
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('No rangka ditolak server.')).toBeInTheDocument()
    expect(screen.getByText('Catatan ditolak server.')).toBeInTheDocument()
    expect(screen.queryByText('Tidak disorot.')).not.toBeInTheDocument()
    expect(screen.queryByText('Belum dapat disimpan')).not.toBeInTheDocument()
  })

  it.each([
    {
      name: 'validasi tanpa rincian',
      answer: (): Answer => json(422, { kode: 'validasi_gagal', pesan: 'Ditolak tanpa rincian.' }),
      texts: ['Belum dapat disimpan', 'Ditolak tanpa rincian.'],
    },
    {
      name: 'portal tidak disebut',
      answer: (): Answer => json(400, { kode: 'portal_tidak_disebut', pesan: 'x' }),
      texts: ['Pilih portal entitas di bagian atas halaman, lalu simpan lagi.'],
    },
    {
      name: 'portal tidak dikenal',
      answer: (): Answer => json(400, { kode: 'portal_tidak_dikenal', pesan: 'x' }),
      texts: ['Pilih portal entitas di bagian atas halaman, lalu simpan lagi.'],
    },
    {
      name: 'portal belum siap',
      answer: (): Answer => json(503, { kode: 'portal_belum_siap', pesan: 'x' }),
      texts: ['Basis data entitas ini belum tersedia'],
    },
    {
      name: 'galat API lain',
      answer: (): Answer => json(500, { kode: 'galat_internal', pesan: 'x' }),
      texts: ['Terjadi kesalahan pada sistem'],
    },
    {
      name: 'jaringan putus',
      answer: (): Answer => 'putus',
      texts: ['Server Claim PNC tidak dapat dihubungi'],
    },
  ])('menampilkan galat simpan: $name', async ({ answer, texts }) => {
    installFetch((call) => (call.method === 'POST' ? answer() : undefined))
    await openAdd()
    await fillValid()
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    for (const text of texts) expect(await screen.findByText(text)).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Tambah Master Grouping Sparepart' })).toBeInTheDocument()
  })

  it('tidak menampilkan kotak pesan untuk galat bukan API lalu menutup lewat Batal', async () => {
    installFetch((call) => (call.method === 'POST' ? 'rusak' : undefined))
    await openAdd()
    await fillValid()
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(screen.getByRole('button', { name: 'Simpan' })).toBeEnabled())
    expect(calls.some((c) => c.method === 'POST')).toBe(true)
    expect(screen.queryByText('Terjadi kesalahan pada sistem')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Batal' }))
    expect(
      screen.queryByRole('heading', { name: 'Tambah Master Grouping Sparepart' }),
    ).not.toBeInTheDocument()
  })
})
