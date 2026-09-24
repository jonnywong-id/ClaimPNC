import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/masking'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

/**
 * Baris contoh yang BENTUKNYA sama dengan produksi, isinya tidak.
 *
 * Yang ditiru: sub modul dipisah koma dengan koma di ujung, kuota yang sangat besar, dan
 * satu baris nonaktif. Ketiganya adalah keadaan yang benar-benar ada di portal ASM dan
 * paling mudah salah ditangani layar.
 *
 * Yang TIDAK ditiru: nama penggunanya. Modul ini memuat kewenangan melihat data pribadi —
 * menyalin daftar nama sungguhan ke berkas yang di-commit berarti menerbitkan daftar siapa
 * yang boleh membuka nomor KTP nasabah (`D-69`).
 */
const SAMPLE = [
  {
    id: '1',
    cabang: '100081',
    nama_cabang: 'KANTOR PUSAT',
    login: 'CONTOH.ADMIN',
    modul: 'PNCSearchKlaim',
    sub_modul: 'Registrasi,Dokumen,',
    maks_cari: 100000,
    maks_lihat: 100000,
    lihat_ktp: true,
    lihat_email: true,
    lihat_notelp: true,
    aktif: true,
    dicatat_oleh: 'CONTOH.SUPERVISOR',
    dicatat_pada: '2026-09-01T08:00:00Z',
  },
  {
    id: '2',
    cabang: '100069',
    nama_cabang: 'BOGOR',
    login: 'CONTOH.TEKNIK',
    modul: 'PNCSearchKlaim',
    sub_modul: '',
    maks_cari: 5,
    maks_lihat: 5,
    lihat_ktp: true,
    lihat_email: false,
    lihat_notelp: false,
    aktif: false,
    dicatat_oleh: 'CONTOH.SUPERVISOR',
    dicatat_pada: '2026-09-01T08:00:00Z',
  },
]

const BRANCHES = {
  cabang: [
    { kode: '100001', nama: 'AGENCY MANADO' },
    { kode: '100081', nama: 'KANTOR PUSAT' },
  ],
  total: 2,
  portal: 'ASM',
}

const PORTAL_LIST = {
  portal: [{ id: '202600101', nama: 'ASURANSI SINAR MAS', alias: 'ASM', siap: true }],
  utama: 'ASM',
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function installFetch(reply: (url: string, init?: RequestInit) => Response | Promise<Response>) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })
    if (url === '/api/portal') return Promise.resolve(jsonResponse(200, PORTAL_LIST))
    if (url === '/api/menu') return Promise.resolve(jsonResponse(200, { menu: [] }))
    if (url.startsWith(`${ROUTE}/cabang`)) return Promise.resolve(jsonResponse(200, BRANCHES))
    return Promise.resolve(reply(url, init))
  })
}

