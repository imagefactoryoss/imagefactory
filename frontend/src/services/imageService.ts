import type {
    ApiResponse,
    ImageDetailsResponse,
    CreateImageRequest,
    Image,
    ImageListResponse,
    ImageSearchFilters,
    ImageStats,
    ImageTagListResponse,
    ImageVersion,
    ImageVersionsResponse,
    PopularImagesResponse,
    RecentImagesResponse,
    SearchImagesRequest,
    UpdateImageRequest
} from '../types'
import { api } from './api'

export interface OnDemandScanResponse {
    scan_request_id: string
    image_id: string
    status: 'queued'
    message: string
}

class ImageService {
    // Image CRUD operations
    async createImage(data: CreateImageRequest): Promise<Image> {
        const response = await api.post<ApiResponse<Image>>('/images', data)
        return response.data.data
    }

    async getImage(id: string): Promise<Image> {
        const response = await api.get<ApiResponse<Image>>(`/images/${id}`)
        return response.data.data
    }

    async updateImage(id: string, data: UpdateImageRequest): Promise<Image> {
        const response = await api.put<ApiResponse<Image>>(`/images/${id}`, data)
        return response.data.data
    }

    async deleteImage(id: string): Promise<void> {
        await api.delete(`/images/${id}`)
    }

    // Image listing and search
    async searchImages(params: SearchImagesRequest = {}): Promise<ImageListResponse> {
        const queryParams = new URLSearchParams()

        if (params.query) queryParams.append('query', params.query)
        if (params.status) queryParams.append('status', params.status)
        if (params.registry_provider) queryParams.append('registry_provider', params.registry_provider)
        if (params.architecture) queryParams.append('architecture', params.architecture)
        if (params.os) queryParams.append('os', params.os)
        if (params.language) queryParams.append('language', params.language)
        if (params.framework) queryParams.append('framework', params.framework)
        if (params.tags?.length) queryParams.append('tags', params.tags.join(','))
        if (params.sort_by) queryParams.append('sort_by', params.sort_by)
        if (params.sort_order) queryParams.append('sort_order', params.sort_order)
        if (params.limit) queryParams.append('limit', params.limit.toString())
        if (params.offset) queryParams.append('offset', params.offset.toString())

        const queryString = queryParams.toString()
        const url = `/images/search${queryString ? `?${queryString}` : ''}`

        const response = await api.get<ImageListResponse>(url)
        return response.data
    }

    async getPopularImages(limit: number = 10): Promise<Image[]> {
        const response = await api.get<PopularImagesResponse>(`/images/popular?limit=${limit}`)
        return response.data.images
    }

    async getRecentImages(limit: number = 10): Promise<Image[]> {
        const response = await api.get<RecentImagesResponse>(`/images/recent?limit=${limit}`)
        return response.data.images
    }

    // Image versions
    async getImageVersions(imageId: string): Promise<ImageVersion[]> {
        const response = await api.get<ApiResponse<ImageVersionsResponse>>(`/images/${imageId}/versions`)
        return response.data.data.versions
    }

    async createImageVersion(imageId: string, version: string, data: any): Promise<ImageVersion> {
        const response = await api.post<ApiResponse<ImageVersion>>(`/images/${imageId}/versions`, {
            version,
            ...data
        })
        return response.data.data
    }

    // Image tags
    async getImageTags(imageId: string): Promise<string[]> {
        const response = await api.get<ApiResponse<ImageTagListResponse>>(`/images/${imageId}/tags`)
        return response.data.data.tags
    }

    async addImageTags(imageId: string, tags: string[]): Promise<void> {
        await api.post(`/images/${imageId}/tags`, { tags })
    }

    async removeImageTags(imageId: string, tags: string[]): Promise<void> {
        await api.delete(`/images/${imageId}/tags`, { data: { tags } })
    }

    // Utility methods for filtering
    buildSearchFilters(filters: ImageSearchFilters): SearchImagesRequest {
        const params: SearchImagesRequest = {}

        if (filters.query) params.query = filters.query
        if (filters.status?.length) params.status = filters.status[0] // API expects single status
        if (filters.registry_provider?.length) params.registry_provider = filters.registry_provider[0]
        if (filters.architecture?.length) params.architecture = filters.architecture[0]
        if (filters.os?.length) params.os = filters.os[0]
        if (filters.language?.length) params.language = filters.language[0]
        if (filters.framework?.length) params.framework = filters.framework[0]
        if (filters.tags?.length) params.tags = filters.tags

        return params
    }

    // Image statistics
    async getImageStats(imageId: string): Promise<ImageStats> {
        const response = await api.get<ApiResponse<ImageStats>>(`/images/${imageId}/stats`)
        return response.data.data
    }

    async getImageDetails(imageId: string): Promise<ImageDetailsResponse> {
        const response = await api.get<ApiResponse<ImageDetailsResponse>>(`/images/${imageId}/details`)
        return response.data.data
    }

    async triggerOnDemandScan(imageId: string): Promise<OnDemandScanResponse> {
        const response = await api.post<ApiResponse<OnDemandScanResponse>>(`/images/${imageId}/scan`)
        return response.data.data
    }
}

export const imageService = new ImageService()
