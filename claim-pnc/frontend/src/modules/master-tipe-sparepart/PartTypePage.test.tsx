import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { PartTypePage } from './PartTypePage'

/** Tipe yang sudah disetujui, bertaut kategori yang sah. */
const APPROVED = {
  id_tipe_sparepart: '1',
  nama_tipe_sparepart: 'FUEL FILTER',
  id_kategori_sparepart: '1',
  nama_kategori_sparepart: 'ENGINE',
  status: '1',
  status_label: 'Approve',
}

/**
 * Tipe YATIM: kategorinya tidak ada di master kategori.
 *
 * Di Pega baris seperti ini HILANG dari layar karena inner join-nya. Ia ada di sini supaya
 * selisih perilaku yang disengaja itu benar-benar diuji, bukan hanya dicatat di komentar.
 */
const ORPHAN = {
  id_tipe_sparepart: '7',
  nama_tipe_sparepart: 'SPROCKET',
  id_kategori_sparepart: '99',
  nama_kategori_sparepart: '',
  status: '1',
  status_label: 'Approve',
}

/** Tipe yang menunggu keputusan. */
const PENDING = {
  id_tipe_sparepart: '4',
  nama_tipe_sparepart: 'TRACK ROLLER',
  id_kategori_sparepart: '3',
  nama_kategori_sparepart: 'UNDERCARRIAGE',
  status: '0',
  status_label: 'Waiting Approval',
}

/**
 * Tipe yang DITOLAK.
 *
 * Ia yang membuat perilaku paling mengejutkan pada modul ini dapat diuji: namanya tetap
 * memblokir pemakaian nama itu — di kategori mana pun — padahal barisnya tidak terlihat
 * dari tab mana pun selain tab Reject.
 */
const REJECTED = {
  id_tipe_sparepart: '6',
  nama_tipe_sparepart: 'CONTROL VALVE',
  id_kategori_sparepart: '2',
  nama_kategori_sparepart: 'HYDRAULIC',
  status: '2',
  status_label: 'Reject',
}

const CATEGORY_OPTIONS = {
  kategori: [
    { kode: '1', nama: 'ENGINE' },
    { kode: '2', nama: 'HYDRAULIC' },
    { kode: '3', nama: 'UNDERCARRIAGE' },
  ],
  terpotong: false,
  portal: 'ASM',
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

/** defaultReply melayani daftar dan daftar pilihan; mutasi dijawab pemanggil. */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.method === 'GET') {
      if (call.url.startsWith('/api/master/tipe-sparepart/pilihan')) {
        return { body: CATEGORY_OPTIONS }
      }

      const status = new URL(call.url, 'http://x').searchParams.get('status') ?? '1'

      let rows: unknown[] = []
      if (status === '1') rows = [APPROVED, ORPHAN]
      if (status === '0') rows = [PENDING]
      if (status === '2') rows = [REJECTED]

      return { body: { tipe_sparepart: rows, status, portal: 'ASM' } }
    }

    if (mutation) return mutation(call)
    return { body: { tipe_sparepart: APPROVED, portal: 'ASM' }, status: 200 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <PartTypePage />
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

/** listCalls mengambil permintaan daftar saja — bukan permintaan daftar pilihan. */
function listCalls() {
  return calls.filter(
    (c) => c.method === 'GET' && c.url.startsWith('/api/master/tipe-sparepart?'),
  )
}

/**
 * tab mengambil tombol tab menurut captionnya.
 *
 * Dicari DI DALAM bilah tab, bukan di seluruh halaman: caption tabnya — "Approve" dan
 * "Reject" — adalah caption Pega yang memang ditiru (D-13), dan kata yang sama muncul pada
 * tombol keputusan.
 */
function tab(name: string) {
  return within(
    screen.getByRole('navigation', { name: 'Tab Master Tipe Sparepart' }),
  ).getByRole('button', { name })
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

describe('daftar master tipe sparepart', () => {
  // Grid Pega punya EMPAT kolom, dan yang keempat bukan kolom tabel ini — ia
  // PART_CATEGORY_NAME milik tabel kategori, dibaca lewat JOIN.
  it('menampilkan judul dan keempat kolom grid Pega', async () => {
    installFetch(defaultReply())
    show()

    expect(screen.getByRole('heading', { name: 'Master Tipe Sparepart' })).toBeInTheDocument()

    const table = await screen.findByRole('table')
    for (const header of [
      'ID Tipe Sparepart',
      'Nama Tipe Sparepart',
      'ID Kategori Sparepart',
      'Kategori Sparepart',
    ]) {
      expect(within(table).getByRole('columnheader', { name: header })).toBeInTheDocument()
    }
  })

  // Tabelnya tidak punya kolom pencatat pelaku maupun stempel waktu. Menggambar kolom yang
  // selamanya kosong bukan kesetaraan melainkan peniruan yang keliru.
  it('tidak menggambar kolom User Update maupun Tanggal Update', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    expect(
      within(table).queryByRole('columnheader', { name: 'User Update' }),
    ).not.toBeInTheDocument()
    expect(
      within(table).queryByRole('columnheader', { name: 'Tanggal Update' }),
    ).not.toBeInTheDocument()
  })

  it('membuka pada tab Approve', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(listCalls()[0]?.url).toContain('status=1')
    expect(await screen.findByText('FUEL FILTER')).toBeInTheDocument()
  })

  // Nama kategori ditampilkan berdampingan dengan ID-nya, persis grid Pega.
  it('menampilkan nama kategori induk setiap baris', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(await screen.findByText('ENGINE')).toBeInTheDocument()
  })

  // Baris yatim TETAP terlihat, dengan keterangan alih-alih sel kosong.
  //
  // Di Pega baris ini tidak akan pernah muncul. Menampilkannya adalah satu-satunya cara ia
  // dapat diperbaiki lewat tombol Ubah.
  it('menampilkan baris yang kategorinya hilang beserta keterangannya', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    expect(await screen.findByText('SPROCKET')).toBeInTheDocument()
    expect(screen.getByText('— kategori tidak ditemukan')).toBeInTheDocument()
  })

  // Urutan tab mengikuti layar lama: Approve, Reject, lalu Waiting Approval. Reject berada
  // di TENGAH, dan itu memang urutan yang dilihat petugas hari ini (D-13).
  it('menggambar ketiga tab pada urutan layar lama', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')

    const nav = screen.getByRole('navigation', { name: 'Tab Master Tipe Sparepart' })
    const label = within(nav)
      .getAllByRole('button')
      .map((button) => button.textContent)
    expect(label).toEqual(['Approve', 'Reject', 'Waiting Approval'])
  })

  it('memuat ulang daftar saat tab berganti', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(tab('Waiting Approval'))

    await waitFor(() => {
      expect(listCalls().some((c) => c.url.includes('status=0'))).toBe(true)
    })
    expect(await screen.findByText('TRACK ROLLER')).toBeInTheDocument()
  })

  // Portal ikut dikirim pada SETIAP permintaan (ADR-0030, R-20).
  it('mengirim header portal pada setiap permintaan', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByRole('table')
    for (const call of calls) {
      expect(Object.values(call.header)).toContain('ASM')
    }
  })
})

