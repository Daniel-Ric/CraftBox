package protocol

type DeviceOS int

const DeviceUnknown DeviceOS = -1

const (
	DeviceAndroid DeviceOS = iota + 1
	DeviceIOS
	DeviceOSX
	DeviceFireOS

	DeviceGearVR
	DeviceHololens
	DeviceWin10
	DeviceWin32
	DeviceDedicated

	DeviceTVOS
	DeviceOrbis
	DeviceNX
	DeviceXBOX

	DeviceWP
	DeviceLinux
)
