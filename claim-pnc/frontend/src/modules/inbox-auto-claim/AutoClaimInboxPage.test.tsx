import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { AutoClaimInboxPage } from './AutoClaimInboxPage'

// Ketiga tab beserta tabelnya datang dari server — layar tidak memuat daftarnya sendiri.
//
// Urutannya ditetapkan Work Owner (2026-09-20): Asuransi Kredit lebih dulu, dan tab
// pertama itulah yang terbuka. Ia SENGAJA berbeda dari urutan harness Pega, yang menyebut
// ANEKA lebih dulu.
//
// Label ANEKA juga tidak sama dengan nama rule-nya (`*_AutoClaim`) — itu sebabnya label
// datang dari server, bukan diturunkan dari nama rule.
const TABS = {
  bawaan: 'kredit',
  tab: [
    { kode: 'kredit', label: 'Asuransi Kredit', tabel: 'POOLDATA.TMP_BATCH_CLAIM_KREDIT' },
    { kode: 'aneka', label: 'ANEKA', tabel: 'POOLDATA.TMP_BATCH_AUTO_CLAIM' },
    { kode: 'travel', label: 'Travel', tabel: 'POOLDATA.TMP_BATCH_AUTO_TRAVEL' },
  ],
}

const TEMPLATE = {
  // Empat kolom wajib, mengikuti `Activity/InsertKlaimToTable_Other-Act.xml`. Yang TIDAK
  // ada di sini sama pentingnya: inisialid, prodke, dan currency ketiganya hasil
  // pencarian polis, bukan isian pengunggah.
  kolom_wajib: ['policyno', 'claimamount', 'dateofloss', 'reportdate'],
  kolom_opsional: ['causeofloss', 'keyword', 'alasanklaim'],
  kolom_tanggal: ['dateofloss', 'reportdate'],
  batas_baris: 5000,
}

const BATCH_LIST = {
  portal: 'ASM',
  paginasi: { halaman: 1, ukuran: 15, total: 3, total_halaman: 1 },
  batch: [
    {
      kode_perusahaan: 'MFIN',
      nama_perusahaan: 'Mitra Finansial Nusantara',
      batch: '1',
      tanggal_proses: '12/01/2026',
      jumlah_upload: 4,
      jumlah_proses: 4,
      jumlah_berhasil: 2,
      jumlah_gagal: 2,
      jumlah_belum_proses: 0,
      user_upload: 'ADMINPNC',
    },
    {
      kode_perusahaan: 'MFIN',
      nama_perusahaan: 'Mitra Finansial Nusantara',
      batch: '2',
      tanggal_proses: '05/01/2026',
      jumlah_upload: 2,
      jumlah_proses: 0,
      jumlah_berhasil: 0,
      jumlah_gagal: 0,
      jumlah_belum_proses: 2,
      user_upload: 'ADMINPNC',
    },
    {
      // Batch tanpa baris master. Ia yang membuktikan LEFT JOIN terbawa sampai ke layar.
      kode_perusahaan: 'ZZZZ',
      nama_perusahaan: '',
      batch: '1',
      tanggal_proses: '02/03/2026',
      jumlah_upload: 1,
      jumlah_proses: 1,
      jumlah_berhasil: 0,
      jumlah_gagal: 1,
      jumlah_belum_proses: 0,
      user_upload: 'ADMINPNC',
    },
  ],
}

const LINE_LIST = {
  portal: 'ASM',
  kode_perusahaan: 'MFIN',
  batch: '1',
  paginasi: { halaman: 1, ukuran: 15, total: 2, total_halaman: 1 },
  baris: [
    {
      nomor_polis: '0100120260001',
      prod_ke: '1',
      nomor_klaim: 'PNCN.26.101',
      nomor_aksep: 'AKS-26-0001',
      mata_uang: 'IDR',
      nilai_klaim: '12500000.00',
      penyebab_kerugian: '12002',
      tanggal_kejadian: '03/01/2026',
      tanggal_lapor: '05/01/2026',
      tanggal_proses: '12/01/2026',
      catatan: '',
      nama_objek: '',
      flag_tidak_bayar: '',
      keyword: 'REF-1',
      keterangan: 'Sukses Klaim',
      hasil: 'berhasil',
    },
    {
      nomor_polis: '0100120260003',
      prod_ke: '1',
      nomor_klaim: '',
      nomor_aksep: '',
      mata_uang: 'IDR',
      nilai_klaim: '4300000.00',
      penyebab_kerugian: '',
      tanggal_kejadian: '05/01/2026',
      tanggal_lapor: '07/01/2026',
      tanggal_proses: '12/01/2026',
      catatan: '',
      nama_objek: '',
      flag_tidak_bayar: '',
      keyword: 'REF-3',
      keterangan: 'Penyebab kerugian tidak ditemukan',
      hasil: 'gagal',
    },
  ],
}

