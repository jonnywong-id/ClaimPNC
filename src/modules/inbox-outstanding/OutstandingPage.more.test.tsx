import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, renderHook, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { type ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { useOutstandingList, useOutstandingSummary } from './api'
import { isOpenableHere, OutstandingPage } from './OutstandingPage'
import type { OutstandingClaim, OutstandingSummaryResponse } from './types'


/** Seluruh data KARANGAN (`D-69`). */
function claim(partial: Partial<OutstandingClaim> = {}): OutstandingClaim {
  return {
    klaim_id: 'KLM-0001',
    nomor_klaim: 'PNCN.26.0001',
    nomor_polis: 'POL-FIRE-0001',
    nama_tertanggung: 'PT Bumi Contoh Sentosa',
    nama_bisnis: 'Fire',
    sumber_bisnis: 'Broker',
    nama_cabang: 'JKT',
    group_panel: '006',
    tanggal_pendaftaran: '2026-09-16',
    tanggal_kejadian: '2026-09-08',
    tanggal_lapor: '2026-09-10',
    status_proses: 'New',
    status_tampil: 'On Progress',
    status_klaim: '1147',
    pic_teknik: 'BUDISANTOSO',
    admin_pnc: 'ADMINPNC',
    umur_hari: 4,
    aging_hari: 2,
    posisi_klaim: 'On Progress',
    tahap_kini: 'Input Estimasi',
    pemegang_tugas: 'BUDISANTOSO',
    ...partial,
  }
}

// Urutannya mengikuti `statusOrder` di backend — "ALL Case" paling depan.
const SUMMARY: OutstandingSummaryResponse = {
  status: [
    { kode: 'semua', judul: 'ALL Case', jumlah: 5, dapat_dipilih: true },
    { kode: 'lengkap', judul: 'Complete documents', jumlah: 3, dapat_dipilih: true },
    { kode: 'belum-lengkap', judul: 'Documents not complete', jumlah: 2, dapat_dipilih: true },
    { kode: 'loss-adjuster', judul: 'Loss Adjuster', jumlah: null, dapat_dipilih: false },
  ],
  total: 5,
  pemilik: 'ADMINPNC',
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

type Answer = (url: string) => Response | Promise<Response> | undefined

function stubFetch(answer: Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    const custom = answer(url)
    if (custom) return Promise.resolve(custom)
    if (url.includes('/ringkasan')) return Promise.resolve(json(200, SUMMARY))
    return Promise.resolve(json(200, { klaim: [claim()], total: 1, pemilik: 'ADMINPNC' }))
  })
}

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <OutstandingPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-09-20T12:00:00Z' })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

function listCalls() {
  return calls.filter((c) => c.url.startsWith('/api/inbox-outstanding?'))
}

describe('isOpenableHere', () => {
  it('hanya membuka klaim bernomor PNCN di aplikasi ini', () => {
    expect(isOpenableHere('PNCN.26.0001')).toBe(true)
    expect(isOpenableHere('PNC-1865')).toBe(false)
    expect(isOpenableHere('')).toBe(false)
  })
})

describe('kolom', () => {
  it('menandai isian kosong, klaim Pega, umur lama, dan nada status', async () => {
    stubFetch((url) =>
      url.includes('/ringkasan')
        ? undefined
        : json(200, {
            klaim: [
              claim({
                klaim_id: 'A',
                nomor_klaim: 'PNC-1865',
                nomor_polis: '',
                nama_tertanggung: '',
                nama_bisnis: '',
                sumber_bisnis: '',
                nama_cabang: '',
                admin_pnc: '',
                pic_teknik: '',
                tanggal_pendaftaran: '',
                tanggal_kejadian: '',
                umur_hari: 45,
                status_tampil: 'Reject',
                status_klaim: '',
              }),
              claim({ klaim_id: 'B', nomor_klaim: 'PNCN.26.0002', status_tampil: 'Close' }),
            ],
            total: 2,
            pemilik: 'ADMINPNC',
          }),
    )
    renderPage()

    const pega = await screen.findByText('PNC-1865')
    expect(pega).toHaveAttribute('title', 'Pega claim — open it in Pega')
    expect(pega.closest('a')).toBeNull()
    expect(screen.getByRole('link', { name: 'PNCN.26.0002' })).toHaveAttribute(
      'href',
      '/registrasi/klaim/PNCN.26.0002',
    )

    const rowA = pega.closest('tr')!
    // Sembilan isian kosong + Status ASM kosong ditampilkan sebagai tanda pisah.
    expect(within(rowA).getAllByText('—').length).toBeGreaterThanOrEqual(10)
    expect(within(rowA).getByText('45 hari')).toHaveAttribute('title', 'Berjalan 30 hari atau lebih')
    expect(within(rowA).getByText('Reject')).toHaveClass('bg-red-50')
    expect(screen.getByText('Close')).toHaveClass('bg-slate-100')
    expect(screen.getByText('2 klaim masih berjalan.')).toBeInTheDocument()
  })
})

