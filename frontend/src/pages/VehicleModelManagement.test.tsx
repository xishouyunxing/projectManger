import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { Mock } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import VehicleModelManagement from './VehicleModelManagement'
import api from '../services/api'

vi.mock('../services/api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../services/api')>()
  return {
    ...actual,
    default: {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      delete: vi.fn(),
    },
  }
})

const mockApiGet = api.get as Mock
const mockApiDelete = api.delete as Mock

const renderPage = () =>
  render(
    <MemoryRouter>
      <VehicleModelManagement />
    </MemoryRouter>
  )

describe('VehicleModelManagement pagination', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    let deletedOnPage2 = false
    mockApiGet.mockImplementation((url: string, config?: { params?: { page?: number; page_size?: number; series?: string } }) => {
      if (url === '/vehicle-models') {
        const page = config?.params?.page ?? 1
        const pageSize = config?.params?.page_size ?? 20
        return Promise.resolve({
          data: {
            items: deletedOnPage2 && page > 1
              ? []
              : [
                  {
                    id: page === 1 ? 1 : 2,
                    name: page === 1 ? 'Model A' : 'Model B',
                    code: page === 1 ? 'MA' : 'MB',
                    series: 'S1',
                    description: 'desc',
                    created_at: '2026-01-01T08:00:00.000Z',
                  },
                ],
            total: deletedOnPage2 ? 20 : 40,
            page,
            page_size: pageSize,
          },
        })
      }

      if (url === '/production-lines') {
        return Promise.resolve({ data: [{ id: 1, name: 'Line A' }] })
      }

      return Promise.reject(new Error(`Unhandled GET ${url}`))
    })

    mockApiDelete.mockImplementation(() => {
      deletedOnPage2 = true
      return Promise.resolve({ data: { message: '删除成功' } })
    })
  })

  it('requests paged vehicle models on first load and keeps backend total text', async () => {
    renderPage()

    await screen.findByRole('heading', { name: '车型管理' })
    await screen.findByText('Model A')

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/vehicle-models', {
        params: { page: 1, page_size: 20 },
      })
    })

    expect(screen.getByText('显示第 1 至 20 条，共 40 条记录')).toBeInTheDocument()
  })

  it('requests next page params when table page changes', async () => {
    renderPage()

    await screen.findByText('Model A')
    fireEvent.click(screen.getByTitle('2'))

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/vehicle-models', {
        params: { page: 2, page_size: 20 },
      })
    })

    await screen.findByText('Model B')
  })

  it('resets pagination to first page when clicking reset', async () => {
    renderPage()

    await screen.findByText('Model A')
    fireEvent.click(screen.getByTitle('2'))

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/vehicle-models', {
        params: { page: 2, page_size: 20 },
      })
    })

    const callsBeforeReset = mockApiGet.mock.calls.length
    fireEvent.click(screen.getByRole('button', { name: '重置' }))

    await waitFor(() => {
      expect(mockApiGet.mock.calls.length).toBeGreaterThan(callsBeforeReset)
    })

    expect(mockApiGet.mock.calls.slice(callsBeforeReset)).toContainEqual([
      '/vehicle-models',
      { params: { page: 1, page_size: 20 } },
    ])
  })

  it('falls back to previous page when current page becomes empty after delete', async () => {
    renderPage()

    await screen.findByText('Model A')
    fireEvent.click(screen.getByTitle('2'))
    await screen.findByText('Model B')

    fireEvent.click(screen.getByRole('button', { name: 'delete' }))
    fireEvent.click(await screen.findByRole('button', { name: 'OK' }))

    await waitFor(() => {
      expect(mockApiDelete).toHaveBeenCalledWith('/vehicle-models/2')
    })

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/vehicle-models', {
        params: { page: 2, page_size: 20 },
      })
    })

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/vehicle-models', {
        params: { page: 1, page_size: 20 },
      })
    })
  })
})
