import { render, screen, fireEvent, waitFor, within } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { Mock } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import ProgramManagement from './ProgramManagement'
import api from '../services/api'

vi.mock('../services/api', () => ({
  default: {
    get: vi.fn(),
    post: vi.fn(),
    put: vi.fn(),
    delete: vi.fn(),
  },
}))

vi.mock('../contexts/AuthContext', () => ({
  useAuth: () => ({
    user: { id: 7, name: 'Current User' },
  }),
}))

const mockApiGet = api.get as Mock
const mockApiPost = api.post as Mock
const mockApiPut = api.put as Mock

const programsFixture = [
  {
    id: 101,
    name: 'Program Alpha',
    code: 'PA-01',
    version: 'v1.0.0',
    status: 'in_progress',
    production_line_id: 1,
    vehicle_model_id: 10,
    description: 'Existing program',
    created_at: '2026-01-01T08:00:00.000Z',
    editing_by: 7,
    editing_user: { id: 7, name: 'Current User' },
    production_line: { id: 1, name: 'Line A' },
    vehicle_model: { id: 10, name: 'Model X' },
    custom_field_values: [
      {
        field_id: 12,
        field_name: 'Shift',
        field_type: 'select',
        sort_order: 2,
        value: 'Night',
      },
      {
        field_id: 11,
        field_name: 'Operator Note',
        field_type: 'text',
        sort_order: 1,
        value: 'Saved note',
      },
    ],
  },
  {
    id: 102,
    name: 'Program Bravo',
    code: 'PB-01',
    version: 'v2.0.0',
    status: 'completed',
    production_line_id: 1,
    vehicle_model_id: 10,
    description: 'Second program',
    created_at: '2026-01-02T08:00:00.000Z',
    editing_by: null,
    editing_user: null,
    production_line: { id: 1, name: 'Line A' },
    vehicle_model: { id: 10, name: 'Model X' },
    custom_field_values: {
      '11': 'Other note',
      '12': 'Day',
    },
  },
]

const lineOneFields = [
  {
    id: 11,
    name: 'Operator Note',
    field_type: 'text',
    options_json: '',
    sort_order: 1,
    enabled: true,
  },
  {
    id: 12,
    name: 'Shift',
    field_type: 'select',
    options_json: '["Day","Night"]',
    sort_order: 2,
    enabled: true,
  },
  {
    id: 13,
    name: 'Disabled Field',
    field_type: 'text',
    options_json: '',
    sort_order: 3,
    enabled: false,
  },
]

const lineTwoFields = [
  {
    id: 21,
    name: 'Line B Status',
    field_type: 'text',
    options_json: '',
    sort_order: 1,
    enabled: true,
  },
]

const renderPage = () =>
  render(
    <MemoryRouter initialEntries={['/programs?keyword=Alpha&id=101']}>
      <ProgramManagement />
    </MemoryRouter>
  )

const findProgramRow = (programName: string) => screen.getByText(programName).closest('tr')

const getOpenModal = () => {
  const modalContents = Array.from(document.querySelectorAll('.ant-modal-content'))
  const modal = modalContents[modalContents.length - 1]

  if (!(modal instanceof HTMLElement)) {
    throw new Error('Open modal not found')
  }

  return modal
}

const selectLastMatchingOption = async (text: string) => {
  const options = await screen.findAllByText(text)
  fireEvent.click(options[options.length - 1])
}

const clickEditButtonForProgram = (programName: string) => {
  const row = screen.getByText(programName).closest('tr')
  const buttons = row?.querySelectorAll('button') ?? []

  if (!(buttons[2] instanceof HTMLButtonElement)) {
    throw new Error(`Edit button not found for ${programName}`)
  }

  fireEvent.click(buttons[2])
}

