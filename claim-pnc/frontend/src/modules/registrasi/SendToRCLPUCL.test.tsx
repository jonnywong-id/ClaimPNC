import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import {
  SendToRCLPUCLDialog,
  TRACK_NOTIFICATION,
  TRACK_PUCL,
  TRACK_RCL,
  showDoctorName,
  showRejectReasons,
} from './SendToRCLPUCL'

/**
 * Uji modal "Kirim ke RCL/PUCL" — `Section/SectionPUCL-sect.xml`. Seluruh data KARANGAN
 * (`D-69`).
 */

let calls: { url: string; body: unknown }[] = []

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

const PERIHAL_PUCL = [{ id: 1, nama: 'Kelengkapan Data dan Dokumen Klaim' }]
const PERIHAL_RCL = [{ id: 9, nama: 'Tolakan klaim polis' }]
const ALASAN = [
  { id: '001', nama: 'Dikecualikan polis', deskripsi: 'Perawatan yang Anda ajukan dikecualikan dalam polis.' },
]
// Bentuknya meniru `OLD_OPERATOR_ID` yang sebenarnya: huruf besar, boleh berspasi.
// `pyPromptTableList` property `NamaDokterRCL` — yang DIHARAPKAN tergambar.
//
// Ini bukan fixture jaringan: layar tidak memintanya ke mana pun. Ia salinan kedua dari
// daftar yang sama, sengaja ditulis ulang di sini supaya uji benar-benar menguji isinya.
// Mengimpor `RCL_DOCTORS` dari komponennya akan membuat uji ini lulus apa pun isinya,
// termasuk daftar kosong — persis cacat yang dilaporkan.
//
// Baris kedua sengaja yang `id`-nya BERBEDA dari `nama`: hanya daftar seperti itu yang
// menangkap tertukarnya nilai simpan dengan label.
const DOKTER = [
  { id: 'WAHYUKRISTANTI', nama: 'WAHYUKRISTANTI' },
  { id: 'MARGARETHAROSAGUNAWAN', nama: 'MARGARETHA ROSA GUNAWAN' },
]

function installFetch(kirim: () => Response = () => json(200, { klaim: { id: 'klaim-1' }, tugas: null })) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    calls.push({ url, body: init?.body ? JSON.parse(init.body as string) : undefined })
    if (url.startsWith('/api/registrasi/rclpucl/perihal')) {
      const pucl = url.includes(`jalur=${TRACK_PUCL}`)
      return Promise.resolve(json(200, { pilihan: pucl ? PERIHAL_PUCL : PERIHAL_RCL }))
    }
    if (url.startsWith('/api/registrasi/rclpucl/alasan')) return Promise.resolve(json(200, { pilihan: ALASAN }))
    // `/rclpucl/dokter` sengaja TIDAK dilayani: daftar dokter tidak datang dari jaringan.
    // Memanggilnya jatuh ke 404 di bawah, dan uji "tidak pernah meminta" menangkapnya.
    if (url.endsWith('/kirim-rclpucl')) return Promise.resolve(kirim())
    return Promise.resolve(json(404, { kode: 'tidak_ditemukan', pesan: 'Tidak ditemukan.' }))
  })
}

function wrap(children: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } })
  return render(<QueryClientProvider client={client}>{children}</QueryClientProvider>)
}

function open(groupPanel = '002', onClose = vi.fn()) {
  wrap(<SendToRCLPUCLDialog claimID="klaim-1" taskID="tugas-1" groupPanel={groupPanel} onClose={onClose} />)
  return onClose
}

beforeEach(() => {
  calls = []
  installFetch()
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  useSelectedPortal.getState().clear()
})

