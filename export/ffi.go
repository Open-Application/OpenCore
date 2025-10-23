//go:build windows && !android

package main

/*
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>

// Callback function types for PlatformInterface
typedef void (*WriteLogFunc)(const char* message);
typedef int32_t (*FindConnectionOwnerFunc)(int32_t ipProtocol, const char* sourceAddress, int32_t sourcePort, const char* destinationAddress, int32_t destinationPort);
typedef char* (*PackageNameByUidFunc)(int32_t uid);
typedef int32_t (*UidByPackageNameFunc)(const char* packageName);
typedef void (*InterfaceUpdateFunc)(const char* interfaceName, int32_t interfaceIndex, int32_t isExpensive, int32_t isConstrained);

typedef struct {
    WriteLogFunc writeLog;
    FindConnectionOwnerFunc findConnectionOwner;
    PackageNameByUidFunc packageNameByUid;
    UidByPackageNameFunc uidByPackageName;
    InterfaceUpdateFunc interfaceUpdate;
} PlatformInterface;

static inline void call_writeLog(WriteLogFunc func, const char* message) {
    if (func != NULL) {
        func(message);
    }
}

static inline int32_t call_findConnectionOwner(FindConnectionOwnerFunc func, int32_t ipProtocol, const char* sourceAddress, int32_t sourcePort, const char* destinationAddress, int32_t destinationPort) {
    if (func != NULL) {
        return func(ipProtocol, sourceAddress, sourcePort, destinationAddress, destinationPort);
    }
    return -1;
}

static inline char* call_packageNameByUid(PackageNameByUidFunc func, int32_t uid) {
    if (func != NULL) {
        return func(uid);
    }
    return NULL;
}

static inline int32_t call_uidByPackageName(UidByPackageNameFunc func, const char* packageName) {
    if (func != NULL) {
        return func(packageName);
    }
    return -1;
}
*/
import "C"

import (
	"bytes"
	"context"
	"runtime"
	"sync"
	"unsafe"

	liboc "Open-Application/OpenCore"
	"github.com/sagernet/sing-box"
	"github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing-tun"
	E "github.com/sagernet/sing/common/exceptions"
	"github.com/sagernet/sing/common/json"
)

var (
	serviceRegistry sync.Map
	nextServiceID   int64    = 1
	serviceMutex    sync.Mutex

	platformRegistry sync.Map

	lastError     string
	lastErrorLock sync.Mutex
)

func init() {
	runtime.LockOSThread()
}

func setLastError(err string) {
	lastErrorLock.Lock()
	defer lastErrorLock.Unlock()
	lastError = err
}

func clearLastError() {
	lastErrorLock.Lock()
	defer lastErrorLock.Unlock()
	lastError = ""
}

//export FreeString
func FreeString(str *C.char) {
	if str != nil {
		C.free(unsafe.Pointer(str))
	}
}

//export FreeBytes
func FreeBytes(data *C.char) {
	if data != nil {
		C.free(unsafe.Pointer(data))
	}
}