type Call = {
  url: string
  method: string
  header: Record<string, string>
}

let calls: Call[] = []

type Reply = { body: unknown; status?: number }

function installFetch(map: (call: Call) => Reply) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
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

// SUMMARY WAJIB cocok dengan BATCH_LIST, dan kecocokannya bukan kerapian: aturan modul
// ini adalah angka ringkasan SAMA dengan total grid, sehingga data uji yang melanggarnya
// akan meloloskan layar yang menampilkan dua angka bertentangan.
//
// BATCH_LIST memuat tiga baris — MFIN dua, ZZZZ satu — dan paginasinya menyebut total 3.
// Angka di bawah mengikutinya persis.
//
// TIDAK ADA baris berjumlah nol, dan itu bukan kelalaian: server hanya mengirim
// perusahaan yang punya batch di tab ini (keputusan Work Owner 2026-09-20). Baris nol
// pernah ada di sini, dan justru itu yang menutupi cacatnya — setiap klik padanya
// menghasilkan grid kosong, sehingga penyaringnya terlihat "tidak berfungsi".
const SUMMARY = {
  portal: 'ASM',
  total: 3,
  perusahaan: [
    { kode: 'MFIN', nama: 'Mitra Finansial Nusantara', jumlah_batch: 2 },
    { kode: 'ZZZZ', nama: '', jumlah_batch: 1 },
  ],
}

/** defaultReply melayani kelima GET; unggahan dijawab pemanggil lewat `upload`. */
const PREMIUM_CHOICES = {
  bisnis: [
    { kode: '10104', nama: 'Asuransi Kredit' },
    { kode: '10105', nama: 'Asuransi Kredit Mikro' },
  ],
  sumber_bisnis: [{ kode: 'KRDU', nama: 'Kredit Utama Sejahtera' }],
  portal: 'ASM',
}

function defaultReply(upload?: (call: Call) => Reply, premium?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    // Rute Cek Premi dicocokkan SEBELUM pola rincian batch — /cek-premi/pilihan juga
    // berbentuk dua segmen.
    if (call.url.startsWith('/api/inbox-auto-claim/cek-premi/pilihan'))
      return { body: PREMIUM_CHOICES }
    if (call.url.startsWith('/api/inbox-auto-claim/cek-premi')) {
      if (premium) return premium(call)
      return {
        body: {
          kode_bisnis: '10104',
          kode_sumber_bisnis: 'KRDU',
          total_premi: '150000000',
          total_klaim: '24000000.50',
          portal: 'ASM',
        },
      }
    }
    if (call.url.startsWith('/api/inbox-auto-claim/tab')) return { body: TABS }
    if (call.url.startsWith('/api/inbox-auto-claim/ringkasan')) return { body: SUMMARY }
    if (call.url.startsWith('/api/inbox-auto-claim/format-unggahan')) return { body: TEMPLATE }
    if (call.url.startsWith('/api/inbox-auto-claim/unggah')) {
      if (upload) return upload(call)
      return { body: { batch: [], jumlah_baris: 0, ditolak: [], portal: 'ASM' }, status: 201 }
    }
    // Rincian batch: /api/inbox-auto-claim/MFIN/1?halaman=1
    if (/\/api\/inbox-auto-claim\/[^/?]+\/[^/?]+/.test(call.url)) return { body: LINE_LIST }

    // Daftar batch — dan stub ini benar-benar MENYARING.
    //
    // Versi pertama mengembalikan BATCH_LIST apa adanya untuk permintaan apa pun,
    // sehingga uji penyaring hanya dapat memeriksa URL-nya. Cacat berupa "permintaannya
    // terkirim tetapi tabelnya tidak berubah" — persis yang dilaporkan Work Owner —
    // TIDAK DAPAT ditangkap stub seperti itu.
    const filter = new URL(call.url, 'http://uji').searchParams.get('perusahaan') ?? ''
    if (filter === '') return { body: BATCH_LIST }

    const batch = BATCH_LIST.batch.filter((b) => b.kode_perusahaan === filter)
    return {
      body: {
        ...BATCH_LIST,
        batch,
        paginasi: { ...BATCH_LIST.paginasi, total: batch.length },
      },
    }
  }
}