describe('paginasi dan pencarian', () => {
  it('maju dan mundur satu halaman lalu kembali ke awal saat mencari', async () => {
    stubFetch((url) =>
      url.includes('/ringkasan')
        ? undefined
        : json(200, { klaim: [claim()], total: 60, pemilik: 'ADMINPNC' }),
    )
    renderPage()

    expect(await screen.findByText('Menampilkan 1–1 dari 60')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Sebelumnya' })).toBeDisabled()

    await userEvent.click(screen.getByRole('button', { name: 'Berikutnya' }))
    expect(await screen.findByText('Menampilkan 26–26 dari 60')).toBeInTheDocument()
    expect(listCalls().some((c) => c.url.includes('lewati=25'))).toBe(true)

    await userEvent.click(screen.getByRole('button', { name: 'Sebelumnya' }))
    expect(await screen.findByText('Menampilkan 1–1 dari 60')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Berikutnya' }))
    await screen.findByText('Menampilkan 26–26 dari 60')
    await userEvent.type(screen.getByRole('searchbox'), 'PNCN')
    expect(await screen.findByText('Menampilkan 1–1 dari 60')).toBeInTheDocument()
    await waitFor(() =>
      expect(
        calls.some((c) => c.url.includes('/ringkasan?cari=PNCN')),
      ).toBe(true),
    )
  })

  it('memuat ulang daftar lewat tombol Muat ulang', async () => {
    stubFetch()
    renderPage()
    await screen.findByText('PNCN.26.0001')
    const before = listCalls().length

    await userEvent.click(screen.getByRole('button', { name: /Muat ulang/ }))

    await waitFor(() => expect(listCalls().length).toBe(before + 1))
  })
})

describe('unduhan', () => {
  function stubURL() {
    const revoke = vi.fn()
    vi.stubGlobal('URL', { ...URL, createObjectURL: vi.fn(() => 'blob:uji'), revokeObjectURL: revoke })
    return revoke
  }

  it('memakai nama berkas bawaan bila server tidak menyebutnya', async () => {
    const revoke = stubURL()
    const clicked: string[] = []
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(function (this: HTMLAnchorElement) {
        clicked.push(this.download)
      })
    stubFetch((url) =>
      url.includes('/unduh') ? new Response('a\n', { status: 200 }) : undefined,
    )
    renderPage()
    await screen.findByText('PNCN.26.0001')

    await userEvent.click(screen.getByRole('button', { name: /Unduh CSV/ }))

    await waitFor(() => expect(clicked).toEqual(['inbox-outstanding.csv']))
    expect(revoke).toHaveBeenCalledWith('blob:uji')
    expect(await screen.findByRole('button', { name: /Unduh CSV/ })).toBeEnabled()
    click.mockRestore()
  })

  it('menampilkan galat bila server menolak unduhan', async () => {
    stubFetch((url) => (url.includes('/unduh') ? new Response('', { status: 500 }) : undefined))
    renderPage()
    await screen.findByText('PNCN.26.0001')

    await userEvent.click(screen.getByRole('button', { name: /Unduh CSV/ }))

    expect(await screen.findByText('Berkas tidak dapat diunduh')).toBeInTheDocument()
    expect(screen.getByText('Berkas tidak dapat diunduh.')).toBeInTheDocument()
  })

  it('menampilkan pesan umum bila yang dilempar bukan Error', async () => {
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, init })
      // eslint-disable-next-line @typescript-eslint/prefer-promise-reject-errors
      if (url.includes('/unduh')) return Promise.reject('bukan galat')
      if (url.includes('/ringkasan')) return Promise.resolve(json(200, SUMMARY))
      return Promise.resolve(json(200, { klaim: [claim()], total: 1, pemilik: 'ADMINPNC' }))
    })
    renderPage()
    await screen.findByText('PNCN.26.0001')

    await userEvent.click(screen.getByRole('button', { name: /Unduh CSV/ }))

    expect(await screen.findByText('Terjadi kesalahan pada sistem.')).toBeInTheDocument()
  })

  it('tombol unduh mati ketika portal belum dipilih', async () => {
    useSelectedPortal.setState({ alias: null })
    stubFetch()
    renderPage()

    expect(screen.getByRole('button', { name: /Unduh CSV/ })).toBeDisabled()
    expect(screen.getByRole('button', { name: /Muat ulang/ })).toBeDisabled()
  })
})

