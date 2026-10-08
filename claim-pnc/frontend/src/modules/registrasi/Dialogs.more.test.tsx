import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { CashierDialog } from './CashierDialog'
import { DLADialog } from './DLADialog'
import { PLADialog } from './PLADialog'

/**
 * Uji dialog Transfer Kasir, Print DLA, dan Print PLA, dirender langsung tanpa layar klaim.
 * Seluruh data KARANGAN (`D-69`).
 */

const BASE = '/api/registrasi/klaim/klaim-1'
const ADDRESS = { tugas_id: 'tugas-1', objek: 1, jaminan: 1, adjustment: 1 }

type Answer = Response | Promise<Response> | 'putus' | undefined

let calls: { url: string; body: unknown }[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function file(): Response {
  return new Response('%PDF', {
    status: 200,
    headers: { 'Content-Disposition': 'attachment; filename="dok.pdf"' },
  })
}

function validation(...pesan: string[]): Response {
  return json(422, {
    kode: 'validasi_gagal',
    pesan: 'x',
    detail: pesan.map((p) => ({ kode: 'aturan', field: '', pesan: p })),
  })
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
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(<QueryClientProvider client={client}>{children}</QueryClientProvider>)
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
  vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:contoh')
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
  useSelectedPortal.getState().clear()
})

const PREVIEW = {
  nomor_akseptasi: 'AKS-1',
  penerima: '',
  nomor_rekening: '',
  nama_bank: '',
  email: '',
  nilai_nett_sen: 123456,
  mata_uang: 'IDR',
  masalah: '',
}

