import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { HEADER_PORTAL } from '@/api/client'
import { AppRoute } from '@/app/App'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

const ROUTE = '/api/master/pic-teknik'
const DIRECTORY = `${ROUTE}/direktori`

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: 'contoh.admin@example.invalid',
  perusahaan: 'ASM',
}

/**
 * Contoh yang dipakai peladen tiruan.
 *
 * Isinya dikarang — isi MST_USER_TEKNIK tidak ada di export — dan memakai domain
 * `example.invalid` yang memang dicadangkan supaya tidak mungkin tertukar dengan pegawai
 * sungguhan.
 *
 * Baris ketiga sengaja NONAKTIF: grid Pega menampilkan Status Aktif `0` dan `1`
 * berdampingan, dan tanpa baris nonaktif aturan itu tidak benar-benar teruji.
 */
const SAMPLE = [
  {
    id_operator: 'PICTEKNIK01',
    nama: 'Contoh Kepala Teknik',
    email: 'contoh.kepalateknik@example.invalid',
    bisnis: 'NONMBU',
    kelompok: 'A',
    atasan: '',
    counter_klaim_kurang_1m: 6,
    counter_klaim_lebih_1m: 0,
    beban_kerja: 3,
    grup_panel: '',
    aktif: true,
  },
  {
    id_operator: 'PICTEKNIK03',
    nama: 'Contoh Petugas Teknik',
    email: 'contoh.petugas@example.invalid',
    bisnis: 'NONMBU',
    kelompok: 'C',
    atasan: 'PICTEKNIK01',
    counter_klaim_kurang_1m: 1463,
    counter_klaim_lebih_1m: 2,
    beban_kerja: 9,
    grup_panel: '',
    aktif: true,
  },
  {
    id_operator: 'PICTEKNIK04',
    nama: 'Contoh Petugas Nonaktif',
    email: 'contoh.nonaktif@example.invalid',
    bisnis: 'BONDING',
    kelompok: 'B',
    atasan: 'PICTEKNIK01',
    counter_klaim_kurang_1m: 0,
    counter_klaim_lebih_1m: 0,
    beban_kerja: 0,
    grup_panel: '',
    aktif: false,
  },
]

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
    return Promise.resolve(reply(url, init))
  })
}

function body(init: RequestInit | undefined): Record<string, unknown> {
  return JSON.parse(String(init?.body ?? '{}')) as Record<string, unknown>
}

