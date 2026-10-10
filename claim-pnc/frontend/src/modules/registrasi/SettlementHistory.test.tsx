import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { SettlementHistory } from './SettlementHistory'

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

let urls: string[] = []

function show(answer: () => Response) {
  vi.stubGlobal('fetch', (url: string) => {
    urls.push(url)
    return Promise.resolve(answer())
  })
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <SettlementHistory claimID="klaim-1" object={1} coverage={2} adjustment={3} />
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  urls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('SettlementHistory', () => {
  it('menggambar Status Penerimaan Komite dan Histori Transfer Kasir sesuai InputAdjustment_sect', async () => {
    show(() =>
      json(200, {
        komite: [
          { jenjang: 1, nama_komite: 'KOMITE01', status: '1', tanggal: '2026-10-07T14:49:00+07:00', komentar: 'ok' },
          { jenjang: 2, nama_komite: 'KOMITE02', status: '0', tanggal: '', komentar: '' },
        ],
        kasir: [{ pic_teknik: 'PICTEKNIK', tanggal: '2026-10-08T09:00:00+07:00', status_kasir: '2', komentar: 'Sedang Transfer Kasir' }],
      }),
    )

    const committee = await screen.findByRole('table', { name: 'Status Penerimaan Komite' })
    expect(within(committee).getAllByRole('columnheader').map((h) => h.textContent)).toEqual([
      'Nama Komite', 'Status', 'Tanggal Approve/Reject', 'Komentar',
    ])
    expect(within(committee).getByText('Setuju')).toBeInTheDocument()
    expect(within(committee).getByText('Menunggu')).toBeInTheDocument()

    const cashier = screen.getByRole('table', { name: 'Histori Transfer Kasir' })
    expect(within(cashier).getAllByRole('columnheader').map((h) => h.textContent)).toEqual([
      'PIC Teknik', 'Tanggal Transfer/Reject', 'Status Kasir', 'Komentar',
    ])
    expect(within(cashier).getByText('Sedang Transfer Kasir')).toBeInTheDocument()
    expect(urls[0]).toBe('/api/registrasi/klaim/klaim-1/adjustment/riwayat?objek=1&coverage=2&adjustment=3')
  })

  it('tidak menggambar apa pun bila kedua riwayat kosong', async () => {
    const { container } = show(() => json(200, { komite: [], kasir: [] }))
    await vi.waitFor(() => expect(urls).toHaveLength(1))
    expect(container).toBeEmptyDOMElement()
  })

  it('memberi tahu bila riwayat gagal dimuat', async () => {
    show(() => json(500, { error: { code: 'x', message: 'gagal' } }))
    expect(await screen.findByText('Riwayat komite dan transfer kasir tidak dapat dimuat.')).toBeInTheDocument()
  })
})
