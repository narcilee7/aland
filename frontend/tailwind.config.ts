import type {Config} from 'tailwindcss'

const config: Config = {
  content: ['./index.html', './src/**/*.{ts,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      // 设计 token：与 design doc 6.1 节呼应
      colors: {
        // 深夜大陆基调
        land: {
          1: '#0a0e1a', // 极深靛蓝
          2: '#0d1f15', // 墨绿
          3: '#1a2332', // UI 面板
          4: '#252f3f', // 边框
        },
        // 部落主色（动态注入）
        tribe: {
          theme: 'var(--tribe-theme)',
          accent: 'var(--tribe-accent)',
        },
        // Token 熔炉
        forge: {
          green: '#22c55e',
          amber: '#f59e0b',
          red: '#ef4444',
        },
        ink: {
          DEFAULT: '#e2e8f0',
          dim: '#94a3b8',
          faint: '#475569',
        },
      },
      fontFamily: {
        mono: ['JetBrains Mono', 'SF Mono', 'Menlo', 'monospace'],
        serif: ['Source Han Serif', 'Songti SC', 'Playfair Display', 'serif'],
      },
      letterSpacing: {
        wider: '0.08em',
      },
      keyframes: {
        // 所有生命体的呼吸
        breathe: {
          '0%, 100%': {transform: 'scale(1)', opacity: '0.8'},
          '50%': {transform: 'scale(1.06)', opacity: '1'},
        },
        // 慢速旋转——中央圣所
        slowspin: {
          '0%': {transform: 'rotate(0deg)'},
          '100%': {transform: 'rotate(360deg)'},
        },
      },
      animation: {
        breathe: 'breathe 4s ease-in-out infinite',
        slowspin: 'slowspin 30s linear infinite',
      },
    },
  },
  plugins: [],
}

export default config
