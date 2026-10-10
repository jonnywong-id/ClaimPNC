import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { PanelPage } from './PanelPage'

/**
 * Daftar pilihan Lokasi dan Sisi, persis seperti yang dijawab endpoint `/pilihan`.
 *
 * Isinya konstanta yang ditanam di `Activity/SetLokasiSisiPanel-Act.xml` — satu-satunya
 * daftar pilihan modul ini yang nilainya benar-benar diketahui.
 */
const OPTIONS = {
  lokasi_panel: ['KIRI', 'KANAN', 'DEPAN', 'BELAKANG', 'LAIN-LAIN'],
  sisi_panel: [
    { nilai: '-', label: '-' },
    { nilai: '1', label: 'KIRI' },
    { nilai: '2', label: 'KANAN' },
  ],
  unggah_tersedia: true,
}

/** Panel yang sudah disetujui, dengan DUA lokasi. */
const APPROVED = {
  id_panel: '01000001',
  nama_panel: 'Pintu Depan',
  status_repair: '1',
  status_edit_quantity: '1',
  status_premium_repair: '0',
  status_pecah: '0',
  status_sticker: '1',
  status_sisi: '1',
  status_rusak_parah: '0',
  status_aktif: '1',
  exclusion_c: '0',
  status_approval: '1',
  alasan_tolak: '',
  id_dokumen: '',
  status: '1',
  status_label: 'Approve',
  lokasi: [
    { lokasi_panel: 'KIRI', sisi_panel: '1', sisi_label: 'KIRI' },
    { lokasi_panel: 'KANAN', sisi_panel: '2', sisi_label: 'KANAN' },
  ],
}

/** Panel yang menunggu keputusan, dengan SATU lokasi. */
const PENDING = {
  ...APPROVED,
  id_panel: '01000002',
  nama_panel: 'Kaca Depan',
  status: '0',
  status_label: 'Waiting Approval',
  lokasi: [{ lokasi_panel: 'DEPAN', sisi_panel: '-', sisi_label: '-' }],
}

/**
 * Panel yang ditolak, TANPA lokasi sama sekali.
 *
 * Keadaan yang SAH — layar lama tidak mewajibkan satu pun baris lokasi — dan yang paling
 * mudah terlupa diuji.
 */
