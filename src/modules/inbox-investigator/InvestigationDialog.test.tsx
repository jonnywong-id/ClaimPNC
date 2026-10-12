import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import type { InvestigatorTask } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { useSession } from '@/app/session'

import { InvestigationDialog } from './InvestigationDialog'

/**
 * Uji formulir kerja Investigator — padanan Flow Action `InputInvestigator`.
 *
 * Seluruh nomor, nama, dan alamat di berkas ini KARANGAN (`D-69`).
 */

const TUGAS: InvestigatorTask = {
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
  lini_bisnis: '002',
}

type Call = { url: string; method: string; body: unknown }

let calls: Call[] = []

/** blankForm adalah jawaban formulir yang belum pernah diisi. */
function blankForm(override: Record<string, unknown> = {}) {
  return {
    investigasi: {
      referensi: TUGAS.referensi,
      urutan_survei: 1,
      urutan: 1,
      tanggal_investigasi: '2026-10-05T07:30:00Z',
      dapat_diinvestigasi: '',
      tempat_kejadian: '',
      nama_rumah_sakit: '',
      nama_tempat_lainnya: '',
      alamat_rs_klinik: '',
      nomor_rekam_medik: '',
      nama_pasien: '',
      tanggal_lahir: null,
      verifikasi_tanggal_lahir: '',
      keterangan_tanggal_lahir: '',
      peserta_terdaftar: '',
      keterangan_pendaftaran: '',
      tanggal_perawatan: null,
      tanggal_selesai_perawatan: null,
      total_pengajuan: '',
      tagihan_lunas: '',
      bayar_pasien: 'false',
      bayar_perusahaan: 'false',
      bayar_asuransi_lain: 'false',
      tidak_ada_pembayaran: 'false',
      nama_asuransi_lain: '',
      konfirmasi_kwitansi: '',
      nama_pic_rs: '',
      nama_penelepon: '',
      nama_karyawan: '',
      kode_area_telepon: '',
      nomor_telepon: '',
      ekstensi_telepon: '',
      hasil_investigasi: '',
      ...override,
    },
    tampil: {
      nama_rumah_sakit: false,
      nama_tempat_lainnya: true,
      alamat_rs_klinik: false,
      tanggal_selesai_perawatan: true,
      nama_asuransi_lain: false,
    },
    portal: 'ASM',
  }
}

/** submitReply adalah jawaban simpan yang berhasil. */
const submitReply = {
  pindah: {
    referensi: TUGAS.referensi,
    status_survei: '5',
    status_pnc: '5',
    status_klaim: '1151',
    pada: '2026-10-05T07:30:00Z',
  },
}

/** Jawaban tab Unggah Dokumen. Satu kategori, satu baris — cukup untuk membuktikan
    panelnya tergambar. Seluruh isinya KARANGAN (`D-69`). */
const DOKUMEN = {
  kategori: [
    {
      kode: 'PENDAFTARAN',
      nama: 'PENDAFTARAN',
      dokumen: [
        {
          id: 'D1',
          jenis: 'J1',
          nama: 'Formulir Klaim',
          wajib: true,
          minimal: 1,
          terunggah: 0,
          objek: '',
        },
      ],
    },
  ],
  berkas: [],
}

function installFetch(onSubmit: { body: unknown; status?: number } = { body: submitReply }) {
  vi.stubGlobal(
    'fetch',
    vi.fn((url: string, init?: RequestInit) => {
      const method = init?.method ?? 'GET'
      calls.push({
        url: String(url),
        method,
        body: init?.body ? JSON.parse(String(init.body)) : null,
      })

      if (method === 'POST') {
        return Promise.resolve(
          new Response(JSON.stringify(onSubmit.body), {
            status: onSubmit.status ?? 200,
            headers: { 'Content-Type': 'application/json' },
          }),
        )
      }
      // Tab Unggah Dokumen menembak endpoint Registrasi, bukan endpoint modul ini.
      // Jawabannya berbentuk lain, dan tanpa percabangan ini panel itu menerima badan
      // formulir investigasi — lalu gagal tanpa alasan yang jelas.
      if (String(url).includes('/dokumen')) {
        return Promise.resolve(
          new Response(JSON.stringify(DOKUMEN), {
            status: 200,
            headers: { 'Content-Type': 'application/json' },
          }),
        )
      }
      return Promise.resolve(
        new Response(JSON.stringify(blankForm()), {
          status: 200,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
    }),
  )
}

function show(onClose = vi.fn()) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  })
  render(
    <QueryClientProvider client={client}>
      <InvestigationDialog task={TUGAS} onClose={onClose} />
    </QueryClientProvider>,
  )
  return onClose
}

