import { api } from './api'

export interface AcceptInvitationRequest {
    token: string
    first_name: string
    last_name: string
    password: string
    is_ldap?: boolean // Optional flag indicating if this is an LDAP user
}

export interface AcceptInvitationResponse {
    success: boolean
    message: string
}

class InvitationService {
    async acceptInvitation(data: AcceptInvitationRequest): Promise<AcceptInvitationResponse> {
        const response = await api.post('/invitations/accept', data)
        return response.data
    }
}

export const invitationService = new InvitationService()