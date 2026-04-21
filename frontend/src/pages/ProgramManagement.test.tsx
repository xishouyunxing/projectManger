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
const mockApiDelete = api.delete as Mock

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
      { field_id: 11, field_name: 'Operator Note', field_type: 'text', sort_order: 1, value: 'Saved note' },
      { field_id: 12, field_name: 'Shift', field_type: 'select', sort_order: 2, value: 'Night' },
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
    custom_field_values: { '11': 'Other note', '12': 'Day' },
  },
]

const lineOneFields = [
  { id: 11, name: 'Operator Note', field_type: 'text', options_json: '', sort_order: 1, enabled: true },
  { id: 12, name: 'Shift', field_type: 'select', options_json: '["Day","Night"]', sort_order: 2, enabled: true },
  { id: 13, name: 'Disabled Field', field_type: 'text', options_json: '', sort_order: 3, enabled: false },
]

const lineTwoFields = [
  { id: 21, name: 'Line B Status', field_type: 'text', options_json: '', sort_order: 1, enabled: true },
]

const versionFixture = [
  {
    id: 201,
    version: 'v1.0.0',
    is_current: true,
    change_log: 'Stable release',
    created_at: '2026-01-03T08:00:00.000Z',
    file_count: 2,
    uploader: { name: 'Release Bot' },
    files: [
      {
        id: 301,
        file_name: 'program-alpha-main.bin',
        file_size: 2048,
        created_at: '2026-01-03T08:10:00.000Z',
        file_exists: true,
      },
      {
        id: 302,
        file_name: 'program-alpha-readme.txt',
        file_size: 1024,
        created_at: '2026-01-03T08:12:00.000Z',
        file_exists: true,
      },
    ],
  },
  {
    id: 202,
    version: 'v0.9.0',
    is_current: false,
    change_log: 'Archive release',
    created_at: '2025-12-20T08:00:00.000Z',
    file_count: 1,
    uploader: { name: 'Archive User' },
    files: [
      {
        id: 303,
        file_name: 'program-alpha-legacy.bin',
        file_size: 4096,
        created_at: '2025-12-20T08:15:00.000Z',
        file_exists: true,
      },
    ],
  },
]

const singleVersionFixture = [
  {
    id: 201,
    version: 'v1.0.0',
    is_current: true,
    change_log: 'Stable release',
    created_at: '2026-01-03T08:00:00.000Z',
    file_count: 2,
    uploader: { name: 'Release Bot' },
    files: [
      {
        id: 301,
        file_name: 'program-alpha-main.bin',
        file_size: 2048,
        created_at: '2026-01-03T08:10:00.000Z',
        file_exists: true,
      },
    ],
  },
]

const renderPage = () =>
  render(
    <MemoryRouter initialEntries={['/programs?keyword=Alpha&id=101']}>
      <ProgramManagement />
    </MemoryRouter>
  )

const getOverlay = async () => {
  const overlay = await screen.findByTestId('program-management-overlay')
  if (!(overlay instanceof HTMLElement)) {
    throw new Error('overlay not found')
  }
  return overlay
}

const clickEditButtonForProgram = (programName: string) => {
  const row = screen.getByText(programName).closest('tr')
  const buttons = row?.querySelectorAll('button') ?? []

  if (!(buttons[2] instanceof HTMLButtonElement)) {
    throw new Error(`Edit button not found for ${programName}`)
  }

  fireEvent.click(buttons[2])
}

const clickViewButtonForProgram = (programName: string) => {
  const row = screen.getByText(programName).closest('tr')
  const buttons = row?.querySelectorAll('button') ?? []

  if (!(buttons[0] instanceof HTMLButtonElement)) {
    throw new Error(`View button not found for ${programName}`)
  }

  fireEvent.click(buttons[0])
}

