import { useState, type FormEvent } from 'react'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode, type ReasMember } from '@/api/types'
import { useSelectedPortal } from '@/app/portal'
import { Button } from '@/components/Button'
import { DataTable, type Column } from '@/components/DataTable'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import { useReasMemberList, useSaveReasMemberEmail } from './api'

/**
 * Ukuran halaman.
 *
 * **15**, disamakan dengan Master Login, Master Rekening, Master Status Klaim, Master Status
 * Progres, Master Pasal Kerugian, dan Master Penolakan Klaim.
 *
 * Berbeda dari layar-layar itu, angkanya di sini TIDAK dibaca dari `pyPageSize` gridnya:
 * section `BrowseListMemberReas` tidak ada di export (`R-16`), sehingga ukuran halaman layar
 * lamanya tidak diketahui. Yang dipakai adalah angka yang sudah berlaku di layar master lain
 * — memperkenalkan angka ketiga tanpa dasar hanya akan membuat satu layar terasa berbeda
 * tanpa alasan yang dapat dijelaskan.
 */
const PAGE_SIZE = 10

type MessageContent = { title: string; description: string; tone: ErrorTone }

/** Mengubah galat pemuatan daftar menjadi pesan yang dapat ditindaklanjuti. */
function loadMessage(error: unknown): MessageContent {
  if (error instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      description: 'Periksa koneksi jaringan, lalu muat ulang halaman ini.',
      tone: 'gangguan',
    }
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: 'Portal entitas belum dipilih',
          description:
            'Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian ' +
            'atas halaman ini lebih dulu.',
          tone: 'penolakan',
        }
      case ErrorCode.portalNotReady:
        return {
          title: 'Basis data entitas ini belum tersedia',
          description:
            'Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk ' +
            'melengkapi kredensial basis datanya.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Daftar member reas tidak dapat dimuat',
          description: error.message,
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Terjadi kesalahan pada sistem',
    description: 'Coba muat ulang halaman ini. Bila berulang, hubungi administrator Claim PNC.',
    tone: 'gangguan',
  }
}

/**
 * Mengubah galat penyimpanan menjadi kalimat yang dapat ditindaklanjuti.
 *
 * Dipisahkan dari loadMessage karena keadaannya berbeda: yang gagal di sini bukan membuka
 * layar melainkan menyimpan satu baris, dan pengguna sudah mengetik sesuatu yang tidak boleh
 * hilang.
 */
function saveMessage(error: unknown): string {
  if (error instanceof NetworkError) {
    return 'Server Claim PNC tidak dapat dihubungi. Periksa koneksi jaringan, lalu coba lagi.'
  }
  if (error instanceof APIError) {
    // 422 membawa keterangan per isian dari server; yang pertama sudah cukup karena form
    // ini hanya punya satu isian.
    const detail = error.detail?.[0]?.pesan
    if (detail !== undefined && detail !== '') return detail
    return error.message
  }
  return 'Terjadi kesalahan pada sistem. Coba lagi; bila berulang, hubungi administrator.'
}

