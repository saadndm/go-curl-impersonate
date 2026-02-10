package curl

/*
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"io"
	"mime"
	"os"
	"path"
	"runtime"
	"sync"
	"unsafe"
)

func stringToCUCharPtr(s string) (ptr uintptr, keepAlive []byte) {
	b := []byte(s)
	b = append(b, 0)

	if len(b) == 0 {
		return 0, b
	}
	return uintptr(unsafe.Pointer(&b[0])), b
}

func newCurlError(errno CurlCode) error {
	if errno == E_OK {
		return nil
	}
	return CurlError(errno)
}

// curl_easy interface
type CURL struct {
	handle                                        unsafe.Pointer
	headerFunction                                *func([]byte, any) bool
	writeFunction                                 *func([]byte, any) bool
	readFunction                                  *func([]byte, any) int
	progressFunction                              *func(float64, float64, float64, float64, any) bool
	headerData, writeData, readData, progressData any
	mallocAllocs                                  []unsafe.Pointer
}

type contextMap struct {
	items map[uintptr]*CURL
	sync.RWMutex
}

func (c *contextMap) Set(k uintptr, v *CURL) {
	c.Lock()
	defer c.Unlock()

	c.items[k] = v
}

func (c *contextMap) Get(k uintptr) *CURL {
	c.RLock()
	v := c.items[k]
	c.RUnlock()
	return v
}

func (c *contextMap) Delete(k uintptr) {
	c.Lock()
	defer c.Unlock()

	delete(c.items, k)
}

var context_map = &contextMap{
	items: make(map[uintptr]*CURL),
}

// curl_easy_init - Start a libcurl easy session
func EasyInit() *CURL {
	certPath, err := getEmbeddedCACertPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not prepare embedded CA certificate: %v. SSL connections may fail.\n", err)
	}

	p := CurlEasyInit()
	if p == nil {
		if runtime.GOOS == "windows" && CheckLoad() != nil {
			panic(fmt.Errorf("curl: EasyInit failed because DLL could not be loaded: %v", CheckLoad()))
		}
		panic("curl: EasyInit returned a nil handle")
	}
	c := &CURL{handle: p, mallocAllocs: make([]unsafe.Pointer, 0)}
	context_map.Set(uintptr(p), c)

	if certPath != "" {
		err = c.Setopt(OPT_CAINFO, certPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Failed to set CURLOPT_CAINFO to '%s': %v. SSL connections may fail.\n", certPath, err)
		}
	} else {
		fmt.Fprintln(os.Stderr, "Error: Embedded CA certificate path is empty. CURLOPT_CAINFO not set.")
	}

	return c
}

// curl_easy_duphandle - Clone a libcurl session handle
func (curl *CURL) Duphandle() *CURL {
	if curl.handle == nil {
		panic("curl: Duphandle called on a nil handle")
	}
	p := CurlEasyDuph(curl.handle)
	if p == nil {
		panic("curl: Duphandle returned a nil handle")
	}
	c := &CURL{handle: p, mallocAllocs: make([]unsafe.Pointer, 0)}
	context_map.Set(uintptr(p), c)
	return c
}

// curl_easy_cleanup - End a libcurl easy session
func (curl *CURL) Cleanup() {
	p := curl.handle
	if p != nil {
		CurlEasyCleanup(p)
		curl.MallocFreeAfter(0)
		context_map.Delete(uintptr(p))
		curl.handle = nil
	}
}

// curl_easy_setopt - set options for a curl easy handle
// WARNING: a function pointer is &fun, but function addr is reflect.ValueOf(fun).Pointer()
func (curl *CURL) Setopt(opt EasyOpt, param any) error {
	p := curl.handle
	if p == nil {
		return fmt.Errorf("curl: easy handle is nil")
	}

	switch opt {
	case OPT_WRITEDATA:
		w, ok := param.(io.Writer)
		if !ok {
			return fmt.Errorf("curl: expected io.Writer for WRITEDATA, got %T", param)
		}
		curl.writeData = w
		return newCurlError(CurlEasySetoptPointer(p, int(opt), unsafe.Pointer(p)))
	case OPT_WRITEFUNCTION:
		if param == nil {
			curl.writeFunction = nil
			return newCurlError(CurlEasySetoptFunction(p, int(opt), unsafe.Pointer((*struct{})(nil))))
		}
		f, ok := param.(func([]byte, any) bool)
		if !ok {
			return fmt.Errorf(
				"curl: expected func([]byte,any)bool for WRITEFUNCTION, got %T",
				param,
			)
		}
		curl.writeFunction = &f
		if errCode := CurlEasySetoptPointer(p, int(OPT_WRITEDATA), unsafe.Pointer(p)); errCode != 0 {
			return newCurlError(errCode)
		}
		return newCurlError(
			CurlEasySetoptFunction(p, int(opt), GetWriteCallbackFuncptr()),
		)

	case OPT_READFUNCTION:
		if param == nil {
			curl.readFunction = nil
			curl.readData = nil
			return newCurlError(CurlEasySetoptFunction(p, int(opt), unsafe.Pointer((*struct{})(nil))))
		}
		f, ok := param.(func([]byte, any) int)
		if !ok {
			return fmt.Errorf("curl: expected func([]byte, any) int for READFUNCTION, got %T", param)
		}
		curl.readFunction = &f

		if errCode := CurlEasySetoptPointer(p, int(OPT_READDATA), unsafe.Pointer(p)); errCode != 0 {
			return newCurlError(errCode)
		}
		return newCurlError(CurlEasySetoptFunction(p, int(opt), GetReadCallbackFuncptr()))

	case OPT_HEADERFUNCTION:
		if param == nil {
			curl.headerFunction = nil
			curl.headerData = nil
			return newCurlError(CurlEasySetoptFunction(p, int(opt), unsafe.Pointer((*struct{})(nil))))
		}
		f, ok := param.(func([]byte, any) bool)
		if !ok {
			return fmt.Errorf("curl: expected func([]byte, any) bool for HEADERFUNCTION, got %T", param)
		}
		curl.headerFunction = &f

		if errCode := CurlEasySetoptPointer(p, int(OPT_HEADERDATA), unsafe.Pointer(p)); errCode != 0 {
			return newCurlError(errCode)
		}
		return newCurlError(CurlEasySetoptFunction(p, int(opt), GetHeaderCallbackFuncptr()))

	case OPT_HEADERDATA:
		curl.headerData = param
		// We do NOT set the C-level WRITEDATA because HEADERFUNCTION sets it to 'p' (the context)
		// so that the Go callback wrapper can find the context and then access curl.headerData.
		return nil

	case OPT_XFERINFOFUNCTION:
		if param == nil {
			curl.progressFunction = nil
			curl.progressData = nil
			return newCurlError(CurlEasySetoptFunction(p, int(opt), unsafe.Pointer((*struct{})(nil))))
		}
		fun, ok := param.(func(float64, float64, float64, float64, any) bool)
		if !ok {
			return fmt.Errorf("curl: expected func(float64, float64, float64, float64, any) bool for XFERINFOFUNCTION, got %T", param)
		}
		curl.progressFunction = &fun
		if errCode := CurlEasySetoptPointer(p, int(OPT_XFERINFODATA), unsafe.Pointer(p)); errCode != 0 {
			return newCurlError(errCode)
		}
		if errCode := CurlEasySetoptLong(p, int(OPT_NOPROGRESS), 0); errCode != 0 {
			return newCurlError(errCode)
		}
		return newCurlError(CurlEasySetoptFunction(p, int(opt), GetProgressCallbackFuncptr()))
	}

	if param == nil {
		return newCurlError(CurlEasySetoptPointer(p, int(opt), nil))
	}

	switch v := param.(type) {
	case int:
		return newCurlError(CurlEasySetoptLong(p, int(opt), int64(v)))
	case int32:
		return newCurlError(CurlEasySetoptLong(p, int(opt), int64(v)))
	case int64:
		if isOffTOption(opt) {
			return newCurlError(CurlEasySetoptOffT(p, int(opt), v))
		}
		return newCurlError(CurlEasySetoptLong(p, int(opt), v))
	case bool:
		var val int64 = 0
		if v {
			val = 1
		}
		return newCurlError(CurlEasySetoptLong(p, int(opt), val))
	case string:
		var cStr unsafe.Pointer
		var keepAliveStr []byte

		if runtime.GOOS == "windows" {
			var ptr uintptr
			ptr, keepAliveStr = stringToCUCharPtr(v)
			_ = keepAliveStr
			cStr = unsafe.Pointer(ptr)
		} else {
			cStr = unsafe.Pointer(C.CString(v))
			curl.mallocAddPtr(cStr)
		}
		return newCurlError(CurlEasySetoptString(p, int(opt), cStr))

	case []byte:
		var dataPtr unsafe.Pointer
		if len(v) > 0 {
			dataPtr = C.CBytes(v)
			curl.mallocAddPtr(dataPtr)
		}
		errCode := CurlEasySetoptPointer(p, int(opt), dataPtr)
		if errCode != 0 {
			return newCurlError(errCode)
		}
		if opt == OPT_POSTFIELDS {
			if len(v) > 2147483647 || len(v) < 0 {
				return newCurlError(CurlEasySetoptOffT(p, int(OPT_POSTFIELDSIZE_LARGE), int64(len(v))))
			} else {
				return newCurlError(CurlEasySetoptLong(p, int(OPT_POSTFIELDSIZE), int64(len(v))))
			}
		}
		return nil

	case []string:
		var slistHandle CurlSlist = nil
		var cgoStringsToFreeOnFailure []unsafe.Pointer

		for _, s := range v {
			var cStr unsafe.Pointer
			if runtime.GOOS == "windows" {
				ptr, keepAlive := stringToCUCharPtr(s)
				_ = keepAlive
				cStr = unsafe.Pointer(ptr)
			} else {
				cStr = unsafe.Pointer(C.CString(s))
				cgoStringsToFreeOnFailure = append(cgoStringsToFreeOnFailure, cStr)
			}

			appendedSlist := CurlSlistAppend(slistHandle, cStr)
			if appendedSlist == nil {
				if runtime.GOOS != "windows" {
					for _, tempStrPtr := range cgoStringsToFreeOnFailure {
						C.free(tempStrPtr)
					}
				}
				if slistHandle != nil {
					CurlSlistFreeAll(slistHandle)
				}
				return fmt.Errorf("curl: CurlSlistAppend failed for string: %s", s)
			}
			slistHandle = appendedSlist
		}

		errCode := CurlEasySetoptSlist(p, int(opt), slistHandle)
		if errCode != E_OK {
			if runtime.GOOS != "windows" {
				for _, tempStrPtr := range cgoStringsToFreeOnFailure {
					C.free(tempStrPtr)
				}
			}
			CurlSlistFreeAll(slistHandle)
			return newCurlError(errCode)
		}

		if runtime.GOOS != "windows" {
			for _, tempStrPtr := range cgoStringsToFreeOnFailure {
				curl.mallocAddPtr(tempStrPtr)
			}
		}
		return nil

	case *Form:
		if v == nil || v.head == nil {
			return newCurlError(CurlEasySetoptPointer(p, int(opt), nil))
		}
		return newCurlError(CurlEasySetoptPointer(p, int(opt), unsafe.Pointer(v.head)))

	case unsafe.Pointer:
		return newCurlError(CurlEasySetoptPointer(p, int(opt), v))

	default:
		return fmt.Errorf("curl: unsupported Setopt param type: %T for option %d", param, opt)
	}
}

func isOffTOption(opt EasyOpt) bool {
	switch opt {
	case OPT_INFILESIZE_LARGE, OPT_RESUME_FROM_LARGE, OPT_MAXFILESIZE_LARGE, OPT_POSTFIELDSIZE_LARGE, OPT_MAX_SEND_SPEED_LARGE, OPT_MAX_RECV_SPEED_LARGE, OPT_TIMEVALUE_LARGE:
		return true
	}
	return false
}

// curl_easy_send - sends raw data over an "easy" connection
func (curl *CURL) Send(buffer []byte) (int, error) {
	p := curl.handle
	if p == nil {
		return 0, fmt.Errorf("curl: easy handle is nil")
	}
	buflen := len(buffer)
	var n CSizeT
	var bufPtr unsafe.Pointer
	if buflen > 0 {
		bufPtr = unsafe.Pointer(&buffer[0])
	}

	var nVal uintptr
	errCode := CurlEasySend(p, bufPtr, buflen, unsafe.Pointer(&nVal))
	n = CSizeT(nVal)

	return int(n), newCurlError(errCode)
}

type CSizeT uintptr

// curl_easy_recv - receives raw data on an "easy" connection
func (curl *CURL) Recv(buffer []byte) (int, error) {
	p := curl.handle
	if p == nil {
		return 0, fmt.Errorf("curl: easy handle is nil")
	}
	buflen := len(buffer)
	if buflen == 0 {
		return 0, nil
	}

	var nVal uintptr
	var bufPtr unsafe.Pointer
	if buflen > 0 {
		bufPtr = unsafe.Pointer(&buffer[0])
	}

	ret := CurlEasyRecv(p, bufPtr, buflen, unsafe.Pointer(&nVal))
	bytesRead := int(nVal)

	return bytesRead, newCurlError(ret)
}

// curl_easy_perform - Perform a file transfer
func (curl *CURL) Perform() error {
	p := curl.handle
	if p == nil {
		return fmt.Errorf("curl: easy handle is nil")
	}
	err := newCurlError(CurlEasyPerform(p))

	runtime.KeepAlive(curl.headerData)
	runtime.KeepAlive(curl.writeData)
	runtime.KeepAlive(curl.readData)
	runtime.KeepAlive(curl.progressData)
	return err
}

// curl_easy_pause - pause and unpause a connection
func (curl *CURL) Pause(bitmask int) error {
	p := curl.handle
	if p == nil {
		return fmt.Errorf("curl: easy handle is nil")
	}
	return newCurlError(CurlEasyPause(p, bitmask))
}

// curl_easy_reset - reset all options of a libcurl session handle
func (curl *CURL) Reset() {
	p := curl.handle
	if p != nil {
		CurlEasyReset(p)
		curl.MallocFreeAfter(0)
		curl.headerFunction = nil
		curl.writeFunction = nil
		curl.readFunction = nil
		curl.progressFunction = nil
		curl.headerData = nil
		curl.writeData = nil
		curl.readData = nil
		curl.progressData = nil
	}
}

// curl_easy_escape - URL encodes the given string
func (curl *CURL) Escape(url string) string {
	p := curl.handle
	if p == nil {
		return ""
	}
	var cURL unsafe.Pointer
	if runtime.GOOS == "windows" {
		ptr, _ := stringToCUCharPtr(url)
		cURL = unsafe.Pointer(ptr)
	} else {
		cURL = unsafe.Pointer(C.CString(url))
		defer C.free(cURL)
	}

	escapedCURL := CurlEasyEscape(p, cURL, len(url))
	if escapedCURL == nil {
		return ""
	}
	defer CurlFree(escapedCURL)
	return goStringSys(uintptr(escapedCURL))
}

// curl_easy_unescape - URL decodes the given string
func (curl *CURL) Unescape(url string) string {
	p := curl.handle
	if p == nil {
		return ""
	}
	var cURL unsafe.Pointer
	if runtime.GOOS == "windows" {
		ptr, _ := stringToCUCharPtr(url)
		cURL = unsafe.Pointer(ptr)
	} else {
		cURL = unsafe.Pointer(C.CString(url))
		defer C.free(cURL)
	}

	var outLength int32
	unescapedCURL := CurlEasyUnescape(p, cURL, len(url), unsafe.Pointer(&outLength))
	if unescapedCURL == nil {
		return ""
	}
	defer CurlFree(unescapedCURL)
	return string(unsafe.Slice((*byte)(unescapedCURL), int(outLength)))
}

func goStringSys(ccharPtr uintptr) string {
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
			return ""
		}
	}
	if length == 0 {
		return ""
	}
	return string(unsafe.Slice((*byte)(ptr), length))
}

func (curl *CURL) Getinfo(infoConstant Info) (any, error) {
	p := curl.handle
	if p == nil {
		return nil, fmt.Errorf("curl: easy handle is nil")
	}

	typeMask := GetCurlInfoTypeMask()
	infoType := infoConstant & typeMask

	switch infoType {
	case GetCurlInfoString():
		var cStrPtr uintptr
		errCode := CurlEasyGetinfoString(p, infoConstant, unsafe.Pointer(&cStrPtr))
		if errCode != E_OK {
			return nil, newCurlError(errCode)
		}
		if cStrPtr == 0 {
			return "", nil
		}
		return goStringSys(cStrPtr), nil
	case GetCurlInfoLong():
		var val int32
		errCode := CurlEasyGetinfoLong(p, infoConstant, unsafe.Pointer(&val))
		if errCode != E_OK {
			return nil, newCurlError(errCode)
		}
		return int64(val), nil
	case GetCurlInfoDouble():
		var val float64
		errCode := CurlEasyGetinfoDouble(p, infoConstant, unsafe.Pointer(&val))
		if errCode != E_OK {
			return nil, newCurlError(errCode)
		}
		return val, nil
	case GetCurlInfoSList():
		var slistPtr CurlSlist
		errCode := CurlEasyGetinfoSlist(p, infoConstant, unsafe.Pointer(&slistPtr))
		if errCode != E_OK {
			return nil, newCurlError(errCode)
		}
		if slistPtr == nil {
			return []string{}, nil
		}
		var goSlice []string
		current := slistPtr
		for current != nil {
			dataPtr := *(*uintptr)(unsafe.Pointer(current))
			if dataPtr != 0 {
				goSlice = append(goSlice, goStringSys(dataPtr))
			}
			current = CurlSlist(*(*uintptr)(unsafe.Pointer(uintptr(current) + unsafe.Sizeof(uintptr(0)))))
		}
		return goSlice, nil
	default:
		return nil, fmt.Errorf("curl: Getinfo unsupported info type for constant: %d (type: %d)", infoConstant, infoType)
	}
}

func PrintCurlVersionInfo(infoPtr unsafe.Pointer) {
	if infoPtr == nil {
		fmt.Println("CurlVersionInfoData is nil")
		return
	}
	data := (*CurlVersionInfoDataLayout)(infoPtr)

	fmt.Printf("Age: %d\n", data.Age)
	fmt.Printf("Version: %s\n", goStringSys(data.Version))
	fmt.Printf("VersionNum: 0x%x\n", data.VersionNum)
	fmt.Printf("Host: %s\n", goStringSys(data.Host))
	fmt.Printf("Features: 0x%x\n", data.Features)
	fmt.Printf("SSL Version: %s\n", goStringSys(data.SslVersion))
	fmt.Printf("SSL Version Num: %d\n", data.SslVersionNum)
	fmt.Printf("Libz Version: %s\n", goStringSys(data.LibzVersion))

	if data.Protocols != 0 {
		fmt.Println("Protocols:")
		for i := 0; ; i++ {
			protocolPtr := *(*uintptr)(unsafe.Pointer(data.Protocols + uintptr(i)*unsafe.Sizeof(uintptr(0))))
			if protocolPtr == 0 {
				break
			}
			fmt.Printf("  - %s\n", goStringSys(protocolPtr))
		}
	}
	fmt.Printf("Ares: %s\n", goStringSys(data.Ares))
	fmt.Printf("Ares_num: %d\n", data.AresNum)
	fmt.Printf("Libidn: %s\n", goStringSys(data.Libidn))
}

func (curl *CURL) Impersonate(target string, defaultHeaders bool) error {
	p := curl.handle
	if p == nil {
		return fmt.Errorf("curl: easy handle is nil")
	}

	var cTarget unsafe.Pointer
	var keepAliveTarget []byte

	if runtime.GOOS == "windows" {
		ptr, ka := stringToCUCharPtr(target)
		cTarget = unsafe.Pointer(ptr)
		keepAliveTarget = ka
		_ = keepAliveTarget
	} else {
		cTarget = unsafe.Pointer(C.CString(target))
		defer C.free(cTarget)
	}

	var cDefaultHeaders int = 0
	if defaultHeaders {
		cDefaultHeaders = 1
	}
	return newCurlError(CurlEasyImpersonate(p, cTarget, cDefaultHeaders))
}

func (curl *CURL) GetHandle() unsafe.Pointer {
	return curl.handle
}

func (curl *CURL) MallocGetPos() int {
	return len(curl.mallocAllocs)
}

func (curl *CURL) MallocFreeAfter(from int) {
	if from < 0 || from > len(curl.mallocAllocs) {
		return
	}
	for i := from; i < len(curl.mallocAllocs); i++ {
		if curl.mallocAllocs[i] != nil {
			if runtime.GOOS != "windows" {
				C.free(curl.mallocAllocs[i])
			}
			curl.mallocAllocs[i] = nil
		}
	}
	curl.mallocAllocs = curl.mallocAllocs[:from]
}

func (curl *CURL) mallocAddPtr(ptr unsafe.Pointer) {
	curl.mallocAllocs = append(curl.mallocAllocs, ptr)
}

// A multipart/formdata HTTP POST form
type Form struct {
	head       CurlHttpFormPost
	last       CurlHttpFormPost
	formAllocs []unsafe.Pointer
}

func NewForm() *Form {
	return &Form{formAllocs: make([]unsafe.Pointer, 0)}
}

func (form *Form) addFormAlloc(p unsafe.Pointer) {
	form.formAllocs = append(form.formAllocs, p)
}

func (form *Form) Add(name string, content any) error {
	var cName, cContent unsafe.Pointer
	var length int

	if runtime.GOOS == "windows" {
		ptr, _ := stringToCUCharPtr(name)
		cName = unsafe.Pointer(ptr)
	} else {
		cName = unsafe.Pointer(C.CString(name))
		form.addFormAlloc(cName)
	}

	switch t := content.(type) {
	case string:
		length = len(t)
		if runtime.GOOS == "windows" {
			ptr, _ := stringToCUCharPtr(t)
			cContent = unsafe.Pointer(ptr)
		} else {
			cContent = unsafe.Pointer(C.CString(t))
			form.addFormAlloc(cContent)
		}
	case []byte:
		length = len(t)
		if len(t) > 0 {
			cContent = unsafe.Pointer(&t[0])
		}
	default:
		return fmt.Errorf("curl: form add unsupported content type %T", content)
	}

	retCode := CurlFormaddNameContentLength(unsafe.Pointer(&form.head), unsafe.Pointer(&form.last), cName, cContent, length)
	if retCode != GetCurlFormaddOk() {
		return fmt.Errorf("curl: formadd failed with code %d", retCode)
	}
	return nil
}

func (form *Form) AddWithType(name string, content any, contentType string) error {
	var cName, cContent, cContentType unsafe.Pointer
	var length int

	if runtime.GOOS == "windows" {
		ptrName, _ := stringToCUCharPtr(name)
		cName = unsafe.Pointer(ptrName)
		ptrCType, _ := stringToCUCharPtr(contentType)
		cContentType = unsafe.Pointer(ptrCType)
	} else {
		cName = unsafe.Pointer(C.CString(name))
		form.addFormAlloc(cName)
		cContentType = unsafe.Pointer(C.CString(contentType))
		form.addFormAlloc(cContentType)
	}

	switch t := content.(type) {
	case string:
		length = len(t)
		if runtime.GOOS == "windows" {
			ptrContent, _ := stringToCUCharPtr(t)
			cContent = unsafe.Pointer(ptrContent)
		} else {
			cContent = unsafe.Pointer(C.CString(t))
			form.addFormAlloc(cContent)
		}
	case []byte:
		length = len(t)
		if len(t) > 0 {
			cContent = unsafe.Pointer(&t[0])
		}
	default:
		return fmt.Errorf("curl: form addwithtype unsupported content type %T", content)
	}

	retCode := CurlFormaddNameContentLengthType(unsafe.Pointer(&form.head), unsafe.Pointer(&form.last), cName, cContent, length, cContentType)
	if retCode != GetCurlFormaddOk() {
		return fmt.Errorf("curl: formaddwithtype failed with code %d", retCode)
	}
	return nil
}

func (form *Form) AddFile(name, filename string) error {
	var cName, cFilename, cContentType unsafe.Pointer

	guessedType := guessType(filename)

	if runtime.GOOS == "windows" {
		ptrName, _ := stringToCUCharPtr(name)
		cName = unsafe.Pointer(ptrName)
		ptrFName, _ := stringToCUCharPtr(filename)
		cFilename = unsafe.Pointer(ptrFName)
		if guessedType != "" {
			ptrCType, _ := stringToCUCharPtr(guessedType)
			cContentType = unsafe.Pointer(ptrCType)
		}
	} else {
		cName = unsafe.Pointer(C.CString(name))
		form.addFormAlloc(cName)
		cFilename = unsafe.Pointer(C.CString(filename))
		form.addFormAlloc(cFilename)
		if guessedType != "" {
			cContentType = unsafe.Pointer(C.CString(guessedType))
			form.addFormAlloc(cContentType)
		}
	}

	retCode := CurlFormaddNameFileType(unsafe.Pointer(&form.head), unsafe.Pointer(&form.last), cName, cFilename, cContentType)
	if retCode != GetCurlFormaddOk() {
		return fmt.Errorf("curl: formaddfile failed with code %d", retCode)
	}
	return nil
}

func (form *Form) Free() {
	if form.head != nil {
		CurlFormFree(form.head)
		form.head = nil
		form.last = nil
	}
	if runtime.GOOS != "windows" {
		for _, ptr := range form.formAllocs {
			if ptr != nil {
				C.free(ptr)
			}
		}
	}
	form.formAllocs = nil
}

func guessType(filename string) string {
	ext := path.Ext(filename)
	if ext == "" {
		return "application/octet-stream"
	}
	mimeType := mime.TypeByExtension(ext)
	if mimeType == "" {
		return "application/octet-stream"
	}
	return mimeType
}
