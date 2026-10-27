//go:build windows

// Package wmi provides a dependency-free client for the WMI repository.
//
// 它通过原始的 COM 接口（IWbemLocator、IWbemServices 和 IWbemClassObject）
// 和 WMI 的提供程序说话，只用 Windows 自带的 ole32.dll 和 oleaut32.dll。
// 没有辅助进程，没有第三方的 DLL，也没有 .NET 运行时。
//
// 这个包是**补充那一侧的水管**：硬件层主要靠原厂程序（原厂服务、Quick
// Access 服务）来问机器，只有原厂程序给不了的那几项——风扇加速、风扇表、
// 超频能力、背光原始缓冲区、锁 Windows 键——才直接走到这里。所以它是
// 补充，不是主路。
//
// 下面的虚表槽位号是在这台机器上量出来的，而不是照抄公开的头文件，因为
// WMI 的 COM 接口在不同版本的 Windows 上布局不一样。每一个常量旁边都写着
// 得出它的依据。
package wmi

import (
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unsafe"

	"viper/internal/com"
)

// Vtable slot numbers of the WMI COM interfaces.
//
// 这些值是对着活的接口探出来的：当时临时写了几个探针，一个槽位一个槽位地
// 试，探针用完就删了，结论与依据留在下面。
const (
	// slotConnectServer is IWbemLocator::ConnectServer.
	//
	// 在这个版本上它要八个参数：命名空间在前，IWbemServices 的输出指针
	// 在最后。公开头文件里常见的九参数写法会被拒绝，报
	// WBEM_E_NOT_SUPPORTED。
	slotConnectServer = 3

	// slotExecQuery is IWbemServices::ExecQuery.
	//
	// 它要六个参数，其中有一个 ppCallResult 的位置，五个参数的写法把它
	// 漏掉了。漏掉它，提供程序内部就会出错。
	slotExecQuery = 20

	// slotGetObject is IWbemServices::GetObject.
	slotGetObject = 6

	// slotExecMethod is IWbemServices::ExecMethod.
	//
	// 七个参数。末尾的 ppContext 和 pFlags 必须传 NULL：给它们可写的
	// 指针，提供程序会往里写，然后崩溃。
	slotExecMethod = 24

	// slotGet is IWbemClassObject::Get.
	slotGet = 4

	// slotPut is IWbemClassObject::Put.
	slotPut = 5

	// slotSpawnInstance is IWbemClassObject::SpawnInstance.
	slotSpawnInstance = 15

	// slotGetMethod is IWbemClassObject::GetMethod.
	//
	// WMI 接口公开的头文件不一定是程序运行时那个版本的形状，所以槽位是
	// 在跑 Viper 的这台机器上量出来的。在这台机器上，标准命名空间里的
	// 方法会被拒绝，报 WBEM_E_NOT_FOUND，而厂商硬件的方法用这个槽位是
	// 好的。
	slotGetMethod = 19
)

// errClosed is returned by every call that arrives after the client was
// closed.
var errClosed = errors.New("wmi: 客户端已经关闭")

// Client is a connection to one WMI namespace.
//
// 每个客户端都有自己的线程。COM 的公寓是按线程的，所以连接、调用和对象
// 的释放都得在这一个线程上。Go 的 goroutine 会在线程之间搬家，所以客户端
// 把活儿交给自己的线程，然后等它的回话。
//
// 每个客户端的公寓都是它自己的，所以关掉一个客户端不会打扰还开着的那些。
type Client struct {
	// mu 看守着客户端的状态。写锁只在关闭客户端的时候拿，读锁在调用
	// 运行的时候拿，所以一次调用和关闭客户端不会互相超车。
	mu     sync.RWMutex
	closed bool

	// services 是到命名空间的连接。客户端的线程在连接打开时写它，之后
	// 由同一个线程读。
	services com.COMObject

	// comErr 是公寓没能启动的原因。客户端的线程在第一个活儿之前写它。
	comErr error

	// comOwned 说明这个线程上的公寓是不是客户端开出来的。只有客户端自己
	// 开出来的公寓才能由客户端关掉。
	comOwned bool

	// jobs 把活儿交给客户端的线程。
	jobs chan func()

	// done 在线程干完最后一个活儿之后被关掉。
	done chan struct{}
}