function installDefaultFetch() {
  installFetch((url, init) => {
    if (url.startsWith(DIRECTORY)) {
      return jsonResponse(200, {
        pegawai: {
          id_operator: 'PICTEKNIK05',
          nama: 'Contoh Petugas Baru',
          email: 'contoh.baru@example.invalid',
          atasan: 'PICTEKNIK01',
          nama_atasan: 'Contoh Kepala Teknik',
        },
        portal: 'ASM',
      })
    }
    if (url === ROUTE && init?.method === 'POST') {
      return jsonResponse(201, {
        pic_teknik: { ...SAMPLE[0], ...body(init), nama: 'Contoh Petugas Baru' },
        portal: 'ASM',
      })
    }
    if (url.startsWith(`${ROUTE}/`) && init?.method === 'PUT') {
      return jsonResponse(200, { pic_teknik: { ...SAMPLE[0], ...body(init) }, portal: 'ASM' })
    }
    return jsonResponse(200, { pic_teknik: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/master/pic-teknik']}>
        <AppRoute />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
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

describe('grid', () => {
  /**
   * TUJUH kolom, dengan label dan urutan yang sama persis dengan grid Pega.
   *
   * Uji ini yang menahan kolom karangan masuk kembali — layar ini pernah punya kolom
   * "Nama", "Beban / Kuota", dan penanda "penuh" yang tidak satu pun ada di Pega.
   */
  it('menampilkan tujuh kolom Pega dengan label dan urutan yang sama', async () => {
    installDefaultFetch()
    show()
    await screen.findAllByText('PICTEKNIK01')

    const expected = [
      'Input Nama',
      'Atasan',
      'Status Aktif',
      'Bisnis',
      'Kelompok',
      'Counter Klaim <1M',
      'Counter Klaim >1M',
    ]
    for (const title of expected) {
      // getAllBy: DataTable menggambar judul kolom dua kali — tabel dan kartu layar sempit.
      expect(screen.getAllByText(title).length).toBeGreaterThan(0)
    }
  })

  it('tidak menampilkan kolom yang tidak ada di Pega', async () => {
    installDefaultFetch()
    show()
    await screen.findAllByText('PICTEKNIK01')

    expect(screen.queryAllByText('Beban / Kuota')).toHaveLength(0)
    expect(screen.queryAllByText('Kuota sistem lain')).toHaveLength(0)
    expect(screen.queryAllByText('penuh')).toHaveLength(0)
    // TOTAL_JOB tidak pernah ditampilkan: Report Definition menyebutnya, grid-nya tidak.
    expect(screen.queryAllByText('9')).toHaveLength(0)
  })

  it('menampilkan ID operator pada kolom pertama, bukan nama', async () => {
    installDefaultFetch()
    show()

    expect((await screen.findAllByText('PICTEKNIK01')).length).toBeGreaterThan(0)
    // Nama tidak punya kolom sendiri di Pega; ia hanya muncul di form.
    expect(screen.queryByText('Contoh Kepala Teknik')).not.toBeInTheDocument()
  })

  // Petugas nonaktif IKUT tampil — sama dengan layar lama, yang memuat Status Aktif `0`
  // dan `1` berdampingan. Ini uji terpenting di berkas ini: layar sempat menyaringnya.
  it('menampilkan petugas nonaktif, bukan menyembunyikannya', async () => {
    installDefaultFetch()
    show()

    expect((await screen.findAllByText('PICTEKNIK04')).length).toBeGreaterThan(0)
    // Baris aktif dan nonaktif sama-sama ada.
    expect(screen.getAllByText('PICTEKNIK01').length).toBeGreaterThan(0)
    expect(screen.getAllByText('1').length).toBeGreaterThan(0)
    expect(screen.getAllByText('0').length).toBeGreaterThan(0)
  })

  it('menampilkan pencacah klaim apa adanya', async () => {
    installDefaultFetch()
    show()
    await screen.findAllByText('PICTEKNIK03')

    expect(screen.getByText('1463')).toBeInTheDocument()
  })

  it('menyebut entitas yang menjawab dan membawa token serta portal di header', async () => {
    installDefaultFetch()
    show()
    await screen.findAllByText('PICTEKNIK01')

    expect(screen.getByText(/Portal entitas:/)).toBeInTheDocument()

    const request = calls.find((p) => p.url === ROUTE)
    expect(request?.url).not.toContain('token')
    const header = request?.init?.headers as Record<string, string>
    expect(header['Authorization']).toBe('Bearer token-uji')
    expect(header[HEADER_PORTAL]).toBe('ASM')
  })

  it('tidak menyediakan tombol hapus', async () => {
    installDefaultFetch()
    show()
    await screen.findAllByText('PICTEKNIK01')

    expect(screen.queryByRole('button', { name: /hapus/i })).not.toBeInTheDocument()
  })
})

describe('portal', () => {
  it('menuntun memilih portal sebelum menembak server', async () => {
    installDefaultFetch()
    useSelectedPortal.getState().clear()
    show()

    expect(await screen.findByText(/Portal entitas belum dipilih/)).toBeInTheDocument()
    expect(calls.find((p) => p.url === ROUTE)).toBeUndefined()
  })
})

describe('form', () => {
  /**
   * SEMBILAN isian, dengan label dan urutan yang sama persis dengan form Pega
   * "Memperbaharui Data".
   */
  it('menampilkan sembilan isian Pega dengan label yang sama', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findAllByText('PICTEKNIK01')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))

    expect(screen.getByLabelText('Username')).toBeInTheDocument()
    // getAllBy: "Input Nama" dipakai DUA kali dengan sengaja — judul kolom pertama grid
    // dan label isian hasil pencarian di form. Keduanya memang label Pega.
    expect(screen.getAllByText('Input Nama').length).toBeGreaterThan(0)
    expect(screen.getByLabelText('Email')).toBeInTheDocument()
    expect(screen.getByLabelText('Kelompok')).toBeInTheDocument()
    expect(screen.getByLabelText('Status Aktif')).toBeInTheDocument()
    expect(screen.getByLabelText('Atasan')).toBeInTheDocument()
    expect(screen.getByLabelText('Bisnis')).toBeInTheDocument()
    expect(screen.getByLabelText('Counter Klaim <1M')).toBeInTheDocument()
    expect(screen.getByLabelText('Counter Klaim >1M')).toBeInTheDocument()
  })

  it('tidak memuat isian yang tidak ada di Pega', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findAllByText('PICTEKNIK01')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))

    expect(screen.queryByLabelText('ID Operator')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Lini Bisnis')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Grup')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Beban Kerja')).not.toBeInTheDocument()
    // Pencariannya terpicu saat isian ditinggalkan, tanpa tombol — sama dengan Pega.
    expect(screen.queryByRole('button', { name: /Cari/i })).not.toBeInTheDocument()
  })

  // Kelompok, Status Aktif, Atasan, dan Bisnis adalah DROPDOWN di Pega, bukan teks bebas.
  it('menyajikan Kelompok, Status Aktif, Atasan, dan Bisnis sebagai dropdown', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findAllByText('PICTEKNIK01')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))

    for (const label of ['Kelompok', 'Status Aktif', 'Atasan', 'Bisnis']) {
      expect(screen.getByLabelText(label).tagName).toBe('SELECT')
    }

    /*
      Pilihan Kelompok dan Bisnis berasal dari Property rule Pega, BUKAN dari data yang
      kebetulan tampil:

        Property/TEAM_GROUP_property.xml     A · B · C
        Property/TYPE_BUSINESS_property.xml  NONMBU · TRAVEL · PA · BONDING

      Contoh di berkas ini hanya memuat NONMBU dan BONDING; TRAVEL dan PA tetap harus ada.
      Itulah yang membedakan daftar tetap dari daftar yang diturunkan dari isi tabel.
    */
    const groupOptions = Array.from(screen.getByLabelText('Kelompok').querySelectorAll('option'))
    expect(groupOptions.map((o) => o.value)).toEqual(['', 'A', 'B', 'C'])

    const businessOptions = Array.from(screen.getByLabelText('Bisnis').querySelectorAll('option'))
    expect(businessOptions.map((o) => o.value)).toEqual(['', 'NONMBU', 'TRAVEL', 'PA', 'BONDING'])

    expect(screen.getByRole('option', { name: 'Ya' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Tidak' })).toBeInTheDocument()
  })

  // ATASAN tidak punya daftar nilai di Property rule — ia Text biasa, dan yang
  // membatasinya di Pega adalah autocomplete atas daftar operator.
  it('mengisi dropdown Atasan dari daftar operator, bukan daftar tetap', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findAllByText('PICTEKNIK01')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))

    const options = Array.from(screen.getByLabelText('Atasan').querySelectorAll('option'))
    expect(options.map((o) => o.value)).toEqual([
      '',
      'PICTEKNIK01',
      'PICTEKNIK03',
      'PICTEKNIK04',
    ])
  })

  it('mencari direktori saat Username ditinggalkan, lalu mengisi nama dan atasan', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findAllByText('PICTEKNIK01')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))
    await user.type(screen.getByLabelText('Username'), 'PICTEKNIK05')
    await user.tab()

    expect(await screen.findByText('Contoh Petugas Baru')).toBeInTheDocument()
    await waitFor(() => {
      expect(screen.getByLabelText('Atasan')).toHaveValue('PICTEKNIK01')
    })
    expect(screen.getByText(/Menurut direktori:/)).toBeInTheDocument()
  })

  it('tidak pernah mengirim nama, grup panel, atau beban kerja ke server', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findAllByText('PICTEKNIK01')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))
    await user.type(screen.getByLabelText('Username'), 'PICTEKNIK05')
    await user.tab()
    await screen.findByText('Contoh Petugas Baru')

    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))

    await waitFor(() => {
      expect(calls.find((p) => p.url === ROUTE && p.init?.method === 'POST')).toBeDefined()
    })

    const sent = body(calls.find((p) => p.url === ROUTE && p.init?.method === 'POST')?.init)
    expect(sent).not.toHaveProperty('nama')
    expect(sent).not.toHaveProperty('grup_panel')
    expect(sent).not.toHaveProperty('beban_kerja')
    expect(sent.id_operator).toBe('PICTEKNIK05')
    expect(sent.counter_klaim_kurang_1m).toBe(0)
  })

  // Email kosong DITERIMA. Section Pega tidak memuat satu pun `pyRequired=true`, dan
  // kolomnya NULLABLE — mewajibkannya membuat baris lama yang surelnya kosong tidak
  // dapat disunting sama sekali.
  it('menerima email kosong dan tetap mengirim ke server', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findAllByText('PICTEKNIK01')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))
    await user.type(screen.getByLabelText('Username'), 'PICTEKNIK05')
    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))

    await waitFor(() => expect(calls.find((p) => p.init?.method === 'POST')).toBeDefined())
    expect(screen.queryByText('Email wajib diisi.')).not.toBeInTheDocument()
  })

  // Format email yang salah tetap ditolak — diuji di TechnicianPage.more.test.tsx
  // ("menolak isian kosong dan format email yang salah"), pada form yang Username-nya
  // masih kosong sehingga pencarian direktori tidak ikut mengisi ulang kolom surel.

  it('mengatakan mengulang tidak menolong saat layanan direktori belum terdaftar', async () => {
    installFetch((url) => {
      if (url.startsWith(DIRECTORY)) {
        return jsonResponse(503, {
          kode: 'direktori_pegawai_belum_terdaftar',
          pesan: 'Layanan pencarian pegawai belum terdaftar untuk entitas ini.',
        })
      }
      return jsonResponse(200, { pic_teknik: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
    })
    const user = userEvent.setup()
    show()
    await screen.findAllByText('PICTEKNIK01')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))
    await user.type(screen.getByLabelText('Username'), 'PICTEKNIK05')
    await user.tab()

    expect(await screen.findByText(/Layanan pencarian pegawai belum terdaftar/)).toBeInTheDocument()
    expect(screen.getByText(/Mengulang tidak akan menolong/)).toBeInTheDocument()
  })

  // ID yang tidak ketemu adalah KETERANGAN, bukan penolakan: server menerimanya, sama
  // seperti Pega yang hanya mengisi MCL_NAME apa adanya. Menandai isian merah membuat
  // form tampak tidak dapat disimpan padahal bisa.
  it('menerangkan tanpa menandai salah saat pegawai tidak terdaftar di direktori', async () => {
    installFetch((url) => {
      if (url.startsWith(DIRECTORY)) {
        return jsonResponse(422, {
          kode: 'validasi_gagal',
          pesan: 'Isian belum benar.',
          detail: [
            { field: 'id_operator', pesan: 'ID operator tidak terdaftar di direktori pegawai.' },
          ],
        })
      }
      return jsonResponse(200, { pic_teknik: SAMPLE, total: SAMPLE.length, portal: 'ASM' })
    })
    const user = userEvent.setup()
    show()
    await screen.findAllByText('PICTEKNIK01')

    await user.click(screen.getByRole('button', { name: /Tambah/i }))
    await user.type(screen.getByLabelText('Username'), 'TIDAKTERDAFTAR')
    await user.tab()

    expect(await screen.findByText(/tidak ketemu di master maupun direktori/i)).toBeInTheDocument()
    expect(screen.getByLabelText('Username')).not.toHaveAttribute('aria-invalid', 'true')
    expect(screen.queryByText('Pencarian gagal')).not.toBeInTheDocument()
  })
})

