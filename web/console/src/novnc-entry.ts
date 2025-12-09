// noVNC本地入口文件
// 从node_modules导入，避免CDN依赖
// @ts-expect-error - noVNC没有TypeScript类型定义
import RFB from '@novnc/novnc/lib/rfb.js';

// 扩展Window类型
declare global {
  interface Window {
    RFB: typeof RFB;
  }
}

// 导出RFB供novnc.html使用
window.RFB = RFB;
