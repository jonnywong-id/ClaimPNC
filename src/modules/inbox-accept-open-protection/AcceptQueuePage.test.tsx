import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { AcceptQueuePage } from './AcceptQueuePage'
import type { Protection, ProtectionDetail, ProtectionListResponse, Queue } from './types'

/** Data uji seluruhnya KARANGAN (`D-69`). */
function protection(partial: Partial<Protection> = {}): Protection {
  return {
    nomor_proteksi: 'OPCN.26.0001',
    nomor_polis: '99.001.2026.00000001',
    nomor_klaim: 'PNCN.26.0007',
    tipe_proteksi: '1',
    nama_tipe_proteksi: 'General',
    tanggal_proteksi: '2026-09-17',
    keterangan: 'Menunggu akseptasi.',
    user_create: 'ADMINCONTOH',
    antrean: 'non-premi',
    ...partial,
  }
}

function detail(partial: Partial<ProtectionDetail> = {}): ProtectionDetail {
  return {
    ...protection(),
    nama_tertanggung: 'Tertanggung Contoh',
    polis_mulai: '2026-01-01',
    polis_akhir: '2026-12-31',
    status_akseptasi: '',
    tanggal_akseptasi: '',
    diaksep_oleh: '',
    menunggu_keputusan: true,
    // null adalah keadaan BAWAAN: tipe '1' pada fixture ini memang tidak memunculkan panel
    // Detail Perubahan. Uji yang membutuhkannya mengisinya lewat `partial`.
    detail_perubahan: null,
    ...partial,
  }
}

function listResponse(partial: Partial<ProtectionListResponse> = {}): ProtectionListResponse {
  return { proteksi: [protection()], total: 1, antrean: 'non-premi', ...partial }
}

type Call = { url: string; init: RequestInit | undefined }

let calls: Call[] = []

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
}

/**
 * Memasang stub fetch.
 *
 * Rute `/antrean` — daftar antrean yang boleh dibuka pemanggil — dijawab DI SINI, bukan oleh
 * tiap uji. Layar memanggilnya lebih dulu dan menahan daftar proteksi sampai jawabannya
 * tiba, sehingga uji yang lupa menanganinya akan gagal dengan gejala yang menyesatkan:
 * tabelnya kosong, seolah penyaringnya yang salah.
 *
 * Bawaannya KEDUA antrean, supaya uji yang sedang menguji hal lain tidak diam-diam terhalang
 * kewenangan. Uji kewenangan menimpanya lewat `queues`.
 */
function stubFetch(
  answer: (url: string, init?: RequestInit) => Response | Promise<Response>,
  queues: Queue[] | { status: number } = ['non-premi', 'premi'],
) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, init })

    if (url.includes('/antrean')) {
      if (!Array.isArray(queues)) {
        return Promise.resolve(
          jsonResponse(queues.status, {
            kode: 'tidak_berwenang',
            pesan: 'Access group Anda tidak berwenang atas layar akseptasi proteksi.',
          }),
        )
      }
      return Promise.resolve(jsonResponse(200, { antrean: queues }))
    }

    // Dokumen penunjang dijawab daftar KOSONG secara baku.
    //
    // Sejak sakelar `FITUR_DOKUMEN_PENUNJANG_AKTIF` dinyalakan (2026-10-03), cabang ini
    // BENAR-BENAR terpakai: panelnya dirender pada form akseptasi, dan membuka satu baris
    // selalu menembak jalur ini.
    //
    // Tanpa cabang ini, setiap uji di berkas ini menerima badan daftar proteksi sebagai
    // daftar dokumen — dan yang gagal bukan panelnya, melainkan uji yang sedang menguji hal
    // lain.
    if (url.includes('/dokumen-penunjang')) {
      return Promise.resolve(jsonResponse(200, { data: [] }))
    }

    return Promise.resolve(answer(url, init))
  })
}