// Connect opens a connection to the given WMI namespace.
//
// COM 的公寓和 COM 的安全设置都在客户端的线程上建立，因为安全设置必须在
// 第一个 WMI 对象被创建之前就绪。
func Connect(namespace string) (*Client, error) {
	client := &Client{
		jobs: make(chan func()),
		done: make(chan struct{}),
	}
	go client.serve()

	err := client.submit(func() error {
		if client.comErr != nil {
			return client.comErr
		}
		services, err := connectNamespace(namespace)
		if err != nil {
			return err
		}
		client.services = services
		return nil
	})
	if err != nil {
		client.Close()
		return nil, err
	}
	return client, nil
}

// Close releases the connection.
//
// COM 的对象在创建它们的线程上被释放，那个线程的公寓随后被关掉。这个
// 调用可以多做几次，第二次什么也不做。
func (c *Client) Close() {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return
	}
	c.closed = true
	// 通道是在客户端的锁底下关掉的，所以不会有活儿还在往线程那里去。
	close(c.jobs)
	c.mu.Unlock()

	<-c.done
}

// serve runs the work of the client on one thread.
//
// 线程被锁在这个 goroutine 上，所以 COM 的公寓留在它被启动的地方。最后
// 一个活儿干完之后，COM 的对象被释放，公寓被关掉，两件事都在这个线程上。
func (c *Client) serve() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	defer close(c.done)

	c.comOwned, c.comErr = com.InitCOMOwned()
	for job := range c.jobs {
		job()
	}

	if c.services != 0 {
		c.services.Release()
	}
	if c.comOwned {
		// The apartment is closed on the thread that started it, which
		// is the only thread that may close it.
		com.UninitCOM()
	}
}

// submit hands the work over to the thread of the client and waits for the
// answer.
//
// 活儿交出去的时候客户端的锁还攥在手里，所以客户端不会在活儿还在路上的时候
// 被关掉。等回话的时候锁已经放开：关闭客户端本来就要等线程，一个正在跑的
// 调用不该占着客户端的锁。
func (c *Client) submit(job func() error) error {
	c.mu.RLock()
	if c.closed {
		c.mu.RUnlock()
		return errClosed
	}
	result := make(chan error, 1)
	c.jobs <- func() {
		result <- job()
	}
	c.mu.RUnlock()
	return <-result
}

// Instances returns the object paths of every instance of a class.
//
// 列举 Acer 的 ACPI 实例需要管理员权限。
func (c *Client) Instances(class string) ([]string, error) {
	var paths []string
	err := c.submit(func() error {
		var err error
		paths, err = c.instances(class)
		return err
	})
	if err != nil {
		return nil, err
	}
	return paths, nil
}

// instances lists the instances of a class. It runs on the thread of the
// client.
func (c *Client) instances(class string) ([]string, error) {
	lang, err := com.NewBSTR("WQL")
	if err != nil {
		return nil, err
	}
	defer lang.Free()
	query, err := com.NewBSTR("SELECT * FROM " + class)
	if err != nil {
		return nil, err
	}
	defer query.Free()

	var enum unsafe.Pointer
	r, err := c.services.Call(slotExecQuery,
		uintptr(lang),
		uintptr(query),
		0x30, // WBEM_FLAG_RETURN_IMMEDIATELY | WBEM_FLAG_FORWARD_ONLY
		0,    // ppCallResult
		uintptr(unsafe.Pointer(&enum)),
		0, // pFlags
	)
	if err != nil {
		return nil, err
	}
	if failed(r) {
		return nil, fmt.Errorf("wmi: ExecQuery(%s): %w", class, hresult(r))
	}
	if enum == nil {
		return nil, fmt.Errorf("wmi: ExecQuery(%s): 没有拿到枚举器", class)
	}
	enumerator := com.NewCOMObject(enum)
	defer enumerator.Release()
	com.SetProxyBlanket(enumerator)

	var paths []string
	timeout := int64(-1)
	for i := 0; i < 256; i++ {
		var obj unsafe.Pointer
		var returned uint32
		r, _ := enumerator.Call(4,
			uintptr(uint64(timeout)),
			1,
			uintptr(unsafe.Pointer(&obj)),
			uintptr(unsafe.Pointer(&returned)),
		)
		if obj == nil || returned == 0 || failed(r) {
			break
		}
		instance := com.NewCOMObject(obj)
		com.SetProxyBlanket(instance)

		path, err := readString(instance, "__RELPATH")
		if err == nil && path != "" {
			paths = append(paths, path)
		}
		instance.Release()
	}
	return paths, nil
}

