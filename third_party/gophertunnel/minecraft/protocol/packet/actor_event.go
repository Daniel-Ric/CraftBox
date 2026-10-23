package packet

import (
	"github.com/go-gl/mathgl/mgl32"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
)

const (
	ActorEventJump = iota + 1
	ActorEventHurt
	ActorEventDeath
	ActorEventStartAttacking
	ActorEventStopAttacking
	ActorEventTamingFailed
	ActorEventTamingSucceeded
	ActorEventShakeWetness
	ActorEventUseItem
	ActorEventEatGrass
	ActorEventFishhookBubble
	ActorEventFishhookFishPosition
	ActorEventFishhookHookTime
	ActorEventFishhookTease
	ActorEventSquidFleeing
	ActorEventZombieConverting
	ActorEventPlayAmbient
	ActorEventSpawnAlive
	ActorEventStartOfferFlower
	ActorEventStopOfferFlower
	ActorEventLoveHearts
	ActorEventVillagerAngry
	ActorEventVillagerHappy
	ActorEventWitchHatMagic
	ActorEventFireworksExplode
	ActorEventInLoveHearts
	ActorEventSilverfishMergeAnimation
	ActorEventGuardianAttackSound
	ActorEventDrinkPotion
	ActorEventThrowPotion
	ActorEventCartWithPrimeTNT
	ActorEventPrimeCreeper
	ActorEventAirSupply
	ActorEventAddPlayerLevels
	ActorEventGuardianMiningFatigue
	ActorEventAgentSwingArm
	ActorEventDragonStartDeathAnim
	ActorEventGroundDust
	ActorEventShake
)

const (
	ActorEventFeed = iota + 57
	_
	_
	ActorEventBabyEat
	ActorEventInstantDeath
	ActorEventNotifyTrade
	ActorEventLeashDestroyed
	ActorEventCaravanUpdated
	ActorEventTalismanActivate
	ActorEventUpdateStructureFeature
	ActorEventPlayerSpawnedMob
	ActorEventPuke
	ActorEventUpdateStackSize
	ActorEventStartSwimming
	ActorEventBalloonPop
	ActorEventTreasureHunt
	ActorEventSummonAgent
	ActorEventFinishedChargingItem
	ActorEventLandedOnGround
	ActorEventActorGrowUp
	ActorEventVibrationDetected
	ActorEventDrinkMilk
	ActorEventWetnessStop
	ActorEventKineticDamageDealt
	ActorEventHurtWithoutReceivingDamage
)

type ActorEvent struct {
	EntityRuntimeID uint64

	EventType byte

	EventData int32

	FireAtPosition protocol.Optional[mgl32.Vec3]
}

func (*ActorEvent) ID() uint32 {
	return IDActorEvent
}

func (pk *ActorEvent) Marshal(io protocol.IO) {
	io.ActorRuntimeID(&pk.EntityRuntimeID)
	io.Uint8(&pk.EventType)
	io.Varint32(&pk.EventData)
	protocol.OptionalFunc(io, &pk.FireAtPosition, io.Vec3)
}
