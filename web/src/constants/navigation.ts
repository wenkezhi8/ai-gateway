export const HOME_ROUTE = '/'
export const LOGIN_ROUTE = '/login'
export const DASHBOARD_ROUTE = '/dashboard'
export const DOCS_ROUTE = '/docs'
export const SETTINGS_ROUTE = '/settings'
export const PUBLIC_CHAT_ROUTE = '/p/chat'
export const HEALTH_ROUTE = '/health'

export const POST_LOGOUT_REDIRECT = HOME_ROUTE
export const UNAUTHORIZED_REDIRECT = LOGIN_ROUTE
export const LOGIN_SUCCESS_REDIRECT = DASHBOARD_ROUTE

export const PUBLIC_PRECHECK_PATHS = [
  HOME_ROUTE,
  DOCS_ROUTE,
  LOGIN_ROUTE,
  PUBLIC_CHAT_ROUTE,
  HEALTH_ROUTE
] as const
