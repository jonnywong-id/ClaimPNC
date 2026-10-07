import { createContext, useContext, useRef, useState } from 'react'
import type { ReactNode } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import {
  useBukaDokumen,
  useRCLPUCLClaim,
  useRCLPUCLDocuments,
  useTindakanKlaim,
  useKategoriDokumen,
  useUnggahDokumen,
} from './api'
import type { IsianPenerimaanDokumen, TindakanKlaim } from './api'
import type { ClaimDetailResponse } from './types'

/**
 * Layar kerja RCL/PUCL — section `SendtoRCLPUCL`.
 *
 * # Dari mana bentuknya
 *
 * Dari ketiga rule yang ditambahkan Work Owner pada 2026-09-24, dibaca langsung:
 *
 *	Section/SendtoRCLPUCL-Section.xml                kontainer BER-TAB (`pyHeaderType
 *	                                                 TABBED`); dua tab, Lampiran Surat
 *	                                                 lalu Penerimaan Dokumen
 *	Section/SectionLampiranSuratPUCL-Section.xml     13 isian + tombol "Cetak"
 *	Section/SectionPenerimaanDokumenPUCL-Section.xml grid + 3 isian + 4 tombol
 *	Flow Action/SendtoRCLPUCL-FA.xml                 pra-proses
 *	                                                 `SetDataLampiranSuratRCLPUCL_Act`
 *
 * Urutan isian dan **judulnya** diambil dari `pyLabelFieldValue` tiap sel, bukan dikarang
 * dan bukan disalin dari judul kolom grid (`D-13`).
 *
 * # Kenapa HALAMAN, bukan panel di dalam antrean
 *
 * Karena di Pega ia memang layar tujuan: mengklik Nomor Case menjalankan Open Assignment,
 * dan klaimnya terbuka pada tahap alur kerjanya. Versi pertama modul ini menggambarnya
 * sebagai panel di bawah tabel; itu keliru sejak `SendtoRCLPUCL` diterima, karena panel
 * menyiratkan "pratinjau baris" sementara yang dibuka adalah TAHAP KERJA klaim.
 *
 * Antreannya tetap dapat dikembalikan utuh: tab dan nomor halaman dibawa di alamat, jadi
 * tombol kembali mendarat di tempat yang sama — bukan di tab pertama halaman pertama.
 *
 * # Sembilan isian hidup di CLIPBOARD Pega
 *
 * Bukan "kolomnya belum ditemukan" — Work Owner menjelaskan 2026-09-24 bahwa kesembilannya
 * diambil dari clipboard objek kerja (`.ClaimData.PUCLStatus.NIK` dan seterusnya). Properti
 * clipboard yang tidak dioptimasi memang tidak punya kolom sendiri, sehingga tidak ada yang
 * dapat dibaca kueri biasa selama objek kerjanya masih dimiliki Pega.
 *
 * Dua perlakuan buruk yang dihindari: menghilangkannya membuat layar tampak setara padahal
 * tidak, dan menggambarnya sebagai sel kosong membuat nilai yang ADA tetapi tak terbaca
 * tidak dapat dibedakan dari isian yang memang belum diisi.
 *
 * Yang dipakai: digambar di TEMPATNYA, bertanda "di clipboard Pega".
 */
export function SendtoRCLPUCLPage() {
  const { referensi } = useParams<{ referensi: string }>()
  const [params] = useSearchParams()
  const navigate = useNavigate()

  const key = referensi ? decodeURIComponent(referensi) : ''
  const claim = useRCLPUCLClaim(key || null)

  // `?? null` bukan kerapian. `callAPI` mengembalikan `null` — bukan melempar — untuk
  // jawaban 200 yang badannya BUKAN JSON, misalnya saat alamat `/api/...` dijawab penyaji
  // SPA dengan `index.html`. Tanpa penanganan, keadaan itu menghasilkan halaman yang
  // benar-benar kosong: bukan memuat, bukan galat, bukan data. Itu kelas kegagalan yang
  // paling sulit dilaporkan pengguna, karena tidak ada satu pun yang dapat disebutkan.
  const detail = claim.data ?? null

  const state = screenStateOf(claim.isError, claim.isPending, detail !== null)

  // Tab dan halaman dibawa kembali apa adanya. Petugas yang membuka klaim dari halaman
  // ketiga tab "Kelengkapan Dokumen" harus mendarat di sana lagi.
  const back = params.toString()
    ? `/inbox-rcl-pucl?${params.toString()}`
    : '/inbox-rcl-pucl'

  return (
    <div className="mx-auto max-w-5xl px-4 py-6">
      <header className="flex flex-wrap items-start justify-between gap-3 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">
            {detail ? `Klaim ${detail.no_case}` : 'Layar kerja RCL/PUCL'}
          </h1>
          <p className="mt-1 text-sm text-slate-600">
            Lampiran Surat dan Penerimaan Dokumen — layar kerja jalur RCL/PUCL.
          </p>
        </div>
        <Button tone="kedua" onClick={() => navigate(back)}>
          Kembali ke antrean
        </Button>
      </header>

      <StateNotice state={state} error={claim.error} claimKey={key} />

      {/*
        Layarnya digambar SELALU, apa pun keadaan datanya.
        Dua alasan, dan keduanya lebih kuat daripada kerapian. Pertama, bentuk layar ini
        tidak bergantung pada data — ia tetap Lampiran Surat dan Penerimaan Dokumen dengan
        isian yang sama, dan sembilan di antaranya memang tidak pernah terisi. Kedua, halaman
        yang tidak menggambar apa pun tidak dapat dibedakan dari halaman yang rusak.
      */}
      <WorkScreen detail={detail} />
    </div>
  )
}

/** Keadaan pengambilan isi layar kerja. */
type ScreenState = 'memuat' | 'galat' | 'kosong' | 'siap'

/** Menurunkan keadaan layar dari keadaan pengambilan; galat lebih dulu, lalu memuat. */
function screenStateOf(isError: boolean, isPending: boolean, hasDetail: boolean): ScreenState {
  if (isError) return 'galat'
  if (isPending) return 'memuat'
  return hasDetail ? 'siap' : 'kosong'
}

/** Teks pilihan kosong pada kolom kategori, menurut keadaan pengambilan daftarnya. */
function categoryPlaceholder(isPending: boolean, isError: boolean): string {
  if (isPending) return 'Memuat kategori…'
  if (isError) return 'Kategori gagal dimuat — muat ulang halaman'
  return 'Select..'
}

/**
 * Sebaris keterangan keadaan, digambar DI ATAS layar — bukan menggantikannya.
 *
 * Versi pertama halaman ini mengganti seluruh badan dengan pesan memuat atau kotak galat.
 * Akibatnya satu keadaan yang tidak terduga — `data` bernilai `null` — menghasilkan halaman
 * yang benar-benar kosong, tanpa satu pun petunjuk. Yang dipakai sekarang: layarnya tetap
 * tergambar, keadaannya dinyatakan di atasnya.
 */
function StateNotice({
  state,
  error,
  claimKey,
}: Readonly<{
  state: ScreenState
  error: unknown
  claimKey: string
}>) {
  if (state === 'siap') return null

  if (state === 'memuat') {
    return <p className="mt-4 text-sm text-slate-600">Memuat isi layar kerja…</p>
  }

  if (state === 'galat') {
    return (
      <div className="mt-4">
        <ErrorMessage
          title="Isi klaim tidak dapat diambil"
          description={messageOf(error)}
          tone="gangguan"
        />
      </div>
    )
  }

  // 'kosong' — permintaannya selesai tanpa galat, tetapi tidak membawa data.
  //
  // Sebab yang paling sering: jawaban peladen bukan JSON, misalnya `index.html` yang
  // dikembalikan penyaji SPA untuk alamat `/api/...` yang tidak dilayani. Dikatakan apa
  // adanya beserta apa yang harus diperiksa — pengguna yang melihat layar kosong tidak
  // punya cara lain mengetahuinya.
  return (
    <div className="mt-4">
      <ErrorMessage
        title="Isi klaim tidak terbaca"
        description={
          'Permintaan selesai tanpa galat, tetapi jawabannya tidak memuat data klaim. ' +
          'Layar di bawah digambar kosong supaya bentuknya tetap terlihat. Periksa apakah ' +
          'alamat /api/inbox-rcl-pucl/klaim/' +
          (claimKey === '' ? '…' : claimKey) +
          ' benar-benar dilayani peladen aplikasi, bukan dijawab penyaji halaman.'
        }
        tone="gangguan"
      />
    </div>
  )
}

/**
 * Tanda yang menggantikan nilai pada isian yang hidup di clipboard Pega.
 *
 * Ia BUKAN "kolomnya belum ketemu". Work Owner menjelaskan 2026-09-24 bahwa kesembilan isian
 * ini adalah properti clipboard pada objek kerja — `.ClaimData.PUCLStatus.NIK` dan
 * seterusnya — dan properti clipboard yang tidak dioptimasi memang tidak punya kolom sendiri.
 * Mencarinya lagi ke DBA tidak akan menemukannya.
 */
const DI_CLIPBOARD = Symbol('tersimpan di clipboard Pega')

type FieldValue = string | typeof DI_CLIPBOARD

/** Kedua bagian layar kerja, sebagaimana Pega menamainya. */
type WorkTab = 'lampiran' | 'penerimaan'

/**
 * Kedua bagian layar, digambar menurut section-nya.
 *
 * # Ia DUA TAB, bukan dua bagian bertumpuk
 *
 * Versi sebelumnya menggambar keduanya berurutan pada satu halaman. Itu keliru, dan
 * buktinya ada di kontainernya sendiri:
 *
 *	Section/SendtoRCLPUCL-Section.xml
 *	  <pyHeaderType>TABBED</pyHeaderType>
 *	  <pyTabbedHeader>true</pyTabbedHeader>
 *	  <pyInclude>SectionLampiranSuratPUCL</pyInclude>
 *	  <pyInclude>SectionPenerimaanDokumenPUCL</pyInclude>
 *
 * Bedanya bukan kosmetik. Bertumpuk, petugas melihat tombol "Cetak" dan tombol "Kirim Ke
 * Analyst" pada satu layar sekaligus — dua tindakan yang di Pega berada di tahap yang
 * berbeda dan tidak pernah terlihat bersamaan.
 *
 * # Kenapa tab-nya TIDAK dibawa ke alamat
 *
 * Karena alamat halaman ini sudah membawa tab ANTREAN, yang dipakai tombol kembali. Dua
 * pengertian "tab" pada satu alamat akan membuat yang satu menimpa yang lain, dan akibatnya
 * tombol kembali mendarat di antrean yang keliru.
 */
