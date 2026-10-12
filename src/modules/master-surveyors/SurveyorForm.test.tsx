import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { SurveyorType } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { SurveyorForm } from './SurveyorForm'

/**
 * Isian formulir BERBEDA menurut tipe surveyor, dan uji ini yang membuktikannya.
 *
 * Aturannya dibaca dari `Section/BrowseDetailSuveryorsApprove-Section.xml` — tiap kontrol
 * menyimpan `pyCondition` sendiri terhadap When rule `PNCIsInternalSurveyors`:
 *
 *	selalu          M_SURVEY_ID, NAME, EMAIL, TELEPHONE, ADDRESS
 *	non-internal    TRFKOMITE, FAKSIMILE, STATE, KDPOS, OTHER_CONTACT
 *	internal saja   BRANCHNAME
 *	TIDAK PERNAH    LOGIN_APLIKASI   (pyCondition = 1=2)
 *
 * Tanpa uji ini, aturan tampil hanya pernyataan di komentar — dan pernah sekali tidak
 * benar-benar berlaku di layar.
 *
 * DUA kontrol untuk satu kolom juga diuji di sini: NAME punya kotak teks untuk tipe
 * non-internal DAN daftar pilihan pegawai untuk internal. Keduanya ada di section yang
 * sama, masing-masing dengan syaratnya sendiri.
 */

/** Keempat tipe nyata portal ASM. Kodenya tidak dikarang. */
const TYPES: SurveyorType[] = [
  { kode: '1001', deskripsi: 'INTERNAL SURVEYOR', kode_lama: '' },
  { kode: '1002', deskripsi: 'LOSS ADJUSTER', kode_lama: '' },
  { kode: '1003', deskripsi: 'EXPERT', kode_lama: '' },
  { kode: '1004', deskripsi: 'SURVEY AGENT', kode_lama: '' },
]

/** Isian yang hanya ada pada tipe NON-internal. */
const NON_INTERNAL_ONLY = [
  'Apakah perlu ke direksi?',
  'Fax',
  'Negara',
  'Kode Pos',
  'Nama PIC',
]

/**
 * Isian yang selalu ada, apa pun tipenya.
 *
 * Labelnya "Nama", bukan "Input Nama": yang kedua tidak ada di mana pun pada export —
 * nol kemunculan di keempat section tab. Ia pernah ditulis begitu, dan itu karangan.
 */
const ALWAYS = ['Tipe Surveyor', 'Nama', 'Email', 'Telp', 'Alamat']

/** Contoh pegawai yang dijawab GET /api/master/surveyor/pegawai saat diuji. */
const EMPLOYEES = [
  { nama: 'CONTOH PEGAWAI SATU', login_aplikasi: 'CONTOHPEGAWAI1', email: 'satu@contoh.invalid' },
  { nama: 'CONTOH PEGAWAI DUA', login_aplikasi: 'CONTOHPEGAWAI2', email: 'dua@contoh.invalid' },
]

/** Contoh cabang yang dijawab GET /api/master/surveyor/cabang saat diuji. */
const BRANCHES = [{ kode: '001', nama: 'KANTOR PUSAT' }]

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <SurveyorForm surveyor={null} surveyorTypes={TYPES} onClose={() => undefined} />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

/** Memilih tipe pada dropdown, lalu mengembalikan kendali setelah layar menyesuaikan. */
async function chooseType(label: string) {
  const user = userEvent.setup()
  await user.selectOptions(screen.getByLabelText('Tipe Surveyor'), label)
}