beforeEach(() => {
  calls = []
  useSession.setState({ token: 'token-uji', user: null, validUntil: '2026-12-31T00:00:00Z' })
  useSelectedPortal.getState().select('ASM')
})

afterEach(() => {
  vi.unstubAllGlobals()
  vi.restoreAllMocks()
})

describe('formulir kerja Investigator', () => {
  /** Klaim yang sedang dikerjakan disebut, supaya penyelidik tahu formulir ini milik siapa. */
  it('menyebut klaim yang sedang dikerjakan', async () => {
    installFetch()
    show()

    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByText(TUGAS.nomor_case)).toBeInTheDocument()
    expect(within(dialog).getByText(TUGAS.nama_tertanggung)).toBeInTheDocument()
    expect(within(dialog).getByText(TUGAS.nama_peserta)).toBeInTheDocument()
  })

  /**
   * Akibat menekan Simpan disebutkan SEBELUM tombolnya ditekan.
   *
   * Pekerjaannya hilang dari antrean setelah itu; tanpa keterangan ini ia terbaca seperti
   * data yang lenyap.
   */
  it('menyatakan klaim akan berpindah ke Analyst', async () => {
    installFetch()
    show()

    await screen.findByLabelText('Dapat Diinvestigasi')
    expect(screen.getByText(/berpindah|memindahkan/i)).toBeInTheDocument()
    expect(screen.getByText(/analyst/i)).toBeInTheDocument()
  })

  /**
   * "Tempat Kejadian" mengendalikan tiga isian lain, dan perubahannya terasa SEKETIKA.
   *
   * Keenam syaratnya dibaca dari `pyCondition` section lama. Pengguna yang mengubah
   * pilihannya lalu harus menyimpan dulu untuk melihat isiannya berganti akan menyimpulkan
   * layarnya rusak.
   */
  it('mengganti isian mengikuti pilihan Tempat Kejadian', async () => {
    installFetch()
    show()
    await screen.findByLabelText('Dapat Diinvestigasi')

    // Bawaannya kosong → bukan rumah sakit.
    expect(screen.getByLabelText('Nama Tempat Lainnya')).toBeInTheDocument()
    expect(screen.queryByLabelText('Nama Rumah Sakit')).not.toBeInTheDocument()
    expect(
      screen.queryByLabelText('Alamat RS/Klinik/Lokasi Kejadian *'),
    ).not.toBeInTheDocument()

    await userEvent.selectOptions(screen.getByLabelText('Tempat Kejadian'), '1')

    expect(screen.getByLabelText('Nama Rumah Sakit')).toBeInTheDocument()
    expect(screen.getByLabelText('Alamat RS/Klinik/Lokasi Kejadian *')).toBeInTheDocument()
    expect(screen.queryByLabelText('Nama Tempat Lainnya')).not.toBeInTheDocument()
  })

  /**
   * "Nama Asuransi/Perusahaan Lain" muncul hanya bila pembayarnya perusahaan atau asuransi
   * lain.
   */
  it('menampilkan Nama Asuransi Lain mengikuti kotak pembayaran', async () => {
    installFetch()
    show()
    await screen.findByLabelText('Dapat Diinvestigasi')

    expect(
      screen.queryByLabelText('Nama Asuransi/Perusahaan Lain'),
    ).not.toBeInTheDocument()

    await userEvent.click(screen.getByLabelText('Asuransi Lain'))

    expect(screen.getByLabelText('Nama Asuransi/Perusahaan Lain')).toBeInTheDocument()
  })

  /** Simpan mengirim isian ke jalur formulir, dan menutup dialognya. */
  it('menyimpan isian lalu menutup dialog', async () => {
    installFetch()
    const onClose = show()
    await screen.findByLabelText('Dapat Diinvestigasi')

    await userEvent.selectOptions(screen.getByLabelText('Dapat Diinvestigasi'), '1')
    await userEvent.type(screen.getByLabelText('Hasil Investigasi'), 'Keterangan contoh.')
    await userEvent.click(screen.getByRole('button', { name: /^submit$/i }))

    await vi.waitFor(() => expect(onClose).toHaveBeenCalled())

    const posted = calls.find((one) => one.method === 'POST')
    expect(posted).toBeDefined()
    expect(posted!.url).toContain('/investigasi')

    const body = posted!.body as Record<string, string>
    expect(body['dapat_diinvestigasi']).toBe('1')
    expect(body['hasil_investigasi']).toBe('Keterangan contoh.')
  })

  /**
   * Kotak centang dikirim sebagai TEKS `"true"`/`"false"`, bukan sebagai boolean.
   *
   * Itu nilai yang benar-benar tersimpan, dan berkas Export Data Investigation
   * menuliskannya apa adanya (`P-5`).
   */
  it('mengirim kotak centang sebagai teks true/false', async () => {
    installFetch()
    show()
    await screen.findByLabelText('Dapat Diinvestigasi')

    await userEvent.selectOptions(screen.getByLabelText('Dapat Diinvestigasi'), '1')
    await userEvent.click(screen.getByLabelText('Pasien'))
    await userEvent.click(screen.getByRole('button', { name: /^submit$/i }))

    await vi.waitFor(() => expect(calls.some((one) => one.method === 'POST')).toBe(true))

    const body = calls.find((one) => one.method === 'POST')!.body as Record<string, string>
    expect(body['bayar_pasien']).toBe('true')
    expect(body['bayar_perusahaan']).toBe('false')
  })

  /** Penolakan server ditampilkan, dan dialognya TIDAK ditutup. */
  it('menampilkan penolakan server tanpa menutup dialog', async () => {
    installFetch({
      body: { kode: 'permintaan_cacat', pesan: 'Pilihan "Dapat Diinvestigasi" wajib diisi.' },
      status: 400,
    })
    const onClose = show()
    await screen.findByLabelText('Dapat Diinvestigasi')

    await userEvent.click(screen.getByRole('button', { name: /^submit$/i }))

    // Dicocokkan pada KALIMAT penolakannya, bukan pada nama isiannya: nama itu juga
    // menjadi label isian di formulir, sehingga pencocokan longgar menemukan keduanya.
    expect(
      await screen.findByText('Pilihan "Dapat Diinvestigasi" wajib diisi.'),
    ).toBeInTheDocument()
    expect(onClose).not.toHaveBeenCalled()
  })

  /**
   * Pilihan keenam isian radio DISALIN dari rule Property-nya, bukan ditebak.
   *
   * Empat dari enam sempat ditebak "Ya"/"Tidak" dan salah. Uji ini yang gagal bila
   * tebakan itu kembali — dan dua di antaranya bukan sekadar beda kata:
   *
   *   FlagDOB                  menanyakan KESESUAIAN, bukan ya/tidak
   *   KonfirmasiModelKwitansi  punya TIGA pilihan; yang ketiga tidak akan pernah dapat
   *                            dipilih pengguna bila ditebak dua
   */
  it('memakai pilihan dari rule Property, bukan ya/tidak', async () => {
    installFetch()
    show()
    await screen.findByLabelText('Dapat Diinvestigasi')

    const optionsOf = (label: string) =>
      Array.from(screen.getByLabelText(label).querySelectorAll('option'))
        .map((one) => one.textContent)
        .filter((one) => one && !one.startsWith('—'))

    expect(optionsOf('Dapat Diinvestigasi')).toEqual(['Ya', 'Tidak'])
    expect(optionsOf('Tempat Kejadian')).toEqual(['Rumah Sakit', 'Non Rumah Sakit'])
    expect(optionsOf('Peserta Terdaftar Di RS')).toEqual(['YA', 'TIDAK'])
    expect(optionsOf('Verifikasi Tanggal Lahir')).toEqual(['Sesuai', 'Tidak Sesuai'])
    expect(optionsOf('Tagihan Lunas/Belum Lunas')).toEqual(['LUNAS', 'BELUM LUNAS'])

    // TIGA, bukan dua.
    expect(optionsOf('Konfirmasi Model Kwitansi')).toEqual([
      'SESUAI',
      'TIDAK SESUAI',
      'TIDAK ADA',
    ])
  })

  /**
   * Ketiga puluh isian formulir ada di layar.
   *
   * Tiga di antaranya sempat hilang — Keterangan Tambahan, Nama Karyawan,
   * dan Tagihan Lunas/Belum Lunas — dan ketiganya tidak bergejala: layarnya tampil rapi,
   * tipe bersih, uji lulus. Yang menemukannya adalah membaca ulang section Pega-nya.
   */
  it('menggambar isian yang sempat terlewat', async () => {
    installFetch()
    show()
    await screen.findByLabelText('Dapat Diinvestigasi')

    expect(screen.getByLabelText('Keterangan Tambahan')).toBeInTheDocument()
    expect(screen.getByLabelText('Nama Karyawan')).toBeInTheDocument()
    expect(screen.getByLabelText('Tagihan Lunas/Belum Lunas')).toBeInTheDocument()
  })

  /**
   * "Tanggal Perawatan/Kejadian" adalah KELOMPOK tersendiri di Pega, ber-`pyTitle` sendiri,
   * berisi Tanggal Masuk dan Tanggal Keluar.
   */
  it('mengelompokkan tanggal perawatan seperti di Pega', async () => {
    installFetch()
    show()
    await screen.findByLabelText('Dapat Diinvestigasi')

    const group = screen.getByRole('group', { name: 'Tanggal Perawatan/Kejadian' })
    expect(within(group).getByLabelText('Tanggal Masuk')).toBeInTheDocument()
    expect(within(group).getByLabelText('Tanggal Keluar')).toBeInTheDocument()
  })

  /*
    Judul dan label DISALIN dari rule field-value `pyCaption …` pada section lama — daftar
    yang dapat dibaca utuh, dan karena itu tidak bergantung pada pemasangan label XML yang
    sudah dua kali meleset di berkas yang sama.

    Dua label sempat salah: `PTReg` diberi "Keterangan Tambahan" (milik `RemarksDOB`), dan
    `RemarksDOB` diberi "Keterangan Tambahan Tanggal Lahir" yang tidak ada di Pega sama
    sekali.
  */
  it('memakai judul dan label dari caption Pega', async () => {
    installFetch()
    show()
    await screen.findByLabelText('Dapat Diinvestigasi')

    expect(screen.getByText('Form Investigasi Personal Accident')).toBeInTheDocument()
    expect(screen.getByText('Daftar Pertanyaan')).toBeInTheDocument()
    expect(screen.getByLabelText('PT / Reg')).toBeInTheDocument()
    expect(screen.getByLabelText('Tidak Ada Pembayaran')).toBeInTheDocument()
    expect(screen.queryByLabelText('Keterangan Tambahan Tanggal Lahir')).not.toBeInTheDocument()
  })

  /**
   * Empat isian bertanda bintang di layar lama; penandanya ditulis pada labelnya.
   *
   * "Rumah Sakit" dipilih lebih dulu karena Alamat RS/Klinik BERSYARAT — ia tidak tergambar
   * selama Tempat Kejadian masih kosong.
   */
  it('menandai empat isian wajib', async () => {
    installFetch()
    show()
    await screen.findByLabelText('Dapat Diinvestigasi')
    await userEvent.selectOptions(screen.getByLabelText('Tempat Kejadian'), '1')

    for (const label of [
      'Alamat RS/Klinik/Lokasi Kejadian *',
      'Nomor Rekam Medik *',
      'Nama PIC Yang Dapat Dihubungi *',
      'Nomor Telepon Yang Dapat Dihubungi *',
    ]) {
      expect(screen.getByLabelText(label)).toBeInTheDocument()
    }
  })

  /** Tombol Back menutup tanpa mengirim apa pun. */
  it('menutup lewat Back tanpa menyimpan', async () => {
    installFetch()
    const onClose = show()
    await screen.findByLabelText('Dapat Diinvestigasi')

    await userEvent.click(screen.getByRole('button', { name: /^back$/i }))

    expect(onClose).toHaveBeenCalled()
    expect(calls.some((one) => one.method === 'POST')).toBe(false)
  })
})

