import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useParams } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { InboxAdminPage, detailPath } from './InboxAdminPage'
import type { MetadataResponse, Tab, WorkItem } from './types'

/** Uji tambahan Inbox Admin: galat, lini bisnis, paginasi, dan tombol rincian. Nilai KARANGAN. */

const PATH = '/api/inbox-admin'

const TAB_ALL: Tab = {
  kode: '3',
  nama: 'ALL',
  keterangan: 'Seluruh klaim.',
  kolom: [
    { kunci: 'case_id', judul: 'Case ID' },
    { kunci: 'tanggal_kejadian', judul: 'Date of loss' },
    { kunci: 'aging_total', judul: 'Total Aging' },
  ],
  hanya_milik_saya: false,
  pakai_pencarian: true,
  pakai_lini_bisnis: true,
}

const META: MetadataResponse = {
  tab: [TAB_ALL],
  tab_bawaan: '3',
  lini_bisnis: [{ kode: 'PA', label: 'Personal Accident' }],
  // Alasan sengaja tidak dikirim: jawaban jaringan tidak selalu lengkap.
  tab_dinonaktifkan: [{ kode: '4', nama: 'Not Answered' } as MetadataResponse['tab_dinonaktifkan'][number]],
  portal: 'ASM',
  keterbatasan: [],
}

function row(over: Partial<WorkItem> = {}): WorkItem {
  return {
    referensi: 'REF-1',
    case_id: 'PNC-1',
    no_polis: '',
    nama_tertanggung: '',
    nama_bisnis: '',
    sumber_bisnis: '',
    nama_cabang: '',
    cabang_klaim: '',
    pembuat: '',
    tanggal_kejadian: '2026-03-12',
    tanggal_lapor: null,
    tanggal_input: null,
    catatan: '',
    posisi_klaim: '',
    status_klaim: '',
    status_lod: '',
    tanggal_request: null,
    cabang_polis: '',
    cabang_survei: '',
    pic_klaim: '',
    surveyor: '',
    no_survei: '',
    tanggal_masuk_inbox: null,
    deskripsi_analis: '',
    status_rcl_pucl: '',
    tanggal_cetak_surat: null,
    lama_klaim: '',
    status_kadaluarsa: '',
    aging_lapor: null,
    aging_total: 3,
    aging_lod: null,
    aging_request: null,
    ...over,
  }
}

let calls: string[] = []

type Answer = { status?: number; body?: unknown; fail?: boolean }