function renderPage() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false, gcTime: 0 }, mutations: { retry: false } },
  })

  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <AcceptQueuePage />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-09-30T12:00:00Z' })
  useSelectedPortal.setState({ alias: 'ASM' })
})

afterEach(() => {
  vi.unstubAllGlobals()
})

it('membuka antrean NON PREMI lebih dulu, bukan PREMI', async () => {
  // Arahnya dipilih sengaja: antrean PREMI menyangkut penagihan premi dan pemiliknya satu
  // peran tertentu.
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('OPCN.26.0001')
  expect(calls.some((call) => call.url.includes('antrean=non-premi'))).toBe(true)
})

it('meminta antrean PREMI ke server saat tabnya dipilih', async () => {
  stubFetch(() => jsonResponse(200, listResponse({ antrean: 'premi' })))
  renderPage()

  await screen.findByText('OPCN.26.0001')
  await userEvent.click(screen.getByRole('tab', { name: 'Proteksi Klaim PREMI' }))

  await waitFor(() => {
    expect(calls.some((call) => call.url.includes('antrean=premi'))).toBe(true)
  })
})

it('memakai ketujuh judul kolom yang sama dengan section rujukan', async () => {
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  await screen.findByText('OPCN.26.0001')
  const headers = screen.getAllByRole('columnheader').map((cell) => cell.textContent ?? '')

  for (const title of [
    'No Proteksi',
    'No Polis',
    'No Klaim',
    'Tipe Proteksi',
    'Tanggal Proteksi Dibuat',
    'Keterangan',
    'User Create',
  ]) {
    expect(headers.some((header) => header.includes(title))).toBe(true)
  }
})

