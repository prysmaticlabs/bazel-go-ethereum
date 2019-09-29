package usb

// A lot of the cgo code in this package has been deleted and commented out in bazel, this file is
// used as a stub in the timebeing so that any other packages using this library can
// still function properly and will not be broken.

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
