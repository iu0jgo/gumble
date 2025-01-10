package gumblemalgo

import (
	"encoding/binary"

	"github.com/gen2brain/malgo"
	"github.com/iu0jgo/gumble/gumble"
)

type Stream struct {
	client *gumble.Client
	link   gumble.Detacher

	//deviceSource    *openal.CaptureDevice
	sourceFrameSize int
	sourceStop      chan bool

	//deviceSink  *openal.Device
	//contextSink *openal.Context
	malgoContext *malgo.AllocatedContext
	// playback device
	playbackDeviceInfo   *malgo.DeviceInfo
	playbackDeviceConfig *malgo.DeviceConfig
	playbackDevice       *malgo.Device

	ingoing chan gumble.AudioBuffer

	// capture device
	captureDeviceInfo   *malgo.DeviceInfo
	captureDeviceConfig *malgo.DeviceConfig
	captureDevice       *malgo.Device
	//
	outgoing chan<- gumble.AudioBuffer
}

func New(client *gumble.Client, playbackDeviceName, captureDeviceName string) (*Stream, error) {
	s := &Stream{
		client:          client,
		sourceFrameSize: client.Config.AudioFrameSize(),
	}

	mc, err := malgo.InitContext(nil, malgo.ContextConfig{}, nil)
	if err != nil {
		return nil, err
	}
	s.malgoContext = mc

	// Move to config
	//playbackDeviceName := "Plantronics Blackwire 3225 Series"

	s.SetupDevice(malgo.Playback, playbackDeviceName, func(pOutputSample, pInputSamples []byte, framecount uint32) {

		select {
		case buf := <-s.ingoing:
			//fmt.Printf("ingoing buf len %d\n", len(buf))
			for i, value := range buf {
				//fmt.Printf("buf for %d %d\n", i, value)
				binary.LittleEndian.PutUint16(pOutputSample[i*2:(i+1)*2], uint16(value))
			}
		default:
		}
	})

	// Move to config
	//captureDeviceName := "Plantronics Blackwire 3225 Series"

	s.SetupDevice(malgo.Capture, captureDeviceName, s.sourceRoutine)

	/*
		s.deviceSource = openal.CaptureOpenDevice("", gumble.AudioSampleRate, openal.FormatMono16, uint32(s.sourceFrameSize))

		s.deviceSink = openal.OpenDevice("")
		s.contextSink = s.deviceSink.CreateContext()
		s.contextSink.Activate()

	*/

	s.ingoing = make(chan gumble.AudioBuffer, 8)
	s.playbackDevice.Start()

	s.link = client.Config.AttachAudio(s)

	return s, nil
}

func (s *Stream) Destroy() {
	s.link.Detach()

	defer func() {
		_ = s.malgoContext.Uninit()
		s.malgoContext.Free()
	}()
	/*
		if s.deviceSource != nil {
			s.StopSource()
			s.deviceSource.CaptureCloseDevice()
			s.deviceSource = nil
		}
		if s.deviceSink != nil {
			s.contextSink.Destroy()
			s.deviceSink.CloseDevice()
			s.contextSink = nil
			s.deviceSink = nil
		}
	*/
}

func (s *Stream) StartSource() error {

	if s.sourceStop != nil {
		return ErrInvalidState
	}
	s.captureDevice.Start()
	//s.deviceSource.CaptureStart()
	s.sourceStop = make(chan bool)
	s.outgoing = s.client.AudioOutgoing()
	//go s.sourceRoutine()

	return nil
}

func (s *Stream) StopSource() error {

	if s.sourceStop == nil {
		return ErrInvalidState
	}
	close(s.sourceStop)
	s.sourceStop = nil
	close(s.outgoing)
	s.outgoing = nil
	//s.deviceSource.CaptureStop()
	s.captureDevice.Stop()

	return nil
}

func (s *Stream) OnAudioStream(e *gumble.AudioStreamEvent) {

	go func() {

		for packet := range e.C {

			//fmt.Printf("OnAudioStream packet.AudioBuffer len %d\n", len(packet.AudioBuffer))

			int16Buffer := make([]int16, len(packet.AudioBuffer))
			for i := range int16Buffer {
				int16Buffer[i] = int16(packet.AudioBuffer[i])
				//s.ingoing <- gumble.AudioBuffer(int16Buffer)
			}

			s.ingoing <- gumble.AudioBuffer(int16Buffer[:480])
			s.ingoing <- gumble.AudioBuffer(int16Buffer[480:])
		}

	}()

	/*
		go func() {
			source := openal.NewSource()
			emptyBufs := openal.NewBuffers(8)
			reclaim := func() {
				if n := source.BuffersProcessed(); n > 0 {
					reclaimedBufs := make(openal.Buffers, n)
					source.UnqueueBuffers(reclaimedBufs)
					emptyBufs = append(emptyBufs, reclaimedBufs...)
				}
			}
			var raw [gumble.AudioMaximumFrameSize * 2]byte
			for packet := range e.C {
				samples := len(packet.AudioBuffer)
				if samples > cap(raw) {
					continue
				}
				for i, value := range packet.AudioBuffer {
					binary.LittleEndian.PutUint16(raw[i*2:], uint16(value))
				}
				reclaim()
				if len(emptyBufs) == 0 {
					continue
				}
				last := len(emptyBufs) - 1
				buffer := emptyBufs[last]
				emptyBufs = emptyBufs[:last]
				buffer.SetData(openal.FormatMono16, raw[:samples*2], gumble.AudioSampleRate)
				source.QueueBuffer(buffer)
				if source.State() != openal.Playing {
					source.Play()
				}
			}
			reclaim()
			emptyBufs.Delete()
			source.Delete()
		}()
	*/
}

func (s *Stream) sourceRoutine(pOutputSample, pInputSamples []byte, framecount uint32) {
	int16Buffer := make([]int16, framecount)
	for i := range int16Buffer {
		int16Buffer[i] = int16(binary.LittleEndian.Uint16(pInputSamples[i*2 : (i+1)*2]))
	}
	s.outgoing <- gumble.AudioBuffer(int16Buffer)
}
