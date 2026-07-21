import PalworldTab from './PalworldTab.vue';

export const palworldPlugin = {
  // GameAP exposes backend routes under the compact ID generated from the
  // numeric plugin ID. Keep the frontend registration on that same ID so
  // component props resolve to /api/plugins/h4fkd4cukuk6o/....
  id: 'h4fkd4cukuk6o',
  name: 'Palworld REST Control',
  version: '0.1.0',
  apiVersion: '1.0',
  description: 'Palworld official REST API management in a native GameAP server tab',
  author: 'RANSUROTTO',
  translations: {
    en: { server_tab: 'Palworld', not_palworld: 'This tab is available only for Palworld servers.' },
    'zh-CN': { server_tab: '幻兽帕鲁', not_palworld: '此标签仅适用于幻兽帕鲁服务器。' },
    ru: { server_tab: 'Palworld', not_palworld: 'Эта вкладка доступна только для серверов Palworld.' }
  },
  slots: {
    'server-tabs': [{ component: PalworldTab, order: 40, label: '@:server_tab', icon: 'gamepad', name: 'palworld' }]
  }
};
