package scene

type BlockRun struct {
	Name string `json:"name"`
	End  int    `json:"end"`
}

func (p *Part) recordBlock(name string) {
	end := len(p.Positions) / 3
	if last := len(p.BlockRuns) - 1; last >= 0 && p.BlockRuns[last].Name == name {
		p.BlockRuns[last].End = end
		return
	}
	p.BlockRuns = append(p.BlockRuns, BlockRun{Name: name, End: end})
}
