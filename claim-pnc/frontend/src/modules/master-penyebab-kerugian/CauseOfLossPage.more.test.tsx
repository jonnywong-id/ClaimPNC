import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { CauseOfLossPage } from './CauseOfLossPage'

/** Uji tambahan Master Penyebab Kerugian: cabang galat muat/simpan dan pengurutan. Data KARANGAN. */

const ROUTES = '/api/master/penyebab-kerugian'

const SAMPLE = [
  { id: '1002', deskripsi: 'Contoh B', id_lama: '' },
  { id: '1001', deskripsi: 'Contoh C', id_lama: '' },
]

type Answer = Response | Promise<Response> | 'putus' | 'rusak' | undefined

let methods: string[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function installFetch(answer: (url: string, method: string) => Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    methods.push(method)
    const custom = answer(url, method)
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
    if (custom === 'rusak') {
      return Promise.resolve({
        get status(): number {
          throw new TypeError('rusak')
        },
      } as unknown as Response)
    }
    if (custom) return Promise.resolve(custom)
    if (method === 'GET') {
      return Promise.resolve(json(200, { penyebab_kerugian: SAMPLE, total: 2, portal: 'ASM' }))
    }
    return Promise.resolve(json(200, { penyebab_kerugian: SAMPLE[0], portal: 'ASM' }))
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <CauseOfLossPage />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  methods = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('daftar', () => {
  it('meminta portal dipilih lebih dulu', () => {
    useSelectedPortal.getState().clear()
    installFetch()
    show()

    expect(screen.getByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(screen.queryByRole('table')).not.toBeInTheDocument()
  })

  it.each([
    { name: 'jaringan putus', answer: (): Answer => 'putus', title: 'Tidak dapat menghubungi server' },
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
      title: 'Daftar penyebab kerugian gagal dimuat',
      description: 'Basis data sibuk.',
    },
    {
      name: 'galat bukan API',
      answer: (): Answer => 'rusak',
      title: 'Daftar penyebab kerugian gagal dimuat',
      description: 'Terjadi kesalahan pada sistem. Coba muat ulang.',
    },
  ])('menampilkan galat muat: $name', async ({ answer, title, description }) => {
    installFetch((_url, method) => (method === 'GET' ? answer() : undefined))
    show()

    expect(await screen.findByText(title)).toBeInTheDocument()
    if (description) expect(screen.getByText(description)).toBeInTheDocument()
  })

  it('mengurutkan menurut ID dan deskripsi lalu memuat ulang', async () => {
    installFetch()
    show()
    const table = await screen.findByRole('table')

    await userEvent.click(within(within(table).getByRole('columnheader', { name: /^ID/ })).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('1001')
    await userEvent.click(
      within(within(table).getByRole('columnheader', { name: /^Deskripsi Kerugian/ })).getByRole('button'),
    )
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('Contoh B')

    const before = methods.length
    await userEvent.click(screen.getByRole('button', { name: /Refresh/ }))
    await waitFor(() => expect(methods.length).toBe(before + 1))
  })
})

describe('simpan', () => {
  async function openEdit() {
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah penyebab kerugian Contoh C' }))
    return screen.findByRole('form', { name: 'Memperbaharui Data penyebab kerugian' })
  }

  it('menutup form setelah perubahan tersimpan, dan lewat Batal', async () => {
    installFetch()
    show()
    let form = await openEdit()
    expect(within(form).getByText('1001')).toBeInTheDocument()
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))
    await waitFor(() =>
      expect(screen.queryByRole('form', { name: /Memperbaharui Data/ })).not.toBeInTheDocument(),
    )
    expect(methods).toContain('PUT')

    form = await openEdit()
    await userEvent.click(within(form).getByRole('button', { name: 'Batal' }))
    expect(screen.queryByRole('form', { name: /Memperbaharui Data/ })).not.toBeInTheDocument()
  })

  it.each([
    {
      name: 'validasi tanpa rincian',
      answer: (): Answer => json(422, { kode: 'validasi_gagal', pesan: 'Deskripsi ditolak.' }),
      texts: ['Isian belum benar', 'Deskripsi ditolak.'],
    },
    {
      name: 'tidak ditemukan',
      answer: (): Answer => json(404, { kode: 'penyebab_kerugian_tidak_ditemukan', pesan: 'x' }),
      texts: ['Penyebab kerugian tidak ditemukan'],
    },
    {
      name: 'nomor bentrok',
      answer: (): Answer => json(409, { kode: 'id_penyebab_kerugian_sudah_dipakai', pesan: 'x' }),
      texts: ['Nomor bentrok'],
    },
    {
      name: 'portal tidak disebut',
      answer: (): Answer => json(400, { kode: 'portal_tidak_disebut', pesan: 'x' }),
      texts: ['Pilih portal entitas di bilah atas halaman, lalu simpan lagi.'],
    },
    {
      name: 'portal tidak dikenal',
      answer: (): Answer => json(400, { kode: 'portal_tidak_dikenal', pesan: 'x' }),
      texts: ['Pilih portal entitas di bilah atas halaman, lalu simpan lagi.'],
    },
    {
      name: 'portal belum siap',
      answer: (): Answer => json(503, { kode: 'portal_belum_siap', pesan: 'x' }),
      texts: ['Basis data entitas ini belum tersedia'],
    },
    {
      name: 'galat API lain',
      answer: (): Answer => json(500, { kode: 'galat_internal', pesan: 'Penyimpanan sibuk.' }),
      texts: ['Gagal menyimpan', 'Penyimpanan sibuk.'],
    },
    {
      name: 'jaringan putus',
      answer: (): Answer => 'putus',
      texts: ['Perubahan belum tersimpan. Periksa koneksi lalu coba lagi.'],
    },
    {
      name: 'galat bukan API',
      answer: (): Answer => 'rusak',
      texts: ['Terjadi kesalahan yang tidak terduga. Coba beberapa saat lagi.'],
    },
  ])('menampilkan galat simpan: $name', async ({ answer, texts }) => {
    installFetch((_url, method) => (method === 'PUT' ? answer() : undefined))
    show()
    const form = await openEdit()
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    for (const text of texts) expect(await within(form).findByText(text)).toBeInTheDocument()
  })

  it('menolak deskripsi yang terlalu panjang di layar', async () => {
    installFetch()
    show()
    const form = await openEdit()
    const input = within(form).getByLabelText('Deskripsi Kerugian')
    // maxLength dicabut supaya ketikan melewati batas, dan yang menolaknya skema form.
    input.removeAttribute('maxlength')
    await userEvent.clear(input)
    await userEvent.type(input, 'X'.repeat(101))
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(
      await within(form).findByText('Deskripsi Kerugian paling panjang 100 karakter.'),
    ).toBeInTheDocument()
    expect(methods).not.toContain('PUT')
  })

  it('menahan tombol selama menyimpan', async () => {
    let release: (r: Response) => void = () => {}
    installFetch((url, method) =>
      method === 'PUT' && url.startsWith(`${ROUTES}/`)
        ? new Promise<Response>((resolve) => {
            release = resolve
          })
        : undefined,
    )
    show()
    const form = await openEdit()
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByRole('button', { name: 'Menyimpan…' })).toBeDisabled()
    expect(within(form).getByRole('button', { name: 'Batal' })).toBeDisabled()
    release(json(200, { penyebab_kerugian: SAMPLE[1], portal: 'ASM' }))
    await waitFor(() =>
      expect(screen.queryByRole('form', { name: /Memperbaharui Data/ })).not.toBeInTheDocument(),
    )
  })
})
