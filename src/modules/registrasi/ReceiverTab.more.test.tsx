import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'
import { formatDate } from '@/components/format'

import { ReceiverTab } from './ReceiverTab'
import type { Claim, Receiver, Task } from './types'

/** Uji tab Penerima Klaim (InputReceiver), dirender langsung. Seluruh data KARANGAN (`D-69`). */

type Answer = Response | Promise<Response> | 'putus' | undefined

let calls: { url: string; method: string; body: unknown }[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function installFetch(answer: (url: string, method: string) => Answer) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    calls.push({ url, method, body: init?.body ? JSON.parse(init.body as string) : undefined })
    const custom = answer(url, method)
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
    if (custom) return Promise.resolve(custom)
    return Promise.resolve(json(404, { kode: 'rekening_tidak_ditemukan', pesan: 'Rekening tidak ditemukan.' }))
  })
}

const TASK = { id: 'tugas-1' } as unknown as Task

function claimWith(receivers: Receiver[] | undefined): Claim {
  return { id: 'klaim-1', ...(receivers ? { penerima_klaim: receivers } : {}) } as unknown as Claim
}

function wrap(children: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(<QueryClientProvider client={client}>{children}</QueryClientProvider>)
}

const ACCOUNT = {
  nomor_rekening: '123',
  nama: 'PT Penerima',
  nama_bank: 'BANK CONTOH',
  nama_cabang_bank: 'CABANG A',
  alamat: 'Jl. Contoh',
  id_bank: '014',
  email: 'master@contoh.example',
  telepon: '0800',
  tanggal_approve_kasir: '2026-05-01',
  tanggal_approve_komite: '',
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

describe('ReceiverTab', () => {
  it('menyebut tabel kosong bila klaim belum punya penerima', () => {
    installFetch(() => undefined)
    wrap(<ReceiverTab klaim={claimWith(undefined)} tugas={TASK} lockedReason={null} />)

    expect(screen.getByText('Data Tidak Ada')).toBeInTheDocument()
  })

  it('menambah penerima dari master rekening lalu menutup panel setelah tersimpan', async () => {
    installFetch((url, method) => {
      if (url === '/api/registrasi/rekening/123') return json(200, ACCOUNT)
      if (url === '/api/registrasi/klaim/klaim-1/penerima' && method === 'POST') return json(200, {})
      return undefined
    })
    wrap(<ReceiverTab klaim={claimWith([])} tugas={TASK} lockedReason={null} />)

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const pane = screen.getByRole('group', { name: 'InputReceiver' })
    expect(screen.getByRole('button', { name: 'Tambah' })).toBeDisabled()
    expect(screen.queryByText('Data Tidak Ada')).not.toBeInTheDocument()

    await userEvent.type(within(pane).getByLabelText('No Rekening'), ' 123 ')
    await userEvent.tab()

    expect(await within(pane).findByText('PT Penerima')).toBeInTheDocument()
    expect(within(pane).getByText('CABANG A')).toBeInTheDocument()
    expect(within(pane).getByText(formatDate('2026-05-01'))).toBeInTheDocument()
    await waitFor(() => expect(within(pane).getByLabelText(/Email/)).toHaveValue('master@contoh.example'))
    expect(within(pane).getByLabelText('Telepon')).toHaveValue('0800')

    // Isian yang sudah diubah pengguna tidak ditimpa nilai master.
    await userEvent.clear(within(pane).getByLabelText('Telepon'))
    await userEvent.type(within(pane).getByLabelText('Telepon'), '0811')
    await userEvent.click(within(pane).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(screen.queryByRole('group', { name: 'InputReceiver' })).not.toBeInTheDocument())
    expect(calls.find((c) => c.method === 'POST')?.body).toEqual({
      tugas_id: 'tugas-1',
      id: '',
      nomor_rekening: '123',
      email: 'master@contoh.example',
      telepon: '0811',
    })
  })

  it('menampilkan galat master rekening, pelanggaran isian, dan galat simpan lain', async () => {
    let saveAnswer: () => Answer = () =>
      json(422, {
        kode: 'validasi_gagal',
        pesan: 'x',
        detail: [
          { kode: 'a', field: 'nomor_rekening', pesan: 'Rekening belum disetujui komite.' },
          { kode: 'b', field: 'email', pesan: 'Email wajib diisi.' },
        ],
      })
    installFetch((url, method) => {
      if (url === '/api/registrasi/rekening/999') return json(404, { kode: 'x', pesan: 'Rekening 999 tidak ada.' })
      if (url === '/api/registrasi/rekening/888') return 'putus'
      if (method === 'POST') return saveAnswer()
      return undefined
    })
    wrap(<ReceiverTab klaim={claimWith([])} tugas={TASK} lockedReason={null} />)

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const pane = screen.getByRole('group', { name: 'InputReceiver' })
    const number = within(pane).getByLabelText('No Rekening')

    await userEvent.type(number, '999')
    await userEvent.tab()
    expect(await within(pane).findByText('Rekening 999 tidak ada.')).toBeInTheDocument()
    expect(number).toHaveAttribute('aria-invalid', 'true')

    await userEvent.clear(number)
    await userEvent.type(number, '888')
    await userEvent.tab()
    expect(await within(pane).findByText('Master Rekening cannot be read.')).toBeInTheDocument()

    await userEvent.click(within(pane).getByRole('button', { name: 'Simpan' }))
    expect(await within(pane).findByText('Rekening belum disetujui komite.')).toBeInTheDocument()
    expect(within(pane).getByText('Email wajib diisi.')).toBeInTheDocument()
    expect(within(pane).queryByText('Penerima belum dapat disimpan')).not.toBeInTheDocument()

    saveAnswer = () => json(500, { kode: 'x', pesan: 'Kasir sibuk.' })
    await userEvent.click(within(pane).getByRole('button', { name: 'Simpan' }))
    expect(await within(pane).findByText('Penerima belum dapat disimpan')).toBeInTheDocument()
    expect(within(pane).getByText('Kasir sibuk.')).toBeInTheDocument()

    await userEvent.click(within(pane).getByRole('button', { name: 'Batal' }))
    expect(screen.queryByRole('group', { name: 'InputReceiver' })).not.toBeInTheDocument()
  })

  it('membuka penerima tersimpan, menampilkan isiannya, dan menutupnya lagi', async () => {
    installFetch((url) =>
      url === '/api/registrasi/rekening/555' ? new Promise<Response>(() => {}) : undefined,
    )
    wrap(
      <ReceiverTab
        klaim={claimWith([
          { id: 'R1', nama: 'PT Lama', alamat: '', nama_bank: 'BANK LAMA', nomor_rekening: '555' },
          { id: '', nama: '', alamat: 'Alamat Dua', nama_bank: '', nomor_rekening: '' },
        ])}
        tugas={TASK}
        lockedReason={null}
      />,
    )

    const toggle = screen.getByRole('button', { name: 'PT Lama' })
    expect(toggle).toHaveAttribute('aria-expanded', 'false')
    expect(screen.getByText('Alamat Dua')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '—' })).toBeInTheDocument()

    await userEvent.click(toggle)
    const pane = screen.getByRole('group', { name: 'InputReceiver' })
    expect(toggle).toHaveAttribute('aria-expanded', 'true')
    expect(within(pane).getByLabelText('No Rekening')).toHaveValue('555')
    expect(within(pane).getByText('BANK LAMA')).toBeInTheDocument()
    expect(within(pane).queryByRole('button', { name: 'Batal' })).not.toBeInTheDocument()

    await userEvent.click(toggle)
    expect(screen.queryByRole('group', { name: 'InputReceiver' })).not.toBeInTheDocument()
  })

  it('mengunci tambah dan isian saat tahap terkunci', async () => {
    installFetch(() => undefined)
    wrap(
      <ReceiverTab
        klaim={claimWith([{ id: 'R1', nama: 'PT Lama', alamat: 'A', nama_bank: 'B', nomor_rekening: '' }])}
        tugas={TASK}
        lockedReason="Ambil tugas lebih dulu."
      />,
    )

    expect(screen.getByRole('button', { name: 'Tambah' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Tambah' })).toHaveAttribute('title', 'Ambil tugas lebih dulu.')
    await userEvent.click(screen.getByRole('button', { name: 'PT Lama' }))
    const pane = screen.getByRole('group', { name: 'InputReceiver' })
    expect(within(pane).getByLabelText('No Rekening')).toBeDisabled()
    expect(within(pane).getByRole('button', { name: 'Simpan' })).toHaveAttribute('title', 'Ambil tugas lebih dulu.')
    expect(within(pane).getByRole('button', { name: 'Simpan' })).toBeDisabled()
  })
})

describe('ReceiverTab — simpan penerima tersimpan', () => {
  it('mengirim id penerima dan email yang diketik lalu menutup panel', async () => {
    installFetch((url, method) => {
      if (url === '/api/registrasi/rekening/555') return json(200, { ...ACCOUNT, nomor_rekening: '555', email: '' })
      if (method === 'POST') return json(200, {})
      return undefined
    })
    wrap(
      <ReceiverTab
        klaim={claimWith([{ id: 'R1', nama: 'PT Lama', alamat: '', nama_bank: '', nomor_rekening: '555' }])}
        tugas={TASK}
        lockedReason={null}
      />,
    )

    await userEvent.click(screen.getByRole('button', { name: 'PT Lama' }))
    const pane = screen.getByRole('group', { name: 'InputReceiver' })
    await within(pane).findByText('PT Penerima')
    await userEvent.type(within(pane).getByLabelText(/Email/), 'baru@contoh.example')
    await userEvent.click(within(pane).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(screen.queryByRole('group', { name: 'InputReceiver' })).not.toBeInTheDocument())
    expect(calls.find((c) => c.method === 'POST')?.body).toMatchObject({
      id: 'R1',
      nomor_rekening: '555',
      email: 'baru@contoh.example',
      telepon: '0800',
    })
  })
})
