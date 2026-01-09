//go:build windows

package curl

/*
extern int c_xferinfo_shim_entrypoint(void *clientp, double dltotal, double dlnow, double ultotal, double ulnow);
*/
import "C"
import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

//go:embed libs/windows_amd64/libcurl.dll
var dllData []byte

var cgoProgressCallbackFuncptr uintptr
var onceCgoProgressCallback sync.Once

var (
	libcurlDLL *syscall.DLL
	loadErr    error
	onceLoad   sync.Once

	procCurlVersion       *syscall.Proc
	procCurlVersionInfo   *syscall.Proc
	procCurlGlobalInit    *syscall.Proc
	procCurlGlobalCleanup *syscall.Proc
	procCurlGetDate       *syscall.Proc
	procCurlFree          *syscall.Proc

	procCurlEasyInit        *syscall.Proc
	procCurlEasyDuphandle   *syscall.Proc
	procCurlEasyCleanup     *syscall.Proc
	procCurlEasySetopt      *syscall.Proc
	procCurlEasySend        *syscall.Proc
	procCurlEasyRecv        *syscall.Proc
	procCurlEasyPerform     *syscall.Proc
	procCurlEasyPause       *syscall.Proc
	procCurlEasyReset       *syscall.Proc
	procCurlEasyEscape      *syscall.Proc
	procCurlEasyUnescape    *syscall.Proc
	procCurlEasyGetinfo     *syscall.Proc
	procCurlEasyStrerror    *syscall.Proc
	procCurlEasyImpersonate *syscall.Proc

	procCurlSlistAppend  *syscall.Proc
	procCurlSlistFreeAll *syscall.Proc

	procCurlFormadd  *syscall.Proc
	procCurlFormFree *syscall.Proc

	procCurlMultiInit         *syscall.Proc
	procCurlMultiCleanup      *syscall.Proc
	procCurlMultiAddHandle    *syscall.Proc
	procCurlMultiRemoveHandle *syscall.Proc
	procCurlMultiPerform      *syscall.Proc
	procCurlMultiTimeout      *syscall.Proc
	procCurlMultiSetopt       *syscall.Proc
	procCurlMultiFdset        *syscall.Proc
	procCurlMultiInfoRead     *syscall.Proc
	procCurlMultiStrerror     *syscall.Proc
	procCurlMultiWait         *syscall.Proc

	procCurlShareInit     *syscall.Proc
	procCurlShareCleanup  *syscall.Proc
	procCurlShareSetopt   *syscall.Proc
	procCurlShareStrerror *syscall.Proc

	readCallbackFuncptr   uintptr
	writeCallbackFuncptr  uintptr
	headerCallbackFuncptr uintptr

	offsetCurlMsg_msg         = 0
	offsetCurlMsg_easy_handle = 8
	offsetCurlMsg_data_result = 16
)

