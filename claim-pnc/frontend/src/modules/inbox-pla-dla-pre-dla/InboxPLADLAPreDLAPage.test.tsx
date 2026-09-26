import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { InboxPLADLAPreDLAPage } from './InboxPLADLAPreDLAPage'

/**
 * Seluruh nomor polis, nama tertanggung, dan nomor klaim di berkas ini KARANGAN
 * (`D-69`): data nasabah tidak pernah ditulis ke berkas yang di-commit, dan larangan itu
 * berlaku pada data uji persis seperti pada dokumen.
 */

const KOLOM_ANTREAN = [
  { kunci: 'no_klaim', judul: 'No Klaim', tanggal: false },
  { kunci: 'no_polis', judul: 'No Polis', tanggal: false },
  { kunci: 'nama_tertanggung', judul: 'Nama Tertanggung', tanggal: false },
  { kunci: 'tanggal_register', judul: 'Tanggal Register', tanggal: true },
  { kunci: 'tanggal_kejadian', judul: 'Tanggal Kejadian', tanggal: true },
  { kunci: 'pic_teknik', judul: 'PIC Teknik', tanggal: false },
  { kunci: 'tanggal_advice', judul: 'Tanggal PLA', tanggal: true },
]

const KOLOM_RINCIAN = [
  { kunci: 'no_advice', judul: 'No PLA', tanggal: false },
  { kunci: 'reasuradur', judul: 'Reasuradur', tanggal: false },
  { kunci: 'terkirim', judul: 'Terkirim', tanggal: false },
  { kunci: 'tanggal_kirim', judul: 'Tanggal Kirim', tanggal: true },
]

const METADATA = {
  daftar: [
    {
      kode: 'pla',
      nama: 'PLA',
      keterangan: 'PLA yang sudah terbit tetapi belum dikirim.',
      kolom: KOLOM_ANTREAN,
      kolom_rincian: KOLOM_RINCIAN,
      punya_rincian: true,
      label_pencarian: 'No Klaim',
      label_tanggal: 'Tanggal PLA',
    },
    {
      kode: 'dla',
      nama: 'DLA',
      keterangan: 'DLA yang sudah terbit tetapi belum dikirim.',
      kolom: KOLOM_ANTREAN,
      kolom_rincian: KOLOM_RINCIAN,
      punya_rincian: true,
      label_pencarian: 'No Klaim',
      label_tanggal: 'Tanggal DLA',
    },
    {
      kode: 'pre-dla',
      nama: 'Pre DLA',
      keterangan: 'Pre-DLA yang belum punya Nomor Akseptasi.',
      kolom: KOLOM_ANTREAN,
      kolom_rincian: [],
      punya_rincian: false,
      label_pencarian: 'No Klaim',
      label_tanggal: 'Tanggal Pre DLA',
    },
  ],
  daftar_bawaan: 'pla',
  selisih_terencana: ['Tombol "Send" belum tersedia.'],
  portal: 'ASM',
}

/** Baris LENGKAP. */
const BARIS_LENGKAP = {
  kunci_klaim: 'ASM-FW-GCNMFW-WORK PNC-1001',
  no_klaim: 'PNC-1001',
  no_polis: '00.000.2026.00001',
  nama_tertanggung: 'PT Contoh Satu',
  tanggal_register: '2026-01-05',
  tanggal_kejadian: '2026-01-02',
  pic_teknik: 'BUDI',
  tanggal_advice: '2026-01-10',
}

/**
 * Baris yang tanggal advice-nya KOSONG.
 *
 * Keadaan ini nyata: klaim yang seluruh dokumennya ber-`ISKIRIM = '0'` tetap masuk
 * antrean, tetapi kolom tanggalnya kosong. Sel kosong di sana bukan kerusakan, dan uji
 * ini menjaga layar menggambarnya sebagai tanda hubung alih-alih tanggal apa pun.
 */
