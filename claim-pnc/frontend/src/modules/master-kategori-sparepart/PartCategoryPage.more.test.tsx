import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { PartCategoryPage } from './PartCategoryPage'

/** Uji tambahan Master Kategori Sparepart: cabang galat, keputusan, dan pengurutan. Data KARANGAN. */

const ROUTE = '/api/master/kategori-sparepart'

const ROWS = [
  { id_kategori_sparepart: '9', nama_kategori_sparepart: 'ATTACHMENT', status: '0', status_label: 'Waiting Approval' },
  { id_kategori_sparepart: '4', nama_kategori_sparepart: 'ELECTRICAL', status: '0', status_label: 'Waiting Approval' },
]

type Call = { url: string; method: string; body: unknown }

let calls: Call[] = []

type Answer = Response | Promise<Response> | 'putus' | 'rusak' | undefined

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
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
    if (custom === 'rusak') {
      return Promise.resolve({
        get status(): number {
          throw new TypeError('rusak')
        },
      } as unknown as Response)
    }
    if (custom) return Promise.resolve(custom)
    if (url.endsWith('/keputusan')) {
      return Promise.resolve(
        json(200, { jumlah_berubah: 1, status: '2', status_label: 'Reject', portal: 'ASM' }),
      )
    }
    if (call.method === 'GET') {
      return Promise.resolve(json(200, { kategori_sparepart: ROWS, status: '0', portal: 'ASM' }))
    }
    return Promise.resolve(json(200, { kategori_sparepart: ROWS[0], portal: 'ASM' }))
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <PartCategoryPage />
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
      answer: (): Answer => json(500, { kode: 'galat_internal', pesan: 'Basis data sibuk.' }),
      title: 'Daftar kategori sparepart tidak dapat dimuat',
      description: 'Basis data sibuk.',
    },
    {
      name: 'galat bukan API',
      answer: (): Answer => 'rusak',
      title: 'Terjadi kesalahan pada sistem',
    },
  ])('menampilkan galat muat: $name', async ({ answer, title, description }) => {
    installFetch((call) => (isList(call) ? answer() : undefined))
    show()

    expect(screen.getByText('Memuat daftar kategori sparepart…')).toBeInTheDocument()
    expect(await screen.findByText(title)).toBeInTheDocument()
    if (description) expect(screen.getByText(description)).toBeInTheDocument()
  })

  it('menampilkan pesan kosong per tab, mengurutkan, dan memuat ulang', async () => {
    installFetch((call) =>
      isList(call) && call.url.includes('status=2')
        ? json(200, { kategori_sparepart: [], status: '2', portal: 'ASM' })
        : undefined,
    )
    show()
    const table = await screen.findByRole('table')

    const id = within(table).getByRole('columnheader', { name: /^ID Kategori Sparepart/ })
    await userEvent.click(within(id).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('ELECTRICAL')
    const name = within(table).getByRole('columnheader', { name: /^Nama Kategori Sparepart/ })
    await userEvent.click(within(name).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('ATTACHMENT')

    const before = calls.filter(isList).length
    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(calls.filter(isList).length).toBe(before + 1))

    await userEvent.click(screen.getByRole('button', { name: 'Reject' }))
    expect(await screen.findByText('Belum ada kategori sparepart pada tab Reject.')).toBeInTheDocument()
  })
})

