import { useEffect, useState, type ReactNode } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'

import { APIError } from '@/api/client'
import { Button } from '@/components/Button'
import { DateField } from '@/components/DateField'
import { ErrorMessage } from '@/components/ErrorMessage'
import { Field } from '@/components/Field'
import { NumberField } from '@/components/NumberField'
import { centsToRupiah, formatDate, rupiahToCents } from '@/components/format'

import { useClaimReport, useLookupPolicy, useRegisterClaim, useSaveClaimReport } from './api'
import { EMPTY_DETAIL, FIELD_LIMIT, type ClaimReportDetail, type PolicyLookupResponse } from './types'

/**
 * Form **Input Receive Document** — pengganti flow action dengan nama yang sama pada
 * `Flow/InputReceiveDocument.xml`.
 *
 * # Ia yang membuat tombol "Buat Baru" berarti
 *
 * Di sistem lama, menekan Buat Baru menjalankan `CreateNewCaseRCV`, yang membuat berkas
 * KOSONG lalu menyerahkannya ke alur Receive Document. Alur itu punya satu assignment —
 * "Receive Document" — dan assignment itu merender form ini. Berkasnya lahir kosong justru
 * supaya form inilah yang mengisinya.
 *
 * Tanpa layar ini, tombol Buat Baru menerbitkan berkas yang tidak dapat diapa-apakan —
 * dan itu tampak seperti tombol yang tidak bekerja.
 *
 * # Berkas milik Pega dibuka dalam modus baca saja
 *
 * Selama masa paralel, tepat satu sistem yang menulis sebuah baris (`ADR-0004`, `P-1`).
 * Kewenangannya dihitung SERVER dan dikirim sebagai `dapat_disunting`; layar tidak
 * menyimpulkannya sendiri dari kolom `asal`.
 */
