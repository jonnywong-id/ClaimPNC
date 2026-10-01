import { useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { rclpuclDocumentURL, useRCLPUCLClaim, useRCLPUCLDocuments } from './api'
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

  const state: ScreenState = claim.isError
    ? 'galat'
    : claim.isPending
      ? 'memuat'
      : detail
        ? 'siap'
        : 'kosong'

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
}: {
  state: ScreenState
  error: unknown
  claimKey: string
}) {
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
function WorkScreen({ detail }: { detail: ClaimDetailResponse | null }) {
  const [tab, setTab] = useState<WorkTab>('lampiran')

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
      <WorkTabs active={active} onChange={setTab} showsReceipt={showsReceipt} />

      {!showsReceipt && (
        <p className="mt-3 text-xs text-slate-500">
          Klaim berstatus <span className="font-medium">Notification</span> hanya memiliki
          Lampiran Surat. Layar lama menyembunyikan tab &ldquo;Penerimaan Dokumen&rdquo;
          untuk klaim seperti ini, karena pemberitahuan tidak menunggu dokumen dan tidak
          dikirim kembali ke Analyst.
        </p>
      )}

      {active === 'lampiran' ? (
        <LetterTab detail={detail} />
      ) : (
        <ReceiptTab detail={detail} />
      )}
    </>
  )
}

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
}: {
  active: WorkTab
  onChange: (tab: WorkTab) => void
  showsReceipt: boolean
}) {
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
function LetterTab({ detail }: { detail: ClaimDetailResponse | null }) {
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
      <StackedFields
        fields={[
          ['Status RCL / PUCL / MSIG', letter?.rcl_pucl ?? ''],
          ['Catatan dari Analyst', letter?.deskripsi_analyst ?? ''],
        ]}
      />

      <FieldGroup
        fields={[
          ['UP', letter?.up ?? ''],
          ['No Kontrak', DI_CLIPBOARD],
          ['No Polis', letter?.no_polis ?? ''],
          ['Business Unit / Seksi', DI_CLIPBOARD],
          ['Nama Peserta', letter?.nama_peserta ?? ''],
          ['Jumlah Tagihan', letter?.jumlah_tagihan ?? ''],
          ['Perihal', letter?.perihal ?? ''],
          ['Tanggal Kejadian', letter?.tanggal_kejadian ?? ''],
        ]}
      />

      {/*
        Ketiga Keterangan FULL WIDTH, bukan dua kolom: ketiganya `pxTextArea` di section dan
        isinya kalimat surat yang panjang — di Pega masing-masing memenuhi satu baris penuh.
      */}
      <StackedFields
        fields={[
          ['Keterangan Pembuka', letter?.keterangan_pembuka ?? ''],
          ['Keterangan Isi', letter?.keterangan_isi ?? ''],
          ['Keterangan Penutup', letter?.keterangan_penutup ?? ''],
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
        pilihan (master Perihal RCL/PUCL). Yang tersimpan pada klaim adalah teks pilihannya,
        dan itulah yang digambar di atas.
      </p>

      {/*
        Isian yang paling mudah dilaporkan sebagai kerusakan, padahal BUKAN: "UP" berisi nama
        objek, sama dengan "Nama Peserta", karena kedua penetapan di activity Pega menunjuk
        ekspresi yang sama — dan Work Owner menegaskan itu memang benar. Dinyatakan di
        tempat, bukan hanya di dokumen, karena di sinilah pengguna akan bertanya.
      */}
      {letter && letter.up !== '' && letter.up === letter.nama_peserta && (
        <p className="mt-2 text-xs text-slate-500">
          Kolom <span className="font-medium">UP</span> berisi nama objek yang sama dengan
          Nama Peserta. Itu bukan kekeliruan tampilan — keduanya memang diisi dari sumber
          yang sama di sistem lama.
        </p>
      )}

      {/*
        KEDUA tombol ini saling meniadakan — yang satu jalur non-MSIG, yang lain jalur MSIG
        ber-Notification — sehingga satu klaim tidak pernah menampilkan keduanya. Syaratnya
        dihitung di server; layar hanya menggambar apa yang dikirimkannya.
      */}
      <div className="flex flex-wrap gap-3">
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
          <WriteAction
            label="Download Dokumen"
            caseNumber={detail?.referensi}
            note={
              'Menerbitkan PDF surat DAN menandainya sudah dicetak — klaimnya berpindah ' +
              'dari tab "Cetak Surat" ke "Kelengkapan Dokumen".'
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
function ReceiptTab({ detail }: { detail: ClaimDetailResponse | null }) {
  const receipt = detail?.penerimaan_dokumen
  const buttons = detail?.tombol ?? NO_BUTTONS

  return (
    <>
      <ReceivedDocumentGrid />

      {/*
        Ketiganya FULL WIDTH di section, dan dua di antaranya WAJIB diisi (`pyRequired`
        true). Tanda wajibnya dibawa meski layar ini hanya membaca: ia menyatakan bentuk
        layar lama, dan petugas yang membandingkan keduanya berdampingan mencarinya.
      */}
      {/* Judulnya ada di kepala tab, sama seperti di Pega. */}
      <StackedFields
        fields={[
          ['Email Tertanggung', DI_CLIPBOARD],
          ['Tanggal Kelengkapan Dokumen', DI_CLIPBOARD, true],
          ['Catatan untuk Analyst', receipt?.komentar_pucl ?? '', true],
        ]}
      />

      {/* Ketiganya `pyVisible ALWAYS` — selalu digambar, apa pun jalur dan lini bisnisnya. */}
      <div className="flex flex-wrap gap-3">
        {buttons.unggah_dokumen && (
          <WriteAction
            label="Unggah Dokumen"
            caseNumber={detail?.referensi}
            note="Melampirkan berkas dokumen ke klaim."
          />
        )}
        {/*
          SATU-SATUNYA tombol layar ini yang benar-benar berjalan.

          Ia MEMBACA — `localAction ViewAttachmentPUCL` di Pega hanya membuka daftar lampiran,
          tanpa menyentuh satu pun tabel. Karena itu ia tidak terhalang `P-1`, dan dibangun
          lebih dulu atas persetujuan Work Owner 2026-10-01.
        */}
        {buttons.lihat_dokumen && <ViewDocumentsAction reference={detail?.referensi} />}
        {buttons.save && (
          <WriteAction
            label="Save"
            caseNumber={detail?.referensi}
            note="Menyimpan isian tanpa meneruskan klaimnya."
          />
        )}
      </div>

      {/*
        Ketiga tombol berikut BERSYARAT, dan paling banyak SATU yang muncul: "Tolak Klaim"
        hanya di jalur RCL, dan kedua tombol "Kirim" hanya di jalur PUCL — yang satu untuk PA,
        yang lain untuk Travel. Klaim PUCL pada lini selain keduanya tidak menampilkan satu pun.
      */}
      <div className="flex flex-wrap gap-3">
        {buttons.tolak_klaim && (
          <WriteAction
            label="Tolak Klaim"
            caseNumber={detail?.referensi}
            note="Menolak klaim. Hanya tersedia pada jalur RCL."
          />
        )}
        {buttons.kirim_ke_analyst && (
          <WriteAction
            label="Kirim Ke Analyst"
            caseNumber={detail?.referensi}
            note="Meneruskan klaim kembali ke Analyst setelah dokumennya lengkap."
          />
        )}
        {buttons.kirim_ke_pic_teknik && (
          <WriteAction
            label="Kirim ke PIC Teknik"
            caseNumber={detail?.referensi}
            note="Meneruskan klaim ke PIC Teknik. Jalur PUCL pada lini Travel."
          />
        )}
      </div>

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
}: {
  detail: ClaimDetailResponse | null
  actionCount: string
}) {
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
            MENYIMPAN data, dan selama Pega dan sistem baru berjalan berdampingan data klaim
            hanya boleh diubah dari satu sistem. Kerjakan tindakannya di Pega, cari klaimnya
            dengan Nomor Case di atas.
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
            dimiliki Pega. Isian ini tetap digambar di tempatnya supaya keadaannya terlihat.
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
function ReceivedDocumentGrid() {
  return (
    <div className="mt-6">
      <h3 className="text-xs font-semibold tracking-wide text-slate-500 uppercase">
        Tanggal terima Dokumen
      </h3>
      {/*
        Kedua tautan "Tambah" dan "Hapus" ADA di layar lama, di atas grid. Ia digambar
        mati — sama alasannya dengan kelima tombol tindakan: menghilangkannya
        menyembunyikan bahwa daftar ini dapat diisi, dan menghidupkannya menulis ke objek
        kerja yang masih dimiliki Pega (`P-1`).
      */}
      <p className="mt-2 flex gap-4 text-xs text-slate-400">
        <span>+ Tambah</span>
        <span>Hapus</span>
      </p>

      <table className="mt-2 w-full border-collapse text-sm">
        <thead>
          <tr className="border-b border-slate-200 text-left text-xs text-slate-500">
            <th className="py-1.5 font-medium">Tanggal</th>
            <th className="py-1.5 font-medium">Keterangan</th>
          </tr>
        </thead>
        <tbody>
          <tr>
            <td colSpan={2} className="py-2 text-xs text-slate-400 italic">
              Daftarnya tersimpan di clipboard Pega, bukan sebagai kolom tabel.
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  )
}

/**
 * Tombol tindakan yang MENULIS.
 *
 * Digambar, tetapi tidak dapat ditekan. Alasannya ditulis di sebelahnya, bukan disembunyikan
 * di balik pesan yang baru muncul setelah ditekan: tombol mati tanpa keterangan terbaca
 * sebagai kerusakan, dan petugas akan menekannya berulang kali.
 */
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
 * menariknya lewat JavaScript lebih dulu berarti seluruh berkas melewati memori halaman tanpa
 * satu pun manfaat, dan menghilangkan penampil bawaan peramban.
 */
function ViewDocumentsAction({ reference }: { reference?: string | undefined }) {
  const [open, setOpen] = useState(false)
  const documents = useRCLPUCLDocuments(reference ?? null, open)

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
      <p className="mt-1 text-xs text-slate-500">Membuka dokumen yang sudah dilampirkan.</p>

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
                <li key={document.id} className="flex flex-wrap items-baseline gap-x-3 py-1.5">
                  <a
                    href={rclpuclDocumentURL(reference ?? '', document.id)}
                    target="_blank"
                    rel="noreferrer"
                    className="text-sm font-medium text-biru-700 underline underline-offset-2"
                  >
                    {document.nama || document.id}
                  </a>
                  <span className="text-xs text-slate-500">
                    {[document.kategori, document.sub_kategori].filter(Boolean).join(' · ')}
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
          {documents.data?.catatan && (
            <p className="mt-2 border-t border-slate-100 pt-2 text-[11px] leading-snug text-slate-500">
              {documents.data.catatan}
            </p>
          )}
        </div>
      )}
    </div>
  )
}

function WriteAction({
  label,
  note,
  caseNumber,
}: {
  label: string
  note: string
  // `| undefined` eksplisit karena project memakai `exactOptionalPropertyTypes`: detailnya
  // boleh belum tiba, dan tombolnya tetap digambar.
  caseNumber?: string | undefined
}) {
  const [open, setOpen] = useState(false)
  const [copied, setCopied] = useState(false)

  async function salinNomorCase() {
    if (!caseNumber) return
    try {
      await navigator.clipboard.writeText(caseNumber)
      setCopied(true)
    } catch {
      // Penyalinan ditolak peramban — tombolnya tetap menampilkan nomornya supaya
      // dapat disalin dengan tangan. Kegagalan ini TIDAK dilaporkan sebagai galat:
      // ia bukan kerusakan, dan pesannya akan mengalihkan perhatian dari nomornya.
      setCopied(false)
    }
  }

  return (
    <div className="mt-4">
      <Button
        tone="kedua"
        aria-expanded={open}
        onClick={() => {
          setOpen((v) => !v)
          setCopied(false)
        }}
      >
        {label}
      </Button>
      <p className="mt-1 text-xs text-slate-500">{note}</p>

      {/*
        Susunan keterangannya LANGKAH DULU, alasan belakangan.

        Bentuk sebelumnya membuka dengan "Belum dapat dijalankan dari sini" lalu menjelaskan
        sebabnya, dan baru di baris terakhir menyebut apa yang harus dikerjakan. Itu membuat
        petugas membaca dua kalimat yang tidak dapat ditindaklanjuti sebelum sampai ke satu
        kalimat yang dapat — padahal yang dibutuhkannya cuma langkahnya. Sebabnya tetap
        ditulis, tetapi di bawah dan lebih kecil: ia menjawab "kenapa", dan "kenapa" bukan
        yang dicari orang yang sedang mengerjakan klaim.
      */}
      {open && (
        <div className="mt-2 max-w-md rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2">
          <p className="text-xs font-medium text-slate-800">
            Kerjakan tindakan ini di Pega — salin nomor case, buka klaimnya di sana, lalu
            tekan tombol yang sama.
          </p>
          {caseNumber && (
            <div className="mt-2 flex flex-wrap items-center gap-2">
              <span className="font-mono text-sm text-slate-900">{caseNumber}</span>
              <button
                type="button"
                onClick={salinNomorCase}
                className="rounded-kontrol border border-slate-300 bg-white px-2 py-0.5 text-xs text-slate-700 hover:bg-slate-50"
              >
                {copied ? 'Tersalin' : 'Salin nomor case'}
              </button>
            </div>
          )}
          <p className="mt-2 text-[11px] leading-snug text-slate-500">
            Tombol ini menyimpan perubahan pada klaim. Selama Pega masih melayani produksi,
            data klaim hanya boleh diubah dari satu sistem — karena itu tindakannya
            dikerjakan di sana, bukan di sini.
          </p>
        </div>
      )}
    </div>
  )
}

/**
 * Satu isian: judul, nilai, dan penanda wajib.
 *
 * Penanda wajib dibawa dari `pyRequired` pada sel-nya. Layar ini hanya membaca, sehingga
 * tidak ada yang divalidasi — tetapi ia bagian dari BENTUK layar lama, dan petugas yang
 * membandingkan keduanya berdampingan mencarinya.
 */
type Field = [label: string, value: FieldValue, required?: boolean]

/**
 * Sekelompok isian dalam DUA KOLOM, berpasangan kiri-kanan menurut urutan section.
 *
 * Isian yang hidup di clipboard Pega digambar dengan penanda tersendiri — bukan tanda pisah.
 * Tanda pisah sudah dipakai untuk "kosong", dan kedua keadaan itu berbeda sama sekali: yang
 * kosong memang belum diisi petugas, yang di clipboard punya nilai tetapi nilainya tidak
 * dapat dibaca dari tabel.
 */
function FieldGroup({ title, fields }: { title?: string; fields: Field[] }) {
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
function StackedFields({ title, fields }: { title?: string; fields: Field[] }) {
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
function FieldRow({ field }: { field: Field }) {
  const [label, value, required] = field

  return (
    <div className="flex flex-col">
      <dt className="text-xs font-medium text-slate-500">
        {label}
        {required && (
          <span className="ml-0.5 text-amber-600" title="Wajib diisi di layar lama">
            *
          </span>
        )}
      </dt>
      <dd
        className={
          value === DI_CLIPBOARD
            ? 'text-sm text-slate-400 italic'
            : 'text-sm text-slate-900'
        }
      >
        {value === DI_CLIPBOARD ? 'di clipboard Pega' : value === '' ? '—' : value}
      </dd>
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
