import { useRef, useState, type FormEvent, type ReactNode } from 'react'

import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { SelectField } from '@/components/SelectField'

import { SegmentPanel } from './SegmentPanel'
import {
  useEkspor,
  useFormatFile,
  useKeterangan,
  useKirimSlik,
  useProses,
  useUnggah,
  type HasilTulis,
} from './api'
import { penyaringKosong, type KodeSegmen, type Penyaring } from './types'

/**
 * Monitoring SLINK OJK — menu `MENU_ID 78`, pengganti harness `MonitoringSLINKOJK`.
 *
 * # Apa yang dipantau layar ini
 *
 * SLIK adalah **Sistem Layanan Informasi Keuangan** OJK. Untuk lini Asuransi Kredit dan
 * Surety Bond, setiap klaim yang dibayarkan wajib dilaporkan ke OJK dalam bentuk berkas
 * bersegmen. Layar ini memantau apa yang akan dan sudah dilaporkan.
 *
 * Dua segmen, dan keduanya melaporkan hal yang BERBEDA:
 *
 *	D01  FASILITAS kredit — nomor rekening fasilitas, kolektibilitas, tunggakan, kondisi
 *	F06  DEBITUR individu — identitas, alamat, pekerjaan, pasangan, penghasilan
 *
 * Keduanya bukan dua tampilan dari data yang sama: D01 membaca tabel SLIK yang sudah
 * terisi, F06 membaca berkas klaim sumbernya.
 *
 * # Dua nama isian tanggal yang MENYESATKAN, dan sengaja dipertahankan
 *
 * Isian "Dari" dan "Sampai" menyaring **tanggal registrasi klaim**. Di balik layar nama
 * parameternya `date_of_loss` dan `date_of_request_document` — nama properti Pega yang
 * tidak ada hubungannya dengan isinya. Work Owner memutuskan namanya dipertahankan
 * (2026-09-26) supaya penelusuran ke `GetTempDataD01` tetap langsung.
 *
 * Label di layar TIDAK ikut menyesatkan: yang dibaca pengguna tetap "Dari" dan "Sampai",
 * persis seperti di Pega.
 *
 * # Keenam tombolnya ada
 *
 * Termasuk ketiga yang MENULIS — "Proses Data Klaim", "Upload Data Klaim", dan
 * "SLIK OJK" — yang dibangun pada 2026-09-26 sesudah logikanya ditemukan di luar section:
 * di `InsertDataSlikOJKF06`, `PNCUploadAutoClaimSlikOJK`, dan `QuerySLINKIndividu`.
 *
 * Satu bagian tetap belum tersedia: **alamat layanan SLIK**, karena kontrak
 * `Rest_SendDataClientBasedDebitur` tidak ada di export. Tombolnya menolak dengan
 * keterangan alih-alih diam — lihat CatatanPengiriman.
 *
 * Ketiganya dinonaktifkan sampai "Cari Data" ditekan: ketiganya bekerja atas himpunan
 * yang sedang disaring, dan menekannya sebelum melihat isinya berarti menyusun laporan
 * regulator tanpa tahu isinya apa.
 */
export function MonitoringSlinkOjkPage() {
  const portal = useSelectedPortal((state) => state.alias)

  if (portal === null) {
    return (
      <PageFrame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Kewajiban lapor SLIK adalah kewajiban satu badan hukum kepada OJK, dan ' +
            'aplikasi ini melayani empat. Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </PageFrame>
    )
  }

  return (
    <PageFrame>
      <IsiLayar />
    </PageFrame>
  )
}