const BARIS_TANPA_TANGGAL = {
  kunci_klaim: 'ASM-FW-GCNMFW-WORK PNC-1008',
  no_klaim: 'PNC-1008',
  no_polis: '00.000.2026.00008',
  nama_tertanggung: 'PT Contoh Delapan',
  tanggal_register: '2026-02-05',
  tanggal_kejadian: '2026-01-23',
  pic_teknik: 'RINA',
  tanggal_advice: '',
}

type Call = { url: string; method: string }

let calls: Call[] = []

type Reply = { body: unknown; status?: number }

function installFetch(map: (call: Call) => Reply) {
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

/** jawabanBiasa melayani metadata dan daftar dengan baris yang diberikan. */
function jawabanBiasa(baris: unknown[]): (call: Call) => Reply {
  return (call) => {
    if (call.url.includes('/daftar')) return { body: METADATA }

    if (call.url.includes('/klaim/')) {
      return {
        body: {
          daftar: METADATA.daftar[0],
          kunci_klaim: 'ASM-FW-GCNMFW-WORK PNC-1001',
          baris: [
            {
              no_advice: 'PLA/2026/0001',
              reasuradur: 'Reasuransi Contoh A',
              tipe: 'OR',
              revisi: '0',
              tanggal_dokumen: '2026-01-10',
              terkirim: '',
              tanggal_kirim: '',
              tanggal_terima: '',
              catatan: '',
              email: '',
              no_akseptasi: '',
            },
          ],
          portal: 'ASM',
        },
      }
    }

    return {
      body: {
        daftar: METADATA.daftar[0],
        baris,
        paginasi: {
          halaman: 1,
          ukuran: 10,
          total: baris.length,
          total_halaman: 1,
        },
        penyaring: { cari: '', dari: '', sampai: '' },
        portal: 'ASM',
      },
    }
  }
}

function tampilkan() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <InboxPLADLAPreDLAPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function mulaiSesi() {
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
  mulaiSesi()
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('layar Inbox PLA, DLA, Pre DLA', () => {
  it('menggambar ketiga tab beserta kolom yang dikirim server', async () => {
    installFetch(jawabanBiasa([BARIS_LENGKAP]))
    tampilkan()

    expect(await screen.findByRole('tab', { name: 'PLA' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'DLA' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'Pre DLA' })).toBeInTheDocument()

    for (const kolom of KOLOM_ANTREAN) {
      expect(
        await screen.findByRole('columnheader', { name: new RegExp(kolom.judul) }),
      ).toBeInTheDocument()
    }
  })

  it('menggambar tanggal advice yang kosong sebagai tanda hubung, bukan tanggal', async () => {
    installFetch(jawabanBiasa([BARIS_TANPA_TANGGAL]))
    tampilkan()

    const baris = await screen.findByText('PNC-1008')
    const row = baris.closest('tr')
    expect(row).not.toBeNull()

    // Tanggal register dan tanggal kejadian tetap tergambar; hanya tanggal advice yang
    // kosong. Itu yang membuktikan sel kosongnya bukan akibat seluruh baris gagal
    // diformat.
    expect(within(row as HTMLElement).getByText(/05 Feb 2026/)).toBeInTheDocument()
    expect(within(row as HTMLElement).getByText('—')).toBeInTheDocument()
  })

  it('mengirim rentang tanggal dan kata kunci hanya setelah CARI DATA ditekan', async () => {
    installFetch(jawabanBiasa([BARIS_LENGKAP]))
    tampilkan()

    await screen.findByText('PNC-1001')
    const sebelum = calls.length

    await userEvent.type(screen.getByLabelText('No Klaim'), 'PNC-1001')

    // Mengetik TIDAK menembak basis data. Penyaring di layar ini menyentuh tabel dokumen
    // berisi puluhan juta baris, dan layar lamanya pun memakai tombol.
    expect(calls.length).toBe(sebelum)

    await userEvent.click(screen.getByRole('button', { name: 'CARI DATA' }))

    await waitFor(() => {
      expect(
        calls.some((call) => call.url.includes('cari=PNC-1001')),
      ).toBe(true)
    })
  })

  it('menyebut tanggal APA yang sedang dibatasi, berbeda per tab', async () => {
    installFetch(jawabanBiasa([BARIS_LENGKAP]))
    tampilkan()

    expect(
      await screen.findByLabelText('Tanggal PLA — Dari'),
    ).toBeInTheDocument()

    await userEvent.click(screen.getByRole('tab', { name: 'DLA' }))

    expect(
      await screen.findByLabelText('Tanggal DLA — Dari'),
    ).toBeInTheDocument()
  })

  it('tidak menggambar tombol Rincian pada tab Pre DLA', async () => {
    installFetch(jawabanBiasa([BARIS_LENGKAP]))
    tampilkan()

    expect(
      await screen.findAllByRole('button', { name: 'Rincian' }),
    ).not.toHaveLength(0)

    await userEvent.click(screen.getByRole('tab', { name: 'Pre DLA' }))

    // Tab Pre DLA tidak punya grid rincian — di Pega pun tidak. Menggambar tombolnya
    // akan menjanjikan sesuatu yang dijawab penolakan.
    await waitFor(() => {
      expect(screen.queryByRole('button', { name: 'Rincian' })).toBeNull()
    })
  })

  it('membuka panel rincian dengan kunci klaim yang TERKODEKAN', async () => {
    installFetch(jawabanBiasa([BARIS_LENGKAP]))
    tampilkan()

    const [tombol] = await screen.findAllByRole('button', { name: 'Rincian' })
    expect(tombol).toBeDefined()
    await userEvent.click(tombol as HTMLElement)

    await waitFor(() => {
      // Kunci klaim memuat SPASI. Tanpa pengodean, alamatnya terpotong di spasi itu dan
      // yang sampai ke server hanya awalannya.
      expect(
        calls.some((call) =>
          call.url.includes('/klaim/ASM-FW-GCNMFW-WORK%20PNC-1001'),
        ),
      ).toBe(true)
    })

    expect(await screen.findByText('Detail PLA List')).toBeInTheDocument()
  })

  it('menutup panel rincian saat pengguna berpindah tab', async () => {
    installFetch(jawabanBiasa([BARIS_LENGKAP]))
    tampilkan()

    const [tombol] = await screen.findAllByRole('button', { name: 'Rincian' })
    expect(tombol).toBeDefined()
    await userEvent.click(tombol as HTMLElement)
    expect(await screen.findByText('Detail PLA List')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('tab', { name: 'DLA' }))

    // Rincian milik daftar sebelumnya: klaim yang sama punya grid yang berbeda di tab
    // PLA dan tab DLA, dan tabel yang dibacanya pun berbeda.
    await waitFor(() => {
      expect(screen.queryByText('Detail PLA List')).toBeNull()
    })
  })

  it('menjelaskan MENGAPA antreannya kosong, berbeda saat menyaring', async () => {
    installFetch(jawabanBiasa([]))
    tampilkan()

    expect(
      await screen.findByText(/Tidak ada pemberitahuan yang menunggu dikirim/),
    ).toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('No Klaim'), 'PNC-9999')
    await userEvent.click(screen.getByRole('button', { name: 'CARI DATA' }))

    expect(
      await screen.findByText(/Longgarkan rentang tanggalnya/),
    ).toBeInTheDocument()
  })

  it('menggambar selisih terencana yang dikirim server', async () => {
    installFetch(jawabanBiasa([BARIS_LENGKAP]))
    tampilkan()

    expect(
      await screen.findByText(/Perbedaan yang disengaja terhadap layar Pega \(1\)/),
    ).toBeInTheDocument()
    expect(
      screen.getByText('Tombol "Send" belum tersedia.'),
    ).toBeInTheDocument()
  })

  it('menjelaskan portal yang belum dipilih, bukan menyebutnya kerusakan', async () => {
    installFetch(() => ({
      body: { kode: 'portal_tidak_disebut', pesan: 'Portal belum disebut.' },
      status: 400,
    }))
    tampilkan()

    expect(
      await screen.findByText('Portal entitas belum dipilih'),
    ).toBeInTheDocument()
  })
})