func loadProcedures() {
	var dllPath string
	if len(dllData) > 0 {
		cacheDir, err := os.UserCacheDir()
		if err != nil {
			loadErr = fmt.Errorf("failed to get user cache dir: %w", err)
			return
		}
		cacheDir = filepath.Join(cacheDir, "go-curl-impersonate", "dll_cache")
		if err := os.MkdirAll(cacheDir, 0755); err != nil {
			loadErr = fmt.Errorf("failed to create DLL cache dir '%s': %w", cacheDir, err)
			return
		}

		if err := windows.SetDllDirectory(cacheDir); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to set DLL directory: %v\n", err)
		}

		hasher := sha256.New()
		hasher.Write(dllData)
		dllHash := hex.EncodeToString(hasher.Sum(nil))
		dllName := fmt.Sprintf("libcurl-impersonate-chrome-%s.dll", dllHash[:16])
		dllPath = filepath.Join(cacheDir, dllName)

		if _, err := os.Stat(dllPath); os.IsNotExist(err) {
			if err := os.WriteFile(dllPath, dllData, 0644); err != nil {
				loadErr = fmt.Errorf("failed to write embedded DLL to '%s': %w", dllPath, err)
				return
			}
		}
	} else {
		_, currentFile, _, ok := runtime.Caller(0)
		if !ok {
			loadErr = fmt.Errorf("could not get current file path for DLL loading")
			return
		}
		projectDir := filepath.Dir(filepath.Dir(currentFile))
		dllPath = filepath.Join(projectDir, "libs", "windows", "libcurl.dll")
	}

	dll, err := syscall.LoadDLL(dllPath)
	if err != nil {
		loadErr = fmt.Errorf("failed to load DLL from '%s': %w", dllPath, err)
		return
	}
	libcurlDLL = dll

	mustFindProc := func(name string) *syscall.Proc {
		if loadErr != nil {
			return nil
		}
		proc, err := libcurlDLL.FindProc(name)
		if err != nil {
			loadErr = fmt.Errorf("failed to find procedure %s in DLL: %w", name, err)
			return nil
		}
		return proc
	}

	procCurlVersion = mustFindProc("curl_version")
	procCurlVersionInfo = mustFindProc("curl_version_info")
	procCurlGlobalInit = mustFindProc("curl_global_init")
	procCurlGlobalCleanup = mustFindProc("curl_global_cleanup")
	procCurlGetDate = mustFindProc("curl_getdate")
	procCurlFree = mustFindProc("curl_free")
	procCurlEasyInit = mustFindProc("curl_easy_init")
	procCurlEasyDuphandle = mustFindProc("curl_easy_duphandle")
	procCurlEasyCleanup = mustFindProc("curl_easy_cleanup")
	procCurlEasySetopt = mustFindProc("curl_easy_setopt")
	procCurlEasySend = mustFindProc("curl_easy_send")
	procCurlEasyRecv = mustFindProc("curl_easy_recv")
	procCurlEasyPerform = mustFindProc("curl_easy_perform")
	procCurlEasyPause = mustFindProc("curl_easy_pause")
	procCurlEasyReset = mustFindProc("curl_easy_reset")
	procCurlEasyEscape = mustFindProc("curl_easy_escape")
	procCurlEasyUnescape = mustFindProc("curl_easy_unescape")
	procCurlEasyGetinfo = mustFindProc("curl_easy_getinfo")
	procCurlEasyStrerror = mustFindProc("curl_easy_strerror")
	procCurlEasyImpersonate = mustFindProc("curl_easy_impersonate")
	procCurlSlistAppend = mustFindProc("curl_slist_append")
	procCurlSlistFreeAll = mustFindProc("curl_slist_free_all")
	procCurlFormadd = mustFindProc("curl_formadd")
	procCurlFormFree = mustFindProc("curl_formfree")
	procCurlMultiInit = mustFindProc("curl_multi_init")
	procCurlMultiCleanup = mustFindProc("curl_multi_cleanup")
	procCurlMultiAddHandle = mustFindProc("curl_multi_add_handle")
	procCurlMultiRemoveHandle = mustFindProc("curl_multi_remove_handle")
	procCurlMultiPerform = mustFindProc("curl_multi_perform")
	procCurlMultiTimeout = mustFindProc("curl_multi_timeout")
	procCurlMultiSetopt = mustFindProc("curl_multi_setopt")
	procCurlMultiFdset = mustFindProc("curl_multi_fdset")
	procCurlMultiInfoRead = mustFindProc("curl_multi_info_read")
	procCurlMultiStrerror = mustFindProc("curl_multi_strerror")
	procCurlMultiWait = mustFindProc("curl_multi_wait")
	procCurlShareInit = mustFindProc("curl_share_init")
	procCurlShareCleanup = mustFindProc("curl_share_cleanup")
	procCurlShareSetopt = mustFindProc("curl_share_setopt")
	procCurlShareStrerror = mustFindProc("curl_share_strerror")
}

func initializeCgoCallbacks() {
	cgoProgressCallbackFuncptr = uintptr(unsafe.Pointer(C.c_xferinfo_shim_entrypoint))

	if cgoProgressCallbackFuncptr == 0 {
		err := fmt.Errorf("failed to get Cgo progress callback function pointer (was zero)")
		if loadErr == nil {
			loadErr = err
		} else {
			loadErr = fmt.Errorf("%w; %v", loadErr, err)
		}
		fmt.Fprintln(os.Stderr, "Error: CGO Progress callback pointer is zero after initialization.")
	}
}

func initializeNonFloatSyscallCallbacks() {
	writeCallbackFuncptr = syscall.NewCallback(goWriteFunctionTrampoline)
	readCallbackFuncptr = syscall.NewCallback(goReadFunctionTrampoline)
	headerCallbackFuncptr = syscall.NewCallback(goHeaderFunctionTrampoline)

	if writeCallbackFuncptr == 0 || readCallbackFuncptr == 0 || headerCallbackFuncptr == 0 {
		err := fmt.Errorf("failed to create one or more essential non-float syscall callbacks for libcurl")
		if loadErr == nil {
			loadErr = err
		} else {
			loadErr = fmt.Errorf("%w; %v", loadErr, err)
		}
	}
}

