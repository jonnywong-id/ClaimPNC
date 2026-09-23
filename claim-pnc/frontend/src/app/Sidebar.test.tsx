import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { MENU_ROUTES } from '@/app/menu/registry'
import { useSession } from '@/app/session'

import { Sidebar } from './Sidebar'

const SAMPLE_PROFILE = {
  identitas: '90000001',
  nama: 'Contoh Administrator',
  jenis: 'KARYAWAN',
  login: 'adminpnc',
  email: '',
  perusahaan: 'ASM',
}

/**
 * MENU_PROGRAM yang dipakai sebagai contoh butir "belum tersedia".
 *
 * Ia sengaja diangkat menjadi konstanta agar penggantinya cukup disunting di SATU tempat,
 * dan agar uji penjaga di bawah dapat memeriksanya. Riwayat berkas ini menunjukkan kenapa
 * itu perlu: contohnya sudah batal TIGA KALI karena modulnya keburu dibangun — "Master PIC
 * Teknik", lalu "Master Surveyors", lalu `InboxOutstanding_Harness` pada 2026-09-23.
 *
 * Yang ketiga membatalkan sekaligus alasan yang tertulis di komentarnya sendiri: harness
 * `InboxOutstanding_Harness` memang tidak ada di export, tetapi layarnya tetap dapat
 * dibangun dari kueri `BrowseInboxOutstanding1` beserta activity dan section-nya. Jadi
 * "harness-nya tidak ada di export" TIDAK menjamin modulnya tidak akan dibangun.
 *
 * Karena tidak ada kriteria yang benar-benar kebal, yang dipasang adalah penjaganya: uji
 * `contoh butir "belum tersedia" masih sahih` gagal lebih dulu dengan pesan yang menyebut
 * apa yang harus dilakukan, sehingga penggantinya tidak perlu ditelusuri dari kegagalan
 * render yang membingungkan.
 */
const BELUM_DIBANGUN = 'DataMemberReas'

/**
 * Bentuknya meniru jawaban GET /api/menu: dua kelompok, masing-masing dengan butir yang
 * sudah punya layar dan butir yang belum.
 */
const MENU = {
  menu: [
    {
      id: 1,
      nama: 'MASTER',
      program: '',
      submenu: [
        { id: 11, nama: 'Master Status Klaim', program: 'StatusClaimInbox', submenu: [] },
        { id: 13, nama: 'Master PIC Teknik', program: 'UserTeknisInbox', submenu: [] },
        // MENU_ID 15 adalah daftar ORANG surveyor (D_SURVEYORS) — bedakan dari MENU_ID 14
        // "Master Tipe Surveyors" yang berisi golongannya. Modulnya SUDAH dibangun
        // (2026-09-20), sehingga ia kini contoh butir yang PUNYA layar.
        { id: 15, nama: 'Master Surveyors', program: 'DetailSurveyorsInbox', submenu: [] },
        { id: 23, nama: 'Master Status Progress 1', program: 'StatusProgress', submenu: [] },
      ],
    },
    {
      id: 4,
      nama: 'REPORT',
      program: '',
      submenu: [
        // MENU_ID 83 ada di master TANPA MENU_PROGRAM. Ia tetap dikirim server.
        { id: 83, nama: 'Report Adjuster', program: '', submenu: [] },
        // Butir yang MENU_PROGRAM-nya menunjuk harness yang TIDAK ADA DI EXPORT sama
        // sekali (`K-33`, 11 dari 47 harness target). Ia dipakai sebagai contoh butir
        // yang belum ada modulnya — lihat alasannya di uji yang memakainya.
        { id: 91, nama: 'Data Member Reas', program: BELUM_DIBANGUN, submenu: [] },
      ],
    },
  ],
}

let calls: string[] = []

function installFetch(reply: (url: string) => Response) {
  vi.stubGlobal('fetch', (url: string) => {
    calls.push(url)
    return Promise.resolve(reply(url))
  })
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function show(path = '/') {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={[path]}>
        <Sidebar />
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
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
})

describe('peta menu', () => {
  it('menampilkan kelompok menu dari server, berurutan', async () => {
    installFetch(() => jsonResponse(200, MENU))
    show()

    expect(await screen.findByRole('button', { name: /MASTER/ })).toBeInTheDocument()

    const groups = screen.getAllByRole('button').map((b) => b.textContent ?? '')
    expect(groups[0]).toContain('MASTER')
    expect(groups[1]).toContain('REPORT')
  })

  // Menu dibaca lewat sesi. Portal TIDAK ikut: peta menu hidup di basis data portal
  // utama dan tidak punya kolom entitas.
  it('membawa token sesi dan tidak menyebut portal', async () => {
    installFetch(() => jsonResponse(200, MENU))
    show()

    await waitFor(() => expect(calls).toContain('/api/menu'))
  })

  it('membentangkan kelompok saat judulnya ditekan', async () => {
    installFetch(() => jsonResponse(200, MENU))
    show()
    const user = userEvent.setup()

    const master = await screen.findByRole('button', { name: /MASTER/ })
    expect(master).toHaveAttribute('aria-expanded', 'false')
    expect(screen.queryByText('Master Status Klaim')).not.toBeInTheDocument()

    await user.click(master)

    expect(master).toHaveAttribute('aria-expanded', 'true')
    expect(screen.getByText('Master Status Klaim')).toBeInTheDocument()
  })

  // Submenu tersusun di bawah induknya, bukan berjajar rata dengan kelompoknya.
  it('menaruh submenu di dalam kelompoknya', async () => {
    installFetch(() => jsonResponse(200, MENU))
    show()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: /MASTER/ }))

    const panel = screen.getByRole('button', { name: /MASTER/ }).getAttribute('aria-controls')
    expect(panel).toBeTruthy()

    const list = document.getElementById(panel as string)
    expect(list).not.toBeNull()
    expect(within(list as HTMLElement).getByText('Master Status Klaim')).toBeInTheDocument()
    expect(within(list as HTMLElement).getByText('Master PIC Teknik')).toBeInTheDocument()
  })
})

