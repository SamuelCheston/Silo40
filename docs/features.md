# Silo40 Implemented Features

本文档记录了 Silo40 已实现的数值逻辑与系统功能。本项目采用“逻辑归属前端，数据持久化归属后端”的开发模式。

## 1. 核心数据模型 (Core Models)
已实现严格对齐 `thinking.md` 的数据模型：
- **Silo (地堡)**: 存储全局状态（合法性、凝聚力、叛乱值、历史包袱、500年倒计时、信息碎片）。
- **Profession (部门)**: 存储各部门人口（总计1万）、思潮值 (0-1)、恐慌值、生产力系数。
- **Resource (资源)**: 存储 Food, Energy, Water, Materials 的库存及 `NetBalance` (结余)。
- **Floor (楼层)**: 存储 144 层物理空间的职能分布与稳定性。
- **Agent (特工)**: 存储玩家角色的职业、特质、政治威望及点数。

## 2. 前端游戏引擎 (Frontend Game Engine)
核心逻辑已由 TypeScript 实现，位于 `frontend/src/logic/`：
- **地堡初始化 (Initializer)**: 
    - 实现了万人规模的部门权重分配。
    - 严格按照设定初始化 144 层物理层级（行政、农耕、机械等）。
- **数值演化逻辑 (Engine)**:
    - **事件驱动**: 所有更新基于 `deltaYears` 增量结算。
    - **资源动态平衡**: 根据恐慌部门数量及叛乱状态计算资源净结余。
    - **社会稳定性**: 联动合法性与恐慌值计算叛乱风险。
    - **特工政治系统**: 计算威望产出及政治点数累积。

## 3. 后端抽象 API (Backend Abstract API)
后端使用 Go + Wails 实现，仅提供底层数据支持：
- **数据库持久化 (SQLite)**: 
    - 使用 GORM 实现 `User`, `Agent`, `Silo` 等模型的 CRUD 操作。
    - 提供抽象的 `SaveSilo`, `GetSilo`, `SaveAgent` 等接口。
- **高性能缓存 (FreeCache)**:
    - 提供抽象的 `SetCache`, `GetCache` 接口，用于存储临时游戏状态。
- **自动迁移**: 支持数据库模型与物理表的自动同步。

## 4. UI 系统 (Frontend UI)
- **技术栈**: React + Chakra UI。
- **核心组件**:
    - `TimeWheel`: 沉浸式的时间选择/初始化组件。
    - `TimeWheel.tsx`: 实现平滑的旋转交互与数值选择。
