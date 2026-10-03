import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, renderHook, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { AccountPage } from './AccountPage'
import { useAccountList, useUpdateAccount, type AccountFields } from './api'

/**
 * Uji tambahan Master Rekening: formulir pengajuan, cabang galat, tabel, dan hook yang
 * tidak dipakai layar. Seluruh data KARANGAN (`D-69`).
 */

type Reply = { status: number; body: unknown } | 'putus' | 'rusak'

let reply: Map<string, Reply>
let request: { path: string; metode: string; body: unknown; header: Record<string, string> }[]

/** Jawaban yang membuat callAPI melempar galat selain APIError/NetworkError. */
function brokenResponse(): Response {
  return {
    get status(): number {
      throw new TypeError('jawaban rusak')
    },
  } as unknown as Response
}

function installFakeFetch() {
  request = []
  vi.stubGlobal('fetch', async (path: string, options?: RequestInit) => {
    request.push({
      path,
      metode: options?.method ?? 'GET',
      body: options?.body ? JSON.parse(String(options.body)) : undefined,
      header: (options?.headers ?? {}) as Record<string, string>,
    })
    const method = options?.method ?? 'GET'
    const key = [...reply.keys()]
      .filter((k) => {
        const [m, p] = k.split(' ')
        return m === method && path.startsWith(p!)
      })
      .sort((a, b) => b.length - a.length)[0]
    const result = key ? reply.get(key)! : { status: 404, body: null }
    if (result === 'putus') throw new TypeError('putus')
    if (result === 'rusak') return brokenResponse()
    return new Response(JSON.stringify(result.body), {
      status: result.status,
      headers: { 'Content-Type': 'application/json' },
    })
  })
}

function client() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
}

