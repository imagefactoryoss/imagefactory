import { RepositoryBranchRequest, RepositoryBranchResponse } from '@/types/repositoryBranch'
import { api } from '@/services/api'

class RepositoryBranchClient {
    async listBranches(projectId: string, payload: RepositoryBranchRequest): Promise<RepositoryBranchResponse> {
        const response = await api.post(`/projects/${projectId}/repository-branches`, payload)
        return response.data
    }
}

export const repositoryBranchClient = new RepositoryBranchClient()
