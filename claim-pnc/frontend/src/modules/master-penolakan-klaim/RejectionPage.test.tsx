import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { RejectionPage } from './RejectionPage'

const PARENTS = {
  portal: 'ASM',
  status_1: [
    { id: '1', nama: 'POLIS TIDAK BERLAKU' },
    { id: '2', nama: 'KERUGIAN DIKECUALIKAN POLIS' },
  ],
}

const LIST = {
  portal: 'ASM',
  penolakan_klaim: [
    {
      id: '1',
      nama: 'PREMI BELUM DIBAYAR SAMPAI TANGGAL KEJADIAN',
      id_status_1: '1',
      nama_status_1: 'POLIS TIDAK BERLAKU',
      status: '1',
      status_label: 'APPROVED',
      diajukan_oleh: 'adminpnc',
      diajukan_pada: '2026-09-10T03:00:00Z',
      disetujui_oleh: 'manageradmin',
      disetujui_pada: '2026-09-12T03:00:00Z',
      catatan_persetujuan: 'Sesuai ketentuan polis.',
    },
    {
      id: '2',
      nama: 'PERIODE PERTANGGUNGAN SUDAH BERAKHIR',
      id_status_1: '1',
      nama_status_1: 'POLIS TIDAK BERLAKU',
      status: '0',
      status_label: 'MENUNGGU',
      diajukan_oleh: 'adminpnc',
      diajukan_pada: '2026-09-16T03:00:00Z',
      disetujui_oleh: '',
      disetujui_pada: '',
      catatan_persetujuan: '',
    },
  ],
}

const KOMITE = {
  portal: 'ASM',
  penolakan_komite: [
    { id: '111', catatan: 'NILAI KLAIM DI BAWAH RISIKO SENDIRI' },
    { id: '112', catatan: 'DOKUMEN PENDUKUNG TIDAK MEYAKINKAN' },
  ],
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
 * defaultReply melayani ketiga jalur baca; mutasi dijawab pemanggil lewat `mutation`.
 *
 * Jalur `status-1` diperiksa LEBIH DULU karena ia berawalan sama dengan jalur daftar.
 */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.url.startsWith('/api/master/penolakan-klaim/status-1')) return { body: PARENTS }
    if (call.method === 'GET' && call.url.startsWith('/api/master/penolakan-komite')) {
      return { body: KOMITE }
    }
    if (call.method === 'GET') return { body: LIST }
    if (mutation) return mutation(call)
    return { body: { penolakan_klaim: LIST.penolakan_klaim[0], portal: 'ASM' }, status: 201 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <RejectionPage />
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

describe('tab Penolakan Klaim', () => {
  it('menampilkan judul dan keempat kolom seperti layar lama', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Master Penolakan Klaim' })).toBeInTheDocument()

    const table = await screen.findByRole('table')
    for (const title of [
      'Status Penolakan 1',
      'Status Penolakan 2',
      'Status Aproval',
      'Note Approval',
    ]) {
      expect(within(table).getByRole('columnheader', { name: title })).toBeInTheDocument()
    }
  })

  // Urutan kolom diambil dari label pada `Section/BrowseNoteRejectClaim-Section.xml`.
  // Induk mendahului nama barisnya sendiri — berlawanan dengan dugaan yang wajar, dan
  // sudah pernah tertukar pada modul sekerabat.
  it('menempatkan induk SEBELUM nama barisnya, mengikuti urutan Pega', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    const header = within(table)
      .getAllByRole('columnheader')
      .map((cell) => cell.textContent?.trim())

    expect(header.indexOf('Status Penolakan 1')).toBeLessThan(header.indexOf('Status Penolakan 2'))
  })

  it('menyebut entitas yang sedang dilihat', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.getByText('ASM')).toBeInTheDocument()
  })

  // Setiap permintaan menyebut portalnya. Tanpa header itu server menolak, dan tidak
  // pernah jatuh ke portal utama sebagai cadangan (R-20).
  it('mengirim header portal pada setiap permintaan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(calls.length).toBeGreaterThan(0)
    for (const call of calls) {
      expect(call.header['X-Portal']).toBe('ASM')
    }
  })

  it('menampilkan status persetujuan apa adanya, termasuk yang sudah diputuskan', async () => {
    // Daftar TIDAK disaring "hanya yang menunggu". Penyaring itu milik layar checker;
    // alasannya ada pada doc comment RejectionPage.
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getByText('APPROVED')).toBeInTheDocument()
    expect(within(table).getByText('MENUNGGU')).toBeInTheDocument()
  })
})

