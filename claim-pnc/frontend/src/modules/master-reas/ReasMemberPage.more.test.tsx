import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ReasMemberPage } from './ReasMemberPage'

/** Uji tambahan Master Reas: galat pemuatan, pencarian, dan Refresh. Nilai KARANGAN (`D-69`). */

const ROWS = [
  { kode_reas: 'RE-002', nama_reas: 'Reas Beta', login: 'beta', email: 'b@contoh.invalid', negara: 'Singapura', tipe: '2', cadangan: false },
  { kode_reas: 'RE-001', nama_reas: 'Reas Alfa', login: 'alfa', email: 'a@contoh.invalid', negara: 'Indonesia', tipe: '1', cadangan: true },
]

type Reply = { body?: unknown; status?: number; fail?: boolean }
let calls: string[] = []

function installFetch(map: () => Reply) {
  vi.stubGlobal('fetch', (url: string) => {
    calls.push(url)
    const reply = map()
    if (reply.fail) return Promise.reject(new TypeError('Failed to fetch'))
    return Promise.resolve(
      new Response(JSON.stringify(reply.body ?? {}), {
        status: reply.status ?? 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function show() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <ReasMemberPage />
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
  [{ status: 500, body: { kode: 'galat_internal', pesan: 'x' } }, 'Daftar member reas tidak dapat dimuat'],
])('menjelaskan galat pemuatan %#', async (reply, title) => {
  installFetch(() => reply)
  show()
  expect(await screen.findByText(title)).toBeInTheDocument()
  expect(screen.queryByRole('table')).not.toBeInTheDocument()
})

it('mencari ke seluruh kolom teks dan mengurutkan menurut negara', async () => {
  installFetch(() => ({ body: { member_reas: ROWS, portal: 'ASM' } }))
  const user = userEvent.setup()
  show()

  const table = await screen.findByRole('table')
  await user.click(within(table).getByRole('button', { name: 'Negara' }))
  expect(within(table).getAllByRole('row')[1]).toHaveTextContent('Indonesia')
  await user.click(within(table).getByRole('button', { name: 'Kode Reas' }))
  expect(within(table).getAllByRole('row')[1]).toHaveTextContent('RE-001')

  await user.type(screen.getByLabelText('Cari kode, nama, login, atau email'), 'singapura')
  expect(within(screen.getByRole('table')).getAllByRole('row')).toHaveLength(2)
  expect(screen.getByRole('table')).toHaveTextContent('Reas Beta')
})

it('memuat ulang daftar lewat Refresh', async () => {
  installFetch(() => ({ body: { member_reas: ROWS, portal: 'ASM' } }))
  const user = userEvent.setup()
  show()

  await screen.findByRole('table')
  const before = calls.length
  await user.click(screen.getByRole('button', { name: 'Refresh' }))
  await waitFor(() => expect(calls.length).toBe(before + 1))
})