// Call invokes a WMI method on an instance and returns its output
// parameters.
//
// args 里是输入参数，键是它们在 MOF 里的名字，比如 "gmInput" 或者
// "uiInput"。值的类型按 Acer 的提供程序接受的顺序来试，因为这些提供程序
// 把参数声明成 UInt32，实际却只存有符号的整数或者字符串。
func (c *Client) Call(class, method, instancePath string, args map[string]Value) (map[string]Value, error) {
	var outputs map[string]Value
	err := c.submit(func() error {
		var err error
		outputs, err = c.call(class, method, instancePath, args)
		return err
	})
	if err != nil {
		return nil, err
	}
	return outputs, nil
}

// call invokes a method. It runs on the thread of the client.
func (c *Client) call(class, method, instancePath string, args map[string]Value) (map[string]Value, error) {
	classObj, err := c.getObject(class)
	if err != nil {
		return nil, err
	}
	defer classObj.Release()

	inDef, outDef, err := getMethod(classObj, method)
	if err != nil {
		return nil, err
	}
	if outDef != 0 {
		defer outDef.Release()
	}

	// The in-parameters object is a spawned instance of the method's
	// in-parameters class. Methods without inputs have no such class and
	// are invoked with a null in-parameters object.
	var inParams com.COMObject
	if inDef != 0 {
		inParams, err = spawnInstance(inDef)
		if err != nil {
			return nil, err
		}
		defer inParams.Release()
	}

	for name, value := range args {
		if err := putValue(inParams, name, value); err != nil {
			return nil, err
		}
	}

	outParams, err := c.execMethod(instancePath, method, inParams)
	if err != nil {
		return nil, err
	}
	defer outParams.Release()

	return readOutputs(outParams, outDef)
}

// ReadProperty reads a single numeric property of an object path.
//
// 它是给标准的 CIM 命名空间用的，那边的类带着 CurrentClockSpeed 这样的
// 普通属性。
func (c *Client) ReadProperty(objectPath, name string) (uint64, error) {
	var value uint64
	err := c.submit(func() error {
		var err error
		value, err = c.readProperty(objectPath, name)
		return err
	})
	if err != nil {
		return 0, err
	}
	return value, nil
}

// readProperty reads a numeric property. It runs on the thread of the
// client.
func (c *Client) readProperty(objectPath, name string) (uint64, error) {
	target, err := c.getObject(objectPath)
	if err != nil {
		return 0, err
	}
	defer target.Release()

	value, err := readValue(target, name)
	if err != nil {
		return 0, err
	}
	if value.Kind == KindString {
		return parseNumeric(value.Str)
	}
	return value.UInt64(), nil
}

// ReadBytes reads a property that carries an array of bytes.
//
// 厂商的提供程序把键盘背光的缓冲区交成一个字节的数组，所以缓冲区照原样
// 读回来。
func (c *Client) ReadBytes(objectPath, name string) ([]byte, error) {
	var data []byte
	err := c.submit(func() error {
		var err error
		data, err = c.readBytes(objectPath, name)
		return err
	})
	if err != nil {
		return nil, err
	}
	return data, nil
}

