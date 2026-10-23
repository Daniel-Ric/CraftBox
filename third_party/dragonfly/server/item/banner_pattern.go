package item

type BannerPattern struct {
	Type BannerPatternType
}

func (b BannerPattern) MaxCount() int {
	return 1
}

func (b BannerPattern) EncodeItem() (name string, meta int16) {
	return "minecraft:" + b.Type.String() + "_banner_pattern", 0
}
