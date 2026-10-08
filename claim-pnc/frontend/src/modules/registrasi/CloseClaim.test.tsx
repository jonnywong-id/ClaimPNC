import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { CloseClaimDialog } from './CloseClaim'

/** Uji dialog "Prevent Close Claim" (tombol Tutup Klaim). Seluruh data KARANGAN (`D-69`). */

let calls: { url: string; body: unknown }[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function installFetch(answer: (url: string) => Response | undefined) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, body: init?.body ? JSON.parse(init.body as string) : undefined })
    return Promise.resolve(answer(url) ?? json(404, { kode: 'tidak_ditemukan', pesan: 'x' }))
  })
}

function wrap(children: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(<QueryClientProvider client={client}>{children}</QueryClientProvider>)
}

const URL = '/api/registrasi/tugas/tugas-1/tutup-klaim'

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('CloseClaimDialog', () => {
  it('mengirim isian lalu menutup dialog saat Ya', async () => {
    installFetch((url) => (url === URL ? json(200, { klaim: { id: 'klaim-1' } }) : undefined))
    const onClose = vi.fn()
    wrap(<CloseClaimDialog claimID="klaim-1" taskID="tugas-1" onClose={onClose} />)

    expect(screen.getByText('Apakah anda yakin ingin menutup klaim ini?')).toBeInTheDocument()
    // Tanpa daftar pilihan dan kolom: tampil tetapi nonaktif.
    expect(screen.getByLabelText('Survey Kepuasan')).toBeDisabled()
    expect(screen.getByLabelText('No Reff Broker')).toBeDisabled()

    await userEvent.type(screen.getByLabelText('Catatan *'), 'Klaim selesai.')
    await userEvent.click(screen.getByLabelText('Tutup Sementara'))
    await userEvent.type(screen.getByLabelText('Usulan'), 'U')
    await userEvent.click(screen.getByRole('button', { name: 'Ya' }))

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(calls.find((c) => c.url === URL)?.body).toEqual({
      catatan_tutup: 'Klaim selesai.', usulan: 'U', effort_tutup: '', kendala_tutup: '', tutup_sementara: true,
    })
  })

  it('menampilkan pesan penolakan di isian Catatan dan tetap terbuka', async () => {
    installFetch((url) =>
      url === URL
        ? json(422, { kode: 'validasi_gagal', pesan: 'x', detail: [{ kode: 'tutup_klaim_ditolak', field: 'catatan_tutup', pesan: 'Error Close Harus Di PIC' }] })
        : undefined,
    )
    const onClose = vi.fn()
    wrap(<CloseClaimDialog claimID="klaim-1" taskID="tugas-1" onClose={onClose} />)

    await userEvent.type(screen.getByLabelText('Catatan *'), 'n')
    await userEvent.click(screen.getByRole('button', { name: 'Ya' }))
    expect(await screen.findByText('Error Close Harus Di PIC')).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('Tidak menutup dialog tanpa mengirim apa pun', async () => {
    installFetch(() => undefined)
    const onClose = vi.fn()
    wrap(<CloseClaimDialog claimID="klaim-1" taskID="tugas-1" onClose={onClose} />)
    await userEvent.click(screen.getByRole('button', { name: 'Tidak' }))
    expect(onClose).toHaveBeenCalledTimes(1)
    expect(calls).toHaveLength(0)
  })
})
