import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { gunakanSesi } from '@/app/sesi'

import { HalamanMasterRekening } from './HalamanMasterRekening'

// Seluruh uji di berkas ini berjalan tanpa jaringan: fetch digantikan tiruan yang
// menjawab dari peta jalur di bawah.

type Jawaban = { status: number; badan: unknown }

let jawaban: Map<string, Jawaban>
let permintaan: { jalur: string; metode: string; badan: unknown }[]

function pasangFetchTiruan() {
  permintaan = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (jalur: string, opsi?: RequestInit) => {
      permintaan.push({
        jalur,
        metode: opsi?.method ?? 'GET',
        badan: opsi?.body ? JSON.parse(String(opsi.body)) : undefined,
      })

      const kunci = [...jawaban.keys()].find((k) => jalur.startsWith(k))
      const hasil = kunci ? jawaban.get(kunci)! : { status: 404, badan: null }
      return new Response(JSON.stringify(hasil.badan), {
        status: hasil.status,
        headers: { 'Content-Type': 'application/json' },
      })
    }),
  )
}

function bungkus(anak: React.ReactNode) {
  const klien = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return <QueryClientProvider client={klien}>{anak}</QueryClientProvider>
}

function rekeningContoh(timpa: Record<string, unknown> = {}) {
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
    ...timpa,
  }
}

beforeEach(() => {
  gunakanSesi.setState({
    token: 'token-uji',
    pengguna: null,
    berlakuSampai: '2026-12-31T00:00:00Z',
  })
  jawaban = new Map([
    ['/api/master-rekening/bank', { status: 200, badan: { bank: [{ kode: '014', nama: 'BANK BCA' }] } }],
    ['/api/master-rekening', { status: 200, badan: { rekening: [rekeningContoh()], jumlah: 1, batas: 50, lewati: 0 } }],
  ])
  pasangFetchTiruan()
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('HalamanMasterRekening', () => {
  it('menampilkan kelima tab layar lama', async () => {
    render(bungkus(<HalamanMasterRekening />))

    for (const label of [
      'Cari Data Rekening',
      'Komite Approval',
      'Waiting Approval',
      'Approve',
      'Reject',
    ]) {
      expect(screen.getByRole('button', { name: label })).toBeInTheDocument()
    }
  })

  it('menampilkan rekening beserta status persetujuannya', async () => {
    render(bungkus(<HalamanMasterRekening />))

    expect(await screen.findByText('1234567890')).toBeInTheDocument()
    expect(screen.getByText('BENGKEL CONTOH SEJAHTERA')).toBeInTheDocument()
    expect(screen.getByText('Menunggu')).toBeInTheDocument()
  })

  it('menandai rekening yang disetujui tetapi sudah dinonaktifkan', async () => {
    // Tanpa penanda ini, layar menampilkannya sebagai "Komite Approve" dan petugas
    // mengira rekening itu masih dapat dipakai membayar klaim.
    jawaban.set('/api/master-rekening', {
      status: 200,
      badan: {
        rekening: [
          rekeningContoh({
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

    render(bungkus(<HalamanMasterRekening />))

    expect(await screen.findByText('Komite Approve')).toBeInTheDocument()
    expect(screen.getByText('nonaktif — tidak dapat dipakai')).toBeInTheDocument()
  })

  it('tab Komite Approval meminta antrean komite yang sedang masuk', async () => {
    // Identitas komitenya diambil server dari sesi; klien hanya menyatakan MAU antrean
    // miliknya sendiri. Bila identitasnya dikirim dari klien, siapa pun dapat melihat
    // antrean komite mana pun.
    const pengguna = userEvent.setup()
    render(bungkus(<HalamanMasterRekening />))

    await pengguna.click(screen.getByRole('button', { name: 'Komite Approval' }))

    await waitFor(() => {
      expect(
        permintaan.some((p) => p.jalur.includes('komite_saya=1') && p.jalur.includes('status=0')),
      ).toBe(true)
    })
    expect(permintaan.every((p) => !p.jalur.includes('identitas'))).toBe(true)
  })

  it('tab Reject menyaring ke status ditolak', async () => {
    const pengguna = userEvent.setup()
    render(bungkus(<HalamanMasterRekening />))

    await pengguna.click(screen.getByRole('button', { name: 'Reject' }))

    await waitFor(() => {
      expect(permintaan.some((p) => p.jalur.includes('status=2'))).toBe(true)
    })
  })

  it('tombol Approve dan Reject hanya muncul pada tab yang menunggu keputusan', async () => {
    const pengguna = userEvent.setup()
    render(bungkus(<HalamanMasterRekening />))

    // Tab "Cari Data Rekening" hanya membaca.
    expect(await screen.findByText('1234567890')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Approve' })).toBeNull()

    await pengguna.click(screen.getByRole('button', { name: 'Waiting Approval' }))
    await waitFor(() => {
      expect(screen.getAllByRole('button', { name: 'Approve' }).length).toBeGreaterThan(0)
    })
  })

  it('mengirim keputusan komite beserta keterangannya', async () => {
    const pengguna = userEvent.setup()
    jawaban.set('/api/master-rekening/014/1234567890/keputusan', {
      status: 200,
      badan: rekeningContoh({ status: '1', status_label: 'Komite Approve' }),
    })

    render(bungkus(<HalamanMasterRekening />))
    await pengguna.click(screen.getByRole('button', { name: 'Waiting Approval' }))

    const catatan = await screen.findByPlaceholderText('Keterangan approval atasan')
    await pengguna.type(catatan, 'Disetujui atasan.')
    await pengguna.click(screen.getAllByRole('button', { name: 'Approve' })[0]!)

    await waitFor(() => {
      const keputusan = permintaan.find((p) => p.jalur.includes('/keputusan'))
      expect(keputusan).toBeDefined()
      expect(keputusan!.metode).toBe('POST')
      expect(keputusan!.badan).toMatchObject({ status: '1', catatan: 'Disetujui atasan.' })
    })
  })

  it('memberi tahu saat komite lain sudah memutuskan lebih dulu', async () => {
    const pengguna = userEvent.setup()
    jawaban.set('/api/master-rekening/014/1234567890/keputusan', {
      status: 409,
      badan: { kode: 'keputusan_sudah_diambil', pesan: 'Sudah diputuskan.' },
    })

    render(bungkus(<HalamanMasterRekening />))
    await pengguna.click(screen.getByRole('button', { name: 'Waiting Approval' }))

    const tombol = await screen.findAllByRole('button', { name: 'Approve' })
    await pengguna.click(tombol[0]!)

    expect(
      await screen.findByText('Rekening ini sudah diputuskan komite lain. Muat ulang daftar.'),
    ).toBeInTheDocument()
  })

  it('menampilkan kegagalan pendaftaran ke Kasir, tidak menyembunyikannya', async () => {
    // Rekening yang disetujui tetapi gagal didaftarkan ke Kasir akan menahan
    // pembayaran; satu-satunya yang dapat menindaklanjutinya adalah petugas di layar.
    jawaban.set('/api/master-rekening', {
      status: 200,
      badan: {
        rekening: [
          rekeningContoh({
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

    render(bungkus(<HalamanMasterRekening />))

    expect(await screen.findByText(/Gagal · Rekening sudah terdaftar di Kasir\./)).toBeInTheDocument()
  })
})
