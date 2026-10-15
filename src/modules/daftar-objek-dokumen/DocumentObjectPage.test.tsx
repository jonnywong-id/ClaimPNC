import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { DocumentObjectPage } from './DocumentObjectPage'

const BUSINESSES = {
  portal: 'ASM',
  bisnis: [
    { id: '002', nama: 'PERSONAL ACCIDENT' },
    { id: '003', nama: 'ANEKA' },
    { id: '006', nama: 'FIRE / PROPERTY' },
  ],
}

/**
 * Jawaban daftar. Pemetaan bisnisnya SENGAJA kosong pada setiap baris — itulah yang
 * dikirim server, karena grid hanya menampilkan dua kolom.
 *
 * `id_lama` ikut dikirim meski tidak ditampilkan; uji di bawah yang menjaga ia tetap tidak
 * tampil.
 */
const LIST = {
  portal: 'ASM',
  total: 2,
  objek_dokumen: [
    { id: '10001', objek_dokumen: 'KTP Tertanggung', id_lama: '', bisnis: [] },
    { id: '10002', objek_dokumen: 'Polis Asli', id_lama: '07', bisnis: [] },
  ],
}

/**
 * Jawaban GET satu baris — di sinilah pemetaan bisnisnya ikut, bukan di daftar.
 *
 * Baris kedua pemetaannya SENGAJA tanpa `id`: itu bisnis yang namanya diketik bebas dan
 * tidak ada di master, keadaan sah yang harus ditangani layar.
 */
const DETAIL = {
  portal: 'ASM',
  objek_dokumen: {
    id: '10001',
    objek_dokumen: 'KTP Tertanggung',
    id_lama: '',
    bisnis: [
      { id: '002', nama: 'PERSONAL ACCIDENT' },
      { id: '', nama: 'KENDARAAN BERMOTOR' },
    ],
  },
}

type Call = {
  url: string
  method: string
  body: unknown
  header: Record<string, string>
}

let calls: Call[] = []

/** reply memetakan jalur permintaan ke respons yang dikembalikan fetch tiruan. */
type Reply = { body: unknown; status?: number }

