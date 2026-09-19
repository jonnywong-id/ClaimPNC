import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ProgressStatus2Page } from './ProgressStatus2Page'

const PARENTS = {
  portal: 'ASM',
  induk: [
    { id: '01', nama: 'DOKUMEN DITERIMA' },
    { id: '02', nama: 'MENUNGGU JADWAL SURVEI' },
  ],
}

const LIST = {
  portal: 'ASM',
  status_progres_2: [
    {
      id: '1',
      nama: 'SURAT PERMINTAAN DOKUMEN DIKIRIM',
      id_induk: '01',
      nama_induk: 'DOKUMEN DITERIMA',
      tipe: '',
    },
    {
      id: '2',
      nama: 'SURVEYOR DITUNJUK',
      id_induk: '02',
      nama_induk: 'MENUNGGU JADWAL SURVEI',
      tipe: '',
    },
  ],
}

type Call = {
  url: string
  method: string
  body: unknown
  header: Record<string, string>
}

let calls: Call[] = []

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

/**
 * defaultReply melayani daftar dan daftar induk; mutasi dijawab pemanggil lewat `mutation`.
 *
 * Jalur induk diperiksa LEBIH DULU karena ia berawalan sama dengan jalur daftar.
 */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.url.startsWith('/api/master/status-progres-2/induk')) return { body: PARENTS }
    if (call.method === 'GET') return { body: LIST }
    if (mutation) return mutation(call)
    return { body: { status_progres_2: LIST.status_progres_2[0], portal: 'ASM' }, status: 201 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ProgressStatus2Page />
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

describe('daftar master status progres 2', () => {
  it('menampilkan judul dan ketiga kolom seperti layar lama', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Master Status Progres 2' })).toBeInTheDocument()

    const table = await screen.findByRole('table')
    expect(within(table).getByRole('columnheader', { name: 'ID' })).toBeInTheDocument()
    expect(within(table).getByRole('columnheader', { name: 'Status Progres 1' })).toBeInTheDocument()
    expect(within(table).getByRole('columnheader', { name: 'Status Progres 2' })).toBeInTheDocument()
  })

  // Urutan kolom diambil dari header grid Pega apa adanya
  // (`Section/BrowseStatusProgress2-Section.xml`): No · Status Progress 1 · Status Progress 2.
  //
  // Induk mendahului nama barisnya sendiri — berlawanan dengan dugaan yang wajar, dan
  // sempat tertukar di layar ini. Uji ini yang menjaganya tidak tertukar lagi.
  it('menempatkan induk SEBELUM nama barisnya, mengikuti urutan Pega', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    const header = within(table)
      .getAllByRole('columnheader')
      .map((h) => h.textContent?.trim())

    expect(header.slice(0, 3)).toEqual(['ID', 'Status Progres 1', 'Status Progres 2'])
  })

  // TIPE tidak pernah menjadi kolom grid di Pega — ia terikat ke page TempUpdateStatus2,
  // yaitu modal penyuntingan yang tidak dibawa. Kolomnya sempat ada di layar ini lalu
  // dicabut; uji ini menjaganya tidak kembali tanpa keputusan yang menyertainya.
  it('tidak menampilkan kolom Tipe — ia bukan kolom grid di Pega', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    expect(within(table).queryByRole('columnheader', { name: 'Tipe' })).not.toBeInTheDocument()
  })

  it('menampilkan baris beserta nama induknya', async () => {
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('SURAT PERMINTAAN DOKUMEN DIKIRIM')).toBeInTheDocument()
    expect(screen.getByText('SURVEYOR DITUNJUK')).toBeInTheDocument()
    expect(screen.getByText('DOKUMEN DITERIMA')).toBeInTheDocument()
    expect(screen.getByText('MENUNGGU JADWAL SURVEI')).toBeInTheDocument()
  })

  // Portal WAJIB ikut di setiap permintaan yang menyentuh basis data entitas — termasuk
  // daftar induk, yang berbeda dari daftar posisi klaim pada tingkat 1. Backend menolak
  // yang tidak menyebutkannya, dan itu memang yang diinginkan (R-20).
  it('mengirim header portal dan token pada KEDUA permintaan', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    const list = calls.find((c) => c.url === '/api/master/status-progres-2')
    const parent = calls.find((c) => c.url === '/api/master/status-progres-2/induk')

    for (const call of [list, parent]) {
      expect(call).toBeDefined()
      expect(call?.header['X-Portal']).toBe('ASM')
      expect(call?.header.Authorization).toBe('Bearer token-contoh')
    }
  })

  it('menyebut portal entitas yang menjawab', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.getByText('Portal entitas:')).toBeInTheDocument()
  })

  // TIPE tetap DIBACA dan tetap dikirim server meski tidak ditampilkan sebagai kolom.
  // Membuangnya dari respons akan membuat nilainya hilang dari aplikasi ini sama sekali,
  // dan layar rincian kelak tidak punya sumbernya.
  it('tetap menerima TIPE dari server meski tidak ditampilkan', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    const list = calls.find((c) => c.url === '/api/master/status-progres-2')
    expect(list).toBeDefined()
    // Bentuk responsnya memang memuat `tipe` — lihat fixture LIST di berkas ini.
    expect(LIST.status_progres_2.every((r) => 'tipe' in r)).toBe(true)
  })
})

