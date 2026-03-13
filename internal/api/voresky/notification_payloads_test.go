package voresky_test

import (
	"encoding/json"
	"testing"

	"github.com/dotBeeps/noms/internal/api/voresky"
)

func TestParsePayload_Proposal(t *testing.T) {
	raw := json.RawMessage(`{
		"proposalId": "prop-123",
		"predatorCharacterName": "Sable",
		"preyCharacterName": "Pip",
		"pathName": "Forest Hunt",
		"estimatedDurationSeconds": 3600,
		"initiatedBy": "predator",
		"universe": "Verdant"
	}`)

	result, err := voresky.ParsePayload(voresky.NotifInteractionProposal, raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	payload, ok := result.(*voresky.InteractionProposalPayload)
	if !ok {
		t.Fatalf("expected *InteractionProposalPayload, got %T", result)
	}
	if payload.ProposalID != "prop-123" {
		t.Errorf("ProposalID: got %q, want %q", payload.ProposalID, "prop-123")
	}
	if payload.PredatorCharacterName != "Sable" {
		t.Errorf("PredatorCharacterName: got %q, want %q", payload.PredatorCharacterName, "Sable")
	}
	if payload.PreyCharacterName != "Pip" {
		t.Errorf("PreyCharacterName: got %q, want %q", payload.PreyCharacterName, "Pip")
	}
	if payload.PathName != "Forest Hunt" {
		t.Errorf("PathName: got %q, want %q", payload.PathName, "Forest Hunt")
	}
	if payload.EstimatedDurationSeconds != 3600 {
		t.Errorf("EstimatedDurationSeconds: got %d, want 3600", payload.EstimatedDurationSeconds)
	}
	if payload.InitiatedBy != "predator" {
		t.Errorf("InitiatedBy: got %q, want %q", payload.InitiatedBy, "predator")
	}
	if payload.Universe != "Verdant" {
		t.Errorf("Universe: got %q, want %q", payload.Universe, "Verdant")
	}
	t.Logf("ParsePayload(interaction_proposal) OK: proposalId=%s predator=%s prey=%s path=%q dur=%ds initiatedBy=%s universe=%s",
		payload.ProposalID, payload.PredatorCharacterName, payload.PreyCharacterName,
		payload.PathName, payload.EstimatedDurationSeconds, payload.InitiatedBy, payload.Universe)
}

func TestParsePayload_UnknownType(t *testing.T) {
	raw := json.RawMessage(`{"foo":"bar"}`)
	result, err := voresky.ParsePayload("some_future_type", raw)
	if err != nil {
		t.Fatalf("unexpected error for unknown type: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for unknown type, got %T", result)
	}
	t.Log("ParsePayload(unknown_type) returned (nil, nil) — graceful fallback OK")
}

func TestParsePayload_NilPayload(t *testing.T) {
	// nil raw message
	result, err := voresky.ParsePayload(voresky.NotifInteractionProposal, nil)
	if err != nil {
		t.Fatalf("unexpected error for nil payload: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for nil payload, got %T", result)
	}

	// JSON null
	result2, err2 := voresky.ParsePayload(voresky.NotifInteractionProposal, json.RawMessage("null"))
	if err2 != nil {
		t.Fatalf("unexpected error for null payload: %v", err2)
	}
	if result2 != nil {
		t.Fatalf("expected nil result for null payload, got %T", result2)
	}
	t.Log("ParsePayload(nil payload) and ParsePayload(null payload) both returned (nil, nil) OK")
}

func TestParsePayload_AllTypes(t *testing.T) {
	cases := []struct {
		name      string
		notifType voresky.NotificationType
		raw       json.RawMessage
		wantType  interface{}
	}{
		{
			"interaction_accepted",
			voresky.NotifInteractionAccepted,
			json.RawMessage(`{"interactionId":"i1","predatorCharacterName":"A","preyCharacterName":"B","pathName":"P","universe":"U"}`),
			&voresky.InteractionBasePayload{},
		},
		{
			"interaction_rejected",
			voresky.NotifInteractionRejected,
			json.RawMessage(`{"predatorCharacterName":"A","preyCharacterName":"B","pathName":"P","universe":"U","message":"nope"}`),
			&voresky.InteractionBasePayload{},
		},
		{
			"interaction_counter_proposal",
			voresky.NotifInteractionCounter,
			json.RawMessage(`{"proposalId":"p1","predatorCharacterName":"A","preyCharacterName":"B","pathName":"P","counterCount":2,"changedRespawnTime":true,"changedBranches":false,"changedUniverse":false}`),
			&voresky.InteractionCounterPayload{},
		},
		{
			"interaction_vip_caught",
			voresky.NotifInteractionVipCaught,
			json.RawMessage(`{"interactionId":"i1","predatorCharacterName":"A","preyCharacterName":"B","pathName":"P"}`),
			&voresky.InteractionBasePayload{},
		},
		{
			"interaction_node_changed",
			voresky.NotifInteractionNodeChanged,
			json.RawMessage(`{"interactionId":"i1","newNodeVerbPast":"swallowed"}`),
			&voresky.InteractionNodePayload{},
		},
		{
			"interaction_prey_retreated",
			voresky.NotifInteractionRetreated,
			json.RawMessage(`{"interactionId":"i1","predatorCharacterName":"A","preyCharacterName":"B","retreatedToNode":"Cave"}`),
			&voresky.InteractionRetreatPayload{},
		},
		{
			"interaction_escaped",
			voresky.NotifInteractionEscaped,
			json.RawMessage(`{"interactionId":"i1","predatorCharacterName":"A","preyCharacterName":"B"}`),
			&voresky.InteractionBasePayload{},
		},
		{
			"interaction_released",
			voresky.NotifInteractionReleased,
			json.RawMessage(`{"interactionId":"i1","predatorCharacterName":"A","preyCharacterName":"B"}`),
			&voresky.InteractionBasePayload{},
		},
		{
			"interaction_respawning",
			voresky.NotifInteractionRespawning,
			json.RawMessage(`{"interactionId":"i1","predatorCharacterName":"A","preyCharacterName":"B","pathName":"P","respawnDurationSeconds":86400}`),
			&voresky.InteractionRespawnPayload{},
		},
		{
			"interaction_completed",
			voresky.NotifInteractionCompleted,
			json.RawMessage(`{"interactionId":"i1","predatorCharacterName":"A","preyCharacterName":"B","pathName":"P","hasPointOfNoReturn":true,"finalNodeName":"End"}`),
			&voresky.InteractionCompletedPayload{},
		},
		{
			"interaction_safeword",
			voresky.NotifInteractionSafeword,
			json.RawMessage(`{"pathName":"Forest Hunt"}`),
			&voresky.InteractionSafewordPayload{},
		},
		{
			"housing_invite",
			voresky.NotifHousingInvite,
			json.RawMessage(`{"houseName":"Den","ownerCharacterName":"A","memberCharacterName":"B"}`),
			&voresky.HousingPayload{},
		},
		{
			"housing_invite_accepted",
			voresky.NotifHousingInviteAccepted,
			json.RawMessage(`{"houseName":"Den","ownerCharacterName":"A","memberCharacterName":"B"}`),
			&voresky.HousingPayload{},
		},
		{
			"housing_invite_rejected",
			voresky.NotifHousingInviteRejected,
			json.RawMessage(`{"houseName":"Den","ownerCharacterName":"A","memberCharacterName":"B"}`),
			&voresky.HousingPayload{},
		},
		{
			"housing_request_accepted",
			voresky.NotifHousingRequestAccepted,
			json.RawMessage(`{"houseName":"Den","ownerCharacterName":"A","memberCharacterName":"B"}`),
			&voresky.HousingPayload{},
		},
		{
			"housing_request_rejected",
			voresky.NotifHousingRequestRejected,
			json.RawMessage(`{"houseName":"Den","ownerCharacterName":"A","memberCharacterName":"B"}`),
			&voresky.HousingPayload{},
		},
		{
			"collar_offer",
			voresky.NotifCollarOffer,
			json.RawMessage(`{"ownerCharacterName":"A","petCharacterName":"B"}`),
			&voresky.CollarPayload{},
		},
		{
			"collar_broken",
			voresky.NotifCollarBroken,
			json.RawMessage(`{"ownerCharacterName":"A","petCharacterName":"B"}`),
			&voresky.CollarPayload{},
		},
		{
			"poke",
			voresky.NotifPoke,
			json.RawMessage(`{"universe":"Verdant"}`),
			&voresky.PokePayload{},
		},
		{
			"stalk",
			voresky.NotifStalk,
			json.RawMessage(`{}`),
			&voresky.StalkPayload{},
		},
		{
			"stalk_target_available",
			voresky.NotifStalkTargetAvailable,
			json.RawMessage(`{}`),
			&voresky.StalkPayload{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := voresky.ParsePayload(tc.notifType, tc.raw)
			if err != nil {
				t.Fatalf("ParsePayload(%q) error: %v", tc.notifType, err)
			}
			if result == nil {
				t.Fatalf("ParsePayload(%q) returned nil, want non-nil", tc.notifType)
			}
			t.Logf("ParsePayload(%q) → %T OK", tc.notifType, result)
		})
	}
}
