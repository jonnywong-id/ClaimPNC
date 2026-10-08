import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { InboxPLADLAReasPage } from './InboxPLADLAReasPage'

/**
 * Seluruh nomor polis, nama tertanggung, dan nomor klaim di berkas ini KARANGAN
 * (`D-69`).
 */

const KOLOM = [
  { kunci: 'no_klaim', judul: 'No Klaim', tanggal: false },
  { kunci: 'no_pla', judul: 'No PLA', tanggal: false },
  { kunci: 'no_polis', judul: 'No Polis', tanggal: false },
  { kunci: 'nama_tertanggung', judul: 'Nama Tertanggung', tanggal: false },
  { kunci: 'nama_bisnis', judul: 'Bisnis', tanggal: false },
  { kunci: 'tanggal_register', judul: 'Tanggal Register', tanggal: true },
  { kunci: 'tanggal_kejadian', judul: 'Tanggal Kejadian', tanggal: true },
  { kunci: 'pic_teknik', judul: 'PIC Teknik', tanggal: false },
  { kunci: 'status', judul: 'Status', tanggal: false },
]

const KOLOM_XOL = [
  { kunci: 'tahun', judul: 'Tahun', tanggal: false },
  { kunci: 'penyebab_kerugian', judul: 'Penyebab Kerugian', tanggal: false },
  { kunci: 'jenis', judul: 'Jenis', tanggal: false },
  { kunci: 'tanggal_terakhir', judul: 'Tanggal Terakhir', tanggal: true },
]

const METADATA = {
  daftar: [
    {
      kode: 'pla',
      nama: 'PLA',
      keterangan: 'PLA sudah dikirimkan kepada Anda, DLA belum.',
      jenis: 'daftar-klaim',
      sumber: 'pemberitahuan',
      kolom: KOLOM,
      punya_rincian: true,
    },
    {
      kode: 'dla',
      nama: 'PLA & DLA',
      keterangan: 'DLA sudah dikirimkan kepada Anda.',
      jenis: 'daftar-klaim',
      sumber: 'pemberitahuan',
      kolom: KOLOM,
      punya_rincian: true,
    },
    {
      kode: 'close',
      nama: 'CLOSE CLAIM',
      keterangan: 'Klaim yang sudah selesai.',
      jenis: 'daftar-klaim',
      sumber: 'pemberitahuan',
      kolom: KOLOM,
      punya_rincian: true,
    },
    {
      kode: 'not-answered',
      nama: 'NOT ANSWERED',
      keterangan: 'Ada pesan untuk Anda yang belum Anda jawab.',
      jenis: 'daftar-klaim',
      sumber: 'komunikasi',
      kolom: KOLOM,
      punya_rincian: true,
    },
    {
      // Tampilan XOL BUKAN daftar klaim: ia tidak punya baris, tidak punya tabel
      // ringkas, dan tidak disaring kotak pencarian. Penandanya datang dari SERVER.
      kode: 'xol',
      nama: 'DATA PLA DLA XOL KLAIM',
      keterangan: 'Ringkasan pemberitahuan XOL yang sudah dikirimkan kepada Anda.',
      jenis: 'xol',
      sumber: '',
      kolom: KOLOM_XOL,
      punya_rincian: false,
    },
  ],
  daftar_bawaan: 'pla',
  kolom_xol: KOLOM_XOL,
  portal: 'ASM',
}

const BARIS = {
  kunci_klaim: 'ASM-FW-GCNMFW-WORK PNC-2001',
  no_klaim: 'PNC-2001',
  no_polis: '00.000.2026.02001',
  nama_tertanggung: 'PT Contoh Satu',
  nama_bisnis: 'FIRE',
  tanggal_register: '2026-01-05',
  tanggal_kejadian: '2026-01-02',
  pic_teknik: 'BUDI',
  kode_status: '1147',
  status: 'Register',
  no_pla: 'PLA/2026/2001',
  catatan_tutup: '',
}

/**
 * Baris yang kode statusnya TIDAK ada di master.
 *
 * Keadaan ini nyata: domain kode status 33 nilai, dan data lama memuat kode di luar itu.
 * Layar jatuh ke kodenya alih-alih menggambar sel kosong.
 */
