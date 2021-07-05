package psi

import (
	"strconv"
	"time"

	"github.com/vitmod/godvb"
)

// TODO: Reconsider all descriptors decoding/encoding code. Goals:
// 1. Simple and orthogonal implementation.
// 2. Avoid allocations.

type ServiceType byte

const (
	ZeroServiceType                            = ServiceType(0x00)
	DigitalTelevisionService                   = ServiceType(0x01)
	DigitalRadioSoundService                   = ServiceType(0x02)
	TeletextService                            = ServiceType(0x03)
	NVODReferenceService                       = ServiceType(0x04)
	NVODTimeShiftedService                     = ServiceType(0x05)
	MosaicService                              = ServiceType(0x06)
	FMRadioService                             = ServiceType(0x07)
	DVBSRMService                              = ServiceType(0x08)
	_                                          = ServiceType(0x09)
	AdvancedCodecDigitalRadioSoundService      = ServiceType(0x0a)
	AdvancedCodecMosaicService                 = ServiceType(0x0b)
	DataBroadcastService                       = ServiceType(0x0c)
	_                                          = ServiceType(0x0d)
	RCSMapService                              = ServiceType(0x0e)
	RCSFLSService                              = ServiceType(0x0f)
	DVBMHPService                              = ServiceType(0x10)
	MPEG2HDDigitalTelevisionService            = ServiceType(0x11)
	_                                          = ServiceType(0x12)
	_                                          = ServiceType(0x13)
	_                                          = ServiceType(0x14)
	_                                          = ServiceType(0x15)
	H264SDDigitalTelevisionService             = ServiceType(0x16)
	H264SDNVODTimeShiftedService               = ServiceType(0x17)
	H264SDNVODReferenceService                 = ServiceType(0x18)
	H264HDDigitalTelevisionService             = ServiceType(0x19)
	H264HDNVODTimeShiftedService               = ServiceType(0x1a)
	H264HDNVODReferenceService                 = ServiceType(0x1b)
	H264StereoscopicHDDigitalTelevisionService = ServiceType(0x1c)
	H264StereoscopicHDNVODTimeShiftedService   = ServiceType(0x1d)
	H264StereoscopicHDNVODReferenceService     = ServiceType(0x1e)
	H265DigitalTelevisionService               = ServiceType(0x1f)
)

var stn = []string{
	"digital television",
	"digital radio sound",
	"teletext",
	"NVOD reference",
	"NVOD time-shifted",
	"mosaic",
	"FM radio",
	"DVB SRM",
	"reserved",
	"advanced codec digital radio sound",
	"advanced codec mosaic",
	"data broadcast",
	"reserved",
	"RCS Map",
	"RCS FLS",
	"DVB MHP",
	"MPEG-2 HD digital television",
	"reserved",
	"reserved",
	"reserved",
	"reserved",
	"advanced codec SD digital television",
	"advanced codec SD NVOD time-shifted",
	"advanced codec SD NVOD reference",
	"advanced codec HD digital television",
	"advanced codec HD NVOD time-shifted",
	"advanced codec HD NVOD reference",
	"advanced codec frame compatible plano-stereoscopic HD digital television",
	"advanced codec frame compatible plano-stereoscopic HD NVOD time-shifted",
	"advanced codec frame compatible plano-stereoscopic HD NVOD reference",
	"high hfficiency codec digital television",
}

func (t ServiceType) String() string {
	if t == 0 || t == 0xff || t > H265DigitalTelevisionService && t <= 0x7F {
		return "reserved"
	}
	if t > 0x7F {
		return "user defined"
	}
	return stn[t-1]
}

type ServiceDescriptor struct {
	Type         ServiceType
	ProviderName []byte
	ServiceName  []byte
}

func ParseServiceDescriptor(d Descriptor) (sd ServiceDescriptor, ok bool) {
	if d.Tag() != ServiceTag {
		return
	}
	data := d.Data()
	if len(data) < 2 {
		return
	}
	sd.Type = ServiceType(data[0])
	providerNameLen := data[1]
	data = data[2:]

	if len(data) < int(providerNameLen+1) {
		return
	}
	sd.ProviderName = data[:providerNameLen]
	serviceNameLen := data[providerNameLen]
	data = data[providerNameLen+1:]

	if len(data) < int(serviceNameLen) {
		return
	}
	sd.ServiceName = data[:serviceNameLen]

	ok = true
	return
}

