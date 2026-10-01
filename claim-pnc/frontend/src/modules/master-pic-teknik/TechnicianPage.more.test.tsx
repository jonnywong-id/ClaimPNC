import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { renderHook, render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { Technician } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { useTechnician } from './api'
import { TechnicianForm } from './TechnicianForm'
import { TechnicianPage } from './TechnicianPage'

/**
 * Uji tambahan Master PIC Teknik: cabang galat, validasi form, dan sel tabel.
 * Seluruh nilai KARANGAN (`D-69`).
 */

type Call = { url: string; method: string; body: unknown }
type Reply = { body?: unknown; status?: number; fail?: boolean; hold?: boolean }

let calls: Call[] = []

function installFetch(map: (call: Call) => Reply) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
      body: init?.body ? JSON.parse(init.body as string) : undefined,
    }
    calls.push(call)
    const reply = map(call)
    if (reply.fail) return Promise.reject(new TypeError('Failed to fetch'))
    if (reply.hold) return new Promise(() => {})
    return Promise.resolve(
      new Response(JSON.stringify(reply.body ?? {}), {
        status: reply.status ?? 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

const TECH: Technician = {
  id_operator: 'PICTEKNIK09',
  nama: '',
  email: 'contoh.sembilan@example.invalid',
  lini_bisnis: '',
  grup: '',
  atasan: '',
  kuota: 0,
  kuota_luar: 0,
  beban_kerja: 3,
  grup_panel: '',
  aktif: false,
}

function client() {
  return new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
}

function wrap(children: ReactNode) {
  return render(<QueryClientProvider client={client()}>{children}</QueryClientProvider>)
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
  useSelectedPortal.getState().clear()
})

describe('TechnicianPage — galat dan sel', () => {
  it.each([
    [{ fail: true }, 'Tidak dapat menghubungi server'],
    [{ status: 400, body: { kode: 'portal_tidak_disebut', pesan: 'x' } }, 'Portal entitas belum dipilih'],
    [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
    [{ status: 500, body: { kode: 'galat_internal', pesan: 'Server rusak.' } }, 'Daftar PIC teknik gagal dimuat'],
  ])('menjelaskan galat daftar %#', async (reply, title) => {
    installFetch(() => reply)
    wrap(<TechnicianPage />)
    expect(await screen.findByText(title)).toBeInTheDocument()
  })

  it('menulis tanda pisah untuk grup, atasan, kuota luar kosong dan memuat ulang', async () => {
    installFetch(() => ({
      body: {
        pic_teknik: [TECH, { ...TECH, id_operator: 'PICTEKNIK10', nama: 'Lain', kuota_luar: 2 }],
        total: 2,
        portal: 'ASM',
      },
    }))
    const user = userEvent.setup()
    wrap(<TechnicianPage />)

    // Nama kosong membuat label tombol memakai ID operator.
    expect(
      await screen.findByRole('button', { name: 'Ubah PIC teknik PICTEKNIK09' }),
    ).toBeInTheDocument()
    expect(screen.getAllByText('—').length).toBeGreaterThanOrEqual(3)
    expect(screen.getAllByText('2').length).toBeGreaterThan(0)

    const before = calls.length
    await user.click(screen.getByRole('button', { name: /Refresh/ }))
    await waitFor(() => expect(calls.length).toBe(before + 1))
  })

  it('menyaring baris lewat kotak cari dan mengurutkan menurut beban', async () => {
    installFetch(() => ({
      body: {
        pic_teknik: [
          { ...TECH, nama: 'Petugas Ringan', beban_kerja: 1, kuota_luar: 4 },
          { ...TECH, id_operator: 'PICTEKNIK11', nama: 'Petugas Berat', beban_kerja: 8 },
        ],
        total: 2,
        portal: 'ASM',
      },
    }))
    const user = userEvent.setup()
    wrap(<TechnicianPage />)

    await screen.findAllByText('Petugas Ringan')
    await user.click(screen.getAllByRole('button', { name: /Beban \/ Kuota/ })[0]!)
    await user.click(screen.getAllByRole('button', { name: /Kuota sistem lain/ })[0]!)
    await user.type(screen.getByLabelText(/Cari ID operator/), 'Berat')

    await waitFor(() => expect(screen.queryAllByText('Petugas Ringan')).toHaveLength(0))
    expect(screen.getAllByText('Petugas Berat').length).toBeGreaterThan(0)
  })

  it('mengunci tombol Tambah selagi form tambah terbuka, lalu menutup lewat Batal', async () => {
    installFetch(() => ({ body: { pic_teknik: [], total: 0, portal: 'ASM' } }))
    const user = userEvent.setup()
    wrap(<TechnicianPage />)

    expect(await screen.findByText('Belum ada PIC teknik aktif pada entitas ini.')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /Tambah/ }))
    expect(screen.getByRole('button', { name: /Tambah/ })).toBeDisabled()

    await user.click(screen.getByRole('button', { name: 'Batal' }))
    expect(screen.queryByRole('form', { name: 'Tambah PIC teknik' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Tambah/ })).not.toBeDisabled()
  })
})

describe('TechnicianForm', () => {
  function show(technician: Technician | null = null) {
    const onClose = vi.fn()
    wrap(<TechnicianForm technician={technician} onClose={onClose} />)
    return { onClose }
  }

  it('menolak isian kosong dan format email yang salah tanpa menembak server', async () => {
    installFetch(() => ({ body: {} }))
    const user = userEvent.setup()
    show()

    expect(screen.getByLabelText('ID Operator')).toHaveFocus()
    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))
    expect(await screen.findByText('ID operator wajib diisi.')).toBeInTheDocument()

    await user.type(screen.getByLabelText('Email'), 'bukan-email')
    await user.clear(screen.getByLabelText('Kuota'))
    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))

    expect(await screen.findByText('Format email tidak benar.')).toBeInTheDocument()
    expect(screen.getByText('Kuota harus berupa angka.')).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })

  it('tidak mencari ke direktori bila ID masih kosong', async () => {
    installFetch(() => ({ body: {} }))
    const user = userEvent.setup()
    show()

    await user.click(screen.getByRole('button', { name: 'Cari pegawai di direktori' }))
    expect(calls).toHaveLength(0)
  })

  it('tidak menimpa atasan dan surel yang sudah diisi', async () => {
    installFetch(() => ({
      body: {
        pegawai: {
          id_operator: 'X',
          nama: 'Pegawai Contoh',
          email: 'dari.direktori@example.invalid',
          atasan: 'ATASANDIREKTORI',
          nama_atasan: 'Atasan Contoh',
        },
        portal: 'ASM',
      },
    }))
    const user = userEvent.setup()
    show()

    await user.type(screen.getByLabelText('Atasan'), 'ATASANSAYA')
    await user.type(screen.getByLabelText('Email'), 'saya@example.invalid')
    await user.type(screen.getByLabelText('ID Operator'), 'X')
    // Meninggalkan kolom ID juga memicu pencarian.
    await user.tab()

    expect(await screen.findByText('Pegawai Contoh')).toBeInTheDocument()
    expect(calls[0]?.url).toBe('/api/master/pic-teknik/direktori/X')
    expect(screen.getByLabelText('Atasan')).toHaveValue('ATASANSAYA')
    expect(screen.getByLabelText('Email')).toHaveValue('saya@example.invalid')
  })

  it.each([
    [{ fail: true }, 'Tidak dapat menghubungi server'],
    [
      { status: 503, body: { kode: 'direktori_pegawai_tidak_terhubung', pesan: 'x' } },
      'Direktori pegawai sedang tidak dapat dihubungi',
    ],
    [{ status: 422, body: { kode: 'validasi_gagal', pesan: 'ID kosong.' } }, 'Isian belum benar'],
    [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
    [{ status: 500, body: { kode: 'galat_internal', pesan: 'Rusak.' } }, 'Pencarian gagal'],
  ])('menjelaskan galat pencarian direktori %#', async (reply, title) => {
    installFetch(() => reply)
    const user = userEvent.setup()
    show()

    await user.type(screen.getByLabelText('ID Operator'), 'X')
    await user.click(screen.getByRole('button', { name: 'Cari pegawai di direktori' }))
    expect(await screen.findByText(title)).toBeInTheDocument()
  })

  it.each([
    [{ fail: true }, 'Tidak dapat menghubungi server'],
    [{ status: 409, body: { kode: 'id_operator_sudah_terdaftar', pesan: 'x' } }, 'ID operator sudah terdaftar'],
    [{ status: 404, body: { kode: 'pic_teknik_tidak_ditemukan', pesan: 'x' } }, 'PIC teknik tidak ditemukan'],
    [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
    [
      { status: 503, body: { kode: 'direktori_pegawai_tidak_terhubung', pesan: 'x' } },
      'Direktori pegawai sedang tidak dapat dihubungi',
    ],
  ])('menjelaskan galat simpan %#', async (reply, title) => {
    installFetch(() => reply)
    const user = userEvent.setup()
    const { onClose } = show(TECH)

    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))
    expect(await screen.findByText(title)).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('menyembunyikan kotak pesan untuk validasi simpan yang sudah tersorot di isiannya', async () => {
    installFetch(() => ({
      status: 422,
      body: {
        kode: 'validasi_gagal',
        pesan: 'Isian belum benar.',
        detail: [
          { field: 'grup', pesan: 'Grup tidak dikenal.' },
          { field: 'bukan_isian', pesan: 'diabaikan' },
        ],
      },
    }))
    const user = userEvent.setup()
    show(TECH)

    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))
    expect(await screen.findByText('Grup tidak dikenal.')).toBeInTheDocument()
    expect(screen.queryByText('Isian belum benar')).not.toBeInTheDocument()
    expect(screen.queryByText('diabaikan')).not.toBeInTheDocument()
  })

  it('menjatuhkan galat simpan tak dikenal ke pesan umum lalu menutup setelah berhasil', async () => {
    let attempt = 0
    installFetch(() => {
      attempt++
      return attempt === 1
        ? { status: 500, body: { kode: 'galat_lain', pesan: 'Galat aneh.' } }
        : { body: { pic_teknik: TECH, portal: 'ASM' } }
    })
    const user = userEvent.setup()
    const { onClose } = show(TECH)

    // Petugas nonaktif tidak diberi peringatan penonaktifan.
    expect(screen.queryByText(/hilang dari daftar/)).not.toBeInTheDocument()
    expect(screen.getByLabelText('Status')).toHaveValue('0')

    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))
    expect(await screen.findByText('Pencarian gagal')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))
    expect(calls[1]?.method).toBe('PUT')
    expect(calls[1]?.body).toMatchObject({ id_operator: 'PICTEKNIK09', aktif: false })
  })

  it('menampilkan Mencari selagi direktori belum menjawab', async () => {
    installFetch(() => ({ hold: true }))
    const user = userEvent.setup()
    show()

    await user.type(screen.getByLabelText('ID Operator'), 'X')
    await user.click(screen.getByRole('button', { name: 'Cari pegawai di direktori' }))
    expect(await screen.findByText('Mencari…')).toBeInTheDocument()
  })
})

describe('useTechnician', () => {
  it('tidak menembak tanpa ID dan membuka petugas yang diminta', async () => {
    installFetch(() => ({ body: { pic_teknik: TECH, portal: 'ASM' } }))
    const qc = client()
    const wrapper = ({ children }: { children: ReactNode }) => (
      <QueryClientProvider client={qc}>{children}</QueryClientProvider>
    )

    const empty = renderHook(() => useTechnician(null), { wrapper })
    expect(empty.result.current.fetchStatus).toBe('idle')

    const { result } = renderHook(() => useTechnician('PIC 9'), { wrapper })
    await waitFor(() => expect(result.current.isSuccess).toBe(true))
    expect(calls.map((c) => c.url)).toEqual(['/api/master/pic-teknik/PIC%209'])
    expect(result.current.data?.pic_teknik.id_operator).toBe('PICTEKNIK09')
  })
})
