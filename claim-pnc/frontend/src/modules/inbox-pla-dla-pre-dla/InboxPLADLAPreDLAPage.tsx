import { useMemo, useState, type ReactNode } from 'react'

import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { TabBar } from '@/components/TabBar'

import {
  useEksporPLADLA,
  usePLADLADokumen,
  usePLADLAList,
  usePLADLAMetadata,
  type ParameterDaftar,
} from './api'
import { DocumentPanel } from './DocumentPanel'
import { SearchPanel } from './SearchPanel'
import { formatTanggal, galatIsian, pesanGalat, pesanMuat } from './pesan'
import type { Baris, Daftar, FormPencarian } from './types'

/** Isian penyaring yang kosong. */
const FORM_KOSONG: FormPencarian = { cari: '', dari: '', sampai: '' }

/**
 * Layar Inbox PLA, DLA, Pre DLA.
 *
 * Pengganti `Harness/InboxPLA_harness-Harness.xml` (`MENU_ID 44`).
 *
 * # Apa yang ditampilkan layar ini
 *
 * **Pemberitahuan reasuransi yang sudah terbit tetapi BELUM dikirim** kepada
 * koasuransi/reasuransi. Tiga tab, satu per jenis dokumen:
 *
 *	PLA      Preliminary Loss Advice — memberitahukan nilai ESTIMASI
 *	DLA      Definite Loss Advice — memberitahukan nilai AKSEPTASI
 *	Pre DLA  memberitahukan nilai yang AKAN diaksep, sebelum akseptasi
 *
 * Ia INBOX menurut keempat ciri `D-79`: barisnya pekerjaan, baris HILANG setelah
 * dikerjakan, daftarnya antrean, dan barisnya punya tenggat — klaim menunggu
 * pemberitahuan ke reasuradur.
 *
 * # JANGAN tertukar dengan "Inbox PLA DLA" (`MENU_ID 45`)
 *
 * Keduanya bersebelahan di menu dan judulnya hampir sama, tetapi penyaringnya BERLAWANAN
 * ARAH: layar ini menampilkan dokumen yang belum dikirim, layar itu menampilkan dokumen
 * yang sudah dikirim — dan yang membacanya reasuradur, bukan petugas internal.
 *
 * # Layar BACA-SAJA, dan itu keputusan Work Owner 2026-09-26
 *
 * Ketiga tombol yang menulis — "Send", "Upload File Penunjang", dan "Print Pre DLA" —
 * belum dibangun. "Send" di Pega mengirim surat beserta lampirannya lewat email LALU
 * menandai dokumennya terkirim; mengerjakan penandaan tanpa pengirimannya akan membuat
 * barisnya hilang dari antrean padahal tidak satu pun surat sampai.
 *
 * Ketiadaannya digambar di kaki layar sebagai selisih terencana, bukan disamarkan.
 *
 * # Susunan layar
 *
 *	Judul
 *	Bilah tiga tab                 PLA · DLA · Pre DLA
 *	Keterangan daftar              satu kalimat, tidak ada di Pega
 *	Panel pencarian                Dari · Sampai · No Klaim · CARI DATA
 *	Grid antrean                   7 kolom, paginasi 10 baris
 *	Panel rincian                  terbuka atas permintaan; tidak ada di tab Pre DLA
 *	Selisih terencana              di kaki
 */
