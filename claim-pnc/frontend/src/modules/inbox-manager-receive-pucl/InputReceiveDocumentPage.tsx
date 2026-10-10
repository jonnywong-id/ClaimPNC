import { useNavigate, useParams, useSearchParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'
import { formatPegaFormDate } from '@/components/format'

import { useReceiveDocument, useReceiveDocumentAction } from './api'
import type { DocumentAction, DocumentField, DocumentResponse } from './types'

/**
 * Layar kerja penerimaan dokumen — flow action `InputReceiveDocument`.
 *
 * # Bagaimana layar ini terbuka
 *
 * Di Pega, sel nomor case pada grid Receive adalah TAUTAN. Mengkliknya menjalankan tiga
 * perilaku berurutan, terbaca apa adanya di
 * `Section/InboxManagerReceive_Section-Section.xml`:
 *
 *   1. `runActivity`    `SetAssignmentInboxReceive_act`, parameter `kunci = .pzInsKey`
 *   2. `refresh`        thisSection
 *   3. `openAssignment` `pyInsKey = TempIns.pyNote`
 *
 * Activity-nya sendiri nyaris kosong — `pyUsage = FLOW`, satu penetapan ke `TempIns.pyNote`.
 * Ia kait pra-proses; yang bekerja adalah **Open Assignment** bawaan Pega, yang membuka
 * berkasnya pada tahap alur kerjanya saat itu. Flow action yang menunggu di sana adalah
 * `InputReceiveDocument` (`Flow/InputReceiveDocument.xml`).
 *
 * # Kenapa HALAMAN, bukan panel di bawah tabel
 *
 * Karena di Pega yang terbuka bukan pratinjau baris melainkan berkas pada tahap alur
 * kerjanya, untuk dikerjakan. Panel di bawah tabel menyiratkan yang pertama.
 *
 * Antreannya tetap dapat dikembalikan utuh: tab dan nomor halaman dibawa di alamat, jadi
 * tombol kembali mendarat di tempat yang sama — bukan di tab pertama halaman pertama.
 *
 * # Bentuk layarnya datang dari SERVER
 *
 * Ke-36 judul isian, pengelompokannya, dan tanda terhalangnya dibaca backend dari
 * `Section/InputReceiveDocument-Section.xml`. Menyalinnya ke sini berarti daftar yang sama
 * hidup di dua tempat, dan yang satu akan tertinggal saat yang lain diperbaiki — lihat
 * `DocumentFieldGroups` di backend.
 *
 * # Layar ini MEMBACA
 *
 * Dan itu bukan pekerjaan yang tertunda melainkan keadaan yang terbaca dari artefaknya
 * sendiri: ke-25 sel ber-properti `.ReceiveDocument.*` di section itu bertanda
 * `pyReadOnly=true` TANPA KECUALI. Yang dapat disunting di Pega hanyalah blok Data Pelapor —
 * dan blok itulah yang seluruh isiannya hidup di dalam blob objek kerja, tanpa satu pun
 * kolom yang dapat dibaca maupun ditulis SQL.
 *
 * Kedelapan tombolnya tetap digambar; penekanannya menjawab alasan. Lihat WriteActionBar.
 */
export function InputReceiveDocumentPage() {
  const { referensi } = useParams<{ referensi: string }>()
  const [params] = useSearchParams()
  const navigate = useNavigate()

  const key = referensi ? decodeURIComponent(referensi) : ''
  const berkas = useReceiveDocument(key)

  // `?? null` bukan kerapian. `callAPI` mengembalikan `null` — bukan melempar — untuk
  // jawaban 200 yang badannya BUKAN JSON, misalnya saat alamat `/api/...` dijawab penyaji
  // SPA dengan `index.html`. Tanpa penanganan, keadaan itu menghasilkan halaman yang
  // benar-benar kosong: bukan memuat, bukan galat, bukan data. Itu kelas kegagalan yang
  // paling sulit dilaporkan pengguna, karena tidak ada satu pun yang dapat disebutkan.
  const detail = berkas.data ?? null

  const state: ScreenState = berkas.isError
    ? 'galat'
    : berkas.isPending
      ? 'memuat'
      : detail
        ? 'siap'
        : 'kosong'

  // Tab dan halaman dibawa kembali apa adanya. Petugas yang membuka berkas dari halaman
  // ketiga harus mendarat di sana lagi.
  const back = params.toString()
    ? `/inbox-manager-receive-pucl?${params.toString()}`
    : '/inbox-manager-receive-pucl'

  return (
    <div className="mx-auto max-w-5xl px-4 py-6">
      <header className="flex flex-wrap items-start justify-between gap-3 border-b border-slate-200 pb-4">
        <div>
          <h1 className="text-xl font-semibold text-slate-900">
            {detail ? `Berkas ${detail.no_case}` : 'Layar kerja penerimaan dokumen'}
          </h1>
          <p className="mt-1 text-sm text-slate-600">
            Input Receive Document — layar kerja berkas penerimaan dokumen klaim.
          </p>
          {detail && <CaseSummary detail={detail} />}
        </div>
        <Button tone="kedua" onClick={() => navigate(back)}>
          Kembali ke antrean
        </Button>
      </header>

      <StateNotice state={state} error={berkas.error} documentKey={key} />

      {/*
        Layarnya digambar SELALU, apa pun keadaan datanya.

        Dua alasan, dan keduanya lebih kuat daripada kerapian. Pertama, bentuk layar ini
        tidak bergantung pada data — ia tetap kelima kelompok isian yang sama, dan enam
        belas di antaranya memang tidak pernah terisi. Kedua, halaman yang tidak menggambar
        apa pun tidak dapat dibedakan dari halaman yang rusak.
      */}
      <WorkScreen detail={detail} />
    </div>
  )
}

/** Keadaan pengambilan isi layar kerja. */
type ScreenState = 'memuat' | 'galat' | 'kosong' | 'siap'

/**
 * Keterangan kepala berkas: nomor klaim PNC, jenis klaim, dan status kerjanya.
 *
 * Ketiganya TIDAK digambar sebagai isian di layar lama. Ia ditambahkan karena layar ini
 * dibuka lewat Open Assignment: sebelum mengerjakan berkasnya di Pega, petugas perlu tahu
 * berkas ini sudah menjadi klaim atau belum, dan masih berjalan atau sudah selesai.
 */
function CaseSummary({ detail }: { detail: DocumentResponse }) {
  return (
    <dl className="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-xs text-slate-600">
      <div className="flex gap-1">
        <dt className="font-medium">PNC CaseID:</dt>
        {/*
          Kosong berarti berkasnya BELUM diregistrasi menjadi klaim — keadaan yang sah, dan
          justru itulah yang dikerjakan tombol "Register Klaim". Ia dinyatakan dengan kata,
          bukan dengan tanda pisah, supaya tidak terbaca sebagai data yang hilang.
        */}
        <dd>{detail.no_klaim_pnc || 'belum diregistrasi'}</dd>
      </div>
      <div className="flex gap-1">
        <dt className="font-medium">Jenis Klaim:</dt>
        <dd>{detail.jenis_klaim || '—'}</dd>
      </div>
      <div className="flex gap-1">
        <dt className="font-medium">Status kerja:</dt>
        <dd>{detail.status_kerja || '—'}</dd>
      </div>
    </dl>
  )
}

/**
 * Sebaris keterangan keadaan, digambar DI ATAS layar — bukan menggantikannya.
 *
 * Halaman yang mengganti seluruh badannya dengan pesan memuat atau kotak galat akan, pada
 * satu keadaan yang tidak terduga, menghasilkan halaman yang benar-benar kosong tanpa satu
 * pun petunjuk. Yang dipakai di sini: layarnya tetap tergambar, keadaannya dinyatakan di
 * atasnya.
 */
function StateNotice({
  state,
  error,
  documentKey,
}: {
  state: ScreenState
  error: unknown
  documentKey: string
}) {
  if (state === 'memuat') {
    return (
      <p className="mt-4 text-sm text-slate-500" role="status">
        Memuat berkas…
      </p>
    )
  }

  if (state === 'galat') {
    return (
      <div className="mt-4">
        <ErrorMessage
          title="Berkas tidak dapat dimuat"
          description={messageOf(error)}
          tone="gangguan"
        />
      </div>
    )
  }

  if (state === 'kosong') {
    // Jawaban 200 yang badannya bukan JSON. Dikatakan apa adanya — termasuk kuncinya —
    // supaya yang melaporkannya punya sesuatu yang dapat disebutkan.
    return (
      <div className="mt-4">
        <ErrorMessage
          title="Berkas tidak terbaca"
          description={
            `Peladen menjawab tanpa isi untuk berkas ${documentKey || '(tanpa kunci)'}. ` +
            'Muat ulang halaman; bila berulang, laporkan ke tim teknis beserta kunci di atas.'
          }
          tone="gangguan"
        />
      </div>
    )
  }

  return null
}

/**
 * Badan layar kerja: kelompok isian, lalu bilah tombol.
 *
 * Urutan kelompok mengikuti urutan sel di section, bukan disusun ulang menurut selera —
 * petugas yang membandingkan kedua layar berdampingan membaca isian pada urutan yang sama.
 */
function WorkScreen({ detail }: { detail: DocumentResponse | null }) {
  const groups = detail?.kelompok ?? []
  const values = detail?.nilai ?? {}
  const actions = detail?.tindakan ?? []
  const berkas = detail?.referensi ?? ''

  // Letak tombol datang dari SERVER, bukan disimpulkan dari kodenya di sini. Di layar lama,
  // tombol unggah dan lihat dokumen berada di puncak blok sebelum isian pertama; tombol
  // simpan, transfer, dan register berada sesudah seluruh isian. Membalik keduanya membuat
  // petugas yang membandingkan kedua layar berdampingan mencari tombol di tempat yang salah.
  const atas = actions.filter((action) => action.di_atas)
  const bawah = actions.filter((action) => !action.di_atas)

  return (
    <div className="mt-6 space-y-6">
      <WriteActionBar actions={atas} berkas="" />

      {groups.map((group) => (
        <FieldGroup
          key={group.judul}
          title={group.judul}
          fields={group.isian}
          values={values}
        />
      ))}

      <WriteActionBar actions={bawah} berkas={berkas} />

      <ClipboardNotice groups={groups} />
    </div>
  )
}

/** Satu kelompok isian, digambar sebagai satu panel. */
function FieldGroup({
  title,
  fields,
  values,
}: {
  title: string
  fields: DocumentField[]
  values: Record<string, string>
}) {
  return (
    <section className="rounded-kartu border border-slate-200">
      <h2 className="border-b border-slate-200 bg-slate-50 px-4 py-2 text-sm font-medium text-slate-800">
        {title}
      </h2>
      <dl className="grid gap-x-6 gap-y-3 px-4 py-4 sm:grid-cols-2">
        {fields.map((field) => (
          <FieldRow
            key={field.kunci}
            field={field}
            value={values[field.kunci] ?? ''}
          />
        ))}
      </dl>
    </section>
  )
}

/**
 * Satu isian.
 *
 * # Tiga keadaan yang DIBEDAKAN, dan kenapa
 *
 *   terhalang   belum ada sumbernya  -> ditandai, alasannya dapat dibaca
 *   kosong      sumbernya ada, isinya kosong -> tanda pisah
 *   terisi      nilainya digambar
 *
 * Yang pertama dan kedua terlihat sama bila tidak dibedakan, padahal tindak lanjutnya
 * berbeda sama sekali: yang satu menuntut kolomnya dibuka di Pega, yang lain menuntut
 * petugas mengisinya. Pada layar yang enam belas dari 36 isiannya terhalang, pembedaan itu
 * bukan kehalusan.
 */
function FieldRow({ field, value }: { field: DocumentField; value: string }) {
  const span = field.bertingkat ? 'sm:col-span-2' : ''

  // Tanggal ditulis seperti isian form Pega — `29/01/2020`, `29/01/2020 11:58`. Nilai yang
  // bukan tanggal lolos apa adanya, sehingga pembantu ini aman dipakai untuk SELURUH isian
  // dan tidak perlu daftar "mana yang tanggal" yang akan tertinggal saat isian bertambah.
  const text = formatPegaFormDate(value)

  return (
    <div className={span}>
      <dt className="text-xs font-medium text-slate-500">{field.judul}</dt>
      <dd className="mt-0.5 text-sm text-slate-900">
        {field.terhalang ? (
          <BlockedValue field={field} />
        ) : field.bertingkat ? (
          <span className="block whitespace-pre-wrap">{text || '—'}</span>
        ) : (
          text || '—'
        )}
      </dd>
    </div>
  )
}

/**
 * Isian yang digambar tetapi belum dapat diisi.
 *
 * # Digambar SAMA dengan isian kosong di Pega
 *
 * Yaitu sebagai tanda pisah. Versi sebelumnya menggambarnya sebagai lencana oranye
 * bertuliskan "belum terbawa", dan enam belas lencana itu membuat layar ini terlihat jauh
 * berbeda dari acuannya — dilaporkan Work Owner 2026-10-10 dan dicabut, karena `D-13`
 * menetapkan tampilan mengikuti Pega.
 *
 * # Keterangannya TIDAK hilang, ia pindah
 *
 * Alasan dan pemiliknya tetap datang dari server dan tetap terbaca saat isiannya ditunjuk.
 * Yang hilang hanyalah penandanya di layar.
 *
 * Konsekuensinya disadari dan diterima: isian yang TIDAK AKAN PERNAH terisi kini terlihat
 * sama dengan isian yang kebetulan kosong. Pembedaan itulah alasan lencananya dibuat.
 * Bila kelak pembedaan itu dibutuhkan kembali, bentuknya harus yang tidak menyimpang dari
 * Pega — misalnya penanda halus pada judulnya, bukan lencana berwarna di tempat nilainya.
 */
function BlockedValue({ field }: { field: DocumentField }) {
  const reason = [field.alasan_terhalang, field.pemilik_penghalang && `Menunggu: ${field.pemilik_penghalang}`]
    .filter(Boolean)
    .join('\n\n')

  return (
    <span className="cursor-help text-slate-400" title={reason}>
      —
    </span>
  )
}

/**
 * Bilah kedelapan tombol yang di Pega MENGUBAH data.
 *
 * # Kenapa tombolnya tetap digambar
 *
 * Karena ia pekerjaan NYATA yang dilakukan pengguna layar ini setiap hari. Layar yang
 * kehilangan tombolnya tanpa penjelasan akan dilaporkan sebagai kerusakan, dan penggunanya
 * tidak akan tahu ia masih harus mengerjakannya lewat Pega.
 *
 * # Kenapa penekanannya DIKIRIM ke server
 *
 * Karena alasan mengapa sebuah tombol belum bekerja adalah keadaan SISTEM, bukan keadaan
 * layar — dan ia berubah begitu kemampuannya dibangun. Menuliskan alasannya di sini berarti
 * alasan yang sama hidup di dua tempat, dan yang di layar akan tertinggal pada hari
 * tombolnya mulai bekerja.
 *
 * Penekanannya pun DICATAT di sisi peladen: selama masa paralel, jejak itulah satu-satunya
 * tanda seberapa sering tombol ini benar-benar dibutuhkan.
 */
function WriteActionBar({
  actions,
  berkas,
}: {
  actions: DocumentAction[]
  berkas: string
}) {
  const tindakan = useReceiveDocumentAction()

  if (actions.length === 0) return null

  return (
    <section className="rounded-kartu border border-slate-200 px-4 py-4">
      {/*
        Tanpa judul "Tindakan" — layar lama tidak punya judul apa pun di atas tombolnya;
        tombolnya berdiri langsung di tempatnya. Keterangan di bawah tetap ada karena tidak
        satu pun tombol dapat dijalankan, dan tombol yang diam tanpa penjelasan dilaporkan
        sebagai kerusakan.
      */}
      <p className="text-xs text-slate-600">
        Tombol ini ada di layar lama dan{' '}
        <span className="font-medium">belum satu pun dapat dijalankan di sini</span>. Tekan
        salah satunya untuk membaca alasannya, lalu kerjakan tindakannya lewat Pega.
      </p>

      <div className="mt-3 flex flex-wrap gap-2">
        {actions.map((action) => (
          <Button
            key={action.kode}
            tone="kedua"
            disabled={tindakan.isPending}
            onClick={() => tindakan.mutate({ tindakan: action.kode, berkas })}
            title={`Activity Pega: ${action.activity_pega} · Pemilik: ${action.pemilik}`}
          >
            {action.label}
          </Button>
        ))}
      </div>

      {tindakan.isError && (
        <p className="mt-3 text-xs text-red-800" role="alert">
          {messageOf(tindakan.error)}
        </p>
      )}

      {/*
        Kunci berkas ditampilkan supaya petugas tidak perlu mencarinya saat mengerjakan
        berkas ini di Pega. Ia kunci TEKNIS, bukan data nasabah.
      */}
      {berkas && (
        <p className="mt-3 break-all text-xs text-slate-600">
          <span className="font-medium">Kunci berkas:</span> {berkas}
        </p>
      )}
    </section>
  )
}

/**
 * Keterangan isian yang belum terbawa, dikumpulkan di satu tempat.
 *
 * Ia melengkapi tanda per isian: yang di atas menyatakan isian MANA, yang di sini menyatakan
 * BERAPA dan KENAPA. Tanpa yang kedua, pengguna yang melihat enam belas tanda tersebar akan
 * mengira layarnya rusak alih-alih belum lengkap.
 *
 * Jumlahnya dihitung dari bentuk layar yang dikirim server, bukan ditulis tetap di sini,
 * supaya ia ikut berkurang dengan sendirinya begitu satu penghalang hilang.
 */
function ClipboardNotice({
  groups,
}: {
  groups: DocumentResponse['kelompok']
}) {
  const blocked = groups.flatMap((group) => group.isian.filter((field) => field.terhalang))
  if (blocked.length === 0) return null

  const owners = [...new Set(blocked.map((field) => field.pemilik_penghalang ?? ''))]
    .filter(Boolean)
    .join(', ')

  return (
    <section className="rounded-kartu border border-slate-200 bg-slate-50 px-4 py-3">
      <h2 className="text-sm font-medium text-slate-800">
        {blocked.length} isian belum terbawa dari Pega
      </h2>
      <p className="mt-2 text-xs text-slate-600">
        Isian bertanda <span className="font-medium">belum terbawa</span> hidup di dalam blob
        objek kerja Pega dan tidak punya kolom basis data, sehingga tidak dapat dibaca kueri
        biasa selama objek kerjanya masih dimiliki Pega. Isian itu tetap digambar di tempatnya
        supaya ketiadaannya terlihat, bukan tersamar sebagai isian yang memang belum diisi.
        Tunjuk tandanya untuk membaca alasannya.
        {owners && (
          <>
            {' '}
            <span className="font-medium">Menunggu:</span> {owners}.
          </>
        )}
      </p>
    </section>
  )
}

/** messageOf mengambil pesan yang layak dibaca pengguna dari sebuah galat. */
function messageOf(error: unknown): string {
  if (error instanceof APIError) return error.message
  if (error instanceof Error && error.message !== '') return error.message
  return 'Coba lagi beberapa saat lagi. Bila terus berulang, hubungi tim teknis.'
}
