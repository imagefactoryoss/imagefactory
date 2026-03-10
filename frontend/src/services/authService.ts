import {
    CreateSSOProviderRequest,
    LoginForm,
    LoginResponse,
    MFAChallengeResponse,
    MFASetup,
    MFASetupRequest,
    RefreshTokenRequest,
    SSOProvider,
    TestConnectionRequest,
    UpdateSSOProviderRequest,
    VerifyMFARequest,
    VerifyMFAResponse,
} from '@/types';
import api, { getErrorMessage } from './api';

// AuthService interface
interface AuthService {
    login(form: LoginForm): Promise<LoginResponse>
    getLoginOptions(): Promise<{ ldap_enabled: boolean }>
    getBootstrapStatus(): Promise<{ setup_required: boolean }>
    refreshToken(request: RefreshTokenRequest): Promise<{ access_token: string; refresh_token: string; access_token_expiry?: number }>
    changePassword(currentPassword: string, newPassword: string): Promise<void>
    logout(): Promise<void>
    getProfile(): Promise<any>
    updateProfile(data: Partial<any>): Promise<any>
    requestPasswordReset(email: string): Promise<{ message: string }>
    resetPassword(token: string, password: string): Promise<{ message: string }>
    validateResetToken(token: string): Promise<{ valid: boolean; email?: string; expiresAt?: string }>
}

