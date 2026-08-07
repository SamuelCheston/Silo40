# Silo40 Strategy Game Backend

Silo40 是一款基于《羊毛战记》世界观的深度策略游戏后端。游戏核心围绕地堡（Silo）的生存、思潮演化以及在末世背景下的多路径文明存续。

## 📖 文档说明
- [设计意图 (Thinking)](docs/thinking.md): 详细描述了游戏的背景设定、思潮机制与胜利路径。
- [开发路线图 (Roadmap)](docs/roadmap.md): 本项目的技术实现计划与阶段目标。
- [已实现功能 (Features)](docs/features.md): 记录当前系统已完成的数值逻辑与核心功能。

## 🚀 核心机制
- **思潮系统**: 动态追踪不同职业群体的亲外/排外倾向。
- **事件驱动**: 资源生产与逻辑更新由玩家行为和时间差触发，非定时轮询。
- **多结局引擎**: 支持信息胜利、时间胜利、叛乱幸存等多种博弈结局。

## 🛠 开发指南
本项目使用 Go + Gin + PostgreSQL 构建。

### 快速启动
```bash
go mod download
go run cmd/server/main.go
```

### 接口测试
基础健康检查: `GET /api/v1/health`