function IsiLayar() {
  const keterangan = useKeterangan()
  const ekspor = useEkspor()
  const formatFile = useFormatFile()
  const proses = useProses()
  const unggah = useUnggah()
  const kirimSlik = useKirimSlik()

  // Pemilih berkas disembunyikan dan dipicu tombol, bukan ditampilkan sebagai kotak
  // unggah: layar lama pun hanya punya tombol, dan menambah kotak di tengah bilah tombol
  // mengubah tata letaknya (`D-13`).
  const berkasRef = useRef<HTMLInputElement>(null)

  const [hasilTulis, setHasilTulis] = useState<HasilTulis | null>(null)
  const [galatTulis, setGalatTulis] = useState<string | null>(null)
  const [sedangTulis, setSedangTulis] = useState<string | null>(null)

  // Dua keadaan penyaring, dan pemisahannya disengaja:
  //
  //   form      — yang sedang diketik pengguna
  //   terkirim  — yang sedang ditampilkan tabel
  //
  // Layar lama menuntut "Cari Data" ditekan. Menyatukan keduanya berarti tabel menembak
  // kueri atas puluhan juta baris setiap kali satu huruf diketik.
  const [form, setForm] = useState<Penyaring>(penyaringKosong)
  const [terkirim, setTerkirim] = useState<Penyaring>(penyaringKosong)
  const [halaman, setHalaman] = useState(1)
  const [sudahDicari, setSudahDicari] = useState(false)
  const [galatUnduh, setGalatUnduh] = useState<string | null>(null)

  const segmenAktif = keterangan.data?.segmen.find((item) => item.kode === form.segmen)

  function ubah<K extends keyof Penyaring>(kunci: K, nilai: Penyaring[K]) {
    setForm((sekarang) => ({ ...sekarang, [kunci]: nilai }))
  }

  function cari(event: FormEvent) {
    event.preventDefault()
    setTerkirim(form)
    setHalaman(1)
    setSudahDicari(true)
  }

  // Berpindah segmen MENGOSONGKAN hasil, tidak membawanya serta. Kedua segmen punya
  // kolom yang berbeda seluruhnya, dan menampilkan baris segmen sebelumnya di bawah
  // kepala kolom segmen baru adalah tabel yang isinya bohong.
  function pindahSegmen(kode: KodeSegmen) {
    setForm((sekarang) => ({ ...sekarang, segmen: kode }))
    setSudahDicari(false)
    setHalaman(1)
  }

  async function unduh(jalankan: () => Promise<void>) {
    setGalatUnduh(null)
    try {
      await jalankan()
    } catch (error) {
      setGalatUnduh(error instanceof Error ? error.message : 'Berkas tidak dapat diunduh.')
    }
  }

  /**
   * jalankanTulis membungkus ketiga aksi tulis dengan penanganan yang sama.
   *
   * Satu tempat, bukan tiga: yang berbeda hanyalah apa yang dijalankan, sedangkan
   * pengosongan hasil sebelumnya, penandaan "sedang berjalan", dan penanganan galatnya
   * identik — dan yang tertinggal di salah satu dari tiga tempat adalah tombol yang
   * tampak menggantung.
   */
  async function jalankanTulis(nama: string, jalankan: () => Promise<HasilTulis>) {
    setGalatTulis(null)
    setHasilTulis(null)
    setSedangTulis(nama)
    try {
      setHasilTulis(await jalankan())
    } catch (error) {
      setGalatTulis(error instanceof Error ? error.message : 'Aksi tidak dapat dijalankan.')
    } finally {
      setSedangTulis(null)
    }
  }

  /**
   * prosesDataKlaim menyusun laporan dari data klaim sumber.
   *
   * Ia memakai penyaring yang SUDAH dicari (`terkirim`), bukan yang sedang diketik.
   * Alasannya bukan kerapian: yang tersusun harus persis yang terlihat di tabel, dan
   * memakai `form` berarti menyusun sesuatu yang belum pernah dilihat pelapor.
   */
  function prosesDataKlaim() {
    jalankanTulis('proses', () => proses(terkirim))
  }

  async function unggahBerkas(berkas: File) {
    await jalankanTulis('unggah', () => unggah(berkas))
    // Pemilih berkas dikosongkan supaya berkas yang SAMA dapat diunggah ulang. Tanpa
    // ini, memilih berkas yang sama dua kali tidak memicu `change` sama sekali.
    if (berkasRef.current) berkasRef.current.value = ''
  }

  /**
   * kirimKeSlik mengirim data debitur klaim yang sedang tampil.
   *
   * Sasarannya SAMA dengan "Proses Data Klaim" — himpunan yang sedang disaring — karena
   * cara Pega memilih klaimnya di layar ini tidak dapat dipulihkan dari export. Lihat
   * `SubmitFiltered` di backend.
   */
  function kirimKeSlik() {
    jalankanTulis('kirim', () => kirimSlik(terkirim))
  }

  if (keterangan.isError) {
    return (
      <ErrorMessage
        title="Keterangan layar tidak dapat dimuat"
        description={
          keterangan.error instanceof Error
            ? keterangan.error.message
            : 'Terjadi kesalahan saat membaca susunan kolom.'
        }
        tone="gangguan"
      />
    )
  }

  return (
    <>
      <TabSegmen
        aktif={form.segmen}
        daftar={keterangan.data?.segmen ?? []}
        onPilih={pindahSegmen}
      />

      <form onSubmit={cari} className="mt-4 rounded-kartu border border-slate-200 bg-white p-4">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <SelectField
            id="business_name"
            label="Business Name"
            value={form.business_name}
            onChange={(event) => ubah('business_name', event.target.value)}
            options={(keterangan.data?.business_name ?? []).map((pilihan) => ({
              value: pilihan.nilai,
              label: pilihan.label,
            }))}
            emptyText="— seluruh lini —"
          />

          {/*
            Tidak ada isian "Tipe Generate" di sini.

            Properti `.GenerateType` ada di `Sec_SegmentD01_1-Section.xml`, dan atas dasar
            itu ia sempat dibangun. Tangkapan layar Pega yang berjalan (2026-10-08)
            membuktikan ia tidak tampil — segmen D01 di sana hanya punya Business Name,
            Dari, dan Sampai. Section di export lebih tua daripada yang terpasang.
          */}

          {/*
            Label "Dari" dan "Sampai" mengikuti layar Pega. Keterangan di bawahnya yang
            menjelaskan apa yang sebenarnya disaring — tanggal REGISTRASI klaim, bukan
            tanggal kejadian.
          */}
          <Field
            id="date_of_loss"
            label="Dari"
            type="date"
            value={form.date_of_loss}
            onChange={(event) => ubah('date_of_loss', event.target.value)}
            hint="Tanggal registrasi klaim, batas awal."
          />
          <Field
            id="date_of_request_document"
            label="Sampai"
            type="date"
            value={form.date_of_request_document}
            onChange={(event) => ubah('date_of_request_document', event.target.value)}
            hint="Tanggal registrasi klaim, batas akhir."
          />

        </div>

        {/*
          Tidak ada kotak pencarian di sini, dan itu disengaja.

          Sempat ada — "No Klaim / Contract No". Layar lama tidak punya: sectionnya hanya
          mengikat "Business Name", "Dari", dan "Sampai". Dicabut atas permintaan Work
          Owner (2026-09-27).

          Urutan tombolnya mengikuti layar lama: Cari Data, Export Data, Format File,
          Proses Data Klaim, Upload Data Klaim, SLIK OJK (`D-13`).
        */}
        <div className="mt-4 flex flex-wrap items-center gap-2">
          <Button type="submit" tone="utama">
            Cari Data
          </Button>
          <Button type="button" tone="kedua" onClick={() => { unduh(() => ekspor(terkirim)) }}>
            Export Data
          </Button>
          {/*
            Keempat tombol berikut ada di segmen F06, bukan D01.

            Ketiganya menyusun dan mengirim data FASILITAS KREDIT — yang ditulis
            `T_CLAIM_SLIK_OJK`, sumber segmen F06 — dan berkas "Format File" pun berisi
            kolom fasilitas. Sempat dipasang di D01 mengikuti pemetaan segmen yang keliru.

            Catatan: layar Pega PRODUKSI tidak menampilkan keempatnya. Keempatnya ada di
            `pegadev` dan dibangun atas permintaan Work Owner (2026-09-26), jadi
            dipertahankan — kemungkinan besar kemampuan yang belum naik ke produksi.
          */}
          {form.segmen === 'F06' && (
            <Button type="button" tone="halus" onClick={() => { unduh(formatFile) }}>
              Format File
            </Button>
          )}

          {/*
            Ketiga tombol tulis di bawah SELALU aktif, sama seperti layar lama.

            Sempat dinonaktifkan sampai "Cari Data" ditekan. Pemeriksaan ke
            `Sec_SegmentD01_1-Section.xml` menunjukkan tidak ada satu pun
            `pyDisabledCondition` di sana — di Pega ketiganya dapat ditekan kapan saja,
            termasuk sebelum penyaringnya pernah dijalankan. Work Owner meminta
            perilakunya disamakan (2026-09-27).

            Yang TETAP menonaktifkan hanyalah `sedangTulis` — penjaga terhadap penekanan
            ganda selagi permintaannya berjalan. Pega memblokir layarnya selama posting,
            jadi itu penyetaraan, bukan penambahan.

            Akibat yang disadari: penyaring yang belum pernah dijalankan berarti penyaring
            KOSONG, dan menekan "Proses Data Klaim" saat itu menyusun laporan atas seluruh
            klaim yang memenuhi syarat.
          */}
          {form.segmen === 'F06' && (
            <Button
              type="button"
              tone="kedua"
              disabled={sedangTulis !== null}
              onClick={prosesDataKlaim}
            >
              {sedangTulis === 'proses' ? 'Memproses…' : 'Proses Data Klaim'}
            </Button>
          )}

          {/*
            "Upload Data Klaim" dan "SLIK OJK" HANYA ada di segmen D01.

            Tangkapan layar Pega yang berjalan pada segmen F06 hanya menampilkan dua
            tombol: "Cari Data" dan "Export Data". Keduanya sempat tampil di kedua segmen
            karena `Sec_SegmentF06-Section.xml` memuat labelnya — tetapi label yang ada di
            XML belum tentu dirender, dan layar yang berjalan adalah bukti yang lebih kuat
            daripada berkasnya.
          */}
          {form.segmen === 'F06' && (
            <Button
              type="button"
              tone="kedua"
              disabled={sedangTulis !== null}
              onClick={() => berkasRef.current?.click()}
            >
              {sedangTulis === 'unggah' ? 'Mengunggah…' : 'Upload Data Klaim'}
            </Button>
          )}

          {form.segmen === 'F06' && (
            <Button
              type="button"
              tone="kedua"
              disabled={sedangTulis !== null}
              onClick={kirimKeSlik}
            >
              {sedangTulis === 'kirim' ? 'Mengirim…' : 'SLIK OJK'}
            </Button>
          )}
        </div>

        {/*
          Pemilih berkas disembunyikan dan dipicu tombol di atas — layar lama pun hanya
          punya tombol, dan menambah kotak unggah mengubah tata letaknya.
        */}
        <input
          ref={berkasRef}
          type="file"
          accept=".csv,text/csv"
          className="hidden"
          aria-hidden="true"
          tabIndex={-1}
          onChange={(event) => {
            const berkas = event.target.files?.[0]
            if (berkas) unggahBerkas(berkas)
          }}
        />

        {!sudahDicari && (
          <p className="mt-3 text-sm text-slate-600">
            Tekan <strong>Cari Data</strong> lebih dulu — ketiga tombol yang menyusun dan
            mengirim laporan bekerja atas data yang sedang tampil.
          </p>
        )}

        {galatUnduh && (
          <p className="mt-3 text-sm text-red-700" role="alert">
            {galatUnduh}
          </p>
        )}
        {galatTulis && (
          <p className="mt-3 text-sm text-red-700" role="alert">
            {galatTulis}
          </p>
        )}
        {hasilTulis && <RingkasanTulis hasil={hasilTulis} />}
      </form>

      {segmenAktif && (
        <SegmentPanel
          segmen={segmenAktif}
          penyaring={terkirim}
          halaman={halaman}
          onHalamanChange={setHalaman}
          aktif={sudahDicari}
          kunciBaris={keterangan.data?.kunci_baris ?? 'kunci_baris'}
        />
      )}

      <CatatanPengiriman />
    </>
  )
}