const REJECTED = {
  ...APPROVED,
  id_panel: '01000003',
  nama_panel: 'Bumper Belakang',
  alasan_tolak: 'Nama panel bertabrakan dengan panel yang sudah ada.',
  status: '2',
  status_label: 'Reject',
  lokasi: [],
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
      // FormData TIDAK diurai sebagai JSON. Unggahan dokumen memakai multipart, dan
      // memaksanya lewat JSON.parse akan melempar — kegagalan yang terbaca sebagai cacat
      // layar padahal bentuk permintaannya justru benar.
      body:
        init?.body instanceof FormData
          ? init.body
          : init?.body
            ? JSON.parse(init.body as string)
            : undefined,
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
 * defaultReply melayani daftar pilihan dan daftar panel; mutasi dijawab pemanggil.
 *
 * Jalur `/pilihan` diperiksa LEBIH DULU karena ia berawalan sama dengan jalur daftar —
 * persis alasan yang sama yang membuat rutenya didaftarkan lebih dulu di backend.
 */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.url.startsWith('/api/master/panel/pilihan')) return { body: OPTIONS }

    if (call.url.endsWith('/dokumen') && call.method === 'GET') {
      // Panel contoh belum punya dokumen; itulah jawaban yang WAJAR, bukan galat.
      return { body: { kode: 'dokumen_belum_ada', pesan: 'Panel ini belum punya dokumen.' }, status: 404 }
    }

    if (call.method === 'GET') {
      const status = new URL(call.url, 'https://x').searchParams.get('status') ?? '1'

      let rows: unknown[] = []
      if (status === '1') rows = [APPROVED]
      if (status === '0') rows = [PENDING]
      if (status === '2') rows = [REJECTED]

      return { body: { panel: rows, status, portal: 'ASM' } }
    }

    if (mutation) return mutation(call)
    return { body: { panel: APPROVED, portal: 'ASM' }, status: 200 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <PanelPage />
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

/** listCalls mengambil permintaan daftar saja, membuang permintaan daftar pilihan. */
function listCalls() {
  return calls.filter((c) => c.method === 'GET' && c.url.startsWith('/api/master/panel?'))
}

/**
 * tab mengambil tombol tab menurut captionnya.
 *
 * Dicari DI DALAM bilah tab, bukan di seluruh halaman: caption tabnya — "Approve" dan
 * "Reject" — adalah caption Pega yang memang ditiru (D-13), dan kata yang sama muncul pada
 * tombol keputusan.
 */
function tab(name: string) {
  return within(screen.getByRole('navigation', { name: 'Tab Master Panel' })).getByRole(
    'button',
    { name },
  )
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

describe('daftar master panel', () => {
  // Judul dan KESEBELAS kolom grid ditiru apa adanya dari layar Pega, termasuk huruf
  // besarnya — sembilan kapital dan satu ("Exclusion C") tidak.
  it('menampilkan judul dan kesebelas kolom persis seperti grid Pega', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Master Panel HE' })).toBeInTheDocument()

    const table = await screen.findByRole('table')
    for (const header of [
      'ID',
      'NAMA PANEL',
      'STATUS REPAIR',
      'STATUS EDIT QUANTITY',
      'STATUS PREMIUM REPAIR',
      'STATUS PECAH',
      'STATUS STICKER',
      'STATUS SISI',
      'STATUS RUSAK PARAH',
      'STATUS AKTIF',
      'Exclusion C',
    ]) {
      expect(within(table).getByRole('columnheader', { name: header })).toBeInTheDocument()
    }
  })

  // Grid Pega tidak punya kolom lokasi sama sekali; daftar lokasi hanya muncul di form.
  it('tidak menampilkan kolom lokasi di grid', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    expect(within(table).queryByRole('columnheader', { name: 'Lokasi' })).not.toBeInTheDocument()
  })

  // Urutan tab mengikuti urutan section di BrowsePanelHE: Approve, Reject, lalu Waiting
  // Approval. Reject berada di TENGAH — tidak intuitif, tetapi itulah layar lamanya.
  it('menampilkan tab pada urutan Pega: Approve, Reject, Waiting Approval', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    const nav = screen.getByRole('navigation', { name: 'Tab Master Panel' })
    const label = within(nav)
      .getAllByRole('button')
      .map((b) => b.textContent)
    expect(label).toEqual(['Approve', 'Reject', 'Waiting Approval'])
  })

  // Ketiga tombol unggah layar Pega TIDAK digambar sama sekali — bukan digambar mati.
  //
  // Keputusan Work Owner 2026-09-20, membalik pilihan sebelumnya. Uji ini menjaganya
  // supaya ketiganya tidak kembali tanpa keputusan baru: rule yang menjalankannya masih
  // tidak ada di export, sehingga menampilkannya berarti menjanjikan hal yang belum dapat
  // dikerjakan.
  // Tidak satu pun endpoint unggah ditembak dari layar ini.
  it('tidak menembak endpoint unggah mana pun', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(calls.filter((c) => c.url.includes('/dokumen') || c.url.includes('/unggah-csv')))
      .toHaveLength(0)
  })

  // Ketiga tab adalah tiga nilai status atas SATU endpoint, persis seperti ketiga section
  // pada Section/BrowsePanelHE-Section.xml.
  it('ketiga tab mengirim penyaring status yang benar', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(listCalls()[0]?.url).toContain('status=1')

    const user = userEvent.setup()

    await user.click(tab('Waiting Approval'))
    await waitFor(() => expect(listCalls().at(-1)?.url).toContain('status=0'))

    await user.click(tab('Reject'))
    await waitFor(() => expect(listCalls().at(-1)?.url).toContain('status=2'))
  })

  // Portal entitas disebut di setiap permintaan. Tanpa header itu, backend MENOLAK — ia
  // tidak pernah jatuh ke portal utama sebagai cadangan (R-20).
  it('menyebut portal entitas pada setiap permintaan daftar', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(listCalls()[0]?.header['X-Portal']).toBe('ASM')
  })

  // Nilai penanda ditampilkan APA ADANYA sebagai sandi. Menggantinya dengan "Ya"/"Tidak"
  // berarti menebak domain sembilan kolom yang daftar nilainya tidak ada di export.
  it('menampilkan nilai penanda apa adanya sebagai sandi', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    const row = within(table).getByText('Pintu Depan').closest('tr')
    expect(row).not.toBeNull()
    expect(within(row as HTMLElement).getByText('01000001')).toBeInTheDocument()
    expect(within(row as HTMLElement).getAllByText('1').length).toBeGreaterThan(0)
    expect(within(row as HTMLElement).getAllByText('0').length).toBeGreaterThan(0)
  })
})