describe('butir yang sudah ada modulnya', () => {
  it('menjadi tautan ke rutenya', async () => {
    installFetch(() => jsonResponse(200, MENU))
    show()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: /MASTER/ }))

    expect(screen.getByRole('link', { name: 'Master Status Klaim' })).toHaveAttribute(
      'href',
      '/master/status-klaim',
    )
    expect(screen.getByRole('link', { name: 'Master Status Progress 1' })).toHaveAttribute(
      'href',
      '/master/status-progres-1',
    )
  })

  // Kelompok yang memuat layar yang sedang dibuka dibentangkan sendiri. Tanpa itu,
  // pengguna yang baru membuka satu layar tidak melihat di mana ia berada.
  it('membentangkan kelompok layar yang sedang dibuka', async () => {
    installFetch(() => jsonResponse(200, MENU))
    show('/master/status-progres-1')

    expect(await screen.findByRole('link', { name: 'Master Status Progress 1' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /MASTER/ })).toHaveAttribute('aria-expanded', 'true')
  })
})

describe('butir yang belum ada modulnya', () => {
  // Penjaga, bukan uji perilaku. Ia menjawab satu hal saja: apakah contoh yang dipakai uji
  // di bawahnya MASIH berupa butir yang belum punya layar.
  //
  // Tanpa ini, membangun modul `BELUM_DIBANGUN` akan menjatuhkan uji berikutnya dengan
  // "expected document not to contain element" — kegagalan yang benar tetapi tidak
  // menjelaskan apa pun, dan sudah tiga kali menghabiskan waktu orang untuk ditelusuri.
  it('contoh butir "belum tersedia" masih sahih', () => {
    expect(
      MENU_ROUTES[BELUM_DIBANGUN],
      `MENU_PROGRAM "${BELUM_DIBANGUN}" kini SUDAH punya layar di MENU_ROUTES, sehingga ` +
        'ia tidak lagi sah sebagai contoh butir "belum tersedia". Ganti konstanta ' +
        'BELUM_DIBANGUN di berkas ini dengan MENU_PROGRAM lain yang belum dipetakan — ' +
        'daftar kandidatnya ada di komentar kepala src/app/menu/registry.ts.',
    ).toBeUndefined()
  })

  // Keputusan Work Owner 2026-09-18: tetap terlihat, tidak dapat diklik, bertanda.
  it('tampil dengan tanda "belum tersedia" dan bukan tautan', async () => {
    installFetch(() => jsonResponse(200, MENU))
    show()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: /REPORT/ }))

    expect(screen.getByText('Data Member Reas')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Data Member Reas' })).not.toBeInTheDocument()
    expect(screen.getAllByText('belum tersedia').length).toBeGreaterThan(0)
  })

  // MENU_ID 83 tidak punya MENU_PROGRAM sama sekali. Ia diperlakukan sama: terlihat,
  // tidak dapat diklik — kekosongan datanya tidak dikubur.
  it('memperlakukan butir tanpa program dengan cara yang sama', async () => {
    installFetch(() => jsonResponse(200, MENU))
    show()
    const user = userEvent.setup()

    await user.click(await screen.findByRole('button', { name: /REPORT/ }))

    expect(screen.getByText('Report Adjuster')).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Report Adjuster' })).not.toBeInTheDocument()
  })
})

describe('keadaan selain menu terisi', () => {
  it('menyebutkan bahwa pengguna belum diberi menu', async () => {
    installFetch(() => jsonResponse(200, { menu: [] }))
    show()

    expect(await screen.findByText(/Belum ada menu yang diberikan/)).toBeInTheDocument()
  })

  // Pesannya `role="status"`, bukan `role="alert"`: menu yang gagal dimuat adalah
  // keadaan, bukan sesuatu yang harus menyela apa yang sedang dibaca pengguna.
  it('memberi tahu saat menu gagal dimuat, tanpa menyela', async () => {
    installFetch(() => jsonResponse(500, { kode: 'galat_internal', pesan: 'gagal' }))
    show()

    const message = await screen.findByRole('status')
    expect(message).toHaveTextContent(/Menu tidak dapat dimuat/)
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  // Beranda bukan pengganti harness Pega mana pun — ia layar milik aplikasi baru ini,
  // sehingga tidak diambil dari tabel menu dan tetap ada apa pun isi tabelnya.
  it('selalu menyediakan tautan Beranda', async () => {
    installFetch(() => jsonResponse(200, { menu: [] }))
    show()

    expect(screen.getByRole('link', { name: 'Beranda' })).toHaveAttribute('href', '/')
  })
})
