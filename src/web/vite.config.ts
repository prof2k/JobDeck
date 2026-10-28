import { defineConfig } from 'vite';
import vue from '@vitejs/plugin-vue';

export default defineConfig(({ mode }) => ({
  plugins: [vue(), ...(mode === 'shared' ? [{
      name: 'shared-progress-entry',
      transformIndexHtml: {
        order: 'pre' as const,
        handler(html: string) {
          return html.replace('/src/main.ts', '/src/shared-main.ts');
        },
      },
  }] : [])],
  build: {
    target: 'es2022',
    outDir: mode === 'shared' ? '../sharing/dist' : 'dist',
    emptyOutDir: true,
  },
}));