function installFetch(map: (call: Call) => Reply) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
      body: init?.body ? JSON.parse(init.body as string) : undefined,
      header: (init?.headers as Record<string, string>) ?? {},
    }
    calls.push(call)

    const { body, status = 200 } = map(call)
    return Promise.resolve(
      new Response(JSON.stringify(body), {
        status,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

/** defaultReply melayani daftar, detail, dan bisnis; mutasi dijawab lewat `mutation`. */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.url.startsWith('/api/master/bisnis')) return { body: BUSINESSES }
    if (call.method === 'GET' && call.url.includes('/objek-dokumen/')) return { body: DETAIL }
    if (call.method === 'GET') return { body: LIST }
    if (mutation) return mutation(call)
    return { body: { objek_dokumen: DETAIL.objek_dokumen, portal: 'ASM' }, status: 201 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DocumentObjectPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function startSession() {
  useSession.getState().login({
    token: 'token-contoh',
    user: {
      identitas: '90000001',
      nama: 'Contoh Administrator',
      jenis: 'KARYAWAN',
      login: 'adminpnc',
      email: '',
      perusahaan: 'ASM',
    },
    validUntil: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
  })
}

/** Membuka form ubah pada baris 10001 dan menunggu pemetaan bisnisnya selesai dimuat. */
async function openEditForm() {
  await screen.findByRole('table')
  await userEvent.click(screen.getByRole('button', { name: 'Ubah objek dokumen KTP Tertanggung' }))

  const form = await screen.findByRole('form', { name: 'Memperbaharui Data' })
  await waitFor(() => {
    expect(within(form).getByLabelText('Bisnis baris 1')).toHaveValue('PERSONAL ACCIDENT')
  })
  return form
}

beforeEach(() => {
  calls = []
  window.sessionStorage.clear()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
  startSession()
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('daftar objek dokumen', () => {
  it('menampilkan judul dan kolom seperti layar lama', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Daftar Objek Dokumen' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tambah' })).toBeInTheDocument()

    const table = await screen.findByRole('table')

    // Tombol Refresh diperiksa SESUDAH daftar selesai dimuat, bukan sebelumnya: selama
    // pemuatan ia sengaja berbunyi "Memuat…" dan dinonaktifkan. Memeriksanya lebih awal
    // akan menguji keadaan sesaat, bukan tombolnya.
    expect(screen.getByRole('button', { name: 'Refresh' })).toBeInTheDocument()
    expect(within(table).getByRole('columnheader', { name: /^ID$/ })).toBeInTheDocument()
    expect(
      within(table).getByRole('columnheader', { name: /Daftar Objek Dokumen/ }),
    ).toBeInTheDocument()
    expect(within(table).getByText('KTP Tertanggung')).toBeInTheDocument()
  })

  /**
   * Grid Pega hanya memuat DUA kolom. `id_lama` tetap dikirim server karena Report
   * Definition memuatnya, tetapi layar tidak menampilkannya.
   *
   * Uji ini yang membuat menampilkannya kembali menjadi keputusan, bukan perbaikan yang
   * menyelinap.
   */
  it('tidak menampilkan kolom ID lama', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    expect(within(table).queryByRole('columnheader', { name: /ID Lama/i })).not.toBeInTheDocument()
    expect(within(table).queryByText('07')).not.toBeInTheDocument()
  })

  /**
   * Satu aplikasi melayani empat badan hukum dengan basis data terpisah, dan "data siapa
   * ini" tidak boleh hanya diandaikan pengguna (`R-20`).
   */
  it('menyebut portal entitas yang menjawab', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.getByText(/Portal entitas:/)).toBeInTheDocument()
  })

  it('mengirim header portal pada setiap permintaan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await waitFor(() => expect(calls.length).toBeGreaterThan(0))
    for (const call of calls) {
      expect(call.header['X-Portal']).toBe('ASM')
    }
  })

  /** Tanpa portal, layar menuntun memilihnya — bukan menembak server lalu menampilkan galat. */
  it('menuntun memilih portal bila belum dipilih', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })

  /**
   * Tidak ada tombol hapus. Seluruh grid di layar lama ber-`pyGridDeleteActivityExists=false`,
   * dan `D-66` melarang penghapusan fisik data bernilai bisnis.
   */
  it('tidak menyediakan tombol hapus baris', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    expect(within(table).queryByRole('button', { name: /Hapus/ })).not.toBeInTheDocument()
  })
})

