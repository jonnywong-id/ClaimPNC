import { useMemo, useState, type ReactNode } from 'react'

import { Button } from '@/components/Button'
import { DataTable, pageWindow, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { ChevronIcon } from '@/components/Icon'
import { TabBar } from '@/components/TabBar'

import {
  Tindakan,
  useEksporPLADLA,
  useKirimPreDLA,
  useKirimSurat,
  usePLADLACetak,
  usePLADLADokumen,
  usePLADLAList,
  usePLADLAMetadata,
  useTindakanPLADLA,
  type ParameterDaftar,
  type TindakanValue,
} from './api'
import { DocumentPanel } from './DocumentPanel'
import { PrintPreDLAPanel } from './PrintPreDLAPanel'
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
 * # Tiga tombol sudah MENULIS; dua belum dibangun
 *
 * Yang sudah bekerja: **"Send"** mengirim surat PLA/DLA beserta lampirannya ke reasuradur
 * lalu menandai dokumennya terkirim · **"Kirim Pre DLA"** menandai satu Pre-DLA terkirim ·
 * **"Print Pre DLA"** membuka panelnya.
 *
 * Yang belum: **"Upload File Penunjang"** dan **unduh lampiran**, keduanya menunggu
 * penyimpanan dokumen (`D-16`). Menekannya menjawab alasannya, bukan "halaman tidak
 * ditemukan".
 *
 * # Tombol "Send" TIDAK digambar pada setiap baris
 *
 * Syarat tampilnya dibawa dari Pega dan dihitung peladen — lihat `DocumentPanel`. Baris
 * yang dokumennya sudah terkirim tidak bertombol, sehingga surat kedua ke reasuradur yang
 * sama tidak dapat dipicu dari layar ini.
 *
 * # Susunan layar
 *
 * Urutannya mengikuti ketiga section layar lama baris per baris (`D-13`):
 *
 *	Judul layar
 *	Bilah tiga tab                 PLA · DLA · Pre DLA
 *	Keterangan daftar              satu kalimat, tidak ada di Pega
 *	Export To Excel · Refresh      rata kiri, DI ATAS penyaring
 *	Panel pencarian                Dari · Sampai  lalu  No Klaim · CARI DATA
 *	Judul daftar + pencacah        "Inbox PLA"  ·  "Total Data : n" dan nomor halaman
 *	Grid antrean                   nomor urut + 6 kolom Pega
 *	Panel bawah                    PLA · DLA  -> klik NOMOR KLAIM
 *	                               Pre DLA    -> tombol "Print Pre DLA" pada barisnya
 *	Selisih terencana              di kaki
 *
 * # Yang berubah pada 2026-10-10, dan kenapa
 *
 * Empat hal di atas sebelumnya tidak mengikuti Pega: tombol ekspor berada di kepala grid,
 * judul daftar tidak ada, pencacah hanya muncul di KAKI tabel sebagai bilah
 * First/Previous/Next/Last, dan gridnya bernama kolom bahasa Indonesia berjumlah tujuh —
 * yang ketujuh, "PIC Teknik", tidak ada di satu pun grid Pega. Work Owner meminta layar
 * depan dibuat sama dengan layar lama supaya petugas tidak perlu belajar ulang.
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

  // Panel mana yang terbuka ditentukan TABNYA, bukan penanda tersendiri.
  //
  // Satu klaim tidak pernah punya dua panel sekaligus: tab PLA dan DLA membuka grid
  // rincian, tab Pre DLA membuka panel cetak. Menyimpan dua penanda terpisah membuka
  // keadaan yang tidak mungkin — keduanya terbuka — yang lalu harus dijaga agar tidak
  // terjadi.
  //
  // Kunci yang KOSONG mematikan hook-nya. Itu yang membuat tab Pre DLA tidak menembak
  // kueri grid rincian yang di Pega pun tidak ada.
  const membukaCetak = daftarAktif?.punya_cetak === true
  const kunciDibuka = dibuka?.kunci_klaim ?? ''

  const dokumen = usePLADLADokumen(aktif, membukaCetak ? '' : kunciDibuka)
  const cetak = usePLADLACetak(membukaCetak ? kunciDibuka : '')

  // Penandaan "Kirim Pre DLA" — satu-satunya operasi TULIS di layar ini.
  //
  // Ia terpisah dari `tindakan` yang melayani tombol-tombol yang belum dibangun: yang ini
  // benar-benar bekerja, dan jawabannya bukan alasan melainkan hasil.
  const kirimPreDLA = useKirimPreDLA(kunciDibuka)

  // Pengiriman surat PLA/DLA — satu-satunya operasi di layar ini yang menyentuh dunia
  // di luar perusahaan, dan satu-satunya yang tidak dapat ditarik kembali.
  const kirimSurat = useKirimSurat(aktif, kunciDibuka)

  // Ketiga tombol yang belum dibangun memakai SATU mutation.
  //
  // Bukan tiga: yang dijawab server berbeda-beda, tetapi cara layar menanganinya sama
  // persis — tekan, terima alasannya, gambarkan. Tiga mutation berarti tiga tempat yang
  // harus dijaga tetap sejalan.
  const tindakan = useTindakanPLADLA()

  function minta(kode: TindakanValue) {
    tindakan.mutate({ tindakan: kode, tab: aktif })
  }

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

    // Alasan penolakan milik tombol daftar SEBELUMNYA dibuang.
    //
    // Tanpa ini, penjelasan tentang "Send" tetap tergambar setelah pengguna berpindah
    // ke tab Pre DLA — yang bahkan tidak punya tombol itu.
    tindakan.reset()
    kirimPreDLA.reset()
    kirimSurat.reset()
  }

  function cari() {
    setDikirim(form)
    setPage(1)
    setDibuka(null)
    tindakan.reset()
    kirimPreDLA.reset()
    kirimSurat.reset()
  }

  function bersihkan() {
    setForm(FORM_KOSONG)
    setDikirim(FORM_KOSONG)
    setPage(1)
    setDibuka(null)
    tindakan.reset()
    kirimPreDLA.reset()
    kirimSurat.reset()
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

      {/*
        "Export To Excel" berada DI ATAS panel pencarian dan rata kiri, seperti di Pega —
        bukan di kepala grid. Letaknya terbaca dari ketiga section, tempat tombolnya
        digambar sebelum blok penyaring.

        "Refresh" tidak ada di Pega dan tetap dibawa: ia tidak menulis apa pun, dan tanpa
        tombol itu satu-satunya cara memaksa pembacaan ulang adalah menyegarkan seluruh
        halaman — yang ikut membuang tab, nomor halaman, dan penyaring yang sedang dipakai.
      */}
      <div className="flex flex-wrap items-center gap-2">
        <Button
          type="button"
          tone="kedua"
          disabled={ekspor.isPending}
          onClick={() => ekspor.mutate(parameter)}
        >
          {ekspor.isPending ? 'Menyiapkan…' : 'Export To Excel'}
        </Button>
        <Button
          type="button"
          tone="kedua"
          onClick={() => { list.refetch() }}
          disabled={list.isFetching}
        >
          {list.isFetching ? 'Menyegarkan…' : 'Refresh'}
        </Button>
      </div>

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

      {/*
        Judul daftar beserta pencacah dan nomor halamannya, DI ATAS grid.

        Begitulah layar lama menyusunnya: `pyTitle` "Inbox PLA"
        (`Section/InboxPLA_sect-Section.xml:3561`) di kiri, dan paginator bernomor rata
        kanan sebaris dengannya. Sampai 2026-10-10 keduanya berada di KAKI tabel dalam
        bentuk First/Previous/Next/Last — bilah yang dikarang, dan yang memaksa petugas
        menggulir ke bawah untuk mengetahui ada berapa baris seluruhnya.
      */}
      <div className="flex flex-wrap items-end justify-between gap-3">
        <h2 className="text-base font-semibold text-slate-900">
          Inbox {daftarAktif?.nama ?? ''}
        </h2>

        <PageLinks
          halaman={list.data?.paginasi.halaman ?? 1}
          totalHalaman={list.data?.paginasi.total_halaman ?? 1}
          total={list.data?.paginasi.total ?? 0}
          sibuk={list.isFetching}
          onPilih={setPage}
        />
      </div>

      <DataTable<Baris>
        columns={kolomAntrean(daftarAktif, list.data?.baris ?? [], setDibuka)}
        rows={list.data?.baris ?? []}
        rowKey={(row) => row.kunci_klaim}
        label={`Antrean ${daftarAktif?.nama ?? ''}`}
        // Garis antarkolom, supaya gridnya terbaca berkotak seperti grid Pega.
        gridLines
        // Kotak cari bawaan dimatikan. Ia menyaring HANYA halaman yang sedang terbuka,
        // sehingga petugas dapat diberi tahu "tidak ada" untuk baris yang sebenarnya ada
        // di halaman berikutnya — dan panel "CARI DATA" di atas sudah menembak server.
        hideSearch
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
      />

      {ekspor.error != null && (
        <ErrorMessage
          title="Berkas ekspor tidak dapat diambil"
          description={pesanGalat(ekspor.error)}
          tone="gangguan"
        />
      )}

      {dibuka !== null && daftarAktif && !membukaCetak && (
        <DocumentPanel
          daftar={daftarAktif}
          baris={dibuka}
          data={dokumen.data?.baris}
          isPending={dokumen.isPending}
          isError={dokumen.isError}
          error={dokumen.error}
          onClose={() => setDibuka(null)}
          onUpload={() => minta(Tindakan.unggahPenunjang)}
          onSend={(dokumen) => kirimSurat.mutate(dokumen.no_advice)}
          busy={tindakan.isPending || kirimSurat.isPending}
          pesanKirim={
            kirimSurat.isSuccess
              ? `${kirimSurat.data.pesan} ${kirimSurat.data.penerima} penerima, ` +
                `${kirimSurat.data.lampiran} lampiran.`
              : ''
          }
        />
      )}

      {dibuka !== null && daftarAktif && membukaCetak && (
        <PrintPreDLAPanel
          daftar={daftarAktif}
          baris={dibuka}
          data={cetak.data?.baris}
          isPending={cetak.isPending}
          isError={cetak.isError}
          error={cetak.error}
          onClose={() => setDibuka(null)}
          onKirim={(dokumen) => kirimPreDLA.mutate(dokumen.no_advice)}
          busy={kirimPreDLA.isPending}
          pesanKirim={kirimPreDLA.isSuccess ? kirimPreDLA.data.pesan : ''}
        />
      )}

      {/*
        Alasan penolakan digambar sebagai **penolakan**, bukan gangguan.

        Nadanya menentukan apa yang disimpulkan pengguna: nada gangguan berarti "coba
        lagi nanti", sedangkan yang sebenarnya terjadi adalah kemampuannya memang belum
        dibangun — mengulang tidak akan menolong, dan yang harus dilakukannya ada di
        kalimat itu sendiri.
      */}
      {tindakan.error != null && (
        <ErrorMessage
          title="Tombol ini belum dapat dijalankan"
          description={pesanGalat(tindakan.error)}
          tone="penolakan"
        />
      )}

      {/*
        Penolakan penandaan Pre-DLA digambar TERPISAH dari penolakan tombol yang belum
        dibangun, dan judulnya berbeda.

        Sebab yang paling sering bukan kerusakan melainkan keadaan yang sudah berubah —
        Pre-DLA itu sudah ditandai petugas lain atau lewat Pega. "Tombol ini belum dapat
        dijalankan" akan salah menggambarkannya, dan pengguna akan mengira layarnya rusak
        padahal pekerjaannya justru sudah selesai.
      */}
      {/*
        Kegagalan pengiriman surat digambar TERPISAH, dan judulnya menyebut suratnya.

        Sebabnya bermacam-macam dan akibatnya BERBEDA: surat yang tidak terkirim sama
        sekali aman diulang; surat yang terkirim tetapi gagal dicatat TIDAK boleh
        diulang. Pesan dari server yang membedakannya, dan judul yang seragam akan
        menenggelamkan perbedaan itu.
      */}
      {kirimSurat.error != null && (
        <ErrorMessage
          title="Surat tidak dapat dikirim"
          description={pesanGalat(kirimSurat.error)}
          tone="gangguan"
        />
      )}

      {kirimPreDLA.error != null && (
        <ErrorMessage
          title="Pre-DLA tidak dapat ditandai terkirim"
          description={pesanGalat(kirimPreDLA.error)}
          tone="penolakan"
        />
      )}

    </Bingkai>
  )
}

