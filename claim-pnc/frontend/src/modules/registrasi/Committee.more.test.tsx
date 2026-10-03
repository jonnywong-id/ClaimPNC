import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { CommitteeInbox, CommitteeStatus, TransferCommitteeButton } from './Committee'
import { StagePath } from './StagePath'
import type { CommitteeItem, Settlement } from './types'

/** Uji komponen komite dan jalur tahapan, dirender langsung. Seluruh data KARANGAN (`D-69`). */

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
    return Promise.resolve(json(404, { kode: 'tidak_ditemukan', pesan: 'Tidak ditemukan.' }))
  })
}

function wrap(children: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>{children}</MemoryRouter>
    </QueryClientProvider>,
  )
}

function line(extra: Partial<Settlement> = {}): Settlement {
  return {
    tipe_pembayaran: '1',
    nama_tipe_pembayaran: 'Final',
    mata_uang: 'IDR',
    kurs_e4: 10000,
    nilai_propose_sen: 0,
    nilai_pengajuan_sen: 0,
    loc: 0,
    nilai_salvage_sen: 0,
    nilai_salvage_b_sen: 0,
    nilai_interim_sen: 0,
    nilai_estimasi_sen: 0,
    tipe_resiko: '',
    persen_resiko: 0,
    nilai_resiko_sen: 0,
    nilai_gross_sen: 0,
    share_asm: 0,
    nilai_asm_sen: 0,
    nilai_akseptasi_sen: 0,
    kronologi: '',
    catatan: '',
    status_akseptasi: '',
    nomor_akseptasi: '',
    ...extra,
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

describe('TransferCommitteeButton', () => {
  function button(extra: Partial<Settlement> = {}, lockedReason: string | null = null) {
    return (
      <TransferCommitteeButton
        claimID="klaim-1"
        taskID="tugas-1"
        object={1}
        coverage={2}
        adjustment={3}
        line={line(extra)}
        lockedReason={lockedReason}
      />
    )
  }

  it('mengirim alamat baris dan menampilkan pelanggaran validasi', async () => {
    installFetch((url) =>
      url === '/api/registrasi/klaim/klaim-1/adjustment/komite'
        ? json(422, {
            kode: 'validasi_gagal',
            pesan: 'x',
            detail: [
              { kode: 'a', field: '', pesan: 'Nilai di bawah ambang.' },
              { kode: 'b', field: '', pesan: 'Rekening kosong.' },
            ],
          })
        : undefined,
    )
    wrap(button())

    await userEvent.click(screen.getByRole('button', { name: 'Transfer Komite' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Nilai di bawah ambang. Rekening kosong.')
    expect(calls[0]?.body).toEqual({ tugas_id: 'tugas-1', objek: 1, jaminan: 2, adjustment: 3 })
  })

  it.each([
    { name: 'galat API', answer: (): Answer => json(500, { kode: 'x', pesan: 'Komite sibuk.' }), text: 'Komite sibuk.' },
    { name: 'jaringan putus', answer: (): Answer => 'putus', text: 'Tidak dapat menghubungi server Claim PNC.' },
  ])('menampilkan galat transfer: $name', async ({ answer, text }) => {
    installFetch((url) => (url.endsWith('/adjustment/komite') ? answer() : undefined))
    wrap(button())

    await userEvent.click(screen.getByRole('button', { name: 'Transfer Komite' }))
    expect(await screen.findByRole('alert')).toHaveTextContent(text)
  })

  it('mematikan tombol untuk baris yang sudah ditransfer atau dikunci', () => {
    installFetch(() => undefined)
    const { unmount } = wrap(button({ komite_id: 'KMT-1' }))
    expect(screen.getByRole('button', { name: 'Transfer Komite' })).toHaveAttribute(
      'title',
      'Sudah ditransfer ke komite.',
    )
    expect(screen.getByRole('button', { name: 'Transfer Komite' })).toBeDisabled()
    unmount()

    wrap(button({}, 'Simpan baris lebih dulu.'))
    expect(screen.getByRole('button', { name: 'Transfer Komite' })).toHaveAttribute('title', 'Simpan baris lebih dulu.')
    expect(screen.getByRole('button', { name: 'Transfer Komite' })).toBeDisabled()
  })
})

describe('CommitteeStatus', () => {
  it('menyebut baris yang belum ditransfer', () => {
    installFetch(() => undefined)
    wrap(<CommitteeStatus line={line()} />)
    expect(screen.getByText('Belum ditransfer')).toBeInTheDocument()
  })

  it('menyebut nomor komite selama datanya belum ada', () => {
    installFetch(() => new Promise<Response>(() => {}))
    wrap(<CommitteeStatus line={line({ komite_id: 'KMT-1' })} />)
    expect(screen.getByText('Komite KMT-1')).toBeInTheDocument()
  })

  it.each([
    {
      name: 'disetujui dengan nomor akseptasi',
      status: 'disetujui',
      extra: { nomor_akseptasi: 'AKS-9' },
      text: 'Diakseptasi komite · AKS-9 · KMT-1',
    },
    { name: 'disetujui tanpa nomor', status: 'disetujui', extra: {}, text: 'Diakseptasi komite · KMT-1' },
    { name: 'ditolak', status: 'ditolak', extra: {}, text: 'Ditolak komite · KMT-1' },
  ])('menyebut komite yang $name', async ({ status, extra, text }) => {
    installFetch((url) =>
      url === '/api/registrasi/komite/KMT-1'
        ? json(200, { id: 'KMT-1', nomor_klaim: 'PNCN.26.1', status, anggota: [] })
        : undefined,
    )
    wrap(<CommitteeStatus line={line({ komite_id: 'KMT-1', ...extra })} />)
    expect(await screen.findByText(text)).toBeInTheDocument()
  })

  it('menyebut jenjang yang sedang menunggu beserta rinciannya', async () => {
    installFetch((url) =>
      url === '/api/registrasi/komite/KMT-1'
        ? json(200, {
            id: 'KMT-1',
            nomor_klaim: 'PNCN.26.1',
            status: 'berjalan',
            menunggu: 'KOMITE2',
            anggota: [
              { jenjang: 1, komite: 'KOMITE1', keputusan: '1', catatan: '' },
              { jenjang: 2, komite: 'KOMITE2', keputusan: '0', catatan: '' },
              { jenjang: 3, komite: 'KOMITE3', keputusan: '9', catatan: '' },
            ],
          })
        : undefined,
    )
    wrap(<CommitteeStatus line={line({ komite_id: 'KMT-1' })} />)

    const status = await screen.findByText('Komite KMT-1 · jenjang 2/3 menunggu KOMITE2')
    expect(status).toHaveAttribute('title', '1. KOMITE1: setuju\n2. KOMITE2: menunggu\n3. KOMITE3: 9')
  })

  it('memakai tanda hubung bila tidak ada jenjang yang menunggu', async () => {
    installFetch((url) =>
      url === '/api/registrasi/komite/KMT-1'
        ? json(200, {
            id: 'KMT-1',
            nomor_klaim: 'PNCN.26.1',
            status: 'berjalan',
            anggota: [{ jenjang: 1, komite: 'KOMITE1', keputusan: '2', catatan: '' }],
          })
        : undefined,
    )
    wrap(<CommitteeStatus line={line({ komite_id: 'KMT-1' })} />)

    expect(await screen.findByText('Komite KMT-1 · jenjang —/1 menunggu —')).toHaveAttribute(
      'title',
      '1. KOMITE1: tolak',
    )
  })
})

function item(extra: Partial<CommitteeItem> = {}): CommitteeItem {
  return {
    komite_id: 'KMT-1',
    jenjang: 1,
    jumlah_jenjang: 2,
    klaim_id: 'klaim-1',
    nomor_klaim: 'PNCN.26.1',
    nomor_polis: 'POL-1',
    nama_tertanggung: 'PT Contoh',
    nama_objek: 'Gudang',
    nama_coverage: 'All Risk',
    adjustment: 1,
    nama_tipe_pembayaran: 'Final',
    mata_uang: 'IDR',
    nilai_asm_sen: 150000,
    nilai_komite_sen: 250000,
    tanggal_transfer: '2026-05-01',
    ...extra,
  }
}

describe('CommitteeInbox', () => {
  it('menyebut memuat lalu daftar kosong', async () => {
    installFetch((url) => (url === '/api/registrasi/komite' ? json(200, { komite: [] }) : undefined))
    wrap(<CommitteeInbox />)

    expect(screen.getByText('Memuat putusan komite…')).toBeInTheDocument()
    expect(await screen.findByText('Tidak ada putusan komite yang menunggu Anda.')).toBeInTheDocument()
  })

  it('menampilkan galat daftar', async () => {
    installFetch((url) =>
      url === '/api/registrasi/komite' ? json(500, { kode: 'x', pesan: 'Daftar rusak.' }) : undefined,
    )
    wrap(<CommitteeInbox />)

    expect(await screen.findByText('Daftar komite tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Daftar rusak.')).toBeInTheDocument()
  })

  it('menampilkan baris, mengirim putusan setuju dan tolak, serta galatnya', async () => {
    let fail = false
    installFetch((url, method) => {
      if (url === '/api/registrasi/komite' && method === 'GET') {
        return json(200, {
          komite: [
            item(),
            item({
              komite_id: 'KMT-2',
              klaim_id: '',
              nomor_klaim: 'PNC-77',
              jumlah_jenjang: 0,
              nama_tertanggung: '',
              nama_objek: '',
              nama_coverage: '',
              adjustment: 0,
            }),
          ],
        })
      }
      if (url.endsWith('/putusan')) {
        return fail ? json(409, { kode: 'x', pesan: 'Sudah diputus.' }) : json(200, { id: 'KMT-1', nomor_klaim: '', status: 'berjalan', anggota: [] })
      }
      return undefined
    })
    wrap(<CommitteeInbox />)

    const first = await screen.findByRole('group', { name: 'Komite KMT-1' })
    expect(within(first).getByRole('link', { name: 'PNCN.26.1' })).toHaveAttribute('href', '/registrasi/klaim/klaim-1')
    expect(first).toHaveTextContent('jenjang 1/2')
    expect(first).toHaveTextContent('PT Contoh · Gudang / All Risk · Adjustment 1')
    expect(first).toHaveTextContent(/IDR 1\.500,00 · Nilai komite IDR\s+2\.500,00/)

    const second = screen.getByRole('group', { name: 'Komite KMT-2' })
    expect(within(second).queryByRole('link')).not.toBeInTheDocument()
    expect(second).toHaveTextContent('jenjang 1/—')
    expect(second).toHaveTextContent('— · — / — · Adjustment —')

    await userEvent.type(within(first).getByLabelText('Catatan Komite'), 'Layak')
    await userEvent.click(within(first).getByRole('button', { name: 'Setuju' }))
    await waitFor(() =>
      expect(calls.find((c) => c.url === '/api/registrasi/komite/KMT-1/putusan')?.body).toEqual({
        keputusan: '1',
        catatan: 'Layak',
      }),
    )

    fail = true
    await userEvent.click(within(second).getByRole('button', { name: 'Tolak' }))
    expect(await within(second).findByRole('alert')).toHaveTextContent('Sudah diputus.')
    expect(calls.find((c) => c.url === '/api/registrasi/komite/KMT-2/putusan')?.body).toEqual({
      keputusan: '2',
      catatan: '',
    })
  })
})

describe('StagePath', () => {
  it('tidak menggambar apa pun tanpa jalur, dan memakai id bila nama tahap tidak dikenal', () => {
    const { container, rerender } = render(<StagePath tahap={[]} jalur={[]} currentStage="" />)
    expect(container).toBeEmptyDOMElement()

    rerender(
      <StagePath
        tahap={[{ id: 'a', nama: 'Tahap A', antrean: 'WORKLIST', workbasket: '', router: '', tindakan_keluar: '', pega_id: '' }]}
        jalur={['a', 'tak-dikenal']}
        currentStage="tak-dikenal"
      />,
    )
    expect(screen.getByText('tak-dikenal')).toHaveAttribute('aria-current', 'step')
    expect(screen.getByText('Tahap A')).not.toHaveAttribute('aria-current')
  })
})
