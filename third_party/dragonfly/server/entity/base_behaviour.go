package entity

type BaseBehaviour struct {
	portalTravel *PortalTravelComputer
}

func NewBaseBehaviour() BaseBehaviour {
	return BaseBehaviour{portalTravel: NewPortalTravelComputer()}
}

func (b *BaseBehaviour) PortalTravelComputer() *PortalTravelComputer {
	if b.portalTravel == nil {
		b.portalTravel = NewPortalTravelComputer()
	}
	return b.portalTravel
}
