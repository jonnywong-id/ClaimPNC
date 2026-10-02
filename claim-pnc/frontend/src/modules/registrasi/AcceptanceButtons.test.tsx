import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'

import { AcceptanceButtons, acceptanceRules } from './AcceptanceButtons'
import type { Settlement } from './types'

/** Baris minimal; hanya empat medan yang dibaca aturan tombol. */
function line(over: Partial<Settlement>): Settlement {
  return {
    tipe_pembayaran: '2',
    status_akseptasi: '1',
    status_akseptasi_lod: '',
    nomor_akseptasi: '',
    ...over,
  } as Settlement
}

// Aturan disalin dari ShowAdjustment_sect — setiap kasus menyebut syarat Pega-nya.
describe('aturan tombol akseptasi', () => {
  it('tidak tampil sebelum komite menyetujui (.AcceptanceStatus == 1)', () => {
    expect(acceptanceRules(line({ status_akseptasi: '' }), '006', 'Fire').visible).toBe(false)
    expect(acceptanceRules(line({ status_akseptasi: '0' }), '006', 'Fire').visible).toBe(false)
    expect(acceptanceRules(line({ status_akseptasi: '2' }), '006', 'Fire').visible).toBe(false)
    expect(acceptanceRules(line({}), '006', 'Fire').visible).toBe(true)
  })

  it('baru disetujui komite: Print LOD dan Persetujuan aktif, Print DLA mati', () => {
    const r = acceptanceRules(line({}), '006', 'Fire')
    expect(r.printLODDisabled).toBe(false)
    expect(r.acceptDisabled).toBe(false)
    expect(r.printDLADisabled).toBe(true)
  })

  it('sesudah diakseptasi (LOD 1, nomor terbit): hanya Print DLA yang aktif', () => {
    const r = acceptanceRules(line({ status_akseptasi_lod: '1', nomor_akseptasi: 'A26…' }), '006', 'Fire')
    expect(r.printLODDisabled).toBe(true)
    expect(r.acceptDisabled).toBe(true)
    expect(r.printDLADisabled).toBe(false)
  })

  it('LOD tidak disetujui (0): Persetujuan dan Print DLA mati', () => {
    const r = acceptanceRules(line({ status_akseptasi_lod: '0' }), '006', 'Fire')
    expect(r.acceptDisabled).toBe(true)
    expect(r.printDLADisabled).toBe(true)
  })

  it.each([
    ['PA', '002', 'PA', '2'],
    ['Travel', '005', 'Travel', '2'],
    ['Bonding', '003', 'Bonding', '2'],
    ['BondingKBG', '003', 'BondingKBG', '2'],
    ['Salvage', '006', 'Fire', '3'],
    ['Adjuster Fee', '006', 'Fire', '4'],
  ])('Print LOD mati untuk %s', (_, panel, bisnis, tipe) => {
    const r = acceptanceRules(line({ tipe_pembayaran: tipe }), panel, bisnis)
    expect(r.printLODDisabled).toBe(true)
    expect(r.acceptDisabled).toBe(false)
  })
})

function mount(over: Partial<Settlement>, groupPanel = '006') {
  const apiClient = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(
    <QueryClientProvider client={apiClient}>
      <AcceptanceButtons
        claimID="klaim-1"
        taskID="tugas-1"
        object={1}
        coverage={2}
        adjustment={3}
        line={line(over)}
        groupPanel={groupPanel}
        businessType="Fire"
        receivers={[{ id: 'R1', nama: 'PENERIMA UJI', alamat: '', nama_bank: '', nomor_rekening: '' }]}
      />
    </QueryClientProvider>,
  )
}

const TYPES = {
  email_lod: 'tertanggung@contoh.internal,pic@contoh.internal',
  nama_tertanggung: 'TERTANGGUNG UJI',
  tipe: [
    { id: '1', nama: 'Property - Final', tersedia: false },
    { id: '3', nama: 'Property - Final dengan Co Member', tersedia: true },
  ],
}

