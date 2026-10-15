import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect, useId } from "react";
import { useFieldArray, useForm } from "react-hook-form";
import { z } from "zod";

import { APIError, NetworkError } from "@/api/client";
import {
  ErrorCode,
  PanelErrorCode,
  PanelSide,
  type Panel,
  type PanelInput,
} from "@/api/types";
import { Button } from "@/components/Button";
import { ErrorMessage, type ErrorTone } from "@/components/ErrorMessage";
import { Field } from "@/components/Field";

import { usePanelOptions } from "./api";
import { LocationEditor } from "./LocationEditor";

/**
 * Batas panjang harus sama dengan konstanta di `internal/masterpanel/masterpanel.go`.
 *
 * SELURUHNYA ASUMSI YANG DISADARI: DDL POOLDATA.PANEL_HE dan LOKASI_PANEL_HE belum
 * diterima (R-08), dan sistem lama tidak memeriksa panjang satu pun isian. Batasnya
 * dipasang supaya penolakan datang sebagai kalimat yang menuntun, bukan sebagai ORA-12899.
 *
 * Bila salah satu berubah, KEDUA tempat harus ikut berubah. Uji
 * `TestLengthLimitsAreTheOnesTheFormRepeats` di backend menjaga duplikasinya tetap
 * terlihat.
 */
const MAX = {
  nama: 100,
  kode: 20,
  lokasi: 50,
  baris: 50,
} as const;

/**
 * Kelima sandi sisi yang sah — tiga, dan ketiganya terbaca dari export.
 *
 * Berbeda dari kesembilan penanda STS_* yang daftar pilihannya TIDAK ada di export
 * (`R-16`), daftar ini berasal dari `Activity/SetLokasiSisiPanel-Act.xml` dan dikuatkan
 * `Activity/GetSisiPanel-Act.xml`. Karena itu ia boleh dipakai sebagai aturan validasi.
 */
const SIDE_VALUES = [
  PanelSide.tidakAda,
  PanelSide.kiri,
  PanelSide.kanan,
] as const;

const required = (max: number, label: string) =>
  z
    .string()
    .trim()
    .min(1, `${label} wajib diisi.`)
    .max(max, `${label} paling panjang ${max} karakter.`);

const schema = z.object({
  nama_panel: required(MAX.nama, "Nama panel"),

  // Kesepuluh isian induk WAJIB, dan itu bukan tambahan: kesepuluhnya bertanda
  // `pyRequired=true` pada `Section/BrowsePanelHEApproval-Section.xml`.
  status_repair: required(MAX.kode, "Status repair"),
  status_edit_quantity: required(MAX.kode, "Status edit quantity"),
  status_premium_repair: required(MAX.kode, "Status premium repair"),
  status_pecah: required(MAX.kode, "Status pecah"),
  status_sticker: required(MAX.kode, "Status sticker"),
  status_sisi: required(MAX.kode, "Status sisi"),
  status_rusak_parah: required(MAX.kode, "Status rusak parah"),
  status_aktif: required(MAX.kode, "Status aktif"),
  exclusion_c: required(MAX.kode, "Exclusion C"),

  lokasi: z
    .array(
      z.object({
        lokasi_panel: required(MAX.lokasi, "Lokasi panel"),
        sisi_panel: z
          .string()
          .trim()
          .refine(
            (value) => (SIDE_VALUES as readonly string[]).includes(value),
            'Sisi panel hanya boleh "-", KIRI, atau KANAN.',
          ),
      }),
    )
    .max(
      MAX.baris,
      `Lokasi panel paling banyak ${MAX.baris} baris dalam satu panel.`,
    )
    // Lokasi yang sama pada sisi yang sama tidak boleh muncul dua kali. Sistem lama tidak
    // memeriksanya, dan dua baris kembar membuat `GetDataSisiPanel` — yang membaca satu
    // nilai saja — mengembalikan baris yang mana pun lebih dulu ditemukan.
    .superRefine((list, ctx) => {
      const seen = new Map<string, number>();
      list.forEach((one, index) => {
        const key = `${one.lokasi_panel.trim().toUpperCase()}\u0000${one.sisi_panel}`;
        const first = seen.get(key);
        if (first !== undefined) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: [index, "lokasi_panel"],
            message: `Lokasi ini sudah ada di baris ${first + 1}.`,
          });
          return;
        }
        seen.set(key, index);
      });
    }),
});

export type PanelFields = z.infer<typeof schema>;

/** Nilai yang dikirim ke pemanggil saat form disimpan. */
export type PanelFormValues = PanelInput;

