import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { TravelDocumentPage } from './TravelDocumentPage'

const LIST = {
  portal: 'ASM',
  total: 2,
  dokumen_travel: [
    { id: '100001', judul: 'Paspor' },
    { id: '100002', judul: 'Tiket Perjalanan' },
  ],
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

/** defaultReply melayani daftar; mutasi dijawab pemanggil lewat `mutation`. */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.method === 'GET') return { body: LIST }
    if (mutation) return mutation(call)
    return {
      body: { dokumen_travel: { id: '100003', judul: 'Visa' }, portal: 'ASM' },
      status: 201,
    }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <TravelDocumentPage />
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

describe('daftar Master Dokumen Travel', () => {
  it('menampilkan judul dan kedua kolom seperti layar lama', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Master Dokumen Travel' })).toBeInTheDocument()

    const table = await screen.findByRole('table')
    expect(within(table).getByRole('columnheader', { name: 'ID' })).toBeInTheDocument()
    expect(
      within(table).getByRole('columnheader', { name: 'Judul Dokumen Travel' }),
    ).toBeInTheDocument()
  })

  it('menampilkan baris beserta ID-nya', async () => {
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Paspor')).toBeInTheDocument()
    expect(screen.getByText('Tiket Perjalanan')).toBeInTheDocument()
    expect(screen.getByText('100001')).toBeInTheDocument()
  })

  it('menyebut entitas yang menjawab permintaan', async () => {
    // Pada aplikasi yang melayani empat badan hukum, "data siapa ini" tidak boleh hanya
    // diandaikan pengguna (R-20).
    installFetch(defaultReply())
    show()

    await screen.findByText('Paspor')
    expect(screen.getByText(/Portal entitas:/)).toBeInTheDocument()
  })

  it('mengirim portal entitas di header, bukan di URL', async () => {
    // Nilai di URL ikut tercatat di log peramban, log proxy, dan header Referer.
    installFetch(defaultReply())
    show()

    await screen.findByText('Paspor')
    const list = calls.find((c) => c.method === 'GET')
    expect(list?.header[HEADER_PORTAL]).toBe('ASM')
    expect(list?.url).not.toContain('ASM')
  })

  it('tidak menembak server sebelum portal dipilih', async () => {
    // Menembaknya hanya untuk menerima penolakan akan menampilkan pesan galat pada layar
    // yang sebenarnya belum siap dibuka.
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })

  it('tidak menyediakan tombol hapus', async () => {
    // Layar Pega pun tidak punya, dan DOCID dirujuk baris V_LST_DOC_TRAVEL beserta
    // dokumen klaim yang sudah terunggah (ADR-0012).
    installFetch(defaultReply())
    show()

    await screen.findByText('Paspor')
    expect(screen.queryByRole('button', { name: /hapus/i })).not.toBeInTheDocument()
  })
})

describe('menambah dokumen travel', () => {
  it('mengirim judul tanpa menyertakan ID', async () => {
    // ID diterbitkan server; sentinel "UnknownID" milik sistem lama tidak dibawa.
    installFetch(defaultReply())
    show()
    await screen.findByText('Paspor')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.type(screen.getByLabelText('Judul Dokumen'), 'Visa')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const save = calls.find((c) => c.method === 'POST')
      expect(save?.body).toEqual({ judul: 'Visa' })
    })
  })

  /**
   * Uji ini mengunci keputusan Work Owner 2026-09-21 di sisi layar.
   *
   * Berbeda dari Master Status Klaim, layar ini TIDAK menolak judul kosong — layar Pega
   * pun tidak. Bila kelak seseorang menambahkan `min(1)` pada skemanya, uji ini yang
   * gagal lebih dulu, sehingga penambahan itu menjadi keputusan yang disadari.
   */
  it('mengizinkan judul kosong, persis seperti layar lama', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByText('Paspor')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const save = calls.find((c) => c.method === 'POST')
      expect(save?.body).toEqual({ judul: '' })
    })
  })

  it('membiarkan form terbuka beserta isinya ketika penyimpanan gagal', async () => {
    // Menutupnya lebih dulu akan membuang isian yang baru diketik pengguna.
    installFetch(
      defaultReply(() => ({
        body: { kode: 'galat_internal', pesan: 'Terjadi kesalahan pada sistem.' },
        status: 500,
      })),
    )
    show()
    await screen.findByText('Paspor')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.type(screen.getByLabelText('Judul Dokumen'), 'Visa')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Terjadi kesalahan pada sistem')).toBeInTheDocument()
    expect(screen.getByLabelText('Judul Dokumen')).toHaveValue('Visa')
  })
})

