import { useState, type ReactNode } from 'react'
import { useSearchParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatDate } from '@/components/format'

import { DetailSalvagePanel } from './DetailSalvagePanel'
import { StatusSummary } from './StatusSummary'
import { TambahSalvageForm } from './TambahSalvageForm'
import {
  useExportSalvage,
  useSalvageCounts,
  useSalvageDetail,
  useSalvageList,
  useSalvageMetadata,
} from './api'
import type {
  CatatanSimpan,
  DetailItem,
  DetailKey,
  DetailResponse,
  SalvageRow,
  Tab,
  TabColumn,
} from './types'

/**
 * Inbox Salvage — menu `MENU_ID 71`, pengganti harness `InboxSalvage`.
 *
 * Isinya pengelolaan **salvage**: nilai sisa barang rusak yang dapat dijual kembali, dan
 * yang MENGURANGI nilai bersih klaim (`CONTEXT.md`). Perjalanannya dari klaim yang belum
 * ditandai punya salvage sama sekali sampai barangnya terjual di balai lelang.
 *
 * # Susunan layar, dan dari mana bentuknya
 *
 * Diambil dari `Section/InboxSalvage-Section.xml` dan `InboxSalvageASM-Section.xml`:
 * tombol Tambah dan Refresh di kepala layar, grafik donat dan tabel ringkas "Status
 * Salvage / Jumlah", lalu grid berhalaman 20 baris. Judul kolom TIDAK diterjemahkan —
 * `D-13` menetapkan tampilan meniru Pega, dan itulah teks yang selama ini dibaca pengguna.
 *
 * # Gridnya BARU digambar setelah sebuah status diklik (2026-10-08)
 *
 * Sebelumnya layar jatuh ke `daftar_bawaan` dan langsung menggambar daftar Outstanding,
 * sehingga tabel ringkas di atasnya terbaca seperti hiasan. Keputusan Work Owner: ringkasan
 * dulu, daftarnya menyusul — seperti layar Pega, tempat barisnyalah yang mengirim
 * `Param.tipe`/`Param.tipe2`.
 *
 * Tiga akibat yang ditangani di sini, bukan dibiarkan:
 *
 *  1. Daftar TIDAK ditembak selama belum dipilih — bukan sekadar disembunyikan.
 *  2. Tambah dan Refresh naik ke kepala layar; keduanya dulu hidup di toolbar grid dan
 *     akan ikut hilang bersamanya.
 *  3. Ringkasan yang gagal atau kosong membuat layar jatuh ke `daftar_bawaan`, supaya
 *     satu-satunya pintu masuk yang tidak tergambar tidak berubah menjadi jalan buntu.
 *
 * # Tiga hal yang paling mudah disalahpahami di layar ini
 *
 * PERTAMA — angka pada tabel ringkas TIDAK selalu sama dengan jumlah baris daftarnya.
 * Tiga baris menghitung populasi yang BERBEDA dari daftar yang dibukanya, dan itu keadaan
 * di Pega yang sengaja direplikasi (`P-5`).
 *
 * KEDUA — dua daftar berisi baris yang SAMA PERSIS. "Checker" dan "Salvage Diterima"
 * keduanya menyaring status transfer yang sama; di layar lama keduanya memang dua pintu
 * masuk ke kumpulan baris yang sama.
 *
 * KETIGA — pencarian pada tiga daftar COCOK PERSIS, bukan mengandung. Mengetik separuh
 * nomor klaim di sana tidak menghasilkan apa-apa, dan itu perilaku layar lama apa adanya.
 *
 * Ketiganya TIDAK lagi dinyatakan di layar. Bilah kuning `catatan_daftar` dicabut
 * 2026-10-08, menyusul panel selisih terencana yang dicabut 2026-10-06 — keduanya atas
 * alasan yang sama: isinya bukan galat, sementara warnanya mengatakan sebaliknya kepada
 * setiap orang yang membuka daftar. Keterangannya tetap hidup di kode Go untuk uji
 * kesetaraan gerbang 1 (`D-54`), tempat ia dipetakan ke butir `P-5`.
 *
 * # Portal Insurtech BELUM dibangun
 *
 * Layar lama punya isi tersendiri untuknya (`Section/InboxSalvageInsurtech`), dan judul
 * layarnya pun berbeda — harness memilih di antara keduanya dengan membandingkan
 * `TempGetApp.LSC_ID`. Keputusan Work Owner 2026-09-25: portal ASM dibangun lebih dulu.
 */