export function ClaimReportFormPage() {
  const { id = '' } = useParams()
  const navigate = useNavigate()

  const berkas = useClaimReport(id)
  const simpan = useSaveClaimReport(id)
  const daftar = useRegisterClaim()
  const polis = useLookupPolicy()

  const [values, setValues] = useState<ClaimReportDetail>(EMPTY_DETAIL)
  const [estimateText, setEstimateText] = useState('')
  const [policyResult, setPolicyResult] = useState<PolicyLookupResponse | null>(null)
  const [lookedUpNumber, setLookedUpNumber] = useState<string | null>(null)

  // Isian form diisi SEKALI dari jawaban server, lalu menjadi milik pengguna. Menyalinnya
  // pada setiap render akan menimpa ketikan yang sedang berjalan setiap kali TanStack
  // Query menyegarkan datanya di latar belakang.
  useEffect(() => {
    if (!berkas.data?.isian) return
    setValues(berkas.data.isian)
    setEstimateText(centsToRupiah(berkas.data.isian.estimasi_kerugian))
  }, [berkas.data])

  const [initialLookup, setInitialLookup] = useState(false)
  useEffect(() => {
    const number = berkas.data?.isian?.nomor_polis?.trim() ?? ''
    if (initialLookup || number === '') return
    setInitialLookup(true)
    polis.mutate(number, { onSuccess: (result) => setPolicyResult(result) })
  }, [berkas.data, initialLookup, polis])

  // Isian khusus lini (InputReceiveDocument_sect): Group Panel polis yang tersimpan, atau
  // hasil pencarian polis terakhir.
  const panel = (policyResult?.group_panel || values.group_panel || '').trim()
  const hasPolicy = values.nomor_polis.trim() !== ''
  const pa = panel === '002'

  const registered = berkas.data?.sudah_diregistrasi ?? false
  const editable = (berkas.data?.dapat_disunting ?? false) && !registered
  const violation = simpan.error instanceof APIError ? simpan.error.violations() : {}

  // Tombol yang menulis berkas dimatikan untuk polis Syariah atau bukan PNC — di layar lama
  // `pyDisabledWhen` "SyariahStatus = '1' || TempError.ErrorNotes = '1'".
  const policyBlocked = policyResult?.memblokir ?? false

  function set<K extends keyof ClaimReportDetail>(field: K, value: ClaimReportDetail[K]) {
    setValues((previous) => ({ ...previous, [field]: value }))
  }

  /**
   * lookupPolicy menggantikan `PolisReceiveInternalExternal`: dijalankan saat isian Nomor
   * Polis ditinggalkan atau Enter ditekan.
   *
   * Langkah 12 activity lama SELALU menimpa kelima isian dengan hasil pencarian, juga
   * saat polisnya tidak ditemukan — isiannya menjadi kosong. Itu yang ditiru di sini,
   * supaya data polis lain yang tertinggal di form tidak ikut tersimpan.
   */
  function lookupPolicy() {
    const typed = values.nomor_polis.trim()
    if (!editable) return
    if (typed === '') {
      // Nomor dikosongkan: pesan dan pemblokiran polis sebelumnya tidak berlaku lagi.
      setPolicyResult(null)
      setLookedUpNumber(null)
      return
    }
    if (typed === lookedUpNumber) return
    setLookedUpNumber(typed)
    polis.mutate(typed, {
      onSuccess: (result) => {
        setPolicyResult(result)
        setLookedUpNumber(result.nomor_polis)
        setValues((previous) => ({
          ...previous,
          nomor_polis: result.nomor_polis,
          tertanggung: result.tertanggung,
          nama_bisnis: result.nama_bisnis,
          nomor_rujukan: result.nomor_rujukan,
          group_panel: result.group_panel,
        }))
      },
      onError: () => setLookedUpNumber(null),
    })
  }

  if (berkas.isPending) {
    return (
      <FormFrame id={id}>
        <p className="mt-6 text-sm text-slate-500" role="status">
          Memuat berkas…
        </p>
      </FormFrame>
    )
  }

  if (berkas.isError) {
    return (
      <FormFrame id={id}>
        <div className="mt-6">
          <ErrorMessage
            title="Berkas tidak dapat dibuka"
            description={messageOf(berkas.error)}
            tone="gangguan"
          />
        </div>
      </FormFrame>
    )
  }

  return (
    <FormFrame id={id} report={berkas.data?.laporan.posisi}>
      {registered && (
        <p
          className="mt-4 rounded-kartu border border-blue-200 bg-blue-50/80 px-4 py-3 text-sm text-blue-900"
          role="status"
        >
          This Receive Document is already registered as claim{' '}
          <strong>{berkas.data?.laporan.nomor_klaim}</strong> and can no longer be changed.
        </p>
      )}

      {!editable && !registered && (
        <div className="mt-4">
          <ErrorMessage
            title="Berkas ini hanya dapat dibaca"
            description={
              'Ia masih dikelola sistem lama, dan selama masa paralel hanya satu sistem ' +
              'yang boleh menulis sebuah berkas. Isinya dapat dilihat di sini; ' +
              'mengubahnya dilakukan di Pega.'
            }
            tone="gangguan"
          />
        </div>
      )}

      {simpan.isError && (
        <div className="mt-4">
          <ErrorMessage
            title="Berkas tidak dapat disimpan"
            description={messageOf(simpan.error)}
            tone="penolakan"
          />
        </div>
      )}

      {simpan.isSuccess && !simpan.isPending && (
        <p
          className="mt-4 rounded-kartu border border-blue-200 bg-blue-50/80 px-4 py-3 text-sm text-blue-900"
          role="status"
        >
          Berkas tersimpan.
        </p>
      )}

      <form
        className="mt-6 space-y-6"
        onSubmit={(e) => {
          e.preventDefault()
          if (!editable) return
          simpan.mutate({ ...values, estimasi_kerugian: rupiahToCents(estimateText) || 0 })
        }}
      >
        <Group title="Dokumen masuk">
          <ReadOnly label="Tanggal Input Dokumen" value={formatDate(berkas.data?.laporan.tanggal_masuk ?? '')} />
          <DateField
            id="tanggal_terima_dokumen"
            label="Tanggal Terima Dokumen"
            value={values.tanggal_terima_dokumen}
            onChange={(v) => set('tanggal_terima_dokumen', v)}
            error={violation['tanggal_terima_dokumen']}
            disabled={!editable}
          />
          <Field
            id="nama_pelapor"
            label="Nama Pengirim / Pelapor Dokumen"
            value={values.nama_pelapor}
            onChange={(e) => set('nama_pelapor', e.target.value)}
            maxLength={FIELD_LIMIT.nama}
            error={violation['nama_pelapor']}
            disabled={!editable}
          />
          <Field
            id="email_pelapor"
            label="Email Pengirim"
            type="email"
            value={values.email_pelapor}
            onChange={(e) => set('email_pelapor', e.target.value)}
            maxLength={FIELD_LIMIT.email}
            error={violation['email_pelapor']}
            disabled={!editable}
          />
          <Field
            id="telepon_pelapor"
            label="No. HP Pengirim"
            value={values.telepon_pelapor}
            onChange={(e) => set('telepon_pelapor', e.target.value)}
            maxLength={FIELD_LIMIT.telepon}
            error={violation['telepon_pelapor']}
            disabled={!editable}
          />
          <Field
            id="nama_kurir"
            label="Nama Kurir ASM"
            value={values.nama_kurir}
            onChange={(e) => set('nama_kurir', e.target.value)}
            maxLength={FIELD_LIMIT.nama}
            error={violation['nama_kurir']}
            disabled={!editable}
          />
          <NumberField
            id="jumlah_dokumen"
            label="Total Jumlah Dokumen"
            max={FIELD_LIMIT.jumlahDokumen}
            value={values.jumlah_dokumen}
            onValueChange={(jumlah_dokumen) => set('jumlah_dokumen', jumlah_dokumen)}
            error={violation['jumlah_dokumen']}
            disabled={!editable}
            hint="Rincian per dokumen menunggu modul penyimpanan dokumen."
          />
        </Group>

        <Group title="Polis dan kejadian">
          <div>
            <Field
              id="nomor_polis"
              label="Nomor Polis"
              value={values.nomor_polis}
              onChange={(e) => set('nomor_polis', e.target.value)}
              onBlur={lookupPolicy}
              onKeyDown={(e) => {
                if (e.key === 'Enter') {
                  e.preventDefault()
                  lookupPolicy()
                }
              }}
              maxLength={FIELD_LIMIT.polis}
              error={violation['nomor_polis']}
              disabled={!editable}
              hint={polis.isPending ? 'Mencari polis…' : 'Data polis terisi otomatis setelah nomor diisi.'}
            />
            {polis.isError && (
              <p className="mt-1.5 text-sm text-red-700" role="alert">
                Data polis tidak dapat dibaca: {messageOf(polis.error)}
              </p>
            )}
            {(policyResult?.pesan ?? []).map((notice) => (
              <p
                key={notice.kode}
                role="alert"
                className={[
                  'mt-1.5 text-sm',
                  notice.memblokir ? 'text-red-700' : 'text-amber-700',
                ].join(' ')}
              >
                {notice.pesan}
              </p>
            ))}
          </div>
          <DateField
            id="tanggal_kejadian"
            label="Tanggal Kejadian"
            value={values.tanggal_kejadian}
            onChange={(v) => set('tanggal_kejadian', v)}
            error={violation['tanggal_kejadian']}
            disabled={!editable}
          />
          {/* .Policy.PolicyLeader — tampil bila Nomor Polis terisi dan lini bukan PA (002) atau Travel (005). */}
          {hasPolicy && panel !== '' && panel !== '002' && panel !== '005' && (
            <ReadOnly label="Polis Leader" value={policyResult?.polis_leader ?? ''} />
          )}
          <Field
            id="tertanggung"
            label="Nama Tertanggung"
            value={values.tertanggung}
            onChange={(e) => set('tertanggung', e.target.value)}
            maxLength={FIELD_LIMIT.nama}
            error={violation['tertanggung']}
            disabled={!editable}
          />
          <Field
            id="nama_bisnis"
            label="Nama Bisnis"
            value={values.nama_bisnis}
            onChange={(e) => set('nama_bisnis', e.target.value)}
            maxLength={FIELD_LIMIT.nama}
            error={violation['nama_bisnis']}
            disabled={!editable}
          />
          <Field
            id="nomor_rujukan"
            label="No. Referensi/Placing Slip"
            value={values.nomor_rujukan}
            onChange={(e) => set('nomor_rujukan', e.target.value)}
            maxLength={FIELD_LIMIT.rujukan}
            error={violation['nomor_rujukan']}
            disabled={!editable}
          />
          {/*
            Uang diketik sebagai teks lalu diubah menjadi SEN saat dikirim. Memakai
            <input type="number"> untuk rupiah membuat peramban menyimpannya sebagai
            pecahan biner, dan `ADR-0016` menuntut presisi penuh.
          */}
          <Field
            id="estimasi_kerugian"
            label="Estimasi Kerugian"
            value={estimateText}
            onChange={(e) => setEstimateText(e.target.value)}
            placeholder="2.500.000,00"
            inputMode="decimal"
            error={violation['estimasi_kerugian']}
            disabled={!editable}
            hint="Angka yang disebut pelapor; bukan nilai klaim."
          />
          <TextArea
            id="sumber_laporan"
            label="Source Of Reports"
            value={values.sumber_laporan}
            onChange={(value) => set('sumber_laporan', value)}
            maxLength={FIELD_LIMIT.sumber}
            error={violation['sumber_laporan']}
            disabled={!editable}
            rows={2}
          />
          {/* .ReceiveDocument.EmailLOD — hanya PA, setelah Nomor Polis terisi. */}
          {hasPolicy && pa && (
            <Field
              id="email_tertanggung"
              label="Email Tertanggung"
              type="email"
              value={values.email_tertanggung}
              onChange={(e) => set('email_tertanggung', e.target.value)}
              maxLength={FIELD_LIMIT.email}
              error={violation['email_tertanggung']}
              disabled={!editable}
            />
          )}
          <Field
            id="lokasi_kejadian"
            label="Lokasi Kejadian"
            value={values.lokasi_kejadian}
            onChange={(e) => set('lokasi_kejadian', e.target.value)}
            maxLength={FIELD_LIMIT.lokasi}
            error={violation['lokasi_kejadian']}
            disabled={!editable}
          />
          {/*
            .ReceiveDocument.SIM — hanya PA. Labelnya di section "Lokasi Kejadian" (salah
            salin di Pega): isinya disimpan ke SIMPENGENDARA (Rcv_ProcInsertRecivedDocument),
            lebar 25. Diberi label "SIM Pengendara" atas permintaan Work Owner, 2026-10-01.
          */}
          {pa && (
            <Field
              id="sim_pengendara"
              label="SIM Pengendara"
              value={values.sim_pengendara}
              onChange={(e) => set('sim_pengendara', e.target.value)}
              maxLength={FIELD_LIMIT.sim}
              error={violation['sim_pengendara']}
              disabled={!editable}
            />
          )}
          <Field
            id="subjek_email"
            label="Subject Email"
            value={values.subjek_email}
            onChange={(e) => set('subjek_email', e.target.value)}
            maxLength={FIELD_LIMIT.subjek}
            error={violation['subjek_email']}
            disabled={!editable}
          />
        </Group>

        <Group title="Keterangan" full>
          <TextArea
            id="kronologis"
            label="Kronologis Kejadian"
            value={values.kronologis}
            onChange={(value) => set('kronologis', value)}
            maxLength={FIELD_LIMIT.narasi}
            error={violation['kronologis']}
            disabled={!editable}
          />
          <TextArea
            id="rincian_kerusakan"
            label="Rincian Kerusakan"
            value={values.rincian_kerusakan}
            onChange={(value) => set('rincian_kerusakan', value)}
            maxLength={FIELD_LIMIT.narasi}
            error={violation['rincian_kerusakan']}
            disabled={!editable}
          />
          <TextArea
            id="alasan"
            label="Keterangan Belum Transfer"
            value={values.alasan}
            onChange={(value) => set('alasan', value)}
            maxLength={FIELD_LIMIT.catatan}
            error={violation['alasan']}
            disabled={!editable}
            rows={3}
          />
          <TextArea
            id="keterangan_belum_registrasi"
            label="Keterangan Belum Registrasi"
            value={values.keterangan_belum_registrasi}
            onChange={(value) => set('keterangan_belum_registrasi', value)}
            maxLength={FIELD_LIMIT.catatan}
            error={violation['keterangan_belum_registrasi']}
            disabled={!editable}
            rows={3}
          />
        </Group>

        <div className="flex flex-wrap items-center gap-3 border-t border-slate-200 pt-5">
          {editable && (
            <Button type="submit" tone="utama" disabled={simpan.isPending || policyBlocked}>
              {simpan.isPending ? 'Menyimpan…' : 'Simpan'}
            </Button>
          )}
          {/*
            Register Klaim ADA di layar lama dan tempatnya di sini — tombol keempat pada
            deret bawah `Section/InputReceiveDocument_sect.xml`, memanggil activity
            `CreateRegisterKlaimPNC`, yang langkah pertamanya `Call CreateInputKlaim`.

            Di sini ia memanggil `POST /api/registrasi/klaim`, yang mengerjakan langkah itu:
            membuat klaim, mengambil snapshot polis, menerbitkan nomor `PNCN.YY.xxxx`, dan
            mengisi `RCVID` dengan nomor laporan ini. Tautan balik itulah yang membuat baris
            RCV-nya berpindah keluar dari tab "Not Transferred".

            # Kenapa ia ikut mati saat berkas tidak dapat disunting

            Mendaftarkan klaim MENULIS ke berkas laporan — `NOKLAIM`-nya terisi. Berkas milik
            Pega hanya boleh dibaca selama masa paralel (`P-1`, `ADR-0004`), sehingga tombol
            ini tunduk pada kewenangan yang sama dengan Simpan. Kewenangannya dihitung
            server, bukan disimpulkan layar.

            # Kenapa nomor polis tidak diperiksa di sini

            Ia diperiksa di `useRegisterClaim`, satu tempat, supaya pesan penolakannya sama
            dari mana pun pendaftaran dimulai.

            # Tidak digambar sama sekali setelah berkas menjadi klaim

            Work Owner, 2026-09-29: Simpan dan Register Klaim tidak dimunculkan bila RCVN
            sudah punya PNCN. Server juga menolak Register Klaim kedua.
          */}
          {!registered && (
            <Button
              type="button"
              tone="kedua"
              disabled={!editable || daftar.isPending || policyBlocked}
              onClick={() =>
                daftar.mutate(
                  { nomorLaporan: id ?? '', nomorPolis: values.nomor_polis },
                  { onSuccess: (hasil) => navigate(`/registrasi/klaim/${hasil.klaim.id}`) },
                )
              }
            >
              {daftar.isPending ? 'Mendaftarkan…' : 'Register Klaim'}
            </Button>
          )}

          <Button
            type="button"
            tone="kedua"
            onClick={() => navigate(backToListPath(berkas.data?.laporan.posisi))}
          >
            Kembali ke daftar
          </Button>
        </div>

        {daftar.isError && (
          <div className="mt-3">
            <ErrorMessage
              title="Klaim tidak dapat didaftarkan"
              description={messageOf(daftar.error)}
              tone={toneOf(daftar.error)}
            />
          </div>
        )}
      </form>

      <p className="mt-6 text-xs text-slate-500">
        Tiga bagian form lama belum ada di sini: data pelapor beserta alamatnya — yang di
        sistem lama terikat area Heavy Equipment dan berada di luar lingkup migrasi — daftar
        rincian dokumen yang menunggu modul penyimpanan dokumen, serta riwayat komunikasi
        dan progres yang dimiliki modul lain.
      </p>
    </FormFrame>
  )
}