/** Peladen tiruan yang menjawab daftar dan menerima simpan serta ubah status. */
function installDefaultFetch() {
  installFetch((url, init) => {
    if (url === ROUTE && init?.method === 'POST') {
      const body = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
      return jsonResponse(201, {
        masking: { ...SAMPLE[0], id: '9', ...body, nama_cabang: 'AGENCY MANADO', aktif: true },
        portal: 'ASM',
      })
    }
    if (url.endsWith('/status') && init?.method === 'PUT') {
      const body = JSON.parse(String(init.body ?? '{}')) as { aktif: boolean }
      return jsonResponse(200, { masking: { ...SAMPLE[0], aktif: body.aktif }, portal: 'ASM' })
    }
    if (url.startsWith(`${ROUTE}/`) && init?.method === 'PUT') {
      const body = JSON.parse(String(init.body ?? '{}')) as Record<string, unknown>
      return jsonResponse(200, { masking: { ...SAMPLE[0], ...body }, portal: 'ASM' })
    }
    return jsonResponse(200, { masking: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/master/masking']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  // Layar berada di balik sesi. Tanpa ini SessionGuard melempar ke layar masuk.
  useSession.setState({
    token: 'token-uji',
    user: SAMPLE_PROFILE,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
  useSelectedPortal.getState().clear()
})

describe('daftar', () => {
  // Kolom dan urutannya diambil apa adanya dari `MasterProteksi_Sec`. Uji ini menjaga
  // susunannya, karena itulah yang paling mudah bergeser saat layar disunting kelak — dan
  // pergeserannya tidak menghasilkan galat apa pun, hanya layar yang tidak lagi sama.
  it('menyusun kolom persis seperti grid layar lama', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('CONTOH.ADMIN')

    const header = screen.getAllByRole('columnheader').map((cell) => cell.textContent?.trim())
    expect(header).toEqual([
      'Cabang',
      'Login',
      'Status Aktif',
      'KTP',
      'Email',
      'Notelp',
      'Max Lihat',
      'Max Cari',
      'Lihat Modul',
      'Aksi',
    ])
  })

  it('menampilkan baris nonaktif, tidak menyembunyikannya', async () => {
    installDefaultFetch()
    show()

    // `SearchData.Type == "1"` tidak menyaring apa pun. Sepuluh dari 25 baris produksi
    // berstatus tidak aktif; menyembunyikannya secara baku akan membuat 40% isinya lenyap.
    expect(await screen.findByText('CONTOH.TEKNIK')).toBeInTheDocument()
    expect(screen.getByText('TIDAK AKTIF')).toBeInTheDocument()
    expect(screen.getByText('AKTIF')).toBeInTheDocument()
  })

  it('memecah sub modul menjadi daftar, tanpa koma penutup yang menggantung', async () => {
    installDefaultFetch()
    show()

    expect(await screen.findByText('Registrasi')).toBeInTheDocument()
    expect(screen.getByText('Dokumen')).toBeInTheDocument()
    // Nilai mentahnya tidak ditampilkan apa adanya — koma penutup akan terbaca seperti ada
    // bagian yang hilang.
    expect(screen.queryByText('Registrasi,Dokumen,')).not.toBeInTheDocument()
  })

  it('menampilkan kuota besar dengan pemisah ribuan', async () => {
    installDefaultFetch()
    show()

    // 100.000 benar-benar ada di data produksi. Tanpa pemisah, ia mudah tertukar dengan
    // 10.000 saat dibaca sekilas.
    const cell = await screen.findAllByText('100.000')
    expect(cell.length).toBeGreaterThan(0)
  })

  it('menyebut entitas yang menjawab, bukan hanya yang diminta', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('CONTOH.ADMIN')
    expect(screen.getByText(/Portal entitas:/)).toBeInTheDocument()
  })

  it('mengirim header portal pada setiap permintaan', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('CONTOH.ADMIN')
    const listCall = calls.find((c) => c.url.startsWith(ROUTE) && !c.url.includes('/cabang'))
    expect(listCall).toBeDefined()
    expect((listCall?.init?.headers as Record<string, string>)['X-Portal']).toBe('ASM')
  })
})

describe('pencarian', () => {
  // Empat tipe, sama dengan `SearchData.Type` layar lama.
  it('menawarkan keempat tipe pencarian layar lama', async () => {
    installDefaultFetch()
    show()

    await screen.findByText('CONTOH.ADMIN')
    const pilihan = within(screen.getByLabelText('Tipe Pencarian'))
      .getAllByRole('option')
      .map((option) => option.textContent?.trim())

    expect(pilihan).toEqual(['Semua data', 'Nama cabang', 'Nama pengguna', 'Status aktif'])
  })

  it('mengirim tipe dan kata kunci sebagai parameter, bukan dirangkai', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.type(screen.getByLabelText('Nama Pencarian'), 'teknik')
    await user.click(screen.getByRole('button', { name: 'Cari' }))

    await waitFor(() => {
      const searched = calls.find((c) => c.url.includes('kata_kunci='))
      expect(searched).toBeDefined()
      expect(searched?.url).toContain('cari_di=login')
      expect(searched?.url).toContain('kata_kunci=teknik')
    })
  })

  // Tipe status memakai DROPDOWN, bukan kotak teks — sama seperti layar lama, yang
  // menampilkan kontrol berbeda untuk `SearchData.Type=='4'`.
  it('menukar kotak teks dengan dropdown saat mencari menurut status', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.selectOptions(screen.getByLabelText('Tipe Pencarian'), 'status')

    expect(await screen.findByLabelText('Status Aktif')).toBeInTheDocument()
    expect(screen.queryByLabelText('Nama Pencarian')).not.toBeInTheDocument()

    await user.selectOptions(screen.getByLabelText('Status Aktif'), 'TIDAK AKTIF')
    await user.click(screen.getByRole('button', { name: 'Cari' }))

    await waitFor(() => {
      const searched = calls.find((c) => c.url.includes('cari_di=status'))
      expect(searched).toBeDefined()
      // URLSearchParams menuliskan spasi sebagai '+', bukan '%20'.
      expect(searched?.url).toContain('kata_kunci=TIDAK+AKTIF')
    })
  })
})

