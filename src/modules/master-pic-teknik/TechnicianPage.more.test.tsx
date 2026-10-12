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
  bisnis: '',
  kelompok: '',
  atasan: '',
  counter_klaim_kurang_1m: 0,
  counter_klaim_lebih_1m: 0,
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

  it('menulis tanda pisah untuk atasan, bisnis, dan kelompok kosong lalu memuat ulang', async () => {
    installFetch(() => ({
      body: {
        pic_teknik: [TECH, { ...TECH, id_operator: 'PICTEKNIK10', nama: 'Lain', counter_klaim_lebih_1m: 2 }],
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

  it('menyaring baris lewat kotak cari dan mengurutkan menurut pencacah klaim', async () => {
    installFetch(() => ({
      body: {
        pic_teknik: [
          { ...TECH, nama: 'Petugas Ringan', beban_kerja: 1, counter_klaim_lebih_1m: 4 },
          { ...TECH, id_operator: 'PICTEKNIK11', nama: 'Petugas Berat', beban_kerja: 8 },
        ],
        total: 2,
        portal: 'ASM',
      },
    }))
    const user = userEvent.setup()
    wrap(<TechnicianPage />)

    // Grid menampilkan ID operator, bukan nama — sama dengan Pega. Nama tetap ikut
    // DICARI, dan itulah yang diuji di bawah.
    await screen.findAllByText('PICTEKNIK09')
    await user.click(screen.getAllByRole('button', { name: /Counter Klaim <1M/ })[0]!)
    await user.click(screen.getAllByRole('button', { name: /Counter Klaim >1M/ })[0]!)
    await user.type(screen.getByLabelText(/Cari ID operator/), 'Berat')

    // Mencari lewat NAMA tetap bekerja walau namanya tidak ditampilkan: baris yang namanya
    // tidak cocok hilang, yang cocok bertahan.
    await waitFor(() => expect(screen.queryAllByText('PICTEKNIK09')).toHaveLength(0))
    expect(screen.getAllByText('PICTEKNIK11').length).toBeGreaterThan(0)
  })

  it('mengunci tombol Tambah selagi form tambah terbuka, lalu menutup lewat Batal', async () => {
    installFetch(() => ({ body: { pic_teknik: [], total: 0, portal: 'ASM' } }))
    const user = userEvent.setup()
    wrap(<TechnicianPage />)

    expect(await screen.findByText('Belum ada PIC teknik pada entitas ini.')).toBeInTheDocument()
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

    expect(screen.getByLabelText('Username')).toHaveFocus()
    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))
    expect(await screen.findByText('Username wajib diisi.')).toBeInTheDocument()

    await user.type(screen.getByLabelText('Email'), 'bukan-email')
    await user.clear(screen.getByLabelText('Counter Klaim <1M'))
    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))

    expect(await screen.findByText('Format email tidak benar.')).toBeInTheDocument()
    expect(screen.getByText('Counter Klaim <1M harus berupa angka.')).toBeInTheDocument()
    expect(calls.filter((c) => c.method === 'POST')).toHaveLength(0)
  })

  it('tidak mencari ke direktori bila Username masih kosong', async () => {
    installFetch(() => ({ body: {} }))
    const user = userEvent.setup()
    show()

    await user.tab()
    // Form memuat daftar untuk mengisi dropdown Kelompok, Atasan, dan Bisnis; yang tidak
    // boleh terjadi adalah panggilan ke DIREKTORI.
    expect(calls.filter((c) => c.url.includes('/direktori/'))).toHaveLength(0)
  })

  it('tidak menimpa atasan dan surel yang sudah diisi', async () => {
    installFetch((call) => {
      if (call.url.includes('/direktori/')) {
        return {
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
        }
      }
      // Daftar mengisi dropdown Atasan; tanpa isi, tidak ada yang dapat dipilih.
      return {
        body: {
          pic_teknik: [{ ...TECH, id_operator: 'ATASANSAYA' }],
          total: 1,
          portal: 'ASM',
        },
      }
    })
    const user = userEvent.setup()
    show()

    // Atasan kini DROPDOWN, mengikuti layar Pega — dipilih, bukan diketik.
    await waitFor(() => {
      expect(screen.getByRole('option', { name: 'ATASANSAYA' })).toBeInTheDocument()
    })
    await user.selectOptions(screen.getByLabelText('Atasan'), 'ATASANSAYA')
    await user.type(screen.getByLabelText('Email'), 'saya@example.invalid')
    await user.type(screen.getByLabelText('Username'), 'X')
    // Meninggalkan isian Username memicu pencarian — tanpa tombol, sama dengan Pega.
    await user.tab()

    expect(await screen.findByText('Pegawai Contoh')).toBeInTheDocument()
    expect(calls.find((c) => c.url.includes('/direktori/'))?.url).toBe(
      '/api/master/pic-teknik/direktori/X',
    )
    // Usulan direktori TIDAK menimpa pilihan pengguna.
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

    await user.type(screen.getByLabelText('Username'), 'X')
    await user.tab()
    expect(await screen.findByText(title)).toBeInTheDocument()
  })

  it.each([
    [{ fail: true }, 'Tidak dapat menghubungi server'],
    [{ status: 409, body: { kode: 'id_operator_sudah_terdaftar', pesan: 'x' } }, 'Username sudah terdaftar'],
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
          { field: 'kelompok', pesan: 'Kelompok tidak dikenal.' },
          { field: 'bukan_isian', pesan: 'diabaikan' },
        ],
      },
    }))
    const user = userEvent.setup()
    show(TECH)

    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))
    expect(await screen.findByText('Kelompok tidak dikenal.')).toBeInTheDocument()
    expect(screen.queryByText('Isian belum benar')).not.toBeInTheDocument()
    expect(screen.queryByText('diabaikan')).not.toBeInTheDocument()
  })

  it('menjatuhkan galat simpan tak dikenal ke pesan umum lalu menutup setelah berhasil', async () => {
    // Stub dikunci pada METODE, bukan urutan panggilan: form memuat daftar lebih dulu
    // untuk mengisi dropdown, sehingga menghitung "panggilan pertama" tidak lagi menunjuk
    // penyimpanan.
    let saves = 0
    installFetch((call) => {
      if (call.method !== 'PUT') return { body: { pic_teknik: [], total: 0, portal: 'ASM' } }
      saves++
      return saves === 1
        ? { status: 500, body: { kode: 'galat_lain', pesan: 'Galat aneh.' } }
        : { body: { pic_teknik: TECH, portal: 'ASM' } }
    })
    const user = userEvent.setup()
    const { onClose } = show(TECH)

    expect(screen.getByLabelText('Status Aktif')).toHaveValue('0')

    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))
    expect(await screen.findByText('Gagal menyimpan')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))
    await waitFor(() => expect(onClose).toHaveBeenCalledTimes(1))

    const put = calls.filter((c) => c.method === 'PUT')
    expect(put).toHaveLength(2)
    expect(put[0]?.body).toMatchObject({ id_operator: 'PICTEKNIK09', aktif: false })
  })

  it('menampilkan Mencari selagi direktori belum menjawab', async () => {
    installFetch(() => ({ hold: true }))
    const user = userEvent.setup()
    show()

    await user.type(screen.getByLabelText('Username'), 'X')
    await user.tab()
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
