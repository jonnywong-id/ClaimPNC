import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { CauseOfLossPage } from './CauseOfLossPage'

const BUSINESSES = {
  portal: 'ASM',
  bisnis: [
    { id: '002', nama: 'PERSONAL ACCIDENT' },
    { id: '003', nama: 'ANEKA' },
    { id: '006', nama: 'FIRE / PROPERTY' },
  ],
}

const LIST = {
  portal: 'ASM',
  cause_of_loss: [
    { id: '1001', nama: 'KEBAKARAN', bisnis: [] },
    { id: '1002', nama: 'KEBAKARAN AKIBAT PETIR', bisnis: [] },
  ],
}

/**
 * Jawaban GET satu baris — di sinilah pemetaan bisnisnya ikut, bukan di daftar.
 *
 * Baris kedua pemetaannya SENGAJA tanpa `id`: itu bisnis yang namanya diketik bebas dan
 * tidak ada di master (`pyAllowFreeFormInput=true`), keadaan sah yang harus ditangani
 * layar.
 */
const DETAIL = {
  portal: 'ASM',
  cause_of_loss: {
    id: '1001',
    nama: 'KEBAKARAN',
    bisnis: [
      { id: '006', nama: 'FIRE / PROPERTY' },
      { id: '', nama: 'BENGKEL BARU' },
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
    if (call.method === 'GET' && call.url.includes('/col-simas-online/')) return { body: DETAIL }
    if (call.method === 'GET') return { body: LIST }
    if (mutation) return mutation(call)
    return { body: { cause_of_loss: LIST.cause_of_loss[0], portal: 'ASM' }, status: 201 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <CauseOfLossPage />
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

/** Membuka form ubah pada baris 1001 dan menunggu pemetaan bisnisnya selesai dimuat. */
async function openEditForm() {
  await screen.findByRole('table')
  await userEvent.click(screen.getByRole('button', { name: 'Ubah KEBAKARAN' }))

  const form = await screen.findByRole('form', { name: 'Memperbaharui Data Simas Online' })
  await waitFor(() => {
    expect(within(form).getByLabelText('Bisnis baris 1')).toHaveValue('FIRE / PROPERTY')
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

describe('daftar master COL Simas Online', () => {
  it('menampilkan judul dan kolom seperti layar lama', async () => {
    installFetch(defaultReply())
    show()

    expect(
      screen.getByRole('heading', { name: 'Master Cause Of Loss Simas Online' }),
    ).toBeInTheDocument()

    const table = await screen.findByRole('table')
    expect(within(table).getByRole('columnheader', { name: 'ID' })).toBeInTheDocument()
    expect(within(table).getByRole('columnheader', { name: 'Description' })).toBeInTheDocument()
  })

  // Grid menampilkan ID dan Description saja — persis seperti Pega. Kolom "induk" yang
  // sempat ada di sini ikut hilang bersama MST_COL_ID (Work Owner 2026-09-23).
  it('menampilkan ID dan namanya', async () => {
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('1002')).toBeInTheDocument()
    expect(screen.getByText('KEBAKARAN AKIBAT PETIR')).toBeInTheDocument()
  })

  // Entitas yang menjawab disebut terang-terangan: satu aplikasi melayani empat badan
  // hukum, dan "data siapa ini" tidak boleh hanya diandaikan pengguna (`R-20`).
  it('menyebutkan portal entitas yang menjawab', async () => {
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('ASM')).toBeInTheDocument()
  })

  it('mengirim portal pada setiap permintaan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    for (const call of calls) {
      expect(call.header['X-Portal']).toBe('ASM')
    }
  })

  // Tombol hapus SENGAJA tidak ada: sistem lama tidak punya satu pun DELETE terhadap
  // tabel ini, dan `D-66` melarang penghapusan fisik data bernilai bisnis.
  it('tidak menyediakan tombol hapus baris master', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    expect(within(table).queryByRole('button', { name: /hapus/i })).not.toBeInTheDocument()
  })
})

describe('susunan form mengikuti layar Pega', () => {
  // `D-13`: tata letak ditiru supaya pengguna tidak perlu belajar ulang. Urutannya
  // diambil dari urutan kemunculan di Section/Online_BrowseCauseOfLoss-Section.xml.
  it('menyusun isian dengan urutan dan label yang sama', async () => {
    installFetch(defaultReply())
    show()
    const form = await openEditForm()

    const labels = within(form)
      .getAllByText(/^(ID Kerugian|Nama Cause of loss|Bisnis)$/)
      .map((element) => element.textContent)

    expect(labels).toEqual([
      'ID Kerugian',
      'Nama Cause of loss',
      'Bisnis',
    ])
  })

  // Pega memakai judul yang sama untuk tambah maupun ubah — modusnya ditandai `pyLabel`,
  // bukan oleh judul.
  it('memakai judul yang sama pada mode tambah', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    expect(
      await screen.findByRole('form', { name: 'Memperbaharui Data Simas Online' }),
    ).toBeInTheDocument()
  })

  it('tidak menampilkan ID Kerugian saat menambah', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Memperbaharui Data Simas Online' })

    expect(within(form).queryByText('ID Kerugian')).not.toBeInTheDocument()
    expect(within(form).queryByText('(tidak dapat diubah)')).not.toBeInTheDocument()
  })

  it('menampilkan ID Kerugian tetapi tidak mengizinkannya diubah saat menyunting', async () => {
    installFetch(defaultReply())
    show()
    const form = await openEditForm()

    expect(within(form).getByText('ID Kerugian')).toBeInTheDocument()
    expect(within(form).getByText('(tidak dapat diubah)')).toBeInTheDocument()
    expect(within(form).queryByLabelText('ID Kerugian')).not.toBeInTheDocument()
  })
})

