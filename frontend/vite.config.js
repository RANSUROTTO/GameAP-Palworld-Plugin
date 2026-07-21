import { createPluginConfig } from '@gameap/plugin-sdk/vite';
import { resolve } from 'path';
const config = createPluginConfig({ entry: 'src/index.ts', name: 'plugin', outDir: 'dist' });
config.resolve.alias['@gameap/plugin-sdk'] = resolve(process.cwd(), '../../GameAP/web/plugin-sdk/src/index.ts');
export default config;