func init() {
	onceLoad.Do(func() {
		loadProcedures()
		if loadErr != nil {
			fmt.Fprintf(os.Stderr, "curl_impersonate_windows: DLL loading failed: %v\n", loadErr)
		} else {
			onceCgoProgressCallback.Do(initializeCgoCallbacks)
			initializeNonFloatSyscallCallbacks()
		}
	})
}

func goString(ccharPtr uintptr) string {
	if ccharPtr == 0 {
		return ""
	}
	ptr := unsafe.Pointer(ccharPtr)
	var length int
	for {
		if *(*byte)(unsafe.Pointer(uintptr(ptr) + uintptr(length))) == 0 {
			break
		}
		length++
		if length > (1 << 20) {
			fmt.Fprintf(os.Stderr, "goString: C string too long or not null-terminated at %p\n", ptr)
			return ""
		}
	}
	if length == 0 {
		return ""
	}
	return string(unsafe.Slice((*byte)(ptr), length))
}

func curlEasySetoptRaw(handle unsafe.Pointer, opt int, param uintptr) CurlCode {
	if procCurlEasySetopt == nil || handle == nil {
		return E_BAD_FUNCTION_ARGUMENT
	}
	r1, _, _ := procCurlEasySetopt.Call(uintptr(handle), uintptr(opt), param)
	return CurlCode(r1)
}

func curlEasyGetinfoRaw(handle unsafe.Pointer, info Info, p unsafe.Pointer) CurlCode {
	if procCurlEasyGetinfo == nil || handle == nil {
		return E_BAD_FUNCTION_ARGUMENT
	}
	r1, _, _ := procCurlEasyGetinfo.Call(uintptr(handle), uintptr(info), uintptr(p))
	return CurlCode(r1)
}

func curlMultiSetoptRaw(mhandle MultiHandle, option MultiOption, parameter uintptr) MultiCode {
	if procCurlMultiSetopt == nil || mhandle == nil {
		return M_BAD_HANDLE
	}
	r1, _, _ := procCurlMultiSetopt.Call(uintptr(mhandle), uintptr(option), parameter)
	return MultiCode(r1)
}

func curlShareSetoptRaw(shandle ShareHandle, option ShareOption, parameter uintptr) ShareCode {
	if procCurlShareSetopt == nil || shandle == nil {
		return SHE_BAD_OPTION
	}
	r1, _, _ := procCurlShareSetopt.Call(uintptr(shandle), uintptr(option), parameter)
	return ShareCode(r1)
}

func (e CurlError) Error() string {
	if procCurlEasyStrerror == nil {
		return "curl: (DLL not loaded) unknown error code " + strconv.FormatUint(uint64(e), 10)
	}

	curlCodeParam := uintptr(e)
	r1, _, _ := procCurlEasyStrerror.Call(curlCodeParam)

	if r1 == 0 {
		return "curl: unknown error code " + strconv.FormatUint(uint64(e), 10)
	}

	return goString(r1)
}

