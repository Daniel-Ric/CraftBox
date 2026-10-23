package world

type Structure interface {
	Dimensions() [3]int

	At(x, y, z int, blockAt func(x, y, z int) Block) (Block, Liquid)
}