func MakeServiceDescriptor(typ ServiceType, provider, service string) Descriptor {
	p := EncodeText(provider)
	s := EncodeText(service)
	d := MakeDescriptor(ServiceTag, 1+1+len(p)+1+len(s))
	data := d.Data()[:0]
	data = append(data, byte(typ))
	data = append(data, byte(len(p)))
	data = append(data, p...)
	data = append(data, byte(len(s)))
	data = append(data, s...)
	return d
}

type NetworkNameDescriptor []byte

func ParseNetworkNameDescriptor(d Descriptor) (nnd NetworkNameDescriptor, ok bool) {
	if d.Tag() != NetworkNameTag {
		return
	}
	nnd = NetworkNameDescriptor(d.Data())
	ok = true
	return
}

func MakeNetworkNameDescriptor(netname string) Descriptor {
	nn := EncodeText(netname)
	d := MakeDescriptor(NetworkNameTag, len(nn))
	copy(d.Data(), nn)
	return d
}

type ServiceListDescriptor []byte

func ParseServiceListDescriptor(d Descriptor) (sld ServiceListDescriptor, ok bool) {
	if d.Tag() != ServiceListTag {
		return
	}
	sld = ServiceListDescriptor(d.Data())
	ok = true
	return
}

// Pop returns first (sid, typ) pair from d. Remaining pairs are returned in rd.
// If there is no more pairs to read len(rd) == 0. If an error occurs rd = nil
func (d ServiceListDescriptor) Pop() (sid uint16, typ ServiceType, rd ServiceListDescriptor) {
	if len(d) < 3 {
		return
	}
	sid = decodeU16(d[0:2])
	typ = ServiceType(d[2])
	rd = d[3:]
	return
}

func (sld *ServiceListDescriptor) Append(sid uint16, typ ServiceType) {
	var el [3]byte
	encodeU16(el[0:2], sid)
	el[2] = byte(typ)
	*sld = append(*sld, el[:]...)
}

func (sld ServiceListDescriptor) MakeDescriptor() Descriptor {
	d := MakeDescriptor(ServiceListTag, len(sld))
	copy(d.Data(), sld)
	return d
}

type TerrestrialDeliverySystemDescriptor struct {
	Freq          int64 // center frequency [Hz]
	Bandwidth     int   // bandwidth [Hz]
	HighPrio      bool
	TimeSlicing   bool
	MPEFEC        bool
	Constellation dvb.Modulation
	Hierarchy     dvb.Hierarchy
	CodeRateHP    dvb.CodeRate
	CodeRateLP    dvb.CodeRate
	Guard         dvb.Guard
	TxMode        dvb.TxMode
	OtherFreq     bool
}

func (tds TerrestrialDeliverySystemDescriptor) Bitrate() int64 {
	bitrate := 54e6 * int64(tds.Bandwidth) / 8e6
	switch tds.CodeRateHP {
	case dvb.FEC12:
		bitrate = bitrate * 1 / 2
	case dvb.FEC23:
		bitrate = bitrate * 2 / 3
	case dvb.FEC34:
		bitrate = bitrate * 3 / 4
	case dvb.FEC45:
		bitrate = bitrate * 4 / 5
	case dvb.FEC56:
		bitrate = bitrate * 5 / 6
	case dvb.FEC67:
		bitrate = bitrate * 6 / 7
	case dvb.FEC78:
		bitrate = bitrate * 7 / 8
	default:
		bitrate = 0
	}
	switch tds.Constellation {
	case dvb.QPSK:
		bitrate = bitrate * 2 / 8
	case dvb.QAM16:
		bitrate = bitrate * 4 / 8
	case dvb.QAM32:
		bitrate = bitrate * 5 / 8
	case dvb.QAM64:
		bitrate = bitrate * 6 / 8
	case dvb.QAM128:
		bitrate = bitrate * 7 / 8
	case dvb.QAM256:
		bitrate = bitrate * 8 / 8
	default:
		bitrate = 0
	}
	switch tds.Guard {
	case dvb.Guard32:
		bitrate = bitrate * 32 / 33
	case dvb.Guard16:
		bitrate = bitrate * 16 / 17
	case dvb.Guard8:
		bitrate = bitrate * 8 / 9
	case dvb.Guard4:
		bitrate = bitrate * 4 / 5
	default:
		bitrate = 0
	}
	return bitrate * 188 / 204
}

