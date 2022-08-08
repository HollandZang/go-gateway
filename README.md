#### nginx.conf

```
http {
    # 在此配置转发地址，以实现平滑服务重启 
    upstream backend {
        # 以下都是 网关项目 地址
		server localhost:8080;
		server localhost:8081;
		#server localhost:8082;
	}
	
	server {
	    # 回调地址使用 listen 的端口
		listen       80;
        server_name  localhost;
		
		location / {
			proxy_pass http://backend;
		}
	}
}
```

#### 启动命令行

`go run main.go -k 12345678 -p 8080 -r router.json`

- **-k** **必填** *配置回调解密key*
- **-p** *配置启动端口，默认8080*
- **-r** *配置路由文件，默认router.json*

#### 配置路由文件 router.json

- 数字比较，精度会自适应
- 数字与字符串比较，不会相等：字符串1 != 数字1
- 表达式参考：https://github.com/Knetic/govaluate
- *不建议在表达式中使用结构体变量，原因见* [issues](https://github.com/Knetic/govaluate/issues/61)

#### 回调数据示例

```
data=eyJhbW91bnQiOjEuMCwib3JkZXJObyI6IkhVQTEwMTg2MTY0NDEyIiwicGF5VGltZSI6IjIwMjEtMDctMDUgMTQ6NDI6MzMiLCJnb29kc0lkIjoiMSIsImNoYW5uZWxVaWQiOiIxMTg1OTEzMzk3OTAxMTM5Iiwic2VsZkRlZmluZSI6IumAj%2BS8oOWPguaVsCIsImNoYW5uZWwiOiJodWF3ZWkiLCJxdWlja0NoYW5uZWxJZCI6MjQsInFrQ2hhbm5lbElkIjoyNCwiZ2FtZU9yZGVyIjoiY3BPcmRlcklkXzE2MjU0Njc2MTIxMzAiLCJnb29kc05hbWUiOiLllYblk4FJROiHquWumuS5iea1i%2BivleWVhuWTgS0wMSIsImNoYW5uZWxJZCI6MTEsInN0YXR1cyI6MH0%3D&sign=1741120cdda4b3a50fe59224595da88f
```

#### data解密后的示例

```json
{
  "amount": 1.0,
  "orderNo": "HUA10186164412",
  "payTime": "2021-07-05 14:42:33",
  "goodsId": "1",
  "channelUid": "1185913397901139",
  "selfDefine": "透传参数",
  "channel": "huawei",
  "quickChannelId": 24,
  "qkChannelId": 24,
  "gameOrder": "cpOrderId_1625467612130",
  "goodsName": "商品ID自定义测试商品-01",
  "channelId": 11,
  "status": 0
}
```
