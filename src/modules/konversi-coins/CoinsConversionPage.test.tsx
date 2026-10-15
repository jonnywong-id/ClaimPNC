import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSession } from '@/app/session'

import { CoinsConversionPage } from './CoinsConversionPage'

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

const info = {
  live: { alamat: 'u@live:1521/LIVE', lengkap: true, kurang: [] },
  test: { alamat: 'u@test:1521/TEST', lengkap: true, kurang: [] },
  polis: ['P-1'],
  maks_polis: 500,
}

let urls: string[] = []
let bodies: unknown[] = []

beforeEach(() => {
  urls = []
  bodies = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    urls.push(url)
    if (url.endsWith('/keterangan')) return Promise.resolve(json(info))
    bodies.push(init?.body ? JSON.parse(init.body as string) : undefined)
    return Promise.resolve(
      json({
        uji_coba: true, mulai: '', selesai: '', berhasil: 1, tidak_ditemukan: 0, gagal: 0, total_baris: 3,
        polis: [{ nomor_polis: 'P-1', status: 'berhasil', versi_prodke: ['0', '1'], versi_tanpa_coins: 1,
          baris_dihapus: 2, baris_ditulis: 3 }],
      }),
    )
  })
})

afterEach(() => vi.unstubAllGlobals())

describe('Konversi Coins', () => {
  it('mengisi daftar polis bawaan dan menjalankan uji coba tanpa menyimpan', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(
      <QueryClientProvider client={client}>
        <CoinsConversionPage />
      </QueryClientProvider>,
    )

    expect(await screen.findByText('u@live:1521/LIVE')).toBeInTheDocument()
    expect(screen.getByLabelText('Daftar Nomor Polis')).toHaveValue('P-1')
    await userEvent.click(screen.getByRole('button', { name: 'Uji Coba' }))

    expect(await screen.findByText('Hasil Uji Coba — tidak ada yang tersimpan')).toBeInTheDocument()
    expect(screen.getByText(/3 baris koasuransi/)).toBeInTheDocument()
    expect(screen.getByText(/1 versi tanpa koasuransi/)).toBeInTheDocument()
    expect(bodies[0]).toEqual({ polis: 'P-1', uji_coba: true })
    expect(urls.some((u) => u.endsWith('/api/konversi-coins/jalankan'))).toBe(true)
  })
})
