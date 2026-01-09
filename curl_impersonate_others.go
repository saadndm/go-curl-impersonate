//go:build !windows

package curl

/*
#include <stdlib.h>
#include <curl/curl.h>
#include "compat.h"
#include <sys/types.h>
#include <sys/select.h>

static CURLcode easy_setopt_long_helper(CURL *handle, CURLoption option, long parameter) {
    return curl_easy_setopt(handle, option, parameter);
}
static CURLcode easy_setopt_string_helper(CURL *handle, CURLoption option, char *parameter) {
    return curl_easy_setopt(handle, option, parameter);
}
static CURLcode easy_setopt_slist_helper(CURL *handle, CURLoption option, struct curl_slist *parameter) {
    return curl_easy_setopt(handle, option, parameter);
}
static CURLcode easy_setopt_pointer_helper(CURL *handle, CURLoption option, void *parameter) {
    return curl_easy_setopt(handle, option, parameter);
}
static CURLcode easy_setopt_off_t_helper(CURL *handle, CURLoption option, off_t parameter) {
    return curl_easy_setopt(handle, option, parameter);
}

static CURLcode easy_getinfo_string_helper(CURL *curl, CURLINFO info, char **p) {
    return curl_easy_getinfo(curl, info, p);
}
static CURLcode easy_getinfo_long_helper(CURL *curl, CURLINFO info, long *p) {
    return curl_easy_getinfo(curl, info, p);
}
static CURLcode easy_getinfo_double_helper(CURL *curl, CURLINFO info, double *p) {
    return curl_easy_getinfo(curl, info, p);
}
static CURLcode easy_getinfo_slist_helper(CURL *curl, CURLINFO info, struct curl_slist **p) {
    return curl_easy_getinfo(curl, info, p);
}

static CURLFORMcode formadd_helper_copyname_copycontents_contentslength(
    struct curl_httppost **httppost, struct curl_httppost **last_post,
    const char *name, const char *contents, long contentslength) {
    return curl_formadd(httppost, last_post,
        CURLFORM_COPYNAME, name,
        CURLFORM_COPYCONTENTS, contents,
        CURLFORM_CONTENTSLENGTH, contentslength,
        CURLFORM_END);
}

static CURLFORMcode formadd_helper_copyname_copycontents_contentslength_contenttype(
    struct curl_httppost **httppost, struct curl_httppost **last_post,
    const char *name, const char *contents, long contentslength, const char *contenttype) {
    return curl_formadd(httppost, last_post,
        CURLFORM_COPYNAME, name,
        CURLFORM_COPYCONTENTS, contents,
        CURLFORM_CONTENTSLENGTH, contentslength,
        CURLFORM_CONTENTTYPE, contenttype,
        CURLFORM_END);
}

static CURLFORMcode formadd_helper_copyname_file_contenttype(
    struct curl_httppost **httppost, struct curl_httppost **last_post,
    const char *name, const char *filename, const char *contenttype) {
    return curl_formadd(httppost, last_post,
        CURLFORM_COPYNAME, name,
        CURLFORM_FILE, filename,
        CURLFORM_CONTENTTYPE, contenttype,
        CURLFORM_END);
}

static CURLMcode multi_setopt_long_helper(CURLM *handle, CURLMoption option, long parameter) {
    return curl_multi_setopt(handle, option, parameter);
}
static CURLMcode multi_setopt_pointer_helper(CURLM *handle, CURLMoption option, void *parameter) {
    return curl_multi_setopt(handle, option, parameter);
}

static CURLSHcode share_setopt_long_helper(CURLSH *handle, CURLSHoption option, long parameter) {
    return curl_share_setopt(handle, option, parameter);
}
static CURLSHcode share_setopt_pointer_helper(CURLSH *handle, CURLSHoption option, void *parameter) {
    return curl_share_setopt(handle, option, parameter);
}

static CURLMSG curl_msg_get_msg(CURLMsg *cm) {
    if (cm) return cm->msg;
    return CURLMSG_NONE;
}

static CURL* curl_msg_get_easy_handle(CURLMsg *cm) {
    if (cm) return cm->easy_handle;
    return NULL;
}

static CURLcode curl_msg_get_result(CURLMsg *cm) {
    if (cm && cm->msg == CURLMSG_DONE) return cm->data.result;
    return CURLE_OK;
}

static void* curl_msg_get_whatever(CURLMsg *cm) {
    if (cm && cm->msg != CURLMSG_DONE) return cm->data.whatever;
    return NULL;
}

static char* get_protocol_at_index(char **protocols_array, int index) {
    if (protocols_array == NULL) {
        return NULL;
    }
    return protocols_array[index];
}

typedef size_t (*c_go_write_callback_t)(char *buffer, size_t size, size_t nitems, void *userdata);
typedef size_t (*c_go_read_callback_t)(char *buffer, size_t size, size_t nitems, void *instream);
typedef int (*c_go_xferinfo_callback_t)(void *clientp, curl_off_t dltotal, curl_off_t dlnow, curl_off_t ultotal, curl_off_t ulnow);

extern size_t GoWriteFunctionTrampoline(char *buffer, size_t size, size_t nitems, void *userdata);
extern size_t GoReadFunctionTrampoline(char *buffer, size_t size, size_t nitems, void *instream);
extern size_t GoHeaderFunctionTrampoline(char *buffer, size_t size, size_t nitems, void *userdata);
extern int GoProgressFunctionTrampoline(void *clientp, curl_off_t dltotal, curl_off_t dlnow, curl_off_t ultotal, curl_off_t ulnow);

static c_go_write_callback_t get_c_write_callback_ptr() {
    return GoWriteFunctionTrampoline;
}
static c_go_read_callback_t get_c_read_callback_ptr() {
    return GoReadFunctionTrampoline;
}
static c_go_write_callback_t get_c_header_callback_ptr() {
    return GoHeaderFunctionTrampoline;
}
static c_go_xferinfo_callback_t get_c_progress_callback_ptr() {
    return GoProgressFunctionTrampoline;
}

static CURLMcode multi_wait_helper(CURLM *multi_handle,
                                   struct curl_waitfd extra_fds[],
                                   unsigned int extra_nfds,
                                   int timeout_ms,
                                   int *ret_nfds) {
    return curl_multi_wait(multi_handle,
                           (struct curl_waitfd*)extra_fds,
                           extra_nfds,
                           timeout_ms,
                           ret_nfds);
}
*/
import "C"

