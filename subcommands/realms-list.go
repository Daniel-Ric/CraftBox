package subcommands

import (
	"context"
	"fmt"

	"github.com/Daniel-Ric/CraftBox/locale"
	"github.com/Daniel-Ric/CraftBox/utils/auth"
	"github.com/Daniel-Ric/CraftBox/utils/commands"
)

type RealmListCMD struct{}

func (RealmListCMD) Name() string {
	return "list-realms"
}

func (RealmListCMD) Description() string {
	return locale.Loc("list_realms_synopsis", nil)
}

func (RealmListCMD) Settings() any {
	return nil
}

func (RealmListCMD) Run(ctx context.Context, settings any) error {
	if !auth.Auth.LoggedIn() {
		err := auth.Auth.Login(ctx, nil, "")
		if err != nil {
			return err
		}
	}
	realmsClient, err := auth.Auth.Account().Realms(ctx)
	if err != nil {
		return err
	}
	realms, err := realmsClient.Realms(ctx)
	if err != nil {
		return err
	}
	for _, realm := range realms {
		fmt.Println(locale.Loc("realm_list_line", locale.Strmap{"Name": realm.Name, "Id": realm.ID}))
	}
	return nil
}

func init() {
	commands.RegisterCommand(&RealmListCMD{})
}
