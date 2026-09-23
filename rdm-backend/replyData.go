package main

import (
	"context"
	"errors"
	"slices"
)

type UserReply struct {
	ID          string   `json:"id"`
	DisplayName string   `json:"displayName"`
	UserName    *string  `json:"username"`
	AvatarURL   *string  `json:"avatarUrl"`
	Roles       []string `json:"roles"`
}

func (a *AuthStruct) generateUserReply(ctx context.Context, discordID string) (UserReply, error) {
	guildMember, err := a.discordClient.GetGuildMember(ctx, testServerID, discordID)
	if err != nil {
		return UserReply{}, err
	}
	if guildMember == nil {
		return UserReply{}, errors.New("user not found")
	}
	memberAvailable := guildMember.User != nil
	var userName string

	if memberAvailable {
		userName = guildMember.User.Username
	} else {
		userName = "Discord did a fucky-wucky"
	}

	var displayName string
	if guildMember.Nick != nil || *guildMember.Nick != "" {
		displayName = *guildMember.Nick
	} else {
		displayName = userName
	}

	var avatar *string
	if guildMember.Avatar != nil {
		avatar = guildMember.Avatar
	} else {
		if !memberAvailable {
			avatar = nil
		} else {
			avatar = guildMember.User.Avatar
		}
	}
	var roles []string
	if guildMember.Roles != nil {
		roles = []string{}
		if slices.Contains(guildMember.Roles, rdmParticipationRoleID) {
			roles = append(roles, "voter")
		}

		if slices.Contains(guildMember.Roles, rdmViceAdminRoleID) {
			roles = append(roles, "admin")
		}

		if slices.Contains(guildMember.Roles, rdmAdminRoleID) {
			roles = append(roles, "admin")
		}
	} else {
		roles = nil
	}
	return UserReply{
		ID:          discordID,
		DisplayName: displayName,
		UserName:    &userName,
		AvatarURL:   generateDiscordAvatarURL(discordID, avatar),
		Roles:       roles,
	}, nil
}

func generateDiscordAvatarURL(discordID string, avatarHash *string) *string {
	if avatarHash == nil {
		return nil
	}
	url := `https://cdn.discordapp.com/avatars/` + discordID + `/` + *avatarHash
	return &url
}
