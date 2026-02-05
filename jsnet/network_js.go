package main

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"syscall/js"
	"time"

	"github.com/armon/go-socks5"
	"github.com/elazarl/goproxy"
	"github.com/maypok86/otter"
	promise "github.com/nlepage/go-js-promise"
	"github.com/shynome/err0"
	"github.com/shynome/err0/try"
	"golang.zx2c4.com/wireguard/device"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv6"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

type Network struct {
	stk    *stack.Stack
	nic    tcpip.NICID
	dev    *device.Device
	client *http.Client
	pf     netip.Prefix
}

func (net *Network) ToJS() js.Value {
	root := js.Global().Get("Object").New()
	root.Set("listen", js.FuncOf(net.Listen))
	root.Set("http_proxy", js.FuncOf(net.HTTPProxy))
	root.Set("socks5_proxy", js.FuncOf(net.Socks5Proxy))
	root.Set("fetch", js.FuncOf(net.Fetch))
	root.Set("version", Version)
	return root
}

type netListener = net.Listener

func (net *Network) Listen(this js.Value, args []js.Value) (p any) {
	p, resolve, reject := promise.New()
	go func() (err error) {
		defer err0.Then(&err, nil, func() {
			reject(err.Error())
		})
		if len(args) != 2 {
			reject("rqeuire listen addr and http server implement {fetch(Request):Response|Promise<Response>}")
			return
		}
		if args[0].Type() != js.TypeString {
			reject("addr is unknown")
			return
		}
		addr, cfg := args[0].String(), args[1]
		mux := NewHono(cfg)
		ctx := signal2ctx(cfg.Get("signal"))
		ctx, cancel := context.WithCancel(ctx)

		net.serveTry(ctx, addr, mux)

		root := js.Global().Get("Object").New()
		root.Set("close", js.FuncOf(func(this js.Value, args []js.Value) any {
			cancel()
			return js.Undefined()
		}))
		resolve(root)
		return nil
	}()
	return p
}

func (net *Network) serveTry(ctx context.Context, addrStr string, handler http.Handler) {
	ap := try.To1(netip.ParseAddrPort(addrStr))
	addr := ap.Addr()
	fa := tcpip.FullAddress{
		Port: ap.Port(),
	}
	if addr.Is6() {
		if !addr.IsUnspecified() {
			fa.Addr = tcpip.AddrFrom16(addr.As16())
		}
		l := try.To1(gonet.ListenTCP(net.stk, fa, ipv6.ProtocolNumber))
		go func() {
			<-ctx.Done()
			l.Close()
		}()
		go http.Serve(l, handler)
	}
	if addr.Is4() {
		if !addr.IsUnspecified() {
			fa.Addr = tcpip.AddrFrom4(addr.As4())
		}
		l := try.To1(gonet.ListenTCP(net.stk, fa, ipv4.ProtocolNumber))
		go func() {
			<-ctx.Done()
			l.Close()
		}()
		go http.Serve(l, handler)
	}
}

func (net *Network) HTTPProxy(this js.Value, args []js.Value) (p any) {
	p, resolve, reject := promise.New()
	go func() (err error) {
		defer err0.Then(&err, nil, func() {
			reject(err.Error())
		})
		if len(args) != 2 {
			reject("rqeuire listen addr and empty config {}")
			return
		}
		if args[0].Type() != js.TypeString {
			reject("addr is unknown")
			return
		}
		addr, cfg := args[0].String(), args[1]
		proxy := goproxy.NewProxyHttpServer()

		cache := try.To1(otter.MustBuilder[string, *tls.Certificate](10_000).WithVariableTTL().Build())
		proxy.CertStore = &CertStorage{cache}

		proxy.OnRequest().HandleConnectFunc(func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
			ctx.RoundTripper = goproxy.RoundTripperFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Response, error) {
				return http.DefaultClient.Do(req)
			})
			return goproxy.MitmConnect, host
		})
		proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
			injectJsFetchOptions(req)
			return req, nil
		})
		ctx := signal2ctx(cfg.Get("signal"))
		ctx, cancel := context.WithCancel(ctx)

		net.serveTry(ctx, addr, proxy)

		root := js.Global().Get("Object").New()
		root.Set("close", js.FuncOf(func(this js.Value, args []js.Value) any {
			cancel()
			return js.Undefined()
		}))
		resolve(root)
		return nil
	}()
	return p
}

