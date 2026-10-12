import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSession } from '@/app/session'

import { ObjectItemConversionPage } from './ObjectItemConversionPage'

function json(body: unknown, status = 200): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

const info = {
  live: { alamat: 'u@live:1521/LIVE', lengkap: true, kurang: [] },
  test: { alamat: 'u@test:1521/TEST', lengkap: true, kurang: [] },
  polis: ['P-1'],
  maks_polis: 500,
}

let bodies: unknown[] = []

beforeEach(() => {
  bodies = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    if (url.endsWith('/keterangan')) return Promise.resolve(json(info))
    bodies.push(init?.body ? JSON.parse(init.body as string) : undefined)
    return Promise.resolve(
      json({
        uji_coba: true, mulai: '', selesai: '', berhasil: 1, tidak_ditemukan: 0, gagal: 0, total_item: 2,
        polis: [{ nomor_polis: 'P-1', status: 'berhasil', versi_prodke: ['0'], jumlah_objek: 1,
          objek_diperbarui: 1, item_dihapus: 0, item_ditulis: 2, dokumen_kosong: 0 }],
      }),
    )
  })
})

afterEach(() => vi.unstubAllGlobals())

describe('Konversi Object Item Fire', () => {
  it('mengisi daftar polis bawaan dan menjalankan uji coba tanpa menyimpan', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(
      <QueryClientProvider client={client}>
        <ObjectItemConversionPage />
      </QueryClientProvider>,
    )

    expect(await screen.findByText('u@live:1521/LIVE')).toBeInTheDocument()
    expect(screen.getByLabelText('Daftar Nomor Polis')).toHaveValue('P-1')
    await userEvent.click(screen.getByRole('button', { name: 'Uji Coba' }))

    expect(await screen.findByText('Hasil Uji Coba — tidak ada yang tersimpan')).toBeInTheDocument()
    expect(screen.getByText(/2 baris item/)).toBeInTheDocument()
    expect(bodies[0]).toEqual({ polis: 'P-1', uji_coba: true, prodke: '', indexobject: '' })
  })

  it('mengirim filter PRODKE dan Index Object, dan menolak isian bukan angka', async () => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    render(
      <QueryClientProvider client={client}>
        <ObjectItemConversionPage />
      </QueryClientProvider>,
    )

    expect(await screen.findByText('u@live:1521/LIVE')).toBeInTheDocument()
    await userEvent.type(screen.getByLabelText('PRODKE'), '1a')
    expect(screen.getByText('Isi dengan angka, atau kosongkan.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Uji Coba' })).toBeDisabled()

    await userEvent.clear(screen.getByLabelText('PRODKE'))
    await userEvent.type(screen.getByLabelText('PRODKE'), '39749851')
    await userEvent.type(screen.getByLabelText('Index Object'), '77')
    await userEvent.click(screen.getByRole('button', { name: 'Uji Coba' }))

    expect(await screen.findByText('Hasil Uji Coba — tidak ada yang tersimpan')).toBeInTheDocument()
    expect(bodies.at(-1)).toEqual({ polis: 'P-1', uji_coba: true, prodke: '39749851', indexobject: '77' })
  })
})
