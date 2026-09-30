import * as React from 'react'
import { Slot } from '@radix-ui/react-slot'
import { cva, type VariantProps } from 'class-variance-authority'
import { cn } from '@/lib/utils'

const buttonVariants = cva(
  'inline-flex items-center justify-center gap-2 whitespace-nowrap rounded font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-brand-400 focus-visible:ring-offset-2 disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default:   'bg-brand-400 text-white hover:bg-brand-500 active:bg-brand-600',
        secondary: 'bg-neutral-100 text-neutral-800 hover:bg-neutral-200',
        outline:   'border border-neutral-300 bg-white text-neutral-800 hover:bg-neutral-50',
        ghost:     'text-neutral-700 hover:bg-neutral-100',
        'ghost-brand': 'text-brand-500 hover:bg-brand-50',
        destructive: 'bg-danger text-white hover:bg-red-700',
        link:      'text-brand-500 underline-offset-4 hover:underline p-0 h-auto',
      },
      size: {
        default: 'h-12 px-6 text-base',
        sm:      'h-9 px-4 text-sm',
        lg:      'h-14 px-8 text-lg',
        icon:    'h-10 w-10',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'default',
    },
  },
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, asChild = false, ...props }, ref) => {
    const Comp = asChild ? Slot : 'button'
    return (
      <Comp
        className={cn(buttonVariants({ variant, size, className }))}
        ref={ref}
        {...props}
      />
    )
  },
)
Button.displayName = 'Button'

export { Button, buttonVariants }
