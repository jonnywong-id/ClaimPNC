import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { ReceiveTKATask } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { ReceiveTKAInboxPage } from './ReceiveTKAInboxPage'

/**
 * Seluruh nomor polis, nama tertanggung, nama peserta, dan nomor klaim di berkas ini
 * KARANGAN (`D-69`): data nasabah tidak pernah ditulis ke berkas yang di-commit, dan
 * larangan itu berlaku pada data uji persis seperti pada dokumen.
 */

/** Baris LENGKAP, terdaftar beberapa tahun lalu. */
const LENGKAP: ReceiveTKATask = {
  referensi: 'ASM-FW-GCNMFW-WORK PNC-1546',
  klaim_tersedia: true,
  nomor_klaim: 'PNC-1546',
  nomor_polis: '12000000000002',
  nama_tertanggung: 'PT Contoh Karya Mandiri',
  nama_peserta: 'Peserta Contoh Dua',
  tanggal_kejadian: '2021-08-16',
  tanggal_registrasi: '2023-05-10',
}

/**
 * Baris yang tanggal pendaftarannya TIDAK tercatat.
 *
 * Kolom sumbernya `VARCHAR2` pada basis data, sehingga bentuk yang tidak dikenali diurai
 * menjadi null alih-alih tanggal karangan. Layar menuliskannya sebagai tanda hubung —
 * bukan "0 bulan lalu", yang akan terbaca seolah klaimnya baru masuk.
 */
const TANPA_REGISTRASI: ReceiveTKATask = {
  referensi: 'ASM-FW-GCNMFW-WORK PNC-1955',
  klaim_tersedia: true,
  nomor_klaim: 'PNC-1955',
  nomor_polis: '12200007000005',
  nama_tertanggung: 'PT Contoh Bahari Nusantara',
  nama_peserta: 'Peserta Contoh Tiga',
  tanggal_kejadian: '2026-09-02',
  tanggal_registrasi: null,
}

/**
 * Baris YATIM — pekerjaannya ada di antrean Pega, klaimnya tidak ada di data klaim utama.
 *
 * Ia TETAP TAMPIL: gabungan di server sengaja LEFT, sama seperti Pega yang juga
 * menampilkannya. Yang dimatikan hanyalah isian dan tombolnya, karena penolakannya sudah
 * pasti.
 */
const YATIM: ReceiveTKATask = {
  referensi: 'ASM-FW-GCNMFW-WORK PNC-1977',
  klaim_tersedia: false,
  nomor_klaim: 'PNC-1977',
  nomor_polis: '12200007000006',
  nama_tertanggung: 'PT Contoh Lintas Benua',
  nama_peserta: '',
  tanggal_kejadian: '2026-08-30',
  tanggal_registrasi: '2026-09-01',
}

type Call = { url: string; method: string; body: unknown }

let calls: Call[] = []

type Reply = { body: unknown; status?: number }