describe('menambah penolakan klaim', () => {
  it('mengirim id induk yang dipilih, dan nama induk dibiarkan kosong', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: 'Tambah Penolakan Klaim' })
    await user.selectOptions(within(form).getByLabelText('Status Penolakan 1'), '2')
    await user.type(within(form).getByLabelText('Status Penolakan 2'), 'KERUGIAN AKIBAT KEAUSAN')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const sent = calls.find((call) => call.method === 'POST')
      expect(sent?.body).toEqual({
        nama: 'KERUGIAN AKIBAT KEAUSAN',
        id_status_1: '2',
        nama_status_1: '',
      })
    })
  })

  // INILAH perbaikan yang diputuskan Work Owner 2026-09-19, dilihat dari sisi layar.
  //
  // Memilih induk yang sudah ada TIDAK mengirim nama induk sama sekali, sehingga server
  // tidak punya bahan untuk menerbitkan baris tingkat 1 baru. Di layar lama kedua isian
  // berupa kotak teks bebas, dan isinya selalu diteruskan ke MASTERPENOLAKANKLAIM1 yang
  // menyisipkan baris baru pada setiap simpan.
  it('menyembunyikan isian induk baru sampai pengguna memintanya', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: 'Tambah Penolakan Klaim' })
    expect(within(form).queryByLabelText('Nama Status Penolakan 1 baru')).not.toBeInTheDocument()

    await user.selectOptions(
      within(form).getByLabelText('Status Penolakan 1'),
      '+ Status Penolakan 1 baru…',
    )
    expect(within(form).getByLabelText('Nama Status Penolakan 1 baru')).toBeInTheDocument()
  })

  it('mengirim nama induk baru, dan id induk dibiarkan kosong', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: 'Tambah Penolakan Klaim' })
    await user.selectOptions(
      within(form).getByLabelText('Status Penolakan 1'),
      '+ Status Penolakan 1 baru…',
    )
    await user.type(
      within(form).getByLabelText('Nama Status Penolakan 1 baru'),
      'WILAYAH TIDAK DIJAMIN',
    )
    await user.type(within(form).getByLabelText('Status Penolakan 2'), 'KLAIM DI LUAR WILAYAH')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const sent = calls.find((call) => call.method === 'POST')
      expect(sent?.body).toEqual({
        nama: 'KLAIM DI LUAR WILAYAH',
        id_status_1: '',
        nama_status_1: 'WILAYAH TIDAK DIJAMIN',
      })
    })
  })

  it('menolak menyimpan sebelum induk dipilih', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: 'Tambah Penolakan Klaim' })
    await user.type(within(form).getByLabelText('Status Penolakan 2'), 'X')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText('Status Penolakan 1 wajib dipilih.')).toBeInTheDocument()
    expect(calls.some((call) => call.method === 'POST')).toBe(false)
  })

  it('menyorot pelanggaran yang dilaporkan server pada isiannya', async () => {
    const user = userEvent.setup()
    installFetch(
      defaultReply(() => ({
        status: 422,
        body: {
          kode: 'validasi_gagal',
          pesan: 'Ada isian yang belum benar.',
          detail: [{ kolom: 'nama', pesan: 'Status Penolakan 2 wajib diisi.' }],
        },
      })),
    )
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: 'Tambah Penolakan Klaim' })
    await user.selectOptions(within(form).getByLabelText('Status Penolakan 1'), '1')
    await user.type(within(form).getByLabelText('Status Penolakan 2'), 'X')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText('Status Penolakan 2 wajib diisi.')).toBeInTheDocument()
    // Form TETAP terbuka saat gagal; menutupnya akan membuang isian pengguna.
    expect(within(form).getByLabelText('Status Penolakan 2')).toBeInTheDocument()
  })
})

