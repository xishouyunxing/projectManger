import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { Mock } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import DepartmentManagement from './DepartmentManagement'
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
      <DepartmentManagement />
    </MemoryRouter>
  )

describe('DepartmentManagement pagination', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    let deletedOnPage2 = false
    mockApiGet.mockImplementation((url: string, config?: { params?: { page?: number; page_size?: number } }) => {
      if (url === '/departments') {
        const page = config?.params?.page ?? 1
        const pageSize = config?.params?.page_size ?? 20
        return Promise.resolve({
          data: {
            items: deletedOnPage2 && page > 1 ? [] : [
              {
                id: page === 1 ? 1 : 2,
                name: page === 1 ? '总装部' : '焊装部',
                description: 'dept desc',
                status: 'active',
                created_at: '2026-01-01T08:00:00.000Z',
              },
            ],
            total: deletedOnPage2 ? 20 : 21,
            page,
            page_size: pageSize,
          },
        })
      }

      return Promise.reject(new Error(`Unhandled GET ${url}`))
    })

    mockApiDelete.mockImplementation(() => {
      deletedOnPage2 = true
      return Promise.resolve({ data: { message: '删除成功' } })
    })
  })

  it('requests paged departments on first load and keeps backend total text', async () => {
    renderPage()

    await screen.findByRole('heading', { name: '部门管理' })
    await screen.findByText('总装部')

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/departments', {
        params: { page: 1, page_size: 20 },
      })
    })

    expect(screen.getByText('显示第 1 至 20 条，共 21 条记录')).toBeInTheDocument()
  })

  it('falls back to previous page when current page becomes empty after delete', async () => {
    renderPage()

    await screen.findByText('总装部')
    fireEvent.click(screen.getByTitle('2'))
    await screen.findByText('焊装部')

    fireEvent.click(screen.getByRole('button', { name: 'delete删除' }))
    fireEvent.click(await screen.findByRole('button', { name: 'OK' }))

    await waitFor(() => {
      expect(mockApiDelete).toHaveBeenCalledWith('/departments/2')
    })

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/departments', {
        params: { page: 2, page_size: 20 },
      })
    })

    await waitFor(() => {
      expect(mockApiGet).toHaveBeenCalledWith('/departments', {
        params: { page: 1, page_size: 20 },
      })
    })
  })
})