describe('daftar gagal', () => {
  it('menampilkan pesan dari galat yang bukan galat API', async () => {
    stubFetch((url) =>
      url.includes('/ringkasan')
        ? undefined
        : ({
            get status(): number {
              throw new TypeError('jawaban rusak')
            },
          } as unknown as Response),
    )
    renderPage()

    expect(await screen.findByText('Daftar klaim tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('jawaban rusak')).toBeInTheDocument()
  })
})

describe('ringkasan', () => {
  it('menampilkan kerangka memuat lalu galat jaringan', async () => {
    let release: (r: Response) => void = () => {}
    stubFetch((url) =>
      url.includes('/ringkasan')
        ? new Promise<Response>((resolve) => {
            release = resolve
          })
        : undefined,
    )
    renderPage()

    expect(screen.getByLabelText('Memuat status dokumen')).toBeInTheDocument()
    release(json(500, { kode: 'galat_internal', pesan: 'x' }))
    expect(await screen.findByText('Status dokumen tidak dapat dimuat')).toBeInTheDocument()
    expect(
      screen.getByText('Daftar klaim di bawah tetap dapat dipakai. Coba muat ulang halaman ini.'),
    ).toBeInTheDocument()
  })

  it('menyebut server tidak terhubung pada galat jaringan', async () => {
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push({ url, init })
      if (url.includes('/ringkasan')) return Promise.reject(new TypeError('putus'))
      return Promise.resolve(json(200, { klaim: [claim()], total: 1, pemilik: 'ADMINPNC' }))
    })
    renderPage()

    expect(await screen.findByText('Server Claim PNC tidak dapat dihubungi.')).toBeInTheDocument()
  })

  it('bertahan pada respons tanpa status dan total', async () => {
    stubFetch((url) => (url.includes('/ringkasan') ? json(200, { pemilik: 'X' }) : undefined))
    renderPage()

    // Deret tabnya tidak digambar, dan layar TETAP berdiri — daftar klaim di bawahnya
    // yang menjadi isinya. Respons cacat hanya boleh menghilangkan tabnya, bukan
    // menjatuhkan seluruh halaman.
    expect(await screen.findByText('PNCN.26.0001')).toBeInTheDocument()
    expect(screen.queryByRole('tablist', { name: 'Status dokumen' })).not.toBeInTheDocument()
  })

  it('menggambar tab meski hanya sebagian status yang terhitung', async () => {
    stubFetch((url) =>
      url.includes('/ringkasan')
        ? json(200, {
            status: [
              { kode: 'lengkap', judul: 'Complete documents', jumlah: 0, dapat_dipilih: true },
              { kode: 'semua', judul: 'ALL Case', jumlah: 4, dapat_dipilih: true },
            ],
            total: 4,
            pemilik: 'X',
          })
        : undefined,
    )
    renderPage()

    const tabs = await screen.findByRole('tablist', { name: 'Status dokumen' })
    expect(within(tabs).getByRole('tab', { name: 'Complete documents, 0 klaim' })).toBeInTheDocument()
    expect(within(tabs).getByRole('tab', { name: 'ALL Case, 4 klaim' })).toBeInTheDocument()
  })

  it('memilih dan membatalkan status lewat tab, termasuk lewat ALL Case', async () => {
    stubFetch()
    renderPage()
    const tabs = await screen.findByRole('tablist', { name: 'Status dokumen' })
    const all = within(tabs).getByRole('tab', { name: 'ALL Case, 5 klaim' })
    expect(all).toHaveAttribute('aria-selected', 'true')

    const complete = within(tabs).getByRole('tab', { name: 'Complete documents, 3 klaim' })
    await userEvent.click(complete)
    await waitFor(() => expect(complete).toHaveAttribute('aria-selected', 'true'))
    expect(all).toHaveAttribute('aria-selected', 'false')
    expect(listCalls().some((c) => c.url.includes('status_dokumen=lengkap'))).toBe(true)

    // Menekan tab yang SEDANG aktif tidak membatalkannya.
    //
    // Versi pertama layar ini memperlakukannya sebagai sakelar, dan itu dibuang saat
    // bentuknya mengikuti `SourceTab` Inbox Auto Claim: pada tablist, satu tab SELALU
    // terpilih. Tab yang membatalkan pilihan sudah ada namanya sendiri — "ALL Case" —
    // dan klik yang diam-diam melompat ke tab lain justru yang mengagetkan.
    await userEvent.click(complete)
    await waitFor(() => expect(complete).toHaveAttribute('aria-selected', 'true'))
    expect(all).toHaveAttribute('aria-selected', 'false')

    // "ALL Case" itulah yang mengembalikan daftar ke seluruh status.
    await userEvent.click(within(tabs).getByRole('tab', { name: 'Documents not complete, 2 klaim' }))
    await userEvent.click(all)
    await waitFor(() => expect(all).toHaveAttribute('aria-selected', 'true'))
  })

  // Panah kiri/kanan berpindah tab, dan MELEWATI tab yang belum tersedia.
  //
  // Tanpa itu, pengguna papan ketik berhenti di tab mati yang tidak dapat dibuka dan
  // kehilangan cara meneruskannya.
  it('berpindah tab dengan panah dan melewati tab yang belum tersedia', async () => {
    stubFetch()
    renderPage()
    const tabs = await screen.findByRole('tablist', { name: 'Status dokumen' })
    const all = within(tabs).getByRole('tab', { name: 'ALL Case, 5 klaim' })

    all.focus()
    await userEvent.keyboard('{ArrowRight}')

    // "ALL Case" tab pertama, jadi panah kanan menuju tab berikutnya yang dapat dipilih:
    // "Complete documents". "Loss Adjuster" yang belum tersedia tidak pernah jadi tujuan.
    await waitFor(() =>
      expect(within(tabs).getByRole('tab', { name: 'Complete documents, 3 klaim' })).toHaveAttribute(
        'aria-selected',
        'true',
      ),
    )
    expect(within(tabs).getByRole('tab', { name: 'Loss Adjuster, belum tersedia' })).toBeDisabled()
  })

})