async function uploadButton() {
  return within(await screen.findByRole('tabpanel')).getByRole('button', {
    name: 'Upload Data Klaim',
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <AutoClaimInboxPage />
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

/** Daftar perusahaan di panel kiri. */
async function companyList() {
  return screen.findByRole('navigation', { name: 'Daftar perusahaan' })
}

/** Tombol satu perusahaan di panel kiri, dicari dari awal namanya. */
async function companyButton(name: string) {
  const list = await companyList()
  return within(list).getByRole('button', { name: new RegExp(`^${name}`) })
}

/**
 * Memilih satu perusahaan lalu mengembalikan grid batch di panel kanan.
 *
 * Grid itu DIAMBIL ULANG oleh pemanggil bila perlu: DataTable melepas elemen <table>-nya
 * saat berganti ke keadaan memuat, sehingga simpulan lama menunjuk DOM yang sudah hilang.
 */
async function openCompany(name: string) {
  await userEvent.click(await companyButton(name))
  return screen.findByRole('table', { name: `Batch ${name}` })
}

/** Nama perusahaan di panel kiri, sesuai urutan tampil. */
function companyNames(list: HTMLElement) {
  return within(list)
    .getAllByRole('button')
    .map((b) => b.querySelector('span span')?.textContent)
}

describe('daftar perusahaan di kiri', () => {
  it('menampilkan judul, daftar perusahaan di kiri, dan tanpa grafik', async () => {
    installFetch(defaultReply())
    const { container } = show()

    expect(screen.getByRole('heading', { name: 'Inbox Auto Claim' })).toBeInTheDocument()
    const list = await companyList()
    expect(companyNames(list)).toEqual(['Mitra Finansial Nusantara', 'ZZZZ'])

    // Donut sudah dibuang (keputusan Work Owner 2026-09-27).
    expect(container.querySelector('.recharts-surface')).toBeNull()
  })

  it('menampilkan kode di bawah nama dan jumlah batch per perusahaan', async () => {
    installFetch(defaultReply())
    show()

    const mfin = await companyButton('Mitra Finansial Nusantara')
    expect(mfin).toHaveTextContent('MFIN')
    expect(mfin).toHaveTextContent(/2$/)
  })

  it('menandai perusahaan yang tidak terdaftar di master, bukan menyembunyikannya', async () => {
    installFetch(defaultReply())
    show()

    const list = await companyList()
    expect(within(list).getByText('tidak terdaftar di Master Auto Claim')).toBeInTheDocument()
  })

  it('pencarian menyaring menurut nama maupun kode', async () => {
    installFetch(defaultReply())
    show()

    await companyList()
    const cari = screen.getByLabelText('Cari nama perusahaan')

    await userEvent.type(cari, 'mitra')
    expect(companyNames(await companyList())).toEqual(['Mitra Finansial Nusantara'])

    await userEvent.clear(cari)
    await userEvent.type(cari, 'zzz')
    expect(companyNames(await companyList())).toEqual(['ZZZZ'])

    await userEvent.clear(cari)
    await userEvent.type(cari, 'tidak-ada')
    expect(await screen.findByText(/Tidak ada perusahaan yang cocok/)).toBeInTheDocument()
  })

  it('memuat lebih dari 20 perusahaan dalam daftar yang digulir', async () => {
    // Alasan bentuk ini: tab Asuransi Kredit dapat memuat lebih dari 20 perusahaan.
    const banyak = Array.from({ length: 22 }, (_, i) => ({
      kode: `K${String(i + 1).padStart(2, '0')}`,
      nama: `Perusahaan ${String(i + 1).padStart(2, '0')}`,
      jumlah_batch: 1,
    }))
    installFetch((call) =>
      call.url.startsWith('/api/inbox-auto-claim/ringkasan')
        ? { body: { portal: 'ASM', total: 22, perusahaan: banyak } }
        : defaultReply()(call),
    )
    show()

    expect(companyNames(await companyList())).toHaveLength(22)
  })

  it('tidak ada baris berjumlah nol', async () => {
    installFetch(defaultReply())
    show()

    const angka = within(await companyList())
      .getAllByRole('button')
      .map((b) => Number(b.lastElementChild?.textContent?.trim()))
    expect(angka.length).toBeGreaterThan(0)
    for (const n of angka) expect(n).toBeGreaterThan(0)
  })

  it('tidak memasang dropdown perusahaan', async () => {
    installFetch(defaultReply())
    show()

    await companyList()
    expect(screen.queryByRole('combobox', { name: 'Nama Perusahaan' })).not.toBeInTheDocument()
  })

  it('mengirim header portal pada setiap permintaan data entitas', async () => {
    installFetch(defaultReply())
    show()

    await openCompany('Mitra Finansial Nusantara')

    const dataCall = calls.filter(
      (c) => !c.url.includes('format-unggahan') && !c.url.startsWith('/api/inbox-auto-claim/tab'),
    )
    expect(dataCall.length).toBeGreaterThan(0)
    for (const call of dataCall) {
      expect(call.header['X-Portal']).toBe('ASM')
    }
  })

  it('menaruh Upload Data Klaim di dalam panel tab, bukan di kepala halaman', async () => {
    installFetch(defaultReply())
    show()

    const panel = await screen.findByRole('tabpanel')
    expect(within(panel).getByRole('button', { name: 'Upload Data Klaim' })).toBeInTheDocument()
    // Panel diberi nama oleh tab yang aktif.
    const active = screen.getByRole('tab', { selected: true })
    expect(panel).toHaveAttribute('aria-labelledby', active.id)
  })

  it('menaruh Proses Klaim dan Generate DLA per batch, nonaktif, tanpa panel catatan', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table', { name: 'Batch Mitra Finansial Nusantara' })
    const proses = await within(table).findAllByRole('button', { name: /^Proses klaim batch / })
    const dla = within(table).getAllByRole('button', { name: /^Generate DLA batch / })
    expect(proses.length).toBeGreaterThan(0)
    expect(dla).toHaveLength(proses.length)
    for (const button of [...proses, ...dla]) expect(button).toBeDisabled()

    expect(screen.queryByRole('region', { name: 'Belum tersedia di aplikasi ini' })).toBeNull()
    expect(screen.queryByRole('button', { name: 'Cek Premi' })).toBeNull()
  })
})

describe('batch perusahaan di kanan', () => {
  it('perusahaan pertama langsung terpilih dan batchnya tampil di kanan', async () => {
    // Seperti contoh yang disetujui: panel kanan tidak dibiarkan kosong.
    installFetch(defaultReply())
    show()

    expect(
      await screen.findByRole('table', { name: 'Batch Mitra Finansial Nusantara' }),
    ).toBeInTheDocument()
    expect(await companyButton('Mitra Finansial Nusantara')).toHaveAttribute('aria-current', 'true')
  })

  it('grid batch ringkas tampil DI SAMPING daftar, tanpa kolom perusahaan', async () => {
    // Muat satu layar (Work Owner 2026-09-29): KODE dan Nama Perusahaan sudah di kepala
    // panel, Di Upload digabung ke Diproses.
    installFetch(defaultReply())
    show()

    const grid = await openCompany('Mitra Finansial Nusantara')
    const kepala = within(grid)
      .getAllByRole('columnheader')
      .map((h) => h.textContent)
    expect(kepala).toEqual([
      'Batch',
      'Tgl Proses',
      'Diproses',
      'Berhasil',
      'Gagal',
      'User Upload',
      'Aksi',
    ])
    // Pasangan diproses / diunggah tetap terbaca di satu sel.
    expect(within(grid).getAllByText('/ 4').length).toBeGreaterThan(0)
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    // Grid ada di panel kanan, BUKAN di dalam daftar kiri.
    const panel = screen.getByRole('region', { name: 'Batch perusahaan terpilih' })
    expect(panel.contains(grid)).toBe(true)
    expect((await companyList()).contains(grid)).toBe(false)
  })

  it('menyaring pada KODE perusahaan, bukan namanya', async () => {
    installFetch(defaultReply())
    show()

    await companyList()
    calls = []
    await openCompany('ZZZZ')

    expect(calls.some((c) => c.url.includes('perusahaan=ZZZZ'))).toBe(true)
    expect(calls.some((c) => c.url.includes('Mitra'))).toBe(false)
  })

  it('isi grid hanya milik perusahaan yang dipilih, dan berganti saat memilih yang lain', async () => {
    // Cacat yang pernah dilaporkan Work Owner: grid menampilkan batch perusahaan lain.
    installFetch(defaultReply())
    show()

    const grid = await openCompany('Mitra Finansial Nusantara')
    await waitFor(() =>
      expect(
        within(screen.getByRole('table', { name: 'Batch Mitra Finansial Nusantara' })).getAllByRole(
          'row',
        ),
      ).toHaveLength(3),
    ) // 1 judul + 2 batch MFIN
    expect(screen.queryByText(/kode ZZZZ perlu didaftarkan/)).not.toBeInTheDocument()
    expect(within(grid).getByText('+2 menunggu')).toBeInTheDocument()

    await openCompany('ZZZZ')
    expect(screen.queryByRole('table', { name: 'Batch Mitra Finansial Nusantara' })).toBeNull()
    const lain = screen.getByRole('table', { name: 'Batch ZZZZ' })
    expect(lain).toBeInTheDocument()
    expect(await screen.findByText(/kode ZZZZ perlu didaftarkan/)).toBeInTheDocument()
    expect(await companyButton('ZZZZ')).toHaveAttribute('aria-current', 'true')
  })

  it('mencari tidak menghilangkan batch yang sedang diperiksa', async () => {
    installFetch(defaultReply())
    show()

    await openCompany('ZZZZ')
    await userEvent.type(screen.getByLabelText('Cari nama perusahaan'), 'mitra')

    expect(screen.getByRole('table', { name: 'Batch ZZZZ' })).toBeInTheDocument()
  })

  it('menampilkan Total Data dan tombol halaman grid batch', async () => {
    installFetch(defaultReply())
    show()

    await openCompany('Mitra Finansial Nusantara')
    const bar = screen.getByRole('navigation', { name: 'Navigasi halaman' })
    expect(within(bar).getByText(/Total Data/)).toBeInTheDocument()
    expect(within(bar).getByRole('button', { name: 'Halaman pertama' })).toBeInTheDocument()
  })

  it('tombol ekspor nonaktif pada batch yang belum punya baris berhasil atau gagal', async () => {
    installFetch(defaultReply())
    show()

    await openCompany('Mitra Finansial Nusantara')
    expect(
      await screen.findByRole('button', { name: 'Export berhasil batch 2 MFIN' }),
    ).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Export gagal batch 2 MFIN' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Export berhasil batch 1 MFIN' })).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Export gagal batch 1 MFIN' })).toBeEnabled()
  })
})