function FormFrame({
  id,
  report,
  children,
}: {
  id: string
  report?: string | undefined
  children: ReactNode
}) {
  return (
    <div className="mx-auto max-w-5xl px-4 py-8">
      <nav className="text-xs text-slate-500">
        <Link to="/inbox/laporan-klaim" className="underline hover:text-slate-800">
          Inbox Laporan Klaim
        </Link>
        <span aria-hidden="true"> › </span>
        <span>{id}</span>
      </nav>

      <header className="mt-2 border-b border-slate-200 pb-4">
        <h1 className="text-xl font-semibold text-slate-900">Input Receive Document</h1>
        <p className="mt-1 text-sm text-slate-600">
          Berkas <span className="font-medium text-slate-900">{id}</span>
          {report && <> · {report}</>}
        </p>
      </header>

      {children}
    </div>
  )
}

/** Satu kelompok isian, mengikuti pengelompokan form lama. */
function Group({
  title,
  children,
  full = false,
}: {
  title: string
  children: ReactNode
  full?: boolean
}) {
  return (
    <section className="rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut">
      <h2 className="text-xs font-medium uppercase tracking-wide text-slate-500">{title}</h2>
      <div className={['mt-4 grid gap-4', full ? '' : 'sm:grid-cols-2'].join(' ')}>{children}</div>
    </section>
  )
}

