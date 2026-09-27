import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { deriveLogin } from './SurveyorLoginForm'
import { SurveyorLoginPage } from './SurveyorLoginPage'

/**
 * Seorang leader — baris yang LOGINLEADER-nya kosong.
 *
 * Nama, surel, telepon, dan alamat di berkas ini KARANGAN, dan domainnya `contoh.invalid`
 * yang RFC 2606 cadangkan supaya tidak pernah dapat diselesaikan DNS (`D-69`).
 */
const LEADER = {
  nama: 'Budi Hartono',
  login: 'BudiHartono',
  email: 'budi.hartono@contoh.invalid',
  telp: '021-5550101',
  alamat: 'Jl. Melati Raya No. 12, Jakarta Selatan',
  status_login: 'Member',
  login_leader: '',
}

/**
 * Seorang anggota — baris yang LOGINLEADER-nya menunjuk LEADER.
 *
 * Namanya memuat DUA spasi, sehingga login yang diturunkan darinya tidak sama dengan
 * namanya. Tanpa itu, penurunan LOGIN tampak seperti sekadar menyalin Nama.
 */
const MEMBER = {
  nama: 'Rina Ayu Lestari',
  login: 'RinaAyuLestari',
  email: 'rina.lestari@contoh.invalid',
  telp: '021-5550102',
  alamat: 'Jl. Kenanga No. 7, Jakarta Pusat',
  status_login: 'Member',
  login_leader: 'BudiHartono',
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

/** defaultReply melayani daftar; mutasi dijawab pemanggil. */
function defaultReply(mutation?: (call: Call) => Reply) {
  return (call: Call): Reply => {
    if (call.method === 'GET') {
      return { body: { login_surveyor: [LEADER, MEMBER], portal: 'ASM' } }
    }
    if (mutation) return mutation(call)
    return { body: { login_surveyor: MEMBER, portal: 'ASM' }, status: 200 }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <SurveyorLoginPage />
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

/** mutationCalls mengambil permintaan yang mengubah data saja. */
function mutationCalls() {
  return calls.filter((c) => c.method === 'POST' || c.method === 'PUT')
}

/** openEdit membuka form penyuntingan pada baris yang namanya disebut. */
async function openEdit(user: ReturnType<typeof userEvent.setup>, nama: string) {
  const table = await screen.findByRole('table')
  const row = within(table).getByRole('row', { name: new RegExp(nama) })
  await user.click(within(row).getByRole('button', { name: 'Ubah' }))
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

describe('penurunan Login dari Nama', () => {
  /*
    Ia diuji SENDIRI, terpisah dari layarnya, karena ia harus sama persis dengan
    masterlogin.DeriveLogin di backend — dan kasusnya sama persis dengan yang diuji di sana.

    Bila keduanya berbeda sedikit saja, pengguna melihat login yang bukan yang tersimpan.
  */
  it.each([
    ['Budi Hartono', 'BudiHartono'],
    ['Rina  Ayu', 'RinaAyu'],
    ['Budi.Hartono', 'BudiHartono'],
    ['Hartono, Budi', 'HartonoBudi'],
    ['Siti Nur-Halimah', 'SitiNurHalimah'],
    ['A. B, C-D E', 'ABCDE'],
    ['bUdI hArToNo', 'bUdIhArToNo'],
    ['  Budi  ', 'Budi'],
    ['Budi_Hartono', 'Budi_Hartono'],
    ["O'Brien", "O'Brien"],
    ['- . , ', ''],
  ])('deriveLogin(%j) = %j', (nama, mau) => {
    expect(deriveLogin(nama)).toBe(mau)
  })
})

describe('daftar master login', () => {
  // Kelima kolom dibaca dari `pyLabelFieldValue` pada
  // Section/BrowseLoginSurveyor-Section.xml.
  it('menampilkan judul dan kelima kolom grid Pega', async () => {
    installFetch(defaultReply())
    show()

    expect(
      screen.getByRole('heading', { name: 'Master Login Surveyor' }),
    ).toBeInTheDocument()

    const table = await screen.findByRole('table')
    for (const header of ['Nama', 'Login', 'Email', 'Telp', 'Alamat']) {
      expect(within(table).getByRole('columnheader', { name: header })).toBeInTheDocument()
    }
  })

  // Keduanya TIDAK ada di grid Pega; keduanya hanya muncul saat sebuah baris dibuka.
  it('tidak menggambar kolom Status Login maupun Login Leader di grid', async () => {
    installFetch(defaultReply())
    show()

    const table = await screen.findByRole('table')
    for (const header of ['Status Login', 'Login Leader']) {
      expect(
        within(table).queryByRole('columnheader', { name: header }),
      ).not.toBeInTheDocument()
    }
  })

  it('menampilkan baris yang dijawab server', async () => {
    installFetch(defaultReply())
    show()

    expect(await screen.findByText('Budi Hartono')).toBeInTheDocument()
    expect(screen.getByText('RinaAyuLestari')).toBeInTheDocument()
  })

  // Portal ikut di setiap permintaan; tanpa itu server menolaknya (TKT-F6-002), dan cache
  // dapat menampilkan data entitas lain (R-20).
  it('menyertakan portal aktif pada permintaan daftar', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('Budi Hartono')
    expect(calls[0]?.header['X-Portal']).toBe('ASM')
  })

  // Tabelnya tidak punya kolom APPROVAL, sehingga layar ini memang tidak bertab — berbeda
  // dari seluruh master di rumpun sparepart.
  it('tidak menggambar tab apa pun', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('Budi Hartono')
    for (const caption of ['Approve', 'Reject', 'Waiting Approval']) {
      expect(screen.queryByRole('button', { name: caption })).not.toBeInTheDocument()
    }
  })

  // Keduanya keterbatasan nyata yang tidak terlihat dari layar bila tidak disebutkan.
  it('menyatakan akun belum diterbitkan dan login tidak dapat dihapus', async () => {
    installFetch(defaultReply())
    show()

    await screen.findByText('Budi Hartono')
    expect(screen.getByText(/Akun aplikasi belum diterbitkan/)).toBeInTheDocument()
    expect(screen.getByText(/Login tidak dapat dihapus/)).toBeInTheDocument()
  })
})

describe('menambah login surveyor', () => {
  // Badan permintaan memuat EMPAT isian saja. Ketiga kolom turunan tidak boleh ikut —
  // server menolaknya sebagai permintaan cacat.
  it('mengirim empat isian saja, tanpa kolom turunan', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Budi Hartono')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    await user.type(screen.getByLabelText('Nama'), 'Sari Dewi-Anggraini')
    await user.type(screen.getByLabelText('Email'), 'sari@contoh.invalid')
    await user.type(screen.getByLabelText('Telp'), '021-777')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(mutationCalls()).toHaveLength(1))

    const sent = mutationCalls()[0]
    expect(sent?.method).toBe('POST')
    expect(sent?.body).toEqual({
      nama: 'Sari Dewi-Anggraini',
      email: 'sari@contoh.invalid',
      telp: '021-777',
      alamat: '',
    })
  })

  // Login diperlihatkan sementara pengguna mengetik, meniru aksi `refresh` bereven `change`
  // pada kontrol Nama di layar lama.
  it('memperlihatkan Login yang akan terbentuk saat Nama diketik', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Budi Hartono')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    const form = screen.getByRole('form', { name: 'Tambah Login Surveyor' })
    expect(within(form).getByText(/Terisi otomatis setelah Nama diketik/)).toBeInTheDocument()

    await user.type(screen.getByLabelText('Nama'), 'Sari Dewi-Anggraini')
    expect(within(form).getByText('SariDewiAnggraini')).toBeInTheDocument()
  })

  // Ketiganya wajib di layar Pega (`pyRequired = true`); Alamat tidak.
  it('menolak Nama, Email, dan Telp yang kosong tanpa menembak server', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Budi Hartono')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Nama wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Email wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Telp wajib diisi.')).toBeInTheDocument()
    expect(mutationCalls()).toHaveLength(0)
  })

  // Nama yang seluruhnya terdiri atas karakter yang dibuang menghasilkan Login kosong —
  // padanan langkah kedua CNMInsertMstLoginSurveyor_act.
  it('menolak Nama yang menghasilkan Login kosong', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await screen.findByText('Budi Hartono')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    await user.type(screen.getByLabelText('Nama'), '- . , ')
    await user.type(screen.getByLabelText('Email'), 'x@contoh.invalid')
    await user.type(screen.getByLabelText('Telp'), '021')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(
      await screen.findByText(/Nama harus memuat setidaknya satu huruf atau angka/),
    ).toBeInTheDocument()
    expect(mutationCalls()).toHaveLength(0)
  })

  // Penolakan login ganda adalah kegagalan yang paling sering terjadi di layar ini;
  // membuang isian pengguna saat itu terjadi akan membuatnya mengetik ulang semuanya.
  it('mempertahankan form saat server menolak login ganda', async () => {
    const user = userEvent.setup()
    installFetch(
      defaultReply(() => ({
        status: 409,
        body: {
          kode: 'kunci_login_surveyor_sudah_ada',
          pesan: 'Login sudah terdaftar dengan nama yang sama',
        },
      })),
    )
    show()

    await screen.findByText('Budi Hartono')
    await user.click(screen.getByRole('button', { name: 'Tambah' }))

    await user.type(screen.getByLabelText('Nama'), 'Budi.Hartono')
    await user.type(screen.getByLabelText('Email'), 'x@contoh.invalid')
    await user.type(screen.getByLabelText('Telp'), '021')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Login itu sudah terdaftar')).toBeInTheDocument()
    expect(screen.getByLabelText('Nama')).toHaveValue('Budi.Hartono')
  })
})

