import animate from 'tailwindcss-animate'

/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  theme: {
    extend: {
      colors: {
        // 深空配色 — STARFORGE
        void: {
          900: '#03050d',
          800: '#060a18',
          700: '#0a1024',
          600: '#0f1733',
          500: '#15204a',
        },
        neon: {
          cyan: '#00f0ff',
          magenta: '#ff00aa',
          green: '#00ff88',
          amber: '#ffaa00',
          violet: '#a855f7',
          rose: '#ff2d6f',
        },
        sev: {
          critical: '#ff2d6f',
          major: '#ff7a1a',
          minor: '#ffd400',
          warning: '#5b9eff',
          ok: '#00ff88',
          off: '#525a78',
        },
      },
      fontFamily: {
        mono: [
          'JetBrains Mono',
          'Fira Code',
          'SF Mono',
          'Menlo',
          'Consolas',
          'monospace',
        ],
        display: [
          'Orbitron',
          'JetBrains Mono',
          'SF Mono',
          'Menlo',
          'Consolas',
          'monospace',
        ],
      },
      keyframes: {
        scan: {
          '0%': { transform: 'translateY(-100%)' },
          '100%': { transform: 'translateY(100%)' },
        },
        flicker: {
          '0%, 100%': { opacity: '1' },
          '5%': { opacity: '0.85' },
          '10%': { opacity: '1' },
          '15%': { opacity: '0.95' },
        },
        sweep: {
          '0%': { transform: 'rotate(0deg)' },
          '100%': { transform: 'rotate(360deg)' },
        },
        spinSlow: {
          '0%': { transform: 'rotateY(0deg)' },
          '100%': { transform: 'rotateY(360deg)' },
        },
        pulseRing: {
          '0%': { transform: 'scale(0.85)', opacity: '0.85' },
          '100%': { transform: 'scale(1.6)', opacity: '0' },
        },
        ticker: {
          '0%': { transform: 'translateY(0)' },
          '100%': { transform: 'translateY(-50%)' },
        },
        breathe: {
          '0%, 100%': { opacity: '0.55', filter: 'brightness(1)' },
          '50%': { opacity: '1', filter: 'brightness(1.4)' },
        },
        cursorBlink: {
          '0%, 49%': { opacity: '1' },
          '50%, 100%': { opacity: '0' },
        },
      },
      animation: {
        scan: 'scan 4s linear infinite',
        flicker: 'flicker 4s linear infinite',
        sweep: 'sweep 4s linear infinite',
        spinSlow: 'spinSlow 24s linear infinite',
        pulseRing: 'pulseRing 2.4s ease-out infinite',
        ticker: 'ticker 30s linear infinite',
        breathe: 'breathe 2.8s ease-in-out infinite',
        cursorBlink: 'cursorBlink 1.1s steps(2) infinite',
      },
    },
  },
  plugins: [animate],
}
