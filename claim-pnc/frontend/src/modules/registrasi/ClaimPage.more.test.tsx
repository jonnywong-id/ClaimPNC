import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ClaimPage } from './ClaimPage'

/**
 * Uji tambahan layar klaim: keadaan klaim tanpa tugas, tahap tanpa formulir, cabang galat,
 * dan isian Input Register yang tidak tersentuh uji utama. Seluruh data KARANGAN (`D-69`).
 */

const KLAIM = {
  id: 'klaim-1',
  nomor: '',
  portal: 'ASM',
  polis: {
    nomor: 'POL-FIRE-0001',
    lini: '006',
    nama_lini: 'Fire',
    jenis_bisnis: 'Fire',
    mulai_pertanggungan: '2026-01-01',
    akhir_pertanggungan: '2026-12-31',
    mata_uang: 'USD',
    nama_tertanggung: '',
    deklarasi: false,
    penjamin_kredit: false,
  },
  tanggal_kejadian: '2026-06-05',
  tanggal_lapor: '2026-06-06',
  tanggal_terima_dokumen: '2026-06-07',
  lokasi: 'Gudang A',
  kronologi: '',
  pelapor: { nama: '', telepon: '', email: '', alamat: '', hubungan: 0, hubungan_lainnya: '' },
  prinsip_mengenal_nasabah: '',
  nilai_estimasi_sen: 0,
  mata_uang: '',
  nomor_slik: '',
  ex_gratia: false,
  user_teknis: '',
  rcv_id: '',
  objek: [],
  status_pucl: 0,
  transfer_compliance: false,
  status_proses: 'BERJALAN',
  status_klaim: '',
  flag_klaim: '0',
  status_posisi_progres: 'On Progress',
  tahap_kini: 'input-register',
}

const TUGAS = {
  id: 'tugas-1',
  klaim_id: 'klaim-1',
  nomor_klaim: '',
  tahap: 'input-register',
  nama_tahap: 'Input Register',
  antrean: 'WORKLIST',
  workbasket: '',
  pemilik: '90000001',
  dapat_diambil: false,
  tindakan_keluar: 'InputRegister',
  dibuat_pada: '2026-06-10T03:00:00Z',
}

type Answer = Response | Promise<Response> | 'putus' | 'aneh' | undefined