func decodeCodeRate(b byte) dvb.CodeRate {
	switch b {
	case 0:
		return dvb.FEC12
	case 1:
		return dvb.FEC23
	case 2:
		return dvb.FEC34
	case 3:
		return dvb.FEC56
	case 4:
		return dvb.FEC78
	case 5, 6:
		return dvb.FECAuto

	}
	return dvb.FECNone
}

func ParseTerrestrialDeliverySystemDescriptor(d Descriptor) (tds TerrestrialDeliverySystemDescriptor, ok bool) {
	if d.Tag() != TerrestrialDeliverySystemTag {
		return
	}
	data := d.Data()
	if len(data) < 11 {
		return
	}
	tds.Freq = int64(decodeU32(data[0:4])) * 10
	switch data[4] >> 5 {
	case 0:
		tds.Bandwidth = 8e6
	case 1:
		tds.Bandwidth = 7e6
	case 2:
		tds.Bandwidth = 6e6
	case 3:
		tds.Bandwidth = 5e6
	default:
		return
	}
	tds.HighPrio = data[4]&0x10 != 0
	tds.TimeSlicing = data[4]&0x08 == 0
	tds.MPEFEC = data[4]&0x04 == 0
	switch data[5] >> 6 {
	case 0:
		tds.Constellation = dvb.QPSK
	case 1:
		tds.Constellation = dvb.QAM16
	case 2:
		tds.Constellation = dvb.QAM64
	default:
		return
	}
	tds.Hierarchy = dvb.Hierarchy((data[5] >> 3) & 0x07)
	if tds.Hierarchy > dvb.Hierarchy4 {
		return
	}
	tds.CodeRateHP = decodeCodeRate(data[5] & 0x07)
	if tds.CodeRateHP == dvb.FECAuto || tds.CodeRateHP == dvb.FECNone {
		return
	}
	tds.CodeRateLP = decodeCodeRate(data[6] >> 5)
	if tds.CodeRateLP == dvb.FECAuto {
		return
	}
	tds.Guard = dvb.Guard((data[6] >> 3) & 0x03)
	tds.TxMode = dvb.TxMode((data[6] >> 1) & 0x03)
	if tds.TxMode > dvb.TxMode8k {
		return
	}
	tds.OtherFreq = data[6]&0x01 != 0
	ok = true
	return
}

func encodeCodeRate(fec dvb.CodeRate) byte {
	switch fec {
	case dvb.FEC12:
		return 0
	case dvb.FEC23:
		return 1
	case dvb.FEC34:
		return 2
	case dvb.FEC56:
		return 3
	case dvb.FEC78:
		return 4
	}
	return 7
}

func (tds TerrestrialDeliverySystemDescriptor) MakeDescriptor() Descriptor {
	d := MakeDescriptor(TerrestrialDeliverySystemTag, 11)
	data := d.Data()
	encodeU32(data[0:4], uint32((tds.Freq+5)/10))
	var b byte
	switch tds.Bandwidth {
	case 8e6:
		b = 0 << 5
	case 7e6:
		b = 1 << 5
	case 6e6:
		b = 2 << 5
	case 5e6:
		b = 3 << 5
	default: // Unknown
		b = 7 << 5
	}
	if tds.HighPrio || tds.Hierarchy == dvb.HierarchyNone {
		b |= 0x10
	}
	if !tds.TimeSlicing {
		b |= 0x08
	}
	if !tds.MPEFEC {
		b |= 0x04
	}
	b |= 0x03 // Reserved
	data[4] = b
	switch tds.Constellation {
	case dvb.QPSK:
		b = 0 << 6
	case dvb.QAM16:
		b = 1 << 6
	case dvb.QAM64:
		b = 2 << 6
	default: // Unknown
		b = 3 << 6
	}
	b |= byte(tds.Hierarchy&0x07) << 3
	b |= encodeCodeRate(tds.CodeRateHP)
	data[5] = b
	b = encodeCodeRate(tds.CodeRateLP) << 5
	b |= byte(tds.Guard&0x03) << 3
	b |= byte(tds.TxMode&0x03) << 1
	if tds.OtherFreq {
		b |= 0x01
	}
	data[6] = b
	// Reserved
	data[7] = 0xff
	data[8] = 0xff
	data[9] = 0xff
	data[10] = 0xff
	return d
}

type CAS uint16

