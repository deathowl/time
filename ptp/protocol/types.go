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

package protocol

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"time"
)

// 2 ** 16
const twoPow16 = 65536

// MessageType is type for Message Types
type MessageType uint8

// As per Table 36 Values of messageType field
const (
	MessageSync               MessageType = 0x0
	MessageDelayReq           MessageType = 0x1
	MessagePDelayReq          MessageType = 0x2
	MessagePDelayResp         MessageType = 0x3
	MessageFollowUp           MessageType = 0x8
	MessageDelayResp          MessageType = 0x9
	MessagePDelayRespFollowUp MessageType = 0xA
	MessageAnnounce           MessageType = 0xB
	MessageSignaling          MessageType = 0xC
	MessageManagement         MessageType = 0xD
)

// MessageTypeToString is a map from MessageType to string
var MessageTypeToString = map[MessageType]string{
	MessageSync:               "SYNC",
	MessageDelayReq:           "DELAY_REQ",
	MessagePDelayReq:          "PDELAY_REQ",
	MessagePDelayResp:         "PDELAY_RES",
	MessageFollowUp:           "FOLLOW_UP",
	MessageDelayResp:          "DELAY_RESP",
	MessagePDelayRespFollowUp: "PDELAY_RESP_FOLLOW_UP",
	MessageAnnounce:           "ANNOUNCE",
	MessageSignaling:          "SIGNALING",
	MessageManagement:         "MANAGEMENT",
}

func (m MessageType) String() string {
	return MessageTypeToString[m]
}

// SdoIDAndMsgType is a uint8 where first 4 bites contain SdoID and last 4 bits MessageType
type SdoIDAndMsgType uint8

// MsgType extracts MessageType from SdoIDAndMsgType
func (m SdoIDAndMsgType) MsgType() MessageType {
	return MessageType(m & 0xf) // last 4 bits
}

// NewSdoIDAndMsgType builds new SdoIDAndMsgType from MessageType and flags
func NewSdoIDAndMsgType(msgType MessageType, sdoID uint8) SdoIDAndMsgType {
	return SdoIDAndMsgType(sdoID<<4 | uint8(msgType))
}

// ProbeMsgType reads first 8 bits of data and tries to decode it to SdoIDAndMsgType, then return MessageType
func ProbeMsgType(data []byte) (msg MessageType, err error) {
	if len(data) < 1 {
		return 0, fmt.Errorf("not enough data to probe MsgType")
	}
	return SdoIDAndMsgType(data[0]).MsgType(), nil
}

// TLVType is type for TLV types
type TLVType uint16

// TLV abstracts away any TLV
type TLV interface {
	Type() TLVType
}

const tlvHeadSize = 4

// TLVHead is a common part of all TLVs
type TLVHead struct {
	TLVType     TLVType
	LengthField uint16 // The length of all TLVs shall be an even number of octets
}

// Type implements TLV interface
func (t TLVHead) Type() TLVType {
	return t.TLVType
}

func tlvHeadMarshalBinaryTo(t *TLVHead, b []byte) {
	binary.BigEndian.PutUint16(b, uint16(t.TLVType))
	binary.BigEndian.PutUint16(b[2:], t.LengthField)
}

// As per Table 52 tlvType values
const (
	TLVManagement                           TLVType = 0x0001
	TLVManagementErrorStatus                TLVType = 0x0002
	TLVOrganizationExtension                TLVType = 0x0003
	TLVRequestUnicastTransmission           TLVType = 0x0004
	TLVGrantUnicastTransmission             TLVType = 0x0005
	TLVCancelUnicastTransmission            TLVType = 0x0006
	TLVAcknowledgeCancelUnicastTransmission TLVType = 0x0007
	TLVPathTrace                            TLVType = 0x0008
	TLVAlternateTimeOffsetIndicator         TLVType = 0x0009
	// Remaining 52tlvType TLVs not implemented
)

// TLVTypeToString is a map from TLVType to string
var TLVTypeToString = map[TLVType]string{
	TLVManagement:                           "MANAGEMENT",
	TLVManagementErrorStatus:                "MANAGEMENT_ERROR_STATUS",
	TLVOrganizationExtension:                "ORGANIZATION_EXTENSION",
	TLVRequestUnicastTransmission:           "REQUEST_UNICAST_TRANSMISSION",
	TLVGrantUnicastTransmission:             "GRANT_UNICAST_TRANSMISSION",
	TLVCancelUnicastTransmission:            "CANCEL_UNICAST_TRANSMISSION",
	TLVAcknowledgeCancelUnicastTransmission: "ACKNOWLEDGE_CANCEL_UNICAST_TRANSMISSION",
	TLVPathTrace:                            "PATH_TRACE",
	TLVAlternateTimeOffsetIndicator:         "ALTERNATE_TIME_OFFSET_INDICATOR",
}