function WorkScreen({ detail }: Readonly<{ detail: ClaimDetailResponse | null }>) {
  const [tab, setTab] = useState<WorkTab>('lampiran')

  // Nama tombol yang panelnya sedang terbuka. Satu nilai untuk seluruh layar — lihat
  // WriteActionPanel.
  const [openAction, setOpenAction] = useState<string | null>(null)

  // Tab kedua disembunyikan untuk klaim berstatus Notification, mengikuti syarat pada
  // kontainernya di layar lama. Keputusannya milik SERVER — lihat `types.ts`.
  //
  // Selama isinya belum tiba, tab kedua dianggap ADA: menyembunyikannya lebih dulu akan
  // membuat tab berkedip muncul-hilang pada setiap klaim yang dibuka, dan kedipan itu
  // terbaca sebagai kerusakan.
  //
  // Disembunyikan HANYA bila server menyatakannya `false` secara tegas — bukan bila
  // isiannya sekadar tidak ada. Jawaban yang kehilangan isian ini akan menyembunyikan
  // separuh layar tanpa satu pun galat, dan kegagalan seperti itu tidak dapat dilaporkan
  // penggunanya: yang terlihat hanyalah tab yang tidak pernah ada.
  const showsReceipt = detail?.tab_penerimaan_dokumen_tampil !== false

  // Klaim Notification yang dibuka saat tab kedua sedang terpilih dikembalikan ke tab
  // pertama, bukan dibiarkan menggambar tab yang seharusnya tidak ada.
  const active: WorkTab = showsReceipt ? tab : 'lampiran'

  return (
    <>
      <WorkTabs
        active={active}
        onChange={(next) => {
          setTab(next)
          setOpenAction(null)
        }}
        showsReceipt={showsReceipt}
      />

      {!showsReceipt && (
        <p className="mt-3 text-xs text-slate-500">
          Klaim berstatus <span className="font-medium">Notification</span> hanya memiliki
          Lampiran Surat. Layar lama menyembunyikan tab &ldquo;Penerimaan Dokumen&rdquo;
          untuk klaim seperti ini, karena pemberitahuan tidak menunggu dokumen dan tidak
          dikirim kembali ke Analyst.
        </p>
      )}

      {/*
        SATU panel keterangan untuk seluruh layar, bukan satu per tombol.

        Sebelumnya tiap tombol menyimpan keadaannya sendiri, sehingga menekan dua tombol
        membuka DUA panel sekaligus — dan karena alasannya sama untuk semua tombol, kalimat
        yang sama tergambar dua kali berturut-turut. Work Owner melaporkannya 2026-10-01.

        Keadaannya karena itu diangkat ke sini: membuka satu panel menutup yang lain dengan
        sendirinya. Berpindah tab juga menutupnya — panel milik tombol yang sudah tidak
        terlihat tidak boleh ikut terbawa.
      */}
      <WriteActionPanel.Provider
        value={{
          open: openAction,
          toggle: (label) => setOpenAction((v) => (v === label ? null : label)),
        }}
      >
        {active === 'lampiran' ? (
          <LetterTab detail={detail} />
        ) : (
          <ReceiptTab key={detail?.referensi ?? 'kosong'} detail={detail} />
        )}
      </WriteActionPanel.Provider>
    </>
  )
}

/**
 * Panel keterangan tindakan yang sedang terbuka — paling banyak SATU di seluruh layar.
 *
 * Dipakai lewat context, bukan prop, karena tombolnya tersebar di dua tab dan empat kelompok.
 * Mengalirkannya sebagai prop berarti enam perantara yang tidak memakainya sendiri, dan tiap
 * perantara adalah satu tempat yang dapat lupa meneruskannya.
 */
const WriteActionPanel = createContext<{
  open: string | null
  toggle: (label: string) => void
}>({ open: null, toggle: () => {} })

/**
 * Kepala tab layar kerja.
 *
 * Judulnya persis `pyCaption` kedua sub-section: "Lampiran Surat" dan "Penerimaan Dokumen"
 * (`D-13`).
 */
function WorkTabs({
  active,
  onChange,
  showsReceipt,
}: Readonly<{
  active: WorkTab
  onChange: (tab: WorkTab) => void
  showsReceipt: boolean
}>) {
  const tabs: { key: WorkTab; label: string }[] = [
    { key: 'lampiran', label: 'Lampiran Surat' },
    ...(showsReceipt
      ? [{ key: 'penerimaan' as const, label: 'Penerimaan Dokumen' }]
      : []),
  ]

  return (
    <div
      role="tablist"
      aria-label="Bagian layar kerja RCL/PUCL"
      className="mt-4 flex gap-1 border-b border-slate-200"
    >
      {tabs.map((item) => (
        <button
          key={item.key}
          role="tab"
          type="button"
          aria-selected={active === item.key}
          onClick={() => onChange(item.key)}
          className={[
            'rounded-t-kontrol px-4 py-2 text-sm font-medium transition-colors',
            'duration-150 ease-halus focus:outline-none',
            'focus-visible:ring-2 focus-visible:ring-blue-500/50',
            active === item.key
              ? 'border-b-2 border-blue-600 text-blue-700'
              : 'border-b-2 border-transparent text-slate-500 hover:text-slate-800',
          ].join(' ')}
        >
          {item.label}
        </button>
      ))}
    </div>
  )
}

/**
 * Susunan tombol ketika detailnya belum tiba: TIDAK ADA satu pun.
 *
 * Bukan "semuanya" — menggambar tombol lebih dulu lalu menghilangkannya begitu data tiba
 * membuat layar berkedip, dan sekejap menyatakan tindakan yang ternyata tidak tersedia untuk
 * klaim itu.
 */
const NO_BUTTONS: ClaimDetailResponse['tombol'] = {
  download_dokumen: false,
  tutup_klaim: false,
  unggah_dokumen: false,
  lihat_dokumen: false,
  save: false,
  tolak_klaim: false,
  kirim_ke_analyst: false,
  kirim_ke_pic_teknik: false,
}

/**
 * Bagian pertama — `SectionLampiranSuratPUCL`.
 *
 * # Tombolnya DUA, dan sempat salah satu pun tidak benar
 *
 * Versi sebelumnya menggambar satu tombol bernama **"Cetak"**. Nama itu tidak ada di
 * section-nya: `"cetak"` di sana adalah NILAI PARAMETER (`<pyName>tipe</pyName>`), bukan
 * caption tombol. Work Owner melaporkannya 2026-10-01, dan penelusuran membenarkan laporan
 * itu — ketiga sel `pxButton` section ini ber-caption **Pilih** (pemilih Perihal),
 * **Download Dokumen**, dan **Tutup Klaim**.
 */
function LetterTab({ detail }: Readonly<{ detail: ClaimDetailResponse | null }>) {
  const letter = detail?.lampiran_surat
  const buttons = detail?.tombol ?? NO_BUTTONS

  return (
    <>
      {/*
        Dua isian teratas FULL WIDTH, sisanya berpasangan dua kolom — persis susunan
        section-nya, dan persis yang terlihat di layar Pega.

        Urutan pasangannya bukan urutan yang paling rapi dibaca melainkan urutan section:
        kiri UP · No Polis · Nama Peserta · Perihal, kanan No Kontrak · Business Unit /
        Seksi · Jumlah Tagihan · Tanggal Kejadian. Petugas yang membandingkan kedua layar
        berdampingan menelusurinya dari atas ke bawah.
      */}
      {/* Judulnya TIDAK diulang di sini: kepala tab sudah menamainya, dan Pega pun tidak
          mengulangnya di dalam tab. */}
      <Panel>
        {/* Keduanya TANPA kotak — di Pega pun keduanya teks polos di bawah judulnya. */}
        <StackedFields
          fields={[
            ['Status RCL / PUCL / MSIG', letter?.rcl_pucl ?? ''],
            ['Catatan dari Analyst', letter?.deskripsi_analyst ?? ''],
          ]}
        />

        <FieldGroup
          fields={[
            ['UP', letter?.up ?? '', false, 'kotak'],
            ['No Kontrak', DI_CLIPBOARD, false, 'kotak'],
            ['No Polis', letter?.no_polis ?? '', false, 'kotak'],
            ['Business Unit / Seksi', DI_CLIPBOARD, false, 'kotak'],
            ['Nama Peserta', letter?.nama_peserta ?? '', false, 'kotak'],
            ['Jumlah Tagihan', letter?.jumlah_tagihan ?? '', false, 'kotak'],
            ['Perihal', letter?.perihal ?? '', false, 'pilihan'],
            ['Tanggal Kejadian', letter?.tanggal_kejadian ?? '', false, 'tanggal'],
          ]}
        />

        {/*
        Ketiga Keterangan FULL WIDTH, bukan dua kolom: ketiganya `pxTextArea` di section dan
        isinya kalimat surat yang panjang — di Pega masing-masing memenuhi satu baris penuh.
      */}
        <StackedFields
          fields={[
            ['Keterangan Pembuka', letter?.keterangan_pembuka ?? '', false, 'area'],
            ['Keterangan Isi', letter?.keterangan_isi ?? '', false, 'area'],
            ['Keterangan Penutup', letter?.keterangan_penutup ?? '', false, 'area'],
          ]}
        />

        {/*
        "Perihal" bukan isian bebas melainkan PILIHAN dari master
        `POOLDATA.M_PERIHAL_RCLPUCL` — 12 baris, dibaca langsung 2026-09-30.

        # Yang disimpan adalah TEKSNYA, bukan kodenya — dan itu kini terbukti

        `Activity/InputPerihalRCLPUCL_act-Act.xml` diterima 2026-10-01 dan menetapkan:

            primary.ClaimData.PUCLStatus.Perihal := pyReportContentPage.pxResults(1).PERIHAL_NAME

        `ID_PERIHAL` hanya mampir ke halaman sementara `TempPerihal`, lalu dibuang
        `Page-Remove` di langkah terakhir. Jadi klaim TIDAK menyimpan kunci masternya —
        hanya teks yang terpilih saat itu.

        Dua akibat yang perlu diingat saat master Perihal kelak dibangun: tidak ada kunci
        asing yang dapat ditelusuri balik, dan mengubah teks sebuah baris master TIDAK
        mengubah klaim yang sudah memakainya. Keduanya perilaku sistem lama apa adanya.

        Kalimat di bawah ini sebelumnya menyebut Perihal "ada di clipboard". Itu sudah tidak
        benar sejak kolom `PERIHAL` ditemukan (§84.2), dan sudah diperbaiki.
      */}
        <p className="mt-2 text-xs text-slate-500">
          <span className="font-medium">Perihal</span> dipilih dari daftar baku berisi 12
          pilihan (master Perihal RCL/PUCL). Yang tersimpan pada klaim adalah teks
          pilihannya, dan itulah yang digambar di atas.
        </p>

        {/*
        Isian yang paling mudah dilaporkan sebagai kerusakan, padahal BUKAN: "UP" berisi nama
        objek, sama dengan "Nama Peserta", karena kedua penetapan di activity Pega menunjuk
        ekspresi yang sama — dan Work Owner menegaskan itu memang benar. Dinyatakan di
        tempat, bukan hanya di dokumen, karena di sinilah pengguna akan bertanya.
      */}
        {letter && letter.up !== '' && letter.up === letter.nama_peserta && (
          <p className="mt-2 text-xs text-slate-500">
            Kolom <span className="font-medium">UP</span> berisi nama objek yang sama
            dengan Nama Peserta. Itu bukan kekeliruan tampilan — keduanya memang diisi
            dari sumber yang sama di sistem lama.
          </p>
        )}

        {/*
        KEDUA tombol ini saling meniadakan — yang satu jalur non-MSIG, yang lain jalur MSIG
        ber-Notification — sehingga satu klaim tidak pernah menampilkan keduanya. Syaratnya
        dihitung di server; layar hanya menggambar apa yang dikirimkannya.
      */}
        {/* Tombolnya di KANAN BAWAH panel, mengikuti layar lama. */}
        <div className="flex flex-wrap justify-end gap-3">
          {/*
          Ia melakukan DUA hal sekaligus, dan keduanya perlu disebut.

          Rangkaiannya `InsertMitraPA(tipe="cetak")` lalu `PUCLPost`. Di dalam `PUCLPost`:
          `AttachAsPDFC` (HTMLToPDF → AttachToWork → View) MEMBUAT dan membuka PDF suratnya,
          sementara `TanggalCetakDokumenPUCL` ikut terisi — kolom yang MEMINDAHKAN klaim dari
          tab "Cetak Surat" ke "Kelengkapan Dokumen".

          Keterangan sebelumnya menyatakan tombol ini "tidak mengunduh apa pun". ITU SALAH,
          dan arahnya berbahaya ke sisi sebaliknya: ia membuat orang mengira suratnya tidak
          terbit. Yang benar — ia menerbitkan surat DAN memindahkan klaimnya, sehingga bukan
          tombol lihat-lihat.

          `ASMForceCaseClose` TIDAK berlaku untuknya: syarat `param.Status==""` ber-`true=3`,
          yaitu melewati langkah itu. Ia hanya jalan untuk "Tolak Klaim" di jalur RCL.
        */}
          {buttons.download_dokumen && (
            <ClaimAction
              label="Download Dokumen"
              aksi="cetak"
              reference={detail?.referensi}
              note={
                'Menerbitkan surat RCL/PUCL, melampirkannya ke klaim, lalu mengunduhnya. ' +
                'Klaimnya berpindah dari tab "Cetak Surat" ke "Kelengkapan Dokumen".'
              }
            />
          )}
          {buttons.tutup_klaim && (
            <WriteAction
              label="Tutup Klaim"
              caseNumber={detail?.referensi}
              note="Menutup klaim MSIG — satu-satunya tindakan yang tersedia pada jalur ini."
            />
          )}
        </div>
      </Panel>

      <ScreenFooter detail={detail} actionCount="Tindakan di atas" />
    </>
  )
}