it('mengirim keputusan setuju ke jalur akseptasi', async () => {
  stubFetch((url, init) => {
    if (init?.method === 'PUT') return jsonResponse(200, detail({ status_akseptasi: '1' }))
    if (url.includes('/OPCN.26.0001')) return jsonResponse(200, detail())
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await userEvent.click(await screen.findByRole('button', { name: 'OPCN.26.0001' }))
  await userEvent.click(await screen.findByRole('button', { name: 'Setujui' }))

  await waitFor(() => {
    const decision = calls.find((call) => call.init?.method === 'PUT')
    expect(decision?.url).toContain('/OPCN.26.0001/akseptasi')
    expect(decision?.init?.body).toContain('setuju')
  })
})

it('mengirim keputusan tolak, bukan setuju, saat tombol Tolak ditekan', async () => {
  stubFetch((url, init) => {
    if (init?.method === 'PUT') return jsonResponse(200, detail({ status_akseptasi: '2' }))
    if (url.includes('/OPCN.26.0001')) return jsonResponse(200, detail())
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await userEvent.click(await screen.findByRole('button', { name: 'OPCN.26.0001' }))
  await userEvent.click(await screen.findByRole('button', { name: 'Tolak' }))

  await waitFor(() => {
    const decision = calls.find((call) => call.init?.method === 'PUT')
    expect(decision?.init?.body).toContain('tolak')
  })
})

// Antrean BERSAMA: baris dapat diputuskan petugas lain kapan saja. Menampilkan tombol yang
// pasti ditolak hanya membuat pengguna mencobanya.
it('menyembunyikan tombol keputusan untuk proteksi yang sudah diputuskan', async () => {
  stubFetch((url) => {
    if (url.includes('/OPCN.26.0001')) {
      return jsonResponse(
        200,
        detail({
          status_akseptasi: '1',
          diaksep_oleh: 'KOLEKSICONTOH',
          tanggal_akseptasi: '2026-09-20',
          menunggu_keputusan: false,
        }),
      )
    }
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await userEvent.click(await screen.findByRole('button', { name: 'OPCN.26.0001' }))

  expect(await screen.findByText(/disetujui/)).toBeInTheDocument()
  expect(screen.getByText(/KOLEKSICONTOH/)).toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Setujui' })).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Tolak' })).not.toBeInTheDocument()
})

// Petugas kedua yang kalah cepat BUKAN sedang melakukan kesalahan; pesannya harus
// menyatakan keputusannya sudah diambil, bukan galat teknis.
it('menyampaikan bahwa keputusan sudah diambil petugas lain', async () => {
  stubFetch((url, init) => {
    if (init?.method === 'PUT') {
      return jsonResponse(409, {
        kode: 'konflik',
        pesan:
          'Permintaan proteksi ini sudah diakseptasi. Muat ulang daftar untuk melihat keputusannya.',
      })
    }
    if (url.includes('/OPCN.26.0001')) return jsonResponse(200, detail())
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await userEvent.click(await screen.findByRole('button', { name: 'OPCN.26.0001' }))
  await userEvent.click(await screen.findByRole('button', { name: 'Setujui' }))

  expect(await screen.findByText(/sudah diakseptasi/)).toBeInTheDocument()
})

it('menuntun memilih entitas lebih dulu dan tidak meminta data milik portal', async () => {
  useSelectedPortal.setState({ alias: null })
  stubFetch(() => jsonResponse(200, listResponse()))
  renderPage()

  expect(await screen.findByText('Pilih entitas lebih dulu')).toBeInTheDocument()

  // Yang dijaga: tidak ada permintaan atas data MILIK PORTAL — daftar proteksi tinggal di
  // basis data tiap entitas, dan memintanya tanpa portal aktif berarti menebak entitasnya
  // (`R-20`).
  //
  // Rute `/antrean` DIKECUALIKAN, dan itu bukan kelonggaran: kewenangan seseorang sama di
  // keempat portal (`D-78`) dan tabelnya tinggal di basis data utama, sehingga ia tidak
  // menuntut portal aktif sama sekali.
  const milikPortal = calls.filter((call) => !call.url.includes('/antrean'))
  expect(milikPortal).toHaveLength(0)
})

it('hanya menggambar tab antrean yang menjadi hak pemanggil', async () => {
  // Layar lama tidak punya pemilihan: grid yang bukan haknya TIDAK PERNAH dirender
  // (`Section/InputProtection_Section-Section.xml:1592` dan `:6036`). Menggambar tab yang
  // pasti dijawab 403 hanya membuat pengguna mencobanya.
  stubFetch(() => jsonResponse(200, listResponse({ antrean: 'premi' })), ['premi'])
  renderPage()

  await screen.findByText('OPCN.26.0001')

  expect(screen.getByRole('tab', { name: 'Proteksi Klaim PREMI' })).toBeInTheDocument()
  expect(screen.queryByRole('tab', { name: 'Proteksi Klaim NON PREMI' })).not.toBeInTheDocument()

  // Antrean yang dibuka mengikuti haknya, bukan bawaan NON PREMI.
  expect(calls.some((call) => call.url.includes('antrean=premi'))).toBe(true)
  expect(calls.some((call) => call.url.includes('antrean=non-premi'))).toBe(false)
})

it('menjelaskan penolakan kewenangan, bukan menampilkannya sebagai gangguan', async () => {
  // Petugas yang salah membuka layar perlu tahu bahwa ia MEMANG tidak berhak — bukan
  // mengira datanya hilang lalu melaporkannya sebagai gangguan.
  stubFetch(() => jsonResponse(200, listResponse()), { status: 403 })
  renderPage()

  expect(await screen.findByText('Anda tidak berwenang atas layar ini')).toBeInTheDocument()

  // Tidak ada tab, dan daftar proteksi tidak pernah diminta.
  expect(screen.queryByRole('tab')).not.toBeInTheDocument()
  expect(calls.some((call) => call.url.includes('antrean=non-premi'))).toBe(false)
})

it('menampilkan panel Detail Perubahan DOL beserta nilai sebelum dan sesudahnya', async () => {
  stubFetch((url) => {
    if (url.includes('/OPCN.26.0001'))
      return jsonResponse(
        200,
        detail({
          tipe_proteksi: '7',
          nama_tipe_proteksi: 'Perubahan DOL',
          detail_perubahan: {
            judul: 'Detail Perubahan DOL',
            dol_sebelum: '2026-08-03',
            dol_sesudah: '2026-08-05',
            penyebab_sebelum: '',
            penyebab_sesudah: '',
            nama_objek: 'Objek Contoh',
            nama_cabang: 'Cabang Contoh',
            kosong: false,
          },
        }),
      )
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await userEvent.click(await screen.findByRole('button', { name: 'OPCN.26.0001' }))

  // Yang diperiksa KEDUA nilainya, bukan sekadar panelnya muncul: panel yang tampil tanpa
  // isi tidak memberi tahu petugas apa pun tentang apa yang diubah.
  expect(await screen.findByText('Detail Perubahan DOL')).toBeInTheDocument()
  expect(screen.getByText('Current Date Of Loss')).toBeInTheDocument()
  expect(screen.getByText('Next Date Of Loss')).toBeInTheDocument()
  expect(screen.getByText('Objek Contoh')).toBeInTheDocument()
  expect(screen.getByText('Cabang Contoh')).toBeInTheDocument()
})

it('memakai label Cause Of Loss, bukan label DOL, pada permintaan tipe 8', async () => {
  stubFetch((url) => {
    if (url.includes('/OPCN.26.0001'))
      return jsonResponse(
        200,
        detail({
          tipe_proteksi: '8',
          detail_perubahan: {
            judul: 'Detail Perubahan Cause Of Loss',
            dol_sebelum: '',
            dol_sesudah: '',
            penyebab_sebelum: '12001',
            penyebab_sesudah: '12002',
            nama_objek: '',
            nama_cabang: '',
            kosong: false,
          },
        }),
      )
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await userEvent.click(await screen.findByRole('button', { name: 'OPCN.26.0001' }))

  expect(await screen.findByText('Cause Of Loss Dipilih')).toBeInTheDocument()
  expect(screen.getByText('Next Cause Of Loss')).toBeInTheDocument()
  expect(screen.getByText('12001')).toBeInTheDocument()
  expect(screen.getByText('12002')).toBeInTheDocument()
  expect(screen.queryByText('Current Date Of Loss')).not.toBeInTheDocument()
})

it('tetap menampilkan panelnya untuk baris warisan yang rinciannya kosong', async () => {
  stubFetch((url) => {
    if (url.includes('/OPCN.26.0001'))
      return jsonResponse(
        200,
        detail({
          tipe_proteksi: '7',
          detail_perubahan: {
            judul: 'Detail Perubahan DOL',
            dol_sebelum: '',
            dol_sesudah: '',
            penyebab_sebelum: '',
            penyebab_sesudah: '',
            nama_objek: '',
            nama_cabang: '',
            kosong: true,
          },
        }),
      )
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await userEvent.click(await screen.findByRole('button', { name: 'OPCN.26.0001' }))

  // Menyembunyikannya akan membuat permintaan perubahan tampak seolah tidak mengubah apa
  // pun. Seluruh baris warisan Pega berkeadaan begini.
  expect(await screen.findByText('Detail Perubahan DOL')).toBeInTheDocument()
  expect(
    screen.getByText('Rincian perubahan tidak tersedia untuk permintaan ini.'),
  ).toBeInTheDocument()
})

it('tidak menampilkan panel Detail Perubahan pada tipe yang tidak memilikinya', async () => {
  stubFetch((url) => {
    if (url.includes('/OPCN.26.0001')) return jsonResponse(200, detail())
    return jsonResponse(200, listResponse())
  })
  renderPage()

  await userEvent.click(await screen.findByRole('button', { name: 'OPCN.26.0001' }))
  await screen.findByText('Akseptasi Proteksi')

  expect(screen.queryByText(/Detail Perubahan/)).not.toBeInTheDocument()
})