func (t TLVType) String() string {
	return TLVTypeToString[t]
}

// IntFloat is a float64 stored in int64
type IntFloat int64

// Value decodes IntFloat to float64
func (t IntFloat) Value() float64 {
	return float64(t) / twoPow16
}

/*
TimeInterval is the time interval expressed in nanoseconds, multiplied by 2**16.
Positive or negative time intervals outside the maximum range of this data type shall be encoded as the largest
positive and negative values of the data type, respectively.
For example, 2.5 ns is expressed as 0000 0000 0002 8000 base 16
*/
type TimeInterval IntFloat

// Nanoseconds decodes TimeInterval to human-understandable nanoseconds
func (t TimeInterval) Nanoseconds() float64 {
	return IntFloat(t).Value()
}

func (t TimeInterval) String() string {
	return fmt.Sprintf("TimeInterval(%fns)", t.Nanoseconds())
}

// NewTimeInterval returns TimeInterval built from Nanoseconds
func NewTimeInterval(ns float64) TimeInterval {
	return TimeInterval(ns * twoPow16)
}

/*
Correction is the value of the correction measured in nanoseconds and multiplied by 2**16.
For example, 2.5 ns is represented as 0000 0000 0002 8000 base 16
A value of one in all bits, except the most significant, of the field shall indicate that the correction is too big to be represented.
*/
type Correction IntFloat

// Nanoseconds decodes Correction to human-understandable nanoseconds
func (t Correction) Nanoseconds() float64 {
	return IntFloat(t).Value()
}

func (t Correction) String() string {
	if t.TooBig() {
		return "Correction(Too big)"
	}
	return fmt.Sprintf("Correction(%fns)", t.Nanoseconds())
}

// TooBig means correction is too big to be represented.
func (t Correction) TooBig() bool {
	return t == 0x7fffffffffffffff // one in all bits, except the most significant
}

// NewCorrection returns Correction built from Nanoseconds
func NewCorrection(ns float64) Correction {
	return Correction(ns * twoPow16)
}

// The ClockIdentity type identifies unique entities within a PTP Network, e.g. a PTP Instance or an entity of a common service.
type ClockIdentity uint64

// String formats ClockIdentity same way ptp4l pmc client does
func (c ClockIdentity) String() string {
	ptr := make([]byte, 8)
	binary.BigEndian.PutUint64(ptr, uint64(c))
	return fmt.Sprintf("%02x%02x%02x.%02x%02x.%02x%02x%02x",
		ptr[0], ptr[1], ptr[2], ptr[3],
		ptr[4], ptr[5], ptr[6], ptr[7],
	)
}

// MAC turns ClockIdentity into the MAC address it was based upon. EUI-48 is assumed.
func (c ClockIdentity) MAC() net.HardwareAddr {
	asBytes := [8]byte{}
	binary.BigEndian.PutUint64(asBytes[:], uint64(c))
	mac := make(net.HardwareAddr, 6)
	mac[0] = asBytes[0]
	mac[1] = asBytes[1]
	mac[2] = asBytes[2]
	mac[3] = asBytes[5]
	mac[4] = asBytes[6]
	mac[5] = asBytes[7]
	return mac
}

// NewClockIdentity creates new ClockIdentity from MAC address
func NewClockIdentity(mac net.HardwareAddr) (ClockIdentity, error) {
	b := [8]byte{}
	macLen := len(mac)
	if macLen == 6 { // EUI-48
		b[0] = mac[0]
		b[1] = mac[1]
		b[2] = mac[2]
		b[3] = 0xFF
		b[4] = 0xFE
		b[5] = mac[3]
		b[6] = mac[4]
		b[7] = mac[5]
	} else if macLen == 8 { // EUI-64
		copy(b[:], mac)
	} else {
		return 0, fmt.Errorf("unsupported MAC %v, must be either EUI48 or EUI64", mac)
	}
	return ClockIdentity(binary.BigEndian.Uint64(b[:])), nil
}

// The PortIdentity type identifies a PTP Port or a Link Port
type PortIdentity struct {
	ClockIdentity ClockIdentity
	PortNumber    uint16
}

// String formats PortIdentity same way ptp4l pmc client does
func (p PortIdentity) String() string {
	return fmt.Sprintf("%s-%d", p.ClockIdentity, p.PortNumber)
}