describe('mengubah', () => {
  it('memakai judul Pega dan mengunci Username', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findAllByText('PICTEKNIK01')

    await user.click(screen.getAllByRole('button', { name: /^Ubah PIC teknik/i })[0]!)

    expect(screen.getByText('Memperbaharui Data')).toBeInTheDocument()
    expect(screen.queryByLabelText('Username')).not.toBeInTheDocument()
    expect(screen.getByText(/Tidak dapat diubah/)).toBeInTheDocument()
  })

  it('mengirim perubahan lewat PUT dengan nama isian Pega', async () => {
    installDefaultFetch()
    const user = userEvent.setup()
    show()
    await screen.findAllByText('PICTEKNIK01')

    await user.click(screen.getAllByRole('button', { name: /^Ubah PIC teknik/i })[0]!)
    await user.selectOptions(screen.getByLabelText('Kelompok'), 'C')
    await user.click(screen.getByRole('button', { name: /^Simpan$/ }))

    await waitFor(() => {
      expect(calls.find((p) => p.init?.method === 'PUT')).toBeDefined()
    })

    const request = calls.find((p) => p.init?.method === 'PUT')
    expect(request?.url).toBe(`${ROUTE}/PICTEKNIK01`)
    expect(body(request?.init).kelompok).toBe('C')
  })
})
