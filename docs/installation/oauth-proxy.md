# OAuth 代理配置

OAuth 登录和绑定账号时，后端需要请求第三方 OAuth 服务的 token 接口和用户信息接口。例如 GitHub 登录会请求：

- `https://github.com/login/oauth/access_token`
- `https://api.github.com/user`

如果服务器所在网络无法直连这些地址，可以只为 OAuth 请求配置代理，避免影响模型中转、图片拉取、同步任务等其它后端 HTTP 请求。

## 代理优先级

OAuth 请求按以下顺序选择代理：

```text
{PROVIDER}_OAUTH_PROXY > OAUTH_PROXY > HTTP_PROXY/HTTPS_PROXY/NO_PROXY > 直连
```

说明：

- `{PROVIDER}_OAUTH_PROXY` 是某个 OAuth provider 的专用代理，优先级最高。
- `OAUTH_PROXY` 是所有 OAuth provider 共用的代理。
- 如果前两者都没有配置，则 fallback 到 Go 标准库支持的 `HTTP_PROXY`、`HTTPS_PROXY`、`NO_PROXY` 环境变量。
- 环境变量名同时兼容小写形式，例如 `github_oauth_proxy`、`oauth_proxy`、`https_proxy`。

## 支持的代理格式

```bash
http://127.0.0.1:7890
https://127.0.0.1:7890
socks5://127.0.0.1:1080
socks5h://127.0.0.1:1080
```

带认证的代理示例：

```bash
OAUTH_PROXY=http://username:password@127.0.0.1:7890
```

## 常用配置

只让 GitHub OAuth 走代理：

```bash
GITHUB_OAUTH_PROXY=http://127.0.0.1:7890
```

让所有 OAuth provider 走同一个代理：

```bash
OAUTH_PROXY=http://127.0.0.1:7890
```

只配置标准全局代理时，OAuth 也会 fallback 使用它：

```bash
HTTPS_PROXY=http://127.0.0.1:7890
HTTP_PROXY=http://127.0.0.1:7890
NO_PROXY=localhost,127.0.0.1,::1
```

## Provider 专用变量名

内置 provider 的专用代理变量：

| Provider | 环境变量 |
| --- | --- |
| GitHub | `GITHUB_OAUTH_PROXY` |
| Discord | `DISCORD_OAUTH_PROXY` |
| OIDC | `OIDC_OAUTH_PROXY` |
| Linux DO | `LINUX_DO_OAUTH_PROXY` |

自定义 OAuth provider 使用 slug 生成变量名：

```text
github-enterprise -> GITHUB_ENTERPRISE_OAUTH_PROXY
my_oidc -> MY_OIDC_OAUTH_PROXY
```

## Docker 部署

如果代理运行在宿主机上，容器内的 `127.0.0.1` 指向容器自身，不是宿主机。Linux Docker 可以使用 `host.docker.internal` 并添加 `extra_hosts`：

```yaml
services:
  new-api:
    environment:
      - GITHUB_OAUTH_PROXY=http://host.docker.internal:7890
      # 或者为所有 OAuth provider 配置：
      # - OAUTH_PROXY=http://host.docker.internal:7890
    extra_hosts:
      - "host.docker.internal:host-gateway"
```

配置后需要重启后端服务。

## 排查

如果页面提示类似：

```text
Unable to connect to GitHub server, please try again later
```

说明后端在 OAuth 回调阶段连接第三方 OAuth 服务失败。可以查看后端日志中的关键字：

```text
[OAuth-GitHub] ExchangeToken error
[OAuth-GitHub] GetUserInfo error
```

常见原因包括：

- 服务器无法访问第三方 OAuth 服务。
- 代理地址在容器内不可达。
- 代理协议写错，例如代理只支持 HTTP，却配置成 `socks5://`。
- `NO_PROXY` 规则误匹配，导致请求绕过代理。
