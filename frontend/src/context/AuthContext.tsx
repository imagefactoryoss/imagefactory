import React, { createContext, useContext, useEffect, useState } from 'react'

interface User {
    id: string
    email: string
    name: string
    role: string
    tenantId?: string
}

interface AuthContextType {
    user: User | null
    isAuthenticated: boolean
    login: (email: string, password: string) => Promise<void>
    logout: () => void
    isLoading: boolean
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export const useAuth = () => {
    const context = useContext(AuthContext)
    if (context === undefined) {
        throw new Error('useAuth must be used within an AuthProvider')
    }
    return context
}

interface AuthProviderProps {
    children: React.ReactNode
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
    const [user, setUser] = useState<User | null>(null)
    const [isLoading, setIsLoading] = useState(true)

    useEffect(() => {
        // Check for existing session
        const checkAuth = async () => {
            try {
                // TODO: Implement actual auth check
                const storedUser = localStorage.getItem('user')
                if (storedUser) {
                    setUser(JSON.parse(storedUser))
                }
            } catch (error) {
            } finally {
                setIsLoading(false)
            }
        }

        checkAuth()
    }, [])

    const login = async (email: string, _password: string) => {
        setIsLoading(true)
        try {
            // TODO: Implement actual login
            const mockUser: User = {
                id: '1',
                email,
                name: 'Test User',
                role: 'admin'
            }
            setUser(mockUser)
            localStorage.setItem('user', JSON.stringify(mockUser))
        } catch (error) {
            throw error
        } finally {
            setIsLoading(false)
        }
    }

    const logout = () => {
        setUser(null)
        localStorage.removeItem('user')
    }

    const value: AuthContextType = {
        user,
        isAuthenticated: !!user,
        login,
        logout,
        isLoading
    }

    return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}