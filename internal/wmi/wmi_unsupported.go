//go:build !windows

package wmi

import "errors"

// ErrUnsupported is returned by every entry point on non-Windows systems.
var ErrUnsupported = errors.New("wmi: WMI 客户端只在 Windows 上有")

// Client is a connection to one WMI namespace.
//
// Windows 之外没有 WMI，所以这个客户端开不起来。这个类型留着，是为了让
// 调用者不必知道自己跑在哪个平台上。
type Client struct{}

// Connect always fails on non-Windows systems.
func Connect(namespace string) (*Client, error) { return nil, ErrUnsupported }

// Close does nothing on non-Windows systems.
func (c *Client) Close() {}

// Instances always fails on non-Windows systems.
func (c *Client) Instances(class string) ([]string, error) { return nil, ErrUnsupported }

// Call always fails on non-Windows systems.
func (c *Client) Call(class, method, instancePath string, args map[string]Value) (map[string]Value, error) {
	return nil, ErrUnsupported
}

// ReadProperty always fails on non-Windows systems.
func (c *Client) ReadProperty(objectPath, name string) (uint64, error) { return 0, ErrUnsupported }

// ReadBytes always fails on non-Windows systems.
func (c *Client) ReadBytes(objectPath, name string) ([]byte, error) { return nil, ErrUnsupported }
