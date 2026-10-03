import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { CauseOfLossPage } from './CauseOfLossPage'

/** Uji tambahan Master COL Simas Online: cabang galat muat/simpan dan pengurutan. Data KARANGAN. */

const ROUTE = '/api/master/col-simas-online'

const LIST = {
  portal: 'ASM',
  cause_of_loss: [
    { id: '1002', nama: 'BANJIR', bisnis: [] },
    { id: '1001', nama: 'KEBAKARAN', bisnis: [] },
  ],
}

const DETAIL = {
  portal: 'ASM',
  cause_of_loss: { id: '1001', nama: 'KEBAKARAN', bisnis: [{ id: '006', nama: 'FIRE' }] },
}

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
    if (url.startsWith('/api/master/bisnis')) {
      return Promise.resolve(json(200, { portal: 'ASM', bisnis: [{ id: '006', nama: 'FIRE' }] }))
    }
    if (call.method === 'GET' && url.startsWith(`${ROUTE}/`)) return Promise.resolve(json(200, DETAIL))
    if (call.method === 'GET') return Promise.resolve(json(200, LIST))
    return Promise.resolve(json(200, DETAIL))
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

const isList = (call: Call) => call.method === 'GET' && call.url === ROUTE

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
      title: 'Daftar tidak dapat dimuat',
      description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
    },
    {
      name: 'galat bukan API',
      answer: (): Answer => 'rusak',
      title: 'Daftar tidak dapat dimuat',
      description: 'Coba beberapa saat lagi.',
    },
  ])('menampilkan galat muat: $name', async ({ answer, title, description }) => {
    installFetch((call) => (isList(call) ? answer() : undefined))
    show()

    expect(screen.getByText('Memuat daftar cause of loss…')).toBeInTheDocument()
    expect(await screen.findByText(title)).toBeInTheDocument()
    if (description) expect(screen.getByText(description)).toBeInTheDocument()
  })

  it('menampilkan pesan kosong', async () => {
    installFetch((call) => (isList(call) ? json(200, { portal: 'ASM', cause_of_loss: [] }) : undefined))
    show()

    expect(await screen.findByText('Belum ada cause of loss pada entitas ini.')).toBeInTheDocument()
  })

  it('mengurutkan menurut ID dan Description lalu memuat ulang', async () => {
    installFetch()
    show()
    const table = await screen.findByRole('table')

    const id = within(table).getByRole('columnheader', { name: /^ID/ })
    await userEvent.click(within(id).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('1001')

    const name = within(table).getByRole('columnheader', { name: /^Description/ })
    await userEvent.click(within(name).getByRole('button'))
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('BANJIR')

    const before = calls.filter(isList).length
    await userEvent.click(screen.getByRole('button', { name: 'Refresh' }))
    await waitFor(() => expect(calls.filter(isList).length).toBe(before + 1))
  })
})

describe('simpan', () => {
  async function openEdit() {
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah KEBAKARAN' }))
    const form = await screen.findByRole('form', { name: 'Memperbaharui Data Simas Online' })
    await waitFor(() => expect(within(form).getByLabelText('Bisnis baris 1')).toHaveValue('FIRE'))
    return form
  }

  it('menyebut pemetaan bisnis yang masih dimuat dan menahan tombol Ubah', async () => {
    let release: (r: Response) => void = () => {}
    installFetch((call) =>
      call.method === 'GET' && call.url === `${ROUTE}/1001`
        ? new Promise<Response>((resolve) => {
            release = resolve
          })
        : undefined,
    )
    show()
    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Ubah KEBAKARAN' }))

    expect(await screen.findByText('Memuat bisnis yang sudah dipilih…')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Ubah' })).toBeDisabled()
    release(json(200, DETAIL))
    expect(await screen.findByDisplayValue('FIRE')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Ubah' })).toBeEnabled()
  })

  it('menyorot pelanggaran grid bisnis di bawah gridnya', async () => {
    installFetch((call) =>
      call.method === 'PUT'
        ? json(422, {
            kode: 'validasi_gagal',
            pesan: 'x',
            detail: [{ kolom: 'bisnis', pesan: 'Nama bisnis tidak boleh kembar.' }],
          })
        : undefined,
    )
    show()
    const form = await openEdit()
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    expect(await within(form).findByRole('alert')).toHaveTextContent('Nama bisnis tidak boleh kembar.')
    expect(within(form).queryByText('Belum dapat disimpan')).not.toBeInTheDocument()
  })

  it.each([
    {
      name: 'baris sudah tidak ada',
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
    installFetch((call) => (call.method === 'PUT' ? answer() : undefined))
    show()
    const form = await openEdit()
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    expect(await within(form).findByText(title)).toBeInTheDocument()
  })

  it('tidak menampilkan kotak pesan untuk galat bukan API', async () => {
    installFetch((call) => (call.method === 'PUT' ? 'rusak' : undefined))
    show()
    const form = await openEdit()
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    await waitFor(() => expect(within(form).getByRole('button', { name: 'Ubah' })).toBeEnabled())
    expect(calls.some((c) => c.method === 'PUT')).toBe(true)
    expect(within(form).queryByText('Terjadi kesalahan pada sistem')).not.toBeInTheDocument()
  })

  it('menolak nama bisnis yang terlalu panjang di layar', async () => {
    installFetch()
    show()
    const form = await openEdit()

    const input = within(form).getByLabelText('Bisnis baris 1')
    await userEvent.clear(input)
    // maxLength dicabut supaya ketikan melewati batas, dan yang menolaknya skema form.
    input.removeAttribute('maxlength')
    await userEvent.type(input, 'X'.repeat(101))
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    expect(await within(form).findByText('Nama bisnis paling panjang 100 karakter.')).toBeInTheDocument()
    expect(calls.some((c) => c.method === 'PUT')).toBe(false)
  })
})