const BARIS_STATUS_ASING = {
  ...BARIS,
  kunci_klaim: 'ASM-FW-GCNMFW-WORK PNC-2099',
  no_klaim: 'PNC-2099',
  kode_status: '9999',
  status: '',
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

/** jawabanBiasa melayani metadata, daftar, ringkasan, dan grid XOL. */
function jawabanBiasa(baris: unknown[]): (call: Call) => Reply {
  return (call) => {
    if (call.url.includes('/daftar')) return { body: METADATA }

    if (call.url.includes('/ringkas')) {
      return {
        body: {
          baris: [{ kode_status: '1147', status: 'Register', jumlah: 3 }],
          portal: 'ASM',
        },
      }
    }

    if (call.url.includes('/xol')) {
      return {
        body: {
          baris: [
            {
              tahun: '2026',
              penyebab_kerugian: 'Kebakaran',
              jenis: 'PLA',
              tanggal_terakhir: '2026-02-10',
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
        cari: '',
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
        <InboxPLADLAReasPage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

function mulaiSesi() {
  useSession.getState().login({
    token: 'token-contoh',
    user: {
      identitas: '90000001',
      nama: 'Mitra Reasuransi Contoh',
      jenis: 'KARYAWAN',
      login: 'reascontoh',
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

describe('layar Inbox PLA DLA milik reasuradur', () => {
  it('menggambar SELURUH tampilan beserta kolom yang dikirim server', async () => {
    installFetch(jawabanBiasa([BARIS]))
    tampilkan()

    // Judulnya diambil APA ADANYA dari Pega (`D-13`) — termasuk "PLA & DLA" dan
    // "CLOSE CLAIM", yang sebelum 2026-09-28 dinamai "DLA" dan "Close".
    expect(await screen.findByRole('tab', { name: 'PLA' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'PLA & DLA' })).toBeInTheDocument()
    expect(screen.getByRole('tab', { name: 'CLOSE CLAIM' })).toBeInTheDocument()

    // Ketiga daftar komunikasi dan tampilan XOL baru dibangun 2026-09-28. Sebelumnya
    // keempatnya ada di Pega tanpa satu pun tab yang menuju ke sana.
    expect(
      screen.getByRole('tab', { name: 'NOT ANSWERED' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('tab', { name: /DATA PLA DLA XOL KLAIM/ }),
    ).toBeInTheDocument()

    for (const kolom of KOLOM) {
      expect(
        await screen.findByRole('columnheader', { name: new RegExp(kolom.judul) }),
      ).toBeInTheDocument()
    }
  })

  it('menggambar tabel ringkas Status / Jumlah', async () => {
    installFetch(jawabanBiasa([BARIS]))
    tampilkan()

    const ringkas = await screen.findByRole('region', {
      name: 'Ringkasan jumlah klaim per status',
    })

    expect(within(ringkas).getByText('Status / Jumlah')).toBeInTheDocument()
    expect(within(ringkas).getByText('Register')).toBeInTheDocument()

    // Dicari DI DALAM panel ringkasnya. Angka "3" juga muncul di bilah halaman dan di
    // tempat lain; pencarian di seluruh halaman akan menemukan yang salah.
    expect(within(ringkas).getByText('3')).toBeInTheDocument()
  })

  it('menggambar kode status ketika artinya tidak ada di master', async () => {
    installFetch(jawabanBiasa([BARIS_STATUS_ASING]))
    tampilkan()

    // Sel kosong akan membuat barisnya tampak rusak; kodenya masih dapat ditelusuri.
    expect(await screen.findByText('9999')).toBeInTheDocument()
  })

  it('menggambar grid XOL hanya setelah tampilannya DIPILIH', async () => {
    installFetch(jawabanBiasa([BARIS]))
    tampilkan()

    await screen.findByText('PNC-2001')

    // Sebelum tabnya dibuka, gridnya TIDAK tergambar dan permintaannya tidak dikirim.
    //
    // Di Pega pun begitu: wadahnya bersyarat `TempView.CityID==7`. Sebelum 2026-09-28
    // panel ini digambar permanen di kaki halaman, sehingga gabungan dua tabel XOL
    // dijalankan pada setiap pembukaan layar — termasuk bagi mitra yang tidak pernah
    // membukanya.
    expect(calls.some((call) => call.url.includes('/xol'))).toBe(false)

    await userEvent.click(
      screen.getByRole('tab', { name: /DATA PLA DLA XOL KLAIM/ }),
    )

    expect(await screen.findByText('Kebakaran')).toBeInTheDocument()

    await waitFor(() => {
      expect(calls.some((call) => call.url.includes('/xol'))).toBe(true)
    })

    // Keterangan ini yang memberi tahu mitra mengapa angkanya berbeda dari Pega.
    expect(
      screen.getByText(/Di Pega ia ditulis tetap ke satu mitra/),
    ).toBeInTheDocument()
  })

  it('mengambil ringkasan lewat permintaan TERSENDIRI dari daftarnya', async () => {
    installFetch(jawabanBiasa([BARIS]))
    tampilkan()

    await screen.findByText('PNC-2001')

    await waitFor(() => {
      expect(calls.some((call) => call.url.includes('/ringkas'))).toBe(true)
    })
  })

  it('menolak petugas internal dengan pesan yang menunjuk menu yang benar', async () => {
    installFetch((call) => {
      if (call.url.includes('/daftar')) return { body: METADATA }
      return {
        body: {
          kode: 'bukan_reasuradur_terdaftar',
          pesan:
            'Layar ini menampilkan klaim yang pemberitahuannya dikirimkan kepada ' +
            'satu mitra reasuransi. Bila Anda petugas internal, pakailah menu ' +
            '"Inbox PLA, DLA, Pre DLA".',
        },
        status: 403,
      }
    })
    tampilkan()

    expect(
      await screen.findByText('Layar ini untuk mitra reasuransi'),
    ).toBeInTheDocument()
    expect(
      screen.getByText(/pakailah menu "Inbox PLA, DLA, Pre DLA"/),
    ).toBeInTheDocument()

    // Penolakan menggantikan SELURUH isi layar: menggambar tabel dan tombol ekspor di
    // bawahnya hanya menawarkan hal-hal yang seluruhnya akan ditolak.
    expect(screen.queryByRole('tab', { name: 'PLA' })).toBeNull()
    expect(
      screen.queryByRole('button', { name: 'Export To Excel' }),
    ).toBeNull()
  })

  it('tidak menembak grid XOL ketika pemanggilnya sudah ditolak', async () => {
    installFetch((call) => {
      if (call.url.includes('/daftar')) return { body: METADATA }
      return {
        body: { kode: 'bukan_reasuradur_terdaftar', pesan: 'Bukan mitra.' },
        status: 403,
      }
    })
    tampilkan()

    await screen.findByText('Layar ini untuk mitra reasuransi')

    // Satu sebab, satu penolakan. Tanpa penjagaan ini pengguna menerima dua penolakan
    // untuk hal yang sama — dan yang kedua digambar sebagai kerusakan.
    expect(calls.some((call) => call.url.includes('/xol'))).toBe(false)
  })

  it('mengirim kata kunci hanya setelah Search Data ditekan', async () => {
    installFetch(jawabanBiasa([BARIS]))
    tampilkan()

    await screen.findByText('PNC-2001')
    const sebelum = calls.length

    await userEvent.type(screen.getByLabelText(/Claim No/), 'PNC-2001')
    expect(calls.length).toBe(sebelum)

    await userEvent.click(screen.getByRole('button', { name: 'Search Data' }))

    await waitFor(() => {
      expect(calls.some((call) => call.url.includes('cari=PNC-2001'))).toBe(true)
    })
  })

  it('menjelaskan MENGAPA daftarnya kosong', async () => {
    installFetch(jawabanBiasa([]))
    tampilkan()

    expect(
      await screen.findByText(/Belum ada klaim pada tahap ini/),
    ).toBeInTheDocument()
  })

  // Keputusan Work Owner 2026-10-06: panel selisih terencana DIHAPUS dari seluruh layar.
  //
  // Daftarnya tetap hidup di kode Go untuk uji kesetaraan gerbang 1 (`D-54`); yang berubah
  // adalah ia berhenti menjadi isi layar.
  it('tidak lagi menggambar panel selisih terencana', async () => {
    installFetch(jawabanBiasa([BARIS]))
    tampilkan()

    await screen.findByRole('table')
    expect(
      screen.queryByText(/Perbedaan yang disengaja terhadap layar Pega/),
    ).not.toBeInTheDocument()
  })
})