/**
 * Bagian kedua — `SectionPenerimaanDokumenPUCL`.
 *
 * # Section-nya memuat TUJUH tombol, bukan empat
 *
 * Empat yang digambar sebelumnya benar namanya, tetapi tiga di antara tujuh belum ada, dan
 * satu digambar TANPA SYARAT padahal syaratnya menentukan siapa yang menerima klaimnya:
 *
 * | Tombol               | Syarat `pyCondition`                 |
 * |----------------------|--------------------------------------|
 * | Unggah Dokumen       | `ALWAYS`                             |
 * | Lihat Dokumen        | `ALWAYS`                             |
 * | Save                 | `ALWAYS`                             |
 * | Tolak Klaim          | `RCL_PUCL = 1`                       |
 * | Kirim Ke Analyst     | `RCL_PUCL = 2 && IsPA`               |
 * | Kirim ke PIC Teknik  | `RCL_PUCL = 2 && IsTravel`           |
 * | Reminder PUCL        | `1==2` — **tidak pernah digambar**   |
 *
 * # "Reminder PUCL" sengaja TIDAK digambar
 *
 * Syaratnya `1==2`, yang tidak pernah benar. Ia tombol yang dimatikan dengan cara dikarang
 * syaratnya alih-alih dihapus — jejak yang lazim pada sistem berumur panjang. Perilaku yang
 * ditiru adalah perilakunya yang NYATA: tidak muncul. Menggambarnya "supaya lengkap" akan
 * menambah tindakan yang tidak pernah ada di layar lama (`P-5`).
 *
 * Hal yang sama berlaku pada sel "Tolak Klaim" KEDUA, yang syaratnya juga `1==2`.
 */
function ReceiptTab({ detail }: Readonly<{ detail: ClaimDetailResponse | null }>) {
  const receipt = detail?.penerimaan_dokumen
  const buttons = detail?.tombol ?? NO_BUTTONS

  // Kedua isian WAJIB disimpan sebagai keadaan layar, bukan dibaca langsung dari `detail`.
  //
  // Alasannya tombol "Save": ia mengirim apa yang DIKETIK petugas, dan nilai yang diketik
  // belum ada di `detail` sampai tersimpan. Menggambarnya langsung dari `detail` membuat
  // isiannya tidak dapat diubah sama sekali — persis keadaan sampai 2026-10-01.
  //
  // Penyemaiannya memakai KUNCI REMOUNT, bukan useEffect: pemanggil memberi `key` berisi
  // kunci klaim, sehingga komponen ini lahir kembali — beserta nilai awalnya — setiap kali
  // klaim yang dibuka berganti atau isinya baru tiba. useEffect yang menyamakan keadaan
  // dengan prop akan menimpa ketikan petugas setiap kali data disegarkan di latar.
  const [note, setNote] = useState(receipt?.komentar_pucl ?? '')
  const [completeAt, setCompleteAt] = useState(
    untukIsianWaktu(receipt?.tanggal_kelengkapan_dokumen ?? ''),
  )

  return (
    <>
      <Panel>
        <ReceivedDocumentGrid detail={detail} />

        {/*
        Ketiganya FULL WIDTH di section, dan dua di antaranya WAJIB diisi (`pyRequired`
        true). Tanda wajibnya dibawa meski layar ini hanya membaca: ia menyatakan bentuk
        layar lama, dan petugas yang membandingkan keduanya berdampingan mencarinya.
      */}
        {/* Judulnya ada di kepala tab, sama seperti di Pega. */}
        {/*
          "Email Tertanggung" HANYA-BACA, dua isian di bawahnya DAPAT DIKETIK.

          Pembedaannya bukan selera melainkan tempat datanya: kedua isian wajib hidup di
          `TC_PNC_PUCL`, tabel milik aplikasi ini, sementara Email Tertanggung hidup di
          `T_CLAIM_PNC.EMAIL_LOD` — tabel lain yang modul ini tidak tulis.

          Menggambarnya dapat diketik lalu diam-diam tidak menyimpannya jauh lebih buruk
          daripada menggambarnya jelas tidak dapat diketik.

          Ia akan sering tergambar kosong: kolomnya ada tetapi belum pernah terisi pada
          seluruh baris produksi. Itu keadaan data, bukan isian yang hilang.
        */}
        <StackedFields
          fields={[
            ['Email Tertanggung', receipt?.email_tertanggung ?? '', false, 'kotak'],
          ]}
        />

        <EditableField
          label="Tanggal Kelengkapan Dokumen"
          required
          value={completeAt}
          onChange={setCompleteAt}
          type="datetime-local"
        />

        <EditableField
          label="Catatan untuk Analyst"
          required
          value={note}
          onChange={setNote}
          type="area"
        />

        {/*
        Susunan tombolnya mengikuti layar lama: "Unggah Dokumen" dan "Lihat Dokumen" di KIRI
        bawah, "Save" dan tombol Kirim di KANAN bawah, pada BARIS YANG SAMA.

        Sebelumnya keenamnya tergambar sebagai dua baris rata kiri. Petugas yang membandingkan
        kedua layar berdampingan mencari tombol Kirim di sudut kanan, dan tidak menemukannya.
      */}
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div className="flex flex-wrap gap-3">
            {buttons.unggah_dokumen && (
              <UploadDocumentAction reference={detail?.referensi} />
            )}
            {/*
          SATU-SATUNYA tombol layar ini yang benar-benar berjalan.

          Ia MEMBACA — `localAction ViewAttachmentPUCL` di Pega hanya membuka daftar lampiran,
          tanpa menyentuh satu pun tabel. Karena itu ia tidak terhalang `P-1`, dan dibangun
          lebih dulu atas persetujuan Work Owner 2026-10-01.
        */}
            {buttons.lihat_dokumen && (
              <ViewDocumentsAction reference={detail?.referensi} />
            )}
          </div>

          {/*
        Kelompok KANAN. "Save" ikut di sini, bukan bersama kedua tombol dokumen, karena di
        layar lama ia berdampingan dengan tombol Kirim — keduanya menutup pekerjaan, sementara
        kedua tombol kiri mengurus lampirannya.

        Ketiga tombol Kirim/Tolak BERSYARAT, dan paling banyak SATU yang muncul: "Tolak Klaim"
        hanya di jalur RCL, dan kedua tombol "Kirim" hanya di jalur PUCL — yang satu untuk PA,
        yang lain untuk Travel. Klaim PUCL pada lini selain keduanya tidak menampilkan satu pun.
      */}
          <div className="flex flex-wrap justify-end gap-3">
            {buttons.save && (
              <ClaimAction
                label="Save"
                aksi="save"
                reference={detail?.referensi}
                isian={{
                  catatan_untuk_analyst: note,
                  tanggal_kelengkapan_dokumen: completeAt,
                }}
                note="Menyimpan kedua isian di atas tanpa meneruskan klaimnya."
              />
            )}
            {buttons.tolak_klaim && (
              <ClaimAction
                label="Tolak Klaim"
                aksi="tolak"
                reference={detail?.referensi}
                note="Menolak klaim. Hanya tersedia pada jalur RCL."
              />
            )}
              {/*
          Kedua tombol Kirim MEMBAWA kedua isian di atas, sama seperti "Save".

          Di Pega ketiganya mem-posting form yang sama: Finish Assignment mengirim seluruh
          isian flow action, sehingga `PUCLPost` langkah 10 dapat menuliskan `KomentarPUCL`
          bersama penandaan klaimnya. Tombol Kirim yang tidak membawa isian akan MEMBUANG
          catatan yang baru saja diketik petugas — dan Analyst menerima klaim tanpa tahu apa
          yang berubah.

          Keduanya juga `pyRequired`, sehingga peladen MENOLAK yang kosong. Yang memeriksanya
          peladen, bukan layar: aturannya hidup di satu tempat.

          Suratnya ikut terbit dan melampir ke klaim — `AttachAsPDFC` berprekondisi `1==1` —
          tetapi TIDAK dibuka di tab baru. Hanya "Download Dokumen" yang mengunduh.
        */}
            {buttons.kirim_ke_analyst && (
              <ClaimAction
                label="Kirim Ke Analyst"
                aksi="kirim-analyst"
                warna="oranye"
                reference={detail?.referensi}
                isian={{
                  catatan_untuk_analyst: note,
                  tanggal_kelengkapan_dokumen: completeAt,
                }}
                bukaBerkas={false}
                note="Menyimpan kedua isian, melampirkan surat, lalu meneruskan klaim kembali ke Analyst."
              />
            )}
            {buttons.kirim_ke_pic_teknik && (
              <ClaimAction
                label="Kirim ke PIC Teknik"
                aksi="kirim-pic-teknik"
                warna="oranye"
                reference={detail?.referensi}
                isian={{
                  catatan_untuk_analyst: note,
                  tanggal_kelengkapan_dokumen: completeAt,
                }}
                bukaBerkas={false}
                note="Menyimpan kedua isian, melampirkan surat, lalu meneruskan klaim ke PIC Teknik. Jalur PUCL pada lini Travel."
              />
            )}
          </div>
        </div>
      </Panel>

      <ScreenFooter detail={detail} actionCount="Tindakan di atas" />
    </>
  )
}

