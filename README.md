# invest-stock

一个 Go 实现的股票分析与回测项目，包含：

- Web 看板与 REST API
- 用户注册、登录、JWT 鉴权和管理员用户管理
- 单标的回测、策略对比和组合回测
- 技术指标计算、行情数据缓存和实时监控 WebSocket

## 项目结构

```text
cmd/
  analyze/        命令行分析入口
  monitor/        命令行监控入口
  signal/         命令行信号输出入口
  web/            Web 服务、页面、API、静态资源
internal/
  auth/           JWT 与密码哈希
  model/          用户模型与存储
  store/          SQLite 初始化与迁移
pkg/
  backtest/       回测引擎、组合回测、策略注册
  data/           行情数据加载、HTTP 重试、K 线缓存
  indicator/      MA、RSI、MACD、BOLL 等指标
  strategy/       信号与策略逻辑
```

## 本地运行

1. 准备环境变量：

```sh
cp .env.example .env
set -a
. ./.env
set +a
```

2. 设置一个安全的 JWT secret：

```sh
export INVEST_JWT_SECRET="replace-with-a-random-secret-at-least-32-characters"
```

3. 启动 Web 服务：

```sh
go run ./cmd/web
```

默认监听 `http://localhost:8081`，运行时 SQLite 数据库会写入 `data/invest.db`。

## 配置项

| 变量 | 必填 | 默认值 | 说明 |
| --- | --- | --- | --- |
| `INVEST_JWT_SECRET` | 是 | 无 | JWT 签名密钥，至少 32 字符 |
| `INVEST_ADDR` | 否 | `:8081` | Web 服务监听地址 |
| `INVEST_KLINE_TTL` | 否 | `24h` | K 线缓存 TTL，范围 `[1s,24h]` |
| `INVEST_ADMIN_USERNAME` | 否 | 无 | 初始管理员用户名 |
| `INVEST_ADMIN_PASSWORD` | 否 | 无 | 初始管理员密码，至少 8 字符 |

## 测试与质量门禁

本地提交前建议运行：

```sh
make preflight
```

项目 CI 会在 `main` 分支的 push 和 pull request 上执行测试、静态分析、
race detector 和无凭据镜像构建。

## 构建与发布

```sh
make build
make docker-build VERSION=v1.0.0
make release-check VERSION=v1.0.0
git tag v1.0.0 && git push origin v1.0.0
```

标签流水线发布带 SBOM、provenance、语义版本和 Git SHA 的 GHCR 镜像，但不会
连接行情或券商账户，也不会自动部署。运行时 SQLite 必须挂载 `/app/data`；
镜像更新不会替代数据库备份，部署前仍需单独备份 `data/invest.db` 及其
WAL/SHM 文件。

生产服务器只拉取应用镜像，完整步骤见
[Docker 部署](docs/deployment.md)。Compose 不启动数据库、缓存或其他基础设施。

## 维护约定

- API 鉴权只接受 `Authorization: Bearer <token>`，避免把 token 放入 URL。
- WebSocket 鉴权通过 `Sec-WebSocket-Protocol: bearer,<token>` 传递 token。
- 外部行情请求必须有超时、重试和非 2xx 状态处理。
- 回测成本和配置参数必须拒绝 NaN、Inf 和非法范围。
- 新增策略应通过 `pkg/backtest` 的注册机制声明参数，避免硬编码到 Web 层。
- JWT secret 只通过环境或秘密文件提供，不写入镜像。
