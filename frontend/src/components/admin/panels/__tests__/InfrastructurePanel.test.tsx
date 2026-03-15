import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { InfrastructurePanel } from '../InfrastructurePanel';

// Mock AuthContext using alias path
vi.mock('@/context/AuthContext', () => ({
    useAuth: () => ({
        user: {
            token: 'test-token-123',
            id: 'user-1',
            email: 'test@example.com',
        },
    }),
}));

// Mock fetch globally
global.fetch = vi.fn();

const mockNode = {
    id: 'node-uuid-1',
    tenant_id: 'tenant-1',
    name: 'worker-1',
    status: 'ready',
    total_cpu_cores: 8,
    total_memory_gb: 32,
    total_disk_gb: 500,
    used_cpu_cores: 4,
    used_memory_gb: 16,
    used_disk_gb: 250,
    last_heartbeat: '2024-01-01T12:00:00Z',
    maintenance_mode: false,
};

const mockNodesResponse = {
    items: [mockNode],
    total: 5,
    limit: 20,
    offset: 0,
    has_more: false,
};

const mockHealthResponse = {
    total_nodes: 5,
    healthy_nodes: 4,
    offline_nodes: 1,
    maintenance_nodes: 0,
    total_cpu_cores: 40,
    used_cpu_cores: 18,
    total_memory_gb: 160,
    used_memory_gb: 72,
    total_disk_gb: 2500,
    used_disk_gb: 1100,
};