//export Setup
func Setup(basePath *C.char, workingPath *C.char, tempPath *C.char, isTVOS C.int, fixAndroidStack C.int) *C.char {
	if basePath == nil || workingPath == nil || tempPath == nil {
		return C.CString("path parameters cannot be null")
	}

	err := liboc.Setup(&liboc.SetupOptions{
		BasePath:        C.GoString(basePath),
		WorkingPath:     C.GoString(workingPath),
		TempPath:        C.GoString(tempPath),
		IsTVOS:          isTVOS != 0,
		FixAndroidStack: fixAndroidStack != 0,
	})

	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export SetMemoryLimit
func SetMemoryLimit(enabled C.int) {
	liboc.SetMemoryLimit(enabled != 0)
}

//export SetLocale
func SetLocale(localeId *C.char) {
	liboc.SetLocale(C.GoString(localeId))
}


//export ClearServiceError
func ClearServiceError() {
	liboc.ClearServiceError()
}

//export ReadServiceError
func ReadServiceError(errorOut **C.char) *C.char {
	stringBox, err := liboc.ReadServiceError()
	if err != nil {
		return C.CString(err.Error())
	}
	if stringBox != nil {
		*errorOut = C.CString(stringBox.Value)
	}
	return nil
}

//export WriteServiceError
func WriteServiceError(message *C.char) *C.char {
	err := liboc.WriteServiceError(C.GoString(message))
	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export RedirectStderr
func RedirectStderr(path *C.char) *C.char {
	err := liboc.RedirectStderr(C.GoString(path))
	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export Version
func Version() *C.char {
	return C.CString(constant.Version)
}

//export CheckConfig
func CheckConfig(configContent *C.char) *C.char {
	if configContent == nil {
		return C.CString("configContent is null")
	}

	ctx := liboc.BaseContext(nil)
	options, err := parseConfig(ctx, C.GoString(configContent))
	if err != nil {
		return C.CString(err.Error())
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	instance, err := box.New(box.Options{
		Context: ctx,
		Options: options,
	})
	if err != nil {
		return C.CString(err.Error())
	}
	instance.Close()

	return nil
}

//export FormatConfig
func FormatConfig(configContent *C.char, formattedOut **C.char) *C.char {
	if configContent == nil {
		return C.CString("configContent is null")
	}
	if formattedOut == nil {
		return C.CString("formattedOut is null")
	}

	ctx := liboc.BaseContext(nil)
	options, err := parseConfig(ctx, C.GoString(configContent))
	if err != nil {
		return C.CString(err.Error())
	}

	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetIndent("", "  ")
	err = encoder.Encode(options)
	if err != nil {
		return C.CString(err.Error())
	}

	*formattedOut = C.CString(buffer.String())
	return nil
}

//export LibocGetLastError
func LibocGetLastError() *C.char {
	lastErrorLock.Lock()
	defer lastErrorLock.Unlock()

	if lastError == "" {
		return C.CString("")
	}
	return C.CString(lastError)
}

func parseConfig(ctx context.Context, configContent string) (option.Options, error) {
	options, err := json.UnmarshalExtendedContext[option.Options](ctx, []byte(configContent))
	if err != nil {
		return option.Options{}, E.Cause(err, "parse config")
	}
	return options, nil
}

//export NewService
func NewService(configContent *C.char, platformInterface *C.PlatformInterface) C.int64_t {
	clearLastError()

	if platformInterface == nil {
		setLastError("PlatformInterface is null")
		return -1
	}

	if configContent == nil {
		setLastError("configContent is null")
		return -1
	}

	config := C.GoString(configContent)

	goInterface := newWindowsPlatformInterface(platformInterface)
	service, err := liboc.NewService(config, goInterface)

	if err != nil {
		setLastError(err.Error())
		return -1
	}

	serviceMutex.Lock()
	serviceID := nextServiceID
	nextServiceID++
	serviceRegistry.Store(serviceID, service)
	platformRegistry.Store(serviceID, platformInterface)
	serviceMutex.Unlock()

	return C.int64_t(serviceID)
}

//export ServiceStart
func ServiceStart(serviceID C.int64_t) *C.char {
	serviceInterface, ok := serviceRegistry.Load(int64(serviceID))
	if !ok {
		return C.CString("service not found")
	}

	service := serviceInterface.(*liboc.BoxService)
	err := service.Start()

	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

//export ServiceClose
func ServiceClose(serviceID C.int64_t) *C.char {
	serviceInterface, ok := serviceRegistry.Load(int64(serviceID))
	if !ok {
		return C.CString("service not found")
	}

	service := serviceInterface.(*liboc.BoxService)
	err := service.Close()

	serviceRegistry.Delete(int64(serviceID))
	platformRegistry.Delete(int64(serviceID))

	cleanupAllTunDevices()

	if err != nil {
		return C.CString(err.Error())
	}
	return nil
}

func cleanupAllTunDevices() {
	tunDevicesMutex.Lock()
	defer tunDevicesMutex.Unlock()
	for fd, device := range tunDevices {
		device.Close()
		delete(tunDevices, fd)
	}
}

//export ServicePause
func ServicePause(serviceID C.int64_t) {
	serviceInterface, ok := serviceRegistry.Load(int64(serviceID))
	if !ok {
		return
	}

	service := serviceInterface.(*liboc.BoxService)
	service.Pause()
}

//export ServiceWake
func ServiceWake(serviceID C.int64_t) {
	serviceInterface, ok := serviceRegistry.Load(int64(serviceID))
	if !ok {
		return
	}

	service := serviceInterface.(*liboc.BoxService)
	service.Wake()
}

//export ServiceNeedWIFIState
func ServiceNeedWIFIState(serviceID C.int64_t) C.char {
	serviceInterface, ok := serviceRegistry.Load(int64(serviceID))
	if !ok {
		return 0
	}

	service := serviceInterface.(*liboc.BoxService)
	if service.NeedWIFIState() {
		return 1
	}
	return 0
}

type windowsPlatformInterface struct {
	cInterface *C.PlatformInterface
}

func newWindowsPlatformInterface(cInterface *C.PlatformInterface) *windowsPlatformInterface {
	return &windowsPlatformInterface{
		cInterface: cInterface,
	}
}

func (w *windowsPlatformInterface) LocalDNSTransport() liboc.LocalDNSTransport {
	return nil
}

func (w *windowsPlatformInterface) UsePlatformAutoDetectInterfaceControl() bool {
	return false
}

func (w *windowsPlatformInterface) AutoDetectInterfaceControl(fd int32) error {
	return nil
}

func (w *windowsPlatformInterface) OpenTun(options liboc.TunOptions) (int32, error) {
	return -1, E.New("OpenTun is not supported on Windows DLL export")
}

var (
	tunDevicesMutex sync.Mutex
	tunDevices      = make(map[int32]tun.Tun)
)

func storeTunDevice(fd int32, device tun.Tun) {
	tunDevicesMutex.Lock()
	defer tunDevicesMutex.Unlock()
	tunDevices[fd] = device
}

func removeTunDevice(fd int32) {
	tunDevicesMutex.Lock()
	defer tunDevicesMutex.Unlock()
	if device, ok := tunDevices[fd]; ok {
		device.Close()
		delete(tunDevices, fd)
	}
}

func (w *windowsPlatformInterface) WriteLog(message string) {
	if w.cInterface != nil && w.cInterface.writeLog != nil {
		cMessage := C.CString(message)
		C.call_writeLog(w.cInterface.writeLog, cMessage)
		C.free(unsafe.Pointer(cMessage))
	}
}

func (w *windowsPlatformInterface) DisableColors() bool {
	return true
}

func (w *windowsPlatformInterface) UseProcFS() bool {
	return false
}

func (w *windowsPlatformInterface) FindConnectionOwner(ipProtocol int32, sourceAddress string, sourcePort int32, destinationAddress string, destinationPort int32) (int32, error) {
	if w.cInterface == nil || w.cInterface.findConnectionOwner == nil {
		return -1, E.New("FindConnectionOwner callback not set")
	}

	cSourceAddr := C.CString(sourceAddress)
	defer C.free(unsafe.Pointer(cSourceAddr))

	cDestAddr := C.CString(destinationAddress)
	defer C.free(unsafe.Pointer(cDestAddr))

	result := C.call_findConnectionOwner(
		w.cInterface.findConnectionOwner,
		C.int32_t(ipProtocol),
		cSourceAddr,
		C.int32_t(sourcePort),
		cDestAddr,
		C.int32_t(destinationPort),
	)

	return int32(result), nil
}

func (w *windowsPlatformInterface) PackageNameByUid(uid int32) (string, error) {
	if w.cInterface == nil || w.cInterface.packageNameByUid == nil {
		return "", E.New("PackageNameByUid callback not set")
	}

	cResult := C.call_packageNameByUid(w.cInterface.packageNameByUid, C.int32_t(uid))
	if cResult == nil {
		return "", E.New("package not found")
	}
	defer C.free(unsafe.Pointer(cResult))

	return C.GoString(cResult), nil
}

func (w *windowsPlatformInterface) UidByPackageName(packageName string) (int32, error) {
	if w.cInterface == nil || w.cInterface.uidByPackageName == nil {
		return -1, E.New("UidByPackageName callback not set")
	}

	cPackageName := C.CString(packageName)
	defer C.free(unsafe.Pointer(cPackageName))

	result := C.call_uidByPackageName(w.cInterface.uidByPackageName, cPackageName)
	return int32(result), nil
}

func (w *windowsPlatformInterface) StartDefaultInterfaceMonitor(listener liboc.InterfaceUpdateListener) error {
	return nil
}

func (w *windowsPlatformInterface) CloseDefaultInterfaceMonitor(listener liboc.InterfaceUpdateListener) error {
	return nil
}

func (w *windowsPlatformInterface) GetInterfaces() (liboc.NetworkInterfaceIterator, error) {
	return &emptyNetworkInterfaceIterator{}, nil
}

func (w *windowsPlatformInterface) UnderNetworkExtension() bool {
	return false
}

func (w *windowsPlatformInterface) IncludeAllNetworks() bool {
	return false
}

func (w *windowsPlatformInterface) ReadWIFIState() *liboc.WIFIState {
	return nil
}

func (w *windowsPlatformInterface) SystemCertificates() liboc.StringIterator {
	return &emptyStringIterator{}
}

func (w *windowsPlatformInterface) ClearDNSCache() {
}

func (w *windowsPlatformInterface) SendNotification(notification *liboc.Notification) error {
	return nil
}

type emptyStringIterator struct {
	items []string
	index int
}

func (e *emptyStringIterator) Next() string {
	if e.index >= len(e.items) {
		return ""
	}
	item := e.items[e.index]
	e.index++
	return item
}

func (e *emptyStringIterator) HasNext() bool {
	return e.index < len(e.items)
}

func (e *emptyStringIterator) Len() int32 {
	return int32(len(e.items))
}

type emptyNetworkInterfaceIterator struct {
	items []*liboc.NetworkInterface
	index int
}

func (e *emptyNetworkInterfaceIterator) Next() *liboc.NetworkInterface {
	if e.index >= len(e.items) {
		return nil
	}
	item := e.items[e.index]
	e.index++
	return item
}

func (e *emptyNetworkInterfaceIterator) HasNext() bool {
	return e.index < len(e.items)
}

func main() {
}
