import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { LaporanHasilAIPage } from './LaporanHasilAIPage'

/**
 * Jawaban contoh — bentuknya sama persis dengan yang dikirim backend.
 *
 * Isinya mewakili keadaan yang membedakan satu baris dari yang lain, dan tiga di antaranya
 * adalah hal yang paling mudah dikira kerusakan di layar ini:
 *
 *	baris 1   klaim bernomor, AI dan komite sependapat
 *	baris 2   objek KEDUA pada jenjang yang sama — satu klaim, dua baris
 *	baris 3   jenjang komite KEDUA — nomor klaimnya KOSONG
 *	baris 4   komite MENUNGGU, sehingga Total < jumlah baris
 *
 * Kelima kolom yang SELALU kosong dikirim sebagai teks kosong, bukan dihilangkan — persis
 * seperti backend mengirimkannya.
 */
const REPORT = {
  ringkasan: [
    { keputusan: 'Komite', total: 3, diterima: 3, ditolak: 0, menunggu: 1 },
    { keputusan: 'AI', total: 4, diterima: 3, ditolak: 1, menunggu: 0 },
  ],
  baris: [
    {
      id: 'KMT-000101|1|OBJ-01|CVG-01',
      no_klaim: 'PNCN.26.0001',
      nama_object: '',
      komite_status: 'DITERIMA',
      kode_komite_status: '1',
      tanggal_komite: '2026-09-03',
      ai_status: 'DITERIMA',
      tanggal_ai: '2026-09-01',
      note_ai_terima: '',
      note_ai_tolak: '',
      coverage_final: '',
      kategori_kronologi: '',
    },
    {
      id: 'KMT-000101|1|OBJ-02|CVG-01',
      no_klaim: 'PNCN.26.0001',
      nama_object: '',
      komite_status: 'DITERIMA',
      kode_komite_status: '1',
      tanggal_komite: '2026-09-03',
      ai_status: 'DITOLAK',
      tanggal_ai: '2026-09-01',
      note_ai_terima: '',
      note_ai_tolak: '',
      coverage_final: '',
      kategori_kronologi: '',
    },
    {
      id: 'KMT-000101|2|OBJ-01|CVG-01',
      no_klaim: '',
      nama_object: '',
      komite_status: 'DITERIMA',
      kode_komite_status: '1',
      tanggal_komite: '2026-09-05',
      ai_status: 'DITERIMA',
      tanggal_ai: '2026-09-01',
      note_ai_terima: '',
      note_ai_tolak: '',
      coverage_final: '',
      kategori_kronologi: '',
    },
    {
      id: 'KMT-000103|1|OBJ-01|CVG-03',
      no_klaim: 'PNCN.26.0003',
      nama_object: '',
      komite_status: 'MENUNGGU',
      kode_komite_status: '0',
      tanggal_komite: '2026-09-10',
      ai_status: '',
      tanggal_ai: null,
      note_ai_terima: '',
      note_ai_tolak: '',
      coverage_final: '',
      kategori_kronologi: '',
    },
  ],
  paginasi: { halaman: 1, ukuran: 50, total: 4, total_halaman: 1 },
  filter: { dari: '2026-09-01', sampai: '2026-09-30' },
  portal: 'ASM',
}

/** Jawaban 422 yang dikirim backend saat kedua isian tanggal kosong. */
const VALIDATION_ERROR = {
  kode: 'validasi_gagal',
  pesan: 'Permintaan belum benar. Perbaiki yang ditandai lalu coba lagi.',
  detail: [
    { field: 'dari', pesan: 'Tgl Input Dari belum diisi.' },
    { field: 'sampai', pesan: 'Tgl Input Sampai belum diisi.' },
  ],
}

type Call = { url: string; header: Record<string, string> }

let calls: Call[] = []