type Props = {
  /**
   * Baris yang sedang disunting, atau null bila menambah.
   *
   * Mode hanya menentukan judul dan teks tombolnya — seluruh isian dapat diubah pada kedua
   * mode, sama seperti layar lama yang memakai form yang sama untuk keduanya.
   */
  editing: Panel | null;

  /**
   * Nilai yang SUDAH DIPAKAI baris lain, per isian.
   *
   * Dipakai sebagai saran pada kesembilan penanda yang daftar pilihannya TIDAK ADA di
   * export Pega (R-16). Sarannya berasal dari data, bukan dari tebakan.
   */
  knownValues: Record<string, string[]>;

  isSaving: boolean;
  error: unknown;
  onSave: (values: PanelFormValues) => void;
  onCancel: () => void;
};

type MessageContent = { title: string; description: string; tone: ErrorTone };

/**
 * Mengubah galat penyimpanan menjadi pesan yang dapat ditindaklanjuti.
 *
 * Galat validasi TIDAK ditangani di sini — ia disorot per isian (lihat violationsOf). Yang
 * ditampilkan sebagai kotak pesan hanyalah galat yang tidak menunjuk isian tertentu,
 * karena itulah yang tidak dapat diperbaiki pengguna dengan mengetik.
 */
function messageFor(error: unknown): MessageContent | null {
  if (error instanceof NetworkError) {
    return {
      title: "Server Claim PNC tidak dapat dihubungi",
      description:
        "Isian Anda belum tersimpan. Periksa koneksi jaringan, lalu simpan lagi.",
      tone: "gangguan",
    };
  }
  if (error instanceof APIError) {
    switch (error.kode) {
      case PanelErrorCode.nameTaken:
        return {
          title: "Nama panel itu sudah dipakai",
          description:
            "Buka baris yang sudah ada untuk mengubahnya, atau pakai nama lain.",
          tone: "penolakan",
        };
      case ErrorCode.validationFailed:
        // Bila detailnya ada, isiannya sudah disorot satu per satu; kotak pesan hanya akan
        // mengulang hal yang sama.
        return Object.keys(error.violations()).length > 0
          ? null
          : {
              title: "Belum dapat disimpan",
              description: error.message,
              tone: "penolakan",
            };
      case ErrorCode.portalNotStated:
      case ErrorCode.portalUnknown:
        return {
          title: "Portal entitas belum dipilih",
          description:
            "Pilih portal entitas di bagian atas halaman, lalu simpan lagi.",
          tone: "penolakan",
        };
      case ErrorCode.portalNotReady:
        return {
          title: "Basis data entitas ini belum tersedia",
          description:
            "Mengulang tidak akan menolong. Hubungi administrator Claim PNC untuk melengkapi kredensial basis datanya.",
          tone: "gangguan",
        };
      default:
        return {
          title: "Terjadi kesalahan pada sistem",
          description: "Isian Anda belum tersimpan. Coba beberapa saat lagi.",
          tone: "gangguan",
        };
    }
  }
  return null;
}

/** Mengambil pelanggaran per isian dari galat validasi server. */
function violationsOf(error: unknown): Record<string, string> {
  return error instanceof APIError ? error.violations() : {};
}

/** Nama isian induk yang dikenali form ini; dipakai menyorot pelanggaran dari server. */
const FIELD_NAMES = [
  "nama_panel",
  "status_repair",
  "status_edit_quantity",
  "status_premium_repair",
  "status_pecah",
  "status_sticker",
  "status_sisi",
  "status_rusak_parah",
  "status_aktif",
  "exclusion_c",
] as const;

/**
 * Kesembilan penanda beserta caption layar Pega-nya.
 *
 * Kata-katanya dibaca dari `pyLabelFieldValue` pada
 * `Section/BrowsePanelHEApproval-Section.xml` apa adanya (`D-13`: teks layar lama diikuti
 * supaya pengguna tidak perlu belajar ulang) — hanya huruf besarnya yang diturunkan,
 * karena caption Pega seluruhnya kapital dan itu membuat form sulit dipindai mata.
 */
const MARKS = [
  ["status_repair", "Status repair"],
  ["status_edit_quantity", "Status edit quantity"],
  ["status_premium_repair", "Status premium repair"],
  ["status_pecah", "Status pecah"],
  ["status_sticker", "Status sticker"],
  ["status_sisi", "Status sisi"],
  ["status_rusak_parah", "Status rusak parah"],
  ["status_aktif", "Status aktif"],
  ["exclusion_c", "Exclusion C"],
] as const;

