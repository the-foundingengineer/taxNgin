import { AuthLayout } from '@/components/layout/AuthLayout'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { authApi } from '@/lib/api/auth'
import { ApiError } from '@/lib/api/client'
import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { Link, useNavigate } from 'react-router-dom'
import { z } from 'zod'

const schema = z.object({
  nin: z.string().length(11, 'NIN must be exactly 11 digits').regex(/^\d+$/, 'NIN must be digits only'),
  phone: z.string().min(10, 'Enter a valid phone number'),
  email: z.string().email('Invalid email').optional().or(z.literal('')),
  password: z.string().min(8, 'Password must be at least 8 characters'),
  confirm: z.string(),
}).refine(d => d.password === d.confirm, { message: 'Passwords do not match', path: ['confirm'] })

type FormValues = z.infer<typeof schema>

export function RegisterPage() {
  const navigate = useNavigate()
  const [serverError, setServerError] = useState<string | null>(null)

  const { register, handleSubmit, formState: { errors, isSubmitting } } = useForm<FormValues>({
    resolver: zodResolver(schema),
  })

  async function onValid(values: FormValues) {
    setServerError(null)
    try {
      await authApi.register({
        nin: values.nin,
        phone: values.phone,
        email: values.email || undefined,
        password: values.password,
      })
      navigate('/login', { replace: true })
    } catch (e) {
      if (e instanceof ApiError) {
        setServerError(e.status === 409 ? 'An account already exists for this NIN.' : e.message)
      }
    }
  }

  return (
    <AuthLayout title="Sign up for Tax Check" subtitle="Join thousands filing faster">

      {/* Mock Social Logins */}
      <div className="flex gap-4 mb-6">
        <Button variant="outline" type="button" className="flex-1 rounded-xl h-12 text-sm font-medium border-neutral-200">
          <svg className="w-5 h-5 mr-2" viewBox="0 0 24 24">
            <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4" />
            <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853" />
            <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05" />
            <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335" />
          </svg>
          Google
        </Button>
        <Button variant="outline" type="button" className="flex-1 rounded-xl h-12 text-sm font-medium border-neutral-200">
          <svg className="w-5 h-5 mr-2" viewBox="0 0 24 24" fill="currentColor">
            <path d="M17.05 20.28c-.98.95-2.05.8-3.08.35-1.09-.46-2.09-.48-3.24 0-1.44.62-2.2.44-3.06-.35C2.79 15.25 3.51 7.59 9.05 7.31c1.35.07 2.29.74 3.08.8 1.18-.04 2.26-.74 3.58-.79 1.5-.1 2.64.67 3.4 1.74-2.88 1.64-2.39 5.61.56 6.78-.65 1.76-1.55 3.35-2.62 4.44zm-2.05-14c-.65-1.39-1.92-2.28-3.29-2.28-.15 1.48.56 2.87 1.39 3.8.76.84 2.13 1.55 3.19 1.48-.12-1.25-.49-2.24-1.29-3z" />
          </svg>
          Apple
        </Button>
      </div>

      <div className="relative mb-6">
        <div className="absolute inset-0 flex items-center">
          <div className="w-full border-t border-neutral-200" />
        </div>
        <div className="relative flex justify-center text-xs">
          <span className="bg-white px-4 text-neutral-400">Or manually</span>
        </div>
      </div>

      <form onSubmit={handleSubmit(onValid)} className="space-y-4">
        {serverError && (
          <Alert variant="destructive"><AlertDescription>{serverError}</AlertDescription></Alert>
        )}

        <div>
          <Input
            id="nin"
            {...register('nin')}
            inputMode="numeric"
            maxLength={11}
            className={`h-12 rounded-xl border-neutral-200 px-4 text-[15px] focus:ring-info ${errors.nin ? 'border-danger' : ''}`}
            placeholder="11-digit NIN"
          />
          {errors.nin && <p className="text-xs text-danger mt-1.5 ml-1">{errors.nin.message}</p>}
        </div>

        <div>
          <Input
            id="phone"
            {...register('phone')}
            inputMode="tel"
            className={`h-12 rounded-xl border-neutral-200 px-4 text-[15px] focus:ring-info ${errors.phone ? 'border-danger' : ''}`}
            placeholder="Phone Number (e.g. 08012345678)"
          />
          {errors.phone && <p className="text-xs text-danger mt-1.5 ml-1">{errors.phone.message}</p>}
        </div>

        <div>
          <Input
            id="email"
            {...register('email')}
            type="email"
            className={`h-12 rounded-xl border-neutral-200 px-4 text-[15px] focus:ring-info ${errors.email ? 'border-danger' : ''}`}
            placeholder="Email Address (optional)"
          />
          {errors.email && <p className="text-xs text-danger mt-1.5 ml-1">{errors.email.message}</p>}
        </div>

        <div className="flex gap-3">
          <div className="flex-1">
            <Input
              id="password"
              type="password"
              {...register('password')}
              className={`h-12 rounded-xl border-neutral-200 px-4 text-[15px] focus:ring-info ${errors.password ? 'border-danger' : ''}`}
              placeholder="Password"
              autoComplete="new-password"
            />
            {errors.password && <p className="text-xs text-danger mt-1.5 ml-1">{errors.password.message}</p>}
          </div>

          <div className="flex-1">
            <Input
              id="confirm"
              type="password"
              {...register('confirm')}
              className={`h-12 rounded-xl border-neutral-200 px-4 text-[15px] focus:ring-info ${errors.confirm ? 'border-danger' : ''}`}
              placeholder="Confirm"
              autoComplete="new-password"
            />
            {errors.confirm && <p className="text-xs text-danger mt-1.5 ml-1">{errors.confirm.message}</p>}
          </div>
        </div>

        <Button type="submit" className="w-full h-12 rounded-xl bg-info hover:bg-info/90 text-white font-semibold text-[15px] mt-2 transition-colors" disabled={isSubmitting}>
          {isSubmitting ? 'Creating account…' : 'Create Account'}
        </Button>

        <div className="text-center pt-2 text-[15px] text-neutral-600">
          Already have an account? <Link to="/login" className="font-bold text-neutral-900 hover:underline">Sign In</Link>
        </div>
      </form>
    </AuthLayout>
  )
}
