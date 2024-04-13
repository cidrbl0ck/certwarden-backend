package sqlite

import (
	"certwarden-backend/pkg/domain/private_keys"
	"certwarden-backend/pkg/pagination_sort"
	"certwarden-backend/pkg/storage"
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// GetAllKeys returns a slice of all Keys in the db
func (store Storage) GetAllKeys(q pagination_sort.Query) (keys []private_keys.Key, totalRowCount int, err error) {
	// validate and set sort
	sortField := q.SortField()
	switch sortField {
	// allow these as-is
	case "id":
	case "name":
	case "description":
	case "algorithm":
	// default if not in allowed list
	default:
		sortField = "name"
	}

	sort := sortField + " " + q.SortDirection()

	// do query
	ctx, cancel := context.WithTimeout(context.Background(), store.timeout)
	defer cancel()

	// WARNING: SQL Injection is possible if the variables are not properly
	// validated prior to this query being assembled!
	query := fmt.Sprintf(`
	SELECT
		id, name, description, algorithm, pem, api_key, api_key_new, api_key_disabled,
		api_key_via_url, created_at, updated_at,

		count(*) OVER() AS full_count
	FROM
		private_keys
	ORDER BY
		%s
	LIMIT
		$1
	OFFSET
		$2
	`, sort)

	rows, err := store.db.QueryContext(ctx, query,
		q.Limit(),
		q.Offset(),
	)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	// for total row count
	var totalRows int

	var allKeys []private_keys.Key
	for rows.Next() {
		var oneKeyDb keyDb
		err = rows.Scan(
			&oneKeyDb.id,
			&oneKeyDb.name,
			&oneKeyDb.description,
			&oneKeyDb.algorithmValue,
			&oneKeyDb.pem,
			&oneKeyDb.apiKey,
			&oneKeyDb.apiKeyNew,
			&oneKeyDb.apiKeyDisabled,
			&oneKeyDb.apiKeyViaUrl,
			&oneKeyDb.createdAt,
			&oneKeyDb.updatedAt,

			&totalRows,
		)
		if err != nil {
			return nil, 0, err
		}

		convertedKey := oneKeyDb.toKey()

		allKeys = append(allKeys, convertedKey)
	}

	return allKeys, totalRows, nil
}

// GetOneKeyById returns a KeyExtended based on unique id
func (store *Storage) GetOneKeyById(id int) (private_keys.Key, error) {
	return store.getOneKey(id, "")
}

// GetOneKeyByName returns a KeyExtended based on unique name
func (store *Storage) GetOneKeyByName(name string) (private_keys.Key, error) {
	return store.getOneKey(-1, name)
}

// dbGetOneKey returns a KeyExtended based on unique id or unique name
func (store Storage) getOneKey(id int, name string) (private_keys.Key, error) {
	ctx, cancel := context.WithTimeout(context.Background(), store.timeout)
	defer cancel()

	query := `
	SELECT
		id, name, description, algorithm, pem, api_key, api_key_new, api_key_disabled,
		api_key_via_url, created_at, updated_at
	FROM
		private_keys
	WHERE
		id = $1
		OR
		name = $2
	`

	row := store.db.QueryRowContext(ctx, query, id, name)

	var oneKeyDb keyDb
	err := row.Scan(
		&oneKeyDb.id,
		&oneKeyDb.name,
		&oneKeyDb.description,
		&oneKeyDb.algorithmValue,
		&oneKeyDb.pem,
		&oneKeyDb.apiKey,
		&oneKeyDb.apiKeyNew,
		&oneKeyDb.apiKeyDisabled,
		&oneKeyDb.apiKeyViaUrl,
		&oneKeyDb.createdAt,
		&oneKeyDb.updatedAt,
	)

	if err != nil {
		// if no record exists
		if errors.Is(err, sql.ErrNoRows) {
			err = storage.ErrNoRecord
		}
		return private_keys.Key{}, err
	}

	// convert to KeyExtended
	oneKeyExt := oneKeyDb.toKey()

	return oneKeyExt, nil
}

// GetAvailableKeys returns a slice of private keys that exist but are not already associated
// with a known ACME account or certificate
func (store *Storage) GetAvailableKeys() ([]private_keys.Key, error) {
	ctx, cancel := context.WithTimeout(context.Background(), store.timeout)
	defer cancel()

	query := `
		SELECT
			pk.id, pk.name, pk.description, pk.algorithm, pk.pem, pk.api_key, pk.api_key_new,
			pk.api_key_disabled, pk.api_key_via_url, pk.created_at, pk.updated_at
		FROM
		  private_keys pk
		WHERE
			NOT EXISTS(
				SELECT
					aa.private_key_id
				FROM
					acme_accounts aa
				WHERE
					pk.id = aa.private_key_id
			)
			AND
			NOT EXISTS(
				SELECT
					c.private_key_id
				FROM
					certificates c
				WHERE
					pk.id = c.private_key_id
			)
		ORDER BY name
	`

	rows, err := store.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var availableKeys []private_keys.Key
	for rows.Next() {
		var oneKeyDb keyDb

		err = rows.Scan(
			&oneKeyDb.id,
			&oneKeyDb.name,
			&oneKeyDb.description,
			&oneKeyDb.algorithmValue,
			&oneKeyDb.pem,
			&oneKeyDb.apiKey,
			&oneKeyDb.apiKeyNew,
			&oneKeyDb.apiKeyDisabled,
			&oneKeyDb.apiKeyViaUrl,
			&oneKeyDb.createdAt,
			&oneKeyDb.updatedAt,
		)
		if err != nil {
			return nil, err
		}

		oneKeyExt := oneKeyDb.toKey()

		availableKeys = append(availableKeys, oneKeyExt)
	}

	return availableKeys, nil
}
