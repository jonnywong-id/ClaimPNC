import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { CloseClaimPage } from './CloseClaimPage'
import { RequestDialog } from './RequestDialog'
import type { DaftarResponse, KlaimTutup } from './types'

/**
 * Uji tambahan Inbox Close Claim: unduhan CSV, penyaring teks, galat, dan dialog.
 * Seluruh nilai KARANGAN (`D-69`).
 */

function klaim(partial: Partial<KlaimTutup> = {}): KlaimTutup {
  return {
    klaim_id: 'K-1',
    nomor_klaim: 'PNCN.26.0001',
    nomor_polis: 'POL-0001',
    nama_tertanggung: 'PT Contoh',
    nama_bisnis: 'Fire',
    sumber_bisnis: 'Broker',
    nama_cabang: 'JKT',
    pic_teknik: 'PIC',
    admin_pnc: 'ADM',
    tanggal_pendaftaran: '2026-01-12',
    tanggal_kejadian: '2026-01-07',
    tanggal_tutup: '2026-02-22',
    lama_hari: 41,
    status_proses: 'Resolved-Completed',
    status_tampil: 'Close',
    status_klaim_kode: '1163',
    status_klaim_label: 'Paid',
    sudah_transfer: true,
    permintaan_tertunda: [],
    ...partial,
  }
}

function daftar(partial: Partial<DaftarResponse> = {}): DaftarResponse {
  return {
    klaim: [klaim()],
    total: 1,
    permintaan_terbaca: true,
    boleh_mengajukan: true,
    pelaksana_belum_ada: false,
    ...partial,
  }
}

type Call = { url: string; init: RequestInit | undefined }
let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function stubFetch(
  list: () => Response | Promise<Response>,
  extra: (url: string, init?: RequestInit) => Response | Promise<Response> | null = () => null,
) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    const special = extra(url, init)
    if (special) return Promise.resolve(special)
    if (url.includes('/penyaring')) {
      return Promise.resolve(jsonResponse(200, { lini_bisnis: [], status_transfer: [], status_bayar: [] }))
    }
    return Promise.resolve(list())
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <CloseClaimPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

let clicked: { href: string; download: string }[] = []
const revoke = vi.fn()

