# Silo40 Strategy Game

Silo40 是一款基于《羊毛战记》世界观的深度策略游戏。游戏核心围绕地堡（Silo）的生存、思潮演化以及在末世背景下的多路径文明存续。

## 🏗 架构设计
本项目采用 **前端驱动 (Logic-in-Frontend)** 的架构模式：
- **后端 (Go/Wails)**: 提供抽象的数据库和缓存操作接口，不包含具体游戏逻辑。
- **前端 (React/TS)**: 包含完整的游戏引擎、数值演化逻辑、初始化配置及 UI 展现。
- **通信**: 通过 Wails 桥接技术实现前后端的高效调用。

## 📖 文档说明
- [设计意图 (Thinking)](frontend/src/references/thinking.md): 详细描述了游戏的背景设定、思潮机制与胜利路径。
- [开发路线图 (Roadmap)](docs/roadmap.md): 本项目的技术实现计划与阶段目标。
- [已实现功能 (Features)](docs/features.md): 记录当前系统已完成的数值逻辑与核心功能。

## 🚀 核心机制
- **思潮系统**: 动态追踪不同职业群体的亲外/排外倾向。
- **事件驱动**: 资源生产与逻辑更新由玩家行为和时间差触发，非定时轮询。
- **多结局引擎**: 支持信息胜利、时间胜利、叛乱幸存等多种博弈结局。

## 🛠 开发指南
本项目使用 Wails (Go + React + Vite) + SQLite 构建。

### 环境准备
1. 安装 [Wails CLI](https://wails.io/docs/gettingstarted/installation)
2. Go 1.21+
3. Node.js 18+

### 快速启动
```bash
wails dev
```
