import { useEffect, useState } from 'react';
import { message } from 'antd';
import { programApi } from './programApi';
import type {
  BatchImportPreview,
  BatchImportStatus,
  WorkstationMapping,
} from './types';

interface UseProgramBatchImportOptions {
  loadData: () => Promise<void>;
}

export const useProgramBatchImport = ({
  loadData,
}: UseProgramBatchImportOptions) => {
  const [batchImportSupported, setBatchImportSupported] = useState(true);
  const [batchImportVisible, setBatchImportVisible] = useState(false);
  const [batchImportStep, setBatchImportStep] = useState(0);
  const [batchImportPreview, setBatchImportPreview] =
    useState<BatchImportPreview | null>(null);
  const [workstationMappings, setWorkstationMappings] = useState<
    WorkstationMapping[]
  >([]);
  const [batchImportStatus, setBatchImportStatus] =
    useState<BatchImportStatus | null>(null);
  const [batchImportPolling, setBatchImportPolling] = useState<ReturnType<
    typeof setInterval
  > | null>(null);
  const [uploadLoading, setUploadLoading] = useState(false);

  useEffect(
    () => () => {
      if (batchImportPolling) {
        clearInterval(batchImportPolling);
      }
    },
    [batchImportPolling],
  );

  const handleBatchImport = () => {
    if (!batchImportSupported) {
      message.warning('当前环境未启用批量导入接口');
      return;
    }
    setBatchImportVisible(true);
    setBatchImportStep(0);
    setBatchImportPreview(null);
    setWorkstationMappings([]);
    setBatchImportStatus(null);
  };

  const handleBatchUpload = async (file: File) => {
    setUploadLoading(true);
    try {
      const preview = await programApi.uploadBatchArchive(file);
      setBatchImportPreview(preview);
      setWorkstationMappings(
        preview.workstations.map((ws) => ({
          workstation_name: ws.name,
          production_line_id: null,
          vehicle_model_id: null,
        })),
      );
      setBatchImportStep(1);
      message.success('文件解析成功');
    } catch (error: any) {
      console.error('Failed to upload:', error);
      if (error?.response?.status === 404 || error?.response?.status === 405) {
        setBatchImportSupported(false);
        message.warning('批量导入接口未启用');
      } else {
        message.error(error.response?.data?.error || '上传失败');
      }
    } finally {
      setUploadLoading(false);
    }
  };

  const handleMappingChange = (
    workstationName: string,
    field: string,
    value: unknown,
  ) => {
    setWorkstationMappings((prev) =>
      prev.map((m) =>
        m.workstation_name === workstationName ? { ...m, [field]: value } : m,
      ),
    );
  };

  const startBatchImportPolling = (taskId: number) => {
    if (batchImportPolling) clearInterval(batchImportPolling);

    const interval = setInterval(async () => {
      try {
        const status = await programApi.getTaskStatus(taskId);
        setBatchImportStatus(status);

        if (status.status === 'completed' || status.status === 'failed') {
          clearInterval(interval);
          setBatchImportPolling(null);
          if (status.status === 'completed') {
            message.success('批量导入完成');
            void loadData();
          }
        }
      } catch (error: any) {
        console.error('Failed to poll import status:', error);
        if (
          error?.response?.status === 404 ||
          error?.response?.status === 405
        ) {
          clearInterval(interval);
          setBatchImportPolling(null);
          setBatchImportSupported(false);
          message.warning('任务状态接口未启用');
        }
      }
    }, 2000);

    setBatchImportPolling(interval);
  };

  const handleConfirmBatchImport = async () => {
    const invalidMappings = workstationMappings.filter(
      (m) => !m.production_line_id,
    );
    if (invalidMappings.length > 0) {
      message.error('请为所有工位选择生产线');
      return;
    }

    setBatchImportStep(2);

    try {
      const response = await programApi.startBatchImport(
        batchImportPreview?.preview_id,
        workstationMappings,
      );
      message.success('批量导入已开始');
      startBatchImportPolling(response.task_id);
    } catch (error: any) {
      console.error('Failed to start batch import:', error);
      if (error?.response?.status === 404 || error?.response?.status === 405) {
        setBatchImportSupported(false);
        message.warning('批量导入接口未启用');
        setBatchImportStep(0);
      } else {
        message.error(error.response?.data?.error || '启动导入失败');
        setBatchImportStep(1);
      }
    }
  };

  const handleCloseBatchImport = () => {
    if (batchImportPolling) {
      clearInterval(batchImportPolling);
    }
    setBatchImportVisible(false);
    setBatchImportStep(0);
    setBatchImportPreview(null);
    setWorkstationMappings([]);
    setBatchImportStatus(null);
  };

  return {
    batchImportSupported,
    batchImportVisible,
    batchImportStep,
    setBatchImportStep,
    batchImportPreview,
    workstationMappings,
    batchImportStatus,
    uploadLoading,
    handleBatchImport,
    handleBatchUpload,
    handleMappingChange,
    handleConfirmBatchImport,
    handleCloseBatchImport,
  };
};
