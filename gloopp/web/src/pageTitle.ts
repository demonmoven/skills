const APP_NAME = 'Gloop'

export function formatPageTitle(pageTitle?: string | null) {
  const cleanTitle = pageTitle?.trim()
  return cleanTitle ? `${cleanTitle} · ${APP_NAME}` : APP_NAME
}