export function SalvageInboxPage() {
  // Daftar, halaman, dan kata kunci hidup di ALAMAT, bukan di state komponen.
  //
  // Alasannya sama dengan modul inbox lain: layar ini dibuka berpuluh kali sehari, dan
  // petugas yang kembali dari layar lain harus mendarat di tempat ia tinggalkan. Kata
  // kunci ikut, karena pencariannya menyaring DI SERVER — ia bagian dari apa yang sedang
  // dilihat, bukan preferensi tampilan.
  const [params, setParams] = useSearchParams()
  const [adding, setAdding] = useState(false)
  // Catatan di atas layar sesudah Submit, beserta NADA-nya.
  //
  // Sebelumnya ia sekadar teks yang selalu digambar hijau. Sejak Submit benar-benar
  // mengirim ke balai lelang dan benar-benar mengirim surel, salah satunya dapat gagal
  // sementara pengajuannya tetap tersimpan — dan bilah hijau yang berbunyi "GAGAL
  // terkirim" adalah isyarat bercampur yang justru membuat orang berhenti membacanya.
  const [saved, setSaved] = useState<CatatanSimpan | null>(null)

  // Pengajuan yang panel rinciannya sedang terbuka; kosong berarti tertutup.
  //
  // Ia state komponen, BUKAN bagian alamat seperti daftar dan halaman. Alasannya: rincian
  // dibuka untuk dibaca sekali lalu ditutup, bukan untuk ditinggalkan dan kembali — dan
  // menaruhnya di alamat membuat tombol "kembali" peramban menutup panel alih-alih
  // meninggalkan layar, yang bukan yang diharapkan orang.
  const [opened, setOpened] = useState<{ key: DetailKey; reference: string } | null>(null)

  const tabCode = params.get('daftar') ?? ''
  const page = Math.max(1, Number(params.get('halaman') ?? '1') || 1)
  const search = params.get('cari') ?? ''

  const portal = useSelectedPortal((state) => state.alias)
  const meta = useSalvageMetadata()
  const counts = useSalvageCounts()

  const tabs: Tab[] = meta.data?.daftar ?? []

  // Ringkasan yang TIDAK dapat dipakai sebagai navigasi.
  //
  // Dua sebabnya, dan keduanya nyata: permintaan `/ringkas` gagal, atau ia berhasil tetapi
  // tidak mengembalikan satu baris pun.
  const ringkasTidakTerpakai =
    counts.isError || (counts.isSuccess && (counts.data?.baris.length ?? 0) === 0)

  // TIDAK ada daftar yang terbuka sampai sebuah status DIKLIK.
  //
  // Sebelum 2026-10-08 layar jatuh ke `daftar_bawaan` dan langsung menggambar daftar
  // Outstanding — sehingga tabel ringkas di atasnya terbaca seperti hiasan, dan petugas
  // yang membuka layar ini langsung dihadapkan pada satu daftar yang belum tentu ia cari.
  //
  // Keputusan Work Owner 2026-10-08: ringkasan dulu, daftarnya menyusul setelah diklik —
  // seperti layar Pega.
  //
  // # Kenapa `daftar_bawaan` TETAP dipakai, dan hanya di satu keadaan
  //
  // Karena sejak tabel ringkas menjadi satu-satunya pintu masuk, ringkasan yang tidak
  // tergambar berarti layar TANPA PINTU: tidak ada yang dapat diklik, dan kalimat "pilih
  // salah satu Status Salvage" menunjuk ke tempat kosong. Itu lebih buruk daripada
  // keadaan sebelumnya, bukan lebih baik.
  //
  // Karena itu ketika ringkasannya gagal atau kosong, layar jatuh ke `daftar_bawaan` —
  // tetap dapat dipakai, hanya tanpa navigasi. Pada keadaan normal ia tidak pernah
  // tersentuh.
  const active = tabCode || (ringkasTidakTerpakai ? (meta.data?.daftar_bawaan ?? '') : '')
  const tab = tabs.find((candidate) => candidate.kode === active)

  // Daftar belum ditembak sama sekali selama belum ada yang dipilih.
  //
  // Bukan sekadar disembunyikan: permintaan yang tetap berjalan di balik layar membebani
  // basis data untuk jawaban yang tidak pernah dilihat siapa pun — dan di modul ini
  // kuerinya menyentuh puluhan juta baris.
  const list = useSalvageList(active, page, search, meta.isSuccess && active !== '')
  const exporting = useExportSalvage()

  // Permintaan rincian dipegang HALAMAN, bukan panelnya.
  //
  // Sebabnya: jawabannya menentukan APA yang digambar. Klaim yang belum punya pengajuan
  // masuk ke form "Menambahkan Data Salvage" — seperti di layar lama — bukan ke panel
  // rincian. Bila panelnya yang menembak server, keputusan itu diambil sesudah panelnya
  // terlanjur tergambar, dan permintaan yang sama berjalan dua kali.
  const detail = useSalvageDetail(opened?.key ?? 'pengajuan', opened?.reference ?? '')

  // Klaim tanpa pengajuan tidak dibuka sebagai panel, melainkan sebagai form pengajuan
  // baru yang sudah terisi klaimnya.
  const creatingFor =
    opened !== null && detail.data !== undefined && !detail.data.ada_pengajuan
      ? detail.data
      : null

  // Baris daftar "Rejected Checker" membuka FORM SUNTING, bukan panel baca.
  //
  // Daftar itu berisi pengajuan yang checker kembalikan kepada PIC, dan satu-satunya
  // tindakan yang masuk akal di sana adalah memperbaikinya lalu mengirim ulang. Daftar
  // mana yang berperilaku begitu datang dari SERVER (`membuka_form_sunting`), bukan
  // disimpulkan di sini dari kode tabnya — menyimpulkannya berarti daftar yang sama
  // hidup di dua tempat.
  const editingFor =
    opened !== null &&
    detail.data !== undefined &&
    detail.data.ada_pengajuan &&
    tab?.membuka_form_sunting === true
      ? detail.data
      : null

  /** Berpindah daftar mengembalikan ke halaman pertama DAN mengosongkan pencarian. */
  function selectTab(code: string) {
    setParams(
      (current) => {
        const next = new URLSearchParams(current)
        next.set('daftar', code)
        next.delete('halaman')

        // Kata kunci dibuang, bukan dibawa.
        //
        // Alasannya bukan kerapian melainkan arti pencarian yang BERBEDA antardaftar: tiga
        // daftar mencocokkan persis, sepuluh mencocokkan sebagian. Membawa kata kunci
        // "PNC-20" dari daftar Histori ke daftar Checker menghasilkan daftar kosong yang
        // terlihat seperti kerusakan.
        next.delete('cari')
        return next
      },
      { replace: true },
    )

    // Panel rincian ikut ditutup. Pengajuan yang sedang dibuka adalah milik daftar yang
    // baru saja ditinggalkan, dan membiarkannya terbuka di bawah daftar lain membuat
    // rincian itu terbaca seolah milik baris yang sekarang tampil.
    setOpened(null)
  }

  function setPage(next: number) {
    setParams(
      (current) => {
        const updated = new URLSearchParams(current)
        if (next <= 1) updated.delete('halaman')
        else updated.set('halaman', String(next))
        return updated
      },
      { replace: true },
    )
  }

  function setSearch(value: string) {
    setParams(
      (current) => {
        const updated = new URLSearchParams(current)
        if (value === '') updated.delete('cari')
        else updated.set('cari', value)

        // Mencari mengembalikan ke halaman pertama. Tanpa itu, pencarian yang cocok
        // dengan tiga baris pada halaman satu akan menampilkan halaman lima yang kosong.
        updated.delete('halaman')
        return updated
      },
      { replace: true },
    )
  }

  if (portal === null) {
    return (
      <Frame>
        <ErrorMessage
          title="Pilih entitas lebih dulu"
          description={
            'Pengajuan salvage milik satu badan hukum, dan aplikasi ini melayani empat. ' +
            'Pilih portal di bilah atas untuk membukanya.'
          }
          tone="gangguan"
        />
      </Frame>
    )
  }

  if (meta.isError) {
    return (
      <Frame>
        <ErrorMessage
          title="Layar tidak dapat dibuka"
          description={messageOf(meta.error)}
          tone="gangguan"
        />
      </Frame>
    )
  }

  return (
    <Frame
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Button type="button" onClick={() => setAdding(true)}>
            Tambah
          </Button>
          <Button
            type="button"
            tone="kedua"
            onClick={() => {
              // Ringkasan SELALU disegarkan; daftarnya hanya bila memang ada yang terbuka.
              //
              // Memanggil refetch pada permintaan yang sedang padam (`enabled: false`)
              // akan menjalankannya sekali — persis yang dihindari dengan tidak
              // menembaknya sejak awal.
              void counts.refetch()
              if (active !== '') void list.refetch()
            }}
            disabled={counts.isFetching || list.isFetching}
          >
            Refresh
          </Button>
        </div>
      }
    >
      {saved !== null && (
        <div
          className={
            saved.perluPerhatian
              ? 'rounded-kartu border border-amber-300 bg-amber-50 p-4 text-sm text-amber-900'
              : 'rounded-kartu border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-900'
          }
          role="status"
        >
          {saved.pesan}
        </div>
      )}
      <StatusSummary
        rows={counts.data?.baris ?? []}
        active={active}
        onSelect={selectTab}
        isLoading={counts.isLoading}
      />

      {editingFor !== null ? (
        <TambahSalvageForm
          // Kunci memaksa form DIBONGKAR saat berpindah pengajuan — sama alasannya
          // dengan jalur di bawah.
          key={editingFor.id_salvage}
          statusOptions={meta.data?.pilihan_status_salvage ?? []}
          currencyOptions={meta.data?.pilihan_mata_uang ?? []}
          uploadColumns={meta.data?.kolom_berkas_unggahan ?? []}
          editing={{
            id_salvage: editingFor.id_salvage,
            isian: toFormState(editingFor),
            item: toFormItems(editingFor),
          }}
          pilihan={{
            objek: editingFor.pilihan_objek,
            coverage: editingFor.pilihan_coverage,
          }}
          history={editingFor.riwayat}
          onClose={() => setOpened(null)}
          onSaved={(catatan) => {
            setSaved(catatan)
            setOpened(null)
          }}
        />
      ) : creatingFor !== null ? (
        <TambahSalvageForm
          // Kunci memaksa form DIBONGKAR dan dipasang ulang saat berpindah klaim.
          //
          // Tanpanya, isian yang sudah diketik untuk klaim sebelumnya akan tertinggal di
          // form yang sekarang menyebut klaim lain — pengajuan yang tersimpan atas klaim
          // yang salah, tanpa satu pun tanda.
          key={creatingFor.no_klaim}
          statusOptions={meta.data?.pilihan_status_salvage ?? []}
          currencyOptions={meta.data?.pilihan_mata_uang ?? []}
          uploadColumns={meta.data?.kolom_berkas_unggahan ?? []}
          prefill={{
            nomor_klaim: creatingFor.no_klaim,
            nama_object: creatingFor.nama_object,
            nama_coverage: creatingFor.nama_coverage,
            id_object: creatingFor.id_object,
            id_coverage: creatingFor.id_coverage,
          }}
          // Pilihan kedua autocomplete diteruskan dari jawaban yang SUDAH di tangan.
          //
          // Rincian klaim ini baru saja diambil untuk memutuskan form inilah yang
          // digambar, dan jawabannya memuat keduanya. Membiarkan form mencarinya sendiri
          // berarti permintaan kedua untuk jawaban yang sama.
          pilihan={{
            objek: creatingFor.pilihan_objek,
            coverage: creatingFor.pilihan_coverage,
          }}
          history={creatingFor.riwayat}
          onClose={() => setOpened(null)}
          onSaved={(catatan) => {
            setSaved(catatan)
            setOpened(null)
          }}
        />
      ) : adding ? (
        <TambahSalvageForm
          statusOptions={meta.data?.pilihan_status_salvage ?? []}
          currencyOptions={meta.data?.pilihan_mata_uang ?? []}
          uploadColumns={meta.data?.kolom_berkas_unggahan ?? []}
          onClose={() => setAdding(false)}
          onSaved={setSaved}
        />
      ) : active === '' ? (
        /*
          Belum ada status yang dipilih — daftarnya memang BELUM digambar.

          Kalimat ini menggantikan grid, bukan menemaninya. Layar yang menampilkan
          ringkasan lalu berhenti tanpa sepatah kata terbaca seperti layar yang gagal
          memuat; yang membedakan keduanya hanyalah satu kalimat yang menyebut apa yang
          harus dilakukan.
        */
        <p
          className="rounded-kartu border border-slate-200 bg-white p-4 text-sm text-slate-600"
          role="status"
        >
          Pilih salah satu <strong className="font-semibold">Status Salvage</strong> di atas
          untuk membuka daftarnya.
        </p>
      ) : (
        <>
          {/*
            TIDAK ada bilah tab di sini.

            Dulu ada, dan ia dihapus atas permintaan Work Owner (2026-10-03) karena
            menduakan navigasi yang sudah ada: tabel ringkas di atas layar inilah yang
            memilih daftar — di Pega pun begitu, lewat pasangan `Param.tipe`/`Param.tipe2`
            yang dikirim barisnya. Bilah tab menggambar tujuan yang sama untuk kedua
            kalinya, dengan nama yang tidak selalu sama pula.

            Akibatnya tabel ringkas menjadi SATU-SATUNYA jalan ke sebuah daftar. Yang
            menjaganya tetap begitu ada di sisi server:
            `TestEveryVisibleListIsReachableFromACounterRow` menolak daftar yang tidak
            disebut satu baris pencacah pun — tanpa itu, daftar baru dapat lahir dalam
            keadaan tidak dapat dibuka siapa pun.

            Nama daftar yang sedang terbuka tetap terbaca: barisnya ditandai di tabel
            ringkas, dan judulnya ada pada label grid di bawah.
          */}
          {tab && <p className="text-sm text-slate-600">{tab.keterangan}</p>}

          {/*
            Bilah kuning "catatan daftar" DICABUT (2026-10-08).

            Isinya bukan galat dan bukan peringatan — ia keterangan bahwa sebuah selisih
            terhadap Pega memang DISENGAJA. Digambar kuning, ia terbaca sebaliknya: setiap
            kali daftar dibuka, petugas disodori kotak berwarna peringatan yang
            memberitahunya bahwa tidak ada yang perlu dikhawatirkan.

            Keputusan Work Owner 2026-10-08: yang bukan galat tidak digambar. Ia mengikuti
            pencabutan panel selisih terencana (2026-10-06) atas alasan yang sama.

            Keterangannya TIDAK hilang dari sistem — `inboxsalvage.Tab.Notice` tetap hidup
            di Go, tempat uji kesetaraan gerbang 1 memetakannya ke butir `P-5` (`D-54`).
            Yang berubah hanyalah ia tidak lagi dikirim ke layar.
          */}

          <DataTable<SalvageRow>
            columns={columnsOf(tab, (key, reference) => setOpened({ key, reference }))}
            rows={list.data?.baris ?? []}
            rowKey={(row) => `${row.referensi}-${row.no_klaim}`}
            label={`Daftar ${tab?.nama ?? 'salvage'}`}
            isLoading={list.isLoading}
            error={
              list.isError ? (
                <ErrorMessage
                  title="Daftar tidak dapat dimuat"
                  description={messageOf(list.error)}
                  tone="gangguan"
                />
              ) : undefined
            }
            emptyMessage={emptyMessageFor(tab, '')}

            // Penjelasan "kenapa kosong padahal saya mencari" kini BENAR-BENAR sampai
            // ke layar. Sebelum isian ini ada, DataTable selalu menimpanya dengan
            // kalimat bawaannya sendiri begitu kotak pencarian terisi — justru pada
            // satu-satunya keadaan yang menjadi alasan kalimat ini ditulis.
            searchEmptyMessage={emptyMessageFor(tab, search)}
            searchLabel={tab?.label_pencarian ?? 'Cari'}
            serverSearch={{
              value: search,
              onChange: setSearch,
              matchCount: list.data?.paginasi.total,
            }}
            pagination={{
              page: list.data?.paginasi.halaman ?? 1,
              size: list.data?.paginasi.ukuran ?? 20,
              total: list.data?.paginasi.total ?? 0,
              totalPage: list.data?.paginasi.total_halaman ?? 1,
              onPageChange: setPage,
              isLoading: list.isFetching,
            }}
            actions={
              // Hanya Export Data yang tersisa di toolbar grid.
              //
              // Tambah dan Refresh naik ke kepala layar — keduanya tindakan tingkat layar
              // dan harus tetap dapat ditekan saat belum ada daftar yang dibuka. Export
              // TIDAK ikut naik, dan itu bukan kelalaian: ia mengunduh SALINAN DAFTAR YANG
              // SEDANG DILIHAT, sehingga tanpa daftar ia tidak punya arti.
              <Button
                type="button"
                tone="kedua"
                disabled={exporting.isPending}
                onClick={() => exporting.mutate({ tab: active, search })}
              >
                {exporting.isPending ? 'Menyiapkan…' : 'Export Data'}
              </Button>
            }
          />

          {opened !== null && (
            <DetailSalvagePanel
              detailKey={opened.key}
              reference={opened.reference}
              data={detail.data}
              isPending={detail.isPending}
              isError={detail.isError}
              error={detail.error}
              onClose={() => setOpened(null)}
            />
          )}

          {exporting.error != null && (
            <ErrorMessage
              title="Berkas ekspor tidak dapat diambil"
              description={messageOf(exporting.error)}
              tone="gangguan"
            />
          )}
        </>
      )}
    </Frame>
  )
}

