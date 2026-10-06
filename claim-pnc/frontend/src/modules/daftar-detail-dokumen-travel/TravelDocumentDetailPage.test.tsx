import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { TravelDocumentDetailPage } from './TravelDocumentDetailPage'

const DOCUMENTS = {
  portal: 'ASM',
  dokumen: [
    { id: '100001', nama: 'Paspor' },
    { id: '100004', nama: 'Laporan Kehilangan Bagasi' },
  ],
}

/**
 * Jawaban daftar.
 *
 * Kelima kolom inilah SELURUH isi barisnya — tidak ada pembatasan Plan dan Jaminan, dan
 * karena itu tidak ada yang perlu dimuat ulang saat form dibuka.
 */
const LIST = {
  portal: 'ASM',
  total: 2,
  detail_dokumen_travel: [
    {
      id: '00001',
      id_dokumen: '100001',
      nama_dokumen: 'Paspor',
      status_wajib: true,
      minimal_unggah: 1,
    },
    {
      id: '00003',
      id_dokumen: '100004',
      nama_dokumen: 'Laporan Kehilangan Bagasi',
      status_wajib: false,
      minimal_unggah: 2,
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

/** defaultReply melayani daftar dan daftar pilihan ID Dokumen. */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.url.startsWith('/api/master/dokumen-travel-pilihan')) return { body: DOCUMENTS }
    if (call.method === 'GET') return { body: LIST }
    if (mutation) return mutation(call)
    return {
      body: { portal: 'ASM', detail_dokumen_travel: LIST.detail_dokumen_travel[0] },
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
        <TravelDocumentDetailPage />
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

/** Membuka form ubah pada baris 00003. */
async function openEditForm() {
  await screen.findByRole('table')
  await userEvent.click(screen.getByRole('button', { name: 'Ubah Laporan Kehilangan Bagasi' }))
  return screen.findByRole('form', { name: 'Detail Dokumen Travel' })
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

describe('daftar detail dokumen travel', () => {
  it('menampilkan judul dan kelima kolom seperti layar lama', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Detail Dokumen Travel' })).toBeInTheDocument()

    const table = await screen.findByRole('table')
    for (const title of ['ID', 'ID Dokumen', 'Nama Dokumen', 'Status Wajib', 'Minimal Unggah']) {
      expect(within(table).getByRole('columnheader', { name: title })).toBeInTheDocument()
    }
  })

  it('menampilkan status wajib sebagai Ya dan Tidak, bukan 1 dan 0', async () => {
    // Teks itu mengikuti `Activity/BrowseDocTravel-Act.xml`, yang mengubah STSWAJIB
    // menjadi "Ya"/"Tidak" sebelum menampilkannya. Angkanya tetap bentuk penyimpanan.
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getByText('Ya')).toBeInTheDocument()
    expect(within(table).getByText('Tidak')).toBeInTheDocument()
  })

  // Entitas yang menjawab disebut terang-terangan: satu aplikasi melayani empat badan
  // hukum, dan "data siapa ini" tidak boleh hanya diandaikan pengguna (`R-20`).
  it('menyebutkan portal entitas yang menjawab', async () => {
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('ASM')).toBeInTheDocument()
  })

  it('mengirim portal yang dipilih pada setiap permintaan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(calls.length).toBeGreaterThan(0)
    for (const call of calls) {
      expect(call.header['X-Portal']).toBe('ASM')
    }
  })
})

