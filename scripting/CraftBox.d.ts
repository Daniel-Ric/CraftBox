declare const console: {
    log(data: any);
};

function displayChatMessage(msg: string);
function setIngameMap(enabled: boolean);





declare type EventNames = 'PlayerAdd' | 'EntityAdd' | 'EntityUpdate' | 'EntityPropertyChange' | 'ChunkAdd' | 'BlockUpdate' | 'SpawnParticle' | 'Packet';

declare type PlayerInfo = {
    UUID: string;
    Username: string;
    XUID: string;
    IsNPC: boolean;
    HasSkin: boolean;
    RuntimeID: number;
    UniqueID: number;
    Position: [number, number, number];
};

declare type PlayerAddCallback = (player: PlayerInfo, metadata: EntityMetadata, time: number) => boolean | undefined;

declare type EntityUpdate = {
	Entity: Entity;
	Metadata: EntityMetadata;
	PreviousPosition?: [number, number, number];
	ChangedProperties?: {[k: string]: any};
}









declare type EntityAddCallback = (entity: Entity, metadata: EntityMetadata, time: number, isNew: boolean) => boolean;








declare type EntityUpdateCallback = (update: EntityUpdate, time: number) => void;

declare type EntityPropertyChangeCallback = (entity: Entity, name: string, previous: number, newValue: number) => boolean;







declare type ChunkAddCallback = (pos: [number, number, number], time: number) => void;









declare type BlockUpdateCallback = (name: string, properties: {[k: string]: any}, pos: [number, number, number], time: number) => void;











declare type SpawnParticleCallback = (name: string, pos: [number, number, number], time: number) => void;










declare type PacketCallback = (name: string, packet: any, toServer: boolean, time: number) => void;


declare type ChunkDataCallback = (pos: [number, number]) => void;

declare const events: {
    register(name: 'PlayerAdd', callback: PlayerAddCallback): void;
























    register(name: 'EntityAdd', callback: EntityAddCallback): void;
























    register(name: 'EntityUpdate', callback: EntityUpdateCallback): void;

	register(name: 'EntityPropertyChange', callback: EntityPropertyChangeCallback): void;























    register(name: 'ChunkAdd', callback: ChunkAddCallback): void;























    register(name: 'BlockUpdate', callback: BlockUpdateCallback): void;

























    register(name: 'SpawnParticle', callback: SpawnParticleCallback): void;


























    register(name: 'Packet', callback: PacketCallback): void;

	register(name: "ChunkData", callback: ChunkDataCallback): void;
};

interface FileWriter {
	write(data: string): void;
	close(): void;
}

interface FileReader {
	read(): string;
	close(): void;
}

interface FileSystem {
	create(name: string): FileWriter;
}

interface BlockState {
	name: string;
	state: {[k: string]: any};
}

interface BlockEntity {
	x: number;
	y: number;
	z: number;
	id: string;
	data: {[k: string]: any};
};

interface Chunk {
	getBlockAt(x: number, y: number, z: number): BlockState;
	getHighestBlockAt(x: number, z: number): number;
	getBlockEntities(): BlockEntity[];
};

interface Chunks {
	get(x: number, z: number): Chunk|null;
}

declare const fs: FileSystem;

declare const chunks: Chunks;




declare type Entity = {



    RuntimeID: number



    UniqueID: number



    EntityType: string



    Position: [number, number, number];



    Pitch: number;



    Yaw: number;



    Velocity: [number, number, number];




	Properties: {[k: string]: any}
}





declare type ItemInstance = {



    StackNetworkID: number;



    Stack: ItemStack;
}





declare type ItemStack = {



    NetworkID: number;



    MetadataValue: number;



    BlockRuntimeID: number;



    Count: number;



    NBTData: {[k: string]: any};



    CanBePlacedOn: Array<string>;



    CanBreak: Array<string>;



    HasNetworkID: boolean;
}


declare type EntityMetadata = {
    [k: EntityDataKey]: any;
	Flags: {[k: EntityDataFlag]: bool}
};


