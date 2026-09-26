import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { GroupingPage } from './GroupingPage'

/**
 * Daftar Panel dan Tipe Kendaraan, persis seperti yang dijawab endpoint `/pilihan`.
 *
 * Keduanya DIBACA DARI BASIS DATA entitas — bukan konstanta yang ditanam di activity Pega.
 */
const OPTIONS = {
  panel: [
    { kode: 'PNL01', nama: 'BUMPER DEPAN' },
    { kode: 'PNL02', nama: 'KABIN' },
  ],
  tipe_kendaraan: [
    { kode: 'TY01', nama: 'EXCAVATOR' },
    { kode: 'TY02', nama: 'BULLDOZER' },
  ],
  portal: 'ASM',
}

/** Daftar Sisi, berbeda per panel — itulah yang membuktikan penyaringnya bekerja. */
const SIDES: Record<string, unknown[]> = {
  PNL01: [
    { kode: '1', nama: 'KIRI' },
    { kode: '2', nama: 'KANAN' },
  ],
  PNL02: [{ kode: '-', nama: '-' }],
}

/** Jawaban pencarian sparepart, padanan `Activity/SetDataSparepart-Act.xml`. */
const PARTS: Record<string, unknown> = {
  'SP-1001': {
    nomor_sparepart: 'SP-1001',
    nama_sparepart: 'FILTER OLI',
    kategori_sparepart: 'KAT01',
    tipe_sparepart: 'TIP01',
    kode_sparepart: 'KD-1001',
    tanggal_produksi: '12/05/2024',
    portal: 'ASM',
  },
}

/** Grouping yang sudah disetujui, seluruh kolomnya terisi. */
const APPROVED = {
  id_grouping: '1',
  nomor_sparepart: 'SP-1001',
  nama_sparepart: 'FILTER OLI',
  kategori_sparepart: 'KAT01',
  tipe_sparepart: 'TIP01',
  kode_sparepart: 'KD-1001',
  tanggal_produksi: '12/05/2024',
  id_panel: 'PNL02',
  nama_panel: 'KABIN',
  sisi: '1',
  sisi_label: 'KIRI',
  no_rangka: 'MHFXW1234K5678901',
  tipe_kendaraan: 'EXCAVATOR',
  grouping_dengan_no_rangka: '',
  nomor_grup: '0001',
  catatan: 'Pemasangan awal unit.',
  status: '1',
  status_label: 'Approve',
}

/**
 * Baris kedua pada GRUP YANG SAMA.
 *
 * Ia yang membuktikan gunanya layar ini: nomor grupnya identik dengan APPROVED meski
 * sparepart dan panelnya berbeda.
 */
const APPROVED_SAME_GROUP = {
  ...APPROVED,
  id_grouping: '2',
  nomor_sparepart: 'SP-1002',
  nama_sparepart: 'SEAL KIT BOOM',
  id_panel: 'PNL01',
  nama_panel: 'BUMPER DEPAN',
  sisi: '2',
  sisi_label: 'KANAN',
  grouping_dengan_no_rangka: 'MHFXW1234K5678901',
  catatan: '',
}

/** Grouping yang menunggu keputusan. */
const PENDING = {
  ...APPROVED,
  id_grouping: '3',
  nomor_sparepart: 'SP-1003',
  nama_sparepart: 'TRACK LINK',
  no_rangka: 'MHFZZ9876K1234567',
  nomor_grup: '0002',
  status: '0',
  status_label: 'Waiting Approval',
}

/**
 * Grouping yang ditolak, TANPA catatan dan TANPA tipe kendaraan.
 *
 * Keduanya keadaan yang SAH — tidak satu pun rule mewajibkannya — dan keduanya paling mudah
 * terlupa diuji karena baris pertama selalu mengisinya.
 */
const REJECTED = {
  ...APPROVED,
  id_grouping: '4',
  nomor_sparepart: 'SP-1001',
  no_rangka: 'MHFQQ5555K7654321',
  tipe_kendaraan: '',
  catatan: '',
  nomor_grup: '0003',
  status: '2',
  status_label: 'Reject',
}