describe('mengubah dokumen travel', () => {
  it('memuat judul yang ada dan mengirimkannya ke jalur ber-ID', async () => {
    installFetch(
      defaultReply((call) => ({
        body: { dokumen_travel: { id: '100001', judul: call.body as string }, portal: 'ASM' },
      })),
    )
    show()
    await screen.findByText('Paspor')

    await userEvent.click(screen.getByRole('button', { name: 'Ubah dokumen Paspor' }))
    const field = screen.getByLabelText('Judul Dokumen')
    expect(field).toHaveValue('Paspor')

    await userEvent.clear(field)
    await userEvent.type(field, 'Paspor / KITAS')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const save = calls.find((c) => c.method === 'PUT')
      expect(save?.url).toBe('/api/master/dokumen-travel/100001')
      expect(save?.body).toEqual({ judul: 'Paspor / KITAS' })
    })
  })

  it('menampilkan ID sebagai keterangan yang tidak dapat disunting', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByText('Paspor')

    await userEvent.click(screen.getByRole('button', { name: 'Ubah dokumen Paspor' }))
    const form = screen.getByRole('form', { name: 'Memperbaharui Data' })
    expect(within(form).getByText('(tidak dapat diubah)')).toBeInTheDocument()
  })
})

/**
 * Keempat uji di bawah mengunci kesetiaan pada teks dan perilaku layar Pega
 * (keputusan Work Owner 2026-09-21).
 *
 * Tanpa uji, keempatnya tampak seperti kelalaian bagi orang berikutnya — label yang
 * keliru, judul yang menyatakan hal yang salah, dan kotak cari yang hilang semuanya
 * "terlihat seperti bug". Uji inilah yang membuat perbaikannya menjadi keputusan yang
 * disadari, bukan rapi-rapi yang menyelinap.
 */
describe('kesetiaan pada layar Pega', () => {
  it('memakai label "ID Kerugian" seperti layar lama, walau labelnya keliru', async () => {
    // Terikat ke TempMstDocTravel.DOCID di
    // Section/BrowseMasterDocumentTravel-Section.xml:11035. "Kerugian" terbawa dari
    // rule yang diklon; D-13 menang karena isiannya read-only.
    installFetch(defaultReply())
    show()
    await screen.findByText('Paspor')

    await userEvent.click(screen.getByRole('button', { name: 'Ubah dokumen Paspor' }))
    expect(screen.getByText('ID Kerugian')).toBeInTheDocument()
  })

  it('memakai judul form yang SAMA untuk tambah dan ubah', async () => {
    // Pega memakai satu wadah ber-pyTitle "Memperbaharui Data" untuk kedua mode:
    // tombol Tambah menjalankan CNMShowInsertMstDocTravel_dt yang menghapus
    // TempMstDocTravel lalu menyetel pyLabel="Update" — syarat tampil wadah yang sama.
    installFetch(defaultReply())
    show()
    await screen.findByText('Paspor')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    expect(screen.getByRole('form', { name: 'Memperbaharui Data' })).toBeInTheDocument()
    // Isiannya kosong, karena Tambah menghapus halaman temp-nya lebih dulu.
    expect(screen.getByLabelText('Judul Dokumen')).toHaveValue('')
  })

  it('tidak menampilkan kotak pencarian', async () => {
    // Grid Pega tidak punya satu pun pySortFilterProperty yang terisi.
    installFetch(defaultReply())
    show()
    await screen.findByText('Paspor')

    expect(screen.queryByRole('searchbox')).not.toBeInTheDocument()
  })

  it('memaginasi 50 baris per halaman seperti pyPageSize', async () => {
    const many = Array.from({ length: 120 }, (_, i) => ({
      id: `1${String(i + 1).padStart(5, '0')}`,
      judul: `Dokumen ${i + 1}`,
    }))
    installFetch(() => ({ body: { portal: 'ASM', total: many.length, dokumen_travel: many } }))
    show()

    expect(await screen.findByText('Dokumen 1')).toBeInTheDocument()
    expect(screen.getByText('Dokumen 50')).toBeInTheDocument()
    expect(screen.queryByText('Dokumen 51')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Berikutnya' }))
    expect(screen.getByText('Dokumen 51')).toBeInTheDocument()
    expect(screen.queryByText('Dokumen 50')).not.toBeInTheDocument()
  })

  it('tidak menampilkan tombol halaman bila hanya ada satu halaman', async () => {
    // Tombol yang tidak pernah dapat ditekan bukan petunjuk, ia gangguan — dan pada
    // master berisi puluhan baris, itulah keadaan yang paling sering terjadi.
    installFetch(defaultReply())
    show()
    await screen.findByText('Paspor')

    expect(screen.queryByRole('button', { name: 'Berikutnya' })).not.toBeInTheDocument()
  })
})
