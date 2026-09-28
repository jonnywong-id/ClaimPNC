import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { DetailKlaimPanel } from './DetailKlaimPanel'

/**
 * Seluruh nomor polis, nama tertanggung, dan nomor klaim di berkas ini KARANGAN (`D-69`).
 *
 * # Apa yang dijaga uji di berkas ini
 *
 * Panel rincian adalah satu-satunya tempat di modul ini yang MENULIS, dan yang menulis
 * adalah pihak LUAR. Dua hal karena itu diuji lebih tajam daripada tampilan biasa:
 *
 *   - Tombol balas digambar menurut jawaban SERVER (`boleh_dibalas`), bukan menurut
 *     kesimpulan layar atas `sudah_dijawab`. Keduanya DAPAT berbeda pada data lama.
 *   - Penolakan "percakapan sudah dijawab" terbaca sebagai pekerjaan yang SUDAH selesai,
 *     bukan sebagai kegagalan yang layak dicoba ulang.
 */

const KUNCI = 'ASM-FW-GCNMFW-WORK PNC-2001'

const KOLOM_PLA = [
  { kunci: 'nomor', judul: 'No PLA', tanggal: false },
  { kunci: 'tipe', judul: 'Tipe PLA', tanggal: false },
  { kunci: 'nilai', judul: 'Nilai PLA', tanggal: false },
  { kunci: 'tanggal_kirim', judul: 'Tanggal Kirim', tanggal: true },
]

const KOLOM_DLA = [
  { kunci: 'nomor', judul: 'No DLA', tanggal: false },
  { kunci: 'tipe', judul: 'Tipe DLA', tanggal: false },
  { kunci: 'nilai', judul: 'Nilai DLA', tanggal: false },
  { kunci: 'no_akseptasi', judul: 'No Akseptasi', tanggal: false },
  { kunci: 'tanggal_kirim', judul: 'Tanggal Kirim', tanggal: true },
]

const KOLOM_DOKUMEN = [
  { kunci: 'jenis_dokumen', judul: 'Jenis Dokumen', tanggal: false },
  { kunci: 'kategori_dokumen', judul: 'Kategori Dokumen', tanggal: false },
  { kunci: 'nama', judul: 'Nama', tanggal: false },
]

const KOLOM_KOMUNIKASI = [
  { kunci: 'tanggal', judul: 'Date', tanggal: true },
  { kunci: 'pengirim', judul: 'Sender', tanggal: false },
  { kunci: 'pesan', judul: 'Message', tanggal: false },
  { kunci: 'balasan', judul: 'Reply', tanggal: false },
]

const RINCIAN = {
  klaim: {
    kunci_klaim: KUNCI,
    no_klaim: 'PNC-2001',
    no_polis: '00.000.2026.02001',
    nama_tertanggung: 'PT Contoh Satu',
    nama_bisnis: 'FIRE',
    tanggal_register: '2026-01-05',
    tanggal_kejadian: '2026-01-02',
    pic_teknik: 'BUDI',
    kode_status: '1147',
    status: 'Register',
  },
  pla: [
    {
      jenis: 'PLA',
      nomor: 'PLA/2026/2001-R1',
      tipe: 'Treaty',
      nilai: '140000000.00',
      no_akseptasi: '',
      tanggal_dokumen: '2026-01-15',
      tanggal_kirim: '2026-01-15',
    },
  ],
  dla: [
    {
      jenis: 'DLA',
      nomor: 'DLA/2026/2001',
      tipe: 'Treaty',
      nilai: '130000000.00',
      no_akseptasi: 'AKS/2026/2001',
      tanggal_dokumen: '2026-01-16',
      tanggal_kirim: '2026-01-16',
    },
  ],
  komunikasi: [
    {
      id: 'KOM-01',
      tanggal: '2026-02-02',
      pengirim: 'Budi Santoso',
      pesan: 'Mohon konfirmasi nilai estimasi.',
      balasan: '',
      nama_pembalas: '',
      tanggal_balasan: '',
      sudah_dijawab: false,
      boleh_dibalas: true,
    },
    {
      id: 'KOM-03',
      tanggal: '2026-02-04',
      pengirim: 'Mitra Contoh',
      pesan: 'Apakah dokumen sudah lengkap?',
      balasan: 'Sudah lengkap.',
      nama_pembalas: 'Siti Aminah',
      tanggal_balasan: '2026-02-05',
      sudah_dijawab: true,
      boleh_dibalas: false,
    },
  ],
  kolom_pla: KOLOM_PLA,
  kolom_dla: KOLOM_DLA,
  kolom_dokumen: KOLOM_DOKUMEN,
  kolom_komunikasi: KOLOM_KOMUNIKASI,
  portal: 'ASM',
}

