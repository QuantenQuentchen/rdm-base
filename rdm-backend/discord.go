package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"slices"
)

const discordAPIBase = "https://discord.com/api/v10"

var ErrNotGuildMember = errors.New("user is not a member of the guild")

type DiscordClient struct {
	BotToken string
	HTTP     *http.Client
}

// GuildMember represents Discord's Guild Member object.
//
// See:
// https://discord.com/developers/docs/resources/guild#guild-member-object
type GuildMember struct {
	User *DiscordUser `json:"user,omitempty"`

	Nick         *string  `json:"nick"`
	Avatar       *string  `json:"avatar"`
	Roles        []string `json:"roles"`
	JoinedAt     string   `json:"joined_at"`
	PremiumSince *string  `json:"premium_since"`
	Deaf         bool     `json:"deaf"`
	Mute         bool     `json:"mute"`
	Flags        int      `json:"flags"`

	Pending *bool `json:"pending"`

	Permissions *string `json:"permissions"`

	CommunicationDisabledUntil *string `json:"communication_disabled_until"`

	UnusualDMActivityUntil *string `json:"unusual_dm_activity_until"`
}

// DiscordUser represents Discord's User object.
//
// This is intentionally fairly complete so you can use the object
// for more than just its ID.
type DiscordUser struct {
	ID               string  `json:"id"`
	Username         string  `json:"username"`
	Discriminator    string  `json:"discriminator"`
	GlobalName       *string `json:"global_name"`
	Avatar           *string `json:"avatar"`
	Bot              bool    `json:"bot,omitempty"`
	System           bool    `json:"system,omitempty"`
	MFAEnabled       bool    `json:"mfa_enabled,omitempty"`
	Banner           *string `json:"banner,omitempty"`
	AccentColor      *int    `json:"accent_color,omitempty"`
	Locale           *string `json:"locale,omitempty"`
	Verified         bool    `json:"verified,omitempty"`
	Email            *string `json:"email,omitempty"`
	Flags            int     `json:"flags,omitempty"`
	PremiumType      *int    `json:"premium_type,omitempty"`
	PublicFlags      int     `json:"public_flags,omitempty"`
	AvatarDecoration *string `json:"avatar_decoration_data,omitempty"`
}

func NewDiscordClient() *DiscordClient {
	return &DiscordClient{
		BotToken: mustEnv("DISCORD_BOT_TOKEN"),
		HTTP:     http.DefaultClient,
	}
}

func (d *DiscordClient) GetGuildMember(
	ctx context.Context,
	guildID string,
	userID string,
) (*GuildMember, error) {
	url := fmt.Sprintf(
		"%s/guilds/%s/members/%s",
		discordAPIBase,
		guildID,
		userID,
	)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		url,
		nil,
	)
	if err != nil {
		log.Printf("failed to create guild member request: %v", err)
		return nil, err
	}

	req.Header.Set("Authorization", "Bot "+d.BotToken)
	req.Header.Set("Accept", "application/json")

	resp, err := d.HTTP.Do(req)
	if err != nil {
		log.Printf("failed to execute guild member request: %v", err)
		return nil, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("Error closing response body:", err)
			return
		}
	}(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		// Continue below.

	case http.StatusNotFound:
		return nil, ErrNotGuildMember

	default:
		return nil, fmt.Errorf(
			"discord guild member request failed: %s",
			resp.Status,
		)
	}

	var member GuildMember

	if err := json.NewDecoder(resp.Body).Decode(&member); err != nil {
		return nil, fmt.Errorf(
			"decode discord guild member: %w",
			err,
		)
	}

	return &member, nil
}

func (d *DiscordClient) hasAdminRole(ctx context.Context, userID string) bool {
	member, err := d.GetGuildMember(ctx, testServerID, userID)
	if err != nil {
		return false
	}

	return slices.Contains(member.Roles, rdmAdminRoleID) || slices.Contains(member.Roles, rdmViceAdminRoleID)
}
