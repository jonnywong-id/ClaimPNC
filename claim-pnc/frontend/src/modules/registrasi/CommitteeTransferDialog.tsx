import { useEffect, useState } from "react";

import { Button } from "@/components/Button";
import { ErrorMessage } from "@/components/ErrorMessage";
import { formatDateTimeWIB } from "@/components/format";
import { SelectField } from "@/components/SelectField";
import { TextAreaField } from "@/components/TextAreaField";

import {
  useSaveCommitteeNote,
  useTransferCommittee,
  useTransferToAnalyst,
  violationsFrom,
} from "./api";
import { DiagnosisSearch } from "./DiagnosisSearch";
import { isPHKCoverage } from "./TransferToAnalyst";
import type { Claim, CommitteeNote, InsuredItem, Receiver } from "./types";

const PANEL_PA = "002";
const PANEL_TRAVEL = "005";

/** Tahap Estimation PA — satu-satunya tahap tempat Kirim Analyst (setTicketToAnalyst) berlaku. */
const STAGE_ESTIMATE_PA = "estimasi-pa";

/** StatusClaim 1151 Analyst — When IsStatusClaimAnalystPA. */
const STATUS_ANALYST = "1151";

const emptyNote: CommitteeNote = {
  kronologi_kejadian: "",
  jumlah_kerugian: "",
  polis_liability: "",
  remarks: "",
  remarks_investigasi: "",
  diagnosa: "",
  kode_diagnosa: "",
  desc_diagnosa: "",
  penerima_klaim: "",
};

function failureText(failure: unknown): string {
  const violations = violationsFrom(failure);
  if (violations.length > 0) return violations.map((v) => v.pesan).join(" ");
  if (failure instanceof Error) return failure.message;
  return "Terjadi kesalahan pada sistem.";
}

/**
 * Modal "Transfer Claim ke Komite" — local action `ClaimComitee_OC` (`Flow Action/ClaimComitee_OC`,
 * section `ClaimComitee_OC`). Dibuka dua tombol Pega:
 *
 * - "Transfer ke Komite" pada baris grid Adjustment (`Section/ShowAdjustment`, sel 28) —
 *   `adjustment` terisi; Kirim Komite mentransfer baris itu (`ValidationTypePaymentAdj`).
 * - "Transfer ke Analyst" pada jaminan PA (`Section/TrfKomiteButton`) — tanpa `adjustment`.
 *
 * Isian dan syarat tampilnya mengikuti section apa adanya:
 *
 *   Tanggal & Waktu (.TanggalComitee)    selalu, nonaktif
 *   Inisial (.Initial)                   selalu, baca saja
 *   Penerima Klaim (.TempReceiver)       IsTravel, wajib
 *   Kronologi Kejadian                   !IsTravel
 *   Jumlah Kerugian                      BusinessType != 'PA'
 *   Polis Liability                      !IsTravelPA
 *   Remaks / Catatan Analyst (.Remarks)  !IsPA
 *   Remaks / Investigation               IsPA
 *   Kode / Desc Diagnose, Diagnose       IsPA
 *   Remaks / Analysis (.Remarks)         IsPA
 *
 * Tombol: Simpan (PNCSaveButton2) · Aksep (IsAnalisator; SetListComiteeClaimPerObj) · Kirim
 * Analyst (`.IsAnalisTransfer != '1' && !IsPHK && PNCStatus != '5'`; setTicketToAnalyst) · Kirim
 * Komite (`!IsPA || IsPHK`) · Batal. Isian disimpan lebih dulu, lalu tombolnya dijalankan — Pega
 * mengirim keduanya dalam satu submit.
 *
 * Aksep dan Kirim Komite mentransfer satu baris adjustment ke komite: baris yang tombolnya ditekan
 * bila modal dibuka dari grid Adjustment, selain itu baris TERAKHIR jaminan
 * (`SetListComiteeClaimPerObj`: `.AdjustmentList(<LAST>)`). Keduanya memakai penjenjangan komite
 * yang sama (SetEmailKomite*); cabang khusus PA di activity itu (data AI, TKI di bawah Rp 5 jt
 * langsung aksep, kembali ke Estimator PA) belum dibawa.
 *
 * Kirim Komite (`ValidationTypePayment`) untuk Travel mengirim Penerima Klaim; server menolak
 * penerima yang belum dipilih atau data banknya belum lengkap. Modal Kirim Analyst hanya
 * ditutup bila StatusClaim menjadi 1151 (When IsStatusClaimAnalystPA).
 *
 * Isi otomatis PA (PreClaimComitee_OC "Auto Isi Kronologi Kejadian, Jumlah Kerugian &
 * Remark/Analisis", hanya bila jaminan belum pernah disimpan) — lihat prefillPA.
 *
 * Detail Diagnosa (PA): Cari / Pilih kode diagnosa — lihat DiagnosisSearch.
 *
 * Belum dibangun: Kirim Investigator (ValidationTypePayment_PA bergantung `PNCStatus` — Property
 * PNCStatus: 1 Menunggu Persetujuan · 2 Ditolak Komite · 3 Sedang Proses · 4 Batal Survey ·
 * 5 Selesai — yang tidak punya kolom di T_CLAIM_PNC), peringatan dokumen belum lengkap PA
 * (RequiredDocPA), dan isian `.Salvage` (tidak ada kolomnya).
 * Kirim Analyst hanya ditawarkan di tahap Estimation PA, tempat server menjalankannya.
 */