import (
	"strconv"
	"unsafe"
)

func (e CurlError) Error() string {
	cErr := C.CURLcode(e)
	errStrChars := C.curl_easy_strerror(cErr)
	if errStrChars == nil {
		return "curl: unknown error code " + strconv.FormatUint(uint64(e), 10)
	}
	return C.GoString(errStrChars)
}

func CheckLoad() error {
	return nil
}

func GetCurlVersion() string {
	return C.GoString(C.curl_version())
}

func GetCurlVersionInfo(ver uint32) unsafe.Pointer {
	return unsafe.Pointer(C.curl_version_info(C.CURLversion(ver)))
}

func viGetAge(p unsafe.Pointer) uint32 {
	if p == nil {
		return 0
	}
	return uint32((*C.curl_version_info_data)(p).age)
}

func viGetVersion(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return C.GoString((*C.curl_version_info_data)(p).version)
}

func viGetVersionNum(p unsafe.Pointer) uint32 {
	if p == nil {
		return 0
	}
	return uint32((*C.curl_version_info_data)(p).version_num)
}

func viGetHost(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return C.GoString((*C.curl_version_info_data)(p).host)
}

func viGetFeatures(p unsafe.Pointer) int32 {
	if p == nil {
		return 0
	}
	return int32((*C.curl_version_info_data)(p).features)
}

func viGetSslVersion(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return C.GoString((*C.curl_version_info_data)(p).ssl_version)
}

func viGetSslVersionNum(p unsafe.Pointer) int32 {
	if p == nil {
		return 0
	}
	return int32((*C.curl_version_info_data)(p).ssl_version_num)
}

func viGetLibzVersion(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return C.GoString((*C.curl_version_info_data)(p).libz_version)
}

func viGetProtocols(p unsafe.Pointer) []string {
	if p == nil {
		return nil
	}
	data := (*C.curl_version_info_data)(p)
	if data.protocols == nil {
		return nil
	}
	var ps []string
	for i := 0; ; i++ {
		cProto := C.get_protocol_at_index(data.protocols, C.int(i))
		if cProto == nil {
			break
		}
		ps = append(ps, C.GoString(cProto))
	}
	return ps
}

func viGetAres(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return C.GoString((*C.curl_version_info_data)(p).ares)
}

func viGetAresNum(p unsafe.Pointer) int32 {
	if p == nil {
		return 0
	}
	return int32((*C.curl_version_info_data)(p).ares_num)
}