/**
 * PageLinks adalah pencacah "Total Data" beserta nomor halaman, di ATAS grid.
 *
 * # Kenapa bukan bilah halaman bawaan `DataTable`
 *
 * Karena bentuknya berbeda dari yang digambar layar lama. Bilah bawaan berisi empat
 * tombol First/Previous/Next/Last di KAKI tabel; Pega menggambar NOMOR halaman di
 * kepalanya, sebaris dengan judul daftar dan rata kanan, didahului "Total Data :".
 * `D-13` menetapkan tata letaknya ditiru.
 *
 * Jendela nomornya memakai `pageWindow` milik `DataTable` — bukan salinan aturannya.
 * Dua jendela halaman yang terlihat sama tetapi hidup di dua berkas akan berbeda begitu
 * salah satunya disunting.
 *
 * # Dua tombol panah DIPERTAHANKAN, meski Pega hanya menggambar nomor
 *
 * Nomor halaman saja menuntut pengguna membidik sasaran selebar satu digit setiap kali ia
 * maju satu halaman. Keduanya diberi `aria-label` lengkap karena isinya hanya gambar
 * panah, dan panah tanpa nama tidak berarti apa pun bagi pembaca layar.
 */
function PageLinks({
  halaman,
  totalHalaman,
  total,
  sibuk,
  onPilih,
}: {
  halaman: number
  totalHalaman: number
  total: number
  sibuk: boolean
  onPilih: (halaman: number) => void
}) {
  const jumlahHalaman = Math.max(totalHalaman, 1)

  const tombol =
    'inline-flex h-7 min-w-7 items-center justify-center rounded-kontrol border px-2 text-sm ' +
    'transition-colors duration-150 ease-halus ' +
    'focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50 ' +
    'disabled:cursor-not-allowed disabled:opacity-40'

  const diam = `${tombol} border-slate-300 bg-white text-slate-700 hover:enabled:border-slate-400 hover:enabled:bg-slate-100`

  return (
    <div className="flex flex-wrap items-center gap-2">
      {/*
        Pencacahnya `role="status"`, sehingga berpindah halaman diumumkan pembaca layar
        tanpa memindahkan fokus dari tombol yang baru saja ditekan.
      */}
      <p className="text-sm text-slate-600" role="status">
        Total Data : <span className="font-medium text-slate-800">{total}</span>
      </p>

      {jumlahHalaman > 1 && (
        <nav aria-label="Navigasi halaman" className="flex flex-wrap items-center gap-1">
          <button
            type="button"
            className={diam}
            disabled={halaman <= 1 || sibuk}
            aria-label="Halaman sebelumnya"
            onClick={() => onPilih(halaman - 1)}
          >
            <ChevronIcon className="h-4 w-4 rotate-180" />
          </button>

          {pageWindow(halaman, jumlahHalaman).map((item, urutan) =>
            item === 'sela' ? (
              // Sela tidak punya nilai yang dapat dijadikan kunci, dan dua di antaranya
              // dapat muncul sekaligus — urutannya yang membedakan.
              <span
                key={`sela-${urutan}`}
                aria-hidden="true"
                className="px-1 text-sm text-slate-400"
              >
                …
              </span>
            ) : (
              <button
                key={item}
                type="button"
                className={
                  item === halaman
                    ? `${tombol} border-blue-600 bg-blue-600 font-medium text-white`
                    : diam
                }
                // Halaman yang sedang terbuka ditandai `aria-current`, bukan hanya oleh
                // warna — yang tidak terbaca pembaca layar dan tidak terbedakan oleh
                // sekitar satu dari dua belas laki-laki yang buta warna merah-hijau.
                aria-current={item === halaman ? 'page' : undefined}
                aria-label={`Halaman ${item}`}
                disabled={sibuk}
                onClick={() => onPilih(item)}
              >
                {item}
              </button>
            ),
          )}

          <button
            type="button"
            className={diam}
            disabled={halaman >= jumlahHalaman || sibuk}
            aria-label="Halaman berikutnya"
            onClick={() => onPilih(halaman + 1)}
          >
            <ChevronIcon className="h-4 w-4" />
          </button>
        </nav>
      )}
    </div>
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
  baris: Baris[],
  onBuka: (baris: Baris) => void,
): Column<Baris>[] {
  if (!daftar) return []

  // Kolom nomor urut, persis seperti grid Pega yang menomori barisnya 1, 2, 3, … di
  // paling kiri. Judulnya KOSONG — begitu pula di sana, dan nomor urut tidak menuntut
  // penjelasan bagi pembaca layar.
  //
  // Penomorannya dimulai dari satu pada SETIAP halaman, bukan diteruskan dari halaman
  // sebelumnya. Itu yang dilakukan grid Pega, dan nomor ini memang hanya alat menunjuk
  // baris — "yang nomor tiga" — bukan posisi di dalam seluruh antrean.
  //
  // Petanya disusun sekali. Mencari posisi sebuah baris saat menggambar setiap sel
  // berarti menelusuri seluruh daftar sebanyak jumlah barisnya.
  const nomor = new Map(baris.map((row, index) => [row.kunci_klaim, index + 1]))

  const kolomNomor: Column<Baris> = {
    key: 'nomor',
    title: '',
    width: '3rem',
    noSort: true,
    value: (row) => String(nomor.get(row.kunci_klaim) ?? ''),
  }

  const kolom: Column<Baris>[] = daftar.kolom.map((item) => ({
    key: item.kunci,
    title: item.judul,
    value: (row) => nilaiSel(row, item.kunci),
    render: (row) =>
      // Nomor klaim adalah TAUTAN yang membuka grid rincian — bukan teks biasa.
      //
      // Begitu pula di Pega: sel `.BRANCH_CODE` membawa `pyAction = refresh` beserta
      // parameter `caseId` bernilai `.BRANCH_NAME`, yakni menyegarkan grid rincian
      // dengan kunci klaim baris itu
      // (`Section/InboxPLA_sect-Section.xml:171096`).
      //
      // Hanya pada daftar yang PUNYA rincian. Di tab Pre DLA nomor klaimnya teks biasa,
      // dan yang membuka panelnya adalah tombol pada barisnya — tautan yang tidak
      // membuka apa-apa lebih buruk daripada teks biasa.
      item.kunci === 'no_klaim' && daftar.punya_rincian ? (
        <button
          type="button"
          onClick={() => onBuka(row)}
          className="rounded-kontrol px-1 text-left font-medium text-blue-700 underline-offset-2 hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-blue-500"
        >
          {nilaiSel(row, item.kunci) || '—'}
        </button>
      ) : (
        gambarSel(row, item.kunci, item.tanggal)
      ),
  }))

  kolom.unshift(kolomNomor)

  // Kolom aksi hanya ada pada Pre DLA, dan judulnya datang dari server.
  //
  // PLA dan DLA tidak punya kolom ini sama sekali: rinciannya dibuka dengan mengklik
  // nomor klaim, persis seperti di Pega. Tab Pre DLA punya tombol bernama di dalam
  // barisnya, dan letaknya terbukti dari section — label tombolnya berada DI DALAM blok
  // berulang, berdampingan dengan `pyLocalAction`-nya sendiri.
  const judulAksi = judulTombolBaris(daftar)

  if (judulAksi !== '') {
    kolom.push({
      key: 'aksi',
      title: 'Aksi',
      width: daftar.punya_cetak ? '11rem' : '8rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button type="button" tone="halus" onClick={() => onBuka(row)}>
          {judulAksi}
        </Button>
      ),
    })
  }

  return kolom
}

