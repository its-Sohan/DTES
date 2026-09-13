/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        desk: {
          light: '#ECECE8',
          dark: '#111111',
        },
        surface: {
          light: '#FFFFFF',
          dark: '#1E1E1E',
        },
        inset: {
          light: '#F4F4F0',
          dark: '#141414',
        },
        hairline: {
          light: '#CFCFC9',
          dark: '#383835',
          subtleLight: '#DFDFD9',
          subtleDark: '#282826',
        },
        brand: {
          light: '#2563EB',
          dark: '#4B88F0',
        },
        ink: {
          primaryLight: '#141412',
          secondaryLight: '#565650',
          primaryDark: '#F4F4F0',
          secondaryDark: '#A2A29C',
        },
      },
      borderRadius: {
        panel: '6px',
        glass: '16px',
      },
      fontFamily: {
        sans: ['Inter', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif'],
        mono: ['JetBrains Mono', 'IBM Plex Mono', 'Menlo', 'Consolas', 'monospace'],
        bengali: ['Kalpurush', 'Noto Sans Bengali', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
