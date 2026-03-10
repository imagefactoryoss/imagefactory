import { create } from 'zustand'
import { persist } from 'zustand/middleware'

interface ThemeState {
    isDark: boolean
    toggleTheme: () => void
    initTheme: () => void
}

export const useThemeStore = create<ThemeState>()(
    persist(
        (set) => ({
            isDark: false,
            initTheme: () => {
                // Apply theme from localStorage on app init
                const isDark = localStorage.getItem('theme-storage')
                    ? JSON.parse(localStorage.getItem('theme-storage')!).state?.isDark ?? false
                    : false

                if (isDark) {
                    document.documentElement.classList.add('dark')
                } else {
                    document.documentElement.classList.remove('dark')
                }
                set({ isDark })
            },
            toggleTheme: () => {
                set((state) => {
                    const newIsDark = !state.isDark
                    // Apply class to document root
                    if (newIsDark) {
                        document.documentElement.classList.add('dark')
                        localStorage.setItem('theme-storage', JSON.stringify({
                            state: { isDark: true }
                        }))
                    } else {
                        document.documentElement.classList.remove('dark')
                        localStorage.setItem('theme-storage', JSON.stringify({
                            state: { isDark: false }
                        }))
                    }
                    return { isDark: newIsDark }
                })
            },
        }),
        {
            name: 'theme-storage',
        }
    )
)
