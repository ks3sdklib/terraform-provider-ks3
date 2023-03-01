package connectivity

// Region represents ECS region
type Region string

// Constants of region definition
const (
	Qingdao  = Region("cn-qingdao")
	Beijing  = Region("BEJING")
	Hongkong = Region("hongkong")
	Shanghai = Region("shanghai")
)

var ValidRegions = []Region{
	Qingdao, Beijing, Hongkong, Shanghai,
}

var Ks3SseSupportedRegions = []Region{Qingdao, Beijing, Shanghai, Hongkong}
