package entity

type HealthManager struct {
	health float64
	max    float64
}

func NewHealthManager(health, max float64) *HealthManager {
	if health > max {
		health = max
	}
	return &HealthManager{health: health, max: max}
}

func (m *HealthManager) Health() float64 {
	return m.health
}

func (m *HealthManager) AddHealth(health float64) {
	m.health = max(min(m.health+health, m.max), 0)
}

func (m *HealthManager) MaxHealth() float64 {
	return m.max
}

func (m *HealthManager) SetMaxHealth(max float64) {
	if max <= 0 {
		max = 1
	}
	m.max = max
	m.health = min(m.health, max)
}
