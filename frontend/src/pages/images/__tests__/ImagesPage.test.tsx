import { render, screen, waitFor } from '@testing-library/react'
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

describe('ImagesPage capability visibility', () => {
    beforeEach(() => {
        vi.clearAllMocks()
        buildSearchFiltersMock.mockReturnValue({})
        searchImagesMock.mockResolvedValue({ images: [] })
        getPopularImagesMock.mockResolvedValue([])
        getRecentImagesMock.mockResolvedValue([])
    })

    it('hides quarantine workspace shortcut when tenant is not entitled', async () => {
        useCapabilitySurfacesStoreMock.mockReturnValue({
            canViewNavKey: () => false,
            canRunActionKey: () => false,
        })

        render(
            <MemoryRouter initialEntries={['/images']}>
                <ImagesPage />
            </MemoryRouter>
        )

        await waitFor(() => {
            expect(searchImagesMock).toHaveBeenCalledTimes(1)
            expect(getPopularImagesMock).toHaveBeenCalledWith(5)
            expect(getRecentImagesMock).toHaveBeenCalledWith(5)
        })

        expect(screen.queryByText('Quarantine Requests')).not.toBeInTheDocument()
        expect(screen.queryByRole('link', { name: 'Open Quarantine Requests' })).not.toBeInTheDocument()
        expect(await screen.findByText('No popular images')).toBeInTheDocument()
    })

    it('shows shortcut to dedicated quarantine workspace when tenant is entitled', async () => {
        useCapabilitySurfacesStoreMock.mockReturnValue({
            canViewNavKey: (key: string) => key === 'quarantine_requests',
            canRunActionKey: (key: string) => key === 'quarantine.request.create',
        })

        render(
            <MemoryRouter initialEntries={['/images']}>
                <ImagesPage />
            </MemoryRouter>
        )

        expect(await screen.findByText('Quarantine Requests')).toBeInTheDocument()
        expect(screen.getByRole('link', { name: 'Open Quarantine Requests' })).toHaveAttribute('href', '/quarantine/requests')
    })
})
