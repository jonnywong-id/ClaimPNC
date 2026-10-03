import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { BusinessPicker } from './BusinessPicker'
import { DetailForm } from './DetailForm'
import { DetailPage } from './DetailPage'

/**
 * Uji tambahan Detail Penyebab Kerugian: cabang galat, picker, dan status baris.
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

const ROW = {
  id: '990001',
  id_lama: 'COL-0001',
  id_master: '9001',
  nama_master: 'Kebakaran',
  deskripsi_kerugian: 'Kebakaran arus pendek',
  kode_kehilangan: 'FIRE-01',
  status_aktif: '1',
  label_status_aktif: 'Aktif',
  bisnis: [],
}

const OPTIONS = { status_aktif: [{ kode: '1', label: 'Aktif' }] }

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

beforeEach(() => {
  calls = []
  window.sessionStorage.clear()
  useSelectedPortal.getState().select('ASM')
  useSession.getState().login({
    token: 'token-contoh',
    user: {
      identitas: '90000001',
      nama: 'Contoh',
      jenis: 'KARYAWAN',
      login: 'adminpnc',
      email: '',
      perusahaan: 'ASM',
    },
    validUntil: new Date(Date.now() + 30 * 60 * 1000).toISOString(),
  })
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
})

describe('DetailPage — galat pemuatan daftar', () => {
  function listReply(reply: Reply) {
    return (call: Call): Reply => {
      if (call.url.endsWith('/pilihan')) return { body: OPTIONS }
      return reply
    }
  }

  it.each([
    [{ fail: true }, 'Server Claim PNC tidak dapat dihubungi'],
    [{ status: 400, body: { kode: 'portal_tidak_dikenal', pesan: 'x' } }, 'Portal entitas belum dipilih'],
    [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
    [
      { status: 500, body: { kode: 'galat_internal', pesan: 'Server rusak.' } },
      'Daftar detail penyebab kerugian tidak dapat dimuat',
    ],
  ])('menjelaskan galat daftar %#', async (reply, title) => {
    installFetch(listReply(reply))
    wrap(<DetailPage />)
    expect(await screen.findByText(title)).toBeInTheDocument()
  })

  it('menampilkan kalimat kosong bila jawaban daftar tidak memuat baris', async () => {
    // Jawaban tanpa `detail` dibaca sebagai daftar kosong.
    installFetch(listReply({ status: 200, body: { portal: 'ASM' } }))
    wrap(<DetailPage />)
    expect(
      await screen.findByText('Belum ada detail penyebab kerugian pada entitas ini.'),
    ).toBeInTheDocument()
  })
})

describe('DetailPage — baris dan penyuntingan', () => {
  const ROWS = [
    ROW,
    { ...ROW, id: '990002', status_aktif: '', label_status_aktif: 'Belum diisi', id_master: '', nama_master: '' },
    { ...ROW, id: '990003', status_aktif: '0', label_status_aktif: 'Tidak Aktif' },
  ]

  function reply(extra: (call: Call) => Reply | null = () => null) {
    return (call: Call): Reply => {
      const special = extra(call)
      if (special) return special
      if (call.url.endsWith('/pilihan')) return { body: OPTIONS }
      if (/\/detail-penyebab\/\d+$/.test(call.url) && call.method === 'GET') {
        return { body: { detail: ROW, portal: 'ASM' } }
      }
      return { body: { detail: ROWS, portal: 'ASM' } }
    }
  }

  it('memberi warna berbeda per status dan tanda pisah bagi induk kosong', async () => {
    installFetch(reply())
    wrap(<DetailPage />)

    const table = await screen.findByRole('table')
    expect(within(table).getByText('Aktif').className).toContain('emerald')
    expect(within(table).getByText('Belum diisi').className).toContain('slate')
    expect(within(table).getByText('Tidak Aktif').className).toContain('amber')
    expect(within(table).getByText('—')).toBeInTheDocument()
  })

  it('menampilkan pesan memuat selagi baris yang disunting belum tiba', async () => {
    installFetch(reply((call) => (/\/990001$/.test(call.url) ? { hold: true } : null)))
    const user = userEvent.setup()
    wrap(<DetailPage />)

    const table = await screen.findByRole('table')
    const row = within(table).getByRole('row', { name: /990001/ })
    await user.click(within(row).getByRole('button', { name: 'Ubah' }))

    expect(await screen.findByText('Memuat baris…')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Memperbaharui Data' })).toBeInTheDocument()
  })

  it('menyatakan baris tidak dapat dimuat bila pemuatannya gagal', async () => {
    installFetch(
      reply((call) =>
        /\/990001$/.test(call.url)
          ? { status: 500, body: { kode: 'galat_internal', pesan: 'x' } }
          : null,
      ),
    )
    const user = userEvent.setup()
    wrap(<DetailPage />)

    const table = await screen.findByRole('table')
    const row = within(table).getByRole('row', { name: /990001/ })
    await user.click(within(row).getByRole('button', { name: 'Ubah' }))

    expect(await screen.findByText('Baris ini tidak dapat dimuat')).toBeInTheDocument()
  })

  it.each([
    [{ status: 404, body: { kode: 'tidak_ditemukan', pesan: 'x' } }, 'Baris ini sudah tidak ada'],
    [{ status: 503, body: { kode: 'portal_belum_siap', pesan: 'x' } }, 'Basis data entitas ini belum tersedia'],
    [{ status: 500, body: { kode: 'galat_internal', pesan: 'Rusak sekali.' } }, 'Gagal menyimpan'],
    [{ fail: true }, 'Server Claim PNC tidak dapat dihubungi'],
  ])('menjelaskan galat simpan %#', async (failure, title) => {
    installFetch(reply((call) => (call.method === 'PUT' ? failure : null)))
    const user = userEvent.setup()
    wrap(<DetailPage />)

    const table = await screen.findByRole('table')
    const row = within(table).getByRole('row', { name: /990001/ })
    await user.click(within(row).getByRole('button', { name: 'Ubah' }))
    await user.click(await screen.findByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText(title)).toBeInTheDocument()
  })

  it('menempatkan pesan 422 di isiannya lalu menutup form setelah berhasil', async () => {
    let attempt = 0
    installFetch(
      reply((call) => {
        if (call.method !== 'POST') return null
        attempt++
        return attempt === 1
          ? {
              status: 422,
              body: {
                kode: 'validasi_gagal',
                pesan: 'Isian belum benar.',
                detail: [{ field: 'kode_kehilangan', pesan: 'Kode terlalu panjang.' }],
              },
            }
          : { status: 201, body: { detail: ROW, portal: 'ASM' } }
      }),
    )
    const user = userEvent.setup()
    wrap(<DetailPage />)

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.click(await screen.findByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Kode terlalu panjang.')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Simpan' }))
    await waitFor(() =>
      expect(screen.queryByRole('heading', { name: 'Tambah Detail Penyebab Kerugian' })).toBeNull(),
    )
    expect(calls.filter((c) => c.method === 'POST')).toHaveLength(2)
  })
})

describe('DetailForm — isian ID Master Kerugian', () => {
  function show(existing: Parameters<typeof DetailForm>[0]['existing'] = null) {
    const onSubmit = vi.fn()
    wrap(
      <DetailForm
        existing={existing}
        options={OPTIONS.status_aktif}
        fieldError={{}}
        busy={false}
        onSubmit={onSubmit}
        onCancel={() => {}}
      />,
    )
    return { onSubmit }
  }

  it('memilih master dari daftar, lalu mengosongkannya kembali', async () => {
    installFetch(() => ({
      body: { master: [{ id: '9002', nama: 'Kecelakaan Diri', label: 'x' }], bisnis: [], portal: 'ASM' },
    }))
    const user = userEvent.setup()
    const { onSubmit } = show()

    await user.type(screen.getByLabelText('ID Master Kerugian'), 'ke')
    await user.click(await screen.findByRole('button', { name: /Kecelakaan Diri/ }))

    expect(calls[0]?.url).toContain('/pilihan/master?cari=ke')
    expect(screen.getByText('Kecelakaan Diri')).toBeInTheDocument()
    expect(screen.getByText('9002')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Simpan' }))
    expect(onSubmit.mock.calls[0]?.[0]).toMatchObject({
      id_master: '9002',
      nama_master: 'Kecelakaan Diri',
    })

    await user.click(screen.getByRole('button', { name: 'Kosongkan' }))
    expect(screen.queryByText('9002')).not.toBeInTheDocument()
    expect(screen.getByText(/Ketik minimal 2 huruf, lalu pilih dari daftar. Boleh/)).toBeInTheDocument()
  })

  it('menandai induk yang sebutannya tidak ditemukan', () => {
    installFetch(() => ({ body: {} }))
    show({ ...ROW, nama_master: '' })

    expect(screen.getByText('(sebutan tidak ditemukan)')).toBeInTheDocument()
    expect(screen.getByText('induk tidak ada')).toBeInTheDocument()
    expect(screen.getByText('990001')).toBeInTheDocument()
  })

  it('menyebut tidak ada master yang cocok', async () => {
    installFetch(() => ({ body: { master: [], bisnis: [], portal: 'ASM' } }))
    const user = userEvent.setup()
    show()

    await user.type(screen.getByLabelText('ID Master Kerugian'), 'zz')
    expect(
      await screen.findByText('Tidak ada master penyebab kerugian yang cocok.'),
    ).toBeInTheDocument()
  })

  it('menyebut pencarian gagal dan menampilkan Mencari selagi menunggu', async () => {
    let mode: 'hold' | 'fail' = 'hold'
    installFetch(() => (mode === 'hold' ? { hold: true } : { status: 500, body: {} }))
    const user = userEvent.setup()
    show()

    await user.type(screen.getByLabelText('ID Master Kerugian'), 'ab')
    expect(await screen.findByText('Mencari…')).toBeInTheDocument()

    mode = 'fail'
    await user.type(screen.getByLabelText('ID Master Kerugian'), 'c')
    expect(await screen.findByRole('alert')).toHaveTextContent('Pencarian gagal')
  })

  it('tidak membuka daftar untuk kata kunci satu huruf', async () => {
    installFetch(() => ({ body: {} }))
    const user = userEvent.setup()
    show()

    await user.type(screen.getByLabelText('ID Master Kerugian'), 'a')
    expect(screen.queryByText('Mencari…')).not.toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })

  it('mengirim deskripsi, kode, dan status yang diubah', async () => {
    installFetch(() => ({ body: {} }))
    const user = userEvent.setup()
    const { onSubmit } = show()

    await user.type(screen.getByLabelText('Kode Kehilangan'), 'K-9')
    await user.selectOptions(screen.getByLabelText('Status Aktif'), '')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(onSubmit.mock.calls[0]?.[0]).toMatchObject({ kode_kehilangan: 'K-9', status_aktif: '' })
  })
})

describe('BusinessPicker', () => {
  it('menambah dari daftar, menyembunyikan yang sudah dipilih, dan menghapus', async () => {
    installFetch(() => ({
      body: {
        master: [],
        bisnis: [
          { id: '003', nama: 'Aneka' },
          { id: '006', nama: 'Fire' },
        ],
        portal: 'ASM',
      },
    }))
    const onChange = vi.fn()
    const user = userEvent.setup()
    wrap(<BusinessPicker value={[{ id: '006', nama: 'Fire' }]} onChange={onChange} />)

    await user.type(screen.getByLabelText('Tambah lini bisnis'), 'an')
    await user.click(await screen.findByRole('button', { name: /Aneka/ }))

    expect(onChange).toHaveBeenLastCalledWith([
      { id: '006', nama: 'Fire' },
      { id: '003', nama: 'Aneka' },
    ])
    // Fire sudah dipakai, sehingga tidak muncul lagi di daftar pilihan.
    await user.type(screen.getByLabelText('Tambah lini bisnis'), 'fi')
    await screen.findByRole('button', { name: /apa adanya/ })
    expect(screen.queryByRole('button', { name: /^Fire/ })).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Hapus Fire dari daftar bisnis' }))
    expect(onChange).toHaveBeenLastCalledWith([])
  })

  it('menerima nama ketikan bebas dan menandai butir tanpa kode maupun nama', async () => {
    installFetch(() => ({ body: { master: [], bisnis: [], portal: 'ASM' } }))
    const onChange = vi.fn()
    const user = userEvent.setup()
    wrap(<BusinessPicker value={[{ id: '', nama: '' }]} onChange={onChange} />)

    expect(screen.getByText('(tanpa nama)')).toBeInTheDocument()
    expect(screen.getByText('tanpa kode')).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Hapus butir tanpa nama dari daftar bisnis' }),
    ).toBeInTheDocument()

    await user.type(screen.getByLabelText('Tambah lini bisnis'), ' Lain ')
    await user.click(await screen.findByRole('button', { name: /Tambahkan “Lain” apa adanya/ }))

    expect(onChange).toHaveBeenLastCalledWith([
      { id: '', nama: '' },
      { id: '', nama: 'Lain' },
    ])
  })

  it('menyebut pencarian gagal dan Mencari selagi menunggu', async () => {
    let mode: 'hold' | 'fail' = 'hold'
    installFetch(() => (mode === 'hold' ? { hold: true } : { status: 500, body: {} }))
    const user = userEvent.setup()
    wrap(<BusinessPicker value={[]} onChange={() => {}} />)

    expect(screen.getByText(/Belum ada lini bisnis/)).toBeInTheDocument()
    await user.type(screen.getByLabelText('Tambah lini bisnis'), 'ab')
    expect(await screen.findByText('Mencari…')).toBeInTheDocument()

    mode = 'fail'
    await user.type(screen.getByLabelText('Tambah lini bisnis'), 'c')
    expect(await screen.findByRole('alert')).toHaveTextContent('Pencarian gagal')
  })

  it('menyembunyikan isian dan tombol hapus saat dikunci', () => {
    installFetch(() => ({ body: {} }))
    wrap(<BusinessPicker value={[{ id: '003', nama: 'Aneka' }]} onChange={() => {}} disabled />)

    expect(screen.getByText('003')).toBeInTheDocument()
    expect(screen.queryByLabelText('Tambah lini bisnis')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Hapus/ })).not.toBeInTheDocument()
  })
})