export function CommitteeTransferDialog({
  klaim,
  taskID,
  stage,
  object,
  coverage,
  adjustment,
  receivers,
  analyst = false,
  onClose,
}: {
  klaim: Claim;
  taskID: string;
  stage: string;
  /** Objek dan jaminan berbasis 1. */
  object: number;
  coverage: number;
  /** Baris adjustment berbasis 1 bila dibuka dari grid Adjustment. */
  adjustment?: number | undefined;
  receivers: Receiver[];
  /** Pemanggil anggota grup Analyst (IsAnalisator) — tombol Aksep. */
  analyst?: boolean;
  onClose: () => void;
}) {
  const item = klaim.objek[object - 1];
  const cov = item?.coverage[coverage - 1];
  const [note, setNote] = useState<CommitteeNote>(() =>
    prefillPA(klaim, object, coverage, {
      ...emptyNote,
      ...(cov?.isian_komite ?? {}),
    }),
  );
  const [saved, setSaved] = useState(false);
  const [marked, setMarked] = useState(false);

  const save = useSaveCommitteeNote(klaim.id);
  const toCommittee = useTransferCommittee(klaim.id);
  const toAnalyst = useTransferToAnalyst(klaim.id);
  const busy = save.isPending || toCommittee.isPending || toAnalyst.isPending;

  useEffect(() => {
    function onKey(event: KeyboardEvent) {
      if (event.key === "Escape" && !busy) onClose();
    }
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose, busy]);

  if (!item || !cov) return null;

  const panel = klaim.polis.lini;
  const isPA = panel === PANEL_PA;
  const isTravel = panel === PANEL_TRAVEL;
  const isTravelPA = isPA || isTravel;
  const isPHK = isPHKCoverage(cov.id);
  const businessPA = klaim.polis.jenis_bisnis === "PA";

  // Baris yang ditransfer Aksep/Kirim Komite: baris tombolnya, atau baris terakhir jaminan.
  const lines = cov.adjustment ?? [];
  const target = adjustment ?? lines.length;
  const targetLine = lines[target - 1];
  const transferable =
    targetLine !== undefined &&
    !targetLine.komite_id &&
    targetLine.status_akseptasi === "";

  const showAnalyst =
    !cov.sudah_transfer_analis && !isPHK && isPA && stage === STAGE_ESTIMATE_PA;
  const showCommittee = (!isPA || isPHK) && transferable;
  const showAccept = analyst && transferable;
  const receiverMissing = isTravel && note.penerima_klaim === "";
  // setValidasiReceiverClaim_act (saat Penerima Klaim dipilih): Travel dengan Total Klaim > 0 dan
  // penerima yang data banknya belum lengkap menonaktifkan Aksep/Kirim Komite
  // (`TempErrorTotalKlaim.NIK = 1`). Cabang bank (BranchOfBank) tidak punya kolom di
  // T_CLAIM_RECEIVER, sehingga yang diperiksa nama bank dan nomor rekening.
  const receiver = receivers.find((r) => r.id === note.penerima_klaim);
  const receiverIncomplete =
    isTravel &&
    !receiverMissing &&
    (targetLine?.nilai_propose_sen ?? 0) > 0 &&
    (!receiver ||
      receiver.nama_bank.trim() === "" ||
      receiver.nomor_rekening.trim() === "");
  // ValidationProposeValue (PreClaimComitee_OC step NOCEK, hanya PA dengan jaminan PHK), atas
  // adjustment terakhir jaminan: pesan Total Klaim kosong dan CFS belum diunduh, dan Aksep/Kirim
  // Komite nonaktif bila Total Klaim dan Nilai Pengajuan sama-sama 0 (`TempErrorTotalKlaim.NIK`).
  const lastLine = lines.at(-1);
  const proposeWarnings: string[] = [];
  let proposeBlocked = false;
  if (isPA && isPHK && lastLine) {
    if (lastLine.nilai_propose_sen === 0)
      proposeWarnings.push("Nilai TOTAL KLAIM klaim tidak boleh kosong...!");
    if (!coverageHasCFS(cov))
      proposeWarnings.push("Download CFS sebelum TOTAL KLAIM di ISI...!");
    proposeBlocked =
      lastLine.nilai_propose_sen <= 0 && lastLine.nilai_pengajuan_sen <= 0;
  }
  const transferBlocked =
    receiverMissing || receiverIncomplete || proposeBlocked;

  const failure = save.error ?? toCommittee.error ?? toAnalyst.error;
  const set = (field: keyof CommitteeNote) => (value: string) => {
    setSaved(false);
    setNote((n) => ({ ...n, [field]: value }));
  };

  function reset() {
    save.reset();
    toCommittee.reset();
    toAnalyst.reset();
  }

  function saveNote(then?: () => void) {
    reset();
    save.mutate(
      { tugas_id: taskID, objek: object, jaminan: coverage, ...note },
      {
        onSuccess: () => {
          setSaved(true);
          then?.();
        },
      },
    );
  }

  function transfer() {
    saveNote(() =>
      toCommittee.mutate(
        {
          tugas_id: taskID,
          objek: object,
          jaminan: coverage,
          adjustment: target,
          ...(isTravel ? { penerima_klaim: note.penerima_klaim } : {}),
        },
        { onSuccess: onClose },
      ),
    );
  }

  const field = "mt-3";
  const area = (id: keyof CommitteeNote, label: string, rows = 3) => (
    <div className={field}>
      <TextAreaField
        id={`komite-${id}`}
        label={label}
        rows={rows}
        value={note[id] ?? ""}
        disabled={busy}
        onChange={(e) => set(id)(e.target.value)}
      />
    </div>
  );

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-transfer-komite"
    >
      <div className="max-h-full w-full max-w-3xl overflow-y-auto rounded-kartu bg-white p-6 shadow-angkat">
        <h2
          id="judul-transfer-komite"
          className="text-lg font-semibold text-slate-900"
        >
          Transfer Claim ke Komite
        </h2>
        <p className="mt-1 text-sm text-slate-600">
          {item.nama || item.id} · {cov.nama || cov.id}
          {adjustment !== undefined && ` · Adjustment ${adjustment}`}
        </p>

        {proposeWarnings.length > 0 && (
          <div className="mt-4">
            <ErrorMessage
              tone="penolakan"
              title="Periksa adjustment"
              description={proposeWarnings.join(" ")}
            />
          </div>
        )}
        {failure && (
          <div className="mt-4">
            <ErrorMessage
              tone="penolakan"
              title="Belum berhasil"
              description={failureText(failure)}
            />
          </div>
        )}
        {saved && !failure && !marked && (
          <p className="mt-3 text-sm text-green-700">Isian tersimpan.</p>
        )}
        {marked && !failure && (
          <p className="mt-3 text-sm text-green-700">
            Jaminan ini ditandai Transfer ke Analyst.
          </p>
        )}

        <div className="mt-4 grid gap-3 sm:grid-cols-3">
          <div>
            <span className="block text-xs font-medium text-slate-700">
              Tanggal &amp; Waktu
            </span>
            <span className="mt-1 block text-sm text-slate-900">
              {formatDateTimeWIB(note.tanggal_komite) || "—"}
            </span>
          </div>
          <div>
            <span className="block text-xs font-medium text-slate-700">
              Inisial
            </span>
            <span className="mt-1 block text-sm text-slate-900">
              {note.inisial || "—"}
            </span>
          </div>
          {isTravel && (
            <SelectField
              id="komite-penerima"
              label="Penerima Klaim"
              required
              value={note.penerima_klaim}
              disabled={busy}
              options={receivers.map((r) => ({ value: r.id, label: r.nama }))}
              error={
                receiverMissing
                  ? "Penerima Klaim wajib dipilih."
                  : receiverIncomplete
                    ? "Receiver Claim harus di isi"
                    : undefined
              }
              onChange={(e) => set("penerima_klaim")(e.target.value)}
            />
          )}
        </div>

        <h3 className="mt-5 text-sm font-semibold text-slate-800">
          Adjustment
        </h3>
        {!isTravel && area("kronologi_kejadian", "Kronologi Kejadian")}
        {!businessPA && area("jumlah_kerugian", "Jumlah Kerugian")}
        {!isTravelPA && area("polis_liability", "Polis Liability")}
        {!isPA && area("remarks", "Remaks / Catatan Analyst")}
        {isPA && area("remarks_investigasi", "Remaks / Investigation")}
        {isPA && (
          <DiagnosisSearch
            disabled={busy}
            onPick={(d) => {
              setSaved(false);
              setNote((n) => ({
                ...n,
                kode_diagnosa: d.kode,
                desc_diagnosa: d.deskripsi,
              }));
            }}
          />
        )}
        {isPA && (
          <div className="mt-3 grid gap-3 sm:grid-cols-2">
            <TextAreaField
              id="komite-kode_diagnosa"
              label="Kode Diagnose"
              rows={1}
              value={note.kode_diagnosa}
              disabled={busy}
              onChange={(e) => set("kode_diagnosa")(e.target.value)}
            />
            <TextAreaField
              id="komite-desc_diagnosa"
              label="Desc Diagnose"
              rows={1}
              value={note.desc_diagnosa}
              disabled={busy}
              onChange={(e) => set("desc_diagnosa")(e.target.value)}
            />
          </div>
        )}
        {isPA && area("diagnosa", "Diagnose/History of Illness")}
        {isPA && area("remarks", "Remaks / Analysis")}

        {/* Selain Travel dan PA cukup Batal (kiri) dan Kirim Komite (kanan) — Work Owner 2026-10-08,
            mengikuti tata letak section ClaimComitee_OC (layout 2.2 dan 2.1). */}
        {!isTravelPA ? (
          <div className="mt-6 flex flex-wrap justify-between gap-3">
            <Button tone="halus" disabled={busy} onClick={onClose}>
              Batal
            </Button>
            {showCommittee && (
              <Button
                tone="utama"
                disabled={busy || transferBlocked}
                onClick={() => transfer()}
              >
                {toCommittee.isPending ? "Mengirim…" : "Kirim Komite"}
              </Button>
            )}
          </div>
        ) : (
          <div className="mt-6 flex flex-wrap justify-end gap-3">
            <Button tone="halus" disabled={busy} onClick={onClose}>
              Batal
            </Button>
            <Button tone="halus" disabled={busy} onClick={() => saveNote()}>
              {save.isPending && !toCommittee.isPending && !toAnalyst.isPending
                ? "Menyimpan…"
                : "Simpan"}
            </Button>
            {showAnalyst && (
              <Button
                tone="utama"
                disabled={busy}
                onClick={() =>
                  saveNote(() =>
                    toAnalyst.mutate(
                      { taskID, objekID: item.id, coverageID: cov.id },
                      {
                        // Aksi tutup modal ber-When IsStatusClaimAnalystPA (PA dan StatusClaim 1151):
                        // jaminan selain yang terakhir hanya ditandai, dan modal tetap terbuka.
                        onSuccess: (r) =>
                          r.klaim.status_klaim === STATUS_ANALYST
                            ? onClose()
                            : setMarked(true),
                      },
                    ),
                  )
                }
              >
                {toAnalyst.isPending ? "Mengirim…" : "Kirim Analyst"}
              </Button>
            )}
            {showAccept && (
              <Button
                tone="utama"
                disabled={busy || transferBlocked}
                onClick={() => transfer()}
              >
                {toCommittee.isPending ? "Mengirim…" : "Aksep"}
              </Button>
            )}
            {showCommittee && (
              <Button
                tone="utama"
                disabled={busy || transferBlocked}
                onClick={() => transfer()}
              >
                {toCommittee.isPending ? "Mengirim…" : "Kirim Komite"}
              </Button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

const rupiah = new Intl.NumberFormat("id-ID", { maximumFractionDigits: 0 });

/**
 * Isi otomatis PA — PreClaimComitee_OC, langkah "Auto Isi Kronologi Kejadian, Jumlah Kerugian &
 * Remark/Analisis // PA (kalo blm save/flag kosong)". Baris acuannya adjustment TERAKHIR jaminan.
 *
 *   .ExtentOfLoss = "Tertanggung atas nama <objek> mendapatkan santunan kecelakaan sebesar
 *                    Rp <Total Klaim, atau Nilai Pengajuan bila Total Klaim 0>,-"
 *   .Remarks      = bila klaim sudah ditransfer ke Analyst dan jaminan sudah CFS:
 *                   "Klaim diusulkan dibayar sebesar Rp <Total Klaim>,- sesuai dengan santunan
 *                    <nama jaminan> (<penyebab kerugian>)."
 *
 * Penanda "sudah disimpan" Pega (`.pyLabel`, diisi PNCSaveButton2) tidak punya kolom; yang dipakai
 * di sini: isiannya masih kosong. Pemisah ribuan mengikuti format rupiah layar (titik).
 */
export function prefillPA(
  klaim: Claim,
  object: number,
  coverage: number,
  note: CommitteeNote,
): CommitteeNote {
  if (klaim.polis.lini !== PANEL_PA) return note;
  const item = klaim.objek[object - 1];
  const cov = item?.coverage[coverage - 1];
  const last = cov?.adjustment?.at(-1);
  if (!item || !cov || !last) return note;
  const total = rupiah.format(last.nilai_propose_sen / 100);
  const submitted = rupiah.format(last.nilai_pengajuan_sen / 100);
  const out = { ...note };
  if (out.jumlah_kerugian === "") {
    const value = last.nilai_propose_sen === 0 ? submitted : total;
    out.jumlah_kerugian = `Tertanggung atas nama ${item.nama} mendapatkan santunan kecelakaan sebesar Rp ${value},-`;
  }
  if (
    out.remarks === "" &&
    klaim.sudah_transfer_analis === true &&
    coverageHasCFS(cov)
  ) {
    out.remarks = `Klaim diusulkan dibayar sebesar Rp ${total},- sesuai dengan santunan ${cov.nama} (${cov.penyebab_kerugian}).`;
  }
  return out;
}

/** Jaminan sudah dibuatkan Claim Face Sheet (`.IsCFS == "1"`): ada estimasi yang sudah CFS. */
function coverageHasCFS(cov: InsuredItem["coverage"][number]): boolean {
  return (cov.item ?? []).some((i) =>
    i.estimasi.some((e) => e.sudah_cfs === true),
  );
}