describe('tombol akseptasi', () => {
  afterEach(() => vi.unstubAllGlobals())

  it('Print LOD: dialog Email LOD, memilih Tipe PDF lalu mengunduh; jenis tanpa contoh tidak dapat dicetak', async () => {
    const calls: { url: string; body: unknown }[] = []
    vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: () => 'blob:lod', revokeObjectURL: () => undefined }))
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, body: init?.body ? JSON.parse(init.body as string) : null })
      if (url.endsWith('/lod/tipe')) {
        return Promise.resolve(new Response(JSON.stringify(TYPES), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      }
      return Promise.resolve(
        new Response('%PDF-1.3', { status: 200, headers: { 'Content-Type': 'application/pdf', 'Content-Disposition': 'attachment; filename="LOD.pdf"' } }),
      )
    })
    const user = userEvent.setup()
    mount({})

    await user.click(screen.getByRole('button', { name: 'Print LOD' }))
    const dialog = await screen.findByRole('dialog', { name: 'Email LOD' })
    const select = await within(dialog).findByRole('combobox')
    const print = within(dialog).getByRole('button', { name: 'Print LOD' })
    // Isian PrintLODdanEmail diisi awal seperti SetDataEmailTertanggung; Send Email menunggu.
    expect(within(dialog).getByLabelText('Email LOD')).toHaveValue('tertanggung@contoh.internal,pic@contoh.internal')
    expect(within(dialog).getByLabelText('Nama Tertanggung')).toHaveValue('TERTANGGUNG UJI')
    expect(within(dialog).getByLabelText('Catatan')).toHaveValue('')
    expect(within(dialog).getByRole('button', { name: 'Send Email' })).toBeDisabled()

    await user.selectOptions(select, '1')
    expect(within(dialog).getByRole('status')).toHaveTextContent(/template for this LOD type is not available/)
    expect(print).toBeDisabled()

    await user.selectOptions(select, '3')
    await user.click(print)
    expect(await within(dialog).findByText('LOD downloaded.')).toBeInTheDocument()
    expect(calls).toEqual([
      { url: '/api/registrasi/klaim/klaim-1/lod/tipe', body: { tugas_id: 'tugas-1', objek: 1, jaminan: 2, adjustment: 3 } },
      { url: '/api/registrasi/klaim/klaim-1/lod', body: { tugas_id: 'tugas-1', objek: 1, jaminan: 2, adjustment: 3, tipe_pdf: '3' } },
    ])
  })

  it('Persetujuan / Akseptasi: Simpan mengirim isian dan berkas lalu menutup form', async () => {
    const posted: { url: string; form: FormData }[] = []
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      if (url.endsWith('/dokumen')) {
        const docs = { kategori: [{ kode: '10067', nama: 'PAYMENT', dokumen: [{ id: '14904', jenis_id: '10067', nama: 'KWITANSI', wajib: false, minimal: '0', terunggah: 0 }] }], berkas: [] }
        return Promise.resolve(new Response(JSON.stringify(docs), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      }
      if (url.endsWith('/akseptasi/awal')) {
        const initial = {
          tipe_akseptasi: '2',
          pilihan_tipe_akseptasi: [{ id: '1', nama: 'Indemnity' }, { id: '2', nama: 'Interim' }],
          nama_komite_akseptasi: 'KOMITEUJI',
          nilai_lod_sen: 400_000_000,
          peringatan: [],
        }
        return Promise.resolve(new Response(JSON.stringify(initial), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      }
      posted.push({ url, form: init?.body as FormData })
      return Promise.resolve(new Response(JSON.stringify({ klaim: {} }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    })
    const user = userEvent.setup()
    mount({})

    await user.click(screen.getByRole('button', { name: 'Persetujuan / Akseptasi' }))
    const form = screen.getByRole('dialog', { name: 'AcceptationLOD' })
    expect(within(form).getByRole('group', { name: /^Persetujuan Tertanggung/ })).toBeInTheDocument()
    await user.click(within(form).getByLabelText('Setuju'))
    await user.type(within(form).getByLabelText('Tanggal Terima LOD *'), '01/09/2026')
    await user.type(within(form).getByLabelText('Tanggal Boleh Bayar *'), '02/09/2026')
    // AcceptationLOD_PreAct: Tipe Akseptasi, Nama Komite, dan Nilai LOD sudah terisi.
    await waitFor(() => expect(within(form).getByLabelText('Tipe Akseptasi Klaim')).toHaveValue('2'))
    expect(within(form).getByLabelText('Nama Komite Akseptasi')).toHaveValue('KOMITEUJI')
    expect(within(form).getByLabelText('Nilai LOD *')).toHaveValue('4000000,00')
    await user.clear(within(form).getByLabelText('Nilai LOD *'))
    await user.type(within(form).getByLabelText('Nilai LOD *'), '5.000.000')
    await user.selectOptions(within(form).getByLabelText(/^Penerima Klaim/), 'R1')
    await user.click(within(form).getByRole('button', { name: 'Select file(s)' }))
    await within(form).findByRole('option', { name: 'PAYMENT — KWITANSI' })
    await user.selectOptions(within(form).getByLabelText('Jenis dokumen 1'), '14904')
    await user.upload(within(form).getByLabelText('Berkas 1'), new File(['%PDF'], 'lod.pdf', { type: 'application/pdf' }))
    await user.click(within(form).getByRole('button', { name: 'Submit' }))

    await waitFor(() => expect(screen.queryByRole('dialog', { name: 'AcceptationLOD' })).not.toBeInTheDocument())
    expect(posted).toHaveLength(1)
    expect(posted[0]!.url).toBe('/api/registrasi/klaim/klaim-1/akseptasi')
    const isian = JSON.parse(posted[0]!.form.get('isian') as string)
    expect(isian).toMatchObject({
      tugas_id: 'tugas-1', objek: 1, jaminan: 2, adjustment: 3, tipe_akseptasi: '2', persetujuan_tertanggung: '1',
      tanggal_terima_lod: '2026-09-01', tanggal_boleh_bayar: '2026-09-02', nilai_lod_sen: 500_000_000, penerima: 'R1',
      nama_komite_akseptasi: 'KOMITEUJI',
    })
    expect(posted[0]!.form.getAll('jenis_dokumen')).toEqual(['14904'])
    expect((posted[0]!.form.get('berkas') as File).name).toBe('lod.pdf')
  })

  it('Persetujuan / Akseptasi menampilkan pesan penolakan backend', async () => {
    vi.stubGlobal('fetch', (url: string) => {
      if (url.endsWith('/dokumen')) {
        return Promise.resolve(new Response(JSON.stringify({ kategori: [], berkas: [] }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
      }
      const body = { kode: 'validasi_gagal', pesan: 'Validasi gagal', detail: [{ kode: 'akseptasi_premi', field: 'berita_acara', pesan: 'Premi belum lunas, tidak bisa akseptasi adjustment.' }] }
      return Promise.resolve(new Response(JSON.stringify(body), { status: 422, headers: { 'Content-Type': 'application/json' } }))
    })
    const user = userEvent.setup()
    mount({})
    await user.click(screen.getByRole('button', { name: 'Persetujuan / Akseptasi' }))
    const form = screen.getByRole('dialog', { name: 'AcceptationLOD' })
    await user.click(within(form).getByRole('button', { name: 'Submit' }))
    expect(await within(form).findByRole('alert')).toHaveTextContent(/Premi belum lunas/)
  })

  it('AcceptationLOD: Tipe PDF dan Tanggal Cetak LOD baca saja dari Print LOD; Penerima Klaim hanya dropdown', async () => {
    const user = userEvent.setup()
    mount({ tanggal_cetak_lod: '2026-09-30T07:59:00+07:00', tipe_pdf_lod: '15', nama_tipe_pdf_lod: 'Pembayaran Final dengan Co Member Ex Gratia' })
    await user.click(screen.getByRole('button', { name: 'Persetujuan / Akseptasi' }))
    const form = screen.getByRole('dialog', { name: 'AcceptationLOD' })
    expect(within(form).getByText('Pembayaran Final dengan Co Member Ex Gratia')).toBeInTheDocument()
    expect(within(form).getByText('30/09/2026 7:59')).toBeInTheDocument()
    expect(within(form).queryByLabelText(/Tanggal Cetak/)).not.toBeInTheDocument()
    expect(within(form).getAllByText(/^Penerima Klaim/)).toHaveLength(1)
    expect(within(form).getByLabelText('Remark')).toBeInTheDocument()
    expect(within(form).getByLabelText('Tidak Setuju')).toBeInTheDocument()
  })

  it('form Travel tanpa isian non-Travel dan tanpa Nilai LOD', async () => {
    const user = userEvent.setup()
    mount({}, '005')
    await user.click(screen.getByRole('button', { name: 'Persetujuan / Akseptasi' }))
    const form = screen.getByRole('dialog', { name: 'AcceptationLOD' })
    expect(within(form).queryByLabelText('Tanggal Boleh Bayar *')).not.toBeInTheDocument()
    expect(within(form).queryByLabelText('Nilai LOD *')).not.toBeInTheDocument()
  })

  it('Print DLA membuka daftar DLA per penerima', async () => {
    const requests: string[] = []
    vi.stubGlobal('fetch', (url: string) => {
      requests.push(url)
      return Promise.resolve(
        new Response(
          JSON.stringify({
            baru_terbit: 1,
            ex_gratia: false,
            peringatan: [],
            dla: [
              { nomor: 'J261000000000000001', penerima: 'KOASURADUR UJI', tipe: 'COINS', catatan: '', email: '', tanggal: '2026-09-30', nilai: '2500000.00', mata_uang: 'IDR', sudah_cetak: false, sudah_kirim: false },
            ],
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
      )
    })
    const user = userEvent.setup()
    mount({ status_akseptasi_lod: '1', nomor_akseptasi: 'A26' })
    await user.click(screen.getByRole('button', { name: 'Print DLA' }))
    const dialog = await screen.findByRole('dialog', { name: 'Print DLA' })
    expect(await within(dialog).findByText('KOASURADUR UJI')).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: 'PRINT' })).toBeEnabled()
    expect(within(dialog).getByRole('button', { name: 'SEND ALL DLA' })).toBeDisabled()
    expect(requests.some((u) => u.endsWith('/dla/daftar'))).toBe(true)
    vi.unstubAllGlobals()
  })

  it('Print DLA klaim ex gratia menyatakan tidak ada DLA', async () => {
    vi.stubGlobal('fetch', () =>
      Promise.resolve(
        new Response(JSON.stringify({ baru_terbit: 0, ex_gratia: true, peringatan: [], dla: [] }), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      ),
    )
    const user = userEvent.setup()
    mount({ status_akseptasi_lod: '1', nomor_akseptasi: 'A26' })
    await user.click(screen.getByRole('button', { name: 'Print DLA' }))
    const dialog = await screen.findByRole('dialog', { name: 'Print DLA' })
    expect(await within(dialog).findByText(/ex gratia/)).toBeInTheDocument()
    vi.unstubAllGlobals()
  })

  it('kedua form tampil sebagai pop up modal yang tertutup dengan Escape', async () => {
    vi.stubGlobal('fetch', () =>
      Promise.resolve(new Response(JSON.stringify({ kategori: [], berkas: [], tipe: [], email_lod: '', nama_tertanggung: '' }), { status: 200, headers: { 'Content-Type': 'application/json' } })),
    )
    const user = userEvent.setup()
    mount({})
    for (const [button, title] of [['Print LOD', 'Email LOD'], ['Persetujuan / Akseptasi', 'AcceptationLOD']] as const) {
      await user.click(screen.getByRole('button', { name: button }))
      const popup = screen.getByRole('dialog', { name: title })
      expect(popup).toHaveAttribute('aria-modal', 'true')
      await user.keyboard('{Escape}')
      expect(screen.queryByRole('dialog', { name: title })).not.toBeInTheDocument()
    }
  })

  it('tidak menggambar apa pun sebelum komite menyetujui', () => {
    const { container } = mount({ status_akseptasi: '' })
    expect(container).toBeEmptyDOMElement()
  })
})