/**
 * toFormState menerjemahkan rincian pengajuan menjadi isian form sunting.
 *
 * # Kenapa ia di sini, bukan di dalam form
 *
 * Karena ini pemetaan antara dua KONTRAK — bentuk jawaban rincian dan bentuk permintaan
 * simpan — dan keduanya memang berbeda nama. `estimasi` di satu sisi adalah
 * `minimum_salvage` di sisi lain, dan keduanya `PNC_SALVAGE.ESTIMASINILAI`. Menaruhnya di
 * dalam form berarti form harus mengenal bentuk rincian pula.
 *
 * Enam isian pada rincian TIDAK dipetakan — tanggal transfer GA, tanggal dan nomor
 * akseptasi, nama pemenang lelang, dan tanggal lelang. Keenamnya tidak digambar form ini,
 * dan mengirimkannya kembali berarti menuliskan ulang nilai yang tidak pernah terlihat
 * pengguna.
 *
 * # "Status Salvage" SENGAJA tidak diisi, dan itu bukan kelalaian
 *
 * Di layar lama pun kolom itu tetap `--Pilih--` saat pengajuan yang sudah ada dibuka.
 * Alasannya terbaca dari datanya: `STSTRANSFER` memikul dua arti yang BERTABRAKAN pada
 * kode yang sama. Nilai `4` berarti **"Waive Salvage"** di daftar pilihan ini
 * (`inboxsalvage.StatusOptions`), tetapi **"Rejected Checker"** sebagai penyaring daftar.
 *
 * Memetakannya akan memilihkan "Waive Salvage" untuk setiap pengajuan yang baru saja
 * ditolak checker — keputusan bernilai uang yang tidak pernah diambil siapa pun. Yang
 * dipilih PIC harus diketiknya sendiri.
 */