/**
 * Isian bernarasi panjang.
 *
 * Ia tidak memakai `Field` karena `Field` membungkus `<input>`, dan kronologis kejadian
 * berlebar 4.000 karakter — satu baris tidak dapat menampungnya. Bentuk, kelas, dan
 * penandaan aria-nya dijaga sama dengan `Field` supaya form tidak terlihat seperti
 * dirakit dari dua aplikasi berbeda.
 */
/** ReadOnly menampilkan isian baca saja dengan label seperti Field. */
function ReadOnly({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span className="block text-sm font-medium text-slate-700">{label}</span>
      <p className="mt-1 min-h-[42px] rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-2 text-slate-900">
        {value || '—'}
      </p>
    </div>
  )
}

function TextArea({
  id,
  label,
  value,
  onChange,
  maxLength,
  error,
  disabled,
  rows = 5,
}: {
  id: string
  label: string
  value: string
  onChange: (value: string) => void
  maxLength: number
  error?: string | undefined
  disabled?: boolean
  rows?: number
}) {
  const errorID = `${id}-galat`
  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-slate-700">
        {label}
      </label>
      <textarea
        id={id}
        rows={rows}
        value={value}
        maxLength={maxLength}
        disabled={disabled}
        onChange={(e) => onChange(e.target.value)}
        aria-invalid={error ? true : undefined}
        aria-describedby={error ? errorID : undefined}
        className={[
          'mt-1.5 w-full rounded-kontrol border bg-white px-3 py-2.5 text-sm text-slate-900',
          'transition-[border-color,box-shadow] duration-150 ease-halus',
          'focus:outline-none focus:ring-4',
          error
            ? 'border-red-400 focus:border-red-500 focus:ring-red-500/15'
            : 'border-slate-300 hover:border-slate-400 focus:border-blue-500 focus:ring-blue-500/15',
          'disabled:cursor-not-allowed disabled:bg-slate-50 disabled:text-slate-500',
        ].join(' ')}
      />
      {error && (
        <p id={errorID} className="mt-1.5 text-sm text-red-700">
          {error}
        </p>
      )}
    </div>
  )
}

