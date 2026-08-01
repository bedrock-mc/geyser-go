package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
	"github.com/sandertv/gophertunnel/minecraft/protocol/packet"
)

func main() {
	address := flag.String("address", "127.0.0.1:19132", "Bedrock/RakNet bridge address")
	username := flag.String("username", "GeyserProbe", "offline-mode Bedrock username")
	readFor := flag.Duration("read-for", 0, "after spawn, read and print Bedrock packets for this duration")
	sendAuthInput := flag.Bool("send-auth-input", false, "send one PlayerAuthInput movement packet after spawn")
	sendBlockAction := flag.Bool("send-block-action", false, "include one block-break action in the auth-input packet")
	sendItemUse := flag.Bool("send-item-use", false, "include one click-air item interaction in the auth-input packet")
	sendStateActions := flag.Bool("send-state-actions", false, "include sprint, sneak, and glide state edges in the auth-input packet")
	sendHeldSlot := flag.Bool("send-held-slot", false, "send one held-hotbar-slot update after spawn")
	sendArm := flag.Bool("send-arm", false, "send one arm-swing animation after spawn")
	sendEntityInteract := flag.Bool("send-entity-interact", false, "send one self entity interaction after spawn")
	sendWindowTake := flag.Bool("send-window-take", false, "after the Paper WindowTest menu opens, take slot 0 to the cursor")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dialer := minecraft.Dialer{
		IdentityData: login.IdentityData{
			Identity: uuid.New().String(), DisplayName: *username,
		},
		EnableBatchReading: true,
	}
	conn, err := dialer.DialContext(ctx, "raknet", *address)
	if err != nil {
		panic(err)
	}
	defer conn.Close()
	if err := conn.DoSpawnContext(ctx); err != nil {
		panic(err)
	}
	if *sendAuthInput {
		input := protocol.NewBitset(packet.PlayerAuthInputBitsetSize)
		input.Set(packet.InputFlagVerticalCollision)
		auth := &packet.PlayerAuthInput{
			Position:  conn.GameData().PlayerPosition,
			Yaw:       conn.GameData().Yaw,
			Pitch:     conn.GameData().Pitch,
			HeadYaw:   conn.GameData().Yaw,
			InputData: input,
		}
		if *sendBlockAction {
			input.Set(packet.InputFlagPerformBlockActions)
			auth.BlockActions = []protocol.PlayerBlockAction{{
				Action:   protocol.PlayerActionStartBreak,
				BlockPos: protocol.BlockPos{int32(auth.Position.X()), int32(auth.Position.Y()) - 2, int32(auth.Position.Z())},
				Face:     1,
			}}
		}
		if *sendItemUse {
			input.Set(packet.InputFlagPerformItemInteraction)
			auth.InteractYaw = auth.Yaw
			auth.InteractPitch = auth.Pitch
			auth.ItemInteractionData = protocol.UseItemTransactionData{
				ActionType: protocol.UseItemActionClickAir,
				Position:   auth.Position,
			}
		}
		if *sendStateActions {
			input.Set(packet.InputFlagStartSprinting)
			input.Set(packet.InputFlagStartSneaking)
			input.Set(packet.InputFlagStartGliding)
		}
		if err := conn.WritePacket(auth); err != nil {
			panic(err)
		}
	}
	if *sendHeldSlot {
		if err := conn.WritePacket(&packet.MobEquipment{
			EntityRuntimeID: conn.GameData().EntityRuntimeID,
			InventorySlot:   0,
			HotBarSlot:      0,
		}); err != nil {
			panic(err)
		}
	}
	if *sendArm {
		if err := conn.WritePacket(&packet.Animate{
			ActionType:      packet.AnimateActionSwingArm,
			EntityRuntimeID: conn.GameData().EntityRuntimeID,
			SwingSource:     packet.AnimateSwingSourceAttack,
		}); err != nil {
			panic(err)
		}
	}
	if *sendEntityInteract {
		if err := conn.WritePacket(&packet.Interact{
			ActionType:            1,
			TargetEntityRuntimeID: conn.GameData().EntityRuntimeID,
		}); err != nil {
			panic(err)
		}
	}
	if *sendWindowTake {
		go func() {
			time.Sleep(2 * time.Second)
			action := &protocol.TakeStackRequestAction{}
			action.Count = 1
			action.Source = protocol.StackRequestSlotInfo{
				Container:      protocol.FullContainerName{ContainerID: protocol.ContainerLevelEntity},
				Slot:           0,
				StackNetworkID: -1,
			}
			action.Destination = protocol.StackRequestSlotInfo{
				Container:      protocol.FullContainerName{ContainerID: protocol.ContainerCursor},
				Slot:           0,
				StackNetworkID: -1,
			}
			_ = conn.WritePacket(&packet.ItemStackRequest{Requests: []protocol.ItemStackRequest{{
				RequestID:   -41,
				Actions:     []protocol.StackRequestAction{action},
				FilterCause: -1,
			}}})
		}()
	}
	_ = conn.WritePacket(&packet.Text{TextType: packet.TextTypeChat, Message: "geyser-go probe"})
	fmt.Printf("Bedrock spawn succeeded: protocol=%d version=%s items=%d world=%q entity=%d\n", conn.Proto().ID(), conn.Proto().Ver(), len(conn.GameData().Items), conn.GameData().WorldName, conn.GameData().EntityRuntimeID)
	if *readFor <= 0 {
		return
	}
	deadline := time.Now().Add(*readFor)
	if err := conn.SetReadDeadline(deadline); err != nil {
		panic(err)
	}
	for {
		batch, err := conn.ReadBatch()
		if err != nil {
			fmt.Printf("Bedrock read finished: %v\n", err)
			return
		}
		for _, pk := range batch {
			switch pk := pk.(type) {
			case *packet.Text:
				fmt.Printf("Bedrock text: type=%d source=%q message=%q\n", pk.TextType, pk.SourceName, pk.Message)
			case *packet.ItemStackResponse:
				for _, response := range pk.Responses {
					fmt.Printf("Bedrock item stack response: request=%d status=%d containers=%d\n", response.RequestID, response.Status, len(response.ContainerInfo))
					for _, container := range response.ContainerInfo {
						fmt.Printf("Bedrock item stack container: id=%d slots=%d\n", container.Container.ContainerID, len(container.SlotInfo))
						for _, slot := range container.SlotInfo {
							fmt.Printf("Bedrock item stack slot: id=%d slot=%d hotbar=%d count=%d network=%d\n", container.Container.ContainerID, slot.Slot, slot.HotbarSlot, slot.Count, slot.StackNetworkID)
						}
					}
				}
			case *packet.InventorySlot:
				containerID := byte(0)
				if container, ok := pk.Container.Value(); ok {
					containerID = container.ContainerID
				}
				fmt.Printf("Bedrock inventory slot: window=%d container=%d slot=%d count=%d network=%d\n", pk.WindowID, containerID, pk.Slot, pk.NewItem.Stack.Count, pk.NewItem.StackNetworkID)
			case *packet.SetDifficulty:
				fmt.Printf("Bedrock difficulty: %d\n", pk.Difficulty)
			case *packet.SetPlayerGameType:
				fmt.Printf("Bedrock game type: %d\n", pk.GameType)
			case *packet.SetTitle:
				fmt.Printf("Bedrock title: action=%d text=%q fadeIn=%d stay=%d fadeOut=%d\n", pk.ActionType, pk.Text, pk.FadeInDuration, pk.RemainDuration, pk.FadeOutDuration)
			case *packet.ChangeDimension:
				fmt.Printf("Bedrock dimension change: dimension=%d position=%v respawn=%t\n", pk.Dimension, pk.Position, pk.Respawn)
			case *packet.Respawn:
				fmt.Printf("Bedrock respawn: state=%d position=%v entity=%d\n", pk.State, pk.Position, pk.EntityRuntimeID)
			case *packet.BossEvent:
				fmt.Printf("Bedrock boss bar: event=%d entity=%d player=%d title=%q health=%.2f color=%d overlay=%d\n", pk.EventType, pk.BossEntityUniqueID, pk.PlayerUniqueID, pk.BossBarTitle, pk.HealthPercentage, pk.Colour, pk.Overlay)
			case *packet.PlaySound:
				fmt.Printf("Bedrock sound: name=%q position=%v volume=%.2f pitch=%.2f\n", pk.SoundName, pk.Position, pk.Volume, pk.Pitch)
			case *packet.StopSound:
				fmt.Printf("Bedrock stop sound: name=%q all=%t\n", pk.SoundName, pk.StopAll)
			case *packet.SetDisplayObjective:
				fmt.Printf("Bedrock scoreboard display: slot=%q objective=%q title=%q criteria=%q order=%d\n", pk.DisplaySlot, pk.ObjectiveName, pk.DisplayName, pk.CriteriaName, pk.SortOrder)
			case *packet.SetScore:
				fmt.Printf("Bedrock scoreboard scores: action=%d entries=%d\n", pk.ActionType, len(pk.Entries))
				for _, entry := range pk.Entries {
					fmt.Printf("Bedrock scoreboard entry: objective=%q id=%d score=%d identity=%d display=%q\n", entry.ObjectiveName, entry.EntryID, entry.Score, entry.IdentityType, entry.DisplayName)
				}
			case *packet.RemoveObjective:
				fmt.Printf("Bedrock scoreboard remove: objective=%q\n", pk.ObjectiveName)
			case *packet.SetActorLink:
				fmt.Printf("Bedrock actor link: ridden=%d rider=%d type=%d immediate=%t riderInitiated=%t\n", pk.EntityLink.RiddenEntityUniqueID, pk.EntityLink.RiderEntityUniqueID, pk.EntityLink.Type, pk.EntityLink.Immediate, pk.EntityLink.RiderInitiated)
			default:
				fmt.Printf("Bedrock packet: %T\n", pk)
			}
		}
	}
}