describe('isian Bisnis boleh diketik bebas', () => {
  // PERILAKU PEGA YANG DIPERTAHANKAN (`pyAllowFreeFormInput=true`): nama bisnis di luar
  // daftar tetap dapat diketik dan disimpan.
  it('mengirim nama bisnis yang diketik meski tidak ada di daftar', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Memperbaharui Data Simas Online' })

    await userEvent.type(within(form).getByLabelText('Nama Cause of loss'), 'BANJIR')
    await userEvent.click(within(form).getByRole('button', { name: 'Tambah Bisnis' }))
    await userEvent.type(within(form).getByLabelText('Bisnis baris 1'), 'BENGKEL BARU')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const posted = calls.find((c) => c.method === 'POST')
      expect(posted?.body).toEqual({
        nama: 'BANJIR',
        bisnis: ['BENGKEL BARU'],
      })
    })
  })

  it('menawarkan nama bisnis dari master sebagai saran', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Memperbaharui Data Simas Online' })
    await userEvent.click(within(form).getByRole('button', { name: 'Tambah Bisnis' }))

    const input = within(form).getByLabelText('Bisnis baris 1')
    const listID = input.getAttribute('list')
    expect(listID).toBeTruthy()

    const options = document.querySelectorAll(`#${CSS.escape(listID as string)} option`)
    expect(Array.from(options).map((o) => o.getAttribute('value'))).toEqual([
      'PERSONAL ACCIDENT',
      'ANEKA',
      'FIRE / PROPERTY',
    ])
  })

  // Pemetaan lama yang tidak punya ID tetap ditampilkan apa adanya — ia keadaan sah,
  // bukan data rusak, dan menyembunyikannya membuat pengguna kehilangan pemetaan tanpa
  // sadar saat menyimpan ulang.
  it('menampilkan pemetaan lama yang tidak punya ID', async () => {
    installFetch(defaultReply())
    show()
    const form = await openEditForm()

    expect(within(form).getByLabelText('Bisnis baris 2')).toHaveValue('BENGKEL BARU')
  })

  it('membuang baris bisnis yang dibiarkan kosong', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Memperbaharui Data Simas Online' })

    await userEvent.type(within(form).getByLabelText('Nama Cause of loss'), 'BANJIR')
    await userEvent.click(within(form).getByRole('button', { name: 'Tambah Bisnis' }))
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const posted = calls.find((c) => c.method === 'POST')
      expect((posted?.body as { bisnis: string[] }).bisnis).toEqual([])
    })
  })
})