/**
 * judulTombolBaris mengambil judul tombol baris, dan bertahan terhadap peladen LAMA.
 *
 * # Kenapa ada cadangan, padahal judulnya memang milik server
 *
 * Karena tombol ini pernah HILANG dari layar, dan penyebabnya justru ketergantungan pada
 * satu medan jawaban. Bila peladen yang berjalan lebih tua daripada berkas layar —
 * keadaan yang biasa terjadi ketika hanya salah satu dari keduanya dibangun ulang —
 * `label_aksi_baris` tidak ada dalam jawabannya. Tanpa cadangan, SELURUH kolom aksi
 * lenyap, termasuk tombol "Rincian" yang sudah lama bekerja.
 *
 * Kegagalan seperti itu tidak menghasilkan satu pun galat: layarnya tampil normal, hanya
 * tanpa tombol. Yang melihatnya akan menyimpulkan tombolnya belum dibangun.
 *
 * # Ia BUKAN sumber kebenaran kedua
 *
 * Cadangannya hanya berlaku ketika medannya benar-benar TIDAK ADA. Medan yang ada tetapi
 * berisi teks kosong dihormati apa adanya — itu keputusan server bahwa daftar ini memang
 * tidak punya tombol baris, dan menimpanya akan menggambar tombol yang sengaja ditiadakan.
 *
 * Judul cadangannya pun sengaja diturunkan dari medan yang SUDAH lama ada
 * (`punya_cetak`, `punya_rincian`), bukan dari mencocokkan kode tab — sehingga tidak ada
 * daftar tab yang perlu dijaga sejalan di dua tempat.
 */
function judulTombolBaris(daftar: Daftar): string {
  if (daftar.label_aksi_baris !== undefined) return daftar.label_aksi_baris

  if (daftar.punya_cetak) return 'Print Pre DLA'

  // `punya_rincian` TIDAK lagi menghasilkan tombol.
  //
  // Sejak 2026-09-27 rincian dibuka dengan mengklik nomor klaim, dan cadangan yang
  // mengembalikan 'Rincian' akan menghidupkan kembali kolom yang baru saja dihapus —
  // pada peladen lama pengguna akan melihat DUA jalan menuju panel yang sama.
  return ''
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