beforeEach(() => {
  // Tiga daftar acuan ditembak formulir ini: negara, pegawai, dan cabang. Jawabannya
  // dibedakan menurut jalur, karena isian Nama untuk surveyor internal benar-benar
  // BERGANTUNG pada isi daftar pegawainya — bukan sekadar ada-tidaknya kontrol.
  vi.stubGlobal('fetch', (input: RequestInfo | URL) => {
    const url = String(typeof input === 'string' ? input : input instanceof URL ? input : input.url)
    const body = url.includes('/pegawai')
      ? { pegawai: EMPLOYEES, portal: 'ASM' }
      : url.includes('/cabang')
        ? { cabang: BRANCHES, portal: 'ASM' }
        : { negara: [], portal: 'ASM' }
    return Promise.resolve(
      new Response(JSON.stringify(body), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
  useSession.setState({ token: 'token-uji' })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('isian formulir menurut tipe surveyor', () => {
  it('sebelum tipe dipilih, HANYA dropdown Tipe Surveyor yang tampil', () => {
    show()

    expect(screen.getByLabelText('Tipe Surveyor')).toBeInTheDocument()

    // Keempat isian "selalu ada" pun belum muncul: di layar lama, dropdown tipe memicu
    // refresh section, dan sebelum itu tidak ada kondisi yang dapat dinilai.
    for (const label of ['Nama', 'Email', 'Telp', 'Alamat']) {
      expect(screen.queryByLabelText(new RegExp(`^${label}$`, 'i'))).not.toBeInTheDocument()
    }
    for (const label of NON_INTERNAL_ONLY) {
      expect(screen.queryByLabelText(new RegExp(`^${label}`, 'i'))).not.toBeInTheDocument()
    }
    expect(screen.queryByLabelText(/^Cabang/i)).not.toBeInTheDocument()
  })

  it('tombol Simpan mati sebelum tipe dipilih, hidup sesudahnya', async () => {
    show()
    expect(screen.getByRole('button', { name: /simpan/i })).toBeDisabled()

    await chooseType('LOSS ADJUSTER')
    expect(screen.getByRole('button', { name: /simpan/i })).toBeEnabled()
  })

  it('menampilkan lima isian yang selalu ada, apa pun tipenya', async () => {
    show()
    await chooseType('LOSS ADJUSTER')
    for (const label of ALWAYS) {
      // Dicocokkan PERSIS, bukan dengan awalan: "Nama" dan "Nama PIC" adalah dua isian
      // berbeda, dan pencocokan berawalan membuat yang pertama menangkap keduanya.
      expect(screen.getByLabelText(new RegExp(`^${label}$`, 'i'))).toBeInTheDocument()
    }
  })

  it('Internal Surveyor: Cabang muncul, lima isian non-internal disembunyikan', async () => {
    show()
    await chooseType('INTERNAL SURVEYOR')

    expect(screen.getByLabelText(/^Cabang/i)).toBeInTheDocument()
    for (const label of NON_INTERNAL_ONLY) {
      expect(screen.queryByLabelText(new RegExp(`^${label}`, 'i'))).not.toBeInTheDocument()
    }
  })

  it.each([
    ['LOSS ADJUSTER'],
    ['EXPERT'],
    ['SURVEY AGENT'],
  ])('%s: lima isian non-internal muncul, Cabang disembunyikan', async (label) => {
    show()
    await chooseType(label)

    for (const field of NON_INTERNAL_ONLY) {
      expect(screen.getByLabelText(new RegExp(`^${field}`, 'i'))).toBeInTheDocument()
    }
    expect(screen.queryByLabelText(/^Cabang/i)).not.toBeInTheDocument()
  })

  it('Internal Surveyor: Nama adalah DAFTAR PILIHAN pegawai, bukan kotak teks', async () => {
    show()
    await chooseType('INTERNAL SURVEYOR')

    const nama = await screen.findByLabelText(/^Nama$/i)
    expect(nama.tagName).toBe('SELECT')

    // Isinya datang dari server, bukan dikarang di layar.
    await screen.findByRole('option', { name: 'CONTOH PEGAWAI SATU' })
    await screen.findByRole('option', { name: 'CONTOH PEGAWAI DUA' })
  })

  it('tipe NON-internal: Nama tetap kotak teks', async () => {
    show()
    await chooseType('LOSS ADJUSTER')
    expect(screen.getByLabelText(/^Nama$/i).tagName).toBe('INPUT')
  })

  it('memilih pegawai ikut mengisi Email — cerminan pyAdditionalFields', async () => {
    const user = userEvent.setup()
    show()
    await chooseType('INTERNAL SURVEYOR')

    await screen.findByRole('option', { name: 'CONTOH PEGAWAI SATU' })
    expect(screen.getByLabelText(/^Email/i)).toHaveValue('')

    await user.selectOptions(screen.getByLabelText(/^Nama$/i), 'CONTOH PEGAWAI SATU')
    expect(screen.getByLabelText(/^Email/i)).toHaveValue('satu@contoh.invalid')
  })

  it('Internal Surveyor: Cabang adalah daftar pilihan, bukan kotak teks', async () => {
    show()
    await chooseType('INTERNAL SURVEYOR')

    const cabang = await screen.findByLabelText(/^Cabang/i)
    expect(cabang.tagName).toBe('SELECT')
    await screen.findByRole('option', { name: 'KANTOR PUSAT' })
  })

  it('Login Aplikasi TIDAK PERNAH muncul, pada tipe mana pun', async () => {
    show()
    for (const label of ['INTERNAL SURVEYOR', 'LOSS ADJUSTER', 'EXPERT', 'SURVEY AGENT']) {
      await chooseType(label)
      expect(screen.queryByLabelText(/login aplikasi/i)).not.toBeInTheDocument()
    }
  })
})