/**
 * Kaki layar — kunci klaim, sifat baca-saja, dan daftar isian clipboard.
 *
 * Dipakai KEDUA tab, bukan hanya salah satunya. Petugas yang membuka tab "Penerimaan
 * Dokumen" lebih dulu tetap membutuhkan kunci klaimnya untuk mengerjakan tindakannya di
 * Pega, dan tetap perlu tahu tombolnya mati karena keputusan — bukan karena rusak.
 */
function ScreenFooter({
  detail,
  actionCount,
}: Readonly<{
  detail: ClaimDetailResponse | null
  actionCount: string
}>) {
  return (
    <>
      {/*
        Ditampilkan dengan sengaja, meski ia BUKAN isian di section mana pun: ia yang dipakai
        petugas membuka klaim yang sama di Pega untuk mengerjakan tindakannya.

        LABELNYA BERUBAH 2026-10-01, karena ISINYA berubah. Sebelumnya ia kunci teknis Pega
        (`ASM-FW-GCNMFW-WORK PNC-1865`) — parameter `inskey` yang dikirim tautan aslinya.
        Sejak layar ini membaca POOLDATA.TC_PNC_PUCL, yang tersedia hanyalah NOMOR CASE
        (`PNC-1865`): tabel datar itu tidak menyimpan kunci teknisnya.

        Labelnya disesuaikan alih-alih dibiarkan, karena petugas MENYALIN nilai ini ke Pega.
        Memanggilnya "kunci klaim" sementara isinya nomor case akan membuat ia ditempelkan ke
        tempat yang tidak menerimanya, lalu dilaporkan sebagai kerusakan.
      */}
      {detail && (
        <p className="mt-6 border-t border-slate-200 pt-3 text-xs text-slate-600">
          Nomor Case: <span className="font-mono text-slate-900">{detail.referensi}</span>
        </p>
      )}

      {detail?.tindakan_masih_di_pega && (
        <div className="mt-3 rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2">
          <p className="text-xs text-slate-700">
            <span className="font-medium">Layar ini baca saja.</span> {actionCount}{' '}
            MENYIMPAN data, dan selama Pega dan sistem baru berjalan berdampingan data
            klaim hanya boleh diubah dari satu sistem. Kerjakan tindakannya di Pega, cari
            klaimnya dengan Nomor Case di atas.
          </p>
        </div>
      )}

      {detail && detail.isian_belum_terpetakan.length > 0 && (
        <div className="mt-3 rounded-kontrol border border-slate-200 bg-white px-3 py-2">
          <p className="text-xs font-medium text-slate-700">
            Isian yang tersimpan di clipboard Pega
          </p>
          <p className="mt-1 text-xs text-slate-600">
            {detail.isian_belum_terpetakan.join(' · ')}
          </p>
          <p className="mt-1 text-xs text-slate-500">
            Kesembilannya properti clipboard pada objek kerja Pega, bukan kolom tabel,
            sehingga nilainya tidak dapat dibaca dari sini selama objek kerjanya masih
            dimiliki Pega. Isian ini tetap digambar di tempatnya supaya keadaannya
            terlihat.
          </p>
        </div>
      )}
    </>
  )
}

/**
 * Grid "Tanggal terima Dokumen" — dua kolomnya `.DateReceived` dan `.Remarks`.
 *
 * Ia digambar sebagai kerangka kosong, bukan dihilangkan. Grid berulang adalah satu-satunya
 * bentuk di layar ini yang tidak dapat diwakili sebuah isian, dan menghapusnya akan
 * menyembunyikan bahwa layar lama memuat DAFTAR di sini — bukan satu tanggal.
 */
function ReceivedDocumentGrid({ detail }: Readonly<{ detail: ClaimDetailResponse | null }>) {
  const partial = detail?.penerimaan_dokumen?.tanggal_terima_dokumen_sebagian === true

  // Barisnya KEADAAN LAYAR, bukan dibaca langsung dari `detail`.
  //
  // Di Pega, "✚ Tambah" dan "Hapus" adalah operasi SISI KLIEN pada page list di clipboard —
  // terverifikasi dari section: `pyAction addRow` tanpa satu pun `pyActivity`, dan `deleteRow`
  // tidak ada sama sekali. Tidak ada permintaan ke peladen saat keduanya ditekan; yang
  // menyimpannya adalah tombol "Save", lewat `Obj-Save`.
  //
  // Keadaannya disemai lewat KUNCI REMOUNT pada pemanggilnya, sama seperti kedua isian di
  // bawah grid — lihat ReceiptTab.
  const [rows, setRows] = useState<ReceivedRow[]>(() =>
    (detail?.penerimaan_dokumen?.tanggal_terima_dokumen ?? []).map((row) => ({
      tanggal: untukIsianWaktu(row.tanggal),
      keterangan: row.keterangan,
    })),
  )

  function ubah(index: number, bagian: Partial<ReceivedRow>) {
    setRows((lama) => lama.map((row, i) => (i === index ? { ...row, ...bagian } : row)))
  }

  return (
    // Grid-nya punya BINGKAINYA SENDIRI di layar lama — kotak di dalam kotak, bukan sekadar
    // tabel yang mengambang di atas isian di bawahnya.
    <div className="rounded-sm border border-slate-300 p-4">
      <h3 className="text-[13px] font-semibold text-slate-800">Tanggal terima Dokumen</h3>

      {/*
        Keduanya TAUTAN BIRU di layar lama, bukan tombol berbingkai — "✚ Tambah" dan "Hapus"
        berdampingan tepat di atas tabelnya.

        Keduanya BEKERJA, dan itu bukan kelonggaran: di Pega pun keduanya tidak memanggil apa
        pun. Catatan lama di tempat ini menyatakan keduanya "MENULIS ke objek kerja yang masih
        dimiliki Pega" — itu TIDAK terbukti, dan dicabut 2026-10-02.

        "Hapus" membuang baris TERAKHIR. Pega membuang baris yang sedang dipilih, dan grid ini
        belum punya pemilihan baris; membuang yang terakhir memasangkannya dengan "Tambah"
        yang menyisipkan di belakang (`pyPosition AFTER`).
      */}
      <div className="mt-3 flex flex-wrap items-center gap-4">
        <button
          type="button"
          onClick={() => setRows((lama) => [...lama, { tanggal: '', keterangan: '' }])}
          className="text-sm font-medium text-blue-700 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50"
        >
          ✚ Tambah
        </button>
        <button
          type="button"
          disabled={rows.length === 0}
          onClick={() => setRows((lama) => lama.slice(0, -1))}
          className="text-sm font-medium text-blue-700 hover:underline focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50 disabled:cursor-not-allowed disabled:text-slate-400 disabled:no-underline"
        >
          Hapus
        </button>
      </div>

      <table className="mt-2 w-full border-collapse text-sm">
        <thead>
          <tr className="border-b border-slate-300 text-left text-[13px] text-slate-800">
            <th className="w-1/2 py-1.5 font-semibold">Tanggal</th>
            <th className="py-1.5 font-semibold">Keterangan</th>
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 ? (
            <tr className="border-b border-slate-200">
              {/*
                "Data Tidak Ada" — kalimat layar lama APA ADANYA (`D-13`), bukan kalimat kami
                sendiri. Yang sebelumnya tertulis di sini menjelaskan keadaannya dengan lebih
                panjang, dan justru karena itu tidak lagi terbaca sebagai layar yang sama.
              */}
              <td colSpan={2} className="py-2 text-sm text-slate-400">
                Data Tidak Ada
              </td>
            </tr>
          ) : (
            rows.map((row, index) => (
              // Kuncinya INDEKS, bukan isinya. Baris baru lahir kosong, sehingga dua baris
              // kosong akan berbagi kunci yang sama bila isinya dipakai — dan React lalu
              // menggambar ulang isian yang sedang diketik.
              <tr key={index} className="border-b border-slate-100">
                <td className="py-1.5 pr-3">
                  <input
                    type="datetime-local"
                    aria-label={`Tanggal baris ${index + 1}`}
                    value={row.tanggal}
                    onChange={(e) => ubah(index, { tanggal: e.target.value })}
                    className="w-full rounded-sm border border-slate-300 bg-white px-2 py-1 text-sm text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                  />
                </td>
                <td className="py-1.5">
                  <input
                    type="text"
                    aria-label={`Keterangan baris ${index + 1}`}
                    value={row.keterangan}
                    onChange={(e) => ubah(index, { keterangan: e.target.value })}
                    className="w-full rounded-sm border border-slate-300 bg-white px-2 py-1 text-sm text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                  />
                </td>
              </tr>
            ))
          )}
        </tbody>
      </table>

      {/*
        DUA keterangan, dan keduanya menyatakan hal yang berbeda.

        Yang pertama datang dari SERVER dan menyangkut apa yang TERBACA; yang kedua menyangkut
        apa yang TERSIMPAN. Menggabungkannya akan membuat batas pembacaan dan batas penyimpanan
        terbaca sebagai satu masalah, padahal pemiliknya berbeda.
      */}
      {partial && rows.length > 0 && (
        <p className="mt-1 text-[11px] leading-snug text-slate-500">
          Baru baris pertama yang terbaca. Bila di Pega ada baris berikutnya, ia belum
          muncul di sini.
        </p>
      )}

      <p className="mt-1 text-[11px] leading-snug text-amber-700">
        Perubahan pada daftar ini <span className="font-medium">belum tersimpan</span> —
        ia hilang saat layar dimuat ulang. Tempat simpannya menunggu tabel yang sudah
        diminta (<span className="font-mono">TC_PNC_PUCL_TERIMA_DOKUMEN</span>).
      </p>
    </div>
  )
}

