import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useParams } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { SurveyInboxPage } from './SurveyInboxPage'
import type { KeteranganResponse, TugasSurvei } from './types'

/**
 * Uji tambahan My Work (Inbox Survey): navigasi ke klaim, paginasi, sel kosong, galat, dan
 * bagian KPI. Seluruh data KARANGAN (`D-69`).
 */

const PATH = '/api/inbox-survey'

function tugas(partial: Partial<TugasSurvei> = {}): TugasSurvei {
  return {
    survei_id: 'SRV-1',
    klaim_id: 'KLM-1',
    index_survei: '1',
    appointment_no: 'APP-1',
    reference_no: 'REF-1',
    claim_no: 'PNCN.26.0101',
    policy_no: '',
    insured_name: 'Contoh',
    cob: 'Marine',
    cause_of_loss: '',
    location: 'Gudang',
    pic_asm: 'PIC',
    pic_loss_adjuster: 'BUDI',
    date_of_loss: '2026-08-20',
    aging: 3,
    status_asm: '',
    jenis_surveyor: '2',
    ...partial,
  }
}

const KOLOM = [
  'appointment_no',
  'reference_no',
  'claim_no',
  'policy_no',
  'insured_name',
  'cob',
  'cause_of_loss',
  'location',
  'pic_asm',
  'pic_loss_adjuster',
  'date_of_loss',
  'aging',
  'status_asm',
  'kolom_asing',
].map((kunci) => ({ kunci, judul: kunci.toUpperCase(), tersedia: kunci !== 'status_asm' }))

const KPI_KOLOM = [
  'penjadwalan_survey',
  'immediate_advice',
  'preliminary_advice',
  'interim_report',
  'update_progress',
  'tanggapan_komunikasi',
  'propose_adjustment',
  'final_report',
  'nilai',
  'tidak_dikenal',
].map((kunci) => ({ kunci, judul: kunci.toUpperCase(), tersedia: true }))

const KETERANGAN: KeteranganResponse = {
  portal: 'ASM',
  kolom: KOLOM,
  tab: [
    { kunci: 'aktif', judul: 'Aktif', tersedia: true },
    { kunci: 'mati', judul: 'Mati', tersedia: false },
  ],
  kolom_kpi: KPI_KOLOM,
  tab_bawaan: 'aktif',
  status_survei: ["ALL", "OUTSTANDING", "FINAL"],
  tipe_report: ["DATA SUMMARY", "DATA DETAIL"],
  kuartal: ["1", "2", "3", "4"],
  ukuran_halaman: 25,
  keterbatasan: [],
}

const IDENTITAS = { login: 'ADJ', nama: 'BUDI', leader: false, cakupan: ['BUDI'], jumlah_tim: 0 }

function kpiRow(kelompok: string, nilai: number) {
  return {
    kelompok,
    penjadwalan_survey: nilai,
    immediate_advice: nilai,
    preliminary_advice: nilai,
    interim_report: nilai,
    update_progress: nilai,
    tanggapan_komunikasi: nilai,
    propose_adjustment: nilai,
    final_report: nilai,
    nilai,
  }
}

type Answer = Response | Promise<Response> | undefined

