import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { LODTypeSelect, lodTypeEditable, lodTypeVisible } from './LODTypeSelect'
import type { Settlement } from './types'

const TYPES = { email_lod: '', nama_tertanggung: '', tipe: [{ id: '15', nama: 'Pembayaran Final dengan Co Member Ex Gratia', tersedia: true }] }

function line(over: Partial<Settlement>): Settlement {
  return { tipe_pembayaran: '2', status_akseptasi: '1', status_akseptasi_lod: '', nomor_akseptasi: '', ...over } as Settlement
}

function mount(over: Partial<Settlement>, panel = '006') {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <LODTypeSelect claimID="klaim-1" taskID="tugas-1" object={1} coverage={2} adjustment={3} line={line(over)}
        groupPanel={panel} businessType="Fire" lockedReason={null} />
    </QueryClientProvider>,
  )
}

describe('dropdown Tipe LOD', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('aturan tampil dan ubah dari ShowAdjustment_sect', () => {
    expect(lodTypeVisible('006', 'Fire')).toBe(true)
    expect(lodTypeVisible('002', 'PA')).toBe(false)
    expect(lodTypeVisible('005', 'Travel')).toBe(false)
    expect(lodTypeVisible('003', 'Bonding')).toBe(false)
    expect(lodTypeEditable(line({}))).toBe(true)
    expect(lodTypeEditable(line({ status_akseptasi_lod: '1', nomor_akseptasi: 'A26' }))).toBe(false)
  })

  it('memilih Tipe LOD menyimpan PDFTYPE baris itu', async () => {
    const calls: { url: string; body: unknown }[] = []
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, body: init?.body ? JSON.parse(init.body as string) : null })
      const body = url.endsWith('/lod/tipe') ? TYPES : { klaim: {} }
      return Promise.resolve(new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    })
    const user = userEvent.setup()
    mount({})
    const select = screen.getByRole('combobox', { name: 'Tipe LOD adjustment 3' })
    await screen.findByRole('option', { name: 'Pembayaran Final dengan Co Member Ex Gratia' })
    await user.selectOptions(select, '15')
    await vi.waitFor(() => expect(calls.some((c) => c.url.endsWith('/lod/pilih'))).toBe(true))
    expect(calls.find((c) => c.url.endsWith('/lod/pilih'))?.body).toEqual({ tugas_id: 'tugas-1', objek: 1, jaminan: 2, adjustment: 3, tipe_pdf: '15' })
  })

  it('setelah akseptasi dropdown tetap tampil, terkunci dengan pilihannya; PA tidak menggambar apa pun', () => {
    mount({ status_akseptasi_lod: '1', nomor_akseptasi: 'A26', tipe_pdf_lod: '4', nama_tipe_pdf_lod: 'Property - Final tanpa Co Member' })
    const select = screen.getByRole('combobox')
    expect(select).toBeDisabled()
    expect(select).toHaveValue('4')
    const { container } = mount({}, '002')
    expect(container).toBeEmptyDOMElement()
  })
})