/** Satu baris grid "Tanggal terima Dokumen", sebagaimana diketik di layar. */
type ReceivedRow = {
  // Berbentuk `YYYY-MM-DDTHH:mm`, yaitu yang diterima `datetime-local`.
  tanggal: string
  keterangan: string
}

/**
 * Tombol tindakan yang MENULIS.
 *
 * Digambar, tetapi tidak dapat ditekan. Alasannya ditulis di sebelahnya, bukan disembunyikan
 * di balik pesan yang baru muncul setelah ditekan: tombol mati tanpa keterangan terbaca
 * sebagai kerusakan, dan petugas akan menekannya berulang kali.
 */
/**
 * Tombol "Unggah Dokumen" — membuka dialog `SetUploadDocPUCL`.
 *
 * # Bentuknya dari mana
 *
 * Dari flow action `SetUploadDocPUCL`, yang merantai empat section:
 *
 *	SetUploadDoc_Detl -> ASMAttachContentScreen -> ASMAttachFilesScreen
 *	                  -> ASMAttachments -> ASMAttachFileList
 *
 * Yang terakhir memuat grid-nya: page list `dragDropFileUpload.pxResults`, berkolom **Name**
 * (`pxTextInput`), **File**, dan **Category** (`pxDropdown`), beserta ikon buang per baris.
 *
 * # Dua hal yang membedakannya dari versi pertama modul ini
 *
 * Versi pertama (2026-10-02 pagi) mengunggah SEKETIKA begitu berkas dipilih, tanpa dialog.
 * Itu lebih sedikit langkah, tetapi BUKAN bentuk layar lama — dan `D-13` menetapkan tata
 * letak mengikuti Pega supaya petugas tidak perlu belajar ulang.
 *
 * Yang hilang karenanya ada dua, dan keduanya nyata: berkas dapat dipilih BANYAK sekaligus,
 * dan namanya dapat diubah SEBELUM dikirim. Keduanya kembali di sini.
 */
function UploadDocumentAction({ reference }: Readonly<{ reference?: string | undefined }>) {
  const [terbuka, setTerbuka] = useState(false)

  return (
    <div className="mt-4">
      <Button tone="utama" disabled={!reference} onClick={() => setTerbuka(true)}>
        Unggah Dokumen
      </Button>
      <p className="mt-1 text-xs text-slate-500">Melampirkan berkas dokumen ke klaim.</p>

      {/*
        Dialognya dilahirkan hanya saat terbuka, bukan disembunyikan dengan CSS: keadaannya —
        daftar berkas yang dipilih — harus bersih setiap kali dibuka, dan kelahiran ulang
        menjaminnya tanpa satu baris kode pembersih.
      */}
      {terbuka && reference && (
        <UploadDocumentDialog reference={reference} onClose={() => setTerbuka(false)} />
      )}
    </div>
  )
}

/** Satu baris pada grid dialog unggah. */
type UploadRow = {
  berkas: File
  // Nama yang DAPAT diketik ulang — kolom "Name". Bawaannya nama berkasnya sendiri.
  nama: string
  // Pilihan kolom "Category" — nama kategori lampiran.
  //
  // Kosong berarti "belum disentuh petugas", BUKAN "tanpa kategori": yang berlaku lalu
  // bawaannya, persis seperti di Pega. Lihat `bawaanKategori`.
  kategori: string
}