beforeEach(() => {
  calls = []
  clicked = []
  revoke.mockReset()
  Object.assign(URL, { createObjectURL: vi.fn(() => 'blob:csv'), revokeObjectURL: revoke })
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(function (
    this: HTMLAnchorElement,
  ) {
    clicked.push({ href: this.href, download: this.download })
  })
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-09-24T12:00:00Z' })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('Unduh CSV', () => {
  it('mengunduh dengan penyaring dan nama dari server, lalu melepas URL objeknya', async () => {
    stubFetch(
      () => jsonResponse(200, daftar()),
      (url) =>
        url.includes('/unduh')
          ? new Response('a,b', {
              status: 200,
              headers: { 'Content-Disposition': 'attachment; filename="tutup.csv"' },
            })
          : null,
    )
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('PNCN.26.0001')

    await user.type(screen.getByLabelText('No Polis'), ' POL-1 ')
    await waitFor(() => expect(calls.some((c) => c.url.includes('no_polis=POL-1'))).toBe(true))
    await user.click(screen.getByRole('button', { name: 'Unduh CSV' }))

    await waitFor(() => expect(clicked).toEqual([{ href: 'blob:csv', download: 'tutup.csv' }]))
    const request = calls.find((c) => c.url.includes('/unduh'))
    expect(request?.url).toBe('/api/inbox-close-claim/unduh?no_polis=POL-1')
    const header = request?.init?.headers as Record<string, string>
    expect(header['Authorization']).toBe('Bearer token-uji')
    expect(header[HEADER_PORTAL]).toBe('ASM')
    expect(revoke).toHaveBeenCalledWith('blob:csv')
    expect(screen.getByRole('button', { name: 'Unduh CSV' })).not.toBeDisabled()
  })

  it('memakai nama cadangan tanpa penyaring bila server tidak menyebut nama', async () => {
    stubFetch(
      () => jsonResponse(200, daftar()),
      (url) => (url.includes('/unduh') ? new Response('x', { status: 200 }) : null),
    )
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('PNCN.26.0001')

    await user.click(screen.getByRole('button', { name: 'Unduh CSV' }))
    await waitFor(() => expect(clicked[0]?.download).toBe('inbox-close-claim.csv'))
    expect(calls.find((c) => c.url.includes('/unduh'))?.url).toBe('/api/inbox-close-claim/unduh')
  })

  it('memakai nama cadangan bila Content-Disposition tidak bernama', async () => {
    stubFetch(
      () => jsonResponse(200, daftar()),
      (url) =>
        url.includes('/unduh')
          ? new Response('x', { status: 200, headers: { 'Content-Disposition': 'attachment' } })
          : null,
    )
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('PNCN.26.0001')

    await user.click(screen.getByRole('button', { name: 'Unduh CSV' }))
    await waitFor(() => expect(clicked[0]?.download).toBe('inbox-close-claim.csv'))
  })

  it('menyebut berkas tidak dapat diunduh saat server menolak', async () => {
    stubFetch(
      () => jsonResponse(200, daftar()),
      (url) => (url.includes('/unduh') ? new Response('', { status: 500 }) : null),
    )
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('PNCN.26.0001')

    await user.click(screen.getByRole('button', { name: 'Unduh CSV' }))
    expect(await screen.findByText('Berkas tidak dapat diunduh')).toBeInTheDocument()
    expect(screen.getByText('Berkas tidak dapat diunduh.')).toBeInTheDocument()
    expect(clicked).toEqual([])
  })

  it('memakai pesan umum bila yang gagal bukan Error', async () => {
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, init })
      if (url.includes('/unduh')) return Promise.reject('putus')
      return Promise.resolve(jsonResponse(200, daftar()))
    })
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('PNCN.26.0001')

    await user.click(screen.getByRole('button', { name: 'Unduh CSV' }))
    expect(await screen.findByText('Terjadi kesalahan pada sistem.')).toBeInTheDocument()
  })

  it('mematikan unduhan saat tidak ada baris', async () => {
    stubFetch(() => jsonResponse(200, daftar({ klaim: [], total: 0 })))
    renderPage()

    expect(await screen.findByText('Tidak ada klaim tutup yang cocok.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Unduh CSV' })).toBeDisabled()
    // Paginasi tidak digambar sama sekali tanpa baris.
    expect(screen.queryByRole('button', { name: 'Berikutnya' })).not.toBeInTheDocument()
  })
})