describe('form master tipe sparepart', () => {
  it('menggambar kedua isian dan dropdown kategorinya', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    expect(screen.getByLabelText('Nama Tipe Sparepart')).toBeInTheDocument()

    const select = await screen.findByLabelText('Kategori Sparepart')
    await waitFor(() => {
      expect(within(select).getByRole('option', { name: 'HYDRAULIC' })).toBeInTheDocument()
    })
  })

  // Pilihan kosongnya memakai caption Pega apa adanya (D-13).
  it('memakai caption ---PILIH KATEGORI--- sebagai pilihan kosong', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    const select = await screen.findByLabelText('Kategori Sparepart')
    expect(
      within(select).getByRole('option', { name: '---PILIH KATEGORI---' }),
    ).toBeInTheDocument()
  })

  // Kedua isian wajib disorot SEKALIGUS, bukan satu per satu.
  it('menolak simpan dengan kedua isian kosong dan menyorot keduanya', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nama tipe sparepart wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Kategori sparepart wajib dipilih.')).toBeInTheDocument()
    expect(calls.filter((c) => c.method === 'POST')).toHaveLength(0)
  })

  it('mengirim kedua isian saat menambah', async () => {
    installFetch(
      defaultReply((call) => ({
        body: { tipe_sparepart: { ...APPROVED, nama_tipe_sparepart: 'SWING MOTOR' }, portal: 'ASM' },
        status: call.method === 'POST' ? 201 : 200,
      })),
    )
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.type(screen.getByLabelText('Nama Tipe Sparepart'), 'SWING MOTOR')
    await userEvent.selectOptions(await screen.findByLabelText('Kategori Sparepart'), '2')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const post = calls.find((c) => c.method === 'POST')
      expect(post?.body).toEqual({
        nama_tipe_sparepart: 'SWING MOTOR',
        id_kategori_sparepart: '2',
      })
    })
  })

  // Menyimpan tidak pernah mengirim status maupun nama kategori — keduanya diturunkan
  // server, dan yang kedua milik tabel lain.
  it('tidak pernah mengirim status maupun nama kategori', async () => {
    installFetch(defaultReply((call) => ({ body: {}, status: call.method === 'POST' ? 201 : 200 })))
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.type(screen.getByLabelText('Nama Tipe Sparepart'), 'SWING MOTOR')
    await userEvent.selectOptions(await screen.findByLabelText('Kategori Sparepart'), '2')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const post = calls.find((c) => c.method === 'POST')
      expect(post).toBeDefined()
      expect(Object.keys(post?.body as object)).toEqual([
        'nama_tipe_sparepart',
        'id_kategori_sparepart',
      ])
    })
  })

  // Membuka Ubah memuat kedua isian dari barisnya.
  it('memuat baris yang disunting ke dalam form', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    const row = (await screen.findByText('FUEL FILTER')).closest('tr')
    await userEvent.click(within(row as HTMLElement).getByRole('button', { name: 'Ubah' }))

    expect(screen.getByLabelText('Nama Tipe Sparepart')).toHaveValue('FUEL FILTER')
    await waitFor(() => {
      expect(screen.getByLabelText('Kategori Sparepart')).toHaveValue('1')
    })

    // ID tipe ditampilkan di dalam FORM dan tidak dapat diubah. Dicari di dalam form, bukan
    // di seluruh halaman: angka "1" juga muncul sebagai ID kategori pada barisnya di tabel.
    const form = screen.getByRole('form', { name: 'Ubah Tipe Sparepart' })
    expect(within(form).getByText('ID Tipe Sparepart')).toBeInTheDocument()
    expect(within(form).getByText(/tidak dapat diubah/)).toBeInTheDocument()
  })

  /*
    Baris yatim dapat dibuka, dan kategorinya yang tidak sah TETAP muncul sebagai pilihan.

    Tanpa itu, dropdown-nya akan kosong diam-diam dan pengguna yang hanya ingin mengubah
    nama akan ikut memindahkan kategorinya tanpa menyadari.
  */
  it('mempertahankan kategori tidak sah sebagai pilihan saat baris yatim dibuka', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    const row = (await screen.findByText('SPROCKET')).closest('tr')
    await userEvent.click(within(row as HTMLElement).getByRole('button', { name: 'Ubah' }))

    const select = await screen.findByLabelText('Kategori Sparepart')
    await waitFor(() => {
      expect(
        within(select).getByRole('option', { name: '99 — kategori tidak ditemukan' }),
      ).toBeInTheDocument()
    })
    expect(select).toHaveValue('99')
  })

  // Akibat menyimpan dinyatakan di muka, bukan ditemukan setelah tombol ditekan.
  it('memperingatkan bahwa menyimpan mengembalikan baris ke antrean persetujuan', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))

    // Dicari DI DALAM form: "Waiting Approval" juga merupakan caption salah satu tab.
    const form = screen.getByRole('form', { name: 'Tambah Tipe Sparepart' })
    expect(within(form).getByText(/Waiting Approval/)).toBeInTheDocument()
    expect(within(form).getByText(/tidak muncul sebagai pilihan/)).toBeInTheDocument()
  })

  // Nama ganda dijelaskan lengkap: ia dapat dipakai baris di kategori LAIN, dan dapat
  // dipakai baris di tab Reject — keduanya tidak terlihat dari tab yang sedang dibuka.
  it('menjelaskan penolakan nama ganda beserta tempat mencarinya', async () => {
    installFetch(
      defaultReply(() => ({
        body: {
          kode: 'kunci_tipe_sparepart_sudah_ada',
          pesan: 'Nama tersebut telah digunakan. Silakan ganti dengan nama yang lain.',
          detail: [{ kolom: 'nama_tipe_sparepart', pesan: 'sudah dipakai' }],
        },
        status: 409,
      })),
    )
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.type(screen.getByLabelText('Nama Tipe Sparepart'), 'FUEL FILTER')
    await userEvent.selectOptions(await screen.findByLabelText('Kategori Sparepart'), '2')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nama itu sudah dipakai')).toBeInTheDocument()
    expect(screen.getByText(/kategori yang berbeda/)).toBeInTheDocument()
    expect(screen.getByText(/tab Reject/)).toBeInTheDocument()
  })

  // Kategori yang persetujuannya dicabut sementara form terbuka dijelaskan sebagai keadaan
  // yang berubah, bukan sebagai isian yang salah ketik.
  it('menjelaskan kategori yang sudah tidak tersedia', async () => {
    installFetch(
      defaultReply(() => ({
        body: {
          kode: 'kategori_sparepart_tidak_ditemukan',
          pesan: 'Kategori sparepart yang dipilih sudah tidak tersedia.',
          detail: [{ kolom: 'id_kategori_sparepart', pesan: 'tidak tersedia' }],
        },
        status: 409,
      })),
    )
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.type(screen.getByLabelText('Nama Tipe Sparepart'), 'SWING MOTOR')
    await userEvent.selectOptions(await screen.findByLabelText('Kategori Sparepart'), '2')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Kategori itu sudah tidak tersedia')).toBeInTheDocument()
  })

  // Form tetap terbuka saat penyimpanan gagal, supaya isian pengguna tidak terbuang.
  it('tidak menutup form ketika penyimpanan gagal', async () => {
    installFetch(
      defaultReply(() => ({
        body: { kode: 'kunci_tipe_sparepart_sudah_ada', pesan: 'sudah dipakai' },
        status: 409,
      })),
    )
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    await userEvent.type(screen.getByLabelText('Nama Tipe Sparepart'), 'FUEL FILTER')
    await userEvent.selectOptions(await screen.findByLabelText('Kategori Sparepart'), '2')
    await userEvent.click(screen.getByRole('button', { name: 'Simpan' }))

    await screen.findByText('Nama itu sudah dipakai')
    expect(screen.getByLabelText('Nama Tipe Sparepart')).toHaveValue('FUEL FILTER')
  })

  // Tanpa kategori yang disetujui, layar mengatakannya alih-alih menampilkan dropdown
  // kosong yang mustahil diisi.
  it('mengatakan bila belum ada kategori yang disetujui', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/tipe-sparepart/pilihan')) {
        return { body: { kategori: [], terpotong: false, portal: 'ASM' } }
      }
      return defaultReply()(call)
    })
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    expect(
      await screen.findByText(/Belum ada kategori sparepart yang disetujui/),
    ).toBeInTheDocument()
  })

  // Daftar yang terpotong dikatakan terang-terangan, bukan dibiarkan terbaca sebagai
  // "kategorinya belum dibuat".
  it('mengatakan bila daftar kategori terpotong', async () => {
    installFetch((call) => {
      if (call.url.startsWith('/api/master/tipe-sparepart/pilihan')) {
        return { body: { ...CATEGORY_OPTIONS, terpotong: true } }
      }
      return defaultReply()(call)
    })
    show()
    await screen.findByRole('table')

    await userEvent.click(screen.getByRole('button', { name: 'Tambah' }))
    expect(await screen.findByText(/Daftar kategori terpotong/)).toBeInTheDocument()
  })
})