/**
 * RingkasanTulis melaporkan hasil satu aksi tulis.
 *
 * # Kenapa "baru" dan "diperbarui" dipisah
 *
 * Karena artinya berbeda bagi pelapor. **Diperbarui** berarti klaim itu SUDAH pernah
 * masuk laporan dan kini masuk lagi — dan itulah yang patut diperiksa sebelum berkasnya
 * dikirim ke OJK. Menjumlahkan keduanya menyembunyikan justru angka yang penting.
 *
 * Baris yang ditolak ditampilkan beserta nomor barisnya, supaya pelapor dapat membuka
 * berkasnya dan memperbaiki baris itu — bukan menebak baris mana yang salah.
 */
function RingkasanTulis({ hasil }: { hasil: HasilTulis }) {
  return (
    <div
      className="mt-3 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3 text-sm"
      role="status"
    >
      <p className="text-slate-800">
        <strong className="font-semibold">{hasil.total} baris</strong> diproses —{' '}
        {hasil.baru} baru, {hasil.diperbarui} diperbarui.
      </p>

      {hasil.diperbarui > 0 && (
        <p className="mt-1 text-slate-600">
          Baris &ldquo;diperbarui&rdquo; adalah klaim yang <strong>sudah pernah</strong>{' '}
          masuk laporan; periksa lebih dulu sebelum berkasnya dikirim ke OJK.
        </p>
      )}

      {hasil.ditolak.length > 0 && (
        <div className="mt-2">
          <p className="font-medium text-amber-900">
            {hasil.ditolak.length} baris ditolak:
          </p>
          <ul className="mt-1 list-disc space-y-0.5 pl-5 text-amber-900">
            {hasil.ditolak.slice(0, 10).map((baris, index) => (
              <li key={`${baris.baris ?? 0}-${baris.no_klaim ?? index}`}>
                {baris.baris ? `Baris ${baris.baris}: ` : ''}
                {baris.no_klaim ? `${baris.no_klaim} — ` : ''}
                {baris.alasan}
              </li>
            ))}
          </ul>
          {hasil.ditolak.length > 10 && (
            <p className="mt-1 text-amber-900">
              …dan {hasil.ditolak.length - 10} baris lainnya.
            </p>
          )}
        </div>
      )}
    </div>
  )
}

