import type { Config } from 'tailwindcss'

export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // ── Semantic tokens (CSS variable-backed, theme-aware) ──
        surface: {
          primary: 'var(--surface-primary)',
          secondary: 'var(--surface-secondary)',
          tertiary: 'var(--surface-tertiary)',
          elevated: 'var(--surface-elevated)',
          hover: 'var(--surface-hover)',
          active: 'var(--surface-active)',
          inset: 'var(--surface-inset)'
        },
        content: {
          primary: 'var(--content-primary)',
          secondary: 'var(--content-secondary)',
          tertiary: 'var(--content-tertiary)',
          placeholder: 'var(--content-placeholder)',
          inverse: 'var(--content-inverse)'
        },
        border: {
          DEFAULT: 'var(--border-default)',
          strong: 'var(--border-strong)'
        },
        accent: {
          DEFAULT: 'var(--accent-color)',
          hover: 'var(--accent-color-hover)',
          subtle: 'var(--accent-subtle)'
        },
        status: {
          success: 'var(--status-success)',
          warning: 'var(--status-warning)',
          error: 'var(--status-error)',
          info: 'var(--status-info)',
          // Text-level tokens (theme-adaptive readability)
          link: 'var(--text-link)',
          cyan: 'var(--text-cyan)',
          purple: 'var(--text-purple)',
          indigo: 'var(--text-indigo)',
          orange: 'var(--text-orange)',
          rose: 'var(--text-rose)',
          'text-success': 'var(--text-success)',
          'text-warning': 'var(--text-warning)',
          'text-error': 'var(--text-error)',
          'text-info': 'var(--text-info)'
        }
      },
      fontFamily: {
        sans: [
          '-apple-system',
          'BlinkMacSystemFont',
          '"SF Pro Text"',
          '"SF Pro Icons"',
          'Inter',
          'system-ui',
          'sans-serif'
        ]
      },
      boxShadow: {
        // Apple-style layered shadows
        card: '0 1px 2px rgba(0,0,0,0.04), 0 1px 3px rgba(0,0,0,0.02)',
        glass: '0 8px 32px rgba(0,0,0,0.32), inset 0 0 0 1px rgba(255,255,255,0.05)',
        'glass-sm': '0 2px 8px rgba(0,0,0,0.12), inset 0 0 0 1px rgba(255,255,255,0.04)'
      },
      borderRadius: {
        xl: '12px',
        '2xl': '16px',
        '3xl': '24px'
      },
      animation: {
        'fade-in': 'fadeIn 0.2s ease-out',
        'slide-in': 'slideIn 0.3s cubic-bezier(0.16, 1, 0.3, 1)',
        'scale-in': 'scaleIn 0.2s cubic-bezier(0.16, 1, 0.3, 1)'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideIn: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.98)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        }
      }
    }
  },
  plugins: []
} satisfies Config
