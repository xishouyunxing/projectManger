import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: 'dist',
    assetsInlineLimit: 0,
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (!id.includes('node_modules')) return undefined

          const modulePath = id.split('node_modules/')[1]
          if (!modulePath) return 'vendor'

          const segments = modulePath.split('/')
          const packageName = segments[0].startsWith('@')
            ? `${segments[0]}-${segments[1]}`
            : segments[0]
          const normalizedPackage = packageName.replace('@', '')

          if (segments[0] === 'antd') {
            const antdGroup = segments[2] || 'core'
            return `vendor-antd-${antdGroup}`
          }

          return `vendor-${normalizedPackage}`
        },
      },
    },
  },
  server: {
    port: 3000,
    host: true,
    fs: {
      allow: ['..'],
    },
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
})