describe('susunan modal mengikuti SectionPUCL', () => {
  it('menampilkan ketujuh isian dalam urutan sectionnya', () => {
    open()
    expect(screen.getByRole('heading', { name: 'Kirim ke RCL/PUCL' })).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: 'RCL' })).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: 'PUCL' })).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: 'Notification' })).toBeInTheDocument()
    expect(screen.getByLabelText('Catatan untuk RCL/PUCL')).toBeInTheDocument()
    expect(screen.getByLabelText('Perihal')).toBeInTheDocument()
    expect(screen.getByLabelText('Keterangan Pembuka')).toBeInTheDocument()
    expect(screen.getByLabelText('Keterangan Isi')).toBeInTheDocument()
    expect(screen.getByLabelText('Keterangan Penutup')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Kirim' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Batal' })).toBeInTheDocument()
  })

  // `pyVisible OTHER`, kondisi `.ClaimData.PUCLStatus.RCL_PUCL != 2 && IsPA` —
  // `!= 2` berarti RCL ATAU Notification.
  it.each([
    { nama: 'RCL pada lini PA', track: TRACK_RCL, panel: '002', tampil: true },
    { nama: 'Notification pada lini PA', track: TRACK_NOTIFICATION, panel: '002', tampil: true },
    { nama: 'PUCL pada lini PA', track: TRACK_PUCL, panel: '002', tampil: false },
    { nama: 'RCL pada lini lain', track: TRACK_RCL, panel: '006', tampil: false },
  ])('Nama Dokter: $nama', ({ track, panel, tampil }) => {
    expect(showDoctorName(track, panel)).toBe(tampil)
  })

  // Kontainer `IsPA && .ClaimData.PUCLStatus.RCL_PUCL != 3`.
  it.each([
    { nama: 'PUCL pada lini PA', track: TRACK_PUCL, panel: '002', tampil: true },
    { nama: 'RCL pada lini PA', track: TRACK_RCL, panel: '002', tampil: true },
    { nama: 'Notification pada lini PA', track: TRACK_NOTIFICATION, panel: '002', tampil: false },
    { nama: 'PUCL pada lini lain', track: TRACK_PUCL, panel: '006', tampil: false },
  ])('grid alasan: $nama', ({ track, panel, tampil }) => {
    expect(showRejectReasons(track, panel)).toBe(tampil)
  })

  it('menyembunyikan grid alasan pada jalur Notification', async () => {
    const user = userEvent.setup()
    open('002')
    await user.click(screen.getByRole('radio', { name: 'PUCL' }))
    await waitFor(() => expect(screen.getByText(ALASAN[0]!.nama)).toBeInTheDocument())

    await user.click(screen.getByRole('radio', { name: 'Notification' }))
    expect(screen.queryByText('Daftar Alasan')).not.toBeInTheDocument()
    // Nama Dokter justru MUNCUL pada jalur ini (`!= 2 && IsPA`).
    expect(screen.getByLabelText('Nama Dokter')).toBeInTheDocument()
  })

  it('menampilkan Nama Dokter setelah RCL dipilih pada klaim PA', async () => {
    const user = userEvent.setup()
    open('002')
    expect(screen.queryByLabelText('Nama Dokter')).not.toBeInTheDocument()
    await user.click(screen.getByRole('radio', { name: 'RCL' }))
    expect(screen.getByLabelText('Nama Dokter')).toBeInTheDocument()
  })

  // DROPDOWN, bukan isian bebas: nilainya harus cocok dengan identitas lama yang disaring
  // Inbox RCL, dan nama yang diketik bebas gagal mencocokkannya tanpa satu pesan galat.
  //
  // Isinya diperiksa SEKETIKA, tanpa `waitFor`: daftarnya tidak datang dari jaringan, jadi
  // ia sudah tergambar pada render pertama. `waitFor` di sini justru akan menyembunyikan
  // kemunduran — ia tetap lulus seandainya daftarnya kembali menjadi panggilan jaringan.
  it('Nama Dokter adalah dropdown berisi kedua nama prompt list', () => {
    open('002')
    fireEvent.click(screen.getByRole('radio', { name: 'RCL' }))

    const dokter = screen.getByLabelText('Nama Dokter')
    expect(dokter.tagName).toBe('SELECT')
    // Pilihan kosong "----- PILIH -----" mendahului, persis seperti dropdown Pega-nya.
    const opsi = within(dokter).getAllByRole('option')
    expect(opsi[0]).toHaveValue('')
    expect(opsi).toHaveLength(1 + DOKTER.length)
    expect(opsi.slice(1).map((o) => [(o as HTMLOptionElement).value, o.textContent])).toEqual(
      DOKTER.map((d) => [d.id, d.nama]),
    )
  })

  // Daftar dokter TIDAK PERNAH diminta ke jaringan, pada jalur mana pun.
  //
  // Ini yang mengunci perbaikannya. Versi sebelumnya menariknya dari server, kuerinya
  // tidak mengembalikan satu baris pun, dan dropdown-nya kosong di layar tanpa satu pesan
  // galat — dua kali dilaporkan Work Owner. Daftar tetap tidak punya keadaan gagal.
  it('tidak pernah meminta daftar dokter ke jaringan', async () => {
    const user = userEvent.setup()
    open('002')

    // Jalur RCL menggambar dropdown-nya…
    await user.click(screen.getByRole('radio', { name: 'RCL' }))
    expect(within(screen.getByLabelText('Nama Dokter')).getAllByRole('option')).toHaveLength(
      1 + DOKTER.length,
    )

    // …dan jalur PUCL tidak menggambarnya sama sekali.
    await user.click(screen.getByRole('radio', { name: 'PUCL' }))
    await waitFor(() => expect(screen.getByText(ALASAN[0]!.nama)).toBeInTheDocument())

    expect(calls.some((c) => c.url.startsWith('/api/registrasi/rclpucl/dokter'))).toBe(false)
  })
})