let calls: { url: string; method: string; body: unknown }[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function response(klaim: Record<string, unknown> = {}, tugas: Record<string, unknown> | null = {}) {
  return {
    klaim: { ...KLAIM, ...klaim },
    tugas: tugas === null ? null : { ...TUGAS, ...tugas },
    jalur: [],
    jejak_keputusan: null,
    large_loss: false,
  }
}

function installFetch(claim: unknown, answer: (url: string, method: string) => Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const method = init?.method ?? 'GET'
    calls.push({ url, method, body: init?.body ? JSON.parse(init.body as string) : undefined })
    const custom = answer(url, method)
    if (custom === 'putus') return Promise.reject(new TypeError('putus'))
    if (custom === 'aneh') {
      return Promise.resolve({
        get status(): number {
          throw 'bukan galat'
        },
      } as unknown as Response)
    }
    if (custom) return Promise.resolve(custom)
    if (url === '/api/registrasi/alur') return Promise.resolve(json(200, { nama: 'Register', mulai: '', tahap: [] }))
    if (url.startsWith('/api/registrasi/wilayah/')) {
      const level = url.split('?')[0]!.split('/').pop()
      const pilihan =
        level === 'negara'
          ? [
              { id: '1', nama: 'INDONESIA' },
              { id: '2', nama: 'SINGAPURA' },
            ]
          : []
      return Promise.resolve(json(200, { pilihan }))
    }
    if (url === '/api/registrasi/klaim/klaim-1') return Promise.resolve(json(200, claim))
    return Promise.resolve(json(200, claim))
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/registrasi/klaim/klaim-1']}>
        <Routes>
          <Route path="/registrasi/klaim/:claimID" element={<ClaimPage />} />
          <Route path="/registrasi" element={<p>Halaman inbox</p>} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: { identitas: '90000001', nama: 'Contoh', jenis: 'KARYAWAN', login: 'x', email: '', perusahaan: 'ASM' },
    validUntil: '2026-12-31T00:00:00Z',
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('keadaan klaim', () => {
  it.each([
    { name: 'jaringan putus', answer: (): Answer => 'putus', text: 'Tidak dapat menghubungi server Claim PNC.' },
    { name: 'galat bukan Error', answer: (): Answer => 'aneh', text: 'Terjadi kesalahan pada sistem.' },
  ])('menampilkan galat muat klaim: $name', async ({ answer, text }) => {
    installFetch(response(), (url) => (url === '/api/registrasi/klaim/klaim-1' ? answer() : undefined))
    show()

    expect(screen.getByText('Memuat klaim…')).toBeInTheDocument()
    expect(await screen.findByText('Klaim tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText(text)).toBeInTheDocument()
  })

  it('menyebut klaim yang sudah selesai pada alur Register', async () => {
    installFetch(response({ tahap_kini: '' }, null))
    show()

    expect(await screen.findByText(/Klaim ini sudah selesai pada alur Register\./)).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Klaim belum bernomor' })).toBeInTheDocument()
    expect(screen.queryByText(/tidak ada tugas terbuka/)).not.toBeInTheDocument()
  })

  it('menyebut tahap yang tugasnya bukan milik pengguna', async () => {
    installFetch(response({ tahap_kini: 'investigator', nomor: 'PNCN.26.0005' }, null))
    show()

    expect(await screen.findByText(/Klaim berada di tahap investigator/)).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'PNCN.26.0005' })).toBeInTheDocument()
  })
})

describe('tahap tanpa formulir', () => {
  // Investigator kini memakai formulir Surveyor (InputInvestigator); Compliance belum punya formulir.
  const AT_COMPLIANCE = response(
    { tahap_kini: 'compliance' },
    { tahap: 'compliance', nama_tahap: 'Compliance', tindakan_keluar: 'InputCompliance' },
  )

  it('menutup tahap lalu menampilkan percabangan yang dilewati', async () => {
    installFetch(AT_COMPLIANCE, (url) =>
      url === '/api/registrasi/tugas/tugas-1/selesai'
        ? json(200, { ...AT_COMPLIANCE, jejak_keputusan: ['Bukan PA', 'Bukan Travel'] })
        : undefined,
    )
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Selesaikan (InputCompliance)' }))
    expect(await screen.findByText('Percabangan yang dilewati')).toBeInTheDocument()
    expect(screen.getByText('Bukan Travel')).toBeInTheDocument()
    expect(calls.find((c) => c.url.endsWith('/selesai'))?.body).toEqual({ action: 'InputCompliance', kembali: false })
  })

  it('mengirim Back dan menampilkan galat penutupan tahap', async () => {
    installFetch(AT_COMPLIANCE, (url) =>
      url === '/api/registrasi/tugas/tugas-1/selesai'
        ? json(409, { kode: 'tugas_sudah_selesai', pesan: 'Tugas sudah selesai.' })
        : undefined,
    )
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Kembali (Back)' }))
    expect(await screen.findByText('Tahap tidak dapat ditutup')).toBeInTheDocument()
    expect(screen.getByText('Tugas sudah selesai.')).toBeInTheDocument()
    expect(calls.find((c) => c.url.endsWith('/selesai'))?.body).toMatchObject({ kembali: true })
  })

  it('tidak menampilkan percabangan bila server tidak mengirimnya', async () => {
    installFetch(AT_COMPLIANCE, (url) =>
      url === '/api/registrasi/tugas/tugas-1/selesai' ? json(200, { ...AT_COMPLIANCE, jejak_keputusan: [] }) : undefined,
    )
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Selesaikan (InputCompliance)' }))
    await waitFor(() => expect(calls.some((c) => c.url.endsWith('/selesai'))).toBe(true))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Selesaikan (InputCompliance)' })).toBeEnabled())
    expect(screen.queryByText('Percabangan yang dilewati')).not.toBeInTheDocument()
  })
})

describe('Input Register', () => {
  it('memakai mata uang polis dan nilai bawaan bila klaim belum mengisinya', async () => {
    installFetch(response(), (url) =>
      url === '/api/registrasi/register/simpan' ? json(500, { kode: 'x', pesan: 'Draf gagal.' }) : undefined,
    )
    show()

    expect(await screen.findByLabelText('Mata Uang')).toHaveValue('USD')
    expect(screen.getByLabelText('Status Pelapor')).toHaveValue('')
    expect(screen.getByLabelText('Status RCL/PUCL')).toHaveValue('0')
    expect(screen.getByRole('radio', { name: 'NORMAL' })).toBeChecked()

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))
    expect(await screen.findByText('Isian belum tersimpan')).toBeInTheDocument()
    expect(screen.getByText('Draf gagal.')).toBeInTheDocument()
  })

  it('meminta keterangan hubungan lain-lain dan komentar suspicious, lalu mengirimnya', async () => {
    installFetch(response(), (url) =>
      url === '/api/registrasi/register' ? json(200, { ...response(), large_loss: true }) : undefined,
    )
    show()

    const hubungan = await screen.findByLabelText('Status Pelapor')
    expect(screen.queryByLabelText('Sebutkan...')).not.toBeInTheDocument()
    await userEvent.type(hubungan, '7')
    await userEvent.type(screen.getByLabelText('Sebutkan...'), 'Kerabat')
    await userEvent.click(screen.getByRole('radio', { name: 'SUSPICIOUS' }))
    await userEvent.type(screen.getByLabelText('Komentar Suspicious'), 'Mencurigakan')
    // Ex-Gratia hanya tampil untuk PA (ClaimSurvey-sect.xml); klaim ini Fire.
    expect(screen.queryByRole('radio', { name: 'YES' })).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('checkbox', { name: 'Transfer Compliance' }))
    await userEvent.click(screen.getByRole('button', { name: 'Next' }))

    expect(await screen.findByText('Notice of Large Losses diterbitkan.')).toBeInTheDocument()
    expect(calls.find((c) => c.url === '/api/registrasi/register')?.body).toMatchObject({
      pelapor: { hubungan: 7, hubungan_lainnya: 'Kerabat' },
      prinsip_mengenal_nasabah: '2',
      komentar_suspicious: 'Mencurigakan',
      transfer_compliance: true,
      kembali: false,
      mata_uang: 'USD',
    })
  })

  it('menampilkan galat simpan tanpa pelanggaran dan daftar banyak pelanggaran', async () => {
    let answer: () => Answer = () => json(500, { kode: 'galat_internal', pesan: 'Basis data sibuk.' })
    installFetch(response(), (url) => (url === '/api/registrasi/register' ? answer() : undefined))
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Next' }))
    expect(await screen.findByText('Penyimpanan gagal')).toBeInTheDocument()
    expect(screen.getByText('Basis data sibuk.')).toBeInTheDocument()

    answer = () =>
      json(422, {
        kode: 'validasi_gagal',
        pesan: 'Klaim belum lengkap.',
        detail: [
          { kode: 'a', field: 'nomor_polis', pesan: 'Polis deklarasi tidak dapat diklaim.' },
          { kode: 'b', field: 'objek', pesan: 'Objek wajib diisi.' },
          { kode: 'c', field: 'penyebab_kerugian', pesan: 'Penyebab kerugian wajib.' },
          { kode: 'd', field: 'spreading', pesan: 'Total spreading harus 100%.' },
        ],
      })
    await userEvent.click(screen.getByRole('button', { name: 'Next' }))
    expect(await screen.findByText('Klaim belum dapat disimpan')).toBeInTheDocument()
    expect(screen.getByText('4 ketentuan belum terpenuhi:')).toBeInTheDocument()
    expect(screen.getAllByText('Polis deklarasi tidak dapat diklaim.')).toHaveLength(2)
    expect(screen.getAllByText('Total spreading harus 100%.')).toHaveLength(2)
  })

  it('menambah dan membuang objek, coverage, serta spreading, dengan ringkasan share', async () => {
    installFetch(response())
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Tambah objek' }))
    await userEvent.click(screen.getByRole('button', { name: 'Tambah coverage' }))
    await userEvent.type(screen.getByLabelText('TSI'), '1.000')
    await userEvent.click(screen.getByRole('button', { name: 'Tambah spreading' }))
    await userEvent.type(screen.getByLabelText('Share persen'), '60,5')

    const summary = screen.getByRole('heading', { name: 'Ringkasan' }).parentElement!
    expect(within(summary).getByText(/belum 100%/)).toBeInTheDocument()
    expect(within(summary).getByText('1')).toBeInTheDocument()

    await userEvent.clear(screen.getByLabelText('Share persen'))
    await userEvent.type(screen.getByLabelText('Share persen'), '100')
    expect(within(summary).queryByText(/belum 100%/)).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Hapus' }))
    expect(screen.queryByLabelText('Share persen')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Hapus coverage' }))
    expect(screen.queryByLabelText('TSI')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Hapus objek' }))
    expect(screen.queryByRole('heading', { name: 'Ringkasan' })).not.toBeInTheDocument()
  })

  it('menyembunyikan isian wilayah Indonesia untuk negara lain dan mempertahankan nama tersimpan', async () => {
    installFetch(
      response({
        wilayah: {
          negara: 'TIMOR LESTE',
          negara_id: '99',
          provinsi: '',
          provinsi_id: '',
          kota: '',
          kota_id: '',
          kabupaten: '',
          kabupaten_id: '',
          kelurahan: '',
          kelurahan_id: '',
          kode_pos: '',
        },
      }),
    )
    show()

    const negara = await screen.findByLabelText('Negara')
    await within(negara).findByRole('option', { name: 'SINGAPURA' })
    // Negara tersimpan yang tidak ada di master tetap tampil sebagai pilihan terpilih.
    expect(negara).toHaveValue('99')
    expect(within(negara).getByRole('option', { name: 'TIMOR LESTE' })).toBeInTheDocument()
    expect(screen.queryByLabelText('Kota')).not.toBeInTheDocument()

    await userEvent.selectOptions(negara, '2')
    expect(negara).toHaveValue('2')
    expect(screen.queryByLabelText('Kota')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Provinsi')).toBeEnabled()
  })

  it('kembali ke inbox lewat Cancel', async () => {
    installFetch(response())
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'Cancel' }))
    expect(await screen.findByText('Halaman inbox')).toBeInTheDocument()
  })
})
