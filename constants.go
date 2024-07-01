package bottledlightning

type xMetaValue byte

const (
	XMetaValue0 xMetaValue = iota
	XMetaValue1
	XMetaValue2
	XMetaValue3
	XMetaValue4
	XMetaValue5
	XMetaValue6
	XMetaValue7
	XMetaValue8
	XMetaValue9
	XMetaValueA
	XMetaValueB
	XMetaValueC
	XMetaValueD
	XMetaValueE
	XMetaValueF
)

const (
	lmdbMaxKeyLen = 511
	maxUintLen32  = 4
	offsetC       = 13
	offsetM       = 9
	offsetX       = 14
)