func CheckLoad() error {
	return loadErr
}
func GetCurlVersion() string {
	if procCurlVersion == nil {
		return "Error: curl_version proc not loaded"
	}
	r1, _, _ := procCurlVersion.Call()
	return goString(r1)
}
func GetCurlVersionInfo(ver uint32) unsafe.Pointer {
	if procCurlVersionInfo == nil {
		return nil
	}
	r1, _, err := procCurlVersionInfo.Call(uintptr(ver))
	if err != nil && err.(syscall.Errno) != 0 {
		return nil
	}
	if r1 == 0 {
		return nil
	}
	return unsafe.Pointer(r1)
}
func viGetAge(p unsafe.Pointer) uint32 {
	return (*CurlVersionInfoDataLayout)(p).Age
}
func viGetVersion(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return goString((*CurlVersionInfoDataLayout)(p).Version)
}
func viGetVersionNum(p unsafe.Pointer) uint32 {
	if p == nil {
		return 0
	}
	return (*CurlVersionInfoDataLayout)(p).VersionNum
}
func viGetHost(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return goString((*CurlVersionInfoDataLayout)(p).Host)
}
func viGetFeatures(p unsafe.Pointer) int32 {
	if p == nil {
		return 0
	}
	return (*CurlVersionInfoDataLayout)(p).Features
}
func viGetSslVersion(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return goString((*CurlVersionInfoDataLayout)(p).SslVersion)
}
func viGetSslVersionNum(p unsafe.Pointer) int32 {
	if p == nil {
		return 0
	}
	return (*CurlVersionInfoDataLayout)(p).SslVersionNum
}
func viGetLibzVersion(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return goString((*CurlVersionInfoDataLayout)(p).LibzVersion)
}
func viGetProtocols(p unsafe.Pointer) []string {
	if p == nil {
		return nil
	}
	layout := (*CurlVersionInfoDataLayout)(p)
	if layout.Protocols == 0 {
		return nil
	}
	var ps []string
	ptrSize := unsafe.Sizeof(uintptr(0))
	for i := 0; ; i++ {
		protocolNamePtrAddr := layout.Protocols + uintptr(i)*ptrSize
		protocolNamePtr := *(*uintptr)(unsafe.Pointer(protocolNamePtrAddr))
		if protocolNamePtr == 0 {
			break
		}
		ps = append(ps, goString(protocolNamePtr))
	}
	return ps
}
func viGetAres(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return goString((*CurlVersionInfoDataLayout)(p).Ares)
}
func viGetAresNum(p unsafe.Pointer) int32 {
	if p == nil {
		return 0
	}
	return (*CurlVersionInfoDataLayout)(p).AresNum
}
func viGetLibidn(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return goString((*CurlVersionInfoDataLayout)(p).Libidn)
}
func viGetIconvVerNum(p unsafe.Pointer) int32 {
	if p == nil {
		return 0
	}
	return (*CurlVersionInfoDataLayout)(p).IconvVerNum
}
func viGetLibsshVersion(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return goString((*CurlVersionInfoDataLayout)(p).LibsshVersion)
}

func CurlGlobalInit(flags int64) CurlCode {
	if procCurlGlobalInit == nil {
		return E_FAILED_INIT
	}
	r1, _, _ := procCurlGlobalInit.Call(uintptr(int32(flags)))
	return CurlCode(r1)
}
func CurlGlobalCleanup() {
	if procCurlGlobalCleanup == nil {
		return
	}
	procCurlGlobalCleanup.Call()
}
func CurlGetDate(dateString unsafe.Pointer, unused unsafe.Pointer) int64 {
	if procCurlGetDate == nil {
		return -1
	}
	r1, _, _ := procCurlGetDate.Call(uintptr(dateString), uintptr(unused))
	return int64(r1)
}

