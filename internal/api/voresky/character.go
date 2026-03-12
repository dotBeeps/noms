package voresky

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Character represents a Voresky character from the game API.
type Character struct {
	ID               string    `json:"id"`
	UserDID          string    `json:"userDid"`
	Name             string    `json:"name"`
	Description      string    `json:"description"`
	Avatar           string    `json:"avatar"`
	AvatarHlsURL     string    `json:"avatarHlsUrl"`
	Banner           string    `json:"banner"`
	BannerHlsURL     string    `json:"bannerHlsUrl"`
	Status           string    `json:"status"`
	FeaturedUniverse string    `json:"featuredUniverse"`
	Position         int       `json:"position"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

// CharacterOwner is the owner info returned alongside a character.
type CharacterOwner struct {
	DID         string `json:"did"`
	Handle      string `json:"handle"`
	DisplayName string `json:"displayName"`
	Avatar      string `json:"avatar"`
}

// CharacterResponse is the shape returned by GET /api/game/characters/[id].
type CharacterResponse struct {
	Character Character      `json:"character"`
	Owner     CharacterOwner `json:"owner"`
	IsOwner   bool           `json:"isOwner"`
}

// CharacterAvatar is the avatar shape returned by the list endpoint, which
// wraps the URL with optional crop parameters.
type CharacterAvatar struct {
	URL  string          `json:"url"`
	Crop json.RawMessage `json:"crop,omitempty"`
}

// listCharacterEntry is the shape of a single character in the list response,
// where avatar is wrapped with crop info.
type listCharacterEntry struct {
	ID               string          `json:"id"`
	UserDID          string          `json:"userDid"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	Avatar           CharacterAvatar `json:"avatar"`
	AvatarHlsURL     string          `json:"avatarHlsUrl"`
	Banner           string          `json:"banner"`
	BannerHlsURL     string          `json:"bannerHlsUrl"`
	Status           string          `json:"status"`
	FeaturedUniverse string          `json:"featuredUniverse"`
	Position         int             `json:"position"`
	CreatedAt        time.Time       `json:"createdAt"`
	UpdatedAt        time.Time       `json:"updatedAt"`
}

// listCharactersResponse is the raw JSON returned by GET /api/game/characters.
type listCharactersResponse struct {
	Characters      []listCharacterEntry `json:"characters"`
	Count           int                  `json:"count"`
	Limit           int                  `json:"limit"`
	MainCharacterID string               `json:"mainCharacterId"`
	PetCharacterIDs []string             `json:"petCharacterIds"`
}

// ListCharactersResult is the parsed result from GetMyCharacters.
type ListCharactersResult struct {
	Characters      []Character
	MainCharacterID string
	PetCharacterIDs []string
}

// GetCharacter fetches a single character by ID.
// Calls GET /api/game/characters/{id}.
func (c *VoreskyClient) GetCharacter(ctx context.Context, id string) (*CharacterResponse, error) {
	resp, err := c.Get(ctx, "/api/game/characters/"+id)
	if err != nil {
		return nil, fmt.Errorf("get character: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, ParseError(resp)
	}

	var cr CharacterResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, fmt.Errorf("decode character response: %w", err)
	}
	return &cr, nil
}

// GetMyCharacters lists all characters for the authenticated user.
// Calls GET /api/game/characters.
func (c *VoreskyClient) GetMyCharacters(ctx context.Context) (*ListCharactersResult, error) {
	resp, err := c.Get(ctx, "/api/game/characters")
	if err != nil {
		return nil, fmt.Errorf("list characters: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, ParseError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read characters response: %w", err)
	}

	var raw listCharactersResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("decode characters response: %w", err)
	}

	characters := make([]Character, len(raw.Characters))
	for i, entry := range raw.Characters {
		characters[i] = Character{
			ID:               entry.ID,
			UserDID:          entry.UserDID,
			Name:             entry.Name,
			Description:      entry.Description,
			Avatar:           entry.Avatar.URL,
			AvatarHlsURL:     entry.AvatarHlsURL,
			Banner:           entry.Banner,
			BannerHlsURL:     entry.BannerHlsURL,
			Status:           entry.Status,
			FeaturedUniverse: entry.FeaturedUniverse,
			Position:         entry.Position,
			CreatedAt:        entry.CreatedAt,
			UpdatedAt:        entry.UpdatedAt,
		}
	}

	return &ListCharactersResult{
		Characters:      characters,
		MainCharacterID: raw.MainCharacterID,
		PetCharacterIDs: raw.PetCharacterIDs,
	}, nil
}
