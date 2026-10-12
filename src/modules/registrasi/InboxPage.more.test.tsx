import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { InboxPage } from './InboxPage'

/** Uji tambahan Inbox Registrasi: keadaan memuat, galat, dan baris tanpa nomor. Data KARANGAN. */

type Answer = Response | Promise<Response> | 'putus' | 'aneh' | undefined

let calls: { url: string; method: string }[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function installFetch(answer: (url: string, method: string) => Answer) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    calls.push({ url, method })
    const custom = answer(url, method)
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
    // Jawaban yang melempar nilai bukan Error saat statusnya dibaca.
    if (custom === 'aneh') {
      return Promise.resolve({
        get status(): number {
          throw 'bukan galat'
        },
      } as unknown as Response)
    }
    if (custom) return Promise.resolve(custom)
    if (url === '/api/registrasi/komite') return Promise.resolve(json(200, { komite: [] }))
    return Promise.resolve(json(200, { tugas: [] }))
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <InboxPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const TASK = {
  id: 'tugas-9',
  klaim_id: 'klaim-9',
  nomor_klaim: '',
  tahap: 'view-polis',
  nama_tahap: '',
  antrean: 'WORKBASKET',
  workbasket: 'AdminPNC',
  pemilik: '',
  dapat_diambil: true,
  tindakan_keluar: '',
  dibuat_pada: '2026-06-10T03:00:00Z',
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('InboxPage', () => {
  it('menyebut pemuatan lalu baris tanpa nomor dan tanpa nama tahap', async () => {
    installFetch((url) => (url === '/api/registrasi/inbox' ? json(200, { tugas: [TASK] }) : undefined))
    show()

    expect(screen.getByText('Memuat pekerjaan…')).toBeInTheDocument()
    expect(await screen.findByText('belum bernomor')).toBeInTheDocument()
    expect(screen.getByText('view-polis')).toBeInTheDocument()
    expect(screen.getByText('Workbasket · AdminPNC')).toBeInTheDocument()
  })

  it('menyebut tugas yang sedang diambil', async () => {
    installFetch((url, method) => {
      if (url === '/api/registrasi/inbox') return json(200, { tugas: [TASK] })
      if (method === 'POST') return new Promise<Response>(() => {})
      return undefined
    })
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Ambil' }))
    expect(await screen.findByRole('button', { name: 'Mengambil…' })).toBeDisabled()
  })

  it.each([
    { name: 'galat API', answer: (): Answer => json(500, { kode: 'x', pesan: 'Inbox rusak.' }), text: 'Inbox rusak.' },
    { name: 'jaringan putus', answer: (): Answer => 'putus', text: 'Tidak dapat menghubungi server Claim PNC.' },
    { name: 'galat bukan Error', answer: (): Answer => 'aneh', text: 'Terjadi kesalahan pada sistem.' },
  ])('menampilkan galat inbox: $name', async ({ answer, text }) => {
    installFetch((url) => (url === '/api/registrasi/inbox' ? answer() : undefined))
    show()

    expect(await screen.findByText('Daftar pekerjaan tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
  })

  it('tidak membuka klaim untuk nomor polis yang hanya spasi, dan menampilkan galatnya', async () => {
    installFetch((url, method) =>
      url === '/api/registrasi/klaim' && method === 'POST'
        ? json(404, { kode: 'polis_tidak_ditemukan', pesan: 'Polis tidak ditemukan.' })
        : undefined,
    )
    show()

    await userEvent.type(screen.getByLabelText('Nomor polis'), '   ')
    expect(screen.getByRole('button', { name: 'Buka klaim' })).toBeDisabled()
    await userEvent.type(screen.getByLabelText('Nomor polis'), '{Enter}')
    expect(calls.some((c) => c.method === 'POST')).toBe(false)

    await userEvent.type(screen.getByLabelText('Nomor polis'), 'POL-X')
    await userEvent.click(screen.getByRole('button', { name: 'Buka klaim' }))
    expect(await screen.findByText('Klaim tidak dapat dibuka')).toBeInTheDocument()
    expect(screen.getByText('Polis tidak ditemukan.')).toBeInTheDocument()
    await waitFor(() => expect(calls.filter((c) => c.method === 'POST')).toHaveLength(1))
  })
})
