import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { CommitteeTransferDialog } from './CommitteeTransferDialog'
import { showTransferToAnalyst } from './TransferToAnalyst'
import type { Claim, InsuredItem } from './types'

/** Uji tombol dan modal "Transfer ke Analyst". Seluruh data KARANGAN (`D-69`). */

let calls: { url: string; body: unknown }[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function installFetch(answer: (url: string) => Response | undefined) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, body: init?.body ? JSON.parse(init.body as string) : undefined })
    return Promise.resolve(answer(url) ?? json(404, { kode: 'tidak_ditemukan', pesan: 'Tidak ditemukan.' }))
  })
}

function wrap(children: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(<QueryClientProvider client={client}>{children}</QueryClientProvider>)
}

function item(ids: string[]): InsuredItem {
  return {
    id: '1',
    nama: 'Peserta Contoh',
    lokasi: '',
    coverage: ids.map((id) => ({ id, nama: 'Jaminan ' + id, penyebab_kerugian: '', tsi_sen: 0, spreading: [] })),
  } as unknown as InsuredItem
}

function claim(extra: Partial<Claim> = {}): Claim {
  return { id: 'klaim-1', polis: { lini: '002' }, ...extra } as unknown as Claim
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

describe('showTransferToAnalyst', () => {
  it('tampil pada setiap jaminan objek PA di tahap Estimation PA', () => {
    const o = item(['10001', '10002'])
    expect(showTransferToAnalyst(claim(), 'estimasi-pa', o, 1)).toBe(true)
    expect(showTransferToAnalyst(claim(), 'estimasi-pa', o, 0)).toBe(true)
    expect(showTransferToAnalyst(claim(), 'kirim-analis', o, 1)).toBe(false)
    expect(showTransferToAnalyst(claim({ polis: { lini: '006' } } as Partial<Claim>), 'estimasi-pa', o, 1)).toBe(false)
  })

  it('tidak tampil bila jaminan sudah ditandai atau jaminan PHK', () => {
    const marked = item(['10001'])
    marked.coverage[0] = { ...marked.coverage[0]!, sudah_transfer_analis: true }
    expect(showTransferToAnalyst(claim(), 'estimasi-pa', marked, 0)).toBe(false)
    expect(showTransferToAnalyst(claim(), 'estimasi-pa', item(['10010']), 0)).toBe(false)
  })
})

describe('CommitteeTransferDialog dari Transfer ke Analyst', () => {
  const paClaim = (): Claim =>
    ({
      id: 'klaim-1',
      polis: { lini: '002', jenis_bisnis: 'PersonalAccident' },
      objek: [item(['10001'])],
    }) as unknown as Claim

  function open(onClose: () => void) {
    wrap(
      <CommitteeTransferDialog
        klaim={paClaim()}
        taskID="tugas-1"
        stage="estimasi-pa"
        object={1}
        coverage={1}
        receivers={[]}
        onClose={onClose}
      />,
    )
  }

  it('menampilkan isian PA, menyimpannya, lalu Kirim Analyst dan menutup modal', async () => {
    installFetch((url) =>
      url === '/api/registrasi/tugas/tugas-1/transfer-analis' || url.endsWith('/jaminan/isian-komite')
        ? json(200, { klaim: { id: 'klaim-1', status_klaim: '1151' }, tugas: null })
        : undefined,
    )
    const onClose = vi.fn()
    open(onClose)

    expect(screen.getByRole('heading', { name: 'Transfer Claim ke Komite' })).toBeInTheDocument()
    expect(screen.getByLabelText('Remaks / Investigation')).toBeInTheDocument()
    expect(screen.getByLabelText('Diagnose/History of Illness')).toBeInTheDocument()
    expect(screen.getByLabelText('Remaks / Analysis')).toBeInTheDocument()
    expect(screen.queryByLabelText('Polis Liability')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Kirim Komite' })).not.toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Remaks / Analysis'), 'Layak')
    await userEvent.click(screen.getByRole('button', { name: 'Kirim Analyst' }))

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(calls.map((c) => c.url)).toEqual([
      '/api/registrasi/klaim/klaim-1/jaminan/isian-komite',
      '/api/registrasi/tugas/tugas-1/transfer-analis',
    ])
    expect(calls[0]?.body).toMatchObject({ tugas_id: 'tugas-1', objek: 1, jaminan: 1, remarks: 'Layak' })
    expect(calls[1]?.body).toEqual({ objek_id: '1', coverage_id: '10001' })
  })

  it('tetap terbuka bila jaminan hanya ditandai (StatusClaim bukan 1151)', async () => {
    installFetch((url) =>
      url.endsWith('/transfer-analis') || url.endsWith('/jaminan/isian-komite')
        ? json(200, { klaim: { id: 'klaim-1', status_klaim: '1147' }, tugas: null })
        : undefined,
    )
    const onClose = vi.fn()
    open(onClose)
    await userEvent.click(screen.getByRole('button', { name: 'Kirim Analyst' }))
    expect(await screen.findByText('Jaminan ini ditandai Transfer ke Analyst.')).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('menampilkan galat dan tetap terbuka bila server menolak', async () => {
    installFetch((url) =>
      url.endsWith('/jaminan/isian-komite')
        ? json(200, { klaim: { id: 'klaim-1' } })
        : url.endsWith('/transfer-analis')
          ? json(409, { kode: 'tugas_sudah_selesai', pesan: 'Tugas sudah selesai.' })
          : undefined,
    )
    const onClose = vi.fn()
    open(onClose)

    await userEvent.click(screen.getByRole('button', { name: 'Kirim Analyst' }))
    expect(await screen.findByText('Tugas sudah selesai.')).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })
})
