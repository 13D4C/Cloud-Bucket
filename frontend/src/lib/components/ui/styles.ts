/**
 * Shared Tailwind CSS class combinations for consistent UI styling
 */
export const styles = {
  // Layout
  container: 'max-w-7xl mx-auto px-4 sm:px-6 lg:px-8',
  
  // Buttons
  button: {
    base: 'inline-flex items-center justify-center font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-2',
    primary: 'bg-accent-500 hover:bg-accent-600 text-white focus:ring-accent-500',
    secondary: 'bg-primary-700 hover:bg-primary-600 text-primary-50 border border-primary-600 focus:ring-primary-600',
    danger: 'bg-red-500 hover:bg-red-600 text-white focus:ring-red-500',
    ghost: 'hover:bg-primary-700 text-primary-300 hover:text-primary-50',
    disabled: 'opacity-50 cursor-not-allowed pointer-events-none',
    size: {
      sm: 'px-3 py-1.5 text-sm gap-1.5',
      md: 'px-4 py-2 text-base gap-2',
      lg: 'px-6 py-3 text-lg gap-2.5'
    }
  },
  
  // Cards
  card: {
    base: 'bg-primary-800 border border-primary-700 rounded-xl',
    padding: 'p-6',
    hover: 'hover:border-primary-600 transition-colors duration-200'
  },
  
  // Tables
  table: {
    container: 'overflow-hidden rounded-xl border border-primary-700 bg-primary-800',
    wrapper: 'overflow-x-auto',
    base: 'w-full text-left',
    header: 'bg-primary-900 border-b border-primary-700',
    headerCell: 'px-6 py-4 text-xs font-medium text-primary-400 uppercase tracking-wider',
    body: 'divide-y divide-primary-700',
    row: 'hover:bg-primary-700 transition-colors duration-150',
    cell: 'px-6 py-4 text-sm text-primary-300',
    cellPrimary: 'px-6 py-4 text-sm font-medium text-primary-50'
  },
  
  // Forms
  form: {
    label: 'block text-sm font-medium text-primary-300 mb-2',
    input: 'w-64 px-4 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 placeholder-primary-400 focus:outline-none focus:ring-2 focus:ring-accent-500 focus:border-transparent transition-all duration-200',
    select: 'w-full px-2 py-2 bg-primary-900 border border-primary-600 rounded-lg text-primary-50 focus:outline-none focus:ring-2 focus:ring-accent-500 focus:border-transparent transition-all duration-200',
    error: 'mt-1 text-sm text-red-400',
    help: 'mt-1 text-sm text-primary-400',
    helper: 'w-full bg-primary-900 text-primary-50 focus:outline-none'
  },
  
  // Badges
  badge: {
    base: 'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
    success: 'bg-green-900/30 text-green-400 border border-green-800',
    warning: 'bg-yellow-900/30 text-yellow-400 border border-yellow-800',
    error: 'bg-red-900/30 text-red-400 border border-red-800',
    info: 'bg-blue-900/30 text-blue-400 border border-blue-800',
    default: 'bg-primary-700 text-primary-300 border border-primary-600'
  },
  
  // Navigation
  nav: {
    item: 'flex items-center gap-3 px-4 py-3 rounded-lg text-primary-300 hover:text-primary-50 hover:bg-primary-700 transition-all duration-200',
    itemActive: 'bg-accent-500/10 text-accent-500 border-l-2 border-accent-500',
    icon: 'w-5 h-5 flex-shrink-0'
  },
  
  // Modals
  modal: {
    backdrop: 'fixed inset-0 bg-black/60 backdrop-blur-sm z-50 flex items-center justify-center p-4',
    content: 'bg-primary-800 rounded-xl border border-primary-700 max-w-lg w-full max-h-[90vh] overflow-y-auto',
    header: 'px-6 py-4 border-b border-primary-700',
    body: 'px-6 py-4',
    footer: 'px-6 py-4 border-t border-primary-700 flex justify-end gap-3'
  },
  
  // Alerts
  alert: {
    base: 'px-4 py-3 rounded-lg border',
    success: 'bg-green-900/20 border-green-800 text-green-400',
    warning: 'bg-yellow-900/20 border-yellow-800 text-yellow-400',
    error: 'bg-red-900/20 border-red-800 text-red-400',
    info: 'bg-blue-900/20 border-blue-800 text-blue-400'
  },
  
  // Utility
  text: {
    h1: 'text-3xl font-bold text-primary-50',
    h2: 'text-2xl font-semibold text-primary-50',
    h3: 'text-xl font-semibold text-primary-50',
    h4: 'text-lg font-medium text-primary-50',
    body: 'text-primary-300',
    muted: 'text-primary-400',
    small: 'text-sm text-primary-400'
  },
  
  // Spacing
  section: 'space-y-6',
  stack: 'space-y-4',
  
  // Grid
  grid: {
    cols2: 'grid grid-cols-1 md:grid-cols-2 gap-6',
    cols3: 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6',
    cols4: 'grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6'
  }
};

/**
 * Helper function to combine style classes
 */
export function cn(...classes: (string | undefined | null | false)[]): string {
  return classes.filter(Boolean).join(' ');
}