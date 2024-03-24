# open-iscsi

在容器中使用 iscsiadm 管理 iscsi 连接需要注意以下几点：

- 由于 iscsiadm 需要连接 iscsid 服务，而容器场景中通常在 host 上运行 iscsid，因此主机上 iscsid 需要配置为开机启动。
- 需要挂载的目录包括：
  - /etc/iscsi
  - /run
  - /sys
  - /dev
  - /lib/modules
- 容器配置 privileged 特权
- iscsiadm 和 iscsid 似乎时通过 127.0.0.1 通信因此容器需要配置为主机网络
- 尽可能保证容器内和 host 上 open-iscsi 版本一致，必要时可以通过映射 iscsiadm 到容器内达成这一目的


参考：
- https://www.docker.com/blog/road-to-containing-iscsi/