describe('CashierDialog', () => {
  it('menampilkan pratinjau, mengirim, lalu menyebut berhasil', async () => {
    installFetch((url) => {
      if (url === `${BASE}/kasir/pratinjau`) return json(200, PREVIEW)
      if (url === `${BASE}/kasir`) return json(200, {})
      return undefined
    })
    const onClose = vi.fn()
    wrap(<CashierDialog claimID="klaim-1" address={ADDRESS} onClose={onClose} />)

    expect(await screen.findByText('AKS-1')).toBeInTheDocument()
    expect(screen.getAllByText('—')).toHaveLength(4)
    expect(screen.getByText(/IDR 1\.234,56/)).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Transfer To Kasir' }))
    expect(await screen.findByText('Berhasil ditransfer ke Kasir.')).toBeInTheDocument()
    expect(calls.find((c) => c.url === `${BASE}/kasir`)?.body).toEqual({ ...ADDRESS, tipe_transfer: '1', fac_out_tidak_dibayar: [] })
    expect(screen.queryByRole('button', { name: 'Transfer To Kasir' })).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Tutup' }))
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('menahan Transfer To Kasir bila pratinjau menyebut masalah, dan menutup lewat Escape', async () => {
    installFetch((url) =>
      url === `${BASE}/kasir/pratinjau`
        ? json(200, { ...PREVIEW, penerima: 'PT Contoh', nomor_rekening: '123', nama_bank: 'BANK', email: 'a@contoh.example', masalah: 'Rekening belum disetujui.' })
        : undefined,
    )
    const onClose = vi.fn()
    wrap(<CashierDialog claimID="klaim-1" address={ADDRESS} onClose={onClose} />)

    expect(await screen.findByText('Rekening belum disetujui.')).toBeInTheDocument()
    expect(screen.getByText('PT Contoh')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Transfer To Kasir' })).toBeDisabled()

    fireEvent.keyDown(document, { key: 'Enter' })
    expect(onClose).not.toHaveBeenCalled()
    fireEvent.keyDown(document, { key: 'Escape' })
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it.each([
    { name: 'pelanggaran validasi', answer: () => validation('Rekening kosong.', 'Nilai nol.'), text: 'Rekening kosong. Nilai nol.' },
    { name: 'galat API', answer: () => json(500, { kode: 'x', pesan: 'Kasir sibuk.' }), text: 'Kasir sibuk.' },
  ])('menampilkan galat pratinjau: $name', async ({ answer, text }) => {
    installFetch((url) => (url === `${BASE}/kasir/pratinjau` ? answer() : undefined))
    wrap(<CashierDialog claimID="klaim-1" address={ADDRESS} onClose={() => {}} />)

    expect(await screen.findByText('Transfer Kasir belum dapat diproses')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Transfer To Kasir' })).toBeDisabled()
  })

  it('menampilkan galat pengiriman', async () => {
    installFetch((url) => {
      if (url === `${BASE}/kasir/pratinjau`) return json(200, PREVIEW)
      if (url === `${BASE}/kasir`) return 'putus'
      return undefined
    })
    wrap(<CashierDialog claimID="klaim-1" address={ADDRESS} onClose={() => {}} />)

    await userEvent.click(await screen.findByRole('button', { name: 'Transfer To Kasir' }))
    expect(await screen.findByText('Tidak dapat menghubungi server Claim PNC.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Batal' })).toBeEnabled()
  })
})

function dla(nomor: string, extra: Record<string, unknown> = {}) {
  return {
    nomor,
    penerima: 'Reas Contoh',
    tipe: 'OR',
    catatan: '',
    email: '',
    tanggal: '',
    nilai: '',
    mata_uang: 'IDR',
    sudah_cetak: false,
    sudah_kirim: false,
    ...extra,
  }
}

describe('DLADialog', () => {
  it('mengirim catatan baris yang belum dicetak lalu memuat ulang daftar', async () => {
    let listed = 0
    installFetch((url) => {
      if (url === `${BASE}/dla/daftar`) {
        listed += 1
        return json(200, {
          baru_terbit: 0,
          ex_gratia: false,
          peringatan: ['Peringatan satu', 'Peringatan dua'],
          dla: [dla('DLA-1'), dla('DLA-2', { sudah_cetak: true, catatan: 'lama', email: 'r@contoh.example' })],
        })
      }
      if (url === `${BASE}/dla`) return file()
      return undefined
    })
    const onClose = vi.fn()
    wrap(<DLADialog claimID="klaim-1" address={ADDRESS} onClose={onClose} />)

    expect(await screen.findByText('DLA-1')).toBeInTheDocument()
    expect(screen.getByText(/Peringatan satu\s+Peringatan dua/)).toBeInTheDocument()
    expect(screen.getByLabelText('Remarks DLA-2')).toHaveAttribute('readonly')
    expect(screen.getByRole('button', { name: 'Kirim' })).toBeDisabled()

    await userEvent.type(screen.getByLabelText('Remarks DLA-1'), 'catatan baru')
    await userEvent.click(screen.getByRole('checkbox', { name: 'Coverage AS PER ORIGINAL POLICY' }))
    await userEvent.click(screen.getByRole('button', { name: 'Print All DLA' }))

    expect(await screen.findByText('DLA diunduh.')).toBeInTheDocument()
    expect(calls.find((c) => c.url === `${BASE}/dla`)?.body).toEqual({
      ...ADDRESS,
      catatan: { 'DLA-1': 'catatan baru' },
      sesuai_polis: true,
    })
    await waitFor(() => expect(listed).toBe(2))

    const row = screen.getByText('DLA-2').closest('tr')!
    await userEvent.click(within(row).getByRole('button', { name: 'PRINT' }))
    await waitFor(() =>
      expect(calls.filter((c) => c.url === `${BASE}/dla`).pop()?.body).toMatchObject({ nomor: 'DLA-2' }),
    )

    fireEvent.keyDown(document, { key: 'Escape' })
    await waitFor(() => expect(onClose).toHaveBeenCalled())
  })

  it('menyebut klaim ex gratia', async () => {
    installFetch((url) =>
      url === `${BASE}/dla/daftar` ? json(200, { baru_terbit: 0, ex_gratia: true, peringatan: [], dla: [] }) : undefined,
    )
    wrap(<DLADialog claimID="klaim-1" address={ADDRESS} onClose={() => {}} />)

    expect(await screen.findByText('No DLA is issued for an ex gratia claim.')).toBeInTheDocument()
    expect(screen.queryByText(/No DLA recipient/)).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Print All DLA' })).toBeDisabled()
  })

  it('menyebut tidak ada penerima DLA', async () => {
    installFetch((url) =>
      url === `${BASE}/dla/daftar` ? json(200, { baru_terbit: 0, ex_gratia: false, peringatan: [], dla: [] }) : undefined,
    )
    wrap(<DLADialog claimID="klaim-1" address={ADDRESS} onClose={() => {}} />)

    expect(await screen.findByText(/No DLA recipient/)).toBeInTheDocument()
  })

  it('menampilkan pelanggaran validasi saat daftar DLA dibuka', async () => {
    installFetch((url) => (url === `${BASE}/dla/daftar` ? validation('DLA terkunci.') : undefined))
    wrap(<DLADialog claimID="klaim-1" address={ADDRESS} onClose={() => {}} />)

    expect(await screen.findByText('DLA belum dapat diproses')).toBeInTheDocument()
    expect(screen.getByText('DLA terkunci.')).toBeInTheDocument()
  })

  it('menampilkan galat cetak dari server', async () => {
    installFetch((url) => {
      if (url === `${BASE}/dla/daftar`) {
        return json(200, { baru_terbit: 1, ex_gratia: false, peringatan: [], dla: [dla('DLA-1')] })
      }
      if (url === `${BASE}/dla`) return json(500, { kode: 'x', pesan: 'Cetak gagal.' })
      return undefined
    })
    wrap(<DLADialog claimID="klaim-1" address={ADDRESS} onClose={() => {}} />)

    await userEvent.click(await screen.findByRole('button', { name: 'PRINT' }))
    expect(await screen.findByText('DLA belum dapat diproses')).toBeInTheDocument()
    expect(screen.getByText('Cetak gagal.')).toBeInTheDocument()
    expect(screen.queryByText('DLA diunduh.')).not.toBeInTheDocument()
  })
})

function pla(nomor: string, catatan = '', email = '') {
  return { nomor, penerima: 'Reas Contoh', tipe: 'OR', catatan, email, tanggal: '' }
}

describe('PLADialog', () => {
  it('menahan cetak sampai remarks disimpan, lalu mencetak satu dan seluruhnya', async () => {
    installFetch((url) => {
      if (url === `${BASE}/pla/daftar`) return json(200, { revisi_cfs: 1, baru_terbit: 0, pla: [pla('PLA-1'), pla('PLA-2', 'ada')] })
      if (url === `${BASE}/pla/catatan`)
        return json(200, { revisi_cfs: 1, baru_terbit: 0, pla: [pla('PLA-1', 'baru', 'reas@contoh.example'), pla('PLA-2', 'ada')] })
      if (url === `${BASE}/pla`) return file()
      return undefined
    })
    const onClose = vi.fn()
    wrap(
      <PLADialog claimID="klaim-1" taskID="tugas-1" object={1} coverage={2} coverageName="All Risk" onClose={onClose} />,
    )

    expect(screen.getByText('All Risk')).toBeInTheDocument()
    expect(await screen.findByText('PLA-1')).toBeInTheDocument()
    expect(screen.getByLabelText('Email PLA-1')).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Simpan Remarks & Email' })).toBeDisabled()

    await userEvent.type(screen.getByLabelText('Remarks PLA-1'), 'baru')
    await userEvent.type(screen.getByLabelText('Email PLA-1'), ' reas@contoh.example ')
    const printOne = within(screen.getByText('PLA-1').closest('tr')!).getByRole('button', { name: 'Print PLA' })
    expect(printOne).toBeDisabled()
    expect(printOne).toHaveAttribute('title', 'Simpan remarks dan email lebih dulu.')
    expect(screen.getByRole('button', { name: 'Print All PLA' })).toBeDisabled()

    await userEvent.click(screen.getByRole('button', { name: 'Simpan Remarks & Email' }))
    await waitFor(() => expect(printOne).toBeEnabled())
    expect(calls.find((c) => c.url === `${BASE}/pla/catatan`)?.body).toEqual({
      tugas_id: 'tugas-1',
      objek: 1,
      jaminan: 2,
      catatan: { 'PLA-1': 'baru' },
      email: { 'PLA-1': 'reas@contoh.example' },
    })
    expect(screen.getByLabelText('Email PLA-1')).toHaveValue('reas@contoh.example')

    await userEvent.click(printOne)
    expect(await screen.findByText('PLA diunduh.')).toBeInTheDocument()
    expect(calls.filter((c) => c.url === `${BASE}/pla`).pop()?.body).toMatchObject({ nomor: 'PLA-1' })

    await userEvent.click(screen.getByRole('button', { name: 'Print All PLA' }))
    await waitFor(() => expect(calls.filter((c) => c.url === `${BASE}/pla`)).toHaveLength(2))
    expect(calls.filter((c) => c.url === `${BASE}/pla`).pop()?.body).toEqual({ tugas_id: 'tugas-1', objek: 1, jaminan: 2 })

    fireEvent.keyDown(document, { key: 'Escape' })
    await waitFor(() => expect(onClose).toHaveBeenCalled())
  })

  it.each([
    { name: 'pelanggaran validasi', answer: () => validation('CFS belum ada.'), text: 'CFS belum ada.' },
    { name: 'jaringan putus', answer: (): Answer => 'putus', text: 'Tidak dapat menghubungi server Claim PNC.' },
  ])('menampilkan galat daftar: $name', async ({ answer, text }) => {
    installFetch((url) => (url === `${BASE}/pla/daftar` ? answer() : undefined))
    wrap(<PLADialog claimID="klaim-1" taskID="tugas-1" object={1} coverage={1} coverageName="X" onClose={() => {}} />)

    expect(await screen.findByText('PLA belum dapat diproses')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Print All PLA' })).toBeDisabled()
  })
})

describe('keadaan menyiapkan', () => {
  it.each([
    { name: 'Transfer Kasir', path: '/kasir/pratinjau', text: 'Menyiapkan data…' },
    { name: 'Print DLA', path: '/dla/daftar', text: 'Menyiapkan DLA…' },
    { name: 'Print PLA', path: '/pla/daftar', text: 'Menyiapkan PLA…' },
  ])('menyebut $name sedang disiapkan', async ({ path, text }) => {
    installFetch((url) => (url === BASE + path ? new Promise<Response>(() => {}) : undefined))
    wrap(
      path.startsWith('/kasir') ? (
        <CashierDialog claimID="klaim-1" address={ADDRESS} onClose={() => {}} />
      ) : path.startsWith('/dla') ? (
        <DLADialog claimID="klaim-1" address={ADDRESS} onClose={() => {}} />
      ) : (
        <PLADialog claimID="klaim-1" taskID="tugas-1" object={1} coverage={1} coverageName="X" onClose={() => {}} />
      ),
    )

    expect(await screen.findByText(text)).toBeInTheDocument()
  })
})
