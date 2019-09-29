package usb



// Enumerate.This doesn't using the real-one so as to avoid building it with bazel
// as the real file/function introduce cgo deps.
func Enumerate(vendorID uint16, productID uint16) ([]DeviceInfo, error) {
	// no-op
	return []DeviceInfo{},nil
}

// Supported is a stub func.
func Supported() bool {
	return true
}

// Open connects to a previsouly discovered USB device.
func (info DeviceInfo) Open() (Device, error) {

    return Device(nil),nil
}