var casn = map[CAS]string{
	0x0100: "Mediaguard",

	0x0b00: "Conax",
	0x0b01: "Conax",
	0x0b02: "Conax",
	0x0b03: "Conax",
	0x0b04: "Conax",
	0x0b05: "Conax",
	0x0b06: "Conax",
	0x0b07: "Conax",
	0x0baa: "Conax",

	0x0d00: "Cryptoworks",
	0x0d02: "Cryptoworks",
	0x0d03: "Cryptoworks",
	0x0d05: "Cryptoworks",
	0x0d07: "Cryptoworks",
	0x0d20: "Cryptoworks",

	0x0500: "Viaccess",

	0x0602: "Irdeto",
	0x0604: "Irdeto",
	0x0606: "Irdeto",
	0x0608: "Irdeto",
	0x0622: "Irdeto",
	0x0626: "Irdeto",
	0x0664: "Irdeto",
	0x0614: "Irdeto",
	0x0692: "Irdeto",

	0x0911: "Videoguard",
	0x0919: "Videoguard",
	0x0960: "Videoguard",
	0x0961: "Videoguard",
	0x093b: "Videoguard",
	0x0963: "Videoguard",
	0x09AC: "Videoguard",
	0x0927: "Videoguard",

	0x0700: "DigiCipher2",

	0x0E00: "PowerVu",

	0x1702: "Nagravision",
	0x1722: "Nagravision",
	0x1762: "Nagravision",
	0x1800: "Nagravision",
	0x1801: "Nagravision",
	0x1810: "Nagravision",
	0x1830: "Nagravision",

	0x4AEA: "Cryptoguard",
}

func (cas CAS) String() string {
	name := casn[cas]
	s := strconv.FormatUint(uint64(cas), 16)
	if name == "" {
		return s
	}
	return name + "(" + s + ")"
}

type CADescriptor struct {
	Sys CAS
	Pid int16
}

func ParseCADescriptor(d Descriptor) (cad CADescriptor, ok bool) {
	if d.Tag() != CATag {
		return
	}
	data := d.Data()
	if len(data) < 4 {
		return
	}
	cad.Sys = CAS(decodeU16(data[0:2]))
	cad.Pid = int16(decodeU16(data[2:4]) & 0x1fff)
	ok = true
	return
}

type ISO639LangDescriptor []byte

func ParseISO639LangDescriptor(d Descriptor) (ld ISO639LangDescriptor, ok bool) {
	if d.Tag() != ISO639LangTag {
		return
	}
	return ISO639LangDescriptor(d.Data()), true
}

type AudioType byte

const (
	UndefinedAudio AudioType = iota
	CleanEffectsAudio
	HearingImpairedAudio
	VisualImpairedCommentaryAudio
)

var atn = []string{
	"undefined",
	"clean effects",
	"hearing impaired",
	"visual impaired commentary",
}

func (t AudioType) String() string {
	if t > VisualImpairedCommentaryAudio {
		return "reserved"
	}
	return atn[t]
}

type ISO639LangCode uint32

// Pop returns first (lc, at) pair from d. Remaining pairs are returned in rd.
// If there is no more pairs to read len(rd) == 0. If an error occurs rd = nil
func (d ISO639LangDescriptor) Pop() (lc ISO639LangCode, at AudioType, rd ISO639LangDescriptor) {
	if len(d) < 4 {
		return
	}
	lc = ISO639LangCode(decodeU24(d[0:3]))
	at = AudioType(d[3])
	rd = d[4:]
	return
}

type StreamIdentifierDescriptor byte

func ParseStreamIdentifierDescriptor(d Descriptor) (sid StreamIdentifierDescriptor, ok bool) {
	if d.Tag() != StreamIdentifierTag {
		return
	}
	data := d.Data()
	if len(data) != 1 {
		return
	}
	return StreamIdentifierDescriptor(data[0]), true
}

type LogicalChannelDescriptor []byte

func ParseLogicalChannelDescriptor(d Descriptor) (lcd LogicalChannelDescriptor, ok bool) {
	if d.Tag() != LogicalChannelTag {
		return
	}
	return LogicalChannelDescriptor(d.Data()), true
}

type LogicalChannelInfo struct {
	Sid     uint16 // Program Id
	LCN     int16  // Logical Channel Number
	Visible bool   // Should be visible on program list?
}

