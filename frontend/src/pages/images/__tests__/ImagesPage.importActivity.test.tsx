import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import ImagesPage from '../ImagesPage'

const useCapabilitySurfacesStoreMock = vi.fn()
const searchImagesMock = vi.fn()
const getPopularImagesMock = vi.fn()
const getRecentImagesMock = vi.fn()
const buildSearchFiltersMock = vi.fn()

vi.mock('@/store/capabilitySurfaces', () => ({
    useCapabilitySurfacesStore: (selector?: any) => {
        const state = useCapabilitySurfacesStoreMock()
        return typeof selector === 'function' ? selector(state) : state
    },
}))

vi.mock('@/utils/permissions', () => ({
    canCreateImages: () => false,
}))

vi.mock('@/services/imageService', () => ({
    imageService: {
        searchImages: (...args: any[]) => searchImagesMock(...args),
        getPopularImages: (...args: any[]) => getPopularImagesMock(...args),
        getRecentImages: (...args: any[]) => getRecentImagesMock(...args),
        buildSearchFilters: (...args: any[]) => buildSearchFiltersMock(...args),
    },
}))

describe('ImagesPage quarantine workspace shortcut', () => {
    beforeEach(() => {
        vi.clearAllMocks()
        buildSearchFiltersMock.mockReturnValue({})
        searchImagesMock.mockResolvedValue({ images: [] })
        getPopularImagesMock.mockResolvedValue([])
        getRecentImagesMock.mockResolvedValue([])
        useCapabilitySurfacesStoreMock.mockReturnValue({
            canViewNavKey: (key: string) => key === 'quarantine_requests',
            canRunActionKey: (key: string) => key === 'quarantine.request.create',
        })
    })

    it('shows dedicated quarantine workspace call-to-action', async () => {
        render(
            <MemoryRouter initialEntries={['/images']}>
                <ImagesPage />
            </MemoryRouter>
        )

        expect(await screen.findByText('Quarantine Requests')).toBeInTheDocument()
        expect(screen.getByRole('link', { name: 'Open Quarantine Requests' })).toHaveAttribute('href', '/quarantine/requests')
    })

    it('uses admin quarantine workspace path when rendered in admin context', async () => {
        render(
            <MemoryRouter initialEntries={['/admin/images']}>
                <ImagesPage />
            </MemoryRouter>
        )

        expect(await screen.findByText('Quarantine Requests')).toBeInTheDocument()
        expect(screen.getByRole('link', { name: 'Open Quarantine Requests' })).toHaveAttribute('href', '/admin/quarantine/requests')
    })
})
