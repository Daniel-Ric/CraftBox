package auth

import "testing"

func TestSelectingMissingAccountDoesNotKeepPreviousIdentity(t *testing.T) {
	a := &authSrv{}
	a.account.Store(&Account{name: "previous"})

	if err := a.SelectAccount("desktop-test-missing-account-34de441e"); err != nil {
		t.Fatal(err)
	}
	if a.Account() != nil {
		t.Fatal("previous account remained selected")
	}
}