describe('form master panel', () => {
  it('menampilkan kesepuluh isian wajib beserta daftar lokasi', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    for (const label of [
      'Nama panel',
      'Status repair',
      'Status edit quantity',
      'Status premium repair',
      'Status pecah',
      'Status sticker',
      'Status sisi',
      'Status rusak parah',
      'Status aktif',
      'Exclusion C',
    ]) {
      expect(screen.getByLabelText(label)).toBeInTheDocument()
    }

    expect(screen.getByRole('button', { name: 'Tambah lokasi' })).toBeInTheDocument()
  })

  // Panel tanpa lokasi tetap dapat disimpan — itu dinyatakan di layar, bukan dibiarkan
  // ditebak.
  it('menyatakan panel tanpa lokasi tetap dapat disimpan', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    expect(
      screen.getByText(/Panel tanpa lokasi tetap dapat disimpan/),
    ).toBeInTheDocument()
  })

  // Kesepuluh isian induk WAJIB; kesepuluhnya bertanda pyRequired=true di layar Pega.
  it('menolak penyimpanan saat isian wajib kosong', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nama panel wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Status repair wajib diisi.')).toBeInTheDocument()

    // Tidak ada permintaan POST yang terkirim: penolakan terjadi di layar.
    expect(calls.filter((c) => c.method === 'POST')).toHaveLength(0)
  })

  it('menambah dan menghapus baris lokasi', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    await user.click(screen.getByRole('button', { name: 'Tambah lokasi' }))
    expect(screen.getByLabelText('Lokasi panel 1')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /Hapus lokasi baris 1/ }))
    await waitFor(() => expect(screen.queryByLabelText('Lokasi panel 1')).not.toBeInTheDocument())
  })

  // Lokasi yang sama pada sisi yang sama ditolak. DITAMBAHKAN terhadap sistem lama, yang
  // tidak memeriksanya sama sekali.
  it('menolak lokasi kembar pada sisi yang sama', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    await user.type(screen.getByLabelText('Nama panel'), 'Kap Mesin')
    for (const label of [
      'Status repair',
      'Status edit quantity',
      'Status premium repair',
      'Status pecah',
      'Status sticker',
      'Status sisi',
      'Status rusak parah',
      'Status aktif',
      'Exclusion C',
    ]) {
      await user.type(screen.getByLabelText(label), '1')
    }

    await user.click(screen.getByRole('button', { name: 'Tambah lokasi' }))
    await user.click(screen.getByRole('button', { name: 'Tambah lokasi' }))

    await user.selectOptions(screen.getByLabelText('Lokasi panel 1'), 'KIRI')
    await user.selectOptions(screen.getByLabelText('Lokasi panel 2'), 'KIRI')

    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Lokasi ini sudah ada di baris 1.')).toBeInTheDocument()
    expect(calls.filter((c) => c.method === 'POST')).toHaveLength(0)
  })

  // Menyimpan mengirim SELURUH daftar lokasi, bukan hanya yang berubah.
  it('mengirim seluruh daftar lokasi saat menyimpan', async () => {
    installFetch(defaultReply(() => ({ body: { panel: APPROVED, portal: 'ASM' }, status: 200 })))
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Ubah' }))

    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.filter((c) => c.method === 'PUT')).toHaveLength(1))

    const sent = calls.find((c) => c.method === 'PUT')?.body as { lokasi: unknown[] }
    expect(sent.lokasi).toHaveLength(2)
  })

  // Menyimpan SELALU mengembalikan baris ke antrean; layar menyatakannya lebih dulu
  // supaya petugas tidak terkejut menemukan panelnya berpindah tab.
  it('menyatakan penyimpanan mengembalikan baris ke antrean persetujuan', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Ubah' }))

    // Dicari DI DALAM form, bukan di seluruh halaman: "Waiting Approval" juga nama
    // salah satu tab, dan pencarian yang tidak dibatasi menemukan keduanya.
    const form = screen.getByRole('form', { name: /Ubah Pintu Depan/ })
    expect(within(form).getByText(/kembali ke/)).toBeInTheDocument()
    expect(within(form).getByText('Waiting Approval')).toBeInTheDocument()
  })
})