/**
 * PanelForm adalah form tambah dan ubah Master Panel.
 *
 * Kesepuluh isian induknya mengikuti caption
 * `Section/BrowsePanelHEApproval-Section.xml` apa adanya (`D-13`), ditambah daftar lokasi
 * yang di Pega digambar sebagai grid tersendiri di dalam form yang sama.
 *
 * # Tiga isian layar lama yang TIDAK ada di sini
 *
 * **ID PANEL.** Diterbitkan server dari kode situs dan sequence
 * (`PEGA_M_PANEL_HE.prc:21`), tidak pernah diketik. Pada mode ubah ia ditampilkan sebagai
 * keterangan.
 *
 * **MEMPERBAHARUI DATA.** Caption tombol pada layar lama, bukan isian.
 *
 * **Unggah lampiran.** Jalur `PNCSaveAttachmentToDB` tidak dibawa modul ini; kolom
 * DOKUMENID baris yang sudah ada dipertahankan apa adanya.
 */
export function PanelForm({
  editing,
  knownValues,
  isSaving,
  error,
  onSave,
  onCancel,
}: Props) {
  const isEditing = editing !== null;
  const headingId = useId();

  const options = usePanelOptions();

  const {
    register,
    handleSubmit,
    setError,
    control,
    formState: { errors },
  } = useForm<PanelFields>({
    resolver: zodResolver(schema),
    defaultValues: {
      nama_panel: editing?.nama_panel ?? "",
      status_repair: editing?.status_repair ?? "",
      status_edit_quantity: editing?.status_edit_quantity ?? "",
      status_premium_repair: editing?.status_premium_repair ?? "",
      status_pecah: editing?.status_pecah ?? "",
      status_sticker: editing?.status_sticker ?? "",
      status_sisi: editing?.status_sisi ?? "",
      status_rusak_parah: editing?.status_rusak_parah ?? "",
      status_aktif: editing?.status_aktif ?? "",
      exclusion_c: editing?.exclusion_c ?? "",
      lokasi: (editing?.lokasi ?? []).map((one) => ({
        lokasi_panel: one.lokasi_panel,
        sisi_panel: one.sisi_panel,
      })),
    },
  });

  const { fields, append, remove } = useFieldArray({ control, name: "lokasi" });

  // Pelanggaran yang dilaporkan server disorot pada isiannya masing-masing, bukan hanya
  // diringkas di satu kotak pesan. Server mengirim SELURUH pelanggaran sekaligus (P-5),
  // dan itu hanya berguna bila layar menyorotnya satu per satu.
  useEffect(() => {
    for (const [column, message] of Object.entries(violationsOf(error))) {
      if ((FIELD_NAMES as readonly string[]).includes(column)) {
        setError(column as (typeof FIELD_NAMES)[number], {
          type: "server",
          message,
        });
      }
    }
  }, [error, setError]);

  const violation = violationsOf(error);
  const message = messageFor(error);

  const locationOptions = options.data?.lokasi_panel ?? [];
  const sideOptions = options.data?.sisi_panel ?? [];

  function submit(values: PanelFields) {
    onSave({
      ...values,
      lokasi: values.lokasi.map((one) => ({
        lokasi_panel: one.lokasi_panel,
        sisi_panel: one.sisi_panel,
      })),
    });
  }

  return (
    <form
      onSubmit={handleSubmit(submit)}
      noValidate
      aria-labelledby={headingId}
      className="space-y-5 rounded-kartu border border-slate-200 bg-white p-5 shadow-lembut"
    >
      <h2 id={headingId} className="text-base font-semibold text-slate-900">
        {isEditing ? `Ubah ${editing.nama_panel}` : "Tambah Master Panel"}
      </h2>

      {message && (
        <ErrorMessage
          title={message.title}
          description={message.description}
          tone={message.tone}
        />
      )}

      {isEditing && (
        <div className="rounded-kontrol border border-slate-200 bg-slate-50 px-3 py-2.5">
          <span className="block text-xs font-medium uppercase tracking-wide text-slate-500">
            ID panel
          </span>
          <span className="mt-0.5 block text-sm text-slate-900">
            {editing.id_panel}
          </span>
          <span className="mt-1 block text-xs text-slate-500">
            Diterbitkan sistem dari kode situs dan nomor urut. Tidak dapat
            diubah.
          </span>
        </div>
      )}

      <Group title="Identitas panel">
        <Field
          id="nama_panel"
          label="Nama panel"
          type="text"
          maxLength={MAX.nama}
          hint="Tidak boleh sama dengan panel lain."
          error={errors.nama_panel?.message}
          disabled={isSaving}
          {...register("nama_panel")}
        />
      </Group>

      <Group title="Penanda perlakuan klaim">
        {/*
          Kesembilan penanda ini dirender pxDropdown atau pxRadioButtons di Pega, dan
          daftar pilihannya ada di rule Field Value yang TIDAK ikut di export (R-16).

          Karena itu tidak satu pun dibuat dropdown: mengarang dua pilihan "Ya/Tidak"
          berarti menebak domain sebuah kolom yang menentukan boleh tidaknya sebuah panel
          diperbaiki. Yang ditawarkan sebagai gantinya adalah NILAI YANG SUDAH DIPAKAI
          BARIS LAIN — saran yang berasal dari data, bukan dari tebakan.

          Kesembilannya tetap WAJIB, karena kesembilannya bertanda pyRequired=true di
          layar Pega.
        */}
        <p className="text-xs text-slate-500">
          Daftar pilihan kesembilan penanda ini tidak ada di export Pega. Saran
          yang muncul saat mengetik adalah nilai yang sudah dipakai panel lain
          pada entitas ini.
        </p>
        <div className="grid gap-4 sm:grid-cols-2">
          {MARKS.map(([name, label]) => (
            <div key={name}>
              <Field
                id={name}
                label={label}
                type="text"
                maxLength={MAX.kode}
                list={`panel-${name}`}
                error={errors[name]?.message}
                disabled={isSaving}
                {...register(name)}
              />
              <Suggestions name={name} values={knownValues[name] ?? []} />
            </div>
          ))}
        </div>
      </Group>

      <Group title="Lokasi panel">
        {options.isError && (
          <ErrorMessage
            title="Daftar pilihan lokasi tidak dapat dimuat"
            description="Baris lokasi tidak dapat ditambah sampai daftarnya termuat. Muat ulang halaman ini."
            tone="gangguan"
          />
        )}
        <LocationEditor
          rows={fields}
          register={register}
          errors={errors}
          locationOptions={locationOptions}
          sideOptions={sideOptions}
          violation={violation}
          isSaving={isSaving}
          onAdd={() =>
            append({ lokasi_panel: "", sisi_panel: PanelSide.tidakAda })
          }
          onRemove={remove}
        />
      </Group>

      <p className="text-xs text-slate-500">
        Baris yang disimpan selalu kembali ke{" "}
        <span className="font-medium">Waiting Approval</span> dan menunggu
        persetujuan — persetujuan sebelumnya tidak berlaku atas isi yang sudah
        berubah.
      </p>

      <div className="flex flex-wrap justify-end gap-2 pt-2">
        <Button tone="halus" onClick={onCancel} disabled={isSaving}>
          Batal
        </Button>
        <Button type="submit" tone="utama" disabled={isSaving}>
          {isSaving ? "Menyimpan…" : "Simpan"}
        </Button>
      </div>
    </form>
  );
}

