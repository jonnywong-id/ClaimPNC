import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ClausePage } from './ClausePage'

const CATEGORY = {
  kategori: [
    { kode: '1', nama: 'Jaminan Polis' },
    { kode: '2', nama: 'Pengecualian' },
    // Kodenya memang KOSONG: ia cabang `else` pada ekspresi Pega, bukan nilai yang belum
    // diisi. Lihat masterpasal.CategoryNotification.
    { kode: '', nama: 'Notifikasi' },
  ],
}

const LIST = {
  portal: 'ASM',
  pasal_kerugian: [
    {
      id: '1',
      no_pasal: 'PSL-001',
      isi_pasal: 'Penanggung menjamin kerugian atas harta benda akibat kebakaran.',
      deskripsi: 'Jaminan dasar kebakaran',
      kategori: '1',
      kategori_label: 'Jaminan Polis',
      // Daftar TIDAK memuat lini bisnis; ia hanya terisi saat satu baris dibaca.
      bisnis: [],
    },
    {
      id: '2',
      no_pasal: 'PSL-002',
      isi_pasal: 'Tidak dijamin kerugian akibat keausan atau cacat tersembunyi.',
      deskripsi: 'Pengecualian umum',
      kategori: '2',
      kategori_label: 'Pengecualian',
      bisnis: [],
    },
  ],
}

const DETAIL = {
  portal: 'ASM',
  pasal_kerugian: {
    ...LIST.pasal_kerugian[1],
    bisnis: [
      { id: '2003', nama: 'Marine Cargo' },
      { id: '', nama: 'Diketik bebas' },
    ],
  },
}

const BUSINESS = {
  portal: 'ASM',
  bisnis: [{ id: '2004', nama: 'Fire / Property' }],
}

type Call = { url: string; method: string; body: unknown; header: Record<string, string> }

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
 * defaultReply melayani keempat jalur baca; mutasi dijawab pemanggil lewat `mutation`.
 *
 * Jalur `kategori` dan `bisnis` diperiksa LEBIH DULU karena keduanya berawalan sama dengan
 * jalur satu baris — persis seperti di chi, yang mencocokkan segmen statis lebih dulu.
 */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.url.startsWith('/api/master/pasal-kerugian/kategori')) return { body: CATEGORY }
    if (call.url.startsWith('/api/master/pasal-kerugian/bisnis')) return { body: BUSINESS }
    if (call.method === 'GET' && call.url === '/api/master/pasal-kerugian') return { body: LIST }
    if (call.method === 'GET') return { body: DETAIL }
    if (mutation) return mutation(call)
    return { body: DETAIL, status: 201 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ClausePage />
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

describe('daftar', () => {
  it('menampilkan judul dan keempat kolom seperti layar lama', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Master Pasal Kerugian' })).toBeInTheDocument()

    const table = await screen.findByRole('table')
    for (const title of ['No Pasal', 'Isi Pasal', 'Deskripsi', 'Kategori']) {
      expect(within(table).getByRole('columnheader', { name: title })).toBeInTheDocument()
    }
  })

  // Urutan kolom diambil dari label pada `Section/BrowsePasalDeatailMaster-Section.xml`.
  // "ISI PASAL" menunjuk `.DESCRIPTION` sedangkan "Deskripsi" menunjuk `.OLD_D_COL_ID` —
  // terbalik dari dugaan yang wajar, dan justru itu yang mudah tertukar saat menyalin.
  it('menempatkan Isi Pasal SEBELUM Deskripsi, mengikuti urutan Pega', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    const header = within(table)
      .getAllByRole('columnheader')
      .map((cell) => cell.textContent?.trim())

    expect(header.indexOf('Isi Pasal')).toBeLessThan(header.indexOf('Deskripsi'))
  })

  it('menyebut entitas yang sedang dilihat', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.getByText(/Portal entitas:/)).toHaveTextContent('ASM')
  })

  it('mengirim portal pada setiap permintaan yang menyentuh basis data entitas', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')

    const listCall = calls.find((call) => call.url === '/api/master/pasal-kerugian')
    expect(listCall?.header['X-Portal']).toBe('ASM')
  })

  // Daftar Kategori isinya milik aplikasi, bukan data entitas. Ia sengaja TIDAK
  // mengirim portal; backend pun tidak menuntutnya di rute itu.
  it('tidak mengirim portal pada daftar Kategori', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')

    const categoryCall = calls.find((call) => call.url.startsWith('/api/master/pasal-kerugian/kategori'))
    expect(categoryCall).toBeDefined()
    expect(categoryCall?.header['X-Portal']).toBeUndefined()
  })

  // Tidak satu pun permintaan yang MENYENTUH BASIS DATA ENTITAS ditembak sebelum portal
  // dipilih. Backend memang menolaknya (TKT-F6-002), tetapi menembaknya hanya untuk
  // menerima penolakan akan menampilkan pesan galat pada layar yang belum siap dibuka.
  //
  // Daftar Kategori DIKECUALIKAN, dan itu disengaja: isinya milik aplikasi, tidak
  // bergantung entitas, dan memuatnya lebih dulu membuat form siap pada saat portal
  // akhirnya dipilih.
  it('menolak memuat daftar sebelum portal dipilih', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()

    const entityCall = calls.filter(
      (call) => !call.url.startsWith('/api/master/pasal-kerugian/kategori'),
    )
    expect(entityCall).toHaveLength(0)
  })
})