/**
 * CatatanPengiriman menjelaskan satu-satunya bagian yang masih belum tersedia.
 *
 * Ketiga tombol tulis sudah dibangun; yang belum adalah **alamat layanan SLIK**, karena
 * kontrak `Rest_SendDataClientBasedDebitur` tidak ada di export Pega. Keterangannya
 * ditulis di sini supaya pelapor yang menekan "SLIK OJK" dan mendapat penolakan tahu
 * sebabnya tanpa menghubungi siapa pun.
 */
function CatatanPengiriman() {
  return (
    <section className="mt-6 rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <h2 className="text-sm font-semibold text-slate-800">Tentang tombol SLIK OJK</h2>
      <p className="mt-2 text-sm text-slate-600">
        Pencatatan pengirimannya sudah berjalan, tetapi <strong>alamat layanan SLIK</strong>{' '}
        belum diatur di aplikasi ini. Selama itu belum ada, tombolnya menolak dengan
        keterangan dan tidak meninggalkan catatan pengiriman yang menyesatkan.
      </p>
    </section>
  )
}

/** TabSegmen menggantikan dropdown "Pilih Segmen" beserta tautan silang antarsegmen. */
function TabSegmen({
  aktif,
  daftar,
  onPilih,
}: {
  aktif: KodeSegmen
  daftar: Array<{ kode: KodeSegmen; label: string }>
  onPilih: (kode: KodeSegmen) => void
}) {
  return (
    <div className="mt-4 overflow-x-auto" role="tablist" aria-label="Segmen SLIK OJK">
      <div className="flex min-w-max items-center gap-1.5 border-b border-slate-200 pb-px">
        {daftar.map((segmen) => {
          const dipilih = segmen.kode === aktif
          return (
            <button
              key={segmen.kode}
              type="button"
              role="tab"
              aria-selected={dipilih}
              onClick={() => onPilih(segmen.kode)}
              className={
                'rounded-t-kontrol px-4 py-2 text-sm font-medium transition ' +
                (dipilih
                  ? 'border-b-2 border-blue-600 bg-white text-blue-700'
                  : 'border-b-2 border-transparent text-slate-600 hover:text-slate-900')
              }
            >
              {segmen.label}
            </button>
          )
        })}
      </div>
    </div>
  )
}
function PageFrame({ children }: { children: ReactNode }) {
  return (
    <div className="mx-auto max-w-[96rem] px-4 py-8">
      <header className="border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Monitoring SLINK OJK</h1>
        <p className="mt-1 text-sm text-slate-600">
          Pemantauan laporan klaim ke Sistem Layanan Informasi Keuangan OJK untuk lini
          Asuransi Kredit dan Surety Bond.
        </p>
      </header>
      {children}
    </div>
  )
}
