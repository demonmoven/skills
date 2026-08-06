import i18n from './i18n'

export type ThemeKey = 'bili' | 'emerald' | 'teal' | 'violet' | 'sakura' | 'obsidian'

export type ThemeMeta = {
  key: ThemeKey
  label: string
  desc: string
  swatch: string
}

export const THEMES: ThemeMeta[] = [
  { key: 'obsidian', label: 'Obsidian', desc: i18n.t('theme.desc.obsidian'), swatch: '#404040' },
  { key: 'bili', label: 'Bili Blue', desc: i18n.t('theme.desc.bili'), swatch: '#0284c7' },
  { key: 'teal', label: 'Deep Teal', desc: i18n.t('theme.desc.teal'), swatch: '#0891b2' },
  { key: 'emerald', label: 'Emerald', desc: i18n.t('theme.desc.emerald'), swatch: '#16a34a' },
  { key: 'violet', label: 'Deep Violet', desc: i18n.t('theme.desc.violet'), swatch: '#7c3aed' },
  { key: 'sakura', label: 'Sakura', desc: i18n.t('theme.desc.sakura'), swatch: '#db2777' },
]

const STORAGE_KEY = 'gloop-theme'
const DEFAULT_THEME: ThemeKey = 'emerald'

export function getStoredTheme(): ThemeKey {
  try {
    const v = localStorage.getItem(STORAGE_KEY)
    if (v && THEMES.some((t) => t.key === v)) return v as ThemeKey
  } catch {
    /* ignore */
  }
  return DEFAULT_THEME
}

export function applyTheme(theme: ThemeKey) {
  const root = document.documentElement
  root.setAttribute('data-theme', theme)
  try {
    localStorage.setItem(STORAGE_KEY, theme)
  } catch {
    /* ignore */
  }
}

export function initTheme() {
  applyTheme(getStoredTheme())
}
