package liboc

import (
	"context"

	mDNS "github.com/miekg/dns"
	"github.com/sagernet/sing-box/adapter"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/dns"
	"github.com/sagernet/sing-box/option"
)

type ExchangeContext struct {
	context   context.Context
	message   interface{}
	addresses interface{}
	error     error
}

func (c *ExchangeContext) OnCancel(callback Func) {
	go func() {
		<-c.context.Done()
		callback.Invoke()
	}()
}

type LocalDNSTransport interface {
	Raw() bool
	Lookup(ctx *ExchangeContext, network string, domain string) error
	Exchange(ctx *ExchangeContext, message []byte) error
}

type Func interface {
	Invoke()
}

type platformTransport struct {
	dns.TransportAdapter
	iif LocalDNSTransport
}

func newPlatformTransport(iif LocalDNSTransport, tag string, options option.LocalDNSServerOptions) *platformTransport {
	return &platformTransport{
		TransportAdapter: dns.NewTransportAdapterWithLocalOptions(C.DNSTypeLocal, tag, options),
		iif:              iif,
	}
}

func (p *platformTransport) Start(stage adapter.StartStage) error {
	return nil
}

func (p *platformTransport) Close() error {
	return nil
}

func (p *platformTransport) Exchange(ctx context.Context, message *mDNS.Msg) (*mDNS.Msg, error) {
	if p.iif == nil {

		response := new(mDNS.Msg)
		response.SetReply(message)
		response.Rcode = mDNS.RcodeNameError
		return response, nil
	}

	msgBytes, err := message.Pack()
	if err != nil {
		return nil, err
	}

	exchCtx := &ExchangeContext{
		context: ctx,
	}

	err = p.iif.Exchange(exchCtx, msgBytes)
	if err != nil {
		return nil, err
	}

	if exchCtx.error != nil {
		return nil, exchCtx.error
	}

	if respBytes, ok := exchCtx.message.([]byte); ok {
		response := new(mDNS.Msg)
		err = response.Unpack(respBytes)
		if err != nil {
			return nil, err
		}
		return response, nil
	}

	response := new(mDNS.Msg)
	response.SetReply(message)
	response.Rcode = mDNS.RcodeNameError
	return response, nil
}