describe('menambah', () => {
  it('mengirim isian form, dan menyembunyikan isian status', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: /Tambah/ }))

    const form = await screen.findByRole('form', { name: 'Tambah data masking' })

    // Baris baru selalu aktif; menawarkan pilihan yang tidak berpengaruh hanya
    // membingungkan.
    expect(within(form).queryByLabelText('Status')).not.toBeInTheDocument()

    await user.click(await within(form).findByRole('button', { name: /AGENCY MANADO/ }))
    await user.type(within(form).getByLabelText('Nama User'), 'PENGGUNA.BARU')
    await user.clear(within(form).getByLabelText('Max Cari Data'))
    await user.type(within(form).getByLabelText('Max Cari Data'), '5')
    await user.clear(within(form).getByLabelText('Max Lihat Data'))
    await user.type(within(form).getByLabelText('Max Lihat Data'), '7')
    await user.click(within(form).getByLabelText('Nomor KTP'))
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const posted = calls.find((c) => c.init?.method === 'POST')
      expect(posted).toBeDefined()

      const body = JSON.parse(String(posted?.init?.body ?? '{}')) as Record<string, unknown>
      expect(body['login']).toBe('PENGGUNA.BARU')
      expect(body['cabang']).toBe('100001')
      expect(body['maks_cari']).toBe(5)
      expect(body['maks_lihat']).toBe(7)
      expect(body['lihat_ktp']).toBe(true)

      // Ketiganya TIDAK pernah dikirim klien, dan server menolak badan yang memuatnya.
      expect(body).not.toHaveProperty('id')
      expect(body).not.toHaveProperty('nama_cabang')
      expect(body).not.toHaveProperty('dicatat_oleh')
    })
  })

  it('menolak isian kosong sebelum menembak server', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: /Tambah/ }))

    const form = await screen.findByRole('form', { name: 'Tambah data masking' })
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(await within(form).findByText('Nama pengguna wajib diisi.')).toBeInTheDocument()
    expect(within(form).getByText('Cabang wajib dipilih.')).toBeInTheDocument()
    expect(calls.some((c) => c.init?.method === 'POST')).toBe(false)
  })

  it('menerima kuota nol — ia kewenangan yang sah, bukan isian kosong', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: /Tambah/ }))

    const form = await screen.findByRole('form', { name: 'Tambah data masking' })
    await user.click(await within(form).findByRole('button', { name: /AGENCY MANADO/ }))
    await user.type(within(form).getByLabelText('Nama User'), 'TANPA.KUOTA')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const posted = calls.find((c) => c.init?.method === 'POST')
      expect(posted).toBeDefined()
      const body = JSON.parse(String(posted?.init?.body ?? '{}')) as Record<string, unknown>
      expect(body['maks_cari']).toBe(0)
    })
  })

  it('menampilkan penolakan pasangan ganda dengan pesan yang dapat ditindaklanjuti', async () => {
    installFetch((url, init) => {
      if (url === ROUTE && init?.method === 'POST') {
        return jsonResponse(409, {
          kode: 'masking_pengguna_sudah_ada',
          pesan: 'Pengguna itu sudah punya data masking di cabang tersebut.',
        })
      }
      return jsonResponse(200, { masking: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
    })
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: /Tambah/ }))

    const form = await screen.findByRole('form', { name: 'Tambah data masking' })
    await user.click(await within(form).findByRole('button', { name: /KANTOR PUSAT/ }))
    await user.type(within(form).getByLabelText('Nama User'), 'CONTOH.ADMIN')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    expect(
      await within(form).findByText(/sudah punya data masking di cabang ini/i),
    ).toBeInTheDocument()
  })
})