/*
Timestamp type represents a positive time with respect to the epoch.
The secondsField member is the integer portion of the timestamp in units of seconds.
The nanosecondsField member is the fractional portion of the timestamp in units of nanoseconds.
The nanosecondsField member is always less than 10**9 .
For example:
+2.000000001 seconds is represented by secondsField = 0000 0000 0002 base 16 and nanosecondsField= 0000 0001 base 16.
*/
type Timestamp struct {
	Seconds     [6]uint8 // uint48
	Nanoseconds uint32
}

// Time turns Timestamp into normal Go time.Time
func (t Timestamp) Time() time.Time {
	b := append([]byte{0x0, 0x0}, t.Seconds[:]...)
	secs := binary.BigEndian.Uint64(b)
	return time.Unix(int64(secs), int64(t.Nanoseconds))
}

func (t Timestamp) String() string {
	if t.Nanoseconds == 0 && t.Seconds == [6]uint8{0, 0, 0, 0, 0, 0} {
		return "Timestamp(empty)"
	}
	return fmt.Sprintf("Timestamp(%s)", t.Time())
}

// NewTimestamp allows to create Timestamp from time.Time
func NewTimestamp(t time.Time) Timestamp {
	ts := Timestamp{
		Nanoseconds: uint32(t.Nanosecond()),
	}
	b := [8]byte{}
	binary.BigEndian.PutUint64(b[:], uint64(t.Unix()))
	// take last 6 bytes from 8 bytes of int64
	copy(ts.Seconds[:], b[2:])
	return ts
}

// ClockClass represents a PTP clock class
type ClockClass uint8

// Available Clock Classes
// https://datatracker.ietf.org/doc/html/rfc8173#section-7.6.2.4
const (
	ClockClass6  ClockClass = 6
	ClockClass7  ClockClass = 7
	ClockClass13 ClockClass = 13
	ClockClass52 ClockClass = 52
	Clockclass58 ClockClass = 58
)

// ClockAccuracy represents a PTP clock accuracy
type ClockAccuracy uint8

// Available Clock Accuracy
// https://datatracker.ietf.org/doc/html/rfc8173#section-7.6.2.5
const (
	ClockAccuracyNanosecond25       ClockAccuracy = 0x20
	ClockAccuracyNanosecond100      ClockAccuracy = 0x21
	ClockAccuracyNanosecond250      ClockAccuracy = 0x22
	ClockAccuracyMicrosecond1       ClockAccuracy = 0x23
	ClockAccuracyMicrosecond2point5 ClockAccuracy = 0x24
	ClockAccuracyMicrosecond10      ClockAccuracy = 0x25
	ClockAccuracyMicrosecond25      ClockAccuracy = 0x26
	ClockAccuracyMicrosecond100     ClockAccuracy = 0x27
	ClockAccuracyMicrosecond250     ClockAccuracy = 0x28
	ClockAccuracyMillisecond1       ClockAccuracy = 0x29
	ClockAccuracyMillisecond2point5 ClockAccuracy = 0x2A
	ClockAccuracyMillisecond10      ClockAccuracy = 0x2B
	ClockAccuracyMillisecond25      ClockAccuracy = 0x2C
	ClockAccuracyMillisecond100     ClockAccuracy = 0x2D
	ClockAccuracyMillisecond250     ClockAccuracy = 0x2E
	ClockAccuracySecond1            ClockAccuracy = 0x2F
	ClockAccuracySecond10           ClockAccuracy = 0x30
	ClockAccuracySecondGreater10    ClockAccuracy = 0x31
	ClockAccuracyUnknown            ClockAccuracy = 0xFE
)

// ClockAccuracyFromOffset returns PTP Clock Accuracy covering the time.Duration
func ClockAccuracyFromOffset(offset time.Duration) ClockAccuracy {
	if offset < 0 {
		offset *= -1
	}

	// https://datatracker.ietf.org/doc/html/rfc8173#section-7.6.2.4
	if offset <= 25*time.Nanosecond {
		return ClockAccuracyNanosecond25
	} else if offset <= 100*time.Nanosecond {
		return ClockAccuracyNanosecond100
	} else if offset <= 250*time.Nanosecond {
		return ClockAccuracyNanosecond250
	} else if offset <= time.Microsecond {
		return ClockAccuracyMicrosecond1
	} else if offset <= 2500*time.Nanosecond {
		return ClockAccuracyMicrosecond2point5
	} else if offset <= 10*time.Microsecond {
		return ClockAccuracyMicrosecond10
	} else if offset <= 25*time.Microsecond {
		return ClockAccuracyMicrosecond25
	} else if offset <= 100*time.Microsecond {
		return ClockAccuracyMicrosecond100
	} else if offset <= 250*time.Microsecond {
		return ClockAccuracyMicrosecond250
	} else if offset <= time.Millisecond {
		return ClockAccuracyMillisecond1
	} else if offset <= 2500*time.Microsecond {
		return ClockAccuracyMillisecond2point5
	} else if offset <= 10*time.Millisecond {
		return ClockAccuracyMillisecond10
	} else if offset <= 25*time.Millisecond {
		return ClockAccuracyMillisecond25
	} else if offset <= 100*time.Millisecond {
		return ClockAccuracyMillisecond100
	} else if offset <= 250*time.Millisecond {
		return ClockAccuracyMillisecond250
	} else if offset <= time.Second {
		return ClockAccuracySecond1
	} else if offset <= 10*time.Second {
		return ClockAccuracySecond10
	}

	return ClockAccuracySecondGreater10
}