/**
 * Group membungkus sekumpulan isian di bawah satu judul.
 *
 * Ia `<fieldset>` dan bukan `<div>` supaya pembaca layar mengumumkan judulnya saat kursor
 * masuk ke salah satu isian di dalamnya.
 */
function Group({
  title,
  children,
}: {
  title: string;
  children: React.ReactNode;
}) {
  return (
    <fieldset className="space-y-4 rounded-kontrol border border-slate-200 p-4">
      <legend className="px-1 text-sm font-semibold text-slate-700">
        {title}
      </legend>
      {children}
    </fieldset>
  );
}

/**
 * Suggestions menggambar daftar saran untuk sebuah isian.
 *
 * Isinya berasal dari nilai yang SUDAH DIPAKAI baris lain pada entitas yang sedang dibuka
 * — bukan dari daftar yang dikarang. Ia menjawab pertanyaan yang tidak dapat dijawab
 * export Pega ("nilai apa yang sah di kolom ini?") dengan satu-satunya sumber yang
 * tersedia: data itu sendiri.
 *
 * Tidak menggambar apa pun bila belum ada nilai yang terpakai, supaya daftar kosong tidak
 * muncul sebagai kotak saran yang selalu hampa.
 */
function Suggestions({ name, values }: { name: string; values: string[] }) {
  if (values.length === 0) return null;

  return (
    <datalist id={`panel-${name}`}>
      {values.map((value) => (
        <option key={value} value={value} />
      ))}
    </datalist>
  );
}
