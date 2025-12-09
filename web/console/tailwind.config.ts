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
        }
      },
      fontFamily: {
        sans: [
          'Inter',
          'ui-sans-serif',
          'system-ui',
          'Helvetica Neue',
          'Arial',
          'Noto Sans',
          'sans-serif'
        ]
      },
      boxShadow: {
        card: '0 1px 2px rgba(0,0,0,0.06), 0 1px 3px rgba(0,0,0,0.1)'
      }
    }
  },
  plugins: []
} satisfies Config
