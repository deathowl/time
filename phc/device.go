/*
Copyright (c) Facebook, Inc. and its affiliates.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package phc

import (
	"fmt"
	"unsafe"

	"github.com/facebook/time/phc/unix" // a temporary shim for "golang.org/x/sys/unix" until v0.27.0 is cut
	"github.com/vtolstov/go-ioctl"
)

// Missing from sys/unix package, defined in Linux include/uapi/linux/ptp_clock.h
const (
	ptpMaxSamples = 25
	ptpClkMagic   = '='
	nsPerSec      = uint32(1000000000)
)

// ioctlPTPSysOffsetExtended is an IOCTL to get extended offset
var ioctlPTPSysOffsetExtended = ioctl.IOWR(ptpClkMagic, 9, unsafe.Sizeof(PTPSysOffsetExtended{}))

// ioctlPTPSysOffsetPrecise is an IOCTL to get precise offset
var ioctlPTPSysOffsetPrecise = ioctl.IOWR(ptpClkMagic, 8, unsafe.Sizeof(PTPSysOffsetPrecise{}))

// ioctlPTPClockGetCaps is an IOCTL to get PTP clock capabilities
var ioctlPTPClockGetcaps = ioctl.IOR(ptpClkMagic, 1, unsafe.Sizeof(PTPClockCaps{}))

// iocPinGetfunc is an IOCTL req corresponding to PTP_PIN_GETFUNC in linux/ptp_clock.h
var iocPinGetfunc = ioctl.IOWR(ptpClkMagic, 6, unsafe.Sizeof(rawPinDesc{}))

// iocPinSetfunc is an IOCTL req corresponding to PTP_PIN_SETFUNC in linux/ptp_clock.h
var iocPinSetfunc = ioctl.IOW(ptpClkMagic, 7, unsafe.Sizeof(rawPinDesc{}))

// iocPinSetfunc is an IOCTL req corresponding to PTP_PIN_SETFUNC2 in linux/ptp_clock.h
var iocPinSetfunc2 = ioctl.IOW(ptpClkMagic, 16, unsafe.Sizeof(rawPinDesc{}))

// ioctlPTPPeroutRequest2 is an IOCTL req corresponding to PTP_PEROUT_REQUEST2 in linux/ptp_clock.h
var ioctlPTPPeroutRequest2 = ioctl.IOW(ptpClkMagic, 12, unsafe.Sizeof(PTPPeroutRequest{}))

// ioctlExtTTSRequest2 is an IOCTL req corresponding to PTP_EXTTS_REQUEST2 in linux/ptp_clock.h
var ioctlExtTTSRequest2 = ioctl.IOW(ptpClkMagic, 11, unsafe.Sizeof(PTPExtTTSRequest{}))

// PTPSysOffsetExtended as defined in linux/ptp_clock.h
type PTPSysOffsetExtended struct {
	NSamples uint32    /* Desired number of measurements. */
	Reserved [3]uint32 /* Reserved for future use. */
	/*
	 * Array of [system, phc, system] time stamps. The kernel will provide
	 * 3*n_samples time stamps.
	 * - system time right before reading the lowest bits of the PHC timestamp
	 * - PHC time
	 * - system time immediately after reading the lowest bits of the PHC timestamp
	 */
	TS [ptpMaxSamples][3]PTPClockTime
}

// PTPSysOffsetPrecise as defined in linux/ptp_clock.h
type PTPSysOffsetPrecise struct {
	Device      PTPClockTime
	SysRealTime PTPClockTime
	SysMonoRaw  PTPClockTime
	Reserved    [4]uint32 /* Reserved for future use. */
}

// PinDesc represents the C struct ptp_pin_desc as defined in linux/ptp_clock.h
type PinDesc struct {
	Name  string  // Hardware specific human readable pin name
	Index uint    // Pin index in the range of zero to ptp_clock_caps.n_pins - 1
	Func  PinFunc // Which of the PTP_PF_xxx functions to use on this pin
	Chan  uint    // The specific channel to use for this function.
	// private fields
	dev *Device
}