function toFormState(detail: DetailResponse) {
  return {
    nomor_klaim: detail.no_klaim,
    id_object: detail.id_object,
    nama_object: detail.nama_object,
    id_coverage: detail.id_coverage,
    nama_coverage: detail.nama_coverage,
    tanggal_input: detail.tanggal_input_salvage,
    jenis_salvage: detail.jenis_salvage,
    lokasi_salvage: detail.lokasi_salvage,
    lokasi_salvage_di_jabodetabek: detail.lokasi_salvage_di_jabodetabek,
    mata_uang: detail.mata_uang,
    minimum_salvage: detail.estimasi,
    quantity_salvage: detail.quantity_salvage,
    nilai_penawaran: detail.nilai_penawaran,
    remark: detail.remark,
    email: detail.email,
    nama_pic_survey: detail.nama_pic_survey,
    no_telp_pic_survey: detail.no_telp_pic_survey,
    email_pic_survey: detail.email_pic_survey,
  }
}

/**
 * toFormItems menerjemahkan grid barang rincian menjadi baris grid form.
 *
 * `total_nilai` dan ketiga isian lelang TIDAK dibawa: form ini tidak menggambarnya, dan
 * procedure penyimpan detail item pun tidak punya tempat untuk `total_nilai`.
 */
function toFormItems(detail: DetailResponse): DetailItem[] {
  return detail.barang.map((barang) => ({
    nama_item: barang.nama_barang,
    jumlah_item: String(barang.jumlah),
    satuan: barang.satuan,
    remark: barang.remark,
  }))
}