describe('mengubah penolakan klaim', () => {
  it('mengirim PUT ke baris yang dipilih', async () => {
    const user = userEvent.setup()
    installFetch(
      defaultReply(() => ({ body: { penolakan_klaim: LIST.penolakan_klaim[1], portal: 'ASM' } })),
    )
    show()

    const table = await screen.findByRole('table')
    await user.click(
      within(table).getByRole('button', { name: 'Ubah PERIODE PERTANGGUNGAN SUDAH BERAKHIR' }),
    )

    const form = await screen.findByRole('form', { name: 'Ubah Penolakan Klaim' })
    await user.clear(within(form).getByLabelText('Status Penolakan 2'))
    await user.type(within(form).getByLabelText('Status Penolakan 2'), 'TEKS BARU')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const sent = calls.find((call) => call.method === 'PUT')
      expect(sent?.url).toBe('/api/master/penolakan-klaim/2')
      expect(sent?.body).toEqual({ nama: 'TEKS BARU', id_status_1: '1', nama_status_1: '' })
    })
  })

  // Akibat menyimpan dinyatakan SEBELUM pengguna menekan Simpan.
  //
  // `MASTERPENOLAKANKLAIM2.prc:14` menyetel STATUS='0' pada setiap pengubahan, dan
  // perilaku itu dipertahankan atas keputusan Work Owner 2026-09-19. Baris yang belum
  // diputuskan tidak kehilangan apa pun, sehingga peringatannya tidak muncul di sana.
  it('memperingatkan bahwa menyimpan membatalkan persetujuan yang sudah ada', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    await user.click(
      within(table).getByRole('button', {
        name: 'Ubah PREMI BELUM DIBAYAR SAMPAI TANGGAL KEJADIAN',
      }),
    )

    const form = await screen.findByRole('form', { name: 'Ubah Penolakan Klaim' })
    expect(
      within(form).getByText('Menyimpan akan membatalkan persetujuan'),
    ).toBeInTheDocument()
  })

  it('tidak memperingatkan apa pun pada baris yang masih menunggu', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    await user.click(
      within(table).getByRole('button', { name: 'Ubah PERIODE PERTANGGUNGAN SUDAH BERAKHIR' }),
    )

    const form = await screen.findByRole('form', { name: 'Ubah Penolakan Klaim' })
    expect(
      within(form).queryByText('Menyimpan akan membatalkan persetujuan'),
    ).not.toBeInTheDocument()
  })

  it('tidak menyediakan cara mengisi kolom persetujuan', async () => {
    // Keempatnya diisi layar Inbox Manager (MENU_ID 58), bukan layar ini. Keputusan Work
    // Owner 2026-09-19: di sini baca-saja.
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    await user.click(
      within(table).getByRole('button', { name: 'Ubah PERIODE PERTANGGUNGAN SUDAH BERAKHIR' }),
    )

    const form = await screen.findByRole('form', { name: 'Ubah Penolakan Klaim' })
    for (const label of ['Status Aproval', 'Note Approval', 'Approve By', 'Tanggal Approve']) {
      expect(within(form).queryByLabelText(label)).not.toBeInTheDocument()
    }
  })
})

describe('tab Penolakan Komite', () => {
  it('menampilkan kedua kolomnya setelah tab dipilih', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('tab', { name: 'Penolakan Komite' }))

    const table = await screen.findByRole('table')
    expect(within(table).getByRole('columnheader', { name: 'ID Master' })).toBeInTheDocument()
    expect(
      within(table).getByRole('columnheader', { name: 'Note Komite Reject' }),
    ).toBeInTheDocument()
    expect(within(table).getByText('NILAI KLAIM DI BAWAH RISIKO SENDIRI')).toBeInTheDocument()
  })

  it('mengirim satu isian saja saat menambah', async () => {
    const user = userEvent.setup()
    installFetch(
      defaultReply(() => ({
        status: 201,
        body: { penolakan_komite: { id: '113', catatan: 'X' }, portal: 'ASM' },
      })),
    )
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('tab', { name: 'Penolakan Komite' }))
    await user.click(await screen.findByRole('button', { name: 'Tambah' }))

    const form = await screen.findByRole('form', { name: 'Tambah Penolakan Komite' })
    await user.type(within(form).getByLabelText('Note Komite Reject'), 'CATATAN BARU')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const sent = calls.find((call) => call.method === 'POST')
      expect(sent?.url).toBe('/api/master/penolakan-komite')
      expect(sent?.body).toEqual({ catatan: 'CATATAN BARU' })
    })
  })

  it('mengirim PUT ke baris yang dipilih saat mengubah', async () => {
    const user = userEvent.setup()
    installFetch(
      defaultReply(() => ({
        body: { penolakan_komite: { id: '112', catatan: 'TEKS BARU' }, portal: 'ASM' },
      })),
    )
    show()

    await screen.findByRole('table')
    await user.click(screen.getByRole('tab', { name: 'Penolakan Komite' }))

    const table = await screen.findByRole('table')
    await user.click(
      within(table).getByRole('button', { name: 'Ubah DOKUMEN PENDUKUNG TIDAK MEYAKINKAN' }),
    )

    const form = await screen.findByRole('form', { name: 'Ubah Penolakan Komite' })
    await user.clear(within(form).getByLabelText('Note Komite Reject'))
    await user.type(within(form).getByLabelText('Note Komite Reject'), 'TEKS BARU')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const sent = calls.find((call) => call.method === 'PUT')
      expect(sent?.url).toBe('/api/master/penolakan-komite/112')
      expect(sent?.body).toEqual({ catatan: 'TEKS BARU' })
    })
  })
})

describe('tanpa portal', () => {
  it('menyuruh memilih entitas dan tidak menembak API sama sekali', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(
      await screen.findByText(
        'Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu.',
      ),
    ).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })
})