const jsFetchOptInPrefix = "Js.fetch."
const jsFetchOptPrefix = "js.fetch:"

func injectJsFetchOptions(r *http.Request) {
	for k, vv := range r.Header {
		if strings.HasPrefix(k, jsFetchOptInPrefix) {
			r.Header.Del(k)
			k = jsFetchOptPrefix + k[len(jsFetchOptInPrefix):]
			r.Header[k] = vv
		}
	}
}

type CertStorage struct {
	otter.CacheWithVariableTTL[string, *tls.Certificate]
}

var _ goproxy.CertStorage = (*CertStorage)(nil)

func (c *CertStorage) Fetch(hostname string, gen func() (*tls.Certificate, error)) (*tls.Certificate, error) {
	if cert, ok := c.Get(hostname); ok {
		return cert, nil
	}
	cert, err := gen()
	if err != nil {
		return nil, err
	}
	c.Set(hostname, cert, 360*24*time.Hour)
	return cert, nil
}

func (n *Network) Socks5Proxy(this js.Value, args []js.Value) (p any) {
	p, resolve, reject := promise.New()
	go func() (err error) {
		defer err0.Then(&err, nil, func() {
			reject(err.Error())
		})
		if len(args) != 2 {
			reject("rqeuire listen addr and empty config {}")
			return
		}
		if args[0].Type() != js.TypeString {
			reject("addr is unknown")
			return
		}
		addr, jsCfg := args[0].String(), args[1]

		cfg := &socks5.Config{
			Rules: &RuleSetAllowAll{},
			Dial: func(ctx context.Context, network, addr string) (net.Conn, error) {
				ap, err := netip.ParseAddrPort(addr)
				if err != nil {
					return nil, err
				}
				nic := n.nic
				if dst := ap.Addr(); dst.IsLoopback() || (n.pf.Addr().Compare(dst) == 0) {
					nic += 1
				}
				fa, pn := convertToFullAddr(nic, ap)
				switch network {
				case "tcp", "tcp4", "tcp6":
					return gonet.DialContextTCP(ctx, n.stk, fa, pn)
				case "udp", "udp4", "udp6":
					return gonet.DialUDP(n.stk, nil, &fa, pn)
				}
				return nil, net.UnknownNetworkError(network)
			},
		}
		s := try.To1(socks5.New(cfg))

		ctx := signal2ctx(jsCfg.Get("signal"))
		ctx, cancel := context.WithCancel(ctx)

		ap := try.To1(netip.ParseAddrPort(addr))
		fa := tcpip.FullAddress{Port: ap.Port()}
		pn := ipv6.ProtocolNumber
		if ap.Addr().Is4() {
			pn = ipv4.ProtocolNumber
		}
		l := try.To1(gonet.ListenTCP(n.stk, fa, pn))
		go func() {
			<-ctx.Done()
			l.Close()
		}()
		go s.Serve(l)

		root := js.Global().Get("Object").New()
		root.Set("close", js.FuncOf(func(this js.Value, args []js.Value) any {
			cancel()
			return js.Undefined()
		}))
		resolve(root)
		return nil
	}()
	return p
}

type RuleSetAllowAll struct{}

var _ socks5.RuleSet = (*RuleSetAllowAll)(nil)

func (RuleSetAllowAll) Allow(ctx context.Context, req *socks5.Request) (context.Context, bool) {
	return ctx, true
}
