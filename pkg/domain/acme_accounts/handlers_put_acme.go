package acme_accounts

import (
	"certwarden-backend/pkg/acme"
	"certwarden-backend/pkg/output"
	"certwarden-backend/pkg/validation"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/julienschmidt/httprouter"
)

// changeEmailPayload is the struct for updating an account's email address
// do not export and do not add id/updatedAt fields since this does not get
// sent to storage
type changeEmailPayload struct {
	Email *string `json:"email"`
}

// ChangeEmail() is a handler that updates an ACME account with the specified
// email address and saves the updated address to storage
func (service *Service) ChangeEmail(w http.ResponseWriter, r *http.Request) *output.JsonError {
	// get id from param
	idParam := httprouter.ParamsFromContext(r.Context()).ByName("id")
	id, err := strconv.Atoi(idParam)
	if err != nil {
		service.logger.Debug(err)
		return output.JsonErrValidationFailed(err)
	}

	// decode payload
	var payload changeEmailPayload
	err = json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		service.logger.Debug(err)
		return output.JsonErrValidationFailed(err)
	}

	// validation
	// id
	account, outErr := service.getAccount(id)
	if outErr != nil {
		return outErr
	}

	// email (allow user to try blank -- ACME Server may reject though)
	// but do reject field was missing
	if payload.Email == nil ||
		(*payload.Email != "" && !validation.EmailValid(*payload.Email)) {
		service.logger.Debug(ErrEmailBad)
		return output.JsonErrValidationFailed(ErrEmailBad)
	}
	// end validation

	// get AccountKey
	acmeAccountKey, err := account.AcmeAccountKey()
	if err != nil {
		service.logger.Error(err)
		return output.JsonErrInternal(err)
	}

	// make ACME update email payload
	acmePayload := acme.UpdateAccountPayload{
		Contact: emailToContact(*payload.Email),
	}

	// send the email update to ACME
	acmeService, err := service.acmeServerService.AcmeService(account.AcmeServer.ID)
	if err != nil {
		service.logger.Error(err)
		return output.JsonErrInternal(err)
	}

	var acmeAccount AcmeAccount
	acmeAccount.Account, err = acmeService.UpdateAccount(acmePayload, acmeAccountKey)
	if err != nil {
		service.logger.Error(err)
		return output.JsonErrInternal(err)
	}

	// add additional details to the payload before saving
	acmeAccount.ID = id
	acmeAccount.UpdatedAt = int(time.Now().Unix())

	// save ACME response to account
	updatedAcct, err := service.storage.PutAcmeAccountResponse(acmeAccount)
	if err != nil {
		service.logger.Error(err)
		return output.JsonErrStorageGeneric(err)
	}

	detailedResp, err := updatedAcct.detailedResponse(service)
	if err != nil {
		err = fmt.Errorf("failed to generate account summary response (%s)", err)
		service.logger.Error(err)
		return output.JsonErrInternal(err)
	}

	// write response
	response := &accountResponse{}
	response.StatusCode = http.StatusOK
	response.Message = "updated account"
	response.Account = detailedResp

	err = service.output.WriteJSON(w, response)
	if err != nil {
		service.logger.Errorf("failed to write json (%s)", err)
		return output.JsonErrWriteJsonError(err)
	}

	return nil
}

// RolloverKeyPayload is used to change an account's private key
type RolloverKeyPayload struct {
	ID           int  `json:"-"`
	PrivateKeyID *int `json:"private_key_id"`
	UpdatedAt    int  `json:"-"`
}

// RolloverKey changes the private key used for an account
func (service *Service) RolloverKey(w http.ResponseWriter, r *http.Request) *output.JsonError {
	// decode payload
	var payload RolloverKeyPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		service.logger.Debug(err)
		return output.JsonErrValidationFailed(err)
	}

	// get id from param
	idParam := httprouter.ParamsFromContext(r.Context()).ByName("id")
	payload.ID, err = strconv.Atoi(idParam)
	if err != nil {
		service.logger.Debug(err)
		return output.JsonErrValidationFailed(err)
	}

	// validation
	// id
	account, outErr := service.getAccount(payload.ID)
	if outErr != nil {
		return outErr
	}

	// new private key
	if payload.PrivateKeyID == nil || !service.keys.KeyAvailable(*payload.PrivateKeyID) {
		err = errors.New("invalid private key specified for account key rollover")
		service.logger.Debug(err)
		return output.JsonErrValidationFailed(err)
	}
	// end validation

	// get AccountKey
	oldAcmeAccountKey, err := account.AcmeAccountKey()
	if err != nil {
		service.logger.Error(err)
		return output.JsonErrInternal(err)
	}

	// fetch new private key
	newKey, err := service.storage.GetOneKeyById(*payload.PrivateKeyID)
	if err != nil {
		service.logger.Error(err)
		return output.JsonErrStorageGeneric(err)
	}

	// get crypto key from the new key
	newCryptoKey, err := newKey.CryptoPrivateKey()
	if err != nil {
		service.logger.Error(err)
		return output.JsonErrInternal(err)
	}

	// send the rollover to ACME
	acmeService, err := service.acmeServerService.AcmeService(account.AcmeServer.ID)
	if err != nil {
		service.logger.Error(err)
		return output.JsonErrInternal(err)
	}

	err = acmeService.RolloverAccountKey(newCryptoKey, oldAcmeAccountKey)
	if err != nil {
		service.logger.Error(err)
		return output.JsonErrInternal(err)
	}

	// add additional details to the payload before saving
	payload.UpdatedAt = int(time.Now().Unix())

	// update private key id in db
	updatedAcct, err := service.storage.PutNewAccountKey(payload)
	if err != nil {
		service.logger.Error(err)
		return output.JsonErrStorageGeneric(err)
	}

	detailedResp, err := updatedAcct.detailedResponse(service)
	if err != nil {
		err = fmt.Errorf("failed to generate account summary response (%s)", err)
		service.logger.Error(err)
		return output.JsonErrInternal(err)
	}

	// write response
	response := &accountResponse{}
	response.StatusCode = http.StatusOK
	response.Message = "updated account"
	response.Account = detailedResp

	err = service.output.WriteJSON(w, response)
	if err != nil {
		service.logger.Errorf("failed to write json (%s)", err)
		return output.JsonErrWriteJsonError(err)
	}

	return nil
}
