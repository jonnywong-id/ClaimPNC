import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { AcceptanceNumber, acceptanceNumberRules } from './AcceptanceNumber'
import type { Settlement } from './types'

function line(over: Partial<Settlement>): Settlement {
  return { tipe_pembayaran: '2', status_akseptasi: '1', status_akseptasi_lod: '1', nomor_akseptasi: 'A26', ...over } as Settlement
}

// Aturan disalin dari InputAdjustment_sect — setiap kasus menyebut syarat Pega-nya.
describe('aturan Nomor Akseptasi', () => {
  it('tampil hanya bila LOD disetujui dan nomor akseptasi terbit', () => {
    expect(acceptanceNumberRules(line({})).visible).toBe(true)
    expect(acceptanceNumberRules(line({ nomor_akseptasi: '' })).visible).toBe(false)
    expect(acceptanceNumberRules(line({ status_akseptasi_lod: '0' })).visible).toBe(false)
  })

  it('Transfer Kasir: bukan salvage, mati bila sudah pernah ditransfer', () => {
    expect(acceptanceNumberRules(line({})).cashierVisible).toBe(true)
    expect(acceptanceNumberRules(line({})).cashierDisabled).toBe(false)
    expect(acceptanceNumberRules(line({ tipe_pembayaran: '3' })).cashierVisible).toBe(false)
    expect(acceptanceNumberRules(line({ sudah_transfer_kasir: true })).cashierDisabled).toBe(true)
  })
})

function mount(over: Partial<Settlement>) {
  const apiClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={apiClient}>
      <AcceptanceNumber claimID="klaim-1" taskID="tugas-1" object={1} coverage={2} adjustment={3} line={line(over)} />
    </QueryClientProvider>,
  )
}

const TYPES = [
  { id: '1', nama: 'Pembayaran Biasa' },
  { id: '2', nama: 'Join Placement' },
  { id: '3', nama: 'Fronting' },
]

describe('tombol Nomor Akseptasi', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('Print mengunduh Draft Persetujuan baris itu', async () => {
    const calls: { url: string; body: unknown }[] = []
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: () => 'blob:draft', revokeObjectURL: () => undefined }))
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, body: init?.body ? JSON.parse(init.body as string) : null })
      return Promise.resolve(
        new Response('%PDF-1.3', {
          status: 200,
          headers: { 'Content-Type': 'application/pdf', 'Content-Disposition': 'attachment; filename="DraftPersetujuan_NO.A26.pdf"' },
        }),
      )
    })
    const user = userEvent.setup()
    mount({})
    expect(screen.getByText('A26')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Print' }))
    await vi.waitFor(() => expect(calls).toHaveLength(1))
    expect(calls[0]).toEqual({
      url: '/api/registrasi/klaim/klaim-1/akseptasi/draft',
      body: { tugas_id: 'tugas-1', objek: 1, jaminan: 2, adjustment: 3 },
    })
  })

  it('Print menampilkan pesan penolakan backend', async () => {
    vi.stubGlobal('fetch', () => {
      const body = { kode: 'validasi_gagal', pesan: 'Validasi gagal', detail: [{ kode: 'draft_akseptasi_tata_letak', field: 'akseptasi', pesan: 'The Draft Persetujuan layout for Travel and Personal Accident is not available yet.' }] }
      return Promise.resolve(new Response(JSON.stringify(body), { status: 422, headers: { 'Content-Type': 'application/json' } }))
    })
    const user = userEvent.setup()
    mount({})
    await user.click(screen.getByRole('button', { name: 'Print' }))
    expect(await screen.findByText(/layout for Travel and Personal Accident/)).toBeInTheDocument()
  })

  it('Transfer Kasir membuka dialog konfirmasi dengan ringkasan pembayaran', async () => {
    vi.stubGlobal('fetch', () =>
      Promise.resolve(
        new Response(
          JSON.stringify({ nomor_akseptasi: 'A26', penerima: 'PT PENERIMA', nomor_rekening: '123', nama_bank: 'BANK', email: 'e@x', nilai_nett_sen: 150000, mata_uang: 'IDR', masalah: '', konfirmasi: 'Apakah Anda Yakin Akseptasi : A26 DiTransfer Ke KASIR?', tipe_transfer: TYPES, fac_out: [] }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
      ),
    )
    const user = userEvent.setup()
    mount({})
    await user.click(screen.getByRole('button', { name: 'Transfer Kasir' }))
    const dialog = await screen.findByRole('dialog', { name: 'Transfer Pembayaran' })
    expect(await within(dialog).findByText('PT PENERIMA')).toBeInTheDocument()
    expect(within(dialog).getByText('Apakah Anda Yakin Akseptasi : A26 DiTransfer Ke KASIR?')).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: 'Transfer To Kasir' })).toBeEnabled()
  })

  it('Fronting: Fac-out yang dicentang ikut terkirim sebagai tidak dibayar', async () => {
    const posted: unknown[] = []
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      if (url.endsWith('/kasir/pratinjau')) {
        const body = {
          nomor_akseptasi: 'A26', penerima: 'PT PENERIMA', nomor_rekening: '123', nama_bank: 'BANK', email: 'e@x',
          nilai_nett_sen: 150000, mata_uang: 'IDR', masalah: '', konfirmasi: 'Apakah Anda Yakin Akseptasi : A26 DiTransfer Ke KASIR?',
          tipe_transfer: TYPES, fac_out: [{ nomor_dla: 'H26-1', nama_facout: 'REAS UJI', nilai_bayar: '2500.00', mata_uang: 'IDR' }],
        }
        return Promise.resolve(new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      }
      posted.push(JSON.parse(init?.body as string))
      return Promise.resolve(new Response(JSON.stringify({ klaim: {} }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    })
    const user = userEvent.setup()
    mount({})
    await user.click(screen.getByRole('button', { name: 'Transfer Kasir' }))
    const dialog = await screen.findByRole('dialog', { name: 'Transfer Pembayaran' })
    expect(within(dialog).queryByText('REAS UJI')).not.toBeInTheDocument()
    await user.selectOptions(await within(dialog).findByRole('combobox'), '3')
    await user.click(within(dialog).getByLabelText('Fac-out H26-1 tidak dibayar'))
    await user.click(within(dialog).getByRole('button', { name: 'Transfer To Kasir' }))
    await vi.waitFor(() => expect(posted).toHaveLength(1))
    expect(posted[0]).toEqual({ tugas_id: 'tugas-1', objek: 1, jaminan: 2, adjustment: 3, tipe_transfer: '3', fac_out_tidak_dibayar: ['H26-1'] })
  })

  it('Transfer Kasir mati sesudah ditransfer; tidak ada untuk salvage', () => {
    mount({ sudah_transfer_kasir: true })
    expect(screen.getByRole('button', { name: 'Transfer Kasir' })).toBeDisabled()
  })

  it('tidak menggambar apa pun sebelum nomor akseptasi terbit', () => {
    const { container } = mount({ nomor_akseptasi: '' })
    expect(container).toBeEmptyDOMElement()
  })
})