// readBytes reads an array of bytes. It runs on the thread of the client.
func (c *Client) readBytes(objectPath, name string) ([]byte, error) {
	target, err := c.getObject(objectPath)
	if err != nil {
		return nil, err
	}
	defer target.Release()

	value, err := readValue(target, name)
	if err != nil {
		return nil, err
	}
	if value.Kind != KindBytes {
		return nil, fmt.Errorf("wmi: %s 不是 %s 上的字节数组", name, objectPath)
	}
	return value.Bytes, nil
}

// parseNumeric parses a decimal or hexadecimal number from a string.
func parseNumeric(text string) (uint64, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0, nil
	}
	base := 10
	if strings.HasPrefix(text, "0x") || strings.HasPrefix(text, "0X") {
		text = text[2:]
		base = 16
	}
	return strconv.ParseUint(text, base, 64)
}

// connectNamespace opens a connection to a namespace of WMI.
//
// 客户端的线程上公寓已经在跑了，所以只剩 COM 的安全设置和定位器要办。
func connectNamespace(namespace string) (com.COMObject, error) {
	if err := com.InitCOMSecurity(); err != nil {
		return 0, err
	}

	locator, err := com.CoCreateInstance(com.CLSIDWbemLocator, com.IIDWbemLocator, com.CLSCTXInprocServer)
	if err != nil {
		return 0, err
	}
	defer locator.Release()

	ns, err := com.NewBSTR(namespace)
	if err != nil {
		return 0, err
	}
	defer ns.Free()

	var services unsafe.Pointer
	r, err := locator.Call(slotConnectServer,
		uintptr(ns), // strNamespacePath
		0,           // strServer
		0,           // strLocale
		0,           // strUser
		0,           // strPassword
		0,           // lSecurityFlags
		0,           // strAuthority
		uintptr(unsafe.Pointer(&services)),
	)
	if err != nil {
		return 0, err
	}
	if failed(r) {
		return 0, fmt.Errorf("wmi: ConnectServer(%s): %w", namespace, hresult(r))
	}
	if services == nil {
		return 0, fmt.Errorf("wmi: ConnectServer(%s): 没有拿到 services 指针", namespace)
	}
	result := com.NewCOMObject(services)
	com.SetProxyBlanket(result)
	return result, nil
}

// getObject fetches the class object of a WMI class.
func (c *Client) getObject(class string) (com.COMObject, error) {
	path, err := com.NewBSTR(class)
	if err != nil {
		return 0, err
	}
	defer path.Free()

	var obj unsafe.Pointer
	r, err := c.services.Call(slotGetObject,
		uintptr(path),
		0, // lFlags
		0, // ppCallResult
		uintptr(unsafe.Pointer(&obj)),
		0, // ppContext
	)
	if err != nil {
		return 0, err
	}
	if failed(r) {
		return 0, fmt.Errorf("wmi: GetObject(%s): %w", class, hresult(r))
	}
	if obj == nil {
		return 0, fmt.Errorf("wmi: GetObject(%s): 没有拿到对象", class)
	}
	result := com.NewCOMObject(obj)
	com.SetProxyBlanket(result)
	return result, nil
}

// getMethod returns the in-parameters and out-parameters definitions of a
// method. The in-parameters definition is nil for methods without inputs.
func getMethod(class com.COMObject, method string) (com.COMObject, com.COMObject, error) {
	name, err := com.NewBSTR(method)
	if err != nil {
		return 0, 0, err
	}
	defer name.Free()

	var inDef, outDef unsafe.Pointer
	r, err := class.Call(slotGetMethod,
		uintptr(name),
		0, // lFlags
		uintptr(unsafe.Pointer(&inDef)),
		uintptr(unsafe.Pointer(&outDef)),
	)
	if err != nil {
		return 0, 0, err
	}
	if failed(r) {
		return 0, 0, fmt.Errorf("wmi: GetMethod(%s): %w", method, hresult(r))
	}

	var inObj, outObj com.COMObject
	if inDef != nil {
		inObj = com.NewCOMObject(inDef)
		com.SetProxyBlanket(inObj)
	}
	if outDef != nil {
		outObj = com.NewCOMObject(outDef)
		com.SetProxyBlanket(outObj)
	}
	return inObj, outObj, nil
}