export function InboxPLADLAPreDLAPage() {
  const meta = usePLADLAMetadata()

  const [tab, setTab] = useState('')
  const [page, setPage] = useState(1)

  // Penyaring dipegang DUA KALI: yang sedang diketik, dan yang sudah dikirim.
  //
  // Tanpa pemisahan itu, setiap huruf yang diketik akan menembak basis data — dan
  // penyaring di layar ini menyentuh tabel dokumen berisi puluhan juta baris. Layar
  // lamanya pun memakai tombol "CARI DATA", bukan pencarian saat mengetik.
  const [form, setForm] = useState<FormPencarian>(FORM_KOSONG)
  const [dikirim, setDikirim] = useState<FormPencarian>(FORM_KOSONG)

  const [dibuka, setDibuka] = useState<Baris | null>(null)

  const daftar = meta.data?.daftar ?? []
  const aktif = tab || meta.data?.daftar_bawaan || ''
  const daftarAktif = daftar.find((item) => item.kode === aktif)

  const parameter: ParameterDaftar = useMemo(
    () => ({ tab: aktif, page, ...dikirim }),
    [aktif, page, dikirim],
  )

  const list = usePLADLAList(parameter, aktif !== '')
  const ekspor = useEksporPLADLA()

  const dokumen = usePLADLADokumen(aktif, dibuka?.kunci_klaim ?? '')

  /**
   * Berpindah daftar mengembalikan halaman ke satu dan MENUTUP panel rincian.
   *
   * Panelnya ditutup karena isinya milik daftar sebelumnya: klaim yang sama punya grid
   * rincian yang berbeda di tab PLA dan tab DLA — tabel yang dibacanya pun berbeda.
   * Membiarkannya terbuka akan menampilkan rincian PLA di bawah antrean DLA.
   *
   * Penyaringnya sengaja TIDAK dibersihkan: rentang tanggal yang sedang dipakai biasanya
   * masih relevan di daftar berikutnya, dan menghapusnya memaksa pengguna mengetiknya
   * ulang setiap kali ia membandingkan ketiga antrean.
   */
  function pilihDaftar(kode: string) {
    setTab(kode)
    setPage(1)
    setDibuka(null)
  }

  function cari() {
    setDikirim(form)
    setPage(1)
    setDibuka(null)
  }

  function bersihkan() {
    setForm(FORM_KOSONG)
    setDikirim(FORM_KOSONG)
    setPage(1)
    setDibuka(null)
  }

  if (meta.isPending) {
    return (
      <Bingkai>
        <p className="text-sm text-slate-600">Memuat keterangan layar…</p>
      </Bingkai>
    )
  }

  if (meta.isError) {
    const pesan = pesanMuat(meta.error)
    return (
      <Bingkai>
        <ErrorMessage
          title={pesan.title}
          description={pesan.description}
          tone={pesan.tone}
        />
      </Bingkai>
    )
  }

  return (
    <Bingkai>
      <TabBar
        tabs={daftar.map((item) => ({
          kode: item.kode,
          nama: item.nama,
          keterangan: item.keterangan,
        }))}
        active={aktif}
        onSelect={pilihDaftar}
        label="Jenis pemberitahuan reasuransi"
      />

      {daftarAktif && (
        <p className="text-sm text-slate-600">{daftarAktif.keterangan}</p>
      )}

      <SearchPanel
        form={form}
        onChange={(perubahan) => setForm({ ...form, ...perubahan })}
        onSubmit={cari}
        onReset={bersihkan}
        busy={list.isFetching}
        labelTanggal={daftarAktif?.label_tanggal ?? 'Tanggal'}
        labelPencarian={daftarAktif?.label_pencarian ?? 'No Klaim'}
        fieldError={galatIsian(list.error)}
      />

      <DataTable<Baris>
        columns={kolomAntrean(daftarAktif, setDibuka)}
        rows={list.data?.baris ?? []}
        rowKey={(row) => row.kunci_klaim}
        label={`Antrean ${daftarAktif?.nama ?? ''}`}
        isLoading={list.isLoading}
        error={
          list.isError ? (
            <ErrorMessage
              title="Antrean tidak dapat dimuat"
              description={pesanGalat(list.error)}
              tone="gangguan"
            />
          ) : undefined
        }
        emptyMessage={pesanKosong(dikirim)}
        pagination={{
          page: list.data?.paginasi.halaman ?? 1,
          size: list.data?.paginasi.ukuran ?? 10,
          total: list.data?.paginasi.total ?? 0,
          totalPage: list.data?.paginasi.total_halaman ?? 1,
          onPageChange: setPage,
          isLoading: list.isFetching,
        }}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <Button
              type="button"
              tone="kedua"
              onClick={() => void list.refetch()}
              disabled={list.isFetching}
            >
              Refresh
            </Button>
            <Button
              type="button"
              tone="kedua"
              disabled={ekspor.isPending}
              onClick={() => ekspor.mutate(parameter)}
            >
              {ekspor.isPending ? 'Menyiapkan…' : 'Export To Excel'}
            </Button>
          </div>
        }
      />

      {ekspor.error != null && (
        <ErrorMessage
          title="Berkas ekspor tidak dapat diambil"
          description={pesanGalat(ekspor.error)}
          tone="gangguan"
        />
      )}

      {dibuka !== null && daftarAktif && (
        <DocumentPanel
          daftar={daftarAktif}
          baris={dibuka}
          data={dokumen.data?.baris}
          isPending={dokumen.isPending}
          isError={dokumen.isError}
          error={dokumen.error}
          onClose={() => setDibuka(null)}
        />
      )}

      <SelisihTerencana butir={meta.data?.selisih_terencana ?? []} />
    </Bingkai>
  )
}

/** Bingkai adalah judul layar beserta ruang isinya. */
function Bingkai({ children }: { children: ReactNode }) {
  return (
    <div className="space-y-5 p-6">
      <header>
        <h1 className="text-lg font-semibold text-slate-900">
          Inbox PLA, DLA, Pre DLA
        </h1>
        <p className="mt-1 text-sm text-slate-600">
          Pemberitahuan kepada koasuransi dan reasuransi yang sudah terbit tetapi belum
          dikirim.
        </p>
      </header>

      {children}
    </div>
  )
}