function installFetch(options: { failValidation?: boolean } = {}) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, header: (init?.headers as Record<string, string>) ?? {} })

    if (url.includes('/ekspor')) {
      return Promise.resolve(
        new Response('No Klaim\nPNCN.26.0001\n', {
          status: 200,
          headers: {
            'Content-Type': 'text/csv; charset=utf-8',
            'Content-Disposition': 'attachment; filename="laporan-hasil-ai.csv"',
          },
        }),
      )
    }

    if (options.failValidation) {
      return Promise.resolve(
        new Response(JSON.stringify(VALIDATION_ERROR), {
          status: 422,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    }

    return Promise.resolve(
      new Response(JSON.stringify(REPORT), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/laporan-hasil-ai']}>
        <LaporanHasilAIPage />
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

/** Mengisi kedua isian tanggal lalu menekan Cari Data. */
async function search(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText('Tgl Input Dari'), '2026-09-01')
  await user.type(screen.getByLabelText('Tgl Input Sampai'), '2026-09-30')
  await user.click(screen.getByRole('button', { name: 'Cari Data' }))
}

beforeEach(() => {
  calls = []
  window.sessionStorage.clear()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
  startSession()
  useSelectedPortal.getState().select('ASM')

  // simpanBerkas membuat tautan lalu menekannya; jsdom tidak dapat bernavigasi.
  vi.spyOn(URL, 'createObjectURL').mockReturnValue('blob:contoh')
  vi.spyOn(URL, 'revokeObjectURL').mockImplementation(() => {})
  vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {})
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('layar Laporan Hasil AI', () => {
  // Layar lama pun tidak menampilkan apa pun sebelum "Cari Data" ditekan. Menembak server
  // pada pemuatan pertama hanya menghasilkan 422 pada layar yang belum disentuh siapa pun.
  it('tidak menembak server sebelum Cari Data ditekan', async () => {
    installFetch()
    show()

    expect(
      await screen.findByRole('heading', { name: 'Laporan Hasil Data AI', level: 1 }),
    ).toBeVisible()
    expect(calls).toHaveLength(0)
    expect(screen.getByRole('button', { name: 'Export To Excel' })).toBeDisabled()
  })

  // Label keduanya disalin APA ADANYA dari Pega, termasuk yang menyesatkan: yang disaring
  // sebenarnya TANGGALKOMITE. Keputusan Work Owner 2026-09-26.
  it('memakai label isian persis seperti layar lama', async () => {
    installFetch()
    show()

    expect(await screen.findByLabelText('Tgl Input Dari')).toBeVisible()
    expect(screen.getByLabelText('Tgl Input Sampai')).toBeVisible()
  })

  it('mengirim kedua tanggal beserta header portal saat Cari Data ditekan', async () => {
    installFetch()
    const user = userEvent.setup()
    show()

    await search(user)

    await waitFor(() => expect(calls.length).toBeGreaterThan(0))
    expect(calls[0]?.url).toContain('/api/laporan-hasil-ai?')
    expect(calls[0]?.url).toContain('dari=2026-09-01')
    expect(calls[0]?.url).toContain('sampai=2026-09-30')
    expect(calls[0]?.header['X-Portal']).toBe('ASM')
  })

  it('menggambar kedua grid — ringkasan lalu rincian', async () => {
    installFetch()
    const user = userEvent.setup()
    show()

    await search(user)

    expect(await screen.findByRole('heading', { name: 'Ringkasan' })).toBeVisible()
    expect(screen.getByRole('heading', { name: 'Rincian' })).toBeVisible()
  })

  // Urutannya mengikuti `<APPEND>` pada activity lama: Komite lebih dulu, baru AI.
  it('menggambar ringkasan dengan Komite di baris pertama', async () => {
    installFetch()
    const user = userEvent.setup()
    show()

    await search(user)

    const table = await screen.findByRole('table', {
      name: 'Pencacah keputusan AI dan keputusan komite',
    })
    const rows = within(table).getAllByRole('row')
    expect(within(rows[1]!).getByText('Komite')).toBeVisible()
    expect(within(rows[2]!).getByText('AI')).toBeVisible()
  })

  // Kolom "Menunggu" TIDAK ada di layar lama; ia ditambahkan atas keputusan Work Owner
  // 2026-09-26 supaya selisih antara Total dan jumlah baris dapat dibaca.
  it('menggambar kolom Menunggu pada ringkasan', async () => {
    installFetch()
    const user = userEvent.setup()
    show()

    await search(user)

    const table = await screen.findByRole('table', {
      name: 'Pencacah keputusan AI dan keputusan komite',
    })
    expect(within(table).getByRole('columnheader', { name: /Menunggu/ })).toBeVisible()
  })

  // Kesepuluh kolom digambar, termasuk lima yang SELALU kosong. Menghapusnya akan membuat
  // layar baru berbeda dari yang dihafal pengguna sekaligus menyembunyikan bahwa kueri
  // layar lamanya belum selesai.
  it('menggambar kesepuluh kolom rincian, termasuk lima yang selalu kosong', async () => {
    installFetch()
    const user = userEvent.setup()
    show()

    await search(user)

    const table = await screen.findByRole('table', {
      name: 'Penilaian AI beserta keputusan komitenya',
    })
    for (const title of [
      'No Klaim',
      'Object Name',
      'Komite Status',
      'Tanggal Komite',
      'AI Status',
      'Tanggal AI',
      'Note AI Terima',
      'Note AI Tolak',
      'Coverage Final',
      'Kategori Kronologi',
    ]) {
      expect(
        within(table).getByRole('columnheader', { name: new RegExp(title) }),
      ).toBeVisible()
    }
  })

  // Padanan `CASE WHEN B.KOMITEKE = '1' THEN B.NO_KLAIM ELSE '' END`.
  it('membiarkan No Klaim kosong pada baris jenjang komite kedua', async () => {
    installFetch()
    const user = userEvent.setup()
    show()

    await search(user)

    const table = await screen.findByRole('table', {
      name: 'Penilaian AI beserta keputusan komitenya',
    })
    const rows = within(table).getAllByRole('row')

    // rows[0] adalah baris judul; baris ketiga data ada di rows[3].
    expect(within(rows[1]!).getByText('PNCN.26.0001')).toBeVisible()
    expect(within(rows[3]!).queryByText('PNCN.26.0001')).toBeNull()
  })

  // Tanggal digambar memakai pemformat baku aplikasi, bukan teks waktu Pega mentah yang
  // tergambar di layar lama.
  it('memformat tanggal, dan menandai yang kosong', async () => {
    installFetch()
    const user = userEvent.setup()
    show()

    await search(user)

    const table = await screen.findByRole('table', {
      name: 'Penilaian AI beserta keputusan komitenya',
    })
    expect(within(table).getAllByText('3 September 2026').length).toBeGreaterThan(0)

    // Baris keempat belum dinilai AI, sehingga Tanggal AI-nya null.
    const rows = within(table).getAllByRole('row')
    expect(within(rows[4]!).getAllByText('—').length).toBeGreaterThan(0)
  })

  // Pesan per isian datang dari server, bukan dihitung ulang di layar — dua tempat yang
  // dapat berselisih akan membuat selisihnya tidak pernah terlihat.
  it('menandai kedua isian saat server menolak dengan 422', async () => {
    installFetch({ failValidation: true })
    const user = userEvent.setup()
    show()

    await search(user)

    expect(await screen.findByText('Tgl Input Dari belum diisi.')).toBeVisible()
    expect(screen.getByText('Tgl Input Sampai belum diisi.')).toBeVisible()
  })

  // Ekspor memakai penyaring yang SUDAH DIKIRIM. Berkas yang isinya berbeda dari yang
  // terlihat di layar adalah berkas yang tidak dapat dicocokkan dengan apa pun.
  it('mengunduh berkas memakai penyaring yang sedang tergambar', async () => {
    installFetch()
    const user = userEvent.setup()
    show()

    await search(user)
    await screen.findByRole('heading', { name: 'Rincian' })

    await user.click(screen.getByRole('button', { name: 'Export To Excel' }))

    await waitFor(() => {
      expect(calls.some((call) => call.url.includes('/ekspor'))).toBe(true)
    })
    const exportCall = calls.find((call) => call.url.includes('/ekspor'))
    expect(exportCall?.url).toContain('dari=2026-09-01')
    expect(exportCall?.url).toContain('sampai=2026-09-30')
    expect(exportCall?.header['X-Portal']).toBe('ASM')
  })
})
