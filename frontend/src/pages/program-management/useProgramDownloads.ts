import { message } from 'antd';
import { programApi } from './programApi';
import type { Program } from './types';

interface UseProgramDownloadsOptions {
  currentProgram: Program | null;
  versionsPage: number;
  versionsPageSize: number;
  loadVersions: (
    programId: number,
    page?: number,
    pageSize?: number,
  ) => Promise<unknown>;
}

const downloadFileUrl = (fileId: number) => `/files/download/${fileId}`;
const latestProgramArchiveUrl = (programId: number) =>
  `/files/download/program/${programId}/latest`;
const versionArchiveUrl = (programId: number, version: string) =>
  `/files/download/version/${version}?program_id=${programId}`;

export const useProgramDownloads = ({
  currentProgram,
  versionsPage,
  versionsPageSize,
  loadVersions,
}: UseProgramDownloadsOptions) => {
  const downloadWithAuth = async (url: string, fallbackName: string) => {
    const response = await programApi.downloadBlob(url);
    const blob = new Blob([response.data]);

    const contentDisposition = response.headers['content-disposition'];
    let filename = fallbackName;
    if (contentDisposition) {
      const match = /filename\*=UTF-8''([^;]+)|filename="?([^";]+)"?/i.exec(
        contentDisposition,
      );
      const encodedName = match?.[1] || match?.[2];
      if (encodedName) {
        filename = decodeURIComponent(encodedName);
      }
    }

    const urlObject = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = urlObject;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    link.remove();
    window.URL.revokeObjectURL(urlObject);
  };

  const handleDownload = async (record: Program) => {
    try {
      const versionResult = await programApi.listProgramVersions(
        record.id,
        1,
        20,
      );
      const versions = versionResult.versions;
      if (versions.length === 0) {
        message.warning('该程序暂无上传的文件');
        return;
      }

      const latestVersion =
        versions.find((version) => version.is_current) || versions[0];
      const files = latestVersion?.files || [];
      if (files.length > 0) {
        if (files.length === 1) {
          const file = files[0];
          await downloadWithAuth(downloadFileUrl(file.id), file.file_name);
        } else {
          await downloadWithAuth(
            latestProgramArchiveUrl(record.id),
            `${record.code || record.id}_${latestVersion.version}.zip`,
          );
          message.success('正在打包下载最新版本的所有文件...');
        }
      } else {
        message.warning('该程序暂无可用文件');
      }
    } catch (error) {
      console.error('Failed to download:', error);
      message.error('下载失败');
    }
  };

  const handleDownloadFile = async (fileId: number, fileName: string) => {
    await downloadWithAuth(downloadFileUrl(fileId), fileName);
  };

  const handleDownloadVersion = async (program: Program, version: string) => {
    await downloadWithAuth(
      versionArchiveUrl(program.id, version),
      `${program.code || program.id}_${version}.zip`,
    );
  };

  const handleDeleteSingleFile = async (fileId: number) => {
    try {
      await programApi.deleteFile(fileId);
      message.success('文件删除成功');
      if (currentProgram) {
        await loadVersions(currentProgram.id, versionsPage, versionsPageSize);
      }
    } catch (error: any) {
      console.error('Failed to delete file:', error);
      message.error(error.response?.data?.error || '删除文件失败');
    }
  };

  return {
    downloadWithAuth,
    handleDownload,
    handleDownloadFile,
    handleDownloadVersion,
    handleDeleteSingleFile,
  };
};
