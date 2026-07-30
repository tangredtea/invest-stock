# invest-stock

一个 Go 实现的股票分析与回测项目，提供 Web 看板、行情分析、策略信号和组合回测。

## 本地运行

```sh
cp .env.example .env
export INVEST_JWT_SECRET="replace-with-a-random-secret-at-least-32-characters"
go run ./cmd/web
```

默认监听 `http://localhost:8081`，SQLite 数据写入 `data/invest.db`。

## 测试、构建与发布

```sh
make preflight
make build
make docker-build VERSION=v1.0.0
make release-check VERSION=v1.0.0
git tag v1.0.0 && git push origin v1.0.0
```

CI 只运行测试、静态分析、race detector 和无凭据镜像构建。标签流水线发布带
SBOM、provenance、语义版本和 Git SHA 的 GHCR 镜像，但不会连接行情或券商账户，
也不会自动部署。运行时 SQLite 必须挂载 `/app/data`；镜像更新不会替代数据库
备份，部署前仍需单独备份 `data/invest.db` 及其 WAL/SHM 文件。

## 维护约定

- JWT secret 只通过环境或秘密文件提供，不写入镜像。
- 外部行情请求必须设置超时、重试并检查非 2xx 响应。
- 回测参数必须拒绝 NaN、Inf 和非法范围。
- 新增策略通过 `pkg/backtest` 注册，不在 Web 层硬编码。