// ClockQuality represents the quality of a clock.
type ClockQuality struct {
	ClockClass              ClockClass
	ClockAccuracy           ClockAccuracy
	OffsetScaledLogVariance uint16
}

// TimeSource indicates the immediate source of time used by the Grandmaster PTP Instance
type TimeSource uint8

// TimeSource values, Table 6 timeSource enumeration
const (
	TimeSourceAtomicClock        TimeSource = 0x10
	TimeSourceGNSS               TimeSource = 0x20
	TimeSourceTerrestrialRadio   TimeSource = 0x30
	TimeSourceSerialTimeCode     TimeSource = 0x39
	TimeSourcePTP                TimeSource = 0x40
	TimeSourceNTP                TimeSource = 0x50
	TimeSourceHandSet            TimeSource = 0x60
	TimeSourceOther              TimeSource = 0x90
	TimeSourceInternalOscillator TimeSource = 0xa0
)

// TimeSourceToString is a map from TimeSource to string
var TimeSourceToString = map[TimeSource]string{
	TimeSourceAtomicClock:        "ATOMIC_CLOCK",
	TimeSourceGNSS:               "GNSS",
	TimeSourceTerrestrialRadio:   "TERRESTRIAL_RADIO",
	TimeSourceSerialTimeCode:     "SERIAL_TIME_CODE",
	TimeSourcePTP:                "PTP",
	TimeSourceNTP:                "NTP",
	TimeSourceHandSet:            "HAND_SET",
	TimeSourceOther:              "OTHER",
	TimeSourceInternalOscillator: "INTERNAL_OSCILLATOR",
}

func (t TimeSource) String() string {
	return TimeSourceToString[t]
}

// LogInterval shall be the logarithm, to base 2, of the requested period in seconds.
// In layman's terms, it's specified as a power of two in seconds.
type LogInterval int8

// Duration returns LogInterval as time.Duration
func (i LogInterval) Duration() time.Duration {
	secs := math.Pow(2, float64(i))
	return time.Duration(secs * float64(time.Second))
}

// NewLogInterval returns new LogInterval from time.Duration.
// The values of these logarithmic attributes shall be selected from integers in the range -128 to 127 subject to
// further limits established in the applicable PTP Profile.
func NewLogInterval(d time.Duration) (LogInterval, error) {
	li := int(math.Log2(d.Seconds()))
	if li > 127 {
		return 0, fmt.Errorf("logInterval %d is too big", li)
	}
	if li < -128 {
		return 0, fmt.Errorf("logInterval %d is too small", li)
	}
	return LogInterval(li), nil
}

/*
PTPText data type is used to represent textual material in PTP messages.
TextField is encoded as UTF-8.
The most significant byte of the leading text symbol shall be the element of the array with index 0.
UTF-8 encoding has variable length, thus LengthField can be larger than number of characters.
type PTPText struct {
	LengthField uint8
	TextField   []byte
}
*/
type PTPText string

// UnmarshalBinary populates ptptext from bytes
func (p *PTPText) UnmarshalBinary(rawBytes []byte) error {
	var length uint8
	reader := bytes.NewReader(rawBytes)
	if err := binary.Read(reader, binary.BigEndian, &length); err != nil {
		return fmt.Errorf("reading PTPText LengthField: %w", err)
	}
	if length == 0 {
		// can be zero len, just empty string
		return nil
	}
	if len(rawBytes) < int(length+1) {
		return fmt.Errorf("text field is too short, need %d got %d", len(rawBytes), length+1)
	}
	text := make([]byte, length)
	if err := binary.Read(reader, binary.BigEndian, text); err != nil {
		return fmt.Errorf("reading PTPText TextField of len=%d: %w", length, err)
	}
	*p = PTPText(text)
	return nil
}