func CurlEasyInit() unsafe.Pointer {
	if procCurlEasyInit == nil {
		return nil
	}
	r1, _, _ := procCurlEasyInit.Call()
	return unsafe.Pointer(r1)
}
func CurlEasyDuph(handle unsafe.Pointer) unsafe.Pointer {
	if procCurlEasyDuphandle == nil || handle == nil {
		return nil
	}
	r1, _, _ := procCurlEasyDuphandle.Call(uintptr(handle))
	return unsafe.Pointer(r1)
}
func CurlEasyCleanup(handle unsafe.Pointer) {
	if procCurlEasyCleanup == nil || handle == nil {
		return
	}
	procCurlEasyCleanup.Call(uintptr(handle))
}
func CurlEasySetoptLong(handle unsafe.Pointer, opt int, val int64) CurlCode {
	return curlEasySetoptRaw(handle, opt, uintptr(int32(val)))
}
func CurlEasySetoptString(handle unsafe.Pointer, opt int, val unsafe.Pointer) CurlCode {
	return curlEasySetoptRaw(handle, opt, uintptr(val))
}
func CurlEasySetoptSlist(handle unsafe.Pointer, opt int, slist CurlSlist) CurlCode {
	return curlEasySetoptRaw(handle, opt, uintptr(slist))
}
func CurlEasySetoptPointer(handle unsafe.Pointer, opt int, ptr unsafe.Pointer) CurlCode {
	return curlEasySetoptRaw(handle, opt, uintptr(ptr))
}
func CurlEasySetoptOffT(handle unsafe.Pointer, opt int, val int64) CurlCode {
	return curlEasySetoptRaw(handle, opt, uintptr(val))
}
func CurlEasySetoptFunction(handle unsafe.Pointer, opt int, funcPtr unsafe.Pointer) CurlCode {
	return curlEasySetoptRaw(handle, opt, uintptr(funcPtr))
}
func CurlEasySend(handle unsafe.Pointer, buffer unsafe.Pointer, buflen int, nSent unsafe.Pointer) CurlCode {
	if procCurlEasySend == nil || handle == nil {
		return E_BAD_FUNCTION_ARGUMENT
	}
	r1, _, _ := procCurlEasySend.Call(uintptr(handle), uintptr(buffer), uintptr(buflen), uintptr(nSent))
	return CurlCode(r1)
}
func CurlEasyRecv(handle unsafe.Pointer, buffer unsafe.Pointer, buflen int, nRecv unsafe.Pointer) CurlCode {
	if procCurlEasyRecv == nil || handle == nil {
		return E_BAD_FUNCTION_ARGUMENT
	}
	r1, _, _ := procCurlEasyRecv.Call(uintptr(handle), uintptr(buffer), uintptr(buflen), uintptr(nRecv))
	return CurlCode(r1)
}
func CurlEasyPerform(handle unsafe.Pointer) CurlCode {
	if procCurlEasyPerform == nil || handle == nil {
		return E_BAD_FUNCTION_ARGUMENT
	}
	r1, _, _ := procCurlEasyPerform.Call(uintptr(handle))
	return CurlCode(r1)
}
func CurlEasyPause(handle unsafe.Pointer, bitmask int) CurlCode {
	if procCurlEasyPause == nil || handle == nil {
		return E_BAD_FUNCTION_ARGUMENT
	}
	r1, _, _ := procCurlEasyPause.Call(uintptr(handle), uintptr(bitmask))
	return CurlCode(r1)
}
func CurlEasyReset(handle unsafe.Pointer) {
	if procCurlEasyReset == nil || handle == nil {
		return
	}
	procCurlEasyReset.Call(uintptr(handle))
}
func CurlEasyEscape(handle unsafe.Pointer, url unsafe.Pointer, length int) unsafe.Pointer {
	if procCurlEasyEscape == nil {
		return nil
	}
	r1, _, _ := procCurlEasyEscape.Call(uintptr(handle), uintptr(url), uintptr(length))
	return unsafe.Pointer(r1)
}
func CurlEasyUnescape(handle unsafe.Pointer, url unsafe.Pointer, inlength int, outlength unsafe.Pointer) unsafe.Pointer {
	if procCurlEasyUnescape == nil {
		return nil
	}
	r1, _, _ := procCurlEasyUnescape.Call(uintptr(handle), uintptr(url), uintptr(inlength), uintptr(outlength))
	return unsafe.Pointer(r1)
}
func CurlEasyGetinfoString(handle unsafe.Pointer, info Info, p unsafe.Pointer) CurlCode {
	return curlEasyGetinfoRaw(handle, info, p)
}
func CurlEasyGetinfoLong(handle unsafe.Pointer, info Info, p unsafe.Pointer) CurlCode {
	return curlEasyGetinfoRaw(handle, info, p)
}
func CurlEasyGetinfoDouble(handle unsafe.Pointer, info Info, p unsafe.Pointer) CurlCode {
	return curlEasyGetinfoRaw(handle, info, p)
}
func CurlEasyGetinfoSlist(handle unsafe.Pointer, info Info, p unsafe.Pointer) CurlCode {
	return curlEasyGetinfoRaw(handle, info, p)
}
func CurlEasyImpersonate(handle unsafe.Pointer, target unsafe.Pointer, defaultHeaders int) CurlCode {
	if procCurlEasyImpersonate == nil || handle == nil {
		return E_BAD_FUNCTION_ARGUMENT
	}
	r1, _, _ := procCurlEasyImpersonate.Call(uintptr(handle), uintptr(target), uintptr(defaultHeaders))
	return CurlCode(r1)
}
func CurlEasyStrerror(code CurlCode) string {
	if procCurlEasyStrerror == nil {
		return "Error: curl_easy_strerror proc not loaded"
	}
	r1, _, _ := procCurlEasyStrerror.Call(uintptr(code))
	return goString(r1)
}