func viGetLibidn(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return C.GoString((*C.curl_version_info_data)(p).libidn)
}

func viGetIconvVerNum(p unsafe.Pointer) int32 {
	if p == nil {
		return 0
	}
	return int32((*C.curl_version_info_data)(p).iconv_ver_num)
}

func viGetLibsshVersion(p unsafe.Pointer) string {
	if p == nil {
		return ""
	}
	return C.GoString((*C.curl_version_info_data)(p).libssh_version)
}

func CurlGlobalInit(flags int64) CurlCode {
	return CurlCode(C.curl_global_init(C.long(flags)))
}

func CurlGlobalCleanup() {
	C.curl_global_cleanup()
}

func CurlGetDate(date unsafe.Pointer, unused unsafe.Pointer) int64 {
	return int64(C.curl_getdate((*C.char)(date), (*C.time_t)(unused)))
}

func CurlEasyInit() unsafe.Pointer {
	return unsafe.Pointer(C.curl_easy_init())
}

func CurlEasyDuph(handle unsafe.Pointer) unsafe.Pointer {
	return unsafe.Pointer(C.curl_easy_duphandle(handle))
}

func CurlEasyCleanup(handle unsafe.Pointer) {
	C.curl_easy_cleanup(handle)
}

func CurlEasySetoptLong(handle unsafe.Pointer, opt int, val int64) CurlCode {
	return CurlCode(C.easy_setopt_long_helper(handle, C.CURLoption(opt), C.long(val)))
}

func CurlEasySetoptString(handle unsafe.Pointer, opt int, val unsafe.Pointer) CurlCode {
	return CurlCode(C.easy_setopt_string_helper(handle, C.CURLoption(opt), (*C.char)(val)))
}

func CurlEasySetoptSlist(handle unsafe.Pointer, opt int, val CurlSlist) CurlCode {
	return CurlCode(C.easy_setopt_slist_helper(handle, C.CURLoption(opt), (*C.struct_curl_slist)(val)))
}

func CurlEasySetoptPointer(handle unsafe.Pointer, opt int, val unsafe.Pointer) CurlCode {
	return CurlCode(C.easy_setopt_pointer_helper(handle, C.CURLoption(opt), val))
}

func CurlEasySetoptOffT(handle unsafe.Pointer, opt int, val int64) CurlCode {
	return CurlCode(C.easy_setopt_off_t_helper(handle, C.CURLoption(opt), C.off_t(val)))
}

func CurlEasySetoptFunction(handle unsafe.Pointer, opt int, funcPtr unsafe.Pointer) CurlCode {
	return CurlCode(C.easy_setopt_pointer_helper(handle, C.CURLoption(opt), funcPtr))
}

func CurlEasySend(handle unsafe.Pointer, buf unsafe.Pointer, buflen int, n unsafe.Pointer) CurlCode {
	return CurlCode(C.curl_easy_send(handle, buf, C.size_t(buflen), (*C.size_t)(n)))
}

func CurlEasyRecv(handle unsafe.Pointer, buf unsafe.Pointer, buflen int, n unsafe.Pointer) CurlCode {
	return CurlCode(C.curl_easy_recv(handle, buf, C.size_t(buflen), (*C.size_t)(n)))
}

func CurlEasyPerform(handle unsafe.Pointer) CurlCode {
	return CurlCode(C.curl_easy_perform(handle))
}

func CurlEasyPause(handle unsafe.Pointer, bitmask int) CurlCode {
	return CurlCode(C.curl_easy_pause(handle, C.int(bitmask)))
}

func CurlEasyReset(handle unsafe.Pointer) {
	C.curl_easy_reset(handle)
}

func CurlEasyEscape(handle unsafe.Pointer, url unsafe.Pointer, length int) unsafe.Pointer {
	return unsafe.Pointer(C.curl_easy_escape(handle, (*C.char)(url), C.int(length)))
}

func CurlEasyUnescape(handle unsafe.Pointer, url unsafe.Pointer, inlength int, outlength unsafe.Pointer) unsafe.Pointer {
	return unsafe.Pointer(C.curl_easy_unescape(handle, (*C.char)(url), C.int(inlength), (*C.int)(outlength)))
}

func CurlEasyGetinfoString(handle unsafe.Pointer, info Info, p unsafe.Pointer) CurlCode {
	return CurlCode(C.easy_getinfo_string_helper(handle, C.CURLINFO(info), (**C.char)(p)))
}

