/** @type {import('tailwindcss').Config} */
export default {
  darkMode: 'class',
  content: ['./index.html', './src/**/*.{vue,js}'],
  theme: {
    extend: {
      colors: {
        brand: { 50: '#fff7ed', 100: '#ffedd5', 200: '#fed7aa', 300: '#fdba74', 400: '#fb923c', 500: '#f97316', 600: '#ea580c', 700: '#c2410c', 800: '#9a3412', 900: '#7c2d12' },
        ink: { 50: '#f8f7f4', 100: '#f1efe9', 200: '#e4e0d6', 300: '#cfc9bb', 400: '#a19a8a', 500: '#7a7466', 600: '#5c574c', 700: '#433f37', 800: '#2b2924', 900: '#1b1a17', 950: '#11100e' }
      },
      fontFamily: {
        sans: ['"PingFang SC"', '"Microsoft YaHei"', '"Noto Sans SC"', 'system-ui', 'sans-serif'],
        hand: ['"LXGW WenKai"', '"Kaiti SC"', 'serif']
      },
      boxShadow: { soft: '0 10px 40px -12px rgba(124, 45, 18, 0.18)' }
    }
  }
}
