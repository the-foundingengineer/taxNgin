import type { Config } from 'tailwindcss'

const config: Config = {
  darkMode: ['class'],
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // Wealthsimple-inspired trust teal
        brand: {
          '50':  '#f0fafb',
          '100': '#d9f2f5',
          '200': '#b3e5eb',
          '300': '#7dcfda',
          '400': '#45b4c2',
          '500': '#2a98a8',
          '600': '#1f7a88',
          '700': '#1a6170',
          '800': '#1a505e',
          '900': '#1b4450',
        },
        // Warm grey (not cold blue-grey — Wealthsimple uses warm neutrals)
        neutral: {
          '50':  '#f8f8f7',
          '100': '#f0efed',
          '200': '#e4e2de',
          '300': '#cbc9c4',
          '400': '#a09e97',
          '500': '#7a7872',
          '600': '#5e5d57',
          '700': '#4a4944',
          '800': '#3a3935',
          '900': '#272622',
        },
        success:  { DEFAULT: '#16a34a', light: '#dcfce7', dark: '#14532d' },
        warning:  { DEFAULT: '#d97706', light: '#fef3c7', dark: '#78350f' },
        danger:   { DEFAULT: '#dc2626', light: '#fee2e2', dark: '#7f1d1d' },
        info:     { DEFAULT: '#2563eb', light: '#dbeafe' },
        // shadcn/ui CSS variable aliases
        background: 'hsl(var(--background))',
        foreground:  'hsl(var(--foreground))',
        card: {
          DEFAULT:    'hsl(var(--card))',
          foreground: 'hsl(var(--card-foreground))',
        },
        primary: {
          DEFAULT:    'hsl(var(--primary))',
          foreground: 'hsl(var(--primary-foreground))',
        },
        secondary: {
          DEFAULT:    'hsl(var(--secondary))',
          foreground: 'hsl(var(--secondary-foreground))',
        },
        muted: {
          DEFAULT:    'hsl(var(--muted))',
          foreground: 'hsl(var(--muted-foreground))',
        },
        accent: {
          DEFAULT:    'hsl(var(--accent))',
          foreground: 'hsl(var(--accent-foreground))',
        },
        destructive: {
          DEFAULT:    'hsl(var(--destructive))',
          foreground: 'hsl(var(--destructive-foreground))',
        },
        border: 'hsl(var(--border))',
        input:  'hsl(var(--input))',
        ring:   'hsl(var(--ring))',
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
        mono: ['JetBrains Mono', 'ui-monospace', 'monospace'],
      },
      fontSize: {
        xs:   ['0.75rem',  { lineHeight: '1.125rem' }],
        sm:   ['0.875rem', { lineHeight: '1.375rem' }],
        base: ['1rem',     { lineHeight: '1.625rem' }],
        lg:   ['1.125rem', { lineHeight: '1.75rem'  }],
        xl:   ['1.25rem',  { lineHeight: '1.875rem' }],
        '2xl': ['1.5rem',  { lineHeight: '2rem'     }],
        '3xl': ['1.875rem',{ lineHeight: '2.25rem'  }],
      },
      borderRadius: {
        sm:   '0.375rem',
        DEFAULT: '0.5rem',
        lg:   '0.75rem',
        xl:   '1rem',
        '2xl':'1.5rem',
        full: '9999px',
      },
      boxShadow: {
        card:       '0 1px 3px 0 rgb(0 0 0 / 0.06), 0 1px 2px -1px rgb(0 0 0 / 0.06)',
        'card-hover':'0 4px 6px -1px rgb(0 0 0 / 0.08), 0 2px 4px -2px rgb(0 0 0 / 0.05)',
        modal:      '0 20px 25px -5px rgb(0 0 0 / 0.10), 0 8px 10px -6px rgb(0 0 0 / 0.08)',
      },
    },
  },
  plugins: [],
}

export default config