func CurlFormaddNameContentLength(httppost, last_post, name, content unsafe.Pointer, length int) uint32 {
	if procCurlFormadd == nil {
		return E_UNKNOWN_OPTION
	}
	r1, _, err := syscall.SyscallN(procCurlFormadd.Addr(),
		uintptr(httppost), uintptr(last_post),
		uintptr(FORM_COPYNAME), uintptr(name),
		uintptr(FORM_COPYCONTENTS), uintptr(content),
		uintptr(FORM_CONTENTSLENGTH), uintptr(int32(length)),
		uintptr(FORM_END))
	if err != 0 {
		return FORMADD_MEMORY
	}
	return uint32(r1)
}
func CurlFormaddNameContentLengthType(httppost, last_post, name, content unsafe.Pointer, length int, ctype unsafe.Pointer) uint32 {
	if procCurlFormadd == nil {
		return E_UNKNOWN_OPTION
	}
	r1, _, err := syscall.SyscallN(procCurlFormadd.Addr(),
		uintptr(httppost), uintptr(last_post),
		uintptr(FORM_COPYNAME), uintptr(name),
		uintptr(FORM_COPYCONTENTS), uintptr(content),
		uintptr(FORM_CONTENTSLENGTH), uintptr(int32(length)),
		uintptr(FORM_CONTENTTYPE), uintptr(ctype),
		uintptr(FORM_END))
	if err != 0 {
		return FORMADD_MEMORY
	}
	return uint32(r1)
}
func CurlFormaddNameFileType(httppost, last_post, name, filename, ctype unsafe.Pointer) uint32 {
	if procCurlFormadd == nil {
		return E_UNKNOWN_OPTION
	}
	r1, _, err := syscall.SyscallN(procCurlFormadd.Addr(),
		uintptr(httppost), uintptr(last_post),
		uintptr(FORM_COPYNAME), uintptr(name),
		uintptr(FORM_FILE), uintptr(filename),
		uintptr(FORM_CONTENTTYPE), uintptr(ctype),
		uintptr(FORM_END))
	if err != 0 {
		return FORMADD_MEMORY
	}
	return uint32(r1)
}
func CurlSlistAppend(slist CurlSlist, str unsafe.Pointer) CurlSlist {
	if procCurlSlistAppend == nil {
		return nil
	}
	r1, _, _ := procCurlSlistAppend.Call(uintptr(slist), uintptr(str))
	return CurlSlist(r1)
}
func CurlSlistFreeAll(slist CurlSlist) {
	if procCurlSlistFreeAll == nil || slist == nil {
		return
	}
	procCurlSlistFreeAll.Call(uintptr(slist))
}
func CurlFormFree(form CurlHttpFormPost) {
	if procCurlFormFree == nil || form == nil {
		return
	}
	procCurlFormFree.Call(uintptr(form))
}
func CurlFree(ptr unsafe.Pointer) {
	if procCurlFree == nil || ptr == nil {
		return
	}
	procCurlFree.Call(uintptr(ptr))
}
func GetCurlFormaddOk() uint32 {
	return uint32(FORMADD_OK)
}

