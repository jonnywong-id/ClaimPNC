import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { APIError, NetworkError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { AdvicePanel } from './AdvicePanel'
import { ApprovalPanel } from './ApprovalPanel'
import { ClaimPanel } from './ClaimPanel'
import { isValidationError, messageOf, violationsOf } from './errors'
import type { MasterXOL } from './types'

/** Uji tambahan Inbox XOL: galat per panel, pengurutan, dan pembantu galat. Nilai KARANGAN. */

const PATH = '/api/inbox-xol'

const MASTER: MasterXOL = {
  id: 'XOL-001',
  nama: 'XOL Contoh',
  tahun: '2024',
  kurs: 15500,
  min_limit: 2000,
  min_limit_idr: 31000000,
  tipe: '',
  group_business: '',
  jumlah_group_business: 1,
  status_komite: '1',
  menunggu_komite: false,
  catatan_komite: '',
  pic: 'PIC',
  catatan_pic: '',
}

type Answer = { status?: number; body?: unknown; fail?: boolean }
let calls: string[] = []

function stub(answer: (url: string) => Answer) {
  vi.stubGlobal('fetch', (url: string) => {
    calls.push(url)
    const reply = answer(url)
    if (reply.fail) return Promise.reject(new TypeError('Failed to fetch'))
    return Promise.resolve(
      new Response(JSON.stringify(reply.body ?? {}), {
        status: reply.status ?? 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    )
  })
}

function wrap(children: ReactNode) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={client}>{children}</QueryClientProvider>)
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

describe('errors', () => {
  it('mengambil pesan dari galat API dan jaringan, dan menyembunyikan galat lain', () => {
    expect(messageOf(new APIError('x', 'Pesan server.', 500))).toBe('Pesan server.')
    expect(messageOf(new NetworkError())).toBe('Tidak dapat menghubungi server Claim PNC.')
    expect(messageOf(new Error('SELECT * FROM rahasia'))).toBe(
      'Terjadi kesalahan yang tidak dikenali. Coba lagi beberapa saat lagi.',
    )
  })

  it('mengenali galat validasi saja sebagai pelanggaran isian', () => {
    const validation = new APIError('validasi_gagal', 'x', 422, [{ field: 'tahun', pesan: 'Tahun wajib.' }])
    expect(violationsOf(validation)).toEqual({ tahun: 'Tahun wajib.' })
    expect(isValidationError(validation)).toBe(true)

    const other = new APIError('galat_internal', 'x', 500, [{ field: 'tahun', pesan: 'abaikan' }])
    expect(violationsOf(other)).toEqual({})
    expect(isValidationError(other)).toBe(false)
    expect(isValidationError('bukan galat')).toBe(false)
  })
})

describe('ClaimPanel', () => {
  /**
   * Daftar perjanjian dimuat MODAL, bukan layar belakangnya — grid akumulasi tidak lagi
   * memerlukannya, karena backend yang me-loop seluruh perjanjian.
   */
  it('menyatakan daftar perjanjian yang gagal dimuat, di dalam modalnya', async () => {
    stub((url) => (url === `${PATH}/perjanjian` ? { status: 500, body: { kode: 'x', pesan: 'Master rusak.' } } : {}))
    const user = userEvent.setup()
    wrap(<ClaimPanel />)

    await user.click(screen.getByRole('button', { name: 'INSERT DOL DAN COL' }))
    const modal = await screen.findByRole('dialog', { name: 'Insert DOL dan COL' })

    expect(within(modal).getByText('Daftar perjanjian XOL tidak dapat dimuat')).toBeInTheDocument()
    expect(within(modal).getByText('Master rusak.')).toBeInTheDocument()
  })

  /**
   * Tombol "Simpan" tetap DIGAMBAR dan tetap dapat ditekan, lalu ditolak server dengan
   * sebabnya. Menyembunyikannya akan membuat pengguna melaporkan fitur yang hilang —
   * alasan yang sama yang membuat rute `POST /inbox-xol/dol-col` ada di backend.
   */
  it('mengirim penyimpanan dan menampilkan penolakan server apa adanya', async () => {
    stub((url) => {
      if (url === `${PATH}/perjanjian`) return { body: { perjanjian: [MASTER] } }
      if (url === `${PATH}/dol-col`) {
        return {
          status: 409,
          body: {
            kode: 'aksi_belum_tersedia',
            pesan: 'Aksi ini belum tersedia di aplikasi baru.',
          },
        }
      }
      return {}
    })
    const user = userEvent.setup()
    wrap(<ClaimPanel />)

    await user.click(screen.getByRole('button', { name: 'INSERT DOL DAN COL' }))
    const modal = await screen.findByRole('dialog', { name: 'Insert DOL dan COL' })
    await user.click(within(modal).getByRole('button', { name: 'Simpan' }))

    expect(await screen.findByText('Belum dapat disimpan')).toBeInTheDocument()
    expect(screen.getByText('Aksi ini belum tersedia di aplikasi baru.')).toBeInTheDocument()
    expect(calls.filter((u) => u === `${PATH}/dol-col`)).toHaveLength(1)
  })

  /**
   * Dua hal yang dulu rusak dan diperbaiki bersamaan, diuji dalam satu alur karena
   * keduanya hanya terbukti bila simpannya benar-benar berhasil:
   *
   *  1. Date Of Loss dapat dipilih dari KALENDER. Versi sebelumnya memakai isian teks
   *     polos, sehingga tanggalnya harus diketik lebih dulu — tidak ada kalender yang
   *     dapat dibuka sama sekali.
   *  2. Simpan benar-benar MENYIMPAN. Rute `POST /inbox-xol/dol-col` dulu selalu
   *     menolak dengan 409.
   *
   * Yang diperiksa pada badan permintaannya adalah BENTUK TANGGALNYA. Isian menyimpannya
   * sebagai ISO selama di layar, dan kolom `DOL` menyimpannya sebagai `DD/MM/YYYY`;
   * mengirim ISO ke server berarti baris yang tersimpan tidak pernah terbaca grid mana
   * pun, tanpa satu pun pesan galat.
   */
  it('menyimpan DOL dan COL, dengan tanggal berbentuk DD/MM/YYYY', async () => {
    const bodies: unknown[] = []
    vi.stubGlobal('fetch', (url: string, init?: RequestInit) => {
      calls.push(url)
      if (url === `${PATH}/dol-col`) {
        bodies.push(JSON.parse(String(init?.body)))
        return Promise.resolve(
          new Response(JSON.stringify({ jumlah_baris: 2 }), {
            status: 201,
            headers: { 'Content-Type': 'application/json' },
          }),
        )
      }
      const body =
        url === `${PATH}/perjanjian`
          ? { perjanjian: [MASTER] }
          : url.startsWith(`${PATH}/sebab-kerugian`)
            ? { sebab_kerugian: [{ id: '1', deskripsi: 'Kebakaran' }] }
            : {}
      return Promise.resolve(
        new Response(JSON.stringify(body), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    })

    const user = userEvent.setup()
    wrap(<ClaimPanel />)

    await user.click(screen.getByRole('button', { name: 'INSERT DOL DAN COL' }))
    const modal = await screen.findByRole('dialog', { name: 'Insert DOL dan COL' })

    // Tombol kalendernya ada — inilah yang hilang di versi sebelumnya.
    expect(
      within(modal).getByRole('button', { name: 'Pilih Date Of Loss dari kalender' }),
    ).toBeInTheDocument()

    await user.type(within(modal).getByLabelText('Date Of Loss'), '15032024')
    await user.selectOptions(within(modal).getByLabelText('Cause Of Loss'), 'Kebakaran')
    await user.click(await within(modal).findByRole('button', { name: /Pilih perjanjian/ }))
    await user.click(within(modal).getByRole('button', { name: 'Simpan' }))

    await waitFor(() => expect(bodies).toHaveLength(1))
    expect(bodies[0]).toEqual({
      id_master: 'XOL-001',
      tanggal_kejadian: '15/03/2024',
      sebab_kerugian: 'Kebakaran',
    })

    // Jumlah barisnya disebut, bukan sekadar "tersimpan": satu simpan menulis satu baris
    // per Group Business perjanjian.
    expect(await screen.findByText(/2 baris ditulis/)).toBeInTheDocument()
  })

  it('menyatakan akumulasi yang gagal dimuat', async () => {
    stub((url) => {
      if (url === `${PATH}/perjanjian`) return { body: { perjanjian: [MASTER] } }
      return { fail: true }
    })
    wrap(<ClaimPanel />)

    expect(await screen.findByText('Akumulasi klaim tidak dapat dimuat')).toBeInTheDocument()
    // Keterangan kurs TIDAK menyebut angka: tiap baris dibagi kurs perjanjiannya sendiri.
    expect(
      screen.getByText(/masing-masing dibagi kursnya sendiri/),
    ).toBeInTheDocument()
  })

  it('mengurutkan akumulasi dan rincian, serta menyatakan rincian yang gagal dimuat', async () => {
    let breakdownFails = false
    stub((url) => {
      if (url === `${PATH}/perjanjian`) return { body: { perjanjian: [MASTER] } }
      if (url.startsWith(`${PATH}/klaim/rincian`)) {
        return breakdownFails
          ? { status: 500, body: { kode: 'x', pesan: 'Rincian rusak.' } }
          : {
              body: {
                baris: [
                  { group_business: 'Marine', kode_group_business: '20', jumlah_klaim: 3, nilai_outstanding: 9, nilai_akseptasi: 1, sumber: 'bisnis', kurs_tidak_tersedia: false },
                  { group_business: 'Fire', kode_group_business: '10', jumlah_klaim: 1, nilai_outstanding: 2, nilai_akseptasi: 5, sumber: 'bisnis', kurs_tidak_tersedia: false },
                ],
              },
            }
      }
      if (url.startsWith(`${PATH}/klaim`)) {
        return {
          body: {
            perjanjian: { ...MASTER, tipe: 'PROPERTY' },
            baris: [
              { id_master: 'XOL-001', tanggal_kejadian: '2024', sebab_kerugian: 'GEMPA', group_business: 'B', nilai_outstanding: 1, nilai_akseptasi: 1 },
              { id_master: 'XOL-001', tanggal_kejadian: '2023', sebab_kerugian: 'BANJIR', group_business: 'A', nilai_outstanding: 2, nilai_akseptasi: 2 },
            ],
          },
        }
      }
      return {}
    })
    const user = userEvent.setup()
    wrap(<ClaimPanel />)

    // Tabel ditunjuk lewat JUDULNYA, bukan posisinya: layar rincian memuat tiga tabel —
    // akumulasi, rincian, dan grid Master tahun XOL — dan "yang terakhir" berpindah
    // setiap kali susunannya berubah.
    //
    // Judulnya muncul lebih dulu daripada tabelnya — DataTable menggambar kepala
    // sectionnya sejak keadaan memuat. Karena itu tabelnya DITUNGGU, bukan diambil
    // begitu judulnya ada.
    const tabel = async (judul: RegExp) => {
      const section = (await screen.findByText(judul)).closest('section')!
      await waitFor(() => expect(section.querySelector('table')).not.toBeNull())
      return section.querySelector('table')!
    }

    const table = await tabel(/^DATA XOL BASED ON DOL AND COL$/)

    // Baris data diambil lewat DOM, bukan `getAllByRole('row')`: baris yang dapat dibuka
    // diberi `role="button"` oleh DataTable, dan peran itu MENGGANTIKAN peran `row`
    // bawaannya.
    const baris = () => [...table.querySelectorAll('tbody tr')]
    const first = () => baris()[0]?.textContent ?? ''
    await user.click(within(table).getByRole('button', { name: 'Date Of Loss' }))
    expect(first()).toContain('2023')
    await user.click(within(table).getByRole('button', { name: 'Cause Of Loss' }))
    expect(first()).toContain('BANJIR')
    await user.click(within(table).getByRole('button', { name: 'Group Business' }))
    expect(first()).toContain('A')
    await user.click(within(table).getByRole('button', { name: 'OS Value' }))
    await user.click(within(table).getByRole('button', { name: 'Accepted Value' }))
    expect(first()).toContain('GEMPA')

    // Rincian dibuka dengan mengeklik BARISNYA.
    await user.click(baris()[0]!)
    // Grid rincian TIDAK punya judul (seperti layar lama), jadi ia ditunjuk lewat kepala
    // kolom khasnya.
    //
    // Diambil yang TERAKHIR, bukan yang pertama: isi baris yang terbentang berada di
    // dalam `<td>` tabel luarnya, sehingga tabel luar pun "mengandung" teks itu.
    await screen.findAllByText('Number of Claim')
    const detail = [...document.querySelectorAll('table')]
      .filter((t) => t.textContent?.includes('Number of Claim'))
      .at(-1)!
    await within(detail).findByText('Marine')
    const firstDetail = () => within(detail).getAllByRole('row')[1]?.textContent ?? ''
    await user.click(within(detail).getByRole('button', { name: 'Group Business' }))
    expect(firstDetail()).toContain('Fire')
    await user.click(within(detail).getByRole('button', { name: 'Number of Claim' }))
    expect(firstDetail()).toContain('Fire')
    await user.click(within(detail).getByRole('button', { name: 'OS Value (USD)' }))
    expect(firstDetail()).toContain('Fire')
    await user.click(within(detail).getByRole('button', { name: 'Accept Value (USD)' }))
    expect(firstDetail()).toContain('Marine')

    // Mengeklik baris yang sama sekali lagi menutup rinciannya.
    await user.click(baris()[0]!)
    expect(screen.queryAllByText('Number of Claim')).toHaveLength(0)

    breakdownFails = true
    await user.click(baris()[1]!)
    expect(await screen.findByText('Rincian tidak dapat dimuat')).toBeInTheDocument()
    expect(screen.getByText('Rincian rusak.')).toBeInTheDocument()
  })

  it('menaruh Export To Excel pada setiap baris rincian, dengan group business barisnya', async () => {
    stub((url) => {
      if (url === `${PATH}/perjanjian`) return { body: { perjanjian: [MASTER] } }
      if (url.startsWith(`${PATH}/klaim/rincian`)) {
        return {
          body: {
            baris: [
              { group_business: 'Marine', kode_group_business: '20', jumlah_klaim: 3, nilai_outstanding: 9, nilai_akseptasi: 1, sumber: 'bisnis', kurs_tidak_tersedia: false },
              { group_business: 'Treaty Inward', kode_group_business: '', jumlah_klaim: 1, nilai_outstanding: 2, nilai_akseptasi: 5, sumber: 'treaty', kurs_tidak_tersedia: false },
            ],
          },
        }
      }
      if (url.startsWith(`${PATH}/klaim`)) {
        return {
          body: {
            perjanjian: MASTER,
            baris: [
              { id_master: 'XOL-001', tanggal_kejadian: '12/03/2024', sebab_kerugian: 'BANJIR', group_business: 'A', nilai_outstanding: 1, nilai_akseptasi: 1 },
            ],
          },
        }
      }
      return {}
    })
    const user = userEvent.setup()
    wrap(<ClaimPanel />)

    const section = (await screen.findByText(/^DATA XOL BASED ON DOL AND COL$/)).closest('section')!
    await waitFor(() => expect(section.querySelector('tbody tr')).not.toBeNull())
    await user.click(section.querySelectorAll('tbody tr')[0]!)

    // Tombol, bukan tautan: sesi dikirim sebagai header `Authorization`, dan peramban
    // tidak mengirim header apa pun pada navigasi biasa — tautan polos dijawab
    // `sesi_tidak_sah`, bukan berkas.
    const tombol = await screen.findAllByRole('button', { name: 'Export To Excel' })
    // Satu tombol per baris rincian — di layar lama pun tombolnya berada DI DALAM baris,
    // dengan parameter diambil dari properti baris itu.
    expect(tombol).toHaveLength(2)

    await user.click(tombol[0]!)
    const baris1 = new URL(
      calls.find((u) => u.includes('/klaim/rincian/unduh'))!,
      'http://x',
    ).searchParams
    expect(baris1.get('tanggal_kejadian')).toBe('12/03/2024')
    expect(baris1.get('sebab_kerugian')).toBe('BANJIR')
    expect(baris1.get('kode_group_business')).toBe('20')
    expect(baris1.get('nama_group_business')).toBe('Marine')

    // Baris treaty inward tidak punya kode group business; ia memilih cabang kueri
    // tersendiri, persis seperti `Param.grpbzid` di sistem lama.
    await user.click(tombol[1]!)
    const unduhan = calls.filter((u) => u.includes('/klaim/rincian/unduh'))
    const baris2 = new URL(unduhan[unduhan.length - 1]!, 'http://x').searchParams
    expect(baris2.get('kode_group_business')).toBe('treaty')
  })

  /**
   * Keempat kendali unggahan digambar, dan hanya "Upload Inward" yang masih ditolak.
   *
   * "Upload MBU Salvage" berjalan penuh — seluruh rantai rule-nya ada di export.
   * "Upload Inward" belum: activity `ConvertDataCsvInwardToPage` tidak ada, sehingga
   * pemetaan kolomnya tidak dapat dibaca siapa pun.
   */
  it('mengunggah MBU Salvage dan melaporkan baris yang ditolak', async () => {
    stub((url) => {
      if (url === `${PATH}/perjanjian`) return { body: { perjanjian: [MASTER] } }
      if (url.startsWith(`${PATH}/klaim/rincian`)) {
        return {
          body: {
            baris: [
              { group_business: 'Marine', kode_group_business: '20', jumlah_klaim: 3, nilai_outstanding: 9, nilai_akseptasi: 1, sumber: 'bisnis', kurs_tidak_tersedia: false },
            ],
          },
        }
      }
      if (url === `${PATH}/unggah/mbu-salvage`) {
        return {
          body: {
            jumlah_baris: 3,
            jumlah_tersimpan: 2,
            ditolak: [{ baris: 4, no_klaim: 'PNC-3', alasan: 'Cause Of Loss kosong.' }],
          },
        }
      }
      if (url.startsWith(`${PATH}/klaim`)) {
        return {
          body: {
            perjanjian: MASTER,
            baris: [
              { id_master: 'XOL-001', tanggal_kejadian: '12/03/2024', sebab_kerugian: 'BANJIR', group_business: 'A', nilai_outstanding: 1, nilai_akseptasi: 1 },
            ],
          },
        }
      }
      return {}
    })
    const user = userEvent.setup()
    wrap(<ClaimPanel />)

    const section = (await screen.findByText(/^DATA XOL BASED ON DOL AND COL$/)).closest('section')!
    await waitFor(() => expect(section.querySelector('tbody tr')).not.toBeNull())
    await user.click(section.querySelectorAll('tbody tr')[0]!)

    const berkas = new File(
      ['ClaimNo;DateOfLoss;Currency;SalvageOS;SalvageAksep;CauseOfLoss\n'],
      'salvage.csv',
      { type: 'text/csv' },
    )
    await user.upload(await screen.findByLabelText('Berkas MBU Salvage'), berkas)

    expect(await screen.findByText('2 dari 3 baris tersimpan.')).toBeInTheDocument()
    // Nomor barisnya disebut — tanpa itu, berkas ratusan baris tidak dapat diperbaiki.
    expect(screen.getByText(/Baris 4 \(PNC-3\): Cause Of Loss kosong\./)).toBeInTheDocument()
    expect(calls.filter((u) => u === `${PATH}/unggah/mbu-salvage`)).toHaveLength(1)
  })

  it('menggambar tautan Format dan menampilkan penolakan server pada Upload Inward', async () => {
    stub((url) => {
      if (url === `${PATH}/perjanjian`) return { body: { perjanjian: [MASTER] } }
      if (url.startsWith(`${PATH}/klaim/rincian`)) {
        return {
          body: {
            baris: [
              { group_business: 'Marine', kode_group_business: '20', jumlah_klaim: 3, nilai_outstanding: 9, nilai_akseptasi: 1, sumber: 'bisnis', kurs_tidak_tersedia: false },
            ],
          },
        }
      }
      if (url.startsWith(`${PATH}/unggah/`)) {
        return {
          status: 409,
          body: { kode: 'aksi_belum_tersedia', pesan: 'Aksi ini belum tersedia di aplikasi baru.' },
        }
      }
      if (url.startsWith(`${PATH}/klaim`)) {
        return {
          body: {
            perjanjian: MASTER,
            baris: [
              { id_master: 'XOL-001', tanggal_kejadian: '12/03/2024', sebab_kerugian: 'BANJIR', group_business: 'A', nilai_outstanding: 1, nilai_akseptasi: 1 },
            ],
          },
        }
      }
      return {}
    })
    const user = userEvent.setup()
    wrap(<ClaimPanel />)

    const section = (await screen.findByText(/^DATA XOL BASED ON DOL AND COL$/)).closest('section')!
    await waitFor(() => expect(section.querySelector('tbody tr')).not.toBeNull())
    await user.click(section.querySelectorAll('tbody tr')[0]!)

    // Berkas contoh dapat diambil — `jenis` persis `Param.jenis` tombol lama.
    //
    // Keduanya TOMBOL, bukan tautan: sesi dikirim sebagai header `Authorization`, dan
    // tautan polos karena itu dijawab `sesi_tidak_sah`, bukan berkas.
    await user.click(await screen.findByRole('button', { name: 'Format MBU Salvage' }))
    expect(calls).toContain(`${PATH}/format-unggah?jenis=1`)

    await user.click(screen.getByRole('button', { name: 'Format Inward' }))
    expect(calls).toContain(`${PATH}/format-unggah?jenis=2`)

    await user.click(screen.getByRole('button', { name: 'Upload Inward' }))
    expect(await screen.findByText('Upload Inward belum tersedia')).toBeInTheDocument()
    expect(screen.getByText('Aksi ini belum tersedia di aplikasi baru.')).toBeInTheDocument()
    expect(calls.filter((u) => u === `${PATH}/unggah/inward`)).toHaveLength(1)

    // Berkasnya TIDAK ikut dikirim — tidak ada pemilih berkas untuk Inward.
    expect(screen.queryByLabelText('Berkas Inward')).not.toBeInTheDocument()
  })

  it('menggambar Generate PLA dan Generate DLA, dan menyatakan penolakan server', async () => {
    stub((url) => {
      if (url === `${PATH}/perjanjian`) return { body: { perjanjian: [MASTER] } }
      if (url.startsWith(`${PATH}/klaim/rincian`)) {
        return {
          body: {
            baris: [
              { group_business: 'Marine', kode_group_business: '20', jumlah_klaim: 3, nilai_outstanding: 9, nilai_akseptasi: 1, sumber: 'bisnis', kurs_tidak_tersedia: false },
            ],
          },
        }
      }
      if (url.startsWith(`${PATH}/generate/`)) {
        return {
          status: 409,
          body: { kode: 'aksi_belum_tersedia', pesan: 'Aksi ini belum tersedia di aplikasi baru.' },
        }
      }
      if (url.startsWith(`${PATH}/klaim`)) {
        return {
          body: {
            perjanjian: MASTER,
            baris: [
              { id_master: 'XOL-001', tanggal_kejadian: '12/03/2024', sebab_kerugian: 'BANJIR', group_business: 'A', nilai_outstanding: 1, nilai_akseptasi: 1 },
            ],
          },
        }
      }
      return {}
    })
    const user = userEvent.setup()
    wrap(<ClaimPanel />)

    const section = (await screen.findByText(/^DATA XOL BASED ON DOL AND COL$/)).closest('section')!
    await waitFor(() => expect(section.querySelector('tbody tr')).not.toBeNull())
    await user.click(section.querySelectorAll('tbody tr')[0]!)

    await user.click(await screen.findByRole('button', { name: 'Generate PLA' }))
    expect(await screen.findByText('Generate PLA belum tersedia')).toBeInTheDocument()
    expect(calls.filter((u) => u === `${PATH}/generate/pla`)).toHaveLength(1)

    await user.click(screen.getByRole('button', { name: 'Generate DLA' }))
    expect(await screen.findByText('Generate DLA belum tersedia')).toBeInTheDocument()
    expect(calls.filter((u) => u === `${PATH}/generate/dla`)).toHaveLength(1)
  })
})

describe('AdvicePanel', () => {
  it('menyatakan penyebab kerugian yang gagal dimuat dan pencarian yang gagal', async () => {
    stub((url) => {
      if (url === `${PATH}/sebab-kerugian`) return { status: 500, body: { kode: 'x', pesan: 'Sebab rusak.' } }
      if (url.startsWith(`${PATH}/pla-dla`)) return { status: 500, body: { kode: 'galat_internal', pesan: 'Pencarian rusak.' } }
      return {}
    })
    const user = userEvent.setup()
    wrap(<AdvicePanel />)

    expect(await screen.findByText('Daftar penyebab kerugian tidak dapat dimuat')).toBeInTheDocument()
    await user.type(screen.getByLabelText('Date Of Loss'), '2024')
    await user.click(screen.getByRole('checkbox', { name: 'DLA' }))
    await user.click(screen.getByRole('button', { name: 'Cari Data' }))

    expect(await screen.findByText('Pencarian tidak dapat dijalankan')).toBeInTheDocument()
    expect(screen.getByText('Pencarian rusak.')).toBeInTheDocument()
  })

  it('mengurutkan hasil dan menandai yang masih menunggu komite', async () => {
    const advice = (over: Record<string, unknown>) => ({
      nomor: 'PLA/1 / 0',
      nomor_asli: 'PLA/1',
      revisi: '0',
      nama_insurance: 'Reas B',
      nama_layer: 'Layer 2',
      tahun: '2024',
      sebab_kerugian: 'BANJIR',
      kurs: 15500,
      share_percent: 10,
      email: 'b@contoh.invalid',
      remark: 'R2',
      status_persetujuan: '0',
      sudah_disetujui: false,
      catatan_persetujuan: '',
      catatan_pic: '',
      batas_layer: '',
      negara: '',
      tanggal_terbit: '',
      tipe: 'PLA',
      ...over,
    })
    stub((url) => {
      if (url === `${PATH}/sebab-kerugian`) return { body: { sebab_kerugian: [{ id: '1', deskripsi: 'BANJIR' }] } }
      if (url.startsWith(`${PATH}/pla-dla`)) {
        return {
          body: {
            pemberitahuan: [
              advice({}),
              advice({ nomor: 'PLA/0 / 0', nomor_asli: 'PLA/0', nama_insurance: 'Reas A', nama_layer: 'Layer 1', kurs: 100, share_percent: 5, email: 'a@contoh.invalid', remark: 'R1', tahun: '2023' }),
            ],
          },
        }
      }
      return {}
    })
    const user = userEvent.setup()
    wrap(<AdvicePanel />)

    await screen.findByRole('option', { name: 'BANJIR' })
    await user.type(screen.getByLabelText('Date Of Loss'), '2024')
    await user.selectOptions(screen.getByLabelText('Cause Of Loss'), 'BANJIR')
    await user.click(screen.getByRole('checkbox', { name: 'PLA' }))
    await user.click(screen.getByRole('button', { name: 'Cari Data' }))

    const table = await screen.findByRole('table')
    expect(within(table).getAllByText('Menunggu komite')).toHaveLength(2)
    const first = () => within(table).getAllByRole('row')[1]?.textContent ?? ''
    for (const title of ['NO PLA / DLA', 'Nama Insurance', 'Nama Layer', 'Tahun', 'Kurs (IDR)', 'Share Percent', 'Email', 'Remark']) {
      await user.click(within(table).getByRole('button', { name: title }))
      expect(first()).toContain('Reas A')
    }
    expect(screen.getByRole('button', { name: 'Unduh perhitungan (CSV)' })).toBeInTheDocument()
  })
})

describe('ApprovalPanel', () => {
  it('menyatakan antrean persetujuan yang gagal dimuat', async () => {
    stub(() => ({ fail: true }))
    wrap(<ApprovalPanel active />)

    expect((await screen.findAllByText('Antrean persetujuan tidak dapat dimuat')).length).toBeGreaterThan(0)
  })

  it('mengurutkan perjanjian menurut kurs', async () => {
    stub(() => ({
      body: {
        pemberitahuan: [],
        perjanjian: [
          { ...MASTER, id: 'XOL-9', kurs: 20000, tahun: '2025' },
          { ...MASTER, id: 'XOL-8', kurs: 10000, tahun: '2024' },
        ],
      },
    }))
    const user = userEvent.setup()
    wrap(<ApprovalPanel active />)

    await screen.findByText('XOL-9')
    const kurs = screen.getByRole('button', { name: 'Kurs Value' })
    const table = kurs.closest('table') as HTMLElement
    await user.click(kurs)
    await waitFor(() => expect(within(table).getAllByRole('row')[1]).toHaveTextContent('XOL-8'))
  })
})