type Call = {
  url: string
  method: string
  body: unknown
  header: Record<string, string>
}

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
 * defaultReply melayani ketiga jalur acuan dan daftar grouping; mutasi dijawab pemanggil.
 *
 * Ketiga jalur bersegmen statis diperiksa LEBIH DULU karena ketiganya berawalan sama dengan
 * jalur daftar — persis alasan yang sama yang membuat rutenya didaftarkan lebih dulu di
 * backend.
 */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.url.startsWith('/api/master/grouping-sparepart/pilihan')) {
      return { body: OPTIONS }
    }
    if (call.url.startsWith('/api/master/grouping-sparepart/sisi')) {
      const panel = new URL(call.url, 'http://x').searchParams.get('id_panel') ?? ''
      return { body: { sisi: SIDES[panel] ?? [], portal: 'ASM' } }
    }
    if (call.url.startsWith('/api/master/grouping-sparepart/sparepart')) {
      const number = new URL(call.url, 'http://x').searchParams.get('nomor') ?? ''
      const found = PARTS[number]
      if (!found) {
        return {
          body: {
            kode: 'sparepart_tidak_ditemukan',
            pesan: 'Data Sparepart tidak ditemukan.',
            detail: [{ kolom: 'nomor_sparepart', pesan: 'Tidak ada.' }],
          },
          status: 404,
        }
      }
      return { body: found }
    }

    if (call.method === 'GET') {
      const status = new URL(call.url, 'http://x').searchParams.get('status') ?? '1'

      let rows: unknown[] = []
      if (status === '1') rows = [APPROVED, APPROVED_SAME_GROUP]
      if (status === '0') rows = [PENDING]
      if (status === '2') rows = [REJECTED]

      return { body: { grouping: rows, status, portal: 'ASM' } }
    }

    if (mutation) return mutation(call)
    return { body: { grouping: APPROVED, portal: 'ASM' }, status: 200 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <GroupingPage />
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

/** listCalls mengambil permintaan daftar saja, membuang ketiga permintaan acuan. */
function listCalls() {
  return calls.filter(
    (c) => c.method === 'GET' && c.url.startsWith('/api/master/grouping-sparepart?'),
  )
}

beforeEach(() => {
  calls = []
  startSession()
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
})

describe('GroupingPage', () => {
  it('menampilkan keenam kolom grid Pega beserta nomor grup', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('SP-1001')

    // Keenam judul kolom diambil dari grid Pega apa adanya — termasuk "No Sparepart" dan
    // "Sisi Panel", yang di FORM bernama "Nomor Sparepart" dan "Sisi".
    for (const title of [
      'ID',
      'No Sparepart',
      'Nama Sparepart',
      'Nama Panel',
      'Sisi Panel',
      'No Rangka',
    ]) {
      expect(screen.getByRole('columnheader', { name: title })).toBeInTheDocument()
    }

    // Kolom yang DITAMBAHKAN; tanpa itu pengguna tidak punya cara melihat baris mana yang
    // tergabung dengan baris mana.
    expect(screen.getByRole('columnheader', { name: 'Nomor Grup' })).toBeInTheDocument()
  })

  it('menampilkan sebutan sisi, bukan sandinya', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('SP-1001')
    expect(screen.getByText('KIRI')).toBeInTheDocument()
    expect(screen.getByText('KANAN')).toBeInTheDocument()
  })

  it('menampilkan dua baris satu grup dengan nomor grup yang sama', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('SP-1002')
    // Dua baris, satu nomor grup — bentuk grouping yang sebenarnya.
    expect(screen.getAllByText('0001')).toHaveLength(2)
  })

  it('ketiga tab berurutan Approve, Reject, Waiting Approval seperti layar lama', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('SP-1001')

    const nav = screen.getByRole('navigation', { name: 'Tab Master Grouping Sparepart' })
    const label = within(nav)
      .getAllByRole('button')
      .map((one) => one.textContent)
    expect(label).toEqual(['Approve', 'Reject', 'Waiting Approval'])
  })

  it('berpindah tab menembak penyaring status yang berbeda', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('SP-1001')
    await userEvent.click(screen.getByRole('button', { name: 'Waiting Approval' }))

    await waitFor(() => {
      expect(listCalls().some((c) => c.url.includes('status=0'))).toBe(true)
    })
  })

  it('tombol keputusan hanya muncul di tab Waiting Approval', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('SP-1001')
    expect(screen.queryByRole('button', { name: 'Approve terpilih' })).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Waiting Approval' }))
    expect(await screen.findByRole('button', { name: 'Approve terpilih' })).toBeInTheDocument()
  })

  it('keputusan borongan mengirim SATU permintaan berisi seluruh baris tercentang', async () => {
    installFetch(
      defaultReply((call) => ({
        body: {
          jumlah_berubah: 1,
          status: '1',
          status_label: 'Approve',
          portal: 'ASM',
        },
        status: call.method === 'POST' ? 200 : 200,
      })),
    )
    show()

    await screen.findByText('SP-1001')
    await userEvent.click(screen.getByRole('button', { name: 'Waiting Approval' }))
    await screen.findByText('SP-1003')

    await userEvent.click(screen.getByRole('checkbox'))
    await userEvent.click(screen.getByRole('button', { name: 'Approve terpilih' }))

    await waitFor(() => {
      const decision = calls.find((c) => c.url.endsWith('/keputusan'))
      expect(decision).toBeDefined()
      expect(decision?.body).toEqual({ id_grouping: ['3'], status: '1' })
    })
  })

  it('form tambah mengisi lima isian turunan dari pencarian sparepart', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('SP-1001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    const number = screen.getByLabelText('Nomor Sparepart')
    await userEvent.type(number, 'SP-1001')
    await userEvent.tab()

    // Pencariannya dicakup ke FORM, bukan ke seluruh halaman: "FILTER OLI" juga muncul di
    // baris grid di belakangnya, dan yang sedang diuji adalah isian turunan pada form.
    const form = screen.getByRole('form', { name: /Tambah Master Grouping Sparepart/i })

    // Kelimanya BACA-SAJA dan datang dari server, bukan diketik pengguna.
    await waitFor(() => expect(within(form).getByText('FILTER OLI')).toBeInTheDocument())
    expect(within(form).getByText('KAT01')).toBeInTheDocument()
    expect(within(form).getByText('TIP01')).toBeInTheDocument()
  })

  it('nomor sparepart yang tidak ada memunculkan pesan yang menuntun', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('SP-1001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    await userEvent.type(screen.getByLabelText('Nomor Sparepart'), 'SP-TIDAK-ADA')
    await userEvent.tab()

    expect(await screen.findByText('Data Sparepart tidak ditemukan')).toBeInTheDocument()
  })

  it('daftar Sisi baru dibaca setelah panelnya dipilih', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('SP-1001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    // Belum ada panel dipilih: tidak satu pun permintaan sisi dikirim.
    expect(calls.some((c) => c.url.includes('/sisi'))).toBe(false)

    await userEvent.selectOptions(await screen.findByLabelText('Nama Panel'), 'PNL01')

    await waitFor(() => {
      expect(calls.some((c) => c.url.includes('id_panel=PNL01'))).toBe(true)
    })
    expect(await screen.findByRole('option', { name: 'KIRI' })).toBeInTheDocument()
  })

  it('mengganti panel membuang sisi yang sudah dipilih', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('SP-1001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    await userEvent.selectOptions(await screen.findByLabelText('Nama Panel'), 'PNL01')
    await screen.findByRole('option', { name: 'KIRI' })
    await userEvent.selectOptions(screen.getByLabelText('Sisi'), '1')
    expect(screen.getByLabelText('Sisi')).toHaveValue('1')

    await userEvent.selectOptions(screen.getByLabelText('Nama Panel'), 'PNL02')
    await waitFor(() => expect(screen.getByLabelText('Sisi')).toHaveValue(''))
  })

  it('menolak menggabungkan baris dengan nomor rangkanya sendiri', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('SP-1001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    await userEvent.type(screen.getByLabelText('Nomor Sparepart'), 'SP-1001')
    await userEvent.selectOptions(await screen.findByLabelText('Nama Panel'), 'PNL01')
    await screen.findByRole('option', { name: 'KIRI' })
    await userEvent.selectOptions(screen.getByLabelText('Sisi'), '1')
    await userEvent.type(screen.getByLabelText('No Rangka'), 'MHFAAA1111')
    await userEvent.type(screen.getByLabelText('Grouping Dengan No Rangka'), 'MHFAAA1111')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(
      await screen.findByText(/sama dengan No Rangka baris ini sendiri/i),
    ).toBeInTheDocument()
    expect(calls.some((c) => c.method === 'POST')).toBe(false)
  })

  it('menolak menyimpan sebelum keempat isian wajib terisi', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('SP-1001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nomor sparepart wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Nama panel wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('No rangka wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Sisi wajib diisi.')).toBeInTheDocument()
    expect(calls.some((c) => c.method === 'POST')).toBe(false)
  })

  it('badan permintaan tambah TIDAK memuat satu pun isian turunan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('SP-1001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    await userEvent.type(screen.getByLabelText('Nomor Sparepart'), 'SP-1001')
    await userEvent.selectOptions(await screen.findByLabelText('Nama Panel'), 'PNL01')
    await screen.findByRole('option', { name: 'KIRI' })
    await userEvent.selectOptions(screen.getByLabelText('Sisi'), '1')
    await userEvent.type(screen.getByLabelText('No Rangka'), 'MHFBBB2222')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const created = calls.find((c) => c.method === 'POST')
      expect(created).toBeDefined()

      const body = created?.body as Record<string, unknown>
      // Kedelapan isian yang memang dikirim.
      expect(Object.keys(body).sort()).toEqual(
        [
          'catatan',
          'grouping_dengan_no_rangka',
          'id_panel',
          'nama_panel',
          'no_rangka',
          'nomor_sparepart',
          'sisi',
          'tipe_kendaraan',
        ].sort(),
      )
      // Nama panelnya ikut, bukan hanya kodenya: NAMA_PANEL yang menjadi anggota kunci alami.
      expect(body['nama_panel']).toBe('BUMPER DEPAN')
      expect(body['id_panel']).toBe('PNL01')
    })
  })

  it('galat duplikat ditampilkan sebagai pesan yang menuntun', async () => {
    installFetch(
      defaultReply(() => ({
        body: {
          kode: 'grouping_sudah_ada',
          pesan: 'Data sudah ada.',
          detail: [{ kolom: 'nomor_sparepart', pesan: 'Kombinasinya sudah dipakai.' }],
        },
        status: 409,
      })),
    )
    show()

    await screen.findByText('SP-1001')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    await userEvent.type(screen.getByLabelText('Nomor Sparepart'), 'SP-1001')
    await userEvent.selectOptions(await screen.findByLabelText('Nama Panel'), 'PNL01')
    await screen.findByRole('option', { name: 'KIRI' })
    await userEvent.selectOptions(screen.getByLabelText('Sisi'), '1')
    await userEvent.type(screen.getByLabelText('No Rangka'), 'MHFCCC3333')

    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Data sudah ada')).toBeInTheDocument()
    // Form TIDAK ditutup: menutupnya akan membuang isian pengguna.
    expect(screen.getByLabelText('No Rangka')).toBeInTheDocument()
  })

  it('setiap permintaan menyebut portal entitas yang sedang dibuka', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('SP-1001')
    for (const call of calls) {
      expect(call.header['X-Portal']).toBe('ASM')
    }
  })

  it('tanpa portal, layar meminta memilih entitas alih-alih menembak server', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(listCalls()).toHaveLength(0)
  })
})
