# B-8 · Choose Surveyor & Hasil Survei

| | |
|---|---|
| **Nama di sistem lama** | **Choose Surveyor** / **InputSurveyor** — tahap pada `Flow/Register_Flow.xml`; inbox `InboxSurvey_Harness`, `SurveyorsInbox`, `DetailSurveyorsInbox`, `ViewDetailHasilSurveyorInternal1`; master `MasterLoginSurvey` |
| **Kode modul** | `B-8` |
| **Gelombang** | 4 — Persetujuan |
| **Ukuran** | 30 activity |
| **Bergantung pada** | `B-6` Penugasan |
| **Kesiapan** | **SEBAGIAN** |

## Apa yang dikerjakan modul ini

Menugaskan **surveyor atau loss adjuster** untuk meninjau kerugian di lapangan, lalu mencatat hasil
survei beserta foto dokumentasinya. Surveyor dapat internal (ASM) maupun eksternal.

Modul ini juga menyimpan komponen **KPI adjuster** — dan di sanalah kendala terbesarnya.

## Rule Pega yang digantikan

| Jenis | Nama |
|---|---|
| Flow | tahap `Choose Surveyor` / `InputSurveyor` pada `Flow/Register_Flow.xml` |
| Harness | `InboxSurvey_Harness` · `SurveyorsInbox` · `DetailSurveyorsInbox` · `ViewDetailHasilSurveyorInternal1` |
| Master | `MasterLoginSurvey` · `INSERT_SURVEYORLIST` |
| Objek DB | `INSERT_KPIADJUSTER` |

## Penghalang modul ini

| Jenis | Isi | Pemilik |
|---|---|---|
| **Artefak** | When rule `IsSurvey` — hilang dari export | **Tim Pega** (`R-16`) |
| **Keputusan** | **Rumus skor KPI adjuster tidak ada di mana pun.** `INSERT_KPIADJUSTER` hanya menyimpan **10 komponen tanpa agregasi** — tidak ada rule yang menghitung skornya | **Work Owner** |
| **Keputusan** | Kunci upsert hanya `CASEID` — **adjuster kedua menimpa yang pertama**. Benar? | **Work Owner** |

## Daftar tiket

| Tiket | Judul | Status |
|---|---|---|
| [TKT-B08-001](issues/01-penugasan-surveyor-dan-hasil-survei.md) | Penugasan surveyor dan pencatatan hasil survei | `needs-info` |
| [TKT-B08-002](issues/02-kpi-adjuster.md) | Komponen dan skor KPI adjuster | `needs-info` |
