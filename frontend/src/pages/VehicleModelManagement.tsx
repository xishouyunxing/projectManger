import { useEffect, useState } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  message,
  Typography,
  Tag,
  Popconfirm,
  Drawer,
  Select,
  Badge,
  Empty,
  DatePicker,
  ConfigProvider,
  Tooltip,
} from 'antd';
import { PlusOutlined, DeleteOutlined, EyeOutlined, CarOutlined, ApartmentOutlined, SearchOutlined, EditOutlined, AppstoreOutlined } from '@ant-design/icons';
import api, { extractListData, extractPagedListData } from '../services/api';

const { Title, Text } = Typography;
const { TextArea } = Input;

const VehicleModelManagement = () => {
  const [vehicleModels, setVehicleModels] = useState<any[]>([]);
  const [productionLines, setProductionLines] = useState<any[]>([]);
  const [programs, setPrograms] = useState<any[]>([]);
  const [loading, setLoading] = useState(false);
  const [tablePagination, setTablePagination] = useState({ current: 1, pageSize: 20, total: 0 });
  const [modalVisible, setModalVisible] = useState(false);
  const [drawerVisible, setDrawerVisible] = useState(false);
  const [programDetailVisible, setProgramDetailVisible] = useState(false);
  const [currentModel, setCurrentModel] = useState<any>(null);
  const [currentProgram, setCurrentProgram] = useState<any>(null);
  const [programVersions, setProgramVersions] = useState<any[]>([]);
  const [sharedPrograms, setSharedPrograms] = useState<any[]>([]);
  const [collapsedLines, setCollapsedLines] = useState<number[]>([]);
  const [form] = Form.useForm();

  // 筛选相关状态
  const [filterSeries, setFilterSeries] = useState<string | null>(null);
  const [searchKeyword, setSearchKeyword] = useState('');
  const [filterDateRange, setFilterDateRange] = useState<[string | null, string | null]>([null, null]);

  // 获取所有系列（去重）
  const allSeries = [...new Set(vehicleModels.map((m: any) => m.series).filter(Boolean))];

  // 筛选后的车型列表
  

  // 重置筛选
  const handleResetFilter = () => {
    setFilterSeries(null);
    setSearchKeyword('');
    setFilterDateRange([null, null]);
  };

  useEffect(() => {
    loadData(1, tablePagination.pageSize);
  }, [searchKeyword, filterSeries, filterDateRange]);

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async (page = tablePagination.current, pageSize = tablePagination.pageSize) => {
    setLoading(true);
    try {
      const [modelsRes, linesRes] = await Promise.all([
        api.get('/vehicle-models', {
          params: {
            page,
            page_size: pageSize,
            ...(searchKeyword ? { keyword: searchKeyword } : {}),
            ...(filterSeries ? { series: filterSeries } : {}),
            ...(filterDateRange[0] ? { date_from: filterDateRange[0] } : {}),
            ...(filterDateRange[1] ? { date_to: filterDateRange[1] } : {}),
          },
        }),
        api.get('/production-lines'),
      ]);
      const modelsPaged = extractPagedListData(modelsRes.data);
      const fallbackToPreviousPage = page > 1 && modelsPaged.items.length === 0;
      if (fallbackToPreviousPage) {
        await loadData(page - 1, pageSize);
        return;
      }
      setVehicleModels(modelsPaged.items);
      setTablePagination({
        current: modelsPaged.page || page,
        pageSize: modelsPaged.pageSize || pageSize,
        total: modelsPaged.total,
      });
      setProductionLines(extractListData(linesRes.data));
    } catch (error) {
      console.error('Failed to load data:', error);
    } finally {
      setLoading(false);
    }
  };

  const handleAdd = () => {
    setCurrentModel(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: any) => {
    setCurrentModel(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await api.delete(`/vehicle-models/${id}`);
      message.success('删除成功');
      loadData(tablePagination.current, tablePagination.pageSize);
    } catch (error) {
      console.error('Failed to delete:', error);
    }
  };

  const handleSubmit = async (values: any) => {
    try {
      if (currentModel) {
        await api.put(`/vehicle-models/${currentModel.id}`, values);
        message.success('更新成功');
      } else {
        await api.post('/vehicle-models', values);
        message.success('创建成功');
      }
      setModalVisible(false);
      loadData(tablePagination.current, tablePagination.pageSize);
    } catch (error) {
      console.error('Failed to submit:', error);
    }
  };

  const handleViewPrograms = async (record: any) => {
    setCurrentModel(record);
    setLoading(true);
    try {
      const response = await api.get(`/programs/by-vehicle/${record.id}`);
      setPrograms(response.data);
      setCollapsedLines(productionLines.map((line: any) => line.id));
      setDrawerVisible(true);
    } catch (error) {
      console.error('Failed to load programs:', error);
      message.error('加载程序列表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleViewProgramDetail = async (program: any) => {
    setCurrentProgram(program);
    setLoading(true);
    try {
      const [versionsRes, mappingRes] = await Promise.all([
        api.get(`/files/program/${program.id}`),
        api.get(`/program-mappings/by-child/${program.id}`),
      ]);
      setProgramVersions(versionsRes.data?.versions || []);
      setSharedPrograms(mappingRes.data?.parent_program ? [mappingRes.data.parent_program] : []);
      setProgramDetailVisible(true);
    } catch (error) {
      console.error('Failed to load program detail:', error);
      message.error('加载程序详情失败');
      setProgramVersions([]);
      setSharedPrograms([]);
      setProgramDetailVisible(true);
    } finally {
      setLoading(false);
    }
  };

  const columns = [
    {
      title: '车型名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => (
        <span style={{ color: '#2D3335', fontSize: '14px', fontWeight: 700, fontFamily: 'Inter, sans-serif' }}>
          {text}
        </span>
      ),
    },
    {
      title: '车型编号',
      dataIndex: 'code',
      key: 'code',
      render: (text: string) => (
        <span style={{ color: '#5A6062', fontSize: '14px', fontWeight: 500, fontFamily: 'Inter, sans-serif' }}>
          {text}
        </span>
      ),
    },
    {
      title: '系列',
      dataIndex: 'series',
      key: 'series',
      render: (text: string) => (
        <div style={{ background: '#EBEEF0', borderRadius: '4px', display: 'inline-block', padding: '2px 8px' }}>
          <span style={{ color: '#2D3335', fontSize: '12px', fontWeight: 700, letterSpacing: '0.6px' }}>
            {text || '-'}
          </span>
        </div>
      ),
    },
    {
      title: '操作',
      key: 'action',
      align: 'right' as const,
      render: (_: any, record: any) => (
        <Space size="small">
          <Tooltip title="查看程序">
            <Button
              type="text"
              icon={<EyeOutlined style={{ color: '#5A6062' }} />}
              onClick={() => handleViewPrograms(record)}
              style={{ width: '32px', height: '32px', borderRadius: '4px', background: '#F8F9FA' }}
            />
          </Tooltip>
          <Tooltip title="编辑">
            <Button
              type="text"
              icon={<EditOutlined style={{ color: '#5A6062' }} />}
              onClick={() => handleEdit(record)}
              style={{ width: '32px', height: '32px', borderRadius: '4px', background: '#F8F9FA' }}
            />
          </Tooltip>
          <Popconfirm
            title="确定删除?"
            onConfirm={() => handleDelete(record.id)}
          >
            <Tooltip title="删除">
              <Button 
                type="text"
                icon={<DeleteOutlined style={{ color: '#A83836' }} />} 
                style={{ width: '32px', height: '32px', borderRadius: '4px', background: 'rgba(168, 56, 54, 0.05)' }} 
              />
            </Tooltip>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="management-page">
      {/* 顶部标题区 */}
      <div className="management-page-header">
        <div>
          <div className="management-page-breadcrumb">
            <span>基础数据</span>
            <span style={{ margin: '0 8px', fontFamily: 'Inter, sans-serif' }}>/</span>
            <span className="active">车型管理</span>
          </div>
          <Title level={2} className="management-page-title">
            车型管理
          </Title>
        </div>
        <Space>
          <Button 
            type="primary" 
            icon={<PlusOutlined />} 
            onClick={handleAdd}
            style={{
              background: 'linear-gradient(176deg, #005BC1 0%, #3D89FF 100%)',
              border: 'none',
              boxShadow: '0px 4px 6px -4px rgba(0, 91, 193, 0.10), 0px 10px 15px -3px rgba(0, 91, 193, 0.10)',
              borderRadius: '8px',
              height: '44px',
              padding: '0 24px',
              fontWeight: 600,
              fontSize: '16px'
            }}
          >
            新建车型
          </Button>
        </Space>
      </div>

      {/* 筛选区域 */}
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
            }
          }
        }}
      >
        <div className="management-filter-panel">
          <div className="management-filter-field flex">
            <div className="management-filter-label">车型名称/编号</div>
            <Input 
              placeholder="搜索参数..." 
              value={searchKeyword} 
              onChange={(e) => setSearchKeyword(e.target.value)}
            />
          </div>
          <div className="management-filter-field">
            <div className="management-filter-label">系列</div>
            <Select 
              placeholder="全部" 
              value={filterSeries} 
              onChange={setFilterSeries}
              allowClear
              style={{ width: '100%' }}
            >
              {allSeries.map((series) => (
                <Select.Option key={series} value={series}>
                  {series}
                </Select.Option>
              ))}
            </Select>
          </div>
          <div className="management-filter-field">
            <div className="management-filter-label">创建日期</div>
            <DatePicker.RangePicker 
              style={{ width: '100%' }}
              onChange={(_, dateStrings) => {
                setFilterDateRange([dateStrings[0] || null, dateStrings[1] || null]);
              }}
            />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <Button 
              onClick={() => {}}
              icon={<SearchOutlined />}
              style={{ height: '40px', width: '115px', borderRadius: '8px', background: '#DEE3E6', color: '#2D3335', fontWeight: 700, border: 'none' }}
            >
              查询
            </Button>
            <Button 
              type="text"
              onClick={handleResetFilter} 
              style={{ height: '40px', color: '#005BC1', fontWeight: 700, letterSpacing: '1.2px', border: 'none', background: 'transparent' }}
            >
              重置
            </Button>
          </div>
        </div>
      </ConfigProvider>

      <div className="management-table-card">
        <Table
          className="custom-table"
          columns={columns}
          dataSource={vehicleModels}
          rowKey="id"
          loading={loading}
          pagination={{
            current: tablePagination.current,
            pageSize: tablePagination.pageSize,
            total: tablePagination.total,
            onChange: (page, pageSize) => loadData(page, pageSize),
            showTotal: (total, range) => `显示第 ${range[0]} 至 ${range[1]} 条，共 ${total} 条记录`,
            style: { padding: '16px 24px', margin: 0, background: 'rgba(241, 244, 245, 0.50)' }
          }}
          locale={{
            emptyText: (
              <div style={{ padding: '40px 0' }}>
                <CarOutlined style={{ fontSize: '48px', color: '#d9d9d9', marginBottom: '16px' }} />
                <div style={{ color: '#999', marginBottom: '16px' }}>
                  暂无车型数据
                </div>
                <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
                  创建第一个车型
                </Button>
              </div>
            ),
          }}
        />
      </div>
      <Modal
        title={currentModel ? '编辑车型' : '新建车型'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item name="name" label="车型名称" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="code" label="车型编号" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item name="series" label="系列">
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>

      <Drawer
        title={null}
        placement="right"
        onClose={() => setDrawerVisible(false)}
        open={drawerVisible}
        width={630}
        className="vehicle-program-list-drawer"
        styles={{ body: { padding: 0, background: '#F8FAFC' } }}
      >
        <div style={{ minHeight: '100%', background: '#F8FAFC' }}>
          <div style={{ padding: '24px 24px 16px', borderBottom: '1px solid rgba(241, 244, 245, 0.9)', background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(12px)' }}>
            <div style={{ color: '#2D3335', fontSize: '18px', fontWeight: 700 }}>{currentModel?.name} - 程序列表</div>
            <div style={{ color: '#5A6062', fontSize: '12px', marginTop: '4px' }}>按程序卡片查看当前车型下的已配置程序</div>
          </div>

          <div style={{ padding: '24px', display: 'flex', flexDirection: 'column', gap: '20px' }}>
            {programs.length === 0 ? (
              <div style={{ padding: '64px 0' }}>
                <Empty description="暂无程序数据" image={Empty.PRESENTED_IMAGE_SIMPLE} />
              </div>
            ) : (
              productionLines
                .filter((line: any) => programs.some((p: any) => p.production_line_id === line.id))
                .map((line: any) => {
                  const linePrograms = programs.filter((p: any) => p.production_line_id === line.id);
                  const isCollapsed = collapsedLines.includes(line.id);

                  return (
                    <div key={line.id} style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
                      <button
                        type="button"
                        onClick={() => setCollapsedLines((current) => current.includes(line.id) ? current.filter((id) => id !== line.id) : [...current, line.id])}
                        style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '12px', padding: '10px 12px', borderRadius: '12px', border: '1px solid #E5E9EC', background: '#FFFFFF', cursor: 'pointer' }}
                      >
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <ApartmentOutlined style={{ color: '#005BC1', fontSize: '16px' }} />
                          <span style={{ color: '#2D3335', fontSize: '14px', fontWeight: 700 }}>{line.name}</span>
                          <Badge count={linePrograms.length} style={{ backgroundColor: '#005BC1' }} />
                        </div>
                        <span style={{ color: '#5A6062', fontSize: '12px', fontWeight: 600 }}>{isCollapsed ? '展开' : '收起'}</span>
                      </button>

                      {!isCollapsed && linePrograms.map((program: any) => {
                        const isCompleted = program.status === 'completed';
                        const statusLabel = isCompleted ? '已完成' : '进行中';
                        const statusBackground = isCompleted ? 'rgba(61, 137, 255, 0.2)' : 'rgba(222, 204, 253, 0.4)';
                        const statusColor = isCompleted ? '#005BC1' : '#50426B';

                        return (
                          <div
                            key={program.id}
                            style={{
                              width: '100%',
                              background: '#FFFFFF',
                              borderRadius: '12px',
                              borderLeft: '4px solid #005BC1',
                              boxShadow: '0px 25px 50px -12px rgba(30, 58, 138, 0.05)',
                              display: 'flex',
                              flexDirection: 'column',
                              overflow: 'hidden'
                            }}
                          >
                            <div style={{ padding: '20px 20px 12px 24px', display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                              <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                                <div style={{ width: '40px', height: '40px', background: 'rgba(0, 91, 193, 0.05)', borderRadius: '8px', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
                                  <AppstoreOutlined style={{ color: '#005BC1', fontSize: '20px' }} />
                                </div>
                                <div>
                                  <div style={{ color: '#2D3335', fontSize: '16px', lineHeight: '24px', fontWeight: 700, fontFamily: 'Inter, sans-serif' }}>
                                    {program.name}
                                  </div>
                                  <div style={{ color: '#5A6062', fontSize: '12px', lineHeight: '16px', fontFamily: 'Liberation Mono, monospace' }}>
                                    ID: {program.code}
                                  </div>
                                </div>
                              </div>
                              <div style={{ background: statusBackground, color: statusColor, borderRadius: '8px', padding: '2px 8px', fontSize: '10px', lineHeight: '24px', fontWeight: 500, letterSpacing: '0.5px' }}>
                                {statusLabel}
                              </div>
                            </div>
                            <div style={{ borderTop: '1px solid #F1F4F5', padding: '13px 20px 13px 24px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                              <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                                <span style={{ color: '#5A6062', fontSize: '11px', lineHeight: '24px', fontWeight: 500 }}>当前版本：</span>
                                <div style={{ background: 'rgba(0, 91, 193, 0.05)', borderRadius: '8px', padding: '2px 8px' }}>
                                  <span style={{ color: '#005BC1', fontSize: '11px', lineHeight: '24px', fontWeight: 500 }}>
                                    {program.version || '暂无版本'}
                                  </span>
                                </div>
                              </div>
                              <button
                                type="button"
                                onClick={() => handleViewProgramDetail(program)}
                                style={{ border: 'none', background: 'transparent', color: '#005BC1', fontSize: '12px', fontWeight: 700, cursor: 'pointer', padding: 0 }}
                              >
                                详情
                              </button>
                            </div>
                          </div>
                        );
                      })}
                    </div>
                  );
                })
            )}
          </div>
        </div>
      </Drawer>

      <Drawer
        title={
          <Space>
            <AppstoreOutlined />
            <span>{currentProgram?.name || '程序详情'}</span>
          </Space>
        }
        placement="right"
        onClose={() => setProgramDetailVisible(false)}
        open={programDetailVisible}
        width={520}
      >
        {currentProgram && (
          <Space direction="vertical" size="middle" style={{ width: '100%' }}>
            <div style={{ background: '#F8F9FA', padding: '16px', borderRadius: '12px' }}>
              <div style={{ color: '#5A6062', fontSize: '12px', fontWeight: 700, marginBottom: '8px' }}>程序名称</div>
              <div style={{ color: '#2D3335', fontSize: '18px', fontWeight: 800 }}>{currentProgram.name}</div>
            </div>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px' }}>
              <div style={{ background: '#fff', border: '1px solid #EBEEF0', padding: '14px', borderRadius: '12px' }}>
                <div style={{ color: '#5A6062', fontSize: '12px', fontWeight: 700, marginBottom: '6px' }}>程序编号</div>
                <div style={{ color: '#2D3335', fontSize: '14px', fontWeight: 600 }}>{currentProgram.code || '-'}</div>
              </div>
              <div style={{ background: '#fff', border: '1px solid #EBEEF0', padding: '14px', borderRadius: '12px' }}>
                <div style={{ color: '#5A6062', fontSize: '12px', fontWeight: 700, marginBottom: '6px' }}>当前版本</div>
                <div style={{ color: '#005BC1', fontSize: '14px', fontWeight: 700 }}>{currentProgram.version || '暂无版本'}</div>
              </div>
              <div style={{ background: '#fff', border: '1px solid #EBEEF0', padding: '14px', borderRadius: '12px' }}>
                <div style={{ color: '#5A6062', fontSize: '12px', fontWeight: 700, marginBottom: '6px' }}>生产线</div>
                <div style={{ color: '#2D3335', fontSize: '14px', fontWeight: 600 }}>{currentProgram.production_line?.name || '-'}</div>
              </div>
              <div style={{ background: '#fff', border: '1px solid #EBEEF0', padding: '14px', borderRadius: '12px' }}>
                <div style={{ color: '#5A6062', fontSize: '12px', fontWeight: 700, marginBottom: '6px' }}>状态</div>
                <div>
                  <Tag color={currentProgram.status === 'completed' ? 'blue' : 'purple'}>
                    {currentProgram.status === 'completed' ? '已完成' : '进行中'}
                  </Tag>
                </div>
              </div>
            </div>

            <div style={{ background: '#fff', border: '1px solid #EBEEF0', padding: '16px', borderRadius: '12px' }}>
              <div style={{ color: '#5A6062', fontSize: '12px', fontWeight: 700, marginBottom: '8px' }}>程序描述</div>
              <div style={{ color: '#2D3335', fontSize: '14px', lineHeight: 1.7 }}>
                {currentProgram.description || '暂无描述'}
              </div>
            </div>

            <div style={{ background: '#fff', border: '1px solid #EBEEF0', padding: '16px', borderRadius: '12px' }}>
              <div style={{ color: '#5A6062', fontSize: '12px', fontWeight: 700, marginBottom: '10px' }}>版本信息</div>
              {programVersions.length > 0 ? (
                <Space direction="vertical" size="small" style={{ width: '100%' }}>
                  {programVersions.slice(0, 3).map((version: any) => (
                    <div key={version.version} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px 12px', background: '#F8F9FA', borderRadius: '10px' }}>
                      <div>
                        <div style={{ color: '#2D3335', fontSize: '13px', fontWeight: 700 }}>{version.version}</div>
                        <div style={{ color: '#5A6062', fontSize: '12px' }}>{version.change_log || '暂无版本说明'}</div>
                      </div>
                      {version.is_current ? <Tag color="blue">当前版本</Tag> : <Tag>历史版本</Tag>}
                    </div>
                  ))}
                </Space>
              ) : (
                <Text type="secondary">暂无版本记录</Text>
              )}
            </div>


            <div style={{ background: '#fff', border: '1px solid #EBEEF0', padding: '16px', borderRadius: '12px' }}>
              <div style={{ color: '#5A6062', fontSize: '12px', fontWeight: 700, marginBottom: '10px' }}>关联程序</div>
              {sharedPrograms.length > 0 ? (
                <Space direction="vertical" size="small" style={{ width: '100%' }}>
                  {sharedPrograms.map((program: any) => (
                    <div key={program.id} style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '10px 12px', background: '#F8F9FA', borderRadius: '10px' }}>
                      <div>
                        <div style={{ color: '#2D3335', fontSize: '13px', fontWeight: 700 }}>{program.name}</div>
                        <div style={{ color: '#5A6062', fontSize: '12px' }}>{program.code} / {program.production_line?.name || '-'}</div>
                      </div>
                      <Tag color={program.status === 'completed' ? 'blue' : 'purple'}>
                        {program.status === 'completed' ? '已完成' : '进行中'}
                      </Tag>
                    </div>
                  ))}
                </Space>
              ) : (
                <Text type="secondary">暂无关联程序</Text>
              )}
            </div>

            <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
              <Button type="primary" onClick={() => window.location.href = `/programs?keyword=${encodeURIComponent(currentProgram.name)}`}>前往程序管理</Button>
            </div>
          </Space>
        )}
      </Drawer>
    </div>
  );
};

export default VehicleModelManagement;
