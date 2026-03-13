package voresky

import "encoding/json"

// ─── Interaction: proposal ──────────────────────────────────────────────────

// InteractionProposalPayload is the payload for interaction_proposal notifications.
// Sent to the recipient when someone proposes an interaction.
type InteractionProposalPayload struct {
	ProposalID               string `json:"proposalId"`
	PredatorCharacterName    string `json:"predatorCharacterName"`
	PreyCharacterName        string `json:"preyCharacterName"`
	PathName                 string `json:"pathName"`
	EstimatedDurationSeconds int    `json:"estimatedDurationSeconds"`
	InitiatedBy              string `json:"initiatedBy"` // "predator" or "prey"
	Universe                 string `json:"universe,omitempty"`
	IsDoOver                 bool   `json:"isDoOver,omitempty"`
	ReplacesInteractionID    string `json:"replacesInteractionId,omitempty"`
}

// ─── Interaction: base (accepted, rejected, vip_caught, escaped, released) ──

// InteractionBasePayload is the shared payload shape for accepted, rejected,
// vip_caught, escaped, and released notifications.
// Fields that are only present on some types use omitempty.
type InteractionBasePayload struct {
	InteractionID         string `json:"interactionId,omitempty"`
	PredatorCharacterName string `json:"predatorCharacterName"`
	PreyCharacterName     string `json:"preyCharacterName"`
	PathName              string `json:"pathName,omitempty"`
	Universe              string `json:"universe,omitempty"`
	IsDoOver              bool   `json:"isDoOver,omitempty"`
	Message               string `json:"message,omitempty"` // rejection message
}

// ─── Interaction: counter-proposal ──────────────────────────────────────────

// InteractionCounterPayload is the payload for interaction_counter_proposal notifications.
type InteractionCounterPayload struct {
	ProposalID            string `json:"proposalId"`
	PredatorCharacterName string `json:"predatorCharacterName"`
	PreyCharacterName     string `json:"preyCharacterName"`
	PathName              string `json:"pathName"`
	CounterCount          int    `json:"counterCount"`
	ChangedRespawnTime    bool   `json:"changedRespawnTime"`
	ChangedBranches       bool   `json:"changedBranches"`
	ChangedUniverse       bool   `json:"changedUniverse"`
	Universe              string `json:"universe,omitempty"`
	Message               string `json:"message,omitempty"`
}

// ─── Interaction: node changed ───────────────────────────────────────────────

// InteractionNodePayload is the payload for interaction_node_changed notifications.
type InteractionNodePayload struct {
	InteractionID   string `json:"interactionId"`
	NewNodeVerbPast string `json:"newNodeVerbPast,omitempty"`
	Universe        string `json:"universe,omitempty"`
}

// ─── Interaction: prey retreated ─────────────────────────────────────────────

// InteractionRetreatPayload is the payload for interaction_prey_retreated notifications.
type InteractionRetreatPayload struct {
	InteractionID         string `json:"interactionId"`
	PredatorCharacterName string `json:"predatorCharacterName"`
	PreyCharacterName     string `json:"preyCharacterName"`
	RetreatedToNode       string `json:"retreatedToNode"`
	Universe              string `json:"universe,omitempty"`
}

// ─── Interaction: respawning ─────────────────────────────────────────────────

// InteractionRespawnPayload is the payload for interaction_respawning notifications.
type InteractionRespawnPayload struct {
	InteractionID          string `json:"interactionId"`
	PredatorCharacterName  string `json:"predatorCharacterName"`
	PreyCharacterName      string `json:"preyCharacterName"`
	PathName               string `json:"pathName"`
	RespawnDurationSeconds int    `json:"respawnDurationSeconds"`
	Universe               string `json:"universe,omitempty"`
}

// ─── Interaction: completed ──────────────────────────────────────────────────

// InteractionCompletedPayload is the payload for interaction_completed notifications.
type InteractionCompletedPayload struct {
	InteractionID         string `json:"interactionId"`
	PredatorCharacterName string `json:"predatorCharacterName"`
	PreyCharacterName     string `json:"preyCharacterName"`
	PathName              string `json:"pathName"`
	HasPointOfNoReturn    bool   `json:"hasPointOfNoReturn"`
	FinalNodeName         string `json:"finalNodeName"`
	VerbPast              string `json:"verbPast,omitempty"`
	Universe              string `json:"universe,omitempty"`
}