describe('ProgramManagement unified editor overlay', () => {
  beforeEach(() => {
    vi.clearAllMocks()

    mockApiGet.mockImplementation((url: string, config?: { responseType?: string }) => {
      if (config?.responseType === 'blob') {
        return Promise.resolve({
          data: new Blob(['zip-data']),
          headers: { 'content-disposition': 'attachment; filename="program-alpha-v1.0.0.zip"' },
        })
      }
      if (url === '/programs') return Promise.resolve({ data: programsFixture })
      if (url === '/production-lines') return Promise.resolve({ data: [{ id: 1, name: 'Line A' }, { id: 2, name: 'Line B' }] })
      if (url === '/vehicle-models') return Promise.resolve({ data: [{ id: 10, name: 'Model X' }] })
      if (url === '/production-lines/1/custom-fields') return Promise.resolve({ data: lineOneFields })
      if (url === '/production-lines/2/custom-fields') return Promise.resolve({ data: lineTwoFields })
      if (url === '/files/program/101') return Promise.resolve({ data: { versions: versionFixture } })
      if (url === '/relations/related/101') return Promise.resolve({ data: [] })
      return Promise.resolve({ data: { versions: [] } })
    })

    mockApiPost.mockResolvedValue({ data: { id: 104 } })
    mockApiPut.mockResolvedValue({ data: {} })
    mockApiDelete.mockResolvedValue({ data: {} })
  })

  it('renders the management page list with custom field tips', async () => {
    renderPage()

    expect(screen.getByRole('button', { name: /批量导入/ })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /导出Excel/ })).toBeInTheDocument()
    expect(await screen.findByText('Program Alpha')).toBeInTheDocument()
    expect(await screen.findByText('Saved note')).toBeInTheDocument()
    expect(await screen.findByText('Night')).toBeInTheDocument()
  })

  it('opens the unified editor overlay and loads version, file, and adaptive property sections', async () => {
    renderPage()

    await screen.findByText('Program Alpha')
    clickEditButtonForProgram('Program Alpha')

    await waitFor(() => {
      expect(mockApiGet.mock.calls.some(([url]) => url === '/production-lines/1/custom-fields')).toBe(true)
    })

    const overlay = await getOverlay()
    expect(within(overlay).getAllByRole('button', { name: /编辑/ }).length).toBeGreaterThan(1)
    expect(within(overlay).getByText('属性')).toBeInTheDocument()
    expect(within(overlay).getByText('历史版本')).toBeInTheDocument()
    expect(within(overlay).getByText('文件资产')).toBeInTheDocument()
    expect(within(overlay).getByText('program-alpha-main.bin')).toBeInTheDocument()
    expect(within(overlay).getByText('产线')).toBeInTheDocument()
    expect(within(overlay).getByText('车型')).toBeInTheDocument()
    expect(within(overlay).getByText('Operator Note')).toBeInTheDocument()
    expect(within(overlay).getByText('Shift')).toBeInTheDocument()
    expect(within(overlay).queryByText('数据策略')).not.toBeInTheDocument()
    expect(within(overlay).queryByText('Disabled Field')).not.toBeInTheDocument()
    expect(within(overlay).queryByText('上传时间')).not.toBeInTheDocument()
    expect(within(overlay).getAllByText(/\d{4}.*\d{1,2}.*\d{1,2}/).length).toBeGreaterThan(0)

    const nameInput = screen.getByDisplayValue('Program Alpha') as HTMLInputElement
    const codeInput = screen.getByDisplayValue('PA-01') as HTMLInputElement
    const descriptionInput = screen.getByDisplayValue('Stable release') as HTMLTextAreaElement
    const noteInput = screen.getByDisplayValue('Saved note') as HTMLInputElement

    expect(nameInput).toBeDisabled()
    expect(codeInput).toBeDisabled()
    expect(descriptionInput).toBeDisabled()
    expect(noteInput).toBeDisabled()
    expect(screen.getAllByText('Night').length).toBeGreaterThan(0)

    const versionTitles = within(overlay).getAllByText(/v\d+\.\d+\.\d+/).map((node) => node.textContent)
    expect(versionTitles.slice(0, 2)).toEqual(['v1.0.0', 'v0.9.0'])
  })

  it('submits base program updates from the unified editor overlay', async () => {
    renderPage()

    await screen.findByText('Program Alpha')
    clickEditButtonForProgram('Program Alpha')

    const overlay = await getOverlay()

    await waitFor(() => {
      expect(mockApiGet.mock.calls.some(([url]) => url === '/production-lines/1/custom-fields')).toBe(true)
    })

    fireEvent.submit(overlay.querySelector('form') as HTMLFormElement)

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

    expect(mockApiPut).toHaveBeenCalledWith('/programs/101/custom-field-values', {
      values: [
        { field_id: 11, value: 'Saved note' },
        { field_id: 12, value: 'Night' },
      ]
    })
  })

  it('enables property editing only after clicking edit and saves via the property action', async () => {
    renderPage()

    await screen.findByText('Program Alpha')
    clickEditButtonForProgram('Program Alpha')

    const overlay = await getOverlay()
    const nameInput = screen.getByDisplayValue('Program Alpha') as HTMLInputElement
    const noteInput = screen.getByDisplayValue('Saved note') as HTMLInputElement
    const descriptionInput = screen.getByDisplayValue('Stable release') as HTMLTextAreaElement

    expect(nameInput).toBeDisabled()
    expect(noteInput).toBeDisabled()
    expect(descriptionInput).toBeDisabled()

    const propertyCard = within(overlay).getByText('属性').closest('.program-editor-property-strip-content') as HTMLElement
    fireEvent.click(within(propertyCard).getByRole('button', { name: /编辑/ }))

    await waitFor(() => {
      expect(nameInput).not.toBeDisabled()
    })
    expect(noteInput).not.toBeDisabled()
    expect(descriptionInput).toBeDisabled()
    expect(within(propertyCard).getByRole('button', { name: '取消' })).toBeInTheDocument()
    expect(within(propertyCard).getByRole('button', { name: /保存/ })).toBeInTheDocument()

    fireEvent.change(nameInput, { target: { value: 'Program Alpha Updated' } })
    fireEvent.change(noteInput, { target: { value: 'Updated note' } })
    fireEvent.click(within(propertyCard).getByRole('button', { name: /保存/ }))

    await waitFor(() => {
      expect(mockApiPut).toHaveBeenCalledWith('/programs/101', {
        name: 'Program Alpha Updated',
        code: 'PA-01',
        production_line_id: 1,
        vehicle_model_id: 10,
        status: 'in_progress',
        description: 'Existing program',
      })
    })

    expect(mockApiPut).toHaveBeenCalledWith('/programs/101/custom-field-values', {
      values: [
        { field_id: 11, value: 'Updated note' },
        { field_id: 12, value: 'Night' },
      ]
    })

    await waitFor(() => {
      expect(mockApiGet.mock.calls.filter(([url]) => url === '/programs').length).toBeGreaterThan(1)
    })
  })

  it('cancels property edits and restores the original values', async () => {
    renderPage()

    await screen.findByText('Program Alpha')
    clickEditButtonForProgram('Program Alpha')

    const overlay = await getOverlay()
    const propertyCard = within(overlay).getByText('属性').closest('.program-editor-property-strip-content') as HTMLElement
    const nameInput = screen.getByDisplayValue('Program Alpha') as HTMLInputElement
    const noteInput = screen.getByDisplayValue('Saved note') as HTMLInputElement

    fireEvent.click(within(propertyCard).getByRole('button', { name: /编辑/ }))

    await waitFor(() => {
      expect(nameInput).not.toBeDisabled()
    })

    fireEvent.change(nameInput, { target: { value: 'Program Alpha Updated' } })
    fireEvent.change(noteInput, { target: { value: 'Updated note' } })
    fireEvent.click(within(propertyCard).getByRole('button', { name: '取消' }))

    await waitFor(() => {
      expect(nameInput).toBeDisabled()
    })

    expect(nameInput.value).toBe('Program Alpha')
    expect(noteInput.value).toBe('Saved note')
    expect(within(propertyCard).queryByRole('button', { name: '取消' })).not.toBeInTheDocument()
    expect(within(propertyCard).getByRole('button', { name: /编辑/ })).toBeInTheDocument()
    expect(mockApiPut).not.toHaveBeenCalled()
  })
  it('retransfers the selected version from the main editor card', async () => {
    renderPage()

    await screen.findByText('Program Alpha')
    clickEditButtonForProgram('Program Alpha')

    const overlay = await getOverlay()

    fireEvent.click(within(overlay).getByRole('button', { name: '重传此版本' }))

    await waitFor(() => {
      expect(screen.getByText('上传程序文件')).toBeInTheDocument()
    })

    expect(screen.getByDisplayValue('101')).toBeInTheDocument()
    expect(screen.getByDisplayValue('v1.0.0')).toBeInTheDocument()
  })

  it('keeps property edit state and values while switching selected version', async () => {
    renderPage()

    await screen.findByText('Program Alpha')
    clickEditButtonForProgram('Program Alpha')

    const overlay = await getOverlay()
    const propertyCard = within(overlay).getByText('属性').closest('.program-editor-property-strip-content') as HTMLElement
    const nameInput = screen.getByDisplayValue('Program Alpha') as HTMLInputElement

    fireEvent.click(within(propertyCard).getByRole('button', { name: /编辑/ }))

    await waitFor(() => {
      expect(nameInput).not.toBeDisabled()
    })

    fireEvent.change(nameInput, { target: { value: 'Program Alpha Draft' } })
    fireEvent.click(within(overlay).getByText('v0.9.0'))

    await waitFor(() => {
      expect(within(overlay).getByText('版本 v0.9.0')).toBeInTheDocument()
    })

    expect(nameInput).not.toBeDisabled()
    expect(nameInput.value).toBe('Program Alpha Draft')
    expect(within(propertyCard).getByRole('button', { name: /保存/ })).toBeInTheDocument()
  }, 10000)

  it('opens the redesigned readonly version modal', async () => {
    renderPage()

    await screen.findByText('Program Alpha')
    clickViewButtonForProgram('Program Alpha')

    const overlay = await screen.findByTestId('program-view-overlay')
    expect(overlay).toBeInTheDocument()
    expect(screen.getByText(/程序详情/)).toBeInTheDocument()
    expect(screen.getByText('版本记录')).toBeInTheDocument()
    expect(screen.getByText('只读模式')).toBeInTheDocument()
    expect(screen.getByText('当前版本概览')).toBeInTheDocument()
    expect(screen.getByText('版本说明')).toBeInTheDocument()
    expect(screen.getByText('Stable release')).toBeInTheDocument()
    expect(screen.getByText('文件资产')).toBeInTheDocument()
    expect(screen.getByText('产线: Line A')).toBeInTheDocument()
    expect(screen.getByText('车型: Model X')).toBeInTheDocument()
    expect(screen.getByText('Operator Note: Saved note')).toBeInTheDocument()
    expect(screen.getByText('Shift: Night')).toBeInTheDocument()
    expect(within(overlay).getByRole('button', { name: '下载全部' })).toBeInTheDocument()
    expect(within(overlay).getByRole('button', { name: '进入编辑' })).toBeInTheDocument()
    expect(screen.getByText('program-alpha-main.bin')).toBeInTheDocument()
    expect(screen.getByText('program-alpha-readme.txt')).toBeInTheDocument()
  }, 10000)

  it('enters editor from readonly version modal', async () => {
    renderPage()

    await screen.findByText('Program Alpha')
    clickViewButtonForProgram('Program Alpha')

    const viewOverlay = await screen.findByTestId('program-view-overlay')
    fireEvent.click(within(viewOverlay).getByRole('button', { name: '进入编辑' }))

    expect(await screen.findByTestId('program-management-overlay')).toBeInTheDocument()
  }, 10000)

  it('shows the active version style even when there is only one version', async () => {
    mockApiGet.mockImplementation((url: string, config?: { responseType?: string }) => {
      if (config?.responseType === 'blob') {
        return Promise.resolve({
          data: new Blob(['zip-data']),
          headers: { 'content-disposition': 'attachment; filename="program-alpha-v1.0.0.zip"' },
        })
      }
      if (url === '/programs') return Promise.resolve({ data: programsFixture })
      if (url === '/production-lines') return Promise.resolve({ data: [{ id: 1, name: 'Line A' }, { id: 2, name: 'Line B' }] })
      if (url === '/vehicle-models') return Promise.resolve({ data: [{ id: 10, name: 'Model X' }] })
      if (url === '/production-lines/1/custom-fields') return Promise.resolve({ data: lineOneFields })
      if (url === '/production-lines/2/custom-fields') return Promise.resolve({ data: lineTwoFields })
      if (url === '/files/program/101') return Promise.resolve({ data: { versions: singleVersionFixture } })
      if (url === '/relations/related/101') return Promise.resolve({ data: [] })
      return Promise.resolve({ data: { versions: [] } })
    })

    renderPage()

    await screen.findByText('Program Alpha')
    clickEditButtonForProgram('Program Alpha')

    const overlay = await getOverlay()
    const versionButton = overlay.querySelector('.program-editor-version-button')

    expect(versionButton).toHaveClass('active')
  })
})