describe('keputusan', () => {
  async function openPending() {
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Waiting Approval' }))
    await waitFor(() => expect(screen.getAllByRole('checkbox')).toHaveLength(2))
  }

  it('mengatur pilihan lalu menolak', async () => {
    installFetch()
    await openPending()

    await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih ATTACHMENT' }))
    await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih ELECTRICAL' }))
    expect(screen.getByText('2 kategori')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih ATTACHMENT' }))
    expect(screen.getByText('1 kategori')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Bersihkan' }))
    expect(screen.getByText('Centang kategori yang akan diputuskan.')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih ELECTRICAL' }))
    await userEvent.click(screen.getByRole('button', { name: 'Reject terpilih' }))

    expect(await screen.findByText(/dipindahkan ke/)).toHaveTextContent(
      '1 kategori sparepart dipindahkan ke Reject.',
    )
    expect(calls.find((c) => c.url.endsWith('/keputusan'))?.body).toEqual({
      id_kategori_sparepart: ['4'],
      status: '2',
    })
  })

  it.each([
    {
      name: 'galat API',
      answer: (): Answer => json(409, { kode: 'konflik', pesan: 'Sudah diputuskan.' }),
      text: 'Sudah diputuskan.',
    },
    {
      name: 'jaringan putus',
      answer: (): Answer => 'putus',
      text: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
    },
  ])('menampilkan galat keputusan: $name', async ({ answer, text }) => {
    installFetch((call) => (call.url.endsWith('/keputusan') ? answer() : undefined))
    await openPending()
    await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih ELECTRICAL' }))
    await userEvent.click(screen.getByRole('button', { name: 'Approve terpilih' }))

    expect(await screen.findByText('Keputusan belum tersimpan')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
  })
})

describe('form', () => {
  async function openAdd() {
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Kategori Sparepart' })
    await userEvent.type(within(form).getByLabelText('Nama Kategori Sparepart'), 'HIDROLIK')
    return form
  }

  it('menyorot pelanggaran server pada isiannya', async () => {
    installFetch((call) =>
      call.method === 'POST'
        ? json(422, {
            kode: 'validasi_gagal',
            pesan: 'x',
            detail: [
              { kolom: 'nama_kategori_sparepart', pesan: 'Nama ditolak server.' },
              { kolom: 'lain', pesan: 'Diabaikan.' },
            ],
          })
        : undefined,
    )
    const form = await openAdd()
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText('Nama ditolak server.')).toBeInTheDocument()
    expect(within(form).queryByText('Diabaikan.')).not.toBeInTheDocument()
    expect(within(form).queryByText('Belum dapat disimpan')).not.toBeInTheDocument()
  })

  it.each([
    {
      name: 'validasi tanpa rincian',
      answer: (): Answer => json(422, { kode: 'validasi_gagal', pesan: 'Ditolak tanpa rincian.' }),
      title: 'Belum dapat disimpan',
    },
    {
      name: 'sudah tidak ada',
      answer: (): Answer => json(404, { kode: 'tidak_ditemukan', pesan: 'x' }),
      title: 'Baris ini sudah tidak ada',
    },
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
      title: 'Terjadi kesalahan pada sistem',
    },
    {
      name: 'jaringan putus',
      answer: (): Answer => 'putus',
      title: 'Server Claim PNC tidak dapat dihubungi',
    },
  ])('menampilkan galat simpan: $name', async ({ answer, title }) => {
    installFetch((call) => (call.method === 'POST' ? answer() : undefined))
    const form = await openAdd()
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText(title)).toBeInTheDocument()
  })

  it('tidak menampilkan kotak pesan untuk galat bukan API lalu menutup lewat Batal', async () => {
    installFetch((call) => (call.method === 'POST' ? 'rusak' : undefined))
    const form = await openAdd()
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(within(form).getByRole('button', { name: 'Simpan' })).toBeEnabled())
    expect(calls.some((c) => c.method === 'POST')).toBe(true)
    expect(within(form).queryByText('Terjadi kesalahan pada sistem')).not.toBeInTheDocument()
    await userEvent.click(within(form).getByRole('button', { name: 'Batal' }))
    expect(screen.queryByRole('form', { name: 'Tambah Kategori Sparepart' })).not.toBeInTheDocument()
  })

  it('menolak nama yang terlalu panjang di layar', async () => {
    installFetch()
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const input = await screen.findByLabelText('Nama Kategori Sparepart')
    // maxLength dicabut supaya ketikan melewati batas, dan yang menolaknya skema form.
    input.removeAttribute('maxlength')
    await userEvent.type(input, 'X'.repeat(101))
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(
      await screen.findByText('Nama kategori sparepart paling panjang 100 karakter.'),
    ).toBeInTheDocument()
    expect(calls.some((c) => c.method === 'POST')).toBe(false)
  })
})
