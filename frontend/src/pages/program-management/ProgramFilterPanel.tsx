import { Button, ConfigProvider, DatePicker, Input, Select } from 'antd';
import ProgramCustomFieldFilter from '../../components/program/ProgramCustomFieldFilter';
import type {
  ProductionLine,
  ProgramCustomFieldDefinition,
  VehicleModel,
} from './types';

interface ProgramFilterPanelProps {
  searchKeyword: string;
  filterProductionLine: number | null;
  filterVehicleModel: number | null;
  filterStatus: string | null;
  productionLines: ProductionLine[];
  vehicleModels: VehicleModel[];
  customFieldFilters: ProgramCustomFieldDefinition[];
  customFieldFilterValues: Record<string, string>;
  onSearchKeywordChange: (value: string) => void;
  onFilterProductionLineChange: (value?: number) => void;
  onFilterVehicleModelChange: (value?: number) => void;
  onFilterStatusChange: (value?: string) => void;
  onDateRangeChange: (range: [string | null, string | null]) => void;
  onReset: () => void;
  onCustomFieldFilterChange: (fieldId: string, value: string) => void;
}

const ProgramFilterPanel = ({
  searchKeyword,
  filterProductionLine,
  filterVehicleModel,
  filterStatus,
  productionLines,
  vehicleModels,
  customFieldFilters,
  customFieldFilterValues,
  onSearchKeywordChange,
  onFilterProductionLineChange,
  onFilterVehicleModelChange,
  onFilterStatusChange,
  onDateRangeChange,
  onReset,
  onCustomFieldFilterChange,
}: ProgramFilterPanelProps) => {
  return (
    <ConfigProvider
      theme={{
        components: {
          Input: {
            controlHeight: 36,
            borderRadius: 8,
            colorBorder: 'transparent',
            colorPrimaryHover: 'transparent',
            controlOutline: 'none',
          },
          Select: {
            controlHeight: 36,
            borderRadius: 8,
            colorBorder: 'transparent',
            colorPrimaryHover: 'transparent',
            controlOutline: 'none',
          },
          DatePicker: {
            controlHeight: 36,
            borderRadius: 8,
            colorBorder: 'transparent',
            colorPrimaryHover: 'transparent',
            controlOutline: 'none',
          },
        },
      }}
    >
      <div className="management-filter-panel">
        <div className="management-filter-field flex">
          <div className="management-filter-label">程序名称/编号</div>
          <Input
            style={{ width: '192px', maxWidth: '100%' }}
            placeholder="搜索参数..."
            value={searchKeyword}
            onChange={(e) => onSearchKeywordChange(e.target.value)}
          />
        </div>
        <div className="management-filter-field">
          <div className="management-filter-label">生产线</div>
          <Select
            placeholder="所有生产线"
            value={filterProductionLine}
            onChange={onFilterProductionLineChange}
            allowClear
            style={{ width: '168px', maxWidth: '100%' }}
          >
            {productionLines.map((line) => (
              <Select.Option key={line.id} value={line.id}>
                {line.name}
              </Select.Option>
            ))}
          </Select>
        </div>
        <div className="management-filter-field">
          <div className="management-filter-label">车型</div>
          <Select
            placeholder="所有车型"
            value={filterVehicleModel}
            onChange={onFilterVehicleModelChange}
            allowClear
            style={{ width: '168px', maxWidth: '100%' }}
          >
            {vehicleModels.map((model) => (
              <Select.Option key={model.id} value={model.id}>
                {model.name}
              </Select.Option>
            ))}
          </Select>
        </div>
        <div className="management-filter-field">
          <div className="management-filter-label">状态</div>
          <Select
            placeholder="所有状态"
            value={filterStatus}
            onChange={onFilterStatusChange}
            allowClear
            style={{ width: '148px', maxWidth: '100%' }}
          >
            <Select.Option value="completed">已完成</Select.Option>
            <Select.Option value="in_progress">进行中</Select.Option>
          </Select>
        </div>
        <div className="management-filter-field">
          <div className="management-filter-label">创建日期</div>
          <DatePicker.RangePicker
            style={{ width: '176px', maxWidth: '100%' }}
            onChange={(_, dateStrings) => {
              onDateRangeChange([dateStrings[0] || null, dateStrings[1] || null]);
            }}
          />
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <Button
            onClick={() => {}}
            style={{
              height: '40px',
              width: '104px',
              borderRadius: '8px',
              background: '#DEE3E6',
              color: '#2D3335',
              fontWeight: 700,
              border: 'none',
            }}
          >
            查询
          </Button>
          <Button
            type="text"
            onClick={onReset}
            style={{
              height: '40px',
              color: '#005BC1',
              fontWeight: 700,
              letterSpacing: '1.2px',
              border: 'none',
              background: 'transparent',
            }}
          >
            重置
          </Button>
        </div>

        {filterProductionLine && customFieldFilters.length > 0 && (
          <div
            style={{
              gridColumn: '1 / -1',
              marginTop: '4px',
              paddingTop: '20px',
              borderTop: '1px solid rgba(173, 179, 181, 0.18)',
            }}
          >
            <div style={{ padding: '0' }}>
              <div
                style={{
                  display: 'flex',
                  alignItems: 'flex-start',
                  gap: '14px',
                  flexWrap: 'wrap',
                }}
              >
                <div
                  style={{
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    minWidth: '84px',
                    minHeight: '36px',
                    alignSelf: 'center',
                    flexShrink: 0,
                  }}
                >
                  <span
                    style={{
                      color: '#64748B',
                      fontSize: '11px',
                      fontWeight: 700,
                      letterSpacing: '0.14em',
                      textTransform: 'uppercase',
                      whiteSpace: 'nowrap',
                    }}
                  >
                    自定义筛选
                  </span>
                </div>
                <div
                  style={{
                    width: '1px',
                    alignSelf: 'stretch',
                    margin: '4px 0',
                    background: 'rgba(173, 179, 181, 0.18)',
                  }}
                />
                <div style={{ flex: 1, minWidth: '260px' }}>
                  <div
                    style={{
                      color: '#5A6062',
                      fontSize: '10px',
                      fontWeight: 700,
                      letterSpacing: '0.12em',
                      textTransform: 'uppercase',
                      marginBottom: '10px',
                    }}
                  >
                    当前产线字段
                  </div>
                  <ProgramCustomFieldFilter
                    fields={customFieldFilters}
                    values={customFieldFilterValues}
                    onChange={onCustomFieldFilterChange}
                  />
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </ConfigProvider>
  );
};

export default ProgramFilterPanel;