describe('hook', () => {
  function wrapper({ children }: { children: ReactNode }) {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    return <QueryClientProvider client={client}>{children}</QueryClientProvider>
  }

  it('mengirim penyaring tahap dan cabang yang dipangkas ke daftar dan ringkasan', async () => {
    stubFetch()
    const filter = { stage: ' estimasi ', branch: ' JKT ', search: '  ' }
    const list = renderHook(() => useOutstandingList(filter), { wrapper })
    const summary = renderHook(() => useOutstandingSummary(filter), { wrapper })

    await waitFor(() => expect(list.result.current.isSuccess).toBe(true))
    await waitFor(() => expect(summary.result.current.isSuccess).toBe(true))
    expect(calls.map((c) => c.url)).toEqual(
      expect.arrayContaining([
        '/api/inbox-outstanding?tahap=estimasi&cabang=JKT&batas=25',
        '/api/inbox-outstanding/ringkasan?tahap=estimasi&cabang=JKT',
      ]),
    )
  })
})

describe('tanpa sesi', () => {
  it('tidak mengunduh apa pun bila sesi tidak ada', async () => {
    useSession.setState({ token: null })
    stubFetch()
    renderPage()

    await userEvent.click(screen.getByRole('button', { name: /Unduh CSV/ }))

    expect(calls.some((c) => c.url.includes('/unduh'))).toBe(false)
    expect(screen.getByRole('button', { name: /Unduh CSV/ })).toBeEnabled()
  })
})
