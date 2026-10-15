import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ClauseAIPage } from './ClauseAIPage'

/** Uji tambahan Master Pasal AI: galat pemuatan dan kalimat kosong. Nilai KARANGAN. */

type Reply = { body?: unknown; status?: number; fail?: boolean }

let calls: string[] = []

function installFetch(map: (url: string) => Reply) {
  vi.stubGlobal('fetch', (url: string) => {
    calls.push(url)
    const reply = map(url)
    if (reply.fail) return Promise.reject(new TypeError('Failed to fetch'))
    return Promise.resolve(
      new Response(JSON.stringify(reply.body ?? {}), {
        status: reply.status ?? 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function empty() {
  return {
    pasal_ai: [],
    paginasi: { halaman: 1, ukuran_halaman: 25, jumlah_halaman: 1, jumlah_baris: 0 },
    portal: 'ASM',
  }
}

function show() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <ClauseAIPage />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: null,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
})

it.each([
  [{ fail: true }, 'Server Claim PNC tidak dapat dihubungi'],
  [{ status: 400, body: { kode: 'portal_tidak_disebut', pesan: 'x' } }, 'Portal entitas belum dipilih'],
  [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
  [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
  [{ status: 500, body: { kode: 'galat_internal', pesan: 'Tabel tidak terbaca.' } }, 'Daftar Pasal AI tidak dapat dimuat'],
])('menjelaskan galat pemuatan %#', async (reply, title) => {
  installFetch(() => reply)
  show()

  expect(await screen.findByText(title)).toBeInTheDocument()
  expect(screen.queryByRole('table')).not.toBeInTheDocument()
})

it('menyebut pesan server pada galat yang tidak dikenali', async () => {
  installFetch(() => ({ status: 500, body: { kode: 'galat_internal', pesan: 'Tabel tidak terbaca.' } }))
  show()

  expect(await screen.findByText('Tabel tidak terbaca.')).toBeInTheDocument()
})

it('menyebut kalimat kosong sesuai ada tidaknya kata kunci, tanpa paginator', async () => {
  installFetch(() => ({ body: empty() }))
  const user = userEvent.setup()
  show()

  expect(await screen.findByText('Belum ada Pasal AI pada entitas ini.')).toBeInTheDocument()
  expect(screen.queryByRole('navigation', { name: 'Halaman tabel' })).not.toBeInTheDocument()

  await user.type(screen.getByLabelText('Cari'), '  banjir  ')
  await user.keyboard('{Enter}')

  expect(await screen.findByText('Tidak ada Pasal AI yang cocok dengan "banjir".')).toBeInTheDocument()
  await waitFor(() => expect(calls.at(-1)).toContain('cari=banjir'))
})

it('menyebut portal dari jawaban server, lalu tanda pisah tanpa portal', async () => {
  installFetch(() => ({ body: { ...empty(), portal: 'SMI' } }))
  const view = show()

  expect(await screen.findByText('SMI')).toBeInTheDocument()
  view.unmount()

  useSelectedPortal.getState().clear()
  show()
  expect(screen.getByText('—')).toBeInTheDocument()
})
