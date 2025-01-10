package gumblemalgo

import (
	"errors"
	"fmt"

	"github.com/gen2brain/malgo"
)

var (
	ErrDeviceNotFound = errors.New("device not found")
	ErrInvalidState   = errors.New("invalid state")
)

func (s *Stream) SetupDevice(deviceType malgo.DeviceType, deviceName string, audioProc malgo.DataProc) error {
	devices, err := s.RetrieveDeviceList(deviceType, true)
	if err != nil {
		return err
	}

	deviceInfo, found := LookupDeviceByName(deviceName, devices)
	if !found {
		return ErrDeviceNotFound
	}

	deviceConfig := malgo.DefaultDeviceConfig(deviceType)

	deviceConfig.PeriodSizeInFrames = uint32(s.sourceFrameSize)
	deviceConfig.SampleRate = 48000

	switch deviceType {
	case malgo.Playback:
		s.playbackDeviceInfo = &deviceInfo
		deviceConfig.Playback.DeviceID = deviceInfo.ID.Pointer()
		deviceConfig.Playback.Format = malgo.FormatS16
		deviceConfig.Playback.Channels = 1

		s.playbackDeviceConfig = &deviceConfig
	case malgo.Capture:
		s.captureDeviceInfo = &deviceInfo
		deviceConfig.Capture.DeviceID = deviceInfo.ID.Pointer()
		deviceConfig.Capture.Format = malgo.FormatS16
		deviceConfig.Capture.Channels = 1

		s.captureDeviceConfig = &deviceConfig
	}

	deviceCallbacks := malgo.DeviceCallbacks{
		Data: audioProc,
	}

	device, err := malgo.InitDevice(s.malgoContext.Context, deviceConfig, deviceCallbacks)
	if err != nil {
		return err
	}

	switch deviceType {
	case malgo.Playback:
		s.playbackDevice = device
	case malgo.Capture:
		s.captureDevice = device
	}

	return nil
}

func LookupDeviceByName(name string, devices []malgo.DeviceInfo) (malgo.DeviceInfo, bool) {
	for _, device := range devices {
		if device.Name() == name {
			return device, true
		}
	}
	return malgo.DeviceInfo{}, false
}

func (s *Stream) RetrieveDeviceList(deviceType malgo.DeviceType, print bool) ([]malgo.DeviceInfo, error) {
	devices, err := s.malgoContext.Devices(deviceType)
	if err != nil {
		return nil, err
	}

	if print {
		deviceTypeString := DecodeDeviceType(deviceType)
		fmt.Printf("%s Devices:\n", deviceTypeString)
		for _, device := range devices {
			fmt.Printf("- %s\n", device.Name())
		}
	}

	return devices, nil
}

func DecodeDeviceType(deviceType malgo.DeviceType) string {
	switch deviceType {
	case malgo.Playback:
		return "Playback"
	case malgo.Capture:
		return "Capture"
	case malgo.Duplex:
		return "Duplex"
	case malgo.Loopback:
		return "Loopback"
	}
	return "Unknown"
}