func CurlMultiInit() MultiHandle {
	if procCurlMultiInit == nil {
		return nil
	}
	r1, _, _ := procCurlMultiInit.Call()
	return MultiHandle(r1)
}
func CurlMultiCleanup(mhandle MultiHandle) MultiCode {
	if procCurlMultiCleanup == nil || mhandle == nil {
		return M_BAD_HANDLE
	}
	r1, _, _ := procCurlMultiCleanup.Call(uintptr(mhandle))
	return MultiCode(r1)
}
func CurlMultiAddHandle(mhandle MultiHandle, easyHandle unsafe.Pointer) MultiCode {
	if procCurlMultiAddHandle == nil || mhandle == nil || easyHandle == nil {
		return M_BAD_HANDLE
	}
	r1, _, _ := procCurlMultiAddHandle.Call(uintptr(mhandle), uintptr(easyHandle))
	return MultiCode(r1)
}
func CurlMultiRemoveHandle(mhandle MultiHandle, easyHandle unsafe.Pointer) MultiCode {
	if procCurlMultiRemoveHandle == nil || mhandle == nil || easyHandle == nil {
		return M_BAD_HANDLE
	}
	r1, _, _ := procCurlMultiRemoveHandle.Call(uintptr(mhandle), uintptr(easyHandle))
	return MultiCode(r1)
}
func CurlMultiPerform(mhandle MultiHandle, runningHandles unsafe.Pointer) MultiCode {
	if procCurlMultiPerform == nil || mhandle == nil {
		return M_BAD_HANDLE
	}
	r1, _, _ := procCurlMultiPerform.Call(uintptr(mhandle), uintptr(runningHandles))
	return MultiCode(r1)
}
func CurlMultiTimeout(mhandle MultiHandle, timeoutMs unsafe.Pointer) MultiCode {
	if procCurlMultiTimeout == nil || mhandle == nil {
		return M_BAD_HANDLE
	}
	r1, _, _ := procCurlMultiTimeout.Call(uintptr(mhandle), uintptr(timeoutMs))
	return MultiCode(r1)
}
func CurlMultiSetoptLong(mhandle MultiHandle, option MultiOption, val int64) MultiCode {
	return curlMultiSetoptRaw(mhandle, option, uintptr(int32(val)))
}
func CurlMultiSetoptPointer(mhandle MultiHandle, option MultiOption, ptr unsafe.Pointer) MultiCode {
	return curlMultiSetoptRaw(mhandle, option, uintptr(ptr))
}
func CurlMultiFdset(mhandle MultiHandle, readFdSet FdSetPlaceholder, writeFdSet FdSetPlaceholder, excFdSet FdSetPlaceholder, maxFd unsafe.Pointer) MultiCode {
	if procCurlMultiFdset == nil || mhandle == nil {
		return M_BAD_HANDLE
	}
	r1, _, _ := procCurlMultiFdset.Call(uintptr(mhandle), uintptr(readFdSet), uintptr(writeFdSet), uintptr(excFdSet), uintptr(maxFd))
	return MultiCode(r1)
}
func CurlMultiStrerror(code MultiCode) string {
	if procCurlMultiStrerror == nil {
		return "Error: curl_multi_strerror proc not loaded"
	}
	r1, _, _ := procCurlMultiStrerror.Call(uintptr(code))
	return goString(r1)
}
func CurlMultiWait(mhandle MultiHandle, extraFds unsafe.Pointer, numExtraFds int, timeoutMs int, numFdsReady unsafe.Pointer) MultiCode {
	if procCurlMultiWait == nil || mhandle == nil {
		return M_BAD_HANDLE
	}
	r1, _, _ := procCurlMultiWait.Call(uintptr(mhandle), uintptr(extraFds), uintptr(numExtraFds), uintptr(timeoutMs), uintptr(numFdsReady))
	return MultiCode(r1)
}
func CurlMultiInfoRead(mhandle MultiHandle, msgsInQueue unsafe.Pointer) CurlMsg {
	if procCurlMultiInfoRead == nil || mhandle == nil {
		return nil
	}
	r1, _, _ := procCurlMultiInfoRead.Call(uintptr(mhandle), uintptr(msgsInQueue))
	return CurlMsg(r1)
}

func CurlShareInit() ShareHandle {
	if procCurlShareInit == nil {
		return nil
	}
	r1, _, _ := procCurlShareInit.Call()
	return ShareHandle(r1)
}
func CurlShareCleanup(shandle ShareHandle) ShareCode {
	if procCurlShareCleanup == nil || shandle == nil {
		return SHE_INVALID
	}
	r1, _, _ := procCurlShareCleanup.Call(uintptr(shandle))
	return ShareCode(r1)
}
func CurlShareSetoptLong(shandle ShareHandle, option ShareOption, val int64) ShareCode {
	return curlShareSetoptRaw(shandle, option, uintptr(int32(val)))
}
func CurlShareSetoptPointer(shandle ShareHandle, option ShareOption, ptr unsafe.Pointer) ShareCode {
	return curlShareSetoptRaw(shandle, option, uintptr(ptr))
}
func CurlShareStrerror(code ShareCode) string {
	if procCurlShareStrerror == nil {
		return "Error: curl_share_strerror proc not loaded"
	}
	r1, _, _ := procCurlShareStrerror.Call(uintptr(code))
	return goString(r1)
}

func GetCurlmOk() MultiCode           { return M_OK }
func GetCurlshOk() ShareCode          { return SHE_OK }
func GetCurlmsgDone() CurlMultiMsgTag { return CURLMSG_DONE }
func GetCurlWritefuncPause() C.size_t { return C.size_t(WRITEFUNC_PAUSE) }
func GetCurlReadfuncAbort() C.size_t  { return C.size_t(READFUNC_ABORT) }
func GetCurlReadfuncPause() C.size_t  { return C.size_t(READFUNC_PAUSE) }

