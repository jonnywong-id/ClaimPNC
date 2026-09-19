import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { useLocation, useNavigate, type Location } from 'react-router-dom'
import { z } from 'zod'

import { APIError, NetworkError } from '@/api/client'
import { ErrorCode } from '@/api/types'
import { FormField } from '@/components/FormField'
import { ErrorMessage, type ErrorTone } from '@/components/ErrorMessage'

import { useLogin } from './api'

const schema = z.object({
  username: z.string().trim().min(1, 'Nama pengguna wajib diisi.'),
  password: z.string().min(1, 'Kata sandi wajib diisi.'),
})

type FormValues = z.infer<typeof schema>

type MessageBody = {
  title: string
  note: string
  tone: ErrorTone
}

/**
 * Mengubah galat menjadi pesan yang dapat ditindaklanjuti.
 *
 * Ketiga jenis galat masuk sengaja diberi pesan BERBEDA, karena tindak lanjutnya
 * berbeda: mengetik ulang, menghubungi administrator, atau menunggu. Yang TIDAK
 * dibedakan adalah "pengguna tidak ada" dan "kata sandi salah" — keduanya memakai satu
 * pesan yang sama supaya keberadaan akun tidak bocor.
 */
function messageFor(failure: unknown): MessageBody {
  if (failure instanceof NetworkError) {
    return {
      title: 'Server Claim PNC tidak dapat dihubungi',
      note: 'Periksa koneksi jaringan Anda, lalu coba lagi.',
      tone: 'gangguan',
    }
  }
  if (failure instanceof APIError) {
    switch (failure.kode) {
      case ErrorCode.wrongCredential:
        return {
          title: 'Nama pengguna atau kata sandi salah',
          note: 'Periksa kembali isian Anda, lalu coba masuk lagi.',
          tone: 'penolakan',
        }
      case ErrorCode.userInactive:
        return {
          title: 'Akun Anda tidak aktif',
          note:
            'Mengetik ulang tidak akan menolong. Hubungi administrator Claim PNC untuk mengaktifkan kembali akun Anda.',
          tone: 'penolakan',
        }
      case ErrorCode.identitySystemDown:
        return {
          title: 'Sistem identitas sedang tidak dapat dihubungi',
          note:
            'Ini bukan kesalahan Anda. Tunggu beberapa saat, lalu coba lagi — mencoba berulang kali tidak mempercepat pemulihan.',
          tone: 'gangguan',
        }
      default:
        return {
          title: 'Terjadi kesalahan pada sistem',
          note: 'Coba beberapa saat lagi. Bila berulang, hubungi administrator Claim PNC.',
          tone: 'gangguan',
        }
    }
  }
  return {
    title: 'Terjadi kesalahan pada sistem',
    note: 'Coba beberapa saat lagi.',
    tone: 'gangguan',
  }
}

type RouteState = { from?: Location }

export function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const login = useLogin()

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { username: '', password: '' },
  })

  // Setelah berhasil masuk, pengguna kembali ke halaman yang tadi ia tuju — bukan
  // selalu ke beranda. Ini yang membuat sesi habis di tengah pekerjaan tidak terasa
  // seperti kehilangan tempat.
  const target = (location.state as RouteState | null)?.from?.pathname ?? '/'

  const submit = handleSubmit((values) => {
    login.mutate(values, { onSuccess: () => navigate(target, { replace: true }) })
  })

  const failure = login.isError ? messageFor(login.error) : null

  return (
    <main className="flex min-h-screen items-center justify-center bg-slate-100 px-4 py-10">
      <div className="w-full max-w-md">
        <header className="mb-6 text-center">
          <h1 className="text-2xl font-semibold text-slate-900">Claim PNC</h1>
          <p className="mt-1 text-sm text-slate-600">Masuk dengan akun kantor Anda.</p>
        </header>

        <form
          onSubmit={submit}
          noValidate
          className="space-y-5 rounded-lg border border-slate-200 bg-white p-6 shadow-sm"
        >
          {failure && (
            <ErrorMessage title={failure.title} note={failure.note} tone={failure.tone} />
          )}

          <FormField
            id="username"
            label="Nama pengguna"
            type="text"
            autoComplete="username"
            autoFocus
            failure={errors.username?.message}
            {...register('username')}
          />

          <FormField
            id="password"
            label="Kata sandi"
            type="password"
            autoComplete="current-password"
            failure={errors.password?.message}
            {...register('password')}
          />

          <button
            type="submit"
            disabled={login.isPending}
            className="w-full rounded bg-slate-900 px-4 py-2 font-medium text-white hover:bg-slate-800 disabled:cursor-not-allowed disabled:bg-slate-400"
          >
            {login.isPending ? 'Memeriksa…' : 'Masuk'}
          </button>
        </form>

        <p className="mt-4 text-center text-xs text-slate-500">
          Kesulitan masuk? Hubungi administrator Claim PNC.
        </p>
      </div>
    </main>
  )
}
