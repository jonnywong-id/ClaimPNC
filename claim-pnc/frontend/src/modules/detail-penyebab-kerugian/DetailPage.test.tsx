import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { DetailPage } from './DetailPage'

/**
 * Baris contoh.
 *
 * Seluruh isinya KARANGAN. Tidak ada nomor polis, nama tertanggung, maupun nilai uang di
 * berkas ini (`D-69`).
 */
const AKTIF = {
  id: '990001',
  id_lama: 'COL-0001',
  id_master: '9001',
  nama_master: 'Kebakaran',
  deskripsi_kerugian: 'Kebakaran akibat hubungan arus pendek',
  kode_kehilangan: 'FIRE-01',
  status_aktif: '1',
  label_status_aktif: 'Aktif',
  bisnis: [{ id: '006', nama: 'Fire / Property' }],
}

/** Baris TIDAK AKTIF — ia yang membuktikan daftar tidak menyaringnya. */
const NONAKTIF = {
  id: '990005',
  id_lama: 'COL-0005',
  id_master: '9003',
  nama_master: 'Pencurian dan Perampokan',
  deskripsi_kerugian: 'Kehilangan tanpa unsur pemaksaan',
  kode_kehilangan: 'THEFT-02',
  status_aktif: '0',
  label_status_aktif: 'Tidak Aktif',
  bisnis: [],
}

/**
 * Baris YATIM — induknya tidak ada, sehingga `nama_master` kosong.
 *
 * Ia mungkin ada karena tidak ada foreign key yang diketahui (`R-08`), dan layar harus
 * tetap menampilkannya alih-alih menyembunyikannya.
 */
const YATIM = {
  id: '990008',
  id_lama: 'COL-0008',
  id_master: '9099',
  nama_master: '',
  deskripsi_kerugian: 'Pembatalan perjalanan karena sakit mendadak',
  kode_kehilangan: 'TRV-03',
  status_aktif: '1',
  label_status_aktif: 'Aktif',
  bisnis: [{ id: '005', nama: 'Travel' }],
}