/**
 * kolomAntrean menerjemahkan kolom yang DIKIRIM SERVER menjadi kolom DataTable.
 *
 * # Kenapa kolomnya tidak ditulis di sini
 *
 * Karena ketujuh kolomnya adalah hasil pembacaan ketiga kueri Pega, dan tempat pembacaan
 * itu tercatat adalah backend. Alias Pega-nya menyesatkan — `"BRANCH_NAME"` berarti kunci
 * klaim, `"BRANCH_CODE"` berarti nomor klaim, `"END_DATE"` berarti tanggal kejadian — dan
 * menulis ulang pemetaannya di layar berarti dua tempat yang dapat bergeser.
 *
 * Yang tetap milik layar adalah cara satu sel DIGAMBAR: tanggal diformat, sel kosong
 * diberi tanda hubung.
 */
function kolomAntrean(
  daftar: Daftar | undefined,
  onBuka: (baris: Baris) => void,
): Column<Baris>[] {
  if (!daftar) return []

  const kolom: Column<Baris>[] = daftar.kolom.map((item) => ({
    key: item.kunci,
    title: item.judul,
    value: (row) => nilaiSel(row, item.kunci),
    render: (row) => gambarSel(row, item.kunci, item.tanggal),
  }))

  // Tombol rincian hanya digambar pada daftar yang PUNYA grid rincian.
  //
  // Tab Pre DLA tidak punya — di Pega pun tidak. Menggambar tombolnya di sana akan
  // menjanjikan sesuatu yang dijawab penolakan, dan penolakan yang dapat dicegah dengan
  // tidak menggambar tombolnya bukan penolakan yang layak ditampilkan.
  if (daftar.punya_rincian) {
    kolom.push({
      key: 'aksi',
      title: 'Aksi',
      width: '8rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button type="button" tone="halus" onClick={() => onBuka(row)}>
          Rincian
        </Button>
      ),
    })
  }

  return kolom
}

/** nilaiSel mengambil isi satu sel sebagai TEKS — yang dicari dan diurutkan. */
function nilaiSel(row: Baris, kunci: string): string {
  const sel = (row as unknown as Record<string, unknown>)[kunci]
  return typeof sel === 'string' ? sel : ''
}

/** gambarSel menggambar satu sel. */
function gambarSel(row: Baris, kunci: string, tanggal: boolean) {
  const isi = nilaiSel(row, kunci)
  if (tanggal) return formatTanggal(isi)
  if (isi === '') return <span className="text-slate-400">—</span>
  return isi
}

/**
 * pesanKosong menjelaskan MENGAPA antreannya kosong.
 *
 * Kosong karena tidak ada pekerjaan dan kosong karena penyaringnya terlalu sempit
 * terlihat sama — dan hanya yang kedua yang dapat ditindaklanjuti pengguna.
 */
function pesanKosong(penyaring: FormPencarian): string {
  const menyaring =
    penyaring.cari !== '' || penyaring.dari !== '' || penyaring.sampai !== ''

  if (menyaring) {
    return (
      'Tidak ada yang cocok dengan penyaring ini. Longgarkan rentang tanggalnya, ' +
      'atau kosongkan kotak No Klaim.'
    )
  }
  return 'Tidak ada pemberitahuan yang menunggu dikirim pada daftar ini.'
}

/**
 * SelisihTerencana menggambar selisih terhadap layar Pega di kaki halaman.
 *
 * # Kenapa ia digambar, bukan sekadar dicatat di kode
 *
 * Karena petugas yang membandingkan layar ini dengan Pega berdampingan AKAN menemukan
 * selisihnya — dan selisih yang tidak dinyatakan akan dilaporkan sebagai kerusakan, lalu
 * ditelusuri ulang oleh orang yang tidak tahu bahwa ia disengaja.
 */
function SelisihTerencana({ butir }: { butir: string[] }) {
  if (butir.length === 0) return null

  return (
    <details className="rounded-kartu border border-slate-200 bg-slate-50 p-4">
      <summary className="cursor-pointer text-sm font-medium text-slate-800">
        Perbedaan yang disengaja terhadap layar Pega ({butir.length})
      </summary>
      <ul className="mt-3 space-y-2 text-sm text-slate-600">
        {butir.map((isi) => (
          <li key={isi} className="flex gap-2">
            <span aria-hidden className="text-slate-400">
              •
            </span>
            <span>{isi}</span>
          </li>
        ))}
      </ul>
    </details>
  )
}