function stubFetch(list: (url: string) => Answer, meta: Answer = { body: META }) {
  vi.stubGlobal('fetch', (url: string) => {
    calls.push(url)
    const answer = url === `${PATH}/tab` ? meta : list(url)
    if (answer.fail) return Promise.reject(new TypeError('Failed to fetch'))
    return Promise.resolve(
      new Response(JSON.stringify(answer.body ?? {}), {
        status: answer.status ?? 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function page(rows: WorkItem[], halaman = 1, total = rows.length, totalHalaman = 1) {
  return {
    body: {
      tab: TAB_ALL,
      baris: rows,
      paginasi: { halaman, ukuran: 25, total, total_halaman: totalHalaman },
      penyaring: { bisnis: '', cari: '' },
      portal: 'ASM',
    },
  }
}

function Detail() {
  const { key } = useParams()
  return <p>Rincian {key}</p>
}

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter initialEntries={['/inbox-admin']}>
        <Routes>
          <Route path="/inbox-admin" element={<InboxAdminPage />} />
          <Route path="/view-claim/:key" element={<Detail />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({
    token: 'token-uji',
    user: null,
    validUntil: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
  })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSession.getState().clear()
})

it('menyatakan layar tidak dapat dibuka saat bentuknya gagal dimuat', async () => {
  stubFetch(() => page([]), { status: 500, body: { kode: 'galat_internal', pesan: 'Bentuk rusak.' } })
  renderPage()

  expect(await screen.findByText('Layar tidak dapat dibuka')).toBeInTheDocument()
  expect(screen.getByText('Bentuk rusak.')).toBeInTheDocument()
})

it('memakai pesan umum saat antrean gagal karena jaringan', async () => {
  stubFetch(() => ({ fail: true }))
  renderPage()

  expect(await screen.findByText('Antrean tidak dapat dimuat')).toBeInTheDocument()
  expect(
    screen.getByText('Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'),
  ).toBeInTheDocument()
})

it('menyaring per lini bisnis dan menjelaskan kekosongannya', async () => {
  stubFetch((url) => (url.includes('bisnis=PA') ? page([]) : page([row()])))
  const user = userEvent.setup()
  renderPage()

  await screen.findByRole('button', { name: 'Lihat Detail Klaim' })
  // Tab yang dinonaktifkan tanpa alasan memakai keterangan baku.
  expect(screen.getByText(/"Not Answered" tidak dibawa ke sistem baru — tidak dipakai\./)).toBeInTheDocument()

  await user.selectOptions(screen.getByLabelText('Bisnis'), 'PA')
  // Tab bawaan dipilih server, sehingga parameter tab tidak dikirim. Daftar Status Register
  // ikut dihitung ulang dengan lini bisnis yang sama.
  await waitFor(() => expect(calls).toContain(`${PATH}?bisnis=PA`))
  await waitFor(() => expect(calls).toContain(`${PATH}/jumlah?bisnis=PA`))
  expect(await screen.findByText('Tidak ada baris pada lini bisnis yang dipilih.')).toBeInTheDocument()
})

it('menyebut antrean kosong pada tab yang tidak dibatasi pemiliknya', async () => {
  stubFetch(() => page([]))
  renderPage()

  expect(await screen.findByText('Antrean ini sedang kosong.')).toBeInTheDocument()
  expect(screen.queryByRole('status')).not.toBeInTheDocument()
})

it('maju dan mundur halaman serta menyebut rentang baris', async () => {
  stubFetch((url) => (url.includes('halaman=2') ? page([row({ case_id: 'PNC-26' })], 2, 30, 2) : page([row()], 1, 30, 2)))
  const user = userEvent.setup()
  renderPage()

  expect(await screen.findByText('Menampilkan 1–1 dari 30 baris.')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Sebelumnya' })).toBeDisabled()
  await user.click(screen.getByRole('button', { name: 'Berikutnya' }))
  expect(await screen.findByText('Menampilkan 26–26 dari 30 baris.')).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Berikutnya' })).toBeDisabled()
  await user.click(screen.getByRole('button', { name: 'Sebelumnya' }))
  expect(await screen.findByText('Menampilkan 1–1 dari 30 baris.')).toBeInTheDocument()
  expect(calls.some((url) => url.includes('halaman=2'))).toBe(true)
})

it('membuka rincian dengan case ID bila referensi kosong, dan mematikan tombol tanpa kunci', async () => {
  stubFetch(() => page([row({ referensi: '', case_id: 'PNC-7' }), row({ referensi: '', case_id: '' })]))
  const user = userEvent.setup()
  renderPage()

  // Tabel TERAKHIR: yang pertama adalah daftar Status Register.
  await screen.findAllByRole('button', { name: 'Lihat Detail Klaim' })
  const table = screen.getAllByRole('table').at(-1)!
  const buttons = within(table).getAllByRole('button', { name: 'Lihat Detail Klaim' })
  expect(buttons[1]).toBeDisabled()
  expect(within(table).getAllByText('3 hari').length).toBeGreaterThan(0)

  await user.click(buttons[0]!)
  expect(await screen.findByText('Rincian PNC-7')).toBeInTheDocument()
})

it('menampilkan Export LOD hanya di tab Branch Claim, dan tombol Auto Claim di setiap tab', async () => {
  const branch: Tab = { ...TAB_ALL, kode: '12', nama: 'Branch Claim' }
  stubFetch(() => page([]), { body: { ...META, tab: [TAB_ALL, branch] } })
  const user = userEvent.setup()
  renderPage()

  expect(await screen.findByRole('button', { name: 'Export Hasil Auto Claim' })).toBeInTheDocument()
  expect(screen.getByRole('button', { name: 'Export Klaim Gagal' })).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Export LOD' })).not.toBeInTheDocument()

  await user.click(screen.getByRole('tab', { name: /Branch Claim/ }))
  expect(await screen.findByRole('button', { name: 'Export LOD' })).toBeInTheDocument()
})

it('mengunduh ekspor lewat rute ekspor modul', async () => {
  stubFetch(() => page([]))
  vi.stubGlobal('URL', Object.assign(URL, { createObjectURL: () => 'blob:x', revokeObjectURL: () => {} }))
  const user = userEvent.setup()
  renderPage()

  await user.click(await screen.findByRole('button', { name: 'Export Klaim Gagal' }))
  await waitFor(() => expect(calls).toContain(`${PATH}/ekspor/klaim-gagal`))
})

it('menampilkan jumlah setiap antrean pada daftar Status Register', async () => {
  const outstanding: Tab = { ...TAB_ALL, kode: '3', nama: 'Outstanding' }
  stubFetch(
    (url) =>
      url.startsWith(`${PATH}/jumlah`)
        ? { body: { jumlah: [{ kode: '3', nama: 'Outstanding', jumlah: 561, gagal: false }], portal: 'ASM' } }
        : page([]),
    { body: { ...META, tab: [outstanding] } },
  )
  renderPage()

  const tab = await screen.findByRole('tab', { name: /Outstanding/ })
  const rowOfTab = tab.closest('tr')!
  expect(await within(rowOfTab).findByText('561')).toBeInTheDocument()
})

it('menampilkan Pilih Kanwil bagi manajer dan mengirim kanwil pilihannya', async () => {
  stubFetch((url) =>
    url.startsWith(`${PATH}/batas`)
      ? { body: { manajer: true, cabang: '', kanwil: [{ kode: '1', label: 'Kanwil 1' }], portal: 'ASM' } }
      : page([]),
  )
  const user = userEvent.setup()
  renderPage()

  await user.selectOptions(await screen.findByLabelText('Pilih Kanwil'), '1')
  await waitFor(() => expect(calls).toContain(`${PATH}?kanwil=1`))
  await waitFor(() => expect(calls).toContain(`${PATH}/jumlah?kanwil=1`))
})

it('menyebut cabang yang membatasi antrean petugas cabang, tanpa dropdown kanwil', async () => {
  stubFetch((url) =>
    url.startsWith(`${PATH}/batas`)
      ? { body: { manajer: false, cabang: '100351', kanwil: [], portal: 'ASM' } }
      : page([]),
  )
  renderPage()

  expect(await screen.findByText(/dibatasi cabang Anda \(100351\)/)).toBeInTheDocument()
  expect(screen.queryByLabelText('Pilih Kanwil')).not.toBeInTheDocument()
})

it('membuka klaim PNCN langsung di halaman klaim registrasi, klaim Pega ke View Claim', () => {
  expect(detailPath({ case_id: 'PNCN.26.35', referensi: 'PNCN.26.35' })).toBe('/registrasi/klaim/PNCN.26.35')
  expect(detailPath({ case_id: 'PNC-1865', referensi: 'ASM-FW-GCNMFW-WORK PNC-1865' })).toBe(
    '/view-claim/ASM-FW-GCNMFW-WORK%20PNC-1865',
  )
  expect(detailPath({ case_id: '', referensi: '' })).toBeNull()
})
