# cu-gomang-ule3-test

`cu-gomang`（ule3 分支）组件功能自测套件。

每条用例通过 etcd 向 key `dmidecode -s system-uuid`（gomang 的主机标识）下发任务，
等待 gomang 执行，再校验回写到 `<key>_<type>_result` 的结果。

## 测试内容

| 用例 | 任务类型 | 预期 |
|------|---------|------|
| execute_scripts_test | execute_scripts（shell） | status=1（成功） |

## 前置条件

- 目标机已安装 `gomang` RPM，或其 yum 源中包含 gomang 包（setup 阶段检查，
  缺失时自动 `yum install gomang`，随后通过 `gomang install` 注册 systemd 服务）
- 工具齐全：`dmidecode`、`etcdctl`、`jq`、`base64`、`yum`
- 以 **root** 运行（tone 要求 root；dmidecode/yum 也需要）

## 执行流程

- `setup()` 依次执行 `setup_etcd.sh`（安装并配置 etcd，**不开认证**——ule3 的
  gomang 连接 etcd 时无用户名/密码）和 `setup_gomang.sh`（检查 gomang RPM 已装
  否则 `yum install gomang`、添加 /etc/hosts 域名映射、`gomang install` 注册
  服务、重启并做就绪探测）。每次 `tone run` 都会执行，保证每次运行自包含。
- `run()` 执行用例。每条用例包装一个 `verify_*.sh`，输出
  `<用例>: pass|fail`；详细日志写入每用例独立的 `.log` 文件。
- `teardown()` 执行 `teardown.sh`（停止 gomang 并删除
  `/etc/systemd/system/gomang.service` + `systemctl daemon-reload`，卸载 etcd）
  以复原环境。

## 配置说明（换主机时按需调整）

- `verify_*.sh` —— `ENDPOINT` 的 IP 自动取本机 IP（etcd 部署在本机，监听
  `0.0.0.0:2379`），端口固定；无认证。
- `verify_*.sh` —— `KEY` 默认取本机系统 UUID（`dmidecode -s system-uuid`，tone 以 root 运行故不加 sudo）；取不到时用 `-k <key>` 指定。
- `setup_gomang.sh` —— `HOSTS_IP` 自动取本机 IP，`HOSTS_NAME` 为 etcd 内部域名
  （ule3 gomang 内置默认 EtcdURI 指向该域名，故只需 hosts 映射，无需 gomang.ini）。

## 通过 tone 运行

```bash
sudo tone install cu-gomang-ule3-test
sudo tone run cu-gomang-ule3-test
```

## 本地运行（不经过 tone）

```bash
cd cu-gomang-ule3-test
sudo bash run.sh                          # 完整流程：setup -> run -> teardown
bash cases/execute_scripts_test.sh        # 单条用例（需环境已就绪）
```

## 查看结果

`tone run` 完成后查看：
- `result/cu-gomang-ule3-test/<场景>/1/result.json` —— 每条用例的 pass/fail
- `result/cu-gomang-ule3-test/<场景>/1/stdout.log` —— 完整套件输出
- 每条用例的详细日志（路径由用例打印，位于 `TONE_CURRENT_RESULT_DIR` 下）
