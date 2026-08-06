import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import zhCN from './locales/zh-CN.json'

// i18n 初始化：仅 bundled zh-CN，不上 language detector / backend loader。
// world.* 保留中文世界观词（冒险者/剑士/法师/委托），term.* 保留英文技术术语（Checker/Fanout 等）。
// 详见 Slice 1 key 命名规则文档。
i18n.use(initReactI18next).init({
  resources: {
    'zh-CN': { translation: zhCN },
  },
  lng: 'zh-CN',
  fallbackLng: 'zh-CN',
  interpolation: {
    escapeValue: false,
  },
  returnNull: false,
})

export default i18n
