import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ClauseAIPage } from './ClauseAIPage'
import type { ClauseAI } from './types'

/**
 * Baris contoh. Seluruhnya KARANGAN — isi tabel Master Pasal AI belum pernah dilihat siapa
 * pun di tim ini, karena isi `POOLDATA.MST_PASAL_AI` belum pernah dilihat siapa pun di tim ini.
 */
const FIRE: ClauseAI = {
  id: '001',
  no_pasal: '1',
  ayat: '1',
  kejadian: 'Kebakaran akibat hubungan arus pendek pada instalasi listrik.',
}

const LIGHTNING: ClauseAI = {
  id: '004',
  no_pasal: '2',
  ayat: '1',
  kejadian: 'Petir yang menyambar langsung bangunan yang dipertanggungkan.',
}

type Call = { url: string; method: string }

let calls: Call[] = []

/** installFetch memasang fetch tiruan yang menjawab menurut pemeta yang diberikan. */
function installFetch(map: (call: Call) => { body: unknown; status?: number }) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = { url, method: init?.method ?? 'GET' }
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

/** page menyusun badan jawaban daftar beserta keterangan paginasinya. */
function page(rows: ClauseAI[], total = rows.length, number = 1) {
  return {
    pasal_ai: rows,
    paginasi: {
      halaman: number,
      ukuran_halaman: 25,
      jumlah_halaman: Math.max(1, Math.ceil(total / 25)),
      jumlah_baris: total,
    },
    portal: 'ASM',
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ClauseAIPage />
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
  useSelectedPortal.getState().select('ASM')
  startSession()
  installFetch(() => ({ body: page([FIRE, LIGHTNING]) }))
})

afterEach(() => {
  vi.unstubAllGlobals()
})

it('menggambar ketiga kolom Pega — No Pasal, Ayat, Kejadian', async () => {
  show()

  const table = await screen.findByRole('table')
  for (const title of ['No Pasal', 'Ayat', 'Kejadian']) {
    expect(within(table).getByRole('columnheader', { name: title })).toBeInTheDocument()
  }
})

it('menampilkan baris yang dikirim server', async () => {
  show()

  expect(await screen.findByText(/Kebakaran akibat hubungan arus pendek/)).toBeInTheDocument()
  expect(screen.getByText(/Petir yang menyambar langsung/)).toBeInTheDocument()
})

