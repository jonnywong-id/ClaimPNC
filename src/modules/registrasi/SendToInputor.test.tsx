import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { AnalystNoteNotice, CHANNEL_SEND_TO_INPUTOR, SendToInputorDialog, latestAnalystNote } from './SendToInputor'
import type { Communication } from './types'

/** Uji modal "Kirim ke Inputor" dan Catatan dari Analyst. Seluruh data KARANGAN (`D-69`). */

type Answer = Response | 'putus' | undefined

let calls: { url: string; body: unknown }[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function installFetch(answer: (url: string) => Answer) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, body: init?.body ? JSON.parse(init.body as string) : undefined })
    const custom = answer(url)
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
    if (custom) return Promise.resolve(custom)
    return Promise.resolve(json(404, { kode: 'tidak_ditemukan', pesan: 'Tidak ditemukan.' }))
  })
}

function wrap(children: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(<QueryClientProvider client={client}>{children}</QueryClientProvider>)
}

function comm(extra: Partial<Communication>): Communication {
  return {
    kasus_id: 'K1', id: '1', tanggal: '2026-10-03T09:15:00+07:00', pengirim: 'Analis Contoh', pesan: 'Lengkapi KTP.',
    balasan: '', penjawab: '', tanggal_balasan: '', kanal: CHANNEL_SEND_TO_INPUTOR, ...extra,
  }
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

describe('SendToInputorDialog', () => {
  it('mengirim catatan lalu menutup modal', async () => {
    installFetch((url) => (url === '/api/registrasi/tugas/tugas-1/kirim-inputor' ? json(200, { klaim: { id: 'klaim-1' }, tugas: null }) : undefined))
    const onClose = vi.fn()
    wrap(<SendToInputorDialog claimID="klaim-1" taskID="tugas-1" onClose={onClose} />)

    expect(screen.getByRole('heading', { name: 'Kirim ke Inputor' })).toBeInTheDocument()
    await userEvent.type(screen.getByLabelText('Note'), 'Lengkapi KTP.')
    await userEvent.click(screen.getByRole('button', { name: 'Kirim' }))

    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(calls.find((c) => c.url.endsWith('/kirim-inputor'))?.body).toEqual({ catatan: 'Lengkapi KTP.' })
  })

  it('menampilkan galat dan tetap terbuka bila server menolak', async () => {
    installFetch((url) =>
      url.endsWith('/kirim-inputor') ? json(409, { kode: 'tugas_sudah_selesai', pesan: 'Tugas sudah selesai.' }) : undefined,
    )
    const onClose = vi.fn()
    wrap(<SendToInputorDialog claimID="klaim-1" taskID="tugas-1" onClose={onClose} />)

    await userEvent.click(screen.getByRole('button', { name: 'Kirim' }))
    expect(await screen.findByText('Klaim belum terkirim ke Inputor')).toBeInTheDocument()
    expect(screen.getByText('Tugas sudah selesai.')).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('menahan Kirim untuk catatan lebih dari 4000 karakter, dan menutup lewat Escape', () => {
    installFetch(() => undefined)
    const onClose = vi.fn()
    wrap(<SendToInputorDialog claimID="klaim-1" taskID="tugas-1" onClose={onClose} />)

    fireEvent.change(screen.getByLabelText('Note'), { target: { value: 'a'.repeat(4001) } })
    expect(screen.getByRole('button', { name: 'Kirim' })).toBeDisabled()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })
})

describe('Catatan dari Analyst', () => {
  it('memilih catatan SENDTOINPUTOR terbaru yang berisi', () => {
    expect(
      latestAnalystNote([
        comm({ id: '3', kanal: '1001', pesan: 'pesan cabang' }),
        comm({ id: '2', pesan: '  ' }),
        comm({ id: '1', pesan: 'Lengkapi KTP.' }),
      ])?.id,
    ).toBe('1')
    expect(latestAnalystNote([comm({ kanal: '' })])).toBeUndefined()
  })

  it('menampilkan catatan dari tab Progress pada layar Input Register', async () => {
    installFetch((url) =>
      url === '/api/registrasi/klaim/klaim-1/progres' ? json(200, { progres: [], komunikasi: [comm({})] }) : undefined,
    )
    wrap(<AnalystNoteNotice claimID="klaim-1" />)

    const note = await screen.findByRole('note', { name: 'Catatan dari Analyst' })
    expect(note).toHaveTextContent('Lengkapi KTP.')
    expect(note).toHaveTextContent('Analis Contoh')
  })

  it('tidak menampilkan apa pun tanpa catatan', async () => {
    installFetch((url) => (url === '/api/registrasi/klaim/klaim-1/progres' ? json(200, { progres: [], komunikasi: [] }) : undefined))
    wrap(<AnalystNoteNotice claimID="klaim-1" />)
    await waitFor(() => expect(calls.some((c) => c.url.endsWith('/progres'))).toBe(true))
    expect(screen.queryByRole('note')).not.toBeInTheDocument()
  })
})
