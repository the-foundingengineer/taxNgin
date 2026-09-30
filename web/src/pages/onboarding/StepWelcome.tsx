import { useForm } from 'react-hook-form'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { ChevronRight } from 'lucide-react'

const schema = z.object({
    firstName: z.string().min(1, 'Please enter your first name'),
    lastName: z.string().min(1, 'Please enter your last name'),
    dateOfBirth: z.string().min(1, 'Please enter your date of birth'),
})

export type WelcomeData = z.infer<typeof schema>

interface Props {
    defaultValues?: Partial<WelcomeData>
    onNext: (data: WelcomeData) => void
}

export function StepWelcome({ defaultValues, onNext }: Props) {
    const {
        register,
        handleSubmit,
        formState: { errors },
    } = useForm<WelcomeData>({
        resolver: zodResolver(schema),
        defaultValues,
    })

    return (
        <div>
            {/* Heading */}
            <div className="mb-10">
                <div className="inline-flex items-center gap-2 px-3 py-1.5 rounded-full text-xs font-semibold mb-6"
                    style={{ background: 'rgba(28,78,216,0.07)', color: '#1C4ED8', border: '1px solid rgba(28,78,216,0.14)' }}>
                    <span className="material-symbols-outlined text-[16px]">waving_hand</span> Welcome to Tax Check
                </div>
                <h1 className="text-[32px] lg:text-[38px] font-bold text-neutral-900 leading-tight mb-3">
                    Let's start by getting<br />to know you.
                </h1>
                <p className="text-neutral-500 text-base leading-relaxed">
                    We just need a few details — takes less than a minute.
                </p>
            </div>

            {/* Form */}
            <form onSubmit={handleSubmit(onNext)} className="space-y-5">
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div>
                        <label className="block text-sm font-semibold text-neutral-700 mb-1.5">First name</label>
                        <Input
                            {...register('firstName')}
                            placeholder="e.g. Amara"
                            className={`h-12 rounded-xl border-neutral-200 px-4 text-[15px] transition-shadow focus:shadow-[0_0_0_3px_rgba(28,78,216,0.12)] ${errors.firstName ? 'border-red-400' : ''}`}
                        />
                        {errors.firstName && (
                            <p className="text-xs text-red-500 mt-1.5">{errors.firstName.message}</p>
                        )}
                    </div>
                    <div>
                        <label className="block text-sm font-semibold text-neutral-700 mb-1.5">Last name</label>
                        <Input
                            {...register('lastName')}
                            placeholder="e.g. Okonkwo"
                            className={`h-12 rounded-xl border-neutral-200 px-4 text-[15px] transition-shadow focus:shadow-[0_0_0_3px_rgba(28,78,216,0.12)] ${errors.lastName ? 'border-red-400' : ''}`}
                        />
                        {errors.lastName && (
                            <p className="text-xs text-red-500 mt-1.5">{errors.lastName.message}</p>
                        )}
                    </div>
                </div>

                <div className="max-w-xs">
                    <label className="block text-sm font-semibold text-neutral-700 mb-1.5">
                        Date of birth
                        <span
                            className="ml-2 text-xs font-normal cursor-pointer"
                            style={{ color: '#1C4ED8' }}
                        >
                            Why we ask
                        </span>
                    </label>
                    <Input
                        {...register('dateOfBirth')}
                        type="date"
                        className={`h-12 rounded-xl border-neutral-200 px-4 text-[15px] transition-shadow focus:shadow-[0_0_0_3px_rgba(28,78,216,0.12)] ${errors.dateOfBirth ? 'border-red-400' : ''}`}
                    />
                    {errors.dateOfBirth && (
                        <p className="text-xs text-red-500 mt-1.5">{errors.dateOfBirth.message}</p>
                    )}
                    <p className="text-xs text-neutral-400 mt-1.5">Used to verify your tax identity — never shared.</p>
                </div>

                {/* CTA */}
                <div className="pt-4">
                    <Button
                        type="submit"
                        className="h-12 px-8 rounded-xl text-white font-semibold text-[15px] flex items-center gap-2 transition-all hover:gap-3"
                        style={{ background: 'linear-gradient(135deg, #1C4ED8 0%, #2563eb 100%)' }}
                    >
                        Let's go <ChevronRight className="w-4 h-4" />
                    </Button>
                </div>
            </form>
        </div>
    )
}