/**
 * Layar Master Reas.
 *
 * Pengganti `Harness/DataMemberReas-harness.xml` atas tabel POOLDATA.T_REINSURER
 * (MENU_ID 35).
 *
 * # Apa yang ditampilkan layar ini
 *
 * **Daftar member reasuransi** — pihak yang menerima pemberitahuan PLA, Pre-DLA, dan DLA
 * atas klaim entitas ini, beserta login portal dan alamat surel tujuannya.
 *
 * # Layar BACA-SAJA, dan itu keputusan berdasar bukti
 *
 * Harness lamanya memuat satu grid dan satu tombol Refresh. Ketiadaan tombol **Tambah**
 * terkalibrasi — indeks rule `Harness/MasterLoginSurvey-Harness.xml`, yang layarnya terbukti
 * punya dua tombol, menyebut `PYBUTTONLABEL!REFRESH` **dan** `PYBUTTONLABEL!TAMBAH`;
 * `DataMemberReas` hanya menyebut `REFRESH`.
 *
 * Dan satu-satunya penulis `T_REINSURER` di sistem lama adalah **alur PLA/DLA**, bukan layar
 * master:
 *
 *	Database/UPDATEREAS.prc              prosedur upsert-nya
 *	RDB List/UpdateEmailReas-SQL.xml     satu-satunya pemanggil prosedur itu
 *	Activity/UpdateDetailPLA2-Act.xml    memanggilnya  ← layar detail PLA
 *	Activity/UpdateDetailDLA2-Act.xml    memanggilnya  ← layar detail DLA
 *
 * Jadi baris reasuransi lahir dan berubah sebagai efek samping pengiriman PLA/DLA, bukan
 * lewat pemeliharaan master. Layar ini meniru itu apa adanya (`P-5`) alih-alih menambah
 * kewenangan yang tidak pernah ada — pada tabel yang menentukan ke mana pemberitahuan klaim
 * dikirim, kewenangan yang tidak pernah diminta siapa pun adalah risiko tanpa imbalan.
 *
 * # Kepala kolomnya DIBERITAHU Work Owner, bukan direkonstruksi
 *
 * Section grid `BrowseListMemberReas` **tidak ada di antara 2.634 berkas export** (`R-16`),
 * sehingga daftar kolomnya sempat direkonstruksi dari kueri yang ada. Rekonstruksi itu
 * **meleset di dua tempat**, dan Work Owner mengoreksinya pada 2026-10-05 dengan menyebut
 * kepala kolom layar Pega apa adanya:
 *
 *	No · Nama Reinsurer · Login · Email · Tipe · Aksi
 *
 * Yang meleset: "Kode Reas" ternyata adalah kolom **"No"** — sama seperti layar master lain
 * yang menaruh ID induk di sana — dan **"Negara" tidak punya kolom sama sekali**.
 *
 * `COUNTRY` tetap dibaca dan tetap dikirim server; ia terbaca di panel **Detail**, karena
 * isinya ikut tercetak di dokumen PLA/DLA (`GetDataPreDLA`, `BrowseAllDataXOL_PLA`) dan
 * tidak boleh terkirim ke pihak luar tanpa satu pun tempat untuk dilihat petugas.
 *
 * # Satu perusahaan dapat muncul beberapa kali, dan kolom Tipe yang menjelaskannya
 *
 * Tanpa kolom Tipe, daftar akan terlihat memuat nama yang sama berkali-kali tanpa sebab.
 * Lihat catatan pada kolomnya.
 *
 * # Blok catatan di kaki halaman DIHAPUS
 *
 * Ditetapkan Work Owner 2026-10-05. Isinya menyebut nama rule Pega, nama kolom Oracle, dan
 * nomor keputusan — tidak berarti apa-apa bagi petugas klaim, dan membuat layar tampak belum
 * selesai. Catatan resminya hidup di doc comment paket `masterreas` dan di `docs/`.
 */