func CurlEasyGetinfoLong(handle unsafe.Pointer, info Info, p unsafe.Pointer) CurlCode {
	return CurlCode(C.easy_getinfo_long_helper(handle, C.CURLINFO(info), (*C.long)(p)))
}

func CurlEasyGetinfoDouble(handle unsafe.Pointer, info Info, p unsafe.Pointer) CurlCode {
	return CurlCode(C.easy_getinfo_double_helper(handle, C.CURLINFO(info), (*C.double)(p)))
}

func CurlEasyGetinfoSlist(handle unsafe.Pointer, info Info, p unsafe.Pointer) CurlCode {
	return CurlCode(C.easy_getinfo_slist_helper(handle, C.CURLINFO(info), (**C.struct_curl_slist)(p)))
}

func CurlEasyImpersonate(handle unsafe.Pointer, target unsafe.Pointer, defaultHeaders int) CurlCode {
	return CurlCode(C.curl_easy_impersonate(handle, (*C.char)(target), C.int(defaultHeaders)))
}

func CurlEasyStrerror(e CurlCode) string {
	return C.GoString(C.curl_easy_strerror(C.CURLcode(e)))
}

func CurlFormaddNameContentLength(httppost unsafe.Pointer, last_post unsafe.Pointer, name unsafe.Pointer, content unsafe.Pointer, length int) uint32 {
	return uint32(C.formadd_helper_copyname_copycontents_contentslength(
		(**C.struct_curl_httppost)(httppost),
		(**C.struct_curl_httppost)(last_post),
		(*C.char)(name), (*C.char)(content), C.long(length)))
}

func CurlFormaddNameContentLengthType(httppost unsafe.Pointer, last_post unsafe.Pointer, name unsafe.Pointer, content unsafe.Pointer, length int, ctype unsafe.Pointer) uint32 {
	return uint32(C.formadd_helper_copyname_copycontents_contentslength_contenttype(
		(**C.struct_curl_httppost)(httppost),
		(**C.struct_curl_httppost)(last_post),
		(*C.char)(name), (*C.char)(content), C.long(length), (*C.char)(ctype)))
}

func CurlFormaddNameFileType(httppost unsafe.Pointer, last_post unsafe.Pointer, name unsafe.Pointer, filename unsafe.Pointer, ctype unsafe.Pointer) uint32 {
	return uint32(C.formadd_helper_copyname_file_contenttype(
		(**C.struct_curl_httppost)(httppost),
		(**C.struct_curl_httppost)(last_post),
		(*C.char)(name), (*C.char)(filename), (*C.char)(ctype)))
}

func CurlSlistAppend(slist CurlSlist, str unsafe.Pointer) CurlSlist {
	return CurlSlist(C.curl_slist_append((*C.struct_curl_slist)(slist), (*C.char)(str)))
}

func CurlSlistFreeAll(slist CurlSlist) {
	C.curl_slist_free_all((*C.struct_curl_slist)(slist))
}

func CurlFormFree(form CurlHttpFormPost) {
	C.curl_formfree((*C.struct_curl_httppost)(form))
}

func CurlFree(ptr unsafe.Pointer) {
	C.curl_free(ptr)
}

func GetCurlFormaddOk() uint32 {
	return uint32(C.CURL_FORMADD_OK)
}

func CurlMultiInit() MultiHandle {
	return MultiHandle(C.curl_multi_init())
}

func CurlMultiCleanup(mhandle MultiHandle) MultiCode {
	return MultiCode(C.curl_multi_cleanup(unsafe.Pointer(mhandle)))
}

func CurlMultiAddHandle(mhandle MultiHandle, easyHandle unsafe.Pointer) MultiCode {
	return MultiCode(C.curl_multi_add_handle(unsafe.Pointer(mhandle), easyHandle))
}

func CurlMultiRemoveHandle(mhandle MultiHandle, easyHandle unsafe.Pointer) MultiCode {
	return MultiCode(C.curl_multi_remove_handle(unsafe.Pointer(mhandle), easyHandle))
}

func CurlMultiPerform(mhandle MultiHandle, runningHandles unsafe.Pointer) MultiCode {
	return MultiCode(C.curl_multi_perform(unsafe.Pointer(mhandle), (*C.int)(runningHandles)))
}

func CurlMultiTimeout(mhandle MultiHandle, timeoutMs unsafe.Pointer) MultiCode {
	return MultiCode(C.curl_multi_timeout(unsafe.Pointer(mhandle), (*C.long)(timeoutMs)))
}

