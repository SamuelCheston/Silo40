# 公开仓库内容边界

本仓库只发布项目源码、项目原创机制与文案，以及已确认属于小说的事实层材料。任何影视改编独有的人名、组织名、情节、工艺、场景、对照研究笔记与外部资料汇编均不得入库。自动扫描是发布门禁，不构成法律意见或绝对版权保证。

`docs/research/01-worldbuilding.md` 至 `06-silo40-event-plan.md` 属于公开文档，已按抽象化口径改写：只保留设定结构、数值关系与机制位，不写具体人名、组织名、集数或纪年。它们不再列入本地专用路径；扫描器仍对其执行内容级检查。

## 只存本地、不得入库

以下路径由 `.gitignore` 按路径拦截：

- `docs/research/local/`
- `docs/local/`
- `references/local/`
- `frontend/src/references/`
- `/research/`
- `/notes/`
- `/.local-references/`
- `/local-references/`

`.trae/`、`outputs/`、`build/`、`.playable-data/`、`.screenshots/`、前端依赖与构建目录、数据库和编辑器文件同样只保留在本地。仓库外的参考资料不因复制发布工作树而获得入库资格。

## 内容级检查

`.gitignore` 只控制路径，不能识别文件内容。`scripts/check_public_content.py` 因此独立检查禁用名称、专有情节点、工艺表述、改编纪年、已移除事件和本地专用路径；扫描器不依赖第三方 Python 包，发现一项即非零退出。

```bash
# 首次发布前，扫描整个工作树（含尚未跟踪的文件）
python3 scripts/check_public_content.py --all

# 日常检查，扫描 Git 已跟踪文件；无 .git 时仍扫描整个工作树
python3 scripts/check_public_content.py

# 验证扫描器自身的拦截与放行行为
python3 scripts/check_public_content.py --self-test
```

现有 GitHub Actions 在依赖安装与构建前执行自检和全量扫描。新增禁用规则时，必须先运行自检，再运行全量扫描；不得通过删除规则、跳过扫描或忽略失败来放行。

## 回流处理

发现残留时，从公开工作树删除或改写为不含专有构件的原创抽象机制，再重新运行两种扫描。原始研究材料只能留在仓库外的本地位置，不得为了保留出处、对照或否定性说明而复制进公开仓库。