describe('menambah dan menyunting', () => {
  // NAMA KOSONG DITERIMA — kesetaraan dengan Pega, yang menandai seluruh isiannya
  // `pyRequired=false` dan tidak punya satu pun Validate rule.
  it('mengirim nama kosong tanpa menahannya, sama seperti Pega', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Memperbaharui Data Simas Online' })
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const posted = calls.find((c) => c.method === 'POST')
      expect((posted?.body as { nama: string }).nama).toBe('')
    })
  })

  // BISNIS KEMBAR DITERIMA — grid Pega tidak punya penanda keunikan sama sekali.
  it('mengirim bisnis kembar apa adanya, sama seperti Pega', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Memperbaharui Data Simas Online' })

    await userEvent.click(within(form).getByRole('button', { name: 'Tambah Bisnis' }))
    await userEvent.type(within(form).getByLabelText('Bisnis baris 1'), 'ANEKA')
    await userEvent.click(within(form).getByRole('button', { name: 'Tambah Bisnis' }))
    await userEvent.type(within(form).getByLabelText('Bisnis baris 2'), 'ANEKA')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const posted = calls.find((c) => c.method === 'POST')
      expect((posted?.body as { bisnis: string[] }).bisnis).toEqual(['ANEKA', 'ANEKA'])
    })
  })

  // Form ditutup HANYA setelah server menjawab berhasil; menutupnya lebih dulu akan
  // membuang isian pengguna saat penyimpanan gagal.
  it('mempertahankan isian saat penyimpanan gagal', async () => {
    installFetch(
      defaultReply(() => ({
        body: { kode: 'validasi_gagal', pesan: 'Ada isian yang belum benar.', detail: [] },
        status: 422,
      })),
    )
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Memperbaharui Data Simas Online' })
    await userEvent.type(within(form).getByLabelText('Nama Cause of loss'), 'BANJIR')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      expect(screen.getByLabelText('Nama Cause of loss')).toHaveValue('BANJIR')
    })
  })

  // Server mengirim SELURUH pelanggaran sekaligus (`P-5`), dan itu hanya berguna bila
  // layar menyorotnya pada isiannya masing-masing.
  it('menyorot pelanggaran yang dilaporkan server pada isiannya', async () => {
    installFetch(
      defaultReply(() => ({
        body: {
          kode: 'validasi_gagal',
          pesan: 'Ada isian yang belum benar.',
          detail: [{ kolom: 'nama', pesan: 'Nama Cause of loss paling panjang 100 karakter.' }],
        },
        status: 422,
      })),
    )
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Memperbaharui Data Simas Online' })
    await userEvent.type(within(form).getByLabelText('Nama Cause of loss'), 'BANJIR')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(
      await within(form).findByText('Nama Cause of loss paling panjang 100 karakter.'),
    ).toBeInTheDocument()
  })

  // Inti modul ini: daftar TIDAK membawa pemetaan bisnis, sehingga baris yang dibuka
  // harus dimuat ulang dari server. Tanpa itu, form tampak seolah seluruh bisnisnya
  // sudah dihapus — dan menyimpannya benar-benar menghapusnya.
  it('memuat ulang baris yang dibuka supaya pemetaan bisnisnya ikut terbawa', async () => {
    installFetch(defaultReply())
    show()
    await openEditForm()

    expect(calls.some((c) => c.url.includes('/col-simas-online/1001'))).toBe(true)
  })

  it('menyimpan lewat PUT ke ID yang sedang disunting', async () => {
    installFetch(
      defaultReply((call) =>
        call.method === 'PUT'
          ? { body: { cause_of_loss: DETAIL.cause_of_loss, portal: 'ASM' } }
          : { body: {}, status: 500 },
      ),
    )
    show()
    const form = await openEditForm()

    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    await waitFor(() => {
      const sent = calls.find((c) => c.method === 'PUT')
      expect(sent?.url).toContain('/api/master/col-simas-online/1001')
      expect((sent?.body as { bisnis: string[] }).bisnis).toEqual([
        'FIRE / PROPERTY',
        'BENGKEL BARU',
      ])
    })
  })

  // Baris yang dihapus pengguna memang harus hilang: layar mengirim keadaan akhir grid
  // apa adanya, dan server menggantikan seluruh pemetaannya.
  it('mengirim pemetaan tanpa baris yang dihapus pengguna', async () => {
    installFetch(
      defaultReply((call) =>
        call.method === 'PUT'
          ? { body: { cause_of_loss: DETAIL.cause_of_loss, portal: 'ASM' } }
          : { body: {}, status: 500 },
      ),
    )
    show()
    const form = await openEditForm()

    await userEvent.click(within(form).getByRole('button', { name: 'Hapus bisnis baris 1' }))
    await userEvent.click(within(form).getByRole('button', { name: 'Ubah' }))

    await waitFor(() => {
      const sent = calls.find((c) => c.method === 'PUT')
      expect((sent?.body as { bisnis: string[] }).bisnis).toEqual(['BENGKEL BARU'])
    })
  })
})

describe('keadaan yang menghalangi', () => {
  it('menuntun memilih portal ketika belum dipilih', () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(screen.getByText('Portal entitas belum dipilih')).toBeInTheDocument()
  })

  // Kegagalan memuat daftar bisnis tidak menutup form dan tidak menghalangi penyimpanan:
  // nama bisnis memang boleh diketik sendiri.
  it('tetap membuka form ketika daftar bisnis gagal dimuat', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/bisnis')) {
        return { body: { kode: 'galat_internal', pesan: 'gagal' }, status: 500 }
      }
      return { body: LIST }
    })
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    expect(
      await screen.findByRole('form', { name: 'Memperbaharui Data Simas Online' }),
    ).toBeInTheDocument()
    expect(screen.getByText('Daftar bisnis tidak dapat dimuat')).toBeInTheDocument()
  })
})