declare type EntityDataKey =
	"Flags" |
	"StructuralIntegrity" |
	"Variant" |
	"ColorIndex" |
	"Name" |
	"Owner" |
	"Target" |
	"AirSupply" |
	"EffectColor" |
	"EffectAmbience" |
	"JumpDuration" |
	"Hurt" |
	"HurtDirection" |
	"RowTimeLeft" |
	"RowTimeRight" |
	"Value" |
	"DisplayTileRuntimeID" |
	"DisplayOffset" |
	"CustomDisplay" |
	"Swell" |
	"OldSwell" |
	"SwellDirection" |
	"ChargeAmount" |
	"CarryBlockRuntimeID" |
	"ClientEvent" |
	"UsingItem" |
	"PlayerFlags" |
	"PlayerIndex" |
	"BedPosition" |
	"PowerX" |
	"PowerY" |
	"PowerZ" |
	"AuxPower" |
	"FishX" |
	"FishZ" |
	"FishAngle" |
	"AuxValueData" |
	"LeashHolder" |
	"Scale" |
	"HasNPC" |
	"NPCData" |
	"Actions" |
	"AirSupplyMax" |
	"MarkVariant" |
	"ContainerType" |
	"ContainerSize" |
	"ContainerStrengthModifier" |
	"BlockTarget" |
	"Inventory" |
	"TargetA" |
	"TargetB" |
	"TargetC" |
	"AerialAttack" |
	"Width" |
	"Height" |
	"FuseTime" |
	"SeatOffset" |
	"SeatLockPassengerRotation" |
	"SeatLockPassengerRotationDegrees" |
	"SeatRotationOffset" |
	"SeatRotationOffstDegrees" |
	"DataRadius" |
	"DataWaiting" |
	"DataParticle" |
	"PeekID" |
	"AttachFace" |
	"Attached" |
	"AttachedPosition" |
	"TradeTarget" |
	"Career" |
	"HasCommandBlock" |
	"CommandName" |
	"LastCommandOutput" |
	"TrackCommandOutput" |
	"ControllingSeatIndex" |
	"Strength" |
	"StrengthMax" |
	"DataSpellCastingColor" |
	"DataLifetimeTicks" |
	"PoseIndex" |
	"DataTickOffset" |
	"AlwaysShowNameTag" |
	"ColorTwoIndex" |
	"NameAuthor" |
	"Score" |
	"BalloonAnchor" |
	"PuffedState" |
	"BubbleTime" |
	"Agent" |
	"SittingAmount" |
	"SittingAmountPrevious" |
	"EatingCounter" |
	"FlagsTwo" |
	"LayingAmount" |
	"LayingAmountPrevious" |
	"DataDuration" |
	"DataSpawnTime" |
	"DataChangeRate" |
	"DataChangeOnPickup" |
	"DataPickupCount" |
	"InteractText" |
	"TradeTier" |
	"MaxTradeTier" |
	"TradeExperience" |
	"SkinID" |
	"SpawningFrames" |
	"CommandBlockTickDelay" |
	"CommandBlockExecuteOnFirstTick" |
	"AmbientSoundInterval" |
	"AmbientSoundIntervalRange" |
	"AmbientSoundEventName" |
	"FallDamageMultiplier" |
	"NameRawText" |
	"CanRideTarget" |
	"LowTierCuredTradeDiscount" |
	"HighTierCuredTradeDiscount" |
	"NearbyCuredTradeDiscount" |
	"NearbyCuredDiscountTimeStamp" |
	"HitBox" |
	"IsBuoyant" |
	"FreezingEffectStrength" |
	"BuoyancyData" |
	"GoatHornCount" |
	"BaseRuntimeID" |
	"MovementSoundDistanceOffset" |
	"HeartbeatIntervalTicks" |
	"HeartbeatSoundEvent" |
	"PlayerLastDeathPosition" |
	"PlayerLastDeathDimension" |
	"PlayerHasDied" |
	"CollisionBox" |
	"VisibleMobEffects";


declare type EntityDataFlag =
    "OnFire" |
	"Sneaking" |
	"Riding" |
	"Sprinting" |
	"UsingItem" |
	"Invisible" |
	"Tempted" |
	"InLove" |
	"Saddled" |
	"Powered" |
	"Ignited" |
	"Baby" |
	"Converting" |
	"Critical" |
	"ShowName" |
	"AlwaysShowName" |
	"NoAI" |
	"Silent" |
	"WallClimbing" |
	"Climb" |
	"Swim" |
	"Fly" |
	"Walk" |
	"Resting" |
	"Sitting" |
	"Angry" |
	"Interested" |
	"Charged" |
	"Tamed" |
	"Orphaned" |
	"Leashed" |
	"Sheared" |
	"Gliding" |
	"Elder" |
	"Moving" |
	"Breathing" |
	"Chested" |
	"Stackable" |
	"ShowBottom" |
	"Standing" |
	"Shaking" |
	"Idling" |
	"Casting" |
	"Charging" |
	"KeyboardControlled" |
	"PowerJump" |
	"Dash" |
	"Lingering" |
	"HasCollision" |
	"HasGravity" |
	"FireImmune" |
	"Dancing" |
	"Enchanted" |
	"ReturnTrident" |
	"ContainerPrivate" |
	"Transforming" |
	"DamageNearbyMobs" |
	"Swimming" |
	"Bribed" |
	"Pregnant" |
	"LayingEgg" |
	"PassengerCanPick" |
	"TransitionSitting" |
	"Eating" |
	"LayingDown" |
	"Sneezing" |
	"Trusting" |
	"Rolling" |
	"Scared" |
	"InScaffolding" |
	"OverScaffolding" |
	"DescendThroughBlock" |
	"Blocking" |
	"TransitionBlocking" |
	"BlockedUsingShield" |
	"BlockedUsingDamagedShield" |
	"Sleeping" |
	"WantsToWake" |
	"TradeInterest" |
	"DoorBreaker" |
	"BreakingObstruction" |
	"DoorOpener" |
	"Captain" |
	"Stunned" |
	"Roaring" |
	"DelayedAttack" |
	"AvoidingMobs" |
	"AvoidingBlock" |
	"FacingTargetToRangeAttack" |
	"HiddenWhenInvisible" |
	"InUI" |
	"Stalking" |
	"Emoting" |
	"Celebrating" |
	"Admiring" |
	"CelebratingSpecial" |
	"OutOfControl" |
	"RamAttack" |
	"PlayingDead" |
	"InAscendingBlock" |
	"OverDescendingBlock" |
	"Croaking" |
	"DigestMob" |
	"JumpGoal" |
	"Emerging" |
	"Sniffing" |
	"Digging" |
	"SonicBoom" |
	"HasDashTimeout" |
	"PushTowardsClosestSpace" |
	"Scenting" |
	"Rising" |
	"FeelingHappy" |
	"Searching" |
	"Crawling" |
	"TimerFlag1" |
	"TimerFlag2" |
	"TimerFlag3";