// MarshalBinary converts ptptext to []bytes
func (p *PTPText) MarshalBinary() ([]byte, error) {
	rawText := []byte(*p)
	if len(rawText) > 255 {
		return nil, fmt.Errorf("text is too long")
	}
	length := uint8(len(rawText))
	var bytes bytes.Buffer
	if err := binary.Write(&bytes, binary.BigEndian, length); err != nil {
		return nil, err
	}
	if err := binary.Write(&bytes, binary.BigEndian, rawText); err != nil {
		return nil, err
	}
	// padding to make sure packet length is even
	if length%2 != 0 {
		if err := bytes.WriteByte(0); err != nil {
			return nil, err
		}
	}
	return bytes.Bytes(), nil
}

// PortState is a enum describing one of possible states of port state machines
type PortState uint8

// Table 20 PTP state enumeration
const (
	PortStateInitializing PortState = iota + 1
	PortStateFaulty
	PortStateDisabled
	PortStateListening
	PortStatePreMaster
	PortStateMaster
	PortStatePassive
	PortStateUncalibrated
	PortStateSlave
	PortStateGrandMaster /*non-standard extension*/
)

// PortStateToString is a map from PortState to string
var PortStateToString = map[PortState]string{
	PortStateInitializing: "INITIALIZING",
	PortStateFaulty:       "FAULTY",
	PortStateDisabled:     "DISABLED",
	PortStateListening:    "LISTENING",
	PortStatePreMaster:    "PRE_MASTER",
	PortStateMaster:       "MASTER",
	PortStatePassive:      "PASSIVE",
	PortStateUncalibrated: "UNCALIBRATED",
	PortStateSlave:        "SLAVE",
	PortStateGrandMaster:  "GRAND_MASTER",
}

func (ps PortState) String() string {
	return PortStateToString[ps]
}

// TransportType is a enum describing network transport protocol types
type TransportType uint16

// Table 3 networkProtocol enumeration
const (
	/* 0 is Reserved in spec. Use it for UDS */
	TransportTypeUDS TransportType = iota
	TransportTypeUDPIPV4
	TransportTypeUDPIPV6
	TransportTypeIEEE8023
	TransportTypeDeviceNet
	TransportTypeControlNet
	TransportTypePROFINET
)

// TransportTypeToString is a map from TransportType to string
var TransportTypeToString = map[TransportType]string{
	TransportTypeUDS:        "UDS",
	TransportTypeUDPIPV4:    "UDP_IPV4",
	TransportTypeUDPIPV6:    "UDP_IPV6",
	TransportTypeIEEE8023:   "IEEE_802_3",
	TransportTypeDeviceNet:  "DEVICENET",
	TransportTypeControlNet: "CONTROLNET",
	TransportTypePROFINET:   "PROFINET",
}

func (t TransportType) String() string {
	return TransportTypeToString[t]
}

// 5.3.6 PortAddress
type PortAddress struct {
	NetworkProtocol TransportType
	AddressLength   uint16
	AddressField    []byte
}

func (p *PortAddress) UnmarshalBinary(b []byte) error {
	if len(b) < 8 {
		return fmt.Errorf("not enough data to decode PortAddress")
	}
	p.NetworkProtocol = TransportType(binary.BigEndian.Uint16(b[0:]))
	p.AddressLength = binary.BigEndian.Uint16(b[2:])
	if len(b) < 4+int(p.AddressLength) {
		return fmt.Errorf("not enough data to decode PortAddress address")
	}

	p.AddressField = net.IP(b[4 : 4+p.AddressLength])
	return nil
}

func (p *PortAddress) IP() (net.IP, error) {
	if p.NetworkProtocol != TransportTypeUDPIPV4 && p.NetworkProtocol != TransportTypeUDPIPV6 {
		return nil, fmt.Errorf("unsupported network protocol %s (%d)", p.NetworkProtocol, p.NetworkProtocol)
	}
	return net.IP(p.AddressField), nil
}

// MarshalBinary converts ptptext to []bytes
func (p *PortAddress) MarshalBinary() ([]byte, error) {
	var bytes bytes.Buffer
	if err := binary.Write(&bytes, binary.BigEndian, p.NetworkProtocol); err != nil {
		return nil, err
	}
	if err := binary.Write(&bytes, binary.BigEndian, p.AddressLength); err != nil {
		return nil, err
	}
	if err := binary.Write(&bytes, binary.BigEndian, p.AddressField); err != nil {
		return nil, err
	}
	return bytes.Bytes(), nil
}
