import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSession } from '@/app/session'

import { AccountPage } from './AccountPage'

// Seluruh uji di berkas ini berjalan tanpa jaringan: fetch digantikan tiruan yang
// menjawab dari peta jalur di bawah.

type Reply = { status: number; body: unknown }

let reply: Map<string, Reply>
let request: { path: string; metode: string; body: unknown }[]

function installFakeFetch() {
  request = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (path: string, options?: RequestInit) => {
      request.push({
        path,
        metode: options?.method ?? 'GET',
        body: options?.body ? JSON.parse(String(options.body)) : undefined,
      })

      // Kunci TERPANJANG yang cocok, bukan yang pertama ditemukan. `/api/master-rekening`
      // adalah awalan dari `/api/master-rekening/014/.../keputusan`, sehingga pencocokan
      // berdasar urutan akan menjawab POST keputusan dengan badan daftar rekening — dan
      // uji galatnya lulus palsu karena tidak pernah melihat galat yang dipasangnya.
      const key = [...reply.keys()]
        .filter((k) => path.startsWith(k))
        .sort((a, b) => b.length - a.length)[0]
      const result = key ? reply.get(key)! : { status: 404, body: null }
      return new Response(JSON.stringify(result.body), {
        status: result.status,
        headers: { 'Content-Type': 'application/json' },
      })
    }),
  )
}

function wrap(children: React.ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return <QueryClientProvider client={client}>{children}</QueryClientProvider>
}

function sampleAccount(overrides: Record<string, unknown> = {}) {
  return {
    nomor_rekening: '1234567890',
    nama_pemilik: 'BENGKEL CONTOH SEJAHTERA',
    nama_bank: 'BANK CONTOH',
    cabang_bank: 'JAKARTA PUSAT',
    alamat_bank: 'JL. CONTOH NO. 1',
    kode_bank: '014',
    tipe_rekening: 'BIASA',
    aktif: true,
    email: 'keuangan@contoh.co.id',
    telepon: '0211234567',
    nik: '3171000000000000',
    catatan: '',
    id_dokumen: 'DOK-001',
    diinput_oleh: '3171999',
    status: '0',
    status_label: 'Menunggu',
    komite_approval: 'KOMITE-01',
    diinput_pada: '2026-09-17T03:00:00Z',
    status_layanan: '',
    id_rekening_kasir: '',
    respons_kasir: '',
    dapat_dipakai: false,
    ...overrides,
  }
}