describe('aksi pada baris', () => {
  // `ActionMaskingData_Sec` memasang syarat `.STS_AKTF=='AKTIF'` pada KEDUA tombolnya.
  // Baris nonaktif karena itu tidak punya aksi apa pun di layar lama.
  it('hanya menampilkan aksi pada baris aktif', async () => {
    installDefaultFetch()
    show()

    expect(
      await screen.findByRole('button', { name: 'Ubah masking CONTOH.ADMIN' }),
    ).toBeInTheDocument()
    expect(
      screen.getByRole('button', { name: 'Nonaktifkan masking CONTOH.ADMIN' }),
    ).toBeInTheDocument()

    // Baris nonaktif: tidak ada Ubah, tidak ada Nonaktifkan.
    expect(
      screen.queryByRole('button', { name: 'Ubah masking CONTOH.TEKNIK' }),
    ).not.toBeInTheDocument()
    expect(
      screen.queryByRole('button', { name: /masking CONTOH\.TEKNIK/ }),
    ).not.toBeInTheDocument()
  })

  it('tombolnya disebut Nonaktifkan, bukan Hapus', async () => {
    installDefaultFetch()
    show()

    // Layar lama menamainya DELETE padahal ia tidak pernah membuang baris. Nama yang
    // keliru membuat petugas mengira datanya hilang lalu menambahkannya lagi dari awal.
    await screen.findByText('CONTOH.ADMIN')
    expect(screen.queryByRole('button', { name: /Hapus/ })).not.toBeInTheDocument()
  })

  it('mengirim status lewat jalurnya sendiri', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: 'Nonaktifkan masking CONTOH.ADMIN' }))

    await waitFor(() => {
      const sent = calls.find((c) => c.url.endsWith('/status'))
      expect(sent).toBeDefined()
      expect(sent?.init?.method).toBe('PUT')

      // Badannya menyebut keadaan yang dituju, bukan menyuruh "balikkan". Dua klik yang
      // tiba bersamaan tidak boleh saling meniadakan sampai pengguna tidak tahu kewenangan
      // itu akhirnya hidup atau mati.
      const body = JSON.parse(String(sent?.init?.body ?? '{}')) as { aktif: boolean }
      expect(body.aktif).toBe(false)
    })
  })
})

describe('mengubah', () => {
  // Form layar lama MEMUAT isian "STATUS" (`MasterProteksi_Sec:22449`), sehingga status
  // ikut tersimpan saat form disimpan.
  it('menampilkan isian status dan mengirimkannya', async () => {
    installDefaultFetch()
    show()
    const user = userEvent.setup()

    await screen.findByText('CONTOH.ADMIN')
    await user.click(screen.getByRole('button', { name: 'Ubah masking CONTOH.ADMIN' }))

    const form = await screen.findByRole('form', { name: 'Ubah data masking' })
    expect(within(form).getByLabelText('Status')).toBeInTheDocument()

    await user.selectOptions(within(form).getByLabelText('Status'), 'false')
    await user.click(within(form).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => {
      const sent = calls.find((c) => c.init?.method === 'PUT' && !c.url.endsWith('/status'))
      expect(sent).toBeDefined()
      const body = JSON.parse(String(sent?.init?.body ?? '{}')) as Record<string, unknown>
      expect(body['aktif']).toBe(false)
    })
  })
})

describe('portal', () => {
  it('menuntun memilih portal alih-alih menampilkan galat', async () => {
    installDefaultFetch()
    useSelectedPortal.getState().clear()
    show()

    expect(await screen.findByText('Portal entitas belum dipilih')).toBeInTheDocument()
    // Permintaan daftar tidak pernah ditembakkan tanpa portal.
    expect(calls.some((c) => c.url.startsWith(ROUTE))).toBe(false)
  })
})