describe('tanpa persetujuan di layar ini', () => {
  /*
    Layar Master Panel di Pega TIDAK punya persetujuan sama sekali.

    Ketiga tab hanya punya tombol Simpan, Ubah, dan Upload Document, serta NOL
    `pySelected` — tidak ada centang. Approve/Reject/Select All/Deselect All ada di
    `Section/ApprovalMasterPanelHE`, yang dimuat Harness/UserInbox_Harness dan
    Section/InboxManager_Sec — layar Inbox Manager.

    Keduanya sempat digambar di sini lalu dicabut Work Owner (2026-10-03). Uji ini
    menjaganya supaya tidak kembali tanpa keputusan baru.
  */
  it('tidak menggambar kolom centang pada tab mana pun', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    for (const name of ['Approve', 'Reject', 'Waiting Approval'] as const) {
      await user.click(tab(name))
      await waitFor(() => expect(screen.queryAllByRole('checkbox')).toHaveLength(0))
      expect(
        within(screen.getByRole('table')).queryByRole('columnheader', { name: 'Pilih' }),
      ).not.toBeInTheDocument()
    }
  })

  it('tidak menggambar tombol keputusan maupun isian catatan', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(tab('Waiting Approval'))

    expect(screen.queryByRole('button', { name: 'Approve terpilih' })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Reject terpilih' })).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Catatan')).not.toBeInTheDocument()
  })

  // Tidak satu pun permintaan ke endpoint keputusan boleh terkirim dari layar ini.
  it('tidak pernah menembak endpoint keputusan', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(tab('Waiting Approval'))
    await user.click(tab('Reject'))

    expect(calls.filter((c) => c.url.includes('/keputusan'))).toHaveLength(0)
  })
})

describe('paginasi', () => {
  /** manyRows menyusun sejumlah panel yang hanya berbeda kunci dan namanya. */
  function manyRows(count: number, status: string) {
    return Array.from({ length: count }, (_, i) => ({
      ...APPROVED,
      id_panel: `010001${String(i).padStart(2, '0')}`,
      nama_panel: `Panel Contoh ${i + 1}`,
      status,
      lokasi: [],
    }))
  }

  function installMany(count: number, status = '1') {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/panel/pilihan')) return { body: OPTIONS }
      const wanted = new URL(call.url, 'https://x').searchParams.get('status') ?? '1'
      const rows = wanted === status ? manyRows(count, status) : []
      return { body: { panel: rows, status: wanted, portal: 'ASM' } }
    })
  }

  /*
    Lima belas baris per halaman pada tab Approve.

    Angkanya bukan pilihan: `pyPageSize` pada Section/BrowsePanelHEApprove bernilai
    "Other" dan `pyPageSizeOther` bernilai 15 — dan tangkapan layar Pega yang dikirim Work
    Owner memperlihatkan tepat lima belas baris.
  */
  it('memotong tab Approve pada 15 baris per halaman', async () => {
    installMany(38)
    show()

    const table = await screen.findByRole('table')
    // Satu baris header ditambah 15 baris data.
    expect(within(table).getAllByRole('row')).toHaveLength(16)
    expect(screen.getByRole('status')).toHaveTextContent('Menampilkan 1–15 dari 38 baris.')
  })

  // Tab Reject dan Waiting Approval memakai 50, bukan 15. Perbedaannya ada di Pega dan
  // ditiru apa adanya; lihat catatan pada TABS.
  it('memotong tab Reject pada 50 baris per halaman', async () => {
    installMany(60, '2')
    show()

    // Tabel TIDAK ditunggu lebih dulu: tab awal (Approve) sengaja kosong pada uji ini,
    // dan DataTable menggambar pesan kosong alih-alih tabel. Bilah tab tidak bergantung
    // pada data, sehingga dapat langsung diklik.
    const user = userEvent.setup()
    await user.click(tab('Reject'))

    await waitFor(() =>
      expect(screen.getByRole('status')).toHaveTextContent('Menampilkan 1–50 dari 60 baris.'),
    )
  })

  it('berpindah halaman lewat nomor halaman', async () => {
    installMany(38)
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    await user.click(screen.getByRole('button', { name: 'Halaman 3' }))

    // Halaman ketiga memuat sisanya: 38 − 30 = 8 baris.
    await waitFor(() => {
      const table = screen.getByRole('table')
      expect(within(table).getAllByRole('row')).toHaveLength(9)
    })
    expect(screen.getByRole('status')).toHaveTextContent('Menampilkan 31–38 dari 38 baris.')
  })

  // Daftar yang lebih pendek dari satu halaman tidak menggambar paginator sama sekali.
  it('tidak menggambar paginator bila hanya ada satu halaman', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.queryByRole('navigation', { name: 'Halaman tabel' })).not.toBeInTheDocument()
  })
})

