import { GitProviderListResponse } from '@/types/gitProvider'
import { api } from '@/services/api'

class GitProviderClient {
    async getGitProviders(): Promise<GitProviderListResponse> {
        const response = await api.get('/git-providers')
        return response.data
    }
}

export const gitProviderClient = new GitProviderClient()