// ─── Interaction: safeword ───────────────────────────────────────────────────

// InteractionSafewordPayload is the payload for interaction_safeword notifications.
// Source character is nil on the parent notification to protect who invoked the safeword.
type InteractionSafewordPayload struct {
	PathName string `json:"pathName"`
	Universe string `json:"universe,omitempty"`
}

// ─── Housing ─────────────────────────────────────────────────────────────────

// HousingPayload is shared by all housing_* notification types:
// housing_invite, housing_request, housing_join, housing_leave, housing_kick,
// housing_invite_accepted, housing_invite_rejected,
// housing_request_accepted, housing_request_rejected.
type HousingPayload struct {
	HouseName           string `json:"houseName,omitempty"`
	OwnerCharacterName  string `json:"ownerCharacterName,omitempty"`
	MemberCharacterName string `json:"memberCharacterName,omitempty"`
	Universe            string `json:"universe,omitempty"`
}

// ─── Collar / petplay ────────────────────────────────────────────────────────

// CollarPayload is shared by all collar_* notification types:
// collar_offer, collar_request, collar_accepted, collar_rejected, collar_broken,
// collar_lock_request, collar_locked, collar_unlocked.
type CollarPayload struct {
	OwnerCharacterName string `json:"ownerCharacterName,omitempty"`
	PetCharacterName   string `json:"petCharacterName,omitempty"`
	Universe           string `json:"universe,omitempty"`
}

// ─── Poke / stalk ────────────────────────────────────────────────────────────

// PokePayload is the payload for poke notifications.
type PokePayload struct {
	Universe string `json:"universe,omitempty"`
}

// StalkPayload is the payload for stalk and stalk_target_available notifications.
type StalkPayload struct {
	Universe string `json:"universe,omitempty"`
}

// ─── ParsePayload ────────────────────────────────────────────────────────────

// ParsePayload unmarshals the raw JSON payload of a notification into a typed struct.
// The caller may type-assert the returned value to the appropriate *XxxPayload type.
// Returns (nil, nil) for unknown notification types or a nil/empty raw payload —
// this is a deliberate graceful fallback for forward compatibility.
func ParsePayload(notifType NotificationType, raw json.RawMessage) (interface{}, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}

	var dest interface{}
	switch notifType {
	case NotifInteractionProposal:
		dest = &InteractionProposalPayload{}
	case NotifInteractionAccepted, NotifInteractionRejected,
		NotifInteractionVipCaught, NotifInteractionEscaped, NotifInteractionReleased:
		dest = &InteractionBasePayload{}
	case NotifInteractionCounter:
		dest = &InteractionCounterPayload{}
	case NotifInteractionNodeChanged:
		dest = &InteractionNodePayload{}
	case NotifInteractionRetreated:
		dest = &InteractionRetreatPayload{}
	case NotifInteractionRespawning:
		dest = &InteractionRespawnPayload{}
	case NotifInteractionCompleted:
		dest = &InteractionCompletedPayload{}
	case NotifInteractionSafeword:
		dest = &InteractionSafewordPayload{}
	case NotifHousingInvite, NotifHousingRequest, NotifHousingJoin,
		NotifHousingLeave, NotifHousingKick,
		NotifHousingInviteAccepted, NotifHousingInviteRejected,
		NotifHousingRequestAccepted, NotifHousingRequestRejected:
		dest = &HousingPayload{}
	case NotifCollarOffer, NotifCollarRequest, NotifCollarAccepted,
		NotifCollarRejected, NotifCollarBroken,
		NotifCollarLockRequest, NotifCollarLocked, NotifCollarUnlocked:
		dest = &CollarPayload{}
	case NotifPoke:
		dest = &PokePayload{}
	case NotifStalk, NotifStalkTargetAvailable:
		dest = &StalkPayload{}
	default:
		return nil, nil
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		return nil, err
	}
	return dest, nil
}
