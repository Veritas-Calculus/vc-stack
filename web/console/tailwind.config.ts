import type { Config } from 'tailwindcss'

export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        oxide: {
          50: '#f6f7fb',
          100: '#e9edf6',
          200: '#d2daea',
          300: '#aebbd6',
          400: '#7f96b7',
          500: '#61799a',
          600: '#4a5e7c',
          700: '#374762',
          800: '#2a364c',
          900: '#1c2537',
          950: '#121827'
        },
        // Apple-inspired accent palette
        apple: {
          blue: '#0A84FF',
          indigo: '#5E5CE6',
          purple: '#BF5AF2',
          pink: '#FF375F',
          red: '#FF453A',
          orange: '#FF9F0A',
          yellow: '#FFD60A',
          green: '#30D158',
          teal: '#64D2FF',
          gray1: '#8E8E93',
          gray2: '#636366',
          gray3: '#48484A',
          gray4: '#3A3A3C',
          gray5: '#2C2C2E',
          gray6: '#1C1C1E',
        }
      },
      fontFamily: {
        sans: [
          'Inter',
          '-apple-system',
          'BlinkMacSystemFont',
          'system-ui',
          'Helvetica Neue',
          'Arial',
          'sans-serif'
        ]
      },
      boxShadow: {
        card: '0 1px 2px rgba(0,0,0,0.06), 0 1px 3px rgba(0,0,0,0.1)',
        // Apple-style layered shadows
        'glass': '0 0 0 1px rgba(255,255,255,0.05), 0 2px 8px rgba(0,0,0,0.3), 0 8px 32px rgba(0,0,0,0.2)',
        'glass-sm': '0 0 0 1px rgba(255,255,255,0.04), 0 1px 4px rgba(0,0,0,0.2)',
        'glow-blue': '0 0 20px rgba(10,132,255,0.15)',
        'glow-purple': '0 0 20px rgba(94,92,230,0.15)',
      },
      borderRadius: {
        '2xl': '1rem',
        '3xl': '1.5rem',
      },
      backdropBlur: {
        'xs': '2px',
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-in': 'slideIn 0.25s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        slideIn: {
          '0%': { opacity: '0', transform: 'translateY(-4px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' },
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' },
        },
      },
    }
  },
  plugins: []
} satisfies Config
