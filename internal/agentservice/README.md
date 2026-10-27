# internal/agentservice

原厂 **AcerAgentService** 的协议客户端。

这个服务是这台机器上真正管着灯光、风扇、性能档位与传感器读数的那一路，它
以**系统的身份**运行，所以问它不需要管理员权限。这个包只管协议，不认识
硬件：它把一个 JSON 负载送出去，把一个 JSON 文档收回来，不解释里面的意思。

## 协议

```
请求： "ACER" + uint32(小端) 包标识 + JSON 负载
应答： 纯 JSON 文档 {"data":…,"request":"…","result":"0"}
```

| 包标识 | 问什么 |
| :--- | :--- |
| 0 | 机器的能力（`SUPPORT_OPERATING_MODE_CAPABILITY` 之类） |
| 10 | 监视数据（温度、风扇转速、每核频率、显卡使用率……） |
| 20 | 一个设置的值（`LIGHTING`、`FAN_CONTROL`、`OPERATING_MODE`……） |
| 30 | 带着灯光的设备 |
| 40 | 灯效的能力（原厂服务不实现，只给一个空应答） |
| 100 | 写一个设置 |

| 结果码 | 意思 |
| :--- | :--- |
| `0` | 成功，`data` 里有东西 |
| `3` | 这个请求服务不实现（应答里**没有** data 字段） |
| `4` | 这个请求服务没有执行（应答里**没有** data 字段） |

## 几个细节

* **一条连接带多个请求。** 服务读一个请求、答一个，然后等下一个。客户端
  保持着连接，断掉之后自己重开一次，所以服务重启不会把程序带下去。
* **应答是连续的 JSON 文档**，靠大括号配对切分。字符串里的括号不算，字符串
  里的引号与转义也要算进去。
* **结果码不是成功就是失败。** `Answer.Err()` 把 `3` 变成 `ErrUnsupported`、
  其它编号变成 `ErrFailed`，调用方分得清「服务不认识这个请求」与「服务没
  做成这件事」。
* **应答可以没有数据。** 结果码 `3` 与 `4` 的应答只有 `request` 与 `result`
  两个字段，`Answer.Decode` 会把这种应答报成错，不会当成读到了一组空值。

## 用法

```go
client, err := agentservice.Connect(agentservice.Options{})
if err != nil {
	return err
}
defer client.Close()

answer, err := client.Get(agentservice.FunctionLighting)
if err != nil {
	return err
}
if err, failed := answer.Err(); failed {
	return err
}

var data struct {
	Brightness int `json:"brightness"`
}
if err := answer.Decode(&data); err != nil {
	return err
}
```

## 测试

```powershell
go test ./internal/agentservice/... -cover
```

用例里的替身服务说同一套协议，所以握手、帧的形状、结果码、断线重开、字符串
里的括号与引号都能在没有原厂服务的情况下跑完。覆盖率 93.4%。
