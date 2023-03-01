package connectivity

// Region represents ECS region
type Region string

// Constants of region definition
const (
	GUANGZHOU = Region("GUANGZHOU")
	BEIJING   = Region("BEIJING")
	HONGKONG  = Region("HONGKONG")
	SHANGHAI  = Region("SHANGHAI")
)

var ValidRegions = []Region{
	GUANGZHOU, BEIJING, HONGKONG, SHANGHAI,
}

var Ks3SseSupportedRegions = []Region{GUANGZHOU, BEIJING, SHANGHAI, HONGKONG}
