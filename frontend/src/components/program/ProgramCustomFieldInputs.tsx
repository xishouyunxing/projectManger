import { Form, Input, Select } from 'antd'

type ProgramCustomFieldDefinition = {
  id: number
  name: string
  field_type: 'text' | 'select'
  options_json: string
  sort_order: number
  enabled: boolean
}

type ProgramCustomFieldInputsProps = {
  fields: ProgramCustomFieldDefinition[]
}

const parseOptions = (optionsJson: string) => {
  if (!optionsJson) {
    return []
  }

  try {
    const parsed = JSON.parse(optionsJson)
    return Array.isArray(parsed) ? parsed.filter((option): option is string => typeof option === 'string') : []
  } catch {
    return []
  }
}

const ProgramCustomFieldInputs = ({ fields }: ProgramCustomFieldInputsProps) => {
  if (fields.length === 0) {
    return null
  }

  return (
    <>
      {fields.map((field) => (
        <Form.Item
          key={field.id}
          name={['custom_field_values', String(field.id)]}
          label={field.name}
        >
          {field.field_type === 'select' ? (
            <Select allowClear>
              {parseOptions(field.options_json).map((option) => (
                <Select.Option key={option} value={option}>
                  {option}
                </Select.Option>
              ))}
            </Select>
          ) : (
            <Input />
          )}
        </Form.Item>
      ))}
    </>
  )
}

export default ProgramCustomFieldInputs