describe('tab Unggah Dokumen', () => {
  /*
    Formulir lamanya punya DUA tab — `pyTitle` "Investigasi" dan "Unggah Dokumen" pada
    `Section/InputClaimInvestigasiDetail-Section.xml`.

    Tab kedua sempat tidak dibangun dengan alasan "menunggu modul Dokumen", dan alasan itu
    keliru: panelnya sudah ada di layar Registrasi dan dipakai ulang di sini.
  */
  it('menggambar kedua tab seperti di layar lama', async () => {
    installFetch()
    show()
    await screen.findByLabelText('Dapat Diinvestigasi')

    const tabs = screen.getAllByRole('tab')
    expect(tabs.map((one) => one.textContent)).toEqual(['Investigasi', 'Unggah Dokumen'])
    expect(tabs[0]).toHaveAttribute('aria-selected', 'true')
  })

  it('mengganti isi saat tab Unggah Dokumen dipilih', async () => {
    installFetch()
    show()
    await screen.findByLabelText('Dapat Diinvestigasi')

    await userEvent.click(screen.getByRole('tab', { name: 'Unggah Dokumen' }))

    // Isian investigasi menghilang, kategori dokumen muncul.
    expect(screen.queryByLabelText('Dapat Diinvestigasi')).not.toBeInTheDocument()
    // Judul kolom panel dokumen — penanda yang tidak kembar, tidak seperti nama
    // kategorinya yang muncul dua kali (tab kategori dan judul tabel).
    expect(await screen.findByText('Wajib Unggah')).toBeInTheDocument()
  })

  /*
    `claimID` yang dikirim WAJIB `referensi` (pzInsKey), bukan nomor kasus: itulah bentuk
    yang disimpan `T_CLAIM_OBJECTLIST.CLAIMID`. Diperiksa langsung ke katalog pada
    2026-10-07 — ia cocok dengan PZINSKEY pada 1.499 baris dan dengan PYID pada NOL baris.
  */
  it('meminta dokumen memakai pzInsKey, bukan nomor kasus', async () => {
    installFetch()
    show()
    await screen.findByLabelText('Dapat Diinvestigasi')
    await userEvent.click(screen.getByRole('tab', { name: 'Unggah Dokumen' }))
    await screen.findByText('Wajib Unggah')

    const permintaan = calls.find((one) => one.url.includes('/dokumen'))
    expect(permintaan?.url).toContain(encodeURIComponent(TUGAS.referensi))
    // Awalan kelas Pega-nya ikut terkirim. Nomor kasus saja TIDAK cukup — ia hanya
    // ekor dari pzInsKey, dan `T_CLAIM_OBJECTLIST.CLAIMID` menyimpan kunci utuhnya.
    expect(permintaan?.url).toContain('ASM-FW-GCNMFW-WORK%20')
  })
})
