import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { InvestigatorTask } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { InvestigatorInboxPage } from './InvestigatorInboxPage'

/**
 * Seluruh nomor polis, nama tertanggung, nama peserta, dan nomor case di berkas ini
 * KARANGAN (`D-69`): data nasabah tidak pernah ditulis ke berkas yang di-commit, dan
 * larangan itu berlaku pada data uji persis seperti pada dokumen.
 */

/** Pekerjaan LENGKAP, baru menunggu beberapa jam. */
const BARU: InvestigatorTask = {
  referensi: 'ASM-FW-GCNMFW-WORK PNC-100241',
  nomor_case: 'PNC-100241',
  nomor_polis: '00.000.2026.00001',
  nama_tertanggung: 'PT Contoh Sejahtera Abadi',
  nama_peserta: 'Peserta Contoh Satu',
  nama_bisnis: 'Personal Accident',
  nama_cabang: 'Contoh Pusat',
  nama_admin: 'ADMINCONTOH1',
  tanggal_pendaftaran: '2026-09-21T02:15:00Z',
  tanggal_survey: '2026-09-22T01:00:00Z',
}

/**
 * Pekerjaan TANPA tanggal survei.
 *
 * Kolom kesembilan pada barisnya kosong, dan layar menampilkannya sebagai tanda hubung.
 * Barisnya sendiri TETAP tampil — ia masih menunggu di antrean.
 */
const TANPA_SURVEI: InvestigatorTask = {
  referensi: 'ASM-FW-GCNMFW-WORK PNC-100236',
  nomor_case: 'PNC-100236',
  nomor_polis: '00.000.2026.00003',
  nama_tertanggung: 'PT Contoh Bahari Lestari',
  nama_peserta: 'Peserta Contoh Tiga',
  nama_bisnis: 'Marine Cargo',
  nama_cabang: 'Contoh Cabang Barat',
  nama_admin: 'ADMINCONTOH1',
  tanggal_pendaftaran: '2026-09-15T08:05:00Z',
  tanggal_survey: null,
}

/**
 * Pekerjaan tanpa Nama Peserta DAN tanpa Nama Admin.
 *
 * Dua keadaan sekaligus: klaim yang objek pertanggungannya belum terisi, dan kasus yang
 * dibuat proses terjadwal alih-alih orang (`D-57`).
 */
const LAMA: InvestigatorTask = {
  referensi: 'ASM-FW-GCNMFW-WORK PNC-099876',
  nomor_case: 'PNC-099876',
  nomor_polis: '00.000.2026.00007',
  nama_tertanggung: 'PT Contoh Persada Mandiri',
  nama_peserta: '',
  nama_bisnis: 'Aneka',
  nama_cabang: 'Contoh Cabang Timur',
  nama_admin: '',
  tanggal_pendaftaran: '2026-07-30T07:00:00Z',
  tanggal_survey: '2026-08-03T02:00:00Z',
}

type Call = { url: string; method: string; header: Record<string, string> }

let calls: Call[] = []

type Reply = { body: unknown; status?: number }

function installFetch(map: (call: Call) => Reply) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
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

