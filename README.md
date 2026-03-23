# login-2fa

一个基于 Go 实现的 Linux PAM 二次登录验证工具集，包含：

- `login-2fa`: 命令行工具，显示短机器码和终端二维码，按机器码生成/校验一次性验证码
- `pam_login_2fa.so`: PAM 模块，登录时显示短机器码和二维码，然后要求输入一次性验证码

## 特性

- 机器码短格式：`XXXX-XXXX-XXXX-XXXX`
- 终端二维码显示：使用 Go 库直接渲染
- 密钥运行时动态载入
- 机器码通过开源库 `github.com/denisbrodbeck/machineid` 获取
- 适合内网和离线部署
- GitHub Actions 自动构建 Release

## 仓库结构

```text
cmd/login-2fa/          CLI
cmd/pam_login_2fa/      PAM module
internal/login2fa/      shared logic
scripts/build_login_2fa_go.sh
```

## 编译

```bash
chmod +x scripts/build_login_2fa_go.sh
./scripts/build_login_2fa_go.sh
```

构建脚本每次会随机生成一个新的密钥文件：

- `dist/login-2fa.key`

编译产物会出现在 `dist/`：

- `dist/login-2fa`
- `dist/login-2fa.key`
- `dist/pam_login_2fa.so`
- `dist/pam_login_2fa.h`

## GitHub Actions

仓库已包含两个工作流：

- `CI`: 在 push / PR 时下载 Go Modules、编译 CLI 和 PAM 模块
- `Release`: 在推送 `v*` tag 时自动发布 GitHub Release

Release 当前会生成：

- `login-2fa-linux-amd64.tar.gz`
- `pam_login_2fa-linux-amd64.tar.gz`

说明：

- Release 当前仅发布 `linux-amd64`
- PAM `.so` 依赖目标系统的 PAM ABI，目前工作流默认发布 `linux-amd64`
- 公共 Release 不会附带私密密钥文件，部署时请自行生成并安装

## 使用

显示机器码和终端二维码：

```bash
./dist/login-2fa machine-code
```

按机器码生成动态验证码：

```bash
./dist/login-2fa generate --machine-code 6236-C5AB-71F8-DA75
```

按机器码校验动态验证码：

```bash
./dist/login-2fa verify --machine-code 6236-C5AB-71F8-DA75 --code 123456
```

## PAM 接入示例

把 `pam_login_2fa.so` 放到系统 PAM 模块目录，然后在目标服务里加入：

```pam
auth required pam_login_2fa.so
```

常见模块目录：

- `/lib/security/`
- `/lib/x86_64-linux-gnu/security/`

示例文件见 `examples/pam.d/sshd`.

密钥加载优先级：

1. 环境变量 `LOGIN2FA_MASTER_KEY`
2. 环境变量 `LOGIN2FA_MASTER_KEY_FILE`
3. `/etc/security/login-2fa.key`
4. 当前目录或可执行文件同目录下的 `login-2fa.key`

## 部署前修改

如需调整时间策略，请修改 [internal/login2fa/core.go](./internal/login2fa/core.go)：

- `DefaultStep`
- `DefaultDigits`
- `DefaultWindow`

## 注意

- 这是登录链路模块，先在测试环境验证再上线。
- PAM 侧二维码展示依赖终端支持 Unicode 块字符。
- 构建脚本每次都会生成新密钥，重新构建后旧验证码会失效。
