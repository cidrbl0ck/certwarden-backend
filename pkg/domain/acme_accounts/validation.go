package acme_accounts

import (
	"certwarden-backend/pkg/output"
	"certwarden-backend/pkg/pagination_sort"
	"certwarden-backend/pkg/storage"
	"certwarden-backend/pkg/validation"
	"errors"
	"fmt"
)

var (
	// id
	ErrIdBad = errors.New("account id is invalid")

	// name
	ErrNameBad = errors.New("account name is not valid")

	// email
	ErrEmailBad = errors.New("email is not valid")
)

// getAccount returns the Account for the specified account id.
func (service *Service) getAccount(id int) (Account, *output.JsonError) {
	// if id is not in valid range, it is definitely not valid
	if !validation.IsIdExistingValidRange(id) {
		service.logger.Debug(ErrIdBad)
		return Account{}, output.JsonErrValidationFailed(ErrIdBad)
	}

	// get from storage
	account, err := service.storage.GetOneAccountById(id)
	if err != nil {
		// special error case for no record found
		if errors.Is(err, storage.ErrNoRecord) {
			service.logger.Debug(err)
			return Account{}, output.JsonErrNotFound(fmt.Errorf("account id %d not found", id))
		} else {
			service.logger.Error(err)
			return Account{}, output.JsonErrStorageGeneric(err)
		}
	}

	return account, nil
}

// nameValid returns true if the specified account name is acceptable and
// false if it is not. This check includes validating specified
// characters and also confirms the name is not already in use by another
// account. If an id is specified, the name will also be accepted if the name
// is already in use by the specified id.
func (service *Service) nameValid(accountName string, accountId *int) bool {
	// basic check
	if !validation.NameValid(accountName) {
		return false
	}

	// make sure the name isn't already in use in storage
	account, err := service.storage.GetOneAccountByName(accountName)
	if errors.Is(err, storage.ErrNoRecord) {
		// no rows means name is not in use (valid)
		return true
	} else if err != nil {
		// any other error, invalid
		return false
	}

	// if the returned account is the account being edited, name valid
	if accountId != nil && account.ID == *accountId {
		return true
	}

	return false
}

// GetUsableAccounts returns a list of accounts that have status == valid
// and have also accepted the ToS (which is probably redundant)
func (service *Service) GetUsableAccounts() ([]Account, error) {
	accounts, _, err := service.storage.GetAllAccounts(pagination_sort.Query{})
	if err != nil {
		return nil, err
	}

	// rewrite accounts in place with only valid accounts
	newIndex := 0
	for i := range accounts {
		if accounts[i].Status == "valid" && accounts[i].AcceptedTos {
			accounts[newIndex] = accounts[i]
			newIndex++
		}
	}
	// truncate accounts
	accounts = accounts[:newIndex]

	return accounts, nil
}

// AccountUsable returns true and the Account if the specified account exists
// in storage and it is in the UsableAccounts list
func (service *Service) AccountUsable(accountId int) (bool, *Account) {
	// get usable accounts list
	accounts, err := service.GetUsableAccounts()
	if err != nil {
		return false, nil
	}

	// verify specified account id is usable
	for i := range accounts {
		if accounts[i].ID == accountId {
			return true, &accounts[i]
		}
	}

	return false, nil
}