// Authentication API service
export const authService: AuthService = {
    async getLoginOptions(): Promise<{ ldap_enabled: boolean }> {
        try {
            const response = await api.get('/auth/login-options')
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async getBootstrapStatus(): Promise<{ setup_required: boolean }> {
        try {
            const response = await api.get('/bootstrap/status')
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Login with email/password
    async login(form: LoginForm): Promise<LoginResponse> {
        try {
            const response = await api.post('/auth/login', form)
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Refresh access token
    async refreshToken(request: RefreshTokenRequest): Promise<{ access_token: string; refresh_token: string; access_token_expiry?: number }> {
        try {
            const response = await api.post('/auth/refresh', request)
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Change password
    async changePassword(currentPassword: string, newPassword: string): Promise<void> {
        try {
            await api.post('/auth/change-password', {
                current_password: currentPassword,
                new_password: newPassword,
            })
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Logout
    async logout(): Promise<void> {
        try {
            await api.post('/auth/logout')
        } catch (error: any) {
            // Even if logout fails on server, we should clear local state
        }
    },

    // Get current user profile
    async getProfile() {
        try {
            const response = await api.get('/auth/me')
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Update user profile
    async updateProfile(data: Partial<any>) {
        try {
            const response = await api.patch('/auth/profile', data)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Password Reset
    async requestPasswordReset(email: string): Promise<{ message: string }> {
        try {
            const response = await api.post('/auth/forgot-password', { email })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async resetPassword(token: string, password: string): Promise<{ message: string }> {
        try {
            const response = await api.post('/auth/reset-password', { token, password })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    async validateResetToken(token: string): Promise<{ valid: boolean; email?: string; expiresAt?: string }> {
        try {
            const response = await api.post('/auth/validate-reset-token', { token })
            return response.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },
}

// MFA Service
export const mfaService = {
    // Setup MFA
    async setupMFA(request: MFASetupRequest): Promise<MFASetup> {
        try {
            const response = await api.post('/mfa/setup', request)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Confirm MFA setup
    async confirmMFASetup(challengeId: string, code: string): Promise<void> {
        try {
            await api.post('/mfa/confirm-setup', {
                challengeId,
                code,
            })
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Start MFA challenge
    async startMFAChallenge(userId?: string): Promise<MFAChallengeResponse> {
        try {
            const response = await api.post('/mfa/challenge', {
                userId,
            })
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Verify MFA challenge
    async verifyMFA(request: VerifyMFARequest): Promise<VerifyMFAResponse> {
        try {
            const response = await api.post('/mfa/verify', request)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Disable MFA
    async disableMFA(challengeId: string, code: string): Promise<void> {
        try {
            await api.post('/mfa/disable', {
                challengeId,
                code,
            })
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Get MFA status
    async getMFAStatus(userId?: string) {
        try {
            const response = await api.get(`/mfa/status${userId ? `?userId=${userId}` : ''}`)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },
}

// SSO Service
export const ssoService = {
    // List SSO providers
    async listProviders(): Promise<SSOProvider[]> {
        try {
            const response = await api.get('/sso/configuration')
            const config = response.data

            // Combine SAML and OIDC providers into a single array
            const providers: SSOProvider[] = []

            // Add SAML providers
            if (config.saml_providers) {
                config.saml_providers.forEach((provider: any) => {
                    providers.push({
                        id: provider.id,
                        type: 'saml',
                        name: provider.name,
                        enabled: provider.enabled !== false,
                        config: {
                            entityId: provider.entity_id,
                            singleSignOnURL: provider.sso_url,
                            singleLogoutURL: provider.slo_url,
                            certificate: provider.certificate,
                            attributeMapping: provider.attributes || {},
                            nameIdFormat: 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress',
                        },
                        metadata: provider.metadata ? { entityDescriptor: provider.metadata } : undefined,
                        createdAt: provider.created_at,
                        updatedAt: provider.updated_at,
                    })
                })
            }

            // Add OIDC providers
            if (config.oidc_providers) {
                config.oidc_providers.forEach((provider: any) => {
                    providers.push({
                        id: provider.id,
                        type: 'oidc',
                        name: provider.name,
                        enabled: provider.enabled !== false,
                        config: {
                            entityId: undefined,
                            singleSignOnURL: provider.authorization_url,
                            singleLogoutURL: undefined,
                            certificate: undefined,
                            attributeMapping: provider.attributes || {},
                            nameIdFormat: undefined,
                        },
                        createdAt: provider.created_at,
                        updatedAt: provider.updated_at,
                    })
                })
            }

            return providers
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Get SSO provider by ID
    async getProvider(id: string): Promise<SSOProvider> {
        try {
            const response = await api.get(`/sso/providers/${id}`)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Create SSO provider
    async createProvider(data: CreateSSOProviderRequest): Promise<SSOProvider> {
        try {
            const response = await api.post('/sso/providers', data)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Update SSO provider
    async updateProvider(id: string, data: UpdateSSOProviderRequest): Promise<SSOProvider> {
        try {
            const response = await api.patch(`/sso/providers/${id}`, data)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Delete SSO provider
    async deleteProvider(id: string): Promise<void> {
        try {
            await api.delete(`/sso/providers/${id}`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Test SSO connection
    async testConnection(request: TestConnectionRequest): Promise<{ success: boolean; message: string }> {
        try {
            const response = await api.post('/sso/test-connection', request)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Initiate SSO login
    initiateSSOLogin(providerId: string): void {
        window.location.href = `/api/auth/sso/${providerId}/initiate`
    },

    // Handle SSO callback
    async handleSSOCallback(code: string, state: string): Promise<LoginResponse> {
        try {
            const response = await api.post('/auth/sso/callback', { code, state })
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },
}

// System Config Service
export const systemConfigService = {
    // Get all configurations
    async getConfigs() {
        try {
            const response = await api.get('/system-config')
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Get configuration by ID
    async getConfig(id: string) {
        try {
            const response = await api.get(`/system-config/${id}`)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Create configuration
    async createConfig(data: any) {
        try {
            const response = await api.post('/system-config', data)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Update configuration
    async updateConfig(id: string, data: any) {
        try {
            const response = await api.patch(`/system-config/${id}`, data)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Delete configuration
    async deleteConfig(id: string): Promise<void> {
        try {
            await api.delete(`/system-config/${id}`)
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },

    // Test configuration
    async testConfig(id: string): Promise<{ success: boolean; message: string }> {
        try {
            const response = await api.post(`/system-config/${id}/test`)
            return response.data.data
        } catch (error: any) {
            throw new Error(getErrorMessage(error))
        }
    },
}
