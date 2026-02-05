package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"log/slog"
	"net"
	"net/http"
	"net/netip"

	promise "github.com/nlepage/go-js-promise"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	gojs "github.com/shynome/hack-gojs"
	"github.com/shynome/wgortc/bind"
	config "github.com/shynome/wgortc/bind/config/simple"
	"github.com/shynome/wgortc/device/logger"
	"github.com/shynome/wgortc/device/vtun"
	"golang.zx2c4.com/wireguard/device"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
)

var Version = "dev"

var LicensePubkeyStr string = "MZ8P26kG8OyarZWrYa0QKHZypSfzPjlU7bzxWQwOqRc="

var LicensePubkey ed25519.PublicKey

func init() {
	if len(LicensePubkeyStr) != 0 {
		pubkeyRaw := try.To1(base64.StdEncoding.DecodeString(LicensePubkeyStr))
		LicensePubkey = ed25519.PublicKey(pubkeyRaw)
	}
}

func main() {
	jsVPN := gojs.JSGo.Get("importObject").Get("vpn")
	cfg := jsVPN.Get("config")
	ctx := signal2ctx(cfg.Get("signal"))
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	p, resolve, reject := promise.New()
	_ = resolve
	go func() (err error) {
		defer err0.Then(&err, nil, func() {
			cancel()
			reject(err.Error())
		})
		cfg := try.To1(getConfig[config.Config](cfg))
		try.To(cfg.Normalize())
		slog.SetLogLoggerLevel(cfg.LogLevel)

		b := bind.New(&cfg)
		tdev := try.To1(vtun.CreateTUN("salt-link", bind.MTU))
		logger := logger.New("salt-link")
		dev := device.NewDevice(tdev, b, logger)
		try.To(dev.IpcSet(cfg.IpcConfig()))
		try.To(dev.Up())
		try.To(vtun.RouteUp(tdev, []string{cfg.NAT}))
		b.Device.Store(dev)
		stk, nic := tdev.GetStack(), tdev.NIC()
		client := &http.Client{
			Transport: &http.Transport{
				DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
					ap, err := netip.ParseAddrPort(addr)
					if err != nil {
						return nil, err
					}
					fa, pn := convertToFullAddr(nic, ap)
					return gonet.DialContextTCP(ctx, stk, fa, pn)
				},
			},
		}
		pf := try.To1(netip.ParsePrefix(cfg.NAT))
		net := &Network{
			stk: stk, nic: nic, dev: dev,
			client: client,
			pf:     pf,
		}
		resolve(net.ToJS())
		return
	}()
	jsVPN.Set("connect_result", p)
	<-ctx.Done()
}
