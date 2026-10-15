import { useEffect, useState } from "react";

import { APIError } from "@/api/client";
import { Button } from "@/components/Button";
import { ErrorMessage } from "@/components/ErrorMessage";
import { DownloadIcon } from "@/components/Icon";

import { useExportDetailKlaim } from "./api";
import { ControlStrip, FieldLabel, inputClass } from "./parts";
import type {
  DashboardFilterInput,
  DetailExportInput,
  Filter,
  PeriodInput,
  Tab,
} from "./types";

/**
 * Bilah "Export Data Detail Klaim" di dalam kartu Outstanding.
 *
 * Padanan tombol `pyButtonLabel Export To Excel` di `Section/PNCDashboardOS`, yang menjalankan
 * `Activity/ExportDataDetailKlaim`. Berkasnya CSV 90 kolom dengan nama dan judul kolom yang
 * sama dengan Pega; sumbernya `T_CLAIMLIST_ADMIN` dan `T_CLAIM_PNC` (ketetapan Work Owner
 * 2026-10-14).
 *
 * Ia duduk TEPAT di atas tabel yang diekspornya, mengikuti rancangan layar 2026-10-14 — bukan
 * sebagai kotak tersendiri di atas seluruh tab.
 *
 * Keduanya kosong berarti seluruh periode, sama dengan Pega. Satu saja terisi ditolak server;
 * tombolnya dimatikan pada keadaan itu sebagai cermin aturan server.
 */
export function DetailExportStrip() {
  const [range, setRange] = useState<DetailExportInput>({
    dari: "",
    sampai: "",
  });
  const exportDetail = useExportDetailKlaim();

  const halfFilled = (range.dari === "") !== (range.sampai === "");

  return (
    <>
      <ControlStrip>
        <span className="self-center text-[10px] font-semibold tracking-wide text-slate-500 uppercase">
          Export Data Detail Klaim
        </span>
        <FieldLabel label="Dari">
          <input
            type="date"
            value={range.dari}
            onChange={(event) =>
              setRange({ ...range, dari: event.target.value })
            }
            className={inputClass}
          />
        </FieldLabel>
        <FieldLabel label="Sampai">
          <input
            type="date"
            value={range.sampai}
            onChange={(event) =>
              setRange({ ...range, sampai: event.target.value })
            }
            className={inputClass}
          />
        </FieldLabel>

        {/* Label tombolnya apa adanya dari Pega: `<pyLabel>Export To Excel</pyLabel>`. */}
        <Button
          type="button"
          onClick={() => exportDetail.mutate(range)}
          disabled={halfFilled || exportDetail.isPending}
        >
          <DownloadIcon className="h-4 w-4" />
          {exportDetail.isPending ? "Menyiapkan…" : "Export To Excel"}
        </Button>

        {/* Teks dari rancangan layar Work Owner 2026-10-14. */}
        <p className="max-w-xs self-center text-xs leading-relaxed text-slate-500">
          Kosongkan keduanya untuk seluruh periode — berkas sebesar itu perlu
          sekitar satu menit.
        </p>
      </ControlStrip>

      {exportDetail.isError ? (
        <div className="border-b border-slate-200 px-5 py-3">
          <ErrorMessage
            tone="gangguan"
            title="Berkas tidak dapat diunduh"
            description={messageOf(
              exportDetail.error,
              "Berkas tidak dapat diunduh.",
            )}
          />
        </div>
      ) : null}
    </>
  );
}

/**
 * Bilah penyaring Reinsurer dan Kategori OS di atas tabel silang.
 *
 * Pilihannya datang dari SERVER, termasuk "All": "Kategori OS" dibaca dari master tahapan
 * progres klaim yang dapat berubah tanpa deploy (`BrowseMstProgress1`).
 *
 * Tombolnya dua, dan labelnya dari Pega: `pyButtonLabel Filter` dan `pyButtonLabel Clear
 * Filter`. Dropdown TIDAK memuat ulang sendiri — tombol Filter yang memuatnya.
 */
export function DashboardFilterStrip({
  filters,
  draft,
  onDraftChange,
  onApply,
  onClear,
}: {
  filters: Filter[];
  draft: DashboardFilterInput;
  onDraftChange: (filter: DashboardFilterInput) => void;
  onApply: () => void;
  onClear: () => void;
}) {
  function valueOf(key: string): string {
    return key === "reinsurer" ? draft.reinsurer : draft.kategori_os;
  }

  function change(key: string, value: string) {
    onDraftChange(
      key === "reinsurer"
        ? { ...draft, reinsurer: value }
        : { ...draft, kategori_os: value },
    );
  }

  return (
    <ControlStrip>
      {filters.map((filter) => (
        <FieldLabel key={filter.kunci} label={filter.label}>
          <select
            value={valueOf(filter.kunci)}
            onChange={(event) => change(filter.kunci, event.target.value)}
            className={`min-w-48 ${inputClass}`}
          >
            {filter.pilihan.map((option) => (
              <option key={option.nilai || "semua"} value={option.nilai}>
                {option.label}
              </option>
            ))}
          </select>
        </FieldLabel>
      ))}

      <div className="flex items-center gap-2">
        <Button type="button" tone="utama" onClick={onApply}>
          Filter
        </Button>
        <Button type="button" onClick={onClear}>
          Clear Filter
        </Button>
      </div>
    </ControlStrip>
  );
}

