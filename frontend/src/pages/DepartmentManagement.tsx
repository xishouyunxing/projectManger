import { useEffect, useState } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  message,
  Typography,
  Tag,
  Popconfirm,
} from 'antd';
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons';
import api from '../services/api';

const { Title } = Typography;
const { TextArea } = Input;

const DepartmentManagement = () => {
  const [departments, setDepartments] = useState([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [currentDepartment, setCurrentDepartment] = useState<any>(null);
  const [form] = Form.useForm();

  useEffect(() => {
    loadData();
  }, []);

  const loadData = async () => {
    setLoading(true);
    try {
      const response = await api.get('/departments');
      setDepartments(response.data);
    } catch (error) {
      console.error('Failed to load departments:', error);
      message.error('加载部门列表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleAdd = () => {
    setCurrentDepartment(null);
    form.resetFields();
    setModalVisible(true);
  };

  const handleEdit = (record: any) => {
    setCurrentDepartment(record);
    form.setFieldsValue(record);
    setModalVisible(true);
  };

  const handleDelete = async (id: number) => {
    try {
      await api.delete(`/departments/${id}`);
      message.success('删除成功');
      loadData();
    } catch (error) {
      console.error('Failed to delete department:', error);
      message.error('删除失败');
    }
  };

  const handleSubmit = async (values: any) => {
    try {
      if (currentDepartment) {
        await api.put(`/departments/${currentDepartment.id}`, values);
        message.success('更新成功');
      } else {
        await api.post('/departments', values);
        message.success('创建成功');
      }
      setModalVisible(false);
      loadData();
    } catch (error) {
      console.error('Failed to submit department:', error);
      message.error('操作失败');
    }
  };

  const columns = [
    {
      title: '部门名称',
      dataIndex: 'name',
      key: 'name',
      render: (text: string) => (
        <span style={{ color: '#2D3335', fontSize: '14px', fontWeight: 700, fontFamily: 'Inter, sans-serif' }}>
          {text}
        </span>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'active' ? 'green' : 'red'}>
          {status === 'active' ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      render: (time: string) => {
        const dateObj = new Date(time);
        const dateFmt = dateObj.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' }).replace(/\//g, '-');
        const timeFmt = dateObj.toLocaleTimeString('zh-CN', { hour12: false });
        return (
          <div style={{ display: 'flex', flexDirection: 'column' }}>
            <span style={{ color: '#5A6062', fontSize: '12px', fontWeight: 500, fontFamily: 'Inter, sans-serif' }}>{dateFmt}</span>
            <span style={{ color: '#5A6062', fontSize: '12px', fontWeight: 400, fontFamily: 'Inter, sans-serif', opacity: 0.6 }}>{timeFmt}</span>
          </div>
        );
      },
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: any) => (
        <Space>
          <Button
            type="link"
            size="small"
            onClick={() => handleEdit(record)}
            style={{ padding: 0 }}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定删除?"
            onConfirm={() => handleDelete(record.id)}
          >
            <Button type="link" danger icon={<DeleteOutlined />} size="small" style={{ padding: 0 }}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '0 8px', maxWidth: '1024px', margin: '0 auto', fontFamily: '"WenQuanYi Zen Hei", Inter, Manrope, sans-serif' }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-end', marginTop: '24px' }}>
        <div>
          <div style={{ display: 'flex', alignItems: 'center', marginBottom: '8px' }}>
            <span style={{ color: '#5A6062', fontSize: '12px', fontWeight: 700, letterSpacing: '1.2px' }}>系统</span>
            <span style={{ color: '#5A6062', fontSize: '12px', fontWeight: 700, margin: '0 8px', fontFamily: 'Inter, sans-serif' }}>/</span>
            <span style={{ color: '#005BC1', fontSize: '12px', fontWeight: 700, letterSpacing: '1.2px' }}>部门管理</span>
          </div>
          <Title level={2} style={{ margin: 0, color: '#2D3335', fontSize: '30px', fontWeight: 800 }}>
            部门管理
          </Title>
        </div>
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
          新建部门
        </Button>
      </div>
      <div style={{ marginTop: '32px', background: '#fff', borderRadius: '16px', boxShadow: '0px 12px 40px rgba(0, 91, 193, 0.03)', overflow: 'hidden' }}>
        <Table
          className="custom-table"
          columns={columns}
          dataSource={departments}
          rowKey="id"
          loading={loading}
          pagination={{
            showTotal: (total, range) => `显示第 ${range[0]} 至 ${range[1]} 条，共 ${total} 条记录`,
            style: { padding: '16px 24px', margin: 0, background: 'rgba(241, 244, 245, 0.50)' }
          }}
        />
      </div>

      <Modal
        title={currentDepartment ? '编辑部门' : '新建部门'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        onOk={() => form.submit()}
      >
        <Form form={form} layout="vertical" onFinish={handleSubmit}>
          <Form.Item
            name="name"
            label="部门名称"
            rules={[{ required: true, message: '请输入部门名称' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="description" label="描述">
            <TextArea rows={4} />
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue="active">
            <Select>
              <Select.Option value="active">启用</Select.Option>
              <Select.Option value="inactive">禁用</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>
      {/* 注入表格自定义样式 */}
      <style>{`
        .custom-table .ant-table-thead > tr > th {
          background: #EBEEF0 !important;
          color: #5A6062 !important;
          font-size: 10px !important;
          font-weight: 700 !important;
          letter-spacing: 1px !important;
          border-bottom: 1px solid #DEE3E6 !important;
          padding: 16px 24px !important;
        }
        .custom-table .ant-table-tbody > tr > td {
          padding: 16px 24px !important;
          border-bottom: 1px solid #EBEEF0 !important;
        }
        .custom-table .ant-table {
          border-radius: 16px 16px 0 0 !important;
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

export default DepartmentManagement;