it('TIDAK menyediakan tombol Tambah, Simpan, Ubah, maupun Hapus', async () => {
  show()
  await screen.findByRole('table')

  // Layar lamanya baca-saja (`pyEditingMode = readOnly`); satu-satunya tombol yang ada
  // adalah Cari dan Refresh. Uji ini yang menjaga jalur tulis tidak masuk diam-diam.
  for (const label of ['Tambah', 'Simpan', 'Ubah', 'Hapus', 'Delete']) {
    expect(screen.queryByRole('button', { name: label })).not.toBeInTheDocument()
  }

  expect(screen.getByRole('button', { name: 'Cari' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Refresh' })).toBeInTheDocument()
})

it('TIDAK mencari sambil diketik — hanya saat tombol Cari ditekan', async () => {
  const user = userEvent.setup()
  show()
  await screen.findByRole('table')

  const before = calls.length
  await user.type(screen.getByLabelText('Cari'), 'petir')

  // Even `change` pada kotak cari Pega tidak punya action set sama sekali; yang menjalankan
  // pencarian adalah tombolnya. Mengetik karena itu tidak boleh menembak server.
  expect(calls.length).toBe(before)

  await user.click(screen.getByRole('button', { name: 'Cari' }))
  await waitFor(() => {
    expect(calls.at(-1)?.url).toContain('cari=petir')
  })
})

it('mengirim kata kunci ke server, bukan menyaring di peramban', async () => {
  const user = userEvent.setup()
  installFetch((call) =>
    call.url.includes('cari=petir')
      ? { body: page([LIGHTNING]) }
      : { body: page([FIRE, LIGHTNING]) },
  )
  show()
  await screen.findByRole('table')

  await user.type(screen.getByLabelText('Cari'), 'petir')
  await user.click(screen.getByRole('button', { name: 'Cari' }))

  await waitFor(() => {
    expect(screen.queryByText(/Kebakaran akibat hubungan arus pendek/)).not.toBeInTheDocument()
  })
  expect(screen.getByText(/Petir yang menyambar langsung/)).toBeInTheDocument()
})

it('Refresh mengosongkan kotak cari lebih dulu, meniru tombol Pega', async () => {
  const user = userEvent.setup()
  show()
  await screen.findByRole('table')

  const box = screen.getByLabelText('Cari')
  await user.type(box, 'petir')
  await user.click(screen.getByRole('button', { name: 'Cari' }))
  await waitFor(() => expect(calls.at(-1)?.url).toContain('cari=petir'))

  await user.click(screen.getByRole('button', { name: 'Refresh' }))

  // `setValue TempSearch.Country := ""` mendahului `refresh` pada rangkaian aksi Pega.
  expect(box).toHaveValue('')
  await waitFor(() => expect(calls.at(-1)?.url).not.toContain('cari='))
})

it('menggambar paginator dari keterangan server, bukan dari panjang senarai', async () => {
  // 60 baris di server, 25 di antaranya dikirim — tiga halaman.
  installFetch(() => ({ body: page([FIRE, LIGHTNING], 60) }))
  show()
  await screen.findByRole('table')

  expect(await screen.findByRole('button', { name: 'Halaman 3' })).toBeInTheDocument()
  expect(screen.getByRole('status')).toHaveTextContent('dari 60 baris')
})

it('berpindah halaman menembak server dengan nomor halamannya', async () => {
  const user = userEvent.setup()
  installFetch((call) => ({
    body: page([FIRE, LIGHTNING], 60, call.url.includes('halaman=2') ? 2 : 1),
  }))
  show()
  await screen.findByRole('table')

  await user.click(await screen.findByRole('button', { name: 'Halaman 2' }))

  await waitFor(() => {
    expect(calls.at(-1)?.url).toContain('halaman=2')
  })
})

it('kolomnya TIDAK dapat diurutkan — barisnya hanya satu halaman', async () => {
  show()
  const table = await screen.findByRole('table')

  // Grid Pega ber-`pySortType = NONE` pada ketiga kolom, dan mengurutkan satu halaman dari
  // tiga bukan pengurutan. Judul kolom karena itu bukan tombol.
  for (const title of ['No Pasal', 'Ayat', 'Kejadian']) {
    const header = within(table).getByRole('columnheader', { name: title })
    expect(within(header).queryByRole('button')).not.toBeInTheDocument()
  }
})

it('memakai WP_ID sebagai kunci baris, bukan gabungan No Pasal dan Ayat', async () => {
  // Dua baris dengan No Pasal DAN Ayat yang sama. Tidak ada constraint yang diketahui
  // melarangnya (`R-08`), dan kunci yang kembar membuat React menggambar ulang baris yang
  // salah — tanpa satu pun galat.
  const KEMBAR_A: ClauseAI = { ...FIRE, id: '900', kejadian: 'Kejadian A.' }
  const KEMBAR_B: ClauseAI = { ...FIRE, id: '901', kejadian: 'Kejadian B.' }
  installFetch(() => ({ body: page([KEMBAR_A, KEMBAR_B]) }))
  show()

  const table = await screen.findByRole('table')
  expect(within(table).getByText('Kejadian A.')).toBeInTheDocument()
  expect(within(table).getByText('Kejadian B.')).toBeInTheDocument()
})

it('menahan permintaan sampai portal entitas dipilih', async () => {
  useSelectedPortal.getState().clear()
  show()

  expect(await screen.findByText(/Portal entitas belum dipilih/)).toBeInTheDocument()
  expect(calls).toHaveLength(0)
})
