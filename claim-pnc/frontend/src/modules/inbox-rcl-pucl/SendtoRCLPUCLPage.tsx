import { useNavigate, useParams, useSearchParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { ErrorMessage } from '@/components/ErrorMessage'

import { useRCLPUCLClaim } from './api'
import type { ClaimDetailResponse } from './types'

/**
 * Layar kerja RCL/PUCL — section `SendtoRCLPUCL`.
 *
 * # Dari mana bentuknya
 *
 * Dari ketiga rule yang ditambahkan Work Owner pada 2026-09-24, dibaca langsung:
 *
 *	Section/SendtoRCLPUCL-Section.xml                kontainer; DUA sub-section, urutannya
 *	                                                 Lampiran Surat lalu Penerimaan Dokumen
 *	Section/SectionLampiranSuratPUCL-Section.xml     13 isian + tombol "cetak"
 *	Section/SectionPenerimaanDokumenPUCL-Section.xml grid + 3 isian + 2 tombol
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

/**
 * Kedua bagian layar, digambar menurut section-nya.
 *
 * Dipisah dari komponen halaman supaya `detail` sudah pasti ada — tanpa itu setiap isian
 * harus diperiksa satu per satu, dan pemeriksaan sebanyak itu menyembunyikan bentuk
 * layarnya di antara penjagaan.
 */
function WorkScreen({ detail }: { detail: ClaimDetailResponse | null }) {
  const letter = detail?.lampiran_surat
  const receipt = detail?.penerimaan_dokumen

  return (
    <>
      {/*
        Bagian pertama — `SectionLampiranSuratPUCL`. Ketiga belas isiannya ditulis dalam
        URUTAN section, bukan urutan yang paling rapi dibaca: petugas yang membandingkan
        kedua layar berdampingan menelusurinya dari atas ke bawah.
      */}
      <FieldGroup
        title="Lampiran Surat"
        fields={[
          ['Status RCL / PUCL / MSIG', letter?.rcl_pucl ?? ''],
          ['Catatan dari Analyst', letter?.deskripsi_analyst ?? ''],
          ['UP', letter?.up ?? ''],
          ['No Kontrak', DI_CLIPBOARD],
          ['No Polis', letter?.no_polis ?? ''],
          ['Business Unit / Seksi', DI_CLIPBOARD],
          ['Nama Peserta', letter?.nama_peserta ?? ''],
          ['Jumlah Tagihan', letter?.jumlah_tagihan ?? ''],
          ['Perihal', DI_CLIPBOARD],
          ['Tanggal Kejadian', letter?.tanggal_kejadian ?? ''],
          ['Keterangan Pembuka', DI_CLIPBOARD],
          ['Keterangan Isi', DI_CLIPBOARD],
          ['Keterangan Penutup', DI_CLIPBOARD],
        ]}
      />

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

      <WriteAction
        label="Cetak"
        note={
          'Mengisi tanggal cetak surat, sehingga klaimnya BERPINDAH dari tab "Cetak Surat" ' +
          'ke tab "Kelengkapan Dokumen".'
        }
      />

      {/* Bagian kedua — `SectionPenerimaanDokumenPUCL`. */}
      <ReceivedDocumentGrid />

      <FieldGroup
        title="Penerimaan Dokumen"
        fields={[
          ['Email Tertanggung', DI_CLIPBOARD],
          ['Tanggal Kelengkapan Dokumen', DI_CLIPBOARD],
          ['Catatan untuk Analyst', receipt?.komentar_pucl ?? ''],
        ]}
      />

      <div className="flex flex-wrap gap-3">
        <WriteAction
          label="Unggah Dokumen"
          note="Melampirkan berkas dokumen ke klaim."
        />
        <WriteAction
          label="Kirim ke PIC Teknik"
          note="Meneruskan klaim kembali ke PIC Teknik setelah dokumennya lengkap."
        />
      </div>

      {/*
        Kunci teknisnya ditampilkan dengan sengaja, meski ia BUKAN isian di section mana
        pun. Ia yang dipakai petugas membuka klaim yang sama di Pega, dan ia pula parameter
        `inskey` yang dikirim tautan aslinya.
      */}
      {detail && (
        <p className="mt-6 border-t border-slate-200 pt-3 text-xs text-slate-600">
          Kunci klaim: <span className="font-mono text-slate-900">{detail.referensi}</span>
        </p>
      )}

      {detail?.tindakan_masih_di_pega && (
        <div className="mt-3 rounded-kontrol border border-amber-200 bg-amber-50 px-3 py-2">
          <p className="text-xs text-slate-700">
            <span className="font-medium">Layar ini baca saja.</span> Ketiga tindakan di atas
            MENYIMPAN data, dan selama Pega dan sistem baru berjalan berdampingan data klaim
            hanya boleh diubah dari satu sistem. Kerjakan tindakannya di Pega memakai kunci
            di atas.
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
function WriteAction({ label, note }: { label: string; note: string }) {
  return (
    <div className="mt-4">
      <Button tone="kedua" disabled>
        {label}
      </Button>
      <p className="mt-1 text-xs text-slate-500">{note}</p>
    </div>
  )
}

/**
 * Sekelompok isian bertumpuk, dengan judul bagiannya.
 *
 * Isian yang hidup di clipboard Pega digambar dengan penanda tersendiri — bukan tanda pisah.
 * Tanda pisah sudah dipakai untuk "kosong", dan kedua keadaan itu berbeda sama sekali: yang
 * kosong memang belum diisi petugas, yang di clipboard punya nilai tetapi nilainya tidak
 * dapat dibaca dari tabel.
 */
function FieldGroup({ title, fields }: { title: string; fields: [string, FieldValue][] }) {
  return (
    <div className="mt-6">
      <h3 className="text-xs font-semibold tracking-wide text-slate-500 uppercase">
        {title}
      </h3>
      <dl className="mt-2 grid gap-x-6 gap-y-3 sm:grid-cols-2">
        {fields.map(([label, value]) => (
          <div key={label} className="flex flex-col">
            <dt className="text-xs font-medium text-slate-500">{label}</dt>
            <dd
              className={
                value === DI_CLIPBOARD
                  ? 'text-sm text-slate-400 italic'
                  : 'text-sm text-slate-900'
              }
            >
              {value === DI_CLIPBOARD
                ? 'di clipboard Pega'
                : value === ''
                  ? '—'
                  : value}
            </dd>
          </div>
        ))}
      </dl>
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