// spawnInstance creates an instance of a class object.
func spawnInstance(def com.COMObject) (com.COMObject, error) {
	if def == 0 {
		return 0, fmt.Errorf("wmi: 这个方法没有输入参数")
	}
	var obj unsafe.Pointer
	r, err := def.Call(slotSpawnInstance,
		0, // lFlags
		uintptr(unsafe.Pointer(&obj)),
		0,
	)
	if err != nil {
		return 0, err
	}
	if failed(r) {
		return 0, fmt.Errorf("wmi: SpawnInstance: %w", hresult(r))
	}
	if obj == nil {
		return 0, fmt.Errorf("wmi: SpawnInstance: 没有拿到对象")
	}
	result := com.NewCOMObject(obj)
	com.SetProxyBlanket(result)
	return result, nil
}

// execMethod invokes a method on an object path.
func (c *Client) execMethod(instancePath, method string, inParams com.COMObject) (com.COMObject, error) {
	path, err := com.NewBSTR(instancePath)
	if err != nil {
		return 0, err
	}
	defer path.Free()
	name, err := com.NewBSTR(method)
	if err != nil {
		return 0, err
	}
	defer name.Free()

	var outParams unsafe.Pointer
	r, err := c.services.Call(slotExecMethod,
		uintptr(path),
		uintptr(name),
		0, // lFlags
		0, // ppCallResult
		uintptr(inParams),
		uintptr(unsafe.Pointer(&outParams)),
		0, // pFlags
	)
	if err != nil {
		return 0, err
	}
	if failed(r) {
		return 0, fmt.Errorf("wmi: ExecMethod(%s): %w", method, hresult(r))
	}
	if outParams == nil {
		return 0, fmt.Errorf("wmi: ExecMethod(%s): 没有拿到输出参数", method)
	}
	result := com.NewCOMObject(outParams)
	com.SetProxyBlanket(result)
	return result, nil
}

// readOutputs reads the output parameters of a method call.
//
// 提供程序的名字是主路，因为在每个版本的 WMI 上都读不到输出参数定义的
// 属性表。属性表要是能读出来，里面的名字也一并读掉，这样一个给输出参数
// 起别的名字的提供程序也不会被漏掉。
func readOutputs(outParams, outDef com.COMObject) (map[string]Value, error) {
	result := make(map[string]Value)

	for _, name := range outputNames(outDef) {
		value, err := readValue(outParams, name)
		if err != nil {
			continue
		}
		result[name] = value
	}
	return result, nil
}

// outputNames returns the names of the output parameters to try.
//
// 提供程序的名字排在前面，因为在每个版本的 WMI 上都管用的是它们。输出参数
// 定义的属性表只有在读得出来的时候才加进来。
func outputNames(outDef com.COMObject) []string {
	listed, err := propertyNames(outDef)
	if err != nil {
		// 属性表读不出来，就只剩提供程序的名字了。
		listed = nil
	}
	return mergeOutputNames(listed)
}

// mergeOutputNames puts the names of the property list behind the names of
// the providers.
//
// 提供程序的名字排在前面，因为在每个版本的 WMI 上都管用的是它们。已经
// 在那里的名字不会再加一次。
func mergeOutputNames(listed []string) []string {
	names := make([]string, 0, len(conventionalOutputNames)+len(listed))
	names = append(names, conventionalOutputNames...)
	for _, name := range listed {
		if name == "" {
			continue
		}
		if contains(names, name) {
			continue
		}
		names = append(names, name)
	}
	return names
}

// contains reports whether the list carries the name.
func contains(names []string, name string) bool {
	for _, candidate := range names {
		if candidate == name {
			return true
		}
	}
	return false
}

// readString reads a string property from a class object.
func readString(obj com.COMObject, name string) (string, error) {
	value, err := readValue(obj, name)
	if err != nil {
		return "", err
	}
	return value.String(), nil
}
