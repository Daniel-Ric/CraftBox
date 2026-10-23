package entity

type (
	FoodHealingSource struct{ QuickRegeneration bool }
)

func (FoodHealingSource) HealingSource() {}
