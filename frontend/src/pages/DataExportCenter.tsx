import { useCallback, useEffect, useMemo, useState } from 'react';
import {
  Button,
  Card,
  Checkbox,
  DatePicker,
  Input,
  Select,
  Space,
  Tag,
  Typography,
  message,
  Spin,
} from 'antd';
import {
  FileExcelOutlined,
  ReloadOutlined,
  SearchOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { Dayjs } from 'dayjs';
import api from '../services/api';
import ColumnSelector, {
  type ColumnDef,
} from '../components/export/ColumnSelector';
import ExportPreviewTable from '../components/export/ExportPreviewTable';
import ExportStatsCards from '../components/export/ExportStatsCards';

const { Title, Text } = Typography;

const STORAGE_KEY = 'data-export-selected-columns';

const DataExportCenter = () => {
  // 列元数据
  const [builtinFields, setBuiltinFields] = useState<ColumnDef[]>([]);
  const [customFields, setCustomFields] = useState<ColumnDef[]>([]);
  const [columnsLoading, setColumnsLoading] = useState(true);

  // 用户选择
  const [selectedKeys, setSelectedKeys] = useState<string[]>([]);
  const [filterLineIds, setFilterLineIds] = useState<number[]>([]);
  const [filterVehicleModel, setFilterVehicleModel] = useState<number | null>(
    null,
  );
  const [filterStatus, setFilterStatus] = useState<string | null>(null);
  const [filterKeyword, setFilterKeyword] = useState('');
  const [filterDateRange, setFilterDateRange] = useState<
    [Dayjs | null, Dayjs | null] | null
  >(null);
  const [includeStats, setIncludeStats] = useState(true);

  // 关键词防抖
  const [debouncedKeyword, setDebouncedKeyword] = useState('');
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedKeyword(filterKeyword), 400);
    return () => clearTimeout(timer);
  }, [filterKeyword]);

  // 基础数据
  const [productionLines, setProductionLines] = useState<any[]>([]);
  const [vehicleModels, setVehicleModels] = useState<any[]>([]);

  // 预览数据
  const [previewData, setPreviewData] = useState<Record<string, any>[]>([]);
  const [previewColumns, setPreviewColumns] = useState<
    { key: string; label: string }[]
  >([]);
  const [previewTotal, setPreviewTotal] = useState(0);
  const [previewPage, setPreviewPage] = useState(1);
  const [previewPageSize, setPreviewPageSize] = useState(20);
  const [previewLoading, setPreviewLoading] = useState(false);

  // 统计数据
  const [statsData, setStatsData] = useState({
    totalPrograms: 0,
    completedPrograms: 0,
    inProgressPrograms: 0,
    totalLines: 0,
    totalModels: 0,
    overallRate: 0,
    lineRates: [] as { name: string; rate: number }[],
    modelRates: [] as { name: string; rate: number }[],
  });
  const [statsLoading, setStatsLoading] = useState(false);

  // 导出中
  const [exporting, setExporting] = useState(false);

  const lineNameMap = useMemo(() => {
    const map: Record<number, string> = {};
    productionLines.forEach((l: any) => {
      map[l.id] = l.name;
    });
    return map;
  }, [productionLines]);

  // 加载基础数据
  useEffect(() => {
    loadColumns();
    loadBaseData();
  }, []);

  // 恢复上次列选择
  useEffect(() => {
    if (builtinFields.length > 0) {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        try {
          const parsed = JSON.parse(saved);
          if (Array.isArray(parsed) && parsed.length > 0) {
            setSelectedKeys(parsed);
            return;
          }
        } catch {
          // ignore
        }
      }
      // 默认全选内置字段
      setSelectedKeys(builtinFields.map((f) => f.key));
    }
  }, [builtinFields]);

  // 预览数据加载
  useEffect(() => {
    if (selectedKeys.length > 0) {
      loadPreview();
    }
  }, [selectedKeys, filterLineIds, filterVehicleModel, filterStatus, debouncedKeyword, filterDateRange, previewPage, previewPageSize]);

  // 统计数据加载
  useEffect(() => {
    loadStats();
  }, [filterLineIds, filterVehicleModel, filterStatus, debouncedKeyword, filterDateRange]);

  const loadColumns = async () => {
    setColumnsLoading(true);
    try {
      const res = await api.get('/programs/export/columns');
      setBuiltinFields(res.data.builtin_fields || []);
      setCustomFields(res.data.custom_fields || []);
    } catch (error) {
      console.error('Failed to load columns:', error);
      message.error('加载列配置失败');
    } finally {
      setColumnsLoading(false);
    }
  };

  const loadBaseData = async () => {
    try {
      const [linesRes, modelsRes] = await Promise.all([
        api.get('/production-lines'),
        api.get('/vehicle-models'),
      ]);
      setProductionLines(linesRes.data || []);
      setVehicleModels(modelsRes.data || []);
    } catch (error) {
      console.error('Failed to load base data:', error);
    }
  };

  const loadPreview = useCallback(async () => {
    setPreviewLoading(true);
    try {
      const params: any = {
        columns: selectedKeys.join(','),
        page: previewPage,
        page_size: previewPageSize,
      };
      if (filterLineIds.length > 0)
        params.line_ids = filterLineIds.join(',');
      if (filterVehicleModel) params.vehicle_model_id = filterVehicleModel;
      if (filterStatus) params.status = filterStatus;
      if (debouncedKeyword.trim()) params.keyword = debouncedKeyword.trim();
      if (filterDateRange?.[0])
        params.date_from = filterDateRange[0].format('YYYY-MM-DD');
      if (filterDateRange?.[1])
        params.date_to = filterDateRange[1].format('YYYY-MM-DD');

      const res = await api.get('/programs/export/preview', { params });
      setPreviewData(res.data.items || []);
      setPreviewColumns(res.data.columns || []);
      setPreviewTotal(res.data.total || 0);
    } catch (error) {
      console.error('Failed to load preview:', error);
      message.error('加载预览数据失败');
    } finally {
      setPreviewLoading(false);
    }
  }, [
    selectedKeys,
    filterLineIds,
    filterVehicleModel,
    filterStatus,
    debouncedKeyword,
    filterDateRange,
    previewPage,
    previewPageSize,
  ]);

  const loadStats = useCallback(async () => {
    setStatsLoading(true);
    try {
      const params: any = {};
      if (filterLineIds.length > 0)
        params.line_ids = filterLineIds.join(',');
      if (filterVehicleModel) params.vehicle_model_id = filterVehicleModel;
      if (filterStatus) params.status = filterStatus;
      if (debouncedKeyword.trim()) params.keyword = debouncedKeyword.trim();
      if (filterDateRange?.[0])
        params.date_from = filterDateRange[0].format('YYYY-MM-DD');
      if (filterDateRange?.[1])
        params.date_to = filterDateRange[1].format('YYYY-MM-DD');

      const res = await api.get('/programs/export/stats', { params });
      const d = res.data || {};
      setStatsData({
        totalPrograms: d.total_programs || 0,
        completedPrograms: d.completed_programs || 0,
        inProgressPrograms: d.in_progress_programs || 0,
        totalLines: d.total_lines || 0,
        totalModels: d.total_models || 0,
        overallRate: d.overall_rate || 0,
        lineRates: (d.line_rates || []).map((r: any) => ({
          name: r.name,
          rate: r.rate,
        })),
        modelRates: (d.model_rates || []).map((r: any) => ({
          name: r.name,
          rate: r.rate,
        })),
      });
    } catch (error) {
      console.error('Failed to load stats:', error);
    } finally {
      setStatsLoading(false);
    }
  }, [filterLineIds, filterVehicleModel, filterStatus, debouncedKeyword, filterDateRange]);

  const handleSelectedKeysChange = (keys: string[]) => {
    setSelectedKeys(keys);
    setPreviewPage(1);
    localStorage.setItem(STORAGE_KEY, JSON.stringify(keys));
  };

  const handleExport = async () => {
    if (selectedKeys.length === 0) {
      message.warning('请至少选择一列');
      return;
    }

    setExporting(true);
    try {
      const params: any = {
        columns: selectedKeys.join(','),
        include_stats: includeStats ? 'true' : 'false',
      };
      if (filterLineIds.length > 0)
        params.line_ids = filterLineIds.join(',');
      if (filterVehicleModel) params.vehicle_model_id = filterVehicleModel;
      if (filterStatus) params.status = filterStatus;
      if (debouncedKeyword.trim()) params.keyword = debouncedKeyword.trim();
      if (filterDateRange?.[0])
        params.date_from = filterDateRange[0].format('YYYY-MM-DD');
      if (filterDateRange?.[1])
        params.date_to = filterDateRange[1].format('YYYY-MM-DD');

      const response = await api.get('/programs/export/excel', {
        params,
        responseType: 'blob',
      });

      const url = window.URL.createObjectURL(new Blob([response.data]));
      const link = document.createElement('a');
      link.href = url;

      const contentDisposition = response.headers['content-disposition'];
      let fileName = '程序数据导出.xlsx';
      if (contentDisposition) {
        const match = contentDisposition.match(/filename\*=UTF-8''(.+)/);
        if (match) {
          fileName = decodeURIComponent(match[1]);
        } else {
          const match2 = contentDisposition.match(/filename="?([^";]+)"?/);
          if (match2) fileName = match2[1];
        }
      }

      link.setAttribute('download', fileName);
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);

      message.success('导出成功');
    } catch (error) {
      console.error('Export failed:', error);
      message.error('导出失败');
    } finally {
      setExporting(false);
    }
  };

  const handlePageChange = (page: number, pageSize: number) => {
    setPreviewPage(page);
    setPreviewPageSize(pageSize);
  };

  return (
    <div style={{ padding: 24, maxWidth: 1600, margin: '0 auto' }}>
      {/* 顶部标题 */}
      <Card bordered={false} style={{ marginBottom: 16 }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
          }}
        >
          <Space>
            <FileExcelOutlined style={{ fontSize: 24, color: '#005BC1' }} />
            <div>
              <Title level={3} style={{ margin: 0 }}>
                数据导出中心
              </Title>
              <Text type="secondary" style={{ fontSize: 13 }}>
                动态选择列、筛选数据、预览后导出 Excel
              </Text>
            </div>
          </Space>
          <Button
            icon={<ReloadOutlined />}
            onClick={() => {
              loadPreview();
              loadStats();
            }}
          >
            刷新
          </Button>
        </div>
      </Card>

      <div style={{ display: 'flex', gap: 16, alignItems: 'flex-start' }}>
        {/* 左侧面板 */}
        <div style={{ width: 300, flexShrink: 0 }}>
          {/* 筛选条件 */}
          <Card
            size="small"
            bordered={false}
            style={{ marginBottom: 16 }}
            title={
              <Space>
                <SettingOutlined />
                <span>筛选条件</span>
              </Space>
            }
          >
            <Space direction="vertical" style={{ width: '100%' }} size={12}>
              <div>
                <Text
                  type="secondary"
                  style={{ fontSize: 12, display: 'block', marginBottom: 4 }}
                >
                  关键词搜索
                </Text>
                <Input
                  placeholder="程序名称 / 编号"
                  allowClear
                  prefix={<SearchOutlined style={{ color: '#bfbfbf' }} />}
                  value={filterKeyword}
                  onChange={(e) => {
                    setFilterKeyword(e.target.value);
                    setPreviewPage(1);
                  }}
                />
              </div>
              <div>
                <Text
                  type="secondary"
                  style={{ fontSize: 12, display: 'block', marginBottom: 4 }}
                >
                  创建时间
                </Text>
                <DatePicker.RangePicker
                  style={{ width: '100%' }}
                  value={filterDateRange as any}
                  onChange={(dates) => {
                    setFilterDateRange(dates as [Dayjs | null, Dayjs | null] | null);
                    setPreviewPage(1);
                  }}
                  placeholder={['开始日期', '结束日期']}
                  allowEmpty={[true, true]}
                />
              </div>
              <div>
                <Text
                  type="secondary"
                  style={{ fontSize: 12, display: 'block', marginBottom: 4 }}
                >
                  产线
                </Text>
                <Select
                  mode="multiple"
                  style={{ width: '100%' }}
                  placeholder="全部产线"
                  allowClear
                  value={filterLineIds.length > 0 ? filterLineIds : undefined}
                  onChange={(value) => {
                    setFilterLineIds(value || []);
                    setPreviewPage(1);
                  }}
                  options={productionLines.map((l: any) => ({
                    value: l.id,
                    label: l.name,
                  }))}
                  maxTagCount="responsive"
                />
              </div>
              <div>
                <Text
                  type="secondary"
                  style={{ fontSize: 12, display: 'block', marginBottom: 4 }}
                >
                  车型
                </Text>
                <Select
                  style={{ width: '100%' }}
                  placeholder="全部车型"
                  allowClear
                  value={filterVehicleModel ?? undefined}
                  onChange={(value) => {
                    setFilterVehicleModel(value ?? null);
                    setPreviewPage(1);
                  }}
                  options={vehicleModels.map((m: any) => ({
                    value: m.id,
                    label: m.name,
                  }))}
                />
              </div>
              <div>
                <Text
                  type="secondary"
                  style={{ fontSize: 12, display: 'block', marginBottom: 4 }}
                >
                  状态
                </Text>
                <Select
                  style={{ width: '100%' }}
                  placeholder="全部状态"
                  allowClear
                  value={filterStatus ?? undefined}
                  onChange={(value) => {
                    setFilterStatus(value ?? null);
                    setPreviewPage(1);
                  }}
                  options={[
                    { value: 'completed', label: '已完成' },
                    { value: 'in_progress', label: '进行中' },
                  ]}
                />
              </div>
            </Space>
          </Card>

          {/* 列选择器 */}
          <Card
            size="small"
            bordered={false}
            style={{ marginBottom: 16 }}
            title={
              <Space>
                <span>选择导出列</span>
                <Tag color="blue">{selectedKeys.length}</Tag>
              </Space>
            }
          >
            <Spin spinning={columnsLoading}>
              <ColumnSelector
                builtinFields={builtinFields}
                customFields={customFields}
                selectedKeys={selectedKeys}
                onChange={handleSelectedKeysChange}
                productionLineNames={lineNameMap}
              />
            </Spin>
          </Card>

          {/* 导出选项 */}
          <Card size="small" bordered={false}>
            <Checkbox
              checked={includeStats}
              onChange={(e) => setIncludeStats(e.target.checked)}
            >
              <Text style={{ fontSize: 13 }}>包含完成率统计 Sheet</Text>
            </Checkbox>
            <Text
              type="secondary"
              style={{ fontSize: 11, display: 'block', marginTop: 4, marginBottom: 16 }}
            >
              导出时额外生成产线×车型完成率矩阵
            </Text>
            <Button
              type="primary"
              block
              icon={<FileExcelOutlined />}
              loading={exporting}
              disabled={selectedKeys.length === 0}
              onClick={handleExport}
              style={{
                background: 'linear-gradient(176deg, #005BC1 0%, #3D89FF 100%)',
                border: 'none',
                borderRadius: 8,
                fontWeight: 600,
                height: 40,
              }}
            >
              导出 Excel
            </Button>
            <Text
              type="secondary"
              style={{ fontSize: 11, display: 'block', marginTop: 8, textAlign: 'center' }}
            >
              已选 {selectedKeys.length} 列
            </Text>
          </Card>
        </div>

        {/* 右侧预览区 */}
        <div style={{ flex: 1, minWidth: 0 }}>
          {/* 统计卡片 */}
          <ExportStatsCards stats={statsData} loading={statsLoading} />

          {/* 预览表格 */}
          <Card
            bordered={false}
            title={
              <Space>
                <span style={{ fontWeight: 600 }}>数据预览</span>
                <Text type="secondary" style={{ fontSize: 12 }}>
                  共 {previewTotal} 条记录，已选 {selectedKeys.length} 列
                </Text>
              </Space>
            }
          >
            <ExportPreviewTable
              columns={previewColumns}
              dataSource={previewData}
              loading={previewLoading}
              total={previewTotal}
              page={previewPage}
              pageSize={previewPageSize}
              onPageChange={handlePageChange}
            />
          </Card>
        </div>
      </div>

      {/* 表格样式 */}
      <style>{`
        .custom-table .ant-table-thead > tr > th {
          background: #EBEEF0 !important;
          color: #5A6062 !important;
          font-size: 10px !important;
          font-weight: 700 !important;
          letter-spacing: 1px !important;
          border-bottom: 1px solid #DEE3E6 !important;
          padding: 12px 16px !important;
        }
        .custom-table .ant-table-tbody > tr > td {
          padding: 10px 16px !important;
          border-bottom: 1px solid #EBEEF0 !important;
        }
        .custom-table .ant-table {
          border-radius: 12px 12px 0 0 !important;
        }
        .custom-table .ant-pagination-total-text {
          color: #5A6062;
          font-size: 12px;
          font-weight: 500;
        }
      `}</style>
    </div>
  );
};

export default DataExportCenter;