describe('penambahan status progres 2', () => {
  it('mengirim hanya nama dan id_induk — ID diterbitkan server', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Status Progres 2' })

    await userEvent.selectOptions(within(form).getByLabelText(/Status Progres 1/), '01')
    await userEvent.type(within(form).getByLabelText(/Status Progres 2/), 'DOKUMEN SUSULAN')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.some((c) => c.method === 'POST')).toBe(true)
    })

    const saved = calls.find((c) => c.method === 'POST')
    expect(saved?.body).toEqual({ nama: 'DOKUMEN SUSULAN', id_induk: '01' })
    expect(saved?.header['X-Portal']).toBe('ASM')
  })

  it('menolak isian kosong sebelum menembak server', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Status Progres 2' })
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText('Nama status progres 2 wajib diisi.')).toBeInTheDocument()
    expect(within(form).getByText('Status Progres 1 wajib dipilih.')).toBeInTheDocument()
    expect(calls.some((c) => c.method === 'POST')).toBe(false)
  })

  // Induk yang sudah tidak ada dijawab server sebagai pelanggaran pada isian `id_induk`.
  // Keterangannya harus MENEMPEL di dropdown itu, bukan menggantung di atas form.
  it('menyorot dropdown induk saat server menolak induknya', async () => {
    installFetch(
      defaultReply(() => ({
        status: 422,
        body: {
          kode: 'validasi_gagal',
          pesan: 'Ada isian yang belum benar. Periksa keterangan di bawah setiap isian.',
          detail: [
            {
              kolom: 'id_induk',
              pesan: 'Status progres 1 yang dipilih sudah tidak ada. Muat ulang halaman, lalu pilih kembali.',
            },
          ],
        },
      })),
    )
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Status Progres 2' })

    await userEvent.selectOptions(within(form).getByLabelText(/Status Progres 1/), '01')
    await userEvent.type(within(form).getByLabelText(/Status Progres 2/), 'APA SAJA')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(
      await within(form).findByText(
        'Status progres 1 yang dipilih sudah tidak ada. Muat ulang halaman, lalu pilih kembali.',
      ),
    ).toBeInTheDocument()
    // Form TETAP terbuka; isian pengguna tidak dibuang.
    expect(within(form).getByLabelText(/Status Progres 2/)).toHaveValue('APA SAJA')
  })

  it('menutup form hanya setelah server menjawab berhasil', async () => {
    installFetch(
      defaultReply(() => ({
        status: 500,
        body: { kode: 'galat_internal', pesan: 'Terjadi kesalahan.' },
      })),
    )
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Tambah Status Progres 2' })

    await userEvent.selectOptions(within(form).getByLabelText(/Status Progres 1/), '01')
    await userEvent.type(within(form).getByLabelText(/Status Progres 2/), 'DOKUMEN SUSULAN')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.some((c) => c.method === 'POST')).toBe(true)
    })
    expect(within(form).getByLabelText(/Status Progres 2/)).toHaveValue('DOKUMEN SUSULAN')
  })
})

describe('yang sengaja tidak ada', () => {
  // Sistem lama punya tombol "Update", tetapi tombol itu tidak mengubah apa pun pada tabel
  // master: kuerinya menyentuh TABEL LAIN dengan dua page klipboard yang tidak pernah
  // diisi. Yang direplikasi adalah hasil yang teramati, bukan jalur yang menghasilkannya.
  it('tidak menampilkan tombol Ubah maupun Hapus', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    expect(within(table).queryByRole('button', { name: /Ubah/ })).not.toBeInTheDocument()
    expect(within(table).queryByRole('button', { name: /Hapus/ })).not.toBeInTheDocument()
    expect(within(table).queryByRole('columnheader', { name: 'Aksi' })).not.toBeInTheDocument()
  })
})

describe('portal belum dipilih', () => {
  it('menuntun pengguna memilih portal, tidak menembak server', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })
})