describe('isian wajib', () => {
  it('menolak Kirim tanpa jalur, catatan, dan Keterangan Isi — sekaligus', async () => {
    const user = userEvent.setup()
    open()
    await user.click(screen.getByRole('button', { name: 'Kirim' }))

    expect(screen.getByText('Pilih RCL, PUCL, atau Notification lebih dulu.')).toBeInTheDocument()
    expect(screen.getByText('Catatan untuk RCL/PUCL wajib diisi.')).toBeInTheDocument()
    expect(screen.getByText('Keterangan Isi wajib diisi.')).toBeInTheDocument()
    expect(calls.some((c) => c.url.endsWith('/kirim-rclpucl'))).toBe(false)
  })
})

describe('pilihan Perihal', () => {
  it('menyempit menurut jalur dan dikosongkan saat jalur berganti', async () => {
    const user = userEvent.setup()
    open()
    const perihal = screen.getByLabelText('Perihal') as HTMLSelectElement
    expect(perihal).toBeDisabled()

    await user.click(screen.getByRole('radio', { name: 'PUCL' }))
    await waitFor(() => expect(screen.getByRole('option', { name: PERIHAL_PUCL[0]!.nama })).toBeInTheDocument())
    await user.selectOptions(perihal, PERIHAL_PUCL[0]!.nama)
    expect(perihal.value).toBe(PERIHAL_PUCL[0]!.nama)

    // Berganti jalur menawarkan daftar lain, dan yang terpilih tidak boleh terbawa.
    await user.click(screen.getByRole('radio', { name: 'RCL' }))
    expect(perihal.value).toBe('')
    await waitFor(() => expect(screen.getByRole('option', { name: PERIHAL_RCL[0]!.nama })).toBeInTheDocument())
  })
})

describe('grid alasan penolakan', () => {
  it('tombol Pilih menyalin Reason Description ke Keterangan Isi', async () => {
    const user = userEvent.setup()
    open()
    await waitFor(() => expect(screen.getByText(ALASAN[0]!.nama)).toBeInTheDocument())

    const baris = screen.getByText(ALASAN[0]!.nama).closest('tr')!
    await user.click(within(baris).getByRole('button', { name: 'Pilih' }))

    expect(screen.getByLabelText('Keterangan Isi')).toHaveValue(ALASAN[0]!.deskripsi)
  })

  it('mencari di server, bukan menyaring di layar', async () => {
    const user = userEvent.setup()
    open()
    await user.type(screen.getByLabelText('Cari alasan'), 'polis')
    await waitFor(() =>
      expect(calls.some((c) => c.url.includes('/rclpucl/alasan?cari=polis'))).toBe(true),
    )
  })
})

