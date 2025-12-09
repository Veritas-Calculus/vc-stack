# VC Console (Vite + React + Tailwind)

Oxide Console 风格的仪表盘，集成网络、计算、存储、项目与通知模块。当前为前端骨架与占位页面，后续将对接现有后端 API。

## 开发

```bash
npm install
npm run dev
```

访问 <http://localhost:5173>

## 构建

```bash
npm run build
npm run preview
```

## 测试

```bash
npm run test
```

## 环境变量

创建 `.env` 或使用示例：

```
VITE_API_BASE_URL=https://api.example.com
```

## 目录

- `src/components`：布局和通用组件
- `src/features/*`：功能模块路由与页面
- `src/lib`：API 客户端与状态管理（待添加）

## 注意

- 部分功能（如 VPC/LB/安全组、K8S/镜像/快照、存储后端、项目/通知）目前为 UI 占位，需等后端接口整理后接入。
