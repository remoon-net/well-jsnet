# 如何使用

## 简洁版

```js
// ==UserScript==
// @name        well-net
// @namespace   well.remoon.net
// @match       https://salt.remoon.cn/
// @grant       none
// @version     1.0
// @author      -
// @run-at      document-start
// @description 12/05/2025, 10:07:53
// ==/UserScript==

void (async function main() {
  const WellNet = await import("https://unpkg.com/well-net/index.js")
  const net = await WellNet.connect({
    Key: "IDHBZNpXkYmavc3JhCvCA9bTh6fo2IfB1D/F6mE6xXg=",
    Peer: "ws://127.0.0.1:7799/api/whip#UwJ8NDnTdT/XM1VC8wPF7iu0GMP3FK81qRPSQlHQ7jU=",
  })
  const srv = await net.listen("0.0.0.0:80", {
    fetch(req) {
      return new Response("ok")
    },
  })
  await net.http_proxy("0.0.0.0:1080", {})
})()
```

## 高阶版

```js
// ==UserScript==
// @name        well-net
// @namespace   well.remoon.net
// @match       https://salt.remoon.cn/
// @grant       none
// @version     1.0
// @author      -
// @run-at      document-start
// @description 12/05/2025, 10:07:53
// ==/UserScript==

void (async function main() {
  const WellNet = await import("https://unpkg.com/well-net/index.js")
  const net = await WellNet.Connect({
    Key: "IDHBZNpXkYmavc3JhCvCA9bTh6fo2IfB1D/F6mE6xXg=",
    // LogLevel: "DEBUG",
    // NAT: "192.168.211.1/24", // web 端不用设置这么多东西, 直接就走NAT
    Peers: [
      {
        Pubkey: "UwJ8NDnTdT/XM1VC8wPF7iu0GMP3FK81qRPSQlHQ7jU=",
        // PSK: "",
        Endpoint: "ws://127.0.0.1:7799/api/whip", // 自动连接所必需的
        Auto: 5, // 是否自动连接, 为零或空时不自动连接, 但大概率都是需要自动连接的
        Allow: "192.168.211.3/32", // 选择了这个模式, 则需要自己设定 IP, 要在NAT字段 192.168.211.1/24 范围里的
      },
    ],
  })
  const srv = await net.listen("0.0.0.0:80", {
    fetch(req) {
      return new Response("ok")
    },
  })
  await net.http_proxy("0.0.0.0:1080", {})
  const z = await net.fetch("192.168.211.2:80")
})()
```