describe('InfrastructurePanel', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        window.confirm = vi.fn(() => true);

        (global.fetch as any).mockImplementation((url: string) => {
            if (url.includes('/health')) {
                return Promise.resolve({
                    ok: true,
                    json: async () => mockHealthResponse,
                });
            }
            return Promise.resolve({
                ok: true,
                json: async () => mockNodesResponse,
            });
        });
    });

    describe('Rendering', () => {
        it('renders component with header', async () => {
            render(<InfrastructurePanel />);
            expect(screen.getByText('Infrastructure Nodes')).toBeInTheDocument();
        });

        it('displays total node count', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText(/Total Nodes: 5/)).toBeInTheDocument();
            });
        });

        it('renders nodes table with correct columns', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('Node Name')).toBeInTheDocument();
                expect(screen.getByText('Status')).toBeInTheDocument();
                expect(screen.getByText('CPU')).toBeInTheDocument();
            });
        });

        it('displays infrastructure nodes in table', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('worker-1')).toBeInTheDocument();
            });
        });

        it('shows empty state when no nodes', async () => {
            (global.fetch as any).mockResolvedValue({
                ok: true,
                json: async () => ({ items: [], total: 0, limit: 20, offset: 0, has_more: false }),
            });
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('No infrastructure nodes')).toBeInTheDocument();
            });
        });

        it('displays health metrics', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText(/Healthy Nodes: 4/)).toBeInTheDocument();
            });
        });
    });

    describe('Data Fetching', () => {
        it('fetches nodes on component mount', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(global.fetch).toHaveBeenCalledWith(
                    expect.stringContaining('/api/v1/admin/infrastructure/nodes'),
                    expect.any(Object)
                );
            });
        });

        it('fetches health metrics on mount', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(global.fetch).toHaveBeenCalledWith(
                    expect.stringContaining('/api/v1/admin/infrastructure/health'),
                    expect.any(Object)
                );
            });
        });

        it('includes bearer token in requests', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(global.fetch).toHaveBeenCalledWith(
                    expect.any(String),
                    expect.objectContaining({
                        headers: expect.objectContaining({
                            Authorization: 'Bearer test-token-123',
                        }),
                    })
                );
            });
        });
    });

    describe('CRUD Operations - Create', () => {
        it('opens add node form', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('Add Node')).toBeInTheDocument();
            });
            const addButton = screen.getByText('Add Node');
            fireEvent.click(addButton);
            await waitFor(() => {
                expect(screen.getByText('Add New Node')).toBeInTheDocument();
            });
        });

        it('creates new node with valid data', async () => {
            (global.fetch as any).mockImplementation((url: string) => {
                if (url.includes('/nodes') && url.includes('POST')) {
                    return Promise.resolve({
                        ok: true,
                        json: async () => ({ ...mockNode, id: 'new-node' }),
                    });
                }
                if (url.includes('/health')) {
                    return Promise.resolve({
                        ok: true,
                        json: async () => mockHealthResponse,
                    });
                }
                return Promise.resolve({
                    ok: true,
                    json: async () => mockNodesResponse,
                });
            });
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('Add Node')).toBeInTheDocument();
            });
            const addButton = screen.getByText('Add Node');
            fireEvent.click(addButton);
            await waitFor(() => {
                const nameInput = screen.getByLabelText(/node name/i) as HTMLInputElement;
                fireEvent.change(nameInput, { target: { value: 'new-worker' } });
            });
            const submitButton = screen.getByText('Create Node');
            fireEvent.click(submitButton);
            await waitFor(() => {
                expect(global.fetch).toHaveBeenCalledWith(
                    expect.stringContaining('/api/v1/admin/infrastructure/nodes'),
                    expect.any(Object)
                );
            });
        });

        it('validates required fields', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('Add Node')).toBeInTheDocument();
            });
            const addButton = screen.getByText('Add Node');
            fireEvent.click(addButton);
            await waitFor(() => {
                const submitButton = screen.getByText('Create Node');
                fireEvent.click(submitButton);
            });
            await waitFor(() => {
                expect(screen.getByText(/Node name is required/)).toBeInTheDocument();
            });
        });
    });

    describe('CRUD Operations - Update', () => {
        it('opens edit form with pre-filled values', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('worker-1')).toBeInTheDocument();
            });
            const editButton = screen.getAllByText('Edit')[0];
            fireEvent.click(editButton);
            await waitFor(() => {
                const nameInput = screen.getByDisplayValue('worker-1') as HTMLInputElement;
                expect(nameInput).toBeInTheDocument();
            });
        });

        it('updates node with new data', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('worker-1')).toBeInTheDocument();
            });
            const editButton = screen.getAllByText('Edit')[0];
            fireEvent.click(editButton);
            await waitFor(() => {
                const nameInput = screen.getByDisplayValue('worker-1') as HTMLInputElement;
                fireEvent.change(nameInput, { target: { value: 'updated-worker' } });
            });
            const submitButton = screen.getByText('Update Node');
            fireEvent.click(submitButton);
            await waitFor(() => {
                expect(global.fetch).toHaveBeenCalledWith(
                    expect.stringContaining(`/api/v1/admin/infrastructure/nodes/${mockNode.id}`),
                    expect.any(Object)
                );
            });
        });
    });

    describe('CRUD Operations - Delete', () => {
        it('shows confirmation dialog on delete', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('worker-1')).toBeInTheDocument();
            });
            const deleteButton = screen.getAllByText('Delete')[0];
            fireEvent.click(deleteButton);
            await waitFor(() => {
                expect(window.confirm).toHaveBeenCalled();
            });
        });

        it('deletes node on confirmation', async () => {
            (window.confirm as any).mockReturnValue(true);
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('worker-1')).toBeInTheDocument();
            });
            const initialCallCount = (global.fetch as any).mock.calls.length;
            const deleteButton = screen.getAllByText('Delete')[0];
            fireEvent.click(deleteButton);
            await waitFor(() => {
                expect((global.fetch as any).mock.calls.length).toBeGreaterThan(initialCallCount);
            });
        });

        it('does not delete on confirmation cancel', async () => {
            (window.confirm as any).mockReturnValue(false);
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('worker-1')).toBeInTheDocument();
            });
            const initialCallCount = (global.fetch as any).mock.calls.length;
            const deleteButton = screen.getAllByText('Delete')[0];
            fireEvent.click(deleteButton);
            await waitFor(() => {
                expect((global.fetch as any).mock.calls.length).toBe(initialCallCount);
            });
        });
    });

    describe('Form Management', () => {
        it('closes form on cancel', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('Add Node')).toBeInTheDocument();
            });
            const addButton = screen.getByText('Add Node');
            fireEvent.click(addButton);
            await waitFor(() => {
                expect(screen.getByText('Add New Node')).toBeInTheDocument();
            });
            const cancelButton = screen.getByText('Cancel');
            fireEvent.click(cancelButton);
            await waitFor(() => {
                expect(screen.queryByText('Add New Node')).not.toBeInTheDocument();
            });
        });

        it('resets form after successful submission', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('Add Node')).toBeInTheDocument();
            });
            const addButton = screen.getByText('Add Node');
            fireEvent.click(addButton);
            await waitFor(() => {
                const nameInput = screen.getByLabelText(/node name/i) as HTMLInputElement;
                fireEvent.change(nameInput, { target: { value: 'new-worker' } });
            });
            const submitButton = screen.getByText('Create Node');
            fireEvent.click(submitButton);
            await waitFor(() => {
                expect(screen.queryByText('Add New Node')).not.toBeInTheDocument();
            });
        });
    });

    describe('Pagination', () => {
        it('displays pagination controls', async () => {
            (global.fetch as any).mockImplementation((url: string) => {
                if (url.includes('/health')) {
                    return Promise.resolve({
                        ok: true,
                        json: async () => mockHealthResponse,
                    });
                }
                return Promise.resolve({
                    ok: true,
                    json: async () => ({ ...mockNodesResponse, has_more: true }),
                });
            });
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('Next')).toBeInTheDocument();
            });
        });

        it('disables previous button on first page', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                const prevButton = screen.getByText('Previous') as HTMLButtonElement;
                expect(prevButton.disabled).toBe(true);
            });
        });

        it('disables next button on last page', async () => {
            (global.fetch as any).mockImplementation((url: string) => {
                if (url.includes('/health')) {
                    return Promise.resolve({
                        ok: true,
                        json: async () => mockHealthResponse,
                    });
                }
                return Promise.resolve({
                    ok: true,
                    json: async () => ({ ...mockNodesResponse, has_more: false }),
                });
            });
            render(<InfrastructurePanel />);
            await waitFor(() => {
                const nextButton = screen.getByText('Next') as HTMLButtonElement;
                expect(nextButton.disabled).toBe(true);
            });
        });
    });

    describe('Auto-Refresh', () => {
        it('auto-refresh is enabled by default', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                const autoRefreshToggle = screen.getByLabelText(/auto-refresh/i) as HTMLInputElement;
                expect(autoRefreshToggle.checked).toBe(true);
            });
        });

        it('can disable auto-refresh', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                const autoRefreshToggle = screen.getByLabelText(/auto-refresh/i) as HTMLInputElement;
                fireEvent.click(autoRefreshToggle);
                expect(autoRefreshToggle.checked).toBe(false);
            });
        });

        it('manual refresh button works', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText('Refresh')).toBeInTheDocument();
            });
            const refreshButton = screen.getByText('Refresh');
            const initialCallCount = (global.fetch as any).mock.calls.length;
            fireEvent.click(refreshButton);
            await waitFor(() => {
                expect((global.fetch as any).mock.calls.length).toBeGreaterThan(initialCallCount);
            });
        });
    });

    describe('Status Color Coding', () => {
        it('displays ready status with green background', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                const statusElement = screen.getByText('ready');
                expect(statusElement).toHaveClass('bg-green-600');
            });
        });

        it('displays offline status with red background', async () => {
            const offlineNode = { ...mockNode, status: 'offline' };
            (global.fetch as any).mockImplementation((url: string) => {
                if (url.includes('/health')) {
                    return Promise.resolve({
                        ok: true,
                        json: async () => mockHealthResponse,
                    });
                }
                return Promise.resolve({
                    ok: true,
                    json: async () => ({ ...mockNodesResponse, items: [offlineNode] }),
                });
            });
            render(<InfrastructurePanel />);
            await waitFor(() => {
                const statusElement = screen.getByText('offline');
                expect(statusElement).toHaveClass('bg-red-600');
            });
        });
    });

    describe('Resource Display', () => {
        it('displays CPU usage with progress bar', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText(/4 \/ 8 cores/)).toBeInTheDocument();
            });
        });

        it('displays memory usage with progress bar', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText(/16 \/ 32 GB/)).toBeInTheDocument();
            });
        });

        it('displays disk usage with progress bar', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText(/250 \/ 500 GB/)).toBeInTheDocument();
            });
        });
    });

    describe('Health Metrics', () => {
        it('displays total healthy nodes', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText(/Healthy Nodes: 4/)).toBeInTheDocument();
            });
        });

        it('displays system CPU usage', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText(/System CPU: 18 \/ 40 cores/)).toBeInTheDocument();
            });
        });

        it('displays system memory usage', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText(/System Memory: 72 \/ 160 GB/)).toBeInTheDocument();
            });
        });
    });

    describe('Resource Percentage Calculations', () => {
        it('calculates CPU percentage correctly', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText(/50%/)).toBeInTheDocument();
            });
        });

        it('calculates memory percentage correctly', async () => {
            render(<InfrastructurePanel />);
            await waitFor(() => {
                expect(screen.getByText(/50%/)).toBeInTheDocument();
            });
        });
    });
});