/**
 * Bilah penyaring periode.
 *
 * Kedua bentuknya dibaca dari `Section/DisplayInboxProduktivitas_Sect`, yang punya
 * `<pyCaption Periode>`, `<pyCaption Bulan & Tahun>`, `<pyCaption Dari>`, dan
 * `<pyCaption Sampai>`. Tombolnya satu dan labelnya dari server — Pega memberi label berbeda
 * pada kedua tab ("Cari" dan "Lihat Data"). Tombol "Bersihkan" pada rancangan layar tidak
 * dibuat: Pega tidak punya, dan penyaring dikosongkan dengan mengosongkan isiannya lalu
 * menekan tombol yang sama.
 *
 * Isiannya DRAF dan baru berlaku saat tombolnya ditekan — tanpa itu setiap potongan isian
 * tanggal (tahun "0002", "0020", …) menjadi permintaan yang pasti ditolak.
 */
export function PeriodStrip({
  tab,
  value,
  onChange,
  note,
}: {
  tab: Tab;
  value: PeriodInput | null;
  onChange: (period: PeriodInput | null) => void;
  note?: string | undefined;
}) {
  const [draft, setDraft] = useState<PeriodInput>(
    value ?? { bentuk: "bulan", bulan: "", dari: "", sampai: "" },
  );

  // Draf disetel ulang ketika periode yang BERLAKU berubah dari luar — misalnya saat bagian
  // lain dibuka dan penyaringnya dibuang.
  useEffect(() => {
    setDraft(value ?? { bentuk: "bulan", bulan: "", dari: "", sampai: "" });
  }, [value]);

  const mode = draft.bentuk;

  function update(next: Partial<PeriodInput>) {
    setDraft({ ...draft, ...next });
  }

  /*
    Tombolnya mati HANYA pada rentang yang baru punya satu tanggal — satu-satunya keadaan
    yang pasti ditolak server. Isian KOSONG tetap dapat ditekan: artinya "tanpa penyaring
    periode", dan mematikannya membuat penyaring yang terpasang tidak dapat dihapus.
  */
  const complete =
    mode === "bulan"
      ? draft.bulan !== ""
      : draft.dari !== "" && draft.sampai !== "";
  const empty =
    mode === "bulan"
      ? draft.bulan === ""
      : draft.dari === "" && draft.sampai === "";

  return (
    <ControlStrip>
      <FieldLabel label="Periode">
        <select
          value={mode}
          onChange={(event) =>
            update({ bentuk: event.target.value as PeriodInput["bentuk"] })
          }
          className={inputClass}
        >
          <option value="bulan">Bulan &amp; Tahun</option>
          <option value="rentang">Rentang tanggal</option>
        </select>
      </FieldLabel>

      {mode === "bulan" ? (
        <FieldLabel label="Bulan & Tahun">
          <input
            type="month"
            value={draft.bulan}
            onChange={(event) => update({ bulan: event.target.value })}
            className={inputClass}
          />
        </FieldLabel>
      ) : (
        <>
          <FieldLabel label="Dari">
            <input
              type="date"
              value={draft.dari}
              onChange={(event) => update({ dari: event.target.value })}
              className={inputClass}
            />
          </FieldLabel>
          <FieldLabel label="Sampai">
            <input
              type="date"
              value={draft.sampai}
              onChange={(event) => update({ sampai: event.target.value })}
              className={inputClass}
            />
          </FieldLabel>
        </>
      )}

      <Button
        type="button"
        tone="utama"
        onClick={() => onChange(complete ? draft : null)}
        disabled={!complete && !empty}
      >
        {tab.label_terapkan_periode ?? "Cari"}
      </Button>

      {note ? (
        <p className="max-w-xs self-center text-xs leading-relaxed text-slate-500">
          {note}
        </p>
      ) : null}
    </ControlStrip>
  );
}

/**
 * messageOf membaca pesan galat yang layak dibaca pengguna.
 *
 * Pelanggaran per isian didahulukan di atas pesan amplopnya — amplopnya berbunyi "Permintaan
 * belum benar. Perbaiki yang ditandai lalu coba lagi." tanpa menyebut APA yang ditandai, dan
 * salah satu pelanggaran di layar ini disalin langsung dari Pega ("Periode Up To hanya untuk
 * periode tahun yang sama"). Seluruhnya digabung, meniru Pega yang menampilkan semua pesannya
 * bersamaan (`P-5`).
 */
export function messageOf(error: unknown, fallback: string): string {
  if (error instanceof APIError) {
    const violations = Object.values(error.violations()).filter(
      (text) => text !== "",
    );
    if (violations.length > 0) return violations.join(" ");
    if (error.message) return error.message;
  }
  if (error instanceof Error && error.message) return error.message;
  return fallback;
}
