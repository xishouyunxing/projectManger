import { build } from 'vite'
import { fileURLToPath } from 'url'
import { dirname, resolve } from 'path'

const __filename = fileURLToPath(import.meta.url)
const __dirname = dirname(__filename)

const config = {
  configFile: resolve(__dirname, 'vite.config.ts'),
  root: __dirname,
}

build(config).catch((err) => {
  console.error('Build failed:', err)
  process.exit(1)
})