describe('ProgramManagement custom field integration', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    mockApiGet.mockImplementation((url: string) => {
      if (url === '/programs') {
        return Promise.resolve({ data: programsFixture })
      }

      if (url === '/production-lines') {
        return Promise.resolve({
          data: [
            { id: 1, name: 'Line A' },
            { id: 2, name: 'Line B' },
          ],
        })
      }

      if (url === '/vehicle-models') {
        return Promise.resolve({ data: [{ id: 10, name: 'Model X' }] })
      }

      if (url === '/production-lines/1/custom-fields') {
        return Promise.resolve({ data: lineOneFields })
      }

      if (url === '/production-lines/2/custom-fields') {
        return Promise.resolve({ data: lineTwoFields })
      }

      if (url === '/relations/related/101') {
        return Promise.resolve({ data: [] })
      }

      return Promise.resolve({ data: { versions: [] } })
    })

    mockApiPost.mockResolvedValue({ data: { id: 104 } })
    mockApiPut.mockResolvedValue({ data: {} })
  })

  it('keeps the current main-workspace page structure and renders sorted custom field tips', async () => {
    renderPage()

    expect(screen.getByRole('button', { name: /批量导入/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /导出Excel/ })).toBeInTheDocument()
    expect(screen.getByDisplayValue('Alpha')).toBeInTheDocument()
    expect(await screen.findByText('我正在编辑')).toBeInTheDocument()

    const alphaRow = findProgramRow('Program Alpha')
    const tagTexts = Array.from(alphaRow?.querySelectorAll('.ant-tag') ?? []).map((tag) =>
      tag.textContent?.trim()
    )

    expect(tagTexts).toContain('Saved note')
    expect(tagTexts).toContain('Night')
  })

  it('loads enabled custom field definitions and preserves existing values when editing', async () => {
    renderPage()

    await screen.findByText('Program Alpha')
    clickEditButtonForProgram('Program Alpha')

    await waitFor(() => {
      expect(mockApiGet.mock.calls.some(([url]) => url === '/production-lines/1/custom-fields')).toBe(true)
    })

    const modal = getOpenModal()
    expect(within(modal).getByText('Operator Note')).toBeInTheDocument()
    expect(within(modal).getByDisplayValue('Saved note')).toBeInTheDocument()
    expect(within(modal).getByText('Night')).toBeInTheDocument()
    expect(within(modal).queryByText('Disabled Field')).not.toBeInTheDocument()
  })

  it('saves custom field values after the base program update using the current backend payload contract', async () => {
    renderPage()

    await screen.findByText('Program Alpha')
    clickEditButtonForProgram('Program Alpha')

    const modal = getOpenModal()

    await waitFor(() => {
      expect(mockApiGet.mock.calls.some(([url]) => url === '/production-lines/1/custom-fields')).toBe(true)
    })

    const dynamicInput = document.querySelector('#custom_field_values_11') as HTMLInputElement
    fireEvent.change(dynamicInput, { target: { value: 'Updated custom note' } })

    fireEvent.mouseDown((modal.querySelectorAll('.ant-select-selector')[1]) as Element)
    await selectLastMatchingOption('Day')

    fireEvent.submit(modal.querySelector('form') as HTMLFormElement)

    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalledWith('/programs/101', {
        name: 'Program Alpha',
        code: 'PA-01',
        production_line_id: 1,
        vehicle_model_id: 10,
        status: 'in_progress',
        description: 'Existing program',
      })
    })

    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalledWith('/programs/101/custom-field-values', {
        values: [
          { field_id: 11, value: 'Updated custom note' },
          { field_id: 12, value: 'Day' },
        ],
      })
    })
  })

  it('loads and resets line-scoped custom field filters when the production line filter changes', async () => {
    renderPage()

    await screen.findByText('Program Alpha')
    expect(screen.queryByText('Operator Note')).not.toBeInTheDocument()

    const filterSelects = document.querySelectorAll('.management-filter-panel .ant-select-selector')
    fireEvent.mouseDown(filterSelects[0] as Element)
    await selectLastMatchingOption('Line A')

    await waitFor(() => {
      expect(screen.getByText('Operator Note')).toBeInTheDocument()
      expect(screen.getByText('Shift')).toBeInTheDocument()
    })

    const customFilterSection = screen.getByText('自定义筛选').closest('div')?.parentElement
    const customTextInput = customFilterSection?.querySelector('input.ant-input')
    fireEvent.change(customTextInput as HTMLInputElement, {
      target: { value: 'Saved' },
    })
    expect(findProgramRow('Program Alpha')).toBeInTheDocument()
    expect(screen.queryByText('Program Bravo')).not.toBeInTheDocument()

    const lineFilterSelects = document.querySelectorAll('.management-filter-panel .ant-select-selector')
    fireEvent.mouseDown(lineFilterSelects[1] as Element)
    await selectLastMatchingOption('Night')
    expect(findProgramRow('Program Alpha')).toBeInTheDocument()
    expect(screen.queryByText('Program Bravo')).not.toBeInTheDocument()

    fireEvent.mouseDown(filterSelects[0] as Element)
    await selectLastMatchingOption('Line B')

    await waitFor(() => {
      expect(screen.getByText('Line B Status')).toBeInTheDocument()
    })

    expect(screen.queryByText('Operator Note')).not.toBeInTheDocument()
  })

  it('shows a dedicated custom filter section only after a production line with enabled fields is selected', async () => {
    renderPage()

    expect(screen.queryByText('自定义筛选')).not.toBeInTheDocument()

    const filterSelects = document.querySelectorAll('.management-filter-panel .ant-select-selector')
    fireEvent.mouseDown(filterSelects[0] as Element)
    await selectLastMatchingOption('Line A')

    expect(await screen.findByText('自定义筛选')).toBeInTheDocument()
    expect(screen.getByText('当前产线字段')).toBeInTheDocument()
  })

  it('replaces the custom filter section content when the production line changes', async () => {
    renderPage()

    const filterSelects = document.querySelectorAll('.management-filter-panel .ant-select-selector')
    fireEvent.mouseDown(filterSelects[0] as Element)
    await selectLastMatchingOption('Line A')
    expect(await screen.findByText('Operator Note')).toBeInTheDocument()

    fireEvent.mouseDown(filterSelects[0] as Element)
    await selectLastMatchingOption('Line B')

    expect(await screen.findByText('Line B Status')).toBeInTheDocument()
    expect(screen.queryByText('Operator Note')).not.toBeInTheDocument()
  })
})
