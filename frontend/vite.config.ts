import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
    // transformAssetUrls would try to resolve "/icon.png" (a public asset)
    // through the bundler, which breaks both tests and the Wails asset server.
    plugins: [vue({template: {transformAssetUrls: false}})],
    // Wails' dev asset handler proxies to 127.0.0.1, so do not let Vite bind
    // only to ::1 (Vite's default for "localhost").
    server: {
        host: '127.0.0.1',
        port: 9245,
        strictPort: true,
    },
    test: {
        environment: 'jsdom',
        include: ['src/**/*.spec.ts'],
    },
})
