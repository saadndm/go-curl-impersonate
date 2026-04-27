// libcurl go bingding
package curl

/*
#cgo CFLAGS: -I${SRCDIR}/libs/include -DCURL_STATICLIB

#include <stdlib.h>

static char *string_array_index(char **p, int i) {
  return p[i];
}
*/
import "C"

import (
	"time"
	"unsafe"
)

// curl_global_init - Global libcurl initialisation
func GlobalInit(flags int64) error {
	return newCurlError(CurlGlobalInit(flags))
}

// curl_global_cleanup - global libcurl cleanup
func GlobalCleanup() {
	CurlGlobalCleanup()
}

type VersionInfoData struct {
	Age CurlVersion
	// age >= 0
	Version       string
	VersionNum    uint
	Host          string
	Features      int
	SslVersion    string
	SslVersionNum int
	LibzVersion   string
	Protocols     []string
	// age >= 1
	Ares    string
	AresNum int
	// age >= 2
	Libidn string
	// age >= 3
	IconvVerNum   int
	LibsshVersion string
	BrotliVerNum  uint32
	BrotliVersion string
}

// curl_version - returns the libcurl version string
func Version() string {
	return GetCurlVersion()
}

// curl_version_info - returns run-time libcurl version info
func VersionInfo(ver CurlVersion) *VersionInfoData {
	dataPtr := GetCurlVersionInfo(uint32(ver))
	if dataPtr == nil {
		return nil
	}

	ret := new(VersionInfoData)
	ret.Age = CurlVersion(viGetAge(dataPtr))
	switch age := ret.Age; {
	case age >= 0:
		ret.Version = viGetVersion(dataPtr)
		ret.VersionNum = uint(viGetVersionNum(dataPtr))
		ret.Host = viGetHost(dataPtr)
		ret.Features = int(viGetFeatures(dataPtr))
		ret.SslVersion = viGetSslVersion(dataPtr)
		ret.SslVersionNum = int(viGetSslVersionNum(dataPtr))
		ret.LibzVersion = viGetLibzVersion(dataPtr)
		// ugly but works
		ret.Protocols = viGetProtocols(dataPtr)
		fallthrough
	case age >= 1:
		ret.Ares = viGetAres(dataPtr)
		ret.AresNum = int(viGetAresNum(dataPtr))
		fallthrough
	case age >= 2:
		ret.Libidn = viGetLibidn(dataPtr)
		fallthrough
	case age >= 3:
		ret.IconvVerNum = int(viGetIconvVerNum(dataPtr))
		ret.LibsshVersion = viGetLibsshVersion(dataPtr)
	}
	return ret
}

// curl_getdate - Convert a date string to number of seconds since January 1, 1970
// In golang, we convert it to a *time.Time
func Getdate(date string) *time.Time {
	datestr := C.CString(date)
	defer C.free(unsafe.Pointer(datestr))
	t := CurlGetDate(unsafe.Pointer(datestr), nil)
	if t == -1 {
		return nil
	}
	unix := time.Unix(int64(t), 0).UTC()
	return &unix

	/*
	   // curl_getenv - return value for environment name
	   func Getenv(name string) string {
	           namestr := C.CString(name)
	           defer C.free(unsafe.Pointer(namestr))
	           ret := C.curl_getenv(unsafe.Pointer(namestr))
	           defer C.free(unsafe.Pointer(ret))

	           return C.GoString(ret)
	   }
	*/
}

// TODO: curl_global_init_mem
