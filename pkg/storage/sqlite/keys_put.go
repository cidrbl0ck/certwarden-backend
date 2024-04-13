package sqlite

import (
	"certwarden-backend/pkg/domain/private_keys"
	"context"
)

// PutKeyUpdate updates an existing key in the db using any non-null
// fields specified in the UpdatePayload.
func (store *Storage) PutKeyUpdate(payload private_keys.UpdatePayload) (private_keys.Key, error) {
	// database action
	ctx, cancel := context.WithTimeout(context.Background(), store.timeout)
	defer cancel()

	query := `
	UPDATE
		private_keys
	SET
		name = case when $1 is null then name else $1 end,
		description = case when $2 is null then description else $2 end,
		api_key = case when $3 is null then api_key else $3 end,
		api_key_new = case when $4 is null then api_key_new else $4 end,
		api_key_disabled = case when $5 is null then api_key_disabled else $5 end,
		api_key_via_url = case when $6 is null then api_key_via_url else $6 end,
		updated_at = $7
	WHERE
		id = $8
	`

	_, err := store.db.ExecContext(ctx, query,
		payload.Name,
		payload.Description,
		payload.ApiKey,
		payload.ApiKeyNew,
		payload.ApiKeyDisabled,
		payload.ApiKeyViaUrl,
		payload.UpdatedAt,
		payload.ID,
	)

	if err != nil {
		return private_keys.Key{}, err
	}

	// get updated key to return
	updatedKey, err := store.GetOneKeyById(payload.ID)
	if err != nil {
		return private_keys.Key{}, err
	}

	return updatedKey, nil
}

// PutKeyUpdate sets a key's new api key and updates the updated at time
func (store *Storage) PutKeyNewApiKey(keyId int, newApiKey string, updateTimeUnix int) (err error) {
	// database action
	ctx, cancel := context.WithTimeout(context.Background(), store.timeout)
	defer cancel()

	query := `
	UPDATE
		private_keys
	SET
		api_key_new = $1,
		updated_at = $2
	WHERE
		id = $3
	`

	_, err = store.db.ExecContext(ctx, query,
		newApiKey,
		updateTimeUnix,
		keyId,
	)

	if err != nil {
		return err
	}

	return nil
}

// PutKeyApiKey sets a key's api key and updates the updated at time
func (store *Storage) PutKeyApiKey(keyId int, apiKey string, updateTimeUnix int) (err error) {
	// database action
	ctx, cancel := context.WithTimeout(context.Background(), store.timeout)
	defer cancel()

	query := `
	UPDATE
		private_keys
	SET
		api_key = $1,
		updated_at = $2
	WHERE
		id = $3
	`

	_, err = store.db.ExecContext(ctx, query,
		apiKey,
		updateTimeUnix,
		keyId,
	)

	if err != nil {
		return err
	}

	return nil
}