// SetFunc uses an ioctl to change the pin function
func (pd *PinDesc) SetFunc(pf PinFunc) error {
	if err := pd.dev.setPinFunc(pd.Index, pf, pd.Chan); err != nil {
		return err
	}
	pd.Func = pf
	return nil
}

type rawPinDesc struct {
	Name  [64]byte  // Hardware specific human readable pin name
	Index uint32    // Pin index in the range of zero to ptp_clock_caps.n_pins - 1
	Func  uint32    // Which of the PTP_PF_xxx functions to use on this pin
	Chan  uint32    // The specific channel to use for this function.
	Rsv   [5]uint32 // Reserved for future use.
}

// PTPPeroutRequest as defined in linux/ptp_clock.h
type PTPPeroutRequest struct {
	//   * Represents either absolute start time or phase offset.
	//   * Absolute start time if (flags & PTP_PEROUT_PHASE) is unset.
	//   * Phase offset if (flags & PTP_PEROUT_PHASE) is set.
	//   * If set the signal should start toggling at an
	//	 * unspecified integer multiple of the period, plus this value.
	//	 * The start time should be "as soon as possible".
	StartOrPhase PTPClockTime
	Period       PTPClockTime // Desired period, zero means disable
	Index        uint32       // Which channel to configure
	Flags        uint32       // Configuration flags
	On           PTPClockTime // "On" time of the signal. Must be lower than the period. Valid only if (flags & PTP_PEROUT_DUTY_CYCLE) is set.
}

// Bits of the ptp_extts_request.flags field:
const (
	PTPEnableFeature uint32 = 1 << 0 // Enable feature
	PTPRisingEdge    uint32 = 1 << 1 // Rising edge
	PTPFallingEdge   uint32 = 1 << 2 // Falling edge
	PTPStrictFlags   uint32 = 1 << 3 // Strict flags
	PTPExtOffset     uint32 = 1 << 4 // External offset
)

// PTPExtTTSRequest as defined in linux/ptp_clock.h
type PTPExtTTSRequest struct {
	index uint32
	flags uint32
	rsv   [2]uint32
}

// PTPExtTTS as defined in linux/ptp_clock.h
type PTPExtTTS struct {
	T     PTPClockTime /* Time when event occurred. */
	Index uint32       /* Which channel produced the event. Corresponds to the 'index' field of the PTP_EXTTS_REQUEST and PTP_PEROUT_REQUEST ioctls.*/
	Flags uint32       /* Event flags */
	Rsv   [2]uint32    /* Reserved for future use. */
}

// PTPClockTime as defined in linux/ptp_clock.h
type PTPClockTime struct {
	Sec      int64  /* seconds */
	NSec     uint32 /* nanoseconds */
	Reserved uint32
}

// PTPClockCaps as defined in linux/ptp_clock.h
type PTPClockCaps struct {
	MaxAdj  int32 /* Maximum frequency adjustment in parts per billon. */
	NAalarm int32 /* Number of programmable alarms. */
	NExtTs  int32 /* Number of external time stamp channels. */
	NPerOut int32 /* Number of programmable periodic signals. */
	PPS     int32 /* Whether the clock supports a PPS callback. */
	NPins   int32 /* Number of input/output pins. */
	/* Whether the clock supports precise system-device cross timestamps */
	CrossTimestamping int32
	/* Whether the clock supports adjust phase */
	AdjustPhase int32
	Rsv         [12]int32 /* Reserved for future use. */
}

func (caps *PTPClockCaps) maxAdj() float64 {
	if caps == nil || caps.MaxAdj == 0 {
		return DefaultMaxClockFreqPPB
	}
	return float64(caps.MaxAdj)
}

// IfaceInfo uses an ioctl to get information for the named nic, e.g. eth0.
func IfaceInfo(iface string) (*unix.EthtoolTsInfo, error) {
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to create socket for ioctl: %w", err)
	}
	defer unix.Close(fd)
	return unix.IoctlGetEthtoolTsInfo(fd, iface)
}