describe('rincian batch', () => {
  it('membuka rincian sebagai POP-UP dan menampilkan hasilnya', async () => {
    // Permintaan Work Owner 2026-09-29: Detail dibuka sebagai pop-up, bukan di bawah grid.
    installFetch(defaultReply())
    show()

    await openCompany('Mitra Finansial Nusantara')
    await userEvent.click(await screen.findByRole('button', { name: 'Detail batch 1 MFIN' }))

    const dialog = await screen.findByRole('dialog', { name: 'Rincian batch 1' })
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(
      await within(dialog).findByRole('heading', { name: /Rincian batch 1/ }),
    ).toBeInTheDocument()
    expect(await within(dialog).findByText('PNCN.26.101')).toBeInTheDocument()
    expect(within(dialog).getByText('Penyebab kerugian tidak ditemukan')).toBeInTheDocument()
  })

  it('pop-up rincian tertutup dengan Escape dan dengan tombol Tutup rincian', async () => {
    installFetch(defaultReply())
    show()

    await openCompany('Mitra Finansial Nusantara')
    await userEvent.click(await screen.findByRole('button', { name: 'Detail batch 1 MFIN' }))
    await screen.findByRole('dialog', { name: 'Rincian batch 1' })
    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Detail batch 1 MFIN' }))
    const dialog = await screen.findByRole('dialog', { name: 'Rincian batch 1' })
    await userEvent.click(within(dialog).getByRole('button', { name: 'Tutup rincian' }))
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('menampilkan nilai uang dengan pemisah ribuan tanpa mengubah desimalnya', async () => {
    installFetch(defaultReply())
    show()

    await openCompany('Mitra Finansial Nusantara')
    await userEvent.click(await screen.findByRole('button', { name: 'Detail batch 1 MFIN' }))

    expect(await screen.findByText('12.500.000,00')).toBeInTheDocument()
  })
})

describe('unggah berkas klaim', () => {
  it('dibuka sebagai pop-up yang menyebut tab tujuan, dan Escape menutupnya', async () => {
    // Mengikuti local action Pega yang terbuka sebagai modal (Work Owner 2026-09-27).
    installFetch(defaultReply())
    show()

    await screen.findByRole('tablist', { name: 'Jenis klaim' })
    await userEvent.click(await uploadButton())

    const dialog = await screen.findByRole('dialog', { name: 'Upload Data Klaim' })
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    expect(within(dialog).getByText('Asuransi Kredit')).toBeInTheDocument()

    // Bentuk berkas diminta UNTUK TAB INI: berkas Kredit tidak punya tanggal kejadian
    // maupun tanggal lapor, dan petunjuk ANEKA justru yang membuatnya ditolak.
    await waitFor(() =>
      expect(
        calls.some((c) => c.url === '/api/inbox-auto-claim/format-unggahan?sumber=kredit'),
      ).toBe(true),
    )

    await userEvent.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })

  it('menampilkan judul kolom yang harus ada di berkas', async () => {
    installFetch(defaultReply())
    show()

    await userEvent.click(await uploadButton())

    expect(await screen.findByRole('heading', { name: 'Upload Data Klaim' })).toBeInTheDocument()
    expect(screen.getByText(/policyno, claimamount, dateofloss, reportdate/)).toBeInTheDocument()
  })

  it('menyatakan kode perusahaan tidak perlu diisi karena dicari dari polis', async () => {
    // Ketiadaan kolomnya justru yang paling membingungkan: petugas yang terbiasa
    // mengisi kode perusahaan akan mencari kolomnya dan mengira daftarnya kurang.
    installFetch(defaultReply())
    show()

    await userEvent.click(await uploadButton())

    expect(
      await screen.findByText(/Kode perusahaan, nomor produk, dan mata uang/),
    ).toBeInTheDocument()
  })

  it('mengirim berkas sebagai multipart dan melaporkan nomor batch yang terbit', async () => {
    installFetch(
      defaultReply(() => ({
        status: 201,
        body: {
          portal: 'ASM',
          jumlah_baris: 2,
          ditolak: [],
          batch: [
            {
              kode_perusahaan: 'MFIN',
              nama_perusahaan: 'Mitra Finansial Nusantara',
              batch: '3',
              jumlah_baris: 2,
              jumlah_lolos: 2,
              jumlah_bertanda: 0,
            },
          ],
        },
      })),
    )
    show()

    await userEvent.click(await uploadButton())

    const file = new File(['policyno,claimamount\nP1,1.00\n'], 'klaim.csv', { type: 'text/csv' })
    await userEvent.upload(screen.getByLabelText('Berkas CSV'), file)
    await userEvent.click(screen.getByRole('button', { name: 'Unggah' }))

    // Nomor batch yang terbit disebut di ringkasan: pengguna tidak punya cara lain
    // mengetahuinya, karena nomornya diterbitkan server.
    const note = (await screen.findByText('2 baris tersimpan')).closest('div')
    expect(note).not.toBeNull()
    expect(
      within(note as HTMLElement).getByText(/Mitra Finansial Nusantara — batch/),
    ).toBeInTheDocument()

    // Content-Type SENGAJA tidak diisi klien: peramban yang mengisinya lengkap dengan
    // boundary. Menuliskannya akan membuat server menolak badan permintaan.
    const uploadCall = calls.find((c) => c.url.includes('/unggah'))
    expect(uploadCall?.method).toBe('POST')
    expect(uploadCall?.header['Content-Type']).toBeUndefined()
    expect(uploadCall?.header['X-Portal']).toBe('ASM')
  })

  it('membedakan baris yang TERSIMPAN BERTANDA dari baris yang DITOLAK', async () => {
    // Pembedaan ini bukan kerapian tampilan. Baris "bertanda" tersimpan dan terlihat di
    // grid; baris "ditolak" tidak ada di mana pun. Pengguna yang mengira keduanya sama
    // akan mencari baris yang tidak pernah tersimpan.
    installFetch(
      defaultReply(() => ({
        status: 201,
        body: {
          portal: 'ASM',
          jumlah_baris: 1,
          batch: [
            {
              kode_perusahaan: 'MFIN',
              nama_perusahaan: 'Mitra Finansial Nusantara',
              batch: '3',
              jumlah_baris: 1,
              jumlah_lolos: 0,
              jumlah_bertanda: 1,
            },
          ],
          ditolak: [
            { baris: 4, nomor_polis: '0900120260500', pesan: 'Sumber Bisnis Tidak ditemukan' },
          ],
        },
      })),
    )
    show()

    await userEvent.click(await uploadButton())
    const file = new File(['policyno,claimamount\nP1,1.00\n'], 'klaim.csv', { type: 'text/csv' })
    await userEvent.upload(screen.getByLabelText('Berkas CSV'), file)
    await userEvent.click(screen.getByRole('button', { name: 'Unggah' }))

    const note = (await screen.findByText('1 baris tersimpan')).closest('div')
    expect(note).not.toBeNull()
    const panel = note as HTMLElement

    expect(within(panel).getByText(/1 bertanda gagal/)).toBeInTheDocument()
    expect(within(panel).getByText('1 baris TIDAK tersimpan')).toBeInTheDocument()

    // Nomor barisnya disebut: tanpa itu, pengguna dengan berkas ratusan baris hanya tahu
    // "ada yang gagal" dan harus mencarinya sendiri. Barisnya dicari sebagai satu
    // <li> utuh, bukan dengan mencocokkan potongan teks — potongan "Baris" juga muncul
    // di kalimat penjelas, dan pencocokan longgar akan lulus tanpa membuktikan apa pun.
    const rejected = within(panel)
      .getAllByRole('listitem')
      .map((item) => item.textContent?.replace(/\s+/g, ' ').trim())

    expect(rejected).toContain('Baris 4 — 0900120260500 — Sumber Bisnis Tidak ditemukan')
  })

  it('menampilkan SELURUH pelanggaran baris, bukan yang pertama saja', async () => {
    // Kesetaraan perilaku (P-5): pada berkas ratusan baris, satu galat per percobaan
    // berarti mengunggah ulang ratusan kali.
    installFetch(
      defaultReply(() => ({
        status: 422,
        body: {
          kode: 'validasi_gagal',
          pesan: 'Ada baris yang belum benar. Perbaiki berkasnya lalu unggah lagi.',
          detail: [
            { kolom: 'baris 2 · inisialid', pesan: 'Kode perusahaan wajib diisi.' },
            { kolom: 'baris 3 · nopolis', pesan: 'Nomor polis wajib diisi.' },
          ],
        },
      })),
    )
    show()

    await userEvent.click(await uploadButton())

    const file = new File(['inisialid\n\n'], 'klaim.csv', { type: 'text/csv' })
    await userEvent.upload(screen.getByLabelText('Berkas CSV'), file)
    await userEvent.click(screen.getByRole('button', { name: 'Unggah' }))

    expect(await screen.findByText('2 baris perlu diperbaiki')).toBeInTheDocument()
    expect(screen.getByText('baris 2 · inisialid')).toBeInTheDocument()
    expect(screen.getByText('baris 3 · nopolis')).toBeInTheDocument()
  })
})

describe('portal belum dipilih', () => {
  it('menuntun pengguna memilih portal alih-alih menembak server', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(calls.some((c) => c.url.startsWith('/api/inbox-auto-claim?'))).toBe(false)
    expect(calls.some((c) => c.url.startsWith('/api/inbox-auto-claim/ringkasan'))).toBe(false)
  })
})

describe('tab jenis klaim', () => {
  it('menampilkan ketiga tab data ditambah Cek Premi, dengan Asuransi Kredit terbuka lebih dulu', async () => {
    installFetch(defaultReply())
    show()

    const tablist = await screen.findByRole('tablist', { name: 'Jenis klaim' })
    const nama = within(tablist)
      .getAllByRole('tab')
      .map((t) => t.textContent)
    expect(nama).toEqual(['Asuransi Kredit', 'ANEKA', 'Travel', 'Cek Premi'])
    expect(within(tablist).getByRole('tab', { name: 'Asuransi Kredit' })).toHaveAttribute(
      'aria-selected',
      'true',
    )
  })

  it('berpindah tab mengirim ?sumber= yang baru ke daftar perusahaan', async () => {
    installFetch(defaultReply())
    show()

    const tablist = await screen.findByRole('tablist', { name: 'Jenis klaim' })
    calls = []
    await userEvent.click(within(tablist).getByRole('tab', { name: 'ANEKA' }))

    await waitFor(() => {
      expect(
        calls.some(
          (c) =>
            c.url.startsWith('/api/inbox-auto-claim/ringkasan') && c.url.includes('sumber=aneka'),
        ),
      ).toBe(true)
    })
  })

  it('berpindah tab mengosongkan pencarian dan kembali ke perusahaan pertama', async () => {
    // Kode perusahaan dan kata kunci pada satu tabel belum tentu berlaku di tabel lain.
    installFetch(defaultReply())
    show()

    await openCompany('ZZZZ')
    await userEvent.type(screen.getByLabelText('Cari nama perusahaan'), 'zzz')

    const tablist = screen.getByRole('tablist', { name: 'Jenis klaim' })
    await userEvent.click(within(tablist).getByRole('tab', { name: 'Travel' }))

    expect(
      await screen.findByRole('table', { name: 'Batch Mitra Finansial Nusantara' }),
    ).toBeInTheDocument()
    expect(screen.queryByRole('table', { name: 'Batch ZZZZ' })).toBeNull()
    expect(screen.getByLabelText('Cari nama perusahaan')).toHaveValue('')
  })

  it('grid batch yang dibuka membaca tab yang sedang terbuka', async () => {
    installFetch(defaultReply())
    show()

    const tablist = await screen.findByRole('tablist', { name: 'Jenis klaim' })
    await companyList()
    // Dikosongkan SEBELUM pindah tab: perusahaan pertama langsung terpilih, sehingga
    // batchnya dimuat seketika tab berganti.
    calls = []
    await userEvent.click(within(tablist).getByRole('tab', { name: 'Travel' }))
    await openCompany('Mitra Finansial Nusantara')

    const batchCall = calls.find((c) => c.url.startsWith('/api/inbox-auto-claim?'))
    expect(batchCall?.url).toContain('sumber=travel')
    expect(batchCall?.url).toContain('perusahaan=MFIN')
  })

  it('menyebut tabel sumber sesuai tab yang terbuka', async () => {
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Sumber: POOLDATA.TMP_BATCH_CLAIM_KREDIT')).toBeInTheDocument()

    const tablist = screen.getByRole('tablist', { name: 'Jenis klaim' })
    await userEvent.click(within(tablist).getByRole('tab', { name: 'Travel' }))

    expect(await screen.findByText('Sumber: POOLDATA.TMP_BATCH_AUTO_TRAVEL')).toBeInTheDocument()
  })
})

describe('tab Cek Premi', () => {
  async function openPremiumTab() {
    await userEvent.click(await screen.findByRole('tab', { name: 'Cek Premi' }))
    return screen.findByRole('tabpanel')
  }

  it('menjadi tab keempat tanpa tombol Upload', async () => {
    installFetch(defaultReply())
    show()

    const tabs = await screen.findAllByRole('tab')
    expect(tabs.at(-1)).toHaveTextContent('Cek Premi')

    const panel = await openPremiumTab()
    expect(within(panel).queryByRole('button', { name: 'Upload Data Klaim' })).toBeNull()
    expect(within(panel).getByRole('button', { name: 'Cek Premi' })).toBeDisabled()
  })

  it('mengirim kedua kode dan menampilkan total premi serta total klaim', async () => {
    installFetch(defaultReply())
    show()
    const panel = await openPremiumTab()

    const business = within(panel).getByLabelText('Nama Bisnis')
    await waitFor(() => expect(business).toBeEnabled())
    await userEvent.selectOptions(business, '10104')
    await userEvent.selectOptions(within(panel).getByLabelText('Sumber Bisnis'), 'KRDU')
    await userEvent.click(within(panel).getByRole('button', { name: 'Cek Premi' }))

    const result = await within(panel).findByRole('region', { name: 'Hasil cek premi' })
    expect(result).toHaveTextContent('150.000.000')
    expect(result).toHaveTextContent('24.000.000,50')
    expect(result).toHaveTextContent('Kredit Utama Sejahtera')

    const call = calls.find((c) => c.url.startsWith('/api/inbox-auto-claim/cek-premi?'))
    const query = new URL(call?.url ?? '', 'http://uji').searchParams
    expect(query.get('kode_bisnis')).toBe('10104')
    expect(query.get('kode_sumber_bisnis')).toBe('KRDU')
    expect(call?.header['X-Portal']).toBe('ASM')
  })

  it('menyebut layanan yang mati alih-alih menampilkan angka', async () => {
    installFetch(
      defaultReply(undefined, () => ({
        status: 502,
        body: {
          kode: 'layanan_premi_gagal',
          pesan: 'Layanan cek premi tidak dapat dihubungi. Coba lagi beberapa saat lagi.',
        },
      })),
    )
    show()
    const panel = await openPremiumTab()

    const business = within(panel).getByLabelText('Nama Bisnis')
    await waitFor(() => expect(business).toBeEnabled())
    await userEvent.selectOptions(business, '10104')
    await userEvent.selectOptions(within(panel).getByLabelText('Sumber Bisnis'), 'KRDU')
    await userEvent.click(within(panel).getByRole('button', { name: 'Cek Premi' }))

    expect(
      await within(panel).findByText(/Layanan cek premi tidak dapat dihubungi/),
    ).toBeInTheDocument()
    expect(within(panel).queryByRole('region', { name: 'Hasil cek premi' })).toBeNull()
  })
})