function installFetch(map: (call: Call) => Reply) {
  vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
    const call: Call = {
      url,
      method: init?.method ?? 'GET',
      body: init?.body ? JSON.parse(init.body as string) : undefined,
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

function listBody(tugas: ReceiveTKATask[], terpotong = false) {
  return { tugas, terpotong, batas_baris: 500, portal: 'ASM' }
}

/** reply melayani daftar, dan menjawab Submit dengan hasil yang diberikan. */
function reply(
  tugas: ReceiveTKATask[],
  onSubmit?: (call: Call) => Reply,
  terpotong = false,
): (call: Call) => Reply {
  return (call) => {
    if (call.method === 'POST') {
      return (
        onSubmit?.(call) ?? {
          body: {
            nomor_klaim: (call.body as { nomor_klaim: string }).nomor_klaim,
            tanggal_dokumen_lengkap: (call.body as { tanggal_dokumen_lengkap: string })
              .tanggal_dokumen_lengkap,
            pemberitahuan_dicoba: true,
            pemberitahuan_terkirim: true,
            portal: 'ASM',
          },
        }
      )
    }
    return { body: listBody(tugas, terpotong) }
  }
}

function show() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  return render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ReceiveTKAInboxPage />
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

/** dateInput mengambil isian tanggal pada baris sebuah nomor klaim. */
function dateInput(claim: string) {
  return screen.getByLabelText(`Tanggal dokumen lengkap untuk klaim ${claim}`)
}

/** submitButton mengambil tombol Submit pada baris sebuah nomor klaim. */
function submitButton(claim: string) {
  const cell = screen.getByText(claim).closest('tr')
  if (!cell) throw new Error(`baris ${claim} tidak ditemukan`)
  return within(cell as HTMLElement).getByRole('button', { name: /Submit|Menyimpan/ })
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

describe('daftar Inbox Receive TKA', () => {
  it('menampilkan keenam kolom data dari baris yang dijawab server', async () => {
    installFetch(reply([LENGKAP]))
    show()

    const table = await screen.findByRole('table')

    expect(within(table).getByText('PNC-1546')).toBeInTheDocument()
    expect(within(table).getByText('12000000000002')).toBeInTheDocument()
    expect(within(table).getByText('PT Contoh Karya Mandiri')).toBeInTheDocument()
    expect(within(table).getByText('Peserta Contoh Dua')).toBeInTheDocument()
    // Terdaftar 2023; berapa pun hari ini dibaca, keterangannya memuat "tahun".
    expect(within(table).getByText(/tahun/)).toBeInTheDocument()
  })

  /**
   * Judul kolom ketiga dan keempat sempat diragukan karena Report Definition melabelinya
   * TERBALIK. Yang dipakai adalah Section, dikuatkan badan surel
   * `NotificationKelengkapanTKA` yang memberi label "Nama Tertanggung" pada nilai dari
   * `Param.QQName`.
   *
   * Judul kolom kelima "Date Of Loss" berbahasa Inggris mengikuti caption layar lama apa
   * adanya (`D-13`), meski nama field kontraknya `tanggal_kejadian` (`D-80`).
   */
  it('memakai judul kolom persis seperti caption layar lama', async () => {
    installFetch(reply([LENGKAP]))
    show()

    const table = await screen.findByRole('table')
    for (const title of [
      'Nomor Klaim',
      'No Polis',
      'Nama Tertanggung',
      'Nama Peserta',
      'Date Of Loss',
      'Aging',
      'Tanggal Dokumen Lengkap',
    ]) {
      // getAllByText, bukan getByText: `DataTable` menggambar judul DUA KALI — sekali di
      // kepala tabel untuk tampilan meja, dan sekali sebagai label di dalam setiap kartu
      // untuk tampilan telepon. Keduanya memang harus ada.
      expect(within(table).getAllByText(title).length).toBeGreaterThan(0)
    }
  })

  /**
   * # Uji yang mencegah cacat paling halus di layar ini
   *
   * Server mengirim tanggal murni `2026-08-14`. Menguraikannya dengan `new Date()` akan
   * memperlakukannya sebagai tengah malam UTC, dan menampilkannya dalam WIB menghasilkan
   * **15 Agustus** — kelas cacat yang `R-12` catat dan `F-5` dibangun untuk menutupnya.
   */
  it('tidak menggeser tanggal satu hari saat menampilkannya', async () => {
    installFetch(reply([LENGKAP]))
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getByText('16 Agu 2021')).toBeInTheDocument()
    expect(within(table).queryByText('17 Agu 2021')).not.toBeInTheDocument()
  })

  /**
   * Tanggal pendaftaran yang tidak tercatat ditulis sebagai tanda hubung.
   *
   * Bukan "0 bulan lalu": yang satu berarti "tidak diketahui", yang lain berarti "baru
   * masuk". Menyamakan keduanya menyatakan sesuatu yang tidak diketahui sebagai fakta.
   */
  it('menuliskan tanda hubung bila tanggal pendaftaran tidak tercatat', async () => {
    installFetch(reply([TANPA_REGISTRASI]))
    show()

    const table = await screen.findByRole('table')
    expect(within(table).queryByText(/bulan lalu/)).not.toBeInTheDocument()
    expect(within(table).queryByText(/tahun/)).not.toBeInTheDocument()
  })

  it('menyebutkan pemotongan alih-alih memotong diam-diam', async () => {
    installFetch(reply([LENGKAP], undefined, true))
    show()

    expect(await screen.findByText(/lebih panjang/i)).toBeInTheDocument()
    expect(screen.getByText(/500 pekerjaan pertama/i)).toBeInTheDocument()
  })
})

describe('mengisi tanggal kelengkapan dokumen', () => {
  it('mematikan Submit sampai tanggalnya diisi', async () => {
    installFetch(reply([LENGKAP]))
    show()

    await screen.findByRole('table')
    expect(submitButton('PNC-1546')).toBeDisabled()

    fireEvent.change(dateInput('PNC-1546'), { target: { value: '2026-09-24' } })
    expect(submitButton('PNC-1546')).toBeEnabled()
  })

  /**
   * Submit mengirim nomor klaim BARISNYA sendiri, bukan baris mana pun yang lain.
   *
   * Itu mengikuti sistem lama: tombolnya digambar per baris dan mengirim parameter milik
   * barisnya, dan `SubmitTanggalLengkapTKA` menerimanya dalam bentuk tunggal.
   */
  it('mengirim nomor klaim dan tanggal baris yang ditekan', async () => {
    installFetch(reply([LENGKAP, TANPA_REGISTRASI]))
    show()

    await screen.findByRole('table')
    fireEvent.change(dateInput('PNC-1955'), { target: { value: '2026-09-24' } })
    await userEvent.click(submitButton('PNC-1955'))

    const posted = calls.find((call) => call.method === 'POST')
    expect(posted?.url).toBe('/api/inbox/receive-tka/kelengkapan-dokumen')
    expect(posted?.body).toEqual({
      nomor_klaim: 'PNC-1955',
      tanggal_dokumen_lengkap: '2026-09-24',
    })
  })

  it('memberi tahu bahwa tanggalnya tersimpan dan pemberitahuannya terkirim', async () => {
    installFetch(reply([LENGKAP]))
    show()

    await screen.findByRole('table')
    fireEvent.change(dateInput('PNC-1546'), { target: { value: '2026-09-24' } })
    await userEvent.click(submitButton('PNC-1546'))

    // Dicocokkan pada kalimat spanduk keberhasilannya, bukan pada kata "tersimpan" saja:
    // kaki halaman juga memuatnya, pada kalimat yang menerangkan tanggal yang sudah
    // tersimpan tidak dapat diubah.
    expect(
      await screen.findByText(/Tanggal dokumen lengkap untuk klaim/i),
    ).toBeInTheDocument()
    expect(screen.getByText(/Pemberitahuan sudah dikirim/i)).toBeInTheDocument()
  })

  /**
   * # Perbedaan yang disengaja terhadap sistem lama
   *
   * Surel yang gagal TIDAK membatalkan penyimpanan. Sistem lama mengirim surel sebelum
   * `Commit`, sehingga kegagalannya membuang seluruh pekerjaan pengguna.
   *
   * Layar karena itu harus mengatakan DUA hal sekaligus: tersimpan, dan surelnya gagal.
   */
  it('mengatakan tersimpan meski pemberitahuannya gagal dikirim', async () => {
    installFetch(
      reply([LENGKAP], (call) => ({
        body: {
          nomor_klaim: (call.body as { nomor_klaim: string }).nomor_klaim,
          tanggal_dokumen_lengkap: '2026-09-24',
          pemberitahuan_dicoba: true,
          pemberitahuan_terkirim: false,
          portal: 'ASM',
        },
      })),
    )
    show()

    await screen.findByRole('table')
    fireEvent.change(dateInput('PNC-1546'), { target: { value: '2026-09-24' } })
    await userEvent.click(submitButton('PNC-1546'))

    expect(
      await screen.findByText(/Tanggal dokumen lengkap untuk klaim/i),
    ).toBeInTheDocument()
    expect(screen.getByText(/gagal dikirim/i)).toBeInTheDocument()
  })

  /**
   * Pemberitahuan yang memang belum dipasang DIBEDAKAN dari yang gagal dikirim.
   *
   * Menyatukan keduanya akan membuat lingkungan pengembangan yang tidak punya server surel
   * terus-menerus menampilkan peringatan gagal — dan peringatan yang selalu muncul berhenti
   * dibaca.
   */
  it('membedakan pemberitahuan yang belum aktif dari yang gagal', async () => {
    installFetch(
      reply([LENGKAP], () => ({
        body: {
          nomor_klaim: 'PNC-1546',
          tanggal_dokumen_lengkap: '2026-09-24',
          pemberitahuan_dicoba: false,
          pemberitahuan_terkirim: false,
          portal: 'ASM',
        },
      })),
    )
    show()

    await screen.findByRole('table')
    fireEvent.change(dateInput('PNC-1546'), { target: { value: '2026-09-24' } })
    await userEvent.click(submitButton('PNC-1546'))

    expect(await screen.findByText(/belum aktif di lingkungan ini/i)).toBeInTheDocument()
    expect(screen.queryByText(/gagal dikirim/i)).not.toBeInTheDocument()
  })

  /**
   * Klaim yatim ditolak dengan pesan yang menyebut TINDAKAN yang berbeda: menyegarkan
   * daftar tidak akan menolong.
   */
  it('menjelaskan bahwa menyegarkan daftar tidak menolong pada klaim yatim', async () => {
    installFetch(
      reply([LENGKAP], () => ({
        status: 409,
        body: { kode: 'klaim_tidak_ditemukan', pesan: 'Klaim tidak ditemukan.' },
      })),
    )
    show()

    await screen.findByRole('table')
    fireEvent.change(dateInput('PNC-1546'), { target: { value: '2026-09-24' } })
    await userEvent.click(submitButton('PNC-1546'))

    expect(
      await screen.findByText(/Klaim tidak ditemukan pada data klaim utama/i),
    ).toBeInTheDocument()
    expect(screen.getByText(/menyegarkan daftar tidak akan\s+menolong/i)).toBeInTheDocument()
  })

  it('menyuruh menyegarkan daftar bila barisnya sudah diisi orang lain', async () => {
    installFetch(
      reply([LENGKAP], () => ({
        status: 409,
        body: { kode: 'pekerjaan_tidak_ditemukan', pesan: 'Sudah tidak ada.' },
      })),
    )
    show()

    await screen.findByRole('table')
    fireEvent.change(dateInput('PNC-1546'), { target: { value: '2026-09-24' } })
    await userEvent.click(submitButton('PNC-1546'))

    expect(await screen.findByText(/sudah tidak ada di daftar/i)).toBeInTheDocument()
    // "melihat DAFTAR terbaru", bukan "Tekan Refresh" saja: kaki halaman memuat kalimat
    // yang mirip — "Tekan Refresh untuk melihat KEADAAN terbaru" — dan keduanya memang
    // berbeda maksudnya.
    expect(screen.getByText(/melihat daftar terbaru/i)).toBeInTheDocument()
  })
})

describe('baris yang klaimnya tidak ditemukan', () => {
  /**
   * Ia TETAP TAMPIL. Gabungan di server sengaja LEFT: pekerjaan yang hilang tanpa jejak
   * jauh lebih mahal daripada pekerjaan yang tampil berlebih.
   */
  it('tetap ditampilkan, tidak disembunyikan', async () => {
    installFetch(reply([LENGKAP, YATIM]))
    show()

    const table = await screen.findByRole('table')
    expect(within(table).getByText('PNC-1977')).toBeInTheDocument()
  })

  /**
   * Isian dan tombolnya dimatikan, dan alasannya disebut di atas tabel.
   *
   * Penolakannya sudah pasti; membiarkan pengguna menekannya lalu ditolak hanya membuang
   * waktunya.
   */
  it('mematikan isian dan tombolnya, serta menyebut alasannya', async () => {
    installFetch(reply([LENGKAP, YATIM]))
    show()

    await screen.findByRole('table')

    expect(dateInput('PNC-1977')).toBeDisabled()
    expect(submitButton('PNC-1977')).toBeDisabled()
    expect(
      screen.getByText(/tidak ditemukan pada data\s+klaim utama/i),
    ).toBeInTheDocument()
  })
})

describe('portal entitas', () => {
  /**
   * Entitas yang sedang dibuka disebut terang-terangan. Pada layar yang MENULIS ini lebih
   * dari sekadar kerapian: Submit mengubah data klaim entitas itu (`ADR-0030`, `R-20`).
   */
  it('menyebutkan entitas yang sedang dibuka', async () => {
    installFetch(reply([LENGKAP]))
    show()

    await screen.findByRole('table')
    expect(screen.getByText(/Submit mengubah data klaim entitas/i)).toBeInTheDocument()
  })

  it('tidak menembak server sebelum portal dipilih', async () => {
    useSelectedPortal.getState().clear()
    installFetch(reply([LENGKAP]))
    show()

    expect(await screen.findByText(/Portal entitas belum dipilih/i)).toBeInTheDocument()
    expect(calls).toHaveLength(0)
  })
})