/** Frame adalah judul layar beserta ruang isinya. */
/**
 * Frame adalah kerangka tetap layar ini: judul, keterangan, dan tombol tingkat layar.
 *
 * # Kenapa tombolnya di SINI, bukan di toolbar grid
 *
 * Karena sejak 2026-10-08 gridnya TIDAK digambar sampai sebuah status dipilih, dan tombol
 * yang hidup di dalam toolbar grid ikut hilang bersamanya. "Tambah" adalah tindakan
 * tingkat layar — ia tidak menambah baris pada daftar yang sedang dibuka, ia membuat
 * pengajuan baru — sehingga meletakkannya di dalam daftar sudah keliru bahkan sebelum
 * gridnya disembunyikan.
 *
 * Letaknya di kanan judul mengikuti layar Pega apa adanya (`D-13`).
 */
function Frame({ children, actions }: { children: ReactNode; actions?: ReactNode }) {
  return (
    <div className="space-y-5">
      <header className="flex flex-wrap items-start justify-between gap-3">
        <div>
          <h1 className="text-lg font-semibold text-slate-900">
            Inbox Salvage Asuransi Sinarmas
          </h1>
          <p className="mt-1 text-sm text-slate-600">
            Pengelolaan barang sisa klaim — dari penandaan awal sampai penjualan lewat balai
            lelang.
          </p>
        </div>

        {actions}
      </header>

      {children}
    </div>
  )
}