describe('daftar dan penyaring', () => {
  it('menampilkan galat pemuatan daftar dengan pesan server', async () => {
    stubFetch(() => jsonResponse(500, { kode: 'galat_internal', pesan: 'Basis data mati.' }))
    renderPage()

    expect(await screen.findByText('Daftar klaim tutup tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Basis data mati.')).toBeInTheDocument()
  })

  it('menulis tanda pisah untuk kolom kosong dan menandai klaim yang ditolak', async () => {
    stubFetch(() =>
      jsonResponse(
        200,
        daftar({
          klaim: [
            klaim({
              nomor_klaim: '',
              nomor_polis: '',
              nama_tertanggung: '',
              nama_bisnis: '',
              sumber_bisnis: '',
              nama_cabang: '',
              tanggal_pendaftaran: '',
              pic_teknik: '',
              admin_pnc: '',
              status_tampil: 'Reject',
              status_klaim_label: '',
            }),
            klaim({ klaim_id: 'K-2', nomor_klaim: 'PNCN.26.0002', status_tampil: '' }),
          ],
          total: 2,
        }),
      ),
    )
    renderPage()

    expect(await screen.findByText('belum bernomor')).toBeInTheDocument()
    expect(screen.getByText('Reject').className).toContain('red')
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(9)
    expect(screen.getByText('2 klaim sudah tutup.')).toBeInTheDocument()
  })

  it('mengirim ketiga penyaring teks, lalu membersihkan seluruhnya', async () => {
    stubFetch(() => jsonResponse(200, daftar()))
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('PNCN.26.0001')

    expect(screen.getByRole('button', { name: 'Bersihkan' })).toBeDisabled()
    await user.type(screen.getByLabelText('No Klaim'), 'K9')
    await user.type(screen.getByLabelText('PIC Teknik'), 'BUDI')
    await user.type(screen.getByLabelText('Cari No Klaim / No Polis'), 'cari')

    await waitFor(() => {
      const last = calls.filter((c) => !c.url.includes('/penyaring')).at(-1)
      expect(last?.url).toContain('no_klaim=K9')
      expect(last?.url).toContain('pic=BUDI')
      expect(last?.url).toContain('cari=cari')
    })

    await user.click(screen.getByRole('button', { name: 'Bersihkan' }))
    await waitFor(() => {
      const last = calls.filter((c) => !c.url.includes('/penyaring')).at(-1)
      expect(last?.url).toBe('/api/inbox-close-claim?batas=25')
    })
    expect(screen.getByLabelText('No Klaim')).toHaveValue('')
  })

  it('maju lalu mundur satu halaman', async () => {
    stubFetch(() => jsonResponse(200, daftar({ total: 60 })))
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('PNCN.26.0001')

    expect(screen.getByRole('button', { name: 'Sebelumnya' })).toBeDisabled()
    await user.click(screen.getByRole('button', { name: 'Berikutnya' }))
    await screen.findByText('Menampilkan 26–26 dari 60')
    await user.click(screen.getByRole('button', { name: 'Sebelumnya' }))
    await screen.findByText('Menampilkan 1–1 dari 60')

    const urls = calls.filter((c) => !c.url.includes('/penyaring')).map((c) => c.url)
    expect(urls).toContain('/api/inbox-close-claim?lewati=25&batas=25')
  })

  it('memuat ulang daftar lewat tombolnya', async () => {
    stubFetch(() => jsonResponse(200, daftar()))
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('PNCN.26.0001')

    const before = calls.length
    await user.click(screen.getByRole('button', { name: /Muat ulang/ }))
    await waitFor(() => expect(calls.length).toBe(before + 1))
  })
})

describe('dialog permintaan', () => {
  it('ditutup dengan Batal maupun Escape tanpa mengirim apa pun', async () => {
    stubFetch(() => jsonResponse(200, daftar()))
    const user = userEvent.setup()
    renderPage()
    await screen.findByText('PNCN.26.0001')

    await user.click(screen.getByRole('button', { name: 'Copy Klaim' }))
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByText('Salin klaim ini menjadi klaim baru?')).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: 'Batal' })).toHaveFocus()
    await user.click(within(dialog).getByRole('button', { name: 'Batal' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'ReOpen' }))
    await screen.findByRole('dialog')
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    expect(calls.some((c) => c.init?.method === 'POST')).toBe(false)
  })

  it('menulis tanda pisah, mengunci isian selagi mengirim, dan mengabaikan Escape', async () => {
    const onBatal = vi.fn()
    const user = userEvent.setup()
    render(
      <RequestDialog
        jenis="reopen"
        klaim={klaim({ nomor_klaim: '', nomor_polis: '', nama_tertanggung: '' })}
        onBatal={onBatal}
        onKirim={() => {}}
        sedangMengirim
        galat="Ditolak server."
      />,
    )

    expect(screen.getAllByText('—')).toHaveLength(3)
    expect(screen.getByRole('button', { name: 'Mengirim…' })).toBeDisabled()
    expect(screen.getByLabelText(/Alasan/)).toBeDisabled()
    expect(screen.getByText('Ditolak server.')).toBeInTheDocument()
    await user.keyboard('{Escape}')
    expect(onBatal).not.toHaveBeenCalled()
  })
})