/**
 * backToListPath mengembalikan alamat daftar pada TAB TEMPAT BERKAS INI BERADA.
 *
 * # Kenapa bukan sekadar kembali ke daftar
 *
 * Daftar selalu terbuka pada tab Outstanding — sama seperti layar lama. Berkas yang baru
 * dibuat berposisi "Not Transferred" karena belum bernomor klaim dan belum diserahkan,
 * sehingga ia TIDAK ada di tab itu.
 *
 * Akibatnya petugas yang menekan "Buat Baru" lalu kembali melihat daftar tanpa berkasnya,
 * dan menyimpulkan pembuatannya gagal — padahal barisnya tersimpan. Itu keluhan nyata
 * (Work Owner, 2026-09-24), dan kelas kegagalan yang paling mahal: yang berhasil tetapi
 * tampak gagal.
 *
 * Posisi yang tidak dikenali mengembalikan tab bawaan, bukan menebak.
 */
export function backToListPath(position: string | undefined): string {
  const tab: Record<string, string> = {
    Outstanding: 'outstanding',
    'Not Registered': 'belum-registrasi',
    'Not Transferred': 'belum-diserahkan',
  }
  const kode = position === undefined ? undefined : tab[position]
  return kode === undefined ? '/inbox/laporan-klaim' : `/inbox/laporan-klaim?kategori=${kode}`
}

function messageOf(failure: unknown): string {
  if (failure instanceof APIError) return failure.message
  if (failure instanceof Error) return failure.message
  return 'Terjadi kesalahan pada sistem.'
}

/**
 * toneOf memisahkan penolakan dari gangguan.
 *
 * "Nomor Polis harus diisi" adalah pekerjaan pengguna; "penyimpanan belum siap" bukan.
 * Menggambarkan keduanya dengan warna yang sama membuat petugas mencoba memperbaiki hal
 * yang tidak dapat mereka perbaiki.
 */
function toneOf(failure: unknown): 'penolakan' | 'gangguan' {
  if (failure instanceof APIError && failure.status >= 500) return 'gangguan'
  return 'penolakan'
}