/**
 * columnsOf menerjemahkan kolom yang DIKIRIM SERVER menjadi kolom DataTable.
 *
 * # Kenapa kolomnya tidak ditulis di sini
 *
 * Karena tiga belas daftar memakai EMPAT susunan kolom yang berbeda, dan dua di antaranya
 * berbeda hanya satu kolom. Menuliskannya dengan tangan di layar berarti daftar yang sama
 * hidup di dua tempat — dan yang satu akan tertinggal saat yang lain diperbaiki.
 *
 * Yang tetap milik layar adalah cara satu sel DIGAMBAR: tanggal diformat, nilai uang
 * diratakan kanan. Keduanya urusan tampilan, bukan urusan bentuk data.
 */
function columnsOf(
  tab: Tab | undefined,
  onOpenDetail: (key: DetailKey, reference: string) => void,
): Column<SalvageRow>[] {
  if (!tab) return []

  const columns: Column<SalvageRow>[] = tab.kolom.map((column) => ({
    key: column.kunci,
    title: column.judul,
    value: (row) => valueOf(row, column.kunci),
    render: (row) => renderCell(row, column),
    alignRight: column.angka,
  }))

  // Kolom aksi ada di KETIGA BELAS daftar — itu keadaan di layar lama, tempat sebelas
  // tombol `SetDataDetailSalvage_act` tersebar di seluruh grid-nya.
  //
  // Yang berbeda hanyalah KUNCI yang dikirimkannya: ID pengajuan pada tujuh daftar yang
  // barisnya pengajuan, nomor klaim pada enam daftar yang barisnya klaim dan tidak
  // membawa ID pengajuan sama sekali.
  columns.push({
    key: 'aksi',
    title: 'Aksi',
    width: '8rem',
    noSort: true,
    alignRight: true,
    value: () => '',
    render: (row) => {
      const reference =
        tab.kunci_rincian === 'klaim' ? row.no_klaim : row.id_salvage

      if (reference === '') return <span className="text-slate-400">—</span>

      return (
        <Button
          type="button"
          tone="halus"
          onClick={() => onOpenDetail(tab.kunci_rincian, reference)}
        >
          Detail
        </Button>
      )
    },
  })

  return columns
}

