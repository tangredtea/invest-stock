# Docker 部署

生产环境只部署项目自身的应用镜像。行情和券商接口均由 `.env` 连接外部服务；
SQLite 数据保存在宿主机 `data/`，Compose 不启动数据库或缓存容器。

## 发布

推送语义版本 Tag 后，`.github/workflows/release-image.yml` 运行质量门禁并发布：

- `ghcr.io/tangredtea/invest-stock:vX.Y.Z`
- `ghcr.io/tangredtea/invest-stock:sha-<完整提交 SHA>`

生产部署必须把 `INVEST_STOCK_IMAGE` 固定到仓库返回的 `sha256` digest。

## 启动

服务器只需要 `.env`、`compose.production.yml` 和持久化 `data/`：

```bash
install -d -m 0700 data
export INVEST_STOCK_IMAGE='ghcr.io/tangredtea/invest-stock@sha256:<digest>'
docker compose -f compose.production.yml config --quiet
docker compose -f compose.production.yml pull
docker compose -f compose.production.yml up -d
docker compose -f compose.production.yml ps
```

升级前停止写入并一致性备份 `data/invest.db` 及其 WAL/SHM 文件；验证新容器健康和
OCI revision 后再清理旧镜像。