const PILIHAN_STATUS = {
  status_aktif: [
    { kode: '1', label: 'Aktif' },
    { kode: '0', label: 'Tidak Aktif' },
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

/** defaultReply melayani daftar, pilihan, dan satu baris; mutasi dijawab pemanggil. */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.method === 'GET') {
      if (call.url.includes('/pilihan/master')) {
        return {
          body: {
            master: [{ id: '9002', nama: 'Kecelakaan Diri', label: '9002 — Kecelakaan Diri' }],
            bisnis: [],
            portal: 'ASM',
          },
        }
      }
      if (call.url.includes('/pilihan/bisnis')) {
        return {
          body: { master: [], bisnis: [{ id: '003', nama: 'Aneka' }], portal: 'ASM' },
        }
      }
      if (call.url.endsWith('/pilihan')) {
        return { body: PILIHAN_STATUS }
      }
      if (/\/detail-penyebab\/\d+$/.test(call.url)) {
        return { body: { detail: AKTIF, portal: 'ASM' } }
      }
      return { body: { detail: [AKTIF, NONAKTIF, YATIM], portal: 'ASM' } }
    }
    if (mutation) return mutation(call)
    return { body: { detail: AKTIF, portal: 'ASM' }, status: 200 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <DetailPage />
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

/** mutationCalls mengambil permintaan yang mengubah data saja. */
function mutationCalls() {
  return calls.filter((c) => c.method === 'POST' || c.method === 'PUT')
}

beforeEach(() => {
  calls = []
  window.sessionStorage.clear()
  useSession.getState().clear()
  useSelectedPortal.getState().select('ASM')
  startSession()
})

afterEach(() => {
  vi.unstubAllGlobals()
})

it('menampilkan keempat kolom grid layar lama', async () => {
  installFetch(defaultReply())
  show()

  const table = await screen.findByRole('table')
  // Keempat kolom asli, dari Section/BrowseDetailCauseOfLoss-Section.xml:9113-9920.
  for (const title of ['ID', 'Deskripsi Kerugian', 'Status Aktif', 'Kode Kehilangan']) {
    expect(within(table).getByRole('columnheader', { name: title })).toBeInTheDocument()
  }
})

it('menampilkan baris TIDAK AKTIF, bukan menyembunyikannya', async () => {
  // Report Definition pengisi grid Pega hilang dari export (`R-16`), dan bacaan yang
  // dipilih adalah TIDAK menyaring — lihat banner DetailPage.
  installFetch(defaultReply())
  show()

  expect(await screen.findByText('Kehilangan tanpa unsur pemaksaan')).toBeInTheDocument()
  expect(screen.getByText('Tidak Aktif')).toBeInTheDocument()
})

it('menandai baris yang induknya tidak ada alih-alih menampilkan sel kosong', async () => {
  installFetch(defaultReply())
  show()

  expect(await screen.findByText('9099 (induk tidak ada)')).toBeInTheDocument()
})

it('tidak menyediakan tombol Hapus di baris mana pun', async () => {
  // Layar lamanya tidak punya tombolnya, dan `D-66` melarang penghapusan fisik.
  installFetch(defaultReply())
  show()

  await screen.findByRole('table')
  expect(screen.queryByRole('button', { name: 'Hapus' })).not.toBeInTheDocument()
})

it('menyatakan ID diterbitkan sistem saat menambah baris baru', async () => {
  installFetch(defaultReply())
  const user = userEvent.setup()
  show()

  await screen.findByRole('table')
  await user.click(screen.getByRole('button', { name: 'Tambah' }))

  expect(
    await screen.findByText('Diterbitkan sistem setelah disimpan.'),
  ).toBeInTheDocument()
})

it('menyimpan baris baru tanpa mengirim field yang ditolak server', async () => {
  // `id`, `nama_master`, dan `label_status_aktif` dikirim server tetapi tidak dapat
  // dikirim balik — server menolaknya dengan `permintaan_cacat`.
  installFetch(defaultReply())
  const user = userEvent.setup()
  show()

  await screen.findByRole('table')
  await user.click(screen.getByRole('button', { name: 'Tambah' }))

  await user.type(
    await screen.findByLabelText('Deskripsi Kerugian'),
    'Tanah longsor',
  )
  await user.click(screen.getByRole('button', { name: 'Simpan' }))

  await waitFor(() => expect(mutationCalls()).toHaveLength(1))
  const sent = mutationCalls()[0]?.body as Record<string, unknown>

  expect(sent).not.toHaveProperty('id')
  expect(sent).not.toHaveProperty('nama_master')
  expect(sent).not.toHaveProperty('label_status_aktif')
  expect(sent['deskripsi_kerugian']).toBe('Tanah longsor')
})

it('baris baru diawali Status Aktif, bukan kosong', async () => {
  // Selisih terencana; alasannya pada initialValues di DetailForm.
  installFetch(defaultReply())
  const user = userEvent.setup()
  show()

  await screen.findByRole('table')
  await user.click(screen.getByRole('button', { name: 'Tambah' }))
  await user.click(await screen.findByRole('button', { name: 'Simpan' }))

  await waitFor(() => expect(mutationCalls()).toHaveLength(1))
  expect((mutationCalls()[0]?.body as Record<string, unknown>)['status_aktif']).toBe('1')
})

it('menyimpan seluruh isian kosong, karena Pega pun tidak memeriksa apa pun', async () => {
  // Kesetaraan perilaku (`P-5`); lihat detailpenyebab.Input.Check.
  installFetch(defaultReply())
  const user = userEvent.setup()
  show()

  await screen.findByRole('table')
  await user.click(screen.getByRole('button', { name: 'Tambah' }))
  await user.click(await screen.findByRole('button', { name: 'Simpan' }))

  await waitFor(() => expect(mutationCalls()).toHaveLength(1))
  expect((mutationCalls()[0]?.body as Record<string, unknown>)['deskripsi_kerugian']).toBe('')
})

it('mengirim kembali id_lama yang tidak digambar di layar', async () => {
  // OLD_D_COL_ID tidak punya isian, tetapi harus ikut tersimpan — tanpa itu tautan ke
  // sistem sebelum Pega lenyap pada setiap penyuntingan.
  installFetch(defaultReply())
  const user = userEvent.setup()
  show()

  const table = await screen.findByRole('table')
  const row = within(table).getByRole('row', {
    name: /Kebakaran akibat hubungan arus pendek/,
  })
  await user.click(within(row).getByRole('button', { name: 'Ubah' }))

  await user.click(await screen.findByRole('button', { name: 'Simpan' }))

  await waitFor(() => expect(mutationCalls()).toHaveLength(1))
  expect((mutationCalls()[0]?.body as Record<string, unknown>)['id_lama']).toBe('COL-0001')
})

it('menyunting memakai PUT ke jalur ber-ID, bukan POST', async () => {
  installFetch(defaultReply())
  const user = userEvent.setup()
  show()

  const table = await screen.findByRole('table')
  const row = within(table).getByRole('row', {
    name: /Kebakaran akibat hubungan arus pendek/,
  })
  await user.click(within(row).getByRole('button', { name: 'Ubah' }))
  await user.click(await screen.findByRole('button', { name: 'Simpan' }))

  await waitFor(() => expect(mutationCalls()).toHaveLength(1))
  expect(mutationCalls()[0]?.method).toBe('PUT')
  expect(mutationCalls()[0]?.url).toContain('/990001')
})

it('menampilkan pesan yang menyebut penomoran saat server menjawab kunci ganda', async () => {
  // Ia BUKAN kesalahan pengguna — pesannya tidak boleh menyuruh membetulkan isian.
  installFetch(
    defaultReply(() => ({
      body: {
        kode: 'kunci_detail_penyebab_sudah_ada',
        pesan: 'Nomor ID yang diterbitkan sistem sudah dipakai baris lain.',
      },
      status: 409,
    })),
  )
  const user = userEvent.setup()
  show()

  await screen.findByRole('table')
  await user.click(screen.getByRole('button', { name: 'Tambah' }))
  await user.click(await screen.findByRole('button', { name: 'Simpan' }))

  expect(await screen.findByText('Penomoran ID bentrok')).toBeInTheDocument()
})

it('tombol Tambah tetap dapat ditekan walau portal belum dipilih', async () => {
  // Ia sempat dikunci `portal === null`, dan itu membuat layar ini SATU-SATUNYA dari 18
  // modul yang tombol Tambah-nya kelabu pada sesi baru. Portal tetap ditegakkan — tetapi
  // oleh server saat menyimpan, bukan dengan mengunci tombol.
  installFetch(defaultReply())
  useSelectedPortal.getState().clear()
  show()

  const tambah = await screen.findByRole('button', { name: 'Tambah' })
  expect(tambah).not.toBeDisabled()
})

it('tombol Tambah dikunci hanya selagi form terbuka', async () => {
  installFetch(defaultReply())
  const user = userEvent.setup()
  show()

  await screen.findByRole('table')
  const tambah = screen.getByRole('button', { name: 'Tambah' })
  expect(tambah).not.toBeDisabled()

  await user.click(tambah)
  await screen.findByText('Diterbitkan sistem setelah disimpan.')
  expect(screen.getByRole('button', { name: 'Tambah' })).toBeDisabled()

  await user.click(screen.getByRole('button', { name: 'Batal' }))
  await waitFor(() =>
    expect(screen.getByRole('button', { name: 'Tambah' })).not.toBeDisabled(),
  )
})

it('menyebut portal yang belum dipilih saat penyimpanan ditolak server', async () => {
  // Isian yang sudah diketik TIDAK hilang, dan pesannya mengatakan demikian.
  installFetch(
    defaultReply(() => ({
      body: { kode: 'portal_tidak_disebut', pesan: 'Portal belum disebut.' },
      status: 400,
    })),
  )
  const user = userEvent.setup()
  show()

  await screen.findByRole('table')
  await user.click(screen.getByRole('button', { name: 'Tambah' }))
  await user.click(await screen.findByRole('button', { name: 'Simpan' }))

  expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
})

it('tidak menembak server sebelum portal dipilih', async () => {
  installFetch(defaultReply())
  useSelectedPortal.getState().clear()
  show()

  await waitFor(() => {
    expect(calls.filter((c) => c.url.includes('/master/detail-penyebab?'))).toHaveLength(0)
  })
  expect(screen.getByText('Pilih portal entitas lebih dulu.')).toBeInTheDocument()
})

it('mengirim portal aktif pada setiap permintaan daftar', async () => {
  // Tanpa ini, jawaban dapat datang dari basis data entitas lain (`R-20`).
  installFetch(defaultReply())
  show()

  await screen.findByRole('table')
  const listCall = calls.find((c) => c.url.includes('/master/detail-penyebab'))
  expect(Object.values(listCall?.header ?? {})).toContain('ASM')
})