/** valueOf mengambil isi satu sel sebagai TEKS — yang dicari dan diurutkan. */
function valueOf(row: SalvageRow, key: string): string {
  const cell = (row as unknown as Record<string, unknown>)[key]
  return typeof cell === 'string' ? cell : ''
}

/**
 * renderCell menggambar satu sel.
 *
 * Tiga perlakuan khusus, dan ketiganya urusan tampilan:
 *
 *   tanggal      diformat mengikuti kebiasaan layar lain
 *   nilai uang   diratakan kanan dengan angka tabular, supaya kolomnya sejajar
 *   sel kosong   digambar sebagai tanda hubung, bukan ruang kosong
 *
 * Yang terakhir penting di layar ini: kolom "Catatan" SELALU kosong, dan sel kosong tanpa
 * tanda tidak dapat dibedakan dari kolom yang gagal dimuat.
 */
function renderCell(row: SalvageRow, column: TabColumn): ReactNode {
  const raw = valueOf(row, column.kunci)
  if (raw === '') return <span className="text-slate-400">—</span>

  if (column.kunci === 'tanggal_input' || column.kunci === 'tanggal_kejadian') {
    return formatDate(raw)
  }

  if (column.angka) {
    return <span className="tabular-nums">{formatMoney(raw)}</span>
  }

  return raw
}

