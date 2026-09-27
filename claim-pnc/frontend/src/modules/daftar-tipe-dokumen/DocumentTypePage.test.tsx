import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { HEADER_PORTAL } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { DocumentTypePage } from './DocumentTypePage'

const LIST = {
  portal: 'ASM',
  total: 3,
  tipe_dokumen: [
    { id: '10001', tipe_dokumen: 'Dokumen Registrasi', status_proses: 'Register' },
    { id: '10002', tipe_dokumen: 'Dokumen Survey', status_proses: 'Survey' },
    // Baris ketiga sengaja berisian kosong: keduanya memang dapat tersimpan pada layar
    // tanpa validasi, dan tampilannya harus tetap terbaca.
    { id: '10003', tipe_dokumen: '', status_proses: '' },
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
      body: {
        tipe_dokumen: { id: '10004', tipe_dokumen: 'Dokumen Komite', status_proses: 'Komite' },
        portal: 'ASM',
      },
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
        <DocumentTypePage />
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

describe('daftar Tipe Dokumen', () => {
  it('menampilkan judul dan ketiga kolom seperti layar lama', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Daftar Tipe Dokumen' })).toBeInTheDocument()

    const table = await screen.findByRole('table')
    expect(within(table).getByRole('columnheader', { name: 'ID' })).toBeInTheDocument()
    expect(within(table).getByRole('columnheader', { name: 'Tipe Dokumen' })).toBeInTheDocument()
    expect(within(table).getByRole('columnheader', { name: 'Status Proses' })).toBeInTheDocument()
  })

  // Pada aplikasi yang melayani empat badan hukum, "data siapa ini" tidak boleh hanya
  // diandaikan pengguna (`R-20`).
  it('menyebut entitas yang menjawab, dan mengirim portalnya di header', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.getByText('ASM')).toBeInTheDocument()
    expect(calls[0]?.header[HEADER_PORTAL]).toBe('ASM')
  })

  // Tanpa kotak cari — meniru grid Pega apa adanya (Work Owner 2026-09-21).
  it('tidak menampilkan kotak cari', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.queryByRole('searchbox')).not.toBeInTheDocument()
  })

  // Baris berisian kosong memang sah; ia ditandai supaya tidak tampak seperti baris rusak.
  it('menandai baris yang tersimpan tanpa nama, bukan membiarkannya kosong', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.getByText('(tanpa nama)')).toBeInTheDocument()
  })

  it('tidak menembak server sebelum portal dipilih', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })
})

describe('menambah tipe dokumen', () => {
  it('mengirim kedua isian tanpa ID — ID diterbitkan server', async () => {
    installFetch(defaultReply())
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    await user.type(screen.getByLabelText('Jenis Dokumen'), 'Dokumen Komite')
    await user.type(screen.getByLabelText('Status Proses'), 'Komite')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const posted = calls.find((call) => call.method === 'POST')
      expect(posted?.body).toEqual({ tipe_dokumen: 'Dokumen Komite', status_proses: 'Komite' })
    })
  })

  // Mengunci keputusan Work Owner 2026-09-21 pada tingkat layar: tanpa validasi berarti
  // form kosong pun benar-benar terkirim, bukan ditahan di peramban.
  it('mengirim isian kosong tanpa menahannya — layar ini tanpa validasi', async () => {
    installFetch(defaultReply())
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const posted = calls.find((call) => call.method === 'POST')
      expect(posted?.body).toEqual({ tipe_dokumen: '', status_proses: '' })
    })
  })

  // Form ditutup HANYA setelah server menjawab berhasil; isian pengguna tidak dibuang saat
  // penyimpanan gagal.
  it('mempertahankan isian ketika penyimpanan gagal', async () => {
    installFetch(
      defaultReply(() => ({
        body: { kode: 'galat_internal', pesan: 'Terjadi kesalahan.' },
        status: 500,
      })),
    )
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.type(screen.getByLabelText('Jenis Dokumen'), 'Dokumen Komite')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Terjadi kesalahan pada sistem')).toBeInTheDocument()
    expect(screen.getByLabelText('Jenis Dokumen')).toHaveValue('Dokumen Komite')
  })
})

describe('mengubah tipe dokumen', () => {
  it('memuat isi baris ke form dan menampilkan ID yang tidak dapat diubah', async () => {
    installFetch(defaultReply())
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Ubah tipe dokumen Dokumen Survey' }))

    expect(screen.getByLabelText('Jenis Dokumen')).toHaveValue('Dokumen Survey')
    expect(screen.getByLabelText('Status Proses')).toHaveValue('Survey')
    expect(screen.getByText('(tidak dapat diubah)')).toBeInTheDocument()
  })

  it('menyimpan lewat PUT ke ID barisnya', async () => {
    installFetch(
      defaultReply((call) =>
        call.method === 'PUT'
          ? {
              body: {
                tipe_dokumen: {
                  id: '10002',
                  tipe_dokumen: 'Dokumen Survey Lapangan',
                  status_proses: 'Survey',
                },
                portal: 'ASM',
              },
            }
          : { body: LIST },
      ),
    )
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Ubah tipe dokumen Dokumen Survey' }))

    const nameField = screen.getByLabelText('Jenis Dokumen')
    await user.clear(nameField)
    await user.type(nameField, 'Dokumen Survey Lapangan')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const saved = calls.find((call) => call.method === 'PUT')
      expect(saved?.url).toContain('/api/master/tipe-dokumen/10002')
      expect(saved?.body).toEqual({
        tipe_dokumen: 'Dokumen Survey Lapangan',
        status_proses: 'Survey',
      })
    })
  })

  it('memberi tahu ketika barisnya sudah diubah petugas lain', async () => {
    installFetch(
      defaultReply(() => ({
        body: { kode: 'tipe_dokumen_tidak_ditemukan', pesan: 'Tidak ditemukan.' },
        status: 404,
      })),
    )
    const user = userEvent.setup()
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Ubah tipe dokumen Dokumen Survey' }))
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Tipe dokumen ini sudah tidak ada')).toBeInTheDocument()
  })
})
