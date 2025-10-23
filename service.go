package liboc

import (
	"context"
	"os"
	"runtime/debug"
	"time"

	"github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/adapter"
	"github.com/sagernet/sing-box/common/urltest"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/dns"
	"github.com/sagernet/sing-box/experimental/libbox/platform"
	"github.com/sagernet/sing-box/include"
	"github.com/sagernet/sing-box/log"
	"github.com/sagernet/sing-box/option"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json"
	"github.com/sagernet/sing/service"
	"github.com/sagernet/sing/service/filemanager"
	"github.com/sagernet/sing/service/pause"
)

type BoxService struct {
	ctx                   context.Context
	cancel                context.CancelFunc
	urlTestHistoryStorage adapter.URLTestHistoryStorage
	instance              *box.Box
	clashServer           adapter.ClashServer
	pauseManager          pause.Manager
}

func NewService(configContent string, platformInterface PlatformInterface) (*BoxService, error) {
	ctx := BaseContext(platformInterface)
	service.MustRegister[DeprecatedManager](ctx, new(deprecatedManager))
	options, err := parseConfig(ctx, configContent)
	if err != nil {
		return nil, err
	}
	debug.FreeOSMemory()
	ctx, cancel := context.WithCancel(ctx)
	urlTestHistoryStorage := urltest.NewHistoryStorage()
	ctx = service.ContextWithPtr(ctx, urlTestHistoryStorage)
	platformWrapper := &platformInterfaceWrapper{
		iif:       platformInterface,
		useProcFS: platformInterface.UseProcFS(),
	}
	service.MustRegister[platform.Interface](ctx, platformWrapper)
	instance, err := box.New(box.Options{
		Context:           ctx,
		Options:           options,
		PlatformLogWriter: platformWrapper,
	})
	if err != nil {
		cancel()
		return nil, E.Cause(err, "create service")
	}
	debug.FreeOSMemory()
	return &BoxService{
		ctx:                   ctx,
		cancel:                cancel,
		instance:              instance,
		urlTestHistoryStorage: urlTestHistoryStorage,
		pauseManager:          service.FromContext[pause.Manager](ctx),
		clashServer:           service.FromContext[adapter.ClashServer](ctx),
	}, nil
}

func (s *BoxService) Start() error {
	if sFixAndroidStack {
		var err error
		done := make(chan struct{})
		go func() {
			err = s.instance.Start()
			close(done)
		}()
		<-done
		return err
	} else {
		return s.instance.Start()
	}
}

func (s *BoxService) Close() error {
	s.cancel()
	s.urlTestHistoryStorage.Close()
	var err error
	done := make(chan struct{})
	go func() {
		err = s.instance.Close()
		close(done)
	}()
	select {
	case <-done:
		return err
	case <-time.After(C.FatalStopTimeout):
		os.Exit(1)
		return nil
	}
}

func (s *BoxService) NeedWIFIState() bool {
	return s.instance.Router().NeedWIFIState()
}

func (s *BoxService) Pause() {
	s.pauseManager.DevicePause()
}

func (s *BoxService) Wake() {
	s.pauseManager.DeviceWake()
}

var (
	sBasePath        string
	sWorkingPath     string
	sTempPath        string
	sUserID          int
	sGroupID         int
	sTVOS            bool
	sFixAndroidStack bool
)

func Setup(options *SetupOptions) error {
	sBasePath = options.BasePath
	sWorkingPath = options.WorkingPath
	sTempPath = options.TempPath
	sUserID = os.Getuid()
	sGroupID = os.Getgid()
	sTVOS = options.IsTVOS
	sFixAndroidStack = options.FixAndroidStack

	os.MkdirAll(sWorkingPath, 0o700)
	os.MkdirAll(sTempPath, 0o700)

	return nil
}

type SetupOptions struct {
	BasePath        string
	WorkingPath     string
	TempPath        string
	IsTVOS          bool
	FixAndroidStack bool
}

func SetMemoryLimit(enabled bool) {
	if enabled {
		debug.SetMemoryLimit(256 * 1024 * 1024)
		debug.SetGCPercent(20)
	} else {
		debug.SetMemoryLimit(-1)
		debug.SetGCPercent(100)
	}
}

func SetLocale(localeId string) {
	Set(localeId)
}

func RedirectStderr(path string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	os.Stderr = file
	return nil
}

func BaseContext(platformInterface PlatformInterface) context.Context {
	dnsRegistry := include.DNSTransportRegistry()
	if platformInterface != nil {
		if localTransport := platformInterface.LocalDNSTransport(); localTransport != nil {
			dns.RegisterTransport[option.LocalDNSServerOptions](dnsRegistry, C.DNSTypeLocal, func(ctx context.Context, logger log.ContextLogger, tag string, options option.LocalDNSServerOptions) (adapter.DNSTransport, error) {
				return newPlatformTransport(localTransport, tag, options), nil
			})
		}
	}
	ctx := context.Background()
	ctx = filemanager.WithDefault(ctx, sWorkingPath, sTempPath, sUserID, sGroupID)
	return box.Context(ctx, include.InboundRegistry(), include.OutboundRegistry(), include.EndpointRegistry(), dnsRegistry, include.ServiceRegistry())
}

func parseConfig(ctx context.Context, configContent string) (option.Options, error) {
	options, err := json.UnmarshalExtendedContext[option.Options](ctx, []byte(configContent))
	if err != nil {
		return option.Options{}, E.Cause(err, "parse config")
	}
	return options, nil
}