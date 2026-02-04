# 需求描述

需要一个二层交换报文修改功能。设备会有一对口，通过linux的bridge vlan实现。最后通过nftables把报文送到queue中，匹配中的报文处理之后再放行，没有匹配中的直接放行。

首先有一个web界面配置修改规则。
- 输入：字段(起始地址+偏移量)过滤条件，按照偏移量定义字段，字段的匹配方式有10进制、16进制，五元组也就是内置字段。多字段按照运算符计算，运算符有或并非。可以增加括号。
- 处理：字段运算，运算符有加减乘除，值还可以是自定义的shell函数，比如可以调用其它bin文件，得到一个结果。
- 输出：按照处理规则，报文重组。有的字段可能增加长度，有的可能减小长度，重新组装报文。如果有需要重新计算checksum

开发语言：vue+go+sqlite ，轻量化的web程序，前端配置，后端go处理，数据存储用轻量化的sqlite。

举例：
网络拓扑搭建，

首先把ens38和ens39两个网口的vlan都设置为2，相当于在一个交换机内。
```
root@netvine:~# bridge vlan show
port    vlan ids
ens38    1 Egress Untagged
         2 PVID Egress Untagged

ens39    1 Egress Untagged
         2 PVID Egress Untagged

Bridge   1 PVID Egress Untagged
         2
```

另外做了一个vlanif，给这个vlan_2增加一个ip为192.168.10.100。

```
root@netvine:~# ip a
6: Bridge: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default qlen 1000
    link/ether 00:0c:29:24:44:7d brd ff:ff:ff:ff:ff:ff
    inet6 fe80::ff:aeff:fe5c:a36a/64 scope link
       valid_lft forever preferred_lft forever
15: vlan_2@Bridge: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default qlen 1000
    link/ether 00:0c:29:24:44:7d brd ff:ff:ff:ff:ff:ff
    inet 192.168.10.100/24 scope global vlan_2
       valid_lft forever preferred_lft forever
    inet6 fe80::20c:29ff:fe24:447d/64 scope link
       valid_lft forever preferred_lft forever
```
首先创建一个Bridge，相当于交换机。在Bridge上创建vlan,如果有接口使用vlan，那就存在，如果没有一个接口使用vlan，那就删除vlan接口。新增vlan接口后，判断是否存在，不存在，新建一个。每个接口增加vlan的作用就是增加vlan标签。比如示例中的`2 PVID Egress Untagged`

这些网络配置，vlan设置，vlanif设置都在web界面配置。另外网卡名称自动获取，操作系统是ubuntu24.04。

vlan网络相关的功能参考代码: doc/code/vlan
界面参考：doc/img

待报文字节流：
000400010006002381672e81000008004500005e4ba640004011f49eac10a0edac10013c18d018d0004ac047b2c20a00fcf26469010000004b3c030058480500115104000017000b5c5df2f109000000000001000082304248423130413031595030315f706d742e6f7073657400

nftables规则配置，可以输入五元组信息作为匹配规则，匹配中的送到queue中，go程序会按照前端设置的规则进行处理。

```
table ip netvine-table {
        chain base-rule-chain {
                type filter hook forward priority filter; policy drop;
                queue num 0-3 bypass
        }
}
```

16进制+ascii码展示
0000   00 04 00 01 00 06 00 23 81 67 2e 81 00 00 08 00   .......#.g......
0010   45 00 00 5e 4b a6 40 00 40 11 f4 9e ac 10 a0 ed   E..^K.@.@.......
0020   ac 10 01 3c 18 d0 18 d0 00 4a c0 47 b2 c2 0a 00   ...<.....J.G....
0030   fc f2 64 69 01 00 00 00 4b 3c 03 00 58 48 05 00   ..di....K<..XH..
0040   11 51 04 00 00 17 00 0b 5c 5d f2 f1 09 00 00 00   .Q......\]......
0050   00 00 01 00 00 82 30 42 48 42 31 30 41 30 31 59   ......0BHB10A01Y
0060   50 30 31 5f 70 6d 74 2e 6f 70 73 65 74 00         P01_pmt.opset.

比如我在web页面新建报文重组规则。首先新建字段，字段名称为tagName，起始地址为0x58，长度为16。字段名称为option，起始地址为0x69，长度为5。匹配规则就是 tagName = "BHB10A01YP01_pmt"，并且option="opset"。

修改规则也是复用字段配置，比如tagName替换成"BHB10A01YP01",option修改为opreset。