let urls: string[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

function daftar(data: TugasSurvei[], total = data.length, lewati = 0) {
  return json(200, { portal: 'ASM', identitas: IDENTITAS, data, total, lewati, batas: 25, tab: 'aktif', cari: '' })
}

function installFetch(answer: (url: string) => Answer = () => undefined) {
  vi.stubGlobal('fetch', (url: string) => {
    urls.push(url)
    const custom = answer(url)
    if (custom) return Promise.resolve(custom)
    if (url.includes('/keterangan')) return Promise.resolve(json(200, KETERANGAN))
    if (url.includes('/jumlah-tab')) {
      return Promise.resolve(json(200, { portal: 'ASM', identitas: IDENTITAS, tab: [] }))
    }
    // Diperiksa SEBELUM /kpi — keduanya berawalan sama.
    if (url.includes('/kpi/tahun')) {
      return Promise.resolve(json(200, { portal: 'ASM', tahun: ['2026', '2025'] }))
    }
    if (url.includes('/kpi')) {
      // Bentuk jawabannya mengikuti isian, sama seperti server. Mengembalikan satu bentuk
      // untuk setiap permintaan akan membuat uji judul kolom lulus tanpa menguji apa pun.
      const berkuartal = url.includes('kuartal=')

      return Promise.resolve(
        json(200, {
          portal: 'ASM',
          identitas: IDENTITAS,
          status_survei: 'FINAL',
          tipe_report: 'DATA SUMMARY',
          kuartal: berkuartal ? '3' : '',
          tahun: berkuartal ? '2026' : '',
          bentuk: berkuartal ? 'per-tahun' : 'per-adjuster',
          kolom_awal: [
            berkuartal
              ? { kunci: 'kelompok', judul: 'TAHUN', tersedia: true }
              : { kunci: 'kelompok', judul: 'ADJUSTER', tersedia: true },
          ],
          data: [kpiRow('BUDI', 81), kpiRow('', 70)],
        }),
      )
    }
    return Promise.resolve(daftar([tugas()]))
  })
}

function ClaimTarget() {
  const { nomor } = useParams()
  return <p>Halaman klaim {nomor}</p>
}

function show() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false, gcTime: 0 } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/inbox-survey']}>
        <Routes>
          <Route path="/inbox-survey" element={<SurveyInboxPage />} />
          <Route path="/view-claim/:nomor" element={<ClaimTarget />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

const isList = (url: string) =>
  url.startsWith(`${PATH}?`)

beforeEach(() => {
  urls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

describe('antrean', () => {
  it('membuka halaman klaim saat nomor klaim ditekan', async () => {
    installFetch()
    show()

    await userEvent.click(await screen.findByRole('button', { name: 'PNCN.26.0101' }))
    expect(await screen.findByText('Halaman klaim PNCN.26.0101')).toBeInTheDocument()
  })

  it('menandai sel kosong dan mengabaikan kolom yang tidak dikenal', async () => {
    installFetch((url) =>
      isList(url)
        ? daftar([tugas({ claim_no: '', appointment_no: '', reference_no: '' })])
        : undefined,
    )
    show()

    const row = (await screen.findByText('belum bernomor')).closest('tr')!
    // Appointment, Reference, dan Status ASM kosong → "belum tersedia".
    expect(within(row).getAllByText('belum tersedia')).toHaveLength(3)
    // Policy No dan Cause Of Loss kosong → tanda hubung.
    expect(within(row).getAllByText('—')).toHaveLength(2)
    expect(screen.getByRole('columnheader', { name: /STATUS_ASM · belum tersedia/ })).toBeInTheDocument()
    expect(screen.queryByRole('columnheader', { name: /KOLOM_ASING/ })).not.toBeInTheDocument()
  })

  it('menjelaskan tab belum tersedia tanpa alasan dengan kalimat bawaan', async () => {
    installFetch()
    show()
    await screen.findByText('PNCN.26.0101')

    await userEvent.click(screen.getByRole('tab', { name: 'Mati (belum tersedia)' }))
    expect(await screen.findByText('Tab Mati belum dapat ditampilkan')).toBeInTheDocument()
    expect(
      screen.getByText(
        'Kolom penggeraknya belum tersedia di basis data. Tab lain pada bilah di atas tetap berisi.',
      ),
    ).toBeInTheDocument()
  })

  it('berpindah halaman dan memuat ulang antrean beserta jumlah tab', async () => {
    installFetch((url) => {
      if (!isList(url)) return undefined
      const lewati = Number(new URL(url, 'https://x').searchParams.get('lewati') ?? '0')
      return daftar([tugas({ claim_no: `PNCN.26.${lewati}` })], 60, lewati)
    })
    show()
    await screen.findByText('PNCN.26.0')

    await userEvent.click(screen.getByRole('button', { name: /halaman berikutnya|Berikutnya/i }))
    expect(await screen.findByText('PNCN.26.25')).toBeInTheDocument()
    expect(urls).toContain(`${PATH}?tab=aktif&lewati=25&batas=25`)

    const lists = urls.filter(isList).length
    const counts = urls.filter((u) => u.includes('/jumlah-tab')).length
    await userEvent.click(screen.getByRole('button', { name: /Muat ulang/ }))
    await waitFor(() => expect(urls.filter(isList).length).toBe(lists + 1))
    await waitFor(() => expect(urls.filter((u) => u.includes('/jumlah-tab')).length).toBe(counts + 1))
  })

  it('menampilkan galat antrean dengan pesan umum untuk galat bukan API', async () => {
    installFetch((url) =>
      isList(url)
        ? ({
            get status(): number {
              throw new TypeError('rusak')
            },
          } as unknown as Response)
        : undefined,
    )
    show()

    expect(await screen.findByText('Antrean tidak dapat dimuat')).toBeInTheDocument()
    expect(
      screen.getByText('Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'),
    ).toBeInTheDocument()
  })

  it('menampilkan nama dari hitungan tab selama antrean masih dimuat', async () => {
    installFetch((url) => (isList(url) ? new Promise<Response>(() => {}) : undefined))
    show()

    expect(await screen.findByText('BUDI')).toBeInTheDocument()
    expect(screen.getByText(/Ditampilkan sebagai/)).toBeInTheDocument()
  })
})

describe('KPI', () => {
  async function openKPI() {
    show()
    await screen.findByText('PNCN.26.0101')
    await userEvent.click(screen.getByRole('tab', { name: 'KPI' }))
  }

  /** Mengisi kedua dropdown wajib lalu menekan Cari. */
  async function cari(status = 'FINAL', tipe = 'DATA SUMMARY') {
    await userEvent.selectOptions(screen.getByLabelText(/Status Survey/), status)
    await userEvent.selectOptions(screen.getByLabelText(/Tipe Report/), tipe)
    await userEvent.click(screen.getByRole('button', { name: 'Cari' }))
  }

  it('mengurutkan, menandai kelompok kosong, dan mengirim isian ke server', async () => {
    installFetch()
    await openKPI()
    await cari()

    const table = await screen.findByRole('table', { name: 'Ringkasan KPI adjuster' })

    expect(within(table).getAllByText('81.00').length).toBeGreaterThan(0)
    expect(within(table).getByText('—')).toBeInTheDocument()
    expect(
      within(table).queryByRole('columnheader', { name: /TIDAK_DIKENAL/ }),
    ).not.toBeInTheDocument()

    for (const header of within(table).getAllByRole('columnheader')) {
      await userEvent.click(within(header).getByRole('button'))
    }
    const nilai = within(table).getByRole('columnheader', { name: /^NILAI/ })
    await userEvent.click(within(nilai).getByRole('button'))
    // NILAI sudah urut naik dari putaran di atas; klik kedua membaliknya menjadi turun.
    expect(within(table).getAllByRole('row')[1]).toHaveTextContent('BUDI')

    await waitFor(() =>
      expect(urls).toContain(`${PATH}/kpi?status_survei=FINAL&tipe_report=DATA+SUMMARY`),
    )
  })

  /**
   * Kuartal dan Tahun Kuartal ikut terkirim, dan HANYA setelah Cari ditekan.
   *
   * Mengubah dropdown tanpa menekan Cari tidak boleh menembak apa pun — itu perbedaan nyata
   * dari versi sebelumnya layar ini, yang memuat ulang pada setiap perubahan.
   */
  it('mengirim kuartal dan tahun hanya setelah Cari ditekan', async () => {
    installFetch()
    await openKPI()
    await cari()
    await screen.findByRole('table', { name: 'Ringkasan KPI adjuster' })

    const sebelum = urls.filter((u) => u.includes('/kpi?')).length
    await userEvent.selectOptions(screen.getByLabelText('Kuartal'), '3')
    await userEvent.selectOptions(screen.getByLabelText('Tahun Kuartal'), '2026')
    expect(urls.filter((u) => u.includes('/kpi?')).length).toBe(sebelum)

    await userEvent.click(screen.getByRole('button', { name: 'Cari' }))
    await waitFor(() =>
      expect(urls).toContain(
        `${PATH}/kpi?status_survei=FINAL&tipe_report=DATA+SUMMARY&kuartal=3&tahun=2026`,
      ),
    )
  })

  it('menampilkan galat dan keadaan kosong ringkasan', async () => {
    let fail = true
    installFetch((url) => {
      if (!url.includes('/kpi')) return undefined
      if (fail) return json(500, { kode: 'galat_internal', pesan: 'KPI gagal.' })
      return json(200, {
        portal: 'ASM',
        identitas: IDENTITAS,
        status_survei: 'FINAL',
        tipe_report: 'DATA SUMMARY',
        kuartal: '',
        tahun: '',
        data: [],
      })
    })
    await openKPI()
    await cari()

    expect(await screen.findByText('Ringkasan KPI tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('KPI gagal.')).toBeInTheDocument()

    fail = false
    await userEvent.click(screen.getByRole('button', { name: /Muat ulang/ }))
    expect(
      await screen.findByText('Belum ada penilaian KPI untuk pilihan Anda.'),
    ).toBeInTheDocument()
  })
})

describe('keterangan', () => {
  it('menampilkan pesan umum bila keterangan gagal karena galat bukan API', async () => {
    installFetch((url) =>
      url.includes('/keterangan')
        ? ({
            get status(): number {
              throw new TypeError('rusak')
            },
          } as unknown as Response)
        : undefined,
    )
    show()

    expect(await screen.findByText('Layar tidak dapat disiapkan')).toBeInTheDocument()
    expect(
      screen.getByText('Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'),
    ).toBeInTheDocument()
  })
})