describe('keputusan borongan', () => {
  // Centang hanya ada di tab Waiting Approval — hanya di sanalah ada yang perlu diputuskan.
  it('menggambar centang hanya pada tab Waiting Approval', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    expect(screen.queryByRole('checkbox')).not.toBeInTheDocument()

    await userEvent.click(tab('Waiting Approval'))
    await screen.findByText('TRACK ROLLER')
    expect(screen.getAllByRole('checkbox').length).toBeGreaterThan(0)
  })

  it('mengirim satu permintaan untuk seluruh baris yang dicentang', async () => {
    installFetch(
      defaultReply(() => ({
        body: { jumlah_berubah: 1, status: '1', status_label: 'Approve', portal: 'ASM' },
      })),
    )
    show()
    await screen.findByRole('table')

    await userEvent.click(tab('Waiting Approval'))
    await screen.findByText('TRACK ROLLER')

    await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih TRACK ROLLER' }))
    await userEvent.click(screen.getByRole('button', { name: 'Approve terpilih' }))

    await waitFor(() => {
      const post = calls.find((c) => c.url.endsWith('/keputusan'))
      expect(post?.body).toEqual({ id_tipe_sparepart: ['4'], status: '1' })
    })
  })

  // Tombolnya mati selama belum ada yang dicentang — bukan disembunyikan.
  it('mematikan tombol keputusan selama belum ada yang dicentang', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(tab('Waiting Approval'))
    await screen.findByText('TRACK ROLLER')

    expect(screen.getByRole('button', { name: 'Approve terpilih' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Reject terpilih' })).toBeDisabled()
  })

  // Centang dibuang saat berpindah tab: baris yang dipilih milik tab sebelumnya.
  it('membuang centang saat berpindah tab', async () => {
    installFetch(defaultReply())
    show()
    await screen.findByRole('table')

    await userEvent.click(tab('Waiting Approval'))
    await screen.findByText('TRACK ROLLER')
    await userEvent.click(screen.getByRole('checkbox', { name: 'Pilih TRACK ROLLER' }))
    expect(screen.getByRole('button', { name: 'Approve terpilih' })).toBeEnabled()

    await userEvent.click(tab('Approve'))
    await userEvent.click(tab('Waiting Approval'))
    await screen.findByText('TRACK ROLLER')

    expect(screen.getByRole('button', { name: 'Approve terpilih' })).toBeDisabled()
  })
})

describe('portal belum dipilih', () => {
  // Layar tidak menembak API tanpa portal; ia mengatakan apa yang harus dilakukan lebih
  // dulu (TKT-F6-002, R-20).
  it('meminta portal dipilih tanpa menembak API', async () => {
    useSelectedPortal.getState().clear()
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    expect(listCalls()).toHaveLength(0)
  })
})