describe('form objek dokumen', () => {
  it('membuka form tambah tanpa baris ID', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: 'Memperbaharui Data' })
    expect(within(form).getByLabelText('Daftar Objek Dokumen')).toHaveValue('')
    expect(within(form).queryByText('(tidak dapat diubah)')).not.toBeInTheDocument()
    expect(within(form).getByRole('button', { name: 'Simpan' })).toBeInTheDocument()
  })

  /**
   * Baris yang dibuka dimuat ULANG dari server supaya pemetaan bisnisnya ikut terbawa.
   * Daftar sengaja tidak membawanya, sehingga memakai baris dari daftar akan membuat form
   * tampak seolah seluruh bisnisnya sudah dicabut.
   */
  it('memuat pemetaan bisnis saat baris dibuka untuk disunting', async () => {
    installFetch(defaultReply())
    show()

    const form = await openEditForm()
    expect(within(form).getByText('(tidak dapat diubah)')).toBeInTheDocument()
    expect(within(form).getByLabelText('Daftar Objek Dokumen')).toHaveValue('KTP Tertanggung')
    expect(within(form).getByLabelText('Bisnis baris 2')).toHaveValue('KENDARAAN BERMOTOR')
    expect(within(form).getByRole('button', { name: 'Ubah' })).toBeInTheDocument()
  })

  /** Yang dikirim adalah NAMA bisnis, bukan ID — server yang menyelesaikannya. */
  it('mengirim nama bisnis, bukan ID', async () => {
    installFetch(defaultReply(() => ({ body: { objek_dokumen: DETAIL.objek_dokumen, portal: 'ASM' } })))
    show()

    const form = await openEditForm()
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    await waitFor(() => {
      const saved = calls.find((call) => call.method === 'PUT')
      expect(saved?.body).toEqual({
        objek_dokumen: 'KTP Tertanggung',
        bisnis: ['PERSONAL ACCIDENT', 'KENDARAAN BERMOTOR'],
      })
    })
  })

  /** Baris bisnis yang dibiarkan kosong dibuang sebelum dikirim. */
  it('membuang baris bisnis kosong sebelum mengirim', async () => {
    installFetch(defaultReply(() => ({ body: { objek_dokumen: DETAIL.objek_dokumen, portal: 'ASM' } })))
    show()

    const form = await openEditForm()
    await userEvent.click(within(form).getByRole('button', { name: 'Tambah Bisnis' }))
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    await waitFor(() => {
      const saved = calls.find((call) => call.method === 'PUT')
      expect(saved?.body).toEqual({
        objek_dokumen: 'KTP Tertanggung',
        bisnis: ['PERSONAL ACCIDENT', 'KENDARAAN BERMOTOR'],
      })
    })
  })

  /** Baris bisnis dapat dicabut, dan pencabutannya ikut terkirim. */
  it('mencabut baris bisnis', async () => {
    installFetch(defaultReply(() => ({ body: { objek_dokumen: DETAIL.objek_dokumen, portal: 'ASM' } })))
    show()

    const form = await openEditForm()
    await userEvent.click(within(form).getByRole('button', { name: 'Hapus bisnis baris 2' }))
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    await waitFor(() => {
      const saved = calls.find((call) => call.method === 'PUT')
      expect(saved?.body).toEqual({
        objek_dokumen: 'KTP Tertanggung',
        bisnis: ['PERSONAL ACCIDENT'],
      })
    })
  })

  /**
   * Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan membuang
   * isian pengguna saat penyimpanan gagal.
   */
  it('mempertahankan form saat penyimpanan gagal', async () => {
    installFetch(
      defaultReply(() => ({
        body: { kode: 'galat_internal', pesan: 'gagal' },
        status: 500,
      })),
    )
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: 'Memperbaharui Data' })
    await userEvent.type(within(form).getByLabelText('Daftar Objek Dokumen'), 'Surat Kuasa')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Terjadi kesalahan pada sistem')).toBeInTheDocument()
    expect(screen.getByRole('form', { name: 'Memperbaharui Data' })).toBeInTheDocument()
    expect(within(form).getByLabelText('Daftar Objek Dokumen')).toHaveValue('Surat Kuasa')
  })

  /**
   * Pelanggaran yang dilaporkan server disorot pada isiannya, bukan hanya diringkas di satu
   * kotak — server mengirim seluruhnya sekaligus (`P-5`), dan itu hanya berguna bila layar
   * menyorotnya satu per satu.
   */
  it('menyorot isian yang dilaporkan server', async () => {
    installFetch(
      defaultReply(() => ({
        body: {
          kode: 'validasi_gagal',
          pesan: 'Ada isian yang belum benar.',
          detail: [{ kolom: 'objek_dokumen', pesan: 'Terlalu panjang.' }],
        },
        status: 422,
      })),
    )
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: 'Memperbaharui Data' })
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText('Terlalu panjang.')).toBeInTheDocument()
  })

  /**
   * Keterangan KOSONG tetap dapat disimpan, mengikuti layar Pega yang tidak memvalidasi apa
   * pun (`P-5`).
   */
  it('mengizinkan keterangan kosong', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: 'Memperbaharui Data' })
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const saved = calls.find((call) => call.method === 'POST')
      expect(saved?.body).toEqual({ objek_dokumen: '', bisnis: [] })
    })
  })

  /**
   * Daftar bisnis yang gagal dimuat TIDAK menghalangi penyimpanan: namanya memang boleh
   * diketik sendiri. Yang hilang hanya sarannya, dan layar mengatakannya.
   */
  it('tetap dapat dipakai saat daftar bisnis gagal dimuat', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/bisnis')) {
        return { body: { kode: 'galat_internal', pesan: 'gagal' }, status: 500 }
      }
      if (call.method === 'GET' && call.url.includes('/objek-dokumen/')) return { body: DETAIL }
      if (call.method === 'GET') return { body: LIST }
      return { body: { objek_dokumen: DETAIL.objek_dokumen, portal: 'ASM' }, status: 201 }
    })
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    expect(await screen.findByText('Daftar bisnis tidak dapat dimuat')).toBeInTheDocument()
    const form = screen.getByRole('form', { name: 'Memperbaharui Data' })
    expect(within(form).getByRole('button', { name: 'Simpan' })).toBeEnabled()
  })
})
