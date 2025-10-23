#ifndef LIBOC_H
#define LIBOC_H

#ifdef __cplusplus
extern "C" {
#endif

#include <stdint.h>

typedef void (*LogCallback)(const char* message);
typedef int32_t (*FindConnectionOwnerCallback)(int32_t ipProtocol, const char* sourceAddress, int32_t sourcePort, const char* destinationAddress, int32_t destinationPort);
typedef char* (*PackageNameByUidCallback)(int32_t uid);
typedef int32_t (*UidByPackageNameCallback)(const char* packageName);
typedef void (*InterfaceUpdateCallback)(const char* interfaceName, int32_t interfaceIndex, int32_t isExpensive, int32_t isConstrained);

void FreeString(char* str);

void FreeBytes(char* data);

typedef struct {
    LogCallback writeLog;
    FindConnectionOwnerCallback findConnectionOwner;
    PackageNameByUidCallback packageNameByUid;
    UidByPackageNameCallback uidByPackageName;
    InterfaceUpdateCallback interfaceUpdate;
} PlatformInterface;

char* Setup(const char* basePath, const char* workingPath, const char* tempPath, int isTVOS, int fixAndroidStack);

void SetMemoryLimit(int enabled);

void SetLocale(const char* localeId);

char* Version(void);

char* RedirectStderr(const char* path);

int64_t NewService(const char* configContent, PlatformInterface* platformInterface);

char* LibocGetLastError(void);

char* ServiceStart(int64_t serviceID);

char* ServiceClose(int64_t serviceID);

void ServicePause(int64_t serviceID);

void ServiceWake(int64_t serviceID);

int ServiceNeedWIFIState(int64_t serviceID);

char* CheckConfig(const char* configContent);

char* FormatConfig(const char* configContent, char** formattedOut);

void ClearServiceError(void);

char* ReadServiceError(void);

char* WriteServiceError(const char* message);

#define INTERFACE_TYPE_WIFI     0
#define INTERFACE_TYPE_CELLULAR 1
#define INTERFACE_TYPE_ETHERNET 2
#define INTERFACE_TYPE_OTHER    3

#define IPPROTO_TCP 6
#define IPPROTO_UDP 17

#ifdef __cplusplus
}
#endif

#endif
