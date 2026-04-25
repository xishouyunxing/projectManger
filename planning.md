# 前端包体优化计划（2026-04-25 第四轮）

## 1. 目标

基于上一轮已消除 circular/empty chunk warning 的结果，继续执行前端包体优化，优先解决：
- 仍然过大的公共 `vendor` chunk 影响首屏缓存与下载体积
- 现有手工分包过于粗粒度，导致大量本可跟随懒加载页面延迟加载的依赖被提前合并进公共包
- 在不破坏现有路由懒加载和构建稳定性的前提下进一步改善产物结构

执行过程中持续回写 `task.md`，并在每次回写后重新读取，按最新 `Next Step` 推进。

## 2. 本轮问题清单

### N1. 公共 vendor chunk 仍然过大
- 位置：`frontend/vite.config.ts`、`frontend/src/App.tsx`
- 现象：构建虽已无 circular/empty chunk warning，但仍生成约 1.16 MB 的公共 `vendor` chunk
- 风险：首屏下载压力偏大，缓存命中收益下降，页面级懒加载价值被削弱
- 级别：P2
- 整改目标：
  - 让更多页面依赖跟随路由 chunk 延迟加载
  - 保留基础稳定公共包，避免重新引入循环分包问题
  - 将构建 warning 收敛为更合理的包体结构结果

## 3. 执行顺序

### 阶段 A：计划与状态同步
1. 重写 `planning.md`
2. 重写 `task.md`
3. 回读 `task.md`，按最新 `Next Step` 执行

### 阶段 B：前端包体结构优化
1. 审视当前 `manualChunks` 是否反而阻碍了路由级拆包
2. 调整 `vite.config.ts`，优先采用更少干预的分包策略
3. 如有必要，补充局部懒加载或共享依赖拆分

### 阶段 C：验证与收尾
1. 运行 `npm --prefix frontend run build`
2. 必要时运行 `go -C backend test ./...` 做回归确认
3. 回写 `task.md` 的优化结果、构建结果和剩余观察项

## 4. 验收标准

- 前端构建通过
- 不重新出现 circular chunk / empty chunk warning
- 公共大包显著缩小，更多依赖下沉到按路由加载的 chunk
- 如仍有大 chunk 提示，需明确定位剩余来源与后续优化方向