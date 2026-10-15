import { useMemo, useState } from "react";

import { Button } from "@/components/Button";
import { ErrorMessage } from "@/components/ErrorMessage";

import {
  useInputProgress,
  useProgressInputForm,
  useProgressNeedOptions,
  violationsFrom,
} from "./api";
import type { ProgressInputForm, ProgressOption } from "./types";

/**
 * Modal Input Progress Claim — flow action `inputprogress12`, section `InputProgressClaim1`.
 *
 * Di Pega tombolnya ada pada tiap baris posisi yang belum Done; di sini posisinya dipilih di
 * dalam modal karena grid tab ini menampilkan progres secara datar.
 */

type NeedRow = { key: number; text: string; id: string };

const STATUS_PROMPT = "MOHON ISI STATUS TERLEBIH DAHULU";
const NEED_NOT_FOUND = "STATUS PROGRESS 2 TIDAK DITEMUKAN";
const FOLLOW_UP_PAST =
  "Tanggal Next Follow Up tidak boleh kurang dari hari ini";
const REMARKS_MAX = 2000;

function todayISO(): string {
  const d = new Date();
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

function failureText(err: unknown): string {
  const v = violationsFrom(err);
  if (v.length > 0) return v.map((x) => x.pesan).join(" · ");
  return err instanceof Error ? err.message : "Terjadi kesalahan pada sistem.";
}

export function ProgressInputDialog({
  claimID,
  onClose,
}: {
  claimID: string;
  onClose: () => void;
}) {
  const form = useProgressInputForm(claimID);
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-900/40 px-4 py-8"
      role="dialog"
      aria-modal="true"
      aria-labelledby="judul-input-progress"
    >
      <div className="max-h-full w-full max-w-3xl overflow-y-auto rounded-kartu bg-white p-6 shadow-angkat">
        <h2
          id="judul-input-progress"
          className="text-lg font-semibold text-slate-900"
        >
          Input Progress Claim
        </h2>
        {form.isPending && (
          <p className="mt-3 text-sm text-slate-600">Memuat…</p>
        )}
        {form.error && (
          <div className="mt-3">
            <ErrorMessage
              title="Isian progres tidak dapat dimuat"
              description={failureText(form.error)}
              tone="gangguan"
            />
          </div>
        )}
        {form.data && (
          <ProgressInputBody
            claimID={claimID}
            form={form.data}
            onClose={onClose}
          />
        )}
        {!form.data && (
          <div className="mt-4 flex justify-end">
            <Button tone="halus" onClick={onClose}>
              Tutup
            </Button>
          </div>
        )}
      </div>
    </div>
  );
}