function wrap(children: ReactNode) {
  return <QueryClientProvider client={client()}>{children}</QueryClientProvider>
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
    email: 'keuangan@contoh.example',
    telepon: '',
    nik: '3171000000000000',
    catatan: 'catatan awal',
    id_dokumen: '',
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

function listOf(rows: unknown[], jumlah = rows.length) {
  return { status: 200, body: { rekening: rows, jumlah, batas: 50, lewati: 0 } }
}

beforeEach(() => {
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
  reply = new Map<string, Reply>([
    ['GET /api/master-rekening/bank', { status: 200, body: { bank: [{ kode: '014', nama: 'BANK BCA' }] } }],
    ['GET /api/master-rekening', listOf([sampleAccount()])],
  ])
  installFakeFetch()
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

async function openForm() {
  render(wrap(<AccountPage />))
  await screen.findByText('1234567890')
  await userEvent.click(screen.getByRole('button', { name: 'Tambah rekening' }))
  return screen.findByRole('button', { name: 'Ajukan rekening' })
}

async function fillForm() {
  await userEvent.type(screen.getByLabelText('Nomor rekening'), '9988776655')
  await userEvent.type(screen.getByLabelText('Nama pemilik rekening'), 'PEMILIK CONTOH')
  await screen.findByRole('option', { name: 'BANK BCA' })
  await userEvent.selectOptions(screen.getByLabelText('Bank'), '014')
  await userEvent.type(screen.getAllByLabelText('Nama bank')[0]!, 'BANK BCA')
  await userEvent.type(screen.getByLabelText('Cabang bank'), 'SLEMAN')
  await userEvent.type(screen.getByLabelText('Alamat bank'), 'JL. CONTOH 2')
  await userEvent.selectOptions(screen.getByLabelText('Tipe rekening'), 'VA')
  await userEvent.type(screen.getByLabelText('NIK pemilik rekening'), '3404000000000001')
  await userEvent.type(screen.getByLabelText('Email'), 'pemilik@contoh.example')
  await userEvent.type(screen.getByLabelText('Catatan'), 'ajuan baru')
}

describe('formulir rekening baru', () => {
  it('membuka dan menutup formulir lewat tombol yang sama', async () => {
    await openForm()
    expect(screen.getByText('Rekening baru')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Tutup formulir' }))
    expect(screen.queryByText('Rekening baru')).not.toBeInTheDocument()
  })

  it('menolak isian kosong dan format salah di layar tanpa memanggil server', async () => {
    const submit = await openForm()
    await userEvent.click(submit)

    for (const text of [
      'Nomor rekening wajib diisi.',
      'Nama pemilik rekening wajib diisi.',
      'Nama bank wajib diisi.',
      'Nama cabang bank wajib diisi.',
      'Alamat bank wajib diisi.',
      'Bank wajib dipilih dari daftar.',
      'Tipe rekening wajib dipilih.',
      'Email wajib diisi.',
      'NIK pemilik rekening wajib diisi.',
    ]) {
      expect(await screen.findByText(text)).toBeInTheDocument()
    }
    expect(screen.getByLabelText('Bank')).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByLabelText('Tipe rekening')).toHaveAttribute('aria-invalid', 'true')

    await userEvent.type(screen.getByLabelText('Nomor rekening'), '12A')
    await userEvent.type(screen.getByLabelText('Email'), 'bukan-email')
    await userEvent.click(submit)
    expect(await screen.findByText('Nomor rekening hanya boleh berisi angka.')).toBeInTheDocument()
    expect(screen.getByText('Format email tidak benar.')).toBeInTheDocument()
    expect(request.some((r) => r.metode === 'POST')).toBe(false)
  })

  it('mengirim pengajuan lengkap lalu menutup formulir', async () => {
    reply.set('POST /api/master-rekening', { status: 201, body: sampleAccount() })
    const submit = await openForm()
    await fillForm()
    await userEvent.click(submit)

    await waitFor(() => expect(screen.queryByText('Rekening baru')).not.toBeInTheDocument())
    const sent = request.find((r) => r.metode === 'POST')
    expect(sent?.path).toBe('/api/master-rekening')
    expect(sent?.header['X-Portal']).toBe('ASM')
    expect(sent?.body).toEqual({
      nomor_rekening: '9988776655',
      nama_pemilik: 'PEMILIK CONTOH',
      nama_bank: 'BANK BCA',
      cabang_bank: 'SLEMAN',
      alamat_bank: 'JL. CONTOH 2',
      kode_bank: '014',
      tipe_rekening: 'VA',
      email: 'pemilik@contoh.example',
      telepon: '',
      nik: '3404000000000001',
      id_dokumen: '',
      catatan: 'ajuan baru',
      aktif: true,
      kode_bank_lama: '',
      nomor_rekening_lama: '',
      nama_pemilik_lama: '',
    })
  })

  it.each([
    {
      name: 'nomor sudah terdaftar',
      answer: { status: 409, body: { kode: 'nomor_rekening_sudah_ada', pesan: 'x' } } as Reply,
      title: 'Nomor rekening sudah terdaftar',
    },
    {
      name: 'galat API lain',
      answer: { status: 500, body: { kode: 'galat_internal', pesan: 'x' } } as Reply,
      title: 'Terjadi kesalahan pada sistem',
      description: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
    },
    {
      name: 'jaringan putus',
      answer: 'putus' as Reply,
      title: 'Server Claim PNC tidak dapat dihubungi',
    },
    {
      name: 'galat bukan API',
      answer: 'rusak' as Reply,
      title: 'Terjadi kesalahan pada sistem',
      description: 'Coba beberapa saat lagi.',
    },
  ])('menampilkan galat pengajuan: $name', async ({ answer, title, description }) => {
    reply.set('POST /api/master-rekening', answer)
    const submit = await openForm()
    await fillForm()
    await userEvent.click(submit)

    expect(await screen.findByText(title)).toBeInTheDocument()
    if (description) expect(screen.getByText(description)).toBeInTheDocument()
    // Formulir tetap terbuka supaya pengguna dapat membetulkan isian.
    expect(screen.getByText('Rekening baru')).toBeInTheDocument()
  })

  it('memindahkan galat validasi server ke kolomnya masing-masing', async () => {
    reply.set('POST /api/master-rekening', {
      status: 422,
      body: {
        kode: 'isian_tidak_sah',
        pesan: 'x',
        field: { nomor_rekening: 'Nomor ditolak server.', kolom_asing: 'Diabaikan.' },
      },
    })
    const submit = await openForm()
    await fillForm()
    await userEvent.click(submit)

    expect(await screen.findByText('Ada isian yang belum benar')).toBeInTheDocument()
    expect(await screen.findByText('Nomor ditolak server.')).toBeInTheDocument()
    expect(screen.queryByText('Diabaikan.')).not.toBeInTheDocument()
  })

  it('memberi tahu bila daftar bank gagal dimuat', async () => {
    reply.set('GET /api/master-rekening/bank', { status: 500, body: { kode: 'x', pesan: 'x' } })
    await openForm()

    expect(
      await screen.findByText('Daftar bank tidak dapat dimuat. Muat ulang halaman, lalu coba lagi.'),
    ).toBeInTheDocument()
  })
})

describe('daftar dan tabel', () => {
  it('menampilkan keadaan memuat lalu pesan kosong', async () => {
    reply.set('GET /api/master-rekening', listOf([]))
    render(wrap(<AccountPage />))

    expect(screen.getByText('Memuat data rekening…')).toBeInTheDocument()
    expect(
      await screen.findByText('Tidak ada rekening yang cocok dengan pencarian Anda.'),
    ).toBeInTheDocument()
  })

  it('menampilkan peringatan bila daftar gagal dimuat', async () => {
    reply.set('GET /api/master-rekening', { status: 500, body: { kode: 'x', pesan: 'x' } })
    render(wrap(<AccountPage />))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Daftar rekening tidak dapat dimuat. Coba beberapa saat lagi.',
    )
  })

  it('menampilkan status, keterangan Kasir, dan jumlah yang terpotong', async () => {
    reply.set(
      'GET /api/master-rekening',
      listOf(
        [
          sampleAccount({
            nomor_rekening: '111',
            status: '1',
            status_label: 'Komite Approve',
            status_layanan: 'BERHASIL',
            id_rekening_kasir: 'KSR-9',
          }),
          sampleAccount({
            nomor_rekening: '222',
            status: '1',
            status_label: 'Komite Approve',
            status_layanan: 'BERHASIL',
          }),
          sampleAccount({ nomor_rekening: '333', status: '2', status_label: 'Komite Reject' }),
          sampleAccount({
            nomor_rekening: '444',
            status: '9',
            status_label: '',
            status_layanan: 'GAGAL',
          }),
        ],
        10,
      ),
    )
    render(wrap(<AccountPage />))

    expect(await screen.findByText('Terdaftar · KSR-9')).toBeInTheDocument()
    expect(screen.getByText('Terdaftar')).toBeInTheDocument()
    expect(screen.getByText('Komite Reject')).toHaveClass('bg-red-100')
    expect(screen.getByText('Tidak dikenal')).toHaveClass('bg-amber-100')
    expect(screen.getByText('Gagal')).toBeInTheDocument()
    expect(
      screen.getByText(/Menampilkan 4 dari 10\s+rekening\. Persempit pencarian untuk melihat sisanya\./),
    ).toBeInTheDocument()
  })

  it('mengirim isian pencarian sebagai saringan dan tab Disetujui ke status 1', async () => {
    render(wrap(<AccountPage />))
    await screen.findByText('1234567890')

    await userEvent.type(screen.getByLabelText('No rekening'), ' 12 ')
    await userEvent.type(screen.getByLabelText('Nama pemilik'), 'BENG')
    await userEvent.type(screen.getByLabelText('Nama bank'), 'CON')
    await userEvent.click(screen.getByRole('button', { name: 'Sudah Disetujui' }))

    await waitFor(() =>
      expect(
        request.some(
          (r) =>
            r.path ===
            '/api/master-rekening?status=1&nomor_rekening=12&nama_pemilik=BENG&nama_bank=CON',
        ),
      ).toBe(true),
    )
    expect(screen.getByRole('button', { name: 'Sudah Disetujui' })).toHaveAttribute(
      'aria-current',
      'page',
    )
    // Tombol tambah hanya ada di tab pencarian.
    expect(screen.queryByRole('button', { name: 'Tambah rekening' })).not.toBeInTheDocument()
  })
})

describe('keputusan komite', () => {
  async function openPending() {
    render(wrap(<AccountPage />))
    await userEvent.click(screen.getByRole('button', { name: 'Menunggu Approval' }))
    return screen.findByDisplayValue('catatan awal')
  }

  it('mengirim penolakan dengan keterangan bawaan rekening', async () => {
    reply.set('POST /api/master-rekening/014/1234567890/keputusan', {
      status: 200,
      body: sampleAccount({ status: '2' }),
    })
    await openPending()
    await userEvent.click(screen.getByRole('button', { name: 'Reject' }))

    await waitFor(() => {
      const decision = request.find((r) => r.path.endsWith('/keputusan'))
      expect(decision?.body).toEqual({ status: '2', catatan: 'catatan awal', id_dokumen: '' })
    })
  })

  it('menampilkan pelanggaran isian dari server', async () => {
    reply.set('POST /api/master-rekening/014/1234567890/keputusan', {
      status: 422,
      body: {
        kode: 'isian_tidak_sah',
        pesan: 'x',
        detail: [{ field: 'catatan', pesan: 'Keterangan wajib diisi.' }],
        field: { id_dokumen: 'Buku rekening belum diunggah.' },
      },
    })
    await openPending()
    await userEvent.click(screen.getByRole('button', { name: 'Approve' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      'Buku rekening belum diunggah. Keterangan wajib diisi.',
    )
  })

  it('menampilkan pesan umum untuk galat keputusan yang lain', async () => {
    reply.set('POST /api/master-rekening/014/1234567890/keputusan', 'putus')
    await openPending()
    await userEvent.click(screen.getByRole('button', { name: 'Approve' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Keputusan tidak tersimpan. Coba lagi.')
  })
})

describe('hook', () => {
  function hookWrapper({ children }: { children: ReactNode }) {
    return <QueryClientProvider client={client()}>{children}</QueryClientProvider>
  }

  it('useAccountList menyertakan batas dan lewati pada kueri', async () => {
    const { result } = renderHook(() => useAccountList({ batas: 20, lewati: 40 }), {
      wrapper: hookWrapper,
    })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(request.some((r) => r.path === '/api/master-rekening?batas=20&lewati=40')).toBe(true)
  })

  it('useAccountList tidak memanggil server tanpa sesi', () => {
    useSession.setState({ token: null })
    const { result } = renderHook(() => useAccountList({}), { wrapper: hookWrapper })

    expect(result.current.fetchStatus).toBe('idle')
    expect(request).toHaveLength(0)
  })

  it('useUpdateAccount mengirim PUT ke kunci lama beserta nilai lamanya', async () => {
    reply.set('PUT /api/master-rekening', { status: 200, body: sampleAccount() })
    const values: AccountFields = {
      nomorRekening: '777',
      namaPemilik: 'BARU',
      namaBank: 'BANK',
      cabangBank: 'CAB',
      alamatBank: 'ALM',
      kodeBank: '014',
      tipeRekening: 'BIASA',
      email: 'baru@contoh.example',
      telepon: '0800',
      nik: '1',
      idDokumen: 'D1',
      catatan: 'c',
      aktif: false,
      kodeBankLama: '002',
      nomorRekeningLama: '1/2',
      namaPemilikLama: 'LAMA',
    }
    const { result } = renderHook(() => useUpdateAccount(), { wrapper: hookWrapper })

    result.current.mutate({ kodeBank: '002', nomorRekening: '1/2', values })

    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    const sent = request.find((r) => r.metode === 'PUT')
    expect(sent?.path).toBe('/api/master-rekening/002/1%2F2')
    expect(sent?.body).toMatchObject({
      kode_bank_lama: '002',
      nomor_rekening_lama: '1/2',
      nama_pemilik_lama: 'LAMA',
      aktif: false,
    })
  })
})
