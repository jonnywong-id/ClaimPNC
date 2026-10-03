import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { APIError, NetworkError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { AdvicePanel } from './AdvicePanel'
import { ApprovalPanel } from './ApprovalPanel'
import { ClaimPanel } from './ClaimPanel'
import { isValidationError, messageOf, violationsOf } from './errors'
import type { MasterXOL } from './types'

/** Uji tambahan Inbox XOL: galat per panel, pengurutan, dan pembantu galat. Nilai KARANGAN. */

const PATH = '/api/inbox-xol'

const MASTER: MasterXOL = {
  id: 'XOL-001',
  nama: 'XOL Contoh',
  tahun: '2024',
  kurs: 15500,
  tipe: '',
  group_business: '',
  jumlah_group_business: 1,
  status_komite: '1',
  menunggu_komite: false,
  catatan_komite: '',
  pic: 'PIC',
  catatan_pic: '',
}

type Answer = { status?: number; body?: unknown; fail?: boolean }
let calls: string[] = []

function stub(answer: (url: string) => Answer) {
  vi.stubGlobal('fetch', (url: string) => {
    calls.push(url)
    const reply = answer(url)
    if (reply.fail) return Promise.reject(new TypeError('Failed to fetch'))
    return Promise.resolve(
      new Response(JSON.stringify(reply.body ?? {}), {
        status: reply.status ?? 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function wrap(children: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={client}>{children}</QueryClientProvider>)
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: null,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
})

describe('errors', () => {
  it('mengambil pesan dari galat API dan jaringan, dan menyembunyikan galat lain', () => {
    expect(messageOf(new APIError('x', 'Pesan server.', 500))).toBe('Pesan server.')
    expect(messageOf(new NetworkError())).toBe('Tidak dapat menghubungi server Claim PNC.')
    expect(messageOf(new Error('SELECT * FROM rahasia'))).toBe(
      'Terjadi kesalahan yang tidak dikenali. Coba lagi beberapa saat lagi.',
    )
  })

  it('mengenali galat validasi saja sebagai pelanggaran isian', () => {
    const validation = new APIError('validasi_gagal', 'x', 422, [{ field: 'tahun', pesan: 'Tahun wajib.' }])
    expect(violationsOf(validation)).toEqual({ tahun: 'Tahun wajib.' })
    expect(isValidationError(validation)).toBe(true)

    const other = new APIError('galat_internal', 'x', 500, [{ field: 'tahun', pesan: 'abaikan' }])
    expect(violationsOf(other)).toEqual({})
    expect(isValidationError(other)).toBe(false)
    expect(isValidationError('bukan galat')).toBe(false)
  })
})

describe('ClaimPanel', () => {
  it('menyatakan daftar perjanjian yang gagal dimuat', async () => {
    stub((url) => (url === `${PATH}/perjanjian` ? { status: 500, body: { kode: 'x', pesan: 'Master rusak.' } } : {}))
    wrap(<ClaimPanel />)

    expect(await screen.findByText('Daftar perjanjian XOL tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Master rusak.')).toBeInTheDocument()
  })

  it('menyatakan akumulasi dan rincian yang gagal dimuat, beserta label perjanjian tanpa grup', async () => {
    stub((url) => {
      if (url === `${PATH}/perjanjian`) return { body: { perjanjian: [MASTER] } }
      return { fail: true }
    })
    const user = userEvent.setup()
    wrap(<ClaimPanel />)

    const picker = await screen.findByLabelText('Perjanjian XOL')
    await screen.findByRole('option', { name: /tanpa group business/ })
    await user.selectOptions(picker, 'XOL-001')

    expect(await screen.findByText('Akumulasi klaim tidak dapat dimuat')).toBeInTheDocument()
    // Tanpa jawaban server, keterangan kurs tidak dapat menyebut angkanya.
    expect(screen.getByText('Nilai dibagi kurs perjanjian, mengikuti perhitungan sistem lama.')).toBeInTheDocument()
  })

  it('mengurutkan akumulasi dan rincian, serta menyatakan rincian yang gagal dimuat', async () => {
    let breakdownFails = false
    stub((url) => {
      if (url === `${PATH}/perjanjian`) return { body: { perjanjian: [MASTER] } }
      if (url.startsWith(`${PATH}/klaim/rincian`)) {
        return breakdownFails
          ? { status: 500, body: { kode: 'x', pesan: 'Rincian rusak.' } }
          : {
              body: {
                baris: [
                  { group_business: 'Marine', kode_group_business: '20', jumlah_klaim: 3, nilai_outstanding: 9, nilai_akseptasi: 1, sumber: 'bisnis', kurs_tidak_tersedia: false },
                  { group_business: 'Fire', kode_group_business: '10', jumlah_klaim: 1, nilai_outstanding: 2, nilai_akseptasi: 5, sumber: 'bisnis', kurs_tidak_tersedia: false },
                ],
              },
            }
      }
      if (url.startsWith(`${PATH}/klaim`)) {
        return {
          body: {
            perjanjian: { ...MASTER, tipe: 'PROPERTY' },
            baris: [
              { tanggal_kejadian: '2024', sebab_kerugian: 'GEMPA', group_business: 'B', nilai_outstanding: 1, nilai_akseptasi: 1 },
              { tanggal_kejadian: '2023', sebab_kerugian: 'BANJIR', group_business: 'A', nilai_outstanding: 2, nilai_akseptasi: 2 },
            ],
          },
        }
      }
      return {}
    })
    const user = userEvent.setup()
    wrap(<ClaimPanel />)

    const picker = await screen.findByLabelText('Perjanjian XOL')
    await screen.findByRole('option', { name: /2024/ })
    await user.selectOptions(picker, 'XOL-001')

    const table = await screen.findByRole('table')
    const first = () => within(table).getAllByRole('row')[1]?.textContent ?? ''
    await user.click(within(table).getByRole('button', { name: 'Date Of Loss' }))
    expect(first()).toContain('2023')
    await user.click(within(table).getByRole('button', { name: 'Cause Of Loss' }))
    expect(first()).toContain('BANJIR')
    await user.click(within(table).getByRole('button', { name: 'Group Business' }))
    expect(first()).toContain('A')
    await user.click(within(table).getByRole('button', { name: 'OS Value' }))
    await user.click(within(table).getByRole('button', { name: 'Accepted Value' }))
    expect(first()).toContain('GEMPA')

    await user.click(within(table).getAllByRole('button', { name: 'Lihat rincian' })[0]!)
    const tables = await screen.findAllByRole('table')
    const detail = tables[tables.length - 1]!
    await within(detail).findByText('Marine')
    const firstDetail = () => within(detail).getAllByRole('row')[1]?.textContent ?? ''
    await user.click(within(detail).getByRole('button', { name: 'Group Business' }))
    expect(firstDetail()).toContain('Fire')
    await user.click(within(detail).getByRole('button', { name: 'Total Klaim' }))
    expect(firstDetail()).toContain('Fire')
    await user.click(within(detail).getByRole('button', { name: 'OS Value' }))
    expect(firstDetail()).toContain('Fire')
    await user.click(within(detail).getByRole('button', { name: 'Accept Value' }))
    expect(firstDetail()).toContain('Marine')

    // Menutup lalu memilih perjanjian lain menutup rinciannya.
    await user.click(within(table).getByRole('button', { name: 'Tutup rincian' }))
    expect(screen.getAllByRole('table')).toHaveLength(1)

    breakdownFails = true
    await user.click(within(table).getAllByRole('button', { name: 'Lihat rincian' })[1]!)
    expect(await screen.findByText('Rincian tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Rincian rusak.')).toBeInTheDocument()
  })
})

describe('AdvicePanel', () => {
  it('menyatakan penyebab kerugian yang gagal dimuat dan pencarian yang gagal', async () => {
    stub((url) => {
      if (url === `${PATH}/sebab-kerugian`) return { status: 500, body: { kode: 'x', pesan: 'Sebab rusak.' } }
      if (url.startsWith(`${PATH}/pla-dla`)) return { status: 500, body: { kode: 'galat_internal', pesan: 'Pencarian rusak.' } }
      return {}
    })
    const user = userEvent.setup()
    wrap(<AdvicePanel />)

    expect(await screen.findByText('Daftar penyebab kerugian tidak dapat dimuat')).toBeInTheDocument()
    await user.type(screen.getByLabelText('Tahun XOL'), '2024')
    await user.selectOptions(screen.getByLabelText('Tipe Pemberitahuan'), 'DLA')
    await user.click(screen.getByRole('button', { name: 'Cari Data DLA PLA XOL' }))

    expect(await screen.findByText('Pencarian tidak dapat dijalankan')).toBeInTheDocument()
    expect(screen.getByText('Pencarian rusak.')).toBeInTheDocument()
  })

  it('mengurutkan hasil dan menandai yang masih menunggu komite', async () => {
    const advice = (over: Record<string, unknown>) => ({
      nomor: 'PLA/1 / 0',
      nomor_asli: 'PLA/1',
      revisi: '0',
      nama_insurance: 'Reas B',
      nama_layer: 'Layer 2',
      tahun: '2024',
      sebab_kerugian: 'BANJIR',
      kurs: 15500,
      share_percent: 10,
      email: 'b@contoh.invalid',
      remark: 'R2',
      status_persetujuan: '0',
      sudah_disetujui: false,
      catatan_persetujuan: '',
      catatan_pic: '',
      batas_layer: '',
      negara: '',
      tanggal_terbit: '',
      tipe: 'PLA',
      ...over,
    })
    stub((url) => {
      if (url === `${PATH}/sebab-kerugian`) return { body: { sebab_kerugian: [{ id: '1', deskripsi: 'BANJIR' }] } }
      if (url.startsWith(`${PATH}/pla-dla`)) {
        return {
          body: {
            pemberitahuan: [
              advice({}),
              advice({ nomor: 'PLA/0 / 0', nomor_asli: 'PLA/0', nama_insurance: 'Reas A', nama_layer: 'Layer 1', kurs: 100, share_percent: 5, email: 'a@contoh.invalid', remark: 'R1', tahun: '2023' }),
            ],
          },
        }
      }
      return {}
    })
    const user = userEvent.setup()
    wrap(<AdvicePanel />)

    await screen.findByRole('option', { name: 'BANJIR' })
    await user.type(screen.getByLabelText('Tahun XOL'), '2024')
    await user.selectOptions(screen.getByLabelText('Penyebab Kerugian'), 'BANJIR')
    await user.selectOptions(screen.getByLabelText('Tipe Pemberitahuan'), 'PLA')
    await user.click(screen.getByRole('button', { name: 'Cari Data DLA PLA XOL' }))

    const table = await screen.findByRole('table')
    expect(within(table).getAllByText('Menunggu komite')).toHaveLength(2)
    const first = () => within(table).getAllByRole('row')[1]?.textContent ?? ''
    for (const title of ['NO PLA / DLA', 'Nama Insurance', 'Nama Layer', 'Tahun', 'Kurs (IDR)', 'Share Percent', 'Email', 'Remark']) {
      await user.click(within(table).getByRole('button', { name: title }))
      expect(first()).toContain('Reas A')
    }
    expect(screen.getByRole('link', { name: 'Unduh perhitungan (CSV)' })).toBeInTheDocument()
  })
})

describe('ApprovalPanel', () => {
  it('menyatakan antrean persetujuan yang gagal dimuat', async () => {
    stub(() => ({ fail: true }))
    wrap(<ApprovalPanel active />)

    expect((await screen.findAllByText('Antrean persetujuan tidak dapat dimuat')).length).toBeGreaterThan(0)
  })

  it('mengurutkan perjanjian menurut kurs', async () => {
    stub(() => ({
      body: {
        pemberitahuan: [],
        perjanjian: [
          { ...MASTER, id: 'XOL-9', kurs: 20000, tahun: '2025' },
          { ...MASTER, id: 'XOL-8', kurs: 10000, tahun: '2024' },
        ],
      },
    }))
    const user = userEvent.setup()
    wrap(<ApprovalPanel active />)

    await screen.findByText('XOL-9')
    const kurs = screen.getByRole('button', { name: 'Kurs Value' })
    const table = kurs.closest('table') as HTMLElement
    await user.click(kurs)
    await waitFor(() => expect(within(table).getAllByRole('row')[1]).toHaveTextContent('XOL-8'))
  })
})