function UploadDocumentDialog({
  reference,
  onClose,
}: Readonly<{
  reference: string
  onClose: () => void
}>) {
  const pilih = useRef<HTMLInputElement>(null)
  const [rows, setRows] = useState<UploadRow[]>([])
  const [gagal, setGagal] = useState<string | null>(null)
  const unggah = useUnggahDokumen(reference)
  const kategori = useKategoriDokumen()

  /**
   * Kategori bawaan, mengikuti Pega.
   *
   * Layar lama menggambar **"File"** terpilih pada baris yang belum disentuh — itulah
   * kategori lampiran bawaan Pega, dan 37 baris lampiran klaim PNC memang tersimpan
   * dengannya.
   *
   * Catatan saya sebelumnya menyatakan bawaannya sengaja dikosongkan supaya tidak menebak.
   * Itu keliru: bukan tebakan, melainkan perilaku yang terbukti — dan dicabut 2026-10-02.
   *
   * Dipilih dari daftar yang BENAR-BENAR termuat, bukan ditulis mati: menampilkan nilai
   * terpilih yang tidak ada di dalam daftarnya membuat dropdown tergambar kosong.
   */
  const bawaanKategori =
    (kategori.data?.kategori ?? []).find((pilihan) => pilihan.nilai === 'File')?.nilai ??
    ''

  const kategoriBaris = (row: UploadRow) => row.kategori || bawaanKategori

  async function submit() {
    setGagal(null)
    try {
      // Berurutan, bukan serentak. Ketiga pernyataan basis data tiap unggahan mengambil nomor
      // dari sequence yang sama; mengirimnya serentak tidak salah, tetapi membuat urutan
      // nomor lampiran tidak lagi sejalan dengan urutan yang dilihat petugas di dialog.
      for (const row of rows) {
        await unggah.mutateAsync({
          berkas: row.berkas,
          nama: row.nama,
          kategori: kategoriBaris(row),
        })
      }
      onClose()
    } catch (error) {
      // Dialognya TIDAK ditutup saat gagal: berkas yang sudah dipilih akan hilang, dan
      // petugas harus memilihnya lagi satu per satu tanpa tahu mana yang sudah masuk.
      setGagal(messageOf(error))
    }
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="SetUploadDocPUCL"
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 p-4"
    >
      <div className="flex max-h-full w-full max-w-3xl flex-col rounded-sm border border-slate-300 bg-white shadow-xl">
        {/* Kepala: judul flow action-nya APA ADANYA, beserta silang penutup. */}
        <div className="flex items-center justify-between border-b border-slate-200 px-5 py-3">
          <h2 className="text-[15px] text-slate-800">SetUploadDocPUCL</h2>
          <button
            type="button"
            aria-label="Tutup"
            onClick={onClose}
            className="text-xl leading-none text-slate-400 hover:text-slate-700"
          >
            ×
          </button>
        </div>

        <div className="flex-1 overflow-auto px-5 py-8">
          <div className="flex justify-center">
            <button
              type="button"
              onClick={() => pilih.current?.click()}
              className="rounded-sm border border-blue-400 px-3 py-1 text-sm text-blue-700 hover:bg-blue-50 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50"
            >
              Select file(s)
            </button>
          </div>

          {/*
            `multiple` mengikuti layar lama — kendalinya `pzMultiFilePath`, dan judul tombolnya
            sendiri berbunyi "file(s)".

            `value` dikosongkan sesudah dipilih supaya berkas yang SAMA dapat dipilih lagi;
            tanpa itu `change` tidak terpicu pada pilihan kedua dan tombolnya terlihat rusak.
          */}
          <input
            ref={pilih}
            type="file"
            multiple
            className="hidden"
            aria-label="Berkas yang diunggah"
            onChange={(e) => {
              const dipilih = Array.from(e.target.files ?? [])
              e.target.value = ''
              setRows((lama) => [
                ...lama,
                ...dipilih.map((berkas) => ({
                  berkas,
                  nama: berkas.name,
                  kategori: '',
                })),
              ])
            }}
          />

          {rows.length > 0 && (
            <table className="mt-6 w-full border-collapse text-sm">
              <thead>
                <tr className="text-left text-[13px] font-semibold text-slate-800">
                  <th className="w-1/3 pb-1">Name</th>
                  <th className="w-1/3 pb-1">File</th>
                  <th className="pb-1">Category</th>
                  <th className="w-8 pb-1" />
                </tr>
              </thead>
              <tbody>
                {rows.map((row, index) => (
                  // Kuncinya INDEKS: dua berkas bernama sama dapat dipilih bersamaan, dan
                  // kunci berbasis nama akan membuat React menggambar ulang isian yang sedang
                  // diketik.
                  <tr key={index}>
                    <td className="py-1 pr-2">
                      <input
                        type="text"
                        aria-label={`Name baris ${index + 1}`}
                        value={row.nama}
                        onChange={(e) =>
                          setRows((lama) =>
                            lama.map((r, i) =>
                              i === index ? { ...r, nama: e.target.value } : r,
                            ),
                          )
                        }
                        className="w-full rounded-sm border border-slate-300 bg-sky-50 px-2 py-1 text-sm text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                      />
                    </td>
                    <td className="truncate py-1 pr-2 text-sm text-slate-700">
                      {row.berkas.name}
                    </td>
                    <td className="py-1 pr-2">
                      {/*
                        Daftarnya KATEGORI LAMPIRAN, bukan jenis dokumen — `AcceptanceNote`,
                        `ClaimFaceSheet`, `LOD`, dan seterusnya. Yang mendefinisikannya rule
                        `Rule-Obj-AttachmentCategory`, dan tipe rule itu tidak ada di export
                        sama sekali (`R-16`), sehingga daftarnya diturunkan dari kategori yang
                        benar-benar dipakai lampiran klaim PNC — 30 baris.

                        Catatan di tempat ini sempat menyebut `V_LST_DET_TYPE_DOC` dengan 159
                        barisnya. Itu salah sasaran: view itu memuat nama DOKUMEN, bukan
                        kategori lampiran. Dicabut 2026-10-02.
                      */}
                      <select
                        aria-label={`Category baris ${index + 1}`}
                        className="w-full rounded-sm border border-slate-300 bg-sky-50 px-2 py-1 text-sm text-slate-900 focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30"
                        value={kategoriBaris(row)}
                        onChange={(e) =>
                          setRows((lama) =>
                            lama.map((r, i) =>
                              i === index ? { ...r, kategori: e.target.value } : r,
                            ),
                          )
                        }
                      >
                        {/*
                          "Select.." adalah teks layar lama APA ADANYA (`D-13`), bukan kalimat
                          kami sendiri. Pilihannya TETAP ADA sesudah daftarnya termuat:
                          kategori tidak wajib di layar lama, dan menghapusnya memaksa petugas
                          memilih sesuatu yang mungkin tidak ia ketahui.
                        */}
                        <option value="">
                          {categoryPlaceholder(kategori.isPending, kategori.isError)}
                        </option>
                        {(kategori.data?.kategori ?? []).map((pilihan) => (
                          <option key={pilihan.nilai} value={pilihan.nilai}>
                            {pilihan.nama}
                          </option>
                        ))}
                      </select>
                    </td>
                    <td className="py-1 text-right">
                      <button
                        type="button"
                        aria-label={`Buang baris ${index + 1}`}
                        onClick={() =>
                          setRows((lama) => lama.filter((_, i) => i !== index))
                        }
                        className="text-slate-400 hover:text-red-600"
                      >
                        🗑
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}

          {gagal && (
            <p className="mt-3 rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-slate-700">
              {gagal}
            </p>
          )}
        </div>

        {/* Kaki: Cancel di KIRI, Submit di KANAN — dan Submit berwarna oranye, seperti di Pega. */}
        <div className="flex items-center justify-between border-t border-slate-200 bg-slate-100 px-5 py-3">
          <Button tone="kedua" onClick={onClose} disabled={unggah.isPending}>
            Cancel
          </Button>
          <OrangeButton
            disabled={rows.length === 0 || unggah.isPending}
            onClick={() => void submit()}
          >
            {unggah.isPending ? 'Mengunggah…' : 'Submit'}
          </OrangeButton>
        </div>
      </div>
    </div>
  )
}

/**
 * Tombol "Lihat Dokumen" — tombol layar kerja yang BENAR-BENAR berjalan.
 *
 * # Kenapa daftarnya baru ditarik saat ditekan
 *
 * Karena lampiran tidak selalu diperiksa. Layar kerja dibuka setiap kali nomor case diklik;
 * dokumennya dilihat sebagian. Menariknya bersama isi layar akan membebani setiap pembukaan
 * klaim demi sebagian kecil yang membutuhkannya.
 *
 * # Kenapa berkasnya dibuka lewat TAUTAN, bukan diambil JavaScript
 *
 * Karena isinya berkas, bukan JSON. Peramban sudah tahu cara menampilkan PDF dan gambar;
 * Isinya DIAMBIL lewat JavaScript, lalu diserahkan ke tab baru sebagai blob.
 *
 * Catatan sebelumnya di tempat ini menyatakan sebaliknya — bahwa berkasnya cukup ditautkan
 * supaya tidak melewati memori halaman. Itu mengabaikan satu hal yang menentukan: alamatnya
 * menuntut header, dan navigasi peramban tidak membawa header. Tautan semacam itu tidak
 * pernah membuka dokumennya. Dicabut 2026-10-02.
 *
 * Biaya yang disadari: berkasnya memang melewati memori halaman. Penampil bawaan peramban
 * TETAP dipakai — yang diserahkan ke tab baru adalah blob beserta jenis isinya.
 */
function ViewDocumentsAction({ reference }: Readonly<{ reference?: string | undefined }>) {
  const [open, setOpen] = useState(false)
  const documents = useRCLPUCLDocuments(reference ?? null, open)
  const bukaDokumen = useBukaDokumen(reference ?? null)

  return (
    <div className="mt-4">
      <Button
        tone="kedua"
        aria-expanded={open}
        disabled={!reference}
        onClick={() => setOpen((v) => !v)}
      >
        Lihat Dokumen
      </Button>
      <p className="mt-1 text-xs text-slate-500">
        Membuka dokumen yang sudah dilampirkan.
      </p>

      {open && (
        <div className="mt-2 max-w-xl rounded-kontrol border border-slate-200 bg-white px-3 py-2">
          {documents.isPending && (
            <p className="text-xs text-slate-600">Memuat daftar dokumen…</p>
          )}

          {documents.isError && (
            <p className="text-xs text-rose-700">
              Daftar dokumen tidak dapat diambil. {messageOf(documents.error)}
            </p>
          )}

          {/*
            Daftar KOSONG dan daftar GAGAL dibedakan dengan tegas. Keduanya terlihat sama di
            layar bila tidak dinyatakan, padahal yang satu berarti "klaim ini memang belum
            berdokumen" dan yang lain "jangan percayai layar ini".
          */}
          {documents.data && documents.data.dokumen.length === 0 && (
            <p className="text-xs text-slate-600">
              Belum ada dokumen yang terlampir pada klaim ini.
            </p>
          )}

          {documents.data && documents.data.dokumen.length > 0 && (
            <ul className="divide-y divide-slate-100">
              {documents.data.dokumen.map((document) => (
                <li
                  key={document.id}
                  className="flex flex-wrap items-baseline gap-x-3 py-1.5"
                >
                  {/*
                    TOMBOL, bukan tautan — dan itu bukan pilihan gaya.

                    Alamat isinya menuntut header `Authorization` dan `X-Portal`, sedangkan
                    navigasi peramban tidak membawa header apa pun. Tautan `<a href>` karena
                    itu selalu membuka `{"kode":"sesi_tidak_sah"}` mentah, tidak pernah
                    dokumennya. Diperbaiki 2026-10-02.

                    Tab-nya dibuka DI SINI, masih di dalam penanganan klik: `window.open`
                    sesudah `await` diblokir penghalang pop-up.
                  */}
                  <button
                    type="button"
                    onClick={() => {
                      const tab = window.open('', '_blank')
                      void bukaDokumen
                        .mutateAsync({
                          id: document.id,
                          nama: document.nama || document.id,
                          tab,
                        })
                        .catch(() => {
                          // Pesannya digambar dari `bukaDokumen.error` di bawah daftar;
                          // yang ditangkap di sini hanya penolakan yang tidak tertangani.
                        })
                    }}
                    className="text-left text-sm font-medium text-biru-700 underline underline-offset-2 focus:outline-none focus-visible:ring-2 focus-visible:ring-blue-500/50"
                  >
                    {document.nama || document.id}
                  </button>
                  <span className="text-xs text-slate-500">
                    {[document.kategori, document.sub_kategori]
                      .filter(Boolean)
                      .join(' · ')}
                  </span>
                  <span className="ml-auto text-xs text-slate-500">
                    {document.diunggah_pada}
                  </span>
                </li>
              ))}
            </ul>
          )}

          {/*
            Keterangan ketidaklengkapan datang dari SERVER, bukan ditulis di sini: begitu
            jalur lampiran bawaan Pega ikut terbaca, kalimatnya hilang di satu tempat.
          */}
          {/*
            Kegagalan membuka SATU dokumen dinyatakan terpisah dari kegagalan menarik
            DAFTARNYA. Keduanya berbeda sebabnya, dan menggabungkannya membuat petugas
            mengira seluruh daftarnya tidak dapat dipercaya padahal hanya satu berkas yang
            bermasalah.
          */}
          {bukaDokumen.isError && (
            <p className="mt-2 rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-slate-700">
              {messageOf(bukaDokumen.error)}
            </p>
          )}

          {/*
            Yang DISARING dinyatakan apa adanya, lengkap dengan alasannya.

            Daftar ini mengikuti penyaring `GCNMGetAllAttachments` — report definition di
            balik tombol yang sama di Pega. Lampiran berkategori kode angka tidak pernah
            tergambar di sana, dan sejak 2026-10-02 tidak tergambar di sini pula.

            Jumlahnya disebut, bukan disembunyikan: daftar yang diam-diam lebih pendek
            membuat petugas mencari dokumen yang sebenarnya ada.
          */}
          {(documents.data?.disaring ?? 0) > 0 && (
            <p className="mt-2 border-t border-slate-100 pt-2 text-[11px] leading-snug text-slate-500">
              {documents.data?.disaring} lampiran lama tidak ditampilkan — seluruhnya
              tidak tergambar di layar Pega pula.
            </p>
          )}

          {documents.data?.catatan && (
            <p className="mt-2 text-[11px] leading-snug text-slate-500">
              {documents.data.catatan}
            </p>
          )}
        </div>
      )}
    </div>
  )
}

/**
 * Tombol yang benar-benar MENJALANKAN tindakannya lewat Pega.
 *
 * # Kenapa ia tidak memakai panel WriteAction
 *
 * Karena ia bukan penolakan melainkan tindakan. Yang perlu digambar adalah keadaannya —
 * sedang berjalan, berhasil, atau gagal beserta alasannya — bukan langkah pengganti.
 *
 * Saat GAGAL karena layanan Pega belum tersambung, pesan peladen sudah memuat langkah
 * penggantinya ("kerjakan di Pega; salin nomor case"), sehingga tidak diulang di sini.
 *
 * Pesan BERHASIL pun datang dari peladen, bukan ditulis di sini: ia menyebut AKIBATNYA —
 * "klaimnya berpindah ke tab Kelengkapan Dokumen" — dan akibat itu berbeda per tindakan.
 */
function ClaimAction({
  label,
  note,
  aksi,
  reference,
  isian,
  warna = 'biru',
  bukaBerkas = true,
}: Readonly<{
  label: string
  note: string
  aksi: TindakanKlaim
  reference?: string | undefined

  // isian diisi TIGA tombol — "Save" dan kedua tombol Kirim — karena ketiganya membawa apa
  // yang DIKETIK petugas. "Download Dokumen" dan "Tolak Klaim" memanggil tanpa badan, dan
  // peladen pun hanya membacanya untuk ketiga yang pertama (lihat `carriesReceipt`).
  isian?: IsianPenerimaanDokumen | undefined

  // bukaBerkas menyatakan surat yang terbit juga DIBUKA di tab baru.
  //
  // Hanya "Download Dokumen" yang membukanya — namanya pun berjanji begitu. Kedua tombol
  // Kirim menerbitkan surat yang sama dan MELAMPIRKANNYA, persis seperti `AttachAsPDFC` di
  // Pega, tetapi tidak mengunduhkannya: petugas yang menekan "Kirim" bermaksud meneruskan
  // klaim, bukan membuka berkas. Suratnya tetap dapat diambil lewat "Lihat Dokumen".
  bukaBerkas?: boolean

  // Warna tombolnya mengikuti layar lama (`D-13`), bukan nada sistem desain kami.
  //
  // Di Pega, "Kirim Ke Analyst" ORANYE sementara "Save" dan "Download Dokumen" biru tua.
  // Perbedaan itu bukan hiasan: oranye menandai tindakan yang MEMINDAHKAN klaim ke tangan
  // orang lain, dan petugas mengenalinya dari warna sebelum membaca tulisannya.
  //
  // `Button` tidak diberi nada baru untuk ini. Nada adalah kosakata sistem desain yang
  // dipakai puluhan layar, dan oranye di sini hanya berlaku karena layar lama memakainya.
  warna?: 'biru' | 'oranye'
}>) {
  const tindakan = useTindakanKlaim(reference ?? null, aksi)
  const bukaDokumen = useBukaDokumen(reference ?? null)

  /**
   * Menjalankan tindakannya, lalu MENGUNDUH berkas yang dihasilkannya bila diminta.
   *
   * TIGA tindakan menghasilkan berkas — "Download Dokumen" dan kedua tombol Kirim — karena
   * `PUCLPost` memanggil `AttachAsPDFC` berprekondisi `1==1`. Peladen menerbitkan surat
   * RCL/PUCL, melampirkannya ke klaim, dan mengembalikan baris lampirannya.
   *
   * Yang MEMBUKANYA hanya "Download Dokumen"; lihat `bukaBerkas`.
   *
   * Unduhannya menempuh alamat ISI DOKUMEN yang sudah ada, bukan alamat baru yang
   * mengembalikan berkas dari jalur tindakan. Dengan begitu surat yang baru terbit dan surat
   * lama diambil lewat jalan yang sama persis — termasuk pemeriksaan kepemilikan klaimnya.
   */
  async function jalankan() {
    const hasil = await tindakan.mutateAsync(isian)
    const surat = hasil.dokumen
    if (!surat || !bukaBerkas) return

    // Tab dibuka lebih dulu, masih di dalam rantai klik — lihat useBukaDokumen.
    const tab = window.open('', '_blank')
    await bukaDokumen
      .mutateAsync({ id: surat.id, nama: surat.nama || surat.id, tab })
      .catch(() => {
        // Suratnya SUDAH melampir ke klaim; gagal membukanya bukan gagal menerbitkannya.
        // Petugas tetap dapat mengambilnya lewat "Lihat Dokumen".
      })
  }

  return (
    <div className="mt-4">
      {warna === 'oranye' ? (
        <OrangeButton
          disabled={!reference || tindakan.isPending}
          onClick={() => void jalankan().catch(() => {})}
        >
          {tindakan.isPending ? 'Menjalankan…' : label}
        </OrangeButton>
      ) : (
        <Button
          tone="utama"
          disabled={!reference || tindakan.isPending}
          onClick={() => void jalankan().catch(() => {})}
        >
          {tindakan.isPending ? 'Menjalankan…' : label}
        </Button>
      )}
      <p className="mt-1 text-xs text-slate-500">{note}</p>

      {tindakan.isSuccess && (
        <p className="mt-2 max-w-md rounded-kontrol border border-emerald-200 bg-emerald-50 px-3 py-2 text-xs text-slate-700">
          {tindakan.data?.pesan ?? 'Tindakan dijalankan.'}
        </p>
      )}

      {tindakan.isError && (
        <div className="mt-2 max-w-md rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2">
          <p className="text-xs text-slate-700">{messageOf(tindakan.error)}</p>

          {/*
            Nomor case-nya digambar DI SINI, bukan hanya di kaki layar.

            Pesan peladen menyuruh mengerjakannya di Pega, dan untuk itu petugas butuh nomor
            case-nya. Sebelumnya ia hanya ada di kaki layar, sehingga kalimat yang benar
            berujung pada gulir mencari — lihat CopyCaseNumber.

            Hanya untuk galat "layanan belum tersambung". Galat lain — kewenangan, tindakan
            tidak berlaku, sambungan putus — TIDAK diselesaikan dengan mengerjakannya di Pega,
            dan menawarkan nomor case di sana akan menyesatkan.
          */}
          {belumTersambungKePega(tindakan.error) && (
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <span className="text-xs text-slate-700">Kerjakan di Pega pada klaim</span>
              <CopyCaseNumber caseNumber={reference} />
            </div>
          )}
        </div>
      )}
    </div>
  )
}

/**
 * Benarkah galat ini "layanan Pega belum tersambung"?
 *
 * Diuji lewat KODE galat, bukan lewat teks pesannya. Pesan ditulis untuk dibaca manusia dan
 * boleh diperbaiki kapan saja; kode adalah kontrak. Mencocokkan teks berarti satu perbaikan
 * kalimat di peladen diam-diam menghilangkan nomor case dari layar.
 */
function belumTersambungKePega(error: unknown): boolean {
  return error instanceof APIError && error.kode === 'layanan_pega_belum_tersedia'
}

function WriteAction({
  label,
  note,
  caseNumber,
}: Readonly<{
  label: string
  note: string
  // `| undefined` eksplisit karena project memakai `exactOptionalPropertyTypes`: detailnya
  // boleh belum tiba, dan tombolnya tetap digambar.
  caseNumber?: string | undefined
}>) {
  const panel = useContext(WriteActionPanel)
  const open = panel.open === label

  // Varian TAUTAN dibuang 2026-10-02, bersama panel kedua tautan grid. Keduanya kini bekerja
  // sendiri — lihat ReceivedDocumentGrid — sehingga tidak ada lagi yang memakainya, dan jalur
  // yang tidak dipakai siapa pun adalah jalur yang tidak pernah teruji.
  return (
    <div className="mt-4">
      <Button tone="kedua" aria-expanded={open} onClick={() => panel.toggle(label)}>
        {label}
      </Button>
      <p className="mt-1 text-xs text-slate-500">{note}</p>

      {open && <ActionNote label={label} caseNumber={caseNumber} />}
    </div>
  )
}

/**
 * Panel keterangan satu tindakan yang belum dapat dijalankan dari sini.
 *
 * # Kenapa ia terpisah dari WriteAction
 *
 * Karena LETAKNYA berbeda menurut bentuk tombolnya. Tombol berbingkai menggambarnya tepat di
 * bawah dirinya; tautan di dalam baris alat menggambarnya di bawah SELURUH baris, supaya
 * barisnya tidak terdorong melebar.
 *
 * Isinya satu baris, dan alasan panjangnya TIDAK diulang di sini: ia sudah tertulis sekali di
 * kaki layar. Yang tersisa hanyalah yang BERBEDA antartombol — nama tindakannya dan nomor
 * case yang perlu disalin.
 */
function ActionNote({
  label,
  caseNumber,
}: Readonly<{
  label: string
  caseNumber?: string | undefined
}>) {
  return (
    <div className="mt-2 flex max-w-md flex-wrap items-center gap-2 rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2">
      <span className="text-xs text-slate-700">
        Kerjakan <span className="font-medium">{label}</span> di Pega pada klaim
      </span>
      <CopyCaseNumber caseNumber={caseNumber} />
    </div>
  )
}

/**
 * Nomor case beserta tombol salinnya.
 *
 * # Kenapa ia komponen tersendiri
 *
 * Ia dipakai DUA tempat yang keadaannya berbeda: panel tombol yang memang belum dibangun
 * (WriteAction), dan panel GALAT tombol yang sudah dibangun tetapi layanan Pega-nya belum
 * tersambung (ClaimAction). Keduanya menuntut hal yang sama dari petugas — menyalin nomor
 * case lalu mengerjakannya di Pega — sehingga keduanya wajib menyediakan nomor itu DI TEMPAT
 * pesannya muncul.
 *
 * Sebelum 2026-10-01 hanya WriteAction yang punya. ClaimAction menampilkan pesan
 * "salin nomor case-nya dari layar ini" tanpa satu pun nomor di dekatnya, sehingga petugas
 * harus menggulir ke kaki layar untuk mencarinya. Kalimatnya benar; letaknya yang salah.
 *
 * Keadaan "Tersalin" tidak perlu disetel ulang dengan tangan: pemanggilnya menggambar
 * komponen ini hanya saat panelnya terbuka, sehingga ia lahir kembali bersama panelnya.
 */
function CopyCaseNumber({ caseNumber }: Readonly<{ caseNumber?: string | undefined }>) {
  const [copied, setCopied] = useState(false)

  async function salin() {
    if (!caseNumber) return
    try {
      await navigator.clipboard.writeText(caseNumber)
      setCopied(true)
    } catch {
      // Penyalinan ditolak peramban — nomornya tetap tergambar supaya dapat disalin dengan
      // tangan. Kegagalan ini TIDAK dilaporkan sebagai galat: ia bukan kerusakan, dan
      // pesannya akan mengalihkan perhatian dari nomornya.
      setCopied(false)
    }
  }

  return (
    <>
      <span className="font-mono text-sm text-slate-900">{caseNumber ?? '—'}</span>
      {caseNumber && (
        <button
          type="button"
          onClick={salin}
          className="rounded-kontrol border border-slate-300 bg-white px-2 py-0.5 text-xs text-slate-700 hover:bg-slate-50"
        >
          {copied ? 'Tersalin' : 'Salin nomor case'}
        </button>
      )}
    </>
  )
}

/**
 * Satu isian: judul, nilai, dan penanda wajib.
 *
 * Penanda wajib dibawa dari `pyRequired` pada sel-nya. Layar ini hanya membaca, sehingga
 * tidak ada yang divalidasi — tetapi ia bagian dari BENTUK layar lama, dan petugas yang
 * membandingkan keduanya berdampingan mencarinya.
 */
type Field = [label: string, value: FieldValue, required?: boolean, kind?: FieldKind]

/**
 * Bentuk sebuah isian di layar lama — dan karenanya di layar ini (`D-13`).
 *
 * Layar lama menggambar hampir seluruh isiannya sebagai KOTAK MASUKAN, bukan sebagai teks
 * biasa. Perbedaannya bukan selera: petugas membaca layar ini berdampingan dengan Pega, dan
 * deretan teks polos tidak terbaca sebagai layar yang sama.
 *
 * Kotaknya tetap HANYA DIBACA — layar ini tidak menyunting apa pun. Yang ditiru bentuknya,
 * bukan kemampuannya.
 *
 *   `teks`     tanpa kotak — dipakai dua isian teratas Lampiran Surat, yang di Pega pun
 *              digambar sebagai teks polos di bawah judulnya
 *   `kotak`    kotak satu baris
 *   `area`     kotak tinggi untuk kalimat panjang (`pxTextArea` di section)
 *   `pilihan`  kotak dengan tanda daftar pilihan — "Perihal"
 *   `tanggal`  kotak dengan tanda kalender — "Tanggal Kejadian", "Tanggal Kelengkapan"
 */
type FieldKind = 'teks' | 'kotak' | 'area' | 'pilihan' | 'tanggal'

/**
 * Sekelompok isian dalam DUA KOLOM, berpasangan kiri-kanan menurut urutan section.
 *
 * Isian yang hidup di clipboard Pega digambar dengan penanda tersendiri — bukan tanda pisah.
 * Tanda pisah sudah dipakai untuk "kosong", dan kedua keadaan itu berbeda sama sekali: yang
 * kosong memang belum diisi petugas, yang di clipboard punya nilai tetapi nilainya tidak
 * dapat dibaca dari tabel.
 */
/**
 * Satu isian yang DAPAT DIKETIK, berbentuk sama dengan kotak hanya-baca di sebelahnya.
 *
 * # Kenapa bentuknya sengaja MIRIP yang hanya-baca
 *
 * Karena di layar lama ketiganya memang terlihat sama — yang membedakan hanya dapat atau
 * tidaknya diketik, bukan bingkainya. Membedakannya secara mencolok akan membuat layar ini
 * tidak lagi terbaca sebagai layar yang sama (`D-13`).
 *
 * Yang TIDAK disamakan: isian ini menyala saat difokus. Tanpa itu, petugas tidak punya
 * petunjuk mana yang dapat diisi selain mencobanya satu per satu.
 */
function EditableField({
  label,
  value,
  onChange,
  required,
  type,
}: Readonly<{
  label: string
  value: string
  onChange: (next: string) => void
  required?: boolean
  type: 'teks' | 'area' | 'datetime-local'
}>) {
  const kelas = [
    'mt-1 w-full rounded-sm border border-slate-300 bg-white px-2 py-1.5',
    'text-sm text-slate-900',
    'focus:border-blue-500 focus:outline-none focus:ring-2 focus:ring-blue-500/30',
  ].join(' ')

  return (
    <div className="mt-6 flex flex-col">
      <label className="text-[13px] font-semibold text-slate-800">
        {label}
        {required && <span className="ml-0.5 text-red-500">*</span>}

        {type === 'area' ? (
          <textarea
            className={`${kelas} min-h-[72px]`}
            value={value}
            onChange={(e) => onChange(e.target.value)}
          />
        ) : (
          <input
            type={type === 'datetime-local' ? 'datetime-local' : 'text'}
            className={kelas}
            value={value}
            onChange={(e) => onChange(e.target.value)}
          />
        )}
      </label>
    </div>
  )
}

/**
 * Mengubah tanggal apa adanya dari peladen menjadi bentuk yang diterima `datetime-local`.
 *
 * Peladen mengirimnya sebagaimana tersimpan — antara lain `2026-10-01 18:03:00` atau
 * ber-zona waktu. `datetime-local` hanya menerima `YYYY-MM-DDTHH:mm`, dan nilai yang tidak
 * cocok DIABAIKAN DIAM-DIAM oleh peramban: isiannya tergambar kosong seolah tanggalnya
 * belum pernah diisi.
 *
 * Nilai yang tidak dapat dikenali dikembalikan KOSONG, bukan apa adanya — isian kosong
 * jujur menyatakan "silakan pilih", sementara nilai yang ditolak peramban berbohong.
 */
function untukIsianWaktu(raw: string): string {
  const bersih = raw.trim()
  if (bersih === '') return ''

  // `2026-10-01T18:03…` atau `2026-10-01 18:03…` — keduanya dipotong di menit.
  const cocok = /^(\d{4}-\d{2}-\d{2})[T ](\d{2}:\d{2})/.exec(bersih)
  if (cocok) return `${cocok[1]}T${cocok[2]}`

  // `2026-10-01` saja — jam diisi awal hari, sama seperti bawaan peramban.
  const tanggalSaja = /^(\d{4}-\d{2}-\d{2})$/.exec(bersih)
  if (tanggalSaja) return `${tanggalSaja[1]}T00:00`

  // `11/09/2024 21:35` — bentuk TAMPILAN yang dikirim peladen untuk grid.
  //
  // Ia perlu diterima sejak baris grid dapat diketik: tanpa ini, tanggal yang SUDAH ada
  // tergambar kosong pada isiannya, dan petugas menyangka datanya hilang.
  const tampilan = /^(\d{2})\/(\d{2})\/(\d{4})(?:[ T](\d{2}:\d{2}))?/.exec(bersih)
  if (tampilan) {
    return `${tampilan[3]}-${tampilan[2]}-${tampilan[1]}T${tampilan[4] ?? '00:00'}`
  }

  return ''
}

function Panel({ children }: Readonly<{ children: ReactNode }>) {
  return (
    <div className="mt-4 rounded-sm border border-slate-300 bg-white p-5">{children}</div>
  )
}

function FieldGroup({ title, fields }: Readonly<{ title?: string; fields: Field[] }>) {
  return (
    <div className="mt-6">
      {title && (
        <h3 className="text-xs font-semibold tracking-wide text-slate-500 uppercase">
          {title}
        </h3>
      )}
      <dl className="mt-2 grid gap-x-6 gap-y-3 sm:grid-cols-2">
        {fields.map((field) => (
          <FieldRow key={field[0]} field={field} />
        ))}
      </dl>
    </div>
  )
}

/**
 * Sekelompok isian FULL WIDTH, satu per baris.
 *
 * Dipakai isian yang di section berupa `pxTextArea` atau menempati satu baris penuh —
 * Status, Catatan dari Analyst, ketiga Keterangan, dan ketiga isian Penerimaan Dokumen.
 * Memaksanya ke dua kolom akan memotong kalimat surat yang panjang menjadi kolom sempit,
 * dan itu bukan bentuk layar lama.
 */
function StackedFields({ title, fields }: Readonly<{ title?: string; fields: Field[] }>) {
  return (
    <div className="mt-6">
      {title && (
        <h3 className="text-xs font-semibold tracking-wide text-slate-500 uppercase">
          {title}
        </h3>
      )}
      <dl className="mt-2 grid gap-y-3">
        {fields.map((field) => (
          <FieldRow key={field[0]} field={field} />
        ))}
      </dl>
    </div>
  )
}

/** Satu baris isian, dipakai kedua susunan supaya keduanya tidak dapat menyimpang. */
function FieldRow({ field }: Readonly<{ field: Field }>) {
  const [label, value, required, kind = 'teks'] = field

  const teks =
    value === DI_CLIPBOARD ? 'di clipboard Pega' : value === '' ? '' : String(value)
  const kosong = value === DI_CLIPBOARD || value === ''

  return (
    <div className="flex flex-col">
      {/*
        Label di layar lama BERWARNA GELAP dan setebal teks biasa, bukan abu-abu kecil
        berhuruf besar. Bentuk sebelumnya membuat label terbaca sebagai keterangan tambahan,
        padahal di Pega ia judul isiannya.
      */}
      <dt className="text-[13px] font-semibold text-slate-800">
        {label}
        {required && (
          <span className="ml-0.5 text-red-500" title="Wajib diisi di layar lama">
            *
          </span>
        )}
      </dt>

      <dd className="mt-1">
        {kind === 'teks' ? (
          <span
            className={
              kosong ? 'text-sm text-slate-400 italic' : 'text-sm text-slate-900'
            }
          >
            {kosong ? teks || '—' : teks}
          </span>
        ) : (
          <FieldBox kind={kind} teks={teks} kosong={kosong} />
        )}
      </dd>
    </div>
  )
}

/**
 * Kotak isian HANYA-BACA yang meniru bentuk masukan di layar lama.
 *
 * # Kenapa bukan `<input readOnly>`
 *
 * Karena ia akan dapat difokus dan disorot papan ketik seolah dapat disunting, dan pembaca
 * layar akan mengumumkannya sebagai isian. Layar ini tidak menyunting apa pun; yang ditiru
 * adalah BENTUKNYA, bukan kemampuannya. Nilainya karena itu tetap digambar sebagai teks di
 * dalam kotak.
 *
 * Isian kosong tetap menggambar KOTAKNYA, bukan tanda pisah. Di Pega kotaknya memang ada dan
 * kosong — menghilangkannya akan membuat susunan dua kolom bergeser dan tidak lagi sejajar
 * dengan layar lama.
 */
function FieldBox({
  kind,
  teks,
  kosong,
}: Readonly<{
  kind: FieldKind
  teks: string
  kosong: boolean
}>) {
  const dasar = [
    'w-full rounded-sm border border-slate-300 bg-white px-2 py-1.5',
    'text-sm text-slate-900',
  ].join(' ')

  if (kind === 'area') {
    return (
      <div className={`${dasar} min-h-[72px] whitespace-pre-wrap`}>
        {kosong ? <span className="text-slate-400 italic">{teks}</span> : teks}
      </div>
    )
  }

  return (
    <div className={`${dasar} flex items-center justify-between gap-2`}>
      <span className={kosong ? 'truncate text-slate-400 italic' : 'truncate'}>
        {teks}
      </span>

      {/* Tanda daftar pilihan dan tanda kalender — keduanya ada di layar lama. */}
      {kind === 'pilihan' && (
        <span aria-hidden className="text-[10px] text-blue-600">
          ▾
        </span>
      )}
      {kind === 'tanggal' && (
        <span aria-hidden className="text-xs text-slate-500">
          🗓
        </span>
      )}
    </div>
  )
}

/**
 * messageOf mengambil pesan yang dapat dibaca pengguna dari galat apa pun.
 *
 * Galat dari server sudah berbahasa Indonesia dan menyebut portal; yang lain diganti
 * kalimat umum, karena pesan bawaan peramban ("Failed to fetch") tidak berarti apa pun bagi
 * petugas klaim.
 */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  return 'Sambungan ke peladen gagal. Coba lagi beberapa saat lagi.'
}

/**
 * Tombol ORANYE layar lama — "Submit" pada dialog unggah, dan kedua tombol "Kirim".
 *
 * # Kenapa ia TIDAK menumpang `Button` dengan className
 *
 * Karena bentuk sebelumnya menghasilkan tombol yang TIDAK TERBACA: `Button` bernada `kedua`
 * memasang `bg-white text-slate-700`, dan className menambahkan `bg-orange-500 text-white`.
 * Keduanya berkekhususan SAMA, sehingga yang menang ditentukan urutan kelasnya di dalam CSS
 * — bukan urutan penulisannya di atribut. Hasilnya latar putih dengan tulisan putih, dan
 * tombolnya tergambar kosong. Work Owner melaporkannya dengan tangkapan layar 2026-10-02.
 *
 * Tombol ini karena itu menulis kelasnya sendiri dari nol: tidak ada kelas yang perlu
 * dikalahkan, sehingga tidak ada yang bergantung pada urutan.
 */
function OrangeButton({
  children,
  disabled,
  onClick,
}: Readonly<{
  children: ReactNode
  disabled?: boolean
  onClick: () => void
}>) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className={[
        'inline-flex items-center justify-center rounded-kontrol px-3.5 py-2',
        'text-sm font-medium whitespace-nowrap select-none',
        'bg-orange-500 text-white shadow-aksen',
        'transition-colors duration-150 hover:bg-orange-600 active:bg-orange-700',
        'focus:outline-none focus-visible:ring-4 focus-visible:ring-orange-500/35',
        'disabled:cursor-not-allowed disabled:opacity-55 disabled:shadow-none',
        'disabled:hover:bg-orange-500',
      ].join(' ')}
    >
      {children}
    </button>
  )
}