describe('mengubah login surveyor', () => {
  // Dibaca dari `pyDisabledWhen = TempLoginSurvey.pyLabel='Update'` pada kontrol Nama.
  it('mengunci isian Nama', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await openEdit(user, 'Rina Ayu Lestari')
    expect(screen.getByLabelText('Nama')).toBeDisabled()
  })

  // Keduanya tidak pernah digambar layar Pega; ditampilkan di sini karena nilainya
  // menentukan peran dan tim seseorang.
  it('menampilkan Status Login dan Login Leader sebagai keterangan baca-saja', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await openEdit(user, 'Rina Ayu Lestari')

    const form = screen.getByRole('form', { name: 'Ubah Login Surveyor' })
    expect(within(form).getByText('Status Login')).toBeInTheDocument()
    expect(within(form).getByText('Member')).toBeInTheDocument()
    expect(within(form).getByText('Login Leader')).toBeInTheDocument()
    expect(within(form).getByText('BudiHartono')).toBeInTheDocument()
  })

  // Login adalah kunci baris, dan jalurnya memakai LOGIN — bukan sebuah ID terpisah.
  it('menyimpan ke jalur berkunci LOGIN', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await openEdit(user, 'Rina Ayu Lestari')

    const telp = screen.getByLabelText('Telp')
    await user.clear(telp)
    await user.type(telp, '0899-1')
    await user.click(screen.getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(mutationCalls()).toHaveLength(1))

    const sent = mutationCalls()[0]
    expect(sent?.method).toBe('PUT')
    expect(sent?.url).toBe('/api/master/login/RinaAyuLestari')
  })

  // Baris yang LOGINLEADER-nya kosong benar-benar mungkin, dan tidak dapat dibedakan dari
  // "gagal menemukan timnya" bila tidak dinyatakan.
  it('menyatakan baris yang tidak bertaut ke tim mana pun', async () => {
    const user = userEvent.setup()
    installFetch(defaultReply())
    show()

    await openEdit(user, 'Budi Hartono')

    const form = screen.getByRole('form', { name: 'Ubah Login Surveyor' })
    expect(
      within(form).getByText('tidak bertaut ke tim mana pun'),
    ).toBeInTheDocument()
  })
})