/**
 * Upload Document — satu-satunya unggahan yang digambar layar ini.
 *
 * Berkasnya berujung di GCS lewat `masterpanel/repo/dokumenlink` → `dokumenpenunjang` →
 * `POST /api/v1/upload`. Yang diuji di sini kontrak layarnya, bukan rantai itu: letak
 * tombolnya, bahwa berkasnya DITAHAN sampai Simpan, dan bahwa kegagalan yang melarang
 * pengulangan terbaca berbeda.
 */
describe('unggah dokumen Master Panel', () => {
  beforeEach(() => {
    calls = []
    startSession()
    useSelectedPortal.getState().select('ASM')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  const DOCUMENT_REPLY = {
    data: {
      data_id: '20260000000001',
      image_id: 'IMG000000001',
      nama_berkas: 'panel.pdf',
      tipe_media: 'application/pdf',
      catatan: '',
      diunggah_oleh: 'adminpnc',
      diunggah_pada: '2026-10-07T09:00:00Z',
      id_panel: '01000001',
    },
  }

  function uploadReply(status = 201, body?: unknown) {
    return defaultReply((call) => {
      if (call.url.endsWith('/dokumen') && call.method === 'POST') {
        return { status, body: body ?? DOCUMENT_REPLY }
      }
      return { body: { panel: APPROVED, portal: 'ASM' }, status: 200 }
    })
  }

  function uploadCalls() {
    return calls.filter((c) => c.method === 'POST' && c.url.endsWith('/dokumen'))
  }

  function pdf() {
    return new File(['isi'], 'panel.pdf', { type: 'application/pdf' })
  }

  it('menggambar Upload Document di kepala halaman', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(screen.getByRole('button', { name: 'Upload Document' })).toBeInTheDocument()
  })

  // Tombolnya TIDAK pernah dimatikan — `pyDisabledNew = false` pada sel tombol Pega.
  it('tidak pernah mematikan tombolnya', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/panel/pilihan')) {
        return { body: { ...OPTIONS, unggah_tersedia: false } }
      }
      return defaultReply()(call)
    })
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    expect(screen.getByRole('button', { name: 'Upload Document' })).toBeEnabled()

    // Sebabnya tertulis DI DALAM panel, bukan sebagai tombol mati tanpa keterangan.
    await user.click(screen.getByRole('button', { name: 'Upload Document' }))
    expect(await screen.findByText('Unggah dokumen belum tersedia')).toBeInTheDocument()
  })

  it('membuka panel berisi pemilih berkas dan daftar Nama File', async () => {
    installFetch(defaultReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    expect(screen.queryByLabelText('Berkas dokumen')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Upload Document' }))

    expect(await screen.findByLabelText('Berkas dokumen')).toBeEnabled()
    expect(screen.getByRole('button', { name: 'Pilih Berkas' })).toBeEnabled()
    // Baris kosong daftar "Nama File" — meniru "Data Tidak Ada" pada modal Pega.
    expect(screen.getByText('Nama File')).toBeInTheDocument()
    expect(screen.getByText('Data tidak ada')).toBeInTheDocument()
    // Modal Pega berpasangan Cancel/Submit; "Unggah" tidak pernah ada di sana.
    expect(screen.getByRole('button', { name: 'Submit' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Unggah' })).not.toBeInTheDocument()
  })

  // Berkas DITAHAN sampai Simpan ditekan — Submit hanya menutup panelnya, persis seperti
  // `SaveFilePenunjang` yang hanya menitipkan berkas ke halaman sementara.
  it('menahan berkas sampai Simpan ditekan, lalu mengirimnya SESUDAH panel tersimpan', async () => {
    installFetch(uploadReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    await user.click(screen.getByRole('button', { name: 'Upload Document' }))
    await screen.findByLabelText('Berkas dokumen')
    await user.upload(screen.getByLabelText('Berkas dokumen'), pdf())
    await user.type(screen.getByLabelText('Catatan dokumen'), 'Foto panel')
    await user.click(screen.getByRole('button', { name: 'Submit' }))

    expect(uploadCalls()).toHaveLength(0)

    await user.click(screen.getByRole('button', { name: 'Ubah' }))
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(uploadCalls()).toHaveLength(1))

    const saveIndex = calls.findIndex((c) => c.method === 'PUT')
    const uploadIndex = calls.findIndex((c) => c.method === 'POST' && c.url.endsWith('/dokumen'))
    expect(uploadIndex).toBeGreaterThan(saveIndex)

    const sent = uploadCalls()[0]
    expect(sent).toBeDefined()
    expect(sent!.body).toBeInstanceOf(FormData)
    expect(((sent!.body as FormData).get('berkas') as File).name).toBe('panel.pdf')
    expect((sent!.body as FormData).get('catatan')).toBe('Foto panel')
  })

  it('menyimpan tanpa memilih berkas tidak menembak endpoint dokumen', async () => {
    installFetch(uploadReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')
    await user.click(screen.getByRole('button', { name: 'Ubah' }))
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(calls.filter((c) => c.method === 'PUT')).toHaveLength(1))
    expect(uploadCalls()).toHaveLength(0)
  })

  // Golongan "separuh jalan" HARUS melarang pengulangan — mengulang menumpuk berkas ganda
  // di layanan penyimpanan. Panelnya dibuka kembali supaya pesannya terlihat.
  it('melarang unggah ulang saat berkas terkirim tetapi catatannya gagal', async () => {
    installFetch(
      uploadReply(500, {
        kode: 'unggah_separuh_jalan',
        pesan: 'Berkas sudah terkirim tetapi catatannya gagal disimpan.',
      }),
    )
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    await user.click(screen.getByRole('button', { name: 'Upload Document' }))
    await screen.findByLabelText('Berkas dokumen')
    await user.upload(screen.getByLabelText('Berkas dokumen'), pdf())
    await user.click(screen.getByRole('button', { name: 'Submit' }))

    await user.click(screen.getByRole('button', { name: 'Ubah' }))
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText(/JANGAN unggah ulang/)).toBeInTheDocument()
  })

})

/**
 * Kedua unggah CSV — Master Panel dan Lokasi Panel.
 *
 * Bentuknya sama dengan panel dokumen (label "Berkas CSV", daftar "Nama File", Batal /
 * Submit), tetapi ALIRANNYA berbeda dan itu berasal dari Pega: berkas CSV dikirim SEKETIKA
 * karena ia membawa kuncinya sendiri, sedangkan dokumen menunggu Simpan.
 */
describe('unggah CSV Master Panel', () => {
  beforeEach(() => {
    calls = []
    startSession()
    useSelectedPortal.getState().select('ASM')
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  const CSV_REPLY = {
    data: {
      total: 2,
      baru: 1,
      diperbarui: 0,
      gagal: 1,
      baris: [
        { baris: 2, nama_panel: 'KAP MESIN', id_panel: '01000009', hasil: 'baru', pesan: '' },
        { baris: 3, nama_panel: '', id_panel: '', hasil: 'gagal', pesan: 'Nama panel wajib diisi.' },
      ],
    },
  }

  function csvReply(status = 200, body?: unknown) {
    return defaultReply((call) => {
      if (call.url.includes('/unggah-csv')) {
        return { status, body: body ?? CSV_REPLY }
      }
      return { body: { panel: APPROVED, portal: 'ASM' }, status: 200 }
    })
  }

  function csv() {
    return new File(['NAME\nKAP MESIN\n'], 'panel.csv', { type: 'text/csv' })
  }

  it('menggambar ketiga tombol unggah', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')

    for (const label of ['Upload Document', 'Upload Data Master Panel', 'Upload Data Lokasi Panel']) {
      expect(screen.getByRole('button', { name: label })).toBeInTheDocument()
    }
  })

  // Satu panel terbuka pada satu waktu — ketiganya menempati tempat yang sama.
  it('membuka satu panel saja pada satu waktu', async () => {
    installFetch(csvReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    await user.click(screen.getByRole('button', { name: 'Upload Data Master Panel' }))
    expect(await screen.findByLabelText('Berkas CSV')).toBeInTheDocument()
    expect(screen.queryByLabelText('Berkas dokumen')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Upload Document' }))
    expect(await screen.findByLabelText('Berkas dokumen')).toBeInTheDocument()
    expect(screen.queryByLabelText('Berkas CSV')).not.toBeInTheDocument()
  })

  // Berbeda dari dokumen: berkas CSV DIKIRIM SEKETIKA, tanpa menunggu Simpan.
  it('mengirim berkas master seketika ke jalurnya sendiri', async () => {
    installFetch(csvReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    await user.click(screen.getByRole('button', { name: 'Upload Data Master Panel' }))
    await user.upload(await screen.findByLabelText('Berkas CSV'), csv())
    await user.click(screen.getByRole('button', { name: 'Submit' }))

    await waitFor(() =>
      expect(calls.filter((c) => c.url.endsWith('/unggah-csv'))).toHaveLength(1),
    )
    expect(calls.filter((c) => c.method === 'PUT')).toHaveLength(0)

    const sent = calls.find((c) => c.url.endsWith('/unggah-csv'))
    expect(sent).toBeDefined()
    expect(sent!.body).toBeInstanceOf(FormData)
    expect(((sent!.body as FormData).get('berkas') as File).name).toBe('panel.csv')
  })

  it('mengirim berkas lokasi ke jalur lokasi, bukan jalur master', async () => {
    installFetch(csvReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    await user.click(screen.getByRole('button', { name: 'Upload Data Lokasi Panel' }))
    await user.upload(await screen.findByLabelText('Berkas CSV'), csv())
    await user.click(screen.getByRole('button', { name: 'Submit' }))

    await waitFor(() =>
      expect(calls.filter((c) => c.url.endsWith('/unggah-csv-lokasi'))).toHaveLength(1),
    )
    // Dua endpoint terpisah, bukan satu yang menebak dari isi berkas.
    expect(calls.filter((c) => c.url.endsWith('/unggah-csv'))).toHaveLength(0)
  })

  // Laporan per baris: tanpa nomor barisnya, "1 baris gagal" memaksa pengguna menebak yang
  // mana di antara tiga ratus.
  it('menampilkan laporan per baris beserta nomor dan sebabnya', async () => {
    installFetch(csvReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    await user.click(screen.getByRole('button', { name: 'Upload Data Master Panel' }))
    await user.upload(await screen.findByLabelText('Berkas CSV'), csv())
    await user.click(screen.getByRole('button', { name: 'Submit' }))

    expect(await screen.findByText(/baris dibaca/)).toBeInTheDocument()
    expect(screen.getByText('Nama panel wajib diisi.')).toBeInTheDocument()
    // Baris yang berhasil TETAP tersimpan — tanpa kalimat ini pengguna akan mengunggah ulang
    // seluruh berkas dan menimpa baris yang sudah benar.
    expect(screen.getByText(/tetap tersimpan/)).toBeInTheDocument()
  })

  it('menutup panel lewat Batal tanpa mengirim apa pun', async () => {
    installFetch(csvReply())
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    await user.click(screen.getByRole('button', { name: 'Upload Data Master Panel' }))
    await user.upload(await screen.findByLabelText('Berkas CSV'), csv())
    await user.click(screen.getByRole('button', { name: 'Batal' }))

    expect(screen.queryByLabelText('Berkas CSV')).not.toBeInTheDocument()
    expect(calls.filter((c) => c.url.includes('/unggah-csv'))).toHaveLength(0)
  })

  // Berkas yang tidak terbaca sama sekali dibedakan dari baris yang gagal: di sini TIDAK ada
  // satu baris pun yang masuk, sehingga yang perlu diperbaiki adalah berkasnya.
  it('membedakan berkas yang tidak terbaca dari baris yang gagal', async () => {
    installFetch(
      csvReply(422, {
        kode: 'csv_tidak_sah',
        pesan: 'Header berkas tidak lengkap. Kolom yang kurang: STS_PECAH.',
      }),
    )
    show()

    const user = userEvent.setup()
    await screen.findByRole('table')

    await user.click(screen.getByRole('button', { name: 'Upload Data Master Panel' }))
    await user.upload(await screen.findByLabelText('Berkas CSV'), csv())
    await user.click(screen.getByRole('button', { name: 'Submit' }))

    expect(await screen.findByText('Berkas tidak dapat dibaca')).toBeInTheDocument()
    expect(screen.getByText(/STS_PECAH/)).toBeInTheDocument()
  })
})
