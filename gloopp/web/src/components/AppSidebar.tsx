import { useTranslation } from 'react-i18next'
import Icon from './Icon'
import Logo from './Logo'

export type AppNavItem<T extends string> = {
  key: T
  label: string
  icon: string
}

type Props<T extends string> = {
  items: AppNavItem<T>[]
  currentTab: T
  questOpen: boolean
  collapsed: boolean
  version: string
  navCount: (key: T) => number
  onNavigate: (key: T) => void
  onToggleCollapsed: () => void
}

export default function AppSidebar<T extends string>({
  items,
  currentTab,
  questOpen,
  collapsed,
  version,
  navCount,
  onNavigate,
  onToggleCollapsed,
}: Props<T>) {
  const { t } = useTranslation()
  return (
    <aside className={'sidebar' + (collapsed ? ' collapsed' : '')}>
      <button
        className="brand brand-toggle"
        onClick={onToggleCollapsed}
        aria-label={collapsed ? t('topbar.menu.expandSidebar') : t('topbar.menu.collapseSidebar')}
        title={collapsed ? t('topbar.menu.expandSidebar') : t('topbar.menu.collapseSidebar')}
      >
        <div className="brand-logo-wrap">
          <Logo variant="official" size={36} className="brand-logo" />
          <Icon name="menu" size={20} className="brand-menu-icon" />
        </div>
        {!collapsed && (
          <div className="brand-text">
            <strong>Gloop</strong>
            <span>{t('brand.tagline')}</span>
          </div>
        )}
      </button>

      <nav className="nav">
        {!collapsed && <div className="nav-label">{t('nav.workbench')}</div>}
        {items.map((item) => (
          <button
            key={item.key}
            className={currentTab === item.key && !questOpen ? 'active' : ''}
            onClick={() => onNavigate(item.key)}
            title={collapsed ? item.label : undefined}
          >
            <Icon name={item.icon} />
            {!collapsed && <span>{item.label}</span>}
            {navCount(item.key) > 0 && <em>{navCount(item.key)}</em>}
          </button>
        ))}
      </nav>

      <div className="sidebar-foot">
        {collapsed ? null : (
          <a
            className="external-link-button"
            href="https://bytedance.larkoffice.com/docx/KIJ7dcrjZoQ0WqxqKoBcSb9wnVJ"
            target="_blank"
            rel="noopener noreferrer"
          >
            <Icon name="file-text" />
            <span>{t('nav.docs')}</span>
            <Icon name="open" />
          </a>
        )}
        {!collapsed && (
          <div className="sidebar-version" title={t('nav.version', { version })}>
            v{version}
          </div>
        )}
      </div>
    </aside>
  )
}
