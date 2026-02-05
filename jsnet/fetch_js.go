package main

import (
	"net/netip"
	"syscall/js"

	promise "github.com/nlepage/go-js-promise"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"github.com/shynome/wahttp"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
)

func (n *Network) Fetch(this js.Value, args []js.Value) (p any) {
	p, resolve, reject := promise.New()

	go func() (err error) {
		defer err0.Then(&err, nil, func() {
			reject(err.Error())
		})
		req := try.To1(wahttp.JsRequest(args[0]))
		resp := try.To1(n.client.Do(req))
		resolve(wahttp.GoResponse(resp))
		return nil
	}()

	return p
}

func convertToFullAddr(NICID tcpip.NICID, endpoint netip.AddrPort) (tcpip.FullAddress, tcpip.NetworkProtocolNumber) {
	var protoNumber tcpip.NetworkProtocolNumber
	if endpoint.Addr().Is4() {
		protoNumber = ipv4.ProtocolNumber
	} else {
		protoNumber = ipv6.ProtocolNumber
	}
	return tcpip.FullAddress{
		NIC:  NICID,
		Addr: tcpip.AddrFromSlice(endpoint.Addr().AsSlice()),
		Port: endpoint.Port(),
	}, protoNumber
}