type Call = { url: string; method: string; body: unknown }
type Reply = { body: unknown; status?: number }

let calls: Call[] = []

function installFetch(map: (call: Call) => Reply) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const raw = init?.body
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
      body: typeof raw === 'string' ? JSON.parse(raw) : undefined,
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

/** jawabanBiasa melayani rincian, dokumen, dan balasan. */
function jawabanBiasa(call: Call): Reply {
  if (call.url.includes('/komunikasi/balas')) {
    return { body: { pesan: 'Balasan Anda tersimpan.', portal: 'ASM' } }
  }
  if (call.url.includes('/dokumen')) {
    return {
      body: {
        baris: [
          {
            id: 'DOK-01',
            jenis_dokumen: 'Dokumen Klaim',
            kategori_dokumen: 'Laporan Kerugian',
            nama: 'laporan-kerugian.pdf',
          },
        ],
        kolom: KOLOM_DOKUMEN,
        portal: 'ASM',
      },
    }
  }
  return { body: RINCIAN }
}

function tampilkan() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <DetailKlaimPanel
        kunciKlaim={KUNCI}
        nomorKlaim="PNC-2001"
        onClose={() => {}}
      />
    </QueryClientProvider>,
  )
}

describe('panel rincian klaim milik reasuradur', () => {
  beforeEach(() => {
    calls = []
    useSession.setState({ token: 'token-uji' })
    useSelectedPortal.setState({ alias: 'ASM' })
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('menggambar kepala klaim dengan judul Pega apa adanya', async () => {
    installFetch(jawabanBiasa)
    tampilkan()

    // Ditunggu ISINYA, bukan judul panelnya: judulnya tergambar sebelum datanya tiba,
    // sehingga menunggunya tidak membuktikan apa pun.
    expect(await screen.findByText('PT Contoh Satu')).toBeInTheDocument()

    // Judulnya berbahasa Inggris karena begitulah di Pega (`D-13`). Pembacanya pihak
    // luar, yang tidak dapat kita latih ulang.
    expect(screen.getByText('Claim No')).toBeInTheDocument()
    expect(screen.getByText('Business Name')).toBeInTheDocument()
    expect(screen.getByText('PIC ASM')).toBeInTheDocument()
  })

  it('kunci klaim yang memuat SPASI dikodekan di alamat', async () => {
    installFetch(jawabanBiasa)
    tampilkan()

    await screen.findByText('Rincian Klaim PNC-2001')

    // Tanpa pengodean, spasi pada `ASM-FW-GCNMFW-WORK PNC-2001` memecah alamatnya dan
    // permintaannya tidak pernah sampai ke rute yang benar.
    expect(
      calls.some((call) => call.url.includes('ASM-FW-GCNMFW-WORK%20PNC-2001')),
    ).toBe(true)
  })

  it('grid DLA punya kolom "No Akseptasi"; grid PLA tidak', async () => {
    installFetch(jawabanBiasa)
    tampilkan()

    expect(await screen.findByText('PLA/2026/2001-R1')).toBeInTheDocument()

    const gridPLA = screen.getByRole('table', { name: 'Daftar PLA' })
    const gridDLA = screen.getByRole('table', { name: 'Daftar DLA' })

    // Kolom "No Akseptasi" HANYA ada pada DLA — `T_PLALIST` tidak punya `NOAKSEP`.
    expect(
      within(gridDLA).getByRole('columnheader', { name: /No Akseptasi/ }),
    ).toBeInTheDocument()
    expect(
      within(gridPLA).queryByRole('columnheader', { name: /No Akseptasi/ }),
    ).toBeNull()

    expect(within(gridDLA).getByText('AKS/2026/2001')).toBeInTheDocument()
  })

  it('grid yang kosong menjelaskan SEBABNYA, bukan sekadar kosong', async () => {
    installFetch((call) => {
      if (call.url.includes('/dokumen') || call.url.includes('/balas')) {
        return jawabanBiasa(call)
      }
      return { body: { ...RINCIAN, dla: [] } }
    })
    tampilkan()

    expect(await screen.findByText('PLA/2026/2001-R1')).toBeInTheDocument()
    expect(
      screen.getByText(/Belum ada DLA yang dikirimkan kepada Anda/),
    ).toBeInTheDocument()
  })

  it('dokumen baru diambil setelah tombol Dokumen ditekan', async () => {
    installFetch(jawabanBiasa)
    tampilkan()

    await screen.findByText('PLA/2026/2001-R1')

    // Kuncinya adalah NOMOR PEMBERITAHUAN, bukan klaimnya — sehingga dokumennya tidak
    // ikut permintaan rincian.
    expect(calls.some((call) => call.url.includes('/dokumen'))).toBe(false)

    // Tombol "Dokumen" ada di KEDUA grid; yang ditekan adalah milik baris PLA.
    const gridPLA = screen.getByRole('table', { name: 'Daftar PLA' })
    await userEvent.click(
      within(gridPLA).getByRole('button', { name: 'Dokumen' }),
    )

    expect(await screen.findByText('laporan-kerugian.pdf')).toBeInTheDocument()
    await waitFor(() => {
      expect(
        calls.some(
          (call) =>
            call.url.includes('/dokumen') &&
            call.url.includes('nomor=PLA%2F2026%2F2001-R1') &&
            call.url.includes('jenis=PLA'),
        ),
      ).toBe(true)
    })
  })

  it('tombol balas digambar menurut jawaban SERVER, bukan kesimpulan layar', async () => {
    installFetch(jawabanBiasa)
    tampilkan()

    const komunikasi = await screen.findByRole('table', {
      name: 'Riwayat komunikasi klaim',
    })

    // KOM-01 boleh dibalas; KOM-03 sudah dijawab.
    expect(
      within(komunikasi).getByRole('button', { name: 'Balas Pesan' }),
    ).toBeInTheDocument()
    expect(within(komunikasi).getByText('Sudah dijawab')).toBeInTheDocument()
  })

  it('balasan dikirim dengan nomor percakapan di BADAN permintaan', async () => {
    installFetch(jawabanBiasa)
    tampilkan()

    await screen.findByText('Mohon konfirmasi nilai estimasi.')

    await userEvent.click(screen.getByRole('button', { name: 'Balas Pesan' }))
    await userEvent.type(
      screen.getByLabelText(/Balasan Anda/),
      'Nilai disetujui.',
    )
    await userEvent.click(screen.getByRole('button', { name: 'Kirim Balasan' }))

    await waitFor(() => {
      const balas = calls.find((call) => call.url.includes('/komunikasi/balas'))
      expect(balas).toBeDefined()
      expect(balas?.method).toBe('POST')

      // Nomor percakapan berada di BADAN, bukan di alamat: badan permintaan tidak
      // tercatat di log peladen web maupun riwayat peramban, sementara segmen alamat
      // tercatat di keduanya.
      expect(balas?.body).toEqual({
        percakapan: 'KOM-01',
        balasan: 'Nilai disetujui.',
      })
    })
  })

  it('penolakan "sudah dijawab" terbaca sebagai pekerjaan yang SELESAI', async () => {
    installFetch((call) => {
      if (call.url.includes('/komunikasi/balas')) {
        return {
          body: {
            kode: 'percakapan_sudah_dijawab',
            pesan:
              'Percakapan ini sudah dijawab. Setiap percakapan hanya menyimpan satu ' +
              'balasan, sehingga balasan baru tidak dapat ditambahkan — muat ulang ' +
              'rinciannya untuk membaca balasan yang sudah ada.',
          },
          status: 409,
        }
      }
      return jawabanBiasa(call)
    })
    tampilkan()

    await screen.findByText('Mohon konfirmasi nilai estimasi.')

    await userEvent.click(screen.getByRole('button', { name: 'Balas Pesan' }))
    await userEvent.type(screen.getByLabelText(/Balasan Anda/), 'Menimpa?')
    await userEvent.click(screen.getByRole('button', { name: 'Kirim Balasan' }))

    expect(
      await screen.findByText(/hanya menyimpan satu balasan/),
    ).toBeInTheDocument()
  })

  it('tombol Download ALL menjawab alasan yang menyebut tombolnya sendiri', async () => {
    installFetch((call) => {
      if (call.url.includes('/tindakan')) {
        return {
          body: {
            kode: 'belum_tersedia',
            pesan:
              'Tombol "Download ALL PLA" belum tersedia di sistem baru. Untuk ' +
              'sementara, unduhlah dokumennya satu per satu lewat tombol "Dokumen" ' +
              'pada tiap baris PLA.',
          },
          status: 501,
        }
      }
      return jawabanBiasa(call)
    })
    tampilkan()

    await screen.findByText('PLA/2026/2001-R1')

    await userEvent.click(
      screen.getByRole('button', { name: 'Download ALL PLA' }),
    )

    // Alasannya menyebut tombol yang BENAR-BENAR ditekan, bukan kalimat umum untuk
    // keduanya — dan menyebut apa yang dapat dilakukan hari ini.
    expect(
      await screen.findByText(/unduhlah dokumennya satu per satu/),
    ).toBeInTheDocument()

    await waitFor(() => {
      expect(
        calls.some(
          (call) =>
            call.url.includes('/tindakan') &&
            call.url.includes('tindakan=unduh-semua-pla'),
        ),
      ).toBe(true)
    })
  })
})