function ProgressInputBody({
  claimID,
  form,
  onClose,
}: {
  claimID: string;
  form: ProgressInputForm;
  onClose: () => void;
}) {
  const send = useInputProgress(claimID);
  const open = form.posisi.filter((p) => p.terbuka);
  const [position, setPosition] = useState(open.at(-1)?.id ?? "");
  const [status1, setStatus1] = useState("");
  const [docType, setDocType] = useState("");
  const [email, setEmail] = useState(form.email_progress);
  const [rows, setRows] = useState<NeedRow[]>([{ key: 1, text: "", id: "" }]);
  const [followUp, setFollowUp] = useState("");
  const [status, setStatus] = useState("");
  const [remarks, setRemarks] = useState("");
  const [warnings, setWarnings] = useState<string[]>([]);
  const [done, setDone] = useState(false);

  const docStatus = status1 === form.status_tipe_dokumen;
  const needs = useProgressNeedOptions(status1, docStatus ? docType : "");
  const options = useMemo<ProgressOption[]>(
    () => needs.data ?? [],
    [needs.data],
  );
  const emailRequired = form.status_wajib_email.includes(status1);
  const followUpPast = followUp !== "" && followUp <= todayISO();

  function changeStatus1(v: string) {
    setStatus1(v);
    setDocType("");
    setRows([{ key: Date.now(), text: "", id: "" }]);
  }

  function changeRow(key: number, text: string) {
    const match = options.find((o) => o.nama === text);
    setRows((all) =>
      all.map((r) => (r.key === key ? { ...r, text, id: match?.id ?? "" } : r)),
    );
  }

  function notFound(r: NeedRow): boolean {
    const t = r.text.trim().toUpperCase();
    return (
      t !== "" &&
      r.id === "" &&
      !options.some((o) => o.nama.toUpperCase().includes(t))
    );
  }

  function submit() {
    send.mutate(
      {
        posisi_id: position,
        status_progress_1: status1,
        status_perlu: rows.map((r) => r.id),
        status: status === STATUS_PROMPT ? "" : status,
        remarks,
        tgl_next_follow_up: followUp,
        tipe_dokumen: docStatus ? docType : "",
        email_progress: email,
      },
      {
        onSuccess: (res) => {
          setWarnings(res.peringatan);
          setDone(true);
          if (res.peringatan.length === 0) onClose();
        },
      },
    );
  }

  const input =
    "mt-1 block w-full rounded border border-slate-300 px-2 py-1 text-sm";
  const listID = `status-perlu-${claimID}`;
  return (
    <>
      {open.length === 0 && (
        <div className="mt-3">
          <ErrorMessage
            title="Tidak ada posisi progres yang terbuka"
            description="Seluruh posisi progres klaim ini sudah Done."
            tone="penolakan"
          />
        </div>
      )}
      <div className="mt-4 grid gap-3 md:grid-cols-2">
        <label className="block text-sm">
          <span className="text-xs text-slate-600">Posisi</span>
          <select
            aria-label="Posisi"
            value={position}
            onChange={(e) => setPosition(e.target.value)}
            className={input}
          >
            {open.map((p) => (
              <option key={p.id} value={p.id}>
                {p.posisi} ({p.status})
              </option>
            ))}
          </select>
        </label>
        <label className="block text-sm">
          <span className="text-xs text-slate-600">Tanggal Input</span>
          <input
            aria-label="Tanggal Input"
            type="date"
            value={todayISO()}
            disabled
            className={`${input} bg-slate-100`}
          />
        </label>
        <label className="block text-sm">
          <span className="text-xs text-slate-600">Status Progress 1 *</span>
          <select
            aria-label="Status Progress 1"
            value={status1}
            onChange={(e) => changeStatus1(e.target.value)}
            className={input}
          >
            <option value="">--Pilih--</option>
            {[...form.status_progress_1]
              .sort((a, b) => a.nama.localeCompare(b.nama))
              .map((o) => (
                <option key={o.id} value={o.id}>
                  {o.nama}
                </option>
              ))}
          </select>
        </label>
        {docStatus && (
          <label className="block text-sm">
            <span className="text-xs text-slate-600">Tipe Dokumen</span>
            <select
              aria-label="Tipe Dokumen"
              value={docType}
              onChange={(e) => setDocType(e.target.value)}
              className={input}
            >
              <option value="">--Pilih--</option>
              {form.tipe_dokumen.map((t) => (
                <option key={t} value={t}>
                  {t}
                </option>
              ))}
            </select>
          </label>
        )}
      </div>
      {emailRequired && (
        <label className="mt-3 block text-sm">
          <span className="text-xs text-slate-600">Email Progress Klaim *</span>
          <textarea
            aria-label="Email Progress Klaim"
            rows={2}
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className={input}
          />
        </label>
      )}

      <h3 className="mt-4 text-sm font-semibold text-slate-800">
        status progress 2
      </h3>
      <datalist id={listID}>
        {options.map((o) => (
          <option key={o.id} value={o.nama} />
        ))}
      </datalist>
      <table className="mt-1 w-full border-collapse text-sm">
        <thead>
          <tr className="bg-slate-100 text-left text-xs text-slate-700">
            <th className="p-2">STATUS PERLU *</th>
            <th className="w-16 p-2" />
          </tr>
        </thead>
        <tbody>
          {rows.map((r, i) => (
            <tr key={r.key} className="border-b border-slate-100 align-top">
              <td className="p-2">
                <input
                  aria-label={`STATUS PERLU ${i + 1}`}
                  list={listID}
                  value={r.text}
                  disabled={status1 === ""}
                  placeholder={
                    status1 === ""
                      ? "Pilih Status Progress 1 dulu"
                      : "Ketikan kata kunci, pilih"
                  }
                  onChange={(e) => changeRow(r.key, e.target.value)}
                  className="block w-full rounded border border-slate-300 px-2 py-1 text-sm"
                />
                {notFound(r) && (
                  <p className="mt-1 text-xs text-red-700">{NEED_NOT_FOUND}</p>
                )}
              </td>
              <td className="p-2">
                <button
                  type="button"
                  className="text-xs text-red-700"
                  onClick={() => {
                    if (window.confirm("Hapus baris ini?"))
                      setRows((all) => all.filter((x) => x.key !== r.key));
                  }}
                >
                  Hapus
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <button
        type="button"
        className="mt-1 text-xs text-sky-700"
        onClick={() =>
          setRows((all) => [...all, { key: Date.now(), text: "", id: "" }])
        }
      >
        Tambah
      </button>

      <div className="mt-4 grid gap-3 md:grid-cols-2">
        <label className="block text-sm">
          <span className="text-xs text-slate-600">Tgl Next Follow Up</span>
          <input
            aria-label="Tgl Next Follow Up"
            type="date"
            value={followUp}
            onChange={(e) => setFollowUp(e.target.value)}
            className={input}
          />
          {followUpPast && (
            <p className="mt-1 text-xs text-red-700">{FOLLOW_UP_PAST}</p>
          )}
        </label>
        <label className="block text-sm">
          <span className="text-xs text-slate-600">Status *</span>
          <select
            aria-label="Status"
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            className={input}
          >
            <option value="">{STATUS_PROMPT}</option>
            {form.status.map((s) => (
              <option key={s} value={s}>
                {s}
              </option>
            ))}
          </select>
        </label>
      </div>
      <label className="mt-3 block text-sm">
        <span className="text-xs text-slate-600">Remarks *</span>
        <textarea
          aria-label="Remarks"
          rows={3}
          maxLength={REMARKS_MAX}
          value={remarks}
          onChange={(e) => setRemarks(e.target.value)}
          className={input}
        />
      </label>

      {send.error && (
        <div className="mt-3">
          <ErrorMessage
            title="Progress tidak dapat disimpan"
            description={failureText(send.error)}
            tone="penolakan"
          />
        </div>
      )}
      {done && warnings.length > 0 && (
        <div className="mt-3">
          <ErrorMessage
            tone="gangguan"
            title="Progress tersimpan, dengan catatan"
            description={warnings.join(" | ")}
          />
        </div>
      )}
      <div className="mt-4 flex justify-end gap-2">
        <Button tone="halus" disabled={send.isPending} onClick={onClose}>
          {done ? "Tutup" : "Batal"}
        </Button>
        {!done && (
          <Button
            tone="utama"
            disabled={send.isPending || open.length === 0}
            onClick={submit}
          >
            {send.isPending ? "Menyimpan…" : "Submit"}
          </Button>
        )}
      </div>
    </>
  );
}