/** reply melayani daftar dengan baris yang diberikan. */
function reply(tugas: InvestigatorTask[], terpotong = false): (call: Call) => Reply {
  return () => ({
    body: {
      tugas,
      terpotong,
      batas_baris: 500,
      portal: 'ASM',
    },
  })
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <InvestigatorInboxPage />
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

describe('antrean Inbox Investigator', () => {
  it('menampilkan kesembilan kolom dari baris yang dijawab server', async () => {
    installFetch(reply([BARU, TANPA_SURVEI]))
    show()

    const table = await screen.findByRole('table')
    const rows = within(table).getAllByRole('row')

    // Satu baris kepala + dua baris isi.
    expect(rows).toHaveLength(3)

    expect(within(table).getByText('PNC-100241')).toBeInTheDocument()
    expect(within(table).getByText('00.000.2026.00001')).toBeInTheDocument()
    expect(within(table).getByText('PT Contoh Sejahtera Abadi')).toBeInTheDocument()
    expect(within(table).getByText('Peserta Contoh Satu')).toBeInTheDocument()
    expect(within(table).getByText('Personal Accident')).toBeInTheDocument()
    expect(within(table).getByText('Contoh Pusat')).toBeInTheDocument()
    // Kedua baris contoh dibuat admin yang sama — sengaja, karena satu petugas memang
    // mendaftarkan banyak klaim. Yang dibuktikan di sini kolomnya digambar, bukan nilainya
    // unik.
    expect(within(table).getAllByText('ADMINCONTOH1')).toHaveLength(2)
  })

  /**
   * Aturan paling penting di layar ini.
   *
   * Kolom kesembilan bercaption "Lama Masuk Inbox", tetapi ISINYA tanggal survei — sel
   * Pega-nya terikat `.ClaimData.SurveyResults(1).SurveyDate`. Ketidakcocokan judul dan isi
   * itu ADA di sistem lama dan direplikasi apa adanya (`P-5`).
   */
  it('menampilkan tanggal survei di bawah judul Lama Masuk Inbox', async () => {
    installFetch(reply([BARU]))
    show()

    const table = await screen.findByRole('table')

    // 22 Sep 2026 UTC -> tetap 22 Sep di WIB.
    expect(within(table).getByText(/22 Sep 2026/)).toBeInTheDocument()
    expect(within(table).queryByText(/jam$/)).not.toBeInTheDocument()
    expect(within(table).queryByText(/hari/)).not.toBeInTheDocument()
  })

  it('menampilkan tanda hubung bila klaimnya belum punya data survei', async () => {
    installFetch(reply([TANPA_SURVEI]))
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getAllByText('—').length).toBeGreaterThan(0)
  })

  /** Grid Pega punya SEMBILAN kolom; tidak ada kolom "Tanggal Survey" terpisah. */
  it('menggambar tepat sembilan kolom seperti grid Pega', async () => {
    installFetch(reply([BARU]))
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getAllByRole('columnheader')).toHaveLength(9)
    expect(within(table).queryByText('Tanggal Survey')).not.toBeInTheDocument()
    // Judul dibaca dari sel kepalanya, bukan dari seluruh tabel: DataTable menggambar ulang
    // nama kolom di dalam sel untuk tampilan kartu, sehingga teksnya memang muncul dua kali.
    const headers = within(table)
      .getAllByRole('columnheader')
      .map((cell) => cell.textContent)
    expect(headers).toContain('Lama Masuk Inbox')
  })

  /**
   * Nama Admin kosong berarti kasusnya dibuat proses terjadwal, bukan orang (`D-57`).
   *
   * Ditulis apa adanya alih-alih dibiarkan sebagai sel kosong yang terbaca seperti data
   * hilang.
   */
  it('menyebut kasus tanpa Nama Admin sebagai dibuat proses terjadwal', async () => {
    installFetch(reply([LAMA]))
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getByText('proses terjadwal')).toBeInTheDocument()
  })

  /**
   * Pemotongan DINYATAKAN, tidak dibiarkan senyap seperti `pyMaxRecords = 500` pada sistem
   * lama. Batas yang diketahui adalah batas; batas yang senyap adalah data yang hilang.
   */
  it('menyatakan daftarnya terpotong saat server memberi tahu begitu', async () => {
    installFetch(reply([BARU], true))
    show()

    expect(await screen.findByText(/lebih panjang/i)).toBeInTheDocument()
    expect(screen.getByText(/500 pekerjaan pertama/i)).toBeInTheDocument()
  })

  it('tidak menyatakan terpotong saat seluruh antrean terkirim', async () => {
    installFetch(reply([BARU, TANPA_SURVEI]))
    show()

    await screen.findByRole('table')
    expect(screen.queryByText(/lebih panjang/i)).not.toBeInTheDocument()
  })

  /**
   * Inbox yang bersih adalah keadaan yang DIHARAPKAN, bukan kegagalan — dan pesannya harus
   * mengatakan itu, bukan menampilkan galat.
   */
  it('menyatakan antrean kosong sebagai keadaan biasa, bukan galat', async () => {
    installFetch(reply([]))
    show()

    expect(
      await screen.findByText(/tidak ada pekerjaan yang menunggu/i),
    ).toBeInTheDocument()
  })

  /**
   * Antrean satu badan hukum bukan antrean badan hukum lain. Permintaan WAJIB membawa
   * header portal, dan layar menyebut entitas yang sedang dibuka (`ADR-0030`, `R-20`).
   */
  it('mengirim header portal dan menyebut entitas yang sedang dibuka', async () => {
    installFetch(reply([BARU]))
    show()

    await screen.findByRole('table')

    expect(calls).toHaveLength(1)
    expect(calls[0]?.url).toContain('/api/inbox/investigator')
    expect(calls[0]?.method).toBe('GET')
    expect(Object.values(calls[0]?.header ?? {})).toContain('ASM')
    expect(screen.getByText('ASM')).toBeInTheDocument()
  })

  /** Layar ini BACA-SAJA; tidak boleh ada tombol yang menjanjikan perubahan. */
  it('tidak menawarkan tombol yang mengubah apa pun', async () => {
    installFetch(reply([BARU]))
    show()

    await screen.findByRole('table')

    expect(screen.queryByRole('button', { name: /tambah/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /simpan/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /hapus/i })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /ambil/i })).not.toBeInTheDocument()
  })

  it('memuat ulang antrean saat tombol Refresh ditekan', async () => {
    installFetch(reply([BARU]))
    show()

    await screen.findByRole('table')
    expect(calls).toHaveLength(1)

    await userEvent.click(screen.getByRole('button', { name: /refresh/i }))

    expect(calls.length).toBeGreaterThan(1)
  })

  /**
   * Penyaringan dikerjakan PERAMBAN atas baris yang sudah di tangan — pilihan Work Owner
   * 2026-09-23, meniru grid Pega. Mengetik karena itu TIDAK boleh menembak server.
   */
  it('menyaring di peramban tanpa menembak server lagi', async () => {
    installFetch(reply([BARU, TANPA_SURVEI, LAMA]))
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getAllByRole('row')).toHaveLength(4)

    const before = calls.length
    await userEvent.type(screen.getByRole('searchbox'), '100241')

    expect(within(await screen.findByRole('table')).getAllByRole('row')).toHaveLength(2)
    expect(calls).toHaveLength(before)
  })


  /**
   * Export Data Investigation DIHAPUS atas keputusan Work Owner 2026-09-24.
   *
   * Uji ini menjaga keputusan itu: bila kelak seseorang menghidupkannya kembali tanpa
   * pemetaan kolomnya jelas, inilah yang gagal lebih dulu.
   */
  it('tidak menggambar kendali export sama sekali', async () => {
    installFetch(reply([BARU]))
    show()
    await screen.findByRole('table')

    expect(screen.queryByRole('button', { name: /export/i })).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/pilih investigation/i)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/^dari$/i)).not.toBeInTheDocument()
    expect(screen.queryByLabelText(/^sampai$/i)).not.toBeInTheDocument()
  })

  /** Portal yang belum dipilih menghentikan layar sebelum satu permintaan pun dikirim. */
  it('meminta portal dipilih lebih dulu dan tidak menembak server', async () => {
    useSelectedPortal.getState().clear()
    installFetch(reply([BARU]))
    show()

    expect(await screen.findByText(/portal entitas belum dipilih/i)).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })
})