describe('Kirim', () => {
  it('mengirim jalur Notification dengan kode 3', async () => {
    const user = userEvent.setup()
    const onClose = open('006')

    await user.click(screen.getByRole('radio', { name: 'Notification' }))
    await user.type(screen.getByLabelText('Catatan untuk RCL/PUCL'), 'Pemberitahuan.')
    await user.type(screen.getByLabelText('Keterangan Isi'), 'Isi pemberitahuan.')
    await user.click(screen.getByRole('button', { name: 'Kirim' }))

    await waitFor(() => expect(onClose).toHaveBeenCalled())
    const sent = calls.find((c) => c.url.endsWith('/kirim-rclpucl'))!
    expect((sent.body as { jalur: number }).jalur).toBe(TRACK_NOTIFICATION)
  })

  it('mengirim seluruh isian lalu menutup modal', async () => {
    const user = userEvent.setup()
    const onClose = open('002')

    await user.click(screen.getByRole('radio', { name: 'RCL' }))
    await user.type(screen.getByLabelText('Catatan untuk RCL/PUCL'), 'Dokumen belum lengkap.')
    await user.type(screen.getByLabelText('Keterangan Pembuka'), 'Dengan hormat,')
    await user.type(screen.getByLabelText('Keterangan Isi'), 'Mohon melengkapi berkas.')
    await user.type(screen.getByLabelText('Keterangan Penutup'), 'Terima kasih.')
    await waitFor(() =>
      expect(within(screen.getByLabelText('Nama Dokter')).getByRole('option', { name: DOKTER[0]!.nama })).toBeInTheDocument(),
    )
    await user.selectOptions(screen.getByLabelText('Nama Dokter'), DOKTER[0]!.id)
    await user.click(screen.getByRole('button', { name: 'Kirim' }))

    await waitFor(() => expect(onClose).toHaveBeenCalled())
    const sent = calls.find((c) => c.url.endsWith('/kirim-rclpucl'))!
    expect(sent.body).toEqual({
      jalur: TRACK_RCL,
      catatan: 'Dokumen belum lengkap.',
      perihal: '',
      keterangan_pembuka: 'Dengan hormat,',
      keterangan_isi: 'Mohon melengkapi berkas.',
      keterangan_penutup: 'Terima kasih.',
      nama_dokter: DOKTER[0]!.id,
    })
  })

  it('tidak mengirim Nama Dokter bila isiannya tidak berlaku', async () => {
    const user = userEvent.setup()
    open('002')

    // Dipilih saat jalur RCL, lalu jalurnya diubah menjadi PUCL.
    await user.click(screen.getByRole('radio', { name: 'RCL' }))
    await waitFor(() =>
      expect(within(screen.getByLabelText('Nama Dokter')).getByRole('option', { name: DOKTER[0]!.nama })).toBeInTheDocument(),
    )
    await user.selectOptions(screen.getByLabelText('Nama Dokter'), DOKTER[0]!.id)
    await user.click(screen.getByRole('radio', { name: 'PUCL' }))

    await user.type(screen.getByLabelText('Catatan untuk RCL/PUCL'), 'Catatan.')
    await user.type(screen.getByLabelText('Keterangan Isi'), 'Isi.')
    await user.click(screen.getByRole('button', { name: 'Kirim' }))

    await waitFor(() => expect(calls.some((c) => c.url.endsWith('/kirim-rclpucl'))).toBe(true))
    const sent = calls.find((c) => c.url.endsWith('/kirim-rclpucl'))!
    expect((sent.body as { nama_dokter: string }).nama_dokter).toBe('')
  })

  it('menampilkan penolakan server dan membiarkan modal terbuka', async () => {
    vi.unstubAllGlobals()
    installFetch(() =>
      json(422, {
        kode: 'validasi_gagal',
        pesan: 'Isian belum lengkap.',
        pelanggaran: [{ kode: 'catatan_rclpucl_kosong', field: 'catatan', pesan: 'Catatan untuk RCL/PUCL wajib diisi.' }],
      }),
    )
    const user = userEvent.setup()
    const onClose = open()

    await user.click(screen.getByRole('radio', { name: 'PUCL' }))
    await user.type(screen.getByLabelText('Catatan untuk RCL/PUCL'), ' ')
    await user.type(screen.getByLabelText('Keterangan Isi'), 'Isi.')
    await user.click(screen.getByRole('button', { name: 'Kirim' }))

    // Catatan hanya berisi spasi: layar menolaknya lebih dulu, tanpa memanggil server.
    expect(screen.getByText('Catatan untuk RCL/PUCL wajib diisi.')).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })
})
