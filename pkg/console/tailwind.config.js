module.exports = {
  content: [
    './templates/**/*.templ',
    './templates/**/*.go',
    './assets/input.css'
  ],
  theme: {
    extend: {
      fontFamily: {
        'sans': ['Space Grotesk', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        'mono': ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace'],
      },
      colors: {
        'dark': {
          900: '#0a0a0a',
          800: '#141414',
          700: '#1a1a1a',
          600: '#242424',
        },
        'neutral': {
          900: '#0a0a0a',
          800: '#383838',
          700: '#292929',
          600: '#525252',
          500: '#737373',
          400: '#b3b3b3',
          300: '#d4d4d4',
          200: '#e0e0e0',
          100: '#ededed',
        }
      }
    },
  },
  plugins: [],
}