func CurlMultiSetoptLong(mhandle MultiHandle, option MultiOption, parameter int64) MultiCode {
	return MultiCode(C.multi_setopt_long_helper(unsafe.Pointer(mhandle), C.CURLMoption(option), C.long(parameter)))
}

func CurlMultiSetoptPointer(mhandle MultiHandle, option MultiOption, parameter unsafe.Pointer) MultiCode {
	return MultiCode(C.multi_setopt_pointer_helper(unsafe.Pointer(mhandle), C.CURLMoption(option), parameter))
}

func CurlMultiFdset(mhandle MultiHandle, readFdSet FdSetPlaceholder, writeFdSet FdSetPlaceholder, excFdSet FdSetPlaceholder, maxFd unsafe.Pointer) MultiCode {
	return MultiCode(C.curl_multi_fdset(
		unsafe.Pointer(mhandle),
		(*C.fd_set)(readFdSet), (*C.fd_set)(writeFdSet), (*C.fd_set)(excFdSet),
		(*C.int)(maxFd)))
}

func CurlMultiStrerror(e MultiCode) string {
	return C.GoString(C.curl_multi_strerror(C.CURLMcode(e)))
}

func CurlMultiWait(mhandle MultiHandle, extraFds unsafe.Pointer, numExtraFds int, timeoutMs int, numFdsReady unsafe.Pointer) MultiCode {
	if mhandle == nil {
		return M_BAD_HANDLE
	}

	ret := C.multi_wait_helper(
		unsafe.Pointer(mhandle),
		(*C.struct_curl_waitfd)(extraFds),
		C.uint(numExtraFds),
		C.int(timeoutMs),
		(*C.int)(numFdsReady),
	)
	return MultiCode(ret)
}

func CurlMultiInfoRead(mhandle MultiHandle, msgsInQueue unsafe.Pointer) CurlMsg {
	return CurlMsg(C.curl_multi_info_read(unsafe.Pointer(mhandle), (*C.int)(msgsInQueue)))
}

func CurlShareInit() ShareHandle {
	return ShareHandle(C.curl_share_init())
}

func CurlShareCleanup(shandle ShareHandle) ShareCode {
	return ShareCode(C.curl_share_cleanup(unsafe.Pointer(shandle)))
}

func CurlShareSetoptLong(shandle ShareHandle, option ShareOption, parameter int64) ShareCode {
	return ShareCode(C.share_setopt_long_helper(unsafe.Pointer(shandle), C.CURLSHoption(option), C.long(parameter)))
}

func CurlShareSetoptPointer(shandle ShareHandle, option ShareOption, parameter unsafe.Pointer) ShareCode {
	return ShareCode(C.share_setopt_pointer_helper(unsafe.Pointer(shandle), C.CURLSHoption(option), parameter))
}

func CurlShareStrerror(e ShareCode) string {
	return C.GoString(C.curl_share_strerror(C.CURLSHcode(e)))
}

func GetCurlmOk() MultiCode           { return MultiCode(C.CURLM_OK) }
func GetCurlshOk() ShareCode          { return ShareCode(C.CURLSHE_OK) }
func GetCurlmsgDone() CurlMultiMsgTag { return CurlMultiMsgTag(C.CURLMSG_DONE) }
func GetCurlWritefuncPause() C.size_t { return C.CURL_WRITEFUNC_PAUSE }
func GetCurlReadfuncAbort() C.size_t  { return C.CURL_READFUNC_ABORT }
func GetCurlReadfuncPause() C.size_t  { return C.CURL_READFUNC_PAUSE }

func CurlMsgGetMsg(cm CurlMsg) CurlMultiMsgTag {
	if cm == nil {
		return CurlMultiMsgTag(C.CURLMSG_NONE)
	}
	return CurlMultiMsgTag(C.curl_msg_get_msg((*C.CURLMsg)(cm)))
}

func CurlMsgGetEasyHandle(cm CurlMsg) unsafe.Pointer {
	if cm == nil {
		return nil
	}
	return unsafe.Pointer(C.curl_msg_get_easy_handle((*C.CURLMsg)(cm)))
}

func CurlMsgGetResult(cm CurlMsg) Code {
	if cm == nil {
		return Code(C.CURLE_OK)
	}
	return Code(C.curl_msg_get_result((*C.CURLMsg)(cm)))
}

func CurlMsgGetWhatever(cm CurlMsg) unsafe.Pointer {
	if cm == nil {
		return nil
	}
	return C.curl_msg_get_whatever((*C.CURLMsg)(cm))
}