describe('form detail dokumen travel', () => {
  it('mengisi seluruh isian dari baris yang dibuka', async () => {
    installFetch(defaultReply())
    show()

    const form = await openEditForm()
    expect(within(form).getByLabelText('ID Dokumen')).toHaveValue('100004')
    expect(within(form).getByLabelText('Nama Dokumen')).toHaveValue('Laporan Kehilangan Bagasi')
    // Teks '2', bukan angka 2: isiannya `pxTextInput` di Pega, dan ditiru sebagai isian
    // teks di sini (lihat komentarnya di TravelDocumentDetailForm).
    expect(within(form).getByLabelText('Minimal Unggah')).toHaveValue('2')
    expect(within(form).getByLabelText('Status Wajib')).toHaveValue('0')
  })

  // Uji ini menjaga sebuah KEPUTUSAN, bukan sebuah perhitungan.
  //
  // `Section/BrowseDocumentTravel-Section.xml:3731` memuat grid berulang Plan dan
  // Jaminan tanpa kondisi yang menyembunyikannya, sehingga siapa pun yang membaca export
  // akan menyimpulkan form ini punya grid itu. Grid itu TIDAK ADA di aplikasi Pega yang
  // berjalan — Work Owner memeriksa layarnya langsung dan menetapkannya 2026-10-03, dan
  // modul ini sempat membangunnya lalu mencabutnya.
  it('tidak memuat grid Plan dan Jaminan', async () => {
    installFetch(defaultReply())
    show()

    const form = await openEditForm()
    expect(within(form).queryByText(/Plan dan Jaminan/)).not.toBeInTheDocument()
    expect(within(form).queryByRole('button', { name: 'Tambah Plan' })).not.toBeInTheDocument()
  })

  it('tidak menembak rute plan-travel yang sudah dicabut', async () => {
    installFetch(defaultReply())
    show()

    await openEditForm()
    expect(calls.some((call) => call.url.startsWith('/api/master/plan-travel'))).toBe(false)
  })

  it('mengirim keempat isian apa adanya', async () => {
    let saved: unknown
    installFetch(
      defaultReply((call) => {
        saved = call.body
        return { body: { portal: 'ASM', detail_dokumen_travel: LIST.detail_dokumen_travel[0] }, status: 201 }
      }),
    )
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Detail Dokumen Travel' })

    await userEvent.type(within(form).getByLabelText('ID Dokumen'), '100001')
    await userEvent.type(within(form).getByLabelText('Nama Dokumen'), 'Paspor')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(saved).toBeDefined())
    expect(saved).toEqual({
      id_dokumen: '100001',
      nama_dokumen: 'Paspor',
      status_wajib: false,
      minimal_unggah: 0,
    })
  })

  it('menerima isian kosong karena layar ini tanpa validasi', async () => {
    // Work Owner menetapkan 2026-09-21 layar ini meniru Pega apa adanya. Uji ini ada
    // supaya ketiadaan validasi menjadi keputusan yang terlihat, bukan kelalaian yang
    // kelak "diperbaiki" seseorang tanpa menyadari ia mengubah perilaku.
    let saved: unknown
    installFetch(
      defaultReply((call) => {
        saved = call.body
        return { body: { portal: 'ASM', detail_dokumen_travel: LIST.detail_dokumen_travel[0] }, status: 201 }
      }),
    )
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Detail Dokumen Travel' })
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(saved).toBeDefined())
    expect(saved).toMatchObject({ id_dokumen: '', nama_dokumen: '', minimal_unggah: 0 })
  })

  it('menolak Minimal Unggah yang bukan angka tanpa mengirim apa pun', async () => {
    // Ini SATU-SATUNYA pemeriksaan isian di layar ini, dan ia bukan aturan bisnis
    // melainkan bentuk kolomnya: teks yang bukan angka akan ditolak Oracle sebagai galat
    // yang tidak dapat dibaca pengguna.
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Detail Dokumen Travel' })

    const field = within(form).getByLabelText('Minimal Unggah')
    await userEvent.clear(field)
    await userEvent.type(field, '-2')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    // Dicari lewat TEKSNYA, bukan lewat `role="alert"`: komponen Field bersama tidak
    // menandai pesan galatnya dengan peran itu — hanya ComboField yang menandainya.
    // Ketidakseragaman itu ada di pustaka komponen bersama, bukan di modul ini, dan
    // dicatat di docs/catatan-pengembangan.md alih-alih diperbaiki sepihak dari sini.
    expect(
      await within(form).findByText('Minimal Unggah hanya boleh berisi angka.'),
    ).toBeInTheDocument()
    expect(calls.filter((call) => call.method === 'POST')).toHaveLength(0)
  })

  it('tidak menutup form ketika penyimpanan gagal', async () => {
    // Menutupnya akan membuang isian pengguna — dan pada form yang isinya baru diketik,
    // itu berarti mengetik ulang dari awal.
    installFetch(
      defaultReply(() => ({
        body: { kode: 'galat_internal', pesan: 'Terjadi kesalahan.' },
        status: 500,
      })),
    )
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    const form = await screen.findByRole('form', { name: 'Detail Dokumen Travel' })
    await userEvent.type(within(form).getByLabelText('Nama Dokumen'), 'Paspor')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await screen.findByText('Terjadi kesalahan pada sistem')
    expect(within(form).getByLabelText('Nama Dokumen')).toHaveValue('Paspor')
  })

  it('tetap dapat menyimpan meski daftar dokumen gagal dimuat', async () => {
    // Kode dokumen memang boleh diketik sendiri, sehingga yang hilang saat daftarnya
    // gagal dimuat hanyalah kenyamanan memilih.
    let saved: unknown
    installFetch((call) => {
      if (call.url.startsWith('/api/master/dokumen-travel-pilihan')) {
        return { body: { kode: 'galat_internal', pesan: 'Gagal.' }, status: 500 }
      }
      if (call.method === 'GET') return { body: LIST }
      saved = call.body
      return { body: { portal: 'ASM', detail_dokumen_travel: LIST.detail_dokumen_travel[0] }, status: 201 }
    })
    show()

    await screen.findByRole('table')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await screen.findByText('Daftar dokumen travel tidak dapat dimuat')

    const form = await screen.findByRole('form', { name: 'Detail Dokumen Travel' })
    await userEvent.type(within(form).getByLabelText('Nama Dokumen'), 'Paspor')
    await userEvent.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(saved).toBeDefined())
    expect(saved).toMatchObject({ nama_dokumen: 'Paspor' })
  })
})

describe('layar tanpa portal', () => {
  it('menuntun pengguna memilih entitas, bukan menampilkan galat', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    // Tidak satu pun permintaan ditembakkan: layar yang menuntun lebih berguna daripada
    // pesan galat atas penolakan yang sudah dapat diperkirakan.
    expect(calls).toHaveLength(0)
  })
})