// Pop returns first LogicalChannelInfo from d. Remaining infos are returned in
// rd. If there is no more infos to read len(rd) == 0. If an error occurs
// rd = nil.
func (d LogicalChannelDescriptor) Pop() (lci LogicalChannelInfo, rd LogicalChannelDescriptor) {
	if len(d) < 4 {
		return
	}
	lci.Sid = decodeU16(d[0:2])
	lci.Visible = d[2]&0x80 != 0
	lci.LCN = int16(decodeU16(d[2:4]) & 0x03ff)
	rd = d[4:]
	return
}

func (lcd *LogicalChannelDescriptor) Append(lci LogicalChannelInfo) {
	var el [4]byte
	encodeU16(el[0:2], lci.Sid)
	encodeU16(el[2:4], uint16(lci.LCN)|0xfc00)
	if !lci.Visible {
		el[2] &^= 0x80
	}
	*lcd = append(*lcd, el[:]...)
}

func (lcd LogicalChannelDescriptor) MakeDescriptor() Descriptor {
	d := MakeDescriptor(LogicalChannelTag, len(lcd))
	copy(d.Data(), lcd)
	return d
}

func ParsePrivateDataSpecifier(d Descriptor) (pds uint32, ok bool) {
	if d.Tag() != PrivateDataSpecifierTag {
		return
	}
	data := d.Data()
	if len(data) != 4 {
		return
	}
	pds = decodeU32(data)
	return
}

func MakePrivateDataSpecifierDescriptor(pds uint32) Descriptor {
	d := MakeDescriptor(PrivateDataSpecifierTag, 4)
	encodeU32(d.Data(), pds)
	return d
}

type LocalTimeOffsetDescriptor []byte

func ParseLocalTimeOffsetDescriptor(d Descriptor) (tod LocalTimeOffsetDescriptor, ok bool) {
	if d.Tag() != LocalTimeOffsetTag {
		return
	}
	return LocalTimeOffsetDescriptor(d.Data()), true
}

type LocalTimeOffset struct {
	Country      [3]byte   // ISO 3166 country identifier.
	Region       byte      // Region number in the country (6-bit).
	Offset       int       // Seconds east of UTC.
	TimeOfChange time.Time // Date and time (UTC) when the next time change.
	NextOffset   int       // Next offset (seconds east of UTC).
}

// Pop returns first LocalTimeOffset from d. Remaining offsets are returned in
// rd. If there is no more data to read len(rd) == 0. If an error occurs
// rd = nil.
func (d LocalTimeOffsetDescriptor) Pop() (lto LocalTimeOffset, rd LocalTimeOffsetDescriptor) {
	if len(d) < 13 {
		rd = nil
		return
	}
	rd = d[13:]
	copy(lto.Country[:], d)
	lto.Region = d[3] >> 2
	neg := d[3]&1 != 0
	lto.Offset = decodeBCD(d[4])*3600 + decodeBCD(d[5])*60
	if neg {
		lto.Offset = -lto.Offset
	}
	var err error
	lto.TimeOfChange, err = decodeMJDUTC(d[6:11])
	if err != nil {
		rd = nil
		return
	}
	lto.NextOffset = decodeBCD(d[11])*3600 + decodeBCD(d[12])*60
	if neg {
		lto.NextOffset = -lto.NextOffset
	}
	return
}

// MakeLocalTimeOffsetDescriptor makes descriptor suitable for n time offsets.
func MakeLocalTimeOffsetDescriptor(n int) LocalTimeOffsetDescriptor {
	return make(LocalTimeOffsetDescriptor, n*13)
}

// Update updates n-th time ofset in d.
func (d LocalTimeOffsetDescriptor) Set(n int, lto LocalTimeOffset) {
	buf := d[n*13 : (n+1)*13]
	copy(buf[:3], lto.Country[:])
	buf[3] = lto.Region<<2 | 2
	if lto.Offset < 0 {
		buf[3] |= 1
	}
	min := (lto.Offset + 30) / 60
	buf[4] = encodeBCD(min / 60)
	buf[5] = encodeBCD(min % 60)
	encodeMJDUTC(buf[6:11], lto.TimeOfChange)
	min = (lto.NextOffset + 30) / 60
	buf[11] = encodeBCD(min / 60)
	buf[12] = encodeBCD(min % 60)
}

func (lto LocalTimeOffsetDescriptor) MakeDescriptor() Descriptor {
	d := MakeDescriptor(LocalTimeOffsetTag, len(lto))
	copy(d.Data(), lto)
	return d
}