输出后的报文应该是之前的报文字段中把tagName、option替换成新的值，不再自定义字段内的内容保持不变。
重组后的报文应该是
0000   00 04 00 01 00 06 00 23 81 67 2e 81 00 00 08 00   .......#.g......
0010   45 00 00 5e 4b a6 40 00 40 11 f4 9e ac 10 a0 ed   E..^K.@.@.......
0020   ac 10 01 3c 18 d0 18 d0 00 4a c0 47 b2 c2 0a 00   ...<.....J.G....
0030   fc f2 64 69 01 00 00 00 4b 3c 03 00 58 48 05 00   ..di....K<..XH..
0040   11 51 04 00 00 17 00 0b 5c 5d f2 f1 09 00 00 00   .Q......\]......
0050   00 00 01 00 00 82 30 42 48 42 31 30 41 30 31 59   ......0BHB10A01Y
0060   50 30 31 2e 6f 70 65 72 73 65 74 00               P01.opreset.

需要正确理解需求，相当于把报文都分成了字段，比如tagName之前不是用户定义的字段，叫内置字段F1,tagName和option之间的字段是F2,值为0x2e，最后一个内置字段F3,值是00，。tagName和option是用户自定义的字段。

按照字段规则替换之后，用户自定义的字段值会变化，内置字段不会有改动,最终重组后的报文就相当于是F1+tagName+F2+option+F3。所以自定义完成后，需要在重组的时候自动分割内置字段，重组时按照之前报文顺序，把内置字段和自定义字段拼接起来。最终修改checksum的值。

另外UI界面要更加友好，现在是tagName是json字符串，改成字段+操作符+值，后台拼装成json。另外Output Template修改为额外处理选项，目前只有一个可选的计算checksum就行。


可以配置多条规则，规则可以启用和禁用，还有添加、删除、编辑功能。

还有一个测试模式，每条规则可以单独测试。输入16进制字节流，按照规则展示字段的值，自己运行处理规则，最终输出输出后的报文。这样就可以输入样例报文提前测试。

另外还有一个日志模式，如果触发过滤条件，修改报文发出后，需要在日志内展示原始报文，字段的原始值和修改后值，最终展示报文发送处理结果。

# 程序结构
## 前端
文件夹目录: web
## 后端
文件夹目录: server
## 数据库
文件夹目录: db

# 运行
./start.sh

# 测试
## 使用syslog udp测试
一个设备发送报文，经过设备后，报文送到queue，go程序按照规则处理报文，处理后的报文送到syslog udp，syslog udp再送到设备。

发送syslog udp报文：
```
logger -n 127.0.0.1 -P 514 -p local0.info "test 123"
```

接收syslog udp报文：
```
nc -u -l -p 514
```

这样设置后，发包机（10.10.10.10）想要到达收包机（10.10.10.20）：

ARP 请求会发到 网络A。
您的网桥（Bridge）在 ens38 收到 ARP，转发到 ens39（网络B）。
收包机在 网络B 收到 ARP 并响应。
链路打通，所有 UDP 流量都必须流经 ens38 -> Bridge -> ens39。
此时 NFTables 就能成功拦截流量，Go 程序也能抓到日志了。

测试日志规则
root@matrix:~# nft list ruleset
table bridge netvine-table {
        chain base-rule-chain {
                type filter hook forward priority 0; policy accept;
                udp dport 514 log prefix "test-rule" queue flags bypass to 0-3
        }
}

查看nftables日志，开启日志功能，增加前缀，查看日志
root@matrix:~# tail -f /var/log/kern.log
2026-02-03T16:27:31.466804+08:00 matrix kernel: test-ruleIN=ens38 OUT=ens39 MAC=00:0c:29:79:8e:a0:00:0c:29:33:87:9d:08:00 SRC=10.10.10.10 DST=10.10.10.30 LEN=154 TOS=0x00 PREC=0x00 TTL=64 ID=40274 DF PROTO=UDP SPT=39251 DPT=514 LEN=134
2026-02-03T16:27:32.469976+08:00 matrix kernel: test-ruleIN=ens38 OUT=ens39 MAC=00:0c:29:79:8e:a0:00:0c:29:33:87:9d:08:00 SRC=10.10.10.10 DST=10.10.10.30 LEN=154 TOS=0x00 PREC=0x00 TTL=64 ID=4017 DF PROTO=UDP SPT=45608 DPT=514 LEN=134