export function ReasMemberPage() {
  const portal = useSelectedPortal((state) => state.alias)

  const list = useReasMemberList()
  const rows = list.data?.member_reas ?? []

  // Baris yang sedang disunting lewat tombol Ubah. null berarti formnya tertutup.
  const [editing, setEditing] = useState<ReasMember | null>(null)
  const [email, setEmail] = useState('')

  const save = useSaveReasMemberEmail()

  function openEdit(row: ReasMember) {
    save.reset()
    setEditing(row)
    setEmail(row.email)
  }

  function closeEdit() {
    save.reset()
    setEditing(null)
    setEmail('')
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    if (editing === null) return

    // Form ditutup HANYA setelah server menjawab berhasil. Menutupnya lebih dulu akan
    // membuang isian pengguna saat penyimpanan gagal — dan penolakan "baris sudah tidak ada"
    // benar-benar mungkin di sini, karena alur PLA/DLA menulis ke tabel yang sama.
    save.mutate(
      {
        kode_reas: editing.kode_reas,
        nama_reas: editing.nama_reas,
        tipe: editing.tipe,
        email,
      },
      { onSuccess: closeEdit },
    )
  }

  /*
    ENAM kolom, dan urutannya mengikuti kepala kolom layar Pega apa adanya — ditetapkan Work
    Owner 2026-10-05:

      No · Nama Reinsurer · Login · Email · Tipe · Aksi

    Ia MENGGANTIKAN susunan sebelumnya, yang direkonstruksi dari SELECT terlengkap atas tabel
    ini (`RDB List/BrowseEmailReas-SQL.xml`) karena section gridnya hilang dari export
    (`R-16`). Rekonstruksi itu meleset di dua tempat, dan keduanya kini diperbaiki:

      "Kode Reas"  ->  masuk ke kolom "No"; lihat catatan pada kolomnya
      "Negara"     ->  TIDAK ada kolomnya di Pega, jadi tidak digambar

    Alias klipboard Pega tetap tidak dibawa — `email as "City"` dan
    `reinsurername as "District"` adalah utang `03-CURRENT-ARCHITECTURE.md` §4.2, bukan nama
    yang layak ditiru.
  */
  /*
    LEBAR KOLOM — nilai CSS, bukan kelas Tailwind

    `DataTable` memasangnya lewat `style={{ width }}`, sehingga `'w-24'` tidak berlaku apa-apa
    (dan diam-diam diabaikan peramban).

    Angkanya TIDAK diambil dari `pyWidth` sel Pega seperti pada modul migrasi lain: section
    gridnya hilang dari export (`R-16`), jadi lebar aslinya tidak diketahui. Ia dipilih
    menurut ISI kolomnya, dan dijumlahkan supaya muat di `max-w-6xl` (72rem dikurangi padding
    ≈ 70rem):

        No 3  +  Nama 24  +  Login 12  +  Email 18  +  Tipe 5  +  Aksi 6  =  68rem

    # Kenapa Email ikut diberi lebar, dan kenapa Tipe dipersempit

    Tabelnya `table-auto`, sehingga `width` hanya SARAN — peramban membagi ulang menurut isi.
    Alamat surel adalah teks panjang tanpa spasi, jadi tanpa lebar eksplisit ia merebut ruang
    dari kolom di sebelahnya; itulah yang membuat Nama Reinsurer tampak sempit.

    `Tipe` sebelumnya 9rem untuk isi SATU karakter. Ruang yang dibebaskannya dipindahkan ke
    Nama Reinsurer, yang memuat nama perusahaan reasuransi — teks terpanjang di tabel ini dan
    satu-satunya yang dipakai orang mengenali barisnya.
  */
  const columns: Column<ReasMember>[] = [
    {
      /*
        "No" memuat KODE REAS (`REINSURERID`), bukan nomor urut tampilan.

        Itu konvensi yang sudah berlaku di layar lain — `RejectionPage` menuliskannya
        terang-terangan: "ID induk, bukan nomor urut". Nomor urut tampilan juga tidak dapat
        dibuat di sini: `Column.render` tidak menerima indeks baris, dan angka yang dihitung
        sebelum `DataTable` menyaring serta mengurutkan akan berantakan begitu pengguna
        mengurutkan kolom mana pun.

        Ia sekaligus menjelaskan kenapa "Kode Reas" tidak ada di daftar kepala kolom Pega:
        kolom itulah "No".
      */
      key: 'no',
      title: 'No',
      width: '3rem',
      // Kosong: yang digambar adalah NOMOR URUT, bukan isi baris. Nilai kosong membuatnya
      // tidak ikut tercari — mencari "3" seharusnya tidak menemukan baris ketiga.
      value: () => '',
      // Tidak dapat diurutkan: mengurutkan menurut nomor urut tidak berarti apa-apa, dan
      // hasilnya justru menomori ulang barisnya.
      noSort: true,
      render: (_row, nomor) => <span className="text-slate-500">{nomor}</span>,
    },
    {
      key: 'nama_reas',
      title: 'Nama Reinsurer',
      width: '24rem',
      /*
        Kode reas TIDAK digambar di sini — Pega tidak menempelkannya di bawah nama, dan
        menambahkannya berarti mengarang tampilan (ketetapan Work Owner 2026-10-05).

        Ia tetap ikut di `value`, sehingga tetap DAPAT DICARI meski tidak terlihat. Itu
        menjaga janji label pencariannya ("Cari kode, nama, login, atau email") dan
        menyamakan perilaku pencarian layar dengan penyaring `cari` di server, yang memang
        menyertakan `REINSURERID`.

        Urutannya nama lebih dulu, sehingga pengurutan kolom ini tetap menurut nama.

        `render` WAJIB ada meski isinya sekadar namanya: tanpa itu `DataTable` menggambar
        `value` apa adanya — dan `value` sengaja memuat kode reas, sehingga kodenya akan
        tergambar justru lewat pintu belakang.
      */
      value: (row) => `${row.nama_reas} ${row.kode_reas}`,
      /*
        `minWidth` dipasang pada ISI selnya, bukan hanya lewat `width` kolomnya.

        Sebabnya: tabel `DataTable` memakai `table-auto`, dan di mode itu `width` hanya
        SARAN — peramban membagi ruang menurut panjang isi tiap kolom. Alamat surel adalah
        teks panjang tanpa spasi, sehingga ia menang terhadap nama perusahaan yang pendek
        seperti "AACHEN", dan kolom ini menyusut sampai kepala kolomnya patah dua baris
        ("Nama" / "Reinsurer").

        Lebar minimum pada isinya menaikkan lebar min-content kolom, dan itu BUKAN saran —
        peramban wajib memenuhinya. 16rem cukup memuat kepala kolomnya dalam satu baris
        beserta nama perusahaan yang panjang.

        Memasang `table-fixed` pada komponennya akan menyelesaikan ini secara umum, tetapi
        ia mengubah tata letak SELURUH layar yang memakai `DataTable` — perubahan yang
        menuntut keputusan tersendiri, bukan efek samping dari satu modul.
      */
      render: (row) => (
        <span className="block" style={{ minWidth: '16rem' }}>
          {row.nama_reas}
        </span>
      ),
    },
    {
      key: 'login',
      title: 'Login',
      width: '12rem',
      value: (row) => row.login,
      render: (row) =>
        row.login ? (
          row.login
        ) : (
          // Login kosong berarti mitra pada baris itu tidak akan pernah melihat klaimnya
          // sendiri — lima kueri inbox menyaringnya. Menampilkannya sebagai sel kosong
          // membuat keadaan itu tidak terlihat sama sekali.
          <span className="text-amber-700">belum ada</span>
        ),
    },
    {
      key: 'email',
      title: 'Email',
      width: '18rem',
      value: (row) => row.email,
      render: (row) =>
        row.email ? (
          row.email
        ) : (
          // Baris tanpa surel gagal dalam diam: dokumen PLA/DLA-nya terbit, tercatat
          // terkirim, dan tidak pernah sampai ke siapa pun.
          <span className="text-amber-700">belum ada</span>
        ),
    },
    {
      /*
        Tipe TIDAK ada di SELECT mana pun pada rule lama — ia hanya dipakai sebagai
        PENYARING (`BrowseEmailReas`) dan sebagai pembanding (`GetDataPreDLA`,
        `substr(NODLA,0,1) = TYPE`).

        Ia ditampilkan di sini sebagai PENAMBAHAN yang disadari, dan alasannya bukan
        kelengkapan: tanpa kolom ini, satu perusahaan dengan tiga jenis dokumen muncul
        sebagai tiga baris yang terlihat kembar, dengan surel berbeda-beda dan tanpa satu
        pun keterangan mengapa. Daftar seperti itu tidak dapat dipercaya pembacanya.

        Nilainya ditampilkan APA ADANYA. Artinya dalam bahasa bisnis tidak diketahui
        (`R-16`), dan menerjemahkannya menjadi label berarti mengarang.
      */
      key: 'tipe',
      title: 'Tipe',
      width: '5rem',
      value: (row) => row.tipe,
      /*
        Nilainya digambar APA ADANYA, tanpa penanda "cadangan" — Pega tidak punya penanda
        itu, dan menambahkannya berarti mengarang tampilan (ketetapan Work Owner 2026-10-05).

        Arti `TYPE = '1'` tetap dihitung server dan tetap dikirim sebagai field `cadangan`;
        yang dihapus adalah penggambarannya, bukan datanya. Keterangannya hidup di doc
        comment paket `masterreas` dan di `docs/`.
      */
      render: (row) => row.tipe || <span className="text-slate-400">—</span>,
    },
    {
      /*
        Kolom aksi. Judulnya "Aksi" — satu-satunya judul kolom di aplikasi ini yang sengaja
        TIDAK menyalin Pega, ditetapkan Work Owner 2026-10-03 dan berlaku seluruh modul.
        Mengosongkannya sudah dicoba dan ditolak: kolom tanpa judul tampak TIDAK ADA bagi
        pengguna yang membaca kepala tabel.

        Tombolnya **Ubah** — diberitahukan Work Owner 2026-10-05, dan tidak dapat dibaca dari
        export karena section gridnya hilang (`R-16`).

        Yang dapat diubah hanya **Email**. `Database/UPDATEREAS.prc` pada baris yang sudah ada
        memang hanya menyentuh kolom `EMAIl`; `LOGIN`, `COUNTRY`, dan `COUNTRYID` hanya
        ditulis pada jalur sisip, yang milik alur PLA/DLA dan tidak dibawa layar ini.
      */
      key: 'aksi',
      title: 'Aksi',
      width: '6rem',
      noSort: true,
      alignRight: true,
      value: () => '',
      render: (row) => (
        <Button tone="halus" onClick={() => { openEdit(row) }} disabled={save.isPending}>
          Ubah
        </Button>
      ),
    },
  ]

  return (
    <main className="mx-auto max-w-6xl px-4 py-8">
      <header className="flex flex-wrap items-start justify-between gap-4 border-b border-slate-200 pb-4">
        <div>
          {/* Judulnya "Master Reas" mengikuti MENU_DESC pada
              Database/m_menu_aplikasi_pnc.csv (MENU_ID 35). Caption section lamanya sendiri
              berbunyi "Data Member" — dipakai sebagai keterangan di bawahnya (D-13). */}
          <h1 className="text-xl font-semibold text-slate-900">Master Reas</h1>
          <p className="text-sm text-slate-600">
            Data Member — daftar mitra reasuransi penerima pemberitahuan PLA, Pre-DLA, dan
            DLA pada entitas ini.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {/* SATU tombol, dan itu memang satu-satunya yang ada di harness lamanya:
              `pyButtonLabel Refresh` pada Section/ListMemberReas. Captionnya dipakai apa
              adanya (D-13). */}
          <Button tone="kedua" onClick={() => { list.refetch() }} disabled={list.isFetching}>
            {list.isFetching ? 'Memuat…' : 'Refresh'}
          </Button>
        </div>
      </header>

      {/* Entitas yang sedang dilihat disebut terang-terangan. Satu aplikasi melayani empat
          badan hukum dengan basis data terpisah, dan "data siapa ini" tidak boleh hanya
          diandaikan pengguna (ADR-0030, R-20).

          Pada layar ini ia berarti lebih dari sekadar kerapian: yang tertera adalah alamat
          surel tujuan pemberitahuan klaim, dan mitra satu badan hukum bukan mitra badan
          hukum lain. */}
      <p className="mt-3 text-xs text-slate-500">
        Daftar ini memuat seluruh member reas pada entitas yang sedang dibuka.
        <span className="ml-1">
          Portal entitas:{' '}
          <span className="font-medium text-slate-700">{list.data?.portal ?? portal ?? '—'}</span>
        </span>
      </p>

      {/*
        Form ubah berada DI ATAS tabel, bukan di bawahnya — ketetapan Work Owner 2026-10-05,
        dan pola yang sama dipakai Master Login. Alasannya praktis: form di bawah tabel
        berada di luar layar pada daftar yang panjang, sehingga menekan Ubah tampak seperti
        tidak melakukan apa-apa.
      */}
      {editing !== null && (
        <section className="mt-5 rounded border border-slate-200 bg-slate-50 p-4">
          <h2 className="text-sm font-semibold text-slate-900">
            Ubah Email — {editing.nama_reas}
          </h2>

          {/* Keempat keterangan ini TIDAK dapat diubah; ia ditampilkan supaya petugas tahu
              baris mana yang sedang disunting. Ketiganya yang pertama adalah KUNCI barisnya
              (`UPDATEREAS` menyaring dengan ketiganya), dan Login tidak pernah disentuh
              jalur ubah di sistem lama. */}
          <dl className="mt-3 grid grid-cols-1 gap-x-6 gap-y-2 text-sm sm:grid-cols-4">
            <div>
              <dt className="text-xs text-slate-500">Kode Reas</dt>
              <dd className="text-slate-800">{editing.kode_reas || '—'}</dd>
            </div>
            <div>
              <dt className="text-xs text-slate-500">Login</dt>
              <dd className="text-slate-800">{editing.login || 'belum ada'}</dd>
            </div>
            <div>
              <dt className="text-xs text-slate-500">Negara</dt>
              <dd className="text-slate-800">{editing.negara || '—'}</dd>
            </div>
            <div>
              {/* Tanpa penanda "cadangan", dengan alasan yang sama seperti di kolomnya. */}
              <dt className="text-xs text-slate-500">Tipe</dt>
              <dd className="text-slate-800">{editing.tipe || '—'}</dd>
            </div>
          </dl>

          <form className="mt-4" onSubmit={submit}>
            <label className="block text-sm font-medium text-slate-700" htmlFor="email-reas">
              Email
            </label>
            <input
              id="email-reas"
              className="mt-1 w-full rounded border border-slate-300 px-3 py-2 text-sm sm:max-w-md"
              value={email}
              onChange={(e) => { setEmail(e.target.value) }}
              disabled={save.isPending}
            />

            {save.isError && (
              <p className="mt-2 text-sm text-rose-700">{saveMessage(save.error)}</p>
            )}

            <div className="mt-4 flex flex-wrap items-center gap-2">
              <Button tone="utama" type="submit" disabled={save.isPending}>
                {save.isPending ? 'Menyimpan…' : 'Simpan'}
              </Button>
              <Button tone="halus" onClick={closeEdit} disabled={save.isPending}>
                Batal
              </Button>
            </div>
          </form>
        </section>
      )}

      <section className="mt-6">
        {portal === null ? (
          <ErrorMessage
            title="Portal entitas belum dipilih"
            description="Data master dimiliki masing-masing entitas. Pilih portal entitas di bagian atas halaman ini lebih dulu."
            tone="penolakan"
          />
        ) : list.isPending ? (
          <p className="text-sm text-slate-500">Memuat daftar member reas…</p>
        ) : list.isError ? (
          (() => {
            const message = loadMessage(list.error)
            return (
              <ErrorMessage
                title={message.title}
                description={message.description}
                tone={message.tone}
              />
            )
          })()
        ) : (
          <DataTable
            columns={columns}
            rows={rows}
            /*
              Kunci barisnya TIGA kolom, bukan kode reas saja.

              Kunci alaminya memang begitu — `Database/UPDATEREAS.prc` memeriksa keberadaan
              baris dengan REINSURERID + REINSURERNAME + TYPE sekaligus — dan memakai kode
              reas sendirian akan membuat tiga baris milik satu perusahaan berbagi kunci yang
              sama. React akan menganggap ketiganya satu baris.

              Pemisahnya \u001f (unit separator), bukan tanda baca biasa yang dapat muncul di
              dalam nama perusahaan.
            */
            rowKey={(row) => [row.kode_reas, row.nama_reas, row.tipe].join('\u001f')}
            description="Sumber: POOLDATA.T_REINSURER"
            searchLabel="Cari kode, nama, login, atau email"
            pageSize={PAGE_SIZE}
            emptyMessage="Belum ada member reas pada entitas ini."
          />
        )}
      </section>

      {/*
        BLOK CATATAN DI KAKI HALAMAN DIHAPUS (Work Owner, 2026-10-05). Isinya menyebut nama
        rule Pega, nama kolom Oracle, dan nomor keputusan — tidak berarti apa-apa bagi
        petugas klaim, dan membuat layar tampak belum selesai. Catatan resminya tetap hidup
        di doc comment paket `masterreas` dan di `docs/`, bukan di layar.
      */}
    </main>
  )
}