/**
 * formatMoney menggambar nilai uang dengan pemisah ribuan.
 *
 * Nilainya datang sebagai TEKS dan tetap teks sampai di sini — `D-51` menetapkan nilai
 * uang disimpan presisi penuh dan hanya dibulatkan SAAT DITAMPILKAN. Inilah tempat
 * "saat ditampilkan" itu.
 *
 * Nilai yang tidak terbaca sebagai angka digambar apa adanya, bukan diganti nol. Nol yang
 * tidak dapat dibedakan dari kegagalan pembacaan adalah kelas cacat yang sama dengan
 * `GETSELISIHJAM` (`D-49` butir 10).
 */
function formatMoney(raw: string): string {
  const parsed = Number(raw.replace(',', '.'))
  if (!Number.isFinite(parsed)) return raw
  return parsed.toLocaleString('id-ID', { maximumFractionDigits: 2 })
}

/**
 * emptyMessageFor menjelaskan MENGAPA daftarnya kosong, bukan sekadar menyatakan kosong.
 *
 * Pada daftar yang pencariannya cocok persis, kekosongan hampir selalu berarti pengguna
 * mengetik separuh nomor klaim — dan tanpa keterangan itu, ia akan menyimpulkan datanya
 * hilang.
 */
function emptyMessageFor(tab: Tab | undefined, search: string): string {
  if (search !== '' && tab?.pencarian_cocok_persis === true) {
    return (
      `Tidak ada yang cocok dengan "${search}". Pencarian di daftar ini COCOK PERSIS — ` +
      'ketik nomor klaim atau nama PIC selengkapnya, bukan sebagiannya.'
    )
  }
  if (search !== '') {
    return `Tidak ada pengajuan yang cocok dengan "${search}".`
  }
  return 'Belum ada pengajuan pada daftar ini.'
}

function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