func GetCurlInfoTypeMask() Info { return Info(C.CURLINFO_TYPEMASK) }
func GetCurlInfoString() Info   { return Info(C.CURLINFO_STRING) }
func GetCurlInfoLong() Info     { return Info(C.CURLINFO_LONG) }
func GetCurlInfoDouble() Info   { return Info(C.CURLINFO_DOUBLE) }
func GetCurlInfoSList() Info    { return Info(C.CURLINFO_SLIST) }

func GetWriteCallbackFuncptr() unsafe.Pointer {
	return unsafe.Pointer(C.get_c_write_callback_ptr())
}

func GetReadCallbackFuncptr() unsafe.Pointer {
	return unsafe.Pointer(C.get_c_read_callback_ptr())
}

func GetHeaderCallbackFuncptr() unsafe.Pointer {
	return unsafe.Pointer(C.get_c_header_callback_ptr())
}

func GetProgressCallbackFuncptr() unsafe.Pointer {
	return unsafe.Pointer(C.get_c_progress_callback_ptr())
}

//export GoWriteFunctionTrampoline
func GoWriteFunctionTrampoline(buffer *C.char, size C.size_t, nitems C.size_t, userdata unsafe.Pointer) C.size_t {
	curlHandle := context_map.Get(uintptr(userdata))
	if curlHandle == nil || curlHandle.writeFunction == nil {
		return 0
	}
	bufLen := int(size * nitems)
	if bufLen == 0 {
		return 0
	}
	var goBuf []byte
	if bufLen > 0 {
		goBuf = (*[1 << 30]byte)(unsafe.Pointer(buffer))[:bufLen:bufLen]
	}

	if (*curlHandle.writeFunction)(goBuf, curlHandle.writeData) {
		return C.size_t(bufLen)
	}
	return C.CURL_WRITEFUNC_PAUSE
}

//export GoReadFunctionTrampoline
func GoReadFunctionTrampoline(buffer *C.char, size C.size_t, nitems C.size_t, instream unsafe.Pointer) C.size_t {
	curlHandle := context_map.Get(uintptr(instream))
	if curlHandle == nil || curlHandle.readFunction == nil {
		return C.CURL_READFUNC_ABORT
	}
	bufLen := int(size * nitems)
	if bufLen == 0 {
		return 0
	}
	var goSliceForReading []byte
	if bufLen > 0 {
		goSliceForReading = (*[1 << 30]byte)(unsafe.Pointer(buffer))[:bufLen:bufLen]
	}

	bytesRead := (*curlHandle.readFunction)(goSliceForReading, curlHandle.readData)

	if bytesRead < 0 {
		return C.size_t(uintptr(bytesRead))
	}
	if bytesRead > bufLen {
		return C.CURL_READFUNC_ABORT
	}
	return C.size_t(bytesRead)
}

//export GoHeaderFunctionTrampoline
func GoHeaderFunctionTrampoline(buffer *C.char, size C.size_t, nitems C.size_t, userdata unsafe.Pointer) C.size_t {
	curlHandle := context_map.Get(uintptr(userdata)) // userdata is the *CURL pointer
	if curlHandle == nil || curlHandle.headerFunction == nil {
		return 0
	}
	bufLen := int(size * nitems)
	if bufLen == 0 {
		return 0
	}
	var goBuf []byte
	if bufLen > 0 {
		goBuf = (*[1 << 30]byte)(unsafe.Pointer(buffer))[:bufLen:bufLen]
	}

	if (*curlHandle.headerFunction)(goBuf, curlHandle.headerData) {
		return C.size_t(bufLen)
	}
	return C.CURL_WRITEFUNC_PAUSE
}

//export GoProgressFunctionTrampoline
func GoProgressFunctionTrampoline(clientp unsafe.Pointer, dltotal C.curl_off_t, dlnow C.curl_off_t, ultotal C.curl_off_t, ulnow C.curl_off_t) C.int {
	curlHandle := context_map.Get(uintptr(clientp))
	if curlHandle == nil || curlHandle.progressFunction == nil {
		return 0
	}
	gdltotal := float64(dltotal)
	gdlnow := float64(dlnow)
	gultotal := float64(ultotal)
	gulnow := float64(ulnow)

	if (*curlHandle.progressFunction)(gdltotal, gdlnow, gultotal, gulnow, curlHandle.progressData) {
		return 0
	}
	return 1
}
