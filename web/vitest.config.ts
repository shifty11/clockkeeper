import { defineConfig } from 'vitest/config';
import { resolve } from 'path';

export default defineConfig({
	resolve: {
		alias: {
			'$app/navigation': resolve(__dirname, 'src/lib/__mocks__/app-navigation.ts'),
			'$app/stores': resolve(__dirname, 'src/lib/__mocks__/app-stores.ts'),
			'~': resolve(__dirname, 'src')
		}
	},
	test: {
		environment: 'jsdom',
		include: ['src/**/*.test.ts']
	}
});