describe('form', () => {
  it('memuat satu baris tersendiri saat Ubah ditekan, karena daftar tidak memuat lini bisnis', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Ubah PSL-002' }))

    expect(await screen.findByRole('form', { name: 'Ubah Pasal Kerugian' })).toBeInTheDocument()

    await waitFor(() => {
      expect(calls.some((call) => call.url === '/api/master/pasal-kerugian/2')).toBe(true)
    })

    // Lini bisnisnya baru ada setelah baris itu dibaca tersendiri.
    expect(await screen.findByText('Marine Cargo')).toBeInTheDocument()
  })

  // Butir tanpa kode ditandai terang-terangan. Di Pega ia tampak sama persis dengan yang
  // dipilih dari master, sehingga tidak ada cara membedakannya di layar.
  it('menandai lini bisnis yang diketik bebas sebagai tanpa kode', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Ubah PSL-002' }))

    expect(await screen.findByText('Diketik bebas')).toBeInTheDocument()
    expect(screen.getByText('tanpa kode')).toBeInTheDocument()
  })

  it('menolak menyimpan bila No Pasal kosong, dengan pesan layar lama', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Silahkan ISI No Pasal Terlebih Dahulu.')).toBeInTheDocument()
    expect(calls.some((call) => call.method === 'POST')).toBe(false)
  })

  // Satu-satunya isian wajib adalah No Pasal. Isi Pasal, Deskripsi, dan Kategori boleh
  // kosong seluruhnya — keputusan Work Owner 2026-09-19, "jalankan as is".
  it('menyimpan meski hanya No Pasal yang terisi', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.type(screen.getByLabelText('No Pasal'), 'PSL-010')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.some((call) => call.method === 'POST')).toBe(true)
    })

    const saved = calls.find((call) => call.method === 'POST')
    expect(saved?.body).toEqual({
      no_pasal: 'PSL-010',
      isi_pasal: '',
      deskripsi: '',
      kategori: '',
      bisnis: [],
    })
  })

  // `kategori_label` diturunkan server. Mengirimkannya ditolak 400, jadi layar tidak boleh
  // menyertakannya — dan uji ini yang menjaganya tidak menyelinap masuk kelak.
  it('tidak pernah mengirim kategori_label', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.type(screen.getByLabelText('No Pasal'), 'PSL-011')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(calls.some((call) => call.method === 'POST')).toBe(true)
    })

    const saved = calls.find((call) => call.method === 'POST')
    expect(saved?.body).not.toHaveProperty('kategori_label')
  })
})

describe('hapus', () => {
  // Konfirmasi ini TIDAK ada di Pega. Ia ditambahkan karena penghapusannya permanen dan
  // tabelnya tidak mencatat siapa maupun kapan — bukan karena aturan bisnisnya berubah.
  it('meminta konfirmasi lebih dulu, dan menyebut bahwa penghapusannya permanen', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Hapus PSL-001' }))

    expect(await screen.findByRole('alertdialog')).toHaveTextContent('permanen')
    expect(calls.some((call) => call.method === 'DELETE')).toBe(false)
  })

  it('mengirim DELETE beserta portalnya setelah dikonfirmasi', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply(() => ({ body: { id: '1', portal: 'ASM' }, status: 200 })))
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Hapus PSL-001' }))
    await user.click(await screen.findByRole('button', { name: 'Hapus permanen' }))

    await waitFor(() => {
      expect(calls.some((call) => call.method === 'DELETE')).toBe(true)
    })

    const deleted = calls.find((call) => call.method === 'DELETE')
    expect(deleted?.url).toBe('/api/master/pasal-kerugian/1')
    expect(deleted?.header['X-Portal']).toBe('ASM')
  })

  it('membatalkan konfirmasi tanpa mengirim apa pun', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Hapus PSL-001' }))
    await user.click(await screen.findByRole('button', { name: 'Batal' }))

    await waitFor(() => {
      expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument()
    })
    expect(calls.some((call) => call.method === 'DELETE')).toBe(false)
  })
})