func CurlMsgGetMsg(cm CurlMsg) CurlMultiMsgTag {
	if cm == nil {
		return CURLMSG_NONE
	}
	return CurlMultiMsgTag(*(*int32)(unsafe.Pointer(uintptr(cm) + uintptr(offsetCurlMsg_msg))))
}
func CurlMsgGetEasyHandle(cm CurlMsg) unsafe.Pointer {
	if cm == nil {
		return nil
	}
	return *(*unsafe.Pointer)(unsafe.Pointer(uintptr(cm) + uintptr(offsetCurlMsg_easy_handle)))
}
func CurlMsgGetResult(cm CurlMsg) Code {
	if cm == nil {
		return E_OK
	}
	if CurlMsgGetMsg(cm) == CURLMSG_DONE {
		return Code(*(*int32)(unsafe.Pointer(uintptr(cm) + uintptr(offsetCurlMsg_data_result))))
	}
	return E_OK
}
func CurlMsgGetWhatever(cm CurlMsg) unsafe.Pointer {
	if cm == nil {
		return nil
	}
	msgType := CurlMsgGetMsg(cm)
	if msgType != CURLMSG_DONE {
		ptrToDataUnion := unsafe.Pointer(uintptr(cm) + uintptr(offsetCurlMsg_data_result))
		return *(*unsafe.Pointer)(ptrToDataUnion)
	}
	return nil
}

func GetCurlInfoTypeMask() Info { return INFO_TYPEMASK }
func GetCurlInfoString() Info   { return INFO_STRING }
func GetCurlInfoLong() Info     { return INFO_LONG }
func GetCurlInfoDouble() Info   { return INFO_DOUBLE }
func GetCurlInfoSList() Info    { return INFO_SLIST }

func GetWriteCallbackFuncptr() unsafe.Pointer {
	return unsafe.Pointer(writeCallbackFuncptr)
}
func GetReadCallbackFuncptr() unsafe.Pointer {
	return unsafe.Pointer(readCallbackFuncptr)
}
func GetHeaderCallbackFuncptr() unsafe.Pointer {
	return unsafe.Pointer(headerCallbackFuncptr)
}
func GetProgressCallbackFuncptr() unsafe.Pointer {
	if cgoProgressCallbackFuncptr == 0 {
		onceCgoProgressCallback.Do(initializeCgoCallbacks)
		if cgoProgressCallbackFuncptr == 0 && loadErr == nil {
			fmt.Fprintln(os.Stderr, "Warning: CGO Progress callback pointer is zero; initialization failed.")
		} else if cgoProgressCallbackFuncptr == 0 && loadErr != nil {
			fmt.Fprintf(os.Stderr, "Warning: CGO Progress callback pointer is zero due to: %v\n", loadErr)
		}
	}
	return unsafe.Pointer(cgoProgressCallbackFuncptr)
}

func goWriteFunctionTrampoline(ptr, size, nmemb, userdata uintptr) uintptr {
	curl := context_map.Get(userdata)
	if curl == nil || curl.writeFunction == nil {
		return 0
	}
	bufLen := int(size * nmemb)
	if bufLen == 0 {
		return 0
	}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), bufLen)
	if (*curl.writeFunction)(buf, curl.writeData) {
		return uintptr(bufLen)
	}
	return uintptr(WRITEFUNC_PAUSE)
}

func goReadFunctionTrampoline(buffer, size, nitems, instream uintptr) uintptr {
	curl := context_map.Get(instream)
	if curl == nil || curl.readFunction == nil {
		return uintptr(READFUNC_ABORT)
	}
	bufLen := int(size * nitems)
	goSliceForReading := unsafe.Slice((*byte)(unsafe.Pointer(buffer)), bufLen)
	bytesWrittenByGoFunc := (*curl.readFunction)(goSliceForReading, curl.readData)
	if bytesWrittenByGoFunc < 0 {
		return uintptr(READFUNC_ABORT)
	}
	if bytesWrittenByGoFunc > bufLen {
		return uintptr(READFUNC_ABORT)
	}
	return uintptr(bytesWrittenByGoFunc)
}

func goHeaderFunctionTrampoline(buffer, size, nitems, userdata uintptr) uintptr {
	curl := context_map.Get(userdata)
	if curl == nil || curl.headerFunction == nil {
		return 0
	}
	bufLen := int(size * nitems)
	if bufLen == 0 {
		return 0
	}
	buf := unsafe.Slice((*byte)(unsafe.Pointer(buffer)), bufLen)
	if (*curl.headerFunction)(buf, curl.headerData) {
		return uintptr(bufLen)
	}
	return uintptr(WRITEFUNC_PAUSE)
}