beforeEach(() => {
  useSession.setState({
    token: 'token-uji',
    user: null,
    validUntil: '2026-12-31T00:00:00Z',
  })
  reply = new Map([
    ['/api/master-rekening/bank', { status: 200, body: { bank: [{ kode: '014', nama: 'BANK BCA' }] } }],
    ['/api/master-rekening', { status: 200, body: { rekening: [sampleAccount()], jumlah: 1, batas: 50, lewati: 0 } }],
  ])
  installFakeFetch()
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('AccountPage', () => {
  it('menampilkan kelima tab layar lama', async () => {
    render(wrap(<AccountPage />))

    for (const label of [
      'Cari Data Rekening',
      'Antrean Komite Saya',
      'Menunggu Approval',
      'Sudah Disetujui',
      'Sudah Ditolak',
    ]) {
      expect(screen.getByRole('button', { name: label })).toBeInTheDocument()
    }
  })

  it('menampilkan rekening beserta status persetujuannya', async () => {
    render(wrap(<AccountPage />))

    expect(await screen.findByText('1234567890')).toBeInTheDocument()
    expect(screen.getByText('BENGKEL CONTOH SEJAHTERA')).toBeInTheDocument()
    expect(screen.getByText('Menunggu')).toBeInTheDocument()
  })

  it('menandai rekening yang disetujui tetapi sudah dinonaktifkan', async () => {
    // Tanpa penanda ini, layar menampilkannya sebagai "Komite Approve" dan petugas
    // mengira rekening itu masih dapat dipakai membayar klaim.
    reply.set('/api/master-rekening', {
      status: 200,
      body: {
        rekening: [
          sampleAccount({
            status: '1',
            status_label: 'Komite Approve',
            aktif: false,
            dapat_dipakai: false,
          }),
        ],
        jumlah: 1,
        batas: 50,
        lewati: 0,
      },
    })

    render(wrap(<AccountPage />))

    expect(await screen.findByText('Komite Approve')).toBeInTheDocument()
    expect(screen.getByText('nonaktif — tidak dapat dipakai')).toBeInTheDocument()
  })

  it('tab Komite Approval meminta antrean komite yang sedang masuk', async () => {
    // Identitas komitenya diambil server dari sesi; klien hanya menyatakan MAU antrean
    // miliknya sendiri. Bila identitasnya dikirim dari klien, siapa pun dapat melihat
    // antrean komite mana pun.
    const pengguna = userEvent.setup()
    render(wrap(<AccountPage />))

    await pengguna.click(screen.getByRole('button', { name: 'Antrean Komite Saya' }))

    await waitFor(() => {
      expect(
        request.some((p) => p.path.includes('komite_saya=1') && p.path.includes('status=0')),
      ).toBe(true)
    })
    expect(request.every((p) => !p.path.includes('identitas'))).toBe(true)
  })

  it('tab Reject menyaring ke status ditolak', async () => {
    const pengguna = userEvent.setup()
    render(wrap(<AccountPage />))

    await pengguna.click(screen.getByRole('button', { name: 'Sudah Ditolak' }))

    await waitFor(() => {
      expect(request.some((p) => p.path.includes('status=2'))).toBe(true)
    })
  })

  it('tombol Approve dan Reject hanya muncul pada tab yang menunggu keputusan', async () => {
    const pengguna = userEvent.setup()
    render(wrap(<AccountPage />))

    // Tab "Cari Data Rekening" hanya membaca.
    expect(await screen.findByText('1234567890')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Approve' })).toBeNull()

    await pengguna.click(screen.getByRole('button', { name: 'Menunggu Approval' }))
    await waitFor(() => {
      expect(screen.getAllByRole('button', { name: 'Approve' }).length).toBeGreaterThan(0)
    })
  })

  it('mengirim keputusan komite beserta keterangannya', async () => {
    const pengguna = userEvent.setup()
    reply.set('/api/master-rekening/014/1234567890/keputusan', {
      status: 200,
      body: sampleAccount({ status: '1', status_label: 'Komite Approve' }),
    })

    render(wrap(<AccountPage />))
    await pengguna.click(screen.getByRole('button', { name: 'Menunggu Approval' }))

    const catatan = await screen.findByPlaceholderText('Keterangan approval atasan')
    await pengguna.type(catatan, 'Disetujui atasan.')
    await pengguna.click(screen.getAllByRole('button', { name: 'Approve' })[0]!)

    await waitFor(() => {
      const decision = request.find((p) => p.path.includes('/keputusan'))
      expect(decision).toBeDefined()
      expect(decision!.metode).toBe('POST')
      expect(decision!.body).toMatchObject({ status: '1', catatan: 'Disetujui atasan.' })
    })
  })

  it('memberi tahu saat komite lain sudah memutuskan lebih dulu', async () => {
    const pengguna = userEvent.setup()
    reply.set('/api/master-rekening/014/1234567890/keputusan', {
      status: 409,
      body: { kode: 'keputusan_sudah_diambil', pesan: 'Sudah diputuskan.' },
    })

    render(wrap(<AccountPage />))
    await pengguna.click(screen.getByRole('button', { name: 'Menunggu Approval' }))

    const button = await screen.findAllByRole('button', { name: 'Approve' })
    await pengguna.click(button[0]!)

    expect(
      await screen.findByText('Rekening ini sudah diputuskan komite lain. Muat ulang daftar.'),
    ).toBeInTheDocument()
  })

  it('menampilkan kegagalan pendaftaran ke Kasir, tidak menyembunyikannya', async () => {
    // Rekening yang disetujui tetapi gagal didaftarkan ke Kasir akan menahan
    // pembayaran; satu-satunya yang dapat menindaklanjutinya adalah petugas di layar.
    reply.set('/api/master-rekening', {
      status: 200,
      body: {
        rekening: [
          sampleAccount({
            status: '1',
            status_label: 'Komite Approve',
            status_layanan: 'GAGAL',
            respons_kasir: 'Rekening sudah terdaftar di Kasir.',
          }),
        ],
        jumlah: 1,
        batas: 50,
        lewati: 0,
      },
    })

    render(wrap(<AccountPage />))

    expect(await screen.findByText(/Gagal · Rekening sudah terdaftar di Kasir\./)).toBeInTheDocument()
  })
